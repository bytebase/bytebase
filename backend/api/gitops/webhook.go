package gitops

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/mail"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/activity"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/vcs"
	"github.com/bytebase/bytebase/backend/store/model"

	// Register azure plugin.
	"github.com/bytebase/bytebase/backend/plugin/vcs/azure"
	"github.com/bytebase/bytebase/backend/plugin/vcs/bitbucket"
	"github.com/bytebase/bytebase/backend/plugin/vcs/github"
	"github.com/bytebase/bytebase/backend/plugin/vcs/gitlab"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

const (
	// issueNameTemplate should be consistent with UI issue names generated from the frontend except for the timestamp.
	// Because we cannot get the correct timezone of the client here.
	// Example: "Alter schema: add an email column".
	issueNameTemplate = "%s: %s"
)

func (s *Service) RegisterWebhookRoutes(g *echo.Group) {
	g.POST(":id", func(c echo.Context) error {
		ctx := c.Request().Context()
		// The id start with "/".
		url := strings.TrimPrefix(c.Param("id"), "/")
		workspaceID, projectID, vcsConnectorID, err := common.GetWorkspaceProjectVCSConnectorID(url)
		if err != nil {
			return c.String(http.StatusOK, fmt.Sprintf("invalid id %q", url))
		}
		myWorkspaceID, err := s.store.GetWorkspaceID(ctx)
		if err != nil {
			return c.String(http.StatusOK, fmt.Sprintf("failed to get workspace ID, error %v", err))
		}
		if myWorkspaceID != workspaceID {
			return c.String(http.StatusOK, fmt.Sprintf("invalid workspace id %q, my ID %q", workspaceID, myWorkspaceID))
		}
		project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{ResourceID: &projectID})
		if err != nil {
			return c.String(http.StatusOK, fmt.Sprintf("failed to get project %q, error %v", projectID, err))
		}
		if project == nil || project.Deleted {
			return c.String(http.StatusOK, fmt.Sprintf("project %q does not exist or has been deleted", projectID))
		}
		vcsConnector, err := s.store.GetVCSConnector(ctx, &store.FindVCSConnectorMessage{ProjectID: &projectID, ResourceID: &vcsConnectorID})
		if err != nil {
			return c.String(http.StatusOK, fmt.Sprintf("failed to get project %q VCS connector %q, error %v", projectID, vcsConnectorID, err))
		}
		if vcsConnector == nil {
			return c.String(http.StatusOK, fmt.Sprintf("project %q VCS connector %q does not exist or has been deleted", projectID, vcsConnectorID))
		}
		vcsProvider, err := s.store.GetVCSProvider(ctx, &store.FindVCSProviderMessage{ResourceID: &vcsConnector.VCSResourceID})
		if err != nil {
			return c.String(http.StatusOK, fmt.Sprintf("failed to get VCS provider %q, error %v", vcsConnector.VCSResourceID, err))
		}
		if vcsProvider == nil {
			return c.String(http.StatusOK, fmt.Sprintf("VCS provider %q does not exist or has been deleted", vcsConnector.VCSResourceID))
		}

		body, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return c.String(http.StatusOK, fmt.Sprintf("failed to read body, error %v", err))
		}

		// Validate webhook secret.
		switch vcsProvider.Type {
		case vcs.GitHub:
			secretToken := c.Request().Header.Get("X-Hub-Signature-256")
			ok, err := validateGitHubWebhookSignature256(secretToken, vcsConnector.Payload.WebhookSecretToken, body)
			if err != nil {
				return c.String(http.StatusOK, fmt.Sprintf("failed to validate webhook signature %q, error %v", secretToken, err))
			}
			if !ok {
				return c.String(http.StatusOK, fmt.Sprintf("invalid webhook secret token %q", secretToken))
			}
		case vcs.GitLab:
			secretToken := c.Request().Header.Get("X-Gitlab-Token")
			if secretToken != vcsConnector.Payload.WebhookSecretToken {
				return c.String(http.StatusOK, fmt.Sprintf("invalid webhook secret token %q", secretToken))
			}
		}

		if vcsProvider.Type == vcs.GitLab {
			prInfo, err := getMergeRequestChangeFileContent(ctx, vcsProvider, vcsConnector, body)
			if err != nil {
				return c.String(http.StatusOK, fmt.Sprintf("failed to get pr info from pull request, error %v", err))
			}
			if err := s.createIssueFromPRInfo(ctx, project, prInfo); err != nil {
				return c.String(http.StatusOK, fmt.Sprintf("failed to create issue from pull request %s, error %v", prInfo.url, err))
			}
		}

		var baseVCSPushEvents []vcs.PushEvent

		switch vcsProvider.Type {
		case vcs.GitHub:
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

			var pushEvent github.WebhookPushEvent
			if err := json.Unmarshal(body, &pushEvent); err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "Malformed push event").SetInternal(err)
			}

			// Check webhook branch.
			ok, err := isWebhookEventBranch(pushEvent.Ref, vcsConnector.Payload.Branch)
			if err != nil {
				return err
			}
			if !ok {
				return c.String(http.StatusOK, fmt.Sprintf("committed to branch %q, want branch %q", pushEvent.Ref, vcsConnector.Payload.Branch))
			}

			baseVCSPushEvents = append(baseVCSPushEvents, pushEvent.ToVCS())
		case vcs.GitLab:
			var pushEvent gitlab.WebhookPushEvent
			if err := json.Unmarshal(body, &pushEvent); err != nil {
				return c.String(http.StatusBadRequest, fmt.Sprintf("failed to unmarshal push event, error %v", err))
			}
			// This shouldn't happen as we only setup webhook to receive push event, just in case.
			if pushEvent.ObjectKind != gitlab.WebhookPush {
				return c.String(http.StatusBadRequest, fmt.Sprintf("Invalid webhook event type, got %s, want push", pushEvent.ObjectKind))
			}

			// Check webhook branch.
			ok, err := isWebhookEventBranch(pushEvent.Ref, vcsConnector.Payload.Branch)
			if err != nil {
				return err
			}
			if !ok {
				return c.String(http.StatusOK, fmt.Sprintf("committed to branch %q, want branch %q", pushEvent.Ref, vcsConnector.Payload.Branch))
			}

			baseVCSPushEvent, err := pushEvent.ToVCS()
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to convert GitLab commits").SetInternal(err)
			}
			baseVCSPushEvents = append(baseVCSPushEvents, baseVCSPushEvent)
		case vcs.Bitbucket:
			// This shouldn't happen as we only set up webhook to receive push event, just in case.
			eventType := c.Request().Header.Get("X-Event-Key")
			if eventType != "repo:push" {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid webhook event type, got %q, want %q", eventType, "repo:push"))
			}

			var pushEvent bitbucket.WebhookPushEvent
			if err := json.Unmarshal(body, &pushEvent); err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "Malformed push event").SetInternal(err)
			}

			for _, change := range pushEvent.Push.Changes {
				// Check webhook branch.
				ref := "refs/heads/" + change.New.Name
				ok, err := isWebhookEventBranch(ref, vcsConnector.Payload.Branch)
				if err != nil {
					return err
				}
				if !ok {
					return c.String(http.StatusOK, fmt.Sprintf("committed to branch %q, want branch %q", ref, vcsConnector.Payload.Branch))
				}

				// Squash commits.
				oauthContext := &common.OauthContext{
					AccessToken: vcsProvider.AccessToken,
				}
				var commitList []vcs.Commit
				for _, commit := range change.Commits {
					before := strings.Repeat("0", 40)
					if len(commit.Parents) > 0 {
						before = commit.Parents[0].Hash
					}
					fileDiffList, err := vcs.Get(vcsProvider.Type, vcs.ProviderConfig{}).GetDiffFileList(
						ctx,
						oauthContext,
						vcsProvider.InstanceURL,
						vcsConnector.Payload.ExternalId,
						before,
						commit.Hash,
					)
					if err != nil {
						return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Failed to get diff file list for commit %q", commit.Hash)).SetInternal(err)
					}

					var addedList, modifiedList []string
					for _, f := range fileDiffList {
						switch f.Type {
						case vcs.FileDiffTypeAdded:
							addedList = append(addedList, f.Path)
						case vcs.FileDiffTypeModified:
							modifiedList = append(modifiedList, f.Path)
						}
					}

					// Per Git convention, the message title and body are separated by two new line characters.
					messages := strings.SplitN(commit.Message, "\n\n", 2)
					messageTitle := messages[0]

					authorName := commit.Author.User.Nickname
					authorEmail := ""
					addr, err := mail.ParseAddress(commit.Author.Raw)
					if err == nil {
						authorName = addr.Name
						authorEmail = addr.Address
					}

					commitList = append(commitList,
						vcs.Commit{
							ID:           commit.Hash,
							Title:        messageTitle,
							Message:      commit.Message,
							CreatedTs:    commit.Date.Unix(),
							URL:          commit.Links.HTML.Href,
							AuthorName:   authorName,
							AuthorEmail:  authorEmail,
							AddedList:    addedList,
							ModifiedList: modifiedList,
						},
					)
				}

				baseVCSPushEvents = append(baseVCSPushEvents, vcs.PushEvent{
					VCSType:            vcs.Bitbucket,
					Ref:                ref,
					Before:             change.Old.Target.Hash,
					After:              change.New.Target.Hash,
					RepositoryID:       pushEvent.Repository.FullName,
					RepositoryURL:      pushEvent.Repository.Links.HTML.Href,
					RepositoryFullPath: pushEvent.Repository.FullName,
					AuthorName:         pushEvent.Actor.Nickname,
					CommitList:         commitList,
				})
			}
		case vcs.AzureDevOps:
			pushEvent := new(azure.ServiceHookCodePushEvent)
			if err := json.Unmarshal(body, pushEvent); err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "Malformed push event").SetInternal(err)
			}

			// Validate presumptions.
			// This shouldn't happen as we only setup webhook to receive push event, just in case.
			if pushEvent.EventType != "git.push" {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Presumption failed: invalid webhook event type, got %s, want git.push", pushEvent.EventType))
			}
			// All the examples on https://learn.microsoft.com/en-us/azure/devops/service-hooks/events?view=azure-devops#code-pushed
			// have only one ref update.
			if len(pushEvent.Resource.RefUpdates) != 1 {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Presumption failed: the number of ref updates is not 1, got %d", len(pushEvent.Resource.RefUpdates)))
			}
			if pushEvent.Resource.RefUpdates[0].NewObjectID == strings.Repeat("0", 40) {
				// Users delete branch will trigger a push event, but the refUpdates.NewObjectID is all zero.
				return c.String(http.StatusOK, "Do not process the push event which is generated by deleting branch")
			}

			// Check webhook branch.
			refUpdate := pushEvent.Resource.RefUpdates[0]
			ok, err := isWebhookEventBranch(refUpdate.Name, vcsConnector.Payload.Branch)
			if err != nil {
				return err
			}
			if !ok {
				return c.String(http.StatusOK, fmt.Sprintf("committed to branch %q, want branch %q", refUpdate, vcsConnector.Payload.Branch))
			}

			// Squash commits.
			oauthContext := &common.OauthContext{
				AccessToken: vcsProvider.AccessToken,
			}
			if len(pushEvent.Resource.Commits) == 0 {
				// If users merge one branch to our target branch, the commits list is empty in push event.
				// And we cannot get the commits in range [refUpdates.oldObjectId, refUpdates.newObjectId] from Azure DevOps API.
				// So, when the commit list is empty, we think it is generated by merge,
				// then we need to query the queryPullRequest API to find out if the updateRef.newObjectId
				// is the last commit of the Pull Request, and then we can use the PullRef.newObjectId API
				// to find out if it is the last commit of the Pull Request. Request ID to list the corresponding commits.
				probablePullRequests, err := azure.QueryPullRequest(ctx, oauthContext, vcsConnector.Payload.ExternalId, pushEvent.Resource.RefUpdates[0].NewObjectID)
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, "Failed to query pull request").SetInternal(err)
				}
				var filterOutPullRequestList []*azure.PullRequest
				for _, pullRequest := range probablePullRequests {
					if pullRequest.Status != "completed" {
						continue
					}
					// We only handle the pull request which is merged to the target branch.
					// NOTE: here we should compare with the refUpdates.Name instead of the repository.branch.
					if pullRequest.TargetRefName != pushEvent.Resource.RefUpdates[0].Name {
						continue
					}
					filterOutPullRequestList = append(filterOutPullRequestList, pullRequest)
				}

				if len(filterOutPullRequestList) == 0 {
					return echo.NewHTTPError(http.StatusOK, "No pull request matched")
				}
				if len(filterOutPullRequestList) != 1 {
					return echo.NewHTTPError(http.StatusInternalServerError, errors.Errorf("Expected only one pull request, but got %d, content: %+v", len(filterOutPullRequestList), filterOutPullRequestList).Error())
				}
				// We should backfill the commit list by the commits in the pull request.
				pullRequest := filterOutPullRequestList[0]
				commitsInPullRequest, err := azure.GetPullRequestCommits(ctx, oauthContext, vcsConnector.Payload.ExternalId, pullRequest.ID)
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get pull request commits").SetInternal(err)
				}
				pushEvent.Resource.Commits = commitsInPullRequest
			}
			// For Azure DevOps, if users create a new branch and push it to the remote. It will trigger a code push event,
			// but do not contain any commits in the event. So we do not need to consider the case that the commit id is
			// all zero.
			for _, commit := range pushEvent.Resource.Commits {
				if commit.CommitID == strings.Repeat("0", 40) {
					return echo.NewHTTPError(http.StatusBadRequest, "Presumption failed: the commit id is all zero")
				}
			}
			// Azure DevOps' service hook does not contain the file diff information for each commit, so we need to backfill
			// the file diff information by ourselves.
			// We will use the previous commit id as the base commit id and the current commit id as the target commit id to
			// get the file diff information commit by commit. We use the oldObjectId in resources.refUpdates as the base commit id
			// for the first commit.
			// NOTE: We presume that the sequence of the commits in the code push event is the reverse order of the commit sequence(aka. stack sequence, commit first, appear last) in the repository.
			var backfillCommits []vcs.Commit
			for _, commit := range pushEvent.Resource.Commits {
				changes, err := azure.GetChangesByCommit(ctx, oauthContext, vcsConnector.Payload.ExternalId, commit.CommitID)
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get changes by commit %q", commit.CommitID)).SetInternal(err)
				}

				var addedList, modifiedList []string
				for _, change := range changes.Changes {
					switch change.ChangeType {
					case "add":
						addedList = append(addedList, change.Item.Path)
					case "edit":
						modifiedList = append(modifiedList, change.Item.Path)
					case "rename":
						// To be consistent with VCS, we treat rename as delete + add, but we do not need to handle delete here.
						addedList = append(addedList, change.Item.Path)
					}
				}

				backfillCommits = append(backfillCommits, vcs.Commit{
					ID:           commit.CommitID,
					Title:        commit.Comment,
					Message:      commit.Comment,
					CreatedTs:    commit.Author.Date.Unix(),
					URL:          commit.URL,
					AuthorName:   commit.Author.Name,
					AuthorEmail:  commit.Author.Email,
					AddedList:    addedList,
					ModifiedList: modifiedList,
				})
			}
			if len(backfillCommits) == 1 && len(backfillCommits[0].AddedList) == 1 && strings.HasSuffix(backfillCommits[0].AddedList[0], azure.SQLReviewPipelineFilePath) {
				slog.Debug("start to setup pipeline", slog.String("repository", vcsConnector.Payload.ExternalId))
				// Use workspaceID instead of repo secret as SQL review pipeline token, so that we don't need to re-create the pipeline if users disable then enable the CI.
				workspaceID, err := s.store.GetWorkspaceID(ctx)
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get workspace id with error").SetInternal(err)
				}
				// Setup SQL review pipeline and policy.
				if err := azure.EnableSQLReviewCI(ctx, oauthContext, vcsConnector.Payload.Title, vcsConnector.Payload.ExternalId, vcsConnector.Payload.Branch, workspaceID); err != nil {
					slog.Error("failed to setup pipeline", log.BBError(err), slog.String("repository", vcsConnector.Payload.ExternalId))
					return echo.NewHTTPError(http.StatusInternalServerError, "Failed to setup SQL review pipeline").SetInternal(err)
				}
				return c.String(http.StatusOK, "OK")
			}

			// Backfill web url for commits.
			if pushEvent.Resource.PushID != 0 {
				commitsInPush, err := azure.GetPushCommitsByPushID(ctx, oauthContext, vcsConnector.Payload.ExternalId, pushEvent.Resource.PushID)
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get push commits by push id %d", pushEvent.Resource.PushID)).SetInternal(err)
				}
				// Convert values to map from commitID to remote url.
				commitID2Value := make(map[string]string)
				for _, value := range commitsInPush.Value {
					commitID2Value[value.CommitID] = value.RemoteURL
				}
				for i := range backfillCommits {
					if remoteURL, ok := commitID2Value[backfillCommits[i].ID]; ok {
						backfillCommits[i].URL = remoteURL
					}
				}
			}

			baseVCSPushEvents = append(baseVCSPushEvents, vcs.PushEvent{
				VCSType:            vcs.AzureDevOps,
				Ref:                pushEvent.Resource.RefUpdates[0].Name,
				Before:             pushEvent.Resource.RefUpdates[0].OldObjectID,
				After:              pushEvent.Resource.RefUpdates[0].NewObjectID,
				RepositoryID:       vcsConnector.Payload.ExternalId,
				RepositoryURL:      vcsConnector.Payload.WebUrl,
				RepositoryFullPath: vcsConnector.Payload.ExternalId,
				AuthorName:         pushEvent.Resource.PushedBy.DisplayName,
				CommitList:         backfillCommits,
			})
		}

		repo := &repoInfo{
			repository: vcsConnector,
			project:    project,
			vcs:        vcsProvider,
		}
		oauthContext := &common.OauthContext{
			AccessToken: vcsProvider.AccessToken,
		}
		var allCreatedMessages []string
		for _, baseVCSPushEvent := range baseVCSPushEvents {
			createdMessage, err := s.processPushEvent(ctx, oauthContext, repo, baseVCSPushEvent)
			if err != nil {
				return err
			}
			allCreatedMessages = append(allCreatedMessages, createdMessage)
		}
		return c.String(http.StatusOK, strings.Join(allCreatedMessages, "\n"))
	})
}

type repoInfo struct {
	repository *store.VCSConnectorMessage
	project    *store.ProjectMessage
	vcs        *store.VCSProviderMessage
}

func isWebhookEventBranch(pushEventRef, wantBranch string) (bool, error) {
	branch, err := parseBranchNameFromRefs(pushEventRef)
	if err != nil {
		return false, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid ref: %s", pushEventRef)).SetInternal(err)
	}
	if branch != wantBranch {
		slog.Debug("Skipping repo due to branch filter mismatch", slog.String("branch", branch), slog.String("wantBranch", wantBranch))
		return false, nil
	}
	return true, nil
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
		slog.Debug(
			"ref is not prefix with expected prefix",
			slog.String("ref", ref),
			slog.String("expected prefix", expectedPrefix),
		)
		return ref, errors.Errorf("unexpected ref name %q without prefix %q", ref, expectedPrefix)
	}
	return ref[len(expectedPrefix):], nil
}

func (s *Service) processPushEvent(ctx context.Context, oauthContext *common.OauthContext, repo *repoInfo, pushEvent vcs.PushEvent) (string, error) {
	pushEvent.VCSType = repo.vcs.Type
	distinctFileList := pushEvent.GetDistinctFileList()
	if len(distinctFileList) == 0 {
		var commitIDs []string
		for _, c := range pushEvent.CommitList {
			commitIDs = append(commitIDs, c.ID)
		}
		slog.Warn("No files found from the push event",
			slog.String("repoURL", pushEvent.RepositoryURL),
			slog.String("repoName", pushEvent.RepositoryFullPath),
			slog.String("commits", strings.Join(commitIDs, ",")))
		return "", nil
	}

	filteredDistinctFileList, err := func() ([]vcs.DistinctFileItem, error) {
		// The before commit ID is all zeros when the branch is just created and contains no commits yet.
		if pushEvent.Before == strings.Repeat("0", 40) {
			return distinctFileList, nil
		}
		return filterFilesByCommitsDiff(ctx, oauthContext, repo, distinctFileList, pushEvent.Before, pushEvent.After)
	}()
	if err != nil {
		return "", errors.Wrapf(err, "failed to filtered distinct files by commits diff")
	}

	fileItemList := getFileInfoList(filteredDistinctFileList, repo)

	var migrationDetailList []*migrationDetail
	var activityCreateList []*store.ActivityMessage
	var createdIssueList []string
	var fileNameList []string

	creatorID := s.getIssueCreatorID(ctx, pushEvent.CommitList[0].AuthorEmail)
	for _, fileInfo := range fileItemList {
		// This is a migration-based DDL or DML file.
		migrationDetailListForFile, activityCreateListForFile := s.prepareIssueFromFile(ctx, oauthContext, repo, pushEvent, fileInfo)
		activityCreateList = append(activityCreateList, activityCreateListForFile...)
		migrationDetailList = append(migrationDetailList, migrationDetailListForFile...)
		if len(migrationDetailListForFile) != 0 {
			fileNameList = append(fileNameList, strings.TrimPrefix(fileInfo.item.FileName, repo.repository.Payload.BaseDirectory+"/"))
		}
	}
	if len(migrationDetailList) == 0 {
		return "", nil
	}

	// Create one issue per push event for DDL project, or non-schema files for SDL project.
	migrateType := "Change data"
	for _, d := range migrationDetailList {
		if d.migrationType == db.Migrate {
			migrateType = "Alter schema"
			break
		}
	}
	description := strings.ReplaceAll(fileItemList[0].migrationInfo.Description, "_", " ")
	issueName := fmt.Sprintf(issueNameTemplate, migrateType, description)
	issueDescription := fmt.Sprintf("By VCS files:\n\n%s\n", strings.Join(fileNameList, "\n"))
	if err := s.createIssueFromMigrationDetailsV2(ctx, repo.project, issueName, issueDescription, pushEvent, creatorID, migrationDetailList); err != nil {
		return "", echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create issue %s, error %v", issueName, err)).SetInternal(err)
	}

	for _, activityCreate := range activityCreateList {
		if _, err := s.activityManager.CreateActivity(ctx, activityCreate, &activity.Metadata{}); err != nil {
			slog.Warn("Failed to create project activity for the ignored repository files", log.BBError(err))
		}
	}
	return fmt.Sprintf("Created issue %q from push event", strings.Join(createdIssueList, ",")), nil
}

// Users may merge commits from other branches,
// and some of the commits merged in may already be merged into the main branch.
// In that case, the commits in the push event contains files which are not added in this PR/MR.
// We use the compare API to get the file diffs and filter files by the diffs.
// TODO(dragonly): generate distinct file change list from the commits diff instead of filter.
func filterFilesByCommitsDiff(
	ctx context.Context,
	oauthContext *common.OauthContext,
	repoInfo *repoInfo,
	distinctFileList []vcs.DistinctFileItem,
	beforeCommit, afterCommit string,
) ([]vcs.DistinctFileItem, error) {
	fileDiffList, err := vcs.Get(repoInfo.vcs.Type, vcs.ProviderConfig{}).GetDiffFileList(
		ctx,
		oauthContext,
		repoInfo.vcs.InstanceURL,
		repoInfo.repository.Payload.ExternalId,
		beforeCommit,
		afterCommit,
	)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get file diff list for repository %s", repoInfo.repository.Payload.ExternalId)
	}
	var filteredDistinctFileList []vcs.DistinctFileItem
	for _, file := range distinctFileList {
		for _, diff := range fileDiffList {
			if file.FileName == diff.Path {
				filteredDistinctFileList = append(filteredDistinctFileList, file)
				break
			}
		}
	}
	return filteredDistinctFileList, nil
}

type fileInfo struct {
	item          vcs.DistinctFileItem
	migrationInfo *db.MigrationInfo
}

func getFileInfoList(distinctFileList []vcs.DistinctFileItem, repo *repoInfo) []fileInfo {
	var result []fileInfo
	for _, item := range distinctFileList {
		slog.Debug("Processing file", slog.String("file", item.FileName), slog.String("commit", item.Commit.ID))
		var migrationInfo *db.MigrationInfo
		if filepath.Dir(item.FileName) != repo.repository.Payload.BaseDirectory {
			continue
		}
		filename := filepath.Base(item.FileName)
		if filepath.Ext(filename) != ".sql" {
			continue
		}
		filename = strings.TrimSuffix(filename, filepath.Ext(filename))
		re := regexp.MustCompile(`^[0-9]+`)
		matches := re.FindAllString(filename, -1)
		if len(matches) == 0 {
			continue
		}
		version := matches[0]
		description := strings.TrimPrefix(filename, version)
		description = strings.TrimLeft(description, "_#")
		migrationType := db.Migrate
		if strings.Contains(description, "dml") || strings.Contains(description, "data") {
			migrationType = db.Data
		}
		description = strings.TrimPrefix(description, "ddl")
		description = strings.TrimPrefix(description, "dml")
		description = strings.TrimPrefix(description, "migrate")
		description = strings.TrimPrefix(description, "data")
		description = strings.TrimLeft(description, "_#")
		migrationInfo = &db.MigrationInfo{
			Version:     model.Version{Version: version},
			Type:        migrationType,
			Description: description,
		}
		result = append(result, fileInfo{
			item:          item,
			migrationInfo: migrationInfo,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		mi := result[i].migrationInfo
		mj := result[j].migrationInfo
		return mi.Version.Version < mj.Version.Version
	})
	return result
}

func (s *Service) createIssueFromMigrationDetailsV2(ctx context.Context, project *store.ProjectMessage, issueName, issueDescription string, pushEvent vcs.PushEvent, creatorID int, migrationDetailList []*migrationDetail) error {
	user, err := s.store.GetUserByID(ctx, creatorID)
	if err != nil {
		return errors.Wrapf(err, "failed to get user %v", creatorID)
	}

	var steps []*v1pb.Plan_Step

	environments, err := s.store.ListEnvironmentV2(ctx, &store.FindEnvironmentMessage{})
	if err != nil {
		return err
	}
	orderIndex := make(map[int32]int)
	for i, environment := range environments {
		orderIndex[environment.Order] = i
	}
	allSteps := make([]*v1pb.Plan_Step, len(environments))
	for _, migrationDetail := range migrationDetailList {
		database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: &migrationDetail.databaseID})
		if err != nil {
			return err
		}
		if database == nil {
			return errors.Errorf("database %d not found", migrationDetail.databaseID)
		}
		environment, err := s.store.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{ResourceID: &database.EffectiveEnvironmentID})
		if err != nil {
			return err
		}
		if environment == nil {
			return errors.Errorf("environment %q not found", database.EffectiveEnvironmentID)
		}

		step := allSteps[orderIndex[environment.Order]]
		if step == nil {
			allSteps[orderIndex[environment.Order]] = &v1pb.Plan_Step{}
			step = allSteps[orderIndex[environment.Order]]
		}
		step.Specs = append(step.Specs, &v1pb.Plan_Spec{
			Id: uuid.NewString(),
			Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
				ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
					Type:          getChangeType(migrationDetail.migrationType),
					Target:        common.FormatDatabase(database.InstanceID, database.DatabaseName),
					Sheet:         fmt.Sprintf("projects/%s/sheets/%d", project.ResourceID, migrationDetail.sheetID),
					SchemaVersion: migrationDetail.schemaVersion.Version,
				},
			},
		})
	}
	for _, step := range allSteps {
		if step != nil && len(step.Specs) > 0 {
			steps = append(steps, step)
		}
	}
	childCtx := context.WithValue(ctx, common.PrincipalIDContextKey, creatorID)
	childCtx = context.WithValue(childCtx, common.UserContextKey, user)
	childCtx = context.WithValue(childCtx, common.LoopbackContextKey, true)
	plan, err := s.rolloutService.CreatePlan(childCtx, &v1pb.CreatePlanRequest{
		Parent: fmt.Sprintf("projects/%s", project.ResourceID),
		Plan: &v1pb.Plan{
			Title: issueName,
			Steps: steps,
		},
	})
	if err != nil {
		return errors.Wrapf(err, "failed to create plan")
	}
	issue, err := s.issueService.CreateIssue(childCtx, &v1pb.CreateIssueRequest{
		Parent: fmt.Sprintf("projects/%s", project.ResourceID),
		Issue: &v1pb.Issue{
			Title:       issueName,
			Description: issueDescription,
			Type:        v1pb.Issue_DATABASE_CHANGE,
			Assignee:    fmt.Sprintf("users/%s", api.SystemBotEmail),
			Plan:        plan.Name,
		},
	})
	if err != nil {
		return errors.Wrapf(err, "failed to create issue")
	}
	if _, err := s.rolloutService.CreateRollout(childCtx, &v1pb.CreateRolloutRequest{
		Parent: fmt.Sprintf("projects/%s", project.ResourceID),
		Rollout: &v1pb.Rollout{
			Plan: plan.Name,
		},
	}); err != nil {
		return errors.Wrapf(err, "failed to create rollout")
	}

	issueUID, err := strconv.Atoi(issue.Uid)
	if err != nil {
		return err
	}
	// Create a project activity after successfully creating the issue from the push event.
	activityPayload, err := json.Marshal(
		api.ActivityProjectRepositoryPushPayload{
			VCSPushEvent: pushEvent,
			IssueID:      issueUID,
			IssueName:    issue.Title,
		},
	)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to construct activity payload").SetInternal(err)
	}

	activityCreate := &store.ActivityMessage{
		CreatorUID:        creatorID,
		ResourceContainer: project.GetName(),
		ContainerUID:      project.UID,
		Type:              api.ActivityProjectRepositoryPush,
		Level:             api.ActivityInfo,
		Comment:           fmt.Sprintf("Created issue %q.", issue.Title),
		Payload:           string(activityPayload),
	}
	if _, err := s.activityManager.CreateActivity(ctx, activityCreate, &activity.Metadata{}); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create project activity after creating issue from repository push event: %d", issueUID)).SetInternal(err)
	}

	return nil
}

func getChangeType(migrationType db.MigrationType) v1pb.Plan_ChangeDatabaseConfig_Type {
	switch migrationType {
	case db.Baseline:
		return v1pb.Plan_ChangeDatabaseConfig_BASELINE
	case db.Migrate:
		return v1pb.Plan_ChangeDatabaseConfig_MIGRATE
	case db.MigrateSDL:
		return v1pb.Plan_ChangeDatabaseConfig_MIGRATE_SDL
	case db.Data:
		return v1pb.Plan_ChangeDatabaseConfig_DATA
	}
	return v1pb.Plan_ChangeDatabaseConfig_TYPE_UNSPECIFIED
}

func (s *Service) getIssueCreatorID(ctx context.Context, email string) int {
	creatorID := api.SystemBotID
	if email != "" {
		committerPrincipal, err := s.store.GetUser(ctx, &store.FindUserMessage{
			Email: &email,
		})
		if err != nil {
			slog.Warn("Failed to find the principal with committer email, use system bot instead", slog.String("email", email), log.BBError(err))
		} else if committerPrincipal == nil {
			slog.Info("Principal with committer email does not exist, use system bot instead", slog.String("email", email))
		} else {
			creatorID = committerPrincipal.ID
		}
	}
	return creatorID
}

func (s *Service) findProjectDatabases(ctx context.Context, projectID int) ([]*store.DatabaseMessage, error) {
	// Retrieve the current schema from the database
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{UID: &projectID})
	if err != nil {
		return nil, err
	}
	if project == nil {
		return nil, errors.Errorf("project %d not found", projectID)
	}
	// The database name for PostgreSQL, Oracle, Snowflake and some databases are case sensitive.
	// But the database name for MySQL, TiDB and other databases are case insensitive.
	// So we should find databases by case-insensitive and double-check for case sensitive database engines.
	databases, err := s.store.ListDatabases(ctx,
		&store.FindDatabaseMessage{
			ProjectID: &project.ResourceID,
		},
	)
	if err != nil {
		return nil, errors.Wrap(err, "find database")
	}
	return databases, nil
}

// getIgnoredFileActivityCreate get a warning project activityCreate for the ignored file with given error.
func getIgnoredFileActivityCreate(project *store.ProjectMessage, pushEvent vcs.PushEvent, file string, err error) *store.ActivityMessage {
	payload, marshalErr := json.Marshal(
		api.ActivityProjectRepositoryPushPayload{
			VCSPushEvent: pushEvent,
		},
	)
	if marshalErr != nil {
		slog.Warn("Failed to construct project activity payload for the ignored repository file",
			log.BBError(marshalErr),
		)
		return nil
	}

	return &store.ActivityMessage{
		CreatorUID:        api.SystemBotID,
		ResourceContainer: project.GetName(),
		ContainerUID:      project.UID,
		Type:              api.ActivityProjectRepositoryPush,
		Level:             api.ActivityWarn,
		Comment:           fmt.Sprintf("Ignored file %q, %v.", file, err),
		Payload:           string(payload),
	}
}

// readFileContent reads the content of the given file from the given repository.
func readFileContent(ctx context.Context, oauthContext *common.OauthContext, pushEvent vcs.PushEvent, repo *repoInfo, file string) (string, error) {
	content, err := vcs.Get(repo.vcs.Type, vcs.ProviderConfig{}).ReadFileContent(
		ctx,
		oauthContext,
		repo.vcs.InstanceURL,
		repo.repository.Payload.ExternalId,
		file,
		vcs.RefInfo{
			RefType: vcs.RefTypeCommit,
			RefName: pushEvent.CommitList[len(pushEvent.CommitList)-1].ID,
		},
	)
	if err != nil {
		return "", errors.Wrap(err, "read content")
	}
	return content, nil
}

// prepareIssueFromFile returns a list of update schema details derived
// from the given push event for DDL.
func (s *Service) prepareIssueFromFile(
	ctx context.Context,
	oauthContext *common.OauthContext,
	repoInfo *repoInfo,
	pushEvent vcs.PushEvent,
	fileInfo fileInfo,
) ([]*migrationDetail, []*store.ActivityMessage) {
	content, err := readFileContent(ctx, oauthContext, pushEvent, repoInfo, fileInfo.item.FileName)
	if err != nil {
		return nil, []*store.ActivityMessage{
			getIgnoredFileActivityCreate(
				repoInfo.project,
				pushEvent,
				fileInfo.item.FileName,
				errors.Wrap(err, "Failed to read file content"),
			),
		}
	}

	sheetPayload := &storepb.SheetPayload{
		VcsPayload: &storepb.SheetPayload_VCSPayload{
			PushEvent: utils.ConvertVcsPushEvent(&pushEvent),
		},
	}

	databases, err := s.findProjectDatabases(ctx, repoInfo.project.UID)
	if err != nil {
		activityCreate := getIgnoredFileActivityCreate(repoInfo.project, pushEvent, fileInfo.item.FileName, errors.Wrap(err, "Failed to find project databases"))
		return nil, []*store.ActivityMessage{activityCreate}
	}

	if fileInfo.item.ItemType == vcs.FileItemTypeAdded {
		sheet, err := s.store.CreateSheet(ctx, &store.SheetMessage{
			CreatorID:  api.SystemBotID,
			ProjectUID: repoInfo.project.UID,
			Title:      fileInfo.item.FileName,
			Statement:  content,
			Payload:    sheetPayload,
		})
		if err != nil {
			activityCreate := getIgnoredFileActivityCreate(repoInfo.project, pushEvent, fileInfo.item.FileName, errors.Wrap(err, "Failed to create a sheet"))
			return nil, []*store.ActivityMessage{activityCreate}
		}

		var migrationDetailList []*migrationDetail
		for _, database := range databases {
			migrationDetailList = append(migrationDetailList,
				&migrationDetail{
					migrationType: fileInfo.migrationInfo.Type,
					databaseID:    database.UID,
					sheetID:       sheet.UID,
					schemaVersion: model.Version{Version: fmt.Sprintf("%s-%s", fileInfo.migrationInfo.Version.Version, fileInfo.migrationInfo.Type.GetVersionTypeSuffix())},
				},
			)
		}
		return migrationDetailList, nil
	}

	migrationVersion := fmt.Sprintf("%s-%s", fileInfo.migrationInfo.Version.Version, fileInfo.migrationInfo.Type.GetVersionTypeSuffix())
	if err := s.tryUpdateTasksFromModifiedFile(ctx, databases, fileInfo.item.FileName, migrationVersion, content, pushEvent); err != nil {
		return nil, []*store.ActivityMessage{
			getIgnoredFileActivityCreate(
				repoInfo.project,
				pushEvent,
				fileInfo.item.FileName,
				errors.Wrap(err, "Failed to find project task"),
			),
		}
	}
	return nil, nil
}

func (s *Service) tryUpdateTasksFromModifiedFile(ctx context.Context, databases []*store.DatabaseMessage, fileName, schemaVersion, statement string, pushEvent vcs.PushEvent) error {
	// For modified files, we try to update the existing issue's statement.
	for _, database := range databases {
		taskList, err := s.store.ListTasks(ctx, &api.TaskFind{
			DatabaseID:              &database.UID,
			LatestTaskRunStatusList: &[]api.TaskRunStatus{api.TaskRunNotStarted, api.TaskRunFailed},
			TypeList:                &[]api.TaskType{api.TaskDatabaseSchemaUpdate, api.TaskDatabaseDataUpdate},
			Payload:                 fmt.Sprintf("task.payload->>'schemaVersion' = '%s'", schemaVersion),
		})
		if err != nil {
			return err
		}
		if len(taskList) == 0 {
			continue
		}
		if len(taskList) > 1 {
			slog.Error("Found more than one pending approval or failed tasks for modified VCS file, should be only one task.", slog.Int("databaseID", database.UID), slog.String("schemaVersion", schemaVersion))
			return nil
		}
		task := taskList[0]
		issue, err := s.store.GetIssueV2(ctx, &store.FindIssueMessage{PipelineID: &task.PipelineID})
		if err != nil {
			slog.Error("failed to get issue by pipeline ID", slog.Int("pipeline ID", task.PipelineID), log.BBError(err))
			return nil
		}
		if issue == nil {
			slog.Error("issue not found by pipeline ID", slog.Int("pipeline ID", task.PipelineID))
			return nil
		}

		sheetPayload := &storepb.SheetPayload{
			VcsPayload: &storepb.SheetPayload_VCSPayload{
				PushEvent: utils.ConvertVcsPushEvent(&pushEvent),
			},
		}
		sheet, err := s.store.CreateSheet(ctx, &store.SheetMessage{
			CreatorID:  api.SystemBotID,
			ProjectUID: issue.Project.UID,
			Title:      fileName,
			Statement:  statement,
			Payload:    sheetPayload,
		})
		if err != nil {
			return err
		}

		// TODO(dragonly): Try to patch the failed migration history record to pending, and the statement to the current modified file content.
		slog.Debug("Patching task for modified file VCS push event", slog.String("fileName", fileName), slog.Int("issueID", issue.UID), slog.Int("taskID", task.ID))
		taskPatch := api.TaskPatch{
			ID:        task.ID,
			SheetID:   &sheet.UID,
			UpdaterID: api.SystemBotID,
		}
		if err := patchTask(ctx, s.store, s.activityManager, task, &taskPatch, issue); err != nil {
			slog.Error("Failed to patch task with the same migration version", slog.Int("issueID", issue.UID), slog.Int("taskID", task.ID), log.BBError(err))
			return nil
		}

		if issue.PlanUID != nil {
			plan, err := s.store.GetPlan(ctx, &store.FindPlanMessage{UID: issue.PlanUID})
			if err != nil {
				slog.Error("failed to get plan", slog.Int64("plan ID", *issue.PlanUID), log.BBError(err))
			}
			for _, step := range plan.Config.Steps {
				for _, spec := range step.Specs {
					v, ok := spec.Config.(*storepb.PlanConfig_Spec_ChangeDatabaseConfig)
					if !ok {
						continue
					}
					if v.ChangeDatabaseConfig.SchemaVersion == schemaVersion && v.ChangeDatabaseConfig.Target == common.FormatDatabase(database.InstanceID, database.DatabaseName) {
						v.ChangeDatabaseConfig.Sheet = fmt.Sprintf("projects/%s/sheets/%d", issue.Project.ResourceID, sheet.UID)
					}
				}
			}
			if err := s.store.UpdatePlan(ctx, &store.UpdatePlanMessage{
				UID:       *issue.PlanUID,
				Config:    plan.Config,
				UpdaterID: api.SystemBotID,
			}); err != nil {
				slog.Error("failed to update plan", slog.Int64("plan ID", *issue.PlanUID), log.BBError(err))
			}
		}

		// dismiss stale review, re-find the approval template
		// it's ok if we failed
		if err := func() error {
			issue, err := s.store.UpdateIssueV2(ctx, issue.UID, &store.UpdateIssueMessage{
				PayloadUpsert: &storepb.IssuePayload{
					Approval: &storepb.IssuePayloadApproval{
						ApprovalFindingDone: false,
					},
				},
			}, api.SystemBotID)
			if err != nil {
				return errors.Wrap(err, "failed to update issue payload")
			}
			s.stateCfg.ApprovalFinding.Store(issue.UID, issue)
			return nil
		}(); err != nil {
			slog.Error("Failed to dismiss stale review", log.BBError(err))
		}
	}
	return nil
}

// patchTask patches the statement for a task.
func patchTask(ctx context.Context, stores *store.Store, activityManager *activity.Manager, task *store.TaskMessage, taskPatch *api.TaskPatch, issue *store.IssueMessage) error {
	taskPatched, err := stores.UpdateTaskV2(ctx, taskPatch)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to update task \"%v\"", task.Name)).SetInternal(err)
	}
	if taskPatch.SheetID != nil {
		oldSheetID, err := utils.GetTaskSheetID(task.Payload)
		if err != nil {
			return errors.Wrap(err, "failed to get old sheet ID")
		}
		newSheetID := *taskPatch.SheetID

		// create a task sheet update activity
		payload, err := json.Marshal(api.ActivityPipelineTaskStatementUpdatePayload{
			TaskID:     taskPatched.ID,
			OldSheetID: oldSheetID,
			NewSheetID: newSheetID,
			TaskName:   task.Name,
			IssueName:  issue.Title,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create activity after updating task sheet: %v", taskPatched.Name).SetInternal(err)
		}
		if _, err := activityManager.CreateActivity(ctx, &store.ActivityMessage{
			CreatorUID:        taskPatch.UpdaterID,
			ResourceContainer: issue.Project.GetName(),
			ContainerUID:      taskPatched.PipelineID,
			Type:              api.ActivityPipelineTaskStatementUpdate,
			Payload:           string(payload),
			Level:             api.ActivityInfo,
		}, &activity.Metadata{
			Issue: issue,
		}); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create activity after updating task statement: %v", taskPatched.Name)).SetInternal(err)
		}
	}
	return nil
}

func (s *Service) createIssueFromPRInfo(ctx context.Context, project *store.ProjectMessage, prInfo *pullRequestInfo) error {
	creatorID := api.SystemBotID
	user, err := s.store.GetUser(ctx, &store.FindUserMessage{Email: &prInfo.email})
	if err != nil {
		slog.Error("failed to find user by email", slog.String("email", prInfo.email), log.BBError(err))
	}
	if user != nil {
		creatorID = user.ID
	}

	databases, err := s.store.ListDatabases(ctx, &store.FindDatabaseMessage{ProjectID: &project.ResourceID})
	if err != nil {
		return err
	}
	environments, err := s.store.ListEnvironmentV2(ctx, &store.FindEnvironmentMessage{})
	if err != nil {
		return err
	}
	environmentOrders := make(map[string]int32)
	for _, environment := range environments {
		environmentOrders[environment.ResourceID] = environment.Order
	}
	sort.Slice(databases, func(i, j int) bool {
		return environmentOrders[databases[i].EffectiveEnvironmentID] < environmentOrders[databases[j].EffectiveEnvironmentID]
	})

	var sheets []int
	for _, change := range prInfo.changes {
		sheet, err := s.store.CreateSheet(ctx, &store.SheetMessage{
			CreatorID:  creatorID,
			ProjectUID: project.UID,
			Title:      change.path,
			Statement:  change.content,
		})
		if err != nil {
			return errors.Wrapf(err, "failed to create sheet for file %s", change.path)
		}
		sheets = append(sheets, sheet.UID)
	}

	var steps []*v1pb.Plan_Step
	for i, database := range databases {
		if i == 0 || databases[i].EffectiveEnvironmentID != databases[i-1].EffectiveEnvironmentID {
			steps = append(steps, &v1pb.Plan_Step{})
		}
		step := steps[len(steps)-1]
		for i, change := range prInfo.changes {
			step.Specs = append(step.Specs, &v1pb.Plan_Spec{
				Id: uuid.NewString(),
				Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
					ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
						Type:          change.changeType,
						Target:        common.FormatDatabase(database.InstanceID, database.DatabaseName),
						Sheet:         fmt.Sprintf("projects/%s/sheets/%d", project.ResourceID, sheets[i]),
						SchemaVersion: change.version,
					},
				},
			})
		}
	}

	childCtx := context.WithValue(ctx, common.PrincipalIDContextKey, creatorID)
	childCtx = context.WithValue(childCtx, common.UserContextKey, user)
	childCtx = context.WithValue(childCtx, common.LoopbackContextKey, true)
	plan, err := s.rolloutService.CreatePlan(childCtx, &v1pb.CreatePlanRequest{
		Parent: fmt.Sprintf("projects/%s", project.ResourceID),
		Plan: &v1pb.Plan{
			Title: prInfo.title,
			Steps: steps,
		},
	})
	if err != nil {
		return errors.Wrapf(err, "failed to create plan")
	}
	issue, err := s.issueService.CreateIssue(childCtx, &v1pb.CreateIssueRequest{
		Parent: fmt.Sprintf("projects/%s", project.ResourceID),
		Issue: &v1pb.Issue{
			Title:       prInfo.title,
			Description: prInfo.description,
			Type:        v1pb.Issue_DATABASE_CHANGE,
			Assignee:    fmt.Sprintf("users/%s", api.SystemBotEmail),
			Plan:        plan.Name,
		},
	})
	if err != nil {
		return errors.Wrapf(err, "failed to create issue")
	}
	if _, err := s.rolloutService.CreateRollout(childCtx, &v1pb.CreateRolloutRequest{
		Parent: fmt.Sprintf("projects/%s", project.ResourceID),
		Rollout: &v1pb.Rollout{
			Plan: plan.Name,
		},
	}); err != nil {
		return errors.Wrapf(err, "failed to create rollout")
	}

	issueUID, err := strconv.Atoi(issue.Uid)
	if err != nil {
		return err
	}
	// Create a project activity after successfully creating the issue from the push event.
	activityPayload, err := json.Marshal(
		api.ActivityProjectRepositoryPushPayload{
			// TODO(d): redefine VCS push event.
			IssueID:   issueUID,
			IssueName: issue.Title,
		},
	)
	if err != nil {
		return errors.Wrapf(err, "failed to create activity payload")
	}

	activityCreate := &store.ActivityMessage{
		CreatorUID:        creatorID,
		ResourceContainer: project.GetName(),
		ContainerUID:      project.UID,
		Type:              api.ActivityProjectRepositoryPush,
		Level:             api.ActivityInfo,
		Comment:           fmt.Sprintf("Created issue %q.", issue.Title),
		Payload:           string(activityPayload),
	}
	if _, err := s.activityManager.CreateActivity(ctx, activityCreate, &activity.Metadata{}); err != nil {
		return errors.Wrapf(err, "failed to activity after creating issue %d from push event", issueUID)
	}
	return nil
}
