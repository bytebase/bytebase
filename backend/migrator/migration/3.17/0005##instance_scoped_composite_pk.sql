-- Step 2e: Switch revision, sync_history, changelog to resource_id PKs.

-- Phase A: Drop FKs
ALTER TABLE changelog DROP CONSTRAINT IF EXISTS changelog_sync_history_id_fkey;

-- Phase B: revision — promote resource_id to PK
DROP INDEX IF EXISTS idx_revision_unique_resource_id;
DO $$ BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'revision_pkey' AND conrelid = 'revision'::regclass
    ) THEN
        ALTER TABLE revision DROP CONSTRAINT revision_pkey;
    END IF;
END $$;
ALTER TABLE revision ADD PRIMARY KEY (resource_id);

-- Phase C: sync_history — add resource_id, promote to PK
ALTER TABLE sync_history ADD COLUMN IF NOT EXISTS resource_id text;
UPDATE sync_history SET resource_id = gen_random_uuid()::text WHERE resource_id IS NULL;
ALTER TABLE sync_history ALTER COLUMN resource_id SET NOT NULL;
ALTER TABLE sync_history ALTER COLUMN resource_id SET DEFAULT gen_random_uuid()::text;
DO $$ BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'sync_history_pkey' AND conrelid = 'sync_history'::regclass
    ) THEN
        ALTER TABLE sync_history DROP CONSTRAINT sync_history_pkey;
    END IF;
END $$;
ALTER TABLE sync_history ADD PRIMARY KEY (resource_id);

-- Phase D: changelog — promote resource_id to PK
DROP INDEX IF EXISTS idx_changelog_unique_resource_id;
DO $$ BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'changelog_pkey' AND conrelid = 'changelog'::regclass
    ) THEN
        ALTER TABLE changelog DROP CONSTRAINT changelog_pkey;
    END IF;
END $$;
ALTER TABLE changelog ADD PRIMARY KEY (resource_id);

-- Phase E: changelog — add sync_history text column, backfill, drop old sync_history_id
ALTER TABLE changelog ADD COLUMN IF NOT EXISTS sync_history text;
UPDATE changelog SET sync_history = sync_history.resource_id
    FROM sync_history WHERE changelog.sync_history IS NULL AND changelog.sync_history_id = sync_history.id;
ALTER TABLE changelog DROP COLUMN IF EXISTS sync_history_id;

-- Phase F: Re-add FK from changelog.sync_history to sync_history.resource_id
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'changelog_sync_history_fkey' AND conrelid = 'changelog'::regclass) THEN
        ALTER TABLE changelog ADD CONSTRAINT changelog_sync_history_fkey
            FOREIGN KEY (sync_history) REFERENCES sync_history(resource_id);
    END IF;
END $$;
