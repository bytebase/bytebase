INSERT INTO audit_log (
    created_ts,
    payload
)
SELECT
	activity.created_ts,
    jsonb_strip_nulls(jsonb_build_object(
        'parent', t.parent,
        'method', t.method,
        'resource', t.resource,
        'user', t.user,
        'severity', t.severity,
        'request', to_jsonb(t_req.req::TEXT),
        'response', to_jsonb(t_res.res::TEXT),
        'status', t.status
    ))
FROM activity
LEFT JOIN principal ON principal.id = activity.creator_id
LEFT JOIN instance ON instance.id::TEXT = activity.payload->>'instanceId'
LEFT JOIN LATERAL (
    SELECT
        activity.resource_container AS parent,
        CASE activity.type
            WHEN 'bb.sql.query' THEN '/bytebase.v1.SQLService/Query'
            WHEN 'bb.sql.export' THEN '/bytebase.v1.SQLService/Export'
        END AS method,
        'instances/'||instance.resource_id||COALESCE('/databases/'||(activity.payload->>'databaseName'), '') AS resource,
        'users/'||principal.email AS user,
        'INFO' AS severity,
        CASE 
            WHEN activity.payload->>'error' != '' THEN jsonb_build_object(
                'code', 13,
                'message', activity.payload->>'error'
            )
        END AS status
) AS t ON TRUE
LEFT JOIN LATERAL (
    SELECT
        jsonb_build_object(
            'name', t.resource,
            'statement', activity.payload->>'statement'
        ) AS req
) AS t_req ON TRUE
LEFT JOIN LATERAL (
    SELECT
        CASE activity.type
            WHEN 'bb.sql.query' THEN jsonb_build_object(
                'results', jsonb_build_array(
                    jsonb_strip_nulls(jsonb_build_object(
                        'statement', activity.payload->>'statement',
                        'latency', NULLIF(rtrim((CAST(activity.payload->>'durationNs' AS DECIMAL) / 1000000000)::text, '0.')||'s', 's')
                    ))
                )
            )
        END
        AS res
) AS t_res ON TRUE
WHERE activity.type = 'bb.sql.query' OR activity.type = 'bb.sql.export'