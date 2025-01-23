DELETE FROM issue_comment;

INSERT INTO issue_comment (
    row_status,
    creator_id,
    created_ts,
    updater_id,
    updated_ts,
    issue_id,
    payload
) SELECT
    a.row_status,
    a.creator_id,
    a.created_ts,
    a.updater_id,
    a.updated_ts,
    ai.issue_id,
    p.p
FROM activity AS a
LEFT JOIN LATERAL (
	SELECT
		issue.id AS issue_id,
		project.resource_id AS project_rid
    FROM issue
    LEFT JOIN project ON project.id = issue.project_id
    WHERE issue.id = (SELECT CASE
        WHEN a.type LIKE 'bb.pipeline%' THEN (
            SELECT issue.id FROM issue WHERE issue.pipeline_id = a.container_id
        )
        WHEN a.type LIKE 'bb.issue%' THEN a.container_id
    END)
) AS ai ON TRUE
LEFT JOIN LATERAL (
	SELECT CASE a.type
		WHEN 'bb.issue.comment.create' THEN
			jsonb_build_object('comment', a.comment) ||
			CASE WHEN a.payload ? 'approvalEvent' THEN jsonb_build_object('approval', jsonb_build_object('status', a.payload#>>'{approvalEvent,status}'))
				ELSE '{}'
			END
		WHEN 'bb.issue.field.update' THEN
			jsonb_build_object('issueUpdate', CASE a.payload->>'fieldId'
				WHEN '1' THEN jsonb_build_object('fromTitle', COALESCE(a.payload->>'oldValue',''), 'toTitle', COALESCE(a.payload->>'newValue',''))
				WHEN '3' THEN jsonb_build_object('fromAssignee', COALESCE((SELECT 'users/'||principal.email FROM principal WHERE principal.id=CAST(a.payload->>'oldValue' AS INTEGER)),''), 'toAssignee', COALESCE((SELECT 'users/'||principal.email FROM principal WHERE principal.id=CAST(a.payload->>'newValue' AS INTEGER)), ''))
				WHEN '4' THEN jsonb_build_object('fromDescription', COALESCE(a.payload->>'oldValue', ''), 'toDescription', COALESCE(a.payload->>'newValue',''))
			END)
		WHEN 'bb.issue.status.update' THEN
			jsonb_build_object('issueUpdate',jsonb_build_object('fromStatus', a.payload->>'oldStatus', 'toStatus', a.payload->>'newStatus'))
		WHEN 'bb.pipeline.stage.status.update' THEN
			jsonb_build_object('stageEnd',jsonb_build_object('stage', (SELECT 'projects/'||ai.project_rid||'/rollouts/'||stage.pipeline_id||'/stages/'||stage.id FROM stage WHERE (stage.id::text)=a.payload->>'stageId')))
		WHEN 'bb.pipeline.taskrun.status.update' THEN
			jsonb_build_object('taskUpdate',jsonb_build_object('toStatus', a.payload->>'newStatus', 'tasks', (SELECT jsonb_build_array('projects/'||ai.project_rid||'/rollouts/'||task.pipeline_id||'/stages/'||task.stage_id||'/tasks/'||task.id) FROM task WHERE task.id::text=a.payload->>'taskId')))
		WHEN 'bb.pipeline.task.statement.update' THEN
			jsonb_build_object('taskUpdate',jsonb_build_object('fromSheet','projects/'||ai.project_rid||'/sheets/'||(a.payload->>'oldSheetId'),'toSheet', 'projects/'||ai.project_rid||'/sheets/'||(a.payload->>'newSheetId'), 'tasks', (SELECT jsonb_build_array('projects/'||ai.project_rid||'/rollouts/'||task.pipeline_id||'/stages/'||task.stage_id||'/tasks/'||task.id) FROM task WHERE task.id::text=a.payload->>'taskId')))
		WHEN 'bb.pipeline.task.general.earliest-allowed-time.update' THEN
			jsonb_build_object('taskUpdate',jsonb_build_object('fromEarliestAllowedTime',to_char(to_timestamp(COALESCE(CAST(payload->>'oldEarliestAllowedTs' AS INTEGER), 0)) AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),'toEarliestAllowedTime',to_char(to_timestamp(COALESCE(CAST(payload->>'newEarliestAllowedTs' AS INTEGER), 0)) AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'),'tasks', (SELECT jsonb_build_array('projects/'||ai.project_rid||'/rollouts/'||task.pipeline_id||'/stages/'||task.stage_id||'/tasks/'||task.id) FROM task WHERE task.id::text=a.payload->>'taskId')))
		WHEN 'bb.pipeline.task.prior-backup' THEN
			jsonb_build_object('taskPriorBackup',jsonb_build_object('tables',a.payload->'schemaMetadata','task', (SELECT 'projects/'||ai.project_rid||'/rollouts/'||task.pipeline_id||'/stages/'||task.stage_id||'/tasks/'||task.id FROM task WHERE task.id::text=a.payload->>'taskId')))			
	END AS p
) AS p ON TRUE
WHERE ai.issue_id IS NOT NULL
AND p.p IS NOT NULL
ORDER BY a.id;
