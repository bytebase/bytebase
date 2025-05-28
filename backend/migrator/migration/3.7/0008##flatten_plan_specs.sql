-- Flatten specs from steps array to top-level specs array in plan config
UPDATE plan
SET config = jsonb_set(
    config - 'steps',  -- Remove the steps key
    '{specs}',         -- Add specs at top level
    COALESCE(
        (
            SELECT jsonb_agg(spec)
            FROM jsonb_array_elements(config->'steps') AS step,
                 jsonb_array_elements(step->'specs') AS spec
        ),
        '[]'::jsonb    -- Default to empty array if no specs found
    )
)
WHERE config ? 'steps';  -- Only update rows that have steps