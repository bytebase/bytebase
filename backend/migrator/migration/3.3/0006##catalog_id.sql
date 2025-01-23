UPDATE db_schema
SET config = replace(config::TEXT, 'semanticTypeId', 'semanticType')::JSONB;
UPDATE db_schema
SET config = replace(config::TEXT, 'classificationId', 'classification')::JSONB;