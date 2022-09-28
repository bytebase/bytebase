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
	"sort"
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
	"github.com/bytebase/bytebase/plugin/parser/differ"
	"github.com/bytebase/bytebase/plugin/vcs"
	"github.com/bytebase/bytebase/plugin/vcs/github"
	"github.com/bytebase/bytebase/plugin/vcs/gitlab"
)

const (
	gitlabWebhookPath = "hook/gitlab"
	githubWebhookPath = "hook/github"
)

func (s *Server) registerWebhookRoutes(g *echo.Group) {
	g.POST("/gitlab/:id", func(c echo.Context) error {
		ctx := c.Request().Context()

		webhookEndpointID, repos, httpErr := s.validateWebhookRequest(ctx, c)
		if httpErr != nil {
			return httpErr
		}

		body, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Failed to read webhook request").SetInternal(err)
		}

		var pushEvent gitlab.WebhookPushEvent
		if err := json.Unmarshal(body, &pushEvent); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed push event").SetInternal(err)
		}
		// This shouldn't happen as we only setup webhook to receive push event, just in case.
		if pushEvent.ObjectKind != gitlab.WebhookPush {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid webhook event type, got %s, want push", pushEvent.ObjectKind))
		}
		baseVCSPushEvent := vcs.PushEvent{
			Ref:                common.EscapeForLogging(pushEvent.Ref),
			RepositoryID:       strconv.Itoa(pushEvent.Project.ID),
			RepositoryURL:      pushEvent.Project.WebURL,
			RepositoryFullPath: pushEvent.Project.FullPath,
			AuthorName:         pushEvent.AuthorName,
		}

		filteredRepos, httpErr := filterReposCommon(baseVCSPushEvent, repos)
		if httpErr != nil {
			return httpErr
		}
		filteredRepos = filterReposGitLab(c.Request().Header, filteredRepos)
		if len(filteredRepos) == 0 {
			log.Debug("Empty handle repo list. Ignore this push event.")
			return c.String(http.StatusOK, "OK")
		}
		log.Debug("Process push event in repos", zap.Any("repos", filteredRepos))

		commitList, err := convertGitLabCommitList(pushEvent.CommitList)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to convert GitLab commits").SetInternal(err)
		}
		if len(commitList) == 0 {
			log.Debug("No commit in the GitLab push event. Ignore this push event.",
				zap.String("repoURL", common.EscapeForLogging(pushEvent.Project.WebURL)),
				zap.String("repoName", pushEvent.Project.FullPath),
				zap.String("commits", getCommitsMessage(commitList)))
			c.Response().WriteHeader(http.StatusOK)
			return nil
		}
		baseVCSPushEvent.CommitList = commitList
		createdMessages, httpErr := s.createIssuesFromCommits(ctx, webhookEndpointID, filteredRepos, commitList, baseVCSPushEvent)
		if httpErr != nil {
			return httpErr
		}
		return c.String(http.StatusOK, strings.Join(createdMessages, "\n"))
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

		webhookEndpointID, repos, httpErr := s.validateWebhookRequest(ctx, c)
		if httpErr != nil {
			return httpErr
		}

		body, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Failed to read webhook request").SetInternal(err)
		}

		var pushEvent github.WebhookPushEvent
		if err := json.Unmarshal(body, &pushEvent); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed push event").SetInternal(err)
		}
		baseVCSPushEvent := vcs.PushEvent{
			Ref:                common.EscapeForLogging(pushEvent.Ref),
			RepositoryID:       common.EscapeForLogging(pushEvent.Repository.FullName),
			RepositoryURL:      pushEvent.Repository.HTMLURL,
			RepositoryFullPath: pushEvent.Repository.FullName,
			AuthorName:         pushEvent.Sender.Login,
		}

		filteredRepos, httpErr := filterReposCommon(baseVCSPushEvent, repos)
		if httpErr != nil {
			return httpErr
		}
		filteredRepos, httpErr = filterReposGitHub(c.Request().Header, body, filteredRepos)
		if httpErr != nil {
			return httpErr
		}
		if len(filteredRepos) == 0 {
			log.Debug("Empty handle repo list. Ignore this push event.")
			return c.String(http.StatusOK, "OK")
		}
		log.Debug("Process push event in repos", zap.Any("repos", filteredRepos))

		commitList := convertGitHubCommitList(pushEvent.Commits)
		if len(pushEvent.Commits) == 0 {
			log.Debug("No commit in the GitHub push event. Ignore this push event.",
				zap.String("repoURL", common.EscapeForLogging(pushEvent.Repository.HTMLURL)),
				zap.String("repoName", common.EscapeForLogging(pushEvent.Repository.FullName)),
				zap.String("commits", getCommitsMessage(commitList)))
			c.Response().WriteHeader(http.StatusOK)
			return nil
		}
		baseVCSPushEvent.CommitList = commitList
		createdMessages, httpErr := s.createIssuesFromCommits(ctx, webhookEndpointID, filteredRepos, commitList, baseVCSPushEvent)
		if httpErr != nil {
			return httpErr
		}
		return c.String(http.StatusOK, strings.Join(createdMessages, "\n"))
	})
}

func (s *Server) createIssuesFromCommits(ctx context.Context, webhookEndpointID string, filteredRepos []*api.Repository, commitList []vcs.Commit, baseVCSPushEvent vcs.PushEvent) ([]string, *echo.HTTPError) {
	distinctFileList := dedupMigrationFilesFromCommitList(commitList)
	if len(distinctFileList) == 0 {
		log.Warn("No files found from the push event",
			zap.String("vcsRepoURL", common.EscapeForLogging(baseVCSPushEvent.RepositoryURL)),
			zap.String("vcsRepoName", baseVCSPushEvent.RepositoryFullPath),
			zap.String("commits", getCommitsMessage(commitList)))
		return nil, nil
	}

	var createdMessageList []string
	repoID2ActivityCreateList := make(map[int][]*api.ActivityCreate)
	for _, repo := range filteredRepos {
		pushEvent := baseVCSPushEvent
		pushEvent.VCSType = repo.VCS.Type
		pushEvent.BaseDirectory = repo.BaseDirectory

		createdMessage, created, activityCreateList, httpErr := s.createIssueFromPushEvent(
			ctx,
			&pushEvent,
			repo,
			webhookEndpointID,
			distinctFileList,
		)
		if httpErr != nil {
			return nil, httpErr
		}
		if created {
			createdMessageList = append(createdMessageList, createdMessage)
		}
		repoID2ActivityCreateList[repo.ID] = append(repoID2ActivityCreateList[repo.ID], activityCreateList...)
	}
	if len(createdMessageList) == 0 {
		for _, repo := range filteredRepos {
			if activityCreateList, ok := repoID2ActivityCreateList[repo.ID]; ok {
				for _, activityCreate := range activityCreateList {
					if _, err := s.ActivityManager.CreateActivity(ctx, activityCreate, &ActivityMeta{}); err != nil {
						log.Warn("Failed to create project activity for the ignored repository file", zap.Error(err))
					}
				}
			}
		}
	}
	if len(createdMessageList) == 0 {
		log.Warn("No issue created from the push event",
			zap.String("vcsRepoURL", common.EscapeForLogging(baseVCSPushEvent.RepositoryURL)),
			zap.String("vcsRepoName", baseVCSPushEvent.RepositoryFullPath),
			zap.String("commits", getCommitsMessage(commitList)))
	}
	return createdMessageList, nil
}

func (s *Server) validateWebhookRequest(ctx context.Context, c echo.Context) (string, []*api.Repository, *echo.HTTPError) {
	webhookEndpointID := c.Param("id")
	// In mono-repository settings, one GitLab Project/GitHub Repository may correspond to multiple Bytebase Project/Repository.
	// We need to further filter out the repositories for this webhook push event.
	repos, err := s.store.FindRepository(ctx, &api.RepositoryFind{WebhookEndpointID: &webhookEndpointID})
	if err != nil {
		return "", nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to respond webhook event for endpoint: %v", webhookEndpointID)).SetInternal(err)
	}
	if len(repos) == 0 {
		return "", nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Webhook endpoint not found: %v", webhookEndpointID))
	}
	return webhookEndpointID, repos, nil
}

func filterReposCommon(pushEvent vcs.PushEvent, repos []*api.Repository) ([]*api.Repository, *echo.HTTPError) {
	branch, err := parseBranchNameFromRefs(pushEvent.Ref)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Invalid ref: %s", pushEvent.Ref).SetInternal(err)
	}
	var filteredRepos []*api.Repository
	for _, repo := range repos {
		if repo.BranchFilter != branch {
			log.Debug("Skipping repo due to branch filter mismatch", zap.Int("repoID", repo.ID), zap.String("branch", branch), zap.String("filter", repo.BranchFilter))
			continue
		}
		if repo.VCS == nil {
			log.Debug("Skipping repo due to missing VCS", zap.Int("repoID", repo.ID))
			continue
		}
		if pushEvent.RepositoryID != repo.ExternalID {
			log.Debug("Skipping repo due to external ID mismatch", zap.Int("repoID", repo.ID), zap.String("pushEventExternalID", pushEvent.RepositoryID), zap.String("repoExternalID", repo.ExternalID))
			continue
		}
		filteredRepos = append(filteredRepos, repo)
	}
	return filteredRepos, nil
}

func filterReposGitLab(httpHeader http.Header, repos []*api.Repository) []*api.Repository {
	var filteredRepos []*api.Repository
	for _, repo := range repos {
		if secretToken := httpHeader.Get("X-Gitlab-Token"); secretToken != repo.WebhookSecretToken {
			log.Debug("Skipping repo due to secret token mismatch", zap.Int("repoID", repo.ID), zap.String("headerSecretToken", secretToken), zap.String("repoSecretToken", repo.WebhookSecretToken))
			continue
		}
		filteredRepos = append(filteredRepos, repo)
	}
	return filteredRepos
}

func filterReposGitHub(httpHeader http.Header, httpBody []byte, repos []*api.Repository) ([]*api.Repository, *echo.HTTPError) {
	var filteredRepos []*api.Repository
	for _, repo := range repos {
		validated, err := validateGitHubWebhookSignature256(httpHeader.Get("X-Hub-Signature-256"), repo.WebhookSecretToken, httpBody)
		if err != nil {
			return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to validate GitHub webhook signature").SetInternal(err)
		}
		if !validated {
			log.Debug("Skipping repo due to mismatched payload signature", zap.Int("repoID", repo.ID))
			continue
		}
		filteredRepos = append(filteredRepos, repo)
	}
	return filteredRepos, nil
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

// parseBranchNameFromRefs parses the branch name from the refs field in the request.
// https://docs.github.com/en/rest/git/refs
// https://docs.gitlab.com/ee/user/project/integrations/webhook_events.html#push-events
func parseBranchNameFromRefs(ref string) (string, error) {
	expectedPrefix := "refs/heads/"
	if !strings.HasPrefix(ref, expectedPrefix) || len(expectedPrefix) == len(ref) {
		log.Debug("ref is not prefix with expected prefix", zap.String("escaped ref", common.EscapeForLogging(ref)), zap.String("expected prefix", expectedPrefix))
		return ref, errors.Errorf("unexpected ref name %q without prefix %q", ref, expectedPrefix)
	}
	return ref[len(expectedPrefix):], nil
}

func convertGitLabCommitList(commitList []gitlab.WebhookCommit) ([]vcs.Commit, error) {
	var ret []vcs.Commit
	for _, commit := range commitList {
		createdTime, err := time.Parse(time.RFC3339, commit.Timestamp)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse commit %s's timestamp %s", commit.ID, common.EscapeForLogging(commit.Timestamp))
		}
		ret = append(ret, vcs.Commit{
			ID:           commit.ID,
			Title:        commit.Title,
			Message:      commit.Message,
			CreatedTs:    createdTime.Unix(),
			URL:          commit.URL,
			AuthorName:   commit.Author.Name,
			AuthorEmail:  commit.Author.Email,
			AddedList:    commit.AddedList,
			ModifiedList: commit.ModifiedList,
		})
	}
	return ret, nil
}

func convertGitHubCommitList(commitList []github.WebhookCommit) []vcs.Commit {
	var ret []vcs.Commit
	for _, commit := range commitList {
		// The Distinct is false if the commit has not been pushed before.
		if !commit.Distinct {
			continue
		}
		// Per Git convention, the message title and body are separated by two new line characters.
		messages := strings.SplitN(commit.Message, "\n\n", 2)
		messageTitle := messages[0]

		ret = append(ret, vcs.Commit{
			ID:           commit.ID,
			Title:        messageTitle,
			Message:      commit.Message,
			CreatedTs:    commit.Timestamp.Unix(),
			URL:          commit.URL,
			AuthorName:   commit.Author.Name,
			AuthorEmail:  commit.Author.Email,
			AddedList:    commit.Added,
			ModifiedList: commit.Modified,
		})
	}
	return ret
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
	createdTs int64
	commit    vcs.Commit
	fileName  string
	itemType  fileItemType
}

func dedupMigrationFilesFromCommitList(commitList []vcs.Commit) []distinctFileItem {
	// Use list instead of map because we need to maintain the relative commit order in the source branch.
	var distinctFileList []distinctFileItem
	for _, commit := range commitList {
		log.Debug("Pre-processing commit to dedup migration files...",
			zap.String("id", common.EscapeForLogging(commit.ID)),
			zap.String("title", common.EscapeForLogging(commit.Title)),
		)

		addDistinctFile := func(fileName string, itemType fileItemType) {
			item := distinctFileItem{
				createdTs: commit.CreatedTs,
				commit:    commit,
				fileName:  fileName,
				itemType:  itemType,
			}
			for i, file := range distinctFileList {
				// For the migration file with the same name, keep the one from the latest commit
				if item.fileName == file.fileName {
					if file.createdTs < commit.CreatedTs {
						// A file can be added and then modified in a later commit. We should consider the item as added.
						if file.itemType == fileItemTypeAdded {
							item.itemType = fileItemTypeAdded
						}
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
		for _, modified := range commit.ModifiedList {
			addDistinctFile(modified, fileItemTypeModified)
		}
	}
	return distinctFileList
}

// findProjectDatabases finds the list of databases with given name in the
// project. If the `envName` is not empty, it will be used as a filter condition
// for the result list.
func (s *Server) findProjectDatabases(ctx context.Context, projectID int, tenantMode api.ProjectTenantMode, dbName, envName string) ([]*api.Database, error) {
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

	// Tenant mode does not allow filtering databases by environment and expect
	// multiple databases with the same name.
	if tenantMode == api.TenantModeTenant {
		if envName != "" {
			return nil, errors.Errorf("non-empty environment is not allowed for tenant mode project")
		}
		return foundDatabases, nil
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

// getIgnoredFileActivityCreate get a warning project activityCreate for the ignored file with given error.
func getIgnoredFileActivityCreate(projectID int, pushEvent *vcs.PushEvent, file string, err error) *api.ActivityCreate {
	payload, marshalErr := json.Marshal(
		api.ActivityProjectRepositoryPushPayload{
			VCSPushEvent: *pushEvent,
		},
	)
	if marshalErr != nil {
		log.Warn("Failed to construct project activity payload for the ignored repository file",
			zap.Error(marshalErr),
		)
		return nil
	}

	return &api.ActivityCreate{
		CreatorID:   api.SystemBotID,
		ContainerID: projectID,
		Type:        api.ActivityProjectRepositoryPush,
		Level:       api.ActivityWarn,
		Comment:     fmt.Sprintf("Ignored file %q, %v.", file, err),
		Payload:     string(payload),
	}
}

// readFileContent reads the content of the given file from the given repository.
func (s *Server) readFileContent(ctx context.Context, pushEvent *vcs.PushEvent, webhookEndpointID string, file string) (string, error) {
	// Retrieve the latest AccessToken and RefreshToken as the previous
	// ReadFileContent call may have updated the stored token pair. ReadFileContent
	// will fetch and store the new token pair if the existing token pair has
	// expired.
	repos, err := s.store.FindRepository(ctx, &api.RepositoryFind{WebhookEndpointID: &webhookEndpointID})
	if err != nil {
		return "", errors.Wrapf(err, "get repository by webhook endpoint %q", webhookEndpointID)
	} else if len(repos) == 0 {
		return "", errors.Wrapf(err, "repository not found by webhook endpoint %q", webhookEndpointID)
	}

	// In mono-repository settings, one GitLab Project/GitHub Repository may correspond to multiple Bytebase Project/Repository.
	// In this case, they are the same except for the record ID in database.
	// So we can just use the first one in the list.
	repo := repos[0]
	content, err := vcs.Get(repo.VCS.Type, vcs.ProviderConfig{}).ReadFileContent(
		ctx,
		common.OauthContext{
			ClientID:     repo.VCS.ApplicationID,
			ClientSecret: repo.VCS.Secret,
			AccessToken:  repo.AccessToken,
			RefreshToken: repo.RefreshToken,
			Refresher:    s.refreshToken(ctx, repo.WebURL),
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

// prepareIssueFromPushEventSDL returns the migration info and a list of update
// schema details derived from the given push event for SDL.
func (s *Server) prepareIssueFromPushEventSDL(ctx context.Context, repo *api.Repository, pushEvent *vcs.PushEvent, schemaInfo map[string]string, file string, webhookEndpointID string) ([]*api.MigrationDetail, []*api.ActivityCreate) {
	dbName := schemaInfo["DB_NAME"]
	if dbName == "" {
		log.Debug("Ignored schema file without a database name", zap.String("file", file))
		return nil, nil
	}

	statement, err := s.readFileContent(ctx, pushEvent, webhookEndpointID, file)
	if err != nil {
		activityCreate := getIgnoredFileActivityCreate(repo.ProjectID, pushEvent, file, errors.Wrap(err, "Failed to read file content"))
		return nil, []*api.ActivityCreate{activityCreate}
	}

	activityCreateList := []*api.ActivityCreate{}
	envName := schemaInfo["ENV_NAME"]
	var migrationDetailList []*api.MigrationDetail
	if repo.Project.TenantMode == api.TenantModeTenant {
		migrationDetailList = append(migrationDetailList,
			&api.MigrationDetail{
				DatabaseName: dbName,
				Statement:    statement,
			},
		)
		return migrationDetailList, nil
	}

	databases, err := s.findProjectDatabases(ctx, repo.ProjectID, repo.Project.TenantMode, dbName, envName)
	if err != nil {
		activityCreate := getIgnoredFileActivityCreate(repo.ProjectID, pushEvent, file, errors.Wrap(err, "Failed to find project databases"))
		return nil, []*api.ActivityCreate{activityCreate}
	}

	for _, database := range databases {
		diff, err := s.computeDatabaseSchemaDiff(ctx, database, statement)
		if err != nil {
			activityCreate := getIgnoredFileActivityCreate(repo.ProjectID, pushEvent, file, errors.Wrap(err, "Failed to compute database schema diff"))
			activityCreateList = append(activityCreateList, activityCreate)
			continue
		}

		migrationDetailList = append(migrationDetailList,
			&api.MigrationDetail{
				DatabaseID: database.ID,
				Statement:  diff,
			},
		)
	}

	return migrationDetailList, activityCreateList
}

// prepareIssueFromPushEventDDL returns a list of update schema details derived
// from the given push event for DDL.
func (s *Server) prepareIssueFromPushEventDDL(ctx context.Context, repo *api.Repository, pushEvent *vcs.PushEvent, fileName string, fileType fileItemType, webhookEndpointID string, migrationInfo *db.MigrationInfo) ([]*api.MigrationDetail, []*api.ActivityCreate) {
	statement, err := s.readFileContent(ctx, pushEvent, webhookEndpointID, fileName)
	if err != nil {
		activityCreate := getIgnoredFileActivityCreate(repo.ProjectID, pushEvent, fileName, errors.Wrap(err, "Failed to read file content"))
		return nil, []*api.ActivityCreate{activityCreate}
	}

	var migrationDetailList []*api.MigrationDetail

	// TODO(dragonly): handle modified file for tenant mode.
	if repo.Project.TenantMode == api.TenantModeTenant {
		migrationDetailList = append(migrationDetailList,
			&api.MigrationDetail{
				DatabaseName:  migrationInfo.Database,
				Statement:     statement,
				SchemaVersion: migrationInfo.Version,
			},
		)
		return migrationDetailList, nil
	}

	databases, err := s.findProjectDatabases(ctx, repo.ProjectID, repo.Project.TenantMode, migrationInfo.Database, migrationInfo.Environment)
	if err != nil {
		activityCreate := getIgnoredFileActivityCreate(repo.ProjectID, pushEvent, fileName, errors.Wrap(err, "Failed to find project databases"))
		return nil, []*api.ActivityCreate{activityCreate}
	}

	if fileType == fileItemTypeAdded {
		for _, database := range databases {
			migrationDetailList = append(migrationDetailList,
				&api.MigrationDetail{
					DatabaseID:    database.ID,
					Statement:     statement,
					SchemaVersion: migrationInfo.Version,
				},
			)
		}
		return migrationDetailList, nil
	}

	if err := s.tryUpdateTasksFromModifiedFile(ctx, databases, fileName, migrationInfo.Version, statement); err != nil {
		activityCreate := getIgnoredFileActivityCreate(repo.ProjectID, pushEvent, fileName, errors.Wrap(err, "Failed to find project task"))
		return nil, []*api.ActivityCreate{activityCreate}
	}

	return nil, nil
}

func (s *Server) tryUpdateTasksFromModifiedFile(ctx context.Context, databases []*api.Database, fileName, schemaVersion, statement string) error {
	// For modified files, we try to update the existing issue's statement.
	for _, database := range databases {
		find := &api.TaskFind{
			DatabaseID: &database.ID,
			StatusList: &[]api.TaskStatus{api.TaskPendingApproval, api.TaskFailed},
			TypeList:   &[]api.TaskType{api.TaskDatabaseSchemaUpdate, api.TaskDatabaseDataUpdate},
			Payload:    fmt.Sprintf("payload->>'schemaVersion' = '%s'", schemaVersion),
		}
		taskList, err := s.store.FindTask(ctx, find, true)
		if err != nil {
			return err
		}
		if len(taskList) == 0 {
			continue
		}
		if len(taskList) > 1 {
			log.Error("Found more than one pending approval or failed tasks for modified VCS file, should be only one task.", zap.Int("databaseID", database.ID), zap.String("schemaVersion", schemaVersion))
			return nil
		}
		task := taskList[0]
		taskPatch := api.TaskPatch{
			ID:        task.ID,
			Statement: &statement,
			UpdaterID: api.SystemBotID,
		}
		issue, err := s.store.GetIssueByPipelineID(ctx, task.PipelineID)
		if err != nil {
			log.Error(fmt.Sprintf("Failed to get issue by pipeline ID %d", task.PipelineID), zap.Error(err))
			return nil
		}
		// TODO(dragonly): Try to patch the failed migration history record to pending, and the statement to the current modified file content.
		log.Debug("Patching task for modified file VCS push event", zap.String("fileName", fileName), zap.Int("issueID", issue.ID), zap.Int("taskID", task.ID))
		if _, err := s.patchTask(ctx, task, &taskPatch, issue); err != nil {
			log.Error("Failed to patch task with the same migration version", zap.Int("issueID", issue.ID), zap.Int("taskID", task.ID), zap.Error(err))
			return nil
		}
	}
	return nil
}

// createIssueFromPushEvent attempts to create a new issue for the given files of
// the push event. It returns "created=true" when a new issue has been created,
// along with the creation message to be presented in the UI. An *echo.HTTPError
// is returned in case of the error during the process.
func (s *Server) createIssueFromPushEvent(ctx context.Context, pushEvent *vcs.PushEvent, repo *api.Repository, webhookEndpointID string, fileList []distinctFileItem) (string, bool, []*api.ActivityCreate, *echo.HTTPError) {
	if repo.Project.TenantMode == api.TenantModeTenant {
		if !s.feature(api.FeatureMultiTenancy) {
			return "", false, nil, echo.NewHTTPError(http.StatusForbidden, api.FeatureMultiTenancy.AccessErrorMessage())
		}
	}

	// Filter files that are in the repo's base directory.
	var filesEscaped []string
	var filesToProcess []distinctFileItem
	for _, file := range fileList {
		fileEscaped := common.EscapeForLogging(file.fileName)
		if !strings.HasPrefix(file.fileName, repo.BaseDirectory) {
			log.Debug("Ignored file outside the base directory",
				zap.String("file", fileEscaped),
				zap.String("base_directory", repo.BaseDirectory),
			)
			continue
		}
		filesToProcess = append(filesToProcess, file)
		filesEscaped = append(filesEscaped, fileEscaped)
	}
	if len(filesToProcess) == 0 {
		return "", false, nil, nil
	}
	log.Debug("Processing files", zap.Strings("files", filesEscaped))

	// Parse schema info from each file name. It's used to tell whether each file is a schema file or normal migration file.
	var schemaInfoList []map[string]string
	for _, file := range filesEscaped {
		schemaInfo, err := parseSchemaFileInfo(repo.BaseDirectory, repo.SchemaPathTemplate, file)
		if err != nil {
			log.Debug("Failed to parse schema file info", zap.String("file", file), zap.Error(err))
			return "", false, nil, nil
		}
		schemaInfoList = append(schemaInfoList, schemaInfo)
	}

	var migrationDetailList []*api.MigrationDetail
	var activityCreateList []*api.ActivityCreate
	// Default to DATA migration. If later we found any MIGRATE migration file, we set migrationType to MIGRATE.
	migrationType := db.Data
	if repo.Project.SchemaChangeType == api.ProjectSchemaChangeTypeDDL {
		var nonSchemaFileList []distinctFileItem
		for i, si := range schemaInfoList {
			if si != nil {
				log.Debug("Ignored schema file for non-SDL", zap.String("file", filesEscaped[i]), zap.String("type", string(filesToProcess[i].itemType)))
			} else {
				nonSchemaFileList = append(nonSchemaFileList, filesToProcess[i])
			}
		}
		if len(nonSchemaFileList) == 0 {
			return "", false, nil, nil
		}

		// There are possibly multiple files in the push event.
		// Each file corresponds to a (database name, schema version) pair.
		// We want the migration statements are sorted by the file's schema version, and grouped by the database name.
		fileListSorted, migrationInfoList, err := sortFilesBySchemaVersionGroupByDatabase(repo, nonSchemaFileList)
		if err != nil {
			return "", false, nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to sort files by schema version group by database").SetInternal(err)
		}
		for i, file := range fileListSorted {
			migrationDetailListForFile, activityCreateListForFile := s.prepareIssueFromPushEventDDL(ctx, repo, pushEvent, file.fileName, file.itemType, webhookEndpointID, migrationInfoList[i])
			activityCreateList = append(activityCreateList, activityCreateListForFile...)
			migrationDetailList = append(migrationDetailList, migrationDetailListForFile...)
			// If there's one schema migration file, we set the issue type as MIGRATE.
			if migrationInfoList[i].Type == db.Migrate {
				migrationType = db.Migrate
			}
		}
	} else {
		var schemaFileList []distinctFileItem
		var schemaInfoListForSDL []map[string]string
		var nonSchemaFileList []distinctFileItem
		for i, si := range schemaInfoList {
			if si != nil {
				schemaFileList = append(schemaFileList, filesToProcess[i])
				schemaInfoListForSDL = append(schemaInfoListForSDL, si)
			} else {
				nonSchemaFileList = append(nonSchemaFileList, filesToProcess[i])
			}
		}
		// If there's one schema state file, we set the issue type as MIGRATE.
		if len(schemaFileList) != 0 {
			migrationType = db.Migrate
		}

		// State based migration.
		for i, file := range schemaFileList {
			migrationDetailListForFile, activityCreateListForFile := s.prepareIssueFromPushEventSDL(ctx, repo, pushEvent, schemaInfoListForSDL[i], file.fileName, webhookEndpointID)
			migrationDetailList = append(migrationDetailList, migrationDetailListForFile...)
			activityCreateList = append(activityCreateList, activityCreateListForFile...)
		}

		// For non-schema files, we execute those DATA migration ones.
		fileListSorted, migrationInfoList, err := sortFilesBySchemaVersionGroupByDatabase(repo, nonSchemaFileList)
		if err != nil {
			return "", false, nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to sort files by schema version group by database").SetInternal(err)
		}
		for i, file := range fileListSorted {
			if migrationInfoList[i].Type != db.Data {
				err := errors.Errorf("skip %s type migration script %s. Only allow DATA type migration in state-based migration project", migrationInfoList[i].Type, common.EscapeForLogging(file.fileName))
				activityCreate := getIgnoredFileActivityCreate(repo.ProjectID, pushEvent, file.fileName, err)
				activityCreateList = append(activityCreateList, activityCreate)
				log.Debug("Skip file in state-based migration", zap.Error(err))
				continue
			}
			migrationDetailListForFile, activityCreateListForFile := s.prepareIssueFromPushEventDDL(ctx, repo, pushEvent, file.fileName, file.itemType, webhookEndpointID, migrationInfoList[i])
			activityCreateList = append(activityCreateList, activityCreateListForFile...)
			migrationDetailList = append(migrationDetailList, migrationDetailListForFile...)
		}
	}

	if len(migrationDetailList) == 0 {
		return "", false, activityCreateList, nil
	}

	// Find out the creator principal.
	creatorID := api.SystemBotID
	if pushEvent.FileCommit.AuthorEmail != "" {
		committerPrincipal, err := s.store.GetPrincipalByEmail(ctx, pushEvent.FileCommit.AuthorEmail)
		if err != nil {
			log.Warn("Failed to find the principal with committer email, use system bot instead",
				zap.String("email", common.EscapeForLogging(pushEvent.FileCommit.AuthorEmail)),
				zap.Error(err),
			)
		} else if committerPrincipal == nil {
			log.Debug("Principal with committer email does not exist, use system bot instead",
				zap.String("email", common.EscapeForLogging(pushEvent.FileCommit.AuthorEmail)))
		} else {
			creatorID = committerPrincipal.ID
		}
	}

	// Create the issue.
	createContext, err := json.Marshal(
		&api.MigrationContext{
			MigrationType: migrationType,
			VCSPushEvent:  pushEvent,
			DetailList:    migrationDetailList,
		},
	)
	if err != nil {
		return "", false, activityCreateList, echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal update schema context").SetInternal(err)
	}

	issueType := api.IssueDatabaseSchemaUpdate
	if migrationType == db.Data {
		issueType = api.IssueDatabaseDataUpdate
	}
	commitsMsg := getCommitsMessageShort(pushEvent.CommitList)
	issueCreate := &api.IssueCreate{
		ProjectID:     repo.ProjectID,
		Name:          fmt.Sprintf("%s by %s", migrationType, commitsMsg),
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
		return "", false, activityCreateList, echo.NewHTTPError(http.StatusInternalServerError, errMsg).SetInternal(err)
	}

	// Create a project activity after successfully creating the issue from the push event.
	activityPayload, err := json.Marshal(
		api.ActivityProjectRepositoryPushPayload{
			VCSPushEvent: *pushEvent,
			IssueID:      issue.ID,
			IssueName:    issue.Name,
		},
	)
	if err != nil {
		return "", false, activityCreateList, echo.NewHTTPError(http.StatusInternalServerError, "Failed to construct activity payload").SetInternal(err)
	}

	activityCreate := &api.ActivityCreate{
		CreatorID:   creatorID,
		ContainerID: repo.ProjectID,
		Type:        api.ActivityProjectRepositoryPush,
		Level:       api.ActivityInfo,
		Comment:     fmt.Sprintf("Created issue %q.", issue.Name),
		Payload:     string(activityPayload),
	}
	if _, err = s.ActivityManager.CreateActivity(ctx, activityCreate, &ActivityMeta{}); err != nil {
		return "", false, activityCreateList, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create project activity after creating issue from repository push event: %d", issue.ID)).SetInternal(err)
	}

	return fmt.Sprintf("Created issue %q by %s", issue.Name, commitsMsg), true, activityCreateList, nil
}

func sortFilesBySchemaVersionGroupByDatabase(repo *api.Repository, fileList []distinctFileItem) ([]distinctFileItem, []*db.MigrationInfo, error) {
	type sortItem struct {
		file distinctFileItem
		mi   *db.MigrationInfo
	}
	var sortItemList []sortItem
	for _, file := range fileList {
		// NOTE: We do not want to use filepath.Join here because we always need "/" as the path separator.
		template := path.Join(repo.BaseDirectory, repo.FilePathTemplate)
		mi, err := db.ParseMigrationInfo(file.fileName, template)
		if err != nil {
			fileNameEscaped := common.EscapeForLogging(file.fileName)
			log.Error("Failed to parse migration info", zap.String("file", fileNameEscaped), zap.String("template", template), zap.Error(err))
			return nil, nil, errors.Wrapf(err, "failed to parse migration info with file name %s and template %s", fileNameEscaped, template)
		}
		sortItemList = append(sortItemList, sortItem{file: file, mi: mi})
	}
	sort.Slice(sortItemList, func(i, j int) bool {
		mi := sortItemList[i].mi
		mj := sortItemList[j].mi
		return mi.Database < mj.Database || (mi.Database == mj.Database && mi.Version < mj.Version)
	})

	var fileListSorted []distinctFileItem
	var miListSorted []*db.MigrationInfo
	for _, item := range sortItemList {
		fileListSorted = append(fileListSorted, item.file)
		miListSorted = append(miListSorted, item.mi)
	}

	return fileListSorted, miListSorted, nil
}

func getCommitsMessage(commitList []vcs.Commit) string {
	var commitIDs []string
	for _, c := range commitList {
		commitIDs = append(commitIDs, c.ID)
	}
	return strings.Join(commitIDs, ", ")
}

func getCommitsMessageShort(commitList []vcs.Commit) string {
	if len(commitList) == 0 {
		return ""
	}
	if len(commitList) == 1 {
		return "commit " + commitList[0].ID
	}
	return fmt.Sprintf("commits %s...%s", commitList[0].ID, commitList[len(commitList)-1].ID)
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

	diff, err := differ.SchemaDiff(engine, schema.String(), newSchemaStr)
	if err != nil {
		return "", errors.New("compute schema diff")
	}
	return diff, nil
}
