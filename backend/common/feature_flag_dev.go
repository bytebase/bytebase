//go:build !release

package common

// FeatureFlag in dev build always returns true.
// Because dev schema always has the latest change and is compatible with the new feature code.
func FeatureFlag(_ FeatureFlagType) bool {
	return true
}
