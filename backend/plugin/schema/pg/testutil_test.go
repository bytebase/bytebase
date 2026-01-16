package pg

import (
	"context"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	pgdb "github.com/bytebase/bytebase/backend/plugin/db/pg"
)

// createPgDriver creates and opens a PostgreSQL driver for the specified database.
// This is a shared helper function for tests.
func createPgDriver(ctx context.Context, host, port, database string) (*pgdb.Driver, error) {
	driver := &pgdb.Driver{}
	config := db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Type:     storepb.DataSourceType_ADMIN,
			Username: "postgres",
			Host:     host,
			Port:     port,
			Database: database,
		},
		Password: "root-password",
		ConnectionContext: db.ConnectionContext{
			EngineVersion: "16.0",
			DatabaseName:  database,
		},
	}
	_, err := driver.Open(ctx, storepb.Engine_POSTGRES, config)
	if err != nil {
		return nil, err
	}
	return driver, nil
}
