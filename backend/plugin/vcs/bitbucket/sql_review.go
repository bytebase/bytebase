package bitbucket

import (
	_ "embed"
	"fmt"

	"gopkg.in/yaml.v2"

	"github.com/go-faster/errors"
)

type pipelineStep struct {
	Name   string   `yaml:"name"`
	Image  string   `yaml:"image"`
	Script []string `yaml:"script"`
}

const (
	// CIFilePath is the local path for BitBucket ci file.
	CIFilePath = "bitbucket-pipelines.yml"
	// SQLReviewScriptFilePath is the local path for SQL review CI script.
	SQLReviewScriptFilePath = ".pipeline/bytebase-sql-review.sh"
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

	if _, ok := findSQLReviewCI(prSteps); !ok {
		stepStr := getSQLReviewStep(endpoint, webhookSecret)
		pipelineStep := &pipelineStep{}
		if err := yaml.Unmarshal([]byte(stepStr), pipelineStep); err != nil {
			return "", errors.Wrapf(err, "failed to parse pipeline step")
		}
		prSteps = append(prSteps, pipelineStep)
	}

	bitBucketCI["pipelines"].(map[string]any)["pull-requests"].(map[string]any)["**"] = prSteps

	newContent, err := yaml.Marshal(bitBucketCI)
	if err != nil {
		return "", err
	}

	return string(newContent), nil
}

func findSQLReviewCI(steps []any) (map[string]any, bool) {
	for _, data := range steps {
		if val, ok := data.(map[string]any); ok {
			if val["name"] == "Bytebase SQL Review" {
				return val, true
			}
		}
	}

	return nil, false
}

// pull request pipeline
// https://support.atlassian.com/bitbucket-cloud/docs/pipeline-start-conditions/#Pull-Requests
// https://stackoverflow.com/questions/55019205/how-to-run-pipeline-only-on-pull-request-to-master-branch

// run review script
// https://community.atlassian.com/t5/Bitbucket-questions/Can-bitbucket-pipeline-yml-refer-another-yml-file/qaq-p/1346283

// test artifacts
// https://support.atlassian.com/bitbucket-cloud/docs/test-reporting-in-pipelines/
// https://stackoverflow.com/questions/73716566/how-to-create-test-report-files-for-bitbucket-pipelines

// environment variables
// https://support.atlassian.com/bitbucket-cloud/docs/variables-and-secrets/

// other references
// https://support.atlassian.com/bitbucket-cloud/docs/yaml-anchors/
// https://chrisfrew.in/blog/mastering-bitbucket-pipelines-for-ci-and-cd/

// steps:
// 1. upsert sql review bash script
// 2. create or update the pipeline yaml, add sql review step

/*
example:
- step:
	name: Build and test
	image: node:10.15.0
	script:
		- npm install
		- npm test
		- npm run build
	artifacts: # defining the artifacts to be passed to each future step.
		- dist/**
		- reports/*.txt
*/
