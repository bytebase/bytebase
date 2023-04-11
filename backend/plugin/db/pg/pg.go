// Package pg is the plugin for PostgreSQL driver.
package pg

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"

	// Import pg driver.
	// init() in pgx/v5/stdlib will register it's pgx driver.
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/plugin/parser"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	excludedDatabaseList = map[string]bool{
		// Skip our internal "bytebase" database
		"bytebase": true,
		// Skip internal databases from cloud service providers
		// see https://github.com/bytebase/bytebase/issues/30
		// aws
		"rdsadmin": true,
		// gcp
		"cloudsql":      true,
		"cloudsqladmin": true,
		// system templates.
		"template0": true,
		"template1": true,
	}

	// driverName is the driver name that our driver dependence register, now is "pgx".
	driverName = "pgx"

	_ db.Driver = (*Driver)(nil)
)

func init() {
	db.Register(db.Postgres, newDriver)
}

// Driver is the Postgres driver.
type Driver struct {
	dbBinDir      string
	connectionCtx db.ConnectionContext
	config        db.ConnectionConfig

	db *sql.DB
	// connectionString is the connection string registered by pgx.
	// Unregister connectionString if we don't need it.
	connectionString string
	baseDSN          string
	databaseName     string
}

func newDriver(config db.DriverConfig) db.Driver {
	return &Driver{
		dbBinDir: config.DbBinDir,
	}
}

// Open opens a Postgres driver.
func (driver *Driver) Open(_ context.Context, _ db.Type, config db.ConnectionConfig, connCtx db.ConnectionContext) (db.Driver, error) {
	// Require username for Postgres, as the guessDSN 1st guess is to use the username as the connecting database
	// if database name is not explicitly specified.
	if config.Username == "" {
		return nil, errors.Errorf("user must be set")
	}

	if (config.TLSConfig.SslCert == "" && config.TLSConfig.SslKey != "") ||
		(config.TLSConfig.SslCert != "" && config.TLSConfig.SslKey == "") {
		return nil, errors.Errorf("ssl-cert and ssl-key must be both set or unset")
	}

	databaseName, dsn, err := guessDSN(
		config.Username,
		config.Password,
		config.Host,
		config.Port,
		config.Database,
		config.TLSConfig.SslCA,
		config.TLSConfig.SslCert,
		config.TLSConfig.SslKey,
	)
	if err != nil {
		return nil, err
	}
	if config.ReadOnly {
		dsn = fmt.Sprintf("%s default_transaction_read_only=true", dsn)
	}
	driver.databaseName = databaseName
	driver.baseDSN = dsn
	driver.connectionCtx = connCtx
	driver.config = config

	connectionString, err := registerConnectionConfig(dsn, driver.config.TLSConfig)
	if err != nil {
		return nil, err
	}
	driver.connectionString = connectionString

	db, err := sql.Open(driverName, driver.connectionString)
	if err != nil {
		return nil, err
	}
	driver.db = db
	return driver, nil
}

func registerConnectionConfig(dsn string, tlsConfig db.TLSConfig) (string, error) {
	connConfig, err := pgx.ParseConfig(dsn)
	if err != nil {
		return "", err
	}

	if tlsConfig.SslCA != "" {
		sslConfig, err := tlsConfig.GetSslConfig()
		if err != nil {
			return "", err
		}
		connConfig.TLSConfig = sslConfig
	}

	return stdlib.RegisterConnConfig(connConfig), nil
}

func unregisterConnectionConfig(connectionString string) {
	stdlib.UnregisterConnConfig(connectionString)
}

// guessDSN will guess a valid DB connection and its database name.
func guessDSN(username, password, hostname, port, database, sslCA, sslCert, sslKey string) (string, string, error) {
	// dbname is guessed if not specified.
	m := map[string]string{
		"host":     hostname,
		"port":     port,
		"user":     username,
		"password": password,
	}
	if database != "" {
		m["dbname"] = database
	}

	tlsConfig := db.TLSConfig{
		SslCA:   sslCA,
		SslCert: sslCert,
		SslKey:  sslKey,
	}

	var tokens []string
	for k, v := range m {
		if v != "" {
			tokens = append(tokens, fmt.Sprintf("%s=%s", k, v))
		}
	}
	dsn := strings.Join(tokens, " ")

	if database != "" {
		return database, dsn, nil
	}

	// Some postgres server default behavior is to use username as the database name if not specified,
	// while some postgres server explicitly requires the database name to be present (e.g. render.com).
	guesses := []string{"postgres", username, "template1"}
	//  dsn+" dbname=bytebase"
	for _, guessDatabase := range guesses {
		guessDSN := fmt.Sprintf("%s dbname=%s", dsn, guessDatabase)
		if err := func() error {
			connectionString, err := registerConnectionConfig(guessDSN, tlsConfig)
			if err != nil {
				return err
			}
			defer unregisterConnectionConfig(connectionString)
			db, err := sql.Open(driverName, connectionString)
			if err != nil {
				return err
			}
			defer db.Close()
			return db.Ping()
		}(); err != nil {
			log.Debug("guessDSN attempt failed", zap.Error(err))
			continue
		}
		return guessDatabase, guessDSN, nil
	}
	return "", "", errors.Errorf("cannot connect to the instance, make sure the connection info is correct")
}

// Close closes the driver.
func (driver *Driver) Close(context.Context) error {
	unregisterConnectionConfig(driver.connectionString)
	return driver.db.Close()
}

// Ping pings the database.
func (driver *Driver) Ping(ctx context.Context) error {
	return driver.db.PingContext(ctx)
}

// GetType returns the database type.
func (*Driver) GetType() db.Type {
	return db.Postgres
}

// GetDB gets the database.
func (driver *Driver) GetDB() *sql.DB {
	return driver.db
}

// getDatabases gets all databases of an instance.
func (driver *Driver) getDatabases(ctx context.Context) ([]*storepb.DatabaseMetadata, error) {
	var databases []*storepb.DatabaseMetadata
	rows, err := driver.db.QueryContext(ctx, "SELECT datname, pg_encoding_to_char(encoding), datcollate FROM pg_database;")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		database := &storepb.DatabaseMetadata{}
		if err := rows.Scan(&database.Name, &database.CharacterSet, &database.Collation); err != nil {
			return nil, err
		}
		databases = append(databases, database)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return databases, nil
}

// getVersion gets the version of Postgres server.
func (driver *Driver) getVersion(ctx context.Context) (string, error) {
	query := "SHOW server_version"
	var version string
	if err := driver.db.QueryRowContext(ctx, query).Scan(&version); err != nil {
		if err == sql.ErrNoRows {
			return "", common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return "", util.FormatErrorWithQuery(err, query)
	}
	return version, nil
}

// Execute will execute the statement. For CREATE DATABASE statement, some types of databases such as Postgres
// will not use transactions to execute the statement but will still use transactions to execute the rest of statements.
func (driver *Driver) Execute(ctx context.Context, statement string, createDatabase bool) (int64, error) {
	if createDatabase {
		databases, err := driver.getDatabases(ctx)
		if err != nil {
			return 0, err
		}
		databaseName, err := getDatabaseInCreateDatabaseStatement(statement)
		if err != nil {
			return 0, err
		}
		exist := false
		for _, database := range databases {
			if database.Name == databaseName {
				exist = true
				break
			}
		}
		if exist {
			return 0, err
		}

		f := func(stmt string) error {
			if _, err := driver.db.ExecContext(ctx, stmt); err != nil {
				return err
			}
			return nil
		}
		if _, err := parser.SplitMultiSQLStream(parser.Postgres, strings.NewReader(statement), f); err != nil {
			return 0, err
		}
		return 0, nil
	}

	owner, err := driver.GetCurrentDatabaseOwner()
	if err != nil {
		return 0, err
	}

	var remainingStmts []string
	var nonTransactionStmts []string
	totalRowsAffected := int64(0)
	f := func(stmt string) error {
		// We don't use transaction for creating / altering databases in Postgres.
		// We will execute the statement directly before "\\connect" statement.
		// https://github.com/bytebase/bytebase/issues/202
		if isSuperuserStatement(stmt) {
			// CREATE EVENT TRIGGER statement only supports EXECUTE PROCEDURE in version 10 and before, while newer version supports both EXECUTE { FUNCTION | PROCEDURE }.
			// Since we use pg_dump version 14, the dump uses a new style even for an old version of PostgreSQL.
			// We should convert EXECUTE FUNCTION to EXECUTE PROCEDURE to make the restoration work on old versions.
			// https://www.postgresql.org/docs/14/sql-createeventtrigger.html
			if strings.Contains(strings.ToUpper(stmt), "CREATE EVENT TRIGGER") {
				stmt = strings.ReplaceAll(stmt, "EXECUTE FUNCTION", "EXECUTE PROCEDURE")
			}
			// Use superuser privilege to run privileged statements.
			stmt = fmt.Sprintf("SET LOCAL ROLE NONE;%sSET LOCAL ROLE '%s';", stmt, owner)
			remainingStmts = append(remainingStmts, stmt)
		} else if isNonTransactionStatement(stmt) {
			nonTransactionStmts = append(nonTransactionStmts, stmt)
		} else if !isIgnoredStatement(stmt) {
			remainingStmts = append(remainingStmts, stmt)
		}
		return nil
	}

	if _, err := parser.SplitMultiSQLStream(parser.Postgres, strings.NewReader(statement), f); err != nil {
		return 0, err
	}

	if len(remainingStmts) != 0 {
		tx, err := driver.db.BeginTx(ctx, nil)
		if err != nil {
			return 0, err
		}
		defer tx.Rollback()

		// Set the current transaction role to the database owner so that the owner of created database will be the same as the database owner.
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("SET LOCAL ROLE '%s'", owner)); err != nil {
			return 0, err
		}

		sqlResult, err := tx.ExecContext(ctx, strings.Join(remainingStmts, "\n"))
		if err != nil {
			return 0, err
		}
		if err := tx.Commit(); err != nil {
			return 0, err
		}
		rowsAffected, err := sqlResult.RowsAffected()
		if err != nil {
			// Since we cannot differentiate DDL and DML yet, we have to ignore the error.
			log.Debug("rowsAffected returns error", zap.Error(err))
		} else {
			totalRowsAffected += rowsAffected
		}
	}

	// Run non-transaction statements at the end.
	for _, stmt := range nonTransactionStmts {
		if _, err := driver.db.Exec(stmt); err != nil {
			return 0, err
		}
	}
	return totalRowsAffected, nil
}

func isSuperuserStatement(stmt string) bool {
	upperCaseStmt := strings.ToUpper(stmt)
	if strings.HasPrefix(upperCaseStmt, "GRANT") || strings.HasPrefix(upperCaseStmt, "CREATE EXTENSION") || strings.HasPrefix(upperCaseStmt, "CREATE EVENT TRIGGER") || strings.HasPrefix(upperCaseStmt, "COMMENT ON EVENT TRIGGER") {
		return true
	}
	return false
}

func isIgnoredStatement(stmt string) bool {
	// Extensions created in AWS Aurora PostgreSQL are owned by rdsadmin.
	// We don't have privileges to comment on the extension and have to ignore it.
	upperCaseStmt := strings.ToUpper(stmt)
	return strings.HasPrefix(upperCaseStmt, "COMMENT ON EXTENSION")
}

func isNonTransactionStatement(stmt string) bool {
	// CREATE INDEX CONCURRENTLY cannot run inside a transaction block.
	// CREATE [ UNIQUE ] INDEX [ CONCURRENTLY ] [ [ IF NOT EXISTS ] name ] ON [ ONLY ] table_name [ USING method ] ...
	createIndexReg := regexp.MustCompile(`(?i)CREATE(\s+(UNIQUE\s+)?)INDEX(\s+)CONCURRENTLY`)
	if len(createIndexReg.FindString(stmt)) > 0 {
		return true
	}

	// DROP INDEX CONCURRENTLY cannot run inside a transaction block.
	// DROP INDEX [ CONCURRENTLY ] [ IF EXISTS ] name [, ...] [ CASCADE | RESTRICT ]
	dropIndexReg := regexp.MustCompile(`(?i)DROP(\s+)INDEX(\s+)CONCURRENTLY`)
	return len(dropIndexReg.FindString(stmt)) > 0
}

func getDatabaseInCreateDatabaseStatement(createDatabaseStatement string) (string, error) {
	raw := strings.TrimRight(createDatabaseStatement, ";")
	raw = strings.TrimPrefix(raw, "CREATE DATABASE")
	tokens := strings.Fields(raw)
	if len(tokens) == 0 {
		return "", errors.Errorf("database name not found")
	}
	databaseName := strings.TrimLeft(tokens[0], `"`)
	databaseName = strings.TrimRight(databaseName, `"`)
	return databaseName, nil
}

// GetCurrentDatabaseOwner gets the role of the current database.
func (driver *Driver) GetCurrentDatabaseOwner() (string, error) {
	const query = `
		SELECT
			u.rolname
		FROM
			pg_roles AS u JOIN pg_database AS d ON (d.datdba = u.oid)
		WHERE
			d.datname = current_database();
		`
	rows, err := driver.db.Query(query)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var owner string
	for rows.Next() {
		var o string
		if err := rows.Scan(&o); err != nil {
			return "", err
		}
		owner = o
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	if owner == "" {
		return "", errors.Errorf("owner not found for the current database")
	}
	return owner, nil
}

// QueryConn querys a SQL statement in a given connection.
func (*Driver) QueryConn(ctx context.Context, conn *sql.Conn, statement string, queryContext *db.QueryContext) ([]any, error) {
	singleSQLs, err := parser.SplitMultiSQL(parser.Postgres, statement)
	if err != nil {
		return nil, err
	}
	if len(singleSQLs) == 0 {
		return nil, nil
	}

	// If the statement is an INSERT, UPDATE, or DELETE statement, we will call execute instead of query and return the number of rows affected.
	// https://github.com/postgres/postgres/blob/master/src/bin/psql/common.c#L969
	if len(singleSQLs) == 1 && util.IsAffectedRowsStatement(singleSQLs[0].Text) {
		sqlResult, err := conn.ExecContext(ctx, singleSQLs[0].Text)
		if err != nil {
			return nil, err
		}
		affectedRows, err := sqlResult.RowsAffected()
		if err != nil {
			return nil, err
		}
		field := []string{"Affected Rows"}
		types := []string{"INT"}
		rows := [][]any{{affectedRows}}
		return []any{field, types, rows}, nil
	}
	return util.Query(ctx, db.Postgres, conn, statement, queryContext)
}
