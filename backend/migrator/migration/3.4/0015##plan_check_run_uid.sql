UPDATE plan_check_run
SET config = (config || jsonb_build_object(
		'instanceId',
		(select resource_id from instance where instance.id = (config->'instanceUid')::integer)
	)) - 'instanceUid'
WHERE config ? 'instanceUid';