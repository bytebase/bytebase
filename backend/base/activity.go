package base

// ActivityType is the type for an activity.
type ActivityType string

const (
	// Notifications via webhooks.
	// ActivityNotifyIssueApproved is the type for notifying the creator when the issue approval passes.
	// Will not be stored. Only used for notification.
	ActivityNotifyIssueApproved ActivityType = "bb.notify.issue.approved"
	// ActivityPipelineRollout is the type for notifying releasers to rollout.
	// Will not be stored. Only used for notification.
	ActivityNotifyPipelineRollout ActivityType = "bb.notify.pipeline.rollout"

	// Issue related.

	// ActivityIssueCreate is the type for creating issues.
	// Used for notification only.
	ActivityIssueCreate ActivityType = "bb.issue.create"
	// ActivityIssueCommentCreate is the type for creating issue comments.
	// Used for notification only.
	ActivityIssueCommentCreate ActivityType = "bb.issue.comment.create"
	// ActivityIssueFieldUpdate is the type for updating issue fields.
	// Used for notification only.
	ActivityIssueFieldUpdate ActivityType = "bb.issue.field.update"
	// ActivityIssueStatusUpdate is the type for updating issue status.
	// Used for notification only.
	ActivityIssueStatusUpdate ActivityType = "bb.issue.status.update"
	// ActivityIssueApprovalNotify is the type for notifying issue approval.
	// Used for notification only.
	ActivityIssueApprovalNotify ActivityType = "bb.issue.approval.notify"
	// ActivityPipelineStageStatusUpdate is the type for stage begins or ends.
	// Used for notification only.
	ActivityPipelineStageStatusUpdate ActivityType = "bb.pipeline.stage.status.update"
	// ActivityPipelineTaskStatusUpdate is the type for updating pipeline task status.
	// Deprecated: use `ActivityPipelineTaskRunStatusUpdate` instead.
	ActivityPipelineTaskStatusUpdate ActivityType = "bb.pipeline.task.status.update"
	// ActivityPipelineTaskRunStatusUpdate is the type for updating pipeline task run status.
	// Used for notification only.
	ActivityPipelineTaskRunStatusUpdate ActivityType = "bb.pipeline.taskrun.status.update"
	// ActivityPipelineTaskStatementUpdate is the type for updating pipeline task SQL statement.
	// Deprecated.
	ActivityPipelineTaskStatementUpdate ActivityType = "bb.pipeline.task.statement.update"
	// ActivityPipelineTaskEarliestAllowedTimeUpdate is the type for updating pipeline task the earliest allowed time.
	// Deprecated.
	ActivityPipelineTaskEarliestAllowedTimeUpdate ActivityType = "bb.pipeline.task.general.earliest-allowed-time.update"
	// ActivityPipelineTaskStatementUpdate is the type for updating pipeline task SQL statement.
	// Deprecated.
	ActivityPipelineTaskPriorBackup ActivityType = "bb.pipeline.task.prior-backup"

	// Member related.

	// ActivityMemberCreate is the type for creating members.
	ActivityMemberCreate ActivityType = "bb.member.create"
	// ActivityMemberRoleUpdate is the type for updating member roles.
	ActivityMemberRoleUpdate ActivityType = "bb.member.role.update"
	// ActivityMemberActivate is the type for activating members.
	ActivityMemberActivate ActivityType = "bb.member.activate"
	// ActivityMemberDeactivate is the type for deactivating members.
	ActivityMemberDeactivate ActivityType = "bb.member.deactivate"

	// Project related.

	// ActivityProjectRepositoryPush is the type for pushing repositories.
	ActivityProjectRepositoryPush ActivityType = "bb.project.repository.push"
	// ActivityProjectDatabaseTransfer is the type for transferring databases.
	ActivityProjectDatabaseTransfer ActivityType = "bb.project.database.transfer"
	// ActivityProjectMemberCreate is the type for creating project members.
	ActivityProjectMemberCreate ActivityType = "bb.project.member.create"
	// ActivityProjectMemberDelete is the type for deleting project members.
	ActivityProjectMemberDelete ActivityType = "bb.project.member.delete"

	// SQL Editor related.

	// ActivitySQLQuery is the type for executing query.
	ActivitySQLQuery ActivityType = "bb.sql.query"
	// ActivitySQLExport is the type for exporting SQL.
	ActivitySQLExport ActivityType = "bb.sql.export"
)
