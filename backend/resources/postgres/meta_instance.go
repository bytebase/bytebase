package postgres

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
)

// StartMetadataInstance starts the metadata instance.
func StartMetadataInstance(dataDir, resourceDir, pgBinDir, pgUser, demoName string, port int, mode common.ReleaseMode) (func(), error) {
	pgDataDir := getPostgresDataDir(dataDir, demoName)
	if err := upgradePostgres(dataDir, resourceDir, pgBinDir, pgDataDir, pgUser, port); err != nil {
		return nil, err
	}
	slog.Info("-----Embedded Postgres BEGIN-----")
	slog.Info(fmt.Sprintf("Start embedded Postgres datastorePort=%d pgDataDir=%s", port, pgDataDir))
	if err := initDB(pgBinDir, pgDataDir, pgUser); err != nil {
		return nil, err
	}
	serverLog := mode == common.ReleaseModeDev
	if err := start(port, pgBinDir, pgDataDir, serverLog); err != nil {
		return nil, err
	}
	slog.Info("-----Embedded Postgres END-----")

	return func() {
		if err := stop(pgBinDir, pgDataDir); err != nil {
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

func upgradePostgres(dataDir, resourceDir, pgBinDir, pgDataDir, pgUser string, port int) error {
	version, err := getVersion(pgDataDir)
	if err != nil {
		return err
	}
	if version == "" {
		return nil
	}
	if version == currentVersion {
		return nil
	}

	previousBinDir, err := getPreviousVersionBinDir(resourceDir, version)
	if err != nil {
		return err
	}

	// Dump metadata SQL.
	slog.Info("started metadata SQL dump")
	if err := start(port, previousBinDir, pgDataDir, true /* serverLog */); err != nil {
		return err
	}
	dumpFilePath := path.Join(dataDir, "meta.sql")
	cmd := exec.Command(filepath.Join(previousBinDir, "pg_dump"), "-U", pgUser, "-h", common.GetPostgresSocketDir(), "-p", fmt.Sprintf("%d", port), pgUser)
	dumpFile, err := os.Create(dumpFilePath)
	if err != nil {
		return err
	}
	defer dumpFile.Close()
	cmd.Stdout = dumpFile
	if err := cmd.Start(); err != nil {
		return err
	}
	if err := cmd.Wait(); err != nil {
		return err
	}
	if err := stop(previousBinDir, pgDataDir); err != nil {
		return err
	}
	slog.Info("finished metadata SQL dump")
	if err := os.Rename(pgDataDir, path.Join(dataDir, fmt.Sprintf("pgdata-%d", time.Now().Unix()))); err != nil {
		return err
	}
	slog.Info("renamed old pgdata directory")

	if err := initDB(pgBinDir, pgDataDir, pgUser); err != nil {
		return err
	}
	if err := start(port, pgBinDir, pgDataDir, true /* serverLog */); err != nil {
		return err
	}
	createDatabaseSQL := fmt.Sprintf("CREATE DATABASE %s;", pgUser)
	cmd = exec.Command(filepath.Join(pgBinDir, "psql"), "-U", pgUser, "-h", common.GetPostgresSocketDir(), "-p", fmt.Sprintf("%d", port), "postgres", "-c", createDatabaseSQL)
	if err := cmd.Run(); err != nil {
		return err
	}
	cmd = exec.Command(filepath.Join(pgBinDir, "psql"), "-U", pgUser, "-h", common.GetPostgresSocketDir(), "-p", fmt.Sprintf("%d", port), pgUser, "-f", dumpFilePath)
	if err := cmd.Run(); err != nil {
		return err
	}
	if err := stop(pgBinDir, pgDataDir); err != nil {
		return err
	}

	if err := os.Remove(dumpFilePath); err != nil {
		return err
	}
	slog.Info("deleted metadata SQL dump")

	slog.Info("finished pg version upgrade")
	return nil
}

func getPreviousVersionBinDir(resourceDir, version string) (string, error) {
	prefix, err := getTarName()
	if err != nil {
		return "", err
	}

	switch version {
	case "14":
		return path.Join(resourceDir, prefix, "bin"), nil
	case "16":
		return path.Join(resourceDir, fmt.Sprintf("%s%s", prefix, version), "bin"), nil
	}
	return "", errors.Errorf("invalid postgres version %q", version)
}
