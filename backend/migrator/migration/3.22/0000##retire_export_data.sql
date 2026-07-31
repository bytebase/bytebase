-- Retire the export-data workflow: delete legacy DATABASE_EXPORT issues,
-- their plans/tasks/runs, and drop the export_archive table.

-- Export plans: referenced by a DATABASE_EXPORT issue, or containing an
-- exportDataConfig spec (covers API-created plans that never got an issue).
CREATE TEMPORARY TABLE _export_plan ON COMMIT DROP AS
SELECT project, plan_id AS id FROM issue
WHERE type = 'DATABASE_EXPORT' AND plan_id IS NOT NULL
UNION
SELECT project, id FROM plan
WHERE jsonb_typeof(config->'specs') = 'array'
    AND EXISTS (
        SELECT 1 FROM jsonb_array_elements(config->'specs') AS spec
        WHERE spec ? 'exportDataConfig'
    );

DELETE FROM task_run_log l
USING task_run r, task t
WHERE l.project = r.project AND l.task_run_id = r.id
    AND r.project = t.project AND r.task_id = t.id
    AND (t.type = 'DATABASE_EXPORT' OR (t.project, t.plan_id) IN (SELECT project, id FROM _export_plan));

DELETE FROM task_run r
USING task t
WHERE r.project = t.project AND r.task_id = t.id
    AND (t.type = 'DATABASE_EXPORT' OR (t.project, t.plan_id) IN (SELECT project, id FROM _export_plan));

DELETE FROM task t
WHERE t.type = 'DATABASE_EXPORT' OR (t.project, t.plan_id) IN (SELECT project, id FROM _export_plan);

DELETE FROM plan_check_run c
USING _export_plan p
WHERE c.project = p.project AND c.plan_id = p.id;

DELETE FROM issue_comment ic
USING issue i
WHERE ic.project = i.project AND ic.issue_id = i.id
    AND (i.type = 'DATABASE_EXPORT' OR (i.project, i.plan_id) IN (SELECT project, id FROM _export_plan));

DELETE FROM issue i
WHERE i.type = 'DATABASE_EXPORT' OR (i.project, i.plan_id) IN (SELECT project, id FROM _export_plan);

DELETE FROM plan_webhook_delivery d
USING _export_plan p
WHERE d.project = p.project AND d.plan_id = p.id;

DELETE FROM plan pl
USING _export_plan p
WHERE pl.project = p.project AND pl.id = p.id;

DROP TABLE export_archive;
