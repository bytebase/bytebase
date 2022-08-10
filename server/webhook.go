package server

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
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
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/vcs"
	"github.com/bytebase/bytebase/plugin/vcs/github"
	"github.com/bytebase/bytebase/plugin/vcs/gitlab"
)

var (
	gitlabWebhookPath = "hook/gitlab"
	githubWebhookPath = "hook/github"
)

func (s *Server) registerWebhookRoutes(g *echo.Group) {
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
			err := fmt.Errorf("VCS not found for ID: %v", repo.VCSID)
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

		distinctFileList := dedupMigrationFilesFromCommitList(pushEvent.CommitList)
		var createdMessageList []string
		for _, item := range distinctFileList {
			createdMessage, created, httpErr := s.createIssueFromPushEvent(
				ctx,
				repo,
				vcs.PushEvent{
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
						Added:       common.EscapeForLogging(item.fileName),
					},
				},
				item.fileName,
				webhookEndpointID,
			)
			if httpErr != nil {
				return httpErr
			}

			if created {
				createdMessageList = append(createdMessageList, createdMessage)
			}
		}

		if len(createdMessageList) == 0 {
			msg := "Ignored push event. No applicable file found in the commit list."
			log.Warn(msg,
				zap.String("project", repo.Project.Name),
			)
		}
		return c.String(http.StatusOK, strings.Join(createdMessageList, "\n"))
	})
	g.POST("/github/:id", func(c echo.Context) error {
		ctx := c.Request().Context()

		// This shouldn't happen as we only setup webhook to receive push event, just in case.
		eventType := github.WebhookType(c.Request().Header.Get("X-GitHub-Event"))
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
			err := fmt.Errorf("VCS not found for ID: %v", repo.VCSID)
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

				createdMessage, created, httpErr := s.createIssueFromPushEvent(
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
// 1. If we observe the same migration file multiple times, then we should use the latest migration file. This does not matter
//    for change-based migration since a developer would always create different migration file with incremental names, while it
//    will be important for the state-based migration, since the file name is always the same and we need to use the latest snapshot.
// 2. Maintain the relative commit order between different migration files. If migration file A happens before migration file B,
//    then we should create an issue for migration file A first.
type distinctFileItem struct {
	createdTime time.Time
	commit      gitlab.WebhookCommit
	fileName    string
}

func dedupMigrationFilesFromCommitList(commitList []gitlab.WebhookCommit) []distinctFileItem {
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

		for _, added := range commit.AddedList {
			isNew := true
			item := distinctFileItem{
				createdTime: createdTime,
				commit:      commit,
				fileName:    added,
			}
			for i, file := range distinctFileList {
				// For the migration file with the same name, keep the one from the latest commit
				if added == file.fileName {
					isNew = false
					if file.createdTime.Before(createdTime) {
						distinctFileList[i] = item
					}
					break
				}
			}

			if isNew {
				distinctFileList = append(distinctFileList, item)
			}
		}
	}
	return distinctFileList
}

func (s *Server) createSchemaUpdateIssue(ctx context.Context, repository *api.Repository, mi *db.MigrationInfo, vcsPushEvent vcs.PushEvent, added string, statement string) (string, error) {
	// Find matching database list
	databaseFind := &api.DatabaseFind{
		ProjectID: &repository.ProjectID,
		Name:      &mi.Database,
	}
	databaseList, err := s.store.FindDatabase(ctx, databaseFind)
	if err != nil {
		return "", fmt.Errorf("failed to find database matching database %q referenced by the committed file", mi.Database)
	} else if len(databaseList) == 0 {
		return "", fmt.Errorf("project with ID %d does not own database %q referenced by the committed file", repository.ProjectID, mi.Database)
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

func createTenantSchemaUpdateIssue(mi *db.MigrationInfo, vcsPushEvent vcs.PushEvent, statement string) (string, error) {
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

// createIssueFromPushEvent attempts to create a new issue for the given file of
// the push event. It returns "created=true" when a new issue has been created,
// along with the creation message to be presented in the UI. An *echo.HTTPError
// is returned in case of the error during the process.
func (s *Server) createIssueFromPushEvent(ctx context.Context, repo *api.Repository, pushEvent vcs.PushEvent, file, webhookEndpointID string) (message string, created bool, _ error) {
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
			Comment:     fmt.Sprintf("Ignored committed file %q, %s.", fileEscaped, err.Error()),
			Payload:     string(bytes),
		}
		_, err = s.ActivityManager.CreateActivity(ctx, activityCreate, &ActivityMeta{})
		if err != nil {
			log.Warn("Failed to create project activity to record ignored repository committed file",
				zap.Error(err),
			)
		}
	}

	mi, err := db.ParseMigrationInfo(fileEscaped, filepath.Join(repo.BaseDirectory, repo.FilePathTemplate))
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
		committerPrinciple, err := s.store.GetPrincipalByEmail(ctx, pushEvent.FileCommit.AuthorEmail)
		if err != nil {
			log.Error("failed to find the principal with committer email",
				zap.String("email", common.EscapeForLogging(pushEvent.FileCommit.AuthorEmail)),
				zap.Error(err),
			)
		}
		if committerPrinciple == nil {
			log.Debug("cannot find the principal with committer email, use system bot instead",
				zap.String("email", common.EscapeForLogging(pushEvent.FileCommit.AuthorEmail)),
			)
		} else {
			creatorID = committerPrinciple.ID
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
func isSkipGeneratedSchemaFile(repository *api.Repository, added string) bool {
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
