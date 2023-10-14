package postgres

import (
	"fmt"
	"log/slog"
	"path"

	"github.com/bytebase/bytebase/backend/common"
)

// StartMetadataInstance starts the metadata instance.
func StartMetadataInstance(pgBinDir, dataDir, pgUser, demoName string, port int, mode common.ReleaseMode) (func(), error) {
	pgDataDir := getPostgresDataDir(dataDir, demoName)
	slog.Info("-----Embedded Postgres BEGIN-----")
	slog.Info(fmt.Sprintf("Start embedded Postgres datastorePort=%d pgDataDir=%s", port, pgDataDir))
	if err := InitDB(pgBinDir, pgDataDir, pgUser); err != nil {
		return nil, err
	}
	serverLog := mode == common.ReleaseModeDev
	if err := Start(port, pgBinDir, pgDataDir, serverLog); err != nil {
		return nil, err
	}
	slog.Info("-----Embedded Postgres END-----")

	return func() {
		if err := Stop(pgBinDir, pgDataDir); err != nil {
			panic(err)
		}
	}, nil
}

// getPostgresDataDir returns the postgres data directory of Bytebase.
func getPostgresDataDir(dataDir string, demoName string) string {
	// If demo is specified, we will use demo specific directory to store the demo data. Because
	// we reset the demo data when starting Bytebase and this can prevent accidentally removing the
	// production data.
	if demoName != "" {
		return path.Join(dataDir, "pgdata-demo", demoName)
	}
	return path.Join(dataDir, "pgdata")
}
