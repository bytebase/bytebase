package store_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

// TestReviewRunStoreLifecycle walks the slot through reset, claim, supersede,
// fenced completion, and heartbeat reaping against a real PostgreSQL.
func TestReviewRunStoreLifecycle(t *testing.T) {
	t.Parallel()
	fixture := newStorePostgresFixture(t, `
		INSERT INTO issue (id, creator, project, name, status, type)
		VALUES (201, 'admin@example.com', 'default', 'Issue Default', 'OPEN', 'DATABASE_CHANGE');
	`)
	ctx, db, s := fixture.ctx, fixture.db, fixture.store

	// The first create initializes the slot at attempt 0, AVAILABLE.
	run, err := s.CreateReviewRun(ctx, "default", 201, store.ReviewRunTypeRule)
	require.NoError(t, err)
	require.Equal(t, int64(0), run.Attempt)
	require.Equal(t, storepb.ReviewRun_AVAILABLE, run.Status)

	// A missing issue is a clean NotFound, not an FK failure.
	_, err = s.CreateReviewRun(ctx, "default", 999, store.ReviewRunTypeRule)
	require.Equal(t, common.NotFound, common.ErrorCode(err))

	// Claim moves the slot to RUNNING for this replica and returns the fence.
	claimed, err := s.ClaimAvailableReviewRuns(ctx, "replica-1")
	require.NoError(t, err)
	require.Len(t, claimed, 1)
	require.Equal(t, "default", claimed[0].ProjectID)
	require.Equal(t, int64(201), claimed[0].IssueUID)
	require.Equal(t, store.ReviewRunTypeRule, claimed[0].Type)
	require.Equal(t, int64(0), claimed[0].Attempt)

	// Nothing is left to claim.
	again, err := s.ClaimAvailableReviewRuns(ctx, "replica-1")
	require.NoError(t, err)
	require.Empty(t, again)

	// Creating a new run supersedes the RUNNING execution: the attempt bumps
	// and the slot returns to AVAILABLE.
	run, err = s.CreateReviewRun(ctx, "default", 201, store.ReviewRunTypeRule)
	require.NoError(t, err)
	require.Equal(t, int64(1), run.Attempt)
	require.Equal(t, storepb.ReviewRun_AVAILABLE, run.Status)

	// The superseded execution's completion is fenced off.
	updated, err := s.CompleteReviewRun(ctx, claimed[0], "replica-1", storepb.ReviewRun_DONE, nil, nil)
	require.NoError(t, err)
	require.False(t, updated, "a superseded completion must match zero rows")

	// Only terminal statuses complete a run.
	_, err = s.CompleteReviewRun(ctx, claimed[0], "replica-1", storepb.ReviewRun_AVAILABLE, nil, nil)
	require.Error(t, err)

	// Claim the new attempt and complete it for real.
	claimed, err = s.ClaimAvailableReviewRuns(ctx, "replica-1")
	require.NoError(t, err)
	require.Len(t, claimed, 1)
	require.Equal(t, int64(1), claimed[0].Attempt)
	updated, err = s.CompleteReviewRun(ctx, claimed[0], "replica-1", storepb.ReviewRun_FAILED, &storepb.ReviewRunPayload{Error: "boom"}, nil)
	require.NoError(t, err)
	require.True(t, updated)

	var status, payloadError string
	require.NoError(t, db.QueryRowContext(ctx, `
		SELECT status, payload->>'error' FROM review_run
		WHERE project = 'default' AND issue_id = 201 AND type = 'RULE'
	`).Scan(&status, &payloadError))
	require.Equal(t, "FAILED", status)
	require.Equal(t, "boom", payloadError)

	// A terminal row never completes again.
	updated, err = s.CompleteReviewRun(ctx, claimed[0], "replica-1", storepb.ReviewRun_DONE, nil, nil)
	require.NoError(t, err)
	require.False(t, updated)

	// The reaper fails a RUNNING run whose replica has no recent heartbeat
	// and writes the abandonment error; the reset clears it again.
	run, err = s.CreateReviewRun(ctx, "default", 201, store.ReviewRunTypeRule)
	require.NoError(t, err)
	require.Equal(t, int64(2), run.Attempt)
	claimed, err = s.ClaimAvailableReviewRuns(ctx, "dead-replica")
	require.NoError(t, err)
	require.Len(t, claimed, 1)
	rowsAffected, err := s.FailStaleReviewRuns(ctx, time.Minute)
	require.NoError(t, err)
	require.Equal(t, int64(1), rowsAffected)
	require.NoError(t, db.QueryRowContext(ctx, `
		SELECT status, payload->>'error' FROM review_run
		WHERE project = 'default' AND issue_id = 201 AND type = 'RULE'
	`).Scan(&status, &payloadError))
	require.Equal(t, "FAILED", status)
	require.Contains(t, payloadError, "abandoned")

	// A run whose replica beats survives the reaper.
	run, err = s.CreateReviewRun(ctx, "default", 201, store.ReviewRunTypeRule)
	require.NoError(t, err)
	require.Equal(t, int64(3), run.Attempt)
	claimed, err = s.ClaimAvailableReviewRuns(ctx, "live-replica")
	require.NoError(t, err)
	require.Len(t, claimed, 1)
	_, err = db.ExecContext(ctx, `
		INSERT INTO replica_heartbeat (replica_id, last_heartbeat) VALUES ('live-replica', now())
		ON CONFLICT (replica_id) DO UPDATE SET last_heartbeat = now()
	`)
	require.NoError(t, err)
	rowsAffected, err = s.FailStaleReviewRuns(ctx, time.Minute)
	require.NoError(t, err)
	require.Equal(t, int64(0), rowsAffected)
}

func TestGetReviewRunCollidingIssues(t *testing.T) {
	t.Parallel()
	fixture := newStorePostgresFixture(t, `
		INSERT INTO project (resource_id, workspace, name) VALUES ('other', 'default', 'Other');
		INSERT INTO issue (id, creator, project, name, status, type)
		VALUES (201, 'admin@example.com', 'default', 'Issue Default', 'OPEN', 'DATABASE_CHANGE');
		INSERT INTO issue (id, creator, project, name, status, type)
		VALUES (201, 'admin@example.com', 'other', 'Issue Other', 'OPEN', 'DATABASE_CHANGE');
	`)
	ctx, db, s := fixture.ctx, fixture.db, fixture.store

	got, err := s.GetReviewRun(ctx, "default", 201, store.ReviewRunTypeRule)
	require.NoError(t, err)
	require.Nil(t, got, "a slot that was never created")

	// default: FAILED at attempt 0. other: AVAILABLE at attempt 1.
	_, err = s.CreateReviewRun(ctx, "default", 201, store.ReviewRunTypeRule)
	require.NoError(t, err)
	claimed, err := s.ClaimAvailableReviewRuns(ctx, "replica-1")
	require.NoError(t, err)
	require.Len(t, claimed, 1)
	updated, err := s.CompleteReviewRun(ctx, claimed[0], "replica-1", storepb.ReviewRun_FAILED, &storepb.ReviewRunPayload{Error: "boom"}, nil)
	require.NoError(t, err)
	require.True(t, updated)
	for range 2 {
		_, err = s.CreateReviewRun(ctx, "other", 201, store.ReviewRunTypeRule)
		require.NoError(t, err)
	}

	type slotRow struct {
		Attempt   int64
		Status    string
		CreatedAt time.Time
		UpdatedAt time.Time
	}
	readRaw := func(projectID string) slotRow {
		var row slotRow
		require.NoError(t, db.QueryRowContext(ctx, `
			SELECT attempt, status, created_at, updated_at FROM review_run
			WHERE project = $1 AND issue_id = 201 AND type = 'RULE'
		`, projectID).Scan(&row.Attempt, &row.Status, &row.CreatedAt, &row.UpdatedAt))
		return row
	}
	rawDefault, rawOther := readRaw("default"), readRaw("other")
	require.NotEqual(t, rawDefault.Status, rawOther.Status, "the two slots must differ to tell them apart")

	got, err = s.GetReviewRun(ctx, "default", 201, store.ReviewRunTypeRule)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, "default", got.ProjectID)
	require.Equal(t, int64(201), got.IssueUID)
	require.Equal(t, store.ReviewRunTypeRule, got.Type)
	require.Equal(t, int64(0), got.Attempt)
	require.Equal(t, storepb.ReviewRun_FAILED, got.Status)
	require.Equal(t, "boom", got.Payload.GetError())
	require.True(t, got.CreatedAt.Equal(rawDefault.CreatedAt))
	require.True(t, got.UpdatedAt.Equal(rawDefault.UpdatedAt))

	got, err = s.GetReviewRun(ctx, "other", 201, store.ReviewRunTypeRule)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, "other", got.ProjectID)
	require.Equal(t, int64(1), got.Attempt)
	require.Equal(t, storepb.ReviewRun_AVAILABLE, got.Status)
	require.Empty(t, got.Payload.GetError())
	require.True(t, got.CreatedAt.Equal(rawOther.CreatedAt))
	require.True(t, got.UpdatedAt.Equal(rawOther.UpdatedAt))

	// The reviewer type is part of the key, and so is the issue.
	got, err = s.GetReviewRun(ctx, "default", 201, store.ReviewRunTypeAI)
	require.NoError(t, err)
	require.Nil(t, got)
	got, err = s.GetReviewRun(ctx, "default", 999, store.ReviewRunTypeRule)
	require.NoError(t, err)
	require.Nil(t, got)
}
