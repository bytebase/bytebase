UPDATE setting
SET value = jsonb_build_object(
    'maximumResultSize', coalesce(value::jsonb->>'limit', '104857600'),
    'maximumResultRows', '-1'
    )
WHERE name = 'SQL_RESULT_SIZE_LIMIT';