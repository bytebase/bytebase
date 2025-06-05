package base

// PipelineApprovalValue is value for approval policy.
type PipelineApprovalValue string

// PolicyResourceType is the resource type for a policy.
type PolicyResourceType string

// ReservedTag is the reserved tags for bb.policy.tag.
type ReservedTag string

const (
	// DefaultPolicyID is the ID of the default policy.
	DefaultPolicyID int = 0

	// PipelineApprovalValueManualNever means the pipeline will automatically be approved without user intervention.
	PipelineApprovalValueManualNever PipelineApprovalValue = "MANUAL_APPROVAL_NEVER"
	// PipelineApprovalValueManualAlways means the pipeline should be manually approved by user to proceed.
	PipelineApprovalValueManualAlways PipelineApprovalValue = "MANUAL_APPROVAL_ALWAYS"

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
