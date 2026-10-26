---
name: feature
description: Investiga una funcionalidad y escribe un plan autocontenido en plans/
allowed-tools: Bash, Read, Glob, Grep, Write
---

# Planifica una feature

Recibe en `$ARGUMENTS` la descripción de una funcionalidad. Tu único propósito es investigar y escribir un plan ejecutable; no implementes la funcionalidad.

**Rama:** cada trabajo va en su propia rama con el formato `feature/<nombre-descriptivo-en-kebab-case>` (p. ej. `feature/reordenar-tareas-drag-and-drop`). Tras `git status --short --branch`, si la rama actual no es ya la de este trabajo, créala con `git checkout -b feature/<nombre>` antes de escribir el plan, para que plan y cambios posteriores queden en ella. Los cambios sin commit se arrastran a la rama nueva; no los toques. Indica el nombre de la rama en el informe final.

1. Comprueba `git status --short --branch` y recorre el flujo existente desde rutas Buffalo y acciones hasta modelos, migraciones, templates Plush y assets cuando sean relevantes.
2. Convierte la petición en comportamiento observable y criterios de aceptación. Distingue requisitos explícitos de inferencias.
3. Busca patrones existentes y define el cambio vertical mínimo; no inventes una arquitectura paralela.
4. Identifica los tests que deben demostrar el comportamiento y las condiciones necesarias para ejecutarlos.
5. Escribe un único plan Markdown autocontenido en `plans/`. Usa un nombre kebab-case descriptivo, salvo que el argumento indique una ruta concreta dentro de `plans/`; en ese caso usa exactamente esa ruta.

El plan es el único archivo que puedes crear o modificar. No cambies código de aplicación, tests, configuración ni dependencias.

El plan debe incluir:

- `# Feature: …`
- `## Encargo y criterios de aceptación`
- `## Comportamiento actual y propuesta`
- `## Alcance` (incluido y excluido)
- `## Diseño por capas` (solo las capas afectadas)
- `## Archivos relevantes` (ruta y responsabilidad)
- `## Implementación paso a paso`
- `## Tests y validación`
- `## Riesgos y recuperación`

Cubre persistencia y migraciones, modelo, acción y ruta, renderizado y JavaScript o CSS solo cuando el caso los requiera. Respeta los patrones observados en el repositorio. Las validaciones habituales son comprobar formato con `gofmt -l .`, ejecutar `go vet ./...`, `go test -p 1 ./...` y `go build ./...`; los tests de aplicación necesitan PostgreSQL y las migraciones de test preparadas. Incluye únicamente las comprobaciones pertinentes a la funcionalidad.

Al terminar, informa solo de la ruta del plan, sus decisiones principales y las incertidumbres pendientes. Sugiere `/implement <ruta>` como siguiente paso.

Si te ha invocado otra skill (por ejemplo `adw`), este informe no cierra el trabajo: es el resultado que esa skill necesita para seguir con su siguiente paso sin esperar al usuario.
