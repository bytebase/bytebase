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

	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

var (
	_ db.Driver = (*Driver)(nil)
)

func init() {
	db.Register(db.Redis, newDriver)
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
func (d *Driver) Open(ctx context.Context, _ db.Type, config db.ConnectionConfig, _ db.ConnectionContext) (db.Driver, error) {
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

		options.Dialer = func(ctx context.Context, network, addr string) (net.Conn, error) {
			conn, err := sshClient.Dial(network, addr)
			if err != nil {
				return nil, err
			}
			return &noDeadlineConn{Conn: conn}, nil
		}
	}
	d.rdb = redis.NewUniversalClient(options)

	clusterEnabled, err := d.getClusterEnabled(ctx)
	if err != nil {
		return nil, err
	}

	// switch to cluster if cluster is enabled.
	if clusterEnabled {
		if err := d.rdb.Close(); err != nil {
			log.Warn("failed to close redis driver when switching to redis cluster driver", zap.Error(err))
		}
		d.rdb = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:     []string{addr},
			Username:  config.Username,
			Password:  config.Password,
			TLSConfig: tlsConfig,
			ReadOnly:  config.ReadOnly,
		})
	}

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
func (*Driver) GetType() db.Type {
	return db.Redis
}

// GetDB gets the database.
func (*Driver) GetDB() *sql.DB {
	return nil
}

// Execute will execute the statement. For CREATE DATABASE statement, some types of databases such as Postgres
// will not use transactions to execute the statement but will still use transactions to execute the rest of statements.
func (d *Driver) Execute(ctx context.Context, statement string, createDatabase bool, _ db.ExecuteOptions) (int64, error) {
	if createDatabase {
		return 0, errors.New("redis: cannot create database")
	}

	lines := strings.Split(statement, "\n")
	for i := range lines {
		lines[i] = strings.Trim(lines[i], " \n\t\r")
	}

	if _, err := d.rdb.Pipelined(ctx, func(p redis.Pipeliner) error {
		for _, line := range lines {
			if line == "" {
				continue
			}
			var input []any
			for _, s := range strings.Split(line, " ") {
				input = append(input, s)
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
func (*Driver) Dump(_ context.Context, _ io.Writer, schemaOnly bool) (string, error) {
	if !schemaOnly {
		return "", errors.New("redis: not supported")
	}
	return "", nil
}

// Restore the database from src, which is a full backup.
func (*Driver) Restore(context.Context, io.Reader) error {
	return errors.New("redis: not supported")
}

// QueryConn queries a SQL statement in a given connection.
func (d *Driver) QueryConn(ctx context.Context, _ *sql.Conn, statement string, _ *db.QueryContext) ([]*v1pb.QueryResult, error) {
	startTime := time.Now()
	lines := strings.Split(statement, "\n")
	for i := range lines {
		lines[i] = strings.Trim(lines[i], " \n\t\r")
	}

	var data []*v1pb.QueryRow
	var cmds []*redis.Cmd

	if _, err := d.rdb.Pipelined(ctx, func(p redis.Pipeliner) error {
		for _, line := range lines {
			if line == "" {
				continue
			}
			var input []any
			for _, s := range strings.Split(line, " ") {
				input = append(input, s)
			}
			cmd := p.Do(ctx, input...)
			cmds = append(cmds, cmd)
		}
		return nil
	}); err != nil && err != redis.Nil {
		return nil, err
	}

	for _, cmd := range cmds {
		if cmd.Err() == redis.Nil {
			data = append(data, &v1pb.QueryRow{Values: []*v1pb.RowValue{{Kind: &v1pb.RowValue_StringValue{StringValue: "redis: nil"}}}})
			continue
		}

		// RowValue cannot handle interface{} type
		val := cmd.String()
		data = append(data, &v1pb.QueryRow{Values: []*v1pb.RowValue{{Kind: &v1pb.RowValue_StringValue{StringValue: val}}}})
	}

	return []*v1pb.QueryResult{{
		ColumnNames:     []string{"result"},
		ColumnTypeNames: []string{"TEXT"},
		Rows:            data,
		Latency:         durationpb.New(time.Since(startTime)),
		Statement:       statement,
	}}, nil
}

// RunStatement runs a SQL statement in a given connection.
func (d *Driver) RunStatement(ctx context.Context, _ *sql.Conn, statement string) ([]*v1pb.QueryResult, error) {
	return d.QueryConn(ctx, nil, statement, nil)
}
