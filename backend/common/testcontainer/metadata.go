// Shared metadata Postgres: the package's shared container carries a template
// database migrated once, and each test calls NewMetadataDB for a copy of it.
// A copy costs 44ms against a container start. The package opts in with
//
//	func TestMain(m *testing.M) { testcontainer.Main(m) }

package testcontainer

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/store"
)

const metadataTemplateDatabase = "bbtest_template"

var (
	metaHost, metaPort string
	metaAdminDB        *sql.DB
	metaDatabaseSeq    atomic.Int64
	metaOnce           sync.Once
	metaErr            error
)

// startMetadata migrates the template on the shared container on first use.
func startMetadata(t *testing.T) {
	t.Helper()
	container := SharedPgContainer(t)
	metaOnce.Do(func() {
		metaHost, metaPort = container.GetHost(), container.GetPort()
		metaAdminDB = container.GetDB()
		metaErr = migrateTemplate(context.Background())
	})
	require.NoError(t, metaErr)
}

func migrateTemplate(ctx context.Context) error {
	if _, err := metaAdminDB.ExecContext(ctx, "CREATE DATABASE "+metadataTemplateDatabase); err != nil {
		return err
	}
	db, err := sql.Open("pgx", metadataURL(metadataTemplateDatabase))
	if err != nil {
		return err
	}
	if err := migrator.MigrateSchema(ctx, db); err != nil {
		_ = db.Close()
		return err
	}
	// TEMPLATE refuses to copy a database that still has a session attached.
	return db.Close()
}

// NewMetadataDB gives the test its own migrated database: a raw handle for seeding, a
// Store for the code under test, and the URL for tests that open their own
// connections. The copies are not dropped; the container takes them with it.
func NewMetadataDB(t *testing.T) (*sql.DB, *store.Store, string) {
	t.Helper()
	return NewMetadataDBWithCache(t, false)
}

// NewMetadataDBWithCache is NewMetadataDB with the Store's cache enabled, for the few tests whose
// subject is the cache itself.
func NewMetadataDBWithCache(t *testing.T, enableCache bool) (*sql.DB, *store.Store, string) {
	t.Helper()
	startMetadata(t)
	ctx := context.Background()
	name := fmt.Sprintf("bbtest_%d", metaDatabaseSeq.Add(1))
	_, err := metaAdminDB.ExecContext(ctx,
		fmt.Sprintf("CREATE DATABASE %s TEMPLATE %s", name, metadataTemplateDatabase))
	require.NoError(t, err)

	dsn := metadataURL(name)
	db, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	s, err := store.New(ctx, dsn, enableCache)
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, s.Close())
		require.NoError(t, db.Close())
	})
	return db, s, dsn
}

// metadataURL is the URI form, which pgx accepts and which tests that hand a metadata
// URL to production code (the sample manager, for one) need verbatim.
func metadataURL(database string) string {
	return fmt.Sprintf("postgresql://postgres:root-password@%s:%s/%s?sslmode=disable",
		metaHost, metaPort, database)
}
