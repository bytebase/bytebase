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
	"net/mail"
	"net/url"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/activity"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	advisorDB "github.com/bytebase/bytebase/backend/plugin/advisor/db"
	"github.com/bytebase/bytebase/backend/plugin/db"
	configparser "github.com/bytebase/bytebase/backend/plugin/parser/mybatis/configuration"
	mapperparser "github.com/bytebase/bytebase/backend/plugin/parser/mybatis/mapper"
	"github.com/bytebase/bytebase/backend/plugin/parser/mybatis/mapper/ast"
	"github.com/bytebase/bytebase/backend/plugin/vcs"

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
	issueNameTemplate    = "[%s] %s: %s"
	sdlIssueNameTemplate = "[%s] %s"
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

		nonBytebaseCommitList := filterGitLabBytebaseCommit(pushEvent.CommitList)
		if len(nonBytebaseCommitList) == 0 {
			var commitList []string
			for _, commit := range pushEvent.CommitList {
				commitList = append(commitList, commit.ID)
			}
			log.Debug("all commits are created by Bytebase",
				zap.String("repoURL", pushEvent.Project.WebURL),
				zap.String("repoName", pushEvent.Project.FullPath),
				zap.String("commits", strings.Join(commitList, ", ")),
			)
			return c.String(http.StatusOK, "OK")
		}
		pushEvent.CommitList = nonBytebaseCommitList

		filter := func(repo *store.RepositoryMessage) (bool, error) {
			if c.Request().Header.Get("X-Gitlab-Token") != repo.WebhookSecretToken {
				return false, nil
			}

			return s.isWebhookEventBranch(pushEvent.Ref, repo.BranchFilter)
		}
		repositoryList, err := s.filterRepository(ctx, c.Param("id"), repositoryID, filter)
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

		nonBytebaseCommitList := filterGitHubBytebaseCommit(pushEvent.Commits)
		if len(nonBytebaseCommitList) == 0 {
			var commitList []string
			for _, commit := range pushEvent.Commits {
				commitList = append(commitList, commit.ID)
			}
			log.Debug("all commits are created by Bytebase",
				zap.String("repoURL", pushEvent.Repository.HTMLURL),
				zap.String("repoName", pushEvent.Repository.FullName),
				zap.String("commits", strings.Join(commitList, ", ")),
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

			return s.isWebhookEventBranch(pushEvent.Ref, repo.BranchFilter)
		}
		repositoryList, err := s.filterRepository(ctx, c.Param("id"), repositoryID, filter)
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
				log.Debug("all commits are created by Bytebase",
					zap.String("repoURL", pushEvent.Repository.Links.HTML.Href),
					zap.String("repoName", pushEvent.Repository.FullName),
					zap.String("commits", strings.Join(commitList, ", ")),
				)
				continue
			}

			ref := "refs/heads/" + change.New.Name
			filter := func(repo *store.RepositoryMessage) (bool, error) {
				return s.isWebhookEventBranch(ref, repo.BranchFilter)
			}
			repositoryList, err := s.filterRepository(ctx, c.Param("id"), repositoryID, filter)
			if err != nil {
				return err
			}
			if len(repositoryList) == 0 {
				log.Debug("Empty handle repo list. Ignore this push event.")
				continue
			}
			repo := repositoryList[0]

			var commitList []vcs.Commit
			for _, commit := range nonBytebaseCommitList {
				before := strings.Repeat("0", 40)
				if len(commit.Parents) > 0 {
					before = commit.Parents[0].Hash
				}
				fileDiffList, err := vcs.Get(repo.vcs.Type, vcs.ProviderConfig{}).GetDiffFileList(
					ctx,
					common.OauthContext{
						ClientID:     repo.vcs.ApplicationID,
						ClientSecret: repo.vcs.Secret,
						AccessToken:  repo.repository.AccessToken,
						RefreshToken: repo.repository.RefreshToken,
						Refresher:    utils.RefreshToken(ctx, s.store, repo.repository.WebURL),
					},
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
			return s.isWebhookEventBranch(refUpdate.Name, repo.BranchFilter)
		}
		repositoryList, err := s.filterRepository(ctx, c.Param("id"), repositoryID, filter)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to filter repository").SetInternal(err)
		}
		if len(repositoryList) == 0 {
			log.Debug("Empty handle repo list. Ignore this push event.")
			return c.String(http.StatusOK, "No repository matched")
		}

		setting, err := s.store.GetWorkspaceGeneralSetting(ctx)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to find workspace setting").SetInternal(err)
		}

		oauthContext := common.OauthContext{
			ClientID:     repositoryList[0].vcs.ApplicationID,
			ClientSecret: repositoryList[0].vcs.Secret,
			AccessToken:  repositoryList[0].repository.AccessToken,
			RefreshToken: repositoryList[0].repository.RefreshToken,
			Refresher:    utils.RefreshToken(ctx, s.store, repositoryList[0].repository.WebURL),
			RedirectURL:  fmt.Sprintf("%s/oauth/callback", setting.ExternalUrl),
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
			log.Debug("all commits are created by Bytebase",
				zap.String("repoURL", pushEvent.Resource.Repository.URL),
				zap.String("repoID", repositoryID),
				zap.String("repoName", pushEvent.Resource.Repository.Name),
				zap.String("commits", strings.Join(commitIDs, ", ")),
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
			// Setup SQL review pipeline and policy.
			if err := azure.EnableSQLReviewCI(ctx, oauthContext, repositoryList[0].repository.ExternalID, repositoryList[0].repository.BranchFilter, repositoryList[0].repository.WebhookSecretToken); err != nil {
				log.Error("failed to setup pipeline", zap.Error(err), zap.String("repository", repositoryList[0].repository.ExternalID))
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

		createdMessages, err := s.processPushEvent(ctx, repositoryList, baseVCSPushEvent)
		if err != nil {
			return err
		}
		return c.String(http.StatusOK, strings.Join(createdMessages, "\n"))
	})

	// id is the webhookEndpointID in repository
	// This endpoint is generated and injected into GitHub action & GitLab CI during the VCS setup.
	g.POST("/sql-review/:id", func(c echo.Context) error {
		ctx := c.Request().Context()
		body, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Failed to read SQL review request").SetInternal(err)
		}
		log.Debug("SQL review request received for VCS project",
			zap.String("webhook_endpoint_id", c.Param("id")),
			zap.String("request", string(body)),
		)

		setting, err := s.store.GetWorkspaceGeneralSetting(ctx)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to find workspace setting").SetInternal(err)
		}

		var request api.VCSSQLReviewRequest
		if err := json.Unmarshal(body, &request); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed SQL review request").SetInternal(err)
		}

		workspaceID, err := s.store.GetWorkspaceID(ctx)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		token := c.Request().Header.Get("X-SQL-Review-Token")

		filter := func(repo *store.RepositoryMessage) (bool, error) {
			if !repo.EnableSQLReviewCI {
				log.Debug("Skip repository as the SQL review CI is not enabled.",
					zap.Int("repository_id", repo.UID),
					zap.String("repository_external_id", repo.ExternalID),
				)
				return false, nil
			}

			if !strings.HasPrefix(repo.WebURL, request.WebURL) {
				log.Debug("Skip repository as the web URL is not matched.",
					zap.String("request_web_url", request.WebURL),
					zap.String("repo_web_url", repo.WebURL),
				)
				return false, nil
			}

			// We will use workspace id as token in integration test for skipping the check.
			if token == workspaceID {
				return true, nil
			}

			return token == repo.WebhookSecretToken, nil
		}

		repositoryList, err := s.filterRepository(ctx, c.Param("id"), request.RepositoryID, filter)
		if err != nil {
			return err
		}
		if len(repositoryList) == 0 {
			log.Debug("Empty handle repo list. Ignore this request.")
			return c.JSON(http.StatusOK, &api.VCSSQLReviewResult{
				Status:  advisor.Success,
				Content: []string{},
			})
		}
		repo := repositoryList[0]

		prFiles, err := vcs.Get(repo.vcs.Type, vcs.ProviderConfig{}).ListPullRequestFile(
			ctx,
			common.OauthContext{
				ClientID:     repo.vcs.ApplicationID,
				ClientSecret: repo.vcs.Secret,
				AccessToken:  repo.repository.AccessToken,
				RefreshToken: repo.repository.RefreshToken,
				Refresher:    utils.RefreshToken(ctx, s.store, repo.repository.WebURL),
				RedirectURL:  fmt.Sprintf("%s/oauth/callback", setting.ExternalUrl),
			},
			repo.vcs.InstanceURL,
			repo.repository.ExternalID,
			request.PullRequestID,
		)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to list pull request file").SetInternal(err)
		}

		sqlFileName2Advice := s.sqlAdviceForSQLFiles(ctx, repositoryList, prFiles, setting.ExternalUrl)

		if s.licenseService.IsFeatureEnabled(api.FeatureMybatisSQLReview) == nil {
			// If the commit file list contains the file which extension is xml and the content
			// contains "https://mybatis.org/dtd/mybatis-3-mapper.dtd", we will try to apply
			// sql-review to it.
			// To apply sql-review to it, proceed as follows:
			// 1. Look in the sibling and parent directories for directories containing similar
			// <!DOCTYPE configuration
			//   PUBLIC "-//mybatis.org//DTD Config 3.0//EN"
			//   "https://mybatis.org/dtd/mybatis-3-config.dtd">
			// of the xml file
			// 2. If we can find it, then we will extract the sql from the mapper xml
			// 3. match the environments in the configuration xml, look for the sql-review policy in the environment and apply it.
			var isMybatisMapperXMLRegex = regexp.MustCompile(`(?i)http(s)?://mybatis\.org/dtd/mybatis-3-mapper\.dtd`)

			mybatisMapperXMLFiles := make(map[string]string)
			var commitID string
			for _, prFile := range prFiles {
				if !strings.HasSuffix(prFile.Path, ".xml") {
					continue
				}
				fileContent, err := vcs.Get(repo.vcs.Type, vcs.ProviderConfig{}).ReadFileContent(
					ctx,
					common.OauthContext{
						ClientID:     repo.vcs.ApplicationID,
						ClientSecret: repo.vcs.Secret,
						AccessToken:  repo.repository.AccessToken,
						RefreshToken: repo.repository.RefreshToken,
						Refresher:    utils.RefreshToken(ctx, s.store, repo.repository.WebURL),
					},
					repo.vcs.InstanceURL,
					repo.repository.ExternalID,
					prFile.Path,
					vcs.RefInfo{
						RefType: vcs.RefTypeCommit,
						RefName: prFile.LastCommitID,
					},
				)
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, "Failed to read file content").SetInternal(err)
				}
				if !isMybatisMapperXMLRegex.MatchString(fileContent) {
					continue
				}
				mybatisMapperXMLFiles[prFile.Path] = fileContent
				commitID = prFile.LastCommitID
			}
			if len(mybatisMapperXMLFiles) > 0 {
				mapperAdvices, err := s.sqlAdviceForMybatisMapperFiles(ctx, mybatisMapperXMLFiles, commitID, repo)
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get sql advice for mybatis mapper files").SetInternal(err)
				}
				for filename, mapperAdvice := range mapperAdvices {
					sqlFileName2Advice[filename] = mapperAdvice
				}
			}
		}

		response := &api.VCSSQLReviewResult{}
		switch repo.vcs.Type {
		case vcs.GitHub:
			response = convertSQLAdviceToGitHubActionResult(sqlFileName2Advice)
		case vcs.GitLab:
			response = convertSQLAdviceToGitLabCIResult(sqlFileName2Advice)
		case vcs.AzureDevOps:
			response = convertSQLAdviceToGitLabCIResult(sqlFileName2Advice)
		}

		log.Debug("SQL review finished",
			zap.String("pull_request", request.PullRequestID),
			zap.String("status", string(response.Status)),
			zap.String("content", strings.Join(response.Content, "\n")),
			zap.String("repository_id", request.RepositoryID),
			zap.String("vcs", string(repo.vcs.Type)),
		)

		return c.JSON(http.StatusOK, response)
	})
}

func (s *Server) sqlAdviceForMybatisMapperFiles(ctx context.Context, mybatisMapperContent map[string]string, commitID string, repoInfo *repoInfo) (map[string][]advisor.Advice, error) {
	if len(mybatisMapperContent) == 0 {
		return map[string][]advisor.Advice{}, nil
	}
	if commitID == "" {
		return nil, errors.Errorf("Unexpected empty commit id")
	}

	sqlCheckAdvices := make(map[string][]advisor.Advice)
	mybatisMapperXMLFileData, err := s.buildMybatisMapperXMLFileData(ctx, repoInfo, commitID, mybatisMapperContent)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to build mybatis mapper xml file data")
	}
	var wg sync.WaitGroup
	for _, mybatisMapperXMLFile := range mybatisMapperXMLFileData {
		log.Debug("Mybatis mapper xml file data",
			zap.String("mapper file", mybatisMapperXMLFile.mapperPath),
			zap.String("config file", mybatisMapperXMLFile.configPath),
		)
		if mybatisMapperXMLFile.configContent == "" {
			continue
		}
		wg.Add(1)
		go func(datum *mybatisMapperXMLFileDatum) {
			defer wg.Done()
			adviceList, err := s.sqlAdviceForMybatisMapperFile(ctx, datum)
			if err != nil {
				log.Error(
					"Failed to take SQL review for file",
					zap.String("file", datum.mapperContent),
					zap.String("repository", repoInfo.repository.WebURL),
					zap.Error(err),
				)
				sqlCheckAdvices[datum.configPath] = []advisor.Advice{
					{
						Status:  advisor.Warn,
						Code:    advisor.Internal,
						Title:   "Failed to take SQL review",
						Content: fmt.Sprintf("Failed to take SQL review for file %s with error %v", datum.mapperPath, err),
						Line:    1,
					},
				}
			} else if len(adviceList) > 0 {
				sqlCheckAdvices[datum.mapperPath] = adviceList
			}
		}(mybatisMapperXMLFile)
	}
	wg.Wait()

	return sqlCheckAdvices, nil
}

func (s *Server) sqlAdviceForMybatisMapperFile(ctx context.Context, datum *mybatisMapperXMLFileDatum) ([]advisor.Advice, error) {
	var result []advisor.Advice
	var environmentIDs []string
	// If the configuration file is found, we extract the environment from the configuration file.
	conf, err := configparser.ParseConfiguration(datum.configContent)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to extract environment ids").SetInternal(err)
	}

	allEnvironments, err := s.store.ListEnvironmentV2(ctx, &store.FindEnvironmentMessage{})
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to list environments").SetInternal(err)
	}

	for _, confEnv := range conf.Environments {
		environmentIDs = append(environmentIDs, confEnv.ID)
		for _, env := range allEnvironments {
			if strings.EqualFold(env.Title, confEnv.ID) {
				// If the environment is found, we extract the sql-review policy from the environment.
				policy, err := s.store.GetSQLReviewPolicy(ctx, env.UID)
				if err != nil {
					if e, ok := err.(*common.Error); ok && e.Code == common.NotFound {
						log.Debug("Cannot found SQL review policy in environment", zap.String("Environment", confEnv.ID), zap.Error(err))
						continue
					}
					return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to get SQL review policy").SetInternal(err)
				}
				if policy == nil {
					continue
				}
				engineType, err := extractDBTypeFromJDBCConnectionString(confEnv.JDBCConnString)
				if err != nil {
					return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to extract db type").SetInternal(err)
				}
				if engineType == db.UnknownType {
					continue
				}
				emptyCatalog, err := store.NewEmptyCatalog(engineType)
				if err != nil {
					return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to get empty catalog").SetInternal(err)
				}

				mybatisSQLs, lineMapping, err := extractMybatisMapperSQL(datum.mapperContent, engineType)
				if err != nil {
					return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to extract mybatis mapper sql").SetInternal(err)
				}

				dbType, err := advisorDB.ConvertToAdvisorDBType(string(engineType))
				if err != nil {
					return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to convert to advisor db type").SetInternal(err)
				}
				adviceList, err := advisor.SQLReviewCheck(mybatisSQLs, policy.RuleList, advisor.SQLReviewCheckContext{
					Catalog: emptyCatalog,
					DbType:  dbType,
				})
				if err != nil {
					return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to check sql review").SetInternal(err)
				}
				// Remap the line number to the original file.
				for _, advice := range adviceList {
					for _, line := range lineMapping {
						if advice.Line <= line.SQLLastLine {
							advice.Line = line.OriginalEleLine
							break
						}
					}
					result = append(result, advice)
				}
			}
		}
	}
	if len(result) == 0 {
		return []advisor.Advice{
			{
				Status: advisor.Warn,
				Code:   advisor.NotFound,
				Title:  fmt.Sprintf("SQL review policy not found for environment %s", strings.Join(environmentIDs, ",")),
				// TODO(zp): add link to doc.
				Content: "check doc for details.",
				Line:    1,
			},
		}, nil
	}
	return result, nil
}

func (s *Server) sqlAdviceForSQLFiles(
	ctx context.Context,
	repoInfoList []*repoInfo,
	prFiles []*vcs.PullRequestFile,
	externalURL string,
) map[string][]advisor.Advice {
	distinctFileList := []vcs.DistinctFileItem{}
	for _, prFile := range prFiles {
		if prFile.IsDeleted {
			continue
		}
		distinctFileList = append(distinctFileList, vcs.DistinctFileItem{
			FileName: prFile.Path,
			Commit: vcs.Commit{
				ID: prFile.LastCommitID,
			},
		})
	}

	sqlCheckAdvice := map[string][]advisor.Advice{}
	var wg sync.WaitGroup

	repoID2FileItemList := groupFileInfoByRepo(distinctFileList, repoInfoList)

	for _, fileInfoListInRepo := range repoID2FileItemList {
		for _, file := range fileInfoListInRepo {
			wg.Add(1)
			go func(file fileInfo) {
				defer wg.Done()
				adviceList, err := s.sqlAdviceForFile(ctx, file, externalURL)
				if err != nil {
					log.Error(
						"Failed to take SQL review for file",
						zap.String("file", file.item.FileName),
						zap.String("external_id", file.repoInfo.repository.ExternalID),
						zap.Error(err),
					)
					sqlCheckAdvice[file.item.FileName] = []advisor.Advice{
						{
							Status:  advisor.Warn,
							Code:    advisor.Internal,
							Title:   "Failed to take SQL review",
							Content: fmt.Sprintf("Failed to take SQL review for file %s with error %v", file.item.FileName, err),
							Line:    1,
						},
					}
				} else if adviceList != nil {
					sqlCheckAdvice[file.item.FileName] = adviceList
				}
			}(file)
		}
	}
	wg.Wait()
	return sqlCheckAdvice
}

func (s *Server) sqlAdviceForFile(
	ctx context.Context,
	fileInfo fileInfo,
	externalURL string,
) ([]advisor.Advice, error) {
	log.Debug("Processing file",
		zap.String("file", fileInfo.item.FileName),
		zap.String("vcs", string(fileInfo.repoInfo.vcs.Type)),
	)

	// TODO: support tenant mode project
	if fileInfo.repoInfo.project.TenantMode == api.TenantModeTenant {
		return []advisor.Advice{
			{
				Status:  advisor.Warn,
				Code:    advisor.Unsupported,
				Title:   "Tenant mode is not supported",
				Content: fmt.Sprintf("Project %s a tenant mode project.", fileInfo.repoInfo.project.Title),
				Line:    1,
			},
		}, nil
	}

	// TODO(ed): findProjectDatabases doesn't support the tenant mode.
	// We can use https://github.com/bytebase/bytebase/blob/main/server/issue.go#L691 to find databases in tenant mode project.
	databases, err := s.findProjectDatabases(ctx, fileInfo.repoInfo.project.UID, fileInfo.migrationInfo.Database, fileInfo.migrationInfo.Environment)
	if err != nil {
		log.Debug(
			"Failed to list database migration info",
			zap.String("project", fileInfo.repoInfo.repository.ProjectResourceID),
			zap.String("database", fileInfo.migrationInfo.Database),
			zap.String("environment", fileInfo.migrationInfo.Environment),
			zap.Error(err),
		)
		return nil, errors.Errorf("Failed to list databse with error: %v", err)
	}

	fileContent, err := vcs.Get(fileInfo.repoInfo.vcs.Type, vcs.ProviderConfig{}).ReadFileContent(
		ctx,
		common.OauthContext{
			ClientID:     fileInfo.repoInfo.vcs.ApplicationID,
			ClientSecret: fileInfo.repoInfo.vcs.Secret,
			AccessToken:  fileInfo.repoInfo.repository.AccessToken,
			RefreshToken: fileInfo.repoInfo.repository.RefreshToken,
			Refresher:    utils.RefreshToken(ctx, s.store, fileInfo.repoInfo.repository.WebURL),
			RedirectURL:  fmt.Sprintf("%s/oauth/callback", externalURL),
		},
		fileInfo.repoInfo.vcs.InstanceURL,
		fileInfo.repoInfo.repository.ExternalID,
		fileInfo.item.FileName,
		vcs.RefInfo{
			RefType: vcs.RefTypeCommit,
			RefName: fileInfo.item.Commit.ID,
		},
	)
	if err != nil {
		return nil, errors.Errorf("Failed to read file cotent for %s with error: %v", fileInfo.item.FileName, err)
	}

	// There may exist many databases that match the file name.
	// We just need to use the first one, which has the SQL review policy and can let us take the check.
	for _, database := range databases {
		instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &database.InstanceID})
		if err != nil {
			return nil, err
		}
		if instance == nil {
			return nil, errors.Errorf("cannot found instance %s", database.InstanceID)
		}
		if err := s.licenseService.IsFeatureEnabledForInstance(api.FeatureVCSSQLReviewWorkflow, instance); err != nil {
			log.Debug(err.Error(), zap.String("instance", instance.ResourceID))
			continue
		}

		environment, err := s.store.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{ResourceID: &database.EffectiveEnvironmentID})
		if err != nil {
			return nil, err
		}
		if environment == nil {
			return nil, errors.Errorf("cannot found environment %s", instance.EnvironmentID)
		}
		policy, err := s.store.GetSQLReviewPolicy(ctx, environment.UID)
		if err != nil {
			if e, ok := err.(*common.Error); ok && e.Code == common.NotFound {
				log.Debug("Cannot found SQL review policy in environment", zap.String("Environment", database.EffectiveEnvironmentID), zap.Error(err))
				continue
			}
			return nil, errors.Errorf("Failed to get SQL review policy in environment %v with error: %v", instance.EnvironmentID, err)
		}

		dbType, err := advisorDB.ConvertToAdvisorDBType(string(instance.Engine))
		if err != nil {
			return nil, errors.Errorf("Failed to convert database engine type %v to advisor db type with error: %v", instance.Engine, err)
		}

		advisorMode := advisor.SyntaxModeNormal
		if fileInfo.repoInfo.project.SchemaChangeType == api.ProjectSchemaChangeTypeSDL {
			advisorMode = advisor.SyntaxModeSDL
		}
		catalog, err := s.store.NewCatalog(ctx, database.UID, instance.Engine, advisorMode)
		if err != nil {
			return nil, errors.Errorf("Failed to get catalog for database %v with error: %v", database.UID, err)
		}

		driver, err := s.dbFactory.GetReadOnlyDatabaseDriver(ctx, instance, database)
		if err != nil {
			return nil, err
		}
		connection := driver.GetDB()
		dbSchema, err := s.store.GetDBSchema(ctx, database.UID)
		if err != nil {
			return nil, err
		}
		if dbSchema == nil {
			return nil, errors.Errorf("database schema %v not found", database.UID)
		}
		adviceList, err := advisor.SQLReviewCheck(fileContent, policy.RuleList, advisor.SQLReviewCheckContext{
			Charset:   dbSchema.Metadata.CharacterSet,
			Collation: dbSchema.Metadata.Collation,
			DbType:    dbType,
			Catalog:   catalog,
			Driver:    connection,
			Context:   ctx,
		})
		driver.Close(ctx)
		if err != nil {
			return nil, errors.Errorf("Failed to exec the SQL check for database %v with error: %v", database.UID, err)
		}

		return adviceList, nil
	}

	return []advisor.Advice{
		{
			Status:  advisor.Warn,
			Code:    advisor.Unsupported,
			Title:   "SQL review is disabled",
			Content: fmt.Sprintf("Cannot found SQL review policy or instance license. You can configure the SQL review policy on %s/setting/sql-review, and assign license to the instance", externalURL),
			Line:    1,
		},
	}, nil
}

type repositoryFilter func(*store.RepositoryMessage) (bool, error)

type repoInfo struct {
	repository *store.RepositoryMessage
	project    *store.ProjectMessage
	vcs        *store.ExternalVersionControlMessage
}

func (s *Server) filterRepository(ctx context.Context, webhookEndpointID string, pushEventRepositoryID string, filter repositoryFilter) ([]*repoInfo, error) {
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
			log.Error("failed to find the project",
				zap.String("project_resource_id", repo.ProjectResourceID),
				zap.String("repository_external_id", repo.ExternalID),
			)
			continue
		}
		if project == nil {
			log.Debug("skipping repo due to missing project",
				zap.String("project_resource_id", repo.ProjectResourceID),
				zap.String("repository_external_id", repo.ExternalID),
			)
			continue
		}
		externalVCS, err := s.store.GetExternalVersionControlV2(ctx, repo.VCSUID)
		if err != nil {
			log.Error("failed to find the vcs",
				zap.Int("vcs_uid", repo.VCSUID),
				zap.String("repository_external_id", repo.ExternalID),
			)
			continue
		}
		if externalVCS == nil {
			log.Debug("skipping repo due to missing VCS",
				zap.Int("vcs_uid", repo.VCSUID),
				zap.String("repository_external_id", repo.ExternalID),
			)
			continue
		}

		switch externalVCS.Type {
		case vcs.AzureDevOps:
			if !strings.HasSuffix(repo.ExternalID, pushEventRepositoryID) {
				log.Debug("Skipping repo due to external ID mismatch", zap.Int("repoID", repo.UID), zap.String("pushEventExternalID", pushEventRepositoryID), zap.String("repoExternalID", repo.ExternalID))
				continue
			}
		default:
			if pushEventRepositoryID != repo.ExternalID {
				log.Debug("Skipping repo due to external ID mismatch", zap.Int("repoID", repo.UID), zap.String("pushEventExternalID", pushEventRepositoryID), zap.String("repoExternalID", repo.ExternalID))
				continue
			}
		}

		ok, err := filter(repo)
		if err != nil {
			return nil, err
		}
		if !ok {
			log.Debug("Skipping repo due to mismatched payload signature", zap.Int("repoID", repo.UID))
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

func (*Server) isWebhookEventBranch(pushEventRef, branchFilter string) (bool, error) {
	branch, err := parseBranchNameFromRefs(pushEventRef)
	if err != nil {
		return false, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid ref: %s", pushEventRef)).SetInternal(err)
	}
	ok, err := filepath.Match(branchFilter, branch)
	if err != nil {
		return false, errors.Wrapf(err, "failed to match branch filter")
	}
	if !ok {
		log.Debug("Skipping repo due to branch filter mismatch", zap.String("branch", branch), zap.String("filter", branchFilter))
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
		log.Debug(
			"ref is not prefix with expected prefix",
			zap.String("ref", ref),
			zap.String("expected prefix", expectedPrefix),
		)
		return ref, errors.Errorf("unexpected ref name %q without prefix %q", ref, expectedPrefix)
	}
	return ref[len(expectedPrefix):], nil
}

func (s *Server) processPushEvent(ctx context.Context, repoInfoList []*repoInfo, baseVCSPushEvent vcs.PushEvent) ([]string, error) {
	if len(repoInfoList) == 0 {
		return nil, errors.Errorf("empty repository list")
	}

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

	repo := repoInfoList[0]
	filteredDistinctFileList := distinctFileList
	// The before commit ID is all zeros when the branch is just created and contains no commits yet, and we will encounter an error when we try to get the diff.
	if baseVCSPushEvent.Before != strings.Repeat("0", 40) {
		var err error
		filteredDistinctFileList, err = s.filterFilesByCommitsDiff(ctx, repo, distinctFileList, baseVCSPushEvent.Before, baseVCSPushEvent.After)
		if err != nil {
			return nil, err
		}
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
			pushEvent.BaseDirectory = repoInfo.repository.BaseDirectory
			createdMessage, created, activityCreateList, err := s.processFilesInProject(
				ctx,
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
					if _, err := s.ActivityManager.CreateActivity(ctx, activityCreate, &activity.Metadata{}); err != nil {
						log.Warn("Failed to create project activity for the ignored repository files", zap.Error(err))
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
		log.Warn("Ignored push event because no applicable file found in the commit list", zap.Strings("repos", repoURLs))
	}

	return createdMessageList, nil
}

// Users may merge commits from other branches,
// and some of the commits merged in may already be merged into the main branch.
// In that case, the commits in the push event contains files which are not added in this PR/MR.
// We use the compare API to get the file diffs and filter files by the diffs.
// TODO(dragonly): generate distinct file change list from the commits diff instead of filter.
func (s *Server) filterFilesByCommitsDiff(
	ctx context.Context,
	repoInfo *repoInfo,
	distinctFileList []vcs.DistinctFileItem,
	beforeCommit, afterCommit string,
) ([]vcs.DistinctFileItem, error) {
	fileDiffList, err := vcs.Get(repoInfo.vcs.Type, vcs.ProviderConfig{}).GetDiffFileList(
		ctx,
		common.OauthContext{
			ClientID:     repoInfo.vcs.ApplicationID,
			ClientSecret: repoInfo.vcs.Secret,
			AccessToken:  repoInfo.repository.AccessToken,
			RefreshToken: repoInfo.repository.RefreshToken,
			Refresher:    utils.RefreshToken(ctx, s.store, repoInfo.repository.WebURL),
		},
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
		log.Debug("Processing file", zap.String("file", item.FileName), zap.String("commit", item.Commit.ID))
		migrationInfo, fType, repoInfo, err := getFileInfo(item, repoInfoList)
		if err != nil {
			log.Warn("Failed to get file info for the ignored repository file",
				zap.String("file", item.FileName),
				zap.Error(err),
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
			log.Debug("Ignored file outside the base directory",
				zap.String("file", fileItem.FileName),
				zap.String("base_directory", repoInfo.repository.BaseDirectory),
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
			log.Error("Failed to parse migration file info",
				zap.String("project", repoInfo.repository.ProjectResourceID),
				zap.String("file", fileItem.FileName),
				zap.Error(err),
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
			log.Debug("Failed to parse schema file info",
				zap.String("file", fileItem.FileName),
				zap.Error(err),
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
func (s *Server) processFilesInProject(ctx context.Context, pushEvent vcs.PushEvent, repoInfo *repoInfo, fileInfoList []fileInfo) (string, bool, []*store.ActivityMessage, *echo.HTTPError) {
	if repoInfo.project.TenantMode == api.TenantModeTenant {
		if err := s.licenseService.IsFeatureEnabled(api.FeatureMultiTenancy); err != nil {
			return "", false, nil, echo.NewHTTPError(http.StatusForbidden, err.Error())
		}
	}

	var migrationDetailList []*api.MigrationDetail
	var activityCreateList []*store.ActivityMessage
	var createdIssueList []string
	var fileNameList []string

	creatorID := s.getIssueCreatorID(ctx, pushEvent.CommitList[0].AuthorEmail)
	for _, fileInfo := range fileInfoList {
		if fileInfo.fType == fileTypeSchema {
			if fileInfo.repoInfo.project.SchemaChangeType == api.ProjectSchemaChangeTypeSDL {
				// Create one issue per schema file for SDL project.
				migrationDetailListForFile, activityCreateListForFile := s.prepareIssueFromSDLFile(ctx, repoInfo, pushEvent, fileInfo.migrationInfo, fileInfo.item.FileName)
				activityCreateList = append(activityCreateList, activityCreateListForFile...)
				if len(migrationDetailListForFile) != 0 {
					databaseName := fileInfo.migrationInfo.Database
					issueName := fmt.Sprintf(sdlIssueNameTemplate, databaseName, "Alter schema")
					issueDescription := fmt.Sprintf("Apply schema diff by file %s", strings.TrimPrefix(fileInfo.item.FileName, repoInfo.repository.BaseDirectory+"/"))
					if err := s.createIssueFromMigrationDetailList(ctx, repoInfo.project, issueName, issueDescription, pushEvent, creatorID, migrationDetailListForFile); err != nil {
						return "", false, activityCreateList, echo.NewHTTPError(http.StatusInternalServerError, "Failed to create issue").SetInternal(err)
					}
					createdIssueList = append(createdIssueList, issueName)
				}
			} else {
				log.Debug("Ignored schema file for non-SDL project", zap.String("fileName", fileInfo.item.FileName), zap.String("type", string(fileInfo.item.ItemType)))
			}
		} else { // fileInfo.fType == fileTypeMigration
			// This is a migration-based DDL or DML file and we would allow it for both DDL and SDL schema change type project.
			// For DDL schema change type project, this is expected.
			// For SDL schema change type project, we allow it because:
			// 1) DML is always migration-based.
			// 2) We may have a limitation in SDL implementation.
			// 3) User just wants to break the glass.
			migrationDetailListForFile, activityCreateListForFile := s.prepareIssueFromFile(ctx, repoInfo, pushEvent, fileInfo)
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
		if d.MigrationType == db.Migrate {
			migrateType = "Alter schema"
			break
		}
	}
	// The files are grouped by database names before calling this function, so they have the same database name here.
	databaseName := fileInfoList[0].migrationInfo.Database
	description := strings.ReplaceAll(fileInfoList[0].migrationInfo.Description, "_", " ")
	issueName := fmt.Sprintf(issueNameTemplate, databaseName, migrateType, description)
	issueDescription := fmt.Sprintf("By VCS files:\n\n%s\n", strings.Join(fileNameList, "\n"))
	if err := s.createIssueFromMigrationDetailList(ctx, repoInfo.project, issueName, issueDescription, pushEvent, creatorID, migrationDetailList); err != nil {
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
		if mi.Database < mj.Database {
			return true
		}
		if mi.Database == mj.Database && mi.Version < mj.Version {
			return true
		}
		if mi.Database == mj.Database && mi.Version == mj.Version && mi.Type.GetVersionTypeSuffix() < mj.Type.GetVersionTypeSuffix() {
			return true
		}
		return false
	})
	return ret
}

func (s *Server) createIssueFromMigrationDetailsV2(ctx context.Context, project *store.ProjectMessage, issueName, issueDescription string, pushEvent vcs.PushEvent, creatorID int, migrationDetailList []*api.MigrationDetail) error {
	var steps []*v1pb.Plan_Step
	if len(migrationDetailList) == 1 && migrationDetailList[0].DatabaseID == 0 {
		migrationDetail := migrationDetailList[0]
		deploymentConfig, err := s.store.GetDeploymentConfigV2(ctx, project.UID)
		if err != nil {
			return err
		}
		apiDeploymentConfig, err := deploymentConfig.ToAPIDeploymentConfig()
		if err != nil {
			return err
		}
		deploySchedule, err := api.ValidateAndGetDeploymentSchedule(apiDeploymentConfig.Payload)
		if err != nil {
			return err
		}
		allDatabases, err := s.store.ListDatabases(ctx, &store.FindDatabaseMessage{ProjectID: &project.ResourceID})
		if err != nil {
			return err
		}
		matrix, err := utils.GetDatabaseMatrixFromDeploymentSchedule(deploySchedule, allDatabases)
		if err != nil {
			return err
		}
		changeType := getChangeType(migrationDetail.MigrationType)
		for _, stage := range matrix {
			step := &v1pb.Plan_Step{}
			for _, database := range stage {
				step.Specs = append(step.Specs, &v1pb.Plan_Spec{
					Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
						ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
							Type:          changeType,
							Target:        fmt.Sprintf("instances/%s/databases/%s", database.InstanceID, database.DatabaseName),
							Sheet:         fmt.Sprintf("projects/%s/sheets/%d", project.ResourceID, migrationDetail.SheetID),
							SchemaVersion: migrationDetail.SchemaVersion,
						},
					},
				})
			}
			steps = append(steps, step)
		}
	} else {
		var specs []*v1pb.Plan_Spec
		for _, migrationDetail := range migrationDetailList {
			changeType := getChangeType(migrationDetail.MigrationType)
			var target string
			if migrationDetail.DatabaseID != 0 {
				database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: &migrationDetail.DatabaseID})
				if err != nil {
					return err
				}
				if database == nil {
					return errors.Errorf("database %d not found", migrationDetail.DatabaseID)
				}
				target = fmt.Sprintf("instances/%s/databases/%s", database.InstanceID, database.DatabaseName)
			} else {
				// TODO(d): should never reach this.
				return errors.Errorf("tenant database is not supported yet")
			}
			specs = append(specs, &v1pb.Plan_Spec{
				Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
					ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
						Type:          changeType,
						Target:        target,
						Sheet:         fmt.Sprintf("projects/%s/sheets/%d", project.ResourceID, migrationDetail.SheetID),
						SchemaVersion: migrationDetail.SchemaVersion,
					},
				},
			})
		}
		steps = []*v1pb.Plan_Step{
			{
				Specs: specs,
			},
		}
	}
	childCtx := context.WithValue(ctx, common.PrincipalIDContextKey, creatorID)
	plan, err := s.rolloutService.CreatePlan(childCtx, &v1pb.CreatePlanRequest{
		Parent: fmt.Sprintf("projects/%s", project.ResourceID),
		Plan: &v1pb.Plan{
			Title: issueName,
			Steps: steps,
		},
	})
	if err != nil {
		return errors.Wrapf(err, "failed to create plan for sample project")
	}
	rollout, err := s.rolloutService.CreateRollout(childCtx, &v1pb.CreateRolloutRequest{
		Parent: fmt.Sprintf("projects/%s", project.ResourceID),
		Plan:   plan.Name,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to create rollout for sample project")
	}
	issue, err := s.issueService.CreateIssue(childCtx, &v1pb.CreateIssueRequest{
		Parent: fmt.Sprintf("projects/%s", project.ResourceID),
		Issue: &v1pb.Issue{
			Title:       issueName,
			Description: issueDescription,
			Type:        v1pb.Issue_DATABASE_CHANGE,
			Assignee:    fmt.Sprintf("users/%s", api.SystemBotEmail),
			Plan:        plan.Name,
			Rollout:     rollout.Name,
		},
	})
	if err != nil {
		return errors.Wrapf(err, "failed to create issue for sample project")
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
			IssueName:    issue.Name,
		},
	)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to construct activity payload").SetInternal(err)
	}

	activityCreate := &store.ActivityMessage{
		CreatorUID:   creatorID,
		ContainerUID: project.UID,
		Type:         api.ActivityProjectRepositoryPush,
		Level:        api.ActivityInfo,
		Comment:      fmt.Sprintf("Created issue %q.", issue.Name),
		Payload:      string(activityPayload),
	}
	if _, err := s.ActivityManager.CreateActivity(ctx, activityCreate, &activity.Metadata{}); err != nil {
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

func (s *Server) createIssueFromMigrationDetailList(ctx context.Context, project *store.ProjectMessage, issueName, issueDescription string, pushEvent vcs.PushEvent, creatorID int, migrationDetailList []*api.MigrationDetail) error {
	if s.profile.DevelopmentUseV2Scheduler {
		return s.createIssueFromMigrationDetailsV2(ctx, project, issueName, issueDescription, pushEvent, creatorID, migrationDetailList)
	}

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
		ProjectID:             project.UID,
		Name:                  issueName,
		Type:                  issueType,
		Description:           issueDescription,
		AssigneeID:            api.SystemBotID,
		AssigneeNeedAttention: true,
		CreateContext:         string(createContext),
	}
	issue, err := s.createIssue(ctx, issueCreate, creatorID)
	if err != nil {
		log.Error("Failed to create issue", zap.Any("issueCreate", issueCreate), zap.Error(err))
		errMsg := "Failed to create schema update issue"
		if issueType == api.IssueDatabaseDataUpdate {
			errMsg = "Failed to create data update issue"
		}
		return echo.NewHTTPError(http.StatusInternalServerError, errMsg).SetInternal(err)
	}

	// TODO(p0ny): sheet, for each sheet, update the payload to backtrace the issue.

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

	activityCreate := &store.ActivityMessage{
		CreatorUID:   creatorID,
		ContainerUID: project.UID,
		Type:         api.ActivityProjectRepositoryPush,
		Level:        api.ActivityInfo,
		Comment:      fmt.Sprintf("Created issue %q.", issue.Name),
		Payload:      string(activityPayload),
	}
	if _, err := s.ActivityManager.CreateActivity(ctx, activityCreate, &activity.Metadata{}); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create project activity after creating issue from repository push event: %d", issue.ID)).SetInternal(err)
	}

	return nil
}

func (s *Server) getIssueCreatorID(ctx context.Context, email string) int {
	creatorID := api.SystemBotID
	if email != "" {
		committerPrincipal, err := s.store.GetUser(ctx, &store.FindUserMessage{
			Email: &email,
		})
		if err != nil {
			log.Warn("Failed to find the principal with committer email, use system bot instead", zap.String("email", email), zap.Error(err))
		} else if committerPrincipal == nil {
			log.Info("Principal with committer email does not exist, use system bot instead", zap.String("email", email))
		} else {
			creatorID = committerPrincipal.ID
		}
	}
	return creatorID
}

// findProjectDatabases finds the list of databases with given name in the
// project. If the environmentResourceID is not empty, it will be used as a filter condition
// for the result list.
func (s *Server) findProjectDatabases(ctx context.Context, projectID int, dbName, environmentResourceID string) ([]*store.DatabaseMessage, error) {
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
func getIgnoredFileActivityCreate(projectID int, pushEvent vcs.PushEvent, file string, err error) *store.ActivityMessage {
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

	return &store.ActivityMessage{
		CreatorUID:   api.SystemBotID,
		ContainerUID: projectID,
		Type:         api.ActivityProjectRepositoryPush,
		Level:        api.ActivityWarn,
		Comment:      fmt.Sprintf("Ignored file %q, %v.", file, err),
		Payload:      string(payload),
	}
}

// readFileContent reads the content of the given file from the given repository.
func (s *Server) readFileContent(ctx context.Context, pushEvent vcs.PushEvent, repoInfo *repoInfo, file string) (string, error) {
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
	externalVCS, err := s.store.GetExternalVersionControlV2(ctx, repo.VCSUID)
	if err != nil {
		return "", err
	}
	if externalVCS == nil {
		return "", errors.Errorf("cannot found vcs with id %d", repo.VCSUID)
	}

	content, err := vcs.Get(externalVCS.Type, vcs.ProviderConfig{}).ReadFileContent(
		ctx,
		common.OauthContext{
			ClientID:     externalVCS.ApplicationID,
			ClientSecret: externalVCS.Secret,
			AccessToken:  repo.AccessToken,
			RefreshToken: repo.RefreshToken,
			Refresher:    utils.RefreshToken(ctx, s.store, repo.WebURL),
		},
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
func (s *Server) prepareIssueFromSDLFile(ctx context.Context, repoInfo *repoInfo, pushEvent vcs.PushEvent, schemaInfo *db.MigrationInfo, file string) ([]*api.MigrationDetail, []*store.ActivityMessage) {
	dbName := schemaInfo.Database
	if dbName == "" && repoInfo.project.TenantMode == api.TenantModeDisabled {
		log.Debug("Ignored schema file without a database name", zap.String("file", file))
		return nil, nil
	}

	sdl, err := s.readFileContent(ctx, pushEvent, repoInfo, file)
	if err != nil {
		activityCreate := getIgnoredFileActivityCreate(repoInfo.project.UID, pushEvent, file, errors.Wrap(err, "Failed to read file content"))
		return nil, []*store.ActivityMessage{activityCreate}
	}

	sheetPayload := &storepb.SheetPayload{
		VcsPayload: &storepb.SheetPayload_VCSPayload{
			PushEvent: utils.ConvertVcsPushEvent(&pushEvent),
		},
	}
	payload, err := protojson.Marshal(sheetPayload)
	if err != nil {
		return nil, nil
	}
	sheet, err := s.store.CreateSheet(ctx, &store.SheetMessage{
		CreatorID:  api.SystemBotID,
		ProjectUID: repoInfo.project.UID,
		Name:       file,
		Statement:  sdl,
		Visibility: store.ProjectSheet,
		Source:     store.SheetFromBytebaseArtifact,
		Type:       store.SheetForSQL,
		Payload:    string(payload),
	})
	if err != nil {
		activityCreate := getIgnoredFileActivityCreate(repoInfo.project.UID, pushEvent, file, errors.Wrap(err, "Failed to create a sheet"))
		return nil, []*store.ActivityMessage{activityCreate}
	}

	var migrationDetailList []*api.MigrationDetail
	if repoInfo.project.TenantMode == api.TenantModeTenant {
		return []*api.MigrationDetail{
			{
				MigrationType: db.MigrateSDL,
				SheetID:       sheet.UID,
			},
		}, nil
	}

	databases, err := s.findProjectDatabases(ctx, repoInfo.project.UID, dbName, schemaInfo.Environment)
	if err != nil {
		activityCreate := getIgnoredFileActivityCreate(repoInfo.project.UID, pushEvent, file, errors.Wrap(err, "Failed to find project databases"))
		return nil, []*store.ActivityMessage{activityCreate}
	}

	for _, database := range databases {
		migrationDetailList = append(migrationDetailList,
			&api.MigrationDetail{
				MigrationType: db.MigrateSDL,
				DatabaseID:    database.UID,
				SheetID:       sheet.UID,
			},
		)
	}

	return migrationDetailList, nil
}

// prepareIssueFromFile returns a list of update schema details derived
// from the given push event for DDL.
func (s *Server) prepareIssueFromFile(
	ctx context.Context,
	repoInfo *repoInfo,
	pushEvent vcs.PushEvent,
	fileInfo fileInfo,
) ([]*api.MigrationDetail, []*store.ActivityMessage) {
	content, err := s.readFileContent(ctx, pushEvent, repoInfo, fileInfo.item.FileName)
	if err != nil {
		return nil, []*store.ActivityMessage{
			getIgnoredFileActivityCreate(
				repoInfo.project.UID,
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
	payload, err := protojson.Marshal(sheetPayload)
	if err != nil {
		return nil, nil
	}
	if repoInfo.project.TenantMode == api.TenantModeTenant {
		// A non-YAML file means the whole file content is the SQL statement
		if !fileInfo.item.IsYAML {
			sheet, err := s.store.CreateSheet(ctx, &store.SheetMessage{
				CreatorID:  api.SystemBotID,
				ProjectUID: repoInfo.project.UID,
				Name:       fileInfo.item.FileName,
				Statement:  content,
				Visibility: store.ProjectSheet,
				Source:     store.SheetFromBytebaseArtifact,
				Type:       store.SheetForSQL,
				Payload:    string(payload),
			})
			if err != nil {
				activityCreate := getIgnoredFileActivityCreate(repoInfo.project.UID, pushEvent, fileInfo.item.FileName, errors.Wrap(err, "Failed to create a sheet"))
				return nil, []*store.ActivityMessage{activityCreate}
			}

			return []*api.MigrationDetail{
				{
					MigrationType: fileInfo.migrationInfo.Type,
					SheetID:       sheet.UID,
					SchemaVersion: fmt.Sprintf("%s-%s", fileInfo.migrationInfo.Version, fileInfo.migrationInfo.Type.GetVersionTypeSuffix()),
				},
			}, nil
		}

		var migrationFile api.MigrationFileYAML
		err = yaml.Unmarshal([]byte(content), &migrationFile)
		if err != nil {
			return nil, []*store.ActivityMessage{
				getIgnoredFileActivityCreate(
					repoInfo.project.UID,
					pushEvent,
					fileInfo.item.FileName,
					errors.Wrap(err, "Failed to parse file content as YAML"),
				),
			}
		}

		sheet, err := s.store.CreateSheet(ctx, &store.SheetMessage{
			CreatorID:  api.SystemBotID,
			ProjectUID: repoInfo.project.UID,
			Name:       fileInfo.item.FileName,
			Statement:  migrationFile.Statement,
			Visibility: store.ProjectSheet,
			Source:     store.SheetFromBytebaseArtifact,
			Type:       store.SheetForSQL,
			Payload:    string(payload),
		})
		if err != nil {
			activityCreate := getIgnoredFileActivityCreate(repoInfo.project.UID, pushEvent, fileInfo.item.FileName, errors.Wrap(err, "Failed to create a sheet"))
			return nil, []*store.ActivityMessage{activityCreate}
		}

		var migrationDetailList []*api.MigrationDetail
		for _, database := range migrationFile.Databases {
			dbList, err := s.findProjectDatabases(ctx, repoInfo.project.UID, database.Name, "")
			if err != nil {
				return nil, []*store.ActivityMessage{
					getIgnoredFileActivityCreate(
						repoInfo.project.UID,
						pushEvent,
						fileInfo.item.FileName,
						errors.Wrapf(err, "Failed to find project database %q", database.Name),
					),
				}
			}

			for _, db := range dbList {
				migrationDetailList = append(migrationDetailList,
					&api.MigrationDetail{
						MigrationType: fileInfo.migrationInfo.Type,
						DatabaseID:    db.UID,
						SheetID:       sheet.UID,
						SchemaVersion: fmt.Sprintf("%s-%s", fileInfo.migrationInfo.Version, fileInfo.migrationInfo.Type.GetVersionTypeSuffix()),
					},
				)
			}
		}
		return migrationDetailList, nil
	}

	// TODO(dragonly): handle modified file for tenant mode.
	databases, err := s.findProjectDatabases(ctx, repoInfo.project.UID, fileInfo.migrationInfo.Database, fileInfo.migrationInfo.Environment)
	if err != nil {
		activityCreate := getIgnoredFileActivityCreate(repoInfo.project.UID, pushEvent, fileInfo.item.FileName, errors.Wrap(err, "Failed to find project databases"))
		return nil, []*store.ActivityMessage{activityCreate}
	}

	if fileInfo.item.ItemType == vcs.FileItemTypeAdded {
		sheet, err := s.store.CreateSheet(ctx, &store.SheetMessage{
			CreatorID:  api.SystemBotID,
			ProjectUID: repoInfo.project.UID,
			Name:       fileInfo.item.FileName,
			Statement:  content,
			Visibility: store.ProjectSheet,
			Source:     store.SheetFromBytebaseArtifact,
			Type:       store.SheetForSQL,
			Payload:    string(payload),
		})
		if err != nil {
			activityCreate := getIgnoredFileActivityCreate(repoInfo.project.UID, pushEvent, fileInfo.item.FileName, errors.Wrap(err, "Failed to create a sheet"))
			return nil, []*store.ActivityMessage{activityCreate}
		}

		var migrationDetailList []*api.MigrationDetail
		for _, database := range databases {
			migrationDetailList = append(migrationDetailList,
				&api.MigrationDetail{
					MigrationType: fileInfo.migrationInfo.Type,
					DatabaseID:    database.UID,
					SheetID:       sheet.UID,
					SchemaVersion: fmt.Sprintf("%s-%s", fileInfo.migrationInfo.Version, fileInfo.migrationInfo.Type.GetVersionTypeSuffix()),
				},
			)
		}
		return migrationDetailList, nil
	}

	migrationVersion := fmt.Sprintf("%s-%s", fileInfo.migrationInfo.Version, fileInfo.migrationInfo.Type.GetVersionTypeSuffix())
	if err := s.tryUpdateTasksFromModifiedFile(ctx, databases, fileInfo.item.FileName, migrationVersion, content, pushEvent); err != nil {
		return nil, []*store.ActivityMessage{
			getIgnoredFileActivityCreate(
				repoInfo.project.UID,
				pushEvent,
				fileInfo.item.FileName,
				errors.Wrap(err, "Failed to find project task"),
			),
		}
	}
	return nil, nil
}

func (s *Server) tryUpdateTasksFromModifiedFile(ctx context.Context, databases []*store.DatabaseMessage, fileName, schemaVersion, statement string, pushEvent vcs.PushEvent) error {
	// TODO(p0ny): sheet, create new sheets and update task sheet id.
	// For modified files, we try to update the existing issue's statement.
	for _, database := range databases {
		find := &api.TaskFind{
			DatabaseID: &database.UID,
			StatusList: &[]api.TaskStatus{api.TaskPendingApproval, api.TaskFailed},
			TypeList:   &[]api.TaskType{api.TaskDatabaseSchemaUpdate, api.TaskDatabaseDataUpdate},
			Payload:    fmt.Sprintf("task.payload->>'schemaVersion' = '%s'", schemaVersion),
		}
		if s.profile.DevelopmentUseV2Scheduler {
			find = &api.TaskFind{
				DatabaseID:              &database.UID,
				LatestTaskRunStatusList: &[]api.TaskRunStatus{api.TaskRunNotStarted, api.TaskRunFailed},
				TypeList:                &[]api.TaskType{api.TaskDatabaseSchemaUpdate, api.TaskDatabaseDataUpdate},
				Payload:                 fmt.Sprintf("task.payload->>'schemaVersion' = '%s'", schemaVersion),
			}
		}

		taskList, err := s.store.ListTasks(ctx, find)
		if err != nil {
			return err
		}
		if len(taskList) == 0 {
			continue
		}
		if len(taskList) > 1 {
			log.Error("Found more than one pending approval or failed tasks for modified VCS file, should be only one task.", zap.Int("databaseID", database.UID), zap.String("schemaVersion", schemaVersion))
			return nil
		}
		task := taskList[0]
		issue, err := s.store.GetIssueV2(ctx, &store.FindIssueMessage{PipelineID: &task.PipelineID})
		if err != nil {
			log.Error("failed to get issue by pipeline ID", zap.Int("pipeline ID", task.PipelineID), zap.Error(err))
			return nil
		}
		if issue == nil {
			log.Error("issue not found by pipeline ID", zap.Int("pipeline ID", task.PipelineID), zap.Error(err))
			return nil
		}

		sheetPayload := &storepb.SheetPayload{
			VcsPayload: &storepb.SheetPayload_VCSPayload{
				PushEvent: utils.ConvertVcsPushEvent(&pushEvent),
			},
		}
		payload, err := protojson.Marshal(sheetPayload)
		if err != nil {
			return err
		}
		sheet, err := s.store.CreateSheet(ctx, &store.SheetMessage{
			CreatorID:  api.SystemBotID,
			ProjectUID: issue.Project.UID,
			Name:       fileName,
			Statement:  statement,
			Visibility: store.ProjectSheet,
			Source:     store.SheetFromBytebaseArtifact,
			Type:       store.SheetForSQL,
			Payload:    string(payload),
		})
		if err != nil {
			return err
		}

		// TODO(dragonly): Try to patch the failed migration history record to pending, and the statement to the current modified file content.
		log.Debug("Patching task for modified file VCS push event", zap.String("fileName", fileName), zap.Int("issueID", issue.UID), zap.Int("taskID", task.ID))
		taskPatch := api.TaskPatch{
			ID:        task.ID,
			SheetID:   &sheet.UID,
			UpdaterID: api.SystemBotID,
		}
		if err := s.TaskScheduler.PatchTask(ctx, task, &taskPatch, issue); err != nil {
			log.Error("Failed to patch task with the same migration version", zap.Int("issueID", issue.UID), zap.Int("taskID", task.ID), zap.Error(err))
			return nil
		}

		if issue.PlanUID != nil {
			plan, err := s.store.GetPlan(ctx, &store.FindPlanMessage{UID: issue.PlanUID})
			if err != nil {
				log.Error("failed to get plan", zap.Int64("plan ID", *issue.PlanUID), zap.Error(err))
			}
			for _, step := range plan.Config.Steps {
				for _, spec := range step.Specs {
					v, ok := spec.Config.(*storepb.PlanConfig_Spec_ChangeDatabaseConfig)
					if !ok {
						continue
					}
					if v.ChangeDatabaseConfig.SchemaVersion == schemaVersion && v.ChangeDatabaseConfig.Target == fmt.Sprintf("instances/%s/databases/%s", database.InstanceID, database.DatabaseName) {
						v.ChangeDatabaseConfig.Sheet = fmt.Sprintf("projects/%s/sheets/%d", issue.Project.ResourceID, sheet.UID)
					}
				}
			}
			if err := s.store.UpdatePlan(ctx, &store.UpdatePlanMessage{
				UID:       *issue.PlanUID,
				Config:    plan.Config,
				UpdaterID: api.SystemBotID,
			}); err != nil {
				log.Error("failed to update plan", zap.Int64("plan ID", *issue.PlanUID), zap.Error(err))
			}
		}

		// dismiss stale review, re-find the approval template
		// it's ok if we failed
		if err := func() error {
			if task.Status != api.TaskPendingApproval {
				return nil
			}
			payload := &storepb.IssuePayload{}
			if err := protojson.Unmarshal([]byte(issue.Payload), payload); err != nil {
				return errors.Wrapf(err, "failed to unmarshal original issue payload")
			}
			payload.Approval = &storepb.IssuePayloadApproval{
				ApprovalFindingDone: false,
			}
			payloadBytes, err := protojson.Marshal(payload)
			if err != nil {
				return errors.Wrapf(err, "failed to marshal issue payload")
			}
			payloadStr := string(payloadBytes)
			issue, err := s.store.UpdateIssueV2(ctx, issue.UID, &store.UpdateIssueMessage{
				Payload: &payloadStr,
			}, api.SystemBotID)
			if err != nil {
				return errors.Wrap(err, "failed to update issue payload")
			}
			s.stateCfg.ApprovalFinding.Store(issue.UID, issue)
			return nil
		}(); err != nil {
			log.Error("Failed to dismiss stale review", zap.Error(err))
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
			} else if advice.Status == advisor.Warn && status != advisor.Error {
				status = advice.Status
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
				fmt.Sprintf("<testsuite name=\"%s\">\n%s\n</testsuite>", filePath, strings.Join(testcaseList, "\n")),
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
func extractDBTypeFromJDBCConnectionString(jdbcURL string) (db.Type, error) {
	trimmed := strings.TrimPrefix(strings.TrimSpace(jdbcURL), "jdbc:")
	u, err := url.Parse(trimmed)
	if err != nil {
		return db.UnknownType, err
	}

	switch {
	case strings.HasPrefix(u.Scheme, "mysql"):
		return db.MySQL, nil
	case strings.HasPrefix(u.Scheme, "postgresql"):
		return db.Postgres, nil
	}
	return db.UnknownType, nil
}

// extractMybatisMapperSQL will extract the SQL from mybatis mapper XML.
func extractMybatisMapperSQL(mapperContent string, engineType db.Type) (string, []*ast.MybatisSQLLineMapping, error) {
	mybatisMapperParser := mapperparser.NewParser(mapperContent)
	mybatisMapperNode, err := mybatisMapperParser.Parse()
	if err != nil {
		return "", nil, errors.Wrapf(err, "failed to parse mybatis mapper xml")
	}

	var placeholder string
	switch engineType {
	case db.MySQL:
		placeholder = "?"
	case db.Postgres:
		placeholder = "$1"
	default:
		return "", nil, errors.Errorf("unsupported database type %q", engineType)
	}

	var sb strings.Builder
	lineMapping, err := mybatisMapperNode.RestoreSQLWithLineMapping(mybatisMapperParser.NewRestoreContext().WithRestoreDataNodePlaceholder(placeholder), &sb)
	if err != nil {
		return "", nil, errors.Wrapf(err, "failed to restore mybatis mapper xml")
	}
	return sb.String(), lineMapping, nil
}

// mybatisMapperXMLFileDatum is the metadata of mybatis mapper XML file.
// It maintains the mybatis mapper XML file path, mapper XML file content, and the corresponding mybatis configuration XML content.
type mybatisMapperXMLFileDatum struct {
	// mapperPath is the git ls-tree syntax filepath of the mybatis mapper XML file.
	mapperPath string
	// mapperContent is the content of the mybatis mapper XML file.
	mapperContent string
	// configPath is the git ls-tree syntax filepath of the mybatis configuration XML file,
	// it is empty if the mybatis configuration XML file is not found.
	configPath string
	// configContent is the content of the mybatis configuration XML file,
	// it is empty if the mybatis configuration XML file is not found.
	configContent string
}

// buildMybatisMapperXMLFileData will build the mybatis mapper XML file data.
//
//	ctx: the context.
//	repo: the repository will be list file tree and get file content from.
//	commitID: the commitID is the snapshot of the file tree and file content.
//	mapperFiles: the map of the mybatis mapper XML file path and content.
func (s *Server) buildMybatisMapperXMLFileData(ctx context.Context, repoInfo *repoInfo, commitID string, mapperFiles map[string]string) ([]*mybatisMapperXMLFileDatum, error) {
	if len(mapperFiles) == 0 {
		return []*mybatisMapperXMLFileDatum{}, nil
	}

	var mybatisMapperXMLFileData []*mybatisMapperXMLFileDatum
	// isMybatisConfigXMLRegex is the regex to match the mybatis configuration XML file, if it can match the file content,
	// we regard the file as the mybatis configuration XML file.
	var isMybatisConfigXMLRegex = regexp.MustCompile(`(?i)http(s)?://mybatis\.org/dtd/mybatis-3-config\.dtd`)
	// configPathCache is the cache of the mybatis configuration XML file directory,
	// the key is the mybatis mapper XML file ls-tree syntax directory, and value is the mybatis configuration XML file ls-tree syntax path.
	configPathCache := make(map[string]string)
	// configCache is the cache of the mybatis configuration XML file content,
	// the key is the mybatis configuration XML file ls-tree syntax path, and value is the mybatis configuration XML file content.
	// each value is configPathCache must be the key of configCache.
	configCache := make(map[string]string)

	for mapperFilePath, mapperFileContent := range mapperFiles {
		configPath := mapperFilePath
		datum := &mybatisMapperXMLFileDatum{
			mapperPath:    mapperFilePath,
			mapperContent: mapperFileContent,
		}
		for {
			currentDir := filepath.Dir(configPath)
			// git ls-tree syntax filepath didn't support '.', so we need to replace it with "" to represent the root directory.
			if currentDir == "." {
				currentDir = ""
			}
			if configPath, ok := configPathCache[currentDir]; ok {
				datum.configPath = configPath
				datum.configContent = configCache[configPath]
				break
			}

			filesInDir, err := vcs.Get(repoInfo.vcs.Type, vcs.ProviderConfig{}).FetchRepositoryFileList(
				ctx,
				common.OauthContext{
					ClientID:     repoInfo.vcs.ApplicationID,
					ClientSecret: repoInfo.vcs.Secret,
					AccessToken:  repoInfo.repository.AccessToken,
					RefreshToken: repoInfo.repository.RefreshToken,
					Refresher:    utils.RefreshToken(ctx, s.store, repoInfo.repository.WebURL),
				},
				repoInfo.vcs.InstanceURL,
				repoInfo.repository.ExternalID,
				commitID,
				currentDir,
			)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to fetch repository file list for repository %q commitID %q directory %q", repoInfo.repository.WebURL, commitID, currentDir)
			}

			for _, file := range filesInDir {
				if file.Type != "blob" || !strings.HasSuffix(file.Path, ".xml") {
					continue
				}
				fileContent, err := vcs.Get(repoInfo.vcs.Type, vcs.ProviderConfig{}).ReadFileContent(
					ctx,
					common.OauthContext{
						ClientID:     repoInfo.vcs.ApplicationID,
						ClientSecret: repoInfo.vcs.Secret,
						AccessToken:  repoInfo.repository.AccessToken,
						RefreshToken: repoInfo.repository.RefreshToken,
						Refresher:    utils.RefreshToken(ctx, s.store, repoInfo.repository.WebURL),
					},
					repoInfo.vcs.InstanceURL,
					repoInfo.repository.ExternalID,
					file.Path,
					vcs.RefInfo{
						RefType: vcs.RefTypeCommit,
						RefName: commitID,
					},
				)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to read file content for repository %q commitID %q file %q", repoInfo.repository.WebURL, commitID, file.Path)
				}
				if !isMybatisConfigXMLRegex.MatchString(fileContent) {
					continue
				}
				configPathCache[currentDir] = file.Path
				configCache[file.Path] = fileContent
				datum.configPath = file.Path
				datum.configContent = fileContent
			}
			if currentDir == "" {
				break
			}
			configPath = currentDir
		}
		mybatisMapperXMLFileData = append(mybatisMapperXMLFileData, datum)
	}
	return mybatisMapperXMLFileData, nil
}
