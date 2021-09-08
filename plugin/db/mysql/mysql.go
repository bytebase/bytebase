package mysql

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/bytebase/bytebase/plugin/db"
	"github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
)

//go:embed mysql_migration_schema.sql
var migrationSchema string

var (
	systemDatabases = map[string]bool{
		"information_schema": true,
		"mysql":              true,
		"performance_schema": true,
		"sys":                true,
	}

	_ db.Driver = (*Driver)(nil)
)

func init() {
	db.Register(db.MySQL, newDriver)
}

type Driver struct {
	l             *zap.Logger
	connectionCtx db.ConnectionContext

	db *sql.DB
}

func newDriver(config db.DriverConfig) db.Driver {
	return &Driver{
		l: config.Logger,
	}
}

func (driver *Driver) Open(config db.ConnectionConfig, ctx db.ConnectionContext) (db.Driver, error) {
	protocol := "tcp"
	if strings.HasPrefix(config.Host, "/") {
		protocol = "unix"
	}

	params := []string{"multiStatements=true"}

	port := config.Port
	if port == "" {
		port = "3306"
	}

	tlsConfig, err := config.TlsConfig.GetSslConfig()

	if err != nil {
		return nil, fmt.Errorf("sql: tls config error: %v", err)
	}

	loggedDSN := fmt.Sprintf("%s:<<redacted password>>@%s(%s:%s)/%s?%s", config.Username, protocol, config.Host, port, config.Database, strings.Join(params, "&"))
	dsn := fmt.Sprintf("%s:%s@%s(%s:%s)/%s?%s", config.Username, config.Password, protocol, config.Host, port, config.Database, strings.Join(params, "&"))
	tlsKey := "db.mysql.tls"
	if tlsConfig != nil {
		if err := mysql.RegisterTLSConfig(tlsKey, tlsConfig); err != nil {
			return nil, fmt.Errorf("sql: failed to register tls config: %v", err)
		}
		// TLS config is only used during sql.Open, so should be safe to deregister afterwards.
		defer mysql.DeregisterTLSConfig(tlsKey)
		dsn += fmt.Sprintf("?tls=%s", tlsKey)
	}
	driver.l.Debug("Opening MySQL driver",
		zap.String("dsn", loggedDSN),
		zap.String("environment", ctx.EnvironmentName),
		zap.String("database", ctx.InstanceName),
	)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}
	driver.db = db
	driver.connectionCtx = ctx

	return driver, nil
}

func (driver *Driver) Close(ctx context.Context) error {
	return driver.db.Close()
}

func (driver *Driver) Ping(ctx context.Context) error {
	return driver.db.PingContext(ctx)
}

func (driver *Driver) SyncSchema(ctx context.Context) ([]*db.DBUser, []*db.DBSchema, error) {
	// Query MySQL version
	query := "SELECT VERSION()"
	versionRow, err := driver.db.QueryContext(ctx, query)
	if err != nil {
		return nil, nil, formatErrorWithQuery(err, query)
	}
	defer versionRow.Close()

	var version string
	versionRow.Next()
	if err := versionRow.Scan(&version); err != nil {
		return nil, nil, err
	}
	isMySQL8 := strings.HasPrefix(version, "8.0")

	excludedDatabaseList := []string{
		// Skip our internal "bytebase" database
		"'bytebase'",
	}

	// Skip all system databases
	for k := range systemDatabases {
		excludedDatabaseList = append(excludedDatabaseList, fmt.Sprintf("'%s'", k))
	}

	// Query user info
	query = `
	    SELECT
			user,
			host
		FROM mysql.user
		WHERE user NOT LIKE 'mysql.%'
	`
	userList := make([]*db.DBUser, 0)
	userRows, err := driver.db.QueryContext(ctx, query)

	if err != nil {
		return nil, nil, formatErrorWithQuery(err, query)
	}
	defer userRows.Close()

	for userRows.Next() {
		var user string
		var host string
		if err := userRows.Scan(
			&user,
			&host,
		); err != nil {
			return nil, nil, err
		}

		// Uses single quote instead of backtick to escape because this is a string
		// instead of table (which should use backtick instead). MySQL actually works
		// in both ways. On the other hand, some other MySQL compatible engines might not (OceanBase in this case).
		name := fmt.Sprintf("'%s'@'%s'", user, host)
		query = fmt.Sprintf("SHOW GRANTS FOR %s", name)
		grantRows, err := driver.db.QueryContext(ctx,
			query,
		)
		if err != nil {
			return nil, nil, formatErrorWithQuery(err, query)
		}
		defer grantRows.Close()

		grantList := []string{}
		for grantRows.Next() {
			var grant string
			if err := grantRows.Scan(&grant); err != nil {
				return nil, nil, err
			}
			grantList = append(grantList, grant)
		}

		userList = append(userList, &db.DBUser{
			Name:  name,
			Grant: strings.Join(grantList, "\n"),
		})
	}

	// Query index info
	indexWhere := fmt.Sprintf("TABLE_SCHEMA NOT IN (%s)", strings.Join(excludedDatabaseList, ", "))
	query = `
			SELECT
				TABLE_SCHEMA,
				TABLE_NAME,
				INDEX_NAME,
				COLUMN_NAME,
				'',
				SEQ_IN_INDEX,
				INDEX_TYPE,
				CASE NON_UNIQUE WHEN 0 THEN 1 ELSE 0 END AS IS_UNIQUE,
				1,
				INDEX_COMMENT
			FROM information_schema.STATISTICS
			WHERE ` + indexWhere
	if isMySQL8 {
		query = `
			SELECT
				TABLE_SCHEMA,
				TABLE_NAME,
				INDEX_NAME,
				COLUMN_NAME,
				EXPRESSION,
				SEQ_IN_INDEX,
				INDEX_TYPE,
				CASE NON_UNIQUE WHEN 0 THEN 1 ELSE 0 END AS IS_UNIQUE,
				CASE IS_VISIBLE WHEN 'YES' THEN 1 ELSE 0 END,
				INDEX_COMMENT
			FROM information_schema.STATISTICS
			WHERE ` + indexWhere
	}
	indexRows, err := driver.db.QueryContext(ctx, query)
	if err != nil {
		return nil, nil, formatErrorWithQuery(err, query)
	}
	defer indexRows.Close()

	// dbName/tableName -> indexList map
	indexMap := make(map[string][]db.DBIndex)
	for indexRows.Next() {
		var dbName string
		var tableName string
		var columnName sql.NullString
		var expression sql.NullString
		var index db.DBIndex
		if err := indexRows.Scan(
			&dbName,
			&tableName,
			&index.Name,
			&columnName,
			&expression,
			&index.Position,
			&index.Type,
			&index.Unique,
			&index.Visible,
			&index.Comment,
		); err != nil {
			return nil, nil, err
		}

		if columnName.Valid {
			index.Expression = columnName.String
		} else if expression.Valid {
			index.Expression = expression.String
		}

		key := fmt.Sprintf("%s/%s", dbName, tableName)
		indexList, ok := indexMap[key]
		if ok {
			indexMap[key] = append(indexList, index)
		} else {
			list := make([]db.DBIndex, 0)
			indexMap[key] = append(list, index)
		}
	}

	// Query column info
	columnWhere := fmt.Sprintf("TABLE_SCHEMA NOT IN (%s)", strings.Join(excludedDatabaseList, ", "))
	query = `
			SELECT
				TABLE_SCHEMA,
				TABLE_NAME,
				IFNULL(COLUMN_NAME, ''), 
				ORDINAL_POSITION,
				COLUMN_DEFAULT,
				IS_NULLABLE,
				COLUMN_TYPE,
				IFNULL(CHARACTER_SET_NAME, ''),
				IFNULL(COLLATION_NAME, ''),
				COLUMN_COMMENT
			FROM information_schema.COLUMNS
			WHERE ` + columnWhere
	columnRows, err := driver.db.QueryContext(ctx, query)
	if err != nil {
		return nil, nil, formatErrorWithQuery(err, query)
	}
	defer columnRows.Close()

	// dbName/tableName -> columnList map
	columnMap := make(map[string][]db.DBColumn)
	for columnRows.Next() {
		var dbName string
		var tableName string
		var nullable string
		var defaultStr sql.NullString
		var column db.DBColumn
		if err := columnRows.Scan(
			&dbName,
			&tableName,
			&column.Name,
			&column.Position,
			&defaultStr,
			&nullable,
			&column.Type,
			&column.CharacterSet,
			&column.Collation,
			&column.Comment,
		); err != nil {
			return nil, nil, err
		}

		if defaultStr.Valid {
			column.Default = &defaultStr.String
		}

		key := fmt.Sprintf("%s/%s", dbName, tableName)
		tableList, ok := columnMap[key]
		if ok {
			columnMap[key] = append(tableList, column)
		} else {
			list := make([]db.DBColumn, 0)
			columnMap[key] = append(list, column)
		}
	}

	// Query table info
	tableWhere := fmt.Sprintf("TABLE_SCHEMA NOT IN (%s)", strings.Join(excludedDatabaseList, ", "))
	query = `
			SELECT
				TABLE_SCHEMA, 
				TABLE_NAME,
				IFNULL(UNIX_TIMESTAMP(CREATE_TIME), 0),
				IFNULL(UNIX_TIMESTAMP(UPDATE_TIME), 0),
				TABLE_TYPE,
				IFNULL(ENGINE, ''),
				IFNULL(TABLE_COLLATION, ''),
				IFNULL(TABLE_ROWS, 0),
				IFNULL(DATA_LENGTH, 0),
				IFNULL(INDEX_LENGTH, 0),
				IFNULL(DATA_FREE, 0),
				IFNULL(CREATE_OPTIONS, ''),
				IFNULL(TABLE_COMMENT, '')
			FROM information_schema.TABLES
			WHERE ` + tableWhere
	tableRows, err := driver.db.QueryContext(ctx, query)
	if err != nil {
		return nil, nil, formatErrorWithQuery(err, query)
	}
	defer tableRows.Close()

	// dbName -> tableList map
	tableMap := make(map[string][]db.DBTable)
	for tableRows.Next() {
		var dbName string
		var table db.DBTable
		if err := tableRows.Scan(
			&dbName,
			&table.Name,
			&table.CreatedTs,
			&table.UpdatedTs,
			&table.Type,
			&table.Engine,
			&table.Collation,
			&table.RowCount,
			&table.DataSize,
			&table.IndexSize,
			&table.DataFree,
			&table.CreateOptions,
			&table.Comment,
		); err != nil {
			return nil, nil, err
		}

		key := fmt.Sprintf("%s/%s", dbName, table.Name)
		table.ColumnList = columnMap[key]
		table.IndexList = indexMap[key]

		tableList, ok := tableMap[dbName]
		if ok {
			tableMap[dbName] = append(tableList, table)
		} else {
			list := make([]db.DBTable, 0)
			tableMap[dbName] = append(list, table)
		}
	}

	// Query db info
	where := fmt.Sprintf("SCHEMA_NAME NOT IN (%s)", strings.Join(excludedDatabaseList, ", "))
	query = `
			SELECT 
		    SCHEMA_NAME,
			DEFAULT_CHARACTER_SET_NAME,
			DEFAULT_COLLATION_NAME
		FROM information_schema.SCHEMATA
		WHERE ` + where
	rows, err := driver.db.QueryContext(ctx, query)
	if err != nil {
		return nil, nil, formatErrorWithQuery(err, query)
	}
	defer rows.Close()

	schemaList := make([]*db.DBSchema, 0)
	for rows.Next() {
		var schema db.DBSchema
		if err := rows.Scan(
			&schema.Name,
			&schema.CharacterSet,
			&schema.Collation,
		); err != nil {
			return nil, nil, err
		}

		schema.TableList = tableMap[schema.Name]

		schemaList = append(schemaList, &schema)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	return userList, schemaList, err
}

func (driver *Driver) Execute(ctx context.Context, statement string) error {
	tx, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, statement)

	if err := tx.Commit(); err != nil {
		return err
	}

	return err
}

// Migration related
func (driver *Driver) NeedsSetupMigration(ctx context.Context) (bool, error) {
	const query = `
		SELECT 
		    1
		FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = 'bytebase' AND TABLE_NAME = 'migration_history'
		`
	rows, err := driver.db.QueryContext(ctx, query)
	if err != nil {
		return false, formatErrorWithQuery(err, query)
	}
	defer rows.Close()

	if rows.Next() {
		return false, nil
	}

	return true, nil
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
		if err := driver.Execute(ctx, migrationSchema); err != nil {
			driver.l.Error("Failed to initialize migration schema.",
				zap.Error(err),
				zap.String("environment", driver.connectionCtx.EnvironmentName),
				zap.String("database", driver.connectionCtx.InstanceName),
			)
			return formatErrorWithQuery(err, migrationSchema)
		}
		driver.l.Info("Successfully created migration schema.",
			zap.String("environment", driver.connectionCtx.EnvironmentName),
			zap.String("database", driver.connectionCtx.InstanceName),
		)
	}

	return nil
}

func (driver *Driver) ExecuteMigration(ctx context.Context, m *db.MigrationInfo, statement string) (string, error) {
	tx, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	startedTs := time.Now().Unix()

	// Phase 1 - Precheck before executing migration
	// Check if the same migration version has alraedy been applied
	duplicate, err := checkDuplicateVersion(ctx, tx, m.Namespace, m.Engine, m.Version)
	if err != nil {
		return "", err
	}
	if duplicate {
		return "", fmt.Errorf("database %q has already applied version %s", m.Database, m.Version)
	}

	// Check if there is any higher version already been applied
	version, err := checkOutofOrderVersion(ctx, tx, m.Namespace, m.Engine, m.Version)
	if err != nil {
		return "", err
	}
	if version != nil {
		return "", fmt.Errorf("database %q has already applied version %s which is higher than %s", m.Database, *version, m.Version)
	}

	// If the migration engine is VCS and type is not baseline and is not branch, then we can only proceed if there is existing baseline
	// This check is also wrapped in transaction to avoid edge case where two baselinings are running concurrently.
	if m.Engine == db.VCS && m.Type != db.Baseline && m.Type != db.Branch {
		hasBaseline, err := findBaseline(ctx, tx, m.Namespace)
		if err != nil {
			return "", err
		}

		if !hasBaseline {
			return "", fmt.Errorf("%s has not created migration baseline yet", m.Database)
		}
	}

	// VCS based SQL migration requires existing baselining
	requireBaseline := m.Engine == db.VCS && m.Type == db.Migrate
	sequence, err := findNextSequence(ctx, tx, m.Namespace, requireBaseline)
	if err != nil {
		return "", err
	}

	// Phase 2 - Executing migration
	// Branch migration type always has empty sql.
	// Baseline migration type could also has empty sql when the database is newly created.
	if statement != "" {
		_, err = tx.ExecContext(ctx, statement)
		if err != nil {
			return "", formatError(err)
		}
	}

	// Phase 3 - Dump the schema after migration
	var schemaBuf bytes.Buffer
	err = dumpTxn(ctx, tx, m.Database, &schemaBuf, true /*schemaOnly*/)
	if err != nil {
		return "", formatError(err)
	}

	// Phase 4 - Record migration
	const query = `
		INSERT INTO bytebase.migration_history (
			created_by,
			created_ts,
			updated_by,
			updated_ts,
			namespace,
			sequence,
			` + "`engine`," + `
			` + "`type`," + `
			version,
			description,
			statement,
			` + "`schema`," + `
			execution_duration,
			issue_id,
			payload
		)
		VALUES (?, unix_timestamp(), ?, unix_timestamp(), ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = tx.ExecContext(ctx, query,
		m.Creator,
		m.Creator,
		m.Namespace,
		sequence,
		m.Engine,
		m.Type,
		m.Version,
		m.Description,
		statement,
		schemaBuf.String(),
		time.Now().Unix()-startedTs,
		m.IssueId,
		m.Payload,
	)

	if err != nil {
		return "", formatErrorWithQuery(err, query)
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}

	return schemaBuf.String(), nil
}

func (driver *Driver) FindMigrationHistoryList(ctx context.Context, find *db.MigrationHistoryFind) ([]*db.MigrationHistory, error) {
	tx, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, *v)
	}
	if v := find.Database; v != nil {
		where, args = append(where, "namespace = ?"), append(args, *v)
	}
	if v := find.Version; v != nil {
		where, args = append(where, "version = ?"), append(args, *v)
	}

	var query = `
			SELECT 
		    id,
			created_by,
		    created_ts,
		    updated_by,
		    updated_ts,
			namespace,
			sequence,
			` + "`engine`," + `
			` + "`type`," + `
			version,
			description,
		    statement,
			` + "`schema`," + `
		    execution_duration,
			issue_id,
			payload
		FROM bytebase.migration_history
		WHERE ` + strings.Join(where, " AND ") + `
		ORDER BY created_ts DESC`
	if v := find.Limit; v != nil {
		query += fmt.Sprintf(" LIMIT %d", *v)
	}

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, formatErrorWithQuery(err, query)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into list.
	list := make([]*db.MigrationHistory, 0)
	for rows.Next() {
		var history db.MigrationHistory
		if err := rows.Scan(
			&history.ID,
			&history.Creator,
			&history.CreatedTs,
			&history.Updater,
			&history.UpdatedTs,
			&history.Namespace,
			&history.Sequence,
			&history.Engine,
			&history.Type,
			&history.Version,
			&history.Description,
			&history.Statement,
			&history.Schema,
			&history.ExecutionDuration,
			&history.IssueId,
			&history.Payload,
		); err != nil {
			return nil, err
		}

		list = append(list, &history)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return list, nil
}

func findBaseline(ctx context.Context, tx *sql.Tx, namespace string) (bool, error) {
	query := `
		SELECT 1 FROM bytebase.migration_history WHERE namespace = ? AND ` + "`type` = 'BASELINE'" + `
	`
	args := []interface{}{namespace}
	row, err := tx.QueryContext(ctx, query,
		args...,
	)

	if err != nil {
		return false, formatErrorWithQuery(err, query)
	}
	defer row.Close()

	if !row.Next() {
		return false, nil
	}

	return true, nil
}

func checkDuplicateVersion(ctx context.Context, tx *sql.Tx, namespace string, engine db.MigrationEngine, version string) (bool, error) {
	query := `
		SELECT 1 FROM bytebase.migration_history WHERE namespace = ? AND ` + "`engine` = ? AND version = ?" + `
	`
	args := []interface{}{namespace, engine.String(), version}
	row, err := tx.QueryContext(ctx, query,
		args...,
	)

	if err != nil {
		return false, formatErrorWithQuery(err, query)
	}
	defer row.Close()

	if row.Next() {
		return true, nil
	}
	return false, nil
}

func checkOutofOrderVersion(ctx context.Context, tx *sql.Tx, namespace string, engine db.MigrationEngine, version string) (*string, error) {
	query := `
		SELECT MIN(version) FROM bytebase.migration_history WHERE namespace = ? AND ` + "`engine` = ? AND STRCMP(?, version) = -1" + `
	`
	args := []interface{}{namespace, engine.String(), version}
	row, err := tx.QueryContext(ctx, query,
		args...,
	)

	if err != nil {
		return nil, formatErrorWithQuery(err, query)
	}
	defer row.Close()

	var minVersion sql.NullString
	row.Next()
	if err := row.Scan(&minVersion); err != nil {
		return nil, err
	}

	if minVersion.Valid {
		return &minVersion.String, nil
	}

	return nil, nil
}

func findNextSequence(ctx context.Context, tx *sql.Tx, namespace string, requireBaseline bool) (int, error) {
	query := `
		SELECT MAX(sequence) + 1 FROM bytebase.migration_history WHERE namespace = ?
	`
	args := []interface{}{namespace}
	row, err := tx.QueryContext(ctx, query,
		args...,
	)

	if err != nil {
		return -1, formatErrorWithQuery(err, query)
	}
	defer row.Close()

	var sequence sql.NullInt32
	row.Next()
	if err := row.Scan(&sequence); err != nil {
		return -1, err
	}

	if !sequence.Valid {
		// Returns 1 if we haven't applied any migration for this namespace and doesn't require baselining
		if !requireBaseline {
			return 1, nil
		}

		// This should not happen normally since we already check the baselining exist beforehand. Just in case.
		return -1, fmt.Errorf("unable to generate next migration_sequence, no migration hisotry found for %q, do you forget to baselining?", namespace)
	}

	return int(sequence.Int32), nil
}

func formatError(err error) error {
	if err == nil {
		return nil
	}

	if strings.Contains(err.Error(), "bytebase_idx_unique_migration_history_namespace_version") {
		return fmt.Errorf("version has already been applied")
	} else if strings.Contains(err.Error(), "bytebase_idx_unique_migration_history_namespace_sequence") {
		return fmt.Errorf("concurrent migration")
	}

	return err
}

func formatErrorWithQuery(err error, query string) error {
	return fmt.Errorf("failed to execute error: %w\n\nquery:\n%q", err, query)
}

// Dump and restore
const (
	databaseHeaderFmt = "" +
		"--\n" +
		"-- MySQL database structure for `%s`\n" +
		"--\n"
	useDatabaseFmt = "USE `%s`;\n\n"
	settingsStmt   = "" +
		"SET character_set_client  = %s;\n" +
		"SET character_set_results = %s;\n" +
		"SET collation_connection  = %s;\n" +
		"SET sql_mode              = '%s';\n"
	tableStmtFmt = "" +
		"--\n" +
		"-- Table structure for `%s`\n" +
		"--\n" +
		"%s;\n"
	viewStmtFmt = "" +
		"--\n" +
		"-- View structure for `%s`\n" +
		"--\n" +
		"%s;\n"
	routineStmtFmt = "" +
		"--\n" +
		"-- %s structure for `%s`\n" +
		"--\n" +
		settingsStmt +
		"DELIMITER ;;\n" +
		"%s ;;\n" +
		"DELIMITER ;\n"
	eventStmtFmt = "" +
		"--\n" +
		"-- Event structure for `%s`\n" +
		"--\n" +
		settingsStmt +
		"SET time_zone = '%s';\n" +
		"DELIMITER ;;\n" +
		"%s ;;\n" +
		"DELIMITER ;\n"
	triggerStmtFmt = "" +
		"--\n" +
		"-- Trigger structure for `%s`\n" +
		"--\n" +
		settingsStmt +
		"DELIMITER ;;\n" +
		"%s ;;\n" +
		"DELIMITER ;\n"
)

func (driver *Driver) Dump(ctx context.Context, database string, out io.Writer, schemaOnly bool) error {
	// mysqldump -u root --databases dbName --no-data --routines --events --triggers --compact

	txn, err := driver.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return err
	}
	defer txn.Rollback()

	if err := dumpTxn(ctx, txn, database, out, schemaOnly); err != nil {
		return err
	}

	if err := txn.Commit(); err != nil {
		return err
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
	delimiter := false
	for sc.Scan() {
		line := sc.Text()

		execute := false
		switch {
		case s == "" && line == "":
			continue
		case strings.HasPrefix(line, "--"):
			continue
		case line == "DELIMITER ;;":
			delimiter = true
			continue
		case line == "DELIMITER ;" && delimiter:
			delimiter = false
			execute = true
		case strings.HasSuffix(line, ";"):
			s = s + line + "\n"
			if !delimiter {
				execute = true
			}
		default:
			s = s + line + "\n"
			continue
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
		// Database header.
		header := fmt.Sprintf(databaseHeaderFmt, dbName)
		if _, err := io.WriteString(out, header); err != nil {
			return err
		}
		// Include "USE DATABASE xxx" if dumping multiple databases.
		if len(dumpableDbNames) > 1 {
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
			if !schemaOnly && tbl.tableType == "BASE TABLE" {
				// Include db prefix if dumping multiple databases.
				includeDbPrefix := len(dumpableDbNames) > 1
				if err := exportTableData(txn, dbName, tbl.name, includeDbPrefix, out); err != nil {
					return err
				}
			}
		}

		// Procedure and function (routine) statements.
		routines, err := getRoutines(txn, dbName)
		if err != nil {
			return fmt.Errorf("failed to get routines of database %q: %s", dbName, err)
		}
		for _, rt := range routines {
			if _, err := io.WriteString(out, fmt.Sprintf("%s\n", rt.statement)); err != nil {
				return err
			}
		}

		// Event statements.
		events, err := getEvents(txn, dbName)
		if err != nil {
			return fmt.Errorf("failed to get events of database %q: %s", dbName, err)
		}
		for _, et := range events {
			if _, err := io.WriteString(out, fmt.Sprintf("%s\n", et.statement)); err != nil {
				return err
			}
		}

		// Trigger statements.
		triggers, err := getTriggers(txn, dbName)
		if err != nil {
			return fmt.Errorf("failed to get triggers of database %q: %s", dbName, err)
		}
		for _, tr := range triggers {
			if _, err := io.WriteString(out, fmt.Sprintf("%s\n", tr.statement)); err != nil {
				return err
			}
		}
	}

	return nil
}

// getDatabases gets all databases of an instance.
func getDatabases(txn *sql.Tx) ([]string, error) {
	var dbNames []string
	rows, err := txn.Query("SHOW DATABASES;")
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

// routineSchema describes the schema of a function or procedure (routine).
type routineSchema struct {
	name        string
	routineType string
	statement   string
}

// eventSchema describes the schema of an event.
type eventSchema struct {
	name      string
	statement string
}

// triggerSchema describes the schema of a trigger.
type triggerSchema struct {
	name      string
	statement string
}

// getTables gets all tables of a database.
func getTables(txn *sql.Tx, dbName string) ([]*tableSchema, error) {
	var tables []*tableSchema
	query := fmt.Sprintf("SHOW FULL TABLES FROM `%s`;", dbName)
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var tbl tableSchema
		if err := rows.Scan(&tbl.name, &tbl.tableType); err != nil {
			return nil, err
		}
		tables = append(tables, &tbl)
	}
	for _, tbl := range tables {
		stmt, err := getTableStmt(txn, dbName, tbl.name, tbl.tableType)
		if err != nil {
			return nil, fmt.Errorf("getTableStmt(%q, %q, %q) got error: %s", dbName, tbl.name, tbl.tableType, err)
		}
		tbl.statement = stmt
	}
	return tables, nil
}

// getTableStmt gets the create statement of a table.
func getTableStmt(txn *sql.Tx, dbName, tblName, tblType string) (string, error) {
	switch tblType {
	case "BASE TABLE":
		query := fmt.Sprintf("SHOW CREATE TABLE %s.%s;", dbName, tblName)
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
			return fmt.Sprintf(tableStmtFmt, tblName, stmt), nil
		}
		return "", fmt.Errorf("query %q returned invalid rows", query)
	case "VIEW":
		// This differs from mysqldump as it includes.
		query := fmt.Sprintf("SHOW CREATE VIEW %s.%s;", dbName, tblName)
		rows, err := txn.Query(query)
		if err != nil {
			return "", err
		}
		defer rows.Close()

		if rows.Next() {
			var createStmt, unused string
			if err := rows.Scan(&unused, &createStmt, &unused, &unused); err != nil {
				return "", err
			}
			return fmt.Sprintf(viewStmtFmt, tblName, createStmt), nil
		}
		return "", fmt.Errorf("query %q returned invalid rows", query)
	default:
		return "", fmt.Errorf("unrecognized table type %q for database %q table %q", tblType, dbName, tblName)
	}

}

// exportTableData gets the data of a table.
func exportTableData(txn *sql.Tx, dbName, tblName string, includeDbPrefix bool, out io.Writer) error {
	query := fmt.Sprintf("SELECT * FROM `%s`.`%s`;", dbName, tblName)
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
		dbPrefix := ""
		if includeDbPrefix {
			dbPrefix = fmt.Sprintf("`%s`.", dbName)
		}
		stmt := fmt.Sprintf("INSERT INTO %s`%s` VALUES (%s);\n", dbPrefix, tblName, strings.Join(tokens, ", "))
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

// getRoutines gets all routines of a database.
func getRoutines(txn *sql.Tx, dbName string) ([]*routineSchema, error) {
	var routines []*routineSchema
	for _, routineType := range []string{"FUNCTION", "PROCEDURE"} {
		query := fmt.Sprintf("SHOW %s STATUS WHERE Db = ?;", routineType)
		rows, err := txn.Query(query, dbName)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		cols, err := rows.Columns()
		if err != nil {
			return nil, err
		}
		var values []interface{}
		for i := 0; i < len(cols); i++ {
			values = append(values, new(interface{}))
		}
		for rows.Next() {
			var r routineSchema
			if err := rows.Scan(values...); err != nil {
				return nil, err
			}
			r.name = fmt.Sprintf("%s", *values[1].(*interface{}))
			r.routineType = fmt.Sprintf("%s", *values[2].(*interface{}))

			routines = append(routines, &r)
		}
	}

	for _, r := range routines {
		stmt, err := getRoutineStmt(txn, dbName, r.name, r.routineType)
		if err != nil {
			return nil, fmt.Errorf("getRoutineStmt(%q, %q, %q) got error: %s", dbName, r.name, r.routineType, err)
		}
		r.statement = stmt
	}
	return routines, nil
}

// getRoutineStmt gets the create statement of a routine.
func getRoutineStmt(txn *sql.Tx, dbName, routineName, routineType string) (string, error) {
	query := fmt.Sprintf("SHOW CREATE %s %s.%s;", routineType, dbName, routineName)
	rows, err := txn.Query(query)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	if rows.Next() {
		var sqlmode, stmt, charset, collation, unused string
		if err := rows.Scan(&unused, &sqlmode, &stmt, &charset, &collation, &unused); err != nil {
			return "", err
		}
		return fmt.Sprintf(routineStmtFmt, getReadableRoutineType(routineType), routineName, charset, charset, collation, sqlmode, stmt), nil
	}
	return "", fmt.Errorf("query %q returned invalid rows", query)

}

// getReadableRoutineType gets the printable routine type.
func getReadableRoutineType(s string) string {
	switch s {
	case "FUNCTION":
		return "Function"
	case "PROCEDURE":
		return "Procedure"
	default:
		return s
	}
}

// getEvents gets all events of a database.
func getEvents(txn *sql.Tx, dbName string) ([]*eventSchema, error) {
	var events []*eventSchema
	rows, err := txn.Query(fmt.Sprintf("SHOW EVENTS FROM `%s`;", dbName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	var values []interface{}
	for i := 0; i < len(cols); i++ {
		values = append(values, new(interface{}))
	}
	for rows.Next() {
		var r eventSchema
		if err := rows.Scan(values...); err != nil {
			return nil, err
		}
		r.name = fmt.Sprintf("%s", *values[1].(*interface{}))
		events = append(events, &r)
	}

	for _, r := range events {
		stmt, err := getEventStmt(txn, dbName, r.name)
		if err != nil {
			return nil, fmt.Errorf("getEventStmt(%q, %q) got error: %s", dbName, r.name, err)
		}
		r.statement = stmt
	}
	return events, nil
}

// getEventStmt gets the create statement of an event.
func getEventStmt(txn *sql.Tx, dbName, eventName string) (string, error) {
	query := fmt.Sprintf("SHOW CREATE EVENT %s.%s;", dbName, eventName)
	rows, err := txn.Query(query)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	if rows.Next() {
		var sqlmode, timezone, stmt, charset, collation, unused string
		if err := rows.Scan(&unused, &sqlmode, &timezone, &stmt, &charset, &collation, &unused); err != nil {
			return "", err
		}
		return fmt.Sprintf(eventStmtFmt, eventName, charset, charset, collation, sqlmode, timezone, stmt), nil
	}
	return "", fmt.Errorf("query %q returned invalid rows", query)
}

// getTriggers gets all triggers of a database.
func getTriggers(txn *sql.Tx, dbName string) ([]*triggerSchema, error) {
	var triggers []*triggerSchema
	rows, err := txn.Query(fmt.Sprintf("SHOW TRIGGERS FROM `%s`;", dbName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	var values []interface{}
	for i := 0; i < len(cols); i++ {
		values = append(values, new(interface{}))
	}
	for rows.Next() {
		var tr triggerSchema
		if err := rows.Scan(values...); err != nil {
			return nil, err
		}
		tr.name = fmt.Sprintf("%s", *values[0].(*interface{}))
		triggers = append(triggers, &tr)
	}
	for _, tr := range triggers {
		stmt, err := getTriggerStmt(txn, dbName, tr.name)
		if err != nil {
			return nil, fmt.Errorf("getTriggerStmt(%q, %q) got error: %s", dbName, tr.name, err)
		}
		tr.statement = stmt
	}
	return triggers, nil
}

// getTriggerStmt gets the create statement of a trigger.
func getTriggerStmt(txn *sql.Tx, dbName, triggerName string) (string, error) {
	query := fmt.Sprintf("SHOW CREATE TRIGGER %s.%s;", dbName, triggerName)
	rows, err := txn.Query(query)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	if rows.Next() {
		var sqlmode, stmt, charset, collation, unused string
		if err := rows.Scan(&unused, &sqlmode, &stmt, &charset, &collation, &unused, &unused); err != nil {
			return "", err
		}
		return fmt.Sprintf(triggerStmtFmt, triggerName, charset, charset, collation, sqlmode, stmt), nil
	}
	return "", fmt.Errorf("query %q returned invalid rows", query)
}
