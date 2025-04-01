package cassandra

import (
	"context"
	"database/sql"

	"github.com/gocql/gocql"
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

	session *gocql.Session
}

func newDriver(db.DriverConfig) db.Driver {
	return &Driver{}
}

func (*Driver) Open(_ context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	addrs := []string{
		config.DataSource.Host + ":" + config.DataSource.Port,
	}
	for _, addr := range config.DataSource.AdditionalAddresses {
		addrs = append(addrs, addr.Host+":"+addr.Port)
	}
	cluster := gocql.NewCluster(addrs...)
	cluster.Authenticator = gocql.PasswordAuthenticator{
		Username: config.DataSource.Username,
		Password: config.Password,
	}
	cluster.Keyspace = config.ConnectionContext.DatabaseName

	session, err := cluster.CreateSession()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create session")
	}

	return &Driver{
		config:  config,
		session: session,
	}, nil
}

func (d *Driver) Close(context.Context) error {
	if d.session != nil {
		d.session.Close()
	}
	return nil
}

func (d *Driver) Ping(ctx context.Context) error {
	var version string
	err := d.session.Query("SELECT release_version FROM system.local").WithContext(ctx).Scan(&version)
	if err != nil {
		return errors.Wrapf(err, "failed to ping")
	}
	return nil
}

func (*Driver) GetDB() *sql.DB {
	panic("GetDB() not supported for cassandra")
}

func (*Driver) Execute(context.Context, string, db.ExecuteOptions) (int64, error) {
	return 0, errors.New("tbd")
}
func (*Driver) QueryConn(context.Context, *sql.Conn, string, db.QueryContext) ([]*v1pb.QueryResult, error) {
	return nil, errors.New("tbd")
}
