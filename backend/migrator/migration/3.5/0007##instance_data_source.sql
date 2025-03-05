UPDATE instance
SET metadata = metadata || jsonb_build_object(
	'dataSources', (
        SELECT coalesce(jsonb_agg(data_source.options), '[]'::jsonb)
        FROM data_source
        WHERE data_source.instance = instance.resource_id
    )
);