package db

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
)

//go:embed mysql_migration_schema.sql
var migrationSchema string

var (
	_ Driver = (*MySQLDriver)(nil)
)

func init() {
	register(Mysql, newDriver)
}

type MySQLDriver struct {
	l *zap.Logger

	db *sql.DB
}

func newDriver(config DriverConfig) Driver {
	return &MySQLDriver{
		l: config.Logger,
	}
}

func (driver *MySQLDriver) open(config ConnectionConfig) (Driver, error) {
	protocol := "tcp"
	if strings.HasPrefix(config.Host, "/") {
		protocol = "unix"
	}

	params := []string{"multiStatements=true"}

	port := config.Port
	if port == "" {
		port = "3306"
	}

	dsn := fmt.Sprintf("%s:%s@%s(%s:%s)/%s?%s", config.Username, config.Password, protocol, config.Host, port, config.Database, strings.Join(params, "&"))
	driver.l.Debug("Opening MySQL driver", zap.String("dsn", dsn))
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}
	driver.db = db

	return driver, nil
}

func (driver *MySQLDriver) Close(ctx context.Context) error {
	return driver.db.Close()
}

func (driver *MySQLDriver) Ping(ctx context.Context) error {
	return driver.db.PingContext(ctx)
}

func (driver *MySQLDriver) SyncSchema(ctx context.Context) ([]*DBUser, []*DBSchema, error) {
	excludedDatabaseList := []string{
		"'mysql'",
		"'information_schema'",
		"'performance_schema'",
		"'sys'",
		// Skip our internal "bytebase" database
		"'bytebase'",
	}

	// Query user info
	userList := make([]*DBUser, 0)
	userRows, err := driver.db.QueryContext(ctx, `
	    SELECT
			user,
			host
		FROM mysql.user
		WHERE user NOT LIKE 'mysql.%'
	`)

	if err != nil {
		return nil, nil, err
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
		grantRows, err := driver.db.QueryContext(ctx,
			fmt.Sprintf("SHOW GRANTS FOR %s", name),
		)
		if err != nil {
			return nil, nil, err
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

		userList = append(userList, &DBUser{
			Name:  name,
			Grant: strings.Join(grantList, "\n"),
		})
	}

	// Query index info
	indexWhere := fmt.Sprintf("TABLE_SCHEMA NOT IN (%s)", strings.Join(excludedDatabaseList, ", "))
	indexRows, err := driver.db.QueryContext(ctx, `
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
			WHERE `+indexWhere,
	)
	if err != nil {
		return nil, nil, err
	}
	defer indexRows.Close()

	// dbName/tableName -> indexList map
	indexMap := make(map[string][]DBIndex)
	for indexRows.Next() {
		var dbName string
		var tableName string
		var columnName sql.NullString
		var expression sql.NullString
		var index DBIndex
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
			list := make([]DBIndex, 0)
			indexMap[key] = append(list, index)
		}
	}

	// Query column info
	columnWhere := fmt.Sprintf("TABLE_SCHEMA NOT IN (%s)", strings.Join(excludedDatabaseList, ", "))
	columnRows, err := driver.db.QueryContext(ctx, `
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
			WHERE `+columnWhere,
	)
	if err != nil {
		return nil, nil, err
	}
	defer columnRows.Close()

	// dbName/tableName -> columnList map
	columnMap := make(map[string][]DBColumn)
	for columnRows.Next() {
		var dbName string
		var tableName string
		var nullable string
		var defaultStr sql.NullString
		var column DBColumn
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
			list := make([]DBColumn, 0)
			columnMap[key] = append(list, column)
		}
	}

	// Query table info
	tableWhere := fmt.Sprintf("TABLE_SCHEMA NOT IN (%s)", strings.Join(excludedDatabaseList, ", "))
	tableRows, err := driver.db.QueryContext(ctx, `
			SELECT
				TABLE_SCHEMA, 
				TABLE_NAME,
				UNIX_TIMESTAMP(CREATE_TIME),
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
			WHERE `+tableWhere,
	)
	if err != nil {
		return nil, nil, err
	}
	defer tableRows.Close()

	// dbName -> tableList map
	tableMap := make(map[string][]DBTable)
	for tableRows.Next() {
		var dbName string
		var table DBTable
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
			list := make([]DBTable, 0)
			tableMap[dbName] = append(list, table)
		}
	}

	// Query db info
	where := fmt.Sprintf("SCHEMA_NAME NOT IN (%s)", strings.Join(excludedDatabaseList, ", "))
	rows, err := driver.db.QueryContext(ctx, `
		SELECT 
		    SCHEMA_NAME,
			DEFAULT_CHARACTER_SET_NAME,
			DEFAULT_COLLATION_NAME
		FROM information_schema.SCHEMATA
		WHERE `+where,
	)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	schemaList := make([]*DBSchema, 0)
	for rows.Next() {
		var schema DBSchema
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

func (driver *MySQLDriver) Execute(ctx context.Context, statement string) error {
	tx, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, statement)
	return err
}

func (driver *MySQLDriver) NeedsSetupMigration(ctx context.Context) (bool, error) {
	rows, err := driver.db.QueryContext(ctx, `
		SELECT 
		    1
		FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = 'bytebase' AND TABLE_NAME = 'migration_history'`,
	)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	if rows.Next() {
		return false, nil
	}

	return true, nil
}

func (driver *MySQLDriver) SetupMigrationIfNeeded(ctx context.Context) error {
	setup, err := driver.NeedsSetupMigration(ctx)
	if err != nil {
		return nil
	}

	if setup {
		driver.l.Info("Bytebase migration schema not found, creating schema...")
		if err := driver.Execute(ctx, migrationSchema); err != nil {
			driver.l.Error("Failed to initialize migration schema.", zap.Error(err))
			return formatError(err)
		}
		driver.l.Info("Successfully created migration schema.")
	}

	return nil
}

func (driver *MySQLDriver) ExecuteMigration(ctx context.Context, m *MigrationInfo, statement string) error {
	tx, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	startedTs := time.Now().Unix()

	// Phase 1 - Precheck before executing migration
	// Check if the same migration version has alraedy been applied
	duplicate, err := checkDuplicateVersion(ctx, tx, m.Namespace, m.Engine, m.Version)
	if err != nil {
		return err
	}
	if duplicate {
		return fmt.Errorf("database '%s' has already applied version %s", m.Database, m.Version)
	}

	// Check if there is any higher version already been applied
	version, err := checkOutofOrderVersion(ctx, tx, m.Namespace, m.Engine, m.Version)
	if err != nil {
		return err
	}
	if version != nil {
		return fmt.Errorf("database '%s' has already applied version %s which is higher than %s", m.Database, *version, m.Version)
	}

	// If the migration engine is VCS and type is not baseline, then we can only proceed if there is existing baseline
	// This check is also wrapped in transaction to avoid edge case where two baselinings are running concurrently.
	if m.Engine == VCS && m.Type != Baseline {
		hasBaseline, err := findBaseline(ctx, tx, m.Namespace)
		if err != nil {
			return err
		}

		if !hasBaseline {
			return fmt.Errorf("%s has not created migration baseline yet", m.Database)
		}
	}

	// VCS based SQL migration requires existing baselining
	requireBaseline := m.Engine == VCS && m.Type == Sql
	sequence, err := findNextSequence(ctx, tx, m.Namespace, requireBaseline)
	if err != nil {
		return err
	}

	// Phase 2 - Executing migration unless it's VCS baselining
	if m.Engine != VCS || m.Type != Baseline {
		_, err = tx.ExecContext(ctx, statement)
		if err != nil {
			return formatError(err)
		}
	}

	// Phase 3 - Record migration
	_, err = tx.ExecContext(ctx, `
		INSERT INTO bytebase.migration_history (
			created_by,
			created_ts,
			updated_by,
			updated_ts,
			namespace,
			sequence,
			`+"`engine`,"+`
			`+"`type`,"+`
			version,
			description,
			statement,
			execution_duration,
			issue_id,
			payload
		)
		VALUES (?, unix_timestamp(), ?, unix_timestamp(), ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		m.Creator,
		m.Creator,
		m.Namespace,
		sequence,
		m.Engine,
		m.Type,
		m.Version,
		m.Description,
		statement,
		time.Now().Unix()-startedTs,
		m.IssueId,
		m.Payload,
	)

	if err != nil {
		return formatError(err)
	}

	tx.Commit()

	return nil
}

func (driver *MySQLDriver) FindMigrationHistoryList(ctx context.Context, find *MigrationHistoryFind) ([]*MigrationHistory, error) {
	tx, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.Database; v != nil {
		where, args = append(where, "namespace = ?"), append(args, *v)
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
		return nil, err
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into list.
	list := make([]*MigrationHistory, 0)
	for rows.Next() {
		var history MigrationHistory
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

	return list, nil
}

func findBaseline(ctx context.Context, tx *sql.Tx, namespace string) (bool, error) {
	args := []interface{}{namespace}
	row, err := tx.QueryContext(ctx, `
		SELECT 1 FROM bytebase.migration_history WHERE namespace = ? AND `+"`type` = 'BASELINE'"+`
	`,
		args...,
	)

	if err != nil {
		return false, err
	}
	defer row.Close()

	if !row.Next() {
		return false, nil
	}

	return true, nil
}

func checkDuplicateVersion(ctx context.Context, tx *sql.Tx, namespace string, engine MigrationEngine, version string) (bool, error) {
	args := []interface{}{namespace, engine.String(), version}
	row, err := tx.QueryContext(ctx, `
		SELECT 1 FROM bytebase.migration_history WHERE namespace = ? AND `+"`engine` = ? AND version = ?"+`
	`,
		args...,
	)

	if err != nil {
		return false, err
	}
	defer row.Close()

	if row.Next() {
		return true, nil
	}
	return false, nil
}

func checkOutofOrderVersion(ctx context.Context, tx *sql.Tx, namespace string, engine MigrationEngine, version string) (*string, error) {
	args := []interface{}{namespace, engine.String(), version}
	row, err := tx.QueryContext(ctx, `
		SELECT MIN(version) FROM bytebase.migration_history WHERE namespace = ? AND `+"`engine` = ? AND STRCMP(?, version) = -1"+`
	`,
		args...,
	)

	if err != nil {
		return nil, err
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
	args := []interface{}{namespace}
	row, err := tx.QueryContext(ctx, `
		SELECT MAX(sequence) + 1 FROM bytebase.migration_history WHERE namespace = ?
	`,
		args...,
	)

	if err != nil {
		return -1, err
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
		return -1, fmt.Errorf("unable to generate next migration_sequence, no migration hisotry found for '%s', do you forget to baselining?", namespace)
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
