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
	"sort"
	"strings"

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
	// issueNameTemplate should be consistent with UI issue names generated from the frontend except for the timestamp.
	// Because we cannot get the correct timezone of the client here.
	// Example: "[db-5] Alter schema".
	issueNameTemplate = "[%s] %s"
)

func (s *Server) registerWebhookRoutes(g *echo.Group) {
	g.POST("/gitlab/:id", func(c echo.Context) error {
		ctx := c.Request().Context()

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
		repositoryID := fmt.Sprintf("%v", pushEvent.Project.ID)

		filter := func(token string) (bool, error) {
			return c.Request().Header.Get("X-Gitlab-Token") == token, nil
		}
		repositoryList, err := s.filterRepository(ctx, c.Param("id"), pushEvent.Ref, repositoryID, filter)
		if err != nil {
			return err
		}
		if len(repositoryList) == 0 {
			log.Debug("Empty handle repo list. Ignore this push event.")
			return c.String(http.StatusOK, "OK")
		}

		baseVCSPushEvent, err := pushEvent.ToVCS()
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to convert GitLab commits").SetInternal(err)
		}

		createdMessages, err := s.processPushEvent(ctx, repositoryList, baseVCSPushEvent)
		if err != nil {
			return err
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

		body, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Failed to read webhook request").SetInternal(err)
		}
		var pushEvent github.WebhookPushEvent
		if err := json.Unmarshal(body, &pushEvent); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed push event").SetInternal(err)
		}
		repositoryID := pushEvent.Repository.FullName

		filter := func(token string) (bool, error) {
			ok, err := validateGitHubWebhookSignature256(c.Request().Header.Get("X-Hub-Signature-256"), token, body)
			if err != nil {
				return false, echo.NewHTTPError(http.StatusInternalServerError, "Failed to validate GitHub webhook signature").SetInternal(err)
			}
			return ok, nil
		}
		repositoryList, err := s.filterRepository(ctx, c.Param("id"), pushEvent.Ref, repositoryID, filter)
		if err != nil {
			return err
		}
		if len(repositoryList) == 0 {
			log.Debug("Empty handle repo list. Ignore this push event.")
			return c.String(http.StatusOK, "OK")
		}

		baseVCSPushEvent := pushEvent.ToVCS()

		createdMessages, err := s.processPushEvent(ctx, repositoryList, baseVCSPushEvent)
		if err != nil {
			return err
		}
		return c.String(http.StatusOK, strings.Join(createdMessages, "\n"))
	})
}

type repositoryFilter func(string) (bool, error)

func (s *Server) filterRepository(ctx context.Context, webhookEndpointID string, pushEventRef, pushEventRepositoryID string, filter repositoryFilter) ([]*api.Repository, error) {
	repos, err := s.store.FindRepository(ctx, &api.RepositoryFind{WebhookEndpointID: &webhookEndpointID})
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to respond webhook event for endpoint: %v", webhookEndpointID)).SetInternal(err)
	}
	if len(repos) == 0 {
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Repository for webhook endpoint %s not found", webhookEndpointID))
	}

	branch, err := parseBranchNameFromRefs(pushEventRef)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Invalid ref: %s", pushEventRef).SetInternal(err)
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
		if pushEventRepositoryID != repo.ExternalID {
			log.Debug("Skipping repo due to external ID mismatch", zap.Int("repoID", repo.ID), zap.String("pushEventExternalID", pushEventRepositoryID), zap.String("repoExternalID", repo.ExternalID))
			continue
		}

		ok, err := filter(repo.WebhookSecretToken)
		if err != nil {
			return nil, err
		}
		if !ok {
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
		log.Debug(
			"ref is not prefix with expected prefix",
			zap.String("ref", ref),
			zap.String("expected prefix", expectedPrefix),
		)
		return ref, errors.Errorf("unexpected ref name %q without prefix %q", ref, expectedPrefix)
	}
	return ref[len(expectedPrefix):], nil
}

func (s *Server) processPushEvent(ctx context.Context, repositoryList []*api.Repository, baseVCSPushEvent vcs.PushEvent) ([]string, error) {
	distinctFileList := baseVCSPushEvent.GetDistinctFileList()
	if len(distinctFileList) == 0 {
		var commitIDs []string
		for _, c := range baseVCSPushEvent.CommitList {
			commitIDs = append(commitIDs, c.ID)
		}
		log.Warn("No files found from the push event",
			zap.String("repoURL", baseVCSPushEvent.RepositoryURL),
			zap.String("repoName", baseVCSPushEvent.RepositoryFullPath),
			zap.String("commits", strings.Join(commitIDs, ",")))
		return nil, nil
	}

	var createdMessageList []string
	repoID2FileItemList := groupFileInfoByRepo(distinctFileList, repositoryList)
	for _, fileInfoListInRepo := range repoID2FileItemList {
		// There are possibly multiple files in the push event.
		// Each file corresponds to a (database name, schema version) pair.
		// We want the migration statements are sorted by the file's schema version, and grouped by the database name.
		dbID2FileInfoList := groupFileInfoByDatabase(fileInfoListInRepo)
		for _, fileInfoListInDB := range dbID2FileInfoList {
			fileInfoListSorted := sortFilesBySchemaVersion(fileInfoListInDB)
			repository := fileInfoListSorted[0].repository
			pushEvent := baseVCSPushEvent
			pushEvent.VCSType = repository.VCS.Type
			pushEvent.BaseDirectory = repository.BaseDirectory
			createdMessage, created, activityCreateList, err := s.processFilesInProject(
				ctx,
				pushEvent,
				repository,
				fileInfoListSorted,
			)
			if err != nil {
				return nil, err
			}
			if created {
				createdMessageList = append(createdMessageList, createdMessage)
			} else {
				for _, activityCreate := range activityCreateList {
					if _, err := s.ActivityManager.CreateActivity(ctx, activityCreate, &ActivityMeta{}); err != nil {
						log.Warn("Failed to create project activity for the ignored repository files", zap.Error(err))
					}
				}
			}
		}
	}

	if len(createdMessageList) == 0 {
		var repoURLs []string
		for _, repo := range repositoryList {
			repoURLs = append(repoURLs, repo.WebURL)
		}
		log.Warn("Ignored push event because no applicable file found in the commit list", zap.Strings("repos", repoURLs))
	}

	return createdMessageList, nil
}

type fileInfo struct {
	item          vcs.DistinctFileItem
	migrationInfo *db.MigrationInfo
	fType         fileType
	repository    *api.Repository
}

func groupFileInfoByDatabase(fileInfoList []fileInfo) map[string][]fileInfo {
	dbID2FileInfoList := make(map[string][]fileInfo)
	for _, fileInfo := range fileInfoList {
		dbID2FileInfoList[fileInfo.migrationInfo.Database] = append(dbID2FileInfoList[fileInfo.migrationInfo.Database], fileInfo)
	}
	return dbID2FileInfoList
}

// groupFileInfoByRepo groups information for distinct files in the push event by their corresponding api.Repository.
// In a GitLab/GitHub monorepo, a user could create multiple projects and configure different base directory in the repository.
// Bytebase will create a different api.Repository for each project. If the user decides to do a migration in multiple directories at once,
// the push event will trigger changes in multiple projects. So we first group the files into api.Repository, and create issue(s) in
// each project.
func groupFileInfoByRepo(distinctFileList []vcs.DistinctFileItem, repositoryList []*api.Repository) map[int][]fileInfo {
	repoID2FileItemList := make(map[int][]fileInfo)
	for _, item := range distinctFileList {
		log.Debug("Processing file", zap.String("file", item.FileName), zap.String("commit", item.Commit.ID))
		migrationInfo, fType, repository, err := getFileInfo(item, repositoryList)
		if err != nil {
			log.Warn("Failed to get file info for the ignored repository file",
				zap.String("file", item.FileName),
				zap.Error(err),
			)
			continue
		}
		repoID2FileItemList[repository.ID] = append(repoID2FileItemList[repository.ID], fileInfo{
			item:          item,
			migrationInfo: migrationInfo,
			fType:         fType,
			repository:    repository,
		})
	}
	return repoID2FileItemList
}

type fileType int

const (
	unknownFileType fileType = iota
	migrationFileType
	schemaFileType
)

func getFileInfo(fileItem vcs.DistinctFileItem, repositoryList []*api.Repository) (*db.MigrationInfo, fileType, *api.Repository, error) {
	var migrationInfo *db.MigrationInfo
	var fType fileType
	var fileRepositoryList []*api.Repository
	for _, repository := range repositoryList {
		if !strings.HasPrefix(fileItem.FileName, repository.BaseDirectory) {
			log.Debug("Ignored file outside the base directory",
				zap.String("file", fileItem.FileName),
				zap.String("base_directory", repository.BaseDirectory),
			)
			continue
		}

		allowOmitDatabaseName := repository.Project.TenantMode == api.TenantModeTenant && repository.Project.DBNameTemplate == ""
		// NOTE: We do not want to use filepath.Join here because we always need "/" as the path separator.
		mi, err := db.ParseMigrationInfo(fileItem.FileName, path.Join(repository.BaseDirectory, repository.FilePathTemplate), allowOmitDatabaseName)
		if err != nil {
			log.Error("Failed to parse migration file info",
				zap.Int("project", repository.ProjectID),
				zap.String("file", fileItem.FileName),
				zap.Error(err),
			)
			continue
		}
		if mi != nil {
			migrationInfo = mi
			fType = migrationFileType
			fileRepositoryList = append(fileRepositoryList, repository)
			continue
		}

		si, err := db.ParseSchemaFileInfo(repository.BaseDirectory, repository.SchemaPathTemplate, fileItem.FileName)
		if err != nil {
			log.Debug("Failed to parse schema file info",
				zap.String("file", fileItem.FileName),
				zap.Error(err),
			)
			continue
		}
		if si != nil {
			migrationInfo = si
			fType = schemaFileType
			fileRepositoryList = append(fileRepositoryList, repository)
			continue
		}
	}

	switch len(fileRepositoryList) {
	case 0:
		return nil, unknownFileType, nil, errors.Errorf("file change is not associated with any project")
	case 1:
		return migrationInfo, fType, fileRepositoryList[0], nil
	default:
		var projectList []string
		for _, repository := range fileRepositoryList {
			projectList = append(projectList, repository.Project.Name)
		}
		return nil, unknownFileType, nil, errors.Errorf("file change should be associated with exactly one project but found %s", strings.Join(projectList, ","))
	}
}

// processFilesInProject attempts to create new issue(s) according to the repository type.
// 1. For a state based project, we create one issue per schema file, and one issue for all of the rest migration files (if any).
// 2. For a migration based project, we create one issue for all of the migration files. All schema files are ignored.
// It returns "created=true" when new issue(s) has been created,
// along with the creation message to be presented in the UI. An *echo.HTTPError
// is returned in case of the error during the process.
func (s *Server) processFilesInProject(ctx context.Context, pushEvent vcs.PushEvent, repo *api.Repository, fileInfoList []fileInfo) (string, bool, []*api.ActivityCreate, *echo.HTTPError) {
	if repo.Project.TenantMode == api.TenantModeTenant && !s.feature(api.FeatureMultiTenancy) {
		return "", false, nil, echo.NewHTTPError(http.StatusForbidden, api.FeatureMultiTenancy.AccessErrorMessage())
	}

	var migrationDetailList []*api.MigrationDetail
	var activityCreateList []*api.ActivityCreate
	var createdIssueList []string
	var fileNameList []string

	creatorID := s.getIssueCreatorID(ctx, pushEvent.CommitList[0].AuthorEmail)
	for _, fileInfo := range fileInfoList {
		if fileInfo.fType == schemaFileType {
			if repo.Project.SchemaChangeType == api.ProjectSchemaChangeTypeSDL {
				// Create one issue per schema file for SDL project.
				migrationDetailListForFile, activityCreateListForFile := s.prepareIssueFromSDLFile(ctx, repo, pushEvent, fileInfo.migrationInfo, fileInfo.item.FileName)
				activityCreateList = append(activityCreateList, activityCreateListForFile...)
				if len(migrationDetailListForFile) != 0 {
					databaseName := fileInfo.migrationInfo.Database
					issueName := fmt.Sprintf(issueNameTemplate, databaseName, "Alter schema")
					issueDescription := fmt.Sprintf("Apply schema diff by file %s", strings.TrimPrefix(fileInfo.item.FileName, repo.BaseDirectory+"/"))
					if err := s.createIssueFromMigrationDetailList(ctx, issueName, issueDescription, pushEvent, creatorID, repo.ProjectID, migrationDetailListForFile); err != nil {
						return "", false, activityCreateList, echo.NewHTTPError(http.StatusInternalServerError, "Failed to create issue").SetInternal(err)
					}
					createdIssueList = append(createdIssueList, issueName)
				}
			} else {
				log.Debug("Ignored schema file for non-SDL project", zap.String("fileName", fileInfo.item.FileName), zap.String("type", string(fileInfo.item.ItemType)))
			}
		} else { // fileInfo.fType == migrationFileType
			// This is a migration-based DDL or DML file and we would allow it for both DDL and SDL schema change type project.
			// For DDL schema change type project, this is expected.
			// For SDL schema change type project, we allow it because:
			// 1) DML is always migration-based.
			// 2) We may have a limitation in SDL implementation.
			// 3) User just wants to break the glass.
			migrationDetailListForFile, activityCreateListForFile := s.prepareIssueFromFile(ctx, repo, pushEvent, fileInfo.item.FileName, fileInfo.item.ItemType, fileInfo.migrationInfo)
			activityCreateList = append(activityCreateList, activityCreateListForFile...)
			migrationDetailList = append(migrationDetailList, migrationDetailListForFile...)
			if len(migrationDetailListForFile) != 0 {
				fileNameList = append(fileNameList, strings.TrimPrefix(fileInfo.item.FileName, repo.BaseDirectory+"/"))
			}
		}
	}

	if len(migrationDetailList) == 0 {
		return "", len(createdIssueList) != 0, activityCreateList, nil
	}

	// Create one issue per push event for DDL project, or non-schema files for SDL project.
	migrateType := "Change data"
	for _, d := range migrationDetailList {
		if d.MigrationType == db.Migrate {
			migrateType = "Alter schema"
			break
		}
	}
	// The files are grouped by database names before calling this function, so they have the same database name here.
	databaseName := fileInfoList[0].migrationInfo.Database
	issueName := fmt.Sprintf(issueNameTemplate, databaseName, migrateType)
	issueDescription := fmt.Sprintf("By VCS files %s", strings.Join(fileNameList, ", "))
	if err := s.createIssueFromMigrationDetailList(ctx, issueName, issueDescription, pushEvent, creatorID, repo.ProjectID, migrationDetailList); err != nil {
		return "", len(createdIssueList) != 0, activityCreateList, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create issue %s", issueName)).SetInternal(err)
	}
	createdIssueList = append(createdIssueList, issueName)

	return fmt.Sprintf("Created issue %q from push event", strings.Join(createdIssueList, ",")), true, activityCreateList, nil
}

func sortFilesBySchemaVersion(fileInfoList []fileInfo) []fileInfo {
	var ret []fileInfo
	ret = append(ret, fileInfoList...)
	sort.Slice(ret, func(i, j int) bool {
		mi := ret[i].migrationInfo
		mj := ret[j].migrationInfo
		return mi.Database < mj.Database || (mi.Database == mj.Database && mi.Version < mj.Version)
	})
	return ret
}

func (s *Server) createIssueFromMigrationDetailList(ctx context.Context, issueName, issueDescription string, pushEvent vcs.PushEvent, creatorID, projectID int, migrationDetailList []*api.MigrationDetail) error {
	createContext, err := json.Marshal(
		&api.MigrationContext{
			VCSPushEvent: &pushEvent,
			DetailList:   migrationDetailList,
		},
	)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal update schema context").SetInternal(err)
	}

	// TODO(d): unify issue type for database changes.
	issueType := api.IssueDatabaseDataUpdate
	for _, detail := range migrationDetailList {
		if detail.MigrationType == db.Migrate || detail.MigrationType == db.Baseline {
			issueType = api.IssueDatabaseSchemaUpdate
		}
	}
	issueCreate := &api.IssueCreate{
		ProjectID:     projectID,
		Name:          issueName,
		Type:          issueType,
		Description:   issueDescription,
		AssigneeID:    api.SystemBotID,
		CreateContext: string(createContext),
	}
	issue, err := s.createIssue(ctx, issueCreate, creatorID)
	if err != nil {
		errMsg := "Failed to create schema update issue"
		if issueType == api.IssueDatabaseDataUpdate {
			errMsg = "Failed to create data update issue"
		}
		return echo.NewHTTPError(http.StatusInternalServerError, errMsg).SetInternal(err)
	}

	// Create a project activity after successfully creating the issue from the push event.
	activityPayload, err := json.Marshal(
		api.ActivityProjectRepositoryPushPayload{
			VCSPushEvent: pushEvent,
			IssueID:      issue.ID,
			IssueName:    issue.Name,
		},
	)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to construct activity payload").SetInternal(err)
	}

	activityCreate := &api.ActivityCreate{
		CreatorID:   creatorID,
		ContainerID: projectID,
		Type:        api.ActivityProjectRepositoryPush,
		Level:       api.ActivityInfo,
		Comment:     fmt.Sprintf("Created issue %q.", issue.Name),
		Payload:     string(activityPayload),
	}
	if _, err = s.ActivityManager.CreateActivity(ctx, activityCreate, &ActivityMeta{}); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create project activity after creating issue from repository push event: %d", issue.ID)).SetInternal(err)
	}

	return nil
}

func (s *Server) getIssueCreatorID(ctx context.Context, email string) int {
	creatorID := api.SystemBotID
	if email != "" {
		committerPrincipal, err := s.store.GetPrincipalByEmail(ctx, email)
		if err != nil {
			log.Warn("Failed to find the principal with committer email, use system bot instead", zap.String("email", email), zap.Error(err))
		} else if committerPrincipal == nil {
			log.Warn("Principal with committer email does not exist, use system bot instead", zap.String("email", email))
		} else {
			creatorID = committerPrincipal.ID
		}
	}
	return creatorID
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
	//              schema file looks like "dev/v1##db1", "staging/v1##db1".
	//
	// Pattern 2: 	Like 1, the database name is the same across all environments. All environment shares the same schema file,
	//              say v1##db1, when a new file is added like v2##db1##add_column, we will create a multi stage pipeline where
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
func getIgnoredFileActivityCreate(projectID int, pushEvent vcs.PushEvent, file string, err error) *api.ActivityCreate {
	payload, marshalErr := json.Marshal(
		api.ActivityProjectRepositoryPushPayload{
			VCSPushEvent: pushEvent,
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
func (s *Server) readFileContent(ctx context.Context, pushEvent vcs.PushEvent, repo *api.Repository, file string) (string, error) {
	// Retrieve the latest AccessToken and RefreshToken as the previous
	// ReadFileContent call may have updated the stored token pair. ReadFileContent
	// will fetch and store the new token pair if the existing token pair has
	// expired.
	repos, err := s.store.FindRepository(ctx, &api.RepositoryFind{WebhookEndpointID: &repo.WebhookEndpointID})
	if err != nil {
		return "", errors.Wrapf(err, "get repository by webhook endpoint %q", repo.WebhookEndpointID)
	} else if len(repos) == 0 {
		return "", errors.Wrapf(err, "repository not found by webhook endpoint %q", repo.WebhookEndpointID)
	}

	repo = repos[0]
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
		pushEvent.CommitList[len(pushEvent.CommitList)-1].ID,
	)
	if err != nil {
		return "", errors.Wrap(err, "read content")
	}
	return content, nil
}

// prepareIssueFromSDLFile returns the migration info and a list of update
// schema details derived from the given push event for SDL.
func (s *Server) prepareIssueFromSDLFile(ctx context.Context, repo *api.Repository, pushEvent vcs.PushEvent, schemaInfo *db.MigrationInfo, file string) ([]*api.MigrationDetail, []*api.ActivityCreate) {
	dbName := schemaInfo.Database
	if dbName == "" {
		log.Debug("Ignored schema file without a database name", zap.String("file", file))
		return nil, nil
	}

	statement, err := s.readFileContent(ctx, pushEvent, repo, file)
	if err != nil {
		activityCreate := getIgnoredFileActivityCreate(repo.ProjectID, pushEvent, file, errors.Wrap(err, "Failed to read file content"))
		return nil, []*api.ActivityCreate{activityCreate}
	}

	activityCreateList := []*api.ActivityCreate{}
	envName := schemaInfo.Environment
	var migrationDetailList []*api.MigrationDetail
	if repo.Project.TenantMode == api.TenantModeTenant {
		migrationDetailList = append(migrationDetailList,
			&api.MigrationDetail{
				// TODO(d): make it to SDL migration.
				MigrationType: db.Migrate,
				DatabaseName:  dbName,
				Statement:     statement,
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
				// TODO(d): make it to SDL migration.
				MigrationType: db.Migrate,
				DatabaseID:    database.ID,
				Statement:     diff,
			},
		)
	}

	return migrationDetailList, activityCreateList
}

// prepareIssueFromFile returns a list of update schema details derived
// from the given push event for DDL.
func (s *Server) prepareIssueFromFile(ctx context.Context, repo *api.Repository, pushEvent vcs.PushEvent, fileName string, fileType vcs.FileItemType, migrationInfo *db.MigrationInfo) ([]*api.MigrationDetail, []*api.ActivityCreate) {
	statement, err := s.readFileContent(ctx, pushEvent, repo, fileName)
	if err != nil {
		activityCreate := getIgnoredFileActivityCreate(repo.ProjectID, pushEvent, fileName, errors.Wrap(err, "Failed to read file content"))
		return nil, []*api.ActivityCreate{activityCreate}
	}

	var migrationDetailList []*api.MigrationDetail

	// TODO(dragonly): handle modified file for tenant mode.
	migrationType := db.Migrate
	if migrationInfo.Type == db.Data {
		migrationType = db.Data
	}
	if repo.Project.TenantMode == api.TenantModeTenant {
		migrationDetailList = append(migrationDetailList,
			&api.MigrationDetail{
				MigrationType: migrationType,
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

	if fileType == vcs.FileItemTypeAdded {
		for _, database := range databases {
			migrationDetailList = append(migrationDetailList,
				&api.MigrationDetail{
					MigrationType: migrationType,
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
