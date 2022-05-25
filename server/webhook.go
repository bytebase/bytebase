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

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/vcs"
	"github.com/bytebase/bytebase/plugin/vcs/gitlab"
)

var (
	gitLabWebhookPath = "hook/gitlab"
)

func (s *Server) registerWebhookRoutes(g *echo.Group) {
	g.POST("/gitlab/:id", func(c echo.Context) error {
		ctx := c.Request().Context()
		var b []byte
		b, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Failed to read webhook request").SetInternal(err)
		}

		pushEvent := &gitlab.WebhookPushEvent{}
		if err := json.Unmarshal(b, pushEvent); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed push event").SetInternal(err)
		}

		// This shouldn't happen as we only setup webhook to receive push event, just in case.
		if pushEvent.ObjectKind != gitlab.WebhookPush {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid webhook event type, got %s, want push", pushEvent.ObjectKind))
		}

		webhookEndpointID := c.Param("id")
		repo, err := s.store.GetRepository(ctx, &api.RepositoryFind{WebhookEndpointID: &webhookEndpointID})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to respond webhook event for endpoint: %v", webhookEndpointID)).SetInternal(err)
		}
		if repo == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Endpoint not found: %v", webhookEndpointID))
		}

		if repo.VCS == nil {
			err := fmt.Errorf("VCS not found for ID: %v", repo.VCSID)
			return echo.NewHTTPError(http.StatusInternalServerError, err).SetInternal(err)
		}

		if c.Request().Header.Get("X-Gitlab-Token") != repo.WebhookSecretToken {
			return echo.NewHTTPError(http.StatusBadRequest, "Secret token mismatch")
		}

		if strconv.Itoa(pushEvent.Project.ID) != repo.ExternalID {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project mismatch, got %d, want %s", pushEvent.Project.ID, repo.ExternalID))
		}

		s.l.Debug("Processing gitlab webhook push event...",
			zap.String("project", repo.Project.Name),
		)

		createdMessageList := []string{}
		for _, commit := range pushEvent.CommitList {
			s.l.Debug("Processing commit...",
				zap.String("id", commit.ID),
				zap.String("title", commit.Title),
			)

			for _, added := range commit.AddedList {
				s.l.Debug("Processing added file...",
					zap.String("file", added),
				)

				if !strings.HasPrefix(added, repo.BaseDirectory) {
					s.l.Debug("Ignored committed file, not under base directory.", zap.String("file", added), zap.String("base_directory", repo.BaseDirectory))
					continue
				}

				createdTime, err := time.Parse(time.RFC3339, commit.Timestamp)
				if err != nil {
					s.l.Warn("Ignored committed file, failed to parse commit timestamp.", zap.String("file", added), zap.String("timestamp", commit.Timestamp), zap.Error(err))
				}

				// Ignore the schema file we auto generated to the repository.
				if isSkipGeneratedSchemaFile(repo, added, s.l) {
					s.l.Debug("Ignored generated latest schema file.", zap.String("file", added))
					continue
				}

				vcsPushEvent := vcs.PushEvent{
					VCSType:            repo.VCS.Type,
					BaseDirectory:      repo.BaseDirectory,
					Ref:                pushEvent.Ref,
					RepositoryID:       strconv.Itoa(pushEvent.Project.ID),
					RepositoryURL:      pushEvent.Project.WebURL,
					RepositoryFullPath: pushEvent.Project.FullPath,
					AuthorName:         pushEvent.AuthorName,
					FileCommit: vcs.FileCommit{
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
						ContainerID: repo.ProjectID,
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

				mi, err := db.ParseMigrationInfo(added, filepath.Join(repo.BaseDirectory, repo.FilePathTemplate))
				if err != nil {
					createIgnoredFileActivity(err)
					continue
				}

				// Retrieve sql by reading the file content
				content, err := vcs.Get(vcs.GitLabSelfHost, vcs.ProviderConfig{Logger: s.l}).ReadFileContent(
					ctx,
					common.OauthContext{
						ClientID:     repo.VCS.ApplicationID,
						ClientSecret: repo.VCS.Secret,
						AccessToken:  repo.AccessToken,
						RefreshToken: repo.RefreshToken,
						Refresher:    s.refreshToken(ctx, repo.ID),
					},
					repo.VCS.InstanceURL,
					repo.ExternalID,
					added,
					commit.ID,
				)
				if err != nil {
					createIgnoredFileActivity(err)
					continue
				}

				// Create schema update issue.
				var createContext string
				if repo.Project.TenantMode == api.TenantModeTenant {
					if !s.feature(api.FeatureMultiTenancy) {
						return echo.NewHTTPError(http.StatusForbidden, api.FeatureMultiTenancy.AccessErrorMessage())
					}
					createContext, err = s.createTenantSchemaUpdateIssue(ctx, repo, mi, vcsPushEvent, commit, added, content)
				} else {
					createContext, err = s.createSchemaUpdateIssue(ctx, repo, mi, vcsPushEvent, commit, added, content)
				}
				if err != nil {
					createIgnoredFileActivity(err)
					continue
				}

				issueType := api.IssueDatabaseSchemaUpdate
				if mi.Type == db.Data {
					issueType = api.IssueDatabaseDataUpdate
				}
				issueCreate := &api.IssueCreate{
					ProjectID:     repo.ProjectID,
					Name:          commit.Title,
					Type:          issueType,
					Description:   commit.Message,
					AssigneeID:    api.SystemBotID,
					CreateContext: createContext,
				}
				issue, err := s.createIssue(ctx, issueCreate, api.SystemBotID)
				if err != nil {
					errMsg := "Failed to create schema update issue"
					if issueType == api.IssueDatabaseDataUpdate {
						errMsg = "Failed to create data update issue"
					}
					return echo.NewHTTPError(http.StatusInternalServerError, errMsg).SetInternal(err)
				}

				createdMessageList = append(createdMessageList, fmt.Sprintf("Created issue %q on adding %s", issue.Name, added))

				// Create a project activity after successfully creating the issue as the result of the push event
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
					ContainerID: repo.ProjectID,
					Type:        api.ActivityProjectRepositoryPush,
					Level:       api.ActivityInfo,
					Comment:     fmt.Sprintf("Created issue %q.", issue.Name),
					Payload:     string(bytes),
				}
				if _, err = s.ActivityManager.CreateActivity(ctx, activityCreate, &ActivityMeta{}); err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create project activity after creating issue from repository push event: %d", issue.ID)).SetInternal(err)
				}
			}
		}

		if len(createdMessageList) == 0 {
			msg := "Ignored push event. No applicable file found in the commit list."
			s.l.Warn(msg,
				zap.String("project", repo.Project.Name),
			)
		}
		return c.String(http.StatusOK, strings.Join(createdMessageList, "\n"))
	})
}

func (s *Server) createSchemaUpdateIssue(ctx context.Context, repository *api.Repository, mi *db.MigrationInfo, vcsPushEvent vcs.PushEvent, commit gitlab.WebhookCommit, added string, statement string) (string, error) {
	// Find matching database list
	databaseFind := &api.DatabaseFind{
		ProjectID: &repository.ProjectID,
		Name:      &mi.Database,
	}
	databaseList, err := s.store.FindDatabase(ctx, databaseFind)
	if err != nil {
		return "", fmt.Errorf("failed to find database matching database %q referenced by the committed file", mi.Database)
	} else if len(databaseList) == 0 {
		return "", fmt.Errorf("project with ID[%d] does not own database %q referenced by the committed file", repository.ProjectID, mi.Database)
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
			// Environment name comparison is case insensitive
			if strings.EqualFold(database.Instance.Environment.Name, mi.Environment) {
				filteredDatabaseList = append(filteredDatabaseList, database)
			}
		}
		if len(filteredDatabaseList) == 0 {
			return "", fmt.Errorf("project does not contain committed file database %q for environment %q", mi.Database, mi.Environment)
		}
	} else {
		filteredDatabaseList = databaseList
	}

	// It could happen that for a particular environment a project contain 2 database with the same name.
	// We will emit warning in this case.
	var databaseListByEnv = map[int][]*api.Database{}
	for _, database := range filteredDatabaseList {
		databaseListByEnv[database.Instance.EnvironmentID] = append(databaseListByEnv[database.Instance.EnvironmentID], database)
	}
	var multipleDatabaseForSameEnv []string
	for environmentID, databaseList := range databaseListByEnv {
		if len(databaseList) > 1 {
			multipleDatabaseForSameEnv = append(multipleDatabaseForSameEnv, fmt.Sprintf("file %q database %q environment %d", added, mi.Database, environmentID))
		}
	}
	if len(multipleDatabaseForSameEnv) > 0 {
		return "", fmt.Errorf("ignored committed files with multiple ambiguous databases %s", strings.Join(multipleDatabaseForSameEnv, ", "))
	}

	// Compose the new issue
	m := &api.UpdateSchemaContext{
		MigrationType: mi.Type,
		VCSPushEvent:  &vcsPushEvent,
	}
	for _, database := range filteredDatabaseList {
		m.DetailList = append(m.DetailList,
			&api.UpdateSchemaDetail{
				DatabaseID: database.ID,
				Statement:  statement,
			})
	}
	createContext, err := json.Marshal(m)
	if err != nil {
		return "", fmt.Errorf("failed to construct issue create context payload, error %v", err)
	}
	return string(createContext), nil
}

func (s *Server) createTenantSchemaUpdateIssue(ctx context.Context, repository *api.Repository, mi *db.MigrationInfo, vcsPushEvent vcs.PushEvent, commit gitlab.WebhookCommit, added string, statement string) (string, error) {
	// We don't take environment for tenant mode project because the databases needing schema update are determined by database name and deployment configuration.
	if mi.Environment != "" {
		return "", fmt.Errorf("environment isn't accepted in schema update for tenant mode project")
	}
	m := &api.UpdateSchemaContext{
		MigrationType: mi.Type,
		VCSPushEvent:  &vcsPushEvent,
		DetailList: []*api.UpdateSchemaDetail{
			{
				DatabaseName: mi.Database,
				Statement:    statement,
			},
		},
	}
	createContext, err := json.Marshal(m)
	if err != nil {
		return "", fmt.Errorf("failed to construct issue create context payload, error %v", err)
	}
	return string(createContext), nil
}

// We may write back the latest schema file to the repository after migration and we need to ignore
// this file from the webhook push event.
func isSkipGeneratedSchemaFile(repository *api.Repository, added string, logger *zap.Logger) bool {
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
			logger.Warn("Invalid schema path template.", zap.String("schema_path_template",
				repository.SchemaPathTemplate),
				zap.Error(err),
			)
		}
		if myRegex.MatchString(added) {
			return true
		}
	}
	return false
}
