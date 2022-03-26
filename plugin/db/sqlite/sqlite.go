package sqlite

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"io"
	"io/ioutil"
	"path"
	"strings"

	// embed will embeds the migration schema.
	_ "embed"

	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/db/util"
	"go.uber.org/zap"
)

//go:embed sqlite_migration_schema.sql
var migrationSchema string

var (
	bytebaseDatabase     = "bytebase"
	excludedDatabaseList = map[string]bool{
		// Skip our internal "bytebase" database
		bytebaseDatabase: true,
	}

	_ db.Driver              = (*Driver)(nil)
	_ util.MigrationExecutor = (*Driver)(nil)
)

func init() {
	db.Register(db.SQLite, newDriver)
}

// Driver is the SQLite driver.
type Driver struct {
	dir           string
	db            *sql.DB
	connectionCtx db.ConnectionContext
	l             *zap.Logger
}

func newDriver(config db.DriverConfig) db.Driver {
	return &Driver{
		l: config.Logger,
	}
}

// Open opens a SQLite driver.
func (driver *Driver) Open(ctx context.Context, dbType db.Type, config db.ConnectionConfig, connCtx db.ConnectionContext) (db.Driver, error) {
	// Host is the directory (instance) containing all SQLite databases.
	driver.dir = config.Host

	// If config.Database is empty, we will get a connection to in-memory database.
	if _, err := driver.GetDbConnection(ctx, config.Database); err != nil {
		return nil, err
	}
	driver.connectionCtx = connCtx
	return driver, nil
}

// Close closes the driver.
func (driver *Driver) Close(ctx context.Context) error {
	if driver.db != nil {
		return driver.db.Close()
	}
	return nil
}

// Ping pings the database.
func (driver *Driver) Ping(ctx context.Context) error {
	return driver.db.PingContext(ctx)
}

// GetDbConnection gets a database connection.
// If database is empty, we will get a connect to in-memory database.
func (driver *Driver) GetDbConnection(ctx context.Context, database string) (*sql.DB, error) {
	if driver.db != nil {
		if err := driver.db.Close(); err != nil {
			return nil, err
		}
	}

	dns := path.Join(driver.dir, fmt.Sprintf("%s.db", database))
	if database == "" {
		dns = ":memory:"
	}
	db, err := sql.Open("sqlite3", dns)
	if err != nil {
		return nil, err
	}
	driver.db = db
	return db, nil
}

// GetVersion gets the version.
func (driver *Driver) GetVersion(ctx context.Context) (string, error) {
	var version string
	row := driver.db.QueryRowContext(ctx, "SELECT sqlite_version();")
	if err := row.Scan(&version); err != nil {
		return "", err
	}
	return version, nil
}

// SyncSchema synces the schema.
func (driver *Driver) SyncSchema(ctx context.Context) ([]*db.User, []*db.Schema, error) {
	databases, err := driver.getDatabases()
	if err != nil {
		return nil, nil, err
	}

	var schemaList []*db.Schema
	for _, dbName := range databases {
		if _, ok := excludedDatabaseList[dbName]; ok {
			continue
		}

		var schema db.Schema
		schema.Name = dbName

		sqldb, err := driver.GetDbConnection(ctx, dbName)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get database connection for %q: %s", dbName, err)
		}
		txn, err := sqldb.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
		if err != nil {
			return nil, nil, err
		}
		defer txn.Rollback()
		// Index statements.
		indicesMap := make(map[string][]indexSchema)
		indices, err := getIndices(txn)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get indices from database %q: %s", dbName, err)
		}
		for _, idx := range indices {
			indicesMap[idx.tableName] = append(indicesMap[idx.tableName], idx)
		}

		tbls, err := getTables(txn, indicesMap)
		if err != nil {
			return nil, nil, err
		}
		schema.TableList = tbls

		views, err := getViews(txn)
		if err != nil {
			return nil, nil, err
		}
		schema.ViewList = views

		if err := txn.Commit(); err != nil {
			return nil, nil, err
		}

		schemaList = append(schemaList, &schema)
	}
	return nil, schemaList, nil
}

// getTables gets all tables of a database.
func getTables(txn *sql.Tx, indicesMap map[string][]indexSchema) ([]db.Table, error) {
	var tables []db.Table
	query := "SELECT name FROM sqlite_schema WHERE type ='table' AND name NOT LIKE 'sqlite_%';"
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tableNames []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tableNames = append(tableNames, name)
	}
	for _, name := range tableNames {
		var tbl db.Table
		tbl.Name = name
		tbl.Type = "BASE TABLE"

		// Get columns: cid, name, type, notnull, dflt_value, pk.
		query := fmt.Sprintf("pragma table_info(%s);", name)
		rows, err := txn.Query(query)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			var col db.Column

			var cid int
			var notnull, pk bool
			var name, ctype string
			var dfltValue sql.NullString
			if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
				return nil, err
			}
			col.Position = cid
			col.Name = name
			col.Nullable = !notnull
			col.Type = ctype
			if dfltValue.Valid {
				col.Default = &dfltValue.String
			}

			tbl.ColumnList = append(tbl.ColumnList, col)
		}
		for _, idx := range indicesMap[tbl.Name] {
			query := fmt.Sprintf("pragma index_info(%s);", idx.name)
			rows, err := txn.Query(query)
			if err != nil {
				return nil, err
			}
			defer rows.Close()
			for rows.Next() {
				var dbIdx db.Index
				dbIdx.Name = idx.name
				dbIdx.Unique = idx.unique
				var cid string
				if err := rows.Scan(&dbIdx.Position, &cid, &dbIdx.Expression); err != nil {
					return nil, err
				}
				tbl.IndexList = append(tbl.IndexList, dbIdx)
			}
		}

		tables = append(tables, tbl)
	}
	return tables, nil
}

// getIndices gets all indices of a database.
func getIndices(txn *sql.Tx) ([]indexSchema, error) {
	var indices []indexSchema
	query := "SELECT name, tbl_name, sql FROM sqlite_schema WHERE type ='index' AND name NOT LIKE 'sqlite_%';"
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var idx indexSchema
		if err := rows.Scan(&idx.name, &idx.tableName, &idx.statement); err != nil {
			return nil, err
		}
		idx.unique = strings.Contains(idx.statement, " UNIQUE INDEX ")
		indices = append(indices, idx)
	}
	return indices, nil
}

// indexSchema describes the schema of an index.
type indexSchema struct {
	name      string
	tableName string
	statement string
	unique    bool
}

func getViews(txn *sql.Tx) ([]db.View, error) {
	var views []db.View
	query := "SELECT name, sql FROM sqlite_schema WHERE type ='view' AND name NOT LIKE 'sqlite_%';"
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var view db.View
		if err := rows.Scan(&view.Name, &view.Definition); err != nil {
			return nil, err
		}
		views = append(views, view)
	}
	return views, nil
}

func (driver *Driver) getDatabases() ([]string, error) {
	files, err := ioutil.ReadDir(driver.dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %q, error %w", driver.dir, err)
	}
	var databases []string
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".db") {
			continue
		}
		databases = append(databases, strings.TrimRight(file.Name(), ".db"))
	}
	return databases, nil
}

func (driver *Driver) hasBytebaseDatabase() (bool, error) {
	databases, err := driver.getDatabases()
	if err != nil {
		return false, err
	}
	for _, database := range databases {
		if database == bytebaseDatabase {
			return true, nil
		}
	}
	return false, nil
}

// Execute executes a SQL statement.
func (driver *Driver) Execute(ctx context.Context, statement string) error {
	var remainingStmts []string
	f := func(stmt string) error {
		// This is a fake CREATE DATABASE statement. Engine driver will recognize it and establish a connection to create the database.
		stmt = strings.TrimLeft(stmt, " \t")
		if strings.HasPrefix(stmt, "CREATE DATABASE ") {
			parts := strings.Split(stmt, `'`)
			if len(parts) != 3 {
				return fmt.Errorf("invalid statement %q", stmt)
			}
			db, err := driver.GetDbConnection(ctx, parts[1])
			if err != nil {
				return err
			}
			// We need to query to persist the database file.
			if _, err := db.Query("SELECT 1;"); err != nil {
				return err
			}
		} else if strings.HasPrefix(stmt, "USE ") {
			// ignore this fake use database statement.
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

	if _, err = tx.ExecContext(ctx, strings.Join(remainingStmts, "\n")); err == nil {
		if err := tx.Commit(); err != nil {
			return err
		}
	}

	return err
}

// Query queries a SQL statement.
func (driver *Driver) Query(ctx context.Context, statement string, limit int) ([]interface{}, error) {
	return util.Query(ctx, driver.l, driver.db, statement, limit)
}

// NeedsSetupMigration returns whether it needs to setup migration.
func (driver *Driver) NeedsSetupMigration(ctx context.Context) (bool, error) {
	exist, err := driver.hasBytebaseDatabase()
	if err != nil {
		return false, err
	}
	if !exist {
		return true, nil
	}
	if _, err := driver.GetDbConnection(ctx, bytebaseDatabase); err != nil {
		return false, err
	}

	const query = `
		SELECT
		    1
		FROM sqlite_master
		WHERE type='table' AND name = 'bytebase_migration_history'
	`
	return util.NeedsSetupMigrationSchema(ctx, driver.db, query)
}

// SetupMigrationIfNeeded sets up migration if needed.
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

		if _, err := driver.GetDbConnection(ctx, bytebaseDatabase); err != nil {
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

// FindLatestSequence will return the latest sequence number.
func (Driver) FindLatestSequence(ctx context.Context, tx *sql.Tx, namespace string) (int, error) {
	const findLatestSequenceQuery = `
		SELECT MAX(sequence) FROM bytebase_migration_history
		WHERE namespace = ?
	`
	row, err := tx.QueryContext(ctx, findLatestSequenceQuery,
		namespace,
	)
	if err != nil {
		return -1, util.FormatErrorWithQuery(err, findLatestSequenceQuery)
	}
	defer row.Close()

	var sequence sql.NullInt32
	row.Next()
	if err := row.Scan(&sequence); err != nil {
		return -1, err
	}

	if !sequence.Valid {
		// Returns 0 if we haven't applied any migration for this namespace.
		return 0, nil
	}

	return int(sequence.Int32), nil
}

// InsertPendingHistory will insert the migration record with pending status and return the inserted ID.
func (Driver) InsertPendingHistory(ctx context.Context, tx *sql.Tx, sequence int, prevSchema string, m *db.MigrationInfo, statement string) (int64, error) {
	const insertHistoryQuery = `
	INSERT INTO bytebase_migration_history (
		created_by,
		created_ts,
		updated_by,
		updated_ts,
		release_version,
		namespace,
		sequence,
		source,
		type,
		status,
		version,
		description,
		statement,
		schema,
		schema_prev,
		execution_duration_ns,
		issue_id,
		payload
	)
	VALUES (?, strftime('%s', 'now'), ?, strftime('%s', 'now'), ?, ?, ?, ?,  ?, 'PENDING', ?, ?, ?, ?, ?, 0, ?, ?)
	`
	res, err := tx.ExecContext(ctx, insertHistoryQuery,
		m.Creator,
		m.Creator,
		m.ReleaseVersion,
		m.Namespace,
		sequence,
		m.Source,
		m.Type,
		m.Version,
		m.Description,
		statement,
		prevSchema,
		prevSchema,
		m.IssueID,
		m.Payload,
	)
	if err != nil {
		return int64(0), util.FormatErrorWithQuery(err, insertHistoryQuery)
	}

	insertedID, err := res.LastInsertId()
	if err != nil {
		return int64(0), util.FormatErrorWithQuery(err, insertHistoryQuery)
	}
	return insertedID, nil
}

// UpdateHistoryAsDone will update the migration record as done.
func (Driver) UpdateHistoryAsDone(ctx context.Context, tx *sql.Tx, migrationDurationNs int64, updatedSchema string, insertedID int64) error {
	const updateHistoryAsDoneQuery = `
	UPDATE
		bytebase_migration_history
	SET
		status = 'DONE',
		execution_duration_ns = ?,
		schema = ?
	WHERE id = ?
	`
	_, err := tx.ExecContext(ctx, updateHistoryAsDoneQuery, migrationDurationNs, updatedSchema, insertedID)
	return err
}

// UpdateHistoryAsFailed will update the migration record as failed.
func (Driver) UpdateHistoryAsFailed(ctx context.Context, tx *sql.Tx, migrationDurationNs int64, insertedID int64) error {
	const updateHistoryAsFailedQuery = `
	UPDATE
		bytebase_migration_history
	SET
		status = 'FAILED',
		execution_duration_ns = ?
	WHERE id = ?
	`
	_, err := tx.ExecContext(ctx, updateHistoryAsFailedQuery, migrationDurationNs, insertedID)
	return err
}

// ExecuteMigration will execute the migration.
func (driver *Driver) ExecuteMigration(ctx context.Context, m *db.MigrationInfo, statement string) (int64, string, error) {
	return util.ExecuteMigration(ctx, driver.l, driver, m, statement)
}

// FindMigrationHistoryList finds the migration history.
func (driver *Driver) FindMigrationHistoryList(ctx context.Context, find *db.MigrationHistoryFind) ([]*db.MigrationHistory, error) {
	baseQuery := `
	SELECT
		id,
		created_by,
		created_ts,
		updated_by,
		updated_ts,
		release_version,
		namespace,
		sequence,
		source,
		type,
		status,
		version,
		description,
		statement,
		schema,
		schema_prev,
		execution_duration_ns,
		issue_id,
		payload
		FROM bytebase_migration_history `
	paramNames, params := []string{}, []interface{}{}
	if v := find.ID; v != nil {
		paramNames, params = append(paramNames, "id"), append(params, *v)
	}
	if v := find.Database; v != nil {
		paramNames, params = append(paramNames, "namespace"), append(params, *v)
	}
	if v := find.Version; v != nil {
		paramNames, params = append(paramNames, "version"), append(params, *v)
	}
	if v := find.Source; v != nil {
		paramNames, params = append(paramNames, "source"), append(params, *v)
	}
	if v := find.Sequence; v != nil {
		paramNames, params = append(paramNames, "sequence"), append(params, *v)
	}
	var query = baseQuery +
		db.FormatParamNameInQuestionMark(paramNames) +
		`ORDER BY created_ts DESC`
	if v := find.Limit; v != nil {
		query += fmt.Sprintf(" LIMIT %d", *v)
	}
	return util.FindMigrationHistoryList(ctx, query, params, driver, find, baseQuery)
}

// Dump dumps the database.
func (driver *Driver) Dump(ctx context.Context, database string, out io.Writer, schemaOnly bool) error {
	if database == "" {
		return fmt.Errorf("SQLite can dump one database only at a time")
	}

	// Find all dumpable databases and make sure the existence of the database to be dumped.
	databases, err := driver.getDatabases()
	if err != nil {
		return fmt.Errorf("failed to get databases: %s", err)
	}
	exist := false
	for _, n := range databases {
		if n == database {
			exist = true
			break
		}
	}
	if !exist {
		return fmt.Errorf("database %s not found", database)
	}

	if err := driver.dumpOneDatabase(ctx, database, out, schemaOnly); err != nil {
		return err
	}

	return nil
}

type sqliteSchema struct {
	schemaType string
	name       string
	statement  string
}

func (driver *Driver) dumpOneDatabase(ctx context.Context, database string, out io.Writer, schemaOnly bool) error {
	if _, err := driver.GetDbConnection(ctx, database); err != nil {
		return err
	}

	txn, err := driver.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return err
	}
	defer txn.Rollback()

	// Get all schemas.
	query := "SELECT type, name, sql FROM sqlite_schema;"
	rows, err := txn.QueryContext(ctx, query)
	if err != nil {
		return util.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()

	var sqliteSchemas []sqliteSchema
	for rows.Next() {
		var s sqliteSchema
		if err := rows.Scan(
			&s.schemaType,
			&s.name,
			&s.statement,
		); err != nil {
			return err
		}
		sqliteSchemas = append(sqliteSchemas, s)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, s := range sqliteSchemas {
		// We should skip sqlite sequence table.
		if s.name == "sqlite_sequence" {
			continue
		}
		if _, err := io.WriteString(out, fmt.Sprintf("%s;\n", s.statement)); err != nil {
			return err
		}

		// Dump table data.
		if !schemaOnly && s.schemaType == "table" {
			if err := exportTableData(txn, s.name, out); err != nil {
				return err
			}
		}
	}

	if err := txn.Commit(); err != nil {
		return err
	}

	return nil
}

// exportTableData gets the data of a table.
func exportTableData(txn *sql.Tx, tblName string, out io.Writer) error {
	query := fmt.Sprintf("SELECT * FROM `%s`;", tblName)
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
	refs := make([]interface{}, len(cols))
	for i := 0; i < len(cols); i++ {
		refs[i] = &values[i]
	}
	for rows.Next() {
		if err := rows.Scan(refs...); err != nil {
			return err
		}
		tokens := make([]string, len(cols))
		for i, v := range values {
			switch {
			case v == nil || !v.Valid:
				tokens[i] = "NULL"
			default:
				tokens[i] = fmt.Sprintf("'%s'", v.String)
			}
		}
		stmt := fmt.Sprintf("INSERT INTO '%s' VALUES (%s);\n", tblName, strings.Join(tokens, ", "))
		if _, err := io.WriteString(out, stmt); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(out, "\n"); err != nil {
		return err
	}
	return nil
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
