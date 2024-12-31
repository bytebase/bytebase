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
        'title', algorithm_record->>'title',
        'description', algorithm_record->>'description',
        'algorithm', algorithm_record                                            
    ) AS transformed_algorithm
    FROM algorithms_expanded
),
combined_data AS (
    SELECT
        jsonb_agg(ta.transformed_algorithm) || (SELECT value::jsonb->'types' FROM setting WHERE name = 'bb.workspace.semantic-types') AS types_array
    FROM transformed_algorithms ta
)
UPDATE setting
SET value = jsonb_build_object('types', types_array)
FROM combined_data
WHERE name = 'bb.workspace.semantic-types';