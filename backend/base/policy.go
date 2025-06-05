package base

// PolicyResourceType is the resource type for a policy.
type PolicyResourceType string

const (
	// PolicyResourceTypeUnknown is the unknown resource type.
	PolicyResourceTypeUnknown PolicyResourceType = ""
	// PolicyResourceTypeWorkspace is the resource type for workspaces.
	PolicyResourceTypeWorkspace PolicyResourceType = "WORKSPACE"
	// PolicyResourceTypeEnvironment is the resource type for environments.
	PolicyResourceTypeEnvironment PolicyResourceType = "ENVIRONMENT"
	// PolicyResourceTypeProject is the resource type for projects.
	PolicyResourceTypeProject PolicyResourceType = "PROJECT"

	// ReservedTagReviewConfig is the tag for review config.
	ReservedTagReviewConfig string = "bb.tag.review_config"
)
