# Delayed Notifier

> Scheduled notification service in Go: an OpenAPI-first HTTP API stores notifications in PostgreSQL, a horizontally scalable worker claims due ones with `FOR UPDATE SKIP LOCKED` and dispatches them through RabbitMQ, and Redis serves hot reads.

![Go](https://img.shields.io/badge/Go-1.27-00ADD8?style=flat-square&logo=go&logoColor=white)
![RabbitMQ](https://img.shields.io/badge/RabbitMQ-FF6600?style=flat-square&logo=rabbitmq&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-4169E1?style=flat-square&logo=postgresql&logoColor=white)
![Redis](https://img.shields.io/badge/Redis-DC382D?style=flat-square&logo=redis&logoColor=white)
![OpenAPI](https://img.shields.io/badge/OpenAPI-oapi--codegen-6BA539?style=flat-square&logo=openapiinitiative&logoColor=white)

## Highlights

- **Contract-first HTTP API.** The spec lives in [`open-api.yaml`](internal/transport/http/api/notify/open-api.yaml); server interfaces and DTOs are generated with `oapi-codegen`.
- **Safe concurrent workers.** Pending notifications are claimed in batches with `SELECT … FOR UPDATE SKIP LOCKED`, so any number of worker replicas can run without double-sending.
- **RabbitMQ topology** declared in code: a main exchange, plus wait and retry queues that dead-letter back into it (per-message TTL + `x-dead-letter-exchange`). Notifications due within a short look-ahead window are claimed early and wait in the broker, so they are delivered on time, not on the next poll.
- **At-least-once delivery with the database as the source of truth.** Messages carry only the notification id; the dispatcher locks the row while sending, so duplicates and cancelled notifications are skipped. Failed sends are retried through the retry queue up to `max_attempts`, lost messages are re-claimed once stale.
- **Redis read-through cache** for `GET /notify/:id`, written off the hot path by a buffered async cache writer that degrades gracefully when the buffer is full.
- **Multi-channel model** — `email`, `telegram`, `sms` — enforced at the database level with `CHECK` constraints, so exactly one recipient field matches the channel.
- **Type-safe SQL** with `sqlc`, migrations with `goose`, Clean Architecture layering, graceful shutdown.

## Architecture

```mermaid
flowchart LR
    C[Client] -- "POST /api/notify" --> API[api]
    API --> PG[(PostgreSQL<br/>status = pending)]
    API -. cache .-> R[(Redis)]
    C -- "GET /api/notify/:id" --> API
    API -- read-through --> R

    W[worker × N] -- "claim due batch<br/>FOR UPDATE SKIP LOCKED" --> PG
    W -- publish --> X{{notifications exchange}}
    X --> Q[[notifications queue]]
    WQ[[wait / retry queues<br/>TTL]] -- dead-letter --> X
    Q --> D[dispatcher<br/>email · telegram · sms]
```

## Quick start

```bash
docker compose up --build
```

```bash
curl -X POST localhost:8080/api/notify -H 'Content-Type: application/json' -d '{
  "author_id": "user-1", "subject": "Reminder", "message": "Stand-up in 5 minutes",
  "channel": "email", "email_to": "user@example.com",
  "scheduled_at": "2030-01-01T09:55:00Z"
}'
```

| Service | URL |
|---|---|
| HTTP API | http://localhost:8080/api |
| RabbitMQ management | http://localhost:15672 |

## API

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/notify` | Schedule a notification (`channel`, recipient, `subject`, `message`, `scheduled_at`) |
| `GET` | `/api/notify/:id` | Get a notification and its status (`pending` / `sent` / `failed` / `declined`) |
| `DELETE` | `/api/notify/:id` | Cancel a pending notification (`409` once it is sent or failed) |
| `GET` | `/api/health` | Health check |

## Project layout

```
cmd/api, cmd/worker      # two deployables sharing the domain
internal/
  domain/                # notification model, value objects (channel, status, attempts), ports
  application/           # use cases
  infra/                 # postgres (sqlc, goose), rabbitmq, redis, senders, scheduler loop
  transport/             # HTTP (generated from OpenAPI) and RabbitMQ handlers
configs/                 # per-service YAML configs
```

Lifecycle, connections and transactions come from [`github.com/D1sordxr/packages`](https://github.com/D1sordxr/packages)
(`app`, `postgres`, `postgres/tx`, `rabbitmq`, `redis`, `httpserver`, `cron`).

Email, Telegram and SMS senders are currently logging stubs behind the `ports.Sender` interface;
real integrations plug in without touching the delivery pipeline.

## Development

```bash
go test -race ./...
```

Repository and cache integration tests run when the services are available. They migrate
the database and truncate `notifications`, so use throwaway instances:

```bash
docker run -d --rm --name dn-test-pg -e POSTGRES_PASSWORD=test -p 55432:5432 postgres:15-alpine
docker run -d --rm --name dn-test-redis -p 56379:6379 redis:7-alpine

DELAYED_NOTIFIER_TEST_POSTGRES_DSN='postgres://postgres:test@localhost:55432/postgres?sslmode=disable' \
DELAYED_NOTIFIER_TEST_REDIS_ADDR='localhost:56379' \
  go test -race ./internal/infra/...
```

Code generation:

```bash
cd internal/transport/http/api/notify && oapi-codegen --config=oapi-codegen.yaml open-api.yaml
cd internal/infra/storage/postgres/repositories/notification && sqlc generate
```

## Tech

Go · Gin · oapi-codegen · RabbitMQ (amqp091-go) · pgx / sqlc · goose · go-redis · zerolog · [D1sordxr/packages](https://github.com/D1sordxr/packages) · Docker Compose
