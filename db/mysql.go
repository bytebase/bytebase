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

func (driver *MySQLDriver) Ping(ctx context.Context) error {
	return driver.db.PingContext(ctx)
}

func (driver *MySQLDriver) SyncSchema(ctx context.Context) ([]*DBSchema, error) {
	excludedDatabaseList := []string{
		"'mysql'",
		"'information_schema'",
		"'performance_schema'",
		"'sys'",
	}

	// Query table info
	tableWhere := fmt.Sprintf("TABLE_SCHEMA NOT IN (%s)", strings.Join(excludedDatabaseList, ", "))
	tableRows, err := driver.db.QueryContext(ctx, `
			SELECT
				TABLE_SCHEMA, 
				TABLE_NAME,
				UNIX_TIMESTAMP(CREATE_TIME),
				IFNULL(UNIX_TIMESTAMP(UPDATE_TIME), 0),
				IFNULL(ENGINE, ''),
				IFNULL(TABLE_COLLATION, ''),
				IFNULL(TABLE_ROWS, 0),
				IFNULL(DATA_LENGTH, 0),
				IFNULL(INDEX_LENGTH, 0)
			FROM information_schema.TABLES
			WHERE `+tableWhere,
	)
	if err != nil {
		return nil, err
	}

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
			&table.Engine,
			&table.Collation,
			&table.RowCount,
			&table.DataSize,
			&table.IndexSize,
		); err != nil {
			return nil, err
		}

		tableList, ok := tableMap[dbName]
		if ok {
			tableMap[dbName] = append(tableList, table)
		} else {
			list := make([]DBTable, 0)
			tableMap[dbName] = append(list, table)
		}
	}
	tableRows.Close()

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
		return nil, err
	}
	defer rows.Close()

	list := make([]*DBSchema, 0)
	for rows.Next() {
		var schema DBSchema
		if err := rows.Scan(
			&schema.Name,
			&schema.CharacterSet,
			&schema.Collation,
		); err != nil {
			return nil, err
		}

		schema.TableList = tableMap[schema.Name]

		list = append(list, &schema)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return list, err
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
	duplicate, err := checkDuplicateVersion(ctx, tx, m.Namespace, m.Version)
	if err != nil {
		return err
	}
	if duplicate {
		return fmt.Errorf("%s has already applied version %s", m.Database, m.Version)
	}

	// Check if there is any higher version already been applied
	version, err := checkOutofOrderVersion(ctx, tx, m.Namespace, m.Version)
	if err != nil {
		return err
	}
	if version != nil {
		return fmt.Errorf("%s has already applied version %s which is higher than %s", m.Database, *version, m.Version)
	}

	// If the migration type is not baseline, then we can only proceed if there is existing baseline
	// This check is also wrapped in transaction to avoid edge case where two baselings are running concurrently.
	if m.Type != Baseline {
		hasBaseline, err := findBaseline(ctx, tx, m.Namespace)
		if err != nil {
			return err
		}

		if !hasBaseline {
			return fmt.Errorf("%s has not created migration baseline yet", m.Database)
		}
	}

	sequence, err := findNextSequence(ctx, tx, m.Namespace, m.Type == Baseline)
	if err != nil {
		return err
	}

	// Phase 2 - Executing migration unless it's baselining and has empty statement
	// Baseline can have empty statement
	if !(m.Type == Baseline && statement == "") {
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
			`+"`type`,"+`
			version,
			description,
			statement,
			execution_duration
		)
		VALUES (?, unix_timestamp(), ?, unix_timestamp(), ?, ?, ?, ?, ?, ?, ?)
	`,
		m.Creator,
		m.Creator,
		m.Namespace,
		sequence,
		m.Type,
		m.Version,
		m.Description,
		statement,
		time.Now().Unix()-startedTs,
	)

	if err != nil {
		return formatError(err)
	}

	tx.Commit()

	return nil
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

func checkDuplicateVersion(ctx context.Context, tx *sql.Tx, namespace string, version string) (bool, error) {
	args := []interface{}{namespace, version}
	row, err := tx.QueryContext(ctx, `
		SELECT 1 FROM bytebase.migration_history WHERE namespace = ? AND version = ?
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

func checkOutofOrderVersion(ctx context.Context, tx *sql.Tx, namespace string, version string) (*string, error) {
	args := []interface{}{namespace, version}
	row, err := tx.QueryContext(ctx, `
		SELECT MIN(version) FROM bytebase.migration_history WHERE namespace = ? AND STRCMP(?, version) = -1
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

func findNextSequence(ctx context.Context, tx *sql.Tx, namespace string, baseline bool) (int, error) {
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
		// Returns 1 if we are creating the first baseline
		if baseline {
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
