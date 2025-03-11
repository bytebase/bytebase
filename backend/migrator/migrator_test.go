package migrator

import (
	"fmt"
	"testing"

	"github.com/blang/semver/v4"
	"github.com/stretchr/testify/require"

	_ "github.com/bytebase/bytebase/backend/plugin/db/pg"
)

func TestGetMinorMigrationVersions(t *testing.T) {
	names := []string{latestSchemaFile, "1.0", "1.1", "1.2", "1.3", "1.4"}

	tests := []struct {
		names          []string
		currentVersion semver.Version
		want           []semver.Version
	}{
		{
			names:          names,
			currentVersion: semver.MustParse("1.0.0"),
			want:           []semver.Version{semver.MustParse("1.0.0"), semver.MustParse("1.1.0"), semver.MustParse("1.2.0"), semver.MustParse("1.3.0"), semver.MustParse("1.4.0")},
		},
		{
			names:          names,
			currentVersion: semver.MustParse("1.3.0"),
			want:           []semver.Version{semver.MustParse("1.3.0"), semver.MustParse("1.4.0")},
		},
		{
			names:          names,
			currentVersion: semver.MustParse("1.0.3"),
			want:           []semver.Version{semver.MustParse("1.0.0"), semver.MustParse("1.1.0"), semver.MustParse("1.2.0"), semver.MustParse("1.3.0"), semver.MustParse("1.4.0")},
		},
		{
			names:          names,
			currentVersion: semver.MustParse("1.2.2"),
			want:           []semver.Version{semver.MustParse("1.2.0"), semver.MustParse("1.3.0"), semver.MustParse("1.4.0")},
		},
	}

	for _, test := range tests {
		migrateVersions, _ := getMinorMigrationVersions(test.names, test.currentVersion)
		require.Equal(t, test.want, migrateVersions)
	}
}

func TestGetMinorVersions(t *testing.T) {
	tests := []struct {
		names []string
		want  []semver.Version
	}{
		{
			names: []string{fmt.Sprintf("migration/dev/%s", latestSchemaFile), "migration/release/1.1", "migration/release/1.0"},
			want:  []semver.Version{semver.MustParse("1.0.0"), semver.MustParse("1.1.0")},
		},
		{
			names: []string{fmt.Sprintf("migration/dev/%s", latestSchemaFile)},
			want:  nil,
		},
	}

	for _, test := range tests {
		got, _ := getMinorVersions(test.names)
		require.Equal(t, test.want, got)
	}
}

func TestGetPatchVersions(t *testing.T) {
	tests := []struct {
		names          []string
		minorVersion   semver.Version
		currentVersion semver.Version
		want           []patchVersion
		errPart        string
	}{
		{
			names:          []string{"0000##hello.sql", "0001##world.sql"},
			minorVersion:   semver.MustParse("1.1.0"),
			currentVersion: semver.MustParse("1.2.3"),
			want:           nil,
			errPart:        "",
		},
		{
			names:          []string{"0000##hello.sql", "0001##world.sql"},
			minorVersion:   semver.MustParse("1.1.0"),
			currentVersion: semver.MustParse("1.0.0"),
			want:           []patchVersion{{semver.MustParse("1.1.0"), "0000##hello.sql"}, {semver.MustParse("1.1.1"), "0001##world.sql"}},
			errPart:        "",
		},
		{
			names:          []string{"0000##hello.sql", "0001##world.sql"},
			minorVersion:   semver.MustParse("1.1.0"),
			currentVersion: semver.MustParse("1.1.0"),
			want:           []patchVersion{{semver.MustParse("1.1.1"), "0001##world.sql"}},
			errPart:        "",
		},
		{
			names:          []string{},
			minorVersion:   semver.MustParse("1.1.0"),
			currentVersion: semver.MustParse("1.0.0"),
			want:           nil,
			errPart:        "",
		},
		{
			names:          []string{"0000_hello.sql"},
			minorVersion:   semver.MustParse("1.1.0"),
			currentVersion: semver.MustParse("1.0.0"),
			want:           nil,
			errPart:        "should include '##'",
		},
		{
			names:          []string{"00a0##hello.sql"},
			minorVersion:   semver.MustParse("1.1.0"),
			currentVersion: semver.MustParse("1.0.0"),
			want:           nil,
			errPart:        "should be four digits integer",
		},
	}

	for _, test := range tests {
		got, err := getPatchVersions(test.minorVersion, test.currentVersion, test.names)
		if test.errPart == "" {
			require.NoError(t, err)
		} else {
			require.Contains(t, err.Error(), test.errPart)
		}
		require.Equal(t, test.want, got)
	}
}

func TestGetCutoffVersion(t *testing.T) {
	releaseVersion, err := getProdCutoffVersion()
	require.NoError(t, err)
	require.Equal(t, semver.MustParse("3.5.14"), releaseVersion)
}
