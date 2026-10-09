# Go Service Starter

![Go](https://img.shields.io/badge/Go-1.26.9-00ADD8?logo=go&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-Ready-4169E1?logo=postgresql&logoColor=white)
![Docker](https://img.shields.io/badge/Docker-Multi--stage-2496ED?logo=docker&logoColor=white)
![Kubernetes](https://img.shields.io/badge/Kubernetes-Manifests-326CE5?logo=kubernetes&logoColor=white)

A production-oriented Go service starter that demonstrates the engineering practices expected from a backend service that is small enough to understand and complete enough to discuss in a professional code review.

The domain is intentionally simple. The point of this repository is not the `items` API itself, but the production foundation around it: typed configuration, graceful HTTP lifecycle, PostgreSQL access, migrations, tests, structured logs, metrics, tracing, Docker, CI, and Kubernetes examples.

## Features

- HTTP API built with Go's standard `net/http` package.
- Environment-driven typed configuration with validation.
- Production-oriented HTTP server timeouts and graceful shutdown.
- Consistent JSON responses for success and error cases.
- Request ID middleware and structured access logs with `log/slog`.
- PostgreSQL connection pooling through `pgxpool`.
- Versioned SQL migrations.
- Separate liveness and readiness endpoints.
- Minimal `items` feature with layered handler, service, and repository code.
- Unit tests and opt-in PostgreSQL integration tests.
- Prometheus-compatible metrics at `GET /metrics`.
- OpenTelemetry tracing with W3C trace context propagation.
- Multi-stage Dockerfile with a non-root runtime container.
- Docker Compose stack for local API, PostgreSQL, and migrations.
- GitHub Actions CI with formatting, tests, vet, lint, build, and Docker build checks.
- Basic Kubernetes manifests with probes, resources, security context, and ConfigMap/Secret separation.

## What This Demonstrates

This repository is designed to be evaluated as a backend engineering portfolio project.

It demonstrates:

- How to structure a small Go service without turning it into a framework.
- How to separate HTTP concerns, business rules, persistence, and platform infrastructure.
- How to make runtime behavior observable through logs, metrics, and traces.
- How to make local development reproducible with Docker Compose.
- How to think about production readiness: timeouts, readiness, migrations, CI, security checks, and deployment examples.

## API

```text
GET  /health
GET  /ready
GET  /metrics
POST /items
GET  /items/{id}
```

`GET /health` is a liveness endpoint. It only confirms that the process is running.

```bash
curl http://localhost:8080/health
```

```json
{
  "status": "ok",
  "service": "go-service-starter"
}
```

`GET /ready` is a readiness endpoint. It checks whether the service can receive traffic, including PostgreSQL availability.

```bash
curl http://localhost:8080/ready
```

When PostgreSQL is available:

```json
{
  "status": "ready",
  "service": "go-service-starter",
  "dependencies": {
    "database": "ok"
  }
}
```

When PostgreSQL is not configured or unavailable, the endpoint returns `503 Service Unavailable`.

## Getting Started

### Requirements

- Go 1.26.9 or newer.
- Docker and Docker Compose for the local full stack.
- `psql` only if you want to run migrations manually.
- `kubectl` only if you want to inspect or apply the Kubernetes examples.

### Run Without PostgreSQL

This is useful for checking startup, liveness, logging, metrics, and tracing without a database:

```bash
go run ./cmd/api
```

The API starts at:

```text
http://localhost:8080
```

In this mode:

```text
GET /health -> 200
GET /ready  -> 503
POST /items -> 503
```

That behavior is intentional. The process is alive, but database-backed behavior is not ready.

### Run The Full Local Stack

```bash
docker compose up --build
```

The Compose stack starts:

- `db`: PostgreSQL for local development.
- `migrate`: one-shot migration runner.
- `api`: the Go service built from the local Dockerfile.

Check the service:

```bash
curl http://localhost:8080/health
curl http://localhost:8080/ready
curl http://localhost:8080/metrics
```

Stop the stack:

```bash
docker compose down
```

Remove the local PostgreSQL volume too:

```bash
docker compose down -v
```

## Configuration

Configuration is loaded from environment variables and validated at startup.

Copy the example file when you want a local `.env` reference:

```bash
cp .env.example .env
```

The application does not automatically load `.env` files. Export variables in your shell, configure your process manager, or use Docker Compose/Kubernetes environment configuration.

| Variable | Default | Description |
| --- | --- | --- |
| `APP_NAME` | `go-service-starter` | Service name used in responses and telemetry. |
| `APP_ENV` | `development` | Runtime environment: `development`, `test`, `staging`, or `production`. |
| `HTTP_HOST` | `127.0.0.1` | Address bound by the HTTP server. Containers should use `0.0.0.0`. |
| `HTTP_PORT` | `8080` | HTTP server port. |
| `HTTP_READ_TIMEOUT` | `10s` | Maximum duration for reading the request. |
| `HTTP_READ_HEADER_TIMEOUT` | `5s` | Maximum duration for reading headers. |
| `HTTP_WRITE_TIMEOUT` | `10s` | Maximum duration for writing a response. |
| `HTTP_IDLE_TIMEOUT` | `60s` | Maximum keep-alive idle time. |
| `HTTP_MAX_BODY_BYTES` | `1048576` | Maximum accepted request body size. |
| `LOG_LEVEL` | `info` | Log level: `debug`, `info`, `warn`, or `error`. |
| `DATABASE_URL` | empty | PostgreSQL connection URL. Required when `APP_ENV=production`. |
| `DATABASE_CONNECT_TIMEOUT` | `5s` | Startup database connection timeout. |
| `DATABASE_READINESS_TIMEOUT` | `2s` | Per-request readiness check timeout for PostgreSQL. |
| `TRACING_ENABLED` | `false` | Enables OpenTelemetry tracing. |
| `TRACING_EXPORTER` | `stdout` | Trace exporter. Currently supports `stdout`. |
| `TRACING_SAMPLE_RATIO` | `1` | Sampling ratio between `0` and `1`. |
| `SHUTDOWN_TIMEOUT` | `10s` | Graceful shutdown timeout. |

Example:

```bash
APP_ENV=development HTTP_PORT=9090 go run ./cmd/api
```

## Items API

The `items` feature is intentionally minimal. It exists to demonstrate validation, layering, persistence, tests, errors, metrics, and tracing around a real endpoint.

Create an item:

```bash
curl -X POST http://localhost:8080/items \
  -H 'Content-Type: application/json' \
  -d '{"name":"Example item"}'
```

Example response:

```json
{
  "id": "018fb3b2-0f9d-4f59-8a63-7ef4bb812345",
  "name": "Example item",
  "created_at": "2026-08-23T13:30:00Z",
  "updated_at": "2026-08-23T13:30:00Z"
}
```

Fetch an item:

```bash
curl http://localhost:8080/items/018fb3b2-0f9d-4f59-8a63-7ef4bb812345
```

Behavior:

```text
POST /items      -> 201
POST /items      -> 400 when input is invalid
GET /items/{id}  -> 200
GET /items/{id}  -> 400 when id is not a valid UUID
GET /items/{id}  -> 404 when the item does not exist
```

Validation:

- `name` is required.
- `name` is trimmed before persistence.
- `name` must not exceed 200 characters.
- Item IDs must be valid UUIDs.
- Unknown JSON fields and multiple JSON objects are rejected.

## Database Migrations

Migrations live in `migrations/`.

Current migrations:

```text
000001_create_items_table.up.sql
000001_create_items_table.down.sql
```

Apply the migration manually:

```bash
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f migrations/000001_create_items_table.up.sql
```

Roll it back manually:

```bash
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f migrations/000001_create_items_table.down.sql
```

Docker Compose applies the current migration automatically through the `migrate` service.

## Testing

Run the default test suite:

```bash
go test ./...
```

The default suite includes unit tests and HTTP handler tests. PostgreSQL integration tests are opt-in and are skipped unless `INTEGRATION_DATABASE_URL` is set.

Run integration tests:

```bash
INTEGRATION_DATABASE_URL='postgres://app:app@localhost:5432/app?sslmode=disable' go test -run Integration -v ./internal/items
```

The command above uses the local database started by Docker Compose. Integration tests create a temporary schema named `test_items_*`, run repository operations inside that schema, and drop the schema during cleanup. Use a local or disposable database, never a production database.

## Observability

### Metrics

Prometheus-compatible metrics are exposed at:

```bash
curl http://localhost:8080/metrics
```

Application HTTP metrics:

```text
http_requests_total
http_request_duration_seconds
```

Metrics use `method`, `route`, and `status` labels. Route labels are normalized to avoid high-cardinality series and accidental data exposure. For example, `/items/<uuid>?token=secret` is recorded as `/items/{id}` without the query string.

### Tracing

Tracing is disabled by default. Enable stdout tracing locally:

```bash
TRACING_ENABLED=true go run ./cmd/api
```

The HTTP middleware creates server spans for every request and extracts incoming W3C `traceparent` headers. The resulting context is propagated through handlers and into the `items` repository.

Repository spans:

```text
items.postgres.create
items.postgres.find_by_id
```

Traces intentionally avoid request bodies, headers, query strings, raw item IDs, SQL statements, and SQL parameters.

Example:

```bash
curl http://localhost:8080/health \
  -H 'traceparent: 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01'
```

## Docker

Build the image:

```bash
docker build -t go-service-starter:local .
```

Run without PostgreSQL:

```bash
docker run --rm -p 127.0.0.1:8080:8080 go-service-starter:local
```

Run with PostgreSQL configured:

```bash
docker run --rm -p 127.0.0.1:8080:8080 \
  -e DATABASE_URL='postgres://app:app@host.docker.internal:5432/app?sslmode=disable' \
  go-service-starter:local
```

The Dockerfile uses:

- A Go Alpine build stage.
- A small Alpine runtime stage.
- A statically linked Linux binary.
- CA certificates for outbound TLS.
- A non-root `app` user.
- A container `HEALTHCHECK` on `GET /health`.

For production containers, set `APP_ENV=production` and use a real PostgreSQL URL with the TLS mode required by your provider. Do not bake secrets into the image. Pass secrets through the runtime environment, Kubernetes Secrets, or a secret manager.

## Kubernetes

Basic Kubernetes examples live in:

```text
deployments/kubernetes/
```

Included resources:

```text
namespace.yaml
configmap.yaml
secret.example.yaml
deployment.yaml
service.yaml
kustomization.yaml
```

The Deployment demonstrates:

- `GET /health` liveness probe.
- `GET /ready` readiness probe.
- Two API replicas.
- Basic CPU and memory requests/limits.
- Non-root execution.
- Disabled service account token mounting.
- Dropped Linux capabilities.
- Read-only root filesystem.
- Prometheus scrape annotations for `/metrics`.

Before deployment, publish an image and replace this placeholder in `deployment.yaml`:

```text
ghcr.io/your-org/go-service-starter:latest
```

Create the namespace:

```bash
kubectl apply -f deployments/kubernetes/namespace.yaml
```

Create a real local Secret manifest from the example:

```bash
cp deployments/kubernetes/secret.example.yaml deployments/kubernetes/secret.yaml
```

Edit `deployments/kubernetes/secret.yaml` and set a real `DATABASE_URL`. This file is ignored by Git.

Apply the Secret and the API resources:

```bash
kubectl apply -f deployments/kubernetes/secret.yaml
kubectl apply -k deployments/kubernetes
```

Check rollout status:

```bash
kubectl -n go-service-starter rollout status deployment/go-service-starter
kubectl -n go-service-starter get pods
```

Test through port forwarding:

```bash
kubectl -n go-service-starter port-forward service/go-service-starter 8080:80
curl http://localhost:8080/health
curl http://localhost:8080/ready
```

The Kubernetes examples do not deploy PostgreSQL and do not run migrations. Use a managed PostgreSQL instance or a separate database deployment, then run migrations as part of the release process.

Render manifests locally:

```bash
kubectl kustomize deployments/kubernetes
```
