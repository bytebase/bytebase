INSERT INTO setting (creator_id, updater_id, name, value)
VALUES (1, 1, 'bb.workspace.semantic-types', '{"types": []}')
ON CONFLICT (name) DO NOTHING;

WITH setting_data AS (
    SELECT
        value::jsonb->'algorithms' AS algorithms_json
    FROM setting
    WHERE name='bb.workspace.masking-algorithm'
),
algorithms_expanded AS (
    SELECT jsonb_array_elements(algorithms_json) AS algorithm_record
    FROM setting_data
    WHERE algorithms_json IS NOT NULL AND jsonb_array_length(algorithms_json) > 0
),
transformed_algorithms AS (
    SELECT jsonb_build_object(
        'id', algorithm_record->>'id',
        'title', coalesce(algorithm_record->>'title', ''),
        'description', coalesce(algorithm_record->>'description', ''),
        'algorithm', algorithm_record
    ) AS transformed_algorithm
    FROM algorithms_expanded
),
combined_data AS (
    SELECT
        coalesce(jsonb_agg(ta.transformed_algorithm), '[]'::jsonb) || (SELECT coalesce(value::jsonb->'types', '[]'::jsonb) FROM setting WHERE name = 'bb.workspace.semantic-types') AS types_array
    FROM transformed_algorithms ta
)
UPDATE setting
SET value = jsonb_build_object('types', types_array)
FROM combined_data
WHERE name = 'bb.workspace.semantic-types';