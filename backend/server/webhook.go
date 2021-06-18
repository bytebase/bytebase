package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/bytebase/bytebase"
	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/external/gitlab"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

var (
	gitLabWebhookPath = "hook/gitlab"
)

func (s *Server) registerWebhookRoutes(g *echo.Group) {
	g.POST("/gitlab/:id", func(c echo.Context) error {
		var b []byte
		// TODO: We may save the raw event in the furture for async processing.
		b, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Failed to read webhook request").SetInternal(err)
		}

		pushEvent := &gitlab.WebhookPushEvent{}
		if err := json.Unmarshal(b, pushEvent); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted push event").SetInternal(err)
		}
		s.l.Info((fmt.Sprintf("Push event: %+v", *pushEvent)))

		// This shouldn't happen as we only setup webhook to receive push event, just in case.
		if pushEvent.ObjectKind != gitlab.WebhookPush {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid webhook event type. Expect push, got %s", pushEvent.ObjectKind))
		}

		webhookEndpointId := c.Param("id")
		repositoryFind := &api.RepositoryFind{
			WebhookEndpointId: &webhookEndpointId,
		}
		repository, err := s.RepositoryService.FindRepository(context.Background(), repositoryFind)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Endpoint not found: %v", webhookEndpointId))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to respond webhook event for endpoint: %v", webhookEndpointId)).SetInternal(err)
		}

		if err := s.ComposeRepositoryRelationship(context.Background(), repository, []string{}); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch repository relationship: %v", repository.Name)).SetInternal(err)
		}

		if c.Request().Header.Get("X-Gitlab-Token") != repository.SecretToken {
			return echo.NewHTTPError(http.StatusBadRequest, "Secret token mismatch")
		}

		if strconv.Itoa(pushEvent.Project.ID) != repository.ExternalId {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project mismatch. Expect %s, got %d", repository.ExternalId, pushEvent.Project.ID))
		}

		createdMessageList := []string{}
		for _, commit := range pushEvent.CommitList {
			for _, added := range commit.AddedList {
				if strings.HasPrefix(added, repository.BaseDirectory) && filepath.Ext(added) == ".sql" {
					filename := filepath.Base(added)

					// We expect filename in the {{version}}_{{dbname}}[_{{description}}]
					// The description part is optional
					parts := strings.Split(strings.TrimSuffix(filename, filepath.Ext(added)), "_")
					if len(parts) < 2 {
						s.l.Warn("Added SQL file name does not conform to the expected format. Ignored", zap.String("filename", filename))
						continue
					}
					//version := parts[0]
					dbName := parts[1]
					description := ""
					if len(parts) > 2 {
						description = strings.Join(parts[2:], " ")
					}

					// Retrieve sql by reading the file content
					resp, err := gitlab.GET(
						repository.VCS.InstanceURL,
						fmt.Sprintf("projects/%s/repository/files/%s/raw?ref=%s", repository.ExternalId, url.QueryEscape(added), commit.ID),
						repository.VCS.AccessToken,
					)
					if err != nil {
						s.l.Warn("Failed to read file. Ignored", zap.String("filename", filename), zap.Error(err))
						continue
					}

					b, err := io.ReadAll(resp.Body)
					if err != nil {
						s.l.Warn("Failed to read file response. Ignored", zap.String("filename", filename), zap.Error(err))
						continue
					}

					// Find matching database
					databaseFind := &api.DatabaseFind{
						ProjectId: &repository.ProjectId,
						Name:      &dbName,
					}
					database, err := s.ComposeDatabaseByFind(context.Background(), databaseFind, []string{})
					if err != nil {
						if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
							s.l.Warn(fmt.Sprintf("Project ID %d does not contain database %s. Ignored", repository.ProjectId, dbName),
								zap.Int("project_id", repository.ProjectId),
								zap.String("filename", filename),
							)

						} else {
							s.l.Warn("Failed to find database. Ignored", zap.String("filename", filename), zap.Error(err))
						}
						continue
					}

					// Compose the new issue
					createdTime, err := time.Parse(time.RFC3339, commit.Timestamp)
					if err != nil {
						s.l.Warn("Failed to parse timestamp. Ignored", zap.String("filename", filename), zap.Error(err))
					}
					vcsPushEvent := api.VCSPushEvent{
						Ref:                pushEvent.Ref,
						RepositoryID:       strconv.Itoa(pushEvent.Project.ID),
						RepositoryURL:      pushEvent.Project.WebURL,
						RepositoryFullPath: pushEvent.Project.FullPath,
						AuthorName:         pushEvent.AuthorName,
						FileCommit: api.VCSFileCommit{
							ID:         commit.ID,
							Title:      commit.Title,
							Message:    commit.Message,
							CreatedTs:  createdTime.Unix(),
							URL:        commit.URL,
							AuthorName: commit.Author.Name,
							Added:      added,
						},
					}
					task := &api.TaskCreate{
						InstanceId:   database.InstanceId,
						DatabaseId:   database.ID,
						Name:         description,
						Status:       "PENDING",
						Type:         api.TaskDatabaseSchemaUpdate,
						Statement:    string(b),
						VCSPushEvent: &vcsPushEvent,
					}
					stage := &api.StageCreate{
						EnvironmentId: database.Instance.EnvironmentId,
						TaskList:      []api.TaskCreate{*task},
						Name:          fmt.Sprintf("Update %s schema", database.Name),
					}
					pipeline := &api.PipelineCreate{
						StageList: []api.StageCreate{*stage},
						Name:      commit.Title,
					}
					issueCreate := &api.IssueCreate{
						ProjectId:   database.ProjectId,
						Pipeline:    *pipeline,
						Name:        commit.Title,
						Type:        api.IssueDatabaseSchemaUpdate,
						Description: commit.Message,
						AssigneeId:  api.SYSTEM_BOT_ID,
					}

					issue, err := s.CreateIssue(context.Background(), issueCreate, api.SYSTEM_BOT_ID)
					if err != nil {
						s.l.Warn(fmt.Sprintf("Failed to create update schema task after adding %s", filename), zap.Error(err),
							zap.String("filename", filename))
						continue
					}
					createdMessageList = append(createdMessageList, fmt.Sprintf("Created issue '%s' on adding %s", issue.Name, filename))
				}
			}
		}

		return c.String(http.StatusOK, strings.Join(createdMessageList, "\n"))
	})
}
