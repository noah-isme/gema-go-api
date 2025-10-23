# Phase 6 Operational Runbook

## Contact Delivery Failure
1. Check the application logs for entries tagged with `component=contact_service` and `level=warn` to identify failing submissions.
2. Verify Redis availability; dedupe failures surface as `duplicate submission` errors.
3. If the configured inbox provider is unavailable, switch to the fallback logging provider by toggling `CONTACT_INBOX_PROVIDER` and redeploy.
4. Re-run the delivery workflow by replaying the stored `contact_submissions` row using the reference ID.

## Upload Quota Exceeded
1. Inspect `upload_requests_total` and `upload_rejected_total` metrics in Grafana to determine the rejection reason (size/type/scan/storage).
2. Adjust the `UPLOAD_MAX_MB` environment variable if a quota increase is approved and redeploy.
3. Communicate limits to clients via documentation and retry the upload after validating file size and MIME type locally.

## Cache Flush (Announcements & Activities)
1. Trigger a Redis scan for keys matching `announcements:active:*` or `activities:active:*` and delete them manually using `redis-cli`.
2. Alternatively, wait for TTL expiry (default 45s for activities, configurable via `ANNOUNCEMENTS_CACHE_TTL`).
3. Confirm cache refill by issuing a GET request and observing `X-Cache-Hit: false` on the initial response.

## Seed Rollback
1. Ensure `SEED_ENABLED=true` and obtain the current `X-Seed-Token` value.
2. Re-seed with the last known good payload by POSTing to `/api/seed/announcements` or `/api/seed/gallery` with the rollback dataset.
3. Validate the change via the public endpoints and audit logs emitted by the seed handler (`component=seed_handler`).
