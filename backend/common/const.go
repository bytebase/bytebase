//nolint:revive
package common

const (
	WorkspaceAdmin  = "workspaceAdmin"
	WorkspaceMember = "workspaceMember"
	ProjectOwner    = "projectOwner"
)

const (
	// SystemBotID is the ID of the system robot.
	SystemBotID = 1

	// AllUsers is the email of the pseudo allUsers account.
	AllUsers = "allUsers"

	// PrincipalIDForFirstUser is the principal id for the first user in workspace.
	PrincipalIDForFirstUser = 101

	// ServiceAccountAccessKeyPrefix is the prefix for service account access key.
	ServiceAccountAccessKeyPrefix = "bbs_"
)

// DefaultInstanceMaximumConnections is the maximum number of connections outstanding per instance by default.
const DefaultInstanceMaximumConnections = 10

const (
	// ReservedTagReviewConfig is the tag for review config.
	ReservedTagReviewConfig string = "bb.tag.review_config"
)

const (
	// DefaultProjectID is the resource ID for the default project.
	DefaultProjectID = "default"
	// DefaultTestEnvironmentID is the initial resource ID for the test environment.
	// This can be mutated by the user. But for now this is only used by onboarding flow to create
	// a test instance after first signup, so it's safe to refer it.
	DefaultTestEnvironmentID = "test"
	// DefaultProdEnvironmentID is the initial resource ID for the prod environment.
	// This can be mutated by the user. But for now this is only used by onboarding flow to create
	// a prod instance after first signup, so it's safe to refer it.
	DefaultProdEnvironmentID = "prod"
)
