-- Migration: Convert integer IDs stored in JSONB columns to string resource_ids,
-- rename JSONB keys to match updated proto field names,
-- add resource_id to export_archive/audit_log/query_history, and add created_at to task.

-----------------------
-- 1a. Add resource_id column to export_archive
-----------------------
ALTER TABLE export_archive ADD COLUMN resource_id text NOT NULL DEFAULT gen_random_uuid()::text;
CREATE UNIQUE INDEX uk_export_archive_resource_id ON export_archive(resource_id);

-----------------------
-- 1b. Add created_at column to task, backfill preserving id order
-----------------------
ALTER TABLE task ADD COLUMN created_at timestamptz NOT NULL DEFAULT now();
-- Backfill: assign synthetic timestamps that preserve the serial id ordering.
-- Using epoch + id seconds ensures existing rows maintain their relative order.
UPDATE task SET created_at = '2020-01-01 00:00:00+00'::timestamptz + (id * interval '1 second');

-----------------------
-- 1c. Add resource_id column to audit_log
-----------------------
ALTER TABLE audit_log ADD COLUMN resource_id text NOT NULL DEFAULT gen_random_uuid()::text;
CREATE UNIQUE INDEX idx_audit_log_unique_resource_id ON audit_log(resource_id);

-----------------------
-- 1d. Add resource_id column to query_history
-----------------------
ALTER TABLE query_history ADD COLUMN resource_id text NOT NULL DEFAULT gen_random_uuid()::text;
CREATE UNIQUE INDEX idx_query_history_unique_resource_id ON query_history(resource_id);

-----------------------
-- 2. access_grant.payload: convert issueId from integer to issue.resource_id string
-----------------------
UPDATE access_grant ag
SET payload = jsonb_set(
    ag.payload,
    '{issueId}',
    to_jsonb(i.resource_id)
)
FROM issue i
WHERE (ag.payload->>'issueId') IS NOT NULL
  AND (ag.payload->>'issueId') ~ '^\d+$'
  AND i.id = (ag.payload->>'issueId')::bigint;

-----------------------
-- 3. task_run.result: rename exportArchiveUid to exportArchiveId,
--    convert integer to export_archive.resource_id string
-----------------------
UPDATE task_run tr
SET result = (tr.result - 'exportArchiveUid') || jsonb_build_object('exportArchiveId', ea.resource_id)
FROM export_archive ea
WHERE (tr.result->>'exportArchiveUid') IS NOT NULL
  AND (tr.result->>'exportArchiveUid') ~ '^\d+$'
  AND ea.id = (tr.result->>'exportArchiveUid')::int;

-----------------------
-- 4. changelog.payload: replace integer IDs in taskRun resource name
--    Format: projects/{project}/plans/{planIntID}/rollout/stages/{stage}/tasks/{taskIntID}/taskRuns/{taskRunIntID}
--    Replace planIntID, taskIntID, taskRunIntID with their resource_ids.
-----------------------
UPDATE changelog cl
SET payload = jsonb_set(
    cl.payload,
    '{taskRun}',
    to_jsonb(
        regexp_replace(
            regexp_replace(
                regexp_replace(
                    cl.payload->>'taskRun',
                    '/plans/' || p.id::text || '/rollout/',
                    '/plans/' || p.resource_id || '/rollout/'
                ),
                '/tasks/' || t.id::text || '/taskRuns/',
                '/tasks/' || t.resource_id || '/taskRuns/'
            ),
            '/taskRuns/' || tr.id::text || '$',
            '/taskRuns/' || tr.resource_id
        )
    )
)
FROM task_run tr
JOIN task t ON t.resource_id = tr.task_id
JOIN plan p ON p.resource_id = t.plan_id
WHERE (cl.payload->>'taskRun') IS NOT NULL
  AND (cl.payload->>'taskRun') ~ '/plans/\d+/rollout/'
  AND tr.id = ((regexp_match(cl.payload->>'taskRun', '/taskRuns/(\d+)$'))[1])::int
  AND t.id  = ((regexp_match(cl.payload->>'taskRun', '/tasks/(\d+)/taskRuns/'))[1])::int
  AND p.id  = ((regexp_match(cl.payload->>'taskRun', '/plans/(\d+)/rollout/'))[1])::int;

-----------------------
-- 5. revision.payload: replace integer IDs in taskRun resource name (same format as changelog)
-----------------------
UPDATE revision rv
SET payload = jsonb_set(
    rv.payload,
    '{taskRun}',
    to_jsonb(
        regexp_replace(
            regexp_replace(
                regexp_replace(
                    rv.payload->>'taskRun',
                    '/plans/' || p.id::text || '/rollout/',
                    '/plans/' || p.resource_id || '/rollout/'
                ),
                '/tasks/' || t.id::text || '/taskRuns/',
                '/tasks/' || t.resource_id || '/taskRuns/'
            ),
            '/taskRuns/' || tr.id::text || '$',
            '/taskRuns/' || tr.resource_id
        )
    )
)
FROM task_run tr
JOIN task t ON t.resource_id = tr.task_id
JOIN plan p ON p.resource_id = t.plan_id
WHERE (rv.payload->>'taskRun') IS NOT NULL
  AND (rv.payload->>'taskRun') ~ '/plans/\d+/rollout/'
  AND tr.id = ((regexp_match(rv.payload->>'taskRun', '/taskRuns/(\d+)$'))[1])::int
  AND t.id  = ((regexp_match(rv.payload->>'taskRun', '/tasks/(\d+)/taskRuns/'))[1])::int
  AND p.id  = ((regexp_match(rv.payload->>'taskRun', '/plans/(\d+)/rollout/'))[1])::int;

-----------------------
-- 6. issue_comment.payload: replace integer plan ID in planSpecUpdate.spec resource name
--    Format: projects/{project}/plans/{planIntID}/specs/{spec}
--    Replace planIntID with plan.resource_id.
--    The spec ID itself is a hash, not an integer, so it stays as-is.
-----------------------
UPDATE issue_comment ic
SET payload = jsonb_set(
    ic.payload,
    '{planSpecUpdate,spec}',
    to_jsonb(
        regexp_replace(
            ic.payload->'planSpecUpdate'->>'spec',
            '/plans/' || p.id::text || '/specs/',
            '/plans/' || p.resource_id || '/specs/'
        )
    )
)
FROM plan p
WHERE (ic.payload->'planSpecUpdate'->>'spec') IS NOT NULL
  AND (ic.payload->'planSpecUpdate'->>'spec') ~ '/plans/\d+/specs/'
  AND p.id = ((regexp_match(ic.payload->'planSpecUpdate'->>'spec', '/plans/(\d+)/specs/'))[1])::bigint;
