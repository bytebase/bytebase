-- Before 3.24, Oracle connections ignored the TLS section: they used TLS only
-- when SSL=TRUE was in the extra connection parameters. The driver now applies
-- the section, so turn TLS mode off on every Oracle data source: they keep
-- connecting as they did, and the instance form shows TLS as disabled. A CA
-- kept here would be trusted once TLS is turned on while the form showed
-- System trust, so the CA and client certificate go too, as when TLS is turned
-- off in the form. Deleting useSsl is protojson's form of false.
UPDATE instance
SET metadata = jsonb_set(
    metadata,
    '{dataSources}',
    (
        SELECT jsonb_agg(
            ds - ARRAY[
                'useSsl',
                'obfuscatedSslCa',
                'obfuscatedSslCert',
                'obfuscatedSslKey',
                'obfuscatedSslCaPath',
                'obfuscatedSslCertPath',
                'obfuscatedSslKeyPath'
            ]
            ORDER BY ord
        )
        FROM jsonb_array_elements(metadata->'dataSources') WITH ORDINALITY AS t(ds, ord)
    )
)
WHERE metadata->>'engine' = 'ORACLE'
  AND jsonb_typeof(metadata->'dataSources') = 'array'
  AND metadata->'dataSources' <> '[]';
