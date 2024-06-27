// Package redis implements redis driver
package redis

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/google/shlex"
	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
	"go.uber.org/multierr"
	"golang.org/x/crypto/ssh"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

var (
	_ db.Driver = (*Driver)(nil)
)

func init() {
	db.Register(storepb.Engine_REDIS, newDriver)
}

// Driver is the redis driver.
type Driver struct {
	rdb          redis.UniversalClient
	sshClient    *ssh.Client
	databaseName string
}

func newDriver(_ db.DriverConfig) db.Driver {
	return &Driver{}
}

// Open opens the redis driver.
func (d *Driver) Open(_ context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	addr := fmt.Sprintf("%s:%s", config.Host, config.Port)
	tlsConfig, err := config.TLSConfig.GetSslConfig()
	if err != nil {
		return nil, errors.Wrap(err, "redis: failed to get tls config")
	}

	// connect to 0 by default
	db := 0
	if config.Database != "" {
		database, err := strconv.Atoi(config.Database)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert database %s to int", config.Database)
		}
		db = database
	}
	d.databaseName = fmt.Sprintf("%d", db)

	options := &redis.UniversalOptions{
		Addrs:     []string{addr},
		Username:  config.Username,
		Password:  config.Password,
		TLSConfig: tlsConfig,
		ReadOnly:  config.ReadOnly,
		DB:        db,
	}
	if config.SSHConfig.Host != "" {
		sshClient, err := util.GetSSHClient(config.SSHConfig)
		if err != nil {
			return nil, err
		}
		d.sshClient = sshClient

		options.Dialer = func(_ context.Context, network, addr string) (net.Conn, error) {
			conn, err := sshClient.Dial(network, addr)
			if err != nil {
				return nil, err
			}
			return &noDeadlineConn{Conn: conn}, nil
		}
	}
	d.rdb = redis.NewUniversalClient(options)

	return d, nil
}

type noDeadlineConn struct{ net.Conn }

func (*noDeadlineConn) SetDeadline(time.Time) error      { return nil }
func (*noDeadlineConn) SetReadDeadline(time.Time) error  { return nil }
func (*noDeadlineConn) SetWriteDeadline(time.Time) error { return nil }

// Close closes the redis driver.
func (d *Driver) Close(context.Context) error {
	var err error
	err = multierr.Append(err, d.rdb.Close())
	if d.sshClient != nil {
		err = multierr.Append(err, d.sshClient.Close())
	}
	return err
}

// Ping pings the redis server.
func (d *Driver) Ping(ctx context.Context) error {
	return d.rdb.Ping(ctx).Err()
}

// GetType returns redis.
func (*Driver) GetType() storepb.Engine {
	return storepb.Engine_REDIS
}

// GetDB gets the database.
func (*Driver) GetDB() *sql.DB {
	return nil
}

// Execute will execute the statement. For CREATE DATABASE statement, some types of databases such as Postgres
// will not use transactions to execute the statement but will still use transactions to execute the rest of statements.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	if opts.CreateDatabase {
		return 0, errors.New("redis: cannot create database")
	}

	lines := strings.Split(statement, "\n")
	if _, err := d.rdb.Pipelined(ctx, func(p redis.Pipeliner) error {
		for _, line := range lines {
			fields, err := shlex.Split(line)
			if err != nil {
				return errors.Wrapf(err, "failed to split command %s", line)
			}
			if len(fields) == 0 {
				continue
			}
			var input []any
			for _, v := range fields {
				input = append(input, v)
			}
			_ = p.Do(ctx, input...)
		}
		return nil
	}); err != nil && err != redis.Nil {
		return 0, err
	}

	return 0, nil
}

// Dump and restore
// Dump the database, if dbName is empty, then dump all databases.
// Redis is schemaless, we don't support dump Redis data currently.
func (*Driver) Dump(_ context.Context, _ io.Writer) (string, error) {
	return "", nil
}

// QueryConn queries a SQL statement in a given connection.
func (d *Driver) QueryConn(ctx context.Context, _ *sql.Conn, statement string, queryContext *db.QueryContext) ([]*v1pb.QueryResult, error) {
	if queryContext != nil && queryContext.Explain {
		return nil, errors.New("MongoDB does not support EXPLAIN")
	}

	startTime := time.Now()
	lines := strings.Split(statement, "\n")
	var cmds []*redis.Cmd
	if _, err := d.rdb.Pipelined(ctx, func(p redis.Pipeliner) error {
		for _, line := range lines {
			fields, err := shlex.Split(line)
			if err != nil {
				return errors.Wrapf(err, "failed to split command %s", line)
			}
			if len(fields) == 0 {
				continue
			}
			var input []any
			for _, v := range fields {
				input = append(input, v)
			}
			cmd := p.Do(ctx, input...)
			cmds = append(cmds, cmd)
		}
		return nil
	}); err != nil && err != redis.Nil {
		return nil, err
	}

	var queryResult []*v1pb.QueryResult
	for i, cmd := range cmds {
		if cmd.Err() == redis.Nil {
			queryResult = append(queryResult, &v1pb.QueryResult{
				ColumnNames:     []string{"#", "Value"},
				ColumnTypeNames: []string{"INT", "TEXT"},
				Rows: []*v1pb.QueryRow{
					{
						Values: []*v1pb.RowValue{
							{Kind: &v1pb.RowValue_Int32Value{Int32Value: 1}},
							{Kind: &v1pb.RowValue_NullValue{}},
						},
					},
				},
				Latency:   durationpb.New(time.Since(startTime)),
				Statement: lines[i],
			})
			continue
		}

		// RowValue cannot handle interface{} type
		result := &v1pb.QueryResult{
			ColumnNames:     []string{"#", "Value"},
			ColumnTypeNames: []string{"INT", "TEXT"},
			Statement:       lines[i],
		}
		setQueryResultRows(result, cmd)
		result.Latency = durationpb.New(time.Since(startTime))

		queryResult = append(queryResult, result)
	}

	return queryResult, nil
}

func setQueryResultRows(result *v1pb.QueryResult, cmd *redis.Cmd) {
	val := cmd.Val()
	l, ok := val.([]any)
	if ok {
		for i, v := range l {
			result.Rows = append(result.Rows, getResultRow(i+1, v))
			n := len(result.Rows)
			if (n&(n-1) == 0) && proto.Size(result) > common.MaximumSQLResultSize {
				result.Error = common.MaximumSQLResultSizeExceeded
				return
			}
		}
	} else {
		result.Rows = append(result.Rows, getResultRow(1, val))
	}
}

func getResultRow(i int, v any) *v1pb.QueryRow {
	s := fmt.Sprintf("%v", v)
	return &v1pb.QueryRow{Values: []*v1pb.RowValue{
		{Kind: &v1pb.RowValue_Int32Value{Int32Value: int32(i)}},
		{Kind: &v1pb.RowValue_StringValue{StringValue: s}}},
	}
}

// RunStatement runs a SQL statement in a given connection.
func (d *Driver) RunStatement(ctx context.Context, _ *sql.Conn, statement string) ([]*v1pb.QueryResult, error) {
	return d.QueryConn(ctx, nil, statement, nil)
}
