package actions

import (
	"net/url"
	"strings"

	"todo/models"

	"github.com/gofrs/uuid"
)

func (as *ActionSuite) Test_TasksIndex_EmptyState() {
	res := as.HTML("/").Get()
	as.Equal(200, res.Code)
	as.Contains(res.Body.String(), "No hay tareas todavía")
}

func (as *ActionSuite) Test_TasksIndex_ListsTasks() {
	as.LoadFixture("tres tareas")

	res := as.HTML("/").Get()
	as.Equal(200, res.Code)

	body := res.Body.String()
	as.Contains(body, "Comprar material de oficina")
	as.Contains(body, "Llamar al gestor")
	as.Contains(body, "Publicar la oferta de empleo")
	as.Contains(body, "Completada")
}

func (as *ActionSuite) Test_TasksCreate_Valid() {
	res := as.HTML("/tasks").Post(&models.Task{Title: "  Nueva tarea de prueba  "})
	as.Equal(303, res.Code)
	as.Equal("/", res.Header().Get("Location"))

	task := &models.Task{}
	as.NoError(as.DB.First(task))
	as.Equal("Nueva tarea de prueba", task.Title)
	as.False(task.Completed)
}

func (as *ActionSuite) Test_TasksCreate_EmptyTitle_Invalid() {
	res := as.HTML("/tasks").Post(&models.Task{Title: ""})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "El título es obligatorio.")

	count, err := as.DB.Count(&models.Task{})
	as.NoError(err)
	as.Equal(0, count)
}

func (as *ActionSuite) Test_TasksCreate_OnlySpaces_Invalid() {
	res := as.HTML("/tasks").Post(&models.Task{Title: "     "})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "El título es obligatorio.")

	count, err := as.DB.Count(&models.Task{})
	as.NoError(err)
	as.Equal(0, count)
}

func (as *ActionSuite) Test_TasksToggle_CompletesAndReopens() {
	task := &models.Task{Title: "Pendiente de completar"}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())

	res := as.HTML("/tasks/%s/toggle", task.ID).Put(url.Values{})
	as.Equal(303, res.Code)

	as.NoError(as.DB.Reload(task))
	as.True(task.Completed)

	res = as.HTML("/tasks/%s/toggle", task.ID).Put(url.Values{})
	as.Equal(303, res.Code)

	as.NoError(as.DB.Reload(task))
	as.False(task.Completed)
}

func (as *ActionSuite) Test_TasksToggle_NotFound() {
	id := uuid.Must(uuid.NewV4())
	res := as.HTML("/tasks/%s/toggle", id).Put(url.Values{})
	as.Equal(404, res.Code)
}

func (as *ActionSuite) Test_TasksToggle_InvalidID() {
	res := as.HTML("/tasks/no-es-un-uuid/toggle").Put(url.Values{})
	as.Equal(404, res.Code)
}

func (as *ActionSuite) Test_TasksDestroy() {
	task := &models.Task{Title: "Tarea que se elimina"}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())

	res := as.HTML("/tasks/%s", task.ID).Delete()
	as.Equal(303, res.Code)

	count, err := as.DB.Count(&models.Task{})
	as.NoError(err)
	as.Equal(0, count)
}

func (as *ActionSuite) Test_TasksDestroy_NotFound() {
	id := uuid.Must(uuid.NewV4())
	res := as.HTML("/tasks/%s", id).Delete()
	as.Equal(404, res.Code)
}

func (as *ActionSuite) taskForEditing(completed bool) *models.Task {
	task := &models.Task{Title: "Título original", Completed: completed}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())
	as.NoError(as.DB.Reload(task))
	return task
}

func (as *ActionSuite) Test_TasksIndex_EditLink() {
	task := as.taskForEditing(false)
	res := as.HTML("/").Get()
	as.Equal(200, res.Code)
	as.Contains(res.Body.String(), "/tasks/"+task.ID.String()+"/edit")
	as.Contains(res.Body.String(), "Editar tarea: Título original")
}

func (as *ActionSuite) Test_TasksEdit_PrefillsTitle() {
	task := as.taskForEditing(true)
	res := as.HTML("/tasks/%s/edit", task.ID).Get()
	as.Equal(200, res.Code)
	as.Contains(res.Body.String(), `value="Título original"`)
	as.Contains(res.Body.String(), `value="PUT"`)
	as.Contains(res.Body.String(), "Guardar cambios")
	as.Contains(res.Body.String(), "Cancelar")
	as.NoError(as.DB.Reload(task))
	as.True(task.Completed)
}

func (as *ActionSuite) Test_TasksEdit_NotFound() {
	res := as.HTML("/tasks/%s/edit", uuid.Must(uuid.NewV4())).Get()
	as.Equal(404, res.Code)
}

func (as *ActionSuite) Test_TasksEdit_InvalidID() {
	res := as.HTML("/tasks/no-es-un-uuid/edit").Get()
	as.Equal(404, res.Code)
}

func (as *ActionSuite) Test_TasksEdit_EscapesTitle() {
	task := &models.Task{Title: `<img src=x onerror="alert(1)">`}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())
	res := as.HTML("/tasks/%s/edit", task.ID).Get()
	as.Equal(200, res.Code)
	as.NotContains(res.Body.String(), `<img src=x`)
	as.Contains(res.Body.String(), "&lt;img")
}

func (as *ActionSuite) Test_TasksUpdate_PersistsTitleOnly() {
	task := as.taskForEditing(true)
	id, createdAt, updatedAt, position := task.ID, task.CreatedAt, task.UpdatedAt, task.Position
	res := as.HTML("/tasks/%s", id).Put(url.Values{
		"title": {"  Título actualizado  "}, "completed": {"false"},
		"id": {uuid.Must(uuid.NewV4()).String()}, "created_at": {"2000-01-01T00:00:00Z"},
	})
	as.Equal(303, res.Code)
	as.Equal("/", res.Header().Get("Location"))
	as.NoError(as.DB.Reload(task))
	as.Equal("Título actualizado", task.Title)
	as.Equal(id, task.ID)
	as.True(task.Completed)
	as.True(task.CreatedAt.Equal(createdAt))
	as.False(task.UpdatedAt.Before(updatedAt))
	as.Equal(position, task.Position)
	count, err := as.DB.Count(&models.Task{})
	as.NoError(err)
	as.Equal(1, count)
}

func (as *ActionSuite) Test_TasksUpdate_PendingTaskRemainsPending() {
	task := as.taskForEditing(false)
	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {"Nuevo título"}, "completed": {"true"}})
	as.Equal(303, res.Code)
	as.NoError(as.DB.Reload(task))
	as.Equal("Nuevo título", task.Title)
	as.False(task.Completed)
}

func (as *ActionSuite) Test_TasksUpdate_EmptyTitle() {
	task := as.taskForEditing(false)
	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {""}})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "El título es obligatorio.")
	as.Contains(res.Body.String(), `aria-describedby="title-error"`)
	as.NoError(as.DB.Reload(task))
	as.Equal("Título original", task.Title)
}

func (as *ActionSuite) Test_TasksUpdate_OnlySpaces() {
	task := as.taskForEditing(true)
	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {"   "}})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "El título es obligatorio.")
	as.NoError(as.DB.Reload(task))
	as.Equal("Título original", task.Title)
	as.True(task.Completed)
}

func (as *ActionSuite) Test_TasksUpdate_TitleTooLong() {
	task := as.taskForEditing(false)
	title := strings.Repeat("á", models.TitleMaxLength+1)
	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {title}})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "El título no puede superar los 200 caracteres.")
	as.Contains(res.Body.String(), title)
	as.NoError(as.DB.Reload(task))
	as.Equal("Título original", task.Title)
}

func (as *ActionSuite) Test_TasksUpdate_TitleAtLimit() {
	task := as.taskForEditing(false)
	title := strings.Repeat("á", models.TitleMaxLength)
	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {title}})
	as.Equal(303, res.Code)
	as.NoError(as.DB.Reload(task))
	as.Equal(title, task.Title)
}

func (as *ActionSuite) Test_TasksUpdate_NotFound() {
	res := as.HTML("/tasks/%s", uuid.Must(uuid.NewV4())).Put(url.Values{"title": {"Título nuevo"}})
	as.Equal(404, res.Code)
}

func (as *ActionSuite) Test_TasksUpdate_InvalidID() {
	res := as.HTML("/tasks/no-es-un-uuid").Put(url.Values{"title": {"Título nuevo"}})
	as.Equal(404, res.Code)
}

func (as *ActionSuite) positionedTask(title string, completed bool, position int) *models.Task {
	task := &models.Task{Title: title, Completed: completed, Position: position}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())
	as.NoError(as.DB.Reload(task))
	return task
}

func (as *ActionSuite) assertPositions(tasks ...*models.Task) {
	for want, task := range tasks {
		as.NoError(as.DB.Reload(task))
		as.Equal(want, task.Position, task.Title)
	}
}

func (as *ActionSuite) Test_TasksIndex_SeparatesBlocks() {
	as.positionedTask("Tarea hecha", true, -5)
	as.positionedTask("Pendiente uno", false, 0)
	as.positionedTask("Pendiente dos", false, 1)

	res := as.HTML("/").Get()
	as.Equal(200, res.Code)
	body := res.Body.String()

	as.Contains(body, `data-task-list="pending"`)
	as.Contains(body, `data-task-list="completed"`)

	completedHeading := strings.Index(body, "Completadas")
	done := strings.Index(body, "Tarea hecha")
	for _, title := range []string{"Pendiente uno", "Pendiente dos"} {
		i := strings.Index(body, title)
		as.NotEqual(-1, i)
		as.Less(i, completedHeading)
		as.Less(i, done)
	}
}

func (as *ActionSuite) Test_TasksIndex_OrdersByPosition() {
	third := as.positionedTask("Tercera", false, 2)
	first := as.positionedTask("Primera", false, 0)
	second := as.positionedTask("Segunda", false, 1)

	res := as.HTML("/").Get()
	as.Equal(200, res.Code)
	body := res.Body.String()

	as.Contains(body, `data-task-id="`+first.ID.String()+`"`)
	as.Contains(body, `data-task-id="`+second.ID.String()+`"`)
	as.Contains(body, `data-task-id="`+third.ID.String()+`"`)
	as.Less(strings.Index(body, "Primera"), strings.Index(body, "Segunda"))
	as.Less(strings.Index(body, "Segunda"), strings.Index(body, "Tercera"))
}

func (as *ActionSuite) Test_TasksCreate_PlacesNewTaskFirst() {
	existing := as.positionedTask("Tarea existente", false, 0)

	res := as.HTML("/tasks").Post(&models.Task{Title: "Tarea recién creada"})
	as.Equal(303, res.Code)

	created := &models.Task{}
	as.NoError(as.DB.Where("title = ?", "Tarea recién creada").First(created))
	as.Less(created.Position, existing.Position)

	body := as.HTML("/").Get().Body.String()
	as.Less(strings.Index(body, "Tarea recién creada"), strings.Index(body, "Tarea existente"))
}

func (as *ActionSuite) Test_TasksToggle_MovesToTopOfDestination() {
	as.positionedTask("Completada A", true, 0)
	as.positionedTask("Completada B", true, 1)
	pending := as.positionedTask("Pendiente A", false, 0)
	as.positionedTask("Pendiente B", false, 1)

	res := as.HTML("/tasks/%s/toggle", pending.ID).Put(url.Values{})
	as.Equal(303, res.Code)
	as.NoError(as.DB.Reload(pending))
	as.True(pending.Completed)
	as.Less(pending.Position, 0)

	res = as.HTML("/tasks/%s/toggle", pending.ID).Put(url.Values{})
	as.Equal(303, res.Code)
	as.NoError(as.DB.Reload(pending))
	as.False(pending.Completed)
	as.Less(pending.Position, 1)
}

func (as *ActionSuite) Test_TasksReorder_Persists() {
	a := as.positionedTask("Tarea A", false, 0)
	b := as.positionedTask("Tarea B", false, 1)
	c := as.positionedTask("Tarea C", false, 2)

	res := as.HTML("/tasks/reorder").Post(url.Values{
		"block": {"pending"},
		"ids":   {c.ID.String(), a.ID.String(), b.ID.String()},
	})
	as.Equal(204, res.Code)

	as.assertPositions(c, a, b)

	body := as.HTML("/").Get().Body.String()
	as.Less(strings.Index(body, "Tarea C"), strings.Index(body, "Tarea A"))
	as.Less(strings.Index(body, "Tarea A"), strings.Index(body, "Tarea B"))
}

func (as *ActionSuite) Test_TasksReorder_InvalidBlock() {
	a := as.positionedTask("Tarea A", false, 0)

	res := as.HTML("/tasks/reorder").Post(url.Values{"block": {"otro"}, "ids": {a.ID.String()}})
	as.Equal(422, res.Code)
}

func (as *ActionSuite) Test_TasksReorder_InvalidID() {
	a := as.positionedTask("Tarea A", false, 0)

	res := as.HTML("/tasks/reorder").Post(url.Values{"block": {"pending"}, "ids": {a.ID.String(), "no-es-un-uuid"}})
	as.Equal(422, res.Code)
	as.assertPositions(a)
}

func (as *ActionSuite) Test_TasksReorder_DuplicateIDs() {
	a := as.positionedTask("Tarea A", false, 0)
	b := as.positionedTask("Tarea B", false, 1)

	res := as.HTML("/tasks/reorder").Post(url.Values{"block": {"pending"}, "ids": {b.ID.String(), b.ID.String()}})
	as.Equal(422, res.Code)
	as.assertPositions(a, b)
}

func (as *ActionSuite) Test_TasksReorder_Mismatch() {
	a := as.positionedTask("Tarea A", false, 0)
	b := as.positionedTask("Tarea B", false, 1)
	done := as.positionedTask("Tarea hecha", true, 0)

	res := as.HTML("/tasks/reorder").Post(url.Values{"block": {"pending"}, "ids": {b.ID.String()}})
	as.Equal(409, res.Code)
	as.assertPositions(a, b)

	res = as.HTML("/tasks/reorder").Post(url.Values{"block": {"pending"}, "ids": {b.ID.String(), done.ID.String()}})
	as.Equal(409, res.Code)
	as.assertPositions(a, b)
	as.assertPositions(done)
}
