package store

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/blang/semver/v4"
	"github.com/bytebase/bytebase/common"
	dbdriver "github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/resources/postgres"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestGetMinorMigrationVersions(t *testing.T) {
	names := []string{latestDataFile, latestSchemaFile, "1.0.0", "1.1.0", "1.2.0", "1.3.0", "1.4.0"}

	tests := []struct {
		names                   []string
		releaseCutSchemaVersion semver.Version
		currentVersion          semver.Version
		want                    []semver.Version
	}{
		{
			names:                   names,
			releaseCutSchemaVersion: semver.MustParse("1.0.0"),
			currentVersion:          semver.MustParse("1.0.0"),
			want:                    []semver.Version{semver.MustParse("1.0.0")},
		},
		{
			names:                   names,
			releaseCutSchemaVersion: semver.MustParse("1.3.3"),
			currentVersion:          semver.MustParse("1.3.0"),
			want:                    []semver.Version{semver.MustParse("1.3.0")},
		},
		{
			names:                   names,
			releaseCutSchemaVersion: semver.MustParse("1.3.0"),
			currentVersion:          semver.MustParse("1.0.3"),
			want:                    []semver.Version{semver.MustParse("1.0.0"), semver.MustParse("1.1.0"), semver.MustParse("1.2.0"), semver.MustParse("1.3.0")},
		},
		{
			names:                   names,
			releaseCutSchemaVersion: semver.MustParse("1.3.0"),
			currentVersion:          semver.MustParse("1.2.2"),
			want:                    []semver.Version{semver.MustParse("1.2.0"), semver.MustParse("1.3.0")},
		},
	}

	for _, test := range tests {
		migrateVersions, _, _ := getMinorMigrationVersions(test.names, test.releaseCutSchemaVersion, test.currentVersion)
		require.Equal(t, test.want, migrateVersions)
	}
}

func TestGetMiorVersions(t *testing.T) {
	tests := []struct {
		names []string
		want  []semver.Version
	}{
		{
			names: []string{fmt.Sprintf("migration/dev/%s", latestDataFile), fmt.Sprintf("migration/dev/%s", latestSchemaFile), "migration/dev/1.1.2", "migration/dev/1.0.0"},
			want:  []semver.Version{semver.MustParse("1.0.0"), semver.MustParse("1.1.2")},
		},
		{
			names: []string{fmt.Sprintf("migration/release/%s", latestDataFile), fmt.Sprintf("migration/dev/%s", latestSchemaFile)},
			want:  nil,
		},
	}

	for _, test := range tests {
		got, _ := getMiorVersions(test.names)
		require.Equal(t, test.want, got)
	}
}

func TestGetPatchVersions(t *testing.T) {
	tests := []struct {
		names          []string
		minorVersion   semver.Version
		currentVersion semver.Version
		want           []patchVersion
	}{
		{
			names:          []string{"0000__hello.sql", "0001__world.sql"},
			minorVersion:   semver.MustParse("1.1.0"),
			currentVersion: semver.MustParse("1.2.3"),
			want:           nil,
		},
		{
			names:          []string{"0000__hello.sql", "0001__world.sql"},
			minorVersion:   semver.MustParse("1.1.0"),
			currentVersion: semver.MustParse("1.0.0"),
			want:           []patchVersion{{semver.MustParse("1.1.0"), "0000__hello.sql"}, {semver.MustParse("1.1.1"), "0001__world.sql"}},
		},
		{
			names:          []string{},
			minorVersion:   semver.MustParse("1.1.0"),
			currentVersion: semver.MustParse("1.0.0"),
			want:           nil,
		},
	}

	for _, test := range tests {
		got, _ := getPatchVersions(test.minorVersion, test.currentVersion, test.names)
		require.Equal(t, test.want, got)
	}
}

var (
	pgUser        = "test"
	pgPort        = 6000
	serverVersion = "server-version"
	l             = zap.NewNop()
)

func TestMigrationCompatibility(t *testing.T) {
	pgDir := t.TempDir()
	pgInstance, err := postgres.Install(path.Join(pgDir, "resource"), path.Join(pgDir, "data"), pgUser)
	require.NoError(t, err)
	err = pgInstance.Start(pgPort, os.Stdout, os.Stderr)
	require.NoError(t, err)
	defer pgInstance.Stop(os.Stdout, os.Stderr)

	ctx := context.Background()
	connCfg := dbdriver.ConnectionConfig{
		Username: pgUser,
		Password: "",
		Host:     common.GetPostgresSocketDir(),
		Port:     fmt.Sprintf("%d", pgPort),
	}
	d, err := dbdriver.Open(
		ctx,
		dbdriver.Postgres,
		dbdriver.DriverConfig{Logger: l},
		connCfg,
		dbdriver.ConnectionContext{},
	)
	require.NoError(t, err)

	err = d.SetupMigrationIfNeeded(ctx)
	require.NoError(t, err)

	names, err := fs.Glob(migrationFS, fmt.Sprintf("migration/%s/*", common.ReleaseModeRelease))
	require.NoError(t, err)
	versions, err := getMiorVersions(names)
	require.NoError(t, err)

	// For every version, we create a database with the schema of that version and apply migrations till the latest version in the migration directory.
	// For example, we can 3 versions, 1.0.0, 1.1.0, 1.2.0.
	// Create a database with 1.0.0 schema, apply 1.1.0 migration, and apply 1.2.0 migration.
	// Create a database with 1.1.0 schema, and apply 1.2.0 migration.
	// Create a database with 1.2.0 schema. But there is no migration since it's the latest.
	for i := range versions {
		initialVersion := versions[i]
		initialDatabaseName := getDatabaseName(initialVersion)
		// Passing curVers = nil will create the database.
		ver, err := migrate(ctx, d, nil, initialVersion, common.ReleaseModeRelease, serverVersion, initialDatabaseName, l)
		require.NoError(t, err)
		require.Equal(t, initialVersion, ver)

		currentVersion := initialVersion
		for j := i + 1; j < len(versions); j++ {
			version := versions[j]
			ver, err = migrate(ctx, d, &currentVersion, version, common.ReleaseModeRelease, serverVersion, initialDatabaseName, l)
			require.NoError(t, err)
			require.Equal(t, version, ver)
			currentVersion = version
		}
	}
}

func getDatabaseName(version semver.Version) string {
	return fmt.Sprintf("db%s", strings.ReplaceAll(version.String(), ".", "v"))
}
