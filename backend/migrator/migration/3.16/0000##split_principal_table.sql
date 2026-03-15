-- Split principal table into three tables: principal (END_USER only), service_account, workload_identity.

-- Step 0: Fix legacy emails, create tables, migrate data, clean up principal.
-- All data migration is guarded by checking principal.type column existence,
-- which is dropped at the end — so re-runs skip all data operations.
DO $$
DECLARE
    rec RECORD;
    new_email TEXT;
    base_local TEXT;
BEGIN
    -- Only run data migration if principal still has a type column (i.e., not yet split).
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'principal' AND column_name = 'type'
    ) THEN
        RETURN;
    END IF;

    -- Fix legacy service account emails that don't match the required format.
    -- Valid formats: {name}@service.bytebase.com  OR  {name}@{project}.service.bytebase.com
    -- This MUST run BEFORE dropping FK constraints so that ON UPDATE CASCADE propagates email
    -- changes to all creator/deleter columns automatically.
    FOR rec IN
        SELECT id, email, project
        FROM principal
        WHERE type = 'SERVICE_ACCOUNT'
          AND email NOT LIKE '%@service.bytebase.com'
          AND email NOT LIKE '%@%.service.bytebase.com'
    LOOP
        base_local := split_part(rec.email, '@', 1);
        IF rec.project IS NOT NULL THEN
            new_email := base_local || '@' || rec.project || '.service.bytebase.com';
        ELSE
            new_email := base_local || '@service.bytebase.com';
        END IF;
        IF EXISTS (SELECT 1 FROM principal WHERE email = new_email AND id != rec.id) THEN
            IF rec.project IS NOT NULL THEN
                new_email := base_local || '-' || rec.id || '@' || rec.project || '.service.bytebase.com';
            ELSE
                new_email := base_local || '-' || rec.id || '@service.bytebase.com';
            END IF;
        END IF;
        UPDATE principal SET email = new_email WHERE id = rec.id;
        UPDATE policy
        SET payload = replace(payload::text, 'serviceAccounts/' || rec.email, 'serviceAccounts/' || new_email)::jsonb
        WHERE payload::text LIKE '%serviceAccounts/' || rec.email || '%';
        UPDATE user_group
        SET payload = replace(payload::text, 'serviceAccounts/' || rec.email, 'serviceAccounts/' || new_email)::jsonb
        WHERE payload::text LIKE '%serviceAccounts/' || rec.email || '%';
    END LOOP;

    -- Migrate data into new tables.
    INSERT INTO service_account (deleted, created_at, name, email, service_key_hash, project)
    SELECT deleted, created_at, name, email, password_hash, project
    FROM principal WHERE type = 'SERVICE_ACCOUNT'
    ON CONFLICT (email) DO NOTHING;

    INSERT INTO workload_identity (deleted, created_at, name, email, project, config)
    SELECT deleted, created_at, name, email, project,
           COALESCE(profile->'workloadIdentityConfig', '{}')
    FROM principal WHERE type = 'WORKLOAD_IDENTITY'
    ON CONFLICT (email) DO NOTHING;

    -- Clean up principal table.
    DELETE FROM principal WHERE type != 'END_USER';
END $$;

-- Create tables (idempotent).
CREATE TABLE IF NOT EXISTS service_account (
    deleted boolean NOT NULL DEFAULT FALSE,
    created_at timestamptz NOT NULL DEFAULT now(),
    name text NOT NULL,
    email text NOT NULL PRIMARY KEY,
    service_key_hash text NOT NULL,
    project text REFERENCES project(resource_id)
);

CREATE INDEX IF NOT EXISTS idx_service_account_project ON service_account(project) WHERE project IS NOT NULL;

CREATE TABLE IF NOT EXISTS workload_identity (
    deleted boolean NOT NULL DEFAULT FALSE,
    created_at timestamptz NOT NULL DEFAULT now(),
    name text NOT NULL,
    email text NOT NULL PRIMARY KEY,
    project text REFERENCES project(resource_id),
    -- Stored as WorkloadIdentityConfig (proto/store/store/user.proto)
    config jsonb NOT NULL DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_workload_identity_project ON workload_identity(project) WHERE project IS NOT NULL;

-- Drop FK constraints on creator/deleter columns that can reference SA/WI emails.
-- Keep FKs on oauth2_authorization_code, oauth2_refresh_token, web_refresh_token (END_USER only).
ALTER TABLE plan DROP CONSTRAINT IF EXISTS plan_creator_fkey;
ALTER TABLE task_run DROP CONSTRAINT IF EXISTS task_run_creator_fkey;
ALTER TABLE issue DROP CONSTRAINT IF EXISTS issue_creator_fkey;
ALTER TABLE issue_comment DROP CONSTRAINT IF EXISTS issue_comment_creator_fkey;
ALTER TABLE query_history DROP CONSTRAINT IF EXISTS query_history_creator_fkey;
ALTER TABLE worksheet DROP CONSTRAINT IF EXISTS worksheet_creator_fkey;
ALTER TABLE worksheet_organizer DROP CONSTRAINT IF EXISTS worksheet_organizer_principal_fkey;
ALTER TABLE revision DROP CONSTRAINT IF EXISTS revision_deleter_fkey;
ALTER TABLE release DROP CONSTRAINT IF EXISTS release_creator_fkey;
ALTER TABLE access_grant DROP CONSTRAINT IF EXISTS access_grant_creator_fkey;

-- Add missing index for plan.creator (queried in plan filter).
CREATE INDEX IF NOT EXISTS idx_plan_creator ON plan(creator);

-- Drop columns from principal (idempotent).
ALTER TABLE principal DROP CONSTRAINT IF EXISTS principal_project_type_check;
ALTER TABLE principal DROP CONSTRAINT IF EXISTS principal_type_check;
ALTER TABLE principal DROP COLUMN IF EXISTS type;
ALTER TABLE principal DROP COLUMN IF EXISTS project;
DROP INDEX IF EXISTS idx_principal_project;
