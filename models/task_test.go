package models

import (
	"strings"
	"time"

	"github.com/gofrs/uuid"
)

func (ms *ModelSuite) Test_Task_Create_Valid() {
	task := &Task{Title: "Comprar café"}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.False(verrs.HasAny())

	ms.NotZero(task.ID)
	ms.False(task.Completed)

	count, err := ms.DB.Count(&Task{})
	ms.NoError(err)
	ms.Equal(1, count)
}

func (ms *ModelSuite) Test_Task_Title_Required() {
	task := &Task{Title: ""}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.True(verrs.HasAny())
	ms.Contains(verrs.Get("title"), "El título es obligatorio.")

	count, err := ms.DB.Count(&Task{})
	ms.NoError(err)
	ms.Equal(0, count)
}

func (ms *ModelSuite) Test_Task_Title_OnlySpaces_Invalid() {
	task := &Task{Title: "   \t  "}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.True(verrs.HasAny())
	ms.Contains(verrs.Get("title"), "El título es obligatorio.")
}

func (ms *ModelSuite) Test_Task_Title_MaxLength() {
	task := &Task{Title: strings.Repeat("a", TitleMaxLength)}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.False(verrs.HasAny())
}

func (ms *ModelSuite) Test_Task_Title_TooLong_Invalid() {
	task := &Task{Title: strings.Repeat("a", TitleMaxLength+1)}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.True(verrs.HasAny())
	ms.NotEmpty(verrs.Get("title"))
}

func (ms *ModelSuite) Test_Task_Toggle_Persists() {
	task := &Task{Title: "Revisar el correo"}
	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.False(verrs.HasAny())

	task.Completed = true
	ms.NoError(ms.DB.Update(task))

	reloaded := &Task{}
	ms.NoError(ms.DB.Find(reloaded, task.ID))
	ms.True(reloaded.Completed)
}

func (ms *ModelSuite) createTask(title string, completed bool, position int) *Task {
	task := &Task{Title: title, Completed: completed, Position: position}
	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.False(verrs.HasAny())
	ms.NoError(ms.DB.Reload(task))
	return task
}

func (ms *ModelSuite) Test_Task_TopPosition_EmptyBlock() {
	pos, err := TopPosition(ms.DB, false)
	ms.NoError(err)
	ms.Equal(-1, pos)
}

func (ms *ModelSuite) Test_Task_TopPosition_BelowMinimum() {
	ms.createTask("Pendiente A", false, 0)
	ms.createTask("Pendiente B", false, 3)
	ms.createTask("Completada", true, -10)

	pos, err := TopPosition(ms.DB, false)
	ms.NoError(err)
	ms.Equal(-1, pos)

	pos, err = TopPosition(ms.DB, true)
	ms.NoError(err)
	ms.Equal(-11, pos)
}

func (ms *ModelSuite) Test_Task_LoadTasks_OrdersByBlockAndPosition() {
	c2 := ms.createTask("Completada 2", true, 5)
	p2 := ms.createTask("Pendiente 2", false, 1)
	c1 := ms.createTask("Completada 1", true, -3)
	p3 := ms.createTask("Pendiente 3", false, 7)
	p1 := ms.createTask("Pendiente 1", false, -2)

	tasks, err := LoadTasks(ms.DB)
	ms.NoError(err)

	titles := []string{}
	for _, t := range tasks {
		titles = append(titles, t.Title)
	}
	ms.Equal([]string{p1.Title, p2.Title, p3.Title, c1.Title, c2.Title}, titles)

	pending := tasks.Pending()
	ms.Len(pending, 3)
	ms.Equal(p1.ID, pending[0].ID)
	ms.Equal(p2.ID, pending[1].ID)
	ms.Equal(p3.ID, pending[2].ID)

	completed := tasks.Completed()
	ms.Len(completed, 2)
	ms.Equal(c1.ID, completed[0].ID)
	ms.Equal(c2.ID, completed[1].ID)
}

func (ms *ModelSuite) Test_Task_ReorderTasks_RewritesPositions() {
	a := ms.createTask("A", false, 0)
	b := ms.createTask("B", false, 1)
	c := ms.createTask("C", false, 2)
	updatedAt := map[uuid.UUID]time.Time{a.ID: a.UpdatedAt, b.ID: b.UpdatedAt, c.ID: c.UpdatedAt}

	ms.NoError(ReorderTasks(ms.DB, false, []uuid.UUID{c.ID, a.ID, b.ID}))

	for want, task := range []*Task{c, a, b} {
		ms.NoError(ms.DB.Reload(task))
		ms.Equal(want, task.Position)
		ms.True(task.UpdatedAt.Equal(updatedAt[task.ID]))
	}
}

func (ms *ModelSuite) Test_Task_ReorderTasks_Mismatch() {
	a := ms.createTask("A", false, 0)
	b := ms.createTask("B", false, 1)
	c := ms.createTask("C", false, 2)
	done := ms.createTask("Hecha", true, 0)

	cases := map[string][]uuid.UUID{
		"incompletos":    {a.ID, b.ID},
		"de otro bloque": {a.ID, b.ID, done.ID},
		"inexistente":    {a.ID, b.ID, uuid.Must(uuid.NewV4())},
		"repetidos":      {a.ID, b.ID, b.ID},
	}
	for name, ids := range cases {
		err := ReorderTasks(ms.DB, false, ids)
		ms.ErrorIs(err, ErrReorderMismatch, name)
	}

	for want, task := range []*Task{a, b, c} {
		ms.NoError(ms.DB.Reload(task))
		ms.Equal(want, task.Position)
	}
	ms.NoError(ms.DB.Reload(done))
	ms.Equal(0, done.Position)
}
