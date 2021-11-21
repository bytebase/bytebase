package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/webhook"
	"go.uber.org/zap"
)

type ActivityManager struct {
	s               *Server
	activityService api.ActivityService
}

type ActivityMeta struct {
	issue *api.Issue
}

func NewActivityManager(server *Server, activityService api.ActivityService) *ActivityManager {
	return &ActivityManager{
		s:               server,
		activityService: activityService,
	}
}

func (m *ActivityManager) CreateActivity(ctx context.Context, create *api.ActivityCreate, meta *ActivityMeta) (*api.Activity, error) {
	activity, err := m.activityService.CreateActivity(ctx, create)
	if err != nil {
		return nil, err
	}

	if meta.issue != nil {
		postInbox := false
		switch create.Type {
		case api.ActivityIssueCreate:
			postInbox = true
		case api.ActivityIssueStatusUpdate:
			postInbox = true
		case api.ActivityIssueCommentCreate:
			postInbox = true
		case api.ActivityIssueFieldUpdate:
			postInbox = true
		case api.ActivityPipelineTaskStatusUpdate:
			update := &api.ActivityPipelineTaskStatusUpdatePayload{}
			if err := json.Unmarshal([]byte(activity.Payload), update); err != nil {
				return nil, fmt.Errorf("failed to post webhook event after changing the issue task status: %v, error: %w", meta.issue.Name, err)
			}
			// To reduce noise, for now we only post status update to inbox upon task failure.
			if update.NewStatus == api.TaskFailed {
				postInbox = true
			}
		}

		if postInbox {
			if err := m.s.PostInboxIssueActivity(ctx, meta.issue, activity.ID); err != nil {
				return nil, err
			}
		}

		hookFind := &api.ProjectWebhookFind{
			ProjectID:    &meta.issue.ProjectID,
			ActivityType: &create.Type,
		}
		hookList, err := m.s.ProjectWebhookService.FindProjectWebhookList(ctx, hookFind)
		if err != nil {
			return nil, fmt.Errorf("failed to find project webhook after changing the issue status: %v, error: %w", meta.issue.Name, err)
		}

		// If we need to post webhook event, then we need to make sure the project info exists since we will include
		// the project name in the webhook event.
		if len(hookList) > 0 {
			if meta.issue.Project == nil {
				projectFind := &api.ProjectFind{
					ID: &meta.issue.ProjectID,
				}
				meta.issue.Project, err = m.s.ProjectService.FindProject(ctx, projectFind)
				if err != nil {
					return nil, fmt.Errorf("failed to find project for posting webhook event after changing the issue status: %v, error: %w", meta.issue.Name, err)
				}
			}

			principalFind := &api.PrincipalFind{
				ID: &create.CreatorID,
			}
			updater, err := m.s.PrincipalService.FindPrincipal(ctx, principalFind)
			if err != nil {
				return nil, fmt.Errorf("failed to find updater for posting webhook event after changing the issue status: %v, error: %w", meta.issue.Name, err)
			}

			// Call exteranl webhook endpoint in Go routine to avoid blocking web serveing thread.
			go func() {
				for _, hook := range hookList {
					level := webhook.WebhookInfo
					title := ""
					link := fmt.Sprintf("%s:%d/issue/%s", m.s.frontendHost, m.s.frontendPort, api.IssueSlug(meta.issue))
					metaList := []webhook.WebhookMeta{}
					switch create.Type {
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
						title = "Comment created"
						link += fmt.Sprintf("#activity%d", activity.ID)
					case api.ActivityIssueFieldUpdate:
						update := &api.ActivityIssueFieldUpdatePayload{}
						if err := json.Unmarshal([]byte(activity.Payload), update); err != nil {
							m.s.l.Warn("Failed to post webhook event after changing the issue field, failed to unmarshal paylaod",
								zap.String("issue_name", meta.issue.Name),
								zap.Error(err))
							return
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
										return
									}
									principalFind := &api.PrincipalFind{
										ID: &oldID,
									}
									oldAssignee, err = m.s.PrincipalService.FindPrincipal(ctx, principalFind)
									if err != nil {
										m.s.l.Warn("Failed to post webhook event after changing the issue assignee, failed to find old assignee",
											zap.String("issue_name", meta.issue.Name),
											zap.String("old_assignee_id", update.OldValue),
											zap.Error(err))
										return
									}
								}

								if update.NewValue != "" {
									newID, err := strconv.Atoi(update.NewValue)
									if err != nil {
										m.s.l.Warn("Failed to post webhook event after changing the issue assignee, new assignee id is not number",
											zap.String("issue_name", meta.issue.Name),
											zap.String("old_assignee_id", update.NewValue),
											zap.Error(err))
										return
									}
									principalFind := &api.PrincipalFind{
										ID: &newID,
									}
									newAssignee, err = m.s.PrincipalService.FindPrincipal(ctx, principalFind)
									if err != nil {
										m.s.l.Warn("Failed to post webhook event after changing the issue assignee, failed to find new assignee",
											zap.String("issue_name", meta.issue.Name),
											zap.String("new_assignee_id", update.NewValue),
											zap.Error(err))
										return
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
							m.s.l.Warn("Failed to post webhook event after changing the issue task status, failed to unmarshal paylaod",
								zap.String("issue_name", meta.issue.Name),
								zap.Error(err))
							return
						}

						taskFind := &api.TaskFind{
							ID: &update.TaskID,
						}
						task, err := m.s.TaskService.FindTask(ctx, taskFind)
						if err != nil {
							m.s.l.Warn("Failed to post webhook event after changing the issue task status, failed to find task",
								zap.String("issue_name", meta.issue.Name),
								zap.Int("task_id", update.TaskID),
								zap.Error(err))
							return
						}

						title = fmt.Sprintf("Task changed - %s", task.Name)
						switch update.NewStatus {
						case api.TaskPending:
							if update.OldStatus == api.TaskRunning {
								title = fmt.Sprintf("Task canceled - %s", task.Name)
							} else if update.OldStatus == api.TaskPendingApproval {
								title = fmt.Sprintf("Task approved - %s", task.Name)
							}
						case api.TaskRunning:
							title = fmt.Sprintf("Task started - %s", task.Name)
						case api.TaskDone:
							level = webhook.WebhookSuccess
							title = fmt.Sprintf("Task completed - %s", task.Name)
						case api.TaskFailed:
							level = webhook.WebhookError
							title = fmt.Sprintf("Task failed - %s", task.Name)
						}
					}

					metaList = append(metaList, webhook.WebhookMeta{
						Name:  "Issue",
						Value: meta.issue.Name,
					})
					metaList = append(metaList, webhook.WebhookMeta{
						Name:  "Project",
						Value: meta.issue.Project.Name,
					})

					err := webhook.Post(
						hook.Type,
						webhook.WebhookContext{
							URL:          hook.URL,
							Level:        level,
							Title:        title,
							Description:  create.Comment,
							Link:         link,
							CreatorName:  updater.Name,
							CreatorEmail: updater.Email,
							CreatedTs:    time.Now().Unix(),
							MetaList:     metaList,
						},
					)
					if err != nil {
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
		}
	}

	return activity, nil
}
