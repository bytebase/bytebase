package trino

import (
	"context"
	"database/sql"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func init() {
	db.Register(storepb.Engine_CASSANDRA, newDriver)
}

type Driver struct {
	config db.ConnectionConfig
	db     *sql.DB
}

func newDriver(db.DriverConfig) db.Driver {
	return &Driver{}
}

func (d *Driver) Open(_ context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	addrs := []string{
		config.DataSource.Host + ":" + config.DataSource.Port,
	}
	for _, addr := range config.DataSource.AdditionalAddresses {
		addrs = append(addrs, addr.Host+":"+addr.Port)
	}

	return &Driver{
		config: config,
	}, nil
}

func (d *Driver) Close(context.Context) error {
	return nil
}

func (d *Driver) Ping(ctx context.Context) error {
	return nil
}

func (d *Driver) GetDB() *sql.DB {
	return d.db
}

func (*Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	return 0, errors.New("tbd")
}

func (*Driver) QueryConn(ctx context.Context, conn *sql.Conn, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	return nil, errors.New("tbd")
}
