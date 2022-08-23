UPDATE
	activity
SET
	container_id = task.pipeline_id
FROM
	task
WHERE
	activity.type LIKE 'bb.pipeline.task.%'
	AND task.id = (activity.payload->>'taskId')::int;