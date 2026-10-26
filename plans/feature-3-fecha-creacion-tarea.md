# Feature: Añadir fecha de creación de la tarea al listado

## Encargo y criterios de aceptación

**Encargo (issue #3):** Añade la fecha de creación de la tarea al lado del nombre de la misma en el listado.

**Criterios de aceptación:**

1. En el listado de tareas (`/`), cada fila muestra la fecha de creación (`created_at`) junto al título.
2. El formato es `DD/MM/YYYY`, consistente con la fecha límite ya existente.
3. La fecha se muestra siempre (toda tarea tiene `created_at`).
4. Las tareas existentes no requieren migración: `created_at` ya se rellena automáticamente por Pop.

**Inferencias:**

- "Al lado del nombre" → se coloca justo después del título, antes de la fecha límite si existe. Se usa un estilo sutil para no competir visualmente con el título.
- No se pide que sea editable ni filtrable; solo lectura en el listado.

## Comportamiento actual y propuesta

**Actual:** El listado muestra título, fecha límite (opcional) y estado. No hay indicación de cuándo se creó la tarea.

**Propuesta:** Añadir una etiqueta con la fecha de creación entre el título y la fecha límite, con un estilo similar pero distinguible (mismo tamaño, misma posición de fila).

## Alcance

**Incluido:**

- Renderizar `created_at` en `templates/tasks/index.plush.html`.
- Estilo CSS para la etiqueta de fecha de creación.
- Test que verifique que la fecha aparece en el HTML del listado.

**Excluido:**

- Migración de base de datos (el campo ya existe).
- Cambios en el modelo o las acciones.
- Fecha de creación en la vista de edición.
- Ordenación o filtrado por fecha de creación.

## Diseño por capas

### Template (`templates/tasks/index.plush.html`)

Añadir un `<span class="task-created-date">` después del `<span class="task-title">` y antes del bloque condicional de `task-due-date`:

```html
<span class="task-created-date"><%= t.CreatedAt.Format("02/01/2006") %></span>
```

### CSS (`public/assets/app.css`)

Estilo `.task-created-date` siguiendo el patrón de `.task-due-date`:

```css
.task-created-date {
  align-self: center;
  flex-shrink: 0;
  padding: 0.15rem 0.5rem;
  font-size: 0.75rem;
  color: var(--color-muted);
  white-space: nowrap;
}
```

No lleva fondo ni borde (a diferencia de due-date) para diferenciarla visualmente: es dato informativo, no una fecha de acción.

### Grid de la fila `.task`

Actualmente la fila tiene 4 columnas: `44px minmax(0,1fr) auto auto` (toggle, título, due-date/status, acciones). Con la fecha de creación se necesita una columna `auto` más. Ajustar a:

```
grid-template-columns: 44px minmax(0, 1fr) auto auto auto;
```

Y lo mismo en las variantes con drag-handle (`.js .task-list[data-reorder-url] .task`) y en la media query de 760px.

## Archivos relevantes

| Archivo | Responsabilidad |
|---|---|
| `templates/tasks/index.plush.html` | Renderiza la lista; añadir el `<span>` de fecha de creación |
| `public/assets/app.css` | Estilos; añadir `.task-created-date` y ajustar grids |
| `actions/tasks_test.go` | Tests; añadir test de que la fecha de creación se muestra |

## Implementación paso a paso

1. **`public/assets/app.css`** — Añadir la clase `.task-created-date`. Ajustar `grid-template-columns` en `.task`, `.js .task-list[data-reorder-url] .task`, y las dos variantes en `@media (max-width: 760px)` para incluir una columna `auto` adicional.

2. **`templates/tasks/index.plush.html`** — Después de `<span class="task-title">…</span>` (línea 38), añadir:
   ```html
   <span class="task-created-date"><%= t.CreatedAt.Format("02/01/2006") %></span>
   ```

3. **`actions/tasks_test.go`** — Añadir `Test_TasksIndex_ShowsCreatedDate`: crear una tarea, obtener `/`, verificar que el body contiene la fecha formateada en `DD/MM/YYYY`.

## Tests y validación

- `Test_TasksIndex_ShowsCreatedDate`: crea una tarea, hace GET `/`, verifica que el HTML contiene la fecha de creación formateada.
- `gofmt -l .` sin salida.
- `go vet ./...` OK.
- `go build ./...` OK.
- `GO_ENV=test go test -p 1 ./...` OK.

## Riesgos y recuperación

- **Riesgo:** la fecha de creación de los fixtures puede variar según el momento de ejecución (`<%= now() %>`). El test debe crear su propia tarea con fecha conocida o formatear `time.Now()` del momento del test.
- **Mitigación:** en el test, crear la tarea con `as.DB.Create` y comprobar que la respuesta contiene `time.Now().UTC().Format("02/01/2006")` (la fecha del día del test, que coincidirá con `created_at` asignado por Pop).
