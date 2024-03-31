package hive

import (
	"context"
	"fmt"
	"sync"

	"github.com/beltran/gohive"
	"github.com/pkg/errors"
)

type ConnPool interface {
	Get(dbName string) (*gohive.Connection, error)
	Put(*gohive.Connection)
	Destroy() error
}

type FixedConnPool struct {
	RWMutex         sync.RWMutex
	Connections     chan *gohive.Connection
	IsActivated     bool
	NumMaxConns     int
	PoolConfig      ConnPoolConfig
	HiveConnFactory func(ConnPoolConfig) (*gohive.Connection, error)
}

var _ ConnPool = &FixedConnPool{}

type ConnPoolConfig struct {
	Config *gohive.ConnectConfiguration
	Host   string
	Port   int
}

func PlainSASLConnFactory(poolConfig ConnPoolConfig) (*gohive.Connection, error) {
	conn, err := gohive.Connect(poolConfig.Host, poolConfig.Port, "NONE", poolConfig.Config)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to establish Hive connection")
	}
	return conn, nil
}

func CreateHiveConnPool(
	numMaxConn int,
	poolConfig ConnPoolConfig,
	hiveConnFactory func(ConnPoolConfig) (*gohive.Connection, error),
) (
	*FixedConnPool, error,
) {
	conns := make(chan *gohive.Connection, numMaxConn)

	for i := 0; i < numMaxConn; i++ {
		conn, err := hiveConnFactory(poolConfig)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create Hive connection pool")
		}
		conns <- conn
	}

	return &FixedConnPool{
			RWMutex:         sync.RWMutex{},
			Connections:     conns,
			IsActivated:     true,
			NumMaxConns:     numMaxConn,
			HiveConnFactory: hiveConnFactory,
			PoolConfig:      poolConfig,
		},
		nil
}

func (pool *FixedConnPool) Get(dbName string) (*gohive.Connection, error) {
	pool.RWMutex.RLock()
	if !pool.IsActivated {
		return nil, errors.New("connection pool has been closed")
	}
	pool.RWMutex.RUnlock()

	var conn *gohive.Connection

	select {
	case conn = <-pool.Connections:

	default:
		var err error
		conn, err = pool.HiveConnFactory(pool.PoolConfig)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create new connection")
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
		return errors.New("connection pool has been closed already")
	}
	pool.RWMutex.Unlock()

	for conn := range pool.Connections {
		conn.Close()
	}

	return nil
}
