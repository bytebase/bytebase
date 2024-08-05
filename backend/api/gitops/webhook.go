package gitops

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/utils"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/vcs"
	"github.com/bytebase/bytebase/backend/plugin/vcs/bitbucket"

	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func (s *Service) RegisterWebhookRoutes(g *echo.Group) {
	g.POST(":id", func(c echo.Context) error {
		ctx := c.Request().Context()
		// The id start with "/".
		setting, err := s.store.GetWorkspaceGeneralSetting(ctx)
		if err != nil {
			return c.String(http.StatusOK, fmt.Sprintf("failed to get setting, error %v", err))
		}
		if setting.ExternalUrl == "" {
			return c.String(http.StatusOK, "external URL is empty")
		}

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

		var prInfo *pullRequestInfo
		switch vcsProvider.Type {
		case storepb.VCSType_GITHUB:
			secretToken := c.Request().Header.Get("X-Hub-Signature-256")
			ok, err := validateGitHubWebhookSignature256(secretToken, vcsConnector.Payload.WebhookSecretToken, body)
			if err != nil {
				return c.String(http.StatusOK, fmt.Sprintf("failed to validate webhook signature %q, error %v", secretToken, err))
			}
			if !ok {
				return c.String(http.StatusOK, fmt.Sprintf("invalid webhook secret token %q", secretToken))
			}
			eventType := c.Request().Header.Get("X-GitHub-Event")
			// https://docs.github.com/en/developers/webhooks-and-events/webhooks/about-webhooks#ping-event
			// When we create a new webhook, GitHub will send us a simple ping event to let us know we've set up the webhook correctly.
			// We respond to this event so as not to mislead users.
			switch eventType {
			case "ping":
				return c.String(http.StatusOK, "OK")
			case "pull_request":
			default:
				return c.String(http.StatusOK, fmt.Sprintf(`skip webhook event "%v"`, eventType))
			}

			prInfo, err = getGitHubPullRequestInfo(ctx, vcsProvider, vcsConnector, body)
			if err != nil {
				return c.String(http.StatusOK, fmt.Sprintf("failed to get pr info from pull request, error %v", err))
			}
		case storepb.VCSType_GITLAB:
			secretToken := c.Request().Header.Get("X-Gitlab-Token")
			if secretToken != vcsConnector.Payload.WebhookSecretToken {
				return c.String(http.StatusOK, fmt.Sprintf("invalid webhook secret token %q", secretToken))
			}

			prInfo, err = getGitLabPullRequestInfo(ctx, vcsProvider, vcsConnector, body)
			if err != nil {
				return c.String(http.StatusOK, fmt.Sprintf("failed to get pr info from pull request, error %v", err))
			}
		case storepb.VCSType_BITBUCKET:
			eventType := c.Request().Header.Get("X-Event-Key")
			var action webhookAction
			switch bitbucket.PullRequestEventType(eventType) {
			case bitbucket.PullRequestEventCreated, bitbucket.PullRequestEventUpdated:
				action = webhookActionSQLReview
			case bitbucket.PullRequestEventFulfilled:
				action = webhookActionCreateIssue
			default:
				return c.String(http.StatusOK, fmt.Sprintf(`skip webhook event "%v"`, eventType))
			}

			prInfo, err = getBitBucketPullRequestInfo(ctx, vcsProvider, vcsConnector, body)
			if err != nil {
				return c.String(http.StatusOK, fmt.Sprintf("failed to get pr info from pull request, error %v", err))
			}
			prInfo.action = action
		case storepb.VCSType_AZURE_DEVOPS:
			secretToken := c.Request().Header.Get("X-Azure-Token")
			if secretToken != vcsConnector.Payload.WebhookSecretToken {
				return c.String(http.StatusOK, fmt.Sprintf("invalid webhook secret token %q", secretToken))
			}

			prInfo, err = getAzurePullRequestInfo(ctx, vcsProvider, vcsConnector, body)
			if err != nil {
				return c.String(http.StatusOK, fmt.Sprintf("failed to get pr info from pull request, error %v", err))
			}
		default:
			return nil
		}
		if len(prInfo.changes) == 0 {
			return c.String(http.StatusOK, fmt.Sprintf("no relevant file change directly under the base directory %q for pull request %q", vcsConnector.Payload.BaseDirectory, prInfo.url))
		}

		user := func() *store.UserMessage {
			user, err := s.store.GetUserByEmail(ctx, prInfo.email)
			if err != nil {
				slog.Error("failed to find user by email", slog.String("email", prInfo.email), log.BBError(err))
				return s.store.GetSystemBotUser(ctx)
			}
			if user == nil {
				return s.store.GetSystemBotUser(ctx)
			}
			return user
		}()
		childCtx := context.WithValue(ctx, common.PrincipalIDContextKey, user.ID)
		childCtx = context.WithValue(childCtx, common.UserContextKey, user)

		var comment string
		var commentPrefix string
		switch prInfo.action {
		case webhookActionCreateIssue:
			issue, err := s.createIssueFromPRInfo(childCtx, project, vcsProvider, vcsConnector, prInfo)
			if err != nil {
				return c.String(http.StatusOK, fmt.Sprintf("failed to create issue from pull request %s, error %v", prInfo.url, err))
			}
			comment = getPullRequestComment(setting.ExternalUrl, issue.Name)
			commentPrefix = commentPrefixBytebaseBot
		case webhookActionSQLReview:
			comment, err = s.sqlReviewWithPRInfo(childCtx, project, vcsConnector, vcsProvider.Type, prInfo)
			if err != nil {
				return c.String(http.StatusOK, fmt.Sprintf("failed to exec sql review for pull request %s, error %v", prInfo.url, err))
			}
			comment = fmt.Sprintf("%s\n\n---\n\nClick [here](%s) to check the SQL review config", comment, fmt.Sprintf("%s/sql-review", setting.ExternalUrl))
			commentPrefix = commentPrefixSQLReview
		default:
		}

		if comment != "" {
			if err := upsertPullRequestComment(
				ctx,
				vcsProvider,
				vcsConnector,
				prInfo,
				fmt.Sprintf("%s\n\n%s", commentPrefix, comment),
				func(content string) bool { return strings.HasPrefix(content, commentPrefix) },
			); err != nil {
				slog.Error("failed to upsert comment", slog.String("pr", prInfo.url), log.BBError(err))
				return c.String(http.StatusOK, fmt.Sprintf("failed to create pull request comment, error %v", err))
			}
		}

		return c.String(http.StatusOK, fmt.Sprintf("successfully handle the pull request %v", prInfo.url))
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

func (s *Service) sqlReviewWithPRInfo(ctx context.Context, project *store.ProjectMessage, vcsConnector *store.VCSConnectorMessage, vcsType storepb.VCSType, prInfo *pullRequestInfo) (string, error) {
	instance, database, err := s.getDatabaseSample(ctx, project, vcsConnector)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get database sample")
	}

	content := []string{}
	errorCount := 0
	warnCount := 0

	for _, change := range prInfo.changes {
		adviceStatus, advices, err := s.sqlService.SQLReviewCheck(
			ctx,
			change.content,
			v1pb.CheckRequest_CHANGE_TYPE_UNSPECIFIED,
			instance,
			database,
			nil, /* Override Metadata */
		)
		if err != nil {
			slog.Error("failed to exec sql review", slog.String("instance", instance.ResourceID), slog.String("database", database.DatabaseName), log.BBError(err))
			continue
		}
		if adviceStatus == storepb.Advice_SUCCESS || adviceStatus == storepb.Advice_STATUS_UNSPECIFIED {
			continue
		}

		adviceMessage := []string{}
		for _, advice := range advices {
			if advice.Status == v1pb.Advice_ERROR {
				errorCount++
			} else if advice.Status == v1pb.Advice_WARNING {
				warnCount++
			}
			message := fmt.Sprintf("- **[%s]** %s ([line%d](%s))", advice.Status.String(), advice.Title, advice.Line, getFileWebURLInPR(change.webURL, advice.Line, vcsType))
			adviceMessage = append(adviceMessage, message)
		}

		if len(adviceMessage) > 0 {
			if len(content) > 0 {
				content = append(content, "\n")
			}
			content = append(content, fmt.Sprintf("SQL review for [%s](%s)\n", change.path, change.webURL))
			// We have to use at least 2 \n for Bitbucket.
			// The API docs for Bitbucket sucks.
			// https://community.atlassian.com/t5/Bitbucket-questions/How-to-post-html-comments-on-pull-request-via-2-0-api/qaq-p/1066809
			content = append(content, strings.Join(adviceMessage, "\n\n"))
		}
	}

	if len(content) == 0 {
		return "", nil
	}

	return fmt.Sprintf("\n%d errors, %d warnings\n\n---\n\n%s", errorCount, warnCount, strings.Join(content, "\n")), nil
}

func (s *Service) createIssueFromPRInfo(ctx context.Context, project *store.ProjectMessage, vcsProvider *store.VCSProviderMessage, vcsConnector *store.VCSConnectorMessage, prInfo *pullRequestInfo) (*v1pb.Issue, error) {
	user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
	if !ok {
		return nil, errors.Errorf("cannot found user in context")
	}
	creatorID := user.ID
	creatorName := common.FormatUserUID(user.ID)

	instance, _, err := s.getDatabaseSample(ctx, project, vcsConnector)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database sample")
	}

	var sheets []int
	for _, change := range prInfo.changes {
		sheet, err := s.sheetManager.CreateSheet(ctx, &store.SheetMessage{
			CreatorID:  creatorID,
			ProjectUID: project.UID,
			Title:      change.path,
			Statement:  change.content,

			Payload: &storepb.SheetPayload{
				Engine: instance.Engine,
			},
		})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create sheet for file %s", change.path)
		}
		sheets = append(sheets, sheet.UID)
	}

	steps, err := s.getChangeSteps(ctx, project, vcsConnector, prInfo.changes, sheets)
	if err != nil {
		return nil, err
	}

	plan, err := s.planService.CreatePlan(ctx, &v1pb.CreatePlanRequest{
		Parent: common.FormatProject(project.ResourceID),
		Plan: &v1pb.Plan{
			Title: prInfo.title,
			Steps: steps,
			VcsSource: &v1pb.Plan_VCSSource{
				VcsConnector:   fmt.Sprintf("%s%s/%s%s", common.ProjectNamePrefix, vcsConnector.ProjectID, common.VCSConnectorPrefix, vcsConnector.ResourceID),
				PullRequestUrl: prInfo.url,
				VcsType:        v1pb.VCSType(vcsProvider.Type),
			},
		},
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create plan")
	}
	issue, err := s.issueService.CreateIssue(ctx, &v1pb.CreateIssueRequest{
		Parent: common.FormatProject(project.ResourceID),
		Issue: &v1pb.Issue{
			Title:       prInfo.title,
			Description: prInfo.description,
			Type:        v1pb.Issue_DATABASE_CHANGE,
			Assignee:    common.FormatUserEmail(s.store.GetSystemBotUser(ctx).Email),
			Plan:        plan.Name,
		},
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create issue")
	}
	if _, err := s.rolloutService.CreateRollout(ctx, &v1pb.CreateRolloutRequest{
		Parent: common.FormatProject(project.ResourceID),
		Rollout: &v1pb.Rollout{
			Plan: plan.Name,
		},
	}); err != nil {
		return nil, errors.Wrapf(err, "failed to create rollout")
	}

	issueUID, err := strconv.Atoi(issue.Uid)
	if err != nil {
		return nil, err
	}
	// Create audit log after successfully creating the issue from the push event.
	if err := s.store.CreateAuditLog(ctx, &storepb.AuditLog{
		Parent:   project.GetName(),
		Method:   store.AuditLogMethodProjectRepositoryPush.String(),
		Resource: issue.Name,
		User:     creatorName,
		Severity: storepb.AuditLog_INFO,
		Request:  "",
		Response: "",
		Status:   nil,
	}); err != nil {
		slog.Error("failed to create audit log after creating issue from push event", slog.Int("issue", issueUID), log.BBError(err))
	}

	return issue, nil
}

type isTargetComment func(string) bool

func getCommentIDByContent(ctx context.Context, provider vcs.Provider, repositoryID, pullRequestID string, checkComment isTargetComment) (string, error) {
	comments, err := provider.ListPullRequestComments(ctx, repositoryID, pullRequestID)
	if err != nil {
		return "", errors.Wrapf(err, "failed to list comments for PR %s", pullRequestID)
	}

	for _, comment := range comments {
		if checkComment(comment.Content) {
			return comment.ID, nil
		}
	}
	return "", nil
}

func upsertPullRequestComment(
	ctx context.Context,
	vcsProvider *store.VCSProviderMessage,
	vcsConnector *store.VCSConnectorMessage,
	prInfo *pullRequestInfo,
	comment string,
	checkComment isTargetComment,
) error {
	pullRequestID := getPullRequestID(prInfo.url)
	provider := vcs.Get(
		vcsProvider.Type,
		vcs.ProviderConfig{InstanceURL: vcsProvider.InstanceURL, AuthToken: vcsProvider.AccessToken},
	)

	existedCommentID, err := getCommentIDByContent(ctx, provider, vcsConnector.Payload.ExternalId, pullRequestID, checkComment)
	if err != nil {
		return err
	}
	if existedCommentID != "" {
		if err := provider.UpdatePullRequestComment(ctx, vcsConnector.Payload.ExternalId, pullRequestID, &vcs.PullRequestComment{
			ID:      existedCommentID,
			Content: comment,
		}); err != nil {
			return errors.Wrapf(err, `failed to update comment for PR "%s"`, prInfo.url)
		}
		return nil
	}

	if err := provider.CreatePullRequestComment(ctx, vcsConnector.Payload.ExternalId, pullRequestID, comment); err != nil {
		return errors.Wrapf(err, `failed to create comment for PR "%s"`, prInfo.url)
	}
	return nil
}

func (s *Service) getDatabaseSample(
	ctx context.Context,
	project *store.ProjectMessage,
	vcsConnector *store.VCSConnectorMessage,
) (*store.InstanceMessage, *store.DatabaseMessage, error) {
	sample, err := func() (*store.DatabaseMessage, error) {
		if dbg := vcsConnector.Payload.GetDatabaseGroup(); dbg != "" {
			projectID, databaseGroupID, err := common.GetProjectIDDatabaseGroupID(dbg)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get project id and database group id from %q", dbg)
			}
			if projectID != project.ResourceID {
				return nil, errors.Errorf("project id %q in databaseGroup %q does not match project id %q in plan config", projectID, dbg, project.ResourceID)
			}
			databaseGroup, err := s.store.GetDatabaseGroup(ctx, &store.FindDatabaseGroupMessage{ProjectUID: &project.UID, ResourceID: &databaseGroupID})
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get database group %q", databaseGroupID)
			}
			if databaseGroup == nil {
				return nil, errors.Errorf("database group %q not found", databaseGroupID)
			}
			allDatabases, err := s.store.ListDatabases(ctx, &store.FindDatabaseMessage{ProjectID: &project.ResourceID})
			if err != nil {
				return nil, errors.Wrapf(err, "failed to list databases for project %q", project.ResourceID)
			}

			matchedDatabases, _, err := utils.GetMatchedAndUnmatchedDatabasesInDatabaseGroup(ctx, databaseGroup, allDatabases)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get matched and unmatched databases in database group %q", databaseGroupID)
			}
			if len(matchedDatabases) == 0 {
				return nil, errors.Errorf("no matched databases found in database group %q", databaseGroupID)
			}
			return matchedDatabases[0], nil
		}
		allDatabases, err := s.store.ListDatabases(ctx, &store.FindDatabaseMessage{ProjectID: &project.ResourceID})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to list databases for project %q", project.ResourceID)
		}
		if len(allDatabases) == 0 {
			return nil, errors.Errorf("no database in the project %q", project.ResourceID)
		}
		return allDatabases[0], nil
	}()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to get sample database")
	}

	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &sample.InstanceID})
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to get instance")
	}
	if instance == nil {
		return nil, nil, errors.Errorf("instance not found")
	}
	return instance, sample, nil
}

func (s *Service) getChangeSteps(
	ctx context.Context,
	project *store.ProjectMessage,
	vcsConnector *store.VCSConnectorMessage,
	changes []*fileChange,
	sheetUIDList []int,
) ([]*v1pb.Plan_Step, error) {
	if vcsConnector.Payload.DatabaseGroup != "" {
		step := &v1pb.Plan_Step{}
		for i, change := range changes {
			step.Specs = append(step.Specs, &v1pb.Plan_Spec{
				Id: uuid.NewString(),
				Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
					ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
						Type:          change.changeType,
						Target:        vcsConnector.Payload.DatabaseGroup,
						Sheet:         fmt.Sprintf("projects/%s/sheets/%d", project.ResourceID, sheetUIDList[i]),
						SchemaVersion: change.version,
					},
				},
			})
		}
		return []*v1pb.Plan_Step{
			step,
		}, nil
	}

	databases, err := s.listDatabases(ctx, project)
	if err != nil {
		return nil, err
	}

	var steps []*v1pb.Plan_Step
	for i, database := range databases {
		if i == 0 || databases[i].EffectiveEnvironmentID != databases[i-1].EffectiveEnvironmentID {
			steps = append(steps, &v1pb.Plan_Step{})
		}
		step := steps[len(steps)-1]
		for i, change := range changes {
			step.Specs = append(step.Specs, &v1pb.Plan_Spec{
				Id: uuid.NewString(),
				Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
					ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
						Type:          change.changeType,
						Target:        common.FormatDatabase(database.InstanceID, database.DatabaseName),
						Sheet:         fmt.Sprintf("projects/%s/sheets/%d", project.ResourceID, sheetUIDList[i]),
						SchemaVersion: change.version,
					},
				},
			})
		}
	}

	return steps, nil
}

func (s *Service) listDatabases(ctx context.Context, project *store.ProjectMessage) ([]*store.DatabaseMessage, error) {
	databases, err := s.store.ListDatabases(ctx, &store.FindDatabaseMessage{ProjectID: &project.ResourceID})
	if err != nil {
		return nil, err
	}
	environments, err := s.store.ListEnvironmentV2(ctx, &store.FindEnvironmentMessage{})
	if err != nil {
		return nil, err
	}
	environmentOrders := make(map[string]int32)
	for _, environment := range environments {
		environmentOrders[environment.ResourceID] = environment.Order
	}
	sort.Slice(databases, func(i, j int) bool {
		return environmentOrders[databases[i].EffectiveEnvironmentID] < environmentOrders[databases[j].EffectiveEnvironmentID]
	})

	return databases, nil
}
