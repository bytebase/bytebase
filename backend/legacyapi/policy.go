package api

// PolicyType is the type or name of a policy.
type PolicyType string

// PipelineApprovalValue is value for approval policy.
type PipelineApprovalValue string

// EnvironmentTierValue is the value for environment tier policy.
type EnvironmentTierValue string

// PolicyResourceType is the resource type for a policy.
type PolicyResourceType string

// ReservedTag is the reserved tags for bb.policy.tag.
type ReservedTag string

const (
	// DefaultPolicyID is the ID of the default policy.
	DefaultPolicyID int = 0

	// PolicyTypeRollout is the rollout policy type.
	PolicyTypeRollout PolicyType = "bb.policy.rollout"
	// PolicyTypeEnvironmentTier is the tier of an environment.
	PolicyTypeEnvironmentTier PolicyType = "bb.policy.environment-tier"
	// PolicyTypeMaskingException is the masking exception policy type.
	PolicyTypeMaskingException PolicyType = "bb.policy.masking-exception"
	// PolicyTypeDisableCopyData is the disable copy data policy type.
	PolicyTypeDisableCopyData PolicyType = "bb.policy.disable-copy-data"
	// PolicyTypeExportData is the policy type for data export control.
	PolicyTypeExportData PolicyType = "bb.policy.export-data"
	// PolicyTypeQueryData is the policy type for data query control.
	PolicyTypeQueryData PolicyType = "bb.policy.query-data"
	// PolicyTypeMaskingRule is the masking rule policy type.
	PolicyTypeMaskingRule PolicyType = "bb.policy.masking-rule"
	// PolicyTypeRestrictIssueCreationForSQLReview is the policy type for restricting issue creation for SQL review.
	PolicyTypeRestrictIssueCreationForSQLReview PolicyType = "bb.policy.restrict-issue-creation-for-sql-review"
	// PolicyTypeIAM is the policy for IAM.
	PolicyTypeIAM PolicyType = "bb.policy.iam"
	// PolicyTypeTag is the policy type for resource tags.
	PolicyTypeTag PolicyType = "bb.policy.tag"
	// PolicyTypeDataSourceQuery is the policy type for data source query.
	PolicyTypeDataSourceQuery PolicyType = "bb.policy.data-source-query"

	// PipelineApprovalValueManualNever means the pipeline will automatically be approved without user intervention.
	PipelineApprovalValueManualNever PipelineApprovalValue = "MANUAL_APPROVAL_NEVER"
	// PipelineApprovalValueManualAlways means the pipeline should be manually approved by user to proceed.
	PipelineApprovalValueManualAlways PipelineApprovalValue = "MANUAL_APPROVAL_ALWAYS"

	// EnvironmentTierValueProtected is PROTECTED environment tier value.
	EnvironmentTierValueProtected EnvironmentTierValue = "PROTECTED"
	// EnvironmentTierValueUnprotected is UNPROTECTED environment tier value.
	EnvironmentTierValueUnprotected EnvironmentTierValue = "UNPROTECTED"

	// PolicyResourceTypeUnknown is the unknown resource type.
	PolicyResourceTypeUnknown PolicyResourceType = ""
	// PolicyResourceTypeWorkspace is the resource type for workspaces.
	PolicyResourceTypeWorkspace PolicyResourceType = "WORKSPACE"
	// PolicyResourceTypeEnvironment is the resource type for environments.
	PolicyResourceTypeEnvironment PolicyResourceType = "ENVIRONMENT"
	// PolicyResourceTypeProject is the resource type for projects.
	PolicyResourceTypeProject PolicyResourceType = "PROJECT"
	// PolicyResourceTypeInstance is the resource type for instances.
	PolicyResourceTypeInstance PolicyResourceType = "INSTANCE"

	// ReservedTagReviewConfig is the tag for review config.
	ReservedTagReviewConfig ReservedTag = "bb.tag.review_config"
)

var (
	// AllowedResourceTypes includes allowed resource types for each policy type.
	AllowedResourceTypes = map[PolicyType][]PolicyResourceType{
		PolicyTypeRollout:                           {PolicyResourceTypeEnvironment},
		PolicyTypeEnvironmentTier:                   {PolicyResourceTypeEnvironment},
		PolicyTypeTag:                               {PolicyResourceTypeEnvironment, PolicyResourceTypeProject},
		PolicyTypeDisableCopyData:                   {PolicyResourceTypeEnvironment, PolicyResourceTypeProject},
		PolicyTypeExportData:                        {PolicyResourceTypeWorkspace},
		PolicyTypeQueryData:                         {PolicyResourceTypeWorkspace},
		PolicyTypeMaskingRule:                       {PolicyResourceTypeWorkspace},
		PolicyTypeMaskingException:                  {PolicyResourceTypeProject},
		PolicyTypeRestrictIssueCreationForSQLReview: {PolicyResourceTypeWorkspace, PolicyResourceTypeProject},
		PolicyTypeIAM:                               {PolicyResourceTypeWorkspace},
		PolicyTypeDataSourceQuery:                   {PolicyResourceTypeEnvironment, PolicyResourceTypeProject},
	}
)
