package trino

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"time"

	"github.com/pkg/errors"
	// Import Trino driver for side effects
	_ "github.com/trinodb/trino-go-client/trino"

	"github.com/bytebase/bytebase/backend/plugin/db"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func init() {
	db.Register(storepb.Engine_TRINO, newDriver)
}

type Driver struct {
	config       db.ConnectionConfig
	db           *sql.DB
	databaseName string
}

func newDriver(db.DriverConfig) db.Driver {
	return &Driver{}
}

func (*Driver) Open(_ context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	// Construct Trino DSN
	scheme := "http"
	if config.DataSource.UseSsl {
		scheme = "https"
	}

	// Get user and password
	user := config.DataSource.Username
	if user == "" {
		user = "trino" // default user if not specified
	}

	password := config.Password

	// Set host and port
	host := config.DataSource.Host
	port := config.DataSource.Port
	if port == "" {
		port = "8080" // default Trino port
	}

	// Build URL with query parameters
	u := &url.URL{
		Scheme: scheme,
		Host:   fmt.Sprintf("%s:%s", host, port),
	}

	// Set user info based on whether password exists
	if password != "" {
		u.User = url.UserPassword(user, password)
	} else {
		u.User = url.User(user)
	}

	// Add query parameters
	query := u.Query()
	query.Add("source", "bytebase")

	database := config.DataSource.Database
	if config.ConnectionContext.DatabaseName != "" {
		database = config.ConnectionContext.DatabaseName
	}
	if database == "" {
		database = "system"
	}
	query.Add("catalog", database)
	u.RawQuery = query.Encode()

	// Get DSN from URL
	dsn := u.String()

	// Connect using the Trino driver
	db, err := sql.Open("trino", dsn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to Trino")
	}

	// Set connection pool parameters
	db.SetConnMaxLifetime(30 * time.Second)
	db.SetMaxIdleConns(10)
	db.SetMaxOpenConns(20)

	d := &Driver{
		config:       config,
		db:           db,
		databaseName: config.ConnectionContext.DatabaseName,
	}
	return d, nil
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
