# NATS JetStream Integration

This project uses an outbox pattern to publish `user.created` events to NATS JetStream.

## Components

- API server writes an outbox event in the same DB transaction as user creation.
- `outbox-worker` publishes outbox events to JetStream.
- `consumer` is a sample JetStream subscriber that logs `user.created` and stores an audit log.

## Config

Add this to `config.yaml` (or copy from `config.yaml.example`):

```yaml
nats:
  url: "nats://127.0.0.1:4222"
  stream: "events"
  user_created_subject: "user.created"
  dlq_subject: "user.created.dlq"
  consumer_durable: "user-created-worker"
  ack_wait: "30s"
  max_ack_pending: 256
  consumer_max_deliver: 10
  consumer_backoff: ["1s", "2s", "5s", "10s"]
outbox:
  batch_size: 100
  poll_interval: "2s"
  lock_timeout: "60s"
  max_attempts: 10
```

## Run

1) Migrate:

```sh
go run main.go migration up
```

2) Start the API:

```sh
go run main.go server
```

3) Start the outbox publisher:

```sh
go run main.go outbox-worker
```

4) (Optional) Start the sample consumer:

```sh
go run main.go consumer
```

## Audit Logs

The `consumer` inserts events into `audit_logs` with the raw JSON payload.

## Notes

- If NATS is down, events remain in `outbox_events` and will publish once NATS returns.
- The consumer uses `consumer_backoff` for retries and sends to `dlq_subject` after `consumer_max_deliver`.
- For external access, use port-forwarding:

```sh
kubectl -n nats-dev port-forward svc/nats-dev 4222:4222
```
