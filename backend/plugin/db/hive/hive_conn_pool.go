package hive

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/beltran/gohive"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db"
)

type ConnPool interface {
	Get(dbName string) (*gohive.Connection, error)
	Put(*gohive.Connection)
	Destroy() error
}

type FixedConnPool struct {
	Host        string
	Port        int
	HiveConfig  *gohive.ConnectConfiguration
	SASLConfig  db.SASLConfig
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

	port, err := strconv.Atoi(config.Port)
	if err != nil {
		return nil, errors.Errorf("conversion failure for 'port' [string -> int]")
	}

	// SASL settings.
	switch t := config.SASLConfig.(type) {
	case *db.KerberosConfig:
		// Kerberos.
		hiveConfig.Hostname = config.Host
		hiveConfig.Service = t.Primary
	case *db.PlainSASLConfig:
		// Plain.
		hiveConfig.Username = t.Username
		hiveConfig.Password = t.Password
	default:
		return nil, errors.Errorf("invalid SASL config")
	}

	for i := 0; i < numMaxConn; i++ {
		conn, err := gohive.Connect(config.Host, port, config.SASLConfig.GetTypeName(), hiveConfig)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create Hive connection")
		}
		conns <- conn
	}

	return &FixedConnPool{
			Host:        config.Host,
			Port:        port,
			HiveConfig:  hiveConfig,
			SASLConfig:  config.SASLConfig,
			RWMutex:     sync.RWMutex{},
			Connections: conns,
			IsActivated: true,
			NumMaxConns: numMaxConn,
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
		var err error
		conn, err = gohive.Connect(pool.Host, pool.Port, pool.SASLConfig.GetTypeName(), pool.HiveConfig)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create Hive connection")
		}
	}

	if dbName != "" {
		cursor := conn.Cursor()
		cursor.Exec(context.Background(), fmt.Sprintf("use %s", dbName))
		if cursor.Err != nil {
			return nil, cursor.Err
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

	var errWhenClose error
	for conn := range pool.Connections {
		if err := conn.Close(); err != nil {
			errWhenClose = err
		}
	}

	return errWhenClose
}
