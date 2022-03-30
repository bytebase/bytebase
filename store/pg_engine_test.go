package store

import (
	"testing"

	"github.com/blang/semver/v4"
	"github.com/stretchr/testify/require"
)

func TestGetMigrationVersions(t *testing.T) {
	versions := []semver.Version{semver.MustParse("1.0.0"), semver.MustParse("1.1.0"), semver.MustParse("1.1.1"), semver.MustParse("1.2.0"), semver.MustParse("1.3.0")}

	tests := []struct {
		versions                []semver.Version
		releaseCutSchemaVersion semver.Version
		currentVersion          semver.Version
		want                    []semver.Version
	}{
		{
			versions,
			semver.MustParse("1.0.0"),
			semver.MustParse("1.0.0"),
			nil,
		},
		{
			versions,
			semver.MustParse("1.1.1"),
			semver.MustParse("1.1.1"),
			nil,
		},
		{
			versions,
			semver.MustParse("1.1.1"),
			semver.MustParse("1.0.0"),
			[]semver.Version{semver.MustParse("1.1.0"), semver.MustParse("1.1.1")},
		},
		{
			versions,
			semver.MustParse("1.3.0"),
			semver.MustParse("1.0.0"),
			[]semver.Version{semver.MustParse("1.1.0"), semver.MustParse("1.1.1"), semver.MustParse("1.2.0"), semver.MustParse("1.3.0")},
		},
	}

	for _, test := range tests {
		migrateVersions, _ := getMigrationVersions(test.versions, test.releaseCutSchemaVersion, test.currentVersion)
		require.Equal(t, test.want, migrateVersions)
	}
}
