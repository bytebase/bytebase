// Package redis implements redis driver
package redis

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"

	"github.com/bytebase/bytebase/backend/plugin/db"
)

var (
	_ db.Driver = (*Driver)(nil)
)

func init() {
	db.Register(db.Redis, newDriver)
}

// Driver is the redis driver.
type Driver struct {
	rdb redis.UniversalClient
}

func newDriver(_ db.DriverConfig) db.Driver {
	return &Driver{}
}

// Open opens the redis driver.
func (d *Driver) Open(ctx context.Context, _ db.Type, config db.ConnectionConfig, _ db.ConnectionContext) (db.Driver, error) {
	port := config.Port
	if port == "" {
		port = "6379"
	}
	addr := fmt.Sprintf("%s:%s", config.Host, port)
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

	rdb := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:     []string{addr},
		Username:  config.Username,
		Password:  config.Password,
		TLSConfig: tlsConfig,
		ReadOnly:  config.ReadOnly,
		DB:        db,
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	d.rdb = rdb
	return d, nil
}

// Close closes the redis driver.
func (d *Driver) Close(context.Context) error {
	return d.rdb.Close()
}

// Ping pings the redis server.
func (d *Driver) Ping(ctx context.Context) error {
	return d.rdb.Ping(ctx).Err()
}

// GetType returns redis.
func (*Driver) GetType() db.Type {
	return db.Redis
}

// GetDBConnection is not supported for redis.
func (*Driver) GetDBConnection(context.Context, string) (*sql.DB, error) {
	return nil, errors.New("redis: not supported")
}

// Execute will execute the statement. For CREATE DATABASE statement, some types of databases such as Postgres
// will not use transactions to execute the statement but will still use transactions to execute the rest of statements.
func (d *Driver) Execute(ctx context.Context, statement string, createDatabase bool) (int64, error) {
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
			var input []interface{}
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

// QueryConn executes the statement, returns the results.
func (d *Driver) QueryConn(ctx context.Context, _ *sql.Conn, statement string, _ *db.QueryContext) ([]interface{}, error) {
	lines := strings.Split(statement, "\n")
	for i := range lines {
		lines[i] = strings.Trim(lines[i], " \n\t\r")
	}

	var data []interface{}
	var cmds []*redis.Cmd

	if _, err := d.rdb.Pipelined(ctx, func(p redis.Pipeliner) error {
		for _, line := range lines {
			if line == "" {
				continue
			}
			var input []interface{}
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
		data = append(data, []interface{}{cmd.Val()})
	}

	return []interface{}{[]string{"result"}, []string{"TEXT"}, data}, nil
}

// Dump and restore

// Dump the database, if dbName is empty, then dump all databases.
// Redis is schemaless, we don't support dump Redis data currently.
func (*Driver) Dump(_ context.Context, _ string, _ io.Writer, schemaOnly bool) (string, error) {
	if !schemaOnly {
		return "", errors.New("redis: not supported")
	}
	return "", nil
}

// Restore the database from src, which is a full backup.
func (*Driver) Restore(context.Context, io.Reader) error {
	return errors.New("redis: not supported")
}
