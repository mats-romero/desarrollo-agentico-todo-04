package actions

import (
	"errors"
	"net/http"
	"strings"

	"todo/models"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/pop/v6"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"
)

// setTaskList carga las tareas y deja en el contexto la lista completa y sus
// dos bloques (pendientes y completadas).
func setTaskList(c buffalo.Context, tx *pop.Connection) error {
	tasks, err := models.LoadTasks(tx)
	if err != nil {
		return err
	}

	c.Set("tasks", tasks)
	c.Set("pendingTasks", tasks.Pending())
	c.Set("completedTasks", tasks.Completed())
	return nil
}

// TasksIndex muestra la lista de tareas y el formulario de creación.
func TasksIndex(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)

	if err := setTaskList(c, tx); err != nil {
		return err
	}

	c.Set("task", &models.Task{})
	c.Set("errors", validate.NewErrors())
	return c.Render(http.StatusOK, r.HTML("tasks/index.plush.html"))
}

// TasksCreate crea una tarea nueva a partir del formulario.
func TasksCreate(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)

	task := &models.Task{}
	if err := c.Bind(task); err != nil {
		return err
	}
	task.Title = strings.TrimSpace(task.Title)

	position, err := models.TopPosition(tx, false)
	if err != nil {
		return err
	}
	task.Position = position

	verrs, err := tx.ValidateAndCreate(task)
	if err != nil {
		return err
	}

	if verrs.HasAny() {
		if err := setTaskList(c, tx); err != nil {
			return err
		}

		c.Set("task", task)
		c.Set("errors", verrs)
		return c.Render(http.StatusUnprocessableEntity, r.HTML("tasks/index.plush.html"))
	}

	c.Flash().Add("success", "Tarea creada.")
	return c.Redirect(http.StatusSeeOther, "/")
}

// TasksEdit muestra el formulario de edición de una tarea existente.
func TasksEdit(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)

	task := &models.Task{}
	if err := tx.Find(task, c.Param("task_id")); err != nil {
		return c.Error(http.StatusNotFound, err)
	}

	c.Set("task", task)
	c.Set("errors", validate.NewErrors())
	return c.Render(http.StatusOK, r.HTML("tasks/edit.plush.html"))
}

// TasksUpdate actualiza solo el título, conservando los demás datos de la tarea.
func TasksUpdate(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)

	task := &models.Task{}
	if err := tx.Find(task, c.Param("task_id")); err != nil {
		return c.Error(http.StatusNotFound, err)
	}

	input := struct {
		Title string `form:"title" json:"title"`
	}{}
	if err := c.Bind(&input); err != nil {
		return err
	}
	task.Title = strings.TrimSpace(input.Title)

	verrs, err := tx.ValidateAndUpdate(task)
	if err != nil {
		return err
	}
	if verrs.HasAny() {
		c.Set("task", task)
		c.Set("errors", verrs)
		return c.Render(http.StatusUnprocessableEntity, r.HTML("tasks/edit.plush.html"))
	}

	c.Flash().Add("success", "Tarea actualizada.")
	return c.Redirect(http.StatusSeeOther, "/")
}

// TasksToggle alterna una tarea entre completada y pendiente.
func TasksToggle(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)

	task := &models.Task{}
	if err := tx.Find(task, c.Param("task_id")); err != nil {
		return c.Error(http.StatusNotFound, err)
	}

	task.Completed = !task.Completed
	position, err := models.TopPosition(tx, task.Completed)
	if err != nil {
		return err
	}
	task.Position = position
	if err := tx.Update(task); err != nil {
		return err
	}

	return c.Redirect(http.StatusSeeOther, "/")
}

// TasksReorder guarda el nuevo orden de las tareas de un bloque.
func TasksReorder(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)

	req := c.Request()
	if err := req.ParseForm(); err != nil {
		return c.Error(http.StatusUnprocessableEntity, err)
	}

	block := req.PostForm.Get("block")
	if block != "pending" && block != "completed" {
		return c.Error(http.StatusUnprocessableEntity, errors.New("bloque no válido"))
	}

	rawIDs := req.PostForm["ids"]
	ids := make([]uuid.UUID, 0, len(rawIDs))
	seen := make(map[uuid.UUID]bool, len(rawIDs))
	for _, raw := range rawIDs {
		id, err := uuid.FromString(raw)
		if err != nil {
			return c.Error(http.StatusUnprocessableEntity, err)
		}
		if seen[id] {
			return c.Error(http.StatusUnprocessableEntity, errors.New("ids repetidos"))
		}
		seen[id] = true
		ids = append(ids, id)
	}

	if err := models.ReorderTasks(tx, block == "completed", ids); err != nil {
		if errors.Is(err, models.ErrReorderMismatch) {
			return c.Error(http.StatusConflict, err)
		}
		return err
	}

	return c.Render(http.StatusNoContent, nil)
}

// TasksDestroy elimina una tarea.
func TasksDestroy(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)

	task := &models.Task{}
	if err := tx.Find(task, c.Param("task_id")); err != nil {
		return c.Error(http.StatusNotFound, err)
	}

	if err := tx.Destroy(task); err != nil {
		return err
	}

	c.Flash().Add("success", "Tarea eliminada.")
	return c.Redirect(http.StatusSeeOther, "/")
}
