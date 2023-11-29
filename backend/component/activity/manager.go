// Package activity is a component for managing activities.
package activity

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gosimple/slug"
	"github.com/nyaruka/phonenumbers"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/webhook"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"

	"github.com/pkg/errors"
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

// BatchCreateActivitiesForCreateIssue creates activities for running tasks.
func (m *Manager) BatchCreateActivitiesForRunTasks(ctx context.Context, tasks []*store.TaskMessage, issue *store.IssueMessage, comment string, updaterUID int) error {
	var creates []*store.ActivityMessage
	for _, task := range tasks {
		payload, err := json.Marshal(api.ActivityPipelineTaskRunStatusUpdatePayload{
			TaskID:    task.ID,
			NewStatus: api.TaskRunPending,
			IssueName: issue.Title,
			TaskName:  task.Name,
		})
		if err != nil {
			return errors.Wrapf(err, "failed to marshal activity after changing the task status: %v", task.Name)
		}

		activityCreate := &store.ActivityMessage{
			CreatorUID:   updaterUID,
			ContainerUID: task.PipelineID,
			Type:         api.ActivityPipelineTaskRunStatusUpdate,
			Level:        api.ActivityInfo,
			Comment:      comment,
			Payload:      string(payload),
		}
		creates = append(creates, activityCreate)
	}

	activityList, err := m.store.BatchCreateActivityV2(ctx, creates)
	if err != nil {
		return err
	}
	if len(activityList) == 0 {
		return errors.Errorf("failed to create any activity")
	}
	anyActivity := activityList[0]

	activityType := api.ActivityPipelineTaskRunStatusUpdate
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

	user, err := m.store.GetUserByID(ctx, anyActivity.CreatorUID)
	if err != nil {
		return errors.Wrapf(err, "failed to get principal %d", anyActivity.CreatorUID)
	}

	// Send one webhook post for all activities.
	webhookCtx := webhook.Context{
		Level:        webhook.WebhookInfo,
		ActivityType: string(activityType),
		Title:        fmt.Sprintf("Issue task runs start - %s", issue.Title),
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
		CreatorID:    anyActivity.CreatorUID,
		CreatorName:  user.Name,
		CreatorEmail: user.Email,
	}
	// Call external webhook endpoint in Go routine to avoid blocking web serving thread.
	go postWebhookList(ctx, &webhookCtx, webhookList)

	return nil
}

// BatchCreateActivitiesForSkipTasks creates activities for skipping tasks.
func (m *Manager) BatchCreateActivitiesForSkipTasks(ctx context.Context, tasks []*store.TaskMessage, issue *store.IssueMessage, comment string, updaterID int) error {
	var creates []*store.ActivityMessage
	for _, task := range tasks {
		payload, err := json.Marshal(api.ActivityPipelineTaskRunStatusUpdatePayload{
			TaskID:    task.ID,
			NewStatus: api.TaskRunSkipped,
			IssueName: issue.Title,
			TaskName:  task.Name,
		})
		if err != nil {
			return errors.Wrapf(err, "failed to marshal activity after changing the task status: %v", task.Name)
		}

		activityCreate := &store.ActivityMessage{
			CreatorUID:   updaterID,
			ContainerUID: task.PipelineID,
			Type:         api.ActivityPipelineTaskRunStatusUpdate,
			Level:        api.ActivityInfo,
			Comment:      comment,
			Payload:      string(payload),
		}
		creates = append(creates, activityCreate)
	}

	activityList, err := m.store.BatchCreateActivityV2(ctx, creates)
	if err != nil {
		return err
	}
	if len(activityList) == 0 {
		return errors.Errorf("failed to create any activity")
	}
	anyActivity := activityList[0]

	activityType := api.ActivityPipelineTaskRunStatusUpdate
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

	user, err := m.store.GetUserByID(ctx, anyActivity.CreatorUID)
	if err != nil {
		return errors.Wrapf(err, "failed to get principal %d", anyActivity.CreatorUID)
	}

	// Send one webhook post for all activities.
	webhookCtx := webhook.Context{
		Level:        webhook.WebhookInfo,
		ActivityType: string(activityType),
		Title:        fmt.Sprintf("Issue tasks skipped - %s", issue.Title),
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
		CreatorID:    anyActivity.CreatorUID,
		CreatorName:  user.Name,
		CreatorEmail: user.Email,
	}
	// Call external webhook endpoint in Go routine to avoid blocking web serving thread.
	go postWebhookList(ctx, &webhookCtx, webhookList)

	return nil
}

// BatchCreateActivitiesForCancelTaskRuns creates activities for cancelling task runs.
func (m *Manager) BatchCreateActivitiesForCancelTaskRuns(ctx context.Context, tasks []*store.TaskMessage, issue *store.IssueMessage, comment string, updaterUID int) error {
	var creates []*store.ActivityMessage
	for _, task := range tasks {
		payload, err := json.Marshal(api.ActivityPipelineTaskRunStatusUpdatePayload{
			TaskID:    task.ID,
			NewStatus: api.TaskRunCanceled,
			IssueName: issue.Title,
			TaskName:  task.Name,
		})
		if err != nil {
			return errors.Wrapf(err, "failed to marshal activity after changing the task status: %v", task.Name)
		}

		activityCreate := &store.ActivityMessage{
			CreatorUID:   updaterUID,
			ContainerUID: task.PipelineID,
			Type:         api.ActivityPipelineTaskRunStatusUpdate,
			Level:        api.ActivityInfo,
			Comment:      comment,
			Payload:      string(payload),
		}
		creates = append(creates, activityCreate)
	}

	activityList, err := m.store.BatchCreateActivityV2(ctx, creates)
	if err != nil {
		return err
	}
	if len(activityList) == 0 {
		return errors.Errorf("failed to create any activity")
	}
	anyActivity := activityList[0]

	activityType := api.ActivityPipelineTaskRunStatusUpdate
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

	user, err := m.store.GetUserByID(ctx, anyActivity.CreatorUID)
	if err != nil {
		return errors.Wrapf(err, "failed to get principal %d", anyActivity.CreatorUID)
	}

	// Send one webhook post for all activities.
	webhookCtx := webhook.Context{
		Level:        webhook.WebhookInfo,
		ActivityType: string(activityType),
		Title:        fmt.Sprintf("Issue task runs start - %s", issue.Title),
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
		CreatorID:    anyActivity.CreatorUID,
		CreatorName:  user.Name,
		CreatorEmail: user.Email,
	}
	// Call external webhook endpoint in Go routine to avoid blocking web serving thread.
	go postWebhookList(ctx, &webhookCtx, webhookList)

	return nil
}

// CreateActivity creates an activity.
func (m *Manager) CreateActivity(ctx context.Context, create *store.ActivityMessage, meta *Metadata) (*store.ActivityMessage, error) {
	activity, err := m.store.CreateActivityV2(ctx, create)
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
		if err := m.postInboxIssueActivity(ctx, meta.Issue, activity.UID); err != nil {
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

	updater, err := m.store.GetUserByID(ctx, create.CreatorUID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find updater for posting webhook event after changing the issue status: %v", meta.Issue.Title)
	}
	if updater == nil {
		return nil, errors.Errorf("updater user not found for ID %v", create.CreatorUID)
	}

	webhookCtx, err := m.getWebhookContext(ctx, activity, meta, updater)
	if err != nil {
		slog.Warn("Failed to get webhook context",
			slog.String("issue_name", meta.Issue.Title),
			log.BBError(err))
		return activity, nil
	}
	// Call external webhook endpoint in Go routine to avoid blocking web serving thread.
	go postWebhookList(ctx, webhookCtx, webhookList)

	return activity, nil
}

func postWebhookList(ctx context.Context, webhookCtx *webhook.Context, webhookList []*store.ProjectWebhookMessage) {
	for _, hook := range webhookList {
		webhookCtx := *webhookCtx
		webhookCtx.URL = hook.URL
		webhookCtx.CreatedTs = time.Now().Unix()
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

func (m *Manager) getWebhookContext(ctx context.Context, activity *store.ActivityMessage, meta *Metadata, updater *store.UserMessage) (*webhook.Context, error) {
	var webhookCtx webhook.Context
	var webhookTaskResult *webhook.TaskResult
	var mentions []string

	setting, err := m.store.GetWorkspaceGeneralSetting(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get workspace setting")
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
		link += fmt.Sprintf("#activity%d", activity.UID)
	case api.ActivityIssueFieldUpdate:
		update := new(api.ActivityIssueFieldUpdatePayload)
		if err := json.Unmarshal([]byte(activity.Payload), update); err != nil {
			slog.Warn("Failed to post webhook event after changing the issue field, failed to unmarshal payload",
				slog.String("issue_name", meta.Issue.Title),
				log.BBError(err))
			return nil, err
		}
		switch update.FieldID {
		case api.IssueFieldAssignee:
			{
				var oldAssignee, newAssignee *store.UserMessage
				if update.OldValue != "" {
					oldID, err := strconv.Atoi(update.OldValue)
					if err != nil {
						slog.Warn("Failed to post webhook event after changing the issue assignee, old assignee id is not number",
							slog.String("issue_name", meta.Issue.Title),
							slog.String("old_assignee_id", update.OldValue),
							log.BBError(err))
						return nil, err
					}
					oldAssignee, err = m.store.GetUserByID(ctx, oldID)
					if err != nil {
						slog.Warn("Failed to post webhook event after changing the issue assignee, failed to find old assignee",
							slog.String("issue_name", meta.Issue.Title),
							slog.String("old_assignee_id", update.OldValue),
							log.BBError(err))
						return nil, err
					}
					if oldAssignee == nil {
						err := errors.Errorf("failed to post webhook event after changing the issue assignee, old assignee not found for ID %v", oldID)
						slog.Warn(err.Error(),
							slog.String("issue_name", meta.Issue.Title),
							slog.String("old_assignee_id", update.OldValue),
							log.BBError(err))
						return nil, err
					}
				}

				if update.NewValue != "" {
					newID, err := strconv.Atoi(update.NewValue)
					if err != nil {
						slog.Warn("Failed to post webhook event after changing the issue assignee, new assignee id is not number",
							slog.String("issue_name", meta.Issue.Title),
							slog.String("new_assignee_id", update.NewValue),
							log.BBError(err))
						return nil, err
					}
					newAssignee, err = m.store.GetUserByID(ctx, newID)
					if err != nil {
						slog.Warn("Failed to post webhook event after changing the issue assignee, failed to find new assignee",
							slog.String("issue_name", meta.Issue.Title),
							slog.String("new_assignee_id", update.NewValue),
							log.BBError(err))
						return nil, err
					}

					if oldAssignee != nil && newAssignee != nil {
						title = fmt.Sprintf("Reassigned issue from %s to %s - %s", oldAssignee.Name, newAssignee.Name, meta.Issue.Title)
					} else if newAssignee != nil {
						title = fmt.Sprintf("Assigned issue to %s - %s", newAssignee.Name, meta.Issue.Title)
					} else if oldAssignee != nil {
						title = fmt.Sprintf("Unassigned issue from %s - %s", oldAssignee.Name, meta.Issue.Title)
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
			slog.Warn(
				"failed to post webhook event after stage status updating, failed to unmarshal payload",
				slog.String("issue_name", meta.Issue.Title),
				log.BBError(err))
			return nil, err
		}
		link += fmt.Sprintf("?stage=%d", payload.StageID)
		if payload.StageStatusUpdateType == api.StageStatusUpdateTypeEnd {
			title = fmt.Sprintf("Stage ends - %s", payload.StageName)
		}
	case api.ActivityPipelineTaskStatusUpdate:
		update := &api.ActivityPipelineTaskStatusUpdatePayload{}
		if err := json.Unmarshal([]byte(activity.Payload), update); err != nil {
			slog.Warn("Failed to post webhook event after changing the issue task status, failed to unmarshal payload",
				slog.String("issue_name", meta.Issue.Title),
				log.BBError(err))
			return nil, err
		}

		task, err := m.store.GetTaskV2ByID(ctx, update.TaskID)
		if err != nil {
			slog.Warn("Failed to post webhook event after changing the issue task status, failed to find task",
				slog.String("issue_name", meta.Issue.Title),
				slog.Int("task_id", update.TaskID),
				log.BBError(err))
			return nil, err
		}
		if task == nil {
			err := errors.Errorf("failed to post webhook event after changing the issue task status, task not found for ID %v", update.TaskID)
			slog.Warn(err.Error(),
				slog.String("issue_name", meta.Issue.Title),
				slog.Int("task_id", update.TaskID),
				log.BBError(err))
			return nil, err
		}

		webhookTaskResult = &webhook.TaskResult{
			Name:   task.Name,
			Status: string(task.LatestTaskRunStatus),
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

			skipped, skippedReason, err := getTaskSkippedAndReason(task)
			if err != nil {
				err := errors.Wrap(err, "failed to get skipped and skippedReason from the task")
				slog.Warn(err.Error(), slog.String("task.Payload", task.Payload), log.BBError(err))
				return nil, err
			}
			if skipped {
				title = "Task skipped - " + task.Name
				webhookTaskResult.Status = "SKIPPED"
				webhookTaskResult.SkippedReason = skippedReason
			}
		case api.TaskFailed:
			level = webhook.WebhookError
			title = "Task failed - " + task.Name

			taskRuns, err := m.store.ListTaskRunsV2(ctx, &store.FindTaskRunMessage{
				PipelineUID: &task.PipelineID,
				StageUID:    &task.StageID,
				TaskUID:     &task.ID,
			})
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to list task runs, error: %v", err)
			}

			if len(taskRuns) == 0 {
				err := errors.Errorf("expect at least 1 TaskRun, get 0")
				slog.Warn(err.Error(),
					slog.Any("task", task),
					log.BBError(err))
				return nil, err
			}

			// sort TaskRunList to get the most recent task run result.
			sort.Slice(taskRuns, func(i int, j int) bool {
				return taskRuns[i].ID > taskRuns[j].ID
			})

			var result api.TaskRunResultPayload
			if err := json.Unmarshal([]byte(taskRuns[0].Result), &result); err != nil {
				err := errors.Wrap(err, "failed to unmarshal TaskRun Result")
				slog.Warn(err.Error(),
					slog.Any("TaskRun", taskRuns[0]),
					log.BBError(err))
				return nil, err
			}
			webhookTaskResult.Detail = result.Detail
		}

	case api.ActivityPipelineTaskRunStatusUpdate:
		payload := &api.ActivityPipelineTaskRunStatusUpdatePayload{}
		if err := json.Unmarshal([]byte(activity.Payload), payload); err != nil {
			slog.Warn("Failed to post webhook event after changing the issue task run status, failed to unmarshal payload",
				slog.String("issue_name", meta.Issue.Title),
				log.BBError(err))
			return nil, err
		}

		task, err := m.store.GetTaskV2ByID(ctx, payload.TaskID)
		if err != nil {
			slog.Warn("Failed to post webhook event after changing the issue task status, failed to find task",
				slog.String("issue_name", meta.Issue.Title),
				slog.Int("task_id", payload.TaskID),
				log.BBError(err))
			return nil, err
		}
		if task == nil {
			err := errors.Errorf("failed to post webhook event after changing the issue task status, task not found for ID %v", payload.TaskID)
			slog.Warn(err.Error(),
				slog.String("issue_name", meta.Issue.Title),
				slog.Int("task_id", payload.TaskID),
				log.BBError(err))
			return nil, err
		}

		webhookTaskResult = &webhook.TaskResult{
			Name:   payload.TaskName,
			Status: string(payload.NewStatus),
		}

		switch payload.NewStatus {
		case api.TaskRunPending:
			title = "Task run started - " + payload.TaskName
		case api.TaskRunRunning:
			title = "Task run started to run - " + payload.TaskName
		case api.TaskRunDone:
			level = webhook.WebhookSuccess
			title = "Task run completed - " + payload.TaskName
		case api.TaskRunFailed:
			level = webhook.WebhookError
			title = "Task run failed - " + payload.TaskName

			taskRuns, err := m.store.ListTaskRunsV2(ctx, &store.FindTaskRunMessage{
				PipelineUID: &task.PipelineID,
				StageUID:    &task.StageID,
				TaskUID:     &task.ID,
			})
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to list task runs, error: %v", err)
			}
			if len(taskRuns) == 0 {
				err := errors.Errorf("expect at least 1 TaskRun, get 0")
				slog.Warn(err.Error(),
					slog.Any("task", task),
					log.BBError(err))
				return nil, err
			}

			// sort TaskRunList to get the most recent task run result.
			sort.Slice(taskRuns, func(i int, j int) bool {
				return taskRuns[i].ID > taskRuns[j].ID
			})

			var result api.TaskRunResultPayload
			if err := json.Unmarshal([]byte(taskRuns[0].Result), &result); err != nil {
				err := errors.Wrap(err, "failed to unmarshal TaskRun Result")
				slog.Warn(err.Error(),
					slog.Any("TaskRun", taskRuns[0]),
					log.BBError(err))
				return nil, err
			}
			webhookTaskResult.Detail = result.Detail

		case api.TaskRunCanceled:
			title = "Task run canceled - " + payload.TaskName
		case api.TaskRunSkipped:
			title = "Task skipped - " + payload.TaskName
			_, skippedReason, err := getTaskSkippedAndReason(task)
			if err != nil {
				err := errors.Wrap(err, "failed to get skipped and skippedReason from the task")
				slog.Warn(err.Error(), slog.String("task.Payload", task.Payload), log.BBError(err))
				return nil, err
			}
			webhookTaskResult.SkippedReason = skippedReason
		default:
			title = "Task run changed - " + payload.TaskName
		}

	case api.ActivityNotifyIssueApproved:
		title = "Issue approved - " + meta.Issue.Title

		phone, err := maybeGetPhoneFromUser(meta.Issue.Creator)
		if err != nil {
			slog.Warn("failed to parse phone number", slog.String("issue_title", meta.Issue.Title), log.BBError(err))
		} else if phone != "" {
			mentions = append(mentions, phone)
		}

	case api.ActivityNotifyPipelineRollout:
		payload := &api.ActivityNotifyPipelineRolloutPayload{}
		if err := json.Unmarshal([]byte(activity.Payload), payload); err != nil {
			slog.Warn("failed to unmarshal payload",
				slog.String("issue_name", meta.Issue.Title),
				log.BBError(err))
			return nil, err
		}
		title = fmt.Sprintf("Issue is waiting for rollout (%s) - %s", payload.StageName, meta.Issue.Title)
		var usersGetters []func(context.Context) ([]*store.UserMessage, error)
		if payload.RolloutPolicy.GetAutomatic() {
			usersGetters = append(usersGetters, getUsersFromUsers(meta.Issue.Creator))
		} else {
			for _, workspaceRole := range payload.RolloutPolicy.GetWorkspaceRoles() {
				role := api.Role(strings.TrimPrefix(workspaceRole, "roles/"))
				usersGetters = append(usersGetters, getUsersFromWorkspaceRole(m.store, role))
			}
			for _, projectRole := range payload.RolloutPolicy.GetProjectRoles() {
				role := api.Role(strings.TrimPrefix(projectRole, "roles/"))
				usersGetters = append(usersGetters, getUsersFromProjectRole(m.store, role, meta.Issue.Project.ResourceID))
			}
			for _, issueRole := range payload.RolloutPolicy.GetIssueRoles() {
				switch issueRole {
				case "roles/LAST_APPROVER":
					usersGetters = append(usersGetters, getUsersFromIssueLastApprover(m.store, meta.Issue.Payload.GetApproval()))
				case "roles/CREATOR":
					usersGetters = append(usersGetters, getUsersFromUsers(meta.Issue.Creator))
				}
			}
		}
		mentionedUser := map[int]bool{}
		for _, usersGetter := range usersGetters {
			users, err := usersGetter(ctx)
			if err != nil {
				slog.Warn("failed to get users",
					slog.String("issue_name", meta.Issue.Title),
					log.BBError(err))
				return nil, err
			}
			for _, user := range users {
				if mentionedUser[user.ID] {
					continue
				}
				mentionedUser[user.ID] = true
				phone, err := maybeGetPhoneFromUser(user)
				if err != nil {
					slog.Warn("failed to parse phone number",
						slog.String("issue_name", meta.Issue.Title),
						log.BBError(err))
				}
				if phone != "" {
					mentions = append(mentions, phone)
				}
			}
		}

	case api.ActivityIssueApprovalNotify:
		payload := &api.ActivityIssueApprovalNotifyPayload{}
		if err := json.Unmarshal([]byte(activity.Payload), payload); err != nil {
			slog.Warn("Failed to post webhook event after changing the issue approval node status, failed to unmarshal payload",
				slog.String("issue_name", meta.Issue.Title),
				log.BBError(err))
			return nil, err
		}
		protoPayload := &storepb.ActivityIssueApprovalNotifyPayload{}
		if err := protojson.Unmarshal([]byte(payload.ProtoPayload), protoPayload); err != nil {
			slog.Warn("Failed to post webhook event")
		}
		pendingStep := protoPayload.ApprovalStep

		title = "Issue approval needed - " + meta.Issue.Title

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
				usersGetter = getUsersFromWorkspaceRole(m.store, api.Owner)
			case storepb.ApprovalNode_WORKSPACE_DBA:
				usersGetter = getUsersFromWorkspaceRole(m.store, api.DBA)
			case storepb.ApprovalNode_PROJECT_OWNER:
				usersGetter = getUsersFromProjectRole(m.store, api.Owner, meta.Issue.Project.ResourceID)
			case storepb.ApprovalNode_PROJECT_MEMBER:
				usersGetter = getUsersFromProjectRole(m.store, api.Developer, meta.Issue.Project.ResourceID)
			default:
				return nil, errors.Errorf("invalid group value")
			}
		case *storepb.ApprovalNode_Role:
			role := api.Role(strings.TrimPrefix(val.Role, "roles/"))
			usersGetter = getUsersFromProjectRole(m.store, role, meta.Issue.Project.ResourceID)
		case *storepb.ApprovalNode_ExternalNodeId:
			usersGetter = func(ctx context.Context) ([]*store.UserMessage, error) {
				return nil, nil
			}
		default:
			return nil, errors.Errorf("invalid node payload type")
		}

		users, err := usersGetter(ctx)
		if err != nil {
			slog.Warn("Failed to post webhook event after changing the issue approval node status, failed to get users",
				slog.String("issue_name", meta.Issue.Title),
				log.BBError(err))
			return nil, err
		}
		for _, user := range users {
			phone, err := maybeGetPhoneFromUser(user)
			if err != nil {
				slog.Warn("failed to parse phone number",
					slog.String("issue_name", meta.Issue.Title),
					log.BBError(err))
				continue
			}
			if phone != "" {
				mentions = append(mentions, phone)
			}
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
		TaskResult:          webhookTaskResult,
		Description:         activity.Comment,
		Link:                link,
		CreatorID:           updater.ID,
		CreatorName:         updater.Name,
		CreatorEmail:        updater.Email,
		MentionUsersByPhone: mentions,
	}
	return &webhookCtx, nil
}

func (m *Manager) postInboxIssueActivity(ctx context.Context, issue *store.IssueMessage, activityID int) error {
	if issue.Creator.ID != api.SystemBotID {
		if _, err := m.store.CreateInbox(ctx, &store.InboxMessage{
			ReceiverUID: issue.Creator.ID,
			ActivityUID: activityID,
		}); err != nil {
			return errors.Wrapf(err, "failed to post activity to creator inbox: %d", issue.Creator.ID)
		}
	}

	if issue.Assignee != nil && issue.Assignee.ID != api.SystemBotID && issue.Assignee.ID != issue.Creator.ID {
		if _, err := m.store.CreateInbox(ctx, &store.InboxMessage{
			ReceiverUID: issue.Assignee.ID,
			ActivityUID: activityID,
		}); err != nil {
			return errors.Wrapf(err, "failed to post activity to assignee inbox: %d", issue.Assignee.ID)
		}
	}

	for _, subscriber := range issue.Subscribers {
		if subscriber.ID != api.SystemBotID && subscriber.ID != issue.Creator.ID && (issue.Assignee == nil || subscriber.ID != issue.Assignee.ID) {
			if _, err := m.store.CreateInbox(ctx, &store.InboxMessage{
				ReceiverUID: subscriber.ID,
				ActivityUID: activityID,
			}); err != nil {
				return errors.Wrapf(err, "failed to post activity to subscriber inbox: %d", subscriber.ID)
			}
		}
	}

	return nil
}

func shouldPostInbox(activity *store.ActivityMessage, createType api.ActivityType) (bool, error) {
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
	case api.ActivityNotifyIssueApproved:
		return false, nil
	case api.ActivityNotifyPipelineRollout:
		return false, nil
	}
	return false, nil
}

func getUsersFromWorkspaceRole(s *store.Store, role api.Role) func(context.Context) ([]*store.UserMessage, error) {
	return func(ctx context.Context) ([]*store.UserMessage, error) {
		return s.ListUsers(ctx, &store.FindUserMessage{
			Role: &role,
		})
	}
}

func getUsersFromProjectRole(s *store.Store, role api.Role, projectID string) func(context.Context) ([]*store.UserMessage, error) {
	return func(ctx context.Context) ([]*store.UserMessage, error) {
		projectIAM, err := s.GetProjectPolicy(ctx, &store.GetProjectPolicyMessage{
			ProjectID: &projectID,
		})
		if err != nil {
			return nil, err
		}
		var users []*store.UserMessage
		for _, binding := range projectIAM.Bindings {
			if binding.Role == role {
				users = append(users, binding.Members...)
			}
		}
		return users, nil
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

// getTaskSkippedAndReason gets skipped and skippedReason from a task.
func getTaskSkippedAndReason(task *store.TaskMessage) (bool, string, error) {
	var payload struct {
		Skipped       bool   `json:"skipped,omitempty"`
		SkippedReason string `json:"skippedReason,omitempty"`
	}
	if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil {
		return false, "", err
	}
	return payload.Skipped, payload.SkippedReason, nil
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
	return strconv.FormatInt(int64(*phoneNumber.NationalNumber), 10), nil
}
