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

	// Register postgres driver
	_ "github.com/bytebase/bytebase/plugin/db/pg"
)

var (
	pgUser         = "test"
	pgPort         = 6000
	databaseName   = "migrate"
	initialVersion = semver.MustParse("1.0.0")
	serverVersion  = "server-version"
	l              = zap.NewNop()
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

	// Setup initial schema of 1.0.0.
	err = migrate(ctx, d, nil, initialVersion, serverVersion, databaseName, l)
	require.NoError(t, err)
	currentVersion := initialVersion

	versions, err := getMigrationFileVersions()
	require.NoError(t, err)

	for _, version := range versions {
		if version.EQ(initialVersion) {
			continue
		}
		// Make sure we can migrate.
		err = migrate(ctx, d, &currentVersion, initialVersion, serverVersion, databaseName, l)
		require.NoError(t, err)
		currentVersion = version

		// Make sure we can setup the latest schema.
		versionDatabaseName := fmt.Sprintf("db%s", strings.ReplaceAll(version.String(), ".", "v"))
		err = migrate(ctx, d, nil, version, serverVersion, versionDatabaseName, l)
		require.NoError(t, err)

		// TODO(d): make sure two schemas are the same.
	}
}
