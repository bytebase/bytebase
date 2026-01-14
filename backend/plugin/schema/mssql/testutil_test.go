package mssql

import (
	"context"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	mssqldb "github.com/bytebase/bytebase/backend/plugin/db/mssql"
)

// createMSSQLDriver creates and opens a MSSQL driver connection
func createMSSQLDriver(ctx context.Context, host, port, database string) (db.Driver, error) {
	driver := &mssqldb.Driver{}
	config := db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Type:     storepb.DataSourceType_ADMIN,
			Username: "sa",
			Host:     host,
			Port:     port,
			Database: database,
		},
		Password: "Test123!",
		ConnectionContext: db.ConnectionContext{
			DatabaseName: database,
		},
	}
	return driver.Open(ctx, storepb.Engine_MSSQL, config)
}
