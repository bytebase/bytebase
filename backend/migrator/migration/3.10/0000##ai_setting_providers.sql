-- Migrate AI setting from single provider to multiple providers support
-- The old AISetting had: enabled, provider, endpoint, apiKey, model, version
-- The new AISetting has: providers[] with each provider having: type, endpoint, apiKey, model, version

-- This migration converts the old format to the new format
UPDATE setting
SET value = CASE
    WHEN value = '' OR value = '{}' THEN '{}'
    ELSE jsonb_build_object(
        'providers', 
        CASE
            WHEN (value::jsonb->>'enabled')::boolean = true THEN
                jsonb_build_array(
                    jsonb_build_object
(
                        'type', COALESCE
(value::jsonb->>'provider', 'TYPE_UNSPECIFIED'),
                        'endpoint', COALESCE
(value::jsonb->>'endpoint', ''),
                        'apiKey', COALESCE
(value::jsonb->>'apiKey', ''),
                        'model', COALESCE
(value::jsonb->>'model', ''),
                        'version', COALESCE
(value::jsonb->>'version', '')
                    )
                )
            ELSE '[]'::jsonb
END
    )::text
END
WHERE name = 'AI'
  AND value != ''; -- Only update if it has the old format
