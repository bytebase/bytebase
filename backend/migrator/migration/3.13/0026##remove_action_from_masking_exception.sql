UPDATE policy
SET payload = (
    SELECT jsonb_build_object(
        'maskingExceptions',
        COALESCE(jsonb_agg(elem - 'action'), '[]'::jsonb)
    )
    FROM jsonb_array_elements(payload -> 'maskingExceptions') AS elem
)
WHERE type = 'MASKING_EXCEPTION';
