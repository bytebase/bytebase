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
	"github.com/bytebase/bytebase/backend/plugin/vcs"

	"github.com/bytebase/bytebase/backend/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
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

		var prInfo *pullRequestInfo
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
			eventType := c.Request().Header.Get("X-GitHub-Event")
			// https://docs.github.com/en/developers/webhooks-and-events/webhooks/about-webhooks#ping-event
			// When we create a new webhook, GitHub will send us a simple ping event to let us know we've set up the webhook correctly.
			// We respond to this event so as not to mislead users.
			switch eventType {
			case "ping":
				return c.String(http.StatusOK, "OK")
			case "pull_request":
			default:
				return c.String(http.StatusOK, "OK")
			}

			prInfo, err = getGitHubPullRequestInfo(ctx, vcsProvider, vcsConnector, body)
			if err != nil {
				return c.String(http.StatusOK, fmt.Sprintf("failed to get pr info from pull request, error %v", err))
			}
		case vcs.GitLab:
			secretToken := c.Request().Header.Get("X-Gitlab-Token")
			if secretToken != vcsConnector.Payload.WebhookSecretToken {
				return c.String(http.StatusOK, fmt.Sprintf("invalid webhook secret token %q", secretToken))
			}

			prInfo, err = getGitLabPullRequestInfo(ctx, vcsProvider, vcsConnector, body)
			if err != nil {
				return c.String(http.StatusOK, fmt.Sprintf("failed to get pr info from pull request, error %v", err))
			}
		case vcs.Bitbucket:
			eventType := c.Request().Header.Get("X-Event-Key")
			switch eventType {
			case "pullrequest:created", "pullrequest:updated":
				return c.String(http.StatusOK, "OK")
			case "pullrequest:fulfilled":
			default:
				return c.String(http.StatusOK, "OK")
			}

			prInfo, err = getBitBucketPullRequestInfo(ctx, vcsProvider, vcsConnector, body)
			if err != nil {
				return c.String(http.StatusOK, fmt.Sprintf("failed to get pr info from pull request, error %v", err))
			}
		default:
			return nil
		}
		if err := s.createIssueFromPRInfo(ctx, project, vcsConnector, prInfo); err != nil {
			return c.String(http.StatusOK, fmt.Sprintf("failed to create issue from pull request %s, error %v", prInfo.url, err))
		}
		return nil
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

func (s *Service) createIssueFromPRInfo(ctx context.Context, project *store.ProjectMessage, vcsConnector *store.VCSConnectorMessage, prInfo *pullRequestInfo) error {
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
			VcsSource: &v1pb.Plan_VCSSource{
				VcsConnector:   fmt.Sprintf("%s%s/%s%s", common.ProjectNamePrefix, vcsConnector.ProjectID, common.VCSConnectorPrefix, vcsConnector.ResourceID),
				PullRequestUrl: prInfo.url,
			},
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
