package mysql

import (
	"context"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	mysqldb "github.com/bytebase/bytebase/backend/plugin/db/mysql"
)

// createMySQLDriver creates and opens a MySQL driver for the specified database.
// This is a shared helper function for tests.
func createMySQLDriver(ctx context.Context, host, port, database string) (db.Driver, error) {
	driver := &mysqldb.Driver{}
	config := db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Type:     storepb.DataSourceType_ADMIN,
			Username: "root",
			Host:     host,
			Port:     port,
			Database: database,
		},
		Password: "root-password",
		ConnectionContext: db.ConnectionContext{
			EngineVersion: "8.0",
			DatabaseName:  database,
		},
	}
	return driver.Open(ctx, storepb.Engine_MYSQL, config)
}
