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
// A test binary is its own process, so the package-level state here is per
// package. Starting lazily is what keeps `go test -run` over the tests that
// need no engine from paying for a container they never open.
type sharedContainer struct {
	once      sync.Once
	container *Container
	err       error
}

func (s *sharedContainer) get(t testing.TB, start func(context.Context) (*Container, error)) *Container {
	t.Helper()
	s.once.Do(func() { s.container, s.err = start(context.Background()) })
	require.NoError(t, s.err, "start the shared container")
	return s.container
}

func (s *sharedContainer) close(ctx context.Context) {
	if s.container != nil {
		s.container.Close(ctx)
	}
}

var (
	sharedPg, sharedMySQL, sharedOracle, sharedMSSQL, sharedTiDB sharedContainer
	pgDatabaseSeq                                                atomic.Int64
)

// Main runs the package's tests and stops the containers they shared. Call it
// from the package's TestMain:
//
//	func TestMain(m *testing.M) { testcontainer.Main(m) }
func Main(m *testing.M) {
	defer func() {
		ctx := context.Background()
		for _, s := range []*sharedContainer{&sharedPg, &sharedMySQL, &sharedOracle, &sharedMSSQL, &sharedTiDB} {
			s.close(ctx)
		}
	}()
	m.Run()
}

// SharedPgContainer returns the package's PostgreSQL container, starting it on
// the first call. Tests isolate themselves with NewPgDatabase, or with
// NewMetadataDB for a migrated metadata database.
func SharedPgContainer(t testing.TB) *Container { return sharedPg.get(t, GetPgContainer) }

// SharedMySQLContainer returns the package's MySQL container, starting it on
// the first call. Tests isolate themselves with a database each.
func SharedMySQLContainer(t testing.TB) *Container { return sharedMySQL.get(t, GetTestMySQLContainer) }

// SharedOracleContainer returns the package's Oracle container, starting it on
// the first call. Tests isolate themselves with a user each.
func SharedOracleContainer(t testing.TB) *Container { return sharedOracle.get(t, GetOracleContainer) }

// SharedMSSQLContainer returns the package's SQL Server container, starting it
// on the first call. Tests isolate themselves with a database each.
func SharedMSSQLContainer(t testing.TB) *Container { return sharedMSSQL.get(t, GetMSSQLContainer) }

// SharedTiDBContainer returns the package's TiDB container, starting it on the
// first call. Tests isolate themselves with a database each.
func SharedTiDBContainer(t testing.TB) *Container { return sharedTiDB.get(t, GetTiDBContainer) }

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
