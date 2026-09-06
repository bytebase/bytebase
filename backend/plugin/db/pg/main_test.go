package pg

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
)

// The package shares one PostgreSQL 16 container, started on first use. Each
// test that needs an engine creates a database of its own on it, and names any
// role it creates after that database, so one container is enough and the
// tests run in parallel. Starting it lazily keeps `go test -run` over the unit
// tests from paying for a container they never open.
var (
	pgOnce        sync.Once
	pgContainer   *testcontainer.Container
	pgErr         error
	pgDatabaseSeq atomic.Int64
)

func TestMain(m *testing.M) {
	defer func() {
		if pgContainer != nil {
			pgContainer.Close(context.Background())
		}
	}()
	m.Run()
}

// sharedPgContainer returns the package's PostgreSQL container, starting it on
// the first call.
func sharedPgContainer(t *testing.T) *testcontainer.Container {
	t.Helper()
	pgOnce.Do(func() {
		container, err := testcontainer.GetPgContainer(context.Background())
		if err != nil {
			pgErr = err
			return
		}
		pgContainer = container
	})
	if pgErr != nil {
		t.Fatalf("failed to start the shared PostgreSQL container: %v", pgErr)
	}
	return pgContainer
}

// newTestDatabase creates a database for the test on the shared container and
// returns its name with a superuser handle to it.
func newTestDatabase(t *testing.T, container *testcontainer.Container) (string, *sql.DB) {
	t.Helper()
	name := fmt.Sprintf("test_%d", pgDatabaseSeq.Add(1))
	_, err := container.GetDB().Exec("CREATE DATABASE " + name)
	require.NoError(t, err)
	db, err := sql.Open("pgx", fmt.Sprintf("host=%s port=%s user=postgres password=root-password database=%s", container.GetHost(), container.GetPort(), name))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	return name, db
}
