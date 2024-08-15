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
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/webhook"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"

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
	var activityType api.ActivityType
	//exhaustive:enforce
	switch e.Type {
	case EventTypeIssueCreate:
		activityType = api.ActivityIssueCreate
	case EventTypeIssueUpdate:
		activityType = api.ActivityIssueFieldUpdate
	case EventTypeIssueStatusUpdate:
		activityType = api.ActivityIssueStatusUpdate
	case EventTypeIssueCommentCreate:
		activityType = api.ActivityIssueCommentCreate
	case EventTypeIssueApprovalCreate:
		activityType = api.ActivityIssueApprovalNotify
	case EventTypeIssueApprovalPass:
		activityType = api.ActivityNotifyIssueApproved
	case EventTypeIssueRolloutReady:
		activityType = api.ActivityNotifyPipelineRollout
	case EventTypeStageStatusUpdate:
		activityType = api.ActivityPipelineStageStatusUpdate
	case EventTypeTaskRunStatusUpdate:
		activityType = api.ActivityPipelineTaskRunStatusUpdate
	default:
		return
	}
	webhookList, err := m.store.FindProjectWebhookV2(ctx, &store.FindProjectWebhookMessage{
		ProjectID:    &e.Project.UID,
		ActivityType: &activityType,
	})
	if err != nil {
		slog.Warn("failed to find project webhook", "issue_name", e.Issue.Title, log.BBError(err))
		return
	}

	if len(webhookList) == 0 {
		return
	}

	webhookCtx, err := m.getWebhookContextFromEvent(ctx, e, activityType)
	if err != nil {
		slog.Warn("failed to get webhook context",
			slog.String("issue_name", e.Issue.Title),
			log.BBError(err))
		return
	}
	// Call external webhook endpoint in Go routine to avoid blocking web serving thread.
	go m.postWebhookList(ctx, webhookCtx, webhookList)
}

func (m *Manager) getWebhookContextFromEvent(ctx context.Context, e *Event, activityType api.ActivityType) (*webhook.Context, error) {
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
	link := fmt.Sprintf("%s/projects/%s/issues/%s-%d", setting.ExternalUrl, e.Project.ResourceID, slug.Make(e.Issue.Title), e.Issue.UID)
	switch e.Type {
	case EventTypeIssueCreate:
		title = "Issue created"
		titleZh = "创建工单"

	case EventTypeIssueStatusUpdate:
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
		}

	case EventTypeIssueCommentCreate:
		title = "Comment created"
		titleZh = "工单新评论"

	case EventTypeIssueUpdate:
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

	case EventTypeStageStatusUpdate:
		u := e.StageStatusUpdate
		link += fmt.Sprintf("?stage=%d", u.StageUID)
		title = "Stage ends"
		titleZh = "阶段结束"

	case EventTypeTaskRunStatusUpdate:
		u := e.TaskRunStatusUpdate
		switch u.Status {
		case api.TaskRunPending.String():
			title = "Task run started"
			titleZh = "任务开始"
		case api.TaskRunRunning.String():
			title = "Task run is running"
			titleZh = "任务运行中"
		case api.TaskRunDone.String():
			level = webhook.WebhookSuccess
			title = "Task run completed"
			titleZh = "任务完成"
		case api.TaskRunFailed.String():
			level = webhook.WebhookError
			title = "Task run failed"
			titleZh = "任务失败"
		case api.TaskRunCanceled.String():
			title = "Task run is canceled"
			titleZh = "任务取消"
		case api.TaskRunSkipped.String():
			title = "Task is skipped"
			titleZh = "任务跳过"
		default:
			title = "Task run status changed"
			titleZh = "任务状态变更"
		}

	case EventTypeIssueApprovalPass:
		title = "Issue approved"
		titleZh = "工单审批通过"

		mentionUsers = append(mentionUsers, e.Issue.Creator)
		phone, err := maybeGetPhoneFromUser(e.Issue.Creator)
		if err != nil {
			slog.Warn("failed to parse phone number", slog.String("issue_title", e.Issue.Title), log.BBError(err))
		} else if phone != "" {
			mentions = append(mentions, phone)
		}

	case EventTypeIssueRolloutReady:
		u := e.IssueRolloutReady
		title = "Issue is waiting for rollout"
		titleZh = "工单待发布"
		var usersGetters []func(context.Context) ([]*store.UserMessage, error)
		if u.RolloutPolicy.GetAutomatic() {
			usersGetters = append(usersGetters, getUsersFromUsers(e.Issue.Creator))
		} else {
			for _, workspaceRole := range u.RolloutPolicy.GetWorkspaceRoles() {
				role := api.Role(strings.TrimPrefix(workspaceRole, "roles/"))
				usersGetters = append(usersGetters, m.getUsersFromWorkspaceRole(role))
			}
			for _, projectRole := range u.RolloutPolicy.GetProjectRoles() {
				role := api.Role(strings.TrimPrefix(projectRole, "roles/"))
				usersGetters = append(usersGetters, getUsersFromProjectRole(m.store, role, e.Project.UID))
			}
			for _, issueRole := range u.RolloutPolicy.GetIssueRoles() {
				switch issueRole {
				case "roles/LAST_APPROVER":
					usersGetters = append(usersGetters, getUsersFromIssueLastApprover(m.store, e.Issue.Approval))
				case "roles/CREATOR":
					usersGetters = append(usersGetters, getUsersFromUsers(e.Issue.Creator))
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

	case EventTypeIssueApprovalCreate:
		pendingStep := e.IssueApprovalCreate.ApprovalStep

		title = "Issue approval needed"
		titleZh = "工单待审批"

		if len(pendingStep.Nodes) != 1 {
			slog.Warn("Failed to post webhook event after changing the issue approval node status, pending step nodes length is not 1")
			return nil, errors.Errorf("pending step nodes length is not 1, got %v", len(pendingStep.Nodes))
		}

		node := pendingStep.Nodes[0]

		var usersGetter func(ctx context.Context) ([]*store.UserMessage, error)

		switch val := node.Payload.(type) {
		case *storepb.ApprovalNode_GroupValue_:
			switch val.GroupValue {
			case storepb.ApprovalNode_GROUP_VALUE_UNSPECIFILED:
				return nil, errors.Errorf("invalid group value")
			case storepb.ApprovalNode_WORKSPACE_OWNER:
				usersGetter = m.getUsersFromWorkspaceRole(api.WorkspaceAdmin)
			case storepb.ApprovalNode_WORKSPACE_DBA:
				usersGetter = m.getUsersFromWorkspaceRole(api.WorkspaceDBA)
			case storepb.ApprovalNode_PROJECT_OWNER:
				usersGetter = getUsersFromProjectRole(m.store, api.ProjectOwner, e.Project.UID)
			case storepb.ApprovalNode_PROJECT_MEMBER:
				usersGetter = getUsersFromProjectRole(m.store, api.ProjectDeveloper, e.Project.UID)
			default:
				return nil, errors.Errorf("invalid group value")
			}
		case *storepb.ApprovalNode_Role:
			role := api.Role(strings.TrimPrefix(val.Role, "roles/"))
			usersGetter = getUsersFromProjectRole(m.store, role, e.Project.UID)
		case *storepb.ApprovalNode_ExternalNodeId:
			usersGetter = func(_ context.Context) ([]*store.UserMessage, error) {
				return nil, nil
			}
		default:
			return nil, errors.Errorf("invalid node payload type")
		}

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
	}

	var mentionEndUsers []*store.UserMessage
	for _, u := range mentionUsers {
		if u.Type == api.EndUser {
			mentionEndUsers = append(mentionEndUsers, u)
		}
	}

	webhookCtx = webhook.Context{
		Level:        level,
		ActivityType: string(activityType),
		Title:        title,
		TitleZh:      titleZh,
		Issue: &webhook.Issue{
			ID:          e.Issue.UID,
			Name:        e.Issue.Title,
			Status:      e.Issue.Status,
			Type:        e.Issue.Type,
			Description: e.Issue.Description,
		},
		Project: &webhook.Project{
			ID:   e.Project.UID,
			Name: e.Project.Title,
		},
		Stage:               nil,
		TaskResult:          nil,
		Description:         e.Comment,
		Link:                link,
		CreatorID:           e.Actor.ID,
		CreatorName:         e.Actor.Name,
		CreatorEmail:        e.Actor.Email,
		MentionEndUsers:     mentionEndUsers,
		MentionUsersByPhone: mentions,
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
		webhookCtx.CreatedTs = time.Now().Unix()
		webhookCtx.DirectMessage = hook.Payload.GetDirectMessage()
		go func(webhookCtx *webhook.Context, hook *store.ProjectWebhookMessage) {
			if err := common.Retry(ctx, func() error {
				return webhook.Post(hook.Type, *webhookCtx)
			}); err != nil {
				// The external webhook endpoint might be invalid which is out of our code control, so we just emit a warning
				slog.Warn("Failed to post webhook event on activity",
					slog.String("webhook type", hook.Type),
					slog.String("webhook name", hook.Title),
					slog.String("activity type", webhookCtx.ActivityType),
					slog.String("title", webhookCtx.Title),
					log.BBError(err))
				return
			}
		}(&webhookCtx, hook)
	}
}

func (m *Manager) getUsersFromWorkspaceRole(role api.Role) func(context.Context) ([]*store.UserMessage, error) {
	return func(ctx context.Context) ([]*store.UserMessage, error) {
		policyMessage, err := m.store.GetWorkspaceIamPolicy(ctx)
		if err != nil {
			return nil, err
		}
		return utils.GetUsersByRoleInIAMPolicy(ctx, m.store, role, policyMessage.Policy), nil
	}
}

func getUsersFromProjectRole(s *store.Store, role api.Role, projectUID int) func(context.Context) ([]*store.UserMessage, error) {
	return func(ctx context.Context) ([]*store.UserMessage, error) {
		policyMessage, err := s.GetProjectIamPolicy(ctx, projectUID)
		if err != nil {
			return nil, err
		}

		return utils.GetUsersByRoleInIAMPolicy(ctx, s, role, policyMessage.Policy), nil
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
func ChangeIssueStatus(ctx context.Context, stores *store.Store, webhookManager *Manager, issue *store.IssueMessage, newStatus api.IssueStatus, updater *store.UserMessage, comment string) error {
	updateIssueMessage := &store.UpdateIssueMessage{Status: &newStatus}
	updatedIssue, err := stores.UpdateIssueV2(ctx, issue.UID, updateIssueMessage, updater.ID)
	if err != nil {
		return errors.Wrapf(err, "failed to update issue %q's status", issue.Title)
	}

	webhookManager.CreateEvent(ctx, &Event{
		Actor:   updater,
		Type:    EventTypeIssueStatusUpdate,
		Comment: comment,
		Issue:   NewIssue(updatedIssue),
		Project: NewProject(updatedIssue.Project),
	})
	return nil
}
