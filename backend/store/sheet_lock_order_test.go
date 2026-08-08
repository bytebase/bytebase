package store_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/store"
)

func createSheetInProjectA(ctx context.Context, s *store.Store) error {
	_, err := s.CreateSheets(ctx, "project-a", &store.SheetMessage{Statement: "SELECT 1;"})
	return err
}

const sheetLockOrderSeedSQL = `
	INSERT INTO sheet_blob (sha256, content) VALUES (sha256('SELECT seed;'::bytea), 'SELECT seed;');
	INSERT INTO sheet_blob_ref (project, sha256) VALUES ('project-a', sha256('SELECT seed;'::bytea));
`

// Purge first: project deletion wedges inside its sheet_blob_ref delete while
// CreateSheets queues behind the project purge fence. The create must end in
// a clean NotFound once the purge wins — no deadlock, no FK failure — and the
// purge itself must succeed against the seeded refs.
func TestCreateSheetsDoesNotDeadlockWithProjectDeletion(t *testing.T) {
	err := runWithConcurrentProjectDeletion(t, sheetLockOrderSeedSQL, "sheet_blob_ref", 9905, createSheetInProjectA)
	require.Equal(t, common.NotFound, common.ErrorCode(err))
	require.NotContains(t, strings.ToLower(err.Error()), "foreign key",
		"the create must reject the purged project cleanly, not with an FK failure")
}

// Writer first: CreateSheets holds the purge fence and waits on the project
// row; project deletion queues behind the fence. The create must reject the
// soft-deleted project with NotFound, and the purge must then complete.
func TestCreateSheetsRejectsDeletedProjectDuringProjectDeletion(t *testing.T) {
	operationErr, deleteErr := runWithCreationBeforeProjectDeletion(t, sheetLockOrderSeedSQL, "sheet_blob_ref", createSheetInProjectA)
	require.NoError(t, deleteErr)
	require.Equal(t, common.NotFound, common.ErrorCode(operationErr))
	require.NotContains(t, strings.ToLower(operationErr.Error()), "foreign key",
		"the create must reject the deleted project cleanly, not with an FK failure")
}
