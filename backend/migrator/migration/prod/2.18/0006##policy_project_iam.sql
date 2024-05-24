INSERT INTO policy (
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
	'bb.policy.project-iam',
	jsonb_build_object('bindings', jsonb_agg(t2.binding)),
	'PROJECT',
	t2.project_id,
	FALSE
FROM 
(
	SELECT
		project_id,
		'roles/'||role AS role,
		condition,
		array_agg('users/'||principal_id) AS members
	FROM project_member
	GROUP BY project_id, role, condition
	ORDER BY project_id
) t1
LEFT JOIN LATERAL (
	SELECT
		t1.project_id,
		jsonb_build_object(
			'role', t1.role,
			'members', t1.members,
			'condition', t1.condition
		) AS binding
		GROUP BY t1.project_id
) t2 ON TRUE
GROUP BY t2.project_id
ORDER BY t2.project_id;
