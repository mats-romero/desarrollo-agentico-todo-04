package models

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gobuffalo/pop/v6"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"
)

// TitleMaxLength es el número máximo de caracteres permitido en el título.
const TitleMaxLength = 200

// taskListOrder ordena por bloque (pendientes primero) y, dentro de cada
// bloque, por la posición guardada. Ante posiciones iguales, la más reciente
// queda arriba.
const taskListOrder = "completed asc, position asc, created_at desc, id asc"

// ErrReorderMismatch indica que los ids enviados no coinciden con las tareas
// del bloque.
var ErrReorderMismatch = errors.New("los ids no coinciden con las tareas del bloque")

// Task representa una tarea de la lista.
type Task struct {
	ID        uuid.UUID `json:"id" db:"id" form:"-"`
	Title     string    `json:"title" db:"title" form:"title"`
	Completed bool      `json:"completed" db:"completed" form:"-"`
	Position  int       `json:"position" db:"position" form:"-"`
	CreatedAt time.Time `json:"created_at" db:"created_at" form:"-"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at" form:"-"`
}

// Tasks es una colección de Task.
type Tasks []Task

// Validate se ejecuta en cada llamada a los métodos "pop.Validate*".
func (t *Task) Validate(tx *pop.Connection) (*validate.Errors, error) {
	verrs := validate.NewErrors()

	title := strings.TrimSpace(t.Title)
	if title == "" {
		verrs.Add("title", "El título es obligatorio.")
	} else if len([]rune(title)) > TitleMaxLength {
		verrs.Add("title", fmt.Sprintf("El título no puede superar los %d caracteres.", TitleMaxLength))
	}

	return verrs, nil
}

// LoadTasks carga todas las tareas: primero las pendientes y después las
// completadas, cada bloque ordenado por su posición.
func LoadTasks(tx *pop.Connection) (Tasks, error) {
	tasks := Tasks{}
	if err := tx.Order(taskListOrder).All(&tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

// Pending devuelve las tareas pendientes conservando el orden.
func (ts Tasks) Pending() Tasks {
	return ts.filter(false)
}

// Completed devuelve las tareas completadas conservando el orden.
func (ts Tasks) Completed() Tasks {
	return ts.filter(true)
}

func (ts Tasks) filter(completed bool) Tasks {
	out := Tasks{}
	for _, t := range ts {
		if t.Completed == completed {
			out = append(out, t)
		}
	}
	return out
}

// TopPosition devuelve la posición que coloca una tarea al principio del
// bloque indicado.
func TopPosition(tx *pop.Connection, completed bool) (int, error) {
	row := struct {
		Position int `db:"position"`
	}{}
	err := tx.RawQuery("SELECT COALESCE(MIN(position), 0) - 1 AS position FROM tasks WHERE completed = ?", completed).First(&row)
	if err != nil {
		return 0, err
	}
	return row.Position, nil
}

// ReorderTasks reescribe las posiciones del bloque como 0..n-1 siguiendo el
// orden de ids. Si los ids no coinciden exactamente con las tareas del bloque
// devuelve ErrReorderMismatch sin escribir nada. No modifica updated_at.
func ReorderTasks(tx *pop.Connection, completed bool, ids []uuid.UUID) error {
	tasks := Tasks{}
	if err := tx.Where("completed = ?", completed).All(&tasks); err != nil {
		return err
	}

	if len(ids) != len(tasks) {
		return ErrReorderMismatch
	}

	inBlock := make(map[uuid.UUID]bool, len(tasks))
	for _, t := range tasks {
		inBlock[t.ID] = true
	}

	seen := make(map[uuid.UUID]bool, len(ids))
	for _, id := range ids {
		if !inBlock[id] || seen[id] {
			return ErrReorderMismatch
		}
		seen[id] = true
	}

	for i, id := range ids {
		if err := tx.RawQuery("UPDATE tasks SET position = ? WHERE id = ?", i, id).Exec(); err != nil {
			return err
		}
	}

	return nil
}
