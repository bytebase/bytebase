package snowflake

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"io"
	"strings"

	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/db/util"

	snow "github.com/snowflakedb/gosnowflake"
	"go.uber.org/zap"
)

var (
	systemSchemas = map[string]bool{
		"information_schema": true,
	}
	bytebaseDatabase = "BYTEBASE"
	sysAdminRole     = "SYSADMIN"
	accountAdminRole = "ACCOUNTADMIN"

	_ db.Driver = (*Driver)(nil)
)

func init() {
	db.Register(db.Snowflake, newDriver)
}

// Driver is the Snowflake driver.
type Driver struct {
	connectionCtx db.ConnectionContext
	dbType        db.Type

	db *sql.DB
}

func newDriver(config db.DriverConfig) db.Driver {
	return &Driver{}
}

// Open opens a Snowflake driver.
func (driver *Driver) Open(ctx context.Context, dbType db.Type, config db.ConnectionConfig, connCtx db.ConnectionContext) (db.Driver, error) {
	prefixParts, loggedPrefixParts := []string{config.Username}, []string{config.Username}
	if config.Password != "" {
		prefixParts = append(prefixParts, config.Password)
		loggedPrefixParts = append(loggedPrefixParts, "<<redacted password>>")
	}

	var account, host string
	// Host can also be account e.g. xma12345, or xma12345@host_ip where host_ip is the proxy server IP.
	if strings.Contains(config.Host, "@") {
		parts := strings.Split(config.Host, "@")
		if len(parts) != 2 {
			return nil, fmt.Errorf("driver.Open() has invalid host %q", config.Host)
		}
		account, host = parts[0], parts[1]
	} else {
		account = config.Host
	}

	var params []string
	var suffix string
	if host != "" {
		suffix = fmt.Sprintf("%s:%s", host, config.Port)
		params = append(params, fmt.Sprintf("account=%s", account))
	} else {
		suffix = account
	}

	dsn := fmt.Sprintf("%s@%s/%s", strings.Join(prefixParts, ":"), suffix, config.Database)
	loggedDSN := fmt.Sprintf("%s@%s/%s", strings.Join(loggedPrefixParts, ":"), suffix, config.Database)
	if len(params) > 0 {
		dsn = fmt.Sprintf("%s?%s", dsn, strings.Join(params, "&"))
		loggedDSN = fmt.Sprintf("%s?%s", loggedDSN, strings.Join(params, "&"))
	}
	log.Debug("Opening Snowflake driver",
		zap.String("dsn", loggedDSN),
		zap.String("environment", connCtx.EnvironmentName),
		zap.String("database", connCtx.InstanceName),
	)
	db, err := sql.Open("snowflake", dsn)
	if err != nil {
		panic(err)
	}
	driver.dbType = dbType
	driver.db = db
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
	query := "SELECT CURRENT_VERSION()"
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

func (driver *Driver) useRole(ctx context.Context, role string) error {
	query := fmt.Sprintf("USE ROLE %s", role)
	if _, err := driver.db.ExecContext(ctx, query); err != nil {
		return util.FormatErrorWithQuery(err, query)
	}
	return nil
}

func (driver *Driver) getDatabases(ctx context.Context) ([]string, error) {
	txn, err := driver.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer txn.Rollback()

	databases, err := getDatabasesTxn(ctx, txn)
	if err != nil {
		return nil, err
	}

	if err := txn.Commit(); err != nil {
		return nil, err
	}

	return databases, nil
}

func getDatabasesTxn(ctx context.Context, tx *sql.Tx) ([]string, error) {
	if _, err := tx.ExecContext(ctx, fmt.Sprintf("USE ROLE %s", accountAdminRole)); err != nil {
		return nil, err
	}

	// Filter inbound shared databases because they are immutable and we cannot get their DDLs.
	inboundDatabases := make(map[string]bool)
	shareRows, err := tx.Query("SHOW SHARES;")
	if err != nil {
		return nil, err
	}
	defer shareRows.Close()

	cols, err := shareRows.ColumnTypes()
	if err != nil {
		return nil, err
	}
	// created_on, kind, name, database_name.
	if len(cols) < 4 {
		return nil, nil
	}
	values := make([]*sql.NullString, len(cols))
	refs := make([]interface{}, len(cols))
	for i := 0; i < len(cols); i++ {
		refs[i] = &values[i]
	}
	for shareRows.Next() {
		if err := shareRows.Scan(refs...); err != nil {
			return nil, err
		}
		if values[1].String == "INBOUND" {
			inboundDatabases[values[3].String] = true
		}
	}
	if err := shareRows.Err(); err != nil {
		return nil, err
	}

	query := `
		SELECT
			DATABASE_NAME
		FROM SNOWFLAKE.INFORMATION_SCHEMA.DATABASES`
	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()

	var databases []string
	for rows.Next() {
		var name string
		if err := rows.Scan(
			&name,
		); err != nil {
			return nil, err
		}

		if _, ok := inboundDatabases[name]; !ok {
			databases = append(databases, name)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return databases, nil
}

// Execute executes a SQL statement.
func (driver *Driver) Execute(ctx context.Context, statement string) error {
	count := 0
	f := func(stmt string) error {
		count++
		return nil
	}
	sc := bufio.NewScanner(strings.NewReader(statement))
	if err := util.ApplyMultiStatements(sc, f); err != nil {
		return err
	}

	if count <= 0 {
		return nil
	}

	if err := driver.useRole(ctx, sysAdminRole); err != nil {
		return nil
	}
	tx, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	mctx, err := snow.WithMultiStatement(ctx, count)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(mctx, statement); err != nil {
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
		"-- Snowflake database structure for %s\n" +
		"--\n"
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
	var dumpableDbNames []string
	if database != "" {
		dumpableDbNames = []string{database}
	} else {
		var err error
		dumpableDbNames, err = getDatabasesTxn(ctx, txn)
		if err != nil {
			return fmt.Errorf("failed to get databases: %s", err)
		}
	}

	// Use ACCOUNTADMIN role to dump database;
	if _, err := txn.ExecContext(ctx, fmt.Sprintf("USE ROLE %s", accountAdminRole)); err != nil {
		return err
	}

	for _, dbName := range dumpableDbNames {
		// includeCreateDatabaseStmt should be false if dumping a single database.
		dumpSingleDatabase := len(dumpableDbNames) == 1
		dbName = strings.ToUpper(dbName)
		if err := dumpOneDatabase(ctx, txn, dbName, out, schemaOnly, dumpSingleDatabase); err != nil {
			return err
		}
	}

	return nil
}

// dumpOneDatabase will dump the database DDL schema for a database.
// Note: this operation is not supported on shared databases, e.g. SNOWFLAKE_SAMPLE_DATA.
func dumpOneDatabase(ctx context.Context, txn *sql.Tx, database string, out io.Writer, schemaOnly bool, dumpSingleDatabase bool) error {
	if !dumpSingleDatabase {
		// Database header.
		header := fmt.Sprintf(databaseHeaderFmt, database)
		if _, err := io.WriteString(out, header); err != nil {
			return err
		}
	}

	query := fmt.Sprintf(`SELECT GET_DDL('DATABASE', '%s', true)`, database)
	rows, err := txn.QueryContext(ctx, query)
	if err != nil {
		return util.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()

	var databaseDDL string
	for rows.Next() {
		if err := rows.Scan(
			&databaseDDL,
		); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	// Transform1: if dumpSingleDatabase, we should remove `create or replace database` statement.
	if dumpSingleDatabase {
		lines := strings.Split(databaseDDL, "\n")
		if len(lines) >= 2 {
			lines = lines[2:]
		}
		databaseDDL = strings.Join(lines, "\n")
	}

	// Transform2: remove "create or replace schema PUBLIC;\n\n" because it's created by default.
	schemaStmt := fmt.Sprintf("create or replace schema %s.PUBLIC;", database)
	databaseDDL = strings.ReplaceAll(databaseDDL, schemaStmt+"\n\n", "")
	// If this is the last statement.
	databaseDDL = strings.ReplaceAll(databaseDDL, schemaStmt, "")

	var lines []string
	for _, line := range strings.Split(databaseDDL, "\n") {
		if strings.HasPrefix(strings.ToLower(line), "create ") {
			// Transform3: Remove "DEMO_DB." quantifier.
			line = strings.ReplaceAll(line, fmt.Sprintf(" %s.", database), " ")

			// Transform4 (Important!): replace all `create or replace ` with `create ` to not break existing schema by any chance.
			line = strings.ReplaceAll(line, "create or replace ", "create ")
		}
		lines = append(lines, line)
	}
	databaseDDL = strings.Join(lines, "\n")

	if _, err := io.WriteString(out, databaseDDL); err != nil {
		return err
	}

	return nil
}

// Restore restores a database.
func (driver *Driver) Restore(ctx context.Context, sc *bufio.Scanner) (err error) {
	if err := driver.useRole(ctx, sysAdminRole); err != nil {
		return nil
	}
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
