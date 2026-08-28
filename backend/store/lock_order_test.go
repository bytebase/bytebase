package store_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/store"
)

type projectDeletionLockOrderFixture struct {
	ctx   context.Context
	db    *sql.DB
	store *store.Store
	// pgURL lets a test open a second store against the same database, e.g. one
	// with the caches enabled.
	pgURL string
}

func newProjectDeletionLockOrderFixture(t *testing.T, seedSQL string) *projectDeletionLockOrderFixture {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(context.Background()) })
	db := container.GetDB()
	require.NoError(t, migrator.MigrateSchema(ctx, db))

	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('default');
		INSERT INTO project (resource_id, workspace, name) VALUES ('default', 'default', 'Default');
		INSERT INTO project (resource_id, workspace, name, deleted) VALUES ('project-a', 'default', 'Project A', TRUE);
	`+seedSQL)
	require.NoError(t, err)

	pgURL := fmt.Sprintf(
		"host=%s port=%s user=postgres password=root-password database=postgres",
		container.GetHost(), container.GetPort(),
	)
	s, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, s.Close()) })

	return &projectDeletionLockOrderFixture{ctx: ctx, db: db, store: s, pgURL: pgURL}
}
