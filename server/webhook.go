package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/vcs"
	"github.com/bytebase/bytebase/plugin/vcs/gitlab"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

var (
	gitLabWebhookPath = "hook/gitlab"
)

func (s *Server) registerWebhookRoutes(g *echo.Group) {
	g.POST("/gitlab/:id", func(c echo.Context) error {
		ctx := context.Background()
		var b []byte
		b, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Failed to read webhook request").SetInternal(err)
		}

		pushEvent := &gitlab.WebhookPushEvent{}
		if err := json.Unmarshal(b, pushEvent); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted push event").SetInternal(err)
		}

		// This shouldn't happen as we only setup webhook to receive push event, just in case.
		if pushEvent.ObjectKind != gitlab.WebhookPush {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid webhook event type, got %s, want push", pushEvent.ObjectKind))
		}

		webhookEndpointID := c.Param("id")
		repositoryFind := &api.RepositoryFind{
			WebhookEndpointID: &webhookEndpointID,
		}
		repository, err := s.RepositoryService.FindRepository(ctx, repositoryFind)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Endpoint not found: %v", webhookEndpointID))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to respond webhook event for endpoint: %v", webhookEndpointID)).SetInternal(err)
		}

		if err := s.composeRepositoryRelationship(ctx, repository); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch repository relationship: %v", repository.Name)).SetInternal(err)
		}
		if repository.VCS == nil {
			err := fmt.Errorf("VCS not found for ID: %v", repository.VCSID)
			return echo.NewHTTPError(http.StatusInternalServerError, err).SetInternal(err)
		}

		if c.Request().Header.Get("X-Gitlab-Token") != repository.WebhookSecretToken {
			return echo.NewHTTPError(http.StatusBadRequest, "Secret token mismatch")
		}

		if strconv.Itoa(pushEvent.Project.ID) != repository.ExternalID {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project mismatch, got %d, want %s", pushEvent.Project.ID, repository.ExternalID))
		}

		createdMessageList := []string{}
		for _, commit := range pushEvent.CommitList {
			for _, added := range commit.AddedList {
				if !strings.HasPrefix(added, repository.BaseDirectory) {
					s.l.Debug("Ignored committed file, not under base directory.", zap.String("file", added), zap.String("base_directory", repository.BaseDirectory))
					continue
				}

				createdTime, err := time.Parse(time.RFC3339, commit.Timestamp)
				if err != nil {
					s.l.Warn("Ignored committed file, failed to parse commit timestamp.", zap.String("file", added), zap.String("timestamp", commit.Timestamp), zap.Error(err))
				}

				// Ignored the schema file we auto generated to the repository.
				if repository.SchemaPathTemplate != "" {
					placeholderList := []string{
						"ENV_NAME",
						"DB_NAME",
					}
					schemafilePathRegex := repository.SchemaPathTemplate
					for _, placeholder := range placeholderList {
						schemafilePathRegex = strings.ReplaceAll(schemafilePathRegex, fmt.Sprintf("{{%s}}", placeholder), fmt.Sprintf("(?P<%s>[a-zA-Z0-9+-=/_#?!$. ]+)", placeholder))
					}
					myRegex, err := regexp.Compile(schemafilePathRegex)
					if err != nil {
						s.l.Warn("Invalid schema path template.", zap.String("schema_path_template",
							repository.SchemaPathTemplate),
							zap.Error(err),
						)
					}
					if myRegex.MatchString(added) {
						continue
					}
				}

				vcsPushEvent := vcs.VCSPushEvent{
					VCSType:            repository.VCS.Type,
					BaseDirectory:      repository.BaseDirectory,
					Ref:                pushEvent.Ref,
					RepositoryID:       strconv.Itoa(pushEvent.Project.ID),
					RepositoryURL:      pushEvent.Project.WebURL,
					RepositoryFullPath: pushEvent.Project.FullPath,
					AuthorName:         pushEvent.AuthorName,
					FileCommit: vcs.VCSFileCommit{
						ID:         commit.ID,
						Title:      commit.Title,
						Message:    commit.Message,
						CreatedTs:  createdTime.Unix(),
						URL:        commit.URL,
						AuthorName: commit.Author.Name,
						Added:      added,
					},
				}

				// Create a WARNING project activity if committed file is ignored
				var createIgnoredFileActivity = func(err error) {
					s.l.Warn("Ignored committed file", zap.String("file", added), zap.Error(err))
					bytes, marshalErr := json.Marshal(api.ActivityProjectRepositoryPushPayload{
						VCSPushEvent: vcsPushEvent,
					})
					if marshalErr != nil {
						s.l.Warn("Failed to construct project activity payload to record ignored repository committed file", zap.Error(marshalErr))
						return
					}

					activityCreate := &api.ActivityCreate{
						CreatorID:   api.SystemBotID,
						ContainerID: repository.ProjectID,
						Type:        api.ActivityProjectRepositoryPush,
						Level:       api.ActivityWarn,
						Comment:     fmt.Sprintf("Ignored committed file %q, %s.", added, err.Error()),
						Payload:     string(bytes),
					}
					_, err = s.ActivityManager.CreateActivity(ctx, activityCreate, &ActivityMeta{})
					if err != nil {
						s.l.Warn("Failed to create project activity to record ignored repository committed file", zap.Error(err))
					}
				}

				mi, err := db.ParseMigrationInfo(added, filepath.Join(repository.BaseDirectory, repository.FilePathTemplate))
				if err != nil {
					createIgnoredFileActivity(err)
					continue
				}

				// Retrieve sql by reading the file content
				reader, err := vcs.Get(vcs.GitLabSelfHost, vcs.ProviderConfig{Logger: s.l}).ReadFile(
					ctx,
					common.OauthContext{
						ClientID:     repository.VCS.ApplicationID,
						ClientSecret: repository.VCS.Secret,
						RefreshToken: repository.RefreshToken,
						Refresher:    s.refreshToken(ctx, repository.ID),
					},
					repository.VCS.InstanceURL,
					repository.ExternalID,
					added,
					commit.ID,
				)
				if err != nil {
					createIgnoredFileActivity(err)
					continue
				}
				defer reader.Close()

				b, err := io.ReadAll(reader)
				if err != nil {
					createIgnoredFileActivity(fmt.Errorf("failed to read file response: %w", err))
					continue
				}

				// Find matching database list
				databaseFind := &api.DatabaseFind{
					ProjectID: &repository.ProjectID,
					Name:      &mi.Database,
				}
				databaseList, err := s.composeDatabaseListByFind(ctx, databaseFind)
				if err != nil {
					createIgnoredFileActivity(fmt.Errorf("failed to find database matching database %q referenced by the committed file", mi.Database))
					continue
				} else if len(databaseList) == 0 {
					createIgnoredFileActivity(fmt.Errorf("project ID %d does not own database %q referenced by the committed file", repository.ProjectID, mi.Database))
					continue
				}

				// We support 3 patterns on how to organize the schema files.
				// Pattern 1: 	The database name is the same across all environments. Each environment will have its own directory, so the
				//              schema file looks like "dev/v1__db1", "staging/v1__db1".
				//
				// Pattern 2: 	Like 1, the database name is the same across all environments. All environment shares the same schema file,
				//              say v1__db1, when a new file is added like v2__db1__add_column, we will create a multi stage pipeline where
				//              each stage corresponds to an environment.
				//
				// Pattern 3:  	The database name is different among different environments. In such case, the database name alone is enough
				//             	to identify ambiguity.

				// Further filter by environment name if applicable.
				filteredDatabaseList := []*api.Database{}
				if mi.Environment != "" {
					for _, database := range databaseList {
						// Environment name comparision is case insensitive
						if strings.EqualFold(database.Instance.Environment.Name, mi.Environment) {
							filteredDatabaseList = append(filteredDatabaseList, database)
						}
					}
					if len(filteredDatabaseList) == 0 {
						createIgnoredFileActivity(fmt.Errorf("project does not contain committed file database %q for environment %q", mi.Database, mi.Environment))
						continue
					}
				} else {
					filteredDatabaseList = databaseList
				}

				var pipelineApprovalByEnv = map[int]api.PipelineApprovalValue{}
				{
					// It could happen that for a particular environment a project contain 2 database with the same name.
					// We will emit warning in this case.
					var databaseListByEnv = map[int][]*api.Database{}
					for _, database := range filteredDatabaseList {
						list, ok := databaseListByEnv[database.Instance.EnvironmentID]
						if ok {
							databaseListByEnv[database.Instance.EnvironmentID] = append(list, database)
						} else {
							list := make([]*api.Database, 0)
							databaseListByEnv[database.Instance.EnvironmentID] = append(list, database)
						}

						// Load pipeline approval policy per environment.
						if _, ok := pipelineApprovalByEnv[database.Instance.EnvironmentID]; !ok {
							p, err := s.PolicyService.GetPipelineApprovalPolicy(ctx, database.Instance.EnvironmentID)
							if err != nil {
								createIgnoredFileActivity(fmt.Errorf("failed to find pipeline approval policy for environment %v", database.Instance.EnvironmentID))
								continue
							}
							pipelineApprovalByEnv[database.Instance.EnvironmentID] = p.Value
						}
					}

					var multipleDatabaseForSameEnv = false
					for environmentID, databaseList := range databaseListByEnv {
						if len(databaseList) > 1 {
							multipleDatabaseForSameEnv = true

							s.l.Warn(fmt.Sprintf("Ignored committed file, multiple ambiguous databases named %q for environment %d.", mi.Database, environmentID),
								zap.Int("project_id", repository.ProjectID),
								zap.String("file", added),
							)
						}
					}

					if multipleDatabaseForSameEnv {
						continue
					}
				}

				// Compose the new issue
				m := &api.UpdateSchemaContext{
					MigrationType: mi.Type,
					VCSPushEvent:  &vcsPushEvent,
				}
				for _, database := range filteredDatabaseList {
					m.UpdateSchemaDetailList = append(m.UpdateSchemaDetailList,
						&api.UpdateSchemaDetail{
							DatabaseID: database.ID,
							Statement:  string(b),
						})
				}
				createContext, err := json.Marshal(m)
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, "Failed to construct issue create context payload").SetInternal(err)
				}
				issueCreate := &api.IssueCreate{
					ProjectID:     repository.ProjectID,
					Name:          commit.Title,
					Type:          api.IssueDatabaseSchemaUpdate,
					Description:   commit.Message,
					AssigneeID:    api.SystemBotID,
					CreateContext: string(createContext),
				}

				issue, err := s.createIssue(ctx, issueCreate, api.SystemBotID)
				if err != nil {
					s.l.Warn("Failed to create update schema task for added repository file", zap.Error(err),
						zap.String("file", added))
					continue
				}

				createdMessageList = append(createdMessageList, fmt.Sprintf("Created issue %q on adding %s", issue.Name, added))

				// Create a project activity after successfully creating the issue as the result of the push event
				{
					bytes, err := json.Marshal(api.ActivityProjectRepositoryPushPayload{
						VCSPushEvent: vcsPushEvent,
						IssueID:      issue.ID,
						IssueName:    issue.Name,
					})
					if err != nil {
						return echo.NewHTTPError(http.StatusInternalServerError, "Failed to construct activity payload").SetInternal(err)
					}

					activityCreate := &api.ActivityCreate{
						CreatorID:   api.SystemBotID,
						ContainerID: repository.ProjectID,
						Type:        api.ActivityProjectRepositoryPush,
						Level:       api.ActivityInfo,
						Comment:     fmt.Sprintf("Created issue %q.", issue.Name),
						Payload:     string(bytes),
					}
					_, err = s.ActivityManager.CreateActivity(ctx, activityCreate, &ActivityMeta{})
					if err != nil {
						return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create project activity after creating issue from repository push event: %d", issue.ID)).SetInternal(err)
					}
				}
			}
		}

		return c.String(http.StatusOK, strings.Join(createdMessageList, "\n"))
	})
}
