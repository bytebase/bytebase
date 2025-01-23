UPDATE deployment_config
SET config = (
SELECT jsonb_build_object('schedule', jsonb_build_object('deployments', JSONB_AGG(v_new ORDER BY o1))) FROM (
		SELECT v, jsonb_set(v, '{spec,selector,matchExpressions}', JSONB_AGG(e ORDER BY o2)) -'name' || jsonb_build_object('title', v->>'name') || jsonb_build_object('id', gen_random_uuid()) AS v_new, o1
		FROM (
			SELECT
			v,
			o1,
			e || jsonb_build_object('operator', CASE e->>'operator'
			WHEN 'In' THEN 'IN'
			WHEN 'Exists' THEN 'EXISTS'
			WHEN 'Not_In' THEN 'NOT_IN'
			END
			) AS e,
			o2
			FROM (
				SELECT v, o1, v#>'{spec,selector,matchExpressions}' AS es FROM (
					SELECT v, o1
					FROM jsonb_array_elements(deployment_config.config->'deployments') WITH ORDINALITY AS dso (v, o1)
				) tt1
			) tt2, LATERAL jsonb_array_elements(es) WITH ORDINALITY AS eso(e, o2)
		) tt3
		GROUP BY v, o1
	) tt4
) WHERE NOT (config ? 'schedule')