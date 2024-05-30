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
	Issue   *Issue
	Project *Project

	IssueUpdate         *EventIssueUpdate
	IssueApprovalCreate *EventIssueApprovalCreate
	IssueRolloutReady   *EventIssueRolloutReady
	StageStatusUpdate   *EventStageStatusUpdate
	TaskRunStatusUpdate *EventTaskRunStatusUpdate
}

func NewIssue(i *store.IssueMessage) *Issue {
	return &Issue{
		UID:         i.UID,
		Status:      i.Status.String(),
		Type:        i.Type.String(),
		Title:       i.Title,
		Description: i.Description,
		Creator:     i.Creator,
		Approval:    i.Payload.GetApproval(),
	}
}

func NewProject(p *store.ProjectMessage) *Project {
	return &Project{
		UID:        p.UID,
		ResourceID: p.ResourceID,
		Title:      p.Title,
	}
}

type Issue struct {
	UID         int
	Status      string
	Type        string
	Title       string
	Description string
	Creator     *store.UserMessage
	Approval    *storepb.IssuePayloadApproval
}

type Project struct {
	UID        int
	ResourceID string
	Title      string
}

type EventIssueUpdate struct {
	Path string
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
