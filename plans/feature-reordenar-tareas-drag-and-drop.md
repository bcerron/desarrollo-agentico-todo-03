# Feature: reordenar la lista de tareas con drag and drop

## Encargo y criterios de aceptación

Encargo literal: "Añadir dragon and dropp a la lista de TODOs para poder reordenarla".

Interpretación: el usuario puede arrastrar una tarea a otra posición de la lista y el nuevo orden **se conserva al recargar**.

Criterios de aceptación (explícitos = E, inferidos = I):

1. (E) Arrastrar una tarea de `.task-list` y soltarla en otra posición cambia su posición visualmente.
2. (I) El orden se guarda en PostgreSQL; tras recargar `/` las tareas salen en el orden elegido (lección del bug anterior, `plans/bug-toggle-no-persiste.md`: no basta con cambiar la UI).
3. (I) Una tarea nueva se añade **al final** de la lista (hoy es así por `created_at asc`).
4. (I) Completar, editar o eliminar una tarea no cambia el orden relativo del resto.
5. (I) Hay una alternativa por teclado al arrastre (accesibilidad; el resto de la UI ya usa `aria-*` y `sr-only`).
6. (I) Si el servidor rechaza el reordenado, la UI vuelve al orden real (recarga) en vez de quedar desincronizada.
7. (I) Sin JavaScript la lista sigue funcionando (solo se pierde el reordenado).

## Comportamiento actual y propuesta

Actual:

- `TasksIndex` y la rama de error de `TasksCreate` (`actions/tasks.go`) cargan con `tx.Order("created_at asc, id asc").All(&tasks)`. No existe columna de orden.
- `templates/tasks/index.plush.html` pinta `<ul class="task-list">` con un `<li class="task">` por tarea. `public/assets/app.js` solo gestiona `data-confirm` y el toggle optimista (con `fetch` POST + `FormData`, que lleva `authenticity_token` y `_method`).
- Assets, templates y migraciones van con `go:embed`: cualquier cambio exige recompilar.

Propuesta (cambio vertical mínimo):

- Nueva columna `position integer` en `tasks`; la lista se ordena por `position asc, created_at asc, id asc`.
- Nuevo endpoint `PUT /tasks/reorder` que recibe la lista completa de IDs en el orden deseado y reescribe `position` en una única transacción (la que ya abre `popmw.Transaction`).
- JS con la API nativa HTML5 Drag and Drop sobre los `<li>` (sin dependencias nuevas; la app no usa CDN ni bundler) más botones de teclado en un asa de arrastre.

## Alcance

Incluido:

- Migración con columna `position` y relleno inicial según el orden actual.
- Ordenación en `TasksIndex` y `TasksCreate` (rama de error), y asignación de `position` al crear.
- Acción y ruta `TasksReorder`.
- Asa de arrastre en cada tarea, JS de arrastre y de movimiento por teclado, estilos.
- Tests de modelo y de acción.
- Actualizar la query-string de versión de assets (`?v=3` → `?v=4`) en `templates/application.plush.html`, porque `app.js` y `app.css` cambian y el navegador los cachea.

Excluido:

- Reordenar en la pantalla de edición.
- Soporte de arrastre táctil (HTML5 DnD no funciona en la mayoría de navegadores móviles; ver riesgos). El movimiento por teclado/botones cubre ese caso.
- Filtros, agrupación por completadas, usuarios o listas múltiples.
- Cambiar la semántica de toggle/edición/borrado.

## Diseño por capas

### Persistencia y migraciones

- Nueva migración `migrations/20260902000000_add_position_to_tasks.up.fizz` / `.down.fizz` (formato `.fizz`, como el esquema existente; el prefijo debe ordenarse después de `20260901000001`).
  - `up`: `add_column("tasks", "position", "integer", {"default": 0})` y después un `raw(...)` que rellene con el orden actual:
    ```sql
    UPDATE tasks SET position = t.rn
    FROM (SELECT id, ROW_NUMBER() OVER (ORDER BY created_at, id) AS rn FROM tasks) t
    WHERE tasks.id = t.id
    ```
  - `down`: `drop_column("tasks", "position")`.
- Se mantiene `default 0` y `NOT NULL` implícito por default; no se añade índice (lista pequeña, sin paginación). Las filas de `fixtures/tasks.toml` y el seed no definen `position`, por eso el desempate `created_at, id` es obligatorio en el `ORDER BY`.
- El seed `20260901000001_seed_tasks` se ejecuta antes de la nueva migración, que renumera esas filas; no hay que tocarlo.

### Modelo (`models/task.go`)

- Añadir `Position int \`json:"position" db:"position" form:"-"\`` a `Task`.
- Añadir helper de consulta compartido para no duplicar el `Order` en tres sitios, p. ej. `func (ts *Tasks) LoadOrdered(tx *pop.Connection) error` (o función `ListTasks(tx)`), con `tx.Order("position asc, created_at asc, id asc").All(ts)`.
- Añadir `func NextPosition(tx *pop.Connection) (int, error)` que devuelva `COALESCE(MAX(position), 0) + 1` (`tx.RawQuery(...).First(&n)` o equivalente con pop).

### Acción y ruta

- `actions/app.go`: registrar `app.PUT("/tasks/reorder", TasksReorder)` **antes** de `app.PUT("/tasks/{task_id}", ...)` para que `reorder` no se interprete como `task_id` (el router de Buffalo/gorilla resuelve por orden de registro).
- `actions/tasks.go`:
  - `TasksIndex` y `TasksCreate` usan el helper de ordenación. `TasksCreate` asigna `task.Position` con `NextPosition` antes de `ValidateAndCreate`.
  - `TasksReorder`: lee `ids` (lista de UUID, formato `ids[]`/campo repetido `ids` vía `c.Request().ParseForm()`; el JS enviará `FormData`/`URLSearchParams`, igual que el toggle). Reglas:
    1. Carga todas las tareas; si la lista de IDs no es **exactamente** el conjunto de tareas existentes (IDs inválidos, duplicados, faltantes o sobrantes) → `409 Conflict` sin modificar nada. Cubre pestañas desactualizadas (otra pestaña añadió/borró tareas).
    2. Asigna `position = índice+1` según el orden recibido y persiste con `tx.UpdateColumns("position")` (no `tx.Update`, para no tocar `title`/`completed` ni `updated_at` de forma innecesaria) o con una sentencia `UPDATE` por tarea dentro de la misma transacción.
    3. Responde `204 No Content` (`c.Render(http.StatusNoContent, nil)` / `c.Response().WriteHeader`). Las peticiones sin JS no existen para esta ruta.
  - Los errores de BD se devuelven tal cual (`return err`), como en el resto de handlers.
- CSRF: `csrf.New` acepta el token en cabecera `X-CSRF-Token` o en el campo `authenticity_token`; el JS reutilizará el `authenticity_token` que ya está en el DOM (campo oculto de los formularios) y el `_method=PUT` en el cuerpo con `POST`, igual que hace el toggle, para no depender de que el navegador permita `PUT` directo con form-encoded. **Verificar** en la implementación que el método override funciona con la ruta `PUT /tasks/reorder` (misma mecánica que `Test_TasksToggle_PostWithMethodOverride`).

### Renderizado (`templates/tasks/index.plush.html`)

- En cada `<li class="task">` añadir `data-task-id="<%= t.ID %>"` y `draggable="true"` solo vía JS (para que sin JS no haya arrastre ni asa muerta). El `<ul>` recibe `data-reorder-url="/tasks/reorder"`.
- Insertar al inicio del `<li>` un asa: `<button type="button" class="drag-handle" data-drag-handle aria-label="Reordenar tarea: <%= t.Title %>. Usa Alt + flecha arriba o abajo para moverla." title="Arrastrar para reordenar">` con icono SVG de seis puntos (mismo estilo de iconos que el resto: `stroke="currentColor"`, `aria-hidden="true"`). Debe ocultarse si no hay JS (clase `hidden` por defecto, el JS la muestra) o, más simple, dejarse siempre visible; decidir en la implementación y documentarlo en el PR. Recomendación: ocultar por defecto y mostrar con `.js .drag-handle` añadiendo la clase `js` a `document.documentElement` desde `app.js`.
- Añadir una región `aria-live="polite"` `sr-only` (p. ej. `<p id="reorder-status" class="sr-only" aria-live="polite">`) para anunciar "Tarea movida a la posición N de M".
- Con 0 o 1 tareas el asa no tiene utilidad: no renderizarla si `len(tasks) < 2`.

### JavaScript (`public/assets/app.js`)

Mantener el estilo del archivo (ES5, `var`, listeners delegados sobre `document`, comentarios en español). Añadir:

1. Arrastre con ratón: en `dragstart` (sobre un `.task`), guardar el elemento arrastrado y marcarlo con clase `task-dragging`; en `dragover` sobre otro `.task`, `preventDefault()` y mover el elemento arrastrado antes o después del objetivo según `event.clientY` frente al punto medio del objetivo; en `drop`/`dragend`, limpiar clases y, si el orden cambió respecto al inicial, enviar la petición.
   - Solo iniciar arrastre desde el asa: `draggable` se activa en `mousedown` sobre `[data-drag-handle]` y se desactiva en `dragend`/`mouseup`, para no interferir con seleccionar texto ni con los botones/enlaces de la tarea.
2. Teclado: en `keydown` sobre el asa, `Alt+ArrowUp` / `Alt+ArrowDown` mueven el `<li>` una posición, devuelven el foco al asa y envían la petición; actualizan la región `aria-live`.
3. Guardado: `fetch(url, { method: "POST", body })` con `URLSearchParams` que incluye `authenticity_token` (leído del primer `input[name=authenticity_token]`), `_method=PUT` y un `ids` por cada `.task` en el orden del DOM. Si `!response.ok` o hay error de red, `window.location.reload()` para recuperar el orden real.
4. Evitar peticiones concurrentes desordenadas: serializar guardados (si hay uno en curso, encolar solo el último estado) o ignorar el arrastre hasta que termine; la opción más simple (bloquear con una bandera `saving`) es suficiente.

### CSS (`public/assets/app.css`)

- Estilos para `.drag-handle` (botón sin borde, cursor `grab`, color atenuado, `:hover`/`:focus-visible` coherentes con `.icon-action`), `.task-dragging` (opacidad reducida) y un indicador de destino opcional. Respetar el breakpoint móvil existente (`@media` en torno a las líneas 396-480): en pantallas pequeñas el asa debe seguir visible y con área táctil suficiente.
- Reutilizar variables/colores ya definidos en `app.css`; no introducir paleta nueva.

## Archivos relevantes

| Ruta | Responsabilidad |
|---|---|
| `migrations/20260902000000_add_position_to_tasks.{up,down}.fizz` (nuevos) | Columna `position` y relleno inicial |
| `models/task.go` | Campo `Position`, consulta ordenada, `NextPosition` |
| `models/task_test.go` | Tests de orden y `NextPosition` |
| `actions/app.go` | Ruta `PUT /tasks/reorder` (antes de `/tasks/{task_id}`) |
| `actions/tasks.go` | `TasksReorder`; orden y posición en `TasksIndex`/`TasksCreate` |
| `actions/tasks_test.go` | Tests de reordenado y de orden en el índice |
| `fixtures/tasks.toml` | Opcional: añadir `position` a las tres filas para que el escenario sea determinista |
| `templates/tasks/index.plush.html` | `data-task-id`, asa, región `aria-live`, `data-reorder-url` |
| `templates/application.plush.html` | Subir versión de assets (`?v=4`) |
| `public/assets/app.js` | Arrastre, teclado y guardado |
| `public/assets/app.css` | Estilos del asa y del estado de arrastre |
| `CLAUDE.md` / `README.md` | Mencionar la columna y la ruta solo si ya documentan rutas (revisar antes de tocar) |

## Implementación paso a paso

1. Crear la migración `.up.fizz`/`.down.fizz` y aplicarla en dev (`go run ./cmd/app` migra al arrancar) y en test (`GO_ENV=test soda migrate -e test`). Confirmar con `psql` que las tres filas del seed quedan con `position` 1, 2, 3 en orden de `created_at`.
2. `models/task.go`: añadir `Position`, el helper de consulta ordenada y `NextPosition`.
3. Escribir primero los tests de modelo y de acción (ver sección siguiente) y ver que fallan.
4. `actions/tasks.go`: sustituir los dos `Order("created_at asc, id asc")` por el helper; asignar `Position` en `TasksCreate`; implementar `TasksReorder` con las reglas anteriores.
5. `actions/app.go`: registrar la ruta `PUT /tasks/reorder` antes de la de `{task_id}`.
6. Plantilla: atributos `data-*`, asa y región `aria-live`; subir `?v=3` a `?v=4` en `application.plush.html` (CSS y JS).
7. `app.js`: implementar arrastre, teclado y guardado con recarga ante error. Mantener intacto el comportamiento de `data-confirm` y del toggle.
8. `app.css`: estilos del asa y del arrastre, incluida la vista móvil.
9. Ejecutar la validación completa y la comprobación manual descrita abajo.

## Tests y validación

Tests de modelo (`models/task_test.go`, suite `ModelSuite`):

- Crear tareas con `Position` 2, 1, 3 y comprobar que `LoadOrdered` las devuelve en orden 1, 2, 3.
- Con `Position` igual, el desempate es `created_at` y luego `id`.
- `NextPosition` devuelve 1 en tabla vacía y `max+1` con datos.

Tests de acción (`actions/tasks_test.go`, suite `ActionSuite`; usar `as.HTML("/tasks/reorder").Put(url.Values{"ids": {...}})` como en los tests de toggle):

- `Test_TasksIndex_OrdersByPosition`: dos tareas creadas en un orden y con `Position` invertida aparecen en el HTML en el orden de `Position` (comparar `strings.Index` de los títulos).
- `Test_TasksCreate_AppendsAtEnd`: tras crear una tarea con otras existentes, su `Position` es la mayor.
- `Test_TasksReorder_PersistsOrder`: con tres tareas, enviar los IDs en otro orden → 204 y `Position` 1..3 según el orden enviado; un `GET /` posterior refleja el nuevo orden.
- `Test_TasksReorder_DoesNotChangeOtherFields`: `Title` y `Completed` intactos tras reordenar.
- `Test_TasksReorder_MissingTask_Conflict`: lista incompleta → 409 y orden sin cambios.
- `Test_TasksReorder_DuplicateID_Conflict` y `Test_TasksReorder_InvalidID_Conflict` (UUID mal formado o inexistente) → 409 sin cambios.
- `Test_TasksReorder_PostWithMethodOverride`: `POST` con `_method=PUT` también funciona (es lo que envía el JS).
- `Test_TasksReorder_Route_DoesNotShadowUpdate`: `PUT /tasks/<uuid>` sigue yendo a `TasksUpdate` y `PUT /tasks/reorder` no devuelve 404.
- Actualizar `fixtures/tasks.toml` con `position` si algún test existente depende del orden de inserción.

Los tests de JS no tienen infraestructura en el repo; no se añade framework. Cubrirlo con verificación manual y, si hay entorno, una pasada con el skill `run`/navegador:

1. `docker compose up -d db` y `go run ./cmd/app`; abrir `http://localhost:3000`.
2. Arrastrar la tercera tarea a la primera posición → recargar → el orden se mantiene.
3. Con el teclado, foco en el asa y `Alt+↓`/`Alt+↑` → orden cambia, se anuncia y persiste.
4. Completar, editar y eliminar tareas no altera el orden; añadir una tarea nueva la deja al final.
5. Abrir dos pestañas, añadir una tarea en una, reordenar en la otra → la segunda recibe 409 y recarga mostrando el orden real.
6. Devtools: la petición de reorden devuelve 204 y no hay errores en consola.

Comandos de validación (los tests necesitan PostgreSQL con `todo_test`, `GO_ENV=test` y migraciones de test aplicadas con `soda migrate -e test`, ya que ahora incluyen la nueva migración):

```bash
gofmt -l .            # debe no listar nada
go vet ./...
go build ./...
go test -p 1 ./...    # -p 1 obligatorio: los paquetes comparten BD
```

## Riesgos y recuperación

- **Migración con datos existentes**: el relleno con `ROW_NUMBER()` conserva el orden visible actual. Antes de aplicarla en una BD con datos reales, hacer copia (`pg_dump`). Recuperación: `.down.fizz` elimina la columna; el orden vuelve a `created_at, id`, que es el comportamiento actual.
- **Tests sin migrar**: si la BD `todo_test` no tiene la columna, fallarán todas las suites; correr `soda migrate -e test` primero.
- **Ruta `reorder` vs `{task_id}`**: si se registra después, `PUT /tasks/reorder` iría a `TasksUpdate` y devolvería 404 al fallar `tx.Find`. El orden de registro y el test `Route_DoesNotShadowUpdate` lo cubren.
- **Concurrencia entre pestañas**: exigir el conjunto completo de IDs evita posiciones duplicadas o tareas "perdidas"; el coste es un 409 y una recarga ocasional. Alternativa descartada: aceptar subconjuntos, porque admite estados ambiguos.
- **Caché de assets**: sin subir `?v=`, un navegador con `app.js` antiguo no tendría el arrastre. Incluido en el plan.
- **Táctil**: HTML5 DnD no funciona bien en iOS/Android. Se acepta en este alcance (el teclado y las flechas del asa sirven como alternativa); si se necesita arrastre táctil, es una feature aparte con Pointer Events.
- **Interferencia con el toggle**: el listener `submit` actual no debe verse afectado; los eventos de arrastre son distintos. Comprobarlo manualmente.
- **Incertidumbres para el revisor**: (1) si se prefiere un orden por `position` con huecos (p. ej. incrementos de 1024) para reducir escrituras; el plan usa renumeración completa por simplicidad, válida para listas pequeñas; (2) si el asa debe ocultarse sin JS (recomendado) o dejarse siempre visible; (3) si se quiere soporte táctil ya, lo que cambiaría el diseño del JS.
