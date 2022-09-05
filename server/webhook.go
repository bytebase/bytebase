package server

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/parser"
	"github.com/bytebase/bytebase/plugin/vcs"
	"github.com/bytebase/bytebase/plugin/vcs/github"
	"github.com/bytebase/bytebase/plugin/vcs/gitlab"
)

var (
	gitlabWebhookPath = "hook/gitlab"
	githubWebhookPath = "hook/github"
)

func (s *Server) registerWebhookRoutes(g *echo.Group) {
	readFileContent := func(ctx context.Context, pushEvent vcs.PushEvent, webhookEndpointID, file string) (string, error) {
		// Retrieve the latest AccessToken and RefreshToken as the previous
		// ReadFileContent call may have updated the stored token pair. ReadFileContent
		// will fetch and store the new token pair if the existing token pair has
		// expired.
		repo, err := s.store.GetRepository(ctx, &api.RepositoryFind{WebhookEndpointID: &webhookEndpointID})
		if err != nil {
			return "", errors.Wrapf(err, "get repository by webhook endpoint %q", webhookEndpointID)
		} else if repo == nil {
			return "", errors.Wrapf(err, "repository not found by webhook endpoint %q", webhookEndpointID)
		}

		content, err := vcs.Get(repo.VCS.Type, vcs.ProviderConfig{}).ReadFileContent(
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
			file,
			pushEvent.FileCommit.ID,
		)
		if err != nil {
			return "", errors.Wrap(err, "read content")
		}
		return content, nil
	}

	g.POST("/gitlab/:id", func(c echo.Context) error {
		ctx := c.Request().Context()
		body, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Failed to read webhook request").SetInternal(err)
		}

		pushEvent := &gitlab.WebhookPushEvent{}
		if err := json.Unmarshal(body, pushEvent); err != nil {
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
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Webhook endpoint not found: %v", webhookEndpointID))
		}

		if repo.VCS == nil {
			err := errors.Errorf("VCS not found for ID: %v", repo.VCSID)
			return echo.NewHTTPError(http.StatusInternalServerError, err).SetInternal(err)
		}

		if c.Request().Header.Get("X-Gitlab-Token") != repo.WebhookSecretToken {
			return echo.NewHTTPError(http.StatusBadRequest, "Secret token mismatch")
		}

		if strconv.Itoa(pushEvent.Project.ID) != repo.ExternalID {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project mismatch, got %d, want %s", pushEvent.Project.ID, repo.ExternalID))
		}

		log.Debug("Processing GitLab webhook push event...",
			zap.String("project", repo.Project.Name),
		)

		var createdMessageList []string
		includeModified := repo.Project.SchemaMigrationType == api.ProjectSchemaMigrationTypeSDL
		distinctFileList := dedupMigrationFilesFromCommitList(pushEvent.CommitList, includeModified)
	fileList:
		for _, item := range distinctFileList {
			fileEscaped := common.EscapeForLogging(item.fileName)
			log.Debug("Processing file",
				zap.String("file", fileEscaped),
				zap.String("commit", common.EscapeForLogging(item.commit.ID)),
			)

			if !strings.HasPrefix(fileEscaped, repo.BaseDirectory) {
				log.Debug("Ignored file outside the base directory",
					zap.String("file", fileEscaped),
					zap.String("base_directory", repo.BaseDirectory),
				)
				continue
			}

			schemaInfo, err := parseSchemaFileInfo(repo.BaseDirectory, repo.SchemaPathTemplate, fileEscaped)
			if err != nil {
				log.Debug("Failed to parse schema file info",
					zap.String("file", fileEscaped),
					zap.Error(err),
				)
				continue
			}

			pushEvent := vcs.PushEvent{
				VCSType:            repo.VCS.Type,
				BaseDirectory:      repo.BaseDirectory,
				Ref:                pushEvent.Ref,
				RepositoryID:       strconv.Itoa(pushEvent.Project.ID),
				RepositoryURL:      pushEvent.Project.WebURL,
				RepositoryFullPath: pushEvent.Project.FullPath,
				AuthorName:         pushEvent.AuthorName,
				FileCommit: vcs.FileCommit{
					ID:          item.commit.ID,
					Title:       item.commit.Title,
					Message:     item.commit.Message,
					CreatedTs:   item.createdTime.Unix(),
					URL:         item.commit.URL,
					AuthorName:  item.commit.Author.Name,
					AuthorEmail: item.commit.Author.Email,
					Added:       fileEscaped,
				},
			}

			var migrationInfo *db.MigrationInfo
			var updateSchemaDetails []*api.UpdateSchemaDetail
			if repo.Project.SchemaMigrationType == api.ProjectSchemaMigrationTypeSDL {
				if schemaInfo == nil {
					log.Debug("Ignored non-schema file for SDL",
						zap.String("file", fileEscaped),
					)
					continue
				}

				dbName := schemaInfo["DB_NAME"]
				if dbName == "" {
					log.Debug("Ignored schema file without a database name",
						zap.String("file", fileEscaped),
					)
					continue
				}

				envName := schemaInfo["ENV_NAME"]
				databases, err := s.findProjectDatabases(ctx, repo.ProjectID, dbName, envName)
				if err != nil {
					s.createIgnoredFileActivity(
						ctx,
						repo.ProjectID,
						pushEvent,
						fileEscaped,
						errors.Wrap(err, "Failed to find project databases"),
					)
					continue
				}

				content, err := readFileContent(ctx, pushEvent, webhookEndpointID, fileEscaped)
				if err != nil {
					s.createIgnoredFileActivity(
						ctx,
						repo.ProjectID,
						pushEvent,
						fileEscaped,
						errors.Wrap(err, "Failed to read file content"),
					)
					continue
				}

				for _, database := range databases {
					diff, err := s.computeDatabaseSchemaDiff(ctx, database, content)
					if err != nil {
						s.createIgnoredFileActivity(
							ctx,
							repo.ProjectID,
							pushEvent,
							fileEscaped,
							errors.Wrap(err, "Failed to compute database schema diff"),
						)
						continue fileList
					}

					updateSchemaDetails = append(updateSchemaDetails,
						&api.UpdateSchemaDetail{
							DatabaseID: database.ID,
							Statement:  diff,
						},
					)
				}

				migrationInfo = &db.MigrationInfo{
					Version:     common.DefaultMigrationVersion(),
					Namespace:   dbName,
					Database:    dbName,
					Environment: envName,
					Source:      db.VCS,
					Type:        db.Migrate,
					Description: "Apply schema diff",
				}

				added := strings.NewReplacer(
					"{{ENV_NAME}}", envName,
					"{{DB_NAME}}", dbName,
					"{{VERSION}}", migrationInfo.Version,
					"{{TYPE}}", strings.ToLower(string(migrationInfo.Type)),
					"{{DESCRIPTION}}", strings.ReplaceAll(migrationInfo.Description, " ", "_"),
				).Replace(repo.FilePathTemplate)
				// NOTE: We do not want to use filepath.Join here because we always need "/" as the path separator.
				pushEvent.FileCommit.Added = path.Join(repo.BaseDirectory, added)
			} else {
				if item.itemType != fileItemTypeAdded || schemaInfo != nil {
					log.Debug("Ignored non-added or schema file for non-SDL",
						zap.String("file", fileEscaped),
						zap.String("type", string(item.itemType)),
					)
					continue
				}

				// NOTE: We do not want to use filepath.Join here because we always need "/" as the path separator.
				migrationInfo, err = db.ParseMigrationInfo(fileEscaped, path.Join(repo.BaseDirectory, repo.FilePathTemplate))
				if err != nil {
					s.createIgnoredFileActivity(
						ctx,
						repo.ProjectID,
						pushEvent,
						fileEscaped,
						errors.Wrap(err, "Failed to parse migration info"),
					)
					continue
				}

				databases, err := s.findProjectDatabases(ctx, repo.ProjectID, migrationInfo.Database, migrationInfo.Environment)
				if err != nil {
					s.createIgnoredFileActivity(
						ctx,
						repo.ProjectID,
						pushEvent,
						fileEscaped,
						errors.Wrap(err, "Failed to find project databases"),
					)
					continue
				}

				content, err := readFileContent(ctx, pushEvent, webhookEndpointID, fileEscaped)
				if err != nil {
					s.createIgnoredFileActivity(
						ctx,
						repo.ProjectID,
						pushEvent,
						fileEscaped,
						errors.Wrap(err, "Failed to read file content"),
					)
					continue
				}

				for _, database := range databases {
					updateSchemaDetails = append(updateSchemaDetails,
						&api.UpdateSchemaDetail{
							DatabaseID: database.ID,
							Statement:  content,
						},
					)
				}
			}

			createdMessage, created, httpErr := s.createIssueFromPushEvent(ctx, pushEvent, repo, fileEscaped, migrationInfo, updateSchemaDetails)
			if httpErr != nil {
				return httpErr
			}

			if created {
				createdMessageList = append(createdMessageList, createdMessage)
			}
		}

		if len(createdMessageList) == 0 {
			log.Warn("Ignored push event because no applicable file found in the commit list",
				zap.String("project", repo.Project.Name),
			)
		}
		return c.String(http.StatusOK, strings.Join(createdMessageList, "\n"))
	})
	g.POST("/github/:id", func(c echo.Context) error {
		ctx := c.Request().Context()

		// This shouldn't happen as we only setup webhook to receive push event, just in case.
		eventType := github.WebhookType(c.Request().Header.Get("X-GitHub-Event"))

		// https://docs.github.com/en/developers/webhooks-and-events/webhooks/about-webhooks#ping-event
		// When we create a new webhook, GitHub will send us a simple ping event to let us know we've set up the webhook correctly.
		// We respond to this event so as not to mislead users.
		if eventType == github.WebhookPing {
			return c.String(http.StatusOK, "OK")
		}

		if eventType != github.WebhookPush {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid webhook event type, got %s, want %s", eventType, github.WebhookPush))
		}

		webhookEndpointID := c.Param("id")
		repo, err := s.store.GetRepository(ctx, &api.RepositoryFind{WebhookEndpointID: &webhookEndpointID})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to respond webhook event for endpoint: %v", webhookEndpointID)).SetInternal(err)
		}
		if repo == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Webhook endpoint not found: %v", webhookEndpointID))
		}

		if repo.VCS == nil {
			err := errors.Errorf("VCS not found for ID: %v", repo.VCSID)
			return echo.NewHTTPError(http.StatusInternalServerError, err).SetInternal(err)
		}

		body, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Failed to read webhook request").SetInternal(err)
		}

		// Validate the request body first because there is no point in unmarshalling
		// the request body if the signature doesn't match.
		validated, err := validateGitHubWebhookSignature256(c.Request().Header.Get("X-Hub-Signature-256"), repo.WebhookSecretToken, body)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to validate GitHub webhook signature").SetInternal(err)
		}
		if !validated {
			return echo.NewHTTPError(http.StatusBadRequest, "Mismatched payload signature")
		}

		var pushEvent github.WebhookPushEvent
		if err := json.Unmarshal(body, &pushEvent); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed push event").SetInternal(err)
		}

		if branch, err := parseBranchNameFromGitHubRefs(pushEvent.Ref); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid ref").SetInternal(err)
		} else if branch != repo.BranchFilter {
			return c.String(http.StatusOK, "")
		}

		if pushEvent.Repository.FullName != repo.ExternalID {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project mismatch, got %s, want %s", pushEvent.Repository.FullName, repo.ExternalID))
		}

		log.Debug("Processing GitHub webhook push event...",
			zap.String("project", repo.Project.Name),
		)

		var createdMessageList []string
		for _, commit := range pushEvent.Commits {
			// The Distinct is false if the commit is superseded by a later commit.
			if !commit.Distinct {
				continue
			}

			for _, added := range commit.Added {
				// Per Git convention, the message title and body are separated by two new line characters.
				messages := strings.SplitN(commit.Message, "\n\n", 2)
				messageTitle := messages[0]

				createdMessage, created, httpErr := s.createIssueFromPushEventLegacy(
					ctx,
					repo,
					vcs.PushEvent{
						VCSType:            repo.VCS.Type,
						BaseDirectory:      repo.BaseDirectory,
						Ref:                pushEvent.Ref,
						RepositoryID:       strconv.Itoa(pushEvent.Repository.ID),
						RepositoryURL:      pushEvent.Repository.HTMLURL,
						RepositoryFullPath: pushEvent.Repository.FullName,
						AuthorName:         pushEvent.Sender.Login,
						FileCommit: vcs.FileCommit{
							ID:          commit.ID,
							Title:       messageTitle,
							Message:     commit.Message,
							CreatedTs:   commit.Timestamp.Unix(),
							URL:         commit.URL,
							AuthorName:  commit.Author.Name,
							AuthorEmail: commit.Author.Email,
							Added:       common.EscapeForLogging(added),
						},
					},
					added,
					webhookEndpointID,
				)
				if httpErr != nil {
					return httpErr
				}

				if created {
					createdMessageList = append(createdMessageList, createdMessage)
				}
			}
		}

		if len(createdMessageList) == 0 {
			log.Warn("Ignored push event. No applicable file found in the commit list.",
				zap.String("project", repo.Project.Name),
			)
		}
		return c.String(http.StatusOK, strings.Join(createdMessageList, "\n"))
	})
}

// validateGitHubWebhookSignature256 returns true if the signature matches the
// HMAC hex digested SHA256 hash of the body using the given key.
func validateGitHubWebhookSignature256(signature, key string, body []byte) (bool, error) {
	signature = strings.TrimPrefix(signature, "sha256=")
	m := hmac.New(sha256.New, []byte(key))
	if _, err := m.Write(body); err != nil {
		return false, err
	}
	got := hex.EncodeToString(m.Sum(nil))

	// NOTE: Use constant time string comparison helps mitigate certain timing
	// attacks against regular equality operators, see
	// https://docs.github.com/en/developers/webhooks-and-events/webhooks/securing-your-webhooks#validating-payloads-from-github
	return subtle.ConstantTimeCompare([]byte(signature), []byte(got)) == 1, nil
}

// parseBranchNameFromGitHubRefs is for GitHub.
// GitLab webhook has the option to listen to push events to a certain branch while GitHub doesn't. So we need to manually parse the branch name from the refs field in the request.
// https://docs.github.com/en/rest/git/refs
func parseBranchNameFromGitHubRefs(ref string) (string, error) {
	expectedPrefix := "refs/heads/"
	if !strings.HasPrefix(ref, expectedPrefix) || len(expectedPrefix) == len(ref) {
		log.Debug("ref is not prefix with expected prefix", zap.String("escaped ref", common.EscapeForLogging(ref)), zap.String("expected prefix", expectedPrefix))
		return ref, errors.Errorf("unexpected ref name %q without prefix %q", ref, expectedPrefix)
	}
	return ref[len(expectedPrefix):], nil
}

// fileItemType is the type of a file item.
type fileItemType string

// The list of file item types.
const (
	fileItemTypeAdded    fileItemType = "added"
	fileItemTypeModified fileItemType = "modified"
)

// We are observing the push webhook event so that we will receive the event either when:
// 1. A commit is directly pushed to a branch.
// 2. One or more commits are merged to a branch.
//
// There is a complication to deal with the 2nd type. A typical workflow is a developer first
// commits the migration file to the feature branch, and at a later point, she creates a merge
// request to merge the commit to the main branch. Even the developer only creates a single commit
// on the feature branch, that merge request may contain multiple commits (unless both squash and fast-forward merge are used):
// 1. The original commit on the feature branch.
// 2. The merge request commit.
//
// And both commits would include that added migration file. Since we create an issue per migration file,
// we need to filter the commit list to prevent creating a duplicated issue. GitLab has a limitation to distinguish
// whether the commit is a merge commit (https://gitlab.com/gitlab-org/gitlab/-/issues/30914), so we need to dedup
// ourselves. Below is the filtering algorithm:
//  1. If we observe the same migration file multiple times, then we should use the latest migration file. This does not matter
//     for change-based migration since a developer would always create different migration file with incremental names, while it
//     will be important for the state-based migration, since the file name is always the same and we need to use the latest snapshot.
//  2. Maintain the relative commit order between different migration files. If migration file A happens before migration file B,
//     then we should create an issue for migration file A first.
type distinctFileItem struct {
	createdTime time.Time
	commit      gitlab.WebhookCommit
	fileName    string
	itemType    fileItemType
}

func dedupMigrationFilesFromCommitList(commitList []gitlab.WebhookCommit, includeModified bool) []distinctFileItem {
	// Use list instead of map because we need to maintain the relative commit order in the source branch.
	var distinctFileList []distinctFileItem
	for _, commit := range commitList {
		log.Debug("Pre-processing commit to dedup migration files...",
			zap.String("id", common.EscapeForLogging(commit.ID)),
			zap.String("title", common.EscapeForLogging(commit.Title)),
		)

		createdTime, err := time.Parse(time.RFC3339, commit.Timestamp)
		if err != nil {
			log.Warn("Ignored commit, failed to parse commit timestamp.", zap.String("commit", common.EscapeForLogging(commit.ID)), zap.String("timestamp", common.EscapeForLogging(commit.Timestamp)), zap.Error(err))
		}

		addDistinctFile := func(fileName string, itemType fileItemType) {
			item := distinctFileItem{
				createdTime: createdTime,
				commit:      commit,
				fileName:    fileName,
				itemType:    itemType,
			}
			for i, file := range distinctFileList {
				// For the migration file with the same name, keep the one from the latest commit
				if item.fileName == file.fileName {
					if file.createdTime.Before(createdTime) {
						distinctFileList[i] = item
					}
					return
				}
			}
			distinctFileList = append(distinctFileList, item)
		}

		for _, added := range commit.AddedList {
			addDistinctFile(added, fileItemTypeAdded)
		}

		if includeModified {
			for _, modified := range commit.ModifiedList {
				addDistinctFile(modified, fileItemTypeModified)
			}
		}
	}
	return distinctFileList
}

// findProjectDatabases finds the list of databases with given name in the
// project. If the `envName` is not empty, it will be used as a filter condition
// for the result list.
func (s *Server) findProjectDatabases(ctx context.Context, projectID int, dbName, envName string) ([]*api.Database, error) {
	// Retrieve the current schema from the database
	foundDatabases, err := s.store.FindDatabase(ctx,
		&api.DatabaseFind{
			ProjectID: &projectID,
			Name:      &dbName,
		},
	)
	if err != nil {
		return nil, errors.Wrap(err, "find database")
	} else if len(foundDatabases) == 0 {
		return nil, errors.Errorf("project %d does not have database %q", projectID, dbName)
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
	var filteredDatabases []*api.Database
	if envName != "" {
		for _, database := range foundDatabases {
			// Environment name comparison is case insensitive
			if strings.EqualFold(database.Instance.Environment.Name, envName) {
				filteredDatabases = append(filteredDatabases, database)
			}
		}
		if len(filteredDatabases) == 0 {
			return nil, errors.Errorf("project %d does not have database %q for environment %q", projectID, dbName, envName)
		}
	} else {
		filteredDatabases = foundDatabases
	}

	// In case there are databases with identical name in a project for the same environment.
	marked := make(map[int]struct{})
	for _, database := range filteredDatabases {
		if _, ok := marked[database.Instance.EnvironmentID]; ok {
			return nil, errors.Errorf("project %d has multiple databases %q for environment %q", projectID, dbName, envName)
		}
		marked[database.Instance.EnvironmentID] = struct{}{}
	}
	return filteredDatabases, nil
}

func (s *Server) createSchemaUpdateIssue(ctx context.Context, repository *api.Repository, mi *db.MigrationInfo, vcsPushEvent vcs.PushEvent, added string, statement string) (string, error) {
	// Find matching database list
	databaseFind := &api.DatabaseFind{
		ProjectID: &repository.ProjectID,
		Name:      &mi.Database,
	}
	databaseList, err := s.store.FindDatabase(ctx, databaseFind)
	if err != nil {
		return "", errors.Errorf("failed to find database matching database %q referenced by the committed file", mi.Database)
	} else if len(databaseList) == 0 {
		return "", errors.Errorf("project with ID %d does not own database %q referenced by the committed file", repository.ProjectID, mi.Database)
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
			return "", errors.Errorf("project does not contain committed file database %q for environment %q", mi.Database, mi.Environment)
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
		return "", errors.Errorf("ignored committed files with multiple ambiguous databases %s", strings.Join(multipleDatabaseForSameEnv, ", "))
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
		return "", errors.Wrap(err, "failed to construct issue create context payload")
	}
	return string(createContext), nil
}

func createTenantSchemaUpdateIssue(mi *db.MigrationInfo, vcsPushEvent vcs.PushEvent, statement string) (string, error) {
	// We don't take environment for tenant mode project because the databases needing schema update are determined by database name and deployment configuration.
	if mi.Environment != "" {
		return "", errors.Errorf("environment isn't accepted in schema update for tenant mode project")
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
		return "", errors.Wrap(err, "failed to construct issue create context payload")
	}
	return string(createContext), nil
}

// createIgnoredFileActivity creates a warning project activity for the ignored file with given error.
func (s *Server) createIgnoredFileActivity(ctx context.Context, projectID int, pushEvent vcs.PushEvent, file string, err error) {
	log.Warn("Ignored file",
		zap.String("file", file),
		zap.Error(err),
	)

	payload, marshalErr := json.Marshal(
		api.ActivityProjectRepositoryPushPayload{
			VCSPushEvent: pushEvent,
		},
	)
	if marshalErr != nil {
		log.Warn("Failed to construct project activity payload for the ignored repository file",
			zap.Error(marshalErr),
		)
		return
	}

	activityCreate := &api.ActivityCreate{
		CreatorID:   api.SystemBotID,
		ContainerID: projectID,
		Type:        api.ActivityProjectRepositoryPush,
		Level:       api.ActivityWarn,
		Comment:     fmt.Sprintf("Ignored file %q, %v.", file, err),
		Payload:     string(payload),
	}
	if _, err = s.ActivityManager.CreateActivity(ctx, activityCreate, &ActivityMeta{}); err != nil {
		log.Warn("Failed to create project activity for the ignored repository file",
			zap.Error(err),
		)
	}
}

// createIssueFromPushEvent attempts to create a new issue for the given file of
// the push event. It returns "created=true" when a new issue has been created,
// along with the creation message to be presented in the UI. An *echo.HTTPError
// is returned in case of the error during the process.
func (s *Server) createIssueFromPushEvent(ctx context.Context, pushEvent vcs.PushEvent, repo *api.Repository, file string, mi *db.MigrationInfo, updateSchemaDetails []*api.UpdateSchemaDetail) (message string, created bool, err error) {
	if len(updateSchemaDetails) == 0 {
		return "", false, nil
	}

	// Create schema update issue
	creatorID := api.SystemBotID
	if pushEvent.FileCommit.AuthorEmail != "" {
		committerPrincipal, err := s.store.GetPrincipalByEmail(ctx, pushEvent.FileCommit.AuthorEmail)
		if err != nil {
			log.Error("Failed to find the principal with committer email",
				zap.String("email", common.EscapeForLogging(pushEvent.FileCommit.AuthorEmail)),
				zap.Error(err),
			)
		}
		if committerPrincipal == nil {
			log.Debug("Failed to find the principal with committer email, use system bot instead",
				zap.String("email", common.EscapeForLogging(pushEvent.FileCommit.AuthorEmail)),
			)
		} else {
			creatorID = committerPrincipal.ID
		}
	}

	if repo.Project.TenantMode == api.TenantModeTenant {
		if !s.feature(api.FeatureMultiTenancy) {
			return "", false, echo.NewHTTPError(http.StatusForbidden, api.FeatureMultiTenancy.AccessErrorMessage())
		}

		// We don't take environment for tenant mode project because the databases
		// needing schema update are determined by database name and deployment
		// configuration.
		if mi.Environment != "" {
			return "", false, echo.NewHTTPError(http.StatusInternalServerError, "Environment isn't accepted in schema update for tenant mode project")
		}
	}

	createContext, err := json.Marshal(
		&api.UpdateSchemaContext{
			MigrationType: db.Migrate,
			VCSPushEvent:  &pushEvent,
			DetailList:    updateSchemaDetails,
		},
	)
	if err != nil {
		return "", false, echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal update schema context").SetInternal(err)
	}

	issueType := api.IssueDatabaseSchemaUpdate
	if mi.Type == db.Data {
		issueType = api.IssueDatabaseDataUpdate
	}
	issueCreate := &api.IssueCreate{
		ProjectID:     repo.ProjectID,
		Name:          fmt.Sprintf("%s by %s", mi.Description, strings.TrimPrefix(file, repo.BaseDirectory+"/")),
		Type:          issueType,
		Description:   pushEvent.FileCommit.Message,
		AssigneeID:    api.SystemBotID,
		CreateContext: string(createContext),
	}
	issue, err := s.createIssue(ctx, issueCreate, creatorID)
	if err != nil {
		errMsg := "Failed to create schema update issue"
		if issueType == api.IssueDatabaseDataUpdate {
			errMsg = "Failed to create data update issue"
		}
		return "", false, echo.NewHTTPError(http.StatusInternalServerError, errMsg).SetInternal(err)
	}

	// Create a project activity after successfully creating the issue as the result of the push event
	payload, err := json.Marshal(
		api.ActivityProjectRepositoryPushPayload{
			VCSPushEvent: pushEvent,
			IssueID:      issue.ID,
			IssueName:    issue.Name,
		},
	)
	if err != nil {
		return "", false, echo.NewHTTPError(http.StatusInternalServerError, "Failed to construct activity payload").SetInternal(err)
	}

	activityCreate := &api.ActivityCreate{
		CreatorID:   creatorID,
		ContainerID: repo.ProjectID,
		Type:        api.ActivityProjectRepositoryPush,
		Level:       api.ActivityInfo,
		Comment:     fmt.Sprintf("Created issue %q.", issue.Name),
		Payload:     string(payload),
	}
	if _, err = s.ActivityManager.CreateActivity(ctx, activityCreate, &ActivityMeta{}); err != nil {
		return "", false, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create project activity after creating issue from repository push event: %d", issue.ID)).SetInternal(err)
	}

	return fmt.Sprintf("Created issue %q on adding %s", issue.Name, file), true, nil
}

// Deprecated: Use createIssueFromPushEvent instead.
//
// TODO(jc): Delete this method once refactored webhook handler logic for GitHub.
func (s *Server) createIssueFromPushEventLegacy(ctx context.Context, repo *api.Repository, pushEvent vcs.PushEvent, file, webhookEndpointID string) (message string, created bool, _ error) {
	fileEscaped := common.EscapeForLogging(file)
	log.Debug("Processing added file...",
		zap.String("file", fileEscaped),
		zap.String("commit", common.EscapeForLogging(pushEvent.FileCommit.ID)),
	)

	if !strings.HasPrefix(fileEscaped, repo.BaseDirectory) {
		log.Debug("Ignored committed file, not under base directory.",
			zap.String("file", fileEscaped),
			zap.String("base_directory", repo.BaseDirectory),
		)
		return "", false, nil
	}

	// Ignore the schema file we auto generated to the repository.
	if isSkipGeneratedSchemaFile(repo, fileEscaped) {
		log.Debug("Ignored generated latest schema file.",
			zap.String("file", fileEscaped),
		)
		return "", false, nil
	}

	// Create a WARNING project activity if committed file is ignored
	var createIgnoredFileActivity = func(err error) {
		log.Warn("Ignored committed file",
			zap.String("file", fileEscaped),
			zap.Error(err),
		)
		bytes, marshalErr := json.Marshal(
			api.ActivityProjectRepositoryPushPayload{
				VCSPushEvent: pushEvent,
			},
		)
		if marshalErr != nil {
			log.Warn("Failed to construct project activity payload to record ignored repository committed file",
				zap.Error(marshalErr),
			)
			return
		}

		activityCreate := &api.ActivityCreate{
			CreatorID:   api.SystemBotID,
			ContainerID: repo.ProjectID,
			Type:        api.ActivityProjectRepositoryPush,
			Level:       api.ActivityWarn,
			Comment:     fmt.Sprintf("Ignored committed file, %s.", err.Error()),
			Payload:     string(bytes),
		}
		_, err = s.ActivityManager.CreateActivity(ctx, activityCreate, &ActivityMeta{})
		if err != nil {
			log.Warn("Failed to create project activity to record ignored repository committed file",
				zap.Error(err),
			)
		}
	}

	// NOTE: We do not want to use filepath.Join here because we always need "/" as the path separator.
	mi, err := db.ParseMigrationInfo(fileEscaped, path.Join(repo.BaseDirectory, repo.FilePathTemplate))
	if err != nil {
		createIgnoredFileActivity(err)
		return "", false, nil
	}

	// Retrieve the latest AccessToken and RefreshToken as the previous
	// ReadFileContent call may have updated the stored token pair. ReadFileContent
	// will fetch and store the new token pair if the existing token pair has
	// expired.
	repo2, err := s.store.GetRepository(ctx, &api.RepositoryFind{WebhookEndpointID: &webhookEndpointID})
	if err != nil {
		return "", false, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to respond webhook event for endpoint: %v", webhookEndpointID)).SetInternal(err)
	}
	if repo2 == nil {
		return "", false, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Webhook endpoint not found: %v", webhookEndpointID))
	}

	// Retrieve migration SQL script by reading the file content
	content, err := vcs.Get(repo2.VCS.Type, vcs.ProviderConfig{}).ReadFileContent(
		ctx,
		common.OauthContext{
			ClientID:     repo2.VCS.ApplicationID,
			ClientSecret: repo2.VCS.Secret,
			AccessToken:  repo2.AccessToken,
			RefreshToken: repo2.RefreshToken,
			Refresher:    s.refreshToken(ctx, repo2.ID),
		},
		repo2.VCS.InstanceURL,
		repo2.ExternalID,
		fileEscaped,
		pushEvent.FileCommit.ID,
	)
	if err != nil {
		createIgnoredFileActivity(err)
		return "", false, nil
	}

	// Create schema update issue.
	creatorID := api.SystemBotID
	if pushEvent.FileCommit.AuthorEmail != "" {
		committerPrincipal, err := s.store.GetPrincipalByEmail(ctx, pushEvent.FileCommit.AuthorEmail)
		if err != nil {
			log.Error("failed to find the principal with committer email",
				zap.String("email", common.EscapeForLogging(pushEvent.FileCommit.AuthorEmail)),
				zap.Error(err),
			)
		}
		if committerPrincipal == nil {
			log.Debug("cannot find the principal with committer email, use system bot instead",
				zap.String("email", common.EscapeForLogging(pushEvent.FileCommit.AuthorEmail)),
			)
		} else {
			creatorID = committerPrincipal.ID
		}
	}

	var createContext string
	if repo.Project.TenantMode == api.TenantModeTenant {
		if !s.feature(api.FeatureMultiTenancy) {
			return "", false, echo.NewHTTPError(http.StatusForbidden, api.FeatureMultiTenancy.AccessErrorMessage())
		}
		createContext, err = createTenantSchemaUpdateIssue(mi, pushEvent, content)
	} else {
		createContext, err = s.createSchemaUpdateIssue(ctx, repo, mi, pushEvent, fileEscaped, content)
	}
	if err != nil {
		createIgnoredFileActivity(err)
		return "", false, nil
	}

	issueType := api.IssueDatabaseSchemaUpdate
	if mi.Type == db.Data {
		issueType = api.IssueDatabaseDataUpdate
	}
	issueCreate := &api.IssueCreate{
		ProjectID:     repo.ProjectID,
		Name:          fmt.Sprintf("%s by %s", mi.Description, strings.TrimPrefix(file, repo.BaseDirectory+"/")),
		Type:          issueType,
		Description:   pushEvent.FileCommit.Message,
		AssigneeID:    api.SystemBotID,
		CreateContext: createContext,
	}
	issue, err := s.createIssue(ctx, issueCreate, creatorID)
	if err != nil {
		errMsg := "Failed to create schema update issue"
		if issueType == api.IssueDatabaseDataUpdate {
			errMsg = "Failed to create data update issue"
		}
		return "", false, echo.NewHTTPError(http.StatusInternalServerError, errMsg).SetInternal(err)
	}

	// Create a project activity after successfully creating the issue as the result of the push event
	bytes, err := json.Marshal(
		api.ActivityProjectRepositoryPushPayload{
			VCSPushEvent: pushEvent,
			IssueID:      issue.ID,
			IssueName:    issue.Name,
		},
	)
	if err != nil {
		return "", false, echo.NewHTTPError(http.StatusInternalServerError, "Failed to construct activity payload").SetInternal(err)
	}

	activityCreate := &api.ActivityCreate{
		CreatorID:   creatorID,
		ContainerID: repo.ProjectID,
		Type:        api.ActivityProjectRepositoryPush,
		Level:       api.ActivityInfo,
		Comment:     fmt.Sprintf("Created issue %q.", issue.Name),
		Payload:     string(bytes),
	}
	if _, err = s.ActivityManager.CreateActivity(ctx, activityCreate, &ActivityMeta{}); err != nil {
		return "", false, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create project activity after creating issue from repository push event: %d", issue.ID)).SetInternal(err)
	}

	return fmt.Sprintf("Created issue %q on adding %s", issue.Name, fileEscaped), true, nil
}

// We may write back the latest schema file to the repository after migration and we need to ignore
// this file from the webhook push event.
//
// Deprecated: Use parseSchemaFileInfo instead.
func isSkipGeneratedSchemaFile(repository *api.Repository, added string) bool {
	if repository.SchemaPathTemplate != "" {
		placeholderList := []string{
			"ENV_NAME",
			"DB_NAME",
		}

		// Escape "." characters to match literals instead of using it as a wildcard.
		schemaFilePathRegex := strings.ReplaceAll(repository.SchemaPathTemplate, ".", `\.`)
		for _, placeholder := range placeholderList {
			schemaFilePathRegex = strings.ReplaceAll(schemaFilePathRegex, fmt.Sprintf("{{%s}}", placeholder), fmt.Sprintf("(?P<%s>[a-zA-Z0-9+-=/_#?!$. ]+)", placeholder))
		}
		myRegex, err := regexp.Compile(schemaFilePathRegex)
		if err != nil {
			log.Warn("Invalid schema path template.", zap.String("schema_path_template",
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

// parseSchemaFileInfo attempts to parse the given schema file path to extract
// the schema file info. It returns (nil, nil) if it doesn't looks like a schema
// file path.
//
// The possible keys for the returned map are: "ENV_NAME", "DB_NAME".
func parseSchemaFileInfo(baseDirectory, schemaPathTemplate, file string) (map[string]string, error) {
	if schemaPathTemplate == "" {
		return nil, nil
	}

	// Escape "." characters to match literals instead of using it as a wildcard.
	schemaFilePathRegex := strings.ReplaceAll(schemaPathTemplate, ".", `\.`)

	placeholders := []string{
		"ENV_NAME",
		"DB_NAME",
	}
	for _, placeholder := range placeholders {
		schemaFilePathRegex = strings.ReplaceAll(schemaFilePathRegex, fmt.Sprintf("{{%s}}", placeholder), fmt.Sprintf("(?P<%s>[a-zA-Z0-9+-=/_#?!$. ]+)", placeholder))
	}

	// NOTE: We do not want to use filepath.Join here because we always need "/" as the path separator.
	re, err := regexp.Compile(path.Join(baseDirectory, schemaFilePathRegex))
	if err != nil {
		return nil, errors.Wrap(err, "compile schema file path regex")
	}
	match := re.FindStringSubmatch(file)
	if len(match) == 0 {
		return nil, nil
	}

	info := make(map[string]string)
	// Skip the first item because it is always the empty string, see docstring of
	// the SubexpNames() method.
	for i, name := range re.SubexpNames()[1:] {
		info[name] = match[i+1]
	}
	return info, nil
}

// computeDatabaseSchemaDiff computes the diff between current database schema
// and the given schema. It returns an empty string if there is no applicable
// diff.
func (s *Server) computeDatabaseSchemaDiff(ctx context.Context, database *api.Database, newSchemaStr string) (string, error) {
	driver, err := s.getAdminDatabaseDriver(ctx, database.Instance, database.Name)
	if err != nil {
		return "", errors.Wrap(err, "get admin driver")
	}
	defer func() {
		_ = driver.Close(ctx)
	}()

	var schema bytes.Buffer
	_, err = driver.Dump(ctx, database.Name, &schema, true /* schemaOnly */)
	if err != nil {
		return "", errors.Wrap(err, "dump old schema")
	}

	var engine parser.EngineType
	switch database.Instance.Engine {
	case db.Postgres:
		engine = parser.Postgres
	case db.MySQL:
		engine = parser.MySQL
	default:
		return "", errors.Errorf("unsupported database engine %q", database.Instance.Engine)
	}
	oldSchema, err := parser.Parse(engine, parser.ParseContext{}, schema.String())
	if err != nil {
		return "", errors.Wrap(err, "parse old schema")
	}

	newSchema, err := parser.Parse(engine, parser.ParseContext{}, newSchemaStr)
	if err != nil {
		return "", errors.Wrap(err, "parse new schema")
	}

	diff, err := parser.SchemaDiff(oldSchema, newSchema)
	if err != nil {
		return "", errors.New("compute schema diff")
	}
	return diff, nil
}
