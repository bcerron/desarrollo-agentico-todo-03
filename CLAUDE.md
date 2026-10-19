# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Qué es

Lista de tareas (TODO) en Go con Buffalo + Pop (PostgreSQL), renderizada en servidor con plantillas Plush. El módulo Go se llama `todo`. Además incluye un "trigger" de issues y skills de Claude para un workflow agéntico (ADW). Documentación y mensajes de la app están en español.

## Comandos

```bash
docker compose up -d db          # PostgreSQL (dev en :5432, DB todo_development)
buffalo dev                      # servidor con recarga en :3000 (requiere CLI buffalo v0.18.14)
go run ./cmd/app                 # alternativa sin buffalo CLI
docker compose up -d --build     # app + db completos en Docker

gofmt -l .                       # CI falla si lista archivos
go vet ./...
go build ./...
go test -p 1 ./...               # -p 1 es obligatorio: los paquetes comparten la misma BD
go test -p 1 ./actions -run Test_ActionSuite -testify.m TestName   # un test concreto de la suite
go test -p 1 ./adws              # tests del trigger (usan gh/claude simulados)
```

Los tests necesitan PostgreSQL con la BD `todo_test` y `GO_ENV=test`. En CI se crea con `soda migrate -e test` (`go install github.com/gobuffalo/pop/v6/soda@v6.4.1`). `database.yml` lee `DB_HOST` (por defecto `127.0.0.1`); producción usa `DATABASE_URL`.

## Arquitectura

- `cmd/app/main.go`: ejecuta `models.Migrate()` (migraciones embebidas, aplicadas al arrancar) y luego `actions.App().Serve()`. No hace falta correr `soda` para desarrollo, solo para tests/CI.
- `actions/app.go`: singleton `App()` (con `sync.Once`) que define middleware (forceSSL solo en producción, CSRF, transacción por request vía `popmw.Transaction`) y rutas. Los handlers obtienen la conexión con `c.Value("tx").(*pop.Connection)`.
- `actions/tasks.go`: CRUD de tareas + `PUT /tasks/{id}/toggle`; `/healthz` en `health.go`. Los errores de validación re-renderizan `tasks/index` con la lista cargada.
- `models/`: `models.DB` se conecta en `init()` según `GO_ENV`. `Task.Validate` aplica título obligatorio y máximo `TitleMaxLength` (200 runas).
- `migrations/`, `templates/`, `public/` se sirven con `go:embed` (cada uno tiene su `embed.go`): un cambio en ellos requiere recompilar. Hay migraciones `.fizz` (esquema) y `.sql` (seed).
- Tests: `actions` usa `suite.NewActionWithFixtures` con `fixtures/tasks.toml`; `models` tiene su propia suite.

## Workflow agéntico

- `.claude/skills/{feature,bug,chore}` solo escriben un plan en `plans/`; `/implement plans/<plan>.md` lo ejecuta sin commit, push ni PR. `.claude/commands/` contiene `prime*`, `env/start` y `git/commit`.
- `adws/` (`bash adws/trigger-github.sh [--once]`): sondea issues con la etiqueta de `config/adw.yml` y lanza `claude -p` con el `prompt` configurado. `repo` viene vacío y el prompt referencia `/adw`, que **no existe en el repo**: hay que crearlo antes. `--once` ejecuta Claude de verdad. Detalles en `adws/README.md`.
