---
name: adw
description: Workflow agéntico completo para una issue - clasifica la descripción como feature, bug o chore, crea la rama, planifica, implementa, valida, hace commit y abre un pull request
allowed-tools: Bash, Read, Glob, Grep, Write, Edit, Skill
---

# Workflow agéntico de una issue

Coordina las skills existentes de principio a fin: clasificación → rama → plan (`/feature`, `/bug` o `/chore`) → `/implement` → comprobaciones → `/git:commit` → push y pull request. Nunca hace merge. No hace preguntas: debe poder ejecutarse con `claude -p`.

**Ejecución continua.** Los pasos 1 a 8 son una sola ejecución. Cuando una skill invocada (`feature`, `bug`, `chore`, `implement`, `git:commit`) termina con su informe, ese informe es un resultado intermedio, no el final del trabajo: no lo devuelvas al usuario ni esperes confirmación, y pasa inmediatamente al paso siguiente. Ignora las sugerencias de "siguiente paso" de esas skills (como `Sugiere /implement <ruta>`): el siguiente paso lo marca este workflow. Solo terminas al llegar al **Informe final**, ya sea con el pull request abierto o porque una condición de parada explícita de este documento se ha cumplido.

Uso: `/adw <descripción | url-o-número-de-issue> [--remote <remoto>] [--base <rama>]`

- Si `$ARGUMENTS` (sin las opciones) es una URL de issue de GitHub o `#<número>`, lee la issue con `gh issue view <issue> --json number,title,body,url`; la descripción es el título más el cuerpo, seguidos de la URL. Es el modo que usa `adws/trigger-github.sh`.
- En cualquier otro caso, el texto es la descripción.
- `--base` es la rama de partida y la base del pull request (por defecto `main`). `--remote` es el remoto desde el que se actualiza la base y al que se sube la rama (por defecto `origin`).

Si la descripción está vacía o no permite saber qué hay que hacer, detente e indica el uso correcto.

## 1. Precondiciones

1. Ejecuta `git status --short --branch`. Si hay archivos versionados modificados o en el índice, **detente** y explícalo: el workflow no debe arrastrar trabajo ajeno a la rama de la issue. Los archivos sin seguimiento (`??`) no bloquean: déjalos como están y no los incluyas nunca en el commit.
2. Ejecuta `git fetch <remoto> <base>`, cambia a la base con `git checkout <base>` y actualízala con `git merge --ff-only <remoto>/<base>`. Si el fast-forward falla, detente.
3. Comprueba que la petición no esté ya resuelta o en curso en un pull request anterior (ver abajo). Si lo está, termina aquí: no crees rama ni plan.

### ¿Ya resuelta?

Obtén `OWNER/REPO` de `git remote get-url <remoto>` y lista los pull requests con `gh pr list --repo OWNER/REPO --state all --limit 100 --json number,title,body,state,url,headRefName`. Selecciona como candidatos los que traten la misma funcionalidad, fallo o tarea que la descripción (por título, cuerpo, rama, plan enlazado o `Closes/Refs #<número>` de esta issue), no los que solo compartan palabras sueltas. Revisa cada candidato con `gh pr view <número> --repo OWNER/REPO` y `gh pr diff <número> --repo OWNER/REPO`.

- **Fusionado (`MERGED`) que cubre la petición completa**: confírmalo en el código de la base actualizada (rutas, plantillas, modelos o tests que demuestren el comportamiento pedido). Solo si el código lo confirma, la petición está resuelta:
  - si viene de una issue, ciérrala con `gh issue close <número> --repo OWNER/REPO --reason "not planned" --comment "<texto>"`, donde el texto, en español, indica que ya está resuelta, enlaza el PR y resume en 1-2 líneas dónde está el comportamiento en el código;
  - si viene de una descripción en texto, no hay nada que cerrar: infórmalo.
- **Abierto (`OPEN`) que cubre la petición**: el trabajo ya está en curso. No la cierres ni dupliques el trabajo; si viene de una issue, comenta en ella con `gh issue comment <número> --repo OWNER/REPO --body "<texto>"` enlazando el PR, y termina.
- **Cerrado sin fusionar (`CLOSED`)**: no cuenta como resuelto. Continúa y menciónalo en el cuerpo del pull request.
- **Cobertura parcial o dudosa**: continúa con el flujo normal y cita el PR relacionado en el cuerpo del pull request. Ante la duda, no descartes: cerrar por error una issue válida es peor que repetir un plan.

## 2. Clasificación

Determina el tipo a partir de la descripción y, cuando haga falta, del código afectado (busca las rutas, acciones, modelos o plantillas que menciona):

- **bug**: algo que ya existe no se comporta como debería (error, fallo, dato incorrecto, regresión). La descripción contrasta un comportamiento esperado con uno observado.
- **feature**: una capacidad nueva u observable por el usuario que hoy no existe, o un cambio deliberado del comportamiento existente.
- **chore**: mantenimiento sin cambio de comportamiento observable: dependencias, refactor, estilos internos, documentación, CI, tooling, configuración.

Reglas de desempate:

- Si pide cambiar un comportamiento que funciona como se diseñó, es **feature**, no bug.
- Si el código ya hace lo que la descripción reclama como fallo, no lo clasifiques como bug: es feature (si pide algo distinto) o detente si no hay nada que hacer.
- Si mezcla varios trabajos independientes, detente y pide dividirlo en varias issues.

Anota el tipo elegido y una línea de justificación; ambos van en el informe final.

## 3. Rama

Cada issue va en su propia rama, creada desde la base:

1. Genera un slug kebab-case corto y descriptivo (sin acentos, ≤ 5 palabras). Si viene de una issue, antepón el número: `42-toggle-no-persiste`.
2. Crea la rama `<tipo>/<slug>` con `git checkout -b <tipo>/<slug>`. Si ya existe, detente e infórmalo; no reutilices ni sobrescribas ramas.

La skill de planificación detectará que ya está en la rama del trabajo y no creará otra.

## 4. Plan

Invoca con la herramienta `Skill` la skill del tipo (`feature`, `bug` o `chore`) pasando como argumentos la descripción y la ruta exacta del plan: `plans/<tipo>-<slug>.md`.

Comprueba que el plan existe y contiene las secciones exigidas por esa skill. Si la skill informó de incertidumbres que bloquean la implementación (requisito ambiguo, secretos, decisión de producto), detente aquí y repórtalas. Si no hay bloqueos, continúa sin pausa con el paso 5.

## 5. Implementación

Invoca la skill `implement` con la ruta del plan. Si `implement` se detiene por un bloqueo, detente también y reporta el motivo; no rediseñes la solución. Si termina sin bloqueo, continúa con el paso 6.

## 6. Comprobaciones

Ejecuta, en este orden, las mismas comprobaciones que CI:

```bash
gofmt -l .            # debe no listar nada
go vet ./...
go build ./...
GO_ENV=test go test -p 1 ./...
```

Los tests necesitan PostgreSQL con la BD `todo_test` migrada. Si no está disponible, levántala con `docker compose up -d db` y migra con `GO_ENV=test soda migrate -e test` (requiere `soda` v6.4.1). Si aun así no hay base de datos, no des los tests por buenos: detente sin commit y repórtalo.

Si alguna comprobación falla:

- corrige solo lo que esté dentro del alcance del plan (formato, errores de compilación, tests del cambio) y vuelve a ejecutar **todas** las comprobaciones;
- máximo **3 intentos**. Si siguen fallando, o el arreglo exige salirse del plan, detente sin commit y reporta la salida del error;
- nunca desactives, borres ni debilites tests para obtener verde.

Excepción para fallos preexistentes: si un test falla en un paquete que la rama no modifica (`git diff --name-only <base>...HEAD` y `git status --short` no tocan ese paquete), comprueba si falla igual en la base con `git stash -u`, `git checkout <base>`, el mismo `go test`, y vuelta con `git checkout <tipo>/<slug>` y `git stash pop`. Si también falla en la base, no es una regresión de este trabajo: no lo corrijas, continúa y recógelo en `## Comprobaciones` del pull request con la salida resumida. CI lo volverá a ejecutar sobre el PR.

Revisa también `git diff --check`.

## 7. Commit

Solo con todas las comprobaciones en verde, invoca la skill `git:commit` indicando como alcance el plan y los archivos modificados por la implementación. Si viene de una issue, añade al mensaje una línea `Refs #<número>`. Con el commit creado, continúa con el paso 8.

## 8. Pull request

Solo si el commit se ha creado:

1. Sube la rama: `git push -u <remoto> <tipo>/<slug>`. Nunca uses `--force`.
2. Obtén el repositorio destino `OWNER/REPO` a partir de `git remote get-url <remoto>` y pásalo **siempre** con `--repo`: en un fork, `gh` puede apuntar por defecto al repositorio padre y el PR acabaría en otro sitio.
3. Si ya existe un PR para la rama (`gh pr view <tipo>/<slug> --repo OWNER/REPO`), no crees otro: informa de su URL.
4. Crea el PR con `gh pr create --repo OWNER/REPO --base <base> --head <tipo>/<slug> --title "<primera línea del commit>" --body-file <archivo>`. Escribe el cuerpo en un archivo temporal fuera del repositorio, en español, con:
   - `## Resumen`: qué cambia y por qué, en 2-4 viñetas;
   - `## Tipo`: el tipo elegido y su justificación;
   - `## Plan`: la ruta del plan;
   - `## Comprobaciones`: resultado de cada comprobación del paso 6;
   - si viene de una issue del mismo repositorio, una línea `Closes #<número>`;
   - como última línea: `🤖 Generated with [Claude Code](https://claude.com/claude-code)`.
5. Si el push o la creación del PR fallan (permisos, autenticación, remoto inexistente), detente y repórtalo: el commit local queda hecho.

## Informe final

Resume en pocas líneas:

- si se descartó por estar resuelta o en curso: el PR que la cubre, la evidencia en el código y si la issue se cerró o se comentó;
- tipo elegido y su justificación, rama creada y ruta del plan;
- archivos cambiados;
- resultado de cada comprobación;
- hash y mensaje del commit y URL del pull request, o el punto exacto en que se detuvo y por qué.
