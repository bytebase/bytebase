package webhook

import (
	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

type Event struct {
	Actor   *store.UserMessage
	Type    common.EventType
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
		ResourceID: p.ResourceID,
		Title:      p.Title,
	}
}

func NewRollout(r *store.PipelineMessage) *Rollout {
	return &Rollout{
		UID: r.ID,
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
	ApprovalStep *storepb.ApprovalStep
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
