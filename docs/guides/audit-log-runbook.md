---
status: active
doc_type: guide
owner: engineering
last_reviewed: 2026-04-30
source_of_truth:
  - server/jobs/audit_archive.go
  - server/internal/service/audit_archive.go
  - server/internal/service/audit_service.go
  - server/internal/model/audit_event.go
  - server/internal/model/task.go
  - server/internal/service/task.go
---

# Audit Log Runbook

## Scope

This runbook covers:

- audit write path health (`audit_event`)
- audit export task health (`audit_export_task`)
- archive task health (`audit_archive_daily`)
- incident response and recovery steps

## Primary Signals

Track these daily:

1. Audit write failures
2. Export task failure ratio
3. Archive task success/failure and row throughput
4. Security category event spikes

## Operational Queries

### 1) Audit write failures in last 24h

```sql
SELECT COUNT(*) AS failed_count
FROM audit_event
WHERE occurred_at >= NOW() - INTERVAL '24 hours'
  AND result = 'failed';
```

### 2) Export task failure ratio in last 24h

```sql
SELECT
  COUNT(*) FILTER (WHERE status = 'failed') AS failed_tasks,
  COUNT(*) AS total_tasks
FROM audit_export_task
WHERE created_at >= NOW() - INTERVAL '24 hours';
```

### 3) Archive task execution status (from task history)

```sql
SELECT task_name, status, started_at, finished_at, message
FROM task_executions
WHERE task_name = 'audit_archive_daily'
ORDER BY started_at DESC
LIMIT 20;
```

### 4) Security event spikes in last 1h

```sql
SELECT action, COUNT(*) AS cnt
FROM audit_event
WHERE category = 'security'
  AND occurred_at >= NOW() - INTERVAL '1 hour'
GROUP BY action
ORDER BY cnt DESC;
```

## Alert Rules

Use these baseline thresholds:

1. `audit_write_failed_total_24h > 0` for 15m
2. `audit_export_failed_ratio_24h > 0.2` for 30m
3. `audit_archive_daily` latest run `status != success`
4. Any `security` action count is > 3x 7-day hourly baseline

## Recovery Playbook

### Audit write failures

1. Check DB connectivity and schema (`audit_event` table exists).
2. Validate application logs around failed write stack traces.
3. If migration drift exists, run DB migration and re-verify.
4. Re-run affected business operation if idempotent.

### Export task failures

1. Check export task `error_message`.
2. Verify `uploads/audit-exports/` is writable.
3. Confirm disk space is sufficient.
4. Ask operator to trigger a new export task after fix.

### Archive task failures

1. Check `task_executions` for `audit_archive_daily`.
2. Verify `uploads/audit-archive/` is writable.
3. Validate DB delete permissions on `audit_event`.
4. Re-run task manually from task admin page.

## Data Retention

Current policy:

- online retention: 90 days (`audit_event`)
- archived file retention: 1 year (`uploads/audit-archive/` managed operationally)

## Verification Checklist

Before closing incident:

1. New audit events can be written successfully.
2. Export task can complete to `done`.
3. `audit_archive_daily` can run successfully once.
4. Alert signal returns to baseline range.
