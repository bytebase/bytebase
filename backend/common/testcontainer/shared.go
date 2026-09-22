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

// One container per engine per test package, started on first use and stopped
// by CloseShared. A test binary is its own process, so package-level state here
// is per package. Starting lazily is what keeps `go test -run` over the tests
// that need no engine from paying for a container they never open.
var (
	sharedPg       = startOnce(GetPgContainer)
	sharedTargetPg = startOnce(GetPgContainer)
	sharedPg17     = startOnce(getPg17Container)
	sharedTLSPg    = startOnce(getTLSPgContainer)
	sharedMSSQL    = startOnce(getMSSQLContainer)
	sharedTiDB     = startOnce(getTiDBContainer)
	sharedMongo    = startOnce(getMongoDBContainer)

	startedMu sync.Mutex
	started   []*Container

	databaseSeq atomic.Int64
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
	defer CloseShared()
	m.Run()
}

// CloseShared stops every container this package started. Main calls it; a
// package whose TestMain does more than run the suite defers it there instead.
func CloseShared() {
	startedMu.Lock()
	defer startedMu.Unlock()
	for _, c := range started {
		c.Close(context.Background())
	}
	started = nil
}

// StartSharedPg is SharedPgContainer for a TestMain, which has no testing.TB to
// fail and so takes the error itself.
func StartSharedPg() (*Container, error) { return sharedPg() }

// SharedPgContainer returns the package's PostgreSQL container, starting it on
// the first call. Tests isolate themselves with NewPgDatabase or NewMetadataDB.
func SharedPgContainer(t testing.TB) *Container { return shared(t, sharedPg) }

// SharedTargetPgContainer is a second PostgreSQL, for tests that point product
// code at a server rather than store metadata in one. Keeping the two apart
// keeps a package's target databases out of the metadata container's connection
// budget, which is what caps parallelism.
func SharedTargetPgContainer(t testing.TB) *Container { return shared(t, sharedTargetPg) }

// SharedPg17Container is SharedPgContainer for the PostgreSQL 17 features absent
// in 16, MERGE ... RETURNING being the one; tests take a database each with
// NewPg17Database.
func SharedPg17Container(t testing.TB) *Container { return shared(t, sharedPg17) }

// SharedTLSPgContainer is SharedPgContainer for the TLS-enabled PostgreSQL that
// tests reach with sslmode=verify-full and GetTLSCAPath.
func SharedTLSPgContainer(t testing.TB) *Container { return shared(t, sharedTLSPg) }

// SharedMSSQLContainer is SharedPgContainer for SQL Server; tests take a database each.
func SharedMSSQLContainer(t testing.TB) *Container { return shared(t, sharedMSSQL) }

// SharedTiDBContainer is SharedPgContainer for TiDB; tests take a database each
// with NewTiDBDatabase.
func SharedTiDBContainer(t testing.TB) *Container { return shared(t, sharedTiDB) }

// SharedMongoDBContainer is SharedPgContainer for MongoDB; tests take a database
// each with NewMongoDatabase.
func SharedMongoDBContainer(t testing.TB) *Container { return shared(t, sharedMongo) }

// NewPgDatabase creates an empty database for the test on the shared
// PostgreSQL container and returns its name with a superuser handle to it.
// Anything cluster-wide the test creates, such as a role, should be named
// after the database so that parallel tests cannot collide.
func NewPgDatabase(t testing.TB) (string, *sql.DB) {
	t.Helper()
	return createPgDatabase(t, SharedPgContainer(t))
}

// NewPg17Database is NewPgDatabase on the shared PostgreSQL 17 container.
func NewPg17Database(t testing.TB) (string, *sql.DB) {
	t.Helper()
	return createPgDatabase(t, SharedPg17Container(t))
}

func createPgDatabase(t testing.TB, container *Container) (string, *sql.DB) {
	t.Helper()
	name := newDatabaseName()
	_, err := container.GetDB().Exec("CREATE DATABASE " + name)
	require.NoError(t, err)
	db, err := sql.Open("pgx", fmt.Sprintf("host=%s port=%s user=postgres password=root-password database=%s", container.GetHost(), container.GetPort(), name))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	return name, db
}

// NewTiDBDatabase creates an empty database for the test on the shared TiDB
// container and returns the container with the database's name.
func NewTiDBDatabase(t testing.TB) (*Container, string) {
	t.Helper()
	container := SharedTiDBContainer(t)
	name := newDatabaseName()
	_, err := container.GetDB().Exec("CREATE DATABASE " + name)
	require.NoError(t, err)
	return container, name
}

// NewMongoDatabase reserves a database for the test on the shared MongoDB
// container and returns the container with the database's name. MongoDB creates
// the database on the first write, so the name is all the test needs.
func NewMongoDatabase(t testing.TB) (*Container, string) {
	t.Helper()
	return SharedMongoDBContainer(t), newDatabaseName()
}

// newDatabaseName is unique across the test binary, so tests that run in
// parallel, on the same container or not, never share a database.
func newDatabaseName() string {
	return fmt.Sprintf("test_%d", databaseSeq.Add(1))
}
