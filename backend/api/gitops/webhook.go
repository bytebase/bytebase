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
	"net/url"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/activity"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
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
	// sqlReviewDocs is the URL for SQL review doc.
	sqlReviewDocs = "https://www.bytebase.com/docs/reference/error-code/advisor"

	// issueNameTemplate should be consistent with UI issue names generated from the frontend except for the timestamp.
	// Because we cannot get the correct timezone of the client here.
	// Example: "[db-5] Alter schema: add an email column".
	issueNameTemplate      = "[%s] %s: %s"
	sdlIssueNameTemplate   = "[%s] %s"
	batchIssueNameTemplate = "%s: %s"
)

func (s *Service) RegisterWebhookRoutes(g *echo.Group) {
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

		nonBytebaseCommitList := filterGitLabBytebaseCommit(pushEvent.CommitList)
		if len(nonBytebaseCommitList) == 0 {
			var commitList []string
			for _, commit := range pushEvent.CommitList {
				commitList = append(commitList, commit.ID)
			}
			slog.Debug("all commits are created by Bytebase",
				slog.String("repoURL", pushEvent.Project.WebURL),
				slog.String("repoName", pushEvent.Project.FullPath),
				slog.String("commits", strings.Join(commitList, ", ")),
			)
			return c.String(http.StatusOK, "OK")
		}
		pushEvent.CommitList = nonBytebaseCommitList

		filter := func(repo *store.RepositoryMessage) (bool, error) {
			if c.Request().Header.Get("X-Gitlab-Token") != repo.WebhookSecretToken {
				return false, nil
			}
			return isWebhookEventBranch(pushEvent.Ref, repo.BranchFilter)
		}
		repositoryList, err := s.filterRepository(ctx, c.Param("id"), repositoryID, filter)
		if err != nil {
			return err
		}
		if len(repositoryList) == 0 {
			slog.Debug("Empty handle repo list. Ignore this push event.")
			return c.String(http.StatusOK, "OK")
		}

		repo := repositoryList[0]
		oauthContext := &common.OauthContext{
			AccessToken: repo.vcs.AccessToken,
		}

		baseVCSPushEvent, err := pushEvent.ToVCS()
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to convert GitLab commits").SetInternal(err)
		}

		createdMessages, err := s.processPushEvent(ctx, oauthContext, repositoryList, baseVCSPushEvent)
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

		nonBytebaseCommitList := filterGitHubBytebaseCommit(pushEvent.Commits)
		if len(nonBytebaseCommitList) == 0 {
			var commitList []string
			for _, commit := range pushEvent.Commits {
				commitList = append(commitList, commit.ID)
			}
			slog.Debug("all commits are created by Bytebase",
				slog.String("repoURL", pushEvent.Repository.HTMLURL),
				slog.String("repoName", pushEvent.Repository.FullName),
				slog.String("commits", strings.Join(commitList, ", ")),
			)
			return c.String(http.StatusOK, "OK")
		}
		pushEvent.Commits = nonBytebaseCommitList

		filter := func(repo *store.RepositoryMessage) (bool, error) {
			ok, err := validateGitHubWebhookSignature256(c.Request().Header.Get("X-Hub-Signature-256"), repo.WebhookSecretToken, body)
			if err != nil {
				return false, echo.NewHTTPError(http.StatusInternalServerError, "Failed to validate GitHub webhook signature").SetInternal(err)
			}
			if !ok {
				return false, nil
			}

			return isWebhookEventBranch(pushEvent.Ref, repo.BranchFilter)
		}
		repositoryList, err := s.filterRepository(ctx, c.Param("id"), repositoryID, filter)
		if err != nil {
			return err
		}
		if len(repositoryList) == 0 {
			slog.Debug("Empty handle repo list. Ignore this push event.")
			return c.String(http.StatusOK, "OK")
		}
		repo := repositoryList[0]
		oauthContext := &common.OauthContext{
			AccessToken: repo.vcs.AccessToken,
		}

		baseVCSPushEvent := pushEvent.ToVCS()

		createdMessages, err := s.processPushEvent(ctx, oauthContext, repositoryList, baseVCSPushEvent)
		if err != nil {
			return err
		}
		return c.String(http.StatusOK, strings.Join(createdMessages, "\n"))
	})

	g.POST("/bitbucket/:id", func(c echo.Context) error {
		ctx := c.Request().Context()

		// This shouldn't happen as we only set up webhook to receive push event, just in case.
		eventType := c.Request().Header.Get("X-Event-Key")
		if eventType != "repo:push" {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid webhook event type, got %q, want %q", eventType, "repo:push"))
		}

		body, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Failed to read webhook request").SetInternal(err)
		}
		var pushEvent bitbucket.WebhookPushEvent
		if err := json.Unmarshal(body, &pushEvent); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed push event").SetInternal(err)
		}
		repositoryID := pushEvent.Repository.FullName

		var allCreatedMessages []string
		for _, change := range pushEvent.Push.Changes {
			var nonBytebaseCommitList []bitbucket.WebhookCommit
			nonBytebaseCommitList = append(nonBytebaseCommitList, filterBitbucketBytebaseCommit(change.Commits)...)
			if len(nonBytebaseCommitList) == 0 {
				var commitList []string
				for _, change := range pushEvent.Push.Changes {
					for _, commit := range change.Commits {
						commitList = append(commitList, commit.Hash)
					}
				}
				slog.Debug("all commits are created by Bytebase",
					slog.String("repoURL", pushEvent.Repository.Links.HTML.Href),
					slog.String("repoName", pushEvent.Repository.FullName),
					slog.String("commits", strings.Join(commitList, ", ")),
				)
				continue
			}

			ref := "refs/heads/" + change.New.Name
			filter := func(repo *store.RepositoryMessage) (bool, error) {
				return isWebhookEventBranch(ref, repo.BranchFilter)
			}
			repositoryList, err := s.filterRepository(ctx, c.Param("id"), repositoryID, filter)
			if err != nil {
				return err
			}
			if len(repositoryList) == 0 {
				slog.Debug("Empty handle repo list. Ignore this push event.")
				continue
			}
			repo := repositoryList[0]

			oauthContext := &common.OauthContext{
				AccessToken: repo.vcs.AccessToken,
			}

			var commitList []vcs.Commit
			for _, commit := range nonBytebaseCommitList {
				before := strings.Repeat("0", 40)
				if len(commit.Parents) > 0 {
					before = commit.Parents[0].Hash
				}
				fileDiffList, err := vcs.Get(repo.vcs.Type, vcs.ProviderConfig{}).GetDiffFileList(
					ctx,
					oauthContext,
					repo.vcs.InstanceURL,
					repo.repository.ExternalID,
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

			createdMessages, err := s.processPushEvent(
				ctx,
				oauthContext,
				repositoryList,
				vcs.PushEvent{
					VCSType:            vcs.Bitbucket,
					Ref:                ref,
					Before:             change.Old.Target.Hash,
					After:              change.New.Target.Hash,
					RepositoryID:       pushEvent.Repository.FullName,
					RepositoryURL:      pushEvent.Repository.Links.HTML.Href,
					RepositoryFullPath: pushEvent.Repository.FullName,
					AuthorName:         pushEvent.Actor.Nickname,
					CommitList:         commitList,
				},
			)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to process push event for commit %q", change.New.Target.Hash)).SetInternal(err)
			}
			allCreatedMessages = append(allCreatedMessages, createdMessages...)
		}
		return c.String(http.StatusOK, strings.Join(allCreatedMessages, "\n"))
	})

	g.POST("/azure/:id", func(c echo.Context) error {
		ctx := c.Request().Context()

		body, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Failed to read webhook request").SetInternal(err)
		}
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

		webhookEndpointID := c.Param("id")
		// TODO(zp): find a better way to recognize the refine repository id, we use the format:
		// organization_name/project_name/repository_name
		repositories, err := s.store.ListRepositoryV2(ctx, &store.FindRepositoryMessage{
			WebhookEndpointID: &webhookEndpointID,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to list repositories").SetInternal(err)
		}
		if len(repositories) == 0 {
			return c.String(http.StatusOK, "No repository matched")
		}
		// Check the consistence of the repositories, it should not throw any error by right.
		for i := 1; i < len(repositories); i++ {
			if repositories[i].ExternalID != repositories[0].ExternalID {
				return echo.NewHTTPError(http.StatusPreconditionFailed, "The repositories external id are not consistent: %v", fmt.Sprintf("%v", repositories))
			}
		}
		repositoryID := repositories[0].ExternalID

		// Filter out the repository which does not match the branch filter.
		filter := func(repo *store.RepositoryMessage) (bool, error) {
			refUpdate := pushEvent.Resource.RefUpdates[0]
			return isWebhookEventBranch(refUpdate.Name, repo.BranchFilter)
		}
		repositoryList, err := s.filterRepository(ctx, c.Param("id"), repositoryID, filter)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to filter repository").SetInternal(err)
		}
		if len(repositoryList) == 0 {
			slog.Debug("Empty handle repo list. Ignore this push event.")
			return c.String(http.StatusOK, "No repository matched")
		}

		repo := repositoryList[0]
		oauthContext := &common.OauthContext{
			AccessToken: repo.vcs.AccessToken,
		}

		if len(pushEvent.Resource.Commits) == 0 {
			// If users merge one branch to our target branch, the commits list is empty in push event.
			// And we cannot get the commits in range [refUpdates.oldObjectId, refUpdates.newObjectId] from Azure DevOps API.
			// So, when the commit list is empty, we think it is generated by merge,
			// then we need to query the queryPullRequest API to find out if the updateRef.newObjectId
			// is the last commit of the Pull Request, and then we can use the PullRef.newObjectId API
			// to find out if it is the last commit of the Pull Request. Request ID to list the corresponding commits.
			probablePullRequests, err := azure.QueryPullRequest(ctx, oauthContext, repositoryList[0].repository.ExternalID, pushEvent.Resource.RefUpdates[0].NewObjectID)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to query pull request").SetInternal(err)
			}
			var filterOutPullRequestList []*azure.PullRequest
			for _, pullRequest := range probablePullRequests {
				if pullRequest.Status != "completed" {
					continue
				}
				// We only handle the pull request which is merged to the target branch.
				// NOTE: here we should compare with the refUpdates.Name instead of the repository.BranchFilter
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
			commitsInPullRequest, err := azure.GetPullRequestCommits(ctx, oauthContext, repositoryList[0].repository.ExternalID, pullRequest.ID)
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

		// Filter out all the commits created by Bytebase(e.g. write-back the latest schema) to avoid infinite loop.
		nonBytebaseCommitList := filterAzureBytebaseCommit(pushEvent.Resource.Commits)
		if len(nonBytebaseCommitList) == 0 {
			var commitIDs []string
			for _, commit := range pushEvent.Resource.Commits {
				commitIDs = append(commitIDs, commit.CommitID)
			}
			slog.Debug("all commits are created by Bytebase",
				slog.String("repoURL", pushEvent.Resource.Repository.URL),
				slog.String("repoID", repositoryID),
				slog.String("repoName", pushEvent.Resource.Repository.Name),
				slog.String("commits", strings.Join(commitIDs, ", ")),
			)
			return c.String(http.StatusOK, "OK")
		}
		pushEvent.Resource.Commits = nonBytebaseCommitList

		// Azure DevOps' service hook does not contain the file diff information for each commit, so we need to backfill
		// the file diff information by ourselves.
		// We will use the previous commit id as the base commit id and the current commit id as the target commit id to
		// get the file diff information commit by commit. We use the oldObjectId in resources.refUpdates as the base commit id
		// for the first commit.
		// NOTE: We presume that the sequence of the commits in the code push event is the reverse order of the commit sequence(aka. stack sequence, commit first, appear last) in the repository.
		backfillCommits := make([]vcs.Commit, 0, len(nonBytebaseCommitList))
		for _, commit := range nonBytebaseCommitList {
			changes, err := azure.GetChangesByCommit(ctx, oauthContext, repositoryList[0].repository.ExternalID, commit.CommitID)
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
			slog.Debug("start to setup pipeline", slog.String("repository", repositoryList[0].repository.ExternalID))
			// Use workspaceID instead of repo secret as SQL review pipeline token, so that we don't need to re-create the pipeline if users disable then enable the CI.
			workspaceID, err := s.store.GetWorkspaceID(ctx)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get workspace id with error").SetInternal(err)
			}
			// Setup SQL review pipeline and policy.
			if err := azure.EnableSQLReviewCI(ctx, oauthContext, repositoryList[0].repository.Title, repositoryList[0].repository.ExternalID, repositoryList[0].repository.BranchFilter, workspaceID); err != nil {
				slog.Error("failed to setup pipeline", log.BBError(err), slog.String("repository", repositoryList[0].repository.ExternalID))
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to setup SQL review pipeline").SetInternal(err)
			}
			return c.String(http.StatusOK, "OK")
		}

		// Backfill web url for commits.
		if pushEvent.Resource.PushID != 0 {
			commitsInPush, err := azure.GetPushCommitsByPushID(ctx, oauthContext, repositoryList[0].repository.ExternalID, pushEvent.Resource.PushID)
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

		baseVCSPushEvent := vcs.PushEvent{
			VCSType:            vcs.AzureDevOps,
			Ref:                pushEvent.Resource.RefUpdates[0].Name,
			Before:             pushEvent.Resource.RefUpdates[0].OldObjectID,
			After:              pushEvent.Resource.RefUpdates[0].NewObjectID,
			RepositoryID:       repositoryList[0].repository.ExternalID,
			RepositoryURL:      repositoryList[0].repository.WebURL,
			RepositoryFullPath: repositoryList[0].repository.ExternalID,
			AuthorName:         pushEvent.Resource.PushedBy.DisplayName,
			CommitList:         backfillCommits,
		}

		createdMessages, err := s.processPushEvent(ctx, oauthContext, repositoryList, baseVCSPushEvent)
		if err != nil {
			return err
		}
		return c.String(http.StatusOK, strings.Join(createdMessages, "\n"))
	})
}

type repositoryFilter func(*store.RepositoryMessage) (bool, error)

type repoInfo struct {
	repository *store.RepositoryMessage
	project    *store.ProjectMessage
	vcs        *store.VCSProviderMessage
}

func (s *Service) filterRepository(ctx context.Context, webhookEndpointID string, pushEventRepositoryID string, filter repositoryFilter) ([]*repoInfo, error) {
	repos, err := s.store.ListRepositoryV2(ctx, &store.FindRepositoryMessage{WebhookEndpointID: &webhookEndpointID})
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to respond webhook event for endpoint: %v", webhookEndpointID)).SetInternal(err)
	}
	if len(repos) == 0 {
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Repository for webhook endpoint %s not found", webhookEndpointID))
	}

	var filteredRepos []*repoInfo
	for _, repo := range repos {
		project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
			ResourceID:  &repo.ProjectResourceID,
			ShowDeleted: false,
		})
		if err != nil {
			slog.Error("failed to find the project",
				slog.String("project_resource_id", repo.ProjectResourceID),
				slog.String("repository_external_id", repo.ExternalID),
			)
			continue
		}
		if project == nil {
			slog.Debug("skipping repo due to missing project",
				slog.String("project_resource_id", repo.ProjectResourceID),
				slog.String("repository_external_id", repo.ExternalID),
			)
			continue
		}
		externalVCS, err := s.store.GetVCSProvider(ctx, &store.FindVCSProviderMessage{ID: &repo.VCSUID})
		if err != nil {
			slog.Error("failed to find the vcs",
				slog.Int("vcs_uid", repo.VCSUID),
				slog.String("repository_external_id", repo.ExternalID),
			)
			continue
		}
		if externalVCS == nil {
			slog.Debug("skipping repo due to missing VCS",
				slog.Int("vcs_uid", repo.VCSUID),
				slog.String("repository_external_id", repo.ExternalID),
			)
			continue
		}

		switch externalVCS.Type {
		case vcs.AzureDevOps:
			if !strings.HasSuffix(repo.ExternalID, pushEventRepositoryID) {
				slog.Debug("Skipping repo due to external ID mismatch", slog.Int("repoID", repo.UID), slog.String("pushEventExternalID", pushEventRepositoryID), slog.String("repoExternalID", repo.ExternalID))
				continue
			}
		default:
			if pushEventRepositoryID != repo.ExternalID {
				slog.Debug("Skipping repo due to external ID mismatch", slog.Int("repoID", repo.UID), slog.String("pushEventExternalID", pushEventRepositoryID), slog.String("repoExternalID", repo.ExternalID))
				continue
			}
		}

		ok, err := filter(repo)
		if err != nil {
			return nil, err
		}
		if !ok {
			slog.Debug("Skipping repo due to mismatched payload signature", slog.Int("repoID", repo.UID))
			continue
		}

		filteredRepos = append(filteredRepos, &repoInfo{
			repository: repo,
			project:    project,
			vcs:        externalVCS,
		})
	}
	return filteredRepos, nil
}

func isWebhookEventBranch(pushEventRef, branchFilter string) (bool, error) {
	branch, err := parseBranchNameFromRefs(pushEventRef)
	if err != nil {
		return false, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid ref: %s", pushEventRef)).SetInternal(err)
	}
	ok, err := filepath.Match(branchFilter, branch)
	if err != nil {
		return false, errors.Wrapf(err, "failed to match branch filter")
	}
	if !ok {
		slog.Debug("Skipping repo due to branch filter mismatch", slog.String("branch", branch), slog.String("filter", branchFilter))
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

func (s *Service) processPushEvent(ctx context.Context, oauthContext *common.OauthContext, repoInfoList []*repoInfo, baseVCSPushEvent vcs.PushEvent) ([]string, error) {
	if len(repoInfoList) == 0 {
		return nil, errors.Errorf("empty repository list")
	}

	distinctFileList := baseVCSPushEvent.GetDistinctFileList()
	if len(distinctFileList) == 0 {
		var commitIDs []string
		for _, c := range baseVCSPushEvent.CommitList {
			commitIDs = append(commitIDs, c.ID)
		}
		slog.Warn("No files found from the push event",
			slog.String("repoURL", baseVCSPushEvent.RepositoryURL),
			slog.String("repoName", baseVCSPushEvent.RepositoryFullPath),
			slog.String("commits", strings.Join(commitIDs, ",")))
		return nil, nil
	}

	repo := repoInfoList[0]

	filteredDistinctFileList, err := func() ([]vcs.DistinctFileItem, error) {
		// The before commit ID is all zeros when the branch is just created and contains no commits yet.
		if baseVCSPushEvent.Before == strings.Repeat("0", 40) {
			return distinctFileList, nil
		}
		return filterFilesByCommitsDiff(ctx, oauthContext, repo, distinctFileList, baseVCSPushEvent.Before, baseVCSPushEvent.After)
	}()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to filtered distinct files by commits diff")
	}

	repoID2FileItemList := groupFileInfoByRepo(filteredDistinctFileList, repoInfoList)

	var createdMessageList []string
	for _, fileInfoListInRepo := range repoID2FileItemList {
		// There are possibly multiple files in the push event.
		// Each file corresponds to a (database name, schema version) pair.
		// We want the migration statements are sorted by the file's schema version, and grouped by the database name.
		dbName2FileInfoList := groupFileInfoByDatabase(fileInfoListInRepo)
		for _, fileInfoListInDB := range dbName2FileInfoList {
			fileInfoListSorted := sortFilesBySchemaVersion(fileInfoListInDB)
			repoInfo := fileInfoListSorted[0].repoInfo
			pushEvent := baseVCSPushEvent
			pushEvent.VCSType = repoInfo.vcs.Type
			createdMessage, created, activityCreateList, err := s.processFilesInProject(
				ctx,
				oauthContext,
				pushEvent,
				repoInfo,
				fileInfoListSorted,
			)
			if err != nil {
				return nil, err
			}
			if created {
				createdMessageList = append(createdMessageList, createdMessage)
			} else {
				for _, activityCreate := range activityCreateList {
					if _, err := s.activityManager.CreateActivity(ctx, activityCreate, &activity.Metadata{}); err != nil {
						slog.Warn("Failed to create project activity for the ignored repository files", log.BBError(err))
					}
				}
			}
		}
	}

	if len(createdMessageList) == 0 {
		var repoURLs []string
		for _, repoInfo := range repoInfoList {
			repoURLs = append(repoURLs, repoInfo.repository.WebURL)
		}
		slog.Warn("Ignored push event because no applicable file found in the commit list", slog.Any("repos", repoURLs))
	}

	return createdMessageList, nil
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
		repoInfo.repository.ExternalID,
		beforeCommit,
		afterCommit,
	)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get file diff list for repository %s", repoInfo.repository.ExternalID)
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
	fType         fileType
	repoInfo      *repoInfo
}

func groupFileInfoByDatabase(fileInfoList []fileInfo) map[string][]fileInfo {
	dbID2FileInfoList := make(map[string][]fileInfo)
	for _, fileInfo := range fileInfoList {
		dbID2FileInfoList[fileInfo.migrationInfo.Database] = append(dbID2FileInfoList[fileInfo.migrationInfo.Database], fileInfo)
	}
	return dbID2FileInfoList
}

// groupFileInfoByRepo groups information for distinct files in the push event by their corresponding store.RepositoryMessage.
// In a GitLab/GitHub monorepo, a user could create multiple projects and configure different base directory in the repository.
// Bytebase will create a different store.RepositoryMessage for each project. If the user decides to do a migration in multiple directories at once,
// the push event will trigger changes in multiple projects. So we first group the files into store.RepositoryMessage, and create issue(s) in
// each project.
func groupFileInfoByRepo(distinctFileList []vcs.DistinctFileItem, repoInfoList []*repoInfo) map[int][]fileInfo {
	repoID2FileItemList := make(map[int][]fileInfo)
	for _, item := range distinctFileList {
		slog.Debug("Processing file", slog.String("file", item.FileName), slog.String("commit", item.Commit.ID))
		migrationInfo, fType, repoInfo, err := getFileInfo(item, repoInfoList)
		if err != nil {
			slog.Warn("Failed to get file info for the ignored repository file",
				slog.String("file", item.FileName),
				log.BBError(err),
			)
			continue
		}
		repoID2FileItemList[repoInfo.repository.UID] = append(repoID2FileItemList[repoInfo.repository.UID], fileInfo{
			item:          item,
			migrationInfo: migrationInfo,
			fType:         fType,
			repoInfo:      repoInfo,
		})
	}
	return repoID2FileItemList
}

type fileType int

const (
	fileTypeUnknown fileType = iota
	fileTypeMigration
	fileTypeSchema
)

// getFileInfo processes the file item against the candidate list of
// repositories and returns the parsed migration information, file change type
// and a single matched repository. It returns an error when none or multiple
// repositories are matched.
func getFileInfo(fileItem vcs.DistinctFileItem, repoInfoList []*repoInfo) (*db.MigrationInfo, fileType, *repoInfo, error) {
	var migrationInfo *db.MigrationInfo
	var fType fileType
	var fileRepositoryList []*repoInfo
	for _, repoInfo := range repoInfoList {
		if !strings.HasPrefix(fileItem.FileName, repoInfo.repository.BaseDirectory) {
			slog.Debug("Ignored file outside the base directory",
				slog.String("file", fileItem.FileName),
				slog.String("base_directory", repoInfo.repository.BaseDirectory),
			)
			continue
		}

		// NOTE: We do not want to use filepath.Join here because we always need "/" as the path separator.
		filePathTemplate := path.Join(repoInfo.repository.BaseDirectory, repoInfo.repository.FilePathTemplate)
		allowOmitDatabaseName := false
		if repoInfo.project.TenantMode == api.TenantModeTenant {
			allowOmitDatabaseName = true
			// If the committed file is a YAML file, then the user may have opted-in
			// advanced mode, we need to alter the FilePathTemplate to match ".yml" instead
			// of ".sql".
			if fileItem.IsYAML {
				filePathTemplate = strings.Replace(filePathTemplate, ".sql", ".yml", 1)
			}
		}

		mi, err := db.ParseMigrationInfo(fileItem.FileName, filePathTemplate, allowOmitDatabaseName)
		if err != nil {
			slog.Error("Failed to parse migration file info",
				slog.String("project", repoInfo.repository.ProjectResourceID),
				slog.String("file", fileItem.FileName),
				log.BBError(err),
			)
			continue
		}
		if mi != nil {
			if fileItem.IsYAML && mi.Type != db.Data {
				return nil, fileTypeUnknown, nil, errors.New("only DML is allowed for YAML files in a tenant project")
			}

			migrationInfo = mi
			fType = fileTypeMigration
			fileRepositoryList = append(fileRepositoryList, repoInfo)
			continue
		}

		si, err := db.ParseSchemaFileInfo(repoInfo.repository.BaseDirectory, repoInfo.repository.SchemaPathTemplate, fileItem.FileName)
		if err != nil {
			slog.Debug("Failed to parse schema file info",
				slog.String("file", fileItem.FileName),
				log.BBError(err),
			)
			continue
		}
		if si != nil {
			migrationInfo = si
			fType = fileTypeSchema
			fileRepositoryList = append(fileRepositoryList, repoInfo)
			continue
		}
	}

	switch len(fileRepositoryList) {
	case 0:
		return nil, fileTypeUnknown, nil, errors.Errorf("file change is not associated with any project")
	case 1:
		return migrationInfo, fType, fileRepositoryList[0], nil
	default:
		var projectList []string
		for _, repoInfo := range fileRepositoryList {
			projectList = append(projectList, repoInfo.project.Title)
		}
		return nil, fileTypeUnknown, nil, errors.Errorf("file change should be associated with exactly one project but found %s", strings.Join(projectList, ", "))
	}
}

// processFilesInProject attempts to create new issue(s) according to the repository type.
// 1. For a state based project, we create one issue per schema file, and one issue for all of the rest migration files (if any).
// 2. For a migration based project, we create one issue for all of the migration files. All schema files are ignored.
// It returns "created=true" when new issue(s) has been created,
// along with the creation message to be presented in the UI. An *echo.HTTPError
// is returned in case of the error during the process.
func (s *Service) processFilesInProject(ctx context.Context, oauthContext *common.OauthContext, pushEvent vcs.PushEvent, repoInfo *repoInfo, fileInfoList []fileInfo) (string, bool, []*store.ActivityMessage, *echo.HTTPError) {
	if repoInfo.project.TenantMode == api.TenantModeTenant {
		if err := s.licenseService.IsFeatureEnabled(api.FeatureMultiTenancy); err != nil {
			return "", false, nil, echo.NewHTTPError(http.StatusForbidden, err.Error())
		}
		return s.processFilesInBatchProject(ctx, oauthContext, pushEvent, repoInfo, fileInfoList)
	}

	var migrationDetailList []*migrationDetail
	var activityCreateList []*store.ActivityMessage
	var createdIssueList []string
	var fileNameList []string

	creatorID := s.getIssueCreatorID(ctx, pushEvent.CommitList[0].AuthorEmail)
	for _, fileInfo := range fileInfoList {
		if fileInfo.fType == fileTypeSchema {
			if fileInfo.repoInfo.project.SchemaChangeType == api.ProjectSchemaChangeTypeSDL {
				// Create one issue per schema file for SDL project.
				migrationDetailListForFile, activityCreateListForFile := s.prepareIssueFromSDLFile(ctx, oauthContext, repoInfo, pushEvent, fileInfo.migrationInfo, fileInfo.item.FileName)
				activityCreateList = append(activityCreateList, activityCreateListForFile...)
				if len(migrationDetailListForFile) != 0 {
					databaseName := fileInfo.migrationInfo.Database
					issueName := fmt.Sprintf(sdlIssueNameTemplate, databaseName, "Alter schema")
					issueDescription := fmt.Sprintf("Apply schema diff by file %s", strings.TrimPrefix(fileInfo.item.FileName, repoInfo.repository.BaseDirectory+"/"))
					if err := s.createIssueFromMigrationDetailsV2(ctx, repoInfo.project, issueName, issueDescription, pushEvent, creatorID, migrationDetailListForFile); err != nil {
						return "", false, activityCreateList, echo.NewHTTPError(http.StatusInternalServerError, "Failed to create issue %s, error %v", issueName, err).SetInternal(err)
					}
					createdIssueList = append(createdIssueList, issueName)
				}
			} else {
				slog.Debug("Ignored schema file for non-SDL project", slog.String("fileName", fileInfo.item.FileName), slog.String("type", string(fileInfo.item.ItemType)))
			}
		} else { // fileInfo.fType == fileTypeMigration
			// This is a migration-based DDL or DML file and we would allow it for both DDL and SDL schema change type project.
			// For DDL schema change type project, this is expected.
			// For SDL schema change type project, we allow it because:
			// 1) DML is always migration-based.
			// 2) We may have a limitation in SDL implementation.
			// 3) User just wants to break the glass.
			migrationDetailListForFile, activityCreateListForFile := s.prepareIssueFromFile(ctx, oauthContext, repoInfo, pushEvent, fileInfo)
			activityCreateList = append(activityCreateList, activityCreateListForFile...)
			migrationDetailList = append(migrationDetailList, migrationDetailListForFile...)
			if len(migrationDetailListForFile) != 0 {
				fileNameList = append(fileNameList, strings.TrimPrefix(fileInfo.item.FileName, repoInfo.repository.BaseDirectory+"/"))
			}
		}
	}

	if len(migrationDetailList) == 0 {
		return "", len(createdIssueList) != 0, activityCreateList, nil
	}

	// Create one issue per push event for DDL project, or non-schema files for SDL project.
	migrateType := "Change data"
	for _, d := range migrationDetailList {
		if d.migrationType == db.Migrate {
			migrateType = "Alter schema"
			break
		}
	}
	// The files are grouped by database names before calling this function, so they have the same database name here.
	databaseName := fileInfoList[0].migrationInfo.Database
	description := strings.ReplaceAll(fileInfoList[0].migrationInfo.Description, "_", " ")
	issueName := fmt.Sprintf(issueNameTemplate, databaseName, migrateType, description)
	issueDescription := fmt.Sprintf("By VCS files:\n\n%s\n", strings.Join(fileNameList, "\n"))
	if err := s.createIssueFromMigrationDetailsV2(ctx, repoInfo.project, issueName, issueDescription, pushEvent, creatorID, migrationDetailList); err != nil {
		return "", len(createdIssueList) != 0, activityCreateList, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create issue %s, error %v", issueName, err)).SetInternal(err)
	}
	createdIssueList = append(createdIssueList, issueName)

	return fmt.Sprintf("Created issue %q from push event", strings.Join(createdIssueList, ",")), true, activityCreateList, nil
}

// processFilesInBatchProject creates issues for a batch project.
func (s *Service) processFilesInBatchProject(ctx context.Context, oauthContext *common.OauthContext, pushEvent vcs.PushEvent, repoInfo *repoInfo, fileInfoList []fileInfo) (string, bool, []*store.ActivityMessage, *echo.HTTPError) {
	var activityCreateList []*store.ActivityMessage
	var createdIssueList []string

	creatorID := s.getIssueCreatorID(ctx, pushEvent.CommitList[0].AuthorEmail)
	for _, fileInfo := range fileInfoList {
		if fileInfo.fType == fileTypeSchema {
			if fileInfo.repoInfo.project.SchemaChangeType == api.ProjectSchemaChangeTypeSDL {
				// Create one issue per schema file for SDL project.
				migrationDetailListForFile, activityCreateListForFile := s.prepareIssueFromSDLFile(ctx, oauthContext, repoInfo, pushEvent, fileInfo.migrationInfo, fileInfo.item.FileName)
				activityCreateList = append(activityCreateList, activityCreateListForFile...)
				if len(migrationDetailListForFile) != 0 {
					databaseName := fileInfo.migrationInfo.Database
					issueName := fmt.Sprintf(sdlIssueNameTemplate, databaseName, "Alter schema")
					issueDescription := fmt.Sprintf("Apply schema diff by file %s", strings.TrimPrefix(fileInfo.item.FileName, repoInfo.repository.BaseDirectory+"/"))
					if err := s.createIssueFromMigrationDetailsV2(ctx, repoInfo.project, issueName, issueDescription, pushEvent, creatorID, migrationDetailListForFile); err != nil {
						return "", false, activityCreateList, echo.NewHTTPError(http.StatusInternalServerError, "Failed to create issue %s, error %v", issueName, err).SetInternal(err)
					}
					createdIssueList = append(createdIssueList, issueName)
				}
			} else {
				slog.Debug("Ignored schema file for non-SDL project", slog.String("fileName", fileInfo.item.FileName), slog.String("type", string(fileInfo.item.ItemType)))
			}
		} else { // fileInfo.fType == fileTypeMigration
			migrationDetailListForFile, activityCreateListForFile := s.prepareIssueFromFile(ctx, oauthContext, repoInfo, pushEvent, fileInfo)
			if len(migrationDetailListForFile) != 1 {
				slog.Error("Unexpected number of file number")
			}
			migrationDetail := migrationDetailListForFile[0]
			activityCreateList = append(activityCreateList, activityCreateListForFile...)
			migrateType := "Change data"
			if migrationDetail.migrationType == db.Migrate {
				migrateType = "Alter schema"
			}
			description := strings.ReplaceAll(fileInfoList[0].migrationInfo.Description, "_", " ")
			issueName := fmt.Sprintf(batchIssueNameTemplate, migrateType, description)
			issueDescription := fmt.Sprintf("By VCS file: %s\n", fileInfo.item.FileName)
			if err := s.createIssueFromMigrationDetailsV2(ctx, repoInfo.project, issueName, issueDescription, pushEvent, creatorID, migrationDetailListForFile); err != nil {
				return "", len(createdIssueList) != 0, activityCreateList, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create issue %s, error %v", issueName, err)).SetInternal(err)
			}
			createdIssueList = append(createdIssueList, issueName)
		}
	}

	return fmt.Sprintf("Created issue %q from push event", strings.Join(createdIssueList, ",")), true, activityCreateList, nil
}

func sortFilesBySchemaVersion(fileInfoList []fileInfo) []fileInfo {
	var ret []fileInfo
	ret = append(ret, fileInfoList...)
	sort.Slice(ret, func(i, j int) bool {
		mi := ret[i].migrationInfo
		mj := ret[j].migrationInfo
		if mi.Database < mj.Database {
			return true
		}
		if mi.Database == mj.Database && mi.Version.Version < mj.Version.Version {
			return true
		}
		if mi.Database == mj.Database && mi.Version == mj.Version && mi.Type.GetVersionTypeSuffix() < mj.Type.GetVersionTypeSuffix() {
			return true
		}
		return false
	})
	return ret
}

func (s *Service) createIssueFromMigrationDetailsV2(ctx context.Context, project *store.ProjectMessage, issueName, issueDescription string, pushEvent vcs.PushEvent, creatorID int, migrationDetailList []*migrationDetail) error {
	user, err := s.store.GetUserByID(ctx, creatorID)
	if err != nil {
		return errors.Wrapf(err, "failed to get user %v", creatorID)
	}

	var steps []*v1pb.Plan_Step
	if len(migrationDetailList) == 1 && migrationDetailList[0].databaseID == 0 {
		migrationDetail := migrationDetailList[0]
		changeType := getChangeType(migrationDetail.migrationType)
		steps = []*v1pb.Plan_Step{
			{
				Specs: []*v1pb.Plan_Spec{
					{
						Id: uuid.NewString(),
						Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
							ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
								Type:          changeType,
								Target:        fmt.Sprintf("projects/%s/deploymentConfigs/default", project.ResourceID),
								Sheet:         fmt.Sprintf("projects/%s/sheets/%d", project.ResourceID, migrationDetail.sheetID),
								SchemaVersion: migrationDetail.schemaVersion.Version,
							},
						},
					},
				},
			},
		}
	} else {
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
			if migrationDetail.databaseID == 0 {
				// TODO(d): should never reach this.
				return errors.Errorf("tenant database is not supported yet")
			}
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

// findProjectDatabases finds the list of databases with given name in the
// project. If the environmentResourceID is not empty, it will be used as a filter condition
// for the result list.
func (s *Service) findProjectDatabases(ctx context.Context, projectID int, dbName, environmentResourceID string) ([]*store.DatabaseMessage, error) {
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
	caseInsensitiveDatabases, err := s.store.ListDatabases(ctx,
		&store.FindDatabaseMessage{
			ProjectID:           &project.ResourceID,
			DatabaseName:        &dbName,
			IgnoreCaseSensitive: true,
		},
	)
	if err != nil {
		return nil, errors.Wrap(err, "find database")
	}
	var foundDatabases []*store.DatabaseMessage
	for _, database := range caseInsensitiveDatabases {
		database := database
		instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &database.InstanceID})
		if err != nil {
			return nil, errors.Wrap(err, "find instance")
		}
		if store.IgnoreDatabaseAndTableCaseSensitive(instance) || database.DatabaseName == dbName {
			foundDatabases = append(foundDatabases, database)
		}
	}
	if len(foundDatabases) == 0 {
		return nil, errors.Errorf("project %d does not have database %q", projectID, dbName)
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
	var filteredDatabases []*store.DatabaseMessage
	if environmentResourceID != "" {
		for _, database := range foundDatabases {
			// Environment resource ID comparison is case-sensitive.
			if database.EffectiveEnvironmentID == environmentResourceID {
				filteredDatabases = append(filteredDatabases, database)
			}
		}
		if len(filteredDatabases) == 0 {
			return nil, errors.Errorf("project %d does not have database %q with environment id %q", projectID, dbName, environmentResourceID)
		}
	} else {
		filteredDatabases = foundDatabases
	}

	// In case there are databases with identical name in a project for the same environment.
	marked := make(map[string]bool)
	for _, database := range filteredDatabases {
		if _, ok := marked[database.EffectiveEnvironmentID]; ok {
			return nil, errors.Errorf("project %d has multiple databases %q for environment %q", projectID, dbName, environmentResourceID)
		}
		marked[database.EffectiveEnvironmentID] = true
	}
	return filteredDatabases, nil
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
func (s *Service) readFileContent(ctx context.Context, oauthContext *common.OauthContext, pushEvent vcs.PushEvent, repoInfo *repoInfo, file string) (string, error) {
	// Retrieve the latest AccessToken and RefreshToken as the previous
	// ReadFileContent call may have updated the stored token pair. ReadFileContent
	// will fetch and store the new token pair if the existing token pair has
	// expired.
	repos, err := s.store.ListRepositoryV2(ctx, &store.FindRepositoryMessage{WebhookEndpointID: &repoInfo.repository.WebhookEndpointID})
	if err != nil {
		return "", errors.Wrapf(err, "get repository by webhook endpoint %q", repoInfo.repository.WebhookEndpointID)
	} else if len(repos) == 0 {
		return "", errors.Wrapf(err, "repository not found by webhook endpoint %q", repoInfo.repository.WebhookEndpointID)
	}

	repo := repos[0]
	externalVCS, err := s.store.GetVCSProvider(ctx, &store.FindVCSProviderMessage{ID: &repo.VCSUID})
	if err != nil {
		return "", err
	}
	if externalVCS == nil {
		return "", errors.Errorf("cannot found vcs with id %d", repo.VCSUID)
	}

	content, err := vcs.Get(externalVCS.Type, vcs.ProviderConfig{}).ReadFileContent(
		ctx,
		oauthContext,
		externalVCS.InstanceURL,
		repo.ExternalID,
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

// prepareIssueFromSDLFile returns the migration info and a list of update
// schema details derived from the given push event for SDL.
func (s *Service) prepareIssueFromSDLFile(ctx context.Context, oauthContext *common.OauthContext, repoInfo *repoInfo, pushEvent vcs.PushEvent, schemaInfo *db.MigrationInfo, file string) ([]*migrationDetail, []*store.ActivityMessage) {
	dbName := schemaInfo.Database
	if dbName == "" && repoInfo.project.TenantMode == api.TenantModeDisabled {
		slog.Debug("Ignored schema file without a database name", slog.String("file", file))
		return nil, nil
	}

	sdl, err := s.readFileContent(ctx, oauthContext, pushEvent, repoInfo, file)
	if err != nil {
		activityCreate := getIgnoredFileActivityCreate(repoInfo.project, pushEvent, file, errors.Wrap(err, "Failed to read file content"))
		return nil, []*store.ActivityMessage{activityCreate}
	}

	sheetPayload := &storepb.SheetPayload{
		VcsPayload: &storepb.SheetPayload_VCSPayload{
			PushEvent: utils.ConvertVcsPushEvent(&pushEvent),
		},
	}
	sheet, err := s.store.CreateSheet(ctx, &store.SheetMessage{
		CreatorID:  api.SystemBotID,
		ProjectUID: repoInfo.project.UID,
		Title:      file,
		Statement:  sdl,
		Payload:    sheetPayload,
	})
	if err != nil {
		activityCreate := getIgnoredFileActivityCreate(repoInfo.project, pushEvent, file, errors.Wrap(err, "Failed to create a sheet"))
		return nil, []*store.ActivityMessage{activityCreate}
	}

	var migrationDetailList []*migrationDetail
	if repoInfo.project.TenantMode == api.TenantModeTenant {
		return []*migrationDetail{
			{
				migrationType: db.MigrateSDL,
				sheetID:       sheet.UID,
			},
		}, nil
	}

	databases, err := s.findProjectDatabases(ctx, repoInfo.project.UID, dbName, schemaInfo.Environment)
	if err != nil {
		activityCreate := getIgnoredFileActivityCreate(repoInfo.project, pushEvent, file, errors.Wrap(err, "Failed to find project databases"))
		return nil, []*store.ActivityMessage{activityCreate}
	}

	for _, database := range databases {
		migrationDetailList = append(migrationDetailList,
			&migrationDetail{
				migrationType: db.MigrateSDL,
				databaseID:    database.UID,
				sheetID:       sheet.UID,
			},
		)
	}

	return migrationDetailList, nil
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
	content, err := s.readFileContent(ctx, oauthContext, pushEvent, repoInfo, fileInfo.item.FileName)
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
	if repoInfo.project.TenantMode == api.TenantModeTenant {
		// A non-YAML file means the whole file content is the SQL statement
		if !fileInfo.item.IsYAML {
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

			return []*migrationDetail{
				{
					migrationType: fileInfo.migrationInfo.Type,
					sheetID:       sheet.UID,
					schemaVersion: model.Version{Version: fmt.Sprintf("%s-%s", fileInfo.migrationInfo.Version.Version, fileInfo.migrationInfo.Type.GetVersionTypeSuffix())},
				},
			}, nil
		}

		var migrationFile MigrationFileYAML
		err = yaml.Unmarshal([]byte(content), &migrationFile)
		if err != nil {
			return nil, []*store.ActivityMessage{
				getIgnoredFileActivityCreate(
					repoInfo.project,
					pushEvent,
					fileInfo.item.FileName,
					errors.Wrap(err, "Failed to parse file content as YAML"),
				),
			}
		}

		sheet, err := s.store.CreateSheet(ctx, &store.SheetMessage{
			CreatorID:  api.SystemBotID,
			ProjectUID: repoInfo.project.UID,
			Title:      fileInfo.item.FileName,
			Statement:  migrationFile.Statement,
			Payload:    sheetPayload,
		})
		if err != nil {
			activityCreate := getIgnoredFileActivityCreate(repoInfo.project, pushEvent, fileInfo.item.FileName, errors.Wrap(err, "Failed to create a sheet"))
			return nil, []*store.ActivityMessage{activityCreate}
		}

		var migrationDetailList []*migrationDetail
		for _, database := range migrationFile.Databases {
			dbList, err := s.findProjectDatabases(ctx, repoInfo.project.UID, database.Name, "")
			if err != nil {
				return nil, []*store.ActivityMessage{
					getIgnoredFileActivityCreate(
						repoInfo.project,
						pushEvent,
						fileInfo.item.FileName,
						errors.Wrapf(err, "Failed to find project database %q", database.Name),
					),
				}
			}

			for _, db := range dbList {
				migrationDetailList = append(migrationDetailList,
					&migrationDetail{
						migrationType: fileInfo.migrationInfo.Type,
						databaseID:    db.UID,
						sheetID:       sheet.UID,
						schemaVersion: model.Version{Version: fmt.Sprintf("%s-%s", fileInfo.migrationInfo.Version.Version, fileInfo.migrationInfo.Type.GetVersionTypeSuffix())},
					},
				)
			}
		}
		return migrationDetailList, nil
	}

	// TODO(dragonly): handle modified file for tenant mode.
	databases, err := s.findProjectDatabases(ctx, repoInfo.project.UID, fileInfo.migrationInfo.Database, fileInfo.migrationInfo.Environment)
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

// convertSQLAdviceToGitLabCIResult will convert SQL advice map to GitLab test output format.
// GitLab test report: https://docs.gitlab.com/ee/ci/testing/unit_test_reports.html
// junit XML format: https://llg.cubic.org/docs/junit/
func convertSQLAdviceToGitLabCIResult(adviceMap map[string][]advisor.Advice) *api.VCSSQLReviewResult {
	testsuiteList := []string{}
	status := advisor.Success

	fileList := getSQLAdviceFileList(adviceMap)
	for _, filePath := range fileList {
		adviceList := adviceMap[filePath]
		testcaseList := []string{}
		pathes := strings.Split(filePath, "/")
		filename := pathes[len(pathes)-1]

		errorCount := 0
		failureCount := 0
		for _, advice := range adviceList {
			if advice.Code == 0 {
				continue
			}

			line := advice.Line
			if line <= 0 {
				line = 1
			}

			if advice.Status == advisor.Error {
				status = advice.Status
				errorCount++
			} else if advice.Status == advisor.Warn {
				failureCount++
				if status != advisor.Error {
					status = advice.Status
				}
			}

			content := fmt.Sprintf("Error: %s.\nPlease check the docs at %s#%d",
				advice.Content,
				sqlReviewDocs,
				advice.Code,
			)

			testcase := fmt.Sprintf(
				"<testcase name=\"%s\" classname=\"%s\" file=\"%s#L%d\">\n<failure>\n%s\n</failure>\n</testcase>",
				fmt.Sprintf("[%s] %s#L%d: %s", advice.Status, filename, line, advice.Title),
				filePath,
				filePath,
				line,
				content,
			)

			testcaseList = append(testcaseList, testcase)
		}

		if len(testcaseList) > 0 {
			testsuiteList = append(
				testsuiteList,
				fmt.Sprintf("<testsuite errors=\"%d\" failures=\"%d\" tests=\"%d\" name=\"%s\">\n%s\n</testsuite>", errorCount, failureCount, len(adviceList), filePath, strings.Join(testcaseList, "\n")),
			)
		}
	}

	return &api.VCSSQLReviewResult{
		Status: status,
		Content: []string{
			fmt.Sprintf(
				"<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<testsuites name=\"SQL Review\">\n%s\n</testsuites>",
				strings.Join(testsuiteList, "\n"),
			),
		},
	}
}

// convertSQLAdviceToGitHubActionResult will convert SQL advice map to GitHub action output format.
// GitHub action output message: https://docs.github.com/en/actions/using-workflows/workflow-commands-for-github-actions
func convertSQLAdviceToGitHubActionResult(adviceMap map[string][]advisor.Advice) *api.VCSSQLReviewResult {
	messageList := []string{}
	status := advisor.Success

	fileList := getSQLAdviceFileList(adviceMap)
	for _, filePath := range fileList {
		adviceList := adviceMap[filePath]
		for _, advice := range adviceList {
			if advice.Code == 0 || advice.Status == advisor.Success {
				continue
			}

			line := advice.Line
			if line <= 0 {
				line = 1
			}

			prefix := ""
			if advice.Status == advisor.Error {
				prefix = "error"
				status = advice.Status
			} else {
				prefix = "warning"
				if status != advisor.Error {
					status = advice.Status
				}
			}

			msg := fmt.Sprintf(
				"::%s file=%s,line=%d,col=1,endColumn=2,title=%s (%d)::%s\nDoc: %s#%d",
				prefix,
				filePath,
				line,
				advice.Title,
				advice.Code,
				advice.Content,
				sqlReviewDocs,
				advice.Code,
			)
			// To indent the output message in action
			messageList = append(messageList, strings.ReplaceAll(msg, "\n", "%0A"))
		}
	}
	return &api.VCSSQLReviewResult{
		Status:  status,
		Content: messageList,
	}
}

func getSQLAdviceFileList(adviceMap map[string][]advisor.Advice) []string {
	fileList := []string{}
	fileToErrorCount := map[string]int{}
	for filePath, adviceList := range adviceMap {
		fileList = append(fileList, filePath)

		errorCount := 0
		for _, advice := range adviceList {
			if advice.Status == advisor.Error {
				errorCount++
			}
		}
		fileToErrorCount[filePath] = errorCount
	}
	sort.Strings(fileList)
	sort.Slice(fileList, func(i int, j int) bool {
		if fileToErrorCount[fileList[i]] == fileToErrorCount[fileList[j]] {
			return i < j
		}
		return fileToErrorCount[fileList[i]] > fileToErrorCount[fileList[j]]
	})

	return fileList
}

func filterAzureBytebaseCommit(list []azure.ServiceHookCodePushEventResourceCommit) []azure.ServiceHookCodePushEventResourceCommit {
	var result []azure.ServiceHookCodePushEventResourceCommit
	for _, commit := range list {
		if commit.Author.Name == vcs.BytebaseAuthorName && commit.Author.Email == vcs.BytebaseAuthorEmail {
			continue
		}
		result = append(result, commit)
	}
	return result
}

func filterGitHubBytebaseCommit(list []github.WebhookCommit) []github.WebhookCommit {
	var result []github.WebhookCommit
	for _, commit := range list {
		if commit.Author.Name == vcs.BytebaseAuthorName && commit.Author.Email == vcs.BytebaseAuthorEmail {
			continue
		}
		result = append(result, commit)
	}
	return result
}

func filterGitLabBytebaseCommit(list []gitlab.WebhookCommit) []gitlab.WebhookCommit {
	var result []gitlab.WebhookCommit
	for _, commit := range list {
		if commit.Author.Name == vcs.BytebaseAuthorName && commit.Author.Email == vcs.BytebaseAuthorEmail {
			continue
		}
		result = append(result, commit)
	}
	return result
}

func filterBitbucketBytebaseCommit(list []bitbucket.WebhookCommit) []bitbucket.WebhookCommit {
	bytebaseRaw := fmt.Sprintf("%s <%s>", vcs.BytebaseAuthorName, vcs.BytebaseAuthorEmail)
	var result []bitbucket.WebhookCommit
	for _, commit := range list {
		if commit.Author.Raw == bytebaseRaw {
			continue
		}
		result = append(result, commit)
	}
	return result
}

// extractDBTypeFromJDBCConnectionString will extract the DB type from JDBC connection string. Only support MySQL and Postgres for now.
// It will return UnknownType if the DB type is not supported, and returns error if cannot parse the JDBC connection string.
func extractDBTypeFromJDBCConnectionString(jdbcURL string) (storepb.Engine, error) {
	trimmed := strings.TrimPrefix(strings.TrimSpace(jdbcURL), "jdbc:")
	u, err := url.Parse(trimmed)
	if err != nil {
		return storepb.Engine_ENGINE_UNSPECIFIED, err
	}

	switch {
	case strings.HasPrefix(u.Scheme, "mysql"):
		return storepb.Engine_MYSQL, nil
	case strings.HasPrefix(u.Scheme, "postgresql"):
		return storepb.Engine_POSTGRES, nil
	}
	return storepb.Engine_ENGINE_UNSPECIFIED, nil
}
