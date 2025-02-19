UPDATE policy
SET payload =
	jsonb_set(payload, '{roles}', COALESCE(payload->'projectRoles', '[]'::jsonb) || COALESCE(payload->'workspaceRoles', '[]'::jsonb)) - 'projectRoles' - 'workspaceRoles'
WHERE type = 'bb.policy.rollout' AND (payload ? 'projectRoles' OR payload ? 'workspaceRoles');