package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// Container is a test container with the data-source helpers the tests build
// instances from.
type Container struct {
	*testcontainer.Container
}

// provisionPgInstance starts a Postgres of the test's own, closed when the test
// finishes. Use it for what sharedPgTarget cannot serve: a cluster-wide write
// such as a role, two instances that must be two servers, or a connection the
// test breaks on purpose.
func provisionPgInstance(t *testing.T) *Container {
	t.Helper()
	return provision(t, testcontainer.GetPgContainer)
}

// dataSource returns a data source of the given type and ID pointing at the container.
func (c *Container) dataSource(dsType v1pb.DataSourceType, id string) *v1pb.DataSource {
	return &v1pb.DataSource{
		Type:     dsType,
		Id:       id,
		Host:     c.GetHost(),
		Port:     c.GetPort(),
		Username: "postgres",
		Password: "root-password",
	}
}

// adminDataSource returns an ADMIN data source pointing at the container.
func (c *Container) adminDataSource() *v1pb.DataSource {
	return c.dataSource(v1pb.DataSourceType_ADMIN, "admin")
}

// sharedPgTarget is the Postgres the tests point instances at: one container
// for the package, a database per test. An instance on it must carry
// sync_databases, which createDatabase keeps current, or it syncs every other
// test's databases too.
func sharedPgTarget(t *testing.T) *Container {
	t.Helper()
	tc := testcontainer.SharedTargetPgContainer(t)
	return &Container{tc}
}

// provisionMySQLInstance is provisionPgInstance for MySQL.
func provisionMySQLInstance(t *testing.T) *Container {
	t.Helper()
	return provision(t, testcontainer.GetMySQLContainer)
}

func provision(t *testing.T, start func(context.Context) (*testcontainer.Container, error)) *Container {
	t.Helper()
	ctx := context.Background()
	tc, err := start(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { tc.Close(ctx) })
	return &Container{tc}
}
