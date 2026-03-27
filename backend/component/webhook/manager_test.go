package webhook

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/component/config"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/webhook"
)

// Tests for getWebhookContextFromEvent that don't require a database connection.
// The Manager is constructed with a nil store and a profile with ExternalURL set.
// Test cases where store calls (GetAccountByEmail, GetWorkspaceProfileSetting)
// are NOT reached.

func newTestManager() *Manager {
	return &Manager{
		store: nil,
		profile: &config.Profile{
			ExternalURL: "https://bb.example.com",
		},
	}
}

func TestGetWebhookContext_UnsupportedEventType(t *testing.T) {
	a := require.New(t)
	m := newTestManager()
	ctx := context.Background()

	e := &Event{
		Type:    storepb.Activity_TYPE_UNSPECIFIED,
		Project: &Project{ResourceID: "proj-1", Workspace: "ws-1"},
	}
	_, err := m.getWebhookContextFromEvent(ctx, e, e.Type)
	a.Error(err)
	a.Contains(err.Error(), "unsupported activity type")
}

func TestGetWebhookContext_IssueApproved_NilEventData(t *testing.T) {
	a := require.New(t)
	m := newTestManager()
	ctx := context.Background()

	// ISSUE_APPROVED with nil IssueApproved data — should still produce a context
	// but with no issue/actor/link. No store calls are made when issue is nil.
	e := &Event{
		Type:          storepb.Activity_ISSUE_APPROVED,
		Project:       &Project{ResourceID: "proj-1", Workspace: "ws-1", Title: "Test Project"},
		IssueApproved: nil,
	}
	webhookCtx, err := m.getWebhookContextFromEvent(ctx, e, e.Type)
	a.NoError(err)
	a.NotNil(webhookCtx)
	a.Equal(webhook.WebhookSuccess, webhookCtx.Level)
	a.Equal("Issue approved", webhookCtx.Title)
	a.Equal("工单审批通过", webhookCtx.TitleZh)
	a.Empty(webhookCtx.Link)
	a.Empty(webhookCtx.Description)
	a.Nil(webhookCtx.Issue)
	a.Equal("Test Project", webhookCtx.Project.Title)
}

func TestGetWebhookContext_IssueCreated_NilEventData(t *testing.T) {
	a := require.New(t)
	m := newTestManager()
	ctx := context.Background()

	e := &Event{
		Type:         storepb.Activity_ISSUE_CREATED,
		Project:      &Project{ResourceID: "proj-1", Workspace: "ws-1", Title: "Test Project"},
		IssueCreated: nil,
	}
	webhookCtx, err := m.getWebhookContextFromEvent(ctx, e, e.Type)
	a.NoError(err)
	a.NotNil(webhookCtx)
	a.Equal(webhook.WebhookInfo, webhookCtx.Level)
	a.Equal("Issue created", webhookCtx.Title)
	a.Equal("创建工单", webhookCtx.TitleZh)
	a.Nil(webhookCtx.Issue)
}

func TestGetWebhookContext_IssueSentBack_NilEventData(t *testing.T) {
	a := require.New(t)
	m := newTestManager()
	ctx := context.Background()

	e := &Event{
		Type:     storepb.Activity_ISSUE_SENT_BACK,
		Project:  &Project{ResourceID: "proj-1", Workspace: "ws-1", Title: "Test Project"},
		SentBack: nil,
	}
	webhookCtx, err := m.getWebhookContextFromEvent(ctx, e, e.Type)
	a.NoError(err)
	a.NotNil(webhookCtx)
	a.Equal(webhook.WebhookWarn, webhookCtx.Level)
	a.Equal("Issue sent back", webhookCtx.Title)
	a.Equal("工单被退回", webhookCtx.TitleZh)
}

func TestGetWebhookContext_PipelineFailed_NilEventData(t *testing.T) {
	a := require.New(t)
	m := newTestManager()
	ctx := context.Background()

	e := &Event{
		Type:          storepb.Activity_PIPELINE_FAILED,
		Project:       &Project{ResourceID: "proj-1", Workspace: "ws-1", Title: "Test Project"},
		RolloutFailed: nil,
	}
	webhookCtx, err := m.getWebhookContextFromEvent(ctx, e, e.Type)
	a.NoError(err)
	a.Equal(webhook.WebhookError, webhookCtx.Level)
	a.Equal("Rollout failed", webhookCtx.Title)
	a.Equal("发布失败", webhookCtx.TitleZh)
	a.Nil(webhookCtx.Rollout)
}

func TestGetWebhookContext_PipelineCompleted_NilEventData(t *testing.T) {
	a := require.New(t)
	m := newTestManager()
	ctx := context.Background()

	e := &Event{
		Type:             storepb.Activity_PIPELINE_COMPLETED,
		Project:          &Project{ResourceID: "proj-1", Workspace: "ws-1", Title: "Test Project"},
		RolloutCompleted: nil,
	}
	webhookCtx, err := m.getWebhookContextFromEvent(ctx, e, e.Type)
	a.NoError(err)
	a.Equal(webhook.WebhookSuccess, webhookCtx.Level)
	a.Equal("Rollout completed", webhookCtx.Title)
	a.Equal("发布完成", webhookCtx.TitleZh)
	a.Nil(webhookCtx.Rollout)
}

func TestGetWebhookContext_RolloutFailed_WithData(t *testing.T) {
	a := require.New(t)
	m := newTestManager()
	ctx := context.Background()

	// PIPELINE_FAILED with rollout data — no store call needed (rollout path skips GetAccountByEmail)
	e := &Event{
		Type:    storepb.Activity_PIPELINE_FAILED,
		Project: &Project{ResourceID: "proj-1", Workspace: "ws-1", Title: "Test Project"},
		RolloutFailed: &EventRolloutFailed{
			Rollout:     &Rollout{UID: 10, Title: "Deploy v2"},
			Environment: "environments/prod",
		},
	}
	webhookCtx, err := m.getWebhookContextFromEvent(ctx, e, e.Type)
	a.NoError(err)
	a.Equal(webhook.WebhookError, webhookCtx.Level)
	a.Equal("Rollout failed", webhookCtx.Description)
	a.Equal("environments/prod", webhookCtx.Environment)
	a.Equal("https://bb.example.com/projects/proj-1/plans/10/rollout", webhookCtx.Link)
	a.NotNil(webhookCtx.Rollout)
	a.Equal(10, webhookCtx.Rollout.UID)
	a.Equal("Deploy v2", webhookCtx.Rollout.Title)
	a.Nil(webhookCtx.Issue, "rollout events should not set issue")
}

func TestGetWebhookContext_RolloutCompleted_WithData(t *testing.T) {
	a := require.New(t)
	m := newTestManager()
	ctx := context.Background()

	e := &Event{
		Type:    storepb.Activity_PIPELINE_COMPLETED,
		Project: &Project{ResourceID: "proj-1", Workspace: "ws-1", Title: "Test Project"},
		RolloutCompleted: &EventRolloutCompleted{
			Rollout:     &Rollout{UID: 20, Title: "Deploy v3"},
			Environment: "environments/staging",
		},
	}
	webhookCtx, err := m.getWebhookContextFromEvent(ctx, e, e.Type)
	a.NoError(err)
	a.Equal(webhook.WebhookSuccess, webhookCtx.Level)
	a.Equal("Rollout completed successfully", webhookCtx.Description)
	a.Equal("environments/staging", webhookCtx.Environment)
	a.Equal("https://bb.example.com/projects/proj-1/plans/20/rollout", webhookCtx.Link)
	a.NotNil(webhookCtx.Rollout)
}

func TestGetWebhookContext_ProjectAlwaysSet(t *testing.T) {
	a := require.New(t)
	m := newTestManager()
	ctx := context.Background()

	e := &Event{
		Type:    storepb.Activity_ISSUE_APPROVED,
		Project: &Project{ResourceID: "my-project", Workspace: "ws-1", Title: "My Project"},
	}
	webhookCtx, err := m.getWebhookContextFromEvent(ctx, e, e.Type)
	a.NoError(err)
	a.NotNil(webhookCtx.Project)
	a.Equal("projects/my-project", webhookCtx.Project.Name)
	a.Equal("My Project", webhookCtx.Project.Title)
}

func TestGetWebhookContext_AllEventLevels(t *testing.T) {
	m := newTestManager()
	ctx := context.Background()

	tests := []struct {
		name          string
		eventType     storepb.Activity_Type
		expectedLevel webhook.Level
	}{
		{"ISSUE_CREATED is INFO", storepb.Activity_ISSUE_CREATED, webhook.WebhookInfo},
		{"ISSUE_APPROVAL_REQUESTED is WARN", storepb.Activity_ISSUE_APPROVAL_REQUESTED, webhook.WebhookWarn},
		{"ISSUE_APPROVED is SUCCESS", storepb.Activity_ISSUE_APPROVED, webhook.WebhookSuccess},
		{"ISSUE_SENT_BACK is WARN", storepb.Activity_ISSUE_SENT_BACK, webhook.WebhookWarn},
		{"PIPELINE_FAILED is ERROR", storepb.Activity_PIPELINE_FAILED, webhook.WebhookError},
		{"PIPELINE_COMPLETED is SUCCESS", storepb.Activity_PIPELINE_COMPLETED, webhook.WebhookSuccess},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := require.New(t)
			e := &Event{
				Type:    tt.eventType,
				Project: &Project{ResourceID: "proj-1", Workspace: "ws-1"},
			}
			webhookCtx, err := m.getWebhookContextFromEvent(ctx, e, e.Type)
			a.NoError(err)
			a.Equal(tt.expectedLevel, webhookCtx.Level)
		})
	}
}

func TestGetWebhookContext_ExternalURLInLink(t *testing.T) {
	m := &Manager{
		store: nil,
		profile: &config.Profile{
			ExternalURL: "https://custom.bytebase.io",
		},
	}
	ctx := context.Background()

	e := &Event{
		Type:    storepb.Activity_PIPELINE_FAILED,
		Project: &Project{ResourceID: "test-proj", Workspace: "ws-1"},
		RolloutFailed: &EventRolloutFailed{
			Rollout: &Rollout{UID: 5, Title: "test"},
		},
	}
	webhookCtx, err := m.getWebhookContextFromEvent(ctx, e, e.Type)
	require.NoError(t, err)
	require.Equal(t, "https://custom.bytebase.io/projects/test-proj/plans/5/rollout", webhookCtx.Link)
}
