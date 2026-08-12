-- Saved queries are private to their creator until the access-model redesign
-- ships per-object grants; the design migrates every visibility owner-private
-- with no mapping, so the flag is removed rather than carried. The tables move
-- to the renamed model in the same step.
ALTER TABLE worksheet DROP COLUMN visibility;

ALTER TABLE worksheet RENAME TO saved_query;
ALTER TABLE worksheet_organizer RENAME TO saved_query_organizer;
ALTER TABLE saved_query_organizer RENAME COLUMN worksheet TO saved_query;

ALTER INDEX IF EXISTS idx_worksheet_project RENAME TO idx_saved_query_project;
ALTER INDEX IF EXISTS idx_worksheet_creator_project RENAME TO idx_saved_query_creator_project;
ALTER INDEX IF EXISTS idx_worksheet_organizer_principal RENAME TO idx_saved_query_organizer_principal;
ALTER INDEX IF EXISTS idx_worksheet_organizer_payload RENAME TO idx_saved_query_organizer_payload;

-- The connected database moves from (instance, db_name) columns into the
-- payload as its canonical resource name (SavedQueryPayload.database), which
-- also encodes whether the instance is workspace- or project-scoped. Per the
-- design's carry rule, the reference is kept only when the database still
-- belongs to the saved query's own project; otherwise it is cleared.
UPDATE saved_query
SET payload = COALESCE(saved_query.payload, '{}'::jsonb) || jsonb_build_object(
    'database',
    CASE
        WHEN instance.project IS NOT NULL THEN 'projects/' || instance.project || '/instances/' || instance.resource_id || '/databases/' || db.name
        ELSE 'instances/' || instance.resource_id || '/databases/' || db.name
    END)
FROM db
    JOIN instance ON instance.resource_id = db.instance
WHERE saved_query.instance = db.instance
    AND saved_query.db_name = db.name
    AND db.project = saved_query.project
    -- Skip rows whose instance/database project scoping drifted apart; a
    -- name built from incoherent scopes would never validate again.
    AND (instance.project IS NULL OR instance.project = db.project);

ALTER TABLE saved_query DROP COLUMN instance, DROP COLUMN db_name;
