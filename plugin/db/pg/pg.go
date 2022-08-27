// Package pg is the plugin for PostgreSQL driver.
package pg

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	// Import pg driver.
	// init() in pgx/v4/stdlib will register it's pgx driver.
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/db/util"
	"github.com/bytebase/bytebase/plugin/parser"
)

var (
	systemDatabases = map[string]bool{
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
	pgInstanceDir string
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
		pgInstanceDir: config.PgInstanceDir,
	}
}

// Open opens a Postgres driver.
func (driver *Driver) Open(_ context.Context, _ db.Type, config db.ConnectionConfig, connCtx db.ConnectionContext) (db.Driver, error) {
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

	var guesses []string
	if database != "" {
		guesses = append(guesses, database)
	} else {
		// Guess default database postgres, template1.
		guesses = append(guesses, "")
		guesses = append(guesses, "bytebase")
		guesses = append(guesses, "postgres")
		guesses = append(guesses, "template1")
	}

	//  dsn+" dbname=bytebase"
	for _, guess := range guesses {
		guessDSN := dsn
		if guess != "" {
			guessDSN = fmt.Sprintf("%s dbname=%s", dsn, guess)
		}
		if err := func() error {
			db, err := sql.Open(driverName, guessDSN)
			if err != nil {
				return err
			}
			defer db.Close()
			return db.Ping()
		}(); err != nil {
			continue
		}
		return guess, guessDSN, nil
	}

	if database != "" {
		return "", "", errors.Errorf("cannot connecting %q, make sure the connection info is correct and the database exists", database)
	}
	return "", "", errors.Errorf("cannot connecting instance, make sure the connection info is correct")
}

// Close closes the driver.
func (driver *Driver) Close(context.Context) error {
	return driver.db.Close()
}

// Ping pings the database.
func (driver *Driver) Ping(ctx context.Context) error {
	return driver.db.PingContext(ctx)
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
func (driver *Driver) Execute(ctx context.Context, statement string) error {
	owner, err := driver.GetCurrentDatabaseOwner()
	if err != nil {
		return err
	}

	var remainingStmts []string
	f := func(stmt string) error {
		// We don't use transaction for creating / altering databases in Postgres.
		// https://github.com/bytebase/bytebase/issues/202
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
		} else if strings.HasPrefix(stmt, "GRANT") || strings.HasPrefix(stmt, "ALTER DATABASE") && strings.Contains(stmt, " OWNER TO ") {
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
		} else if isSuperuserStatement(stmt) {
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
		} else {
			remainingStmts = append(remainingStmts, stmt)
		}
		return nil
	}

	if _, err := parser.SplitMultiSQLStream(parser.Postgres, strings.NewReader(statement), f); err != nil {
		return err
	}

	if len(remainingStmts) == 0 {
		return nil
	}

	tx, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Set the current transaction role to the database owner so that the owner of created database will be the same as the database owner.
	if _, err := tx.ExecContext(ctx, fmt.Sprintf("SET LOCAL ROLE %s", owner)); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, strings.Join(remainingStmts, "\n")); err != nil {
		return err
	}

	return tx.Commit()
}

func isSuperuserStatement(stmt string) bool {
	upperCaseStmt := strings.ToUpper(stmt)
	if strings.Contains(upperCaseStmt, "CREATE EVENT TRIGGER") || strings.Contains(upperCaseStmt, "CREATE EXTENSION") || strings.Contains(upperCaseStmt, "COMMENT ON EXTENSION") || strings.Contains(upperCaseStmt, "COMMENT ON EVENT TRIGGER") {
		return true
	}
	return false
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
func (driver *Driver) Query(ctx context.Context, statement string, limit int) ([]interface{}, error) {
	return util.Query(ctx, db.Postgres, driver.db, statement, limit)
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
