package db

import (
	"context"
	"fmt"
	"sync"

	"github.com/bytebase/bytebase"
)

type Type string

const (
	Mysql Type = "MYSQL"
)

func (e Type) String() string {
	switch e {
	case Mysql:
		return "MYSQL"
	}
	return ""
}

type DBSchema struct {
	Name string
}

var (
	driversMu sync.RWMutex
	drivers   = make(map[Type]DriverFunc)
)

type DriverConfig struct {
	Logger *bytebase.Logger
}

type DriverFunc func(DriverConfig) Driver

type Driver interface {
	open(config ConnectionConfig) (Driver, error)
	Ping(ctx context.Context) error
	SyncSchema(ctx context.Context) ([]*DBSchema, error)
}

type ConnectionConfig struct {
	Host     string
	Port     string
	Username string
	Password string
}

// Register makes a database driver available by the provided type.
// If Register is called twice with the same name or if driver is nil,
// it panics.
func register(dbType Type, f DriverFunc) {
	driversMu.Lock()
	defer driversMu.Unlock()
	if f == nil {
		panic("db: Register driver is nil")
	}
	if _, dup := drivers[dbType]; dup {
		panic("db: Register called twice for driver " + dbType)
	}
	drivers[dbType] = f
}

// Open opens a database specified by its database driver type and connection config
func Open(dbType Type, driverConfig DriverConfig, connectionConfig ConnectionConfig) (Driver, error) {
	driversMu.RLock()
	f, ok := drivers[dbType]
	driversMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("db: unknown driver %v", dbType)
	}

	return f(driverConfig).open(connectionConfig)
}
