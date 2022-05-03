package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/bytebase/bytebase/api"
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
		return nil, fmt.Errorf("failed to find project webhook after changing the issue status: %v, error: %w", meta.issue.Name, err)
	}
	if len(webhookList) == 0 {
		return activity, nil
	}

	// If we need to post webhook event, then we need to make sure the project info exists since we will include
	// the project name in the webhook event.
	if meta.issue.Project == nil {
		project, err := m.s.store.GetProjectByID(ctx, meta.issue.ProjectID)
		if err != nil {
			return nil, fmt.Errorf("failed to find project for posting webhook event after changing the issue status: %v, error: %w", meta.issue.Name, err)
		}
		if project == nil {
			return nil, fmt.Errorf("failed to find project ID %v for posting webhook event after changing the issue status %q", meta.issue.ProjectID, meta.issue.Name)
		}
		// TODO(dragonly): revisit the necessity of this function to depend on ActivityMeta.
		meta.issue.Project = project
	}

	updater, err := m.s.store.GetPrincipalByID(ctx, create.CreatorID)
	if err != nil {
		return nil, fmt.Errorf("failed to find updater for posting webhook event after changing the issue status: %v, error: %w", meta.issue.Name, err)
	}
	if updater == nil {
		return nil, fmt.Errorf("updater principal not found for ID %v", create.CreatorID)
	}

	// Call external webhook endpoint in Go routine to avoid blocking web serveing thread.
	go func() {
		webhookCtx, err := m.getWebhookContext(ctx, activity, meta, updater)
		if err != nil {
			return
		}

		for _, hook := range webhookList {
			webhookCtx.URL = hook.URL
			webhookCtx.CreatedTs = time.Now().Unix()
			if err := webhook.Post(hook.Type, webhookCtx); err != nil {
				// The external webhook endpoint might be invalid which is out of our code control, so we just emit a warning
				m.s.l.Warn("Failed to post webhook event after changing the issue status",
					zap.String("webhook_type", hook.Type),
					zap.String("webhook_name", hook.Name),
					zap.String("issue_name", meta.issue.Name),
					zap.String("status", string(meta.issue.Status)),
					zap.Error(err))
			}
		}
	}()

	return activity, nil
}

func (m *ActivityManager) getWebhookContext(ctx context.Context, activity *api.Activity, meta *ActivityMeta, updater *api.Principal) (webhook.Context, error) {
	var webhookCtx webhook.Context
	level := webhook.WebhookInfo
	title := ""
	link := fmt.Sprintf("%s:%d/issue/%s", m.s.frontendHost, m.s.frontendPort, api.IssueSlug(meta.issue))
	switch activity.Type {
	case api.ActivityIssueCreate:
		title = "Issue created - " + meta.issue.Name
	case api.ActivityIssueStatusUpdate:
		switch meta.issue.Status {
		case "OPEN":
			title = "Issue reopened - " + meta.issue.Name
		case "DONE":
			level = webhook.WebhookSuccess
			title = "Issue resolved - " + meta.issue.Name
		case "CANCELED":
			title = "Issue canceled - " + meta.issue.Name
		}
	case api.ActivityIssueCommentCreate:
		title = "Comment created"
		link += fmt.Sprintf("#activity%d", activity.ID)
	case api.ActivityIssueFieldUpdate:
		update := new(api.ActivityIssueFieldUpdatePayload)
		if err := json.Unmarshal([]byte(activity.Payload), update); err != nil {
			m.s.l.Warn("Failed to post webhook event after changing the issue field, failed to unmarshal payload",
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
						m.s.l.Warn("Failed to post webhook event after changing the issue assignee, old assignee id is not number",
							zap.String("issue_name", meta.issue.Name),
							zap.String("old_assignee_id", update.OldValue),
							zap.Error(err))
						return webhookCtx, err
					}
					oldAssignee, err = m.s.store.GetPrincipalByID(ctx, oldID)
					if err != nil {
						m.s.l.Warn("Failed to post webhook event after changing the issue assignee, failed to find old assignee",
							zap.String("issue_name", meta.issue.Name),
							zap.String("old_assignee_id", update.OldValue),
							zap.Error(err))
						return webhookCtx, err
					}
					if oldAssignee == nil {
						err := fmt.Errorf("failed to post webhook event after changing the issue assignee, old assignee not found for ID %v", oldID)
						m.s.l.Warn(err.Error(),
							zap.String("issue_name", meta.issue.Name),
							zap.String("old_assignee_id", update.OldValue),
							zap.Error(err))
						return webhookCtx, err
					}
				}

				if update.NewValue != "" {
					newID, err := strconv.Atoi(update.NewValue)
					if err != nil {
						m.s.l.Warn("Failed to post webhook event after changing the issue assignee, new assignee id is not number",
							zap.String("issue_name", meta.issue.Name),
							zap.String("new_assignee_id", update.NewValue),
							zap.Error(err))
						return webhookCtx, err
					}
					newAssignee, err = m.s.store.GetPrincipalByID(ctx, newID)
					if err != nil {
						m.s.l.Warn("Failed to post webhook event after changing the issue assignee, failed to find new assignee",
							zap.String("issue_name", meta.issue.Name),
							zap.String("new_assignee_id", update.NewValue),
							zap.Error(err))
						return webhookCtx, err
					}
					if newAssignee == nil {
						err := fmt.Errorf("failed to post webhook event after changing the issue assignee, new assignee not found for ID %v", newID)
						m.s.l.Warn(err.Error(),
							zap.String("issue_name", meta.issue.Name),
							zap.String("new_assignee_id", update.NewValue),
							zap.Error(err))
						return webhookCtx, err
					}

					if oldAssignee != nil && newAssignee != nil {
						title = fmt.Sprintf("Reassigned issue from %s to %s", oldAssignee.Name, newAssignee.Name)
					} else if newAssignee != nil {
						title = fmt.Sprintf("Assigned issue to %s", newAssignee.Name)
					} else if oldAssignee != nil {
						title = fmt.Sprintf("Unassigned issue from %s", newAssignee.Name)
					}
				}
			}
		case api.IssueFieldDescription:
			title = "Changed issue description"
		case api.IssueFieldName:
			title = "Changed issue name"
		default:
			title = "Updated issue"
		}
	case api.ActivityPipelineTaskStatusUpdate:
		update := &api.ActivityPipelineTaskStatusUpdatePayload{}
		if err := json.Unmarshal([]byte(activity.Payload), update); err != nil {
			m.s.l.Warn("Failed to post webhook event after changing the issue task status, failed to unmarshal payload",
				zap.String("issue_name", meta.issue.Name),
				zap.Error(err))
			return webhookCtx, err
		}

		task, err := m.s.store.GetTaskByID(ctx, update.TaskID)
		if err != nil {
			m.s.l.Warn("Failed to post webhook event after changing the issue task status, failed to find task",
				zap.String("issue_name", meta.issue.Name),
				zap.Int("task_id", update.TaskID),
				zap.Error(err))
			return webhookCtx, err
		}
		if task == nil {
			err := fmt.Errorf("failed to post webhook event after changing the issue task status, task not found for ID %v", update.TaskID)
			m.s.l.Warn(err.Error(),
				zap.String("issue_name", meta.issue.Name),
				zap.Int("task_id", update.TaskID),
				zap.Error(err))
			return webhookCtx, err
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
		}
	}

	metaList := []webhook.Meta{
		{
			Name:  "Issue",
			Value: meta.issue.Name,
		},
		{
			Name:  "Project",
			Value: meta.issue.Project.Name,
		},
	}
	webhookCtx = webhook.Context{
		Level:        level,
		ActivityType: string(activity.Type),
		Title:        title,
		Issue: webhook.Issue{
			ID:          meta.issue.ID,
			Name:        meta.issue.Name,
			Status:      string(meta.issue.Status),
			Type:        string(meta.issue.Type),
			Description: meta.issue.Description,
		},
		Project: webhook.Project{
			ID:   meta.issue.ProjectID,
			Name: meta.issue.Project.Name,
		},
		Description:  activity.Comment,
		Link:         link,
		CreatorID:    updater.ID,
		CreatorName:  updater.Name,
		CreatorEmail: updater.Email,
		MetaList:     metaList,
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
