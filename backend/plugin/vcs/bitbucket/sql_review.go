package bitbucket

import (
	_ "embed"
	"fmt"

	"github.com/pkg/errors"

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

func getSQLReviewStep(endpoint, webhookSecret string) string {
	return fmt.Sprintf(
		sqlReviewPipelineStep,
		pipelineStepName,
		SQLReviewScriptFilePath,
		endpoint,
		// TODO(ed): insert secret in the repo environment variable and use vcs.SQLReviewAPISecretName.
		webhookSecret,
	)
}

func SetupBitBucketCI(bitBucketCI map[string]any, endpoint, webhookSecret string) (string, error) {
	if _, ok := bitBucketCI["image"]; !ok {
		bitBucketCI["image"] = "atlassian/default-image:4"
	}
	if _, ok := bitBucketCI["pipelines"]; !ok {
		bitBucketCI["pipelines"] = make(map[string]any)
	}
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

	stepStr := getSQLReviewStep(endpoint, webhookSecret)
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
