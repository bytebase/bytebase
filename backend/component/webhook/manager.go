package webhook

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/gosimple/slug"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/iam"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/webhook"
	"github.com/bytebase/bytebase/backend/store"

	"github.com/pkg/errors"
)

// Manager is the webhook manager.
type Manager struct {
	store      *store.Store
	iamManager *iam.Manager
	profile    *config.Profile
}

// NewManager creates an activity manager.
func NewManager(store *store.Store, iamManager *iam.Manager, profile *config.Profile) *Manager {
	return &Manager{
		store:      store,
		iamManager: iamManager,
		profile:    profile,
	}
}

func (m *Manager) CreateEvent(ctx context.Context, e *Event) {
	webhookList, err := m.store.ListProjectWebhooks(ctx, &store.FindProjectWebhookMessage{
		ProjectID: &e.Project.ResourceID,
		EventType: &e.Type,
	})
	if err != nil {
		slog.Warn("failed to find project webhook", "issue_name", e.Issue.Title, log.BBError(err))
		return
	}

	if len(webhookList) == 0 {
		return
	}

	webhookCtx, err := m.getWebhookContextFromEvent(ctx, e, e.Type)
	if err != nil {
		slog.Warn("failed to get webhook context",
			slog.String("issue_name", e.Issue.Title),
			log.BBError(err))
		return
	}
	// Call external webhook endpoint in Go routine to avoid blocking web serving thread.
	go m.postWebhookList(ctx, webhookCtx, webhookList)
}

func (m *Manager) getWebhookContextFromEvent(ctx context.Context, e *Event, eventType storepb.Activity_Type) (*webhook.Context, error) {
	var webhookCtx webhook.Context
	var mentionUsers []*store.UserMessage

	setting, err := m.store.GetWorkspaceProfileSetting(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get workspace setting")
	}

	// Use command-line flag value if set, otherwise use database value
	externalURL := common.GetEffectiveExternalURL(m.profile.ExternalURL, setting.ExternalUrl)

	level := webhook.WebhookInfo
	title := ""
	titleZh := ""
	link := ""
	if e.Issue != nil {
		// TODO(steven): Remove the slug dependency when the legacy issue page is removed.
		link = fmt.Sprintf("%s/projects/%s/issues/%s-%d", externalURL, e.Project.ResourceID, slug.Make(e.Issue.Title), e.Issue.UID)
	} else if e.Rollout != nil {
		link = fmt.Sprintf("%s/projects/%s/plans/%d/rollout", externalURL, e.Project.ResourceID, e.Rollout.UID)
	}
	switch e.Type {
	case storepb.Activity_ISSUE_CREATED:
		title = "Issue created"
		titleZh = "创建工单"
		if e.IssueCreated != nil {
			webhookCtx.Description = fmt.Sprintf("%s created issue %s", e.IssueCreated.CreatorName, e.Issue.Title)
		}

	case storepb.Activity_ISSUE_APPROVAL_REQUESTED:
		level = webhook.WebhookWarn
		title = "Approval required"
		titleZh = "需要审批"
		if e.ApprovalRequested != nil {
			webhookCtx.ApprovalRole = e.ApprovalRequested.ApprovalRole
			webhookCtx.ApprovalRequired = e.ApprovalRequested.RequiredCount
			mentionUsers = make([]*store.UserMessage, 0, len(e.ApprovalRequested.Approvers))
			for _, user := range e.ApprovalRequested.Approvers {
				mentionUsers = append(mentionUsers, &store.UserMessage{
					Name:  user.Name,
					Email: user.Email,
				})
			}
		}

	case storepb.Activity_ISSUE_SENT_BACK:
		level = webhook.WebhookWarn
		title = "Issue sent back"
		titleZh = "工单被退回"
		if e.SentBack != nil {
			webhookCtx.Description = fmt.Sprintf("%s sent back the issue: %s", e.SentBack.ApproverName, e.SentBack.Reason)
			mentionUsers = []*store.UserMessage{
				{
					Name:  e.SentBack.CreatorName,
					Email: e.SentBack.CreatorEmail,
				},
			}
		}

	case storepb.Activity_PIPELINE_FAILED:
		level = webhook.WebhookError
		title = "Pipeline failed"
		titleZh = "流水线失败"
		if e.PipelineFailed != nil {
			failedTasks := make([]webhook.FailedTaskInfo, 0, len(e.PipelineFailed.FailedTasks))
			for _, task := range e.PipelineFailed.FailedTasks {
				failedTasks = append(failedTasks, webhook.FailedTaskInfo{
					Name:         task.TaskName,
					Instance:     task.InstanceName,
					Database:     task.DatabaseName,
					ErrorMessage: task.ErrorMessage,
					FailedAt:     task.FailedAt.Format(time.RFC3339),
				})
			}
			webhookCtx.FailedTasks = failedTasks
			webhookCtx.Description = fmt.Sprintf("%d task(s) failed", len(failedTasks))
		}

	case storepb.Activity_PIPELINE_COMPLETED:
		level = webhook.WebhookSuccess
		title = "Pipeline completed"
		titleZh = "流水线完成"
		if e.PipelineCompleted != nil {
			duration := e.PipelineCompleted.CompletedAt.Sub(e.PipelineCompleted.StartedAt)
			webhookCtx.PipelineMetrics = &webhook.PipelineMetrics{
				TotalTasks:   e.PipelineCompleted.TotalTasks,
				StartedAt:    e.PipelineCompleted.StartedAt.Format(time.RFC3339),
				CompletedAt:  e.PipelineCompleted.CompletedAt.Format(time.RFC3339),
				DurationSecs: int64(duration.Seconds()),
			}
			webhookCtx.Description = fmt.Sprintf("Completed %d task(s) in %s", e.PipelineCompleted.TotalTasks, duration.String())
		}

	default:
		// Unsupported event type
		return nil, errors.Errorf("unsupported activity type %q for generating webhook context", e.Type)
	}

	var mentionEndUsers []*store.UserMessage
	for _, u := range mentionUsers {
		if u.Type == storepb.PrincipalType_END_USER {
			mentionEndUsers = append(mentionEndUsers, u)
		}
	}

	webhookCtx = webhook.Context{
		Level:     level,
		EventType: string(eventType),
		Title:     title,
		TitleZh:   titleZh,
		Issue:     nil,
		Rollout:   nil,
		Project: &webhook.Project{
			Name:  common.FormatProject(e.Project.ResourceID),
			Title: e.Project.Title,
		},
		Description:     e.Comment,
		Link:            link,
		ActorID:         e.Actor.ID,
		ActorName:       e.Actor.Name,
		ActorEmail:      e.Actor.Email,
		MentionEndUsers: mentionEndUsers,
	}
	if e.Issue != nil {
		creatorName := e.Issue.CreatorEmail
		creatorUser, err := m.store.GetUserByEmail(ctx, e.Issue.CreatorEmail)
		if err != nil {
			slog.Warn("failed to get creator user for webhook context",
				slog.String("issue_name", e.Issue.Title),
				log.BBError(err))
		} else {
			creatorName = creatorUser.Name
		}
		webhookCtx.Issue = &webhook.Issue{
			ID:          e.Issue.UID,
			Name:        e.Issue.Title,
			Status:      e.Issue.Status,
			Type:        e.Issue.Type,
			Description: e.Issue.Description,
			Creator: webhook.Creator{
				Name:  creatorName,
				Email: e.Issue.CreatorEmail,
			},
		}
	}
	if e.Rollout != nil {
		webhookCtx.Rollout = &webhook.Rollout{
			UID: e.Rollout.UID,
		}
	}

	return &webhookCtx, nil
}

func (m *Manager) postWebhookList(ctx context.Context, webhookCtx *webhook.Context, webhookList []*store.ProjectWebhookMessage) {
	ctx = context.WithoutCancel(ctx)
	setting, err := m.store.GetAppIMSetting(ctx)
	if err != nil {
		slog.Error("failed to get app im setting", log.BBError(err))
	} else {
		webhookCtx.IMSetting = setting
	}

	for _, hook := range webhookList {
		webhookCtx := *webhookCtx
		webhookCtx.URL = hook.Payload.GetUrl()
		webhookCtx.CreatedTS = time.Now().Unix()
		webhookCtx.DirectMessage = hook.Payload.GetDirectMessage()
		go func(webhookCtx *webhook.Context, hook *store.ProjectWebhookMessage) {
			if err := common.Retry(ctx, func() error {
				return webhook.Post(hook.Payload.GetType(), *webhookCtx)
			}); err != nil {
				// The external webhook endpoint might be invalid which is out of our code control, so we just emit a warning
				slog.Warn("failed to post webhook event on activity",
					slog.String("webhook type", hook.Payload.GetType().String()),
					slog.String("webhook name", hook.Payload.GetTitle()),
					slog.String("activity type", webhookCtx.EventType),
					slog.String("title", webhookCtx.Title),
					log.BBError(err))
				return
			}
		}(&webhookCtx, hook)
	}
}
