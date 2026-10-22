# Feature: Añadir fecha límite a las tareas

## Encargo y criterios de aceptación

**Encargo:** El usuario quiere poder asignar una fecha límite (due date) opcional a cada tarea.

**Criterios de aceptación (explícitos e inferidos):**

1. Al crear una tarea se puede indicar opcionalmente una fecha límite.
2. Al editar una tarea se puede añadir, cambiar o quitar la fecha límite.
3. La fecha límite se muestra junto al título en la lista de tareas.
4. Las tareas vencidas (fecha límite anterior a hoy y no completadas) se distinguen visualmente.
5. La fecha límite es opcional: las tareas existentes siguen funcionando sin ella.
6. Toggle, reorder y destroy no se ven afectados.

**Inferencias:** No se pide ordenar por fecha límite, ni notificaciones, ni hora —solo fecha (date). La columna es nullable.

## Comportamiento actual y propuesta

**Actual:** `Task` tiene `ID`, `Title`, `Completed`, `Position`, `CreatedAt`, `UpdatedAt`. No hay concepto de fecha límite.

**Propuesta:** Añadir campo `DueDate nulls.Time` (columna `due_date timestamp NULL`). El formulario de creación y edición incluyen un `<input type="date">`. La lista muestra la fecha formateada y una clase CSS `task-overdue` cuando la tarea está vencida y no completada.

## Alcance

**Incluido:**
- Migración para añadir columna `due_date`
- Campo `DueDate` en el modelo `Task`
- Binding del campo en `TasksCreate` y `TasksUpdate`
- Input de fecha en los templates `index.plush.html` (formulario de creación) y `edit.plush.html`
- Mostrar la fecha límite en cada tarea de la lista
- Estilo CSS para tareas vencidas
- Tests de modelo y acciones para el nuevo campo
- Actualizar fixture para incluir `due_date`

**Excluido:**
- Ordenar la lista por fecha límite
- Notificaciones o recordatorios
- Componente de hora (solo fecha)
- Validación de fecha mínima (se permite cualquier fecha)

## Diseño por capas

### Persistencia (migración)

Nuevo archivo `migrations/20261022000000_add_due_date_to_tasks.up.fizz`:
```
add_column("tasks", "due_date", "timestamp", {"null": true})
```

Y `migrations/20261022000000_add_due_date_to_tasks.down.fizz`:
```
drop_column("tasks", "due_date")
```

### Modelo (`models/task.go`)

Añadir campo al struct:
```go
DueDate nulls.Time `json:"due_date" db:"due_date" form:"due_date"`
```

Import `github.com/gobuffalo/nulls`. No se necesita validación adicional: el campo es opcional y cualquier fecha es válida.

Añadir método helper `IsOverdue()` que devuelve `true` si `DueDate` tiene valor, la tarea no está completada y la fecha es anterior a hoy (truncando a día).

### Acciones (`actions/tasks.go`)

**`TasksCreate`:** El `c.Bind(task)` ya recogerá `due_date` del formulario porque el tag `form:"due_date"` está en el struct. No hay cambio en el handler salvo que el campo viaja con el bind.

**`TasksUpdate`:** Actualmente solo acepta `Title` con un struct anónimo. Hay que añadir `DueDate` al struct de input y asignarlo al task:
```go
input := struct {
    Title   string     `form:"title" json:"title"`
    DueDate nulls.Time `form:"due_date" json:"due_date"`
}{}
// ...
task.Title = strings.TrimSpace(input.Title)
task.DueDate = input.DueDate
```

### Templates

**`index.plush.html`:** Después del `<span class="task-title">`, mostrar la fecha si existe. Añadir `task-overdue` a la clase del `<li>` cuando aplique. En el formulario de nueva tarea, añadir `<input type="date" name="due_date">`.

**`edit.plush.html`:** Añadir campo `<input type="date">` con label "Fecha límite" después del campo de título. Prepoblar con el valor existente formateado como `YYYY-MM-DD`. Actualizar el hint.

### CSS (`public/assets/app.css`)

- `.task-due-date`: estilo para la fecha mostrada (tamaño reducido, color muted).
- `.task-overdue .task-due-date`: color de alerta (rojo/danger).
- Ajustar grid de `.task` para acomodar la fecha (o mostrarla inline después del título).

## Archivos relevantes

| Archivo | Responsabilidad |
|---|---|
| `models/task.go` | Struct `Task`, método `IsOverdue()` |
| `actions/tasks.go` | Handlers `TasksCreate`, `TasksUpdate` |
| `templates/tasks/index.plush.html` | Lista + formulario de creación |
| `templates/tasks/edit.plush.html` | Formulario de edición |
| `public/assets/app.css` | Estilos |
| `migrations/20261022000000_add_due_date_to_tasks.up.fizz` | Migración up |
| `migrations/20261022000000_add_due_date_to_tasks.down.fizz` | Migración down |
| `fixtures/tasks.toml` | Fixture de tests |
| `models/task_test.go` | Tests de modelo |
| `actions/tasks_test.go` | Tests de acciones |

## Implementación paso a paso

1. **Migración:** Crear `migrations/20261022000000_add_due_date_to_tasks.up.fizz` y `.down.fizz`.

2. **Modelo:** En `models/task.go`:
   - Importar `github.com/gobuffalo/nulls`.
   - Añadir campo `DueDate nulls.Time` al struct.
   - Añadir método `IsOverdue() bool`.

3. **Fixture:** En `fixtures/tasks.toml`, añadir `due_date` a las filas existentes. Una con fecha pasada (para testear overdue), otra con fecha futura, una sin fecha.

4. **Acción TasksUpdate:** En `actions/tasks.go`, ampliar el struct anónimo de input para incluir `DueDate` y asignar `task.DueDate = input.DueDate`.

5. **Template index:** En `templates/tasks/index.plush.html`:
   - Añadir clase `task-overdue` al `<li>` cuando `t.IsOverdue()`.
   - Mostrar `<span class="task-due-date">` con la fecha formateada después de `.task-title`.
   - En el formulario de nueva tarea, añadir input date entre el input de título y el botón.

6. **Template edit:** En `templates/tasks/edit.plush.html`:
   - Añadir label + input date para fecha límite.
   - Prepoblar si `task.DueDate.Valid`.
   - Actualizar hint.

7. **CSS:** En `public/assets/app.css`:
   - Estilo `.task-due-date`.
   - Estilo `.task-overdue .task-due-date`.
   - Ajustar grid si es necesario o mantener la fecha inline con el título.

8. **Tests de modelo** (`models/task_test.go`):
   - `Test_Task_Create_WithDueDate`: crear tarea con fecha y verificar persistencia.
   - `Test_Task_Create_WithoutDueDate`: crear tarea sin fecha (sigue funcionando).
   - `Test_Task_IsOverdue`: verificar lógica de vencimiento.

9. **Tests de acciones** (`actions/tasks_test.go`):
   - `Test_TasksCreate_WithDueDate`: POST con fecha, verificar persistencia.
   - `Test_TasksCreate_WithoutDueDate`: POST sin fecha, campo queda null.
   - `Test_TasksUpdate_ChangesDueDate`: PUT con nueva fecha.
   - `Test_TasksUpdate_ClearsDueDate`: PUT sin fecha borra la existente.
   - `Test_TasksIndex_ShowsDueDate`: verificar que la fecha se renderiza en el HTML.
   - `Test_TasksIndex_ShowsOverdueClass`: verificar clase `task-overdue`.

10. **Comprobaciones:** `gofmt -l .`, `go vet ./...`, `go build ./...`, `GO_ENV=test go test -p 1 ./...`.

## Tests y validación

- Los tests de modelo verifican persistencia del campo nullable y la lógica de `IsOverdue()`.
- Los tests de acciones verifican el binding en create/update y el renderizado en los templates.
- Los tests existentes deben seguir pasando sin cambios (el campo es nullable y opcional).
- Prerequisito: PostgreSQL con `todo_test` migrada (`GO_ENV=test soda migrate -e test`). Las migraciones embebidas se aplican automáticamente al arrancar, pero los tests las necesitan vía soda.

## Riesgos y recuperación

| Riesgo | Mitigación |
|---|---|
| `nulls.Time` no parsea bien el input `date` del formulario HTML | `formam` (el binder de Buffalo) soporta `nulls.Time`; si falla, parsear manualmente en el handler |
| La migración embebida no se recarga en tests | Ejecutar `soda migrate -e test` antes de los tests; CI ya lo hace |
| Zona horaria: `type="date"` envía `YYYY-MM-DD` sin hora | Almacenar como timestamp con hora 00:00 UTC; comparar `IsOverdue()` truncando a día en UTC |
| Los tests de `IsOverdue` dependen de la fecha actual | Usar fechas claramente pasadas (2020) y futuras (2099) para evitar flakiness |
