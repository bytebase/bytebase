package store_test

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

// TestMain starts one Postgres for the package and migrates a template
// database once. Tests copy the template instead of starting a container:
// a copy costs 44ms against 4.3s.
func TestMain(m *testing.M) {
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
	db, err := sql.Open("pgx", fmt.Sprintf(
		"host=%s port=%s user=postgres password=root-password database=%s",
		pgHost, pgPort, templateDatabase))
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

// newTestDB gives the test its own migrated database: a raw handle for seeding,
// a Store for the code under test, and the URL for tests that open their own
// connections. The copies are not dropped; the container takes them with it.
func newTestDB(t *testing.T) (*sql.DB, *store.Store, string) {
	t.Helper()
	ctx := context.Background()
	name := fmt.Sprintf("bbtest_%d", databaseSeq.Add(1))
	_, err := adminDB.ExecContext(ctx,
		fmt.Sprintf("CREATE DATABASE %s TEMPLATE %s", name, templateDatabase))
	require.NoError(t, err)

	url := fmt.Sprintf("host=%s port=%s user=postgres password=root-password database=%s",
		pgHost, pgPort, name)
	db, err := sql.Open("pgx", url)
	require.NoError(t, err)
	s, err := store.New(ctx, url, false)
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, s.Close())
		require.NoError(t, db.Close())
	})
	return db, s, url
}
