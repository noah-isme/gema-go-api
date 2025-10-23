# Changelog

## v0.6.0-alpha
- Added public activity, announcements, and gallery endpoints with Redis caching.
- Introduced contact form submission flow with rate limiting, spam protection, and delivery instrumentation.
- Implemented authenticated upload endpoint with MIME validation, checksum reporting, and storage metadata persistence.
- Added seeding utilities guarded by `SEED_ENABLED` and token authentication.
- Expanded observability with new Prometheus metrics and tracing spans for Phase 6 endpoints.
