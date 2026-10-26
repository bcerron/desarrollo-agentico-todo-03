---
name: bug
description: Investiga un bug y escribe un plan autocontenido en plans/
allowed-tools: Bash, Read, Glob, Grep, Write
---

# Planifica un bug

Recibe en `$ARGUMENTS` la descripción de un bug. Tu único propósito es investigar y escribir un plan; no implementes la corrección.

**Rama:** cada trabajo va en su propia rama con el formato `bug/<nombre-descriptivo-en-kebab-case>` (p. ej. `bug/toggle-no-persiste`). Tras `git status --short --branch`, si la rama actual no es ya la de este trabajo, créala con `git checkout -b bug/<nombre>` antes de escribir el plan, para que plan y cambios posteriores queden en ella. Los cambios sin commit se arrastran a la rama nueva; no los toques. Indica el nombre de la rama en el informe final.

1. Comprueba `git status --short --branch` y lee el código, los tests y la configuración relacionados.
2. Reconstruye el comportamiento esperado y el observado a partir de evidencia del repositorio. No presentes una conjetura como un hecho.
3. Define una reproducción mínima y el test de regresión que debe fallar antes del arreglo.
4. Delimita el cambio mínimo que corrige la causa raíz sin alterar comportamientos ajenos.
5. Escribe un único plan Markdown autocontenido en `plans/`. Usa un nombre kebab-case descriptivo, salvo que el argumento indique una ruta concreta dentro de `plans/`; en ese caso usa exactamente esa ruta.

El plan es el único archivo que puedes crear o modificar. No cambies código de aplicación, tests, configuración ni dependencias.

El plan debe incluir:

- `# Bug: …`
- `## Encargo y resultado esperado`
- `## Evidencia y causa raíz` (hechos observados, incertidumbres y reproducción)
- `## Alcance` (incluido y excluido)
- `## Archivos relevantes` (ruta y responsabilidad)
- `## Implementación paso a paso`
- `## Tests y validación`
- `## Riesgos y recuperación`

Usa solo comandos y herramientas que existan en este repositorio. La validación habitual de la aplicación es comprobar formato con `gofmt -l .`, ejecutar `go vet ./...`, `go test -p 1 ./...` y `go build ./...`; los tests necesitan PostgreSQL y las migraciones de test descritas en `database.yml` y `.github/workflows/ci.yaml`. Incluye únicamente las comprobaciones pertinentes al cambio investigado.

Al terminar, informa solo de la ruta del plan, sus decisiones principales y las incertidumbres pendientes. Sugiere `/implement <ruta>` como siguiente paso.

Si te ha invocado otra skill (por ejemplo `adw`), este informe no cierra el trabajo: es el resultado que esa skill necesita para seguir con su siguiente paso sin esperar al usuario.
