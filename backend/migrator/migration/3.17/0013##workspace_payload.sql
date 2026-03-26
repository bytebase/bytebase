-- Move workspace.name and branding_logo into a JSONB payload column.
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

-- Migrate branding_logo from WORKSPACE_PROFILE setting into workspace.payload.logo.
-- The setting value is protojson (camelCase), so the source key is "brandingLogo".
-- The destination key is "logo" (new WorkspacePayload proto field).
UPDATE workspace w
SET payload = jsonb_set(w.payload, '{logo}', to_jsonb(s.value->>'brandingLogo'))
FROM setting s
WHERE s.workspace = w.resource_id
  AND s.name = 'WORKSPACE_PROFILE'
  AND s.value->>'brandingLogo' IS NOT NULL
  AND s.value->>'brandingLogo' != ''
  AND (w.payload->>'logo' IS NULL OR w.payload->>'logo' = '');
