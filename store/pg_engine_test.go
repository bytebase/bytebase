package store

import (
	"context"
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/blang/semver/v4"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/common/log"
	dbdriver "github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/resources/postgres"
)

func TestGetMinorMigrationVersions(t *testing.T) {
	names := []string{latestDataFile, latestSchemaFile, "1.0", "1.1", "1.2", "1.3", "1.4"}

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
		migrateVersions, _, _ := getMinorMigrationVersions(test.names, test.currentVersion)
		require.Equal(t, test.want, migrateVersions)
	}
}

func TestGetMinorVersions(t *testing.T) {
	tests := []struct {
		names []string
		want  []semver.Version
	}{
		{
			names: []string{fmt.Sprintf("migration/dev/%s", latestDataFile), fmt.Sprintf("migration/dev/%s", latestSchemaFile), "migration/dev/1.1", "migration/dev/1.0"},
			want:  []semver.Version{semver.MustParse("1.0.0"), semver.MustParse("1.1.0")},
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
		names          []string
		minorVersion   semver.Version
		currentVersion semver.Version
		want           []patchVersion
		errPart        string
	}{
		{
			names:          []string{"0000__hello.sql", "0001__world.sql"},
			minorVersion:   semver.MustParse("1.1.0"),
			currentVersion: semver.MustParse("1.2.3"),
			want:           nil,
			errPart:        "",
		},
		{
			names:          []string{"0000__hello.sql", "0001__world.sql"},
			minorVersion:   semver.MustParse("1.1.0"),
			currentVersion: semver.MustParse("1.0.0"),
			want:           []patchVersion{{semver.MustParse("1.1.0"), "0000__hello.sql"}, {semver.MustParse("1.1.1"), "0001__world.sql"}},
			errPart:        "",
		},
		{
			names:          []string{"0000__hello.sql", "0001__world.sql"},
			minorVersion:   semver.MustParse("1.1.0"),
			currentVersion: semver.MustParse("1.1.0"),
			want:           []patchVersion{{semver.MustParse("1.1.1"), "0001__world.sql"}},
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
			errPart:        "should include '__'",
		},
		{
			names:          []string{"00a0__hello.sql"},
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

var (
	pgUser        = "test"
	pgPort        = 6000
	serverVersion = "server-version"
)

func TestMigrationCompatibility(t *testing.T) {
	log.SetLevel(zap.DebugLevel)
	pgDir := t.TempDir()
	pgInstance, err := postgres.Install(path.Join(pgDir, "resource"), path.Join(pgDir, "data"), pgUser)
	require.NoError(t, err)
	err = postgres.Start(pgPort, pgInstance.BaseDir, pgInstance.DataDir, os.Stderr, os.Stderr)
	require.NoError(t, err)
	pgInstance.Port = pgPort

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
		dbdriver.DriverConfig{PgInstanceDir: pgInstance.BaseDir},
		connCfg,
		dbdriver.ConnectionContext{},
	)
	require.NoError(t, err)
	defer d.Close(ctx)

	err = d.SetupMigrationIfNeeded(ctx)
	require.NoError(t, err)

	releaseVersion, err := getProdCutoffVersion()
	require.NoError(t, err)

	// Create a database with release latest schema.
	databaseName := "hidb"
	// Passing curVers = nil will create the database.
	err = migrate(ctx, d, nil, common.ReleaseModeProd, false /*strictDb*/, serverVersion, databaseName)
	require.NoError(t, err)
	// Check migration history.
	histories, err := d.FindMigrationHistoryList(ctx, &dbdriver.MigrationHistoryFind{
		Database: &databaseName,
	})
	require.NoError(t, err)
	require.Len(t, histories, 1)
	require.Equal(t, histories[0].Version, releaseVersion.String())

	// Check no migration after passing current version as the release cutoff version.
	err = migrate(ctx, d, &releaseVersion, common.ReleaseModeProd, false /*strictDb*/, serverVersion, databaseName)
	require.NoError(t, err)
	// Check migration history.
	histories, err = d.FindMigrationHistoryList(ctx, &dbdriver.MigrationHistoryFind{
		Database: &databaseName,
	})
	require.NoError(t, err)
	require.Len(t, histories, 1)

	// Apply migration to dev latest if there are patches.
	err = migrate(ctx, d, &releaseVersion, common.ReleaseModeDev, false /*strictDb*/, serverVersion, databaseName)
	require.NoError(t, err)

	// Check migration history.
	devMigrations, err := getDevMigrations()
	require.NoError(t, err)
	histories, err = d.FindMigrationHistoryList(ctx, &dbdriver.MigrationHistoryFind{
		Database: &databaseName,
	})
	require.NoError(t, err)
	// The extra one is for the initial schema setup.
	require.Len(t, histories, len(devMigrations)+1)

	err = postgres.Stop(pgInstance.BaseDir, pgInstance.DataDir, os.Stdout, os.Stderr)
	require.NoError(t, err)
}

func TestGetCutoffVersion(t *testing.T) {
	releaseVersion, err := getProdCutoffVersion()
	require.NoError(t, err)
	require.Equal(t, semver.MustParse("1.3.4"), releaseVersion)
}
