-- Move data_classification_config_id from column to setting JSONB field

-- Step 1: Migrate data_classification_config_id to setting field
-- Only update rows where data_classification_config_id is not empty
UPDATE project
SET setting = jsonb_set(
    setting,
    '{dataClassificationConfigId}',
    to_jsonb(data_classification_config_id)
)
WHERE data_classification_config_id != '';

-- Step 2: Drop the data_classification_config_id column
ALTER TABLE project DROP COLUMN data_classification_config_id;
