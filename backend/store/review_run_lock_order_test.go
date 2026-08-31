package store_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/store"
)

func createReviewRunInProjectA(ctx context.Context, s *store.Store) error {
	_, err := s.CreateReviewRun(ctx, "project-a", 101, store.ReviewRunTypeRule)
	return err
}

const reviewRunLockOrderSeedSQL = `
	INSERT INTO issue (id, creator, project, name, status, type)
	VALUES (101, 'admin@example.com', 'project-a', 'Issue A', 'OPEN', 'DATABASE_CHANGE');
	INSERT INTO review_run (project, issue_id, type, status)
	VALUES ('project-a', 101, 'RULE', 'DONE');
`

// Purge first: project deletion wedges inside its review_run delete while
// CreateReviewRun queues behind the project purge fence. The create must end in
// a clean NotFound once the purge wins — no deadlock, no FK failure — and the
// purge itself must succeed against the seeded slot row.
func TestCreateReviewRunDoesNotDeadlockWithProjectDeletion(t *testing.T) {
	err := runWithConcurrentProjectDeletion(t, reviewRunLockOrderSeedSQL, "review_run", 9906, createReviewRunInProjectA)
	require.Equal(t, common.NotFound, common.ErrorCode(err))
	require.NotContains(t, strings.ToLower(err.Error()), "foreign key",
		"the create must reject the purged project cleanly, not with an FK failure")
}

// Writer first: CreateReviewRun holds the purge fence and waits on the project
// row; project deletion queues behind the fence. The create must reject the
// soft-deleted project with NotFound, and the purge must then complete.
func TestCreateReviewRunRejectsDeletedProjectDuringProjectDeletion(t *testing.T) {
	operationErr, deleteErr := runWithCreationBeforeProjectDeletion(t, reviewRunLockOrderSeedSQL, "review_run", createReviewRunInProjectA)
	require.NoError(t, deleteErr)
	require.Equal(t, common.NotFound, common.ErrorCode(operationErr))
	require.NotContains(t, strings.ToLower(operationErr.Error()), "foreign key",
		"the create must reject the deleted project cleanly, not with an FK failure")
}
