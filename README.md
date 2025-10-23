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

## Web Lab Workflow

Phase 3 introduces the Web Lab for HTML/CSS/JS assignments. Students can retrieve available assignments and upload a `.zip` archive that is automatically linted, scanned for dangerous files, uploaded to Cloudinary, and scored based on heuristic Lighthouse checks.

### API Endpoints

| Method | Path                               | Description                         |
|--------|------------------------------------|-------------------------------------|
| GET    | `/api/v2/web-lab/assignments`      | List available Web Lab assignments  |
| GET    | `/api/v2/web-lab/assignments/:id`  | Retrieve a single assignment        |
| POST   | `/api/v2/web-lab/submissions`      | Upload a `.zip` submission (JWT required) |

Submission requests must use `multipart/form-data` with fields:

| Field           | Type   | Notes                                                       |
|-----------------|--------|-------------------------------------------------------------|
| `assignment_id` | number | Target assignment ID                                        |
| `file`          | file   | `.zip` archive ≤ 10 MB containing HTML, CSS, and JS sources |

Example project archive layout:

```
landing-page.zip
├── index.html
├── styles/
│   └── style.css
└── scripts/
    └── app.js
```

The service rejects executables (`.exe`) and symbolic links to harden the sandbox pipeline. For full request/response samples see [`docs/WEB-LAB-API.md`](docs/WEB-LAB-API.md).

