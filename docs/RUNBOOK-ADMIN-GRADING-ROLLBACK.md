# 🆘 Runbook: Admin Grading Rollback & Data Recovery

## 1. Context
The admin grading service persists grade updates in two places:
1. `submissions` table (`grade`, `feedback`, `status`, `graded_at`, `graded_by`).
2. `submission_grade_histories` table (append-only audit trail).

Prometheus metrics (`admin_requests_total`, `admin_errors_total`) and OpenTelemetry traces (`grading.update`) provide live visibility into grading issues. Rollbacks should restore both the latest submission state and any incorrect history entries.

## 2. Detection
- **Alert triggers**: spike in `admin_errors_total` or latency >250ms for `admin_latency_seconds{route="/api/admin/submissions/:id/grade"}`.
- **User reports**: duplicated grades, missing feedback, or incorrect scores.
- **Trace inspection**: look for spans with `grading.update` status `Error` in the trace backend (e.g., Jaeger/Tempo).

## 3. Immediate Mitigation
1. **Freeze grading UI**: toggle the feature flag or set Cloudflare rule to return `503` for `/api/admin/submissions/*/grade`.
2. **Capture evidence**: export recent `submission_grade_histories` rows for the affected submission IDs.
3. **Notify stakeholders**: #lms-ops (Slack) and incident ticket in Jira (`OPS-GEMA`).

## 4. Rollback Procedure
1. **Identify good snapshot**
   - Query `submission_grade_histories` ordered by `graded_at` descending.
   - Select the last known good record (pre-incident timestamp).
2. **Restore submission row**
   ```sql
   UPDATE submissions
   SET grade = $1,
       feedback = $2,
       status = 'graded',
       graded_at = $3,
       graded_by = $4
   WHERE id = $5;
   ```
3. **Purge incorrect history (optional)**
   ```sql
   DELETE FROM submission_grade_histories
   WHERE submission_id = $1 AND graded_at > $2;
   ```
4. **Invalidate caches**
   - Delete Redis keys `analytics:summary` and any `student:dashboard:*` entries to force recompute.
5. **Replay activity log (if required)**
   - Use `/api/admin/activities` POST endpoint to append a corrective entry describing the manual rollback.

## 5. Data Recovery Checklist
- [ ] Submission row matches the restored grade & feedback.
- [ ] Audit trail only contains validated history entries.
- [ ] Analytics dashboard refreshed and reflects corrected aggregates.
- [ ] Prometheus alert cleared (latency & errors back to baseline).
- [ ] Trace sampling shows successful `grading.update` spans for post-rollback attempts.

## 6. Verification Tests
- Run `go test ./tests/integration -run AdminEndToEndFlow` to ensure grading + analytics pipeline passes.
- Execute contract test `go test ./tests/contract` to confirm API envelope unchanged.
- Trigger a manual grading via staging UI and confirm metrics + traces.

## 7. Communication & Closure
- Post incident summary in #lms-ops with restored submission IDs, time range affected, and verification steps executed.
- Update the incident ticket with SQL commands executed and attach the exported audit trail.
- Remove the grading freeze flag once verification passes.
