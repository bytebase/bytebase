package hive

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/beltran/gohive"
	"github.com/pkg/errors"
	"go.uber.org/multierr"

	"github.com/bytebase/bytebase/backend/plugin/db"
)

type ConnPool interface {
	Get(dbName string) (*gohive.Connection, error)
	Put(*gohive.Connection)
	Destroy() error
}

type FixedConnPool struct {
	BasicConfig *db.ConnectionConfig
	HiveConfig  *gohive.ConnectConfiguration
	Port        int
	RWMutex     sync.RWMutex
	Connections chan *gohive.Connection
	IsActivated bool
	NumMaxConns int
}

var _ ConnPool = &FixedConnPool{}

func CreateHiveConnPool(
	numMaxConn int,
	config *db.ConnectionConfig,
) (
	*FixedConnPool, error,
) {
	conns := make(chan *gohive.Connection, numMaxConn)
	hiveConfig := gohive.NewConnectConfiguration()
	hiveConfig.Database = config.Database

	switch t := config.SASLConfig.(type) {
	case *db.KerberosConfig:
		hiveConfig.Hostname = t.Instance
		hiveConfig.Service = t.Primary

	case *db.PlainSASLConfig:
		hiveConfig.Username = t.Username
		hiveConfig.Password = t.Password

	default:
		return nil, errors.Errorf("invalid SASL config")
	}

	port, err := strconv.Atoi(config.Port)
	if err != nil {
		return nil, errors.Errorf("conversion failure for 'port' [string -> int]")
	}

	saslTypName := config.SASLConfig.GetTypeName()
	if saslTypName == db.SASLTypeKerberos {
		db.KrbEnvLock()
		defer db.KrbEnvUnlock()
		if err := config.SASLConfig.InitEnv(); err != nil {
			return nil, errors.Wrapf(err, "failed to init SASL environment")
		}
	}

	for i := 0; i < numMaxConn; i++ {
		conn, err := gohive.Connect(config.Host, port, string(saslTypName), hiveConfig)
		// release resources if err.
		if err != nil || conn == nil {
			errs := multierr.Combine(errors.New("failed to establish Hive connection"), err)
			close(conns)
			for conn := range conns {
				errs = multierr.Combine(conn.Close(), errs)
			}
			return nil, errs
		}
		conns <- conn
	}

	return &FixedConnPool{
			RWMutex:     sync.RWMutex{},
			Connections: conns,
			IsActivated: true,
			NumMaxConns: numMaxConn,
			BasicConfig: config,
			HiveConfig:  hiveConfig,
			Port:        port,
		},
		nil
}

func (pool *FixedConnPool) Get(dbName string) (*gohive.Connection, error) {
	pool.RWMutex.RLock()
	if !pool.IsActivated {
		pool.RWMutex.RUnlock()
		return nil, errors.New("connection pool has been closed")
	}
	pool.RWMutex.RUnlock()

	var conn *gohive.Connection

	select {
	case conn = <-pool.Connections:

	default:
		saslTypName := pool.BasicConfig.SASLConfig.GetTypeName()
		if saslTypName == db.SASLTypeKerberos {
			db.KrbEnvLock()
			defer db.KrbEnvUnlock()
			if err := pool.BasicConfig.SASLConfig.InitEnv(); err != nil {
				return nil, errors.Wrapf(err, "failed to init SASL environment")
			}
		}
		var err error
		conn, err = gohive.Connect(pool.BasicConfig.Host, pool.Port, string(saslTypName), pool.HiveConfig)
		if err != nil || conn == nil {
			return nil, errors.Wrapf(err, "failed to get Hive connection")
		}
	}

	if dbName != "" {
		cursor := conn.Cursor()
		cursor.Exec(context.Background(), fmt.Sprintf("use %s", dbName))
		if cursor.Err != nil {
			return nil, multierr.Combine(conn.Close(), cursor.Err)
		}
	}

	return conn, nil
}

func (pool *FixedConnPool) Put(conn *gohive.Connection) {
	pool.RWMutex.RLock()
	if !pool.IsActivated {
		conn.Close()
		pool.RWMutex.RUnlock()
		return
	}
	pool.RWMutex.RUnlock()
	select {
	case pool.Connections <- conn:
		return
	default:
		conn.Close()
	}
}

func (pool *FixedConnPool) Destroy() error {
	pool.RWMutex.Lock()
	if pool.IsActivated {
		pool.IsActivated = false
		close(pool.Connections)
	} else {
		pool.RWMutex.Unlock()
		return errors.New("connection pool has been closed already")
	}
	pool.RWMutex.Unlock()

	var errs error
	for conn := range pool.Connections {
		if err := conn.Close(); err != nil {
			errs = multierr.Combine(err, errs)
		}
	}
	return errs
}
