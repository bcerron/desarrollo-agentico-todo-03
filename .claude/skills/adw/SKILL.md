---
name: adw
description: Workflow agéntico completo para una issue - clasifica la descripción como feature, bug o chore, crea la rama, planifica, implementa, valida y hace commit
allowed-tools: Bash, Read, Glob, Grep, Write, Edit, Skill
---

# Workflow agéntico de una issue

Coordina las skills existentes de principio a fin: clasificación → rama → plan (`/feature`, `/bug` o `/chore`) → `/implement` → comprobaciones → `/git:commit`. No hace push, merge ni abre pull requests. No hace preguntas: debe poder ejecutarse con `claude -p`.

Uso: `/adw <descripción | url-o-número-de-issue> [--remote <remoto>] [--base <rama>]`

- Si `$ARGUMENTS` (sin las opciones) es una URL de issue de GitHub o `#<número>`, lee la issue con `gh issue view <issue> --json number,title,body,url`; la descripción es el título más el cuerpo, seguidos de la URL. Es el modo que usa `adws/trigger-github.sh`.
- En cualquier otro caso, el texto es la descripción.
- `--base` es la rama de partida (por defecto `main`). `--remote` solo se usa para actualizar la base antes de crear la rama; el workflow nunca hace push.

Si la descripción está vacía o no permite saber qué hay que hacer, detente e indica el uso correcto.

## 1. Precondiciones

1. Ejecuta `git status --short --branch`. Si hay cambios sin commit, **detente** y explícalo: el workflow no debe arrastrar trabajo ajeno a la rama de la issue.
2. Si se indicó `--remote`, ejecuta `git fetch <remoto> <base>`. Cambia a la base con `git checkout <base>` y, si hay remoto, `git merge --ff-only <remoto>/<base>`. Si el fast-forward falla, detente.

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

Comprueba que el plan existe y contiene las secciones exigidas por esa skill. Si la skill informó de incertidumbres que bloquean la implementación (requisito ambiguo, secretos, decisión de producto), detente aquí y repórtalas.

## 5. Implementación

Invoca la skill `implement` con la ruta del plan. Si `implement` se detiene por un bloqueo, detente también y reporta el motivo; no rediseñes la solución.

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

Revisa también `git diff --check`.

## 7. Commit

Solo con todas las comprobaciones en verde, invoca la skill `git:commit` indicando como alcance el plan y los archivos modificados por la implementación. Si viene de una issue, añade al mensaje una línea `Refs #<número>`.

## Informe final

Resume en pocas líneas:

- tipo elegido y su justificación, rama creada y ruta del plan;
- archivos cambiados;
- resultado de cada comprobación;
- hash y mensaje del commit, o el punto exacto en que se detuvo y por qué.
