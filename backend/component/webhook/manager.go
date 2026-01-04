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
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/webhook"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"

	"github.com/pkg/errors"
)

// Manager is the webhook manager.
type Manager struct {
	store   *store.Store
	profile *config.Profile
}

// NewManager creates an activity manager.
func NewManager(store *store.Store, profile *config.Profile) *Manager {
	return &Manager{
		store:   store,
		profile: profile,
	}
}

func (m *Manager) CreateEvent(ctx context.Context, e *Event) {
	webhookList, err := m.store.ListProjectWebhooks(ctx, &store.FindProjectWebhookMessage{
		ProjectID: &e.Project.ResourceID,
		EventType: &e.Type,
	})
	if err != nil {
		slog.Warn("failed to find project webhook",
			slog.String("project", e.Project.ResourceID),
			slog.String("event_type", e.Type.String()),
			log.BBError(err))
		return
	}

	if len(webhookList) == 0 {
		return
	}

	webhookCtx, err := m.getWebhookContextFromEvent(ctx, e, e.Type)
	if err != nil {
		slog.Warn("failed to get webhook context",
			slog.String("project", e.Project.ResourceID),
			slog.String("event_type", e.Type.String()),
			log.BBError(err))
		return
	}
	// Call external webhook endpoint in Go routine to avoid blocking web serving thread.
	go m.postWebhookList(ctx, webhookCtx, webhookList)
}

func (m *Manager) getWebhookContextFromEvent(ctx context.Context, e *Event, eventType storepb.Activity_Type) (*webhook.Context, error) {
	var webhookCtx webhook.Context
	var mentionUsers []*store.UserMessage

	// Use command-line flag value if set, otherwise use database value
	externalURL, err := utils.GetEffectiveExternalURL(ctx, m.store, m.profile)
	if err != nil {
		return nil, err
	}

	level := webhook.WebhookInfo
	title := ""
	titleZh := ""
	link := ""
	var actor *User
	var issue *Issue
	var rollout *Rollout

	switch e.Type {
	case storepb.Activity_ISSUE_CREATED:
		title = "Issue created"
		titleZh = "创建工单"
		if e.IssueCreated != nil {
			actor = e.IssueCreated.Creator
			issue = e.IssueCreated.Issue
			link = fmt.Sprintf("%s/projects/%s/issues/%s-%d", externalURL, e.Project.ResourceID, slug.Make(issue.Title), issue.UID)
			webhookCtx.Description = fmt.Sprintf("%s created issue %s", actor.Name, issue.Title)
		}

	case storepb.Activity_ISSUE_APPROVAL_REQUESTED:
		level = webhook.WebhookWarn
		title = "Approval required"
		titleZh = "需要审批"
		if e.ApprovalRequested != nil {
			actor = e.ApprovalRequested.Creator
			issue = e.ApprovalRequested.Issue
			link = fmt.Sprintf("%s/projects/%s/issues/%s-%d", externalURL, e.Project.ResourceID, slug.Make(issue.Title), issue.UID)
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
			actor = e.SentBack.Approver
			issue = e.SentBack.Issue
			link = fmt.Sprintf("%s/projects/%s/issues/%s-%d", externalURL, e.Project.ResourceID, slug.Make(issue.Title), issue.UID)
			webhookCtx.Description = fmt.Sprintf("%s sent back the issue: %s", e.SentBack.Approver.Name, e.SentBack.Reason)
			mentionUsers = []*store.UserMessage{
				{
					Name:  e.SentBack.Creator.Name,
					Email: e.SentBack.Creator.Email,
				},
			}
		}

	case storepb.Activity_PIPELINE_FAILED:
		level = webhook.WebhookError
		title = "Rollout failed"
		titleZh = "发布失败"
		if e.RolloutFailed != nil {
			rollout = e.RolloutFailed.Rollout
			link = fmt.Sprintf("%s/projects/%s/plans/%d/rollout", externalURL, e.Project.ResourceID, rollout.UID)
			webhookCtx.Description = "Rollout failed"
		}

	case storepb.Activity_PIPELINE_COMPLETED:
		level = webhook.WebhookSuccess
		title = "Rollout completed"
		titleZh = "发布完成"
		if e.RolloutCompleted != nil {
			rollout = e.RolloutCompleted.Rollout
			link = fmt.Sprintf("%s/projects/%s/plans/%d/rollout", externalURL, e.Project.ResourceID, rollout.UID)
			webhookCtx.Description = "Rollout completed successfully"
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
		Level:           level,
		EventType:       string(eventType),
		Title:           title,
		TitleZh:         titleZh,
		Link:            link,
		MentionEndUsers: mentionEndUsers,
		Project: &webhook.Project{
			Name:  common.FormatProject(e.Project.ResourceID),
			Title: e.Project.Title,
		},
	}

	// Set actor information if available
	if actor != nil {
		webhookCtx.ActorID = 0 // We don't have ID in User struct
		webhookCtx.ActorName = actor.Name
		webhookCtx.ActorEmail = actor.Email
	}

	// Set issue information if available
	if issue != nil {
		creatorName := issue.Title // Fallback
		creatorUser, err := m.store.GetUserByEmail(ctx, actor.Email)
		if err != nil {
			slog.Warn("failed to get creator user for webhook context",
				slog.String("issue_title", issue.Title),
				log.BBError(err))
		} else {
			creatorName = creatorUser.Name
		}
		webhookCtx.Issue = &webhook.Issue{
			ID:          issue.UID,
			Name:        issue.Title,
			Status:      issue.Status,
			Type:        issue.Type,
			Description: issue.Description,
			Creator: webhook.Creator{
				Name:  creatorName,
				Email: actor.Email,
			},
		}
	}

	// Set rollout information if available
	if rollout != nil {
		webhookCtx.Rollout = &webhook.Rollout{
			UID:   rollout.UID,
			Title: rollout.Title,
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
