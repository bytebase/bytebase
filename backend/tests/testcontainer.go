package tests

import (
	"context"
	"database/sql"
	"testing"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
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

// provisionPgInstance starts a dedicated Postgres container that backs a test
// database instance. The container is closed when the test finishes.
func provisionPgInstance(ctx context.Context, t *testing.T) (*Container, error) {
	c, err := getPgContainer(ctx)
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { c.Close(ctx) })
	return c, nil
}

// dataSource returns a data source of the given type and ID pointing at the container.
func (c *Container) dataSource(dsType v1pb.DataSourceType, id string) *v1pb.DataSource {
	return &v1pb.DataSource{
		Type:     dsType,
		Id:       id,
		Host:     c.host,
		Port:     c.port,
		Username: "postgres",
		Password: "root-password",
	}
}

// adminDataSource returns an ADMIN data source pointing at the container.
func (c *Container) adminDataSource() *v1pb.DataSource {
	return c.dataSource(v1pb.DataSourceType_ADMIN, "admin")
}

// getMySQLContainer creates a MySQL container for testing
func getMySQLContainer(ctx context.Context) (*Container, error) {
	tc, err := testcontainer.GetTestMySQLContainer(ctx)
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
