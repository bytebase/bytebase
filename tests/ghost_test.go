package tests

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/bytebase/bytebase/resources/mysql"
	"github.com/github/gh-ost/go/base"
	"github.com/github/gh-ost/go/logic"
	ghostsql "github.com/github/gh-ost/go/sql"
	_ "github.com/go-sql-driver/mysql"
)

type config struct {
	host           string
	port           int
	user           string
	password       string
	database       string
	table          string
	alterStatement string
	noop           bool
}

const (
	allowedRunningOnMaster              = true
	concurrentCountTableRows            = true
	hooksStatusIntervalSec              = 60
	replicaServerID                     = 99999
	heartbeatIntervalMilliseconds       = 100
	niceRatio                           = 0
	chunkSize                           = 1000
	dmlBatchSize                        = 10
	maxLagMillisecondsThrottleThreshold = 1500
	defaultNumRetries                   = 60
	cutoverLockTimoutSeconds            = 3
	exponentialBackoffMaxInterval       = 64
)

func newMigrationContext(config config) (*base.MigrationContext, error) {
	migrationContext := base.NewMigrationContext()
	migrationContext.InspectorConnectionConfig.Key.Hostname = config.host
	migrationContext.InspectorConnectionConfig.Key.Port = config.port
	migrationContext.CliUser = config.user
	migrationContext.CliPassword = config.password
	migrationContext.DatabaseName = config.database
	migrationContext.OriginalTableName = config.table
	migrationContext.AlterStatement = config.alterStatement
	migrationContext.Noop = config.noop
	// set defaults
	migrationContext.AllowedRunningOnMaster = allowedRunningOnMaster
	migrationContext.ConcurrentCountTableRows = concurrentCountTableRows
	migrationContext.HooksStatusIntervalSec = hooksStatusIntervalSec
	migrationContext.ReplicaServerId = replicaServerID
	migrationContext.CutOverType = base.CutOverAtomic

	if migrationContext.AlterStatement == "" {
		return nil, fmt.Errorf("alterStatement must be provided and must not be empty")
	}
	parser := ghostsql.NewParserFromAlterStatement(migrationContext.AlterStatement)
	migrationContext.AlterStatementOptions = parser.GetAlterStatementOptions()

	if migrationContext.DatabaseName == "" {
		if parser.HasExplicitSchema() {
			migrationContext.DatabaseName = parser.GetExplicitSchema()
		} else {
			return nil, fmt.Errorf("database must be provided and database name must not be empty, or alterStatement must specify database name")
		}
	}
	if migrationContext.OriginalTableName == "" {
		if parser.HasExplicitTable() {
			migrationContext.OriginalTableName = parser.GetExplicitTable()
		} else {
			return nil, fmt.Errorf("table must be provided and table name must not be empty, or alterStatement must specify table name")
		}
	}
	migrationContext.ServeSocketFile = fmt.Sprintf("/tmp/gh-ost.%s.%s.sock", migrationContext.DatabaseName, migrationContext.OriginalTableName)
	migrationContext.SetHeartbeatIntervalMilliseconds(heartbeatIntervalMilliseconds)
	migrationContext.SetNiceRatio(niceRatio)
	migrationContext.SetChunkSize(chunkSize)
	migrationContext.SetDMLBatchSize(dmlBatchSize)
	migrationContext.SetMaxLagMillisecondsThrottleThreshold(maxLagMillisecondsThrottleThreshold)
	migrationContext.SetDefaultNumRetries(defaultNumRetries)
	migrationContext.ApplyCredentials()
	if err := migrationContext.SetCutOverLockTimeoutSeconds(cutoverLockTimoutSeconds); err != nil {
		return nil, err
	}
	if err := migrationContext.SetExponentialBackoffMaxInterval(exponentialBackoffMaxInterval); err != nil {
		return nil, err
	}
	return migrationContext, nil
}

func TestGhostSimpleNoop(t *testing.T) {
	const (
		localhost = "127.0.0.1"
		port      = 13306
		user      = "root"
		database  = "gh_ost_test_db"
		table     = "tbl"
	)
	err := func() error {
		basedir := t.TempDir()
		datadir := filepath.Join(basedir, "data")
		if err := os.Mkdir(datadir, 0755); err != nil {
			return err
		}
		instance, err := mysql.Install(basedir, datadir, "root")
		if err != nil {
			return fmt.Errorf("failed to install MySQL: %v", err)
		}
		if err := instance.Start(port, os.Stdout, os.Stderr, 60 /* waitSec */); err != nil {
			return fmt.Errorf("failed to start MySQL: %v", err)
		}

		defer func() {
			_ = instance.Stop(os.Stdout, os.Stderr)
		}()

		db, err := sql.Open("mysql", fmt.Sprintf("root@tcp(localhost:%d)/mysql", port))
		if err != nil {
			return fmt.Errorf("failed to open MySQL: %v", err)
		}
		defer db.Close()

		if _, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", database)); err != nil {
			return err
		}

		if _, err = db.Exec(fmt.Sprintf("USE %s", database)); err != nil {
			return err
		}

		if _, err = db.Exec(fmt.Sprintf("CREATE TABLE %s (id INT PRIMARY KEY, data INT)", table)); err != nil {
			return err
		}

		tx, err := db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		const n = 100
		for i := 1; i <= n; i++ {
			if _, err := tx.Exec(fmt.Sprintf("INSERT INTO %s VALUES (%v, %v)", table, i, i)); err != nil {
				return err
			}
		}

		if err := tx.Commit(); err != nil {
			return err
		}

		migrationContext, err := newMigrationContext(config{
			host:           localhost,
			port:           port,
			user:           user,
			database:       database,
			table:          table,
			alterStatement: "ALTER TABLE tbl ADD name VARCHAR(64)",
			noop:           true,
		})
		if err != nil {
			return err
		}

		migrator := logic.NewMigrator(migrationContext)
		if err := migrator.Migrate(); err != nil {
			return err
		}

		return nil
	}()

	if err != nil {
		t.Error(err)
	}
}
