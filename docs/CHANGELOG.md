# Changelog

## v1.0.0
- Published combined Coding Lab & Web Lab OpenAPI specification (`docs/api/labs.json`) for FE contract tests.
- Promoted admin contract to v1.0.0 with updated description of analytics caching and activity feed coverage.
- Authored consolidated operations guide (`docs/RUNBOOK-OPS-GUIDE.md`) covering database rollback, seed recovery, Redis flush, SSE reconnect, and WebSocket troubleshooting.
- Documented runbook cross-links in the README alongside CI release tagging updates for `v1.0.0`.
- Added realtime (`docs/api/realtime.json`) and supporting surface (`docs/api/supporting.json`) OpenAPI contracts including SSE/WebSocket examples for contract validation.
- Introduced realtime Prometheus alerts (`docs/observability/realtime-alerts.yaml`) and instrumented chat/SSE error and disconnect metrics.
- Expanded runbooks with NATS reconnect/drain, Redis pub/sub backlog recovery, contact spam detection, upload quota escalation, and seed rollback verification flows.
- Extended CI/CD to publish OpenAPI artifacts, run realtime load tests, and tag both `v0.6.0-alpha` and `v1.0.0` releases automatically.

## v0.6.0-alpha
- Added public activity, announcements, and gallery endpoints with Redis caching.
- Introduced contact form submission flow with rate limiting, spam protection, and delivery instrumentation.
- Implemented authenticated upload endpoint with MIME validation, checksum reporting, and storage metadata persistence.
- Added seeding utilities guarded by `SEED_ENABLED` and token authentication.
- Expanded observability with new Prometheus metrics and tracing spans for Phase 6 endpoints.
