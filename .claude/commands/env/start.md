---
description: Start the TODO development environment and verify that the application responds
allowed-tools: Bash, Read
---

# Arranca el entorno de desarrollo

Levanta la aplicación desde la raíz del repositorio. No mates procesos existentes ni elimines volúmenes.

## Modos

- Usa el modo **host** por defecto.
- Usa el modo **Docker completo** si `$ARGUMENTS` contiene `--docker` o si Go/Buffalo no están disponibles en el host.

## Comprobaciones previas

1. Confirma que `docker compose` está disponible y que el daemon responde.
2. Comprueba si los puertos `3000` o `5432` ya están ocupados.
   - Si pertenecen a este proyecto y el servicio responde, reutilízalos (antes, aplica el paso 3 a la app).
   - Si pertenecen a otro proceso, informa del conflicto y detente. No lo finalices.
3. Si el contenedor `app` de este proyecto ya está en marcha, comprueba si está desfasado respecto al código:
   - Está desfasado si la fecha de creación de su imagen (`docker image inspect <imagen> --format '{{.Created}}'`) es anterior al último commit (`git log -1 --format=%cI`) o a la fecha de modificación de algún archivo que liste `git status --porcelain`. Compara en la misma zona horaria: `Created` viene en UTC.
   - También lo está si el contenedor usa una imagen distinta de la última construida (`docker inspect <contenedor> --format '{{.Image}}'` frente al ID de la imagen).
   - Si está desfasado, ejecuta `docker compose up -d --build app`. Recrea solo la app con el código actual; nunca toques `db` ni sus volúmenes. Es la única excepción permitida a la regla de no finalizar procesos existentes.
   - Espera a que `app` vuelva a figurar como saludable y comprueba `/healthz`.
   - Si no está desfasado, reutilízalo sin cambios.

## Modo host

1. Comprueba que `go` y `buffalo` están disponibles.
   - Si falta alguno, explica el bloqueo y continúa mediante Docker completo.
2. Ejecuta `docker compose up -d db`.
3. Espera a que PostgreSQL figure como saludable en `docker compose ps`.
4. Ejecuta `go mod download`.
5. Inicia `buffalo dev` en segundo plano desde la raíz y conserva su salida para poder diagnosticar el arranque.
6. Espera hasta que `http://localhost:3000/healthz` responda correctamente.

## Modo Docker completo

1. Ejecuta `docker compose up --build -d`.
2. Espera a que `db` y `app` figuren como saludables en `docker compose ps`.
3. Comprueba `http://localhost:3000/healthz`.

## Abrir el navegador

Cuando `/healthz` responda correctamente, abre `http://localhost:3000` en el navegador predeterminado, salvo que `$ARGUMENTS` contenga `--no-browser`:

- Windows: `cmd.exe /c start "" http://localhost:3000`
- macOS: `open http://localhost:3000`
- Linux: `xdg-open http://localhost:3000`

Si el comando falla, no detengas el arranque: indica la URL en el resultado.

## Resultado

Informa de:

- modo utilizado;
- servicios iniciados;
- URL de la aplicación;
- respuesta de `/healthz`;
- si la app estaba desfasada y se reconstruyó (fecha de la imagen anterior y de la nueva);
- si se abrió el navegador;
- cualquier fallback, advertencia o proceso que deba mantenerse activo.
