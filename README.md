# Hangar

Self-hosted deployment platform. Push code, get a live URL. Built with Go, Temporal, Docker and Traefik.

> Hangar is a lightweight self-hosted PaaS that turns any Git repository
> into a running container with a live URL - in under 60 seconds.

## Quick start

```bash
cd infra && docker compose up -d
cd ../api && go run cmd/server/main.go
```

```bash
# Crear una app
curl -X POST http://localhost:3000/api/v1/apps \
  -H "Content-Type: application/json" \
  -d '{"name":"hello","git_url":"https://github.com/user/hello-node"}'

# Disparar un deploy
curl -X POST http://localhost:3000/api/v1/apps/{id}/deploy
```

## Stack

| Capa | Tecnología |
|---|---|
| API | Go + Fiber |
| Workflows | Temporal.io |
| Proxy | Traefik |
| Base de datos | PostgreSQL |
| Cache / Pub-Sub | Redis |
| Containerización | Docker Engine API |
| Dashboard | Next.js |

## Estructura

```
hangar/
  api/          Go API server
  worker/       Temporal workers (Fase 2)
  dashboard/    Next.js UI (Fase 4)
  infra/        Docker Compose + Traefik config
  docs/adr/     Architecture Decision Records
```
