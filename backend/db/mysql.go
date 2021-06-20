package db

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"strings"

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

	dsn := fmt.Sprintf("%s:%s@%s(%s:%s)/%s?%s", config.Username, config.Password, protocol, config.Host, config.Port, config.Database, strings.Join(params, "&"))
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
				ENGINE,
				TABLE_COLLATION,
				TABLE_ROWS,
				DATA_LENGTH,
				INDEX_LENGTH
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
			driver.l.Info(fmt.Sprintf("%v", err))
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

func (driver *MySQLDriver) Execute(ctx context.Context, sql string) (sql.Result, error) {
	tx, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	return tx.ExecContext(ctx, sql)
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
		if _, err := driver.Execute(ctx, migrationSchema); err != nil {
			driver.l.Error("Failed to initialize migration schema.", zap.Error(err))
			return err
		}
		driver.l.Info("Successfully created migration schema.")
	}

	return nil
}
