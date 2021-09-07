package pg

import (
	"bufio"
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"os"

	"github.com/bytebase/bytebase/plugin/db"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

var (
	_ db.Driver = (*Driver)(nil)
)

func init() {
	db.Register(db.Postgres, newDriver)
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
	return nil, fmt.Errorf("not implemented")
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
	return false, fmt.Errorf("not implemented")
}

func (driver *Driver) SetupMigrationIfNeeded(ctx context.Context) error {
	return fmt.Errorf("not implemented")
}

func (driver *Driver) ExecuteMigration(ctx context.Context, m *db.MigrationInfo, statement string) error {
	return fmt.Errorf("not implemented")
}

func (driver *Driver) FindMigrationHistoryList(ctx context.Context, find *db.MigrationHistoryFind) ([]*db.MigrationHistory, error) {
	return nil, fmt.Errorf("not implemented")
}

// Dump and restore
func (driver *Driver) Dump(ctx context.Context, database string, out *os.File, schemaOnly bool) error {
	return fmt.Errorf("not implemented")
}

func (driver *Driver) Restore(ctx context.Context, sc *bufio.Scanner) (err error) {
	return fmt.Errorf("not implemented")
}
