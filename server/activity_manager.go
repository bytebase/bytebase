package server

import (
	"context"
	"fmt"
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

func (m *ActivityManager) CreateActivity(ctx context.Context, create *api.ActivityCreate, meta *ActivityMeta) error {
	activity, err := m.activityService.CreateActivity(ctx, create)
	if err != nil {
		return err
	}

	if meta.issue != nil {
		if err := m.s.PostInboxIssueActivity(context.Background(), meta.issue, activity.ID); err != nil {
			return err
		}

		hookFind := &api.ProjectWebhookFind{
			ProjectId:    &meta.issue.ProjectId,
			ActivityType: &create.Type,
		}
		hookList, err := m.s.ProjectWebhookService.FindProjectWebhookList(context.Background(), hookFind)
		if err != nil {
			return fmt.Errorf("failed to find project webhook hook after changing the issue status: %v, error: %w", meta.issue.Name, err)
		}

		// If we need to post webhook event, then we need to make sure the project info exists since we will include
		// the project name in the webhook event.
		if len(hookList) > 0 {
			if meta.issue.Project == nil {
				projectFind := &api.ProjectFind{
					ID: &meta.issue.ProjectId,
				}
				meta.issue.Project, err = m.s.ProjectService.FindProject(ctx, projectFind)
				if err != nil {
					return fmt.Errorf("failed to find project for posting webhook event after changing the issue status: %v, error: %w", meta.issue.Name, err)
				}
			}

			principalFind := &api.PrincipalFind{
				ID: &create.CreatorId,
			}
			updater, err := m.s.PrincipalService.FindPrincipal(context.Background(), principalFind)
			if err != nil {
				return fmt.Errorf("failed to find updater for posting webhook event after changing the issue status: %v, error: %w", meta.issue.Name, err)
			}

			// Call exteranl webhook endpoint in Go routine to avoid blocking web serveing thread.
			go func() {
				for _, hook := range hookList {
					title := ""
					switch meta.issue.Status {
					case "OPEN":
						title = fmt.Sprintf("Issue reopened - %s", meta.issue.Name)
					case "DONE":
						title = fmt.Sprintf("Issue resolved - %s", meta.issue.Name)
					case "CANCELED":
						title = fmt.Sprintf("Issue canceled - %s", meta.issue.Name)
					}

					err := webhook.Post(
						hook.Type,
						webhook.WebhookContext{
							URL:          hook.URL,
							Title:        title,
							Description:  create.Comment,
							Link:         fmt.Sprintf("%s:%d/issue/%s", m.s.frontendHost, m.s.frontendPort, api.IssueSlug(meta.issue)),
							CreatorName:  updater.Name,
							CreatorEmail: updater.Email,
							CreatedTs:    time.Now().Unix(),
							MetaList: []webhook.WebhookMeta{
								{
									Name:  "Project",
									Value: meta.issue.Project.Name,
								},
							},
						},
					)
					if err != nil {
						// The external webhook endpoint might be invalid which is out of our code control, so we just emit a warning
						m.s.l.Warn("Failed to post webhook event after changing the issue status",
							zap.String("issue_name", meta.issue.Name),
							zap.String("status", string(meta.issue.Status)),
							zap.Error(err))
					}
				}
			}()
		}
	}

	return nil
}
