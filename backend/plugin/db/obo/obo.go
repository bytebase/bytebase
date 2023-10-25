// Package obo is for OceanBase Oracle mode
package obo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"

	// Register OceanBase Oracle mode driver.
	_ "github.com/mattn/go-oci8"
)

func init() {
	db.Register(storepb.Engine_OCEANBASE_ORACLE, newDriver)
}

type Driver struct {
	db           *sql.DB
	databaseName string
}

func newDriver(db.DriverConfig) db.Driver {
	return &Driver{}
}

func (driver *Driver) Open(_ context.Context, _ storepb.Engine, config db.ConnectionConfig, _ db.ConnectionContext) (db.Driver, error) {
	databaseName := func() string {
		if config.Database != "" {
			return config.Database
		}
		i := strings.Index(config.Username, "@")
		if i == -1 {
			return config.Username
		}
		return config.Username[:i]
	}()

	// usename format: {user}@{tenant}#{cluster}
	// user is required, others are optional.
	dsn := fmt.Sprintf("%s/%s@%s:%s/%s", config.Username, config.Password, config.Host, config.Port, databaseName)

	db, err := sql.Open("oci8", dsn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open connection")
	}

	driver.db = db
	driver.databaseName = databaseName
	return driver, nil
}

func (driver *Driver) Close(context.Context) error {
	return driver.db.Close()
}

func (driver *Driver) Ping(ctx context.Context) error {
	return driver.db.PingContext(ctx)
}

func (*Driver) GetType() storepb.Engine {
	return storepb.Engine_OCEANBASE_ORACLE
}

func (*Driver) GetDB() *sql.DB {
	return nil
}

func (*Driver) Execute(context.Context, string, bool, db.ExecuteOptions) (int64, error) {
	return 0, errors.New("not implemented")
}

func (*Driver) QueryConn(context.Context, *sql.Conn, string, *db.QueryContext) ([]*v1pb.QueryResult, error) {
	return nil, errors.New("not implemented")
}

func (*Driver) RunStatement(context.Context, *sql.Conn, string) ([]*v1pb.QueryResult, error) {
	return nil, errors.New("not implemented")
}
