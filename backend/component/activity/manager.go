// Package activity is a component for managing activities.
package activity

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/gosimple/slug"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/webhook"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// Manager is the activity manager.
type Manager struct {
	store *store.Store
}

// Metadata is the activity metadata.
type Metadata struct {
	Issue *store.IssueMessage
}

// NewManager creates an activity manager.
func NewManager(store *store.Store) *Manager {
	return &Manager{
		store: store,
	}
}

// BatchCreateTaskStatusUpdateApprovalActivity creates a batch task status update activities for task approvals.
func (m *Manager) BatchCreateTaskStatusUpdateApprovalActivity(ctx context.Context, taskList []*store.TaskMessage, updaterID int, issue *store.IssueMessage, stageName string) error {
	var createList []*api.ActivityCreate
	for _, task := range taskList {
		payload, err := json.Marshal(api.ActivityPipelineTaskStatusUpdatePayload{
			TaskID:    task.ID,
			OldStatus: api.TaskPendingApproval,
			NewStatus: api.TaskPending,
			IssueName: issue.Title,
			TaskName:  task.Name,
		})
		if err != nil {
			return errors.Wrapf(err, "failed to marshal activity after changing the task status: %v", task.Name)
		}

		activityCreate := &api.ActivityCreate{
			CreatorID:   updaterID,
			ContainerID: task.PipelineID,
			Type:        api.ActivityPipelineTaskStatusUpdate,
			Level:       api.ActivityInfo,
			Payload:     string(payload),
		}
		createList = append(createList, activityCreate)
	}

	activityList, err := m.store.BatchCreateActivity(ctx, createList)
	if err != nil {
		return err
	}
	if len(activityList) == 0 {
		return errors.Errorf("failed to create any activity")
	}
	anyActivity := activityList[0]

	activityType := api.ActivityPipelineTaskStatusUpdate
	webhookList, err := m.store.FindProjectWebhookV2(ctx, &store.FindProjectWebhookMessage{
		ProjectID:    &issue.Project.UID,
		ActivityType: &activityType,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to find project webhook after changing the issue status: %v", issue.Title)
	}
	if len(webhookList) == 0 {
		return nil
	}

	setting, err := m.store.GetWorkspaceGeneralSetting(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to get workspace setting")
	}

	// Send one webhook post for all activities.
	webhookCtx := webhook.Context{
		Level:        webhook.WebhookInfo,
		ActivityType: string(activityType),
		Title:        fmt.Sprintf("Stage tasks approved - %s", stageName),
		Issue: &webhook.Issue{
			ID:          issue.UID,
			Name:        issue.Title,
			Status:      string(issue.Status),
			Type:        string(issue.Type),
			Description: issue.Description,
		},
		Project: &webhook.Project{
			ID:   issue.Project.UID,
			Name: issue.Project.Title,
		},
		Description:  anyActivity.Comment,
		Link:         fmt.Sprintf("%s/issue/%s-%d", setting.ExternalUrl, slug.Make(issue.Title), issue.UID),
		CreatorID:    anyActivity.CreatorID,
		CreatorName:  anyActivity.Creator.Name,
		CreatorEmail: anyActivity.Creator.Email,
	}
	// Call external webhook endpoint in Go routine to avoid blocking web serving thread.
	go postWebhookList(webhookCtx, webhookList)

	return nil
}

// CreateActivity creates an activity.
func (m *Manager) CreateActivity(ctx context.Context, create *api.ActivityCreate, meta *Metadata) (*api.Activity, error) {
	activity, err := m.store.CreateActivity(ctx, create)
	if err != nil {
		return nil, err
	}

	if meta.Issue == nil {
		return activity, nil
	}
	postInbox, err := shouldPostInbox(activity, create.Type)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to post webhook event after changing the issue task status: %s", meta.Issue.Title)
	}
	if postInbox {
		if err := m.postInboxIssueActivity(ctx, meta.Issue, activity.ID); err != nil {
			return nil, err
		}
	}

	webhookList, err := m.store.FindProjectWebhookV2(ctx, &store.FindProjectWebhookMessage{
		ProjectID:    &meta.Issue.Project.UID,
		ActivityType: &create.Type,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find project webhook after changing the issue status: %v", meta.Issue.Title)
	}
	if len(webhookList) == 0 {
		return activity, nil
	}

	updater, err := m.store.GetUserByID(ctx, create.CreatorID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find updater for posting webhook event after changing the issue status: %v", meta.Issue.Title)
	}
	if updater == nil {
		return nil, errors.Errorf("updater user not found for ID %v", create.CreatorID)
	}

	webhookCtx, err := m.getWebhookContext(ctx, activity, meta, updater)
	if err != nil {
		log.Warn("Failed to get webhook context",
			zap.String("issue_name", meta.Issue.Title),
			zap.Error(err))
		return activity, nil
	}
	// Call external webhook endpoint in Go routine to avoid blocking web serving thread.
	go postWebhookList(webhookCtx, webhookList)

	return activity, nil
}

func postWebhookList(webhookCtx webhook.Context, webhookList []*store.ProjectWebhookMessage) {
	for _, hook := range webhookList {
		webhookCtx.URL = hook.URL
		webhookCtx.CreatedTs = time.Now().Unix()
		if err := common.Retry(func() error {
			return webhook.Post(hook.Type, webhookCtx)
		}); err != nil {
			// The external webhook endpoint might be invalid which is out of our code control, so we just emit a warning
			log.Warn("Failed to post webhook event on activity",
				zap.String("webhook type", hook.Type),
				zap.String("webhook name", hook.Title),
				zap.String("activity type", webhookCtx.ActivityType),
				zap.String("title", webhookCtx.Title),
				zap.Error(err))
			return
		}
	}
}

func (m *Manager) getWebhookContext(ctx context.Context, activity *api.Activity, meta *Metadata, updater *store.UserMessage) (webhook.Context, error) {
	var webhookCtx webhook.Context
	var webhookTaskResult *webhook.TaskResult

	setting, err := m.store.GetWorkspaceGeneralSetting(ctx)
	if err != nil {
		return webhookCtx, errors.Wrapf(err, "failed to get workspace setting")
	}

	level := webhook.WebhookInfo
	title := ""
	link := fmt.Sprintf("%s/issue/%s-%d", setting.ExternalUrl, slug.Make(meta.Issue.Title), meta.Issue.UID)
	switch activity.Type {
	case api.ActivityIssueCreate:
		title = fmt.Sprintf("Issue created - %s", meta.Issue.Title)
	case api.ActivityIssueStatusUpdate:
		switch meta.Issue.Status {
		case "OPEN":
			title = fmt.Sprintf("Issue reopened - %s", meta.Issue.Title)
		case "DONE":
			level = webhook.WebhookSuccess
			title = fmt.Sprintf("Issue resolved - %s", meta.Issue.Title)
		case "CANCELED":
			title = fmt.Sprintf("Issue canceled - %s", meta.Issue.Title)
		}
	case api.ActivityIssueCommentCreate:
		title = fmt.Sprintf("Comment created - %s", meta.Issue.Title)
		link += fmt.Sprintf("#activity%d", activity.ID)
	case api.ActivityIssueFieldUpdate:
		update := new(api.ActivityIssueFieldUpdatePayload)
		if err := json.Unmarshal([]byte(activity.Payload), update); err != nil {
			log.Warn("Failed to post webhook event after changing the issue field, failed to unmarshal payload",
				zap.String("issue_name", meta.Issue.Title),
				zap.Error(err))
			return webhookCtx, err
		}
		switch update.FieldID {
		case api.IssueFieldAssignee:
			{
				var oldAssignee, newAssignee *store.UserMessage
				if update.OldValue != "" {
					oldID, err := strconv.Atoi(update.OldValue)
					if err != nil {
						log.Warn("Failed to post webhook event after changing the issue assignee, old assignee id is not number",
							zap.String("issue_name", meta.Issue.Title),
							zap.String("old_assignee_id", update.OldValue),
							zap.Error(err))
						return webhookCtx, err
					}
					oldAssignee, err = m.store.GetUserByID(ctx, oldID)
					if err != nil {
						log.Warn("Failed to post webhook event after changing the issue assignee, failed to find old assignee",
							zap.String("issue_name", meta.Issue.Title),
							zap.String("old_assignee_id", update.OldValue),
							zap.Error(err))
						return webhookCtx, err
					}
					if oldAssignee == nil {
						err := errors.Errorf("failed to post webhook event after changing the issue assignee, old assignee not found for ID %v", oldID)
						log.Warn(err.Error(),
							zap.String("issue_name", meta.Issue.Title),
							zap.String("old_assignee_id", update.OldValue),
							zap.Error(err))
						return webhookCtx, err
					}
				}

				if update.NewValue != "" {
					newID, err := strconv.Atoi(update.NewValue)
					if err != nil {
						log.Warn("Failed to post webhook event after changing the issue assignee, new assignee id is not number",
							zap.String("issue_name", meta.Issue.Title),
							zap.String("new_assignee_id", update.NewValue),
							zap.Error(err))
						return webhookCtx, err
					}
					newAssignee, err = m.store.GetUserByID(ctx, newID)
					if err != nil {
						log.Warn("Failed to post webhook event after changing the issue assignee, failed to find new assignee",
							zap.String("issue_name", meta.Issue.Title),
							zap.String("new_assignee_id", update.NewValue),
							zap.Error(err))
						return webhookCtx, err
					}
					if newAssignee == nil {
						err := errors.Errorf("failed to post webhook event after changing the issue assignee, new assignee not found for ID %v", newID)
						log.Warn(err.Error(),
							zap.String("issue_name", meta.Issue.Title),
							zap.String("new_assignee_id", update.NewValue),
							zap.Error(err))
						return webhookCtx, err
					}

					if oldAssignee != nil && newAssignee != nil {
						title = fmt.Sprintf("Reassigned issue from %s to %s - %s", oldAssignee.Name, newAssignee.Name, meta.Issue.Title)
					} else if newAssignee != nil {
						title = fmt.Sprintf("Assigned issue to %s - %s", newAssignee.Name, meta.Issue.Title)
					} else if oldAssignee != nil {
						title = fmt.Sprintf("Unassigned issue from %s - %s", newAssignee.Name, meta.Issue.Title)
					}
				}
			}
		case api.IssueFieldDescription:
			title = fmt.Sprintf("Changed issue description - %s", meta.Issue.Title)
		case api.IssueFieldName:
			title = fmt.Sprintf("Changed issue name - %s", meta.Issue.Title)
		default:
			title = fmt.Sprintf("Updated issue - %s", meta.Issue.Title)
		}
	case api.ActivityPipelineStageStatusUpdate:
		payload := &api.ActivityPipelineStageStatusUpdatePayload{}
		if err := json.Unmarshal([]byte(activity.Payload), payload); err != nil {
			log.Warn(
				"failed to post webhook event after stage status updating, failed to unmarshal payload",
				zap.String("issue_name", meta.Issue.Title),
				zap.Error(err))
			return webhookCtx, err
		}
		link += fmt.Sprintf("?stage=%d", payload.StageID)
		switch payload.StageStatusUpdateType {
		case api.StageStatusUpdateTypeBegin:
			title = fmt.Sprintf("Stage begins - %s", payload.StageName)
		case api.StageStatusUpdateTypeEnd:
			title = fmt.Sprintf("Stage ends - %s", payload.StageName)
		}

	case api.ActivityPipelineTaskStatusUpdate:
		update := &api.ActivityPipelineTaskStatusUpdatePayload{}
		if err := json.Unmarshal([]byte(activity.Payload), update); err != nil {
			log.Warn("Failed to post webhook event after changing the issue task status, failed to unmarshal payload",
				zap.String("issue_name", meta.Issue.Title),
				zap.Error(err))
			return webhookCtx, err
		}

		task, err := m.store.GetTaskByID(ctx, update.TaskID)
		if err != nil {
			log.Warn("Failed to post webhook event after changing the issue task status, failed to find task",
				zap.String("issue_name", meta.Issue.Title),
				zap.Int("task_id", update.TaskID),
				zap.Error(err))
			return webhookCtx, err
		}
		if task == nil {
			err := errors.Errorf("failed to post webhook event after changing the issue task status, task not found for ID %v", update.TaskID)
			log.Warn(err.Error(),
				zap.String("issue_name", meta.Issue.Title),
				zap.Int("task_id", update.TaskID),
				zap.Error(err))
			return webhookCtx, err
		}

		webhookTaskResult = &webhook.TaskResult{
			Name:   task.Name,
			Status: string(task.Status),
		}

		title = "Task changed - " + task.Name
		switch update.NewStatus {
		case api.TaskPending:
			switch update.OldStatus {
			case api.TaskRunning:
				title = "Task canceled - " + task.Name
			case api.TaskPendingApproval:
				title = "Task approved - " + task.Name
			}
		case api.TaskRunning:
			title = "Task started - " + task.Name
		case api.TaskDone:
			level = webhook.WebhookSuccess
			title = "Task completed - " + task.Name

			skipped, skippedReason, err := utils.GetTaskSkippedAndReason(task)
			if err != nil {
				err := errors.Wrap(err, "failed to get skipped and skippedReason from the task")
				log.Warn(err.Error(), zap.String("task.Payload", task.Payload), zap.Error(err))
				return webhookCtx, err
			}
			if skipped {
				title = "Task skipped - " + task.Name
				webhookTaskResult.Status = "SKIPPED"
				webhookTaskResult.SkippedReason = skippedReason
			}
		case api.TaskFailed:
			level = webhook.WebhookError
			title = "Task failed - " + task.Name

			if len(task.TaskRunList) == 0 {
				err := errors.Errorf("expect at least 1 TaskRun, get 0")
				log.Warn(err.Error(),
					zap.Any("task", task),
					zap.Error(err))
				return webhookCtx, err
			}

			// sort TaskRunList to get the most recent task run result.
			sort.Slice(task.TaskRunList, func(i int, j int) bool {
				return task.TaskRunList[i].UpdatedTs > task.TaskRunList[j].UpdatedTs || (task.TaskRunList[i].UpdatedTs == task.TaskRunList[j].UpdatedTs && task.TaskRunList[i].ID > task.TaskRunList[j].ID)
			})

			var result api.TaskRunResultPayload
			if err := json.Unmarshal([]byte(task.TaskRunList[0].Result), &result); err != nil {
				err := errors.Wrap(err, "failed to unmarshal TaskRun Result")
				log.Warn(err.Error(),
					zap.Any("TaskRun", task.TaskRunList[0]),
					zap.Error(err))
				return webhookCtx, err
			}
			webhookTaskResult.Detail = result.Detail
		}
	}

	webhookCtx = webhook.Context{
		Level:        level,
		ActivityType: string(activity.Type),
		Title:        title,
		Issue: &webhook.Issue{
			ID:          meta.Issue.UID,
			Name:        meta.Issue.Title,
			Status:      string(meta.Issue.Status),
			Type:        string(meta.Issue.Type),
			Description: meta.Issue.Description,
		},
		Project: &webhook.Project{
			ID:   meta.Issue.Project.UID,
			Name: meta.Issue.Project.Title,
		},
		TaskResult:   webhookTaskResult,
		Description:  activity.Comment,
		Link:         link,
		CreatorID:    updater.ID,
		CreatorName:  updater.Name,
		CreatorEmail: updater.Email,
	}
	return webhookCtx, nil
}

func (m *Manager) postInboxIssueActivity(ctx context.Context, issue *store.IssueMessage, activityID int) error {
	if issue.Creator.ID != api.SystemBotID {
		inboxCreate := &api.InboxCreate{
			ReceiverID: issue.Creator.ID,
			ActivityID: activityID,
		}
		if _, err := m.store.CreateInbox(ctx, inboxCreate); err != nil {
			return errors.Wrapf(err, "failed to post activity to creator inbox: %d", issue.Creator.ID)
		}
	}

	if issue.Assignee.ID != api.SystemBotID && issue.Assignee.ID != issue.Creator.ID {
		inboxCreate := &api.InboxCreate{
			ReceiverID: issue.Assignee.ID,
			ActivityID: activityID,
		}
		if _, err := m.store.CreateInbox(ctx, inboxCreate); err != nil {
			return errors.Wrapf(err, "failed to post activity to assignee inbox: %d", issue.Assignee.ID)
		}
	}

	for _, subscriber := range issue.Subscribers {
		if subscriber.ID != api.SystemBotID && subscriber.ID != issue.Creator.ID && subscriber.ID != issue.Assignee.ID {
			inboxCreate := &api.InboxCreate{
				ReceiverID: subscriber.ID,
				ActivityID: activityID,
			}
			if _, err := m.store.CreateInbox(ctx, inboxCreate); err != nil {
				return errors.Wrapf(err, "failed to post activity to subscriber inbox: %d", subscriber.ID)
			}
		}
	}

	return nil
}

func shouldPostInbox(activity *api.Activity, createType api.ActivityType) (bool, error) {
	switch createType {
	case api.ActivityIssueCreate:
		return true, nil
	case api.ActivityIssueStatusUpdate:
		return true, nil
	case api.ActivityIssueCommentCreate:
		return true, nil
	case api.ActivityIssueFieldUpdate:
		return true, nil
	case api.ActivityPipelineTaskStatementUpdate:
		return true, nil
	case api.ActivityPipelineTaskEarliestAllowedTimeUpdate:
		return true, nil
	case api.ActivityPipelineStageStatusUpdate:
		return false, nil
	case api.ActivityPipelineTaskStatusUpdate:
		update := new(api.ActivityPipelineTaskStatusUpdatePayload)
		if err := json.Unmarshal([]byte(activity.Payload), update); err != nil {
			return false, err
		}
		// To reduce noise, for now we only post status update to inbox upon task failure.
		if update.NewStatus == api.TaskFailed {
			return true, nil
		}
	}
	return false, nil
}
