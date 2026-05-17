# Backend Development Environment

## Prerequisites

- **Go 1.26+** — The project uses modern Go features
- **golangci-lint** — For linting (`golangci-lint run`)
- **swag** — For Swagger doc generation (`swag init`)

## Running Tests

```bash
cd backend
go test ./...
```

Run a specific package:

```bash
go test ./internal/service/...
go test ./internal/api/rest/v1/...
```

Run with verbose output:

```bash
go test -v ./internal/service/post/...
```

## Dev Server

The recommended way to start the development environment is through the devops script:

```bash
python3 devops/dev.py start
```

This starts PostgreSQL and the backend/frontend services via Docker Compose.

For hot-reload during backend development, use [air](https://github.com/cosmtrek/air):

```bash
cd backend
air
```

## Linting

```bash
cd backend
golangci-lint run
```

## Swagger Documentation

Generate Swagger docs from annotations:

```bash
cd backend
swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal
```

Swagger UI is available at `/swagger/index.html` when running with `debug = true`.

## Database

The dev environment uses PostgreSQL via Docker Compose. The connection is configured in the dev docker-compose setup with these defaults:

- Host: `localhost`
- Port: `5432`
- Database: `markpost`
- User: `markpost`
- Password: `markpost`

The backend also supports SQLite for lightweight deployments. SQLite is the default driver when no configuration is provided. For local development without Docker, SQLite works with zero configuration.

## Configuration

See [configuration.md](./configuration.md) for the full configuration reference.

For local development, you can create a `markpost.toml` file or use environment variables. The minimum configuration needed is JWT signing keys.

## Building

```bash
cd backend
go build -o markpost-server ./cmd/server/
```
