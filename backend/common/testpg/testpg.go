// Package testpg gives a test package one Postgres instead of one per test: a
// container started once, a template database migrated once, and a cheap copy
// of that template per test. A copy costs 44ms against a 4.3s container.
//
// A package opts in with three lines:
//
//	func TestMain(m *testing.M) { testpg.Main(m) }
//
// and each test then calls New for its own database.
package testpg

import (
	"context"
	"database/sql"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/store"
)

const templateDatabase = "bbtest_template"

var (
	pgHost, pgPort string
	adminDB        *sql.DB
	databaseSeq    atomic.Int64
)

// Main starts one Postgres for the package, migrates a template database, runs
// the tests, and takes the container down. Call it from the package's TestMain.
func Main(m *testing.M) {
	ctx := context.Background()
	container, err := testcontainer.GetPgContainer(ctx)
	if err != nil {
		panic(err)
	}
	defer container.Close(ctx)

	pgHost, pgPort = container.GetHost(), container.GetPort()
	adminDB = container.GetDB()
	if err := migrateTemplate(ctx); err != nil {
		panic(err)
	}
	m.Run()
}

func migrateTemplate(ctx context.Context) error {
	if _, err := adminDB.ExecContext(ctx, "CREATE DATABASE "+templateDatabase); err != nil {
		return err
	}
	db, err := sql.Open("pgx", url(templateDatabase))
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

// New gives the test its own migrated database: a raw handle for seeding, a
// Store for the code under test, and the URL for tests that open their own
// connections. The copies are not dropped; the container takes them with it.
func New(t *testing.T) (*sql.DB, *store.Store, string) {
	t.Helper()
	return NewWithCache(t, false)
}

// NewWithCache is New with the Store's cache enabled, for the few tests whose
// subject is the cache itself.
func NewWithCache(t *testing.T, enableCache bool) (*sql.DB, *store.Store, string) {
	t.Helper()
	ctx := context.Background()
	name := fmt.Sprintf("bbtest_%d", databaseSeq.Add(1))
	_, err := adminDB.ExecContext(ctx,
		fmt.Sprintf("CREATE DATABASE %s TEMPLATE %s", name, templateDatabase))
	require.NoError(t, err)

	dsn := url(name)
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

// url is the URI form, which pgx accepts and which tests that hand a metadata
// URL to production code (the sample manager, for one) need verbatim.
func url(database string) string {
	return fmt.Sprintf("postgresql://postgres:root-password@%s:%s/%s?sslmode=disable",
		pgHost, pgPort, database)
}
