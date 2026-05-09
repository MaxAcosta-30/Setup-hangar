# Hangar

> Self-hosted deployment platform. Push code, get a live URL. Built with Go, Temporal, Docker and Traefik.

Hangar is a lightweight PaaS that turns any Git repository into a running container with a live subdomain - in under 60 seconds. Built as a learning-focused alternative to Heroku/Render for engineers who want full control over their deployment infrastructure without the complexity of Kubernetes.

---

## Quick Start

```bash
# 1. Clone and scaffold
git clone https://github.com/tu-usuario/hangar
cd hangar

# 2. Start infrastructure
cd infra && docker compose up -d && cd ..

# 3. Apply database migrations
docker exec -i infra-postgres-1 psql -U hangar -d hangar \
  < api/internal/db/migrations/001_initial.sql

# 4. Start API server (terminal 1)
cd api && go run cmd/server/main.go

# 5. Start Temporal worker (terminal 2)
cd api && go run cmd/worker/main.go

# 6. Start dashboard (terminal 3)
cd dashboard && npm install && npm run dev
```

Open **http://localhost:3001** - create an app, hit deploy, watch logs stream live.

### CLI deploy flow

```bash
# Create app
curl -X POST http://localhost:3000/api/v1/apps \
  -H "Content-Type: application/json" \
  -d '{"name":"hello","git_url":"https://github.com/tu-usuario/hello-node"}'
# -> {"app":{"id":"...","subdomain":"hello"},"message":"..."}

# Trigger deploy
curl -X POST http://localhost:3000/api/v1/apps/{id}/deploy
# -> {"deployment":{"id":"...","status":"pending"},"workflow_id":"deploy-..."}

# Stream logs in real time
wscat -c "ws://localhost:3000/api/v1/deployments/{deployment_id}/stream"
# -> {"type":"log","message":"[hangar] clonando ...","time":"..."}
# -> {"type":"log","message":"[build] Step 1/4 : FROM node:20-alpine","time":"..."}
# -> {"type":"done","message":"success","time":"..."}

# App is live
curl http://hello.hangar.local
# -> Hello from Hangar!
```

---

## Architecture

```
GitHub / local repo
       |
       v POST /api/v1/apps/:id/deploy
+--------------+     +---------------------------------------------+
|  API Server  |---->|              Temporal Workflow               |
|  Go + Fiber  |     |                                             |
+--------------+     |  CloneRepository -> BuildDockerImage         |
       |             |  -> RunContainer  -> UpdateDeployStatus        |
       |             +------------------+--------------------------+
       |                                | actividades reales
       v                                v
+--------------+     +----------------------------------------------+
|  PostgreSQL  |     |           Docker Engine API                  |
|  apps        |     |  imagen por app · red aislada por tenant     |
|  deployments |     |  128MB RAM · 25% CPU · restart policy        |
|  deploy_logs |     +-------------------+--------------------------+
+--------------+                         |
       |                                 v
       |             +----------------------------------------------+
       |             |   Traefik (proxy dinámico)                   |
       |             |   lee labels del contenedor                  |
       |             |   -> hello.hangar.local                       |
       |             +----------------------------------------------+
       |
       v Pub/Sub por deployment
+--------------+     +----------------------------------------------+
|    Redis     |---->|  WebSocket Handler -> browser                 |
|  logs:{id}   |     |  logs en tiempo real línea por línea         |
+--------------+     +----------------------------------------------+
```

### Por qué Temporal y no BullMQ o cron

Los workflows de deploy duran 2-5 minutos e involucran operaciones no atómicas (clone -> build -> run -> route). Si el servidor se reinicia entre el paso 2 y 3, Temporal retoma desde la última actividad completada. BullMQ pierde el estado si Redis se reinicia. Cron no tiene concepto de estado ni reintentos por step.

Ver [`docs/adr/001-por-que-go-y-temporal.md`](docs/adr/001-por-que-go-y-temporal.md).

---

## Tenant Isolation

Cada app desplegada en Hangar vive en su propia red Docker aislada lateralmente de las demás:

```
App A --+-- hangar-app-{idA}   (privada, solo App A)
        +-- infra_default      (compartida con Traefik)

App B --+-- hangar-app-{idB}   (privada, solo App B)
        +-- infra_default      (compartida con Traefik)

Traefik --- infra_default
```

**Resultado:** App A no puede hacer requests directos a la IP interna de App B. El único camino A-B es HTTP público vía sus subdominios. Traefik sigue pudiendo alcanzar ambas para el routing.

```bash
# Verificar aislamiento
docker network ls | grep hangar-app
docker inspect hangar-{id} \
  --format '{{range $k,$v := .NetworkSettings.Networks}}{{$k}} {{end}}'
# -> hangar-app-{uuid} infra_default
```

Ver [`docs/adr/003-aislamiento-tenants.md`](docs/adr/003-aislamiento-tenants.md).

---

## Resource Limits

Cada contenedor tiene límites fijados en el momento de creación vía Docker Engine API:

| Recurso | Límite |
|---|---|
| RAM | 128 MB máximo |
| CPU | 25% de un core (CPUQuota/CPUPeriod) |
| Restart policy | `unless-stopped` |
| Network egress | Sin límite (acceso a internet permitido) |
| Network ingress from other apps | Bloqueado por red aislada |

```bash
# Verificar límites
docker inspect hangar-{id} --format '{{.HostConfig.Memory}}'
# -> 134217728
docker inspect hangar-{id} --format '{{.HostConfig.CPUQuota}}'
# -> 25000
```

---

## Performance

Mediciones en hardware local (AMD Ryzen 5, 16GB RAM, SSD NVMe):

| Métrica | Valor |
|---|---|
| Clone + build + run (imagen en caché) | ~3s |
| Clone + build + run (imagen fría, node:20-alpine) | ~42s |
| Latencia WebSocket p50 | < 5ms |
| Latencia WebSocket p99 | < 20ms |
| Throughput API (health endpoint) | ~7,150 req/s |
| Overhead de red aislada por app | ~1KB metadata Docker |

> Las mediciones de build dependen fuertemente del tamaño del `Dockerfile` y del caché de capas de Docker. Un build con `npm install` sobre `node:20-alpine` sin caché toma ~40s; con caché de capas ~4s.

---

## API Reference

### Apps

```
GET    /api/v1/apps                    Lista todas las apps
POST   /api/v1/apps                    Crea una app nueva
GET    /api/v1/apps/:id                Detalle de una app
DELETE /api/v1/apps/:id                Elimina app, contenedor y red
POST   /api/v1/apps/:id/deploy         Dispara un nuevo deploy
GET    /api/v1/apps/:id/deployments    Historial de deployments
```

### Deployments

```
GET    /api/v1/deployments/:id/logs    Logs paginados de un deployment
WS     /api/v1/deployments/:id/stream  Logs en tiempo real (WebSocket)
```

### Health

```
GET    /health    Status del servidor y Postgres
```

---

## Project Structure

```
hangar/
|-- api/                         Go API server + Temporal worker
|   |-- cmd/
|   |   |-- server/main.go       Entry point del API
|   |   |-- worker/main.go       Entry point del Temporal worker
|   |-- internal/
|   |   |-- activities/          Implementación real de cada paso del deploy
|   |   |-- db/                  Repositorios (App, Deployment, Logs)
|   |   |-- docker/              Docker Engine API client factory
|   |   |-- handler/             HTTP handlers (apps, health, websocket)
|   |   |-- middleware/          Logger estructurado, error handler
|   |   |-- models/              Structs compartidos (App, Deployment, etc.)
|   |   |-- rdb/                 Redis client + LogChannel helper
|   |   |-- temporal/            Temporal client + TaskQueue constant
|   |   |-- workflows/           DeployWorkflow (orquestación pura)
|   |-- tests/
|       |-- deploy_integration_test.go  Tests end-to-end sin mocks
|-- dashboard/                   Next.js 14 App Router
|   |-- app/
|       |-- page.tsx             Lista de apps
|       |-- apps/[id]/           Detalle + deploy button
|       |-- apps/[id]/deployments/[id]/  Terminal de logs en vivo
|       |-- components/          StatusBadge, DeployButton, LogStream, CreateAppForm
|       |-- lib/api.ts           API client tipado
|-- infra/
|   |-- docker-compose.yml       Postgres, Redis, Temporal, Temporal UI, Traefik
|-- docs/
    |-- adr/
        |-- 001-por-que-go-y-temporal.md
        |-- 002-por-que-kappa-sobre-lambda.md
        |-- 003-aislamiento-tenants.md
```

---

## Running Tests

```bash
# Unit tests (sin dependencias externas)
cd api && go test ./...

# Integration tests (requiere Docker, Postgres, Redis, Temporal corriendo)
cd api
DATABASE_URL=postgres://hangar:hangar@localhost:5432/hangar \
  go test ./tests/... -v -timeout 120s -tags integration
```

**Suite de integración cubre:**
- Health check del API y Postgres
- Creación de apps con generación de subdominio
- Unicidad de subdominios ante colisiones de nombre
- Flujo completo de deploy (clone -> build -> run -> logs en DB)
- Manejo de apps inexistentes (404)

---

## Stack

| Capa | Tecnología | Por qué |
|---|---|---|
| API | Go 1.22 + Fiber | Binario único, goroutines nativas, SDK oficial de Docker |
| Workflows | Temporal.io 1.24 | Resiliencia ante reinicios, visibilidad de estado, reintentos por step |
| Proxy | Traefik v3 | Routing dinámico via Docker labels, sin reinicio |
| Base de datos | PostgreSQL 16 | Event store + read models en una sola tecnología |
| Pub/Sub | Redis 7 | Canal de logs por deployment para WebSocket streaming |
| Containerización | Docker Engine API | Aislamiento de red por tenant, límites de recursos |
| Dashboard | Next.js 14 App Router | Server Components + WebSocket para logs en vivo |

---

## Architecture Decision Records

| ADR | Decisión | Estado |
|---|---|---|
| [001](docs/adr/001-por-que-go-y-temporal.md) | Go + Temporal sobre Node.js + BullMQ | Aceptado |
| [002](docs/adr/002-arquitectura-kappa.md) | Kappa Architecture sobre Lambda | Aceptado |
| [003](docs/adr/003-aislamiento-tenants.md) | Red Docker por tenant sobre red compartida | Aceptado |

---

## Requisitos

- Go 1.22+
- Docker Engine 24+ con Docker Compose v2
- Node.js 20+ (solo para el dashboard)
- Git (para `git clone` en las actividades)
- 4GB RAM disponibles para la infraestructura

---

## Roadmap

- [ ] Webhook de GitHub para auto-deploy en push
- [ ] Soporte para variables de entorno por app
- [ ] Logs de contenedor en vivo (post-deploy)
- [ ] Métricas de CPU/RAM por app en el dashboard
- [ ] Auth básica para el dashboard
- [ ] CLI: `hangar deploy ./mi-proyecto`
