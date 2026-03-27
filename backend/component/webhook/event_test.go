package webhook

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func TestNewIssue(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		a := require.New(t)
		a.Nil(NewIssue(nil))
	})

	t.Run("basic fields", func(t *testing.T) {
		a := require.New(t)
		msg := &store.IssueMessage{
			UID:          42,
			Title:        "Add index to users table",
			Description:  "We need a B-tree index",
			Status:       storepb.Issue_OPEN,
			Type:         storepb.Issue_DATABASE_CHANGE,
			CreatorEmail: "alice@example.com",
			Payload:      &storepb.Issue{},
		}
		got := NewIssue(msg)
		a.Equal(int64(42), got.UID)
		a.Equal("Add index to users table", got.Title)
		a.Equal("We need a B-tree index", got.Description)
		a.Equal("OPEN", got.Status)
		a.Equal("DATABASE_CHANGE", got.Type)
		a.Equal("alice@example.com", got.CreatorEmail)
	})

	t.Run("nil payload", func(t *testing.T) {
		a := require.New(t)
		msg := &store.IssueMessage{
			UID:   1,
			Title: "test",
		}
		got := NewIssue(msg)
		a.NotNil(got)
		a.Nil(got.Approval)
	})
}

func TestNewProject(t *testing.T) {
	a := require.New(t)
	msg := &store.ProjectMessage{
		ResourceID: "my-project",
		Workspace:  "ws-001",
		Title:      "My Project",
	}
	got := NewProject(msg)
	a.Equal("my-project", got.ResourceID)
	a.Equal("ws-001", got.Workspace)
	a.Equal("My Project", got.Title)
}

func TestNewRollout(t *testing.T) {
	a := require.New(t)
	msg := &store.PlanMessage{
		UID:  99,
		Name: "Deploy v2.0",
	}
	got := NewRollout(msg)
	a.Equal(99, got.UID)
	a.Equal("Deploy v2.0", got.Title)
}

func TestEventIssueApproved_Structure(t *testing.T) {
	a := require.New(t)

	event := &EventIssueApproved{
		Approver: &User{Name: "Bob", Email: "bob@example.com"},
		Creator:  &User{Name: "Alice", Email: "alice@example.com"},
		Issue: &Issue{
			UID:          101,
			Title:        "Grant read access",
			Description:  "Need read access to prod DB",
			CreatorEmail: "alice@example.com",
		},
	}

	a.Equal("Bob", event.Approver.Name)
	a.Equal("bob@example.com", event.Approver.Email)
	a.Equal("Alice", event.Creator.Name)
	a.Equal("alice@example.com", event.Creator.Email)
	a.Equal(int64(101), event.Issue.UID)
	a.Equal("Grant read access", event.Issue.Title)
}

func TestEvent_IssueApproved_TypeField(t *testing.T) {
	a := require.New(t)

	event := &Event{
		Type: storepb.Activity_ISSUE_APPROVED,
		IssueApproved: &EventIssueApproved{
			Approver: &User{Name: "Bob", Email: "bob@example.com"},
			Creator:  &User{Name: "Alice", Email: "alice@example.com"},
			Issue:    &Issue{UID: 1, Title: "test"},
		},
	}

	a.Equal(storepb.Activity_ISSUE_APPROVED, event.Type)
	a.NotNil(event.IssueApproved)
	a.Nil(event.IssueCreated)
	a.Nil(event.ApprovalRequested)
	a.Nil(event.SentBack)
	a.Nil(event.RolloutFailed)
	a.Nil(event.RolloutCompleted)
}

func TestEvent_OnlyOneEventTypeSet(t *testing.T) {
	tests := []struct {
		name  string
		event Event
	}{
		{
			name: "ISSUE_CREATED",
			event: Event{
				Type:         storepb.Activity_ISSUE_CREATED,
				IssueCreated: &EventIssueCreated{},
			},
		},
		{
			name: "ISSUE_APPROVAL_REQUESTED",
			event: Event{
				Type:              storepb.Activity_ISSUE_APPROVAL_REQUESTED,
				ApprovalRequested: &EventIssueApprovalRequested{},
			},
		},
		{
			name: "ISSUE_APPROVED",
			event: Event{
				Type:          storepb.Activity_ISSUE_APPROVED,
				IssueApproved: &EventIssueApproved{},
			},
		},
		{
			name: "ISSUE_SENT_BACK",
			event: Event{
				Type:     storepb.Activity_ISSUE_SENT_BACK,
				SentBack: &EventIssueSentBack{},
			},
		},
		{
			name: "PIPELINE_FAILED",
			event: Event{
				Type:          storepb.Activity_PIPELINE_FAILED,
				RolloutFailed: &EventRolloutFailed{},
			},
		},
		{
			name: "PIPELINE_COMPLETED",
			event: Event{
				Type:             storepb.Activity_PIPELINE_COMPLETED,
				RolloutCompleted: &EventRolloutCompleted{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := require.New(t)
			// Verify only the expected event type field is set
			setCount := 0
			if tt.event.IssueCreated != nil {
				setCount++
			}
			if tt.event.ApprovalRequested != nil {
				setCount++
			}
			if tt.event.IssueApproved != nil {
				setCount++
			}
			if tt.event.SentBack != nil {
				setCount++
			}
			if tt.event.RolloutFailed != nil {
				setCount++
			}
			if tt.event.RolloutCompleted != nil {
				setCount++
			}
			a.Equal(1, setCount, "exactly one event type should be set for %s", tt.name)
		})
	}
}
