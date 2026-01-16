-- Merge DATA_SOURCE_QUERY policies into QUERY_DATA policies
-- This migration consolidates the two separate policy types into a single unified QUERY_DATA policy

-- Step 1: Merge all DATA_SOURCE_QUERY policies into QUERY_DATA
-- For resources with existing QUERY_DATA: merge fields into existing policy
-- For resources without QUERY_DATA: create new QUERY_DATA policy with DATA_SOURCE_QUERY fields
INSERT INTO policy (resource_type, resource, type, payload, inherit_from_parent, enforce, updated_at)
SELECT
    p.resource_type,
    p.resource,
    'QUERY_DATA',
    jsonb_build_object(
        'adminDataSourceRestriction',
        COALESCE(p.payload->'adminDataSourceRestriction', '"RESTRICTION_UNSPECIFIED."'::jsonb),
        'disallowDdl',
        COALESCE(p.payload->'disallowDdl', 'false'::jsonb),
        'disallowDml',
        COALESCE(p.payload->'disallowDml', 'false'::jsonb)
    ),
    p.inherit_from_parent,
    p.enforce,
    NOW()
FROM policy p
WHERE p.type = 'DATA_SOURCE_QUERY'
ON CONFLICT (resource_type, resource, type)
DO UPDATE SET
    payload = policy.payload ||
        jsonb_build_object(
            'adminDataSourceRestriction',
            COALESCE(EXCLUDED.payload->'adminDataSourceRestriction', '"RESTRICTION_UNSPECIFIED."'::jsonb),
            'disallowDdl',
            COALESCE(EXCLUDED.payload->'disallowDdl', 'false'::jsonb),
            'disallowDml',
            COALESCE(EXCLUDED.payload->'disallowDml', 'false'::jsonb)
        ),
    updated_at = NOW();

-- Step 2: Delete all DATA_SOURCE_QUERY policies (now merged into QUERY_DATA)
DELETE FROM policy
WHERE type = 'DATA_SOURCE_QUERY';
