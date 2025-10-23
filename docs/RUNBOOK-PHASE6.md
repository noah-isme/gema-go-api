# Phase 6 Operational Runbook

## Contact Delivery Failure
1. Check the application logs for entries tagged with `component=contact_service` and `level=warn` to identify failing submissions.
2. Verify Redis availability; dedupe failures surface as `duplicate submission` errors.
3. If the configured inbox provider is unavailable, switch to the fallback logging provider by toggling `CONTACT_INBOX_PROVIDER` and redeploy.
4. Re-run the delivery workflow by replaying the stored `contact_submissions` row using the reference ID.

## Contact Form Spam Detection
1. Inspect the Prometheus metric `contact_submissions_total{status="spam"}` and Loki logs for `honeypot tripped` annotations.
2. Confirm the honeypot field is rendered correctly on the frontend; rollout regression may expose it to real users.
3. Enable stricter rate limiting temporarily by setting `CONTACT_RATE_LIMIT=3` (per 5 minutes) and redeploying.
4. Rotate the spam keyword allowlist stored in `CONTACT_SPAM_ALLOWLIST` and invalidate cached dedupe keys with `redis-cli --scan --pattern 'contact:dedupe:*' | xargs redis-cli DEL`.
5. Once the spike subsides, restore the default rate limit and monitor `contact_submissions_total` for a balanced sent/spam ratio.

## Upload Quota Exceeded
1. Inspect `upload_requests_total` and `upload_rejected_total` metrics in Grafana to determine the rejection reason (size/type/scan/storage).
2. Adjust the `UPLOAD_MAX_MB` environment variable if a quota increase is approved and redeploy.
3. Communicate limits to clients via documentation and retry the upload after validating file size and MIME type locally.
4. If the rejection reason is `scan`, review the antivirus logs (`component=upload_service` `reason=scan`) and update the safe file extension allowlist.
5. Confirm successful remediation by running an integration upload test (`go test ./tests/integration -run UploadFlow`).

## Cache Flush (Announcements & Activities)
1. Trigger a Redis scan for keys matching `announcements:active:*` or `activities:active:*` and delete them manually using `redis-cli`.
2. Alternatively, wait for TTL expiry (default 45s for activities, configurable via `ANNOUNCEMENTS_CACHE_TTL`).
3. Confirm cache refill by issuing a GET request and observing `X-Cache-Hit: false` on the initial response.

## Seed Rollback
1. Ensure `SEED_ENABLED=true` and obtain the current `X-Seed-Token` value.
2. Re-seed with the last known good payload by POSTing to `/api/seed/announcements` or `/api/seed/gallery` with the rollback dataset.
3. Clear Redis caches for affected resources (`redis-cli --scan --pattern 'announcements:*' | xargs redis-cli DEL`).
4. Validate the change via the public endpoints and audit logs emitted by the seed handler (`component=seed_handler`).
5. Capture a post-rollback snapshot in the change ticket with timestamps and record counts for auditing.
