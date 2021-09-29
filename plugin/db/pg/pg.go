package pg

import (
	"bufio"
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/db/util"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

//go:embed pg_migration_schema.sql
var migrationSchema string

var (
	systemDatabases = map[string]bool{
		"template0": true,
		"template1": true,
	}
	reserved = map[string]bool{
		"AES128":            true,
		"AES256":            true,
		"ALL":               true,
		"ALLOWOVERWRITE":    true,
		"ANALYSE":           true,
		"ANALYZE":           true,
		"AND":               true,
		"ANY":               true,
		"ARRAY":             true,
		"AS":                true,
		"ASC":               true,
		"AUTHORIZATION":     true,
		"BACKUP":            true,
		"BETWEEN":           true,
		"BINARY":            true,
		"BLANKSASNULL":      true,
		"BOTH":              true,
		"BYTEDICT":          true,
		"CASE":              true,
		"CAST":              true,
		"CHECK":             true,
		"COLLATE":           true,
		"COLUMN":            true,
		"CONSTRAINT":        true,
		"CREATE":            true,
		"CREDENTIALS":       true,
		"CROSS":             true,
		"CURRENT_DATE":      true,
		"CURRENT_TIME":      true,
		"CURRENT_TIMESTAMP": true,
		"CURRENT_USER":      true,
		"CURRENT_USER_ID":   true,
		"DEFAULT":           true,
		"DEFERRABLE":        true,
		"DEFLATE":           true,
		"DEFRAG":            true,
		"DELTA":             true,
		"DELTA32K":          true,
		"DESC":              true,
		"DISABLE":           true,
		"DISTINCT":          true,
		"DO":                true,
		"ELSE":              true,
		"EMPTYASNULL":       true,
		"ENABLE":            true,
		"ENCODE":            true,
		"ENCRYPT":           true,
		"ENCRYPTION":        true,
		"END":               true,
		"EXCEPT":            true,
		"EXPLICIT":          true,
		"FALSE":             true,
		"FOR":               true,
		"FOREIGN":           true,
		"FREEZE":            true,
		"FROM":              true,
		"FULL":              true,
		"GLOBALDICT256":     true,
		"GLOBALDICT64K":     true,
		"GRANT":             true,
		"GROUP":             true,
		"GZIP":              true,
		"HAVING":            true,
		"IDENTITY":          true,
		"IGNORE":            true,
		"ILIKE":             true,
		"IN":                true,
		"INITIALLY":         true,
		"INNER":             true,
		"INTERSECT":         true,
		"INTO":              true,
		"IS":                true,
		"ISNULL":            true,
		"JOIN":              true,
		"LEADING":           true,
		"LEFT":              true,
		"LIKE":              true,
		"LIMIT":             true,
		"LOCALTIME":         true,
		"LOCALTIMESTAMP":    true,
		"LUN":               true,
		"LUNS":              true,
		"LZO":               true,
		"LZOP":              true,
		"MINUS":             true,
		"MOSTLY13":          true,
		"MOSTLY32":          true,
		"MOSTLY8":           true,
		"NATURAL":           true,
		"NEW":               true,
		"NOT":               true,
		"NOTNULL":           true,
		"NULL":              true,
		"NULLS":             true,
		"OFF":               true,
		"OFFLINE":           true,
		"OFFSET":            true,
		"OLD":               true,
		"ON":                true,
		"ONLY":              true,
		"OPEN":              true,
		"OR":                true,
		"ORDER":             true,
		"OUTER":             true,
		"OVERLAPS":          true,
		"PARALLEL":          true,
		"PARTITION":         true,
		"PERCENT":           true,
		"PLACING":           true,
		"PRIMARY":           true,
		"RAW":               true,
		"READRATIO":         true,
		"RECOVER":           true,
		"REFERENCES":        true,
		"REJECTLOG":         true,
		"RESORT":            true,
		"RESTORE":           true,
		"RIGHT":             true,
		"SELECT":            true,
		"SESSION_USER":      true,
		"SIMILAR":           true,
		"SOME":              true,
		"SYSDATE":           true,
		"SYSTEM":            true,
		"TABLE":             true,
		"TAG":               true,
		"TDES":              true,
		"TEXT255":           true,
		"TEXT32K":           true,
		"THEN":              true,
		"TO":                true,
		"TOP":               true,
		"TRAILING":          true,
		"TRUE":              true,
		"TRUNCATECOLUMNS":   true,
		"UNION":             true,
		"UNIQUE":            true,
		"USER":              true,
		"USING":             true,
		"VERBOSE":           true,
		"WALLET":            true,
		"WHEN":              true,
		"WHERE":             true,
		"WITH":              true,
		"WITHIN":            true,
		"WITHOUT":           true,
	}
	ident             = regexp.MustCompile(`(?i)^[a-z_][a-z0-9_$]*$`)
	databaseHeaderFmt = "" +
		"--\n" +
		"-- PostgreSQL database structure for %s\n" +
		"--\n"
	useDatabaseFmt             = "\\connect %s;\n\n"
	asToken                    = regexp.MustCompile("(AS )[$]([a-z]+)[$]")
	bytebaseDatabase           = "bytebase"
	createBytebaseDatabaseStmt = "CREATE DATABASE bytebase;"

	_ db.Driver = (*Driver)(nil)
)

func init() {
	db.Register(db.Postgres, newDriver)
}

type Driver struct {
	l             *zap.Logger
	connectionCtx db.ConnectionContext

	db      *sql.DB
	baseDSN string
}

func newDriver(config db.DriverConfig) db.Driver {
	return &Driver{
		l: config.Logger,
	}
}

func (driver *Driver) Open(dbType db.Type, config db.ConnectionConfig, ctx db.ConnectionContext) (db.Driver, error) {
	if (config.TlsConfig.SslCert == "" && config.TlsConfig.SslKey != "") ||
		(config.TlsConfig.SslCert != "" && config.TlsConfig.SslKey == "") {
		return nil, fmt.Errorf("ssl-cert and ssl-key must be both set or unset")
	}

	dsn, err := guessDSN(
		config.Username,
		config.Password,
		config.Host,
		config.Port,
		config.Database,
		config.TlsConfig.SslCA,
		config.TlsConfig.SslCert,
		config.TlsConfig.SslKey,
	)
	if err != nil {
		return nil, err
	}

	// db is closed in the dumper closer.
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	driver.db = db
	driver.baseDSN = dsn
	driver.connectionCtx = ctx

	return driver, nil
}

// guessDSN will guess the dsn of a valid DB connection.
func guessDSN(username, password, hostname, port, database, sslCA, sslCert, sslKey string) (string, error) {
	// dbname is guessed if not specified.
	m := map[string]string{
		"host":     hostname,
		"port":     port,
		"user":     username,
		"password": password,
	}

	if sslCA == "" {
		m["sslmode"] = "disable"
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
		guesses = append(guesses, dsn+" dbname="+database)
	} else {
		// Guess default database postgres, template1.
		guesses = append(guesses, dsn)
		guesses = append(guesses, dsn+" dbname=postgres")
		guesses = append(guesses, dsn+" dbname=template1")
	}

	for _, dsn := range guesses {
		db, err := sql.Open("postgres", dsn)
		if err != nil {
			continue
		}
		defer db.Close()

		if err = db.Ping(); err != nil {
			continue
		}
		return dsn, nil
	}
	return "", fmt.Errorf("cannot find valid dsn for connecting %q, make sure the database exists", database)
}

func (driver *Driver) Close(ctx context.Context) error {
	return driver.db.Close()
}

func (driver *Driver) Ping(ctx context.Context) error {
	return driver.db.PingContext(ctx)
}

func (driver *Driver) SyncSchema(ctx context.Context) ([]*db.DBUser, []*db.DBSchema, error) {
	return nil, nil, fmt.Errorf("not implemented")
}

func (driver *Driver) Execute(ctx context.Context, statement string) error {
	tx, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, statement)
	return err
}

// Migration related
func (driver *Driver) NeedsSetupMigration(ctx context.Context) (bool, error) {
	exist, err := driver.hasBytebaseDatabase(ctx)
	if err != nil {
		return false, err
	}
	if !exist {
		return true, nil
	}
	if err := driver.switchDatabase(bytebaseDatabase); err != nil {
		return false, err
	}

	const query = `
		SELECT 
		    1
		FROM information_schema.tables
		WHERE table_name = 'migration_history'
	`
	return util.NeedsSetupMigrationSchema(ctx, driver.db, query)
}

func (driver *Driver) hasBytebaseDatabase(ctx context.Context) (bool, error) {
	dbNames, err := driver.getDatabases()
	if err != nil {
		return false, err
	}
	exist := false
	for _, dbName := range dbNames {
		if dbName == bytebaseDatabase {
			exist = true
			break
		}
	}
	return exist, nil
}

func (driver *Driver) SetupMigrationIfNeeded(ctx context.Context) error {
	setup, err := driver.NeedsSetupMigration(ctx)
	if err != nil {
		return nil
	}

	if setup {
		driver.l.Info("Bytebase migration schema not found, creating schema...",
			zap.String("environment", driver.connectionCtx.EnvironmentName),
			zap.String("database", driver.connectionCtx.InstanceName),
		)

		exist, err := driver.hasBytebaseDatabase(ctx)
		if err != nil {
			driver.l.Error("Failed to find bytebase database.",
				zap.Error(err),
				zap.String("environment", driver.connectionCtx.EnvironmentName),
				zap.String("database", driver.connectionCtx.InstanceName),
			)
			return fmt.Errorf("failed to find bytebase database error: %v", err)
		}

		if !exist {
			if _, err := driver.db.ExecContext(ctx, createBytebaseDatabaseStmt); err != nil {
				driver.l.Error("Failed to create bytebase database.",
					zap.Error(err),
					zap.String("environment", driver.connectionCtx.EnvironmentName),
					zap.String("database", driver.connectionCtx.InstanceName),
				)
				return util.FormatErrorWithQuery(err, createBytebaseDatabaseStmt)
			}
		}

		if err := driver.switchDatabase(bytebaseDatabase); err != nil {
			driver.l.Error("Failed to switch to bytebase database.",
				zap.Error(err),
				zap.String("environment", driver.connectionCtx.EnvironmentName),
				zap.String("database", driver.connectionCtx.InstanceName),
			)
			return fmt.Errorf("failed to switch to bytebase database error: %v", err)
		}

		if _, err := driver.db.ExecContext(ctx, migrationSchema); err != nil {
			driver.l.Error("Failed to initialize migration schema.",
				zap.Error(err),
				zap.String("environment", driver.connectionCtx.EnvironmentName),
				zap.String("database", driver.connectionCtx.InstanceName),
			)
			return util.FormatErrorWithQuery(err, migrationSchema)
		}
		driver.l.Info("Successfully created migration schema.",
			zap.String("environment", driver.connectionCtx.EnvironmentName),
			zap.String("database", driver.connectionCtx.InstanceName),
		)
	}

	return nil
}

func (driver *Driver) ExecuteMigration(ctx context.Context, m *db.MigrationInfo, statement string) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (driver *Driver) FindMigrationHistoryList(ctx context.Context, find *db.MigrationHistoryFind) ([]*db.MigrationHistory, error) {
	return nil, fmt.Errorf("not implemented")
}

// Dump and restore
func (driver *Driver) Dump(ctx context.Context, database string, out io.Writer, schemaOnly bool) error {
	// pg_dump -d dbName --schema-only+

	// Find all dumpable databases
	dbNames, err := driver.getDatabases()
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
			return fmt.Errorf("database %s not found", database)
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
		includeUseDatabase := len(dumpableDbNames) > 1
		if err := driver.dumpOneDatabase(ctx, dbName, out, schemaOnly, includeUseDatabase); err != nil {
			return err
		}
	}

	return nil
}

func (driver *Driver) Restore(ctx context.Context, sc *bufio.Scanner) (err error) {
	txn, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer txn.Rollback()

	s := ""
	tokenName := ""
	for sc.Scan() {
		line := sc.Text()

		execute := false

		switch {
		case s == "" && line == "":
			continue
		case strings.HasPrefix(line, "--"):
			continue
		case tokenName != "":
			if strings.Contains(line, tokenName) {
				tokenName = ""
			}
		default:
			token := asToken.FindString(line)
			if token != "" {
				identifier := token[3:]
				rest := line[strings.Index(line, identifier)+len(identifier):]
				if !strings.Contains(rest, identifier) {
					tokenName = identifier
				}
			}
		}
		s = s + line + "\n"
		if strings.HasSuffix(line, ";") && tokenName == "" {
			execute = true
		}

		if execute {
			_, err := txn.Exec(s)
			if err != nil {
				return fmt.Errorf("execute query %q failed: %v", s, err)
			}
			s = ""
		}
	}
	if err := sc.Err(); err != nil {
		return err
	}

	if err := txn.Commit(); err != nil {
		return err
	}

	return nil
}

func (driver *Driver) dumpOneDatabase(ctx context.Context, database string, out io.Writer, schemaOnly bool, includeUseDatabase bool) error {
	if err := driver.switchDatabase(database); err != nil {
		return err
	}

	txn, err := driver.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return err
	}
	defer txn.Rollback()

	// Database header.
	header := fmt.Sprintf(databaseHeaderFmt, database)
	if _, err := io.WriteString(out, header); err != nil {
		return err
	}
	// Database statement.
	if includeUseDatabase {
		// Use database statement.
		useStmt := fmt.Sprintf(useDatabaseFmt, database)
		if _, err := io.WriteString(out, useStmt); err != nil {
			return err
		}
	}

	// Schema statements.
	schemas, err := getPgSchemas(txn)
	if err != nil {
		return err
	}
	for _, schema := range schemas {
		if _, err := io.WriteString(out, schema.Statement()); err != nil {
			return err
		}
	}

	// Sequence statements.
	seqs, err := getSequences(txn)
	if err != nil {
		return fmt.Errorf("failed to get sequences from database %q: %s", database, err)
	}
	for _, seq := range seqs {
		if _, err := io.WriteString(out, seq.Statement()); err != nil {
			return err
		}
	}

	// Table statements.
	tables, err := getPgTables(txn)
	if err != nil {
		return fmt.Errorf("failed to get tables from database %q: %s", database, err)
	}

	constraints := make(map[string]bool)
	for _, tbl := range tables {
		if _, err := io.WriteString(out, tbl.Statement()); err != nil {
			return err
		}
		for _, constraint := range tbl.constraints {
			key := fmt.Sprintf("%s.%s.%s", constraint.schemaName, constraint.tableName, constraint.name)
			constraints[key] = true
		}
		if !schemaOnly {
			if err := exportTableData(txn, tbl, out); err != nil {
				return err
			}
		}
	}

	// View statements.
	views, err := getViews(txn)
	if err != nil {
		return fmt.Errorf("failed to get views from database %q: %s", database, err)
	}
	for _, view := range views {
		if _, err := io.WriteString(out, view.Statement()); err != nil {
			return err
		}
	}

	// Index statements.
	indices, err := getIndices(txn)
	if err != nil {
		return fmt.Errorf("failed to get indices from database %q: %s", database, err)
	}
	for _, idx := range indices {
		key := fmt.Sprintf("%s.%s.%s", idx.schemaName, idx.tableName, idx.name)
		if constraints[key] {
			continue
		}
		if _, err := io.WriteString(out, idx.Statement()); err != nil {
			return err
		}
	}

	// Function statements.
	fs, err := getFunctions(txn)
	if err != nil {
		return fmt.Errorf("failed to get functions from database %q: %s", database, err)
	}
	for _, f := range fs {
		if _, err := io.WriteString(out, f.Statement()); err != nil {
			return err
		}
	}

	// Trigger statements.
	triggers, err := getTriggers(txn)
	if err != nil {
		return fmt.Errorf("failed to get triggers from database %q: %s", database, err)
	}
	for _, tr := range triggers {
		if _, err := io.WriteString(out, tr.Statement()); err != nil {
			return err
		}
	}

	// Event statements.
	events, err := getEventTriggers(txn)
	if err != nil {
		return fmt.Errorf("failed to get event triggers from database %q: %s", database, err)
	}
	for _, evt := range events {
		if _, err := io.WriteString(out, evt.Statement()); err != nil {
			return err
		}
	}

	if err := txn.Commit(); err != nil {
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

	dns := driver.baseDSN + " dbname=" + dbName
	db, err := sql.Open("postgres", dns)
	if err != nil {
		return err
	}
	driver.db = db
	return nil
}

// getDatabases gets all databases of an instance.
func (driver *Driver) getDatabases() ([]string, error) {
	var dbNames []string
	rows, err := driver.db.Query("SELECT datname FROM pg_database;")
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

// pgSchema describes a pg schema, a namespace for all schemas.
type pgSchema struct {
	name        string
	schemaOwner string
}

// tableSchema describes the schema of a pg table.
type tableSchema struct {
	schemaName  string
	name        string
	tableowner  string
	columns     []*columnSchema
	constraints []*tableConstraint
}

// columnSchema describes the schema of a pg table column.
type columnSchema struct {
	columnName             string
	dataType               string
	characterMaximumLength string
	columnDefault          string
	isNullable             bool
}

// tableConstraint describes constraint schema of a pg table.
type tableConstraint struct {
	name       string
	schemaName string
	tableName  string
	constraint string
}

// viewSchema describes the schema of a pg view.
type viewSchema struct {
	schemaName string
	name       string
	statement  string
}

// indexSchema describes the schema of a pg index.
type indexSchema struct {
	schemaName string
	name       string
	tableName  string
	statement  string
}

// sequencePgSchema describes the schema of a pg sequence.
type sequencePgSchema struct {
	schemaName   string
	name         string
	dataType     string
	startValue   string
	increment    string
	minimumValue string
	maximumValue string
	cycleOption  string
	cache        string
}

// functionSchema describes the schema of a pg function.
type functionSchema struct {
	schemaName string
	name       string
	statement  string
	language   string
	arguments  string
}

// triggerSchema describes the schema of a pg trigger.
type triggerSchema struct {
	name      string
	statement string
}

// eventTriggerSchema describes the schema of a pg event trigger.
type eventTriggerSchema struct {
	name     string
	enabled  string
	event    string
	owner    string
	tags     string
	funcName string
}

// Statement returns the create statement of a Postgres schema.
func (ps *pgSchema) Statement() string {
	return fmt.Sprintf(""+
		"--\n"+
		"-- Schema structure for %s\n"+
		"--\n"+
		"CREATE SCHEMA %s;\n\n", ps.name, ps.name)
}

// Statement returns the create statement of a table.
func (t *tableSchema) Statement() string {
	s := fmt.Sprintf(""+
		"--\n"+
		"-- Table structure for %s.%s\n"+
		"--\n"+
		"CREATE TABLE %s.%s (\n",
		t.schemaName, t.name, t.schemaName, t.name)
	var cols []string
	for _, v := range t.columns {
		cols = append(cols, "  "+v.Statement())
	}
	s += strings.Join(cols, ",\n")
	s += "\n);\n\n"

	// Add constraints such as primary key, unique, or checks.
	for _, constraint := range t.constraints {
		s += fmt.Sprintf("%s\n", constraint.Statement())
	}
	s += "\n"
	return s
}

// Statement returns the statement of a table column.
func (c *columnSchema) Statement() string {
	s := fmt.Sprintf("%s %s", c.columnName, c.dataType)
	if c.characterMaximumLength != "" {
		s += fmt.Sprintf("(%s)", c.characterMaximumLength)
	}
	if !c.isNullable {
		s = s + " NOT NULL"
	}
	if c.columnDefault != "" {
		s += fmt.Sprintf(" DEFAULT %s", c.columnDefault)
	}
	return s
}

// Statement returns the create statement of a table constraint.
func (c *tableConstraint) Statement() string {
	return fmt.Sprintf(""+
		"ALTER TABLE ONLY %s.%s\n"+
		"    ADD CONSTRAINT %s %s;\n",
		c.schemaName, c.tableName, c.name, c.constraint)
}

// Statement returns the create statement of a view.
func (v *viewSchema) Statement() string {
	return fmt.Sprintf(""+
		"--\n"+
		"-- View structure for %s.%s\n"+
		"--\n"+
		"CREATE VIEW %s.%s AS\n%s\n\n",
		v.schemaName, v.name, v.schemaName, v.name, v.statement)
}

// Statement returns the create statement of a sequence.
func (seq *sequencePgSchema) Statement() string {
	s := fmt.Sprintf(""+
		"--\n"+
		"-- Sequence structure for %s.%s\n"+
		"--\n"+
		"CREATE SEQUENCE %s.%s\n"+
		"    AS %s\n"+
		"    START WITH %s\n"+
		"    INCREMENT BY %s\n",
		seq.schemaName, seq.name, seq.schemaName, seq.name, seq.dataType, seq.startValue, seq.increment)
	if seq.minimumValue == "" {
		s += fmt.Sprintf("    MINVALUE %s\n", seq.minimumValue)
	} else {
		s += fmt.Sprintf("    NO MINVALUE\n")
	}
	if seq.maximumValue == "" {
		s += fmt.Sprintf("    MAXVALUE %s\n", seq.maximumValue)
	} else {
		s += fmt.Sprintf("    NO MAXVALUE\n")
	}
	s += fmt.Sprintf("    CACHE %s", seq.cache)
	switch seq.cycleOption {
	case "YES":
		s += "\n    CYCLE;\n"
	case "NO":
		s += ";\n"
	}
	s += "\n"
	return s
}

// Statement returns the create statement of an index.
func (idx indexSchema) Statement() string {
	return fmt.Sprintf(""+
		"--\n"+
		"-- Index structure for %s.%s\n"+
		"--\n"+
		"%s;\n\n",
		idx.schemaName, idx.name, idx.statement)
}

// Statement returns the create statement of a function.
func (f functionSchema) Statement() string {
	return fmt.Sprintf(""+
		"--\n"+
		"-- Function structure for %s.%s\n"+
		"--\n"+
		"%s;\n\n",
		f.schemaName, f.name, f.statement)
}

// Statement returns the create statement of a trigger.
func (t triggerSchema) Statement() string {
	return fmt.Sprintf(""+
		"--\n"+
		"-- Trigger structure for %s\n"+
		"--\n"+
		"%s;\n\n",
		t.name, t.statement)
}

// Statement returns the create statement of an event trigger.
func (t eventTriggerSchema) Statement() string {
	s := fmt.Sprintf(""+
		"--\n"+
		"-- Event trigger structure for %s\n"+
		"--\n",
		t.name)
	s += fmt.Sprintf("CREATE EVENT TRIGGER %s ON %s", t.name, t.event)
	if t.tags != "" {
		s += fmt.Sprintf("\n         WHEN TAG IN (%s)", t.tags)
	}
	s += fmt.Sprintf("\n   EXECUTE FUNCTION %s();\n", t.funcName)

	if t.enabled != "O" {
		s += fmt.Sprintf("ALTER EVENT TRIGGER %s ", t.name)
		switch t.enabled {
		case "D":
			s += "DISABLE;\n"
		case "A":
			s += "ENABLE ALWAYS;\n"
		case "R":
			s += "ENABLE REPLICA;\n"
		default:
			s += "ENABLE;\n"
		}
	}
	s += "\n"
	return s
}

// getPgSchemas gets all schemas of a database.
func getPgSchemas(txn *sql.Tx) ([]*pgSchema, error) {
	var schemas []*pgSchema
	rows, err := txn.Query("SELECT schema_name, schema_owner FROM information_schema.schemata;")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var schema pgSchema
		if err := rows.Scan(&schema.name, &schema.schemaOwner); err != nil {
			return nil, err
		}
		schema.name = quoteIdentifier(schema.name)
		if ok := pgSystemSchema(schema.name); ok {
			continue
		}
		schemas = append(schemas, &schema)
	}
	return schemas, nil
}

// pgSystemSchema returns whether the schema is a system or user defined schema.
func pgSystemSchema(s string) bool {
	if strings.HasPrefix(s, "pg_toast") {
		return true
	}
	if strings.HasPrefix(s, "pg_temp") {
		return true
	}
	switch s {
	case "pg_catalog":
		return true
	case "public":
		return true
	case "information_schema":
		return true
	}
	return false
}

// getTables gets all tables of a database.
func getPgTables(txn *sql.Tx) ([]*tableSchema, error) {
	constraints, err := getTableConstraints(txn)
	if err != nil {
		return nil, fmt.Errorf("getTableConstraints() got error: %v", err)
	}

	var tables []*tableSchema
	query := "" +
		"SELECT * FROM pg_catalog.pg_tables " +
		"WHERE schemaname NOT IN ('pg_catalog', 'information_schema');"
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var tbl tableSchema
		var schemaname, tablename, tableowner string
		var tablespace sql.NullString
		var hasindexes, hasrules, hastriggers, rowsecurity bool
		if err := rows.Scan(&schemaname, &tablename, &tableowner, &tablespace, &hasindexes, &hasrules, &hastriggers, &rowsecurity); err != nil {
			return nil, err
		}
		tbl.schemaName = quoteIdentifier(schemaname)
		tbl.name = quoteIdentifier(tablename)
		tbl.tableowner = tableowner

		tables = append(tables, &tbl)
	}

	for _, tbl := range tables {
		columns, err := getTableColumns(txn, tbl.schemaName, tbl.name)
		if err != nil {
			return nil, fmt.Errorf("getTable(%q, %q) got error %v", tbl.schemaName, tbl.name, err)
		}
		tbl.columns = columns

		key := fmt.Sprintf("%s.%s", tbl.schemaName, tbl.name)
		v, _ := constraints[key]
		tbl.constraints = v
	}
	return tables, nil
}

// getTableColumns gets the columns of a table.
func getTableColumns(txn *sql.Tx, schemaName, tableName string) ([]*columnSchema, error) {
	query := "" +
		"SELECT column_name, data_type, character_maximum_length, column_default, is_nullable " +
		"FROM INFORMATION_SCHEMA.COLUMNS " +
		"WHERE table_schema=$1 AND table_name=$2;"
	rows, err := txn.Query(query, schemaName, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []*columnSchema
	for rows.Next() {
		var columnName, dataType, isNullable string
		var characterMaximumLength, columnDefault sql.NullString
		if err := rows.Scan(&columnName, &dataType, &characterMaximumLength, &columnDefault, &isNullable); err != nil {
			return nil, err
		}
		isNullBool, err := convertBoolFromYesNo(isNullable)
		if err != nil {
			return nil, err
		}
		c := columnSchema{
			columnName:             columnName,
			dataType:               dataType,
			characterMaximumLength: characterMaximumLength.String,
			columnDefault:          columnDefault.String,
			isNullable:             isNullBool,
		}
		columns = append(columns, &c)
	}
	return columns, nil
}

func convertBoolFromYesNo(s string) (bool, error) {
	switch s {
	case "YES":
		return true, nil
	case "NO":
		return false, nil
	default:
		return false, fmt.Errorf("unrecognized isNullable type %q", s)
	}
}

// getTableConstraints gets all table constraints of a database.
func getTableConstraints(txn *sql.Tx) (map[string][]*tableConstraint, error) {
	query := "" +
		"SELECT n.nspname, conrelid::regclass, conname, pg_get_constraintdef(c.oid) " +
		"FROM pg_constraint c " +
		"JOIN pg_namespace n ON n.oid = c.connamespace " +
		"WHERE n.nspname NOT IN ('pg_catalog', 'information_schema');"
	ret := make(map[string][]*tableConstraint)
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var constraint tableConstraint
		if err := rows.Scan(&constraint.schemaName, &constraint.tableName, &constraint.name, &constraint.constraint); err != nil {
			return nil, err
		}
		if strings.Contains(constraint.tableName, ".") {
			constraint.tableName = constraint.tableName[1+strings.Index(constraint.tableName, "."):]
		}
		constraint.schemaName, constraint.tableName, constraint.name = quoteIdentifier(constraint.schemaName), quoteIdentifier(constraint.tableName), quoteIdentifier(constraint.name)
		key := fmt.Sprintf("%s.%s", constraint.schemaName, constraint.tableName)
		v, _ := ret[key]
		ret[key] = append(v, &constraint)
	}
	return ret, nil
}

// getViews gets all views of a database.
func getViews(txn *sql.Tx) ([]*viewSchema, error) {
	query := "" +
		"SELECT table_schema, table_name FROM information_schema.views " +
		"WHERE table_schema NOT IN ('pg_catalog', 'information_schema');"
	var views []*viewSchema
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var view viewSchema
		if err := rows.Scan(&view.schemaName, &view.name); err != nil {
			return nil, err
		}
		view.schemaName, view.name = quoteIdentifier(view.schemaName), quoteIdentifier(view.name)
		views = append(views, &view)
	}

	for _, view := range views {
		if err = getView(txn, view); err != nil {
			return nil, fmt.Errorf("getPgView(%q, %q) got error %v", view.schemaName, view.name, err)
		}
	}
	return views, nil
}

// getView gets the schema of a view.
func getView(txn *sql.Tx, view *viewSchema) error {
	query := fmt.Sprintf("SELECT pg_get_viewdef('%s.%s', true);", view.schemaName, view.name)
	rows, err := txn.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		if err := rows.Scan(&view.statement); err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("query %q returned multiple rows", query)
}

// getIndices gets all indices of a database.
func getIndices(txn *sql.Tx) ([]*indexSchema, error) {
	query := "" +
		"SELECT schemaname, tablename, indexname, indexdef " +
		"FROM pg_indexes WHERE schemaname NOT IN ('pg_catalog', 'information_schema');"

	var indices []*indexSchema
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var idx indexSchema
		if err := rows.Scan(&idx.schemaName, &idx.tableName, &idx.name, &idx.statement); err != nil {
			return nil, err
		}
		idx.schemaName, idx.tableName, idx.name = quoteIdentifier(idx.schemaName), quoteIdentifier(idx.tableName), quoteIdentifier(idx.name)
		indices = append(indices, &idx)
	}

	return indices, nil
}

// exportTableData gets the data of a table.
func exportTableData(txn *sql.Tx, tbl *tableSchema, out io.Writer) error {
	query := fmt.Sprintf("SELECT * FROM %s.%s;", tbl.schemaName, tbl.name)
	rows, err := txn.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	cols, err := rows.ColumnTypes()
	if err != nil {
		return err
	}
	if len(cols) <= 0 {
		return nil
	}
	values := make([]*sql.NullString, len(cols))
	ptrs := make([]interface{}, len(cols))
	for i := 0; i < len(cols); i++ {
		ptrs[i] = &values[i]
	}
	for rows.Next() {
		if err := rows.Scan(ptrs...); err != nil {
			return err
		}
		tokens := make([]string, len(cols))
		for i, v := range values {
			switch {
			case v == nil || !v.Valid:
				tokens[i] = "NULL"
			case isNumeric(cols[i].ScanType().Name()):
				tokens[i] = v.String
			default:
				tokens[i] = fmt.Sprintf("'%s'", v.String)
			}
		}
		stmt := fmt.Sprintf("INSERT INTO %s.%s VALUES (%s);\n", tbl.schemaName, tbl.name, strings.Join(tokens, ", "))
		if _, err := io.WriteString(out, stmt); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(out, "\n"); err != nil {
		return err
	}
	return nil
}

// isNumeric determines whether the value needs quotes.
// Even if the function returns incorrect result, the data dump will still work.
func isNumeric(t string) bool {
	return strings.Contains(t, "int") || strings.Contains(t, "bool") || strings.Contains(t, "float") || strings.Contains(t, "byte")
}

// getSequences gets all sequences of a database.
func getSequences(txn *sql.Tx) ([]*sequencePgSchema, error) {
	caches := make(map[string]string)
	query := "SELECT seqclass.relnamespace::regnamespace::text, seqclass.relname, seq.seqcache " +
		"FROM pg_catalog.pg_class AS seqclass " +
		"JOIN pg_catalog.pg_sequence AS seq ON (seq.seqrelid = seqclass.relfilenode);"
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var schemaName, seqName, cache string
		if err := rows.Scan(&schemaName, &seqName, &cache); err != nil {
			return nil, err
		}
		schemaName, seqName = quoteIdentifier(schemaName), quoteIdentifier(seqName)
		caches[fmt.Sprintf("%s.%s", schemaName, seqName)] = cache
	}

	var seqs []*sequencePgSchema
	query = "" +
		"SELECT sequence_schema, sequence_name, data_type, start_value, increment, minimum_value, maximum_value, cycle_option " +
		"FROM information_schema.sequences;"
	rows, err = txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var seq sequencePgSchema

		if err := rows.Scan(&seq.schemaName, &seq.name, &seq.dataType, &seq.startValue, &seq.increment, &seq.minimumValue, &seq.maximumValue, &seq.cycleOption); err != nil {
			return nil, err
		}
		seq.schemaName, seq.name = quoteIdentifier(seq.schemaName), quoteIdentifier(seq.name)
		cache, ok := caches[fmt.Sprintf("%s.%s", seq.schemaName, seq.name)]
		if !ok {
			return nil, fmt.Errorf("cannot find cache value for sequence: %q.%q", seq.schemaName, seq.name)
		}
		seq.cache = cache
		seqs = append(seqs, &seq)
	}

	return seqs, nil
}

// getFunctions gets all functions of a database.
func getFunctions(txn *sql.Tx) ([]*functionSchema, error) {
	query := "" +
		"SELECT n.nspname, p.proname, l.lanname, " +
		"  CASE WHEN l.lanname = 'internal' THEN p.prosrc ELSE pg_get_functiondef(p.oid) END as definition, " +
		"  pg_get_function_arguments(p.oid) " +
		"FROM pg_proc p " +
		"LEFT JOIN pg_namespace n ON p.pronamespace = n.oid " +
		"LEFT JOIN pg_language l ON p.prolang = l.oid " +
		"LEFT JOIN pg_type t ON t.oid = p.prorettype " +
		"WHERE n.nspname NOT IN ('pg_catalog', 'information_schema');"

	var fs []*functionSchema
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var f functionSchema
		if err := rows.Scan(&f.schemaName, &f.name, &f.language, &f.statement, &f.arguments); err != nil {
			return nil, err
		}
		f.schemaName, f.name = quoteIdentifier(f.schemaName), quoteIdentifier(f.name)
		fs = append(fs, &f)
	}

	return fs, nil
}

// getTriggers gets all triggers of a database.
func getTriggers(txn *sql.Tx) ([]*triggerSchema, error) {
	query := "SELECT tgname, pg_get_triggerdef(t.oid) FROM pg_trigger AS t;"

	var triggers []*triggerSchema
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var t triggerSchema
		if err := rows.Scan(&t.name, &t.statement); err != nil {
			return nil, err
		}
		t.name = quoteIdentifier(t.name)
		triggers = append(triggers, &t)
	}

	return triggers, nil
}

// getEventTriggers gets all event triggers of a database.
func getEventTriggers(txn *sql.Tx) ([]*eventTriggerSchema, error) {
	query := "" +
		"SELECT evtname, evtenabled, evtevent, pg_get_userbyid(evtowner) AS evtowner, " +
		"  array_to_string(array(SELECT quote_literal(x) FROM unnest(evttags) as t(x)), ', ') AS evttags, " +
		"  e.evtfoid::regproc " +
		"FROM pg_event_trigger e;"

	var triggers []*eventTriggerSchema
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var t eventTriggerSchema
		if err := rows.Scan(&t.name, &t.enabled, &t.event, &t.owner, &t.tags, &t.funcName); err != nil {
			return nil, err
		}
		t.name = quoteIdentifier(t.name)
		triggers = append(triggers, &t)
	}

	return triggers, nil
}

// quoteIdentifier will quote identifiers including keywords, capital charactors, or special charactors.
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
	} else {
		return s
	}
}
