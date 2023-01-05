// Package pg is the plugin for PostgreSQL driver.
package pg

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	// Import pg driver.
	// init() in pgx/v5/stdlib will register it's pgx driver.
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/db/util"
	"github.com/bytebase/bytebase/plugin/parser"
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

	createBytebaseDatabaseStmt = "CREATE DATABASE bytebase;"

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

	db           *sql.DB
	baseDSN      string
	databaseName string

	// strictDatabase should be used only if the user gives only a database instead of a whole instance to access.
	strictDatabase string
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
	if config.StrictUseDb {
		driver.strictDatabase = config.Database
	}

	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, err
	}
	driver.db = db
	return driver, nil
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

	// We should use the default connection dsn without setting sslmode.
	// Some provider might still perform default SSL check at the server side so we
	// shouldn't disable sslmode at the client side.
	// m["sslmode"] = "disable"
	if sslCA != "" {
		m["sslmode"] = "verify-ca"
		m["sslrootcert"] = sslCA
		if sslCert != "" && sslKey != "" {
			m["sslcert"] = sslCert
			m["sslkey"] = sslKey
		}
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
	guesses := []string{"postgres", "bytebase", username, "template1"}
	//  dsn+" dbname=bytebase"
	for _, guessDatabase := range guesses {
		guessDSN := fmt.Sprintf("%s dbname=%s", dsn, guessDatabase)
		if err := func() error {
			db, err := sql.Open(driverName, guessDSN)
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

// GetDBConnection gets a database connection.
func (driver *Driver) GetDBConnection(_ context.Context, database string) (*sql.DB, error) {
	if err := driver.switchDatabase(database); err != nil {
		return nil, err
	}
	return driver.db, nil
}

// getDatabases gets all databases of an instance.
func (driver *Driver) getDatabases(ctx context.Context) ([]*pgDatabaseSchema, error) {
	var dbs []*pgDatabaseSchema
	rows, err := driver.db.QueryContext(ctx, "SELECT datname, pg_encoding_to_char(encoding), datcollate FROM pg_database;")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var d pgDatabaseSchema
		if err := rows.Scan(&d.name, &d.encoding, &d.collate); err != nil {
			return nil, err
		}
		dbs = append(dbs, &d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return dbs, nil
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

// Execute executes a SQL statement.
func (driver *Driver) Execute(ctx context.Context, statement string, createDatabase bool) (int64, error) {
	owner, err := driver.GetCurrentDatabaseOwner()
	if err != nil {
		return 0, err
	}

	connected := false
	var remainingStmts []string
	totalRowsAffected := int64(0)
	f := func(stmt string) error {
		// We don't use transaction for creating / altering databases in Postgres.
		// We will execute the statement directly before "\\connect" statement.
		// https://github.com/bytebase/bytebase/issues/202
		if createDatabase && !connected {
			if strings.HasPrefix(stmt, "CREATE DATABASE ") {
				databases, err := driver.getDatabases(ctx)
				if err != nil {
					return err
				}
				databaseName, err := getDatabaseInCreateDatabaseStatement(stmt)
				if err != nil {
					return err
				}
				exist := false
				for _, database := range databases {
					if database.name == databaseName {
						exist = true
						break
					}
				}
				if !exist {
					if _, err := driver.db.ExecContext(ctx, stmt); err != nil {
						return err
					}
				}
			} else if strings.HasPrefix(stmt, "ALTER DATABASE") && strings.Contains(stmt, " OWNER TO ") {
				if _, err := driver.db.ExecContext(ctx, stmt); err != nil {
					return err
				}
			} else if strings.HasPrefix(stmt, "\\connect ") {
				// For the case of `\connect "dbname";`, we need to use GetDBConnection() instead of executing the statement.
				parts := strings.Split(stmt, `"`)
				if len(parts) != 3 {
					return errors.Errorf("invalid statement %q", stmt)
				}
				if _, err = driver.GetDBConnection(ctx, parts[1]); err != nil {
					return err
				}
				// Update current owner
				if owner, err = driver.GetCurrentDatabaseOwner(); err != nil {
					return err
				}
				connected = true
			} else {
				sqlResult, err := driver.db.ExecContext(ctx, stmt)
				if err != nil {
					return err
				}
				rowsAffected, err := sqlResult.RowsAffected()
				if err != nil {
					// Since we cannot differentiate DDL and DML yet, we have to ignore the error.
					log.Debug("rowsAffected returns error", zap.Error(err))
				} else {
					totalRowsAffected += rowsAffected
				}
			}
		} else {
			if isSuperuserStatement(stmt) {
				// CREATE EVENT TRIGGER statement only supports EXECUTE PROCEDURE in version 10 and before, while newer version supports both EXECUTE { FUNCTION | PROCEDURE }.
				// Since we use pg_dump version 14, the dump uses a new style even for an old version of PostgreSQL.
				// We should convert EXECUTE FUNCTION to EXECUTE PROCEDURE to make the restoration work on old versions.
				// https://www.postgresql.org/docs/14/sql-createeventtrigger.html
				if strings.Contains(strings.ToUpper(stmt), "CREATE EVENT TRIGGER") {
					stmt = strings.ReplaceAll(stmt, "EXECUTE FUNCTION", "EXECUTE PROCEDURE")
				}
				// Use superuser privilege to run privileged statements.
				stmt = fmt.Sprintf("SET LOCAL ROLE NONE;%sSET LOCAL ROLE %s;", stmt, owner)
				remainingStmts = append(remainingStmts, stmt)
			} else if !isIgnoredStatement(stmt) {
				remainingStmts = append(remainingStmts, stmt)
			}
		}
		return nil
	}

	if _, err := parser.SplitMultiSQLStream(parser.Postgres, strings.NewReader(statement), f); err != nil {
		return 0, err
	}

	if len(remainingStmts) == 0 {
		return 0, nil
	}

	tx, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// Set the current transaction role to the database owner so that the owner of created database will be the same as the database owner.
	if _, err := tx.ExecContext(ctx, fmt.Sprintf("SET LOCAL ROLE %s", owner)); err != nil {
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

// Query queries a SQL statement.
func (driver *Driver) Query(ctx context.Context, statement string, queryContext *db.QueryContext) ([]interface{}, error) {
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
		affectedRows, err := driver.Execute(ctx, singleSQLs[0].Text, false)
		if err != nil {
			return nil, err
		}
		field := []string{"Affected Rows"}
		types := []string{"INT"}
		rows := [][]interface{}{{affectedRows}}
		return []interface{}{field, types, rows}, nil
	}
	return util.Query(ctx, db.Postgres, driver.db, statement, queryContext)
}

func (driver *Driver) switchDatabase(dbName string) error {
	if driver.db != nil {
		if err := driver.db.Close(); err != nil {
			return err
		}
	}

	dsn := driver.baseDSN + " dbname=" + dbName
	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return err
	}
	driver.db = db
	driver.databaseName = dbName
	return nil
}
