WITH t AS (
    SELECT
        policy.id as policy_id,
        jsonb_build_object('maskingExceptions', array_agg(t.p)) as payload
    FROM policy
    LEFT JOIN LATERAL (
        SELECT
            e || jsonb_build_object('member', (
                CONCAT('users/', COALESCE((
                SELECT id
                FROM principal
                WHERE principal.email = e->>'member'
                ), 1))
            )) AS p
        FROM jsonb_array_elements(policy.payload->'maskingExceptions') e
    ) AS t ON TRUE
    WHERE policy.type = 'bb.policy.masking-exception'
    GROUP BY policy.id
)
UPDATE policy
SET payload = t.payload
FROM t
WHERE t.policy_id = policy.id;
