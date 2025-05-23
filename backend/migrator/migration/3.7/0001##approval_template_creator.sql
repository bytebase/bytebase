UPDATE setting 
SET value = (
    SELECT jsonb_build_object(
        'rules', 
        jsonb_agg(
            jsonb_set(
                rule::jsonb,
                '{template}',
                (rule->'template')::jsonb - 'creatorId'
            )
        )
    )
    FROM jsonb_array_elements(value::jsonb->'rules') AS rule
)
WHERE name = 'bb.workspace.approval';