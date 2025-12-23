package webhook

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/gosimple/slug"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/iam"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/webhook"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"

	"github.com/pkg/errors"
)

// Manager is the webhook manager.
type Manager struct {
	store      *store.Store
	iamManager *iam.Manager
	profile    *config.Profile
}

// Metadata is the activity metadata.
type Metadata struct {
	Issue *store.IssueMessage
}

type UsersGetter func(ctx context.Context) ([]*store.UserMessage, error)

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
		link = fmt.Sprintf("%s/projects/%s/rollouts/%d", externalURL, e.Project.ResourceID, e.Rollout.UID)
	}
	switch e.Type {
	case storepb.Activity_ISSUE_CREATE:
		title = "Issue created"
		titleZh = "创建工单"

	case storepb.Activity_ISSUE_STATUS_UPDATE:
		switch e.Issue.Status {
		case "OPEN":
			title = "Issue reopened"
			titleZh = "工单重开"
		case "DONE":
			level = webhook.WebhookSuccess
			title = "Issue resolved"
			titleZh = "工单完成"
		case "CANCELED":
			title = "Issue canceled"
			titleZh = "工单取消"
		default:
			title = "Issue status changed"
			titleZh = "工单状态变更"
		}

	case storepb.Activity_ISSUE_COMMENT_CREATE:
		title = "Comment created"
		titleZh = "工单新评论"

	case storepb.Activity_ISSUE_FIELD_UPDATE:
		update := e.IssueUpdate
		switch update.Path {
		case "description":
			title = "Changed issue description"
			titleZh = "工单描述变更"
		case "title":
			title = "Changed issue name"
			titleZh = "工单标题变更"
		default:
			title = "Updated issue"
			titleZh = "工单信息变更"
		}

	case storepb.Activity_ISSUE_PIPELINE_STAGE_STATUS_UPDATE:
		u := e.StageStatusUpdate
		if e.Issue != nil {
			stageID := u.StageID
			if stageID == "" {
				stageID = "-" // Use "-" as a placeholder if StageID is not set.
			}
			link = fmt.Sprintf("%s/projects/%s/issues/%s-%d?stage=%s", setting.ExternalUrl, e.Project.ResourceID, slug.Make(e.Issue.Title), e.Issue.UID, stageID)
		}
		title = "Stage ends"
		titleZh = "阶段结束"

	case storepb.Activity_ISSUE_PIPELINE_TASK_RUN_STATUS_UPDATE:
		u := e.TaskRunStatusUpdate
		switch u.Status {
		case storepb.TaskRun_PENDING.String():
			title = "Task run started"
			titleZh = "任务开始"
		case storepb.TaskRun_RUNNING.String():
			title = "Task run is running"
			titleZh = "任务运行中"
		case storepb.TaskRun_DONE.String():
			level = webhook.WebhookSuccess
			title = "Task run completed"
			titleZh = "任务完成"
		case storepb.TaskRun_FAILED.String():
			level = webhook.WebhookError
			title = "Task run failed"
			titleZh = "任务失败"
		case storepb.TaskRun_CANCELED.String():
			title = "Task run is canceled"
			titleZh = "任务取消"
		case storepb.TaskRun_SKIPPED.String():
			title = "Task is skipped"
			titleZh = "任务跳过"
		default:
			title = "Task run status changed"
			titleZh = "任务状态变更"
		}

	case storepb.Activity_NOTIFY_ISSUE_APPROVED:
		title = "Issue approved"
		titleZh = "工单审批通过"
		creatorUser, err := m.store.GetUserByEmail(ctx, e.Issue.CreatorEmail)
		if err != nil {
			slog.Warn("failed to get creator user for issue notification",
				slog.String("issue_name", e.Issue.Title),
				log.BBError(err))
			// Continue without mentioning the creator if unable to fetch
		} else {
			mentionUsers = append(mentionUsers, creatorUser)
		}

	case storepb.Activity_NOTIFY_PIPELINE_ROLLOUT:
		u := e.IssueRolloutReady
		title = "Issue is waiting for rollout"
		titleZh = "工单待发布"
		var usersGetters []UsersGetter
		if u.RolloutPolicy.GetAutomatic() {
			creatorUser, err := m.store.GetUserByEmail(ctx, e.Issue.CreatorEmail)
			if err != nil {
				slog.Warn("failed to get creator user for issue notification",
					slog.String("issue_name", e.Issue.Title),
					log.BBError(err))
			} else {
				usersGetters = append(usersGetters, getUsersFromUsers(creatorUser))
			}
		} else {
			for _, role := range u.RolloutPolicy.GetRoles() {
				role := strings.TrimPrefix(role, "roles/")
				usersGetters = append(usersGetters, getUsersFromRole(m.store, role, e.Project.ResourceID))
			}
		}
		mentionUsers = getUsersForDirectMessage(ctx, e, usersGetters...)

	case storepb.Activity_ISSUE_APPROVAL_NOTIFY:
		roleWithPrefix := e.IssueApprovalCreate.Role

		title = "Issue approval needed"
		titleZh = "工单待审批"

		var usersGetter UsersGetter
		role := strings.TrimPrefix(roleWithPrefix, "roles/")
		usersGetter = getUsersFromRole(m.store, role, e.Project.ResourceID)
		mentionUsers = getUsersForDirectMessage(ctx, e, usersGetter)

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
		Stage:           nil,
		TaskResult:      nil,
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
	if u := e.TaskRunStatusUpdate; u != nil {
		webhookCtx.TaskResult = &webhook.TaskResult{
			Name:          u.Title,
			Status:        u.Status,
			Detail:        u.Detail,
			SkippedReason: u.SkippedReason,
		}
	}
	if u := e.StageStatusUpdate; u != nil {
		webhookCtx.Stage = &webhook.Stage{
			Name: u.StageTitle,
		}
	}

	return &webhookCtx, nil
}

func getUsersForDirectMessage(ctx context.Context, e *Event, usersGetters ...UsersGetter) []*store.UserMessage {
	mentionedUser := map[int]bool{}
	mentionUsers := []*store.UserMessage{}

	for _, usersGetter := range usersGetters {
		users, err := usersGetter(ctx)
		if err != nil {
			slog.Warn("failed to get users",
				slog.String("event", e.Type.String()),
				slog.String("issue_name", e.Issue.Title),
				slog.Int("issue_uid", e.Issue.UID),
				log.BBError(err))
			continue
		}
		for _, user := range users {
			if mentionedUser[user.ID] {
				continue
			}
			if user.MemberDeleted {
				continue
			}
			mentionedUser[user.ID] = true
			mentionUsers = append(mentionUsers, user)
		}
	}
	return mentionUsers
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

func getUsersFromRole(s *store.Store, role string, projectID string) UsersGetter {
	return func(ctx context.Context) ([]*store.UserMessage, error) {
		projectIAM, err := s.GetProjectIamPolicy(ctx, projectID)
		if err != nil {
			return nil, err
		}
		workspaceIAM, err := s.GetWorkspaceIamPolicy(ctx)
		if err != nil {
			return nil, err
		}

		return utils.GetUsersByRoleInIAMPolicy(ctx, s, role, projectIAM.Policy, workspaceIAM.Policy), nil
	}
}

func getUsersFromUsers(users ...*store.UserMessage) UsersGetter {
	return func(_ context.Context) ([]*store.UserMessage, error) {
		return users, nil
	}
}

// ChangeIssueStatus changes the status of an issue.
func ChangeIssueStatus(ctx context.Context, stores *store.Store, webhookManager *Manager, issue *store.IssueMessage, newStatus storepb.Issue_Status, updater *store.UserMessage, comment string) error {
	updateIssueMessage := &store.UpdateIssueMessage{Status: &newStatus}
	updatedIssue, err := stores.UpdateIssue(ctx, issue.UID, updateIssueMessage)
	if err != nil {
		return errors.Wrapf(err, "failed to update issue %q's status", issue.Title)
	}

	project, err := stores.GetProject(ctx, &store.FindProjectMessage{ResourceID: &updatedIssue.ProjectID})
	if err != nil {
		return errors.Wrapf(err, "failed to get project")
	}
	if project == nil {
		return errors.Errorf("project %s not found", updatedIssue.ProjectID)
	}

	webhookManager.CreateEvent(ctx, &Event{
		Actor:   updater,
		Type:    storepb.Activity_ISSUE_STATUS_UPDATE,
		Comment: comment,
		Issue:   NewIssue(updatedIssue),
		Project: NewProject(project),
	})
	return nil
}
