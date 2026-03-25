-- Change setting PK from (name) to (workspace, name) for multi-workspace support.
ALTER TABLE setting DROP CONSTRAINT IF EXISTS setting_pkey;
DO $$
BEGIN
    ALTER TABLE setting ADD PRIMARY KEY (workspace, name);
EXCEPTION WHEN duplicate_table THEN
    -- PK already exists.
END $$;
DROP INDEX IF EXISTS idx_setting_unique_workspace_name;
