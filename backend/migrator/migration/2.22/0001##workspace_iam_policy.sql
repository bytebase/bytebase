INSERT INTO policy
(
	creator_id,
	updater_id,
	type,
	payload,
	resource_type,
	resource_id,
	inherit_from_parent
)
SELECT
	1,
	1,
	'bb.policy.iam',
	jsonb_build_object('bindings', jsonb_agg(t2.binding) || '{"role": "roles/workspaceMember", "members": ["allUsers"]}'::jsonb),
	'WORKSPACE',
	0,
	FALSE
FROM
(
	SELECT
		'roles/'||role AS role,
		array_agg('users/' || principal_id) AS members
	FROM member
	WHERE role != 'workspaceMember'
	GROUP BY role
) t1
LEFT JOIN LATERAL (
	SELECT
		jsonb_build_object(
			'role', t1.role,
			'members', t1.members
		) AS binding
) t2 ON TRUE;
