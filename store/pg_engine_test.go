package store

import (
	"context"
	"fmt"
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

	versions, err := getMigrationFileVersions()
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
		ver, err := migrate(ctx, d, nil, initialVersion, serverVersion, initialDatabaseName, l)
		require.NoError(t, err)
		require.Equal(t, initialVersion, ver)

		currentVersion := initialVersion
		for j := i + 1; j < len(versions); j++ {
			version := versions[j]
			ver, err = migrate(ctx, d, &currentVersion, version, serverVersion, initialDatabaseName, l)
			require.NoError(t, err)
			require.Equal(t, version, ver)
			currentVersion = version
		}
	}
}

func getDatabaseName(version semver.Version) string {
	return fmt.Sprintf("db%s", strings.ReplaceAll(version.String(), ".", "v"))
}
