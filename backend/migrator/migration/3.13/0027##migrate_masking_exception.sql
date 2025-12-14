UPDATE policy
SET
    type = 'MASKING_EXEMPTION',
    payload = (
        SELECT jsonb_build_object(
            'exemptions',
            COALESCE(
                jsonb_agg(
                    jsonb_build_object(
                        'members', jsonb_build_array(elem->>'member'),
                        'condition', elem->'condition'
                    )
                ),
                '[]'::jsonb
            )
        )
        FROM jsonb_array_elements(payload -> 'maskingExceptions') AS elem
    )
WHERE type = 'MASKING_EXCEPTION';
