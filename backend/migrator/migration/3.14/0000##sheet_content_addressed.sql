-- Add new column with sha256 reference to task_run
ALTER TABLE task_run ADD COLUMN sheet_sha256 bytea REFERENCES sheet_blob(sha256);

-- Backfill task_run.sheet_sha256 from sheet table
UPDATE task_run tr
SET sheet_sha256 = s.sha256
FROM sheet s
WHERE tr.sheet_id = s.id;

-- Backfill task.payload JSONB (Task proto)
-- Converts sheetId (int) to sheetSha256 (hex string)
UPDATE task t
SET payload = jsonb_set(
    t.payload - 'sheetId',
    '{sheetSha256}',
    to_jsonb(encode(s.sha256, 'hex'))
)
FROM sheet s
WHERE (t.payload->>'sheetId')::int = s.id
AND t.payload ? 'sheetId';

-- Backfill plan_check_run.config JSONB (PlanCheckRunConfig proto)
-- Converts sheetUid (int) to sheetSha256 (hex string)
UPDATE plan_check_run pcr
SET config = jsonb_set(
    config - 'sheetUid',
    '{sheetSha256}',
    to_jsonb(encode(s.sha256, 'hex'))
)
FROM sheet s
WHERE (pcr.config->>'sheetUid')::int = s.id
AND pcr.config ? 'sheetUid';

-- Backfill changelog.payload JSONB (ChangelogPayload proto)
-- Converts sheet resource name to sheetSha256 field
UPDATE changelog c
SET payload = jsonb_set(
    c.payload - 'sheet',
    '{sheetSha256}',
    to_jsonb(encode(s.sha256, 'hex'))
)
FROM sheet s
WHERE s.id = (regexp_match(c.payload->>'sheet', '/sheets/(\d+)$'))[1]::int
AND c.payload ? 'sheet';

-- Backfill issue_comment.payload JSONB (IssueCommentPayload.PlanSpecUpdate proto)
-- Converts fromSheet and toSheet resource names to fromSheetSha256 and toSheetSha256 fields
UPDATE issue_comment ic
SET payload = jsonb_set(
    ic.payload #- '{planSpecUpdate,fromSheet}',
    '{planSpecUpdate,fromSheetSha256}',
    to_jsonb(encode(s1.sha256, 'hex'))
)
FROM sheet s1
WHERE s1.id = (regexp_match(ic.payload #>> '{planSpecUpdate,fromSheet}', '/sheets/(\d+)$'))[1]::int
AND ic.payload #> '{planSpecUpdate,fromSheet}' IS NOT NULL;

UPDATE issue_comment ic
SET payload = jsonb_set(
    ic.payload #- '{planSpecUpdate,toSheet}',
    '{planSpecUpdate,toSheetSha256}',
    to_jsonb(encode(s2.sha256, 'hex'))
)
FROM sheet s2
WHERE s2.id = (regexp_match(ic.payload #>> '{planSpecUpdate,toSheet}', '/sheets/(\d+)$'))[1]::int
AND ic.payload #> '{planSpecUpdate,toSheet}' IS NOT NULL;

-- Backfill plan.config JSONB (PlanConfig proto)
-- Converts sheet resource names to sheetSha256 fields in specs array
UPDATE plan p
SET config = (
    SELECT jsonb_set(
        p.config,
        '{specs}',
        jsonb_agg(
            CASE
                WHEN spec #> '{changeDatabaseConfig,sheet}' IS NOT NULL THEN
                    jsonb_set(
                        spec #- '{changeDatabaseConfig,sheet}',
                        '{changeDatabaseConfig,sheetSha256}',
                        to_jsonb(encode(s.sha256, 'hex'))
                    )
                WHEN spec #> '{exportDataConfig,sheet}' IS NOT NULL THEN
                    jsonb_set(
                        spec #- '{exportDataConfig,sheet}',
                        '{exportDataConfig,sheetSha256}',
                        to_jsonb(encode(s.sha256, 'hex'))
                    )
                ELSE spec
            END
        )
    )
    FROM jsonb_array_elements(p.config->'specs') AS spec
    LEFT JOIN sheet s ON s.id = COALESCE(
        (regexp_match(spec #>> '{changeDatabaseConfig,sheet}', '/sheets/(\d+)$'))[1]::int,
        (regexp_match(spec #>> '{exportDataConfig,sheet}', '/sheets/(\d+)$'))[1]::int
    )
)
WHERE config->'specs' IS NOT NULL;

-- Backfill revision.payload JSONB (RevisionPayload proto)
-- Converts sheet resource name to sheetSha256 field
UPDATE revision r
SET payload = jsonb_set(
    r.payload - 'sheet',
    '{sheetSha256}',
    to_jsonb(encode(s.sha256, 'hex'))
)
FROM sheet s
WHERE s.id = (regexp_match(r.payload->>'sheet', '/sheets/(\d+)$'))[1]::int
AND r.payload ? 'sheet';

-- Backfill release.payload JSONB (ReleasePayload proto)
-- Converts sheet resource names to sheetSha256 fields in files array
UPDATE release r
SET payload = (
    SELECT jsonb_set(
        r.payload,
        '{files}',
        jsonb_agg(
            jsonb_set(
                file - 'sheet',
                '{sheetSha256}',
                to_jsonb(encode(s.sha256, 'hex'))
            )
        )
    )
    FROM jsonb_array_elements(r.payload->'files') AS file
    LEFT JOIN sheet s ON s.id = (regexp_match(file->>'sheet', '/sheets/(\d+)$'))[1]::int
)
WHERE payload->'files' IS NOT NULL;

-- Drop old column
ALTER TABLE task_run DROP COLUMN sheet_id;

-- Note: sheet table is kept for now and will be dropped in a future migration
-- after confirming all references have been successfully migrated
