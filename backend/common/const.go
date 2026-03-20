//nolint:revive
package common

const (
	// AllUsers is the email of the pseudo allUsers account.
	AllUsers = "allUsers"

	// ServiceAccountAccessKeyPrefix is the prefix for service account access key.
	ServiceAccountAccessKeyPrefix = "bbs_"

	// WorkloadIdentitySuffix is the email suffix for workload identities.
	WorkloadIdentitySuffix = "workload.bytebase.com"

	// ServiceAccountSuffix is the email suffix for service accounts.
	ServiceAccountSuffix = "service.bytebase.com"
)

const (
	// ReservedTagReviewConfig is the tag for review config.
	ReservedTagReviewConfig string = "bb.tag.review_config"
)

const (
	// defaultProjectPrefix is the prefix for the default project resource ID.
	// Each workspace has its own default project: "default-{workspaceID}".
	defaultProjectPrefix = "default-"
)

// DefaultProjectID returns the default project resource ID for a workspace.
func DefaultProjectID(workspaceID string) string {
	return defaultProjectPrefix + workspaceID
}

// IsDefaultProject returns whether a project resource ID is the default project for the given workspace.
func IsDefaultProject(workspaceID, projectID string) bool {
	return projectID == DefaultProjectID(workspaceID)
}

const (
	// DefaultTestEnvironmentID is the initial resource ID for the test environment.
	// This can be mutated by the user. But for now this is only used by onboarding flow to create
	// a test instance after first signup, so it's safe to refer it.
	DefaultTestEnvironmentID = "test"
	// DefaultProdEnvironmentID is the initial resource ID for the prod environment.
	// This can be mutated by the user. But for now this is only used by onboarding flow to create
	// a prod instance after first signup, so it's safe to refer it.
	DefaultProdEnvironmentID = "prod"
)
