package azure

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/vcs/internal/oauth"
)

// sqlReviewPipeline is the Azure pipeline for SQL review in VCS workflow.
//
//go:embed bytebase-sql-review.yml
var sqlReviewPipeline string

const (
	// SQLReviewPipelineFilePath is the SQL review pipeline file path.
	SQLReviewPipelineFilePath = ".pipelines/pipeline.bytebase-sql-review.yml"
	// sqlReviewPipelineName is the pipeline name for SQL review.
	sqlReviewPipelineName = "Bytebase SQL Review"
)

// SetupSQLReviewCI will setup the SQL review CI content with SQL review endpoint.
func SetupSQLReviewCI(endpoint, branch, token string) string {
	return fmt.Sprintf(sqlReviewPipeline, endpoint, branch, token)
}

type pipeline struct {
	ID   int    `jsno:"id"`
	Name string `jsno:"name"`
}

type pipelineCreate struct {
	Name          string          `json:"name"`
	Configuration *pipelineConfig `json:"configuration"`
}

type pipelineRepository struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type pipelineConfig struct {
	Type       string              `json:"type"`
	Path       string              `json:"path"`
	Repository *pipelineRepository `json:"repository"`
}

type branchPolicyCreate struct {
	IsBlocking bool                 `json:"isBlocking"`
	IsDeleted  bool                 `json:"isDeleted"`
	IsEnabled  bool                 `json:"isEnabled"`
	Settings   *branchPolicySetting `json:"settings"`
	Type       *policy              `json:"type"`
}

type branchPolicySetting struct {
	// BuildDefinitionID is the pipeline ID.
	BuildDefinitionID       int                  `json:"buildDefinitionId"`
	DisplayName             string               `json:"displayName"`
	ManualQueueOnly         bool                 `json:"manualQueueOnly"`
	QueueOnSourceUpdateOnly bool                 `json:"queueOnSourceUpdateOnly"`
	Scope                   []*branchPolicyScope `json:"scope"`
	// ValidDuration is the expiration in hours for build artifacts.
	ValidDuration int `json:"validDuration"`
}

type branchPolicyScope struct {
	// MatchKind should always be "Exact"
	MatchKind string `json:"matchKind"`
	// RefName is the target branch name.
	RefName      string `json:"refName"`
	RepositoryID string `json:"repositoryId"`
}

// policy is the API message for azure policy.
// It is hard code with specific id.
// We can check all the policies in docs: https://learn.microsoft.com/en-us/rest/api/azure/devops/pipelines/pipelines/list?view=azure-devops-rest-7.1
// For branch validation policy, we will hardcode the id as "0609b952-1397-4640-95ec-e00a01b2c241".
type policy struct {
	ID string `json:"id"`
}

// EnableSQLReviewCI enables the SQL review pipeline and policy.
func EnableSQLReviewCI(ctx context.Context, oauthCtx common.OauthContext, repositoryID, branch string) error {
	pipeline, err := createSQLReviewPipeline(ctx, oauthCtx, repositoryID)
	if err != nil {
		return err
	}
	return createSQLReviewBranchPolicy(ctx, oauthCtx, repositoryID, branch, pipeline.ID)
}

// createSQLReviewPipeline creates the SQL Review pipeline.
//
// Docs: https://learn.microsoft.com/en-us/rest/api/azure/devops/pipelines/pipelines/create?view=azure-devops-rest-7.1
func createSQLReviewPipeline(ctx context.Context, oauthCtx common.OauthContext, repositoryID string) (*pipeline, error) {
	parts := strings.Split(repositoryID, "/")
	if len(parts) != 3 {
		return nil, errors.Errorf("invalid repository ID %q", repositoryID)
	}

	values := &url.Values{}
	values.Set("api-version", "7.1-preview.1")

	client := &http.Client{}
	apiURL := fmt.Sprintf("https://dev.azure.com/%s/%s/_apis/pipelines?%s", url.PathEscape(parts[0]), url.PathEscape(parts[1]), values.Encode())
	payload := &pipelineCreate{
		Name: sqlReviewPipelineName,
		Configuration: &pipelineConfig{
			Type: "yaml",
			Path: SQLReviewPipelineFilePath,
			Repository: &pipelineRepository{
				ID:   parts[2],
				Type: "azureReposGit",
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.Wrap(err, "marshal branch create")
	}

	code, _, resp, err := oauth.Post(
		ctx,
		client,
		apiURL,
		&oauthCtx.AccessToken,
		bytes.NewReader(body),
		tokenRefresher(
			oauthContext{
				RefreshToken: oauthCtx.RefreshToken,
				ClientSecret: oauthCtx.ClientSecret,
				RedirectURL:  oauthCtx.RedirectURL,
			},
			oauthCtx.Refresher,
		),
	)
	if err != nil {
		return nil, errors.Wrapf(err, "POST %s", apiURL)
	}
	if code != 200 {
		return nil, errors.Errorf("non-200 POST %s status code %d with body %q", apiURL, code, string(resp))
	}

	result := new(pipeline)
	if err := json.Unmarshal([]byte(resp), result); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal response body")
	}

	return result, nil
}

// createSQLReviewPolicy creates the SQL Review policy.
//
// Docs: https://learn.microsoft.com/en-us/rest/api/azure/devops/policy/configurations/create?view=azure-devops-rest-7.0
func createSQLReviewBranchPolicy(ctx context.Context, oauthCtx common.OauthContext, repositoryID, branch string, pipelineID int) error {
	parts := strings.Split(repositoryID, "/")
	if len(parts) != 3 {
		return errors.Errorf("invalid repository ID %q", repositoryID)
	}

	values := &url.Values{}
	values.Set("api-version", "7.1-preview.1")

	client := &http.Client{}
	apiURL := fmt.Sprintf("https://dev.azure.com/%s/%s/_apis/policy/configurations?%s", url.PathEscape(parts[0]), url.PathEscape(parts[1]), values.Encode())
	payload := &branchPolicyCreate{
		IsBlocking: true,
		IsDeleted:  false,
		IsEnabled:  true,
		Settings: &branchPolicySetting{
			BuildDefinitionID:       pipelineID,
			DisplayName:             sqlReviewPipelineName,
			ManualQueueOnly:         false,
			QueueOnSourceUpdateOnly: true,
			ValidDuration:           720,
			Scope: []*branchPolicyScope{
				{
					MatchKind:    "Exact",
					RefName:      fmt.Sprintf("refs/heads/%s", branch),
					RepositoryID: parts[2],
				},
			},
		},
		Type: &policy{
			ID: "0609b952-1397-4640-95ec-e00a01b2c241",
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return errors.Wrap(err, "marshal policy create")
	}

	code, _, resp, err := oauth.Post(
		ctx,
		client,
		apiURL,
		&oauthCtx.AccessToken,
		bytes.NewReader(body),
		tokenRefresher(
			oauthContext{
				RefreshToken: oauthCtx.RefreshToken,
				ClientSecret: oauthCtx.ClientSecret,
				RedirectURL:  oauthCtx.RedirectURL,
			},
			oauthCtx.Refresher,
		),
	)
	if err != nil {
		return errors.Wrapf(err, "POST %s", apiURL)
	}
	if code != 200 {
		return errors.Errorf("non-200 POST %s status code %d with body %q", apiURL, code, string(resp))
	}

	return nil
}
