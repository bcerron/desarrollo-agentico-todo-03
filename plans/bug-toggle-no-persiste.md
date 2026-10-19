# Bug: al completar una tarea desde la lista, el cambio no persiste al recargar

## Encargo y resultado esperado

Reporte: "Al marcar una tarea como completada, el cambio no persiste al recargar la página".

Resultado esperado: pulsar el botón de completar (o reabrir) en la lista de tareas cambia el estado de la tarea en la base de datos. Tras recargar la página, la tarea conserva el estado mostrado antes de recargar.

Resultado observado: la interfaz muestra la tarea como completada (clase `task-completed`, badge "Completada", botón con `aria-pressed="true"`), pero al recargar vuelve a su estado anterior porque el servidor nunca registró el cambio.

## Evidencia y causa raíz

### Hechos observados en el repositorio

- `public/assets/app.js:8-34`: el listener de `submit` intercepta los formularios `[data-task-toggle]`, hace `event.preventDefault()`, cambia la UI de forma optimista y termina con `fetch(form.action);`. La llamada no pasa `method`, `body` ni cabeceras, así que es un **GET** a `/tasks/{id}/toggle`. No envía el `authenticity_token` ni el campo `_method`, que están en el formulario y solo se usan cuando el formulario se envía de forma nativa.
- `actions/app.go:55`: la única ruta registrada para esa URL es `app.PUT("/tasks/{task_id}/toggle", TasksToggle)`. No existe ruta GET para ella.
- `templates/tasks/index.plush.html:41-43`: el formulario es `method="POST"` con `<input name="_method" value="PUT">`. Buffalo interpreta ese campo y enruta a `TasksToggle`. Sin JS, el botón funciona.
- `actions/tasks.go` (`TasksToggle`): invierte `task.Completed` y persiste con `tx.Update`. La lógica del servidor es correcta.
- `actions/tasks_test.go` (`Test_TasksToggle_CompletesAndReopens`): prueba el endpoint con `as.HTML(...).Put(...)`. Nunca ejercita el camino de JavaScript, así que no detecta el fallo.

### Verificación en ejecución (entorno Docker local, `docker compose up --build -d`)

- `GET /tasks/095bfb69-202a-4b25-a23b-4d65c6f0b3c7/toggle` → **404** (página de error de Buffalo). Esto es lo que hace `app.js` en cada clic.
- `POST /tasks/<id>/toggle` con `_method=PUT` y `authenticity_token` válido → **303** a `/`. Es el camino que funciona.
- Tras ese POST, la tabla `tasks` cambia de estado en la BD (comprobado con `psql`). El cambio de prueba se revirtió después con otro POST idéntico; el estado final coincide con el inicial (`Preparar la reunión del lunes` f, `Enviar el informe mensual` f, `Renovar el dominio de la web` t).

### Causa raíz (confirmada por evidencia directa)

El `fetch` de `app.js` hace una petición GET a una ruta que solo acepta PUT (vía POST con `_method`). El servidor responde 404, el código JS ignora la respuesta y la UI queda desincronizada del servidor. Al recargar, el servidor devuelve el estado real.

### Incertidumbres

- **[Probable]** Que el navegador real se comporte igual que `curl` (GET → 404 y sin efectos secundarios). No se ha probado en navegador; el paso de verificación manual del plan lo confirma.
- **[Suponiendo]** Que no hay otros clientes que dependan del GET actual. Se ha buscado en el repositorio: ningún otro archivo invoca `/toggle`.
- No se ha determinado si la versión en caché de `app.js` (`/assets/app.js?v=2`) influye. Un cambio en el archivo debe ir con una nueva versión del query string (`?v=3`) para forzar recarga en el navegador; ver riesgos.

### Otros defectos relacionados (fuera de alcance, ver abajo)

- La UI no revierte el cambio si la petición falla: el JS no comprueba la respuesta. Un fallo de red o un 404 dejan la UI engañosa sin aviso.
- Dos pestañas abiertas: el toggle invierte el estado en el servidor, así que una pestaña desactualizada puede invertir lo contrario de lo que muestra.

## Alcance

### Incluido

- Corregir el `fetch` de `public/assets/app.js` para que envíe la misma petición que el formulario nativo: `POST` a `form.action` con el cuerpo `FormData(form)` (incluye `authenticity_token` y `_method=PUT`).
- Incrementar la versión de `app.js` en `templates/application.plush.html` (`?v=2` → `?v=3`) para evitar caché.
- Añadir un test de regresión en Go que fije el contrato del endpoint que usa el JS (POST con `_method=PUT` y token CSRF).

### Excluido

- Reversión optimista ante errores de red y comprobación de la respuesta. Es un cambio de comportamiento; requiere decisión de producto (qué mostrar al usuario).
- Concurrencia entre pestañas (cambiar el toggle a "fijar estado"). Es un cambio de API.
- Tests de navegador o de JS. El repositorio no tiene infraestructura para ello y añadirla es una decisión aparte.
- Cualquier cambio en `TasksToggle`, en el modelo o en migraciones: el servidor ya es correcto.
- Cambios de estilo, textos o layout.

## Archivos relevantes

| Ruta | Responsabilidad |
|---|---|
| `public/assets/app.js` | Interceptor de envío de formularios `[data-task-toggle]`. Contiene el `fetch` incorrecto (línea 34). Es el único archivo de código a cambiar. |
| `templates/application.plush.html` | Referencia a `/assets/app.js?v=2`. Hay que subir la versión. |
| `templates/tasks/index.plush.html` | Formulario de toggle (`data-task-toggle`, `_method=PUT`, `authenticity_token`). Referencia del contrato; no se modifica. |
| `actions/app.go` | Registro de rutas. `PUT /tasks/{task_id}/toggle`. No se modifica. |
| `actions/tasks.go` | `TasksToggle`. Correcto; no se modifica. |
| `actions/tasks_test.go` | Tests de la acción. Se añade un test de regresión aquí. |
| `public/embed.go` | `go:embed *`: el JS se incrusta en el binario. Tras cambiarlo hay que recompilar la imagen. |

## Implementación paso a paso

1. **Confirmar el fallo antes de tocar código.** Con la app en marcha, desde el navegador completar una tarea, recargar y comprobar que vuelve a su estado. (Ya comprobado con `curl`; el navegador es la verificación final.)
2. **Corregir el `fetch` en `public/assets/app.js`.** Sustituir la línea `fetch(form.action);` por una petición POST con el cuerpo del formulario:
   ```js
   fetch(form.action, { method: "POST", body: new FormData(form) });
   ```
   Motivo: `FormData(form)` incluye `authenticity_token` y `_method=PUT`, exactamente lo que envía el formulario nativo. Buffalo ya trata `_method` en un POST (verificado con `curl`). No añadir cabeceras ni cambiar el flujo optimista.
3. **Subir la versión del script** en `templates/application.plush.html`: `/assets/app.js?v=2` → `/assets/app.js?v=3`.
4. **Añadir el test de regresión** en `actions/tasks_test.go`, junto a `Test_TasksToggle_CompletesAndReopens`. Debe reproducir el envío que hace el JS: POST con `_method=PUT` (el mismo formato que el formulario) y comprobar que la tarea cambia. Ver sección de tests.
5. **No modificar** ningún otro archivo. Si aparece la necesidad de tocar otra cosa, parar y documentarlo.

## Tests y validación

### Test de regresión (debe fallar antes del arreglo)

Para que el test sea significativo, debe ejercitar la misma forma de petición que el JS corregido, distinta de la que usa el test actual (que usa `PUT` directo). Añadir en `actions/tasks_test.go`:

```go
func (as *ActionSuite) Test_TasksToggle_PostWithMethodOverride() {
	task := as.taskForEditing(false)
	res := as.HTML("/tasks/%s/toggle", task.ID).Post(url.Values{"_method": {"PUT"}})
	as.Equal(303, res.Code)
	as.NoError(as.DB.Reload(task))
	as.True(task.Completed)
}
```

Nota sobre el fallo esperado: este test **pasa** con el código actual, porque el servidor ya acepta POST con `_method=PUT`. Es un test de contrato del servidor, no de la regresión en sí. El fallo real está en el JS, que no tiene tests. Por eso la verificación de la regresión se apoya en el paso manual de navegador. Esto debe declararse explícitamente al implementar; no presentarlo como test que falla antes del arreglo.

Si se quiere un test que falle antes del arreglo, la alternativa es servir `/assets/app.js` y comprobar que contiene `method: "POST"`. Es frágil (depende del texto del código) y no se recomienda sin acuerdo.

### Validación automatizada (orden)

1. `gofmt -l .` → no debe listar archivos.
2. `go vet ./...` → sin errores.
3. `go build ./...` → compila (incluye `public/embed.go` con el JS nuevo).
4. `go test -p 1 ./...` → todos en verde. **Requiere** la BD `todo_test` con migraciones aplicadas. Pasos en el entorno local:
   - `docker compose exec -T db psql -U postgres -c "CREATE DATABASE todo_test;"` (no borra volúmenes; si ya existe, el error es inocuo).
   - `go install github.com/gobuffalo/pop/v6/soda@v6.4.1` y `GO_ENV=test soda migrate -e test` (o el equivalente que use CI, `.github/workflows/ci.yaml`, líneas 51-52).
   - `GO_ENV=test go test -p 1 ./actions -run Test_ActionSuite -testify.m Test_TasksToggle` para el foco, y después la suite completa.
   - **No verificado aún**: que estos comandos funcionen en este equipo Windows; el paso puede requerir ajustes.

### Verificación manual (obligatoria; es la que cubre la regresión real)

1. Tras `docker compose up --build -d app` (el JS está embebido en el binario; hace falta reconstruir), abrir http://localhost:3000 con las herramientas de desarrollo abiertas (pestaña Red).
2. Completar una tarea pendiente. Debe verse una petición **POST** a `/tasks/<id>/toggle` con respuesta **303**, no GET.
3. Recargar la página. La tarea debe seguir completada.
4. Reabrirla, recargar, comprobar que vuelve a pendiente.
5. Restaurar el estado original de las tareas de demo si se ha alterado.
6. Comprobar en la consola del navegador que no hay errores de JS.

### Comprobación de no regresión

- Sin JS (desactivar JavaScript en el navegador): el formulario nativo debe seguir funcionando (POST con `_method=PUT`).
- Borrado y edición no cambian: `Test_TasksDestroy`, `Test_TasksUpdate_*` deben seguir en verde.

## Riesgos y recuperación

- **Caché del navegador:** sin subir `?v=3`, algunos navegadores seguirán ejecutando el JS antiguo tras el despliegue. Mitigación: el cambio de versión es obligatorio en el paso 3.
- **Imagen Docker desactualizada:** el JS se embebe en el binario. Si solo se reinicia el contenedor sin `--build`, el cambio no se aplica. Verificar con `docker compose up --build -d app`.
- **Tests de BD:** `go test` necesita `todo_test`. Si el paso de creación falla, no es un fallo del código; documentarlo y seguir con la verificación manual.
- **Datos del entorno:** la verificación manual cambia el estado de tareas de demo en `todo_development`. Al terminar, dejar el estado original. No eliminar volúmenes (`docker compose down -v` queda prohibido en esta tarea).
- **Recuperación:** el cambio es de dos líneas y un número de versión. Revertir con `git checkout -- public/assets/app.js templates/application.plush.html actions/tasks_test.go` si hiciera falta, y reconstruir la imagen.
- **Riesgo de no arreglar el resto:** tras este cambio, un fallo de red seguirá dejando la UI engañosa (ver "Excluido"). Conviene registrarlo como deuda aparte.
