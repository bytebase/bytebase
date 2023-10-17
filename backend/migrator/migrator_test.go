package migrator

import (
	"context"
	"fmt"
	"path"
	"testing"

	"github.com/blang/semver/v4"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	dbdriver "github.com/bytebase/bytebase/backend/plugin/db"
	_ "github.com/bytebase/bytebase/backend/plugin/db/pg"
	"github.com/bytebase/bytebase/backend/resources/postgres"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
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

var (
	pgPort        = 6000
	serverVersion = "server-version"
)

func TestMigrationCompatibility(t *testing.T) {
	pgDir := t.TempDir()

	pgBinDir, err := postgres.Install(path.Join(pgDir, "resource"))
	require.NoError(t, err)

	stopInstance := postgres.SetupTestInstance(pgBinDir, t.TempDir(), pgPort)
	defer stopInstance()

	ctx := context.Background()
	connCfg := dbdriver.ConnectionConfig{
		Username: postgres.TestPgUser,
		Password: "",
		Host:     common.GetPostgresSocketDir(),
		Port:     fmt.Sprintf("%d", pgPort),
	}
	defaultDriver, err := dbdriver.Open(
		ctx,
		storepb.Engine_POSTGRES,
		dbdriver.DriverConfig{DbBinDir: pgBinDir},
		connCfg,
		dbdriver.ConnectionContext{},
	)
	require.NoError(t, err)
	defer defaultDriver.Close(ctx)
	// Create a database with release latest schema.
	databaseName := "hidb"
	_, err = defaultDriver.Execute(ctx, fmt.Sprintf("CREATE DATABASE %s", databaseName), true, dbdriver.ExecuteOptions{})
	require.NoError(t, err)

	metadataConnConfig := connCfg
	metadataConnConfig.Database = databaseName
	metadataConnConfig.StrictUseDb = true
	metadataDriver, err := dbdriver.Open(
		ctx,
		storepb.Engine_POSTGRES,
		dbdriver.DriverConfig{DbBinDir: pgBinDir},
		metadataConnConfig,
		dbdriver.ConnectionContext{},
	)
	require.NoError(t, err)

	db := store.NewDB(metadataConnConfig, pgBinDir, "", false, "", common.ReleaseModeDev)
	err = db.Open(ctx)
	require.NoError(t, err)
	storeInstance := store.New(db)

	releaseVersion, err := getProdCutoffVersion()
	require.NoError(t, err)

	// Create initial schema.
	err = initializeSchema(ctx, storeInstance, metadataDriver, releaseVersion, serverVersion)
	require.NoError(t, err)
	// Check migration history.
	histories, err := storeInstance.ListInstanceChangeHistoryForMigrator(ctx, &store.FindInstanceChangeHistoryMessage{
		InstanceID: nil,
	})
	require.NoError(t, err)
	require.Len(t, histories, 1)
	require.Equal(t, histories[0].Version.Version, releaseVersion.String())

	// Check no migration after passing current version as the release cutoff version.
	err = migrate(ctx, storeInstance, metadataDriver, releaseVersion, releaseVersion, common.ReleaseModeProd, serverVersion, databaseName)
	require.NoError(t, err)
	// Check migration history.
	histories, err = storeInstance.ListInstanceChangeHistoryForMigrator(ctx, &store.FindInstanceChangeHistoryMessage{
		InstanceID: nil,
	})
	require.NoError(t, err)
	require.Len(t, histories, 1)

	// Apply migration to dev latest if there are patches.
	err = migrate(ctx, storeInstance, metadataDriver, releaseVersion, releaseVersion, common.ReleaseModeDev, serverVersion, databaseName)
	require.NoError(t, err)

	// Check migration history.
	devMigrations, err := getDevMigrations()
	require.NoError(t, err)
	histories, err = storeInstance.ListInstanceChangeHistoryForMigrator(ctx, &store.FindInstanceChangeHistoryMessage{
		InstanceID: nil,
	})
	require.NoError(t, err)
	// The extra one is for the initial schema setup.
	require.Len(t, histories, len(devMigrations)+1)
}

func TestGetCutoffVersion(t *testing.T) {
	releaseVersion, err := getProdCutoffVersion()
	require.NoError(t, err)
	require.Equal(t, semver.MustParse("2.10.3"), releaseVersion)
}
