-- MCP capability moved from WORKSPACE_PROFILE.mcpCapability to its own setting
-- in 3.22. Preserve the legacy value when present and give every other
-- workspace the explicit backward-compatible READ_WRITE default.
--
-- This migration is the only legacy translation. Runtime reads only the MCP
-- row and treats a missing row as invalid metadata.
-- Keep the workspace set stable until the transaction validates the invariant.
LOCK TABLE workspace IN SHARE MODE;

INSERT INTO setting (workspace, name, value)
SELECT
    w.resource_id,
    'MCP',
    jsonb_build_object(
        'capability',
        CASE
            WHEN p.workspace IS NULL THEN to_jsonb('READ_WRITE'::text)
            WHEN jsonb_typeof(p.value) <> 'object' THEN 'null'::jsonb
            ELSE COALESCE(
                p.value->'mcpCapability',
                p.value->'mcp_capability',
                to_jsonb('READ_WRITE'::text)
            )
        END
    )
FROM workspace AS w
LEFT JOIN setting AS p
    ON p.workspace = w.resource_id
    AND p.name = 'WORKSPACE_PROFILE'
ON CONFLICT (workspace, name) DO NOTHING;

-- Existing MCP rows may predate the mandatory capability. Backfill those rows
-- too instead of teaching runtime code to synthesize a value.
UPDATE setting
SET value = value || jsonb_build_object('capability', 'READ_WRITE')
WHERE name = 'MCP'
    AND jsonb_typeof(value) = 'object'
    AND NOT value ? 'capability';

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM workspace AS w
        LEFT JOIN setting AS s
            ON s.workspace = w.resource_id
            AND s.name = 'MCP'
        WHERE s.workspace IS NULL
    ) THEN
        RAISE EXCEPTION 'MCP setting backfill left a workspace without an MCP setting';
    END IF;
END
$$;
