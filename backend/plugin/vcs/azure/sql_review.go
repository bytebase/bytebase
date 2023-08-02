package azure

import (
	_ "embed"
	"fmt"
)

// sqlReviewPipeline is the Azure pipeline for SQL review in VCS workflow.
//
//go:embed bytebase-sql-review.yml
var sqlReviewPipeline string

const (
	// SQLReviewPipelineFilePath is the SQL review pipeline file path.
	SQLReviewPipelineFilePath = ".pipelines/pipeline.bytebase-sql-review.yml"
)

// SetupSQLReviewCI will setup the SQL review CI content with SQL review endpoint.
// Users need to enable the branch policy to trigger pipeline in the pull request.
// https://learn.microsoft.com/en-us/azure/devops/pipelines/repos/azure-repos-git?view=azure-devops&tabs=yaml#pr-triggers
func SetupSQLReviewCI(endpoint, branch string) string {
	return fmt.Sprintf(sqlReviewPipeline, endpoint, branch)
}
