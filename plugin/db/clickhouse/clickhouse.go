package clickhouse

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"io"
	"strings"
	"time"

	clickhouse "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/db/util"
	"go.uber.org/zap"
)

var (
	systemDatabases = map[string]bool{
		"system":             true,
		"information_schema": true,
		"INFORMATION_SCHEMA": true,
	}

	_ db.Driver = (*Driver)(nil)
)

func init() {
	db.Register(db.ClickHouse, newDriver)
}

// Driver is the ClickHouse driver.
type Driver struct {
	connectionCtx db.ConnectionContext
	dbType        db.Type

	db *sql.DB
}

func newDriver(config db.DriverConfig) db.Driver {
	return &Driver{}
}

// Open opens a ClickHouse driver.
func (driver *Driver) Open(ctx context.Context, dbType db.Type, config db.ConnectionConfig, connCtx db.ConnectionContext) (db.Driver, error) {
	port := config.Port
	if port == "" {
		port = "9000"
	}
	addr := fmt.Sprintf("%s:%s", config.Host, port)
	// Set SSL configuration.
	tlsConfig, err := config.TLSConfig.GetSslConfig()
	if err != nil {
		return nil, fmt.Errorf("sql: tls config error: %v", err)
	}
	// Default user name is "default".
	conn := clickhouse.OpenDB(&clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{
			Database: config.Database,
			Username: config.Username,
			Password: config.Password,
		},
		TLS: tlsConfig,
		Settings: clickhouse.Settings{
			"max_execution_time": 60, // 60 seconds.
		},
		DialTimeout: 10 * time.Second,
	})

	log.Debug("Opening ClickHouse driver",
		zap.String("addr", addr),
		zap.String("environment", connCtx.EnvironmentName),
		zap.String("database", connCtx.InstanceName),
	)

	driver.dbType = dbType
	driver.db = conn
	driver.connectionCtx = connCtx

	return driver, nil
}

// Close closes the driver.
func (driver *Driver) Close(ctx context.Context) error {
	return driver.db.Close()
}

// Ping pings the database.
func (driver *Driver) Ping(ctx context.Context) error {
	return driver.db.PingContext(ctx)
}

// GetDbConnection gets a database connection.
func (driver *Driver) GetDbConnection(ctx context.Context, database string) (*sql.DB, error) {
	return driver.db, nil
}

// GetVersion gets the version.
func (driver *Driver) GetVersion(ctx context.Context) (string, error) {
	query := "SELECT VERSION()"
	versionRow, err := driver.db.QueryContext(ctx, query)
	if err != nil {
		return "", util.FormatErrorWithQuery(err, query)
	}
	defer versionRow.Close()

	var version string
	versionRow.Next()
	if err := versionRow.Scan(&version); err != nil {
		return "", err
	}
	return version, nil
}

// Execute executes a SQL statement.
func (driver *Driver) Execute(ctx context.Context, statement string) error {
	tx, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	f := func(stmt string) error {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return err
		}
		return nil
	}
	sc := bufio.NewScanner(strings.NewReader(statement))
	if err := util.ApplyMultiStatements(sc, f); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return err
}

// Query queries a SQL statement.
func (driver *Driver) Query(ctx context.Context, statement string, limit int) ([]interface{}, error) {
	return util.Query(ctx, driver.db, statement, limit)
}

// Dump and restore
const (
	databaseHeaderFmt = "" +
		"--\n" +
		"-- ClickHouse database structure for `%s`\n" +
		"--\n"
	useDatabaseFmt = "USE `%s`;\n\n"
	tableStmtFmt   = "" +
		"--\n" +
		"-- Table structure for `%s`\n" +
		"--\n" +
		"%s;\n"
	viewStmtFmt = "" +
		"--\n" +
		"-- View structure for `%s`\n" +
		"--\n" +
		"%s;\n"
)

// Dump dumps the database.
func (driver *Driver) Dump(ctx context.Context, database string, out io.Writer, schemaOnly bool) (string, error) {
	txn, err := driver.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return "", err
	}
	defer txn.Rollback()

	if err := dumpTxn(ctx, txn, database, out, schemaOnly); err != nil {
		return "", err
	}

	if err := txn.Commit(); err != nil {
		return "", err
	}

	return "", nil
}

// dumpTxn will dump the input database. schemaOnly isn't supported yet and true by default.
func dumpTxn(ctx context.Context, txn *sql.Tx, database string, out io.Writer, schemaOnly bool) error {
	// Find all dumpable databases
	dbNames, err := getDatabases(txn)
	if err != nil {
		return fmt.Errorf("failed to get databases: %s", err)
	}

	var dumpableDbNames []string
	if database != "" {
		exist := false
		for _, n := range dbNames {
			if n == database {
				exist = true
				break
			}
		}
		if !exist {
			return common.Errorf(common.NotFound, fmt.Errorf("database %s not found", database))
		}
		dumpableDbNames = []string{database}
	} else {
		for _, dbName := range dbNames {
			if systemDatabases[dbName] {
				continue
			}
			dumpableDbNames = append(dumpableDbNames, dbName)
		}
	}

	for _, dbName := range dumpableDbNames {
		// Include "USE DATABASE xxx" if dumping multiple databases.
		if len(dumpableDbNames) > 1 {
			// Database header.
			header := fmt.Sprintf(databaseHeaderFmt, dbName)
			if _, err := io.WriteString(out, header); err != nil {
				return err
			}
			dbStmt, err := getDatabaseStmt(txn, dbName)
			if err != nil {
				return fmt.Errorf("failed to get database %q: %s", dbName, err)
			}
			if _, err := io.WriteString(out, dbStmt); err != nil {
				return err
			}
			// Use database statement.
			useStmt := fmt.Sprintf(useDatabaseFmt, dbName)
			if _, err := io.WriteString(out, useStmt); err != nil {
				return err
			}
		}

		// Table and view statement.
		tables, err := getTables(txn, dbName)
		if err != nil {
			return fmt.Errorf("failed to get tables of database %q: %s", dbName, err)
		}
		for _, tbl := range tables {
			if _, err := io.WriteString(out, fmt.Sprintf("%s\n", tbl.statement)); err != nil {
				return err
			}
		}
	}

	return nil
}

// getDatabases gets all databases of an instance.
func getDatabases(txn *sql.Tx) ([]string, error) {
	var dbNames []string
	rows, err := txn.Query("SELECT name FROM system.databases")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		dbNames = append(dbNames, name)
	}
	return dbNames, nil
}

// getDatabaseStmt gets the create statement of a database.
func getDatabaseStmt(txn *sql.Tx, dbName string) (string, error) {
	query := fmt.Sprintf("SHOW CREATE DATABASE IF NOT EXISTS %s;", dbName)
	rows, err := txn.Query(query)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	if rows.Next() {
		var stmt, unused string
		if err := rows.Scan(&unused, &stmt); err != nil {
			return "", err
		}
		return fmt.Sprintf("%s;\n", stmt), nil
	}
	return "", fmt.Errorf("query %q returned empty row", query)
}

// tableSchema describes the schema of a table or view.
type tableSchema struct {
	name      string
	tableType string
	statement string
}

// getTables gets all tables of a database.
func getTables(txn *sql.Tx, dbName string) ([]*tableSchema, error) {
	var tables []*tableSchema
	query := fmt.Sprintf("SELECT name, engine, create_table_query FROM system.tables WHERE database='%s';", dbName)
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var tbl tableSchema
		if err := rows.Scan(&tbl.name, &tbl.tableType, &tbl.statement); err != nil {
			return nil, err
		}
		tables = append(tables, &tbl)
	}

	for _, tbl := range tables {
		// Remove the database prefix from statement.
		tbl.statement = strings.ReplaceAll(tbl.statement, fmt.Sprintf(" %s.%s ", dbName, tbl.name), fmt.Sprintf(" %s ", tbl.name))
		if tbl.tableType == "View" {
			tbl.statement = fmt.Sprintf(viewStmtFmt, tbl.name, tbl.statement)
		} else {
			tbl.statement = fmt.Sprintf(tableStmtFmt, tbl.name, tbl.statement)
		}
	}
	return tables, nil
}

// Restore restores a database.
func (driver *Driver) Restore(ctx context.Context, sc *bufio.Scanner) (err error) {
	txn, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer txn.Rollback()

	f := func(stmt string) error {
		if _, err := txn.Exec(stmt); err != nil {
			return err
		}
		return nil
	}

	if err := util.ApplyMultiStatements(sc, f); err != nil {
		return err
	}

	if err := txn.Commit(); err != nil {
		return err
	}

	return nil
}

// RestoreTx restores the database in the given transaction.
func (driver *Driver) RestoreTx(ctx context.Context, tx *sql.Tx, sc *bufio.Scanner) error {
	return fmt.Errorf("Unimplemented")
}
