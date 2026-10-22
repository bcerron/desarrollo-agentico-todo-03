package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/gobuffalo/pop/v6"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"
)

// TitleMaxLength es el número máximo de caracteres permitido en el título.
const TitleMaxLength = 200

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

// orderedBy es el orden de la lista.
// El desempate por created_at e id es necesario porque position puede repetirse.
const orderedBy = "position asc, created_at asc, id asc"

// likeEscaper escapa los comodines de LIKE para que se busquen literalmente.
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// LoadOrdered carga todas las tareas en el orden de la lista.
func (ts *Tasks) LoadOrdered(tx *pop.Connection) error {
	return tx.Order(orderedBy).All(ts)
}

// LoadOrderedMatching carga, en el orden de la lista, las tareas cuyo título
// contiene query sin distinguir mayúsculas. Los comodines de LIKE se tratan literalmente.
func (ts *Tasks) LoadOrderedMatching(tx *pop.Connection, query string) error {
	pattern := "%" + likeEscaper.Replace(query) + "%"
	return tx.Where(`title ILIKE ? ESCAPE '\'`, pattern).Order(orderedBy).All(ts)
}

// NextPosition devuelve la posición que ocupa una tarea nueva al final de la lista.
func NextPosition(tx *pop.Connection) (int, error) {
	var next int
	if err := tx.RawQuery("SELECT COALESCE(MAX(position), 0) + 1 FROM tasks").First(&next); err != nil {
		return 0, err
	}
	return next, nil
}
