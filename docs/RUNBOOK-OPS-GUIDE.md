# GEMA LMS Operations Guide

This runbook consolidates Phase 6 operational procedures for the GEMA Golang API. Each section contains the exact commands and validation steps required to recover from the most common incidents identified during the migration cut-over.

> **Prerequisites**
> * PostgreSQL superuser access (via `psql` or Cloud SQL Proxy).
> * Redis CLI access to the primary cache cluster.
> * Observability stack access (Grafana, Loki, Tempo/Jaeger).
> * The current `.env` or secret manager entry containing API tokens and feature flags.

## 1. Database Rollback (Hotfix / Failed Deployment)

1. **Identify target migration**
   - List applied migrations: `SELECT version, dirty FROM schema_migrations ORDER BY version DESC LIMIT 5;`
   - Confirm the desired rollback version from the deployment ticket.
2. **Enter maintenance mode**
   - Enable Cloudflare maintenance page for `/api/*`.
   - Set `MAINTENANCE_MODE=true` in the environment and redeploy (disables student writes).
3. **Perform rollback**
   - Use goose: `goose -dir migrations postgres "$DATABASE_URL" down` until the target version.
   - Alternatively, execute the stored rollback script linked in the deployment ticket.
4. **Data verification**
   - Run smoke queries for critical tables:
     ```sql
     SELECT COUNT(*) FROM students;
     SELECT COUNT(*) FROM coding_submissions WHERE created_at > now() - interval '1 hour';
     ```
   - Ensure counts align with pre-rollback snapshots (refer to Grafana panel `db.new_records`).
5. **Exit maintenance mode**
   - Revert `MAINTENANCE_MODE=false`, redeploy, and disable Cloudflare maintenance page.
   - Monitor `api_requests_error_total` for 15 minutes to confirm stability.

## 2. Seed Data Recovery

1. **Verify seed toggle** – Ensure `SEED_ENABLED=true` and retrieve `X-Seed-Token` from secrets.
2. **Determine dataset** – Export the last known good payload from the audit log (`component=seed_handler action=seed.apply`).
3. **Replay seed endpoint**
   ```bash
   curl -X POST "$API_BASE/api/seed/announcements" \
     -H "Authorization: Bearer $ADMIN_TOKEN" \
     -H "X-Seed-Token: $SEED_TOKEN" \
     -H "Content-Type: application/json" \
     -d @announcements-backup.json
   ```
   Repeat for gallery or additional seed scopes as required.
4. **Validate** – Query public endpoints (`/api/announcements`, `/api/gallery`) and confirm `success=true` with restored data. Check Loki logs for `seed_handler` confirmation.

## 3. Redis Cache Flush & Rebuild

1. **Scope keys** – Use `redis-cli -n $CACHE_DB --scan --pattern 'analytics:*'` to preview impact.
2. **Take snapshot** – Run `redis-cli -n $CACHE_DB SAVE` (or confirm latest RDB snapshot) before deleting keys.
3. **Flush strategy**
   - Targeted flush: `redis-cli -n $CACHE_DB --scan --pattern 'analytics:*' | xargs redis-cli -n $CACHE_DB DEL`.
   - Full flush (last resort): `redis-cli -n $CACHE_DB FLUSHDB`.
4. **Warm cache**
   - Hit `/api/admin/analytics` and `/api/v2/student/dashboard` with admin/student tokens to repopulate caches.
   - Verify `X-Cache-Hit: false` on the first request and `true` afterwards.
5. **Monitoring** – Track `cache_rebuild_seconds` and `redis_errors_total` dashboards for spikes.

## 4. Upload Quota Escalation

1. **Confirm rejection reason** – Inspect Grafana panel `upload_rejected_total` grouped by label (`reason`).
2. **Adjust limits**
   - Update environment: `UPLOAD_MAX_MB=25` (example) and redeploy API.
   - For long-term increase, submit storage budget approval before change.
3. **Communicate** – Notify frontend team via #lms-admin with the updated limit and rollout time.
4. **Post-change validation** – Upload a sample file within the new quota and ensure a `success=true` response.

## 5. SSE Reconnect Storms (Real-Time Dashboard)

1. **Detection** – Alerts fire when `sse_reconnect_total` exceeds baseline or `connection_errors_total` spikes.
2. **Immediate action**
   - Scale out API pods: `kubectl scale deploy/gema-api --replicas=6`.
   - Increase Nginx worker connections if applicable (`PROXY_MAX_CONNECTIONS`).
3. **Client mitigation**
   - Toggle feature flag `REALTIME_STREAM_ENABLED=false` to pause new SSE streams.
   - Communicate fallback polling interval to frontend (set to 30s).
4. **Root cause analysis**
   - Review Tempo traces for `component=notification_stream` spans with errors.
   - Inspect Redis for backlog in `notifications:stream` keys.
5. **Recovery** – Re-enable `REALTIME_STREAM_ENABLED` once `sse_active_connections` normalizes.

## 6. WebSocket Error Spikes (Coding Lab Collaboration)

1. **Identify symptom** – CloudWatch/Grafana alert on `websocket_disconnect_total` or latency > 1s for `ws.coding_lab`.
2. **Triage**
   - Check pod logs for `component=collab_hub` errors (`rate-limit`, `auth-failed`).
   - Validate JWT issuance time; expired tokens cause forced disconnects.
3. **Mitigation steps**
   - Increase rate-limit bucket: set `COLLAB_WS_RATE=30` and redeploy.
   - If auth failures, rotate `GEMA_JWT_SECRET` and `GEMA_JWT_REFRESH_SECRET`, invalidate old sessions.
   - Purge stale rooms: `redis-cli -n $CACHE_DB --scan --pattern 'collab:room:*' | xargs redis-cli -n $CACHE_DB DEL`.
4. **Verification** – Observe `websocket_active_sessions` returning to baseline and run end-to-end test `go test ./tests/integration -run CodingCollaborationSuite`.
5. **Post-incident** – Document findings in Confluence and attach Grafana snapshots to the incident ticket.

## 7. NATS Streaming Operations (Real-Time Chat & Notifications)

1. **Connection health** – Inspect Grafana panel `nats_connection_uptime_seconds` and Loki logs tagged `component=chat_service`/`notification_service` for `nats: connection lost` errors.
2. **Force reconnect** – Run `kubectl exec deploy/gema-api -- pkill -f nats` to trigger a reconnect when the server is healthy. Clients automatically backoff with jitter (250 ms → 2 s).
3. **Credential rotation** – Update `NATS_URL`, `NATS_USERNAME`, and `NATS_PASSWORD` in the secret store. Redeploy API pods and confirm a new connection ID via the NATS monitoring endpoint (`/connz`).
4. **Queue drain** – Drain the `gema.realtime.chat` and `gema.realtime.notifications` subjects with:
   ```bash
   nats --server "$NATS_URL" stream purge GEMA_CHAT
   nats --server "$NATS_URL" stream purge GEMA_NOTIFICATIONS
   ```
   Afterwards, replay critical messages by republishing from the audit log if necessary.
5. **Verification** – Ensure `chat_messages_sent` and `notifications_published_total` increase within 2 minutes and WebSocket/SSE clients reconnect successfully.

## 8. Redis Pub/Sub Backlog Recovery

1. **Detect backpressure** – Alerts trigger when `redis_pubsub_pending_messages` exceeds baseline. Confirm using `redis-cli PUBSUB NUMSUB notifications:stream`.
2. **Drain stuck consumers** – Restart the API pods subscribed to the channel:
   ```bash
   kubectl rollout restart deploy/gema-api
   ```
   Monitor pod logs for `component=notification_service` with `resubscribed=true`.
3. **Rebuild consumer groups** – If backlog persists, flush the ephemeral keys `notifications:stream:*` and `chat:stream:*` with `redis-cli --scan --pattern 'notifications:stream:*' | xargs redis-cli DEL`.
4. **Replay latest events** – Publish the most recent notification payload via `curl -X POST $API_BASE/api/admin/notifications` using the archived JSON from Loki.
5. **Stability check** – Validate that `sse_clients_active` and `chat_connections_total` return to normal values and that no new Redis warnings appear.

---

Maintain this guide alongside `docs/RUNBOOK-PHASE6.md` and `docs/RUNBOOK-ADMIN-GRADING-ROLLBACK.md`. Update after each release that introduces new operational toggles or dependencies.
