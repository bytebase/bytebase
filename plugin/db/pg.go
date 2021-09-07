package db

import (
	"bufio"
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"os"

	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

var (
	_ Driver = (*PostgresDriver)(nil)
)

func init() {
	register(Postgres, newPostgresDriver)
}

type PostgresDriver struct {
	l             *zap.Logger
	connectionCtx ConnectionContext

	db *sql.DB
}

func newPostgresDriver(config DriverConfig) Driver {
	return &PostgresDriver{
		l: config.Logger,
	}
}

func (driver *PostgresDriver) open(config ConnectionConfig, ctx ConnectionContext) (Driver, error) {
	return nil, fmt.Errorf("not implemented")
}

func (driver *PostgresDriver) Close(ctx context.Context) error {
	return driver.db.Close()
}

func (driver *PostgresDriver) Ping(ctx context.Context) error {
	return driver.db.PingContext(ctx)
}

func (driver *PostgresDriver) SyncSchema(ctx context.Context) ([]*DBUser, []*DBSchema, error) {
	return nil, nil, fmt.Errorf("not implemented")
}

func (driver *PostgresDriver) Execute(ctx context.Context, statement string) error {
	tx, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, statement)
	return err
}

// Migration related
func (driver *PostgresDriver) NeedsSetupMigration(ctx context.Context) (bool, error) {
	return false, fmt.Errorf("not implemented")
}

func (driver *PostgresDriver) SetupMigrationIfNeeded(ctx context.Context) error {
	return fmt.Errorf("not implemented")
}

func (driver *PostgresDriver) ExecuteMigration(ctx context.Context, m *MigrationInfo, statement string) error {
	return fmt.Errorf("not implemented")
}

func (driver *PostgresDriver) FindMigrationHistoryList(ctx context.Context, find *MigrationHistoryFind) ([]*MigrationHistory, error) {
	return nil, fmt.Errorf("not implemented")
}

// Dump and restore
func (driver *PostgresDriver) Dump(ctx context.Context, database string, out *os.File, schemaOnly bool) error {
	return fmt.Errorf("not implemented")
}

func (driver *PostgresDriver) Restore(ctx context.Context, sc *bufio.Scanner) (err error) {
	return fmt.Errorf("not implemented")
}
