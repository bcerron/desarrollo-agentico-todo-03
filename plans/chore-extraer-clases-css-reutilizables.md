# Chore: extraer estilos repetidos en clases CSS reutilizables

Rama: `chore/extraer-clases-css-reutilizables`

## Encargo y resultado esperado

Encargo literal: "extrae los estilos repetidos en clases CSS reutilizables y aplicarlos sin cambiar el aspecto ni el comportamiento de la aplicacion".

Resultado esperado:

- Las declaraciones que hoy se repiten en `public/assets/app.css` pasan a vivir en **clases reutilizables** (y, para valores literales repetidos, en variables de `:root`), y las plantillas aplican esas clases.
- **Píxel a píxel la app se ve igual** (escritorio, móvil ≤760px y ≤380px, hover, foco, tarea completada, arrastre, errores de validación, flash, página de edición, sin JS).
- **El comportamiento no cambia**: `public/assets/app.js` no se toca y ninguna clase que use (`task`, `task-completed`, `task-title`, `task-status`, `task-actions`, `task-list`, `toggle-label`, `task-dragging`, `js`) se renombra ni se elimina.

## Estado actual y restricciones

Evidencia del repositorio:

- **No hay estilos inline** (`grep -rn 'style=' templates` no devuelve nada). Toda la repetición está dentro de `public/assets/app.css` (526 líneas, CSS plano, sin preprocesador ni bundler).
- Repeticiones detectadas en `app.css`:
  1. **Botones de icono.** `.task-toggle-button, .icon-action` (líneas 140-157) y `.drag-handle` (226-238) repiten `align-items: center; justify-content: center; min-height: 44px; padding: 0; background: transparent; border: 0; border-radius: 6px` y el `display: inline-flex` (en `.drag-handle` lo pone `.js .drag-handle`, línea 258).
  2. **Hover "fantasma" e icono de 18px.** `.icon-action:hover` (216-219) y `.drag-handle:hover` (245-248) son idénticos (`color: var(--color-text); background: #f2f2ee`); `.icon-action svg` (211-214) y `.drag-handle svg` (240-243) son idénticos (`18px × 18px`).
  3. **Foco redundante.** `.drag-handle:focus-visible { outline: 3px solid var(--color-focus); }` (250-252) repite el valor que ya aplica la regla global `button:focus-visible` (348-353). La regla global sigue aportando `outline-offset: 3px`, así que eliminar la específica no cambia nada.
  4. **Texto secundario.** `.list-count` (112-116) y `.field-hint` (355-359) comparten `color: var(--color-muted); font-size: 0.8rem`; solo difieren en `margin-top`.
  5. **Literales repetidos.** `border-radius: 6px` aparece 9 veces; `#f2f2ee` 2 veces (hover fantasma); `#f7e9e4` 2 veces (`.nav-link` y hover de `.completion-icon`).
- `public/` y `templates/` se sirven con `go:embed` (`public/embed.go`, `templates/embed.go`): **hay que recompilar/reiniciar** para ver el cambio.
- `templates/application.plush.html` carga `/assets/app.css?v=4`; el número se sube manualmente cuando cambia el CSS (commit `d8dc18d`).
- Ningún test Go comprueba nombres de clase (`grep -rn 'class=' --include=*.go .` vacío). CI (`.github/workflows/ci.yaml`) ejecuta `gofmt -l .`, `go vet ./...`, `soda migrate -e test`, `go test -p 1 ./...`, `go build ./...` y un job `docker`. Ninguno valida el aspecto visual: la comprobación visual es manual (ver "Tests y validación").
- No existe Playwright ni herramienta de regresión visual en el repo; **no se añade ninguna**.

Restricciones de cascada que el implementador debe respetar (son el punto de ruptura real de este chore):

- `.icon-button` (nueva, especificidad 0,1,0) debe declararse **antes** de `.drag-handle`, porque `.drag-handle { display: none }` tiene la misma especificidad y debe ganar para que el asa no aparezca sin JS.
- `.icon-button-ghost:hover` (0,2,0) debe declararse **antes** de `.icon-danger:hover` (0,2,0) para que el botón de eliminar siga poniéndose rojo.
- Las reglas de las media queries (`.task-toggle-button { margin-left: -4px }`, etc.) no se tocan.

## Alcance

Incluido:

- Variables nuevas en `:root`: `--radius`, `--color-hover`, `--color-primary-soft`.
- Clases nuevas: `.icon-button`, `.icon-button-ghost`, `.text-meta`.
- Aplicarlas en `templates/tasks/index.plush.html` y `templates/tasks/edit.plush.html`, **añadiéndolas** a las clases existentes (nunca sustituyéndolas).
- Eliminar las declaraciones que pasan a la clase compartida y la regla redundante `.drag-handle:focus-visible`.
- Subir `app.css?v=4` a `app.css?v=5`.

Excluido:

- `public/assets/app.js` (incluido su `?v=4`), handlers Go, tests, migraciones, Dockerfile, CI.
- Renombrar o eliminar clases existentes.
- Unificar valores **parecidos pero distintos** (p. ej. `font-size` 0.85rem vs 0.9rem, `padding` de `.flash` vs `.field-error`, colores grises cercanos `#92928a`/`#96968f`): hacerlo cambiaría el aspecto.
- Sustituir `44px` por una variable: aparece en `grid-template-columns` y `calc()` con significados distintos (columna vs objetivo táctil); el beneficio no compensa el riesgo.
- Añadir herramientas de lint CSS o de regresión visual.

## Archivos relevantes

| Ruta | Responsabilidad |
|---|---|
| `public/assets/app.css` | Hoja única de estilos; aquí se crean las variables y clases y se eliminan las duplicaciones. |
| `templates/tasks/index.plush.html` | Lista: asa de arrastre, botón de completar, editar/eliminar, contador. Recibe `icon-button`, `icon-button-ghost` y `text-meta`. |
| `templates/tasks/edit.plush.html` | Formulario de edición: recibe `text-meta` en la pista del campo. |
| `templates/application.plush.html` | Layout; solo cambia `app.css?v=4` → `?v=5`. |
| `public/assets/app.js` | **Solo lectura**: depende de las clases listadas arriba; confirmar que siguen presentes. |
| `public/embed.go`, `templates/embed.go` | Embeben los assets: obligan a recompilar. |

## Implementación paso a paso

### 0. Captura de referencia (antes de tocar nada)

Ver "Tests y validación → Línea base". Sin la línea base no hay forma objetiva de afirmar "sin cambios visuales".

### 1. Variables en `:root` (`public/assets/app.css`)

Añadir al final del bloque `:root` (tras `--font-family`):

```css
  --color-hover: #f2f2ee;
  --color-primary-soft: #f7e9e4;
  --radius: 6px;
```

Sustituir los literales **exactos**:

- `border-radius: 6px;` → `border-radius: var(--radius);` en: `.nav-link`, `.task-toggle-button, .icon-action` (que pasará a `.icon-button`), `.drag-handle` (idem), `input[type="text"]`, `.button`, `.field-error`, `.flash`. No tocar `border-radius: 50%` de `.completion-icon`.
- `background: #f2f2ee;` → `background: var(--color-hover);` (los dos hovers; quedarán en `.icon-button-ghost:hover`).
- `background: #f7e9e4;` → `background: var(--color-primary-soft);` en `.nav-link` y `.task-toggle-button:hover .completion-icon`.

Comprobación: `grep -n "6px;\|#f2f2ee\|#f7e9e4" public/assets/app.css` solo debe mostrar las definiciones de `:root`.

### 2. Clase base `.icon-button`

Sustituir el bloque `.task-toggle-button, .icon-action { … }` (líneas 140-157) por:

```css
.icon-button {
  align-items: center;
  justify-content: center;
  min-height: 44px;
  padding: 0;
  background: transparent;
  border: 0;
  border-radius: var(--radius);
}

.task-toggle-button,
.icon-action {
  position: relative;
  display: inline-flex;
  flex-shrink: 0;
  width: 44px;
  font: inherit;
  color: var(--color-muted);
  cursor: pointer;
  text-decoration: none;
}
```

Nota: `display` **no** va en `.icon-button`, porque `.drag-handle` debe seguir oculto sin JS y visible (`inline-flex`) solo con `.js .drag-handle`. Así no dependemos del orden de cascada para `display`.

Y reducir `.drag-handle` a lo propio:

```css
.drag-handle {
  display: none;
  width: 28px;
  color: #9a9a94;
  cursor: grab;
}
```

### 3. Modificador `.icon-button-ghost` (hover + icono 18px)

Eliminar `.icon-action svg`, `.icon-action:hover`, `.drag-handle svg`, `.drag-handle:hover` y `.drag-handle:focus-visible`. Añadir, **inmediatamente antes de `.icon-danger:hover`**:

```css
.icon-button-ghost svg {
  width: 18px;
  height: 18px;
}

.icon-button-ghost:hover {
  color: var(--color-text);
  background: var(--color-hover);
}
```

`.icon-danger:hover` se queda detrás, sin cambios.

### 4. Utilidad `.text-meta`

Añadir tras `.page-header`/`h1`:

```css
.text-meta {
  color: var(--color-muted);
  font-size: 0.8rem;
}
```

Reducir `.list-count` a `margin-top: 0.4rem;` y `.field-hint` a `margin-top: 0.65rem;`.

### 5. Aplicar las clases en las plantillas

`templates/tasks/index.plush.html`:

- Línea 3: `class="list-count"` → `class="list-count text-meta"`.
- Línea 18: `class="drag-handle"` → `class="drag-handle icon-button icon-button-ghost"`.
- Línea 27: `class="task-toggle-button"` → `class="task-toggle-button icon-button"` (**sin** `icon-button-ghost`: su hover y su icono son distintos).
- Línea 43: `class="icon-action"` → `class="icon-action icon-button icon-button-ghost"`.
- Línea 51: `class="icon-action icon-danger"` → `class="icon-action icon-button icon-button-ghost icon-danger"`.

`templates/tasks/edit.plush.html`:

- Línea 20: `class="field-hint"` → `class="field-hint text-meta"`.

No tocar atributos `data-*`, `aria-*` ni la estructura.

### 6. Cache busting

`templates/application.plush.html` línea 7: `app.css?v=4` → `app.css?v=5`. No cambiar `app.js?v=4`.

### 7. Recompilar y validar

Ver la sección siguiente.

## Tests y validación

### Automática (obligatoria, antes y después)

```bash
docker compose up -d db
gofmt -l .                 # debe salir vacío
go vet ./...
go build ./...
GO_ENV=test go test -p 1 ./...
```

Ningún test cubre CSS: que pasen solo demuestra que las plantillas siguen renderizando y que las acciones no cambian.

Comprobaciones de texto tras el cambio:

```bash
grep -n "6px;\|#f2f2ee\|#f7e9e4" public/assets/app.css          # solo en :root
grep -n "drag-handle:focus-visible\|icon-action svg\|drag-handle svg" public/assets/app.css   # vacío
grep -n "class=\"task\|toggle-label\|task-status\|task-actions\|task-list" templates/tasks/index.plush.html  # las clases que usa app.js siguen ahí
git diff --stat            # solo app.css, index, edit y application
```

### Visual (manual, antes y después) — la validación que importa

Línea base **antes** del paso 1, en `main` o en la rama sin cambios: `go run ./cmd/app` (o `buffalo dev`), crear al menos 3 tareas, completar una, y en DevTools pegar en la consola:

```js
copy(JSON.stringify([...document.querySelectorAll(
  ".nav-link,.list-count,.drag-handle,.task-toggle-button,.icon-action,.icon-danger,.completion-icon,.task-title,input[type=text],.button,.field-hint,.field-error,.flash"
)].map(el => { const s = getComputedStyle(el); return [el.className,
  ["display","width","min-height","padding","margin","color","background-color","border","border-radius","font-size","font-family","cursor","outline","outline-offset","text-decoration-line"]
  .map(p => s.getPropertyValue(p)).join("|")]; }), null, 1))
```

Guardar el resultado (portapapeles) en el scratchpad, **no en el repo**. Repetir en `/tasks/{id}/edit` y tras enviar el formulario vacío (para `.field-error`). Después del cambio, recompilar, repetir y comparar: **la única diferencia permitida es el texto de `className`**. Cualquier otra diferencia es una regresión.

Revisión visual a mano (antes/después, mismo navegador):

1. Escritorio (≥1024px): hover sobre una tarea (aparecen acciones), hover sobre editar (fondo gris), sobre eliminar (fondo rosado y icono rojo), sobre el círculo de completar (relleno rosado con check rojo), sobre el asa (fondo gris).
2. Foco con Tab en asa, completar, editar, eliminar, input y botones: contorno azul de 3px **separado 3px** del elemento.
3. Móvil 375px y 360px: acciones siempre visibles, columnas de la tarea igual de anchas, botón "Añadir" baja de línea a ≤380px.
4. Arrastrar una tarea (opacidad reducida) y Alt+flechas: el orden se guarda al recargar.
5. Completar/reabrir: tachado y círculo gris; recargar mantiene el estado.
6. JS desactivado: **no** aparece el asa; completar funciona por envío normal.
7. Flash de éxito tras crear/editar/eliminar.

## Riesgos y recuperación

| Riesgo | Síntoma | Mitigación |
|---|---|---|
| Orden de cascada: `.icon-button-ghost:hover` después de `.icon-danger:hover` | Eliminar ya no se pone rojo al pasar el ratón | Respetar el paso 3; comprobar hover de eliminar. |
| Poner `display` en `.icon-button` | El asa aparece sin JS o desaparece con JS | `display` se queda en las clases específicas (paso 2). |
| Olvidar añadir `icon-button` en alguna plantilla | Botón con borde/fondo/padding nativos del navegador | El diff de estilos computados lo detecta (`border`, `padding`, `background-color`). |
| Añadir `icon-button-ghost` al botón de completar | Fondo gris al hover del círculo y el icono crece | Paso 5 lo excluye explícitamente. |
| Sustituir o renombrar una clase usada por `app.js` | Completar o reordenar dejan de funcionar | Solo se **añaden** clases; grep de comprobación. |
| CSS antiguo en caché | Estilos rotos tras desplegar | `?v=5`. Además, el CSS viejo mantiene todas las declaraciones en las clases antiguas, así que HTML nuevo + CSS viejo se ve igual. |
| Servidor sin recompilar | "No cambia nada" o se ve el CSS viejo | `go:embed`: reiniciar `go run ./cmd/app` / dejar recompilar a `buffalo dev`. |

Recuperación: el cambio está confinado a 4 archivos de presentación sin migraciones ni datos. `git checkout main -- public/assets/app.css templates/` revierte por completo; si ya hay commit, `git revert <sha>`.
