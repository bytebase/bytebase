package webhook

import (
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

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

type Event struct {
	Actor   *store.UserMessage
	Type    EventType
	Comment string
	Issue   struct {
		UID         int
		Title       string
		Status      string
		Type        string
		Description string
		Creator     *store.UserMessage
		Approval    *storepb.IssuePayloadApproval
	}
	Project struct {
		UID        int
		ResourceID string
		Title      string
	}

	IssueStatusUpdate struct {
		Status string
	}
	IssueUpdate *struct {
		Field string
	}
	IssueApprovalCreate *EventIssueApprovalCreate
	IssueRolloutReady   *EventIssueRolloutReady
	StageStatusUpdate   *EventStageStatusUpdate
	TaskRunStatusUpdate *EventTaskRunStatusUpdate
}

type EventIssueApprovalCreate struct {
	ApprovalStep *storepb.ApprovalStep
}

type EventIssueRolloutReady struct {
	RolloutPolicy *storepb.RolloutPolicy
	StageName     string
}

type EventStageStatusUpdate struct {
	StageTitle string
	StageUID   int
}

type EventTaskRunStatusUpdate struct {
	Title         string
	Status        string
	Detail        string
	SkippedReason string
}
