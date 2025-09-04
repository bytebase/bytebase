-- Migrate instance metadata for Spanner and BigQuery instances with PASSWORD authentication
-- Change authentication type from PASSWORD to GOOGLE_CLOUD_SQL_IAM
-- Move obfuscatedPassword to gcpCredential.obfuscatedContent

UPDATE instance
SET metadata = jsonb_set(
    metadata,
    '{dataSources}',
    (
        SELECT jsonb_agg(
            CASE 
                WHEN (ds->>'authenticationType' = 'PASSWORD'
                      AND ds ? 'obfuscatedPassword')
                THEN 
                    -- Remove obfuscatedPassword field
                    (ds - 'obfuscatedPassword') ||
                    -- Set authenticationType to GOOGLE_CLOUD_SQL_IAM
                    jsonb_build_object('authenticationType', 'GOOGLE_CLOUD_SQL_IAM') ||
                    -- Add gcpCredential with obfuscatedContent
                    jsonb_build_object('gcpCredential', jsonb_build_object('obfuscatedContent', ds->>'obfuscatedPassword'))
                ELSE ds
            END
        )
        FROM jsonb_array_elements(metadata->'dataSources') AS ds
    )
)
WHERE metadata->>'engine' IN ('SPANNER', 'BIGQUERY')
  AND metadata ? 'dataSources'
  AND EXISTS (
      SELECT 1
      FROM jsonb_array_elements(metadata->'dataSources') AS ds
      WHERE ds->>'authenticationType' = 'PASSWORD'
        AND ds ? 'obfuscatedPassword'
  );