UPDATE activity
SET payload = jsonb_build_object('issueCommentCreatePayload',
	CASE
	    WHEN payload ? 'externalApprovalEvent' THEN
			-- If the key 'externalApprovalEvent' exists updates its' value of key 'type' and 'action'.
			CASE
				WHEN payload #> '{externalApprovalEvent, action}' = '"APPROVE"' THEN
					jsonb_set(jsonb_set(payload, '{externalApprovalEvent, type}', '"TYPE_FEISHU"'), '{externalApprovalEvent, action}', '"ACTION_APPROVE"')
				ELSE
					jsonb_set(jsonb_set(payload, '{externalApprovalEvent, type}', '"TYPE_FEISHU"'), '{externalApprovalEvent, action}', '"ACTION_REJECT"')
			END
		WHEN payload ? 'taskRollbackBy' THEN
			-- Proto3 maps int64 to JSON string, so we need to convert them to string.
			jsonb_set(
				jsonb_set(
					jsonb_set(
						jsonb_set(
							payload, '{taskRollbackBy, issueId}', to_jsonb(payload #>> '{taskRollbackBy, issueId}')
						), '{taskRollbackBy, taskId}', to_jsonb(payload #>> '{taskRollbackBy, taskId}')
					), '{taskRollbackBy, rollbackByIssueId}', to_jsonb(payload #>> '{taskRollbackBy, rollbackByIssueId}')
				), '{taskRollbackBy, rollbackByTaskId}', to_jsonb(payload #>> '{taskRollbackBy, rollbackByTaskId}')
			)
		ELSE
			payload
	END
)
WHERE "type"='bb.issue.comment.create';