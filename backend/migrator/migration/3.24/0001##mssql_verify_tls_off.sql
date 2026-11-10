-- Before 3.24, SQL Server connections never verified the server certificate,
-- even with verifyTlsCertificate on. The driver now verifies, so turn it off
-- where TLS is on: those data sources keep connecting as they did, and the
-- instance form shows that the certificate is not verified. Deleting the key
-- is protojson's form of false.
UPDATE instance
SET metadata = jsonb_set(
    metadata,
    '{dataSources}',
    (
        SELECT jsonb_agg(
            CASE
                WHEN ds @> '{"useSsl": true, "verifyTlsCertificate": true}'
                THEN ds - 'verifyTlsCertificate'
                ELSE ds
            END
            ORDER BY ord
        )
        FROM jsonb_array_elements(metadata->'dataSources') WITH ORDINALITY AS t(ds, ord)
    )
)
WHERE metadata->>'engine' = 'MSSQL'
  AND metadata->'dataSources' @> '[{"useSsl": true, "verifyTlsCertificate": true}]';
