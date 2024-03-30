package hive

import (
	"sync"

	"github.com/beltran/gohive"
	"github.com/pkg/errors"
)

type ConnPool interface {
	Get() (*gohive.Connection, error)
	Put(*gohive.Connection)
	Destroy() error
}

type FixedConnPool struct {
	RWMutex     sync.RWMutex
	Connections chan *gohive.Connection
	IsActivated bool
	NumMaxConns int
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
			RWMutex:     sync.RWMutex{},
			Connections: conns,
			IsActivated: true,
			NumMaxConns: numMaxConn,
		},
		nil
}

func (pool *FixedConnPool) Get() (*gohive.Connection, error) {
	pool.RWMutex.RLock()
	if !pool.IsActivated {
		return nil, errors.New("connection pool has been closed")
	}
	pool.RWMutex.RUnlock()
	return <-pool.Connections, nil
}

func (pool *FixedConnPool) Put(conn *gohive.Connection) {
	pool.RWMutex.RLock()
	if !pool.IsActivated {
		conn.Close()
		return
	}
	pool.RWMutex.RUnlock()
	pool.Connections <- conn
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
