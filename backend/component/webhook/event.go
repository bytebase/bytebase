package webhook

import (
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

type Event struct {
	Project *Project
	Type    storepb.Activity_Type

	// Focused event types (only one is set)
	IssueCreated      *EventIssueCreated
	ApprovalRequested *EventIssueApprovalRequested
	SentBack          *EventIssueSentBack
	RolloutFailed     *EventRolloutFailed
	RolloutCompleted  *EventRolloutCompleted
}

func NewIssue(i *store.IssueMessage) *Issue {
	if i == nil {
		return nil
	}
	return &Issue{
		ID:           i.ResourceID,
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
		ID:    r.ResourceID,
		Title: r.Name,
	}
}

type Issue struct {
	ID           string
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
	ID    string
	Title string
}

type EventIssueCreated struct {
	Creator *User
	Issue   *Issue
}

type EventIssueApprovalRequested struct {
	Creator   *User
	Issue     *Issue
	Approvers []User
}

type EventIssueSentBack struct {
	Approver *User
	Creator  *User
	Issue    *Issue
	Reason   string
}

type EventRolloutFailed struct {
	Rollout *Rollout
}

type EventRolloutCompleted struct {
	Rollout *Rollout
}

type User struct {
	Name  string
	Email string
}
