// Package activity is a component for managing activities.
package activity

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/webhook"
	"github.com/bytebase/bytebase/server/component/config"
	"github.com/bytebase/bytebase/store"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// Manager is the activity manager.
type Manager struct {
	store   *store.Store
	profile config.Profile
}

// Metadata is the activity metadata.
type Metadata struct {
	Issue *api.Issue
}

// NewManager creates an activity manager.
func NewManager(store *store.Store, profile config.Profile) *Manager {
	return &Manager{
		store:   store,
		profile: profile,
	}
}

// BatchCreateTaskStatusUpdateApprovalActivity creates a batch task status update activities for task approvals.
func (m *Manager) BatchCreateTaskStatusUpdateApprovalActivity(ctx context.Context, taskStatusPatch *api.TaskStatusPatch, issue *api.Issue, stage *api.Stage, taskList []*api.Task) error {
	var createList []*api.ActivityCreate
	for _, task := range taskList {
		payload, err := json.Marshal(api.ActivityPipelineTaskStatusUpdatePayload{
			TaskID:    task.ID,
			OldStatus: task.Status,
			NewStatus: taskStatusPatch.Status,
			IssueName: issue.Name,
			TaskName:  task.Name,
		})
		if err != nil {
			return errors.Wrapf(err, "failed to marshal activity after changing the task status: %v", task.Name)
		}

		activityCreate := &api.ActivityCreate{
			CreatorID:   taskStatusPatch.UpdaterID,
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
	webhookList, err := m.store.FindProjectWebhook(ctx, &api.ProjectWebhookFind{
		ProjectID:    &issue.ProjectID,
		ActivityType: &activityType,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to find project webhook after changing the issue status: %v", issue.Name)
	}
	if len(webhookList) == 0 {
		return nil
	}
	// Send one webhook post for all activities.
	webhookCtx := webhook.Context{
		Level:        webhook.WebhookInfo,
		ActivityType: string(activityType),
		Title:        fmt.Sprintf("Stage tasks approved - %s", stage.Name),
		Issue: &webhook.Issue{
			ID:          issue.ID,
			Name:        issue.Name,
			Status:      string(issue.Status),
			Type:        string(issue.Type),
			Description: issue.Description,
		},
		Project: &webhook.Project{
			ID:   issue.ProjectID,
			Name: issue.Project.Name,
		},
		Description:  anyActivity.Comment,
		Link:         fmt.Sprintf("%s/issue/%s", m.profile.ExternalURL, api.IssueSlug(issue)),
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
		return nil, errors.Wrapf(err, "failed to post webhook event after changing the issue task status: %s", meta.Issue.Name)
	}
	if postInbox {
		if err := m.postInboxIssueActivity(ctx, meta.Issue, activity.ID); err != nil {
			return nil, err
		}
	}

	hookFind := &api.ProjectWebhookFind{
		ProjectID:    &meta.Issue.ProjectID,
		ActivityType: &create.Type,
	}
	webhookList, err := m.store.FindProjectWebhook(ctx, hookFind)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find project webhook after changing the issue status: %v", meta.Issue.Name)
	}
	if len(webhookList) == 0 {
		return activity, nil
	}

	updater, err := m.store.GetPrincipalByID(ctx, create.CreatorID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find updater for posting webhook event after changing the issue status: %v", meta.Issue.Name)
	}
	if updater == nil {
		return nil, errors.Errorf("updater principal not found for ID %v", create.CreatorID)
	}

	webhookCtx, err := m.getWebhookContext(ctx, activity, meta, updater)
	if err != nil {
		log.Warn("Failed to get webhook context",
			zap.String("issue_name", meta.Issue.Name),
			zap.Error(err))
		return activity, nil
	}
	// Call external webhook endpoint in Go routine to avoid blocking web serving thread.
	go postWebhookList(webhookCtx, webhookList)

	return activity, nil
}

func postWebhookList(webhookCtx webhook.Context, webhookList []*api.ProjectWebhook) {
	for _, hook := range webhookList {
		webhookCtx.URL = hook.URL
		webhookCtx.CreatedTs = time.Now().Unix()
		const maxRetries = 3
		for retries := 0; retries < maxRetries; retries++ {
			if err := webhook.Post(hook.Type, webhookCtx); err != nil {
				// The external webhook endpoint might be invalid which is out of our code control, so we just emit a warning
				log.Warn("Failed to post webhook event on activity",
					zap.String("webhook type", hook.Type),
					zap.String("webhook name", hook.Name),
					zap.String("activity type", webhookCtx.ActivityType),
					zap.String("title", webhookCtx.Title),
					zap.Error(err))
			} else {
				break
			}
		}
	}
}

func (m *Manager) getWebhookContext(ctx context.Context, activity *api.Activity, meta *Metadata, updater *api.Principal) (webhook.Context, error) {
	var webhookCtx webhook.Context
	var webhookTaskResult *webhook.TaskResult
	level := webhook.WebhookInfo
	title := ""
	link := fmt.Sprintf("%s/issue/%s", m.profile.ExternalURL, api.IssueSlug(meta.Issue))
	switch activity.Type {
	case api.ActivityIssueCreate:
		title = fmt.Sprintf("Issue created - %s", meta.Issue.Name)
	case api.ActivityIssueStatusUpdate:
		switch meta.Issue.Status {
		case "OPEN":
			title = fmt.Sprintf("Issue reopened - %s", meta.Issue.Name)
		case "DONE":
			level = webhook.WebhookSuccess
			title = fmt.Sprintf("Issue resolved - %s", meta.Issue.Name)
		case "CANCELED":
			title = fmt.Sprintf("Issue canceled - %s", meta.Issue.Name)
		}
	case api.ActivityIssueCommentCreate:
		title = fmt.Sprintf("Comment created - %s", meta.Issue.Name)
		link += fmt.Sprintf("#activity%d", activity.ID)
	case api.ActivityIssueFieldUpdate:
		update := new(api.ActivityIssueFieldUpdatePayload)
		if err := json.Unmarshal([]byte(activity.Payload), update); err != nil {
			log.Warn("Failed to post webhook event after changing the issue field, failed to unmarshal payload",
				zap.String("issue_name", meta.Issue.Name),
				zap.Error(err))
			return webhookCtx, err
		}
		switch update.FieldID {
		case api.IssueFieldAssignee:
			{
				var oldAssignee, newAssignee *api.Principal
				if update.OldValue != "" {
					oldID, err := strconv.Atoi(update.OldValue)
					if err != nil {
						log.Warn("Failed to post webhook event after changing the issue assignee, old assignee id is not number",
							zap.String("issue_name", meta.Issue.Name),
							zap.String("old_assignee_id", update.OldValue),
							zap.Error(err))
						return webhookCtx, err
					}
					oldAssignee, err = m.store.GetPrincipalByID(ctx, oldID)
					if err != nil {
						log.Warn("Failed to post webhook event after changing the issue assignee, failed to find old assignee",
							zap.String("issue_name", meta.Issue.Name),
							zap.String("old_assignee_id", update.OldValue),
							zap.Error(err))
						return webhookCtx, err
					}
					if oldAssignee == nil {
						err := errors.Errorf("failed to post webhook event after changing the issue assignee, old assignee not found for ID %v", oldID)
						log.Warn(err.Error(),
							zap.String("issue_name", meta.Issue.Name),
							zap.String("old_assignee_id", update.OldValue),
							zap.Error(err))
						return webhookCtx, err
					}
				}

				if update.NewValue != "" {
					newID, err := strconv.Atoi(update.NewValue)
					if err != nil {
						log.Warn("Failed to post webhook event after changing the issue assignee, new assignee id is not number",
							zap.String("issue_name", meta.Issue.Name),
							zap.String("new_assignee_id", update.NewValue),
							zap.Error(err))
						return webhookCtx, err
					}
					newAssignee, err = m.store.GetPrincipalByID(ctx, newID)
					if err != nil {
						log.Warn("Failed to post webhook event after changing the issue assignee, failed to find new assignee",
							zap.String("issue_name", meta.Issue.Name),
							zap.String("new_assignee_id", update.NewValue),
							zap.Error(err))
						return webhookCtx, err
					}
					if newAssignee == nil {
						err := errors.Errorf("failed to post webhook event after changing the issue assignee, new assignee not found for ID %v", newID)
						log.Warn(err.Error(),
							zap.String("issue_name", meta.Issue.Name),
							zap.String("new_assignee_id", update.NewValue),
							zap.Error(err))
						return webhookCtx, err
					}

					if oldAssignee != nil && newAssignee != nil {
						title = fmt.Sprintf("Reassigned issue from %s to %s - %s", oldAssignee.Name, newAssignee.Name, meta.Issue.Name)
					} else if newAssignee != nil {
						title = fmt.Sprintf("Assigned issue to %s - %s", newAssignee.Name, meta.Issue.Name)
					} else if oldAssignee != nil {
						title = fmt.Sprintf("Unassigned issue from %s - %s", newAssignee.Name, meta.Issue.Name)
					}
				}
			}
		case api.IssueFieldDescription:
			title = fmt.Sprintf("Changed issue description - %s", meta.Issue.Name)
		case api.IssueFieldName:
			title = fmt.Sprintf("Changed issue name - %s", meta.Issue.Name)
		default:
			title = fmt.Sprintf("Updated issue - %s", meta.Issue.Name)
		}
	case api.ActivityPipelineTaskStatusUpdate:
		update := &api.ActivityPipelineTaskStatusUpdatePayload{}
		if err := json.Unmarshal([]byte(activity.Payload), update); err != nil {
			log.Warn("Failed to post webhook event after changing the issue task status, failed to unmarshal payload",
				zap.String("issue_name", meta.Issue.Name),
				zap.Error(err))
			return webhookCtx, err
		}

		task, err := m.store.GetTaskByID(ctx, update.TaskID)
		if err != nil {
			log.Warn("Failed to post webhook event after changing the issue task status, failed to find task",
				zap.String("issue_name", meta.Issue.Name),
				zap.Int("task_id", update.TaskID),
				zap.Error(err))
			return webhookCtx, err
		}
		if task == nil {
			err := errors.Errorf("failed to post webhook event after changing the issue task status, task not found for ID %v", update.TaskID)
			log.Warn(err.Error(),
				zap.String("issue_name", meta.Issue.Name),
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
				err := errors.Wrapf(err, "failed to unmarshal TaskRun Result")
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
			ID:          meta.Issue.ID,
			Name:        meta.Issue.Name,
			Status:      string(meta.Issue.Status),
			Type:        string(meta.Issue.Type),
			Description: meta.Issue.Description,
		},
		Project: &webhook.Project{
			ID:   meta.Issue.ProjectID,
			Name: meta.Issue.Project.Name,
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

func (m *Manager) postInboxIssueActivity(ctx context.Context, issue *api.Issue, activityID int) error {
	if issue.CreatorID != api.SystemBotID {
		inboxCreate := &api.InboxCreate{
			ReceiverID: issue.CreatorID,
			ActivityID: activityID,
		}
		if _, err := m.store.CreateInbox(ctx, inboxCreate); err != nil {
			return errors.Wrapf(err, "failed to post activity to creator inbox: %d", issue.CreatorID)
		}
	}

	if issue.AssigneeID != api.SystemBotID && issue.AssigneeID != issue.CreatorID {
		inboxCreate := &api.InboxCreate{
			ReceiverID: issue.AssigneeID,
			ActivityID: activityID,
		}
		if _, err := m.store.CreateInbox(ctx, inboxCreate); err != nil {
			return errors.Wrapf(err, "failed to post activity to assignee inbox: %d", issue.AssigneeID)
		}
	}

	for _, subscriber := range issue.SubscriberList {
		if subscriber.ID != api.SystemBotID && subscriber.ID != issue.CreatorID && subscriber.ID != issue.AssigneeID {
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
