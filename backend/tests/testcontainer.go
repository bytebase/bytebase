package tests

import (
	"context"
	"database/sql"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
)

// Container wraps testcontainer.Container and exposes fields for backward compatibility
type Container struct {
	*testcontainer.Container
	host string
	port string
	db   *sql.DB
}

// getPgContainer creates a PostgreSQL container for testing
func getPgContainer(ctx context.Context) (*Container, error) {
	tc, err := testcontainer.GetPgContainer(ctx)
	if err != nil {
		return nil, err
	}
	return &Container{
		Container: tc,
		host:      tc.GetHost(),
		port:      tc.GetPort(),
		db:        tc.GetDB(),
	}, nil
}

// getMySQLContainer creates a MySQL container for testing
func getMySQLContainer(ctx context.Context) (*Container, error) {
	tc, err := testcontainer.GetMySQLContainer(ctx)
	if err != nil {
		return nil, err
	}
	return &Container{
		Container: tc,
		host:      tc.GetHost(),
		port:      tc.GetPort(),
		db:        tc.GetDB(),
	}, nil
}
