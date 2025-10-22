# GEMA Golang API

This repository hosts the standalone Golang backend for the GEMA Learning Management System migration.

## Project Layout

The initial project layout follows the structure defined in the migration plan:

```
cmd/api            # Application entrypoint
internal/config    # Configuration loading utilities
internal/database  # Database connectivity helpers
internal/handler   # HTTP handlers
internal/router    # HTTP router wiring
internal/utils     # Shared helpers (response format, etc.)
tests              # Unit and integration tests
```

## Getting Started

1. Duplicate `.env.example` into `.env` and adjust the values to match your environment.
2. Install dependencies and verify the codebase:

```bash
go mod tidy
go test ./...
```

3. Run the development server:

```bash
GEMA_JWT_SECRET=dev-secret \
GEMA_JWT_REFRESH_SECRET=dev-refresh \
go run ./cmd/api
```

The API exposes a health check at `GET /api/v1/health`.

## Testing

Run the unit tests with:

```bash
go test ./...
```

## Linting & Formatting

Use the Go toolchain to lint and format the project:

```bash
go fmt ./...
go vet ./...
```

