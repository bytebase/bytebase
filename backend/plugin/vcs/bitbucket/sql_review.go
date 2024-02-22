package bitbucket

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/vcs"
	"github.com/bytebase/bytebase/backend/plugin/vcs/internal/oauth"

	"gopkg.in/yaml.v3"
)

type pipelineStep struct {
	Name   string   `yaml:"name"`
	Image  string   `yaml:"image"`
	Script []string `yaml:"script"`
}

type pipeline struct {
	Step *pipelineStep `yaml:"step"`
}

type pipelineConfig struct {
	Enabled bool `json:"enabled"`
}

const (
	// CIFilePath is the local path for BitBucket ci file.
	CIFilePath = "bitbucket-pipelines.yml"
	// SQLReviewScriptFilePath is the local path for SQL review CI script.
	SQLReviewScriptFilePath = ".pipeline/bytebase-sql-review.sh"
	pipelineStepName        = "Bytebase SQL Review"
)

// sqlReviewPipelineStep is the pipeline step for SQL review in VCS workflow.
//
//go:embed pipeline-step.yml
var sqlReviewPipelineStep string

// SQLReviewScript is the script file for SQL review in VCS workflow.
//
//go:embed bytebase-sql-review.sh
var SQLReviewScript string

func getSQLReviewStep(endpoint string) string {
	return fmt.Sprintf(
		sqlReviewPipelineStep,
		pipelineStepName,
		SQLReviewScriptFilePath,
		endpoint,
		vcs.SQLReviewAPISecretName,
	)
}

// EnableSQLReviewCI enables the pipeline.
//
// Docs: https://developer.atlassian.com/cloud/bitbucket/rest/api-group-pipelines/#api-repositories-workspace-repo-slug-pipelines-config-put.
func EnableSQLReviewCI(ctx context.Context, oauthCtx *common.OauthContext, apiURL, instanceURL, repositoryID string) error {
	body, err := json.Marshal(
		pipelineConfig{
			Enabled: true,
		},
	)
	if err != nil {
		return errors.Wrap(err, "marshal pipeline config")
	}

	client := &http.Client{}
	url := fmt.Sprintf("%s/repositories/%s/pipelines_config", apiURL, repositoryID)
	code, _, resp, err := oauth.Put(
		ctx,
		client,
		url,
		&oauthCtx.AccessToken,
		bytes.NewReader(body),
		tokenRefresher(
			instanceURL,
			oauthContext{
				ClientID:     oauthCtx.ClientID,
				ClientSecret: oauthCtx.ClientSecret,
				RefreshToken: oauthCtx.RefreshToken,
			},
			oauthCtx.Refresher,
		),
	)
	if err != nil {
		return errors.Wrapf(err, "PUT %s", url)
	}
	if code >= 300 {
		return errors.Errorf("failed to update pipeline config from URL %s, status code: %d, body: %s",
			url,
			code,
			resp,
		)
	}
	return nil
}

func SetupBitBucketCI(bitBucketCI map[string]any, endpoint string) (string, error) {
	if _, ok := bitBucketCI["image"]; !ok {
		bitBucketCI["image"] = "atlassian/default-image:4"
	}
	if _, ok := bitBucketCI["pipelines"]; !ok {
		bitBucketCI["pipelines"] = make(map[string]any)
	}
	// enable pipeline for pull request
	// https://support.atlassian.com/bitbucket-cloud/docs/pipeline-start-conditions/#Pull-Requests
	// https://stackoverflow.com/questions/55019205/how-to-run-pipeline-only-on-pull-request-to-master-branch
	if _, ok := bitBucketCI["pipelines"].(map[string]any)["pull-requests"]; !ok {
		bitBucketCI["pipelines"].(map[string]any)["pull-requests"] = make(map[string]any)
	}
	if _, ok := bitBucketCI["pipelines"].(map[string]any)["pull-requests"].(map[string]any)["**"]; !ok {
		bitBucketCI["pipelines"].(map[string]any)["pull-requests"].(map[string]any)["**"] = []any{}
	}

	var prSteps []any
	switch steps := bitBucketCI["pipelines"].(map[string]any)["pull-requests"].(map[string]any)["**"].(type) {
	case []any:
		prSteps = append(prSteps, steps...)
	default:
		prSteps = append(prSteps, steps)
	}

	stepStr := getSQLReviewStep(endpoint)
	pipelineStep := &pipelineStep{}
	if err := yaml.Unmarshal([]byte(stepStr), pipelineStep); err != nil {
		return "", errors.Wrapf(err, "failed to parse pipeline step")
	}

	index := findSQLReviewStepIndex(prSteps)
	if index >= 0 {
		prSteps = append(prSteps[:index], prSteps[index+1:]...)
	}
	prSteps = append(prSteps, &pipeline{
		Step: pipelineStep,
	})

	bitBucketCI["pipelines"].(map[string]any)["pull-requests"].(map[string]any)["**"] = prSteps

	newContent, err := yaml.Marshal(bitBucketCI)
	if err != nil {
		return "", err
	}

	return string(newContent), nil
}

func findSQLReviewStepIndex(steps []any) int {
	for i, data := range steps {
		if val, ok := data.(map[string]any); ok {
			if step, ok := val["step"].(map[string]any); ok {
				if step["name"] == pipelineStepName {
					return i
				}
			}
		}
	}

	return -1
}
