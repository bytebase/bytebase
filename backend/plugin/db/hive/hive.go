package hive

import (
	"context"
	"database/sql"
	"io"
	"strconv"
	"time"

	"github.com/beltran/gohive"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

type Driver struct {
	config   db.ConnectionConfig
	ctx      db.ConnectionContext
	dbClient *gohive.Connection
}

var (
	_ db.Driver = (*Driver)(nil)
)

func (d *Driver) Open(_ context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	// field legality check.
	if config.Username == "" {
		return nil, errors.Errorf("user not set")
	}
	if config.Host == "" {
		return nil, errors.Errorf("hostname not set")
	}
	d.config = config
	d.ctx = config.ConnectionContext

	// initialize database connection.
	configuration := gohive.NewConnectConfiguration()
	port, err := strconv.Atoi(config.Port)
	if err != nil {
		return nil, errors.Errorf("conversion failure for 'port' [string -> int]")
	}
	// TODO(tommy): actually there are various kinds of authentication to choose among [SASL, KERBEROS, NOSASL, PLAIN SASL]
	// "NONE" refers to PLAIN SASL that doesn't need authentication.
	authMethods := "NONE"
	conn, errConn := gohive.Connect(config.Host, port, authMethods, configuration)
	if errConn != nil {
		return nil, errors.Errorf("failed to establish connection")
	}
	d.dbClient = conn
	return d, nil
}

func (d *Driver) Close(_ context.Context) error {
	err := d.dbClient.Close()
	if err != nil {
		return errors.Errorf("faild to close connection")
	}
	return nil
}

func (d *Driver) Ping(ctx context.Context) error {
	if d.dbClient == nil {
		return errors.Errorf("database not connected")
	}
	if _, err := d.Execute(ctx, "SELECT 1", db.ExecuteOptions{}); err != nil {
		return errors.Errorf("bad connection")
	}
	return nil
}

func (*Driver) GetType() storepb.Engine {
	return storepb.Engine_HIVE
}

func (*Driver) GetDB() *sql.DB {
	return nil
}

func (d *Driver) Execute(ctx context.Context, statement string, _ db.ExecuteOptions) (int64, error) {
	var rowCount int64
	cursor := d.dbClient.Cursor()
	cursor.Execute(ctx, statement, false)
	operationStatus := cursor.Poll(false)

	if cursor.Err != nil {
		return 0, errors.Wrapf(cursor.Err, "failed to execute statement")
	}
	rowCount = operationStatus.GetNumModifiedRows()
	cursor.Close()
	return rowCount, nil
}

// Used for execute readonly SELECT statement.
func (d *Driver) QueryConn(ctx context.Context, _ *sql.Conn, statement string, _ *db.QueryContext) ([]*v1pb.QueryResult, error) {
	cursor := d.dbClient.Cursor()
	cursor.Exec(ctx, statement)
	if cursor.Err != nil {
		return nil, errors.Wrapf(cursor.Err, "failed to execute statement")
	}
	// TODO(tommy): func not implemented
	// var results []*v1pb.QueryResult

	// for cursor.HasMore(ctx) {
	// 	for columnName, value := range cursor.RowMap(ctx) {
	// 	}
	// 	cursor.FetchOne(ctx)
	// }

	return nil, errors.Errorf("Not implemeted")
}

// RunStatement will execute the statement and return the result, for both SELECT and non-SELECT statements.
func (*Driver) RunStatement(_ context.Context, _ *sql.Conn, _ string) ([]*v1pb.QueryResult, error) {
	return nil, errors.Errorf("Not implemeted")
}

// Sync schema
// SyncInstance syncs the instance metadata.
func (*Driver) SyncInstance(_ context.Context) (*db.InstanceMetadata, error) {
	return nil, errors.Errorf("Not implemeted")
}

// SyncDBSchema syncs a single database schema.
func (*Driver) SyncDBSchema(_ context.Context) (*storepb.DatabaseSchemaMetadata, error) {
	return nil, errors.Errorf("Not implemeted")
}

// Sync slow query logs
// SyncSlowQuery syncs the slow query logs.
// The returned map is keyed by database name, and the value is list of slow query statistics grouped by query fingerprint.
func (*Driver) SyncSlowQuery(_ context.Context, _ time.Time) (map[string]*storepb.SlowQueryStatistics, error) {
	return nil, errors.Errorf("Not implemeted")
}

// CheckSlowQueryLogEnabled checks if the slow query log is enabled.
func (*Driver) CheckSlowQueryLogEnabled(_ context.Context) error {
	return errors.Errorf("Not implemeted")
}

// Role
// CreateRole creates the role.
func (*Driver) CreateRole(_ context.Context, _ *db.DatabaseRoleUpsertMessage) (*db.DatabaseRoleMessage, error) {
	return nil, errors.Errorf("Not implemeted")
}

// UpdateRole updates the role.
func (*Driver) UpdateRole(_ context.Context, _ string, _ *db.DatabaseRoleUpsertMessage) (*db.DatabaseRoleMessage, error) {
	return nil, errors.Errorf("Not implemeted")
}

// FindRole finds the role by name.
func (*Driver) FindRole(_ context.Context, _ string) (*db.DatabaseRoleMessage, error) {
	return nil, errors.Errorf("Not implemeted")
}

// ListRole lists the role.
func (*Driver) ListRole(_ context.Context) ([]*db.DatabaseRoleMessage, error) {
	return nil, errors.Errorf("Not implemeted")
}

// DeleteRole deletes the role by name.
func (*Driver) DeleteRole(_ context.Context, _ string) error {
	return errors.Errorf("Not implemeted")
}

// Dump and restore
// Dump the database.
// The returned string is the JSON encoded metadata for the logical dump.
// For MySQL, the payload contains the binlog filename and position when the dump is generated.
func (*Driver) Dump(_ context.Context, _ io.Writer, _ bool) (string, error) {
	return "", errors.Errorf("Not implemeted")
}

// Restore the database from src, which is a full backup.
func (*Driver) Restore(_ context.Context, _ io.Reader) error {
	return errors.Errorf("Not implemeted")
}
