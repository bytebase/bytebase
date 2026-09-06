package testcontainer

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

// One container per test package, started on first use and stopped by Main.
// A test binary is its own process, so package-level state here is per
// package. Starting lazily is what keeps `go test -run` over the tests that
// need no engine from paying for a container they never open.
var (
	sharedPg     = startOnce(GetPgContainer)
	sharedMySQL  = startOnce(GetTestMySQLContainer)
	sharedOracle = startOnce(GetOracleContainer)
	sharedMSSQL  = startOnce(GetMSSQLContainer)
	sharedTiDB   = startOnce(GetTiDBContainer)

	startedMu sync.Mutex
	started   []*Container

	pgDatabaseSeq atomic.Int64
)

func startOnce(start func(context.Context) (*Container, error)) func() (*Container, error) {
	return sync.OnceValues(func() (*Container, error) {
		c, err := start(context.Background())
		if err == nil {
			startedMu.Lock()
			started = append(started, c)
			startedMu.Unlock()
		}
		return c, err
	})
}

func shared(t testing.TB, start func() (*Container, error)) *Container {
	t.Helper()
	c, err := start()
	require.NoError(t, err, "start the shared container")
	return c
}

// Main runs the package's tests and stops the containers they shared:
//
//	func TestMain(m *testing.M) { testcontainer.Main(m) }
func Main(m *testing.M) {
	defer func() {
		for _, c := range started {
			c.Close(context.Background())
		}
	}()
	m.Run()
}

// SharedPgContainer returns the package's PostgreSQL container, starting it on
// the first call. Tests isolate themselves with NewPgDatabase or NewMetadataDB.
func SharedPgContainer(t testing.TB) *Container { return shared(t, sharedPg) }

// SharedMySQLContainer is SharedPgContainer for MySQL; tests take a database each.
func SharedMySQLContainer(t testing.TB) *Container { return shared(t, sharedMySQL) }

// SharedOracleContainer is SharedPgContainer for Oracle; tests take a user each.
func SharedOracleContainer(t testing.TB) *Container { return shared(t, sharedOracle) }

// SharedMSSQLContainer is SharedPgContainer for SQL Server; tests take a database each.
func SharedMSSQLContainer(t testing.TB) *Container { return shared(t, sharedMSSQL) }

// SharedTiDBContainer is SharedPgContainer for TiDB; tests take a database each.
func SharedTiDBContainer(t testing.TB) *Container { return shared(t, sharedTiDB) }

// NewPgDatabase creates an empty database for the test on the shared
// PostgreSQL container and returns its name with a superuser handle to it.
// Anything cluster-wide the test creates, such as a role, should be named
// after the database so that parallel tests cannot collide.
func NewPgDatabase(t testing.TB) (string, *sql.DB) {
	t.Helper()
	container := SharedPgContainer(t)
	name := fmt.Sprintf("test_%d", pgDatabaseSeq.Add(1))
	_, err := container.GetDB().Exec("CREATE DATABASE " + name)
	require.NoError(t, err)
	db, err := sql.Open("pgx", fmt.Sprintf("host=%s port=%s user=postgres password=root-password database=%s", container.GetHost(), container.GetPort(), name))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	return name, db
}
