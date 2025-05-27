package base

type EventType string

const (
	EventTypeIssueCreate         = "bb.webhook.event.issue.create"
	EventTypeIssueUpdate         = "bb.webhook.event.issue.update"
	EventTypeIssueStatusUpdate   = "bb.webhook.event.issue.status.update"
	EventTypeIssueCommentCreate  = "bb.webhook.event.issue.comment.create"
	EventTypeIssueApprovalCreate = "bb.webhook.event.issue.approval.create"
	EventTypeIssueApprovalPass   = "bb.webhook.event.issue.approval.pass"
	EventTypeIssueRolloutReady   = "bb.webhook.event.issue.rollout.ready"

	EventTypeStageStatusUpdate   = "bb.webhook.event.stage.status.update"
	EventTypeTaskRunStatusUpdate = "bb.webhook.event.taskRun.status.update"
)
