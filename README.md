# go-impl-postgres-ha

Go API for users with Postgres (HA), idempotency, outbox, and NATS JetStream integration.

## Features

- CRUD Users API (Gin + GORM)
- Idempotency key for `POST /api/users`
- Outbox pattern for reliable event publishing
- JetStream consumer with retry + DLQ
- Cursor-based pagination for users

## Requirements

- Go (see `go.mod`)
- Postgres
- NATS (JetStream enabled)

## Configuration

Copy `config.yaml.example` to `config.yaml` and update values.

```sh
cp config.yaml.example config.yaml
```

Key settings:
- `database.write_dsn` / `database.read_dsn`
- `nats.url`
- `outbox.*`
- `environment` (`dev` or `prod`)

## Run locally

```sh
go run main.go migration up --config config.yaml
go run main.go server --config config.yaml
go run main.go outbox-worker --config config.yaml
go run main.go consumer --config config.yaml
```

## API

- Health: `GET /healthz`
- Create user: `POST /api/users`
- Get user: `GET /api/users/:id`
- Update user: `PATCH /api/users/:id`
- Delete user: `DELETE /api/users/:id`
- List users: `GET /api/users?limit=50&cursor=<cursor>`

## Idempotency

`POST /api/users` requires an idempotency key header:
- `Idempotency-Key` (or `X-Idempotency-Key`)
- For tests only: `X-Test-Bypass-Idempotency: true` (disabled in prod)

## Outbox + NATS

Flow:
1) API writes user + outbox event in the same DB transaction
2) `outbox-worker` publishes events to JetStream
3) `consumer` writes audit logs to `audit_logs`

## Docker

```sh
docker build -t ghcr.io/daffahilmyf/go-impl-postgres-ha:latest .
docker run --rm -p 8080:8080 ghcr.io/daffahilmyf/go-impl-postgres-ha:latest server
```

## Kustomize

```sh
kubectl apply -k kustomize/overlays/dev
```

## Makefile

```sh
make help
```
