---
name: chore
description: Investiga una tarea de mantenimiento y escribe un plan autocontenido en plans/
allowed-tools: Bash, Read, Glob, Grep, Write
---

# Planifica un chore

Recibe en `$ARGUMENTS` una tarea de mantenimiento. Tu único propósito es investigar y escribir un plan prescriptivo; no la implementes.

**Rama:** cada trabajo va en su propia rama con el formato `chore/<nombre-descriptivo-en-kebab-case>` (p. ej. `chore/actualizar-dependencias-go`). Tras `git status --short --branch`, si la rama actual no es ya la de este trabajo, créala con `git checkout -b chore/<nombre>` antes de escribir el plan, para que plan y cambios posteriores queden en ella. Los cambios sin commit se arrastran a la rama nueva; no los toques. Indica el nombre de la rama en el informe final.

1. Comprueba `git status --short --branch` y localiza las convenciones, versiones y automatizaciones relevantes.
2. Verifica el estado actual en el repositorio. No supongas herramientas, paquetes ni scripts que no existan.
3. Identifica compatibilidad, configuración, documentación y posibles efectos operativos.
4. Delimita el cambio mínimo y ordena sus pasos para evitar modificaciones incidentales.
5. Escribe un único plan Markdown autocontenido en `plans/`. Usa un nombre kebab-case descriptivo, salvo que el argumento indique una ruta concreta dentro de `plans/`; en ese caso usa exactamente esa ruta.

El plan es el único archivo que puedes crear o modificar. No cambies código de aplicación, tests, configuración ni dependencias.

El plan debe incluir:

- `# Chore: …`
- `## Encargo y resultado esperado`
- `## Estado actual y restricciones`
- `## Alcance` (incluido y excluido)
- `## Archivos relevantes` (ruta y responsabilidad)
- `## Implementación paso a paso`
- `## Tests y validación`
- `## Riesgos y recuperación`

Deriva de `$ARGUMENTS` y de la evidencia del repositorio los archivos concretos, los puntos de ruptura u otros límites de comportamiento relevantes y las validaciones aplicables. El plan debe preservar el comportamiento observable que no forme parte explícita del encargo y explicar cómo comprobarlo antes y después del cambio.

Detalla comandos reproducibles y el orden de los cambios. Usa solo validaciones disponibles y pertinentes, como `gofmt -l .`, `go vet ./...`, `go test -p 1 ./...`, `go build ./...` o `docker build -t todo .`. No añadas una herramienta ni una dependencia salvo que el encargo lo requiera explícitamente y el plan justifique el coste.

Al terminar, informa solo de la ruta del plan, sus decisiones principales y las incertidumbres pendientes. Sugiere `/implement <ruta>` como siguiente paso.
