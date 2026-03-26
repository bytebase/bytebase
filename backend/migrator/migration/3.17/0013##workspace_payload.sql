-- Move workspace.name into a JSONB payload column for extensibility.
-- The payload stores WorkspacePayload proto (title, branding_logo, etc.)

-- Add payload column if it doesn't exist.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'workspace' AND column_name = 'payload') THEN
        ALTER TABLE workspace ADD COLUMN payload jsonb NOT NULL DEFAULT '{}';
    END IF;
END $$;

-- Migrate name into payload.title if the name column still exists.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'workspace' AND column_name = 'name') THEN
        UPDATE workspace SET payload = jsonb_set(payload, '{title}', to_jsonb(name))
        WHERE payload->>'title' IS NULL OR payload->>'title' = '';
        ALTER TABLE workspace DROP COLUMN name;
    END IF;
END $$;
