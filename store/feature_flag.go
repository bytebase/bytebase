package store

// Have a single place to track all ongoing feature development that requires code to work with both
// old and new schecma. We need to gate the feature to make the code be compatible with both dev
// schema (having the schema change) and prod schema (not having the schema change yet).
//
// Step 1: Declare a feature
//
// Step 2: Gate the feature when the relevant schema hasn't been applied to prod yet
//
// if common.FeatureFlag(common.FeatureFlagNoop) {
//    <<New feature code work with the new schema>>
// } else {
//	  <<Old code work with the old schema>>
// }
//
// Step 3: Remove the feature and gate

// FeatureFlagType is the feature flag type.
type FeatureFlagType string

const (
	// FeatureFlagNoop is a noop feature flag for demonstration purpose.
	FeatureFlagNoop FeatureFlagType = "bb.feature-flag.noop"
)
