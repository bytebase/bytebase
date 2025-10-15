UPDATE setting
SET value = (
    SELECT jsonb_build_object(
        'rules',
        COALESCE(
            (
                SELECT jsonb_agg(
                    jsonb_set(
                        rule::jsonb,
                        '{template}',
                        (rule->'template')::jsonb - 'creatorId'
                    )
                )
                FROM jsonb_array_elements(value::jsonb->'rules') AS rule
            ),
            '[]'::jsonb
        )
    )
)
WHERE name = 'bb.workspace.approval';