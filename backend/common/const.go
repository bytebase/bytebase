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
