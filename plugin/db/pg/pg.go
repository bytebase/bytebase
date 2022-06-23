package pg

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bytebase/bytebase/common/log"

	// Import pg driver.
	// init() in pgx/v4/stdlib will register it's pgx driver
	_ "github.com/jackc/pgx/v4/stdlib"

	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/db/util"
)

var (
	systemDatabases = map[string]bool{
		"template0": true,
		"template1": true,
	}
	ident                      = regexp.MustCompile(`(?i)^[a-z_][a-z0-9_$]*$`)
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
func (driver *Driver) Open(ctx context.Context, dbType db.Type, config db.ConnectionConfig, connCtx db.ConnectionContext) (db.Driver, error) {
	if (config.TLSConfig.SslCert == "" && config.TLSConfig.SslKey != "") ||
		(config.TLSConfig.SslCert != "" && config.TLSConfig.SslKey == "") {
		return nil, fmt.Errorf("ssl-cert and ssl-key must be both set or unset")
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

	if sslCA == "" {
		// We should use the default connection dsn without setting sslmode.
		// Some provider might still perform default SSL check at the server side so we
		// shouldn't disable sslmode at the client side.
		// m["sslmode"] = "disable"
	} else {
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
		db, err := sql.Open(driverName, guessDSN)
		if err != nil {
			continue
		}
		defer db.Close()

		if err = db.Ping(); err != nil {
			continue
		}
		return guess, guessDSN, nil
	}

	if database != "" {
		return "", "", fmt.Errorf("cannot connecting %q, make sure the connection info is correct and the database exists", database)
	}
	return "", "", fmt.Errorf("cannot connecting instance, make sure the connection info is correct")
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
	if err := driver.switchDatabase(database); err != nil {
		return nil, err
	}
	return driver.db, nil
}

// GetVersion gets the version of Postgres server.
func (driver *Driver) GetVersion(ctx context.Context) (string, error) {
	query := "SHOW server_version"
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
	var remainingStmts []string
	f := func(stmt string) error {
		stmt = strings.TrimLeft(stmt, " \t")
		// We don't use transaction for creating / altering databases in Postgres.
		// https://github.com/bytebase/bytebase/issues/202
		if strings.HasPrefix(stmt, "CREATE DATABASE ") {
			databases, err := driver.getDatabases()
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
			// For the case of `\connect "dbname";`, we need to use GetDbConnection() instead of executing the statement.
			parts := strings.Split(stmt, `"`)
			if len(parts) != 3 {
				return fmt.Errorf("invalid statement %q", stmt)
			}
			_, err := driver.GetDbConnection(ctx, parts[1])
			return err
		} else {
			remainingStmts = append(remainingStmts, stmt)
		}
		return nil
	}
	sc := bufio.NewScanner(strings.NewReader(statement))
	if err := util.ApplyMultiStatements(sc, f); err != nil {
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

	owner, err := driver.getCurrentDatabaseOwner(tx)
	if err != nil {
		return err
	}
	// Set the current transaction role to the database owner so that the owner of created database will be the same as the database owner.
	if _, err := tx.ExecContext(ctx, fmt.Sprintf("SET LOCAL ROLE %s", owner)); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, strings.Join(remainingStmts, "\n")); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func getDatabaseInCreateDatabaseStatement(createDatabaseStatement string) (string, error) {
	raw := strings.TrimRight(createDatabaseStatement, ";")
	raw = strings.TrimPrefix(raw, "CREATE DATABASE")
	tokens := strings.Fields(raw)
	if len(tokens) == 0 {
		return "", fmt.Errorf("database name not found")
	}
	databaseName := strings.TrimLeft(tokens[0], `"`)
	databaseName = strings.TrimRight(databaseName, `"`)
	return databaseName, nil
}

func (driver *Driver) getCurrentDatabaseOwner(txn *sql.Tx) (string, error) {
	const query = `
		SELECT
			u.rolname
		FROM
			pg_roles AS u JOIN pg_database AS d ON (d.datdba = u.oid)
		WHERE
			d.datname = current_database();
		`
	rows, err := txn.Query(query)
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
	if owner == "" {
		return "", fmt.Errorf("Owner not found for the current database")
	}
	return owner, nil
}

// Query queries a SQL statement.
func (driver *Driver) Query(ctx context.Context, statement string, limit int) ([]interface{}, error) {
	return util.Query(ctx, driver.db, statement, limit)
}

// Dump and restore

// Dump dumps the database.
func (driver *Driver) Dump(ctx context.Context, database string, out io.Writer, schemaOnly bool) (string, error) {
	// pg_dump -d dbName --schema-only+

	// Find all dumpable databases
	databases, err := driver.getDatabases()
	if err != nil {
		return "", fmt.Errorf("failed to get databases: %s", err)
	}

	var dumpableDbNames []string
	if database != "" {
		exist := false
		for _, n := range databases {
			if n.name == database {
				exist = true
				break
			}
		}
		if !exist {
			return "", fmt.Errorf("database %s not found", database)
		}
		dumpableDbNames = []string{database}
	} else {
		for _, n := range databases {
			if systemDatabases[n.name] {
				continue
			}
			dumpableDbNames = append(dumpableDbNames, n.name)
		}
	}

	for _, dbName := range dumpableDbNames {
		includeUseDatabase := len(dumpableDbNames) > 1
		if err := driver.dumpOneDatabaseWithPgDump(ctx, dbName, out, schemaOnly, includeUseDatabase); err != nil {
			return "", err
		}
	}

	return "", nil
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

func (driver *Driver) dumpOneDatabaseWithPgDump(ctx context.Context, database string, out io.Writer, schemaOnly bool, includeUseDatabase bool) error {
	var args []string
	args = append(args, fmt.Sprintf("--username=%s", driver.config.Username))
	if driver.config.Password == "" {
		args = append(args, "--no-password")
	}
	args = append(args, fmt.Sprintf("--host=%s", driver.config.Host))
	args = append(args, fmt.Sprintf("--port=%s", driver.config.Port))
	if schemaOnly {
		args = append(args, "--schema-only")
	}
	args = append(args, "--inserts")
	args = append(args, "--use-set-session-authorization")
	args = append(args, database)
	pgDumpPath := filepath.Join(driver.pgInstanceDir, "bin", "pg_dump")
	cmd := exec.Command(pgDumpPath, args...)
	if driver.config.Password != "" {
		// Unlike MySQL, PostgreSQL does not support specifying commands in commands, we can do this by means of environment variables.
		cmd.Env = append(cmd.Env, fmt.Sprintf("PGPASSWORD=%s", driver.config.Password))
	}
	cmd.Stderr = os.Stderr
	r, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	s := bufio.NewScanner(r)
	previousLineComment := false
	previousLineEmpty := false
	for s.Scan() {
		line := s.Text()
		// Skip "SET SESSION AUTHORIZATION" till we can support it.
		if strings.HasPrefix(line, "SET SESSION AUTHORIZATION ") {
			continue
		}
		// Skip comment lines.
		if strings.HasPrefix(line, "--") {
			previousLineComment = true
			continue
		}
		if previousLineComment && line == "" {
			previousLineComment = false
			continue
		}
		previousLineComment = false
		// Skip extra empty lines.
		if line == "" {
			if previousLineEmpty {
				continue
			}
			previousLineEmpty = true
		} else {
			previousLineEmpty = false
		}

		if _, err := io.WriteString(out, line); err != nil {
			return err
		}
		if _, err := io.WriteString(out, "\n"); err != nil {
			return err
		}
	}
	if s.Err() != nil {
		log.Warn(s.Err().Error())
	}
	if err := cmd.Wait(); err != nil {
		return err
	}
	return nil
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

// getDatabases gets all databases of an instance.
func (driver *Driver) getDatabases() ([]*pgDatabaseSchema, error) {
	var dbs []*pgDatabaseSchema
	rows, err := driver.db.Query("SELECT datname, pg_encoding_to_char(encoding), datcollate FROM pg_database;")
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
	return dbs, nil
}

func (driver *Driver) strictUseDb() bool {
	return len(driver.strictDatabase) != 0
}

// quoteIdentifier will quote identifiers including keywords, capital characters, or special characters.
func quoteIdentifier(s string) string {
	quote := false
	if reserved[strings.ToUpper(s)] {
		quote = true
	}
	if !ident.MatchString(s) {
		quote = true
	}
	if quote {
		return fmt.Sprintf("\"%s\"", strings.ReplaceAll(s, "\"", "\"\""))
	}
	return s

}
