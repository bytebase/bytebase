-- Convert environment color from legacy #rrggbb strings to google.type.Color
-- JSON objects. Invalid string colors are removed so the setting remains
-- readable by protojson after the field type change.
UPDATE setting AS s
SET value = jsonb_set(s.value, '{environments}', converted.environments, false)
FROM (
    SELECT
        workspace,
        jsonb_agg(
            CASE
                WHEN jsonb_typeof(env->'color') = 'string'
                    AND env->>'color' ~ '^#[0-9A-Fa-f]{6}$'
                    THEN jsonb_set(
                        env,
                        '{color}',
                        jsonb_build_object(
                            'red', get_byte(decode(substr(env->>'color', 2, 2), 'hex'), 0)::float8 / 255,
                            'green', get_byte(decode(substr(env->>'color', 4, 2), 'hex'), 0)::float8 / 255,
                            'blue', get_byte(decode(substr(env->>'color', 6, 2), 'hex'), 0)::float8 / 255
                        ),
                        true
                    )
                WHEN jsonb_typeof(env->'color') = 'string'
                    THEN env - 'color'
                ELSE env
            END
            ORDER BY ord
        ) AS environments
    FROM setting
    CROSS JOIN LATERAL jsonb_array_elements(value->'environments') WITH ORDINALITY AS e(env, ord)
    WHERE name = 'ENVIRONMENT'
        AND jsonb_typeof(value->'environments') = 'array'
    GROUP BY workspace
) AS converted
WHERE s.name = 'ENVIRONMENT'
    AND s.workspace = converted.workspace;

-- Convert project issue label color from #rrggbb strings to google.type.Color
-- JSON objects. Invalid string colors are removed so project settings remain
-- readable by protojson after the field type change.
UPDATE project AS p
SET setting = jsonb_set(p.setting, '{issueLabels}', converted.issue_labels, false)
FROM (
    SELECT
        resource_id,
        jsonb_agg(
            CASE
                WHEN jsonb_typeof(label->'color') = 'string'
                    AND label->>'color' ~ '^#[0-9A-Fa-f]{6}$'
                    THEN jsonb_set(
                        label,
                        '{color}',
                        jsonb_build_object(
                            'red', get_byte(decode(substr(label->>'color', 2, 2), 'hex'), 0)::float8 / 255,
                            'green', get_byte(decode(substr(label->>'color', 4, 2), 'hex'), 0)::float8 / 255,
                            'blue', get_byte(decode(substr(label->>'color', 6, 2), 'hex'), 0)::float8 / 255
                        ),
                        true
                    )
                WHEN jsonb_typeof(label->'color') = 'string'
                    THEN label - 'color'
                ELSE label
            END
            ORDER BY ord
        ) AS issue_labels
    FROM project
    CROSS JOIN LATERAL jsonb_array_elements(setting->'issueLabels') WITH ORDINALITY AS e(label, ord)
    WHERE jsonb_typeof(setting->'issueLabels') = 'array'
    GROUP BY resource_id
) AS converted
WHERE p.resource_id = converted.resource_id;
