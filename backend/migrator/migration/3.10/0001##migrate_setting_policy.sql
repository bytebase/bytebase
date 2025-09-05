-- Migrate setting and policy: consolidate SQL_RESULT_SIZE_LIMIT setting and EXPORT_DATA policy into QUERY_DATA policy

-- Update existing QUERY_DATA policy or insert new one with migrated values
WITH setting_data AS (
    SELECT 
        CASE 
            WHEN s.value IS NOT NULL AND s.value != '' THEN 
                CAST(s.value AS JSONB)
            ELSE 
                '{}'::JSONB
        END as sql_restriction
    FROM setting s 
    WHERE s.name = 'SQL_RESULT_SIZE_LIMIT'
    UNION ALL
    SELECT '{}'::JSONB WHERE NOT EXISTS (
        SELECT 1 FROM setting WHERE name = 'SQL_RESULT_SIZE_LIMIT'
    )
    LIMIT 1
),
export_policy_data AS (
    SELECT 
        CASE 
            WHEN p.payload IS NOT NULL THEN p.payload
            ELSE '{}'::JSONB
        END as export_payload
    FROM policy p
    WHERE p.resource_type = 'WORKSPACE' 
      AND p.type = 'EXPORT_DATA'
    UNION ALL
    SELECT '{}'::JSONB WHERE NOT EXISTS (
        SELECT 1 FROM policy 
        WHERE resource_type = 'WORKSPACE' AND type = 'EXPORT_DATA'
    )
    LIMIT 1
),
combined_data AS (
    SELECT 
        jsonb_build_object(
            'disableExport', COALESCE((export_policy_data.export_payload->>'disable')::boolean, false),
            'maximumResultSize', COALESCE((setting_data.sql_restriction->>'maximumResultSize')::bigint, 104857600),
            'maximumResultRows', COALESCE((setting_data.sql_restriction->>'maximumResultRows')::integer, -1)
        ) as new_payload
    FROM setting_data, export_policy_data
)
INSERT INTO policy (resource_type, resource, type, payload, enforce, inherit_from_parent)
SELECT 
    'WORKSPACE',
    '',
    'QUERY_DATA',
    combined_data.new_payload,
    true,
    true
FROM combined_data
ON CONFLICT (resource_type, resource, type) 
DO UPDATE SET 
    payload = (
        -- Preserve existing timeout and other fields while updating the new ones
        policy.payload || 
        jsonb_build_object(
            'disableExport', COALESCE((SELECT export_policy_data.export_payload->>'disable' FROM export_policy_data)::boolean, false),
            'maximumResultSize', COALESCE((SELECT setting_data.sql_restriction->>'maximumResultSize' FROM setting_data)::bigint, 104857600),
            'maximumResultRows', COALESCE((SELECT setting_data.sql_restriction->>'maximumResultRows' FROM setting_data)::integer, -1)
        )
    ),
    updated_at = now();

-- Remove the SQL_RESULT_SIZE_LIMIT setting
DELETE FROM setting 
WHERE name = 'SQL_RESULT_SIZE_LIMIT';

-- Remove the EXPORT_DATA policy
DELETE FROM policy 
WHERE resource_type = 'WORKSPACE' 
  AND type = 'EXPORT_DATA';