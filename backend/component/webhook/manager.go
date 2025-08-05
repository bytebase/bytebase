package webhook

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/gosimple/slug"
	"github.com/nyaruka/phonenumbers"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
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
}

// Metadata is the activity metadata.
type Metadata struct {
	Issue *store.IssueMessage
}

// NewManager creates an activity manager.
func NewManager(store *store.Store, iamManager *iam.Manager) *Manager {
	return &Manager{
		store:      store,
		iamManager: iamManager,
	}
}

func (m *Manager) CreateEvent(ctx context.Context, e *Event) {
	webhookList, err := m.store.FindProjectWebhookV2(ctx, &store.FindProjectWebhookMessage{
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

func (m *Manager) getWebhookContextFromEvent(ctx context.Context, e *Event, eventType common.EventType) (*webhook.Context, error) {
	var webhookCtx webhook.Context
	var mentions []string
	var mentionUsers []*store.UserMessage

	setting, err := m.store.GetWorkspaceGeneralSetting(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get workspace setting")
	}

	level := webhook.WebhookInfo
	title := ""
	titleZh := ""
	link := ""
	if e.Issue != nil {
		// TODO(steven): Remove the slug dependency when the legacy issue page is removed.
		link = fmt.Sprintf("%s/projects/%s/issues/%s-%d", setting.ExternalUrl, e.Project.ResourceID, slug.Make(e.Issue.Title), e.Issue.UID)
	} else if e.Rollout != nil {
		link = fmt.Sprintf("%s/projects/%s/rollouts/%d", setting.ExternalUrl, e.Project.ResourceID, e.Rollout.UID)
	}
	switch e.Type {
	case common.EventTypeIssueCreate:
		title = "Issue created"
		titleZh = "创建工单"

	case common.EventTypeIssueStatusUpdate:
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

	case common.EventTypeIssueCommentCreate:
		title = "Comment created"
		titleZh = "工单新评论"

	case common.EventTypeIssueUpdate:
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

	case common.EventTypeStageStatusUpdate:
		u := e.StageStatusUpdate
		if e.Issue != nil {
			link = fmt.Sprintf("%s/projects/%s/issues/%s-%d?stage=%s", setting.ExternalUrl, e.Project.ResourceID, slug.Make(e.Issue.Title), e.Issue.UID, u.StageID)
		}
		title = "Stage ends"
		titleZh = "阶段结束"

	case common.EventTypeTaskRunStatusUpdate:
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

	case common.EventTypeIssueApprovalPass:
		title = "Issue approved"
		titleZh = "工单审批通过"

		mentionUsers = append(mentionUsers, e.Issue.Creator)
		phone, err := maybeGetPhoneFromUser(e.Issue.Creator)
		if err != nil {
			slog.Warn("failed to parse phone number", slog.String("issue_title", e.Issue.Title), log.BBError(err))
		} else if phone != "" {
			mentions = append(mentions, phone)
		}

	case common.EventTypeIssueRolloutReady:
		u := e.IssueRolloutReady
		title = "Issue is waiting for rollout"
		titleZh = "工单待发布"
		var usersGetters []func(context.Context) ([]*store.UserMessage, error)
		if u.RolloutPolicy.GetAutomatic() {
			usersGetters = append(usersGetters, getUsersFromUsers(e.Issue.Creator))
		} else {
			for _, role := range u.RolloutPolicy.GetRoles() {
				role := strings.TrimPrefix(role, "roles/")
				usersGetters = append(usersGetters, getUsersFromRole(m.store, role, e.Project.ResourceID))
			}
			for _, issueRole := range u.RolloutPolicy.GetIssueRoles() {
				switch issueRole {
				case "roles/LAST_APPROVER":
					usersGetters = append(usersGetters, getUsersFromIssueLastApprover(m.store, e.Issue.Approval))
				case "roles/CREATOR":
					usersGetters = append(usersGetters, getUsersFromUsers(e.Issue.Creator))
				default:
					// Skip unknown issue roles
				}
			}
		}
		mentionedUser := map[int]bool{}
		for _, usersGetter := range usersGetters {
			users, err := usersGetter(ctx)
			if err != nil {
				slog.Warn("failed to get users",
					slog.String("issue_name", e.Issue.Title),
					log.BBError(err))
				return nil, err
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
				phone, err := maybeGetPhoneFromUser(user)
				if err != nil {
					slog.Warn("failed to parse phone number",
						slog.String("issue_name", e.Issue.Title),
						log.BBError(err))
					continue
				}
				if phone != "" {
					mentions = append(mentions, phone)
				}
			}
		}

	case common.EventTypeIssueApprovalCreate:
		pendingStep := e.IssueApprovalCreate.ApprovalStep

		title = "Issue approval needed"
		titleZh = "工单待审批"

		if len(pendingStep.Nodes) != 1 {
			slog.Warn("Failed to post webhook event after changing the issue approval node status, pending step nodes length is not 1")
			return nil, errors.Errorf("pending step nodes length is not 1, got %v", len(pendingStep.Nodes))
		}

		node := pendingStep.Nodes[0]

		var usersGetter func(ctx context.Context) ([]*store.UserMessage, error)

		role := strings.TrimPrefix(node.Role, "roles/")
		usersGetter = getUsersFromRole(m.store, role, e.Project.ResourceID)

		users, err := usersGetter(ctx)
		if err != nil {
			slog.Warn("Failed to post webhook event after changing the issue approval node status, failed to get users",
				slog.String("issue_name", e.Issue.Title),
				log.BBError(err))
			return nil, err
		}
		for _, user := range users {
			mentionUsers = append(mentionUsers, user)
			phone, err := maybeGetPhoneFromUser(user)
			if err != nil {
				slog.Warn("failed to parse phone number",
					slog.String("issue_name", e.Issue.Title),
					log.BBError(err))
				continue
			}
			if phone != "" {
				mentions = append(mentions, phone)
			}
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
		Stage:               nil,
		TaskResult:          nil,
		Description:         e.Comment,
		Link:                link,
		ActorID:             e.Actor.ID,
		ActorName:           e.Actor.Name,
		ActorEmail:          e.Actor.Email,
		MentionEndUsers:     mentionEndUsers,
		MentionUsersByPhone: mentions,
	}
	if e.Issue != nil {
		webhookCtx.Issue = &webhook.Issue{
			ID:          e.Issue.UID,
			Name:        e.Issue.Title,
			Status:      e.Issue.Status,
			Type:        e.Issue.Type,
			Description: e.Issue.Description,
			Creator:     e.Issue.Creator,
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
		webhookCtx.URL = hook.URL
		webhookCtx.CreatedTS = time.Now().Unix()
		webhookCtx.DirectMessage = hook.Payload.GetDirectMessage()
		go func(webhookCtx *webhook.Context, hook *store.ProjectWebhookMessage) {
			if err := common.Retry(ctx, func() error {
				return webhook.Post(hook.Type, *webhookCtx)
			}); err != nil {
				// The external webhook endpoint might be invalid which is out of our code control, so we just emit a warning
				slog.Warn("Failed to post webhook event on activity",
					slog.String("webhook type", hook.Type),
					slog.String("webhook name", hook.Title),
					slog.String("activity type", webhookCtx.EventType),
					slog.String("title", webhookCtx.Title),
					log.BBError(err))
				return
			}
		}(&webhookCtx, hook)
	}
}

func getUsersFromRole(s *store.Store, role string, projectID string) func(context.Context) ([]*store.UserMessage, error) {
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

func getUsersFromUsers(users ...*store.UserMessage) func(context.Context) ([]*store.UserMessage, error) {
	return func(_ context.Context) ([]*store.UserMessage, error) {
		return users, nil
	}
}

func getUsersFromIssueLastApprover(s *store.Store, approval *storepb.IssuePayloadApproval) func(context.Context) ([]*store.UserMessage, error) {
	return func(ctx context.Context) ([]*store.UserMessage, error) {
		var userUID int
		if approvers := approval.GetApprovers(); len(approvers) > 0 {
			userUID = int(approvers[len(approvers)-1].PrincipalId)
		}
		if userUID == 0 {
			return nil, nil
		}
		user, err := s.GetUserByID(ctx, userUID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get user")
		}
		return []*store.UserMessage{user}, nil
	}
}

func maybeGetPhoneFromUser(user *store.UserMessage) (string, error) {
	if user == nil {
		return "", nil
	}
	if user.Phone == "" {
		return "", nil
	}
	phoneNumber, err := phonenumbers.Parse(user.Phone, "")
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse phone number %q", user.Phone)
	}
	if phoneNumber == nil {
		return "", nil
	}
	if phoneNumber.NationalNumber == nil {
		return "", nil
	}
	return strconv.FormatInt(int64(*phoneNumber.NationalNumber), 10), nil
}

// ChangeIssueStatus changes the status of an issue.
func ChangeIssueStatus(ctx context.Context, stores *store.Store, webhookManager *Manager, issue *store.IssueMessage, newStatus storepb.Issue_Status, updater *store.UserMessage, comment string) error {
	updateIssueMessage := &store.UpdateIssueMessage{Status: &newStatus}
	updatedIssue, err := stores.UpdateIssueV2(ctx, issue.UID, updateIssueMessage)
	if err != nil {
		return errors.Wrapf(err, "failed to update issue %q's status", issue.Title)
	}

	// In the ChangeIssueStatus function
	webhookManager.CreateEvent(ctx, &Event{
		Actor:   updater,
		Type:    common.EventTypeIssueStatusUpdate,
		Comment: comment,
		Issue:   NewIssue(updatedIssue),
		Project: NewProject(updatedIssue.Project),
	})
	return nil
}
