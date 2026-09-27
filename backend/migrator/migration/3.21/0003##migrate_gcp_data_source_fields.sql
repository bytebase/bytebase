-- Migrate GCP data sources to the project_id/instance_id fields.
--
-- Spanner data sources used to store the instance path in host:
--   "host": "projects/<project>/instances/<instance>"
-- BigQuery data sources used to store the project ID in host:
--   "host": "<project>"
-- These now live in dedicated fields:
--   "projectId": "<project>", "instanceId": "<instance>"
-- and host/port are repurposed as an optional Google API endpoint override.

UPDATE instance
SET metadata = jsonb_set(
    metadata,
    '{dataSources}',
    (
        SELECT jsonb_agg(
            CASE
                WHEN ds->>'host' ~ '^projects/[^/]+/instances/[^/]+$'
                THEN (
                    ds || jsonb_build_object(
                        'projectId', split_part(ds->>'host', '/', 2),
                        'instanceId', split_part(ds->>'host', '/', 4)
                    )
                ) - 'host'
                ELSE ds
            END
            ORDER BY ord
        )
        FROM jsonb_array_elements(metadata->'dataSources') WITH ORDINALITY AS t(ds, ord)
    )
)
WHERE metadata->>'engine' = 'SPANNER'
  AND jsonb_typeof(metadata->'dataSources') = 'array'
  AND jsonb_array_length(metadata->'dataSources') > 0;

UPDATE instance
SET metadata = jsonb_set(
    metadata,
    '{dataSources}',
    (
        SELECT jsonb_agg(
            CASE
                WHEN ds ? 'host' AND NOT ds ? 'projectId'
                THEN (ds || jsonb_build_object('projectId', ds->>'host')) - 'host'
                ELSE ds
            END
            ORDER BY ord
        )
        FROM jsonb_array_elements(metadata->'dataSources') WITH ORDINALITY AS t(ds, ord)
    )
)
WHERE metadata->>'engine' = 'BIGQUERY'
  AND jsonb_typeof(metadata->'dataSources') = 'array'
  AND jsonb_array_length(metadata->'dataSources') > 0;
