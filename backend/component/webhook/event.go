package webhook

import (
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

type Event struct {
	Actor   *store.UserMessage
	Type    storepb.Activity_Type
	Comment string
	// nullable
	Issue   *Issue
	Project *Project
	Rollout *Rollout

	IssueUpdate         *EventIssueUpdate
	IssueApprovalCreate *EventIssueApprovalCreate
	IssueRolloutReady   *EventIssueRolloutReady
	StageStatusUpdate   *EventStageStatusUpdate
	TaskRunStatusUpdate *EventTaskRunStatusUpdate
}

func NewIssue(i *store.IssueMessage) *Issue {
	if i == nil {
		return nil
	}
	return &Issue{
		UID:          i.UID,
		Status:       i.Status.String(),
		Type:         i.Type.String(),
		Title:        i.Title,
		Description:  i.Description,
		CreatorEmail: i.CreatorEmail,
		Approval:     i.Payload.GetApproval(),
	}
}

func NewProject(p *store.ProjectMessage) *Project {
	return &Project{
		ResourceID: p.ResourceID,
		Title:      p.Title,
	}
}

func NewRollout(r *store.PlanMessage) *Rollout {
	return &Rollout{
		UID: int(r.UID),
	}
}

type Issue struct {
	UID          int
	Status       string
	Type         string
	Title        string
	Description  string
	CreatorEmail string
	Approval     *storepb.IssuePayloadApproval
}

type Project struct {
	ResourceID string
	Title      string
}

type Rollout struct {
	UID int
}

type EventIssueUpdate struct {
	Path string
}

type EventIssueApprovalCreate struct {
	Role string
}

type EventIssueRolloutReady struct {
	RolloutPolicy *storepb.RolloutPolicy
	StageName     string
}

type EventStageStatusUpdate struct {
	StageTitle string
	StageID    string
}

type EventTaskRunStatusUpdate struct {
	Title         string
	Status        string
	Detail        string
	SkippedReason string
}
