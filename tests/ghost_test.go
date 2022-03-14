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
	execute        bool
}

const localhost = "127.0.0.1"

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
	migrationContext.Noop = !config.execute
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
	const port = 13306
	basedir := t.TempDir()
	datadir := filepath.Join(basedir, "data")
	if err := os.Mkdir(datadir, 0755); err != nil {
		t.Fatal(err)
	}
	mysql, err := mysql.Install(basedir, datadir, "root")
	if err != nil {
		t.Fatalf("failed to start MySQL: %v", err)
	}
	if err := mysql.Start(port, os.Stdout, os.Stderr, 60); err != nil {
		t.Fatalf("failed to start MySQL: %v", err)
	}

	defer func() {
		err := mysql.Stop(os.Stdout, os.Stderr)
		if err != nil {
			t.Fatalf("failed to stop MySQL: %v", err)
		}
	}()

	db, err := sql.Open("mysql", fmt.Sprintf("root@tcp(localhost:%d)/mysql", mysql.Port()))
	if err != nil {
		t.Fatalf("failed to open MySQL: %v", err)
	}
	defer db.Close()

	_, err = db.Exec("CREATE DATABASE gh_ost_test_db")
	if err != nil {
		t.Fatalf("failed to CREATE DATABASE gh_ost_test_db")
	}

	_, err = db.Exec("USE gh_ost_test_db")
	if err != nil {
		t.Fatalf("failed to USE gh_ost_test_db")
	}

	_, err = db.Exec("CREATE TABLE tbl (id int primary key, data int)")
	if err != nil {
		t.Fatalf("failed to CREATE TABLE tbl (id int primary key, data int)")
	}

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("failed to start a transaction: %v", err)
	}
	defer tx.Rollback()
	for i := 1; i <= 1000; i++ {
		_, err := tx.Exec(fmt.Sprintf("INSERT INTO tbl values (%v, %v)", i, i))
		if err != nil {
			t.Fatalf("failed to insert: %v", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	migrationContext, err := newMigrationContext(config{
		host:           localhost,
		port:           port,
		user:           "root",
		database:       "gh_ost_test_db",
		table:          "tbl",
		alterStatement: "alter table tbl add name varchar(64)",
	})

	if err != nil {
		t.Fatalf("failed to setup migrationContext: %v", err)
	}

	migrator := logic.NewMigrator(migrationContext)
	if err := migrator.Migrate(); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	rows, err := db.Query(`SELECT * FROM tbl`)
	if err != nil {
		t.Fatalf("failed to SELECT * FROM tbl: %v", err)
	}
	defer rows.Close()

	var (
		id   int
		data int
	)

	for rows.Next() {
		if err := rows.Scan(&id, &data); err != nil {
			t.Fatalf("failed to scan: %v", err)
		}
		if id != data {
			t.Errorf("data mismatch, expect id: %v, data: %v, get id: %v, data: %v", id, id, id, data)
		}
	}
}
