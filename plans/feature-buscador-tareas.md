# Feature: buscador de tareas

## Encargo y criterios de aceptación

Encargo literal: "Añade un buscador de tareas".

Interpretación: en la página de la lista (`/`) hay un campo para buscar tareas por su título; al buscar, la lista muestra solo las tareas que coinciden.

Criterios de aceptación (explícitos = E, inferidos = I):

1. (E) La página `/` muestra un formulario de búsqueda con un campo de texto etiquetado.
2. (I) La búsqueda filtra por **título**, sin distinguir mayúsculas/minúsculas, por coincidencia parcial (subcadena). Es el único campo de texto de la tarea.
3. (I) La búsqueda funciona sin JavaScript: el formulario hace `GET /?q=<texto>` y el filtrado ocurre en el servidor. La URL es compartible y el navegador conserva el historial.
4. (I) Los resultados mantienen el orden de la lista (`position`, como `LoadOrdered`), incluidas tareas completadas.
5. (I) Con búsqueda activa, el campo conserva el texto buscado, la cabecera indica cuántos resultados hay para ese texto y aparece un enlace "Limpiar búsqueda" que vuelve a `/`.
6. (I) Si nada coincide se muestra un mensaje específico ("Ninguna tarea coincide con «…»"), distinto del estado vacío de una lista sin tareas.
7. (I) `q` vacío o solo con espacios equivale a no buscar (lista completa).
8. (I) Los caracteres `%`, `_` y `\` del texto buscado se tratan literalmente (no como comodines SQL), y el texto se pasa como parámetro (sin inyección SQL). El texto se escapa en el HTML (Plush lo hace por defecto).
9. (I) Mientras hay búsqueda activa **no se ofrece reordenar** (sin asas de arrastre ni `data-reorder-url`): `TasksReorder` exige la lista completa de IDs y respondería 409 con un subconjunto.

## Comportamiento actual y propuesta

Actual:

- `GET /` → `TasksIndex` (`actions/tasks.go`) carga todas las tareas con `models.Tasks.LoadOrdered` y renderiza `tasks/index.plush.html`. No existe búsqueda en ninguna capa.
- La plantilla pinta el contador "N tareas en tu lista", el estado vacío "No hay tareas todavía…" y, si hay más de una tarea, asas de arrastre y `data-reorder-url` en `.task-list`, que activan el reordenado de `public/assets/app.js`.
- La rama de error de `TasksCreate` re-renderiza la misma plantilla con la lista completa.

Propuesta (cambio vertical mínimo, sin JS nuevo ni migraciones):

- Nuevo método de modelo `Tasks.LoadOrderedMatching(tx, query)` que reutiliza el orden de `LoadOrdered` y añade `WHERE title ILIKE ? ESCAPE '\'` con el patrón escapado.
- `TasksIndex` lee `q`, lo normaliza (trim, recorte a `TitleMaxLength` runas) y usa el método nuevo cuando no está vacío. Expone `query` a la plantilla; `TasksCreate` (rama de error) también expone `query` vacío para que la plantilla no falle.
- Formulario `GET /` en la plantilla, entre la cabecera y la lista, y variantes de contador/estado vacío/reordenado según haya búsqueda.

## Alcance

Incluido:

- Método de modelo con escape de comodines y test de modelo.
- Lectura de `q` en `TasksIndex`; variable `query` en ambas rutas de render de `tasks/index`.
- Formulario de búsqueda, contador de resultados, enlace de limpiar, estado sin resultados, desactivar reordenado con búsqueda activa.
- Estilos mínimos para el formulario reutilizando los existentes.
- Tests de acción.

Excluido:

- Búsqueda en vivo con JavaScript mientras se escribe.
- Conservar la búsqueda tras completar, editar o eliminar (esas acciones siguen redirigiendo a `/`).
- Filtros por estado (completada/pendiente), índices (`pg_trgm`) o búsqueda de texto completo. Con listas de tareas personales un `ILIKE` secuencial es suficiente.
- Cambios en `TasksReorder`.

## Diseño por capas

### Modelo (`models/task.go`)

```go
// LoadOrderedMatching carga, en el orden de la lista, las tareas cuyo título
// contiene query sin distinguir mayúsculas. Los comodines de LIKE se tratan literalmente.
func (ts *Tasks) LoadOrderedMatching(tx *pop.Connection, query string) error {
	pattern := "%" + likeEscaper.Replace(query) + "%"
	return tx.Where(`title ILIKE ? ESCAPE '\'`, pattern).
		Order("position asc, created_at asc, id asc").
		All(ts)
}

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
```

- Extraer la cadena de orden a una constante no exportada (`orderedBy = "position asc, created_at asc, id asc"`) usada por ambos métodos, para no duplicarla. El comentario de desempate existente se conserva sobre la constante o sobre `LoadOrdered`.
- PostgreSQL usa `standard_conforming_strings = on` por defecto, así que `'\'` es una barra literal.

### Acción (`actions/tasks.go`)

En `TasksIndex`:

```go
query := searchQuery(c.Param("q"))

tasks := models.Tasks{}
var err error
if query == "" {
	err = tasks.LoadOrdered(tx)
} else {
	err = tasks.LoadOrderedMatching(tx, query)
}
if err != nil {
	return err
}

c.Set("query", query)
```

Helper en el mismo archivo:

```go
// searchQuery normaliza el texto de búsqueda: sin espacios en los extremos y
// como mucho TitleMaxLength caracteres, porque un título nunca es más largo.
func searchQuery(raw string) string
```

En la rama de error de `TasksCreate`, añadir `c.Set("query", "")` junto a los demás `c.Set`.

No hay ruta nueva: se reutiliza `GET /`.

### Renderizado (`templates/tasks/index.plush.html`)

- Cabecera: si `query != ""`, el contador dice `N resultado(s) para «<query>»` (singular "resultado" con 1); si no, el texto actual.
- Formulario tras `</header>`:

```html
<form action="/" method="GET" class="search-form" role="search">
  <label for="task-search" class="sr-only">Buscar tareas</label>
  <input id="task-search" type="search" name="q" value="<%= query %>" placeholder="Buscar tareas…" maxlength="200" autocomplete="off">
  <button type="submit" class="button">Buscar</button>
  <%= if (query != "") { %><a href="/" class="search-clear">Limpiar búsqueda</a><% } %>
</form>
```

  Sin `authenticity_token`: es un GET y CSRF no se aplica.
- Estado vacío: si `len(tasks) == 0` y `query != ""` → `<p class="empty-state">Ninguna tarea coincide con «<%= query %>».</p>`; si no, el mensaje actual.
- Reordenado: sustituir las dos condiciones `len(tasks) > 1` por `len(tasks) > 1 && query == ""` (atributo `data-reorder-url` y asa de arrastre). Si Plush no admite `&&` en esa posición, calcular antes `<% let reorderable = len(tasks) > 1 && query == "" %>` y usar `reorderable`.

### CSS (`public/assets/app.css`)

- Ampliar el selector `input[type="text"]` a `input[type="text"], input[type="search"]` para que el campo herede el estilo.
- Añadir:

```css
.search-form {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  margin-bottom: 1rem;
}

.search-form input {
  flex: 1;
}

.search-clear {
  color: var(--color-muted);
  font-size: 0.85rem;
  white-space: nowrap;
}
```

No hace falta JS: `app.js` solo activa el reordenado si existe `.task-list[data-reorder-url]`.

## Archivos relevantes

- `models/task.go`: `Task`, `LoadOrdered`; recibe `LoadOrderedMatching` y el escape de comodines.
- `models/task_test.go`: tests de modelo (`ModelSuite`).
- `actions/tasks.go`: `TasksIndex` (lee `q`) y `TasksCreate` (expone `query`).
- `actions/tasks_test.go`: tests de acción (`ActionSuite`, fixture `tres tareas`, helper `createPositioned`).
- `templates/tasks/index.plush.html`: formulario, contador, estado vacío, condiciones de reordenado.
- `public/assets/app.css`: estilos del buscador.
- `public/assets/app.js`: solo referencia; no cambia.
- `templates/`, `public/` van con `go:embed`: recompilar tras cambiarlos.

## Implementación paso a paso

1. `models/task.go`: añadir la constante de orden, `likeEscaper` y `LoadOrderedMatching`; hacer que `LoadOrdered` use la constante.
2. `models/task_test.go`: tests de modelo (ver abajo).
3. `actions/tasks.go`: helper `searchQuery`, cambios en `TasksIndex` y `c.Set("query", "")` en la rama de error de `TasksCreate`.
4. `templates/tasks/index.plush.html`: formulario, contador condicional, estado sin resultados y condición de reordenado.
5. `public/assets/app.css`: selector de input y clases `.search-form`, `.search-clear`.
6. `actions/tasks_test.go`: tests de acción (ver abajo).
7. `gofmt -l .`, `go vet ./...`, `go build ./...`, `GO_ENV=test go test -p 1 ./...`.

## Tests y validación

Modelo (`models/task_test.go`):

- `Test_Tasks_LoadOrderedMatching_CaseInsensitiveSubstring`: crea "Comprar café", "Llamar al gestor", "comprar pan"; buscar "COMPRAR" devuelve las dos de comprar, en orden de `position`.
- `Test_Tasks_LoadOrderedMatching_EscapesWildcards`: crea "100% hecho" y "1000 hecho", y "a_b" y "axb"; buscar "0%" devuelve solo "100% hecho"; buscar "a_b" devuelve solo "a_b".

Acción (`actions/tasks_test.go`):

- `Test_TasksIndex_ShowsSearchForm`: `GET /` contiene `name="q"` y `role="search"`.
- `Test_TasksIndex_Search_FiltersByTitle`: fixture `tres tareas`; `GET /?q=gestor` contiene "Llamar al gestor", no contiene "Comprar material de oficina" ni "Publicar la oferta de empleo", contiene "1 resultado para" y el enlace "Limpiar búsqueda".
- `Test_TasksIndex_Search_CaseInsensitive`: `GET /?q=COMPRAR` contiene "Comprar material de oficina".
- `Test_TasksIndex_Search_NoResults`: `GET /?q=inexistente` contiene "Ninguna tarea coincide" y no "No hay tareas todavía".
- `Test_TasksIndex_Search_BlankShowsAll`: `GET /?q=%20%20` contiene las tres tareas y no "Limpiar búsqueda".
- `Test_TasksIndex_Search_DisablesReorder`: crea dos tareas que coinciden con la búsqueda; con `?q=` no contiene `data-reorder-url` ni `data-drag-handle`; sin `q` sí contiene `data-reorder-url`.
- `Test_TasksIndex_Search_EscapesQuery`: `GET /?q=<script>` no contiene `<script>` literal en el cuerpo (sí la versión escapada).
- Los tests existentes (incluido `Test_TasksCreate_EmptyTitle_Invalid`, que re-renderiza la plantilla) deben seguir pasando: demuestran que `query` está definido en la rama de error.

Para construir URLs con `q` usar `url.Values{"q": {...}}.Encode()` (el paquete `net/url` ya se importa en el test).

Comprobaciones: `gofmt -l .` (vacío), `go vet ./...`, `go build ./...`, `GO_ENV=test go test -p 1 ./...` con PostgreSQL y `todo_test` migrada (`docker compose up -d db`, `GO_ENV=test soda migrate -e test`).

## Riesgos y recuperación

- **Variable `query` no definida en un render**: Plush falla con "unknown identifier". Mitigación: `c.Set("query", …)` en `TasksIndex` y en la rama de error de `TasksCreate` (únicos renders de `tasks/index`); cubierto por los tests de creación inválida.
- **Reordenar con la lista filtrada** daría 409 y recarga. Mitigación: sin asas ni `data-reorder-url` mientras hay búsqueda (criterio 9, test dedicado).
- **Escape de `LIKE`**: si `standard_conforming_strings` estuviera desactivado, `'\'` sería inválido. Es el valor por defecto de PostgreSQL desde 9.1; el test de comodines lo detectaría.
- **Pérdida del filtro tras acciones** (toggle, editar, borrar redirigen a `/`): limitación aceptada y fuera de alcance.
- Recuperación: el cambio no toca esquema ni datos; revertir el commit basta.
