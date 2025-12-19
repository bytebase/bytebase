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
    payload - 'sheetId',
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
-- Updates sheet resource name from projects/{project}/sheets/{id} to projects/{project}/sheets/{sha256}
UPDATE changelog c
SET payload = jsonb_set(
    payload,
    '{sheet}',
    to_jsonb(regexp_replace(payload->>'sheet', '/sheets/\d+$', '/sheets/' || encode(s.sha256, 'hex')))
)
FROM sheet s
WHERE s.id = (regexp_match(payload->>'sheet', '/sheets/(\d+)$'))[1]::int
AND payload ? 'sheet';

-- Backfill issue_comment.payload JSONB (IssueCommentPayload.PlanSpecUpdate proto)
-- Updates fromSheet and toSheet resource names
UPDATE issue_comment ic
SET payload = CASE
    WHEN payload #> '{planSpecUpdate,fromSheet}' IS NOT NULL THEN
        jsonb_set(
            payload,
            '{planSpecUpdate,fromSheet}',
            to_jsonb(regexp_replace(payload #>> '{planSpecUpdate,fromSheet}', '/sheets/\d+$', '/sheets/' || encode(s1.sha256, 'hex')))
        )
    ELSE payload
END
FROM sheet s1
WHERE s1.id = (regexp_match(payload #>> '{planSpecUpdate,fromSheet}', '/sheets/(\d+)$'))[1]::int
AND payload #> '{planSpecUpdate,fromSheet}' IS NOT NULL;

UPDATE issue_comment ic
SET payload = CASE
    WHEN payload #> '{planSpecUpdate,toSheet}' IS NOT NULL THEN
        jsonb_set(
            payload,
            '{planSpecUpdate,toSheet}',
            to_jsonb(regexp_replace(payload #>> '{planSpecUpdate,toSheet}', '/sheets/\d+$', '/sheets/' || encode(s2.sha256, 'hex')))
        )
    ELSE payload
END
FROM sheet s2
WHERE s2.id = (regexp_match(payload #>> '{planSpecUpdate,toSheet}', '/sheets/(\d+)$'))[1]::int
AND payload #> '{planSpecUpdate,toSheet}' IS NOT NULL;

-- Backfill plan.config JSONB (PlanConfig proto)
-- Updates sheet resource names in specs array for ChangeDatabaseConfig and ExportDataConfig
UPDATE plan p
SET config = (
    SELECT jsonb_set(
        p.config,
        '{specs}',
        jsonb_agg(
            CASE
                WHEN spec #> '{changeDatabaseConfig,sheet}' IS NOT NULL THEN
                    jsonb_set(
                        spec,
                        '{changeDatabaseConfig,sheet}',
                        to_jsonb(regexp_replace(spec #>> '{changeDatabaseConfig,sheet}', '/sheets/\d+$', '/sheets/' || encode(s.sha256, 'hex')))
                    )
                WHEN spec #> '{exportDataConfig,sheet}' IS NOT NULL THEN
                    jsonb_set(
                        spec,
                        '{exportDataConfig,sheet}',
                        to_jsonb(regexp_replace(spec #>> '{exportDataConfig,sheet}', '/sheets/\d+$', '/sheets/' || encode(s.sha256, 'hex')))
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
-- Updates sheet resource name
UPDATE revision r
SET payload = jsonb_set(
    payload,
    '{sheet}',
    to_jsonb(regexp_replace(payload->>'sheet', '/sheets/\d+$', '/sheets/' || encode(s.sha256, 'hex')))
)
FROM sheet s
WHERE s.id = (regexp_match(payload->>'sheet', '/sheets/(\d+)$'))[1]::int
AND payload ? 'sheet';

-- Backfill release.payload JSONB (ReleasePayload proto)
-- Updates sheet resource names in files array
UPDATE release r
SET payload = (
    SELECT jsonb_set(
        r.payload,
        '{files}',
        jsonb_agg(
            jsonb_set(
                file,
                '{sheet}',
                to_jsonb(regexp_replace(file->>'sheet', '/sheets/\d+$', '/sheets/' || encode(s.sha256, 'hex')))
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
