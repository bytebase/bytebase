package v1

import (
	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/advisor"
)

// Environment is the API message for an environment.
type Environment struct {
	ID int `json:"id"`

	// Related fields
	EnvironmentTierPolicy  *api.EnvironmentTierPolicy  `json:"environmentTierPolicy"`
	PipelineApprovalPolicy *api.PipelineApprovalPolicy `json:"pipelineApprovalPolicy"`
	BackupPlanPolicy       *api.BackupPlanPolicy       `json:"backupPlanPolicy"`
	SQLReviewPolicy        *advisor.SQLReviewPolicy    `json:"sqlReviewPolicy"`

	// Domain specific fields
	Name  string `json:"name"`
	Order int    `json:"order"`
}

// EnvironmentUpsert is the API message for upsert an environment.
type EnvironmentUpsert struct {
	// Related fields
	EnvironmentTierPolicy  *api.EnvironmentTierPolicy  `json:"environmentTierPolicy"`
	PipelineApprovalPolicy *api.PipelineApprovalPolicy `json:"pipelineApprovalPolicy"`
	BackupPlanPolicy       *api.BackupPlanPolicy       `json:"backupPlanPolicy"`
	SQLReviewPolicy        *advisor.SQLReviewPolicy    `json:"sqlReviewPolicy"`

	// Domain specific fields
	Name  *string `json:"name"`
	Order *int    `json:"order"`
}
