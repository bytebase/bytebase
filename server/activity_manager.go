package server

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
	"github.com/bytebase/bytebase/store"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// ActivityManager is the activity manager.
type ActivityManager struct {
	s     *Server
	store *store.Store
}

// ActivityMeta is the activity metadata.
type ActivityMeta struct {
	issue *api.Issue
}

// NewActivityManager creates an activity manager.
func NewActivityManager(server *Server, store *store.Store) *ActivityManager {
	return &ActivityManager{
		s:     server,
		store: store,
	}
}

// BatchCreateTaskStatusUpdateApprovalActivity creates a batch task status update activities for task approvals.
func (m *ActivityManager) BatchCreateTaskStatusUpdateApprovalActivity(ctx context.Context, taskStatusPatch *api.TaskStatusPatch, issue *api.Issue, stage *api.Stage, taskList []*api.Task) error {
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
	webhookList, err := m.s.store.FindProjectWebhook(ctx, &api.ProjectWebhookFind{
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
		Link:         fmt.Sprintf("%s/issue/%s", m.s.profile.ExternalURL, api.IssueSlug(issue)),
		CreatorID:    anyActivity.CreatorID,
		CreatorName:  anyActivity.Creator.Name,
		CreatorEmail: anyActivity.Creator.Email,
	}
	// Call external webhook endpoint in Go routine to avoid blocking web serving thread.
	go postWebhookList(webhookCtx, webhookList, issue)

	return nil
}

// CreateActivity creates an activity.
func (m *ActivityManager) CreateActivity(ctx context.Context, create *api.ActivityCreate, meta *ActivityMeta) (*api.Activity, error) {
	activity, err := m.store.CreateActivity(ctx, create)
	if err != nil {
		return nil, err
	}

	if meta.issue == nil {
		return activity, nil
	}
	postInbox, err := shouldPostInbox(activity, create.Type)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to post webhook event after changing the issue task status: %s", meta.issue.Name)
	}
	if postInbox {
		if err := m.s.postInboxIssueActivity(ctx, meta.issue, activity.ID); err != nil {
			return nil, err
		}
	}

	hookFind := &api.ProjectWebhookFind{
		ProjectID:    &meta.issue.ProjectID,
		ActivityType: &create.Type,
	}
	webhookList, err := m.s.store.FindProjectWebhook(ctx, hookFind)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find project webhook after changing the issue status: %v", meta.issue.Name)
	}
	if len(webhookList) == 0 {
		return activity, nil
	}

	updater, err := m.s.store.GetPrincipalByID(ctx, create.CreatorID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find updater for posting webhook event after changing the issue status: %v", meta.issue.Name)
	}
	if updater == nil {
		return nil, errors.Errorf("updater principal not found for ID %v", create.CreatorID)
	}

	webhookCtx, err := m.getWebhookContext(ctx, activity, meta, updater)
	if err != nil {
		log.Warn("Failed to get webhook context",
			zap.String("issue_name", meta.issue.Name),
			zap.Error(err))
		return activity, nil
	}
	// Call external webhook endpoint in Go routine to avoid blocking web serving thread.
	go postWebhookList(webhookCtx, webhookList, meta.issue)

	return activity, nil
}

func postWebhookList(webhookCtx webhook.Context, webhookList []*api.ProjectWebhook, issue *api.Issue) {
	for _, hook := range webhookList {
		webhookCtx.URL = hook.URL
		webhookCtx.CreatedTs = time.Now().Unix()
		if err := webhook.Post(hook.Type, webhookCtx); err != nil {
			// The external webhook endpoint might be invalid which is out of our code control, so we just emit a warning
			log.Warn("Failed to post webhook event after changing the issue status",
				zap.String("webhook_type", hook.Type),
				zap.String("webhook_name", hook.Name),
				zap.String("issue_name", issue.Name),
				zap.String("status", string(issue.Status)),
				zap.Error(err))
		}
	}
}

func (m *ActivityManager) getWebhookContext(ctx context.Context, activity *api.Activity, meta *ActivityMeta, updater *api.Principal) (webhook.Context, error) {
	var webhookCtx webhook.Context
	var webhookTask *webhook.Task
	level := webhook.WebhookInfo
	title := ""
	link := fmt.Sprintf("%s/issue/%s", m.s.profile.ExternalURL, api.IssueSlug(meta.issue))
	switch activity.Type {
	case api.ActivityIssueCreate:
		title = fmt.Sprintf("Issue created - %s", meta.issue.Name)
	case api.ActivityIssueStatusUpdate:
		switch meta.issue.Status {
		case "OPEN":
			title = fmt.Sprintf("Issue reopened - %s", meta.issue.Name)
		case "DONE":
			level = webhook.WebhookSuccess
			title = fmt.Sprintf("Issue resolved - %s", meta.issue.Name)
		case "CANCELED":
			title = fmt.Sprintf("Issue canceled - %s", meta.issue.Name)
		}
	case api.ActivityIssueCommentCreate:
		title = fmt.Sprintf("Comment created - %s", meta.issue.Name)
		link += fmt.Sprintf("#activity%d", activity.ID)
	case api.ActivityIssueFieldUpdate:
		update := new(api.ActivityIssueFieldUpdatePayload)
		if err := json.Unmarshal([]byte(activity.Payload), update); err != nil {
			log.Warn("Failed to post webhook event after changing the issue field, failed to unmarshal payload",
				zap.String("issue_name", meta.issue.Name),
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
							zap.String("issue_name", meta.issue.Name),
							zap.String("old_assignee_id", update.OldValue),
							zap.Error(err))
						return webhookCtx, err
					}
					oldAssignee, err = m.s.store.GetPrincipalByID(ctx, oldID)
					if err != nil {
						log.Warn("Failed to post webhook event after changing the issue assignee, failed to find old assignee",
							zap.String("issue_name", meta.issue.Name),
							zap.String("old_assignee_id", update.OldValue),
							zap.Error(err))
						return webhookCtx, err
					}
					if oldAssignee == nil {
						err := errors.Errorf("failed to post webhook event after changing the issue assignee, old assignee not found for ID %v", oldID)
						log.Warn(err.Error(),
							zap.String("issue_name", meta.issue.Name),
							zap.String("old_assignee_id", update.OldValue),
							zap.Error(err))
						return webhookCtx, err
					}
				}

				if update.NewValue != "" {
					newID, err := strconv.Atoi(update.NewValue)
					if err != nil {
						log.Warn("Failed to post webhook event after changing the issue assignee, new assignee id is not number",
							zap.String("issue_name", meta.issue.Name),
							zap.String("new_assignee_id", update.NewValue),
							zap.Error(err))
						return webhookCtx, err
					}
					newAssignee, err = m.s.store.GetPrincipalByID(ctx, newID)
					if err != nil {
						log.Warn("Failed to post webhook event after changing the issue assignee, failed to find new assignee",
							zap.String("issue_name", meta.issue.Name),
							zap.String("new_assignee_id", update.NewValue),
							zap.Error(err))
						return webhookCtx, err
					}
					if newAssignee == nil {
						err := errors.Errorf("failed to post webhook event after changing the issue assignee, new assignee not found for ID %v", newID)
						log.Warn(err.Error(),
							zap.String("issue_name", meta.issue.Name),
							zap.String("new_assignee_id", update.NewValue),
							zap.Error(err))
						return webhookCtx, err
					}

					if oldAssignee != nil && newAssignee != nil {
						title = fmt.Sprintf("Reassigned issue from %s to %s - %s", oldAssignee.Name, newAssignee.Name, meta.issue.Name)
					} else if newAssignee != nil {
						title = fmt.Sprintf("Assigned issue to %s - %s", newAssignee.Name, meta.issue.Name)
					} else if oldAssignee != nil {
						title = fmt.Sprintf("Unassigned issue from %s - %s", newAssignee.Name, meta.issue.Name)
					}
				}
			}
		case api.IssueFieldDescription:
			title = fmt.Sprintf("Changed issue description - %s", meta.issue.Name)
		case api.IssueFieldName:
			title = fmt.Sprintf("Changed issue name - %s", meta.issue.Name)
		default:
			title = fmt.Sprintf("Updated issue - %s", meta.issue.Name)
		}
	case api.ActivityPipelineTaskStatusUpdate:
		update := &api.ActivityPipelineTaskStatusUpdatePayload{}
		if err := json.Unmarshal([]byte(activity.Payload), update); err != nil {
			log.Warn("Failed to post webhook event after changing the issue task status, failed to unmarshal payload",
				zap.String("issue_name", meta.issue.Name),
				zap.Error(err))
			return webhookCtx, err
		}

		task, err := m.s.store.GetTaskByID(ctx, update.TaskID)
		if err != nil {
			log.Warn("Failed to post webhook event after changing the issue task status, failed to find task",
				zap.String("issue_name", meta.issue.Name),
				zap.Int("task_id", update.TaskID),
				zap.Error(err))
			return webhookCtx, err
		}
		if task == nil {
			err := errors.Errorf("failed to post webhook event after changing the issue task status, task not found for ID %v", update.TaskID)
			log.Warn(err.Error(),
				zap.String("issue_name", meta.issue.Name),
				zap.Int("task_id", update.TaskID),
				zap.Error(err))
			return webhookCtx, err
		}

		webhookTask = &webhook.Task{
			ID:     task.ID,
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
				return task.TaskRunList[i].UpdatedTs > task.TaskRunList[j].UpdatedTs
			})

			var result api.TaskRunResultPayload
			if err := json.Unmarshal([]byte(task.TaskRunList[0].Result), &result); err != nil {
				err := errors.Wrapf(err, "failed to unmarshal TaskRun Result")
				log.Warn(err.Error(),
					zap.Any("TaskRun", task.TaskRunList[0]),
					zap.Error(err))
				return webhookCtx, err
			}
			webhookTask.Description = result.Detail
		}
	}

	webhookCtx = webhook.Context{
		Level:        level,
		ActivityType: string(activity.Type),
		Title:        title,
		Issue: &webhook.Issue{
			ID:          meta.issue.ID,
			Name:        meta.issue.Name,
			Status:      string(meta.issue.Status),
			Type:        string(meta.issue.Type),
			Description: meta.issue.Description,
		},
		Project: &webhook.Project{
			ID:   meta.issue.ProjectID,
			Name: meta.issue.Project.Name,
		},
		Task:         webhookTask,
		Description:  activity.Comment,
		Link:         link,
		CreatorID:    updater.ID,
		CreatorName:  updater.Name,
		CreatorEmail: updater.Email,
	}
	return webhookCtx, nil
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
