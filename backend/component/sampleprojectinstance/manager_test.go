package sampleprojectinstance

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/store"
)

func TestFailureKindOf(t *testing.T) {
	require.Equal(t, FailureUnknown, FailureKindOf(errors.New("unexpected")))
	require.Equal(t, FailureFailedPrecondition, FailureKindOf(newFailure(FailureFailedPrecondition, errors.New("invalid target"))))
	require.Equal(t, FailureUnavailable, FailureKindOf(errors.Join(errors.New("wrapped"), newFailure(FailureUnavailable, errors.New("offline")))))
	require.Equal(t, FailureDeadlineExceeded, FailureKindOf(newFailure(FailureDeadlineExceeded, context.DeadlineExceeded)))
}

func TestMapTargetErrorUsesManagerFailureVocabulary(t *testing.T) {
	require.Equal(t, FailureFailedPrecondition, FailureKindOf(mapTargetError(newTargetFailure(targetFailureStatic, errors.New("invalid target")))))
	require.Equal(t, FailureUnavailable, FailureKindOf(mapTargetError(newTargetFailure(targetFailureUnavailable, errors.New("offline")))))
	require.Equal(t, FailureDeadlineExceeded, FailureKindOf(mapTargetError(context.DeadlineExceeded)))
	require.Equal(t, FailureUnknown, FailureKindOf(mapTargetError(newTargetFailure(targetFailureInvariant, errors.New("collision")))))
}

func newManagerStore(t *testing.T) (context.Context, *sql.DB, *store.Store) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(context.Background()) })
	db := container.GetDB()
	require.NoError(t, migrator.MigrateSchema(ctx, db))
	_, err := db.ExecContext(ctx, `INSERT INTO workspace (resource_id) VALUES ('workspace-a'), ('workspace-b')`)
	require.NoError(t, err)
	pgURL := fmt.Sprintf("host=%s port=%s user=postgres password=root-password database=postgres", container.GetHost(), container.GetPort())
	s, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, s.Close()) })
	return ctx, db, s
}
