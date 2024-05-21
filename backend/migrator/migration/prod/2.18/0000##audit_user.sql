UPDATE audit_log
SET payload = payload || jsonb_build_object('user',
	CONCAT('users/', COALESCE((
		SELECT id
		FROM principal
		WHERE principal.email = REPLACE((audit_log.payload->>'user')::text, 'users/', '')
	), 1)))
WHERE payload ? 'user';
