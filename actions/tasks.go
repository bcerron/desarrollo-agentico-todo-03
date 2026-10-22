package actions

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"todo/models"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v6"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"
)

// TasksIndex muestra la lista de tareas y el formulario de creación.
func TasksIndex(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)

	tasks := models.Tasks{}
	if err := tasks.LoadOrdered(tx); err != nil {
		return err
	}

	c.Set("tasks", tasks)
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
	task.DueDate = parseDueDate(c.Request().FormValue("due_date"))

	position, err := models.NextPosition(tx)
	if err != nil {
		return err
	}
	task.Position = position

	verrs, err := tx.ValidateAndCreate(task)
	if err != nil {
		return err
	}

	if verrs.HasAny() {
		tasks := models.Tasks{}
		if err := tasks.LoadOrdered(tx); err != nil {
			return err
		}

		c.Set("tasks", tasks)
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
	task.DueDate = parseDueDate(c.Request().FormValue("due_date"))

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
	if err := tx.Update(task); err != nil {
		return err
	}

	return c.Redirect(http.StatusSeeOther, "/")
}

// TasksReorder guarda el orden de la lista. Recibe en "ids" el conjunto completo
// de tareas en el orden deseado; si no coincide con el existente responde 409.
func TasksReorder(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)

	if err := c.Request().ParseForm(); err != nil {
		return c.Error(http.StatusBadRequest, err)
	}
	ids := c.Request().Form["ids"]

	tasks := models.Tasks{}
	if err := tasks.LoadOrdered(tx); err != nil {
		return err
	}

	if len(ids) != len(tasks) {
		return c.Error(http.StatusConflict, errors.New("la lista de tareas ha cambiado"))
	}
	byID := make(map[uuid.UUID]*models.Task, len(tasks))
	for i := range tasks {
		byID[tasks[i].ID] = &tasks[i]
	}
	ordered := make([]*models.Task, 0, len(ids))
	for _, raw := range ids {
		id, err := uuid.FromString(raw)
		if err != nil {
			return c.Error(http.StatusConflict, errors.New("identificador de tarea no válido"))
		}
		task, ok := byID[id]
		if !ok {
			return c.Error(http.StatusConflict, errors.New("la lista de tareas ha cambiado"))
		}
		delete(byID, id)
		ordered = append(ordered, task)
	}

	for i, task := range ordered {
		task.Position = i + 1
		if err := tx.UpdateColumns(task, "position"); err != nil {
			return err
		}
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

// parseDueDate convierte una cadena "YYYY-MM-DD" en nulls.Time.
// Si la cadena está vacía o no es válida, devuelve un nulls.Time sin valor.
func parseDueDate(raw string) nulls.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nulls.Time{}
	}
	t, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return nulls.Time{}
	}
	return nulls.NewTime(t)
}
