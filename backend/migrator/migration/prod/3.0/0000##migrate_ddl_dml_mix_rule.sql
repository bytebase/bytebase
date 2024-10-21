WITH t1 AS (
	SELECT review_config.id, jsonb_strip_nulls(jsonb_build_object('type', e.type, 'level', e.level, 'engine', e.engine, 'payload', e.payload, 'comment', e.comment)) AS rule FROM review_config
	LEFT JOIN LATERAL (
		SELECT x.type, x.level, x.engine, x.payload, x.comment
		FROM jsonb_to_recordset(review_config.payload->'sqlReviewRules') AS x(type TEXT, level TEXT, engine TEXT, payload TEXT, comment TEXT)
		WHERE type != 'statement.disallow-mix-ddl-dml'
		UNION ALL 
		SELECT 'statement.disallow-mix-in-ddl', x.level, x.engine, x.payload, x.comment
		FROM jsonb_to_recordset(review_config.payload->'sqlReviewRules') AS x(type TEXT, level TEXT, engine TEXT, payload TEXT, comment TEXT)
		WHERE type = 'statement.disallow-mix-ddl-dml'
		UNION ALL 
		SELECT 'statement.disallow-mix-in-dml', x.level, x.engine, x.payload, x.comment
		FROM jsonb_to_recordset(review_config.payload->'sqlReviewRules') AS x(type TEXT, level TEXT, engine TEXT, payload TEXT, comment TEXT)
		WHERE type = 'statement.disallow-mix-ddl-dml'
	) AS e ON TRUE
), t2 AS (
	SELECT t1.id, jsonb_build_object('sqlReviewRules', jsonb_agg(t1.rule)) AS payload
	FROM t1
	GROUP BY t1.id
)
UPDATE review_config
SET payload = t2.payload
FROM t2
WHERE review_config.id = t2.id;
