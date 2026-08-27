-- MCP capability moved from WORKSPACE_PROFILE.mcpCapability to its own setting
-- in 3.22. Preserve the legacy value when present and give every other
-- workspace the explicit backward-compatible READ_WRITE default.
--
-- Keep the legacy key for older replicas during a rolling upgrade. New code
-- reads only the MCP row.
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

-- Older builds could create an MCP row by updating only a sibling field. Make
-- those rows explicit too, so runtime never has to distinguish a missing key
-- from a value it cannot interpret.
UPDATE setting
SET value = value || jsonb_build_object('capability', 'READ_WRITE')
WHERE name = 'MCP'
    AND jsonb_typeof(value) = 'object'
    AND NOT value ? 'capability';
