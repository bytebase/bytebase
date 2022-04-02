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

func TestGetMinorVersions(t *testing.T) {
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
		got, _ := getMinorVersions(test.names)
		require.Equal(t, test.want, got)
	}
}

func TestGetPatchVersions(t *testing.T) {
	tests := []struct {
		names                   []string
		minorVersion            semver.Version
		releaseCutSchemaVersion semver.Version
		currentVersion          semver.Version
		want                    []patchVersion
		errPart                 string
	}{
		{
			names:                   []string{"0000__hello.sql", "0001__world.sql"},
			minorVersion:            semver.MustParse("1.1.0"),
			releaseCutSchemaVersion: semver.MustParse("1.3.0"),
			currentVersion:          semver.MustParse("1.2.3"),
			want:                    nil,
			errPart:                 "",
		},
		{
			names:                   []string{"0000__hello.sql", "0001__world.sql"},
			minorVersion:            semver.MustParse("1.1.0"),
			releaseCutSchemaVersion: semver.MustParse("1.3.0"),
			currentVersion:          semver.MustParse("1.0.0"),
			want:                    []patchVersion{{semver.MustParse("1.1.0"), "0000__hello.sql"}, {semver.MustParse("1.1.1"), "0001__world.sql"}},
			errPart:                 "",
		},
		{
			names:                   []string{"0000__hello.sql", "0001__world.sql"},
			minorVersion:            semver.MustParse("1.1.0"),
			releaseCutSchemaVersion: semver.MustParse("1.1.0"),
			currentVersion:          semver.MustParse("1.0.0"),
			want:                    []patchVersion{{semver.MustParse("1.1.0"), "0000__hello.sql"}},
			errPart:                 "",
		},
		{
			names:                   []string{"0000__hello.sql", "0001__world.sql"},
			minorVersion:            semver.MustParse("1.1.0"),
			releaseCutSchemaVersion: semver.MustParse("1.3.0"),
			currentVersion:          semver.MustParse("1.1.0"),
			want:                    []patchVersion{{semver.MustParse("1.1.1"), "0001__world.sql"}},
			errPart:                 "",
		},
		{
			names:                   []string{},
			minorVersion:            semver.MustParse("1.1.0"),
			releaseCutSchemaVersion: semver.MustParse("1.3.0"),
			currentVersion:          semver.MustParse("1.0.0"),
			want:                    nil,
			errPart:                 "",
		},
		{
			names:                   []string{"0000_hello.sql"},
			minorVersion:            semver.MustParse("1.1.0"),
			releaseCutSchemaVersion: semver.MustParse("1.1.0"),
			currentVersion:          semver.MustParse("1.0.0"),
			want:                    nil,
			errPart:                 "should include '__'",
		},
		{
			names:                   []string{"00a0__hello.sql"},
			minorVersion:            semver.MustParse("1.1.0"),
			releaseCutSchemaVersion: semver.MustParse("1.1.0"),
			currentVersion:          semver.MustParse("1.0.0"),
			want:                    nil,
			errPart:                 "should be four digits integer",
		},
	}

	for _, test := range tests {
		got, err := getPatchVersions(test.minorVersion, test.releaseCutSchemaVersion, test.currentVersion, test.names)
		if test.errPart == "" {
			require.NoError(t, err)
		} else {
			require.Contains(t, err.Error(), test.errPart)
		}
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

	// Create a database with dev latest schema.
	names, err := fs.Glob(migrationFS, fmt.Sprintf("migration/%s/*", common.ReleaseModeDev))
	require.NoError(t, err)
	versions, err := getMinorVersions(names)
	require.NoError(t, err)
	require.Len(t, versions, 1)
	names, err = fs.Glob(migrationFS, fmt.Sprintf("migration/%s/%d.%d/*.sql", common.ReleaseModeDev, versions[0].Major, versions[0].Minor))
	require.NoError(t, err)
	devVersion := versions[0]
	devPatches := len(names)
	if devPatches > 0 {
		devVersion.Patch = uint64(devPatches - 1)
	}
	devDatabaseName := getDatabaseName(devVersion)
	// Passing curVers = nil will create the database.
	ver, err := migrate(ctx, d, nil, devVersion, common.ReleaseModeDev, serverVersion, devDatabaseName, l)
	require.NoError(t, err)
	require.Equal(t, devVersion, ver)

	// Create a database with release latest schema, and apply migration to dev latest.
	names, err = fs.Glob(migrationFS, fmt.Sprintf("migration/%s/*", common.ReleaseModeDev))
	require.NoError(t, err)
	versions, err = getMinorVersions(names)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(versions), 1)
	minorVersion := versions[len(versions)-1]
	names, err = fs.Glob(migrationFS, fmt.Sprintf("migration/%s/%d.%d/*.sql", common.ReleaseModeRelease, minorVersion.Major, minorVersion.Minor))
	require.NoError(t, err)
	prodVersion := minorVersion
	prodVersion.Patch = uint64(len(names))
	prodDatabaseName := getDatabaseName(prodVersion)
	// Passing curVers = nil will create the database.
	ver, err = migrate(ctx, d, nil, prodVersion, common.ReleaseModeRelease, serverVersion, prodDatabaseName, l)
	require.NoError(t, err)
	require.Equal(t, prodVersion, ver)
	// Apply migration to dev latest if there are patches.
	if devPatches > 0 {
		ver, err = migrate(ctx, d, &prodVersion, devVersion, common.ReleaseModeDev, serverVersion, prodDatabaseName, l)
		require.NoError(t, err)
		require.Equal(t, devVersion, ver)
	}
}

func getDatabaseName(version semver.Version) string {
	return fmt.Sprintf("db%s", strings.ReplaceAll(version.String(), ".", "v"))
}
