-- Keep only the first READ_ONLY data source in instance metadata.
-- Legacy rows may contain multiple READ_ONLY data sources, which makes
-- automatic data source resolution ambiguous.

UPDATE instance
SET metadata = jsonb_set(
    metadata,
    '{dataSources}',
    (
        SELECT jsonb_agg(filtered.ds ORDER BY filtered.ord)
        FROM (
            SELECT ds, ord,
                   sum(CASE WHEN ds->>'type' = 'READ_ONLY' THEN 1 ELSE 0 END)
                       OVER (ORDER BY ord ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW) AS read_only_rank
            FROM jsonb_array_elements(metadata->'dataSources') WITH ORDINALITY AS elem(ds, ord)
        ) AS filtered
        WHERE filtered.ds->>'type' <> 'READ_ONLY' OR filtered.read_only_rank = 1
    )
)
WHERE jsonb_typeof(metadata->'dataSources') = 'array'
  AND (
      SELECT count(*)
      FROM jsonb_array_elements(metadata->'dataSources') AS ds
      WHERE ds->>'type' = 'READ_ONLY'
  ) > 1;
