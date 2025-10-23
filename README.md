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

## CI/CD Pipeline

Automated checks run on every pull request and push to `main` via GitHub Actions (`.github/workflows/ci.yml`). The pipeline executes the following stages sequentially:

1. **Lint** – verifies formatting (`gofmt -l`) and runs `go vet ./...`.
2. **Unit & Integration Tests** – executes `go test ./...` covering service, handler, and database-backed flows.
3. **Contract Tests** – validates the admin analytics response envelope against the published JSON schema (`go test ./tests/contract`).
4. **Build** – compiles the API binary (`go build ./cmd/api`).
5. **Deploy Staging** – triggers the staging deployment script after all checks succeed.
6. **Release Tagging** – automatically creates or updates the `v0.4.0-alpha` tag pointing at the passing commit on `main`.

The workflow mirrors the local developer checklist: `go fmt ./... && go vet ./... && go test ./... && go build ./cmd/api`.

## Frontend Integration (Admin LMS UI)

Frontend clients consume the admin APIs via the OpenAPI contract located at [`docs/api/admin.json`](docs/api/admin.json).

- **Authentication** – include the JWT access token in the `Authorization: Bearer <token>` header.
- **Correlation IDs** – forward the `X-Correlation-ID` header to preserve trace continuity with the backend logs and metrics.
- **Error Handling** – responses follow the `{ success, message, data }` envelope; check `success` before accessing payload fields.
- **Caching Hints** – analytics endpoints surface the `cache_hit` flag to determine whether to refresh dashboards aggressively.
- **Telemetry** – Prometheus counters/histograms (`admin_requests_total`, `admin_latency_seconds`, `admin_errors_total`) expose request patterns and error rates for UI observability dashboards. Metrics are published via the shared `/metrics` endpoint.

## Web Lab Workflow

Phase 3 introduces the Web Lab for HTML/CSS/JS assignments. Students can retrieve available assignments and upload a `.zip` archive that is automatically linted, scanned for dangerous files, uploaded to Cloudinary, and scored based on heuristic Lighthouse checks.

### API Endpoints

| Method | Path                               | Description                         |
|--------|------------------------------------|-------------------------------------|
| GET    | `/api/v2/web-lab/assignments`      | List available Web Lab assignments  |
| GET    | `/api/v2/web-lab/assignments/:id`  | Retrieve a single assignment        |
| POST   | `/api/v2/web-lab/submissions`      | Upload a `.zip` submission (JWT required) |

### Admin API Endpoints

All administrative routes require a valid JWT with the `admin` or `teacher` role. Responses follow the standardized envelope (`success`, `message`, and `data`).

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/admin/students` | Paginated student directory with search & filters |
| GET | `/api/admin/students/:id` | Retrieve a single student profile |
| PATCH | `/api/admin/students/:id` | Update student metadata, flags, and status |
| DELETE | `/api/admin/students/:id` | Soft-delete a student with audit logging |
| POST | `/api/admin/assignments` | Create tutorial assignments with rubric & max score |
| PATCH | `/api/admin/assignments/:id` | Update assignment metadata |
| DELETE | `/api/admin/assignments/:id` | Delete an assignment (cascades submissions) |
| PATCH | `/api/admin/submissions/:id/grade` | Grade or re-grade a submission (idempotent) |
| GET | `/api/admin/analytics` | Aggregated platform analytics with caching |
| GET | `/api/admin/activities` | List administrative activity logs |
| POST | `/api/admin/activities` | Manually append an activity log entry |

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

