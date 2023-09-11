//go:build release

package common

// FeatureFlag in release build always returns false.
// Because we would remove the FeatureFlag check entirely after the schema change for the feature
// is rolled out to prod.
func FeatureFlag(_ FeatureFlagType) bool {
	return false
}
