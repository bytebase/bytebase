package trino

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"

	"github.com/pkg/errors"
	// Import the Trino driver to register it with database/sql
	_ "github.com/trinodb/trino-go-client/trino"

	"github.com/bytebase/bytebase/backend/plugin/db"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func init() {
	db.Register(storepb.Engine_TRINO, newDriver)
}

type Driver struct {
	config db.ConnectionConfig
	db     *sql.DB
}

func newDriver(db.DriverConfig) db.Driver {
	return &Driver{}
}

func (*Driver) Open(_ context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	// Construct Trino DSN
	var scheme string
	if config.DataSource.UseSsl {
		scheme = "https"
	} else {
		scheme = "http"
	}

	// Build query parameters
	queryParams := url.Values{}
	// Add catalog if specified in config
	if config.DataSource.Database != "" {
		queryParams.Add("catalog", config.DataSource.Database)
	}

	// Add user and password
	user := config.DataSource.Username
	if user == "" {
		user = "trino" // default user if not specified
	}

	password := config.Password
	if password == "" {
		password = config.DataSource.Password // try to get from DataSource if not provided directly
	}

	host := config.DataSource.Host
	port := config.DataSource.Port
	if port == "" {
		port = "8080" // default Trino port
	}

	// Construct DSN string with user and password if provided
	var dsn string
	if password != "" {
		dsn = fmt.Sprintf("%s://%s:%s@%s:%s", scheme, user, url.QueryEscape(password), host, port)
	} else {
		dsn = fmt.Sprintf("%s://%s@%s:%s", scheme, user, host, port)
	}
	if len(queryParams) > 0 {
		dsn = dsn + "?" + queryParams.Encode()
	}

	// Open connection
	db, err := sql.Open("trino", dsn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to Trino")
	}

	return &Driver{
		config: config,
		db:     db,
	}, nil
}

func (d *Driver) Close(context.Context) error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

func (d *Driver) Ping(ctx context.Context) error {
	if d.db != nil {
		return d.db.PingContext(ctx)
	}
	return errors.New("database connection not established")
}

func (d *Driver) GetDB() *sql.DB {
	return d.db
}

// func (*Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
func (*Driver) Execute(_ context.Context, _ string, _ db.ExecuteOptions) (int64, error) {
	return 0, errors.New("tbd")
}

// func (*Driver) QueryConn(ctx context.Context, conn *sql.Conn, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
func (*Driver) QueryConn(_ context.Context, _ *sql.Conn, _ string, _ db.QueryContext) ([]*v1pb.QueryResult, error) {
	return nil, errors.New("tbd")
}
