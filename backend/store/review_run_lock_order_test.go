package store_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

// reviewRunLockOrderSeed seeds a RUNNING review run claimed by replica-1.
const reviewRunLockOrderSeed = `
	INSERT INTO issue (id, creator, project, name, status, type)
	VALUES (301, 'admin@example.com', '%[1]s', 'Issue', 'OPEN', 'DATABASE_CHANGE');
	INSERT INTO review_run (project, issue_id, attempt, status, replica_id)
	VALUES ('%[1]s', 301, 0, 'RUNNING', 'replica-1');
`

// reviewRunPriorResultSeed adds a prior OPEN result, which gives the
// completion's resolve UPDATE a row to lock. It is the AI review's, and the
// completion posts a rule's.
const reviewRunPriorResultSeed = `
	INSERT INTO issue_comment (project, issue_id, payload, thread_state)
	VALUES ('%[1]s', 301, '{"comment":"prior","reviewMetadata":{"runType":"AI","priority":"P0","title":"Prior"}}', 'OPEN');
`

func reviewRunLockOrderCompletion(fixture *storePostgresFixture, projectID string) (bool, error) {
	claimed := &store.ClaimedReviewRun{ProjectID: projectID, IssueUID: reviewResultIssue, Attempt: 0}
	return fixture.store.CompleteReviewRun(fixture.ctx, claimed, "replica-1", storepb.ReviewRun_DONE, nil,
		[]*store.IssueCommentMessage{reviewResult(projectID, "new", storepb.ReviewRuleType_REQUIRE_WHERE)})
}

// TestCompleteReviewRunLockOrder parks a DONE completion mid-transaction and
// races the two writers that share its rows: a re-run, which must win and
// leave the completion with nothing written, and a project purge, which must
// wait for the completion rather than deadlock with it, whether the resolve
// UPDATE locked a prior result or the first lock is the run slot's.
func TestCompleteReviewRunLockOrder(t *testing.T) {
	t.Parallel()

	t.Run("re-run lands mid-completion", func(t *testing.T) {
		t.Parallel()
		fixture := newStorePostgresFixture(t, fmt.Sprintf(reviewRunLockOrderSeed+reviewRunPriorResultSeed, "default"))
		const barrierID = 9971
		barrier := newMaintenanceLockBarrier(fixture.ctx, t, fixture.db, barrierID)
		installMaintenanceLockBarrier(t, fixture.db, barrierID, "AFTER UPDATE ON issue_comment FOR EACH STATEMENT")

		type completion struct {
			updated bool
			err     error
		}
		completed := make(chan completion, 1)
		go func() {
			updated, err := reviewRunLockOrderCompletion(fixture, "default")
			completed <- completion{updated: updated, err: err}
		}()
		waitForMaintenanceBarrier(fixture.ctx, t, fixture.db, barrierID)

		// The re-run touches only the slot, so it lands while the completion
		// holds the comment locks.
		run, err := fixture.store.CreateReviewRun(fixture.ctx, "default", reviewResultIssue)
		require.NoError(t, err)
		require.Equal(t, int64(1), run.Attempt)
		barrier.release(t)

		select {
		case c := <-completed:
			require.NoError(t, c.err)
			require.False(t, c.updated, "the superseded completion must match zero rows")
		case <-time.After(maintenanceLockWait):
			t.Fatal("timed out waiting for the completion")
		}
		require.Equal(t, []reviewComment{{Comment: "prior", ThreadState: "OPEN"}},
			listReviewComments(t, fixture, "default"), "a superseded completion writes nothing")
		var status string
		var attempt int64
		require.NoError(t, fixture.db.QueryRowContext(fixture.ctx,
			`SELECT status, attempt FROM review_run WHERE project = 'default' AND issue_id = $1`,
			reviewResultIssue).Scan(&status, &attempt))
		require.Equal(t, "AVAILABLE", status)
		require.Equal(t, int64(1), attempt)
	})

	// The purge deletes issue_comment, then review_run, then issue. With a
	// prior result it waits on the comment row the completion resolved; with
	// none, on the run slot the completion fenced. Either way the completion
	// never waits on the purge.
	for _, tc := range []struct {
		name      string
		seed      string
		barrierID int
		trigger   string
		prior     []reviewComment
	}{
		{
			name:      "purge waits on the resolved result",
			seed:      reviewRunLockOrderSeed + reviewRunPriorResultSeed,
			barrierID: 9972,
			trigger:   "AFTER UPDATE ON issue_comment FOR EACH STATEMENT",
			prior:     []reviewComment{{Comment: "prior", ThreadState: "RESOLVED"}},
		},
		{
			name:      "purge waits on the fenced run slot",
			seed:      reviewRunLockOrderSeed,
			barrierID: 9973,
			trigger:   "AFTER INSERT ON issue_comment FOR EACH STATEMENT",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fixture := newStorePostgresFixture(t, fmt.Sprintf(tc.seed, "project-a"))
			barrier := newMaintenanceLockBarrier(fixture.ctx, t, fixture.db, tc.barrierID)
			installMaintenanceLockBarrier(t, fixture.db, tc.barrierID, tc.trigger)

			type completion struct {
				updated bool
				err     error
			}
			completed := make(chan completion, 1)
			go func() {
				updated, err := reviewRunLockOrderCompletion(fixture, "project-a")
				completed <- completion{updated: updated, err: err}
			}()
			waitForMaintenanceBarrier(fixture.ctx, t, fixture.db, tc.barrierID)
			completionPID := maintenanceBarrierWaitingPID(fixture.ctx, t, fixture.db, tc.barrierID)

			purged := make(chan error, 1)
			go func() {
				purged <- fixture.store.DeleteProjects(fixture.ctx, "default", "project-a")
			}()
			waitForBackendBlockedByPID(fixture.ctx, t, fixture.db, completionPID)
			barrier.release(t)

			select {
			case c := <-completed:
				require.NoError(t, c.err)
				require.True(t, c.updated, "the completion commits while the purge waits")
			case <-time.After(maintenanceLockWait):
				t.Fatal("timed out waiting for the completion")
			}
			// The result the completion inserted is invisible to the purge's
			// comment DELETE, so its issue DELETE fails the foreign key: the
			// accepted best-effort outcome for a writer racing a purge, which
			// a retry then completes.
			select {
			case err := <-purged:
				require.ErrorContains(t, err, "issue_comment_project_issue_id_fkey")
			case <-time.After(maintenanceLockWait):
				t.Fatal("timed out waiting for the purge")
			}
			require.Equal(t, append(tc.prior, reviewComment{Comment: "new", ThreadState: "OPEN"}),
				listReviewComments(t, fixture, "project-a"), "the failed purge rolled back; the completion's writes stand")
			require.NoError(t, fixture.store.DeleteProjects(fixture.ctx, "default", "project-a"))
			require.Empty(t, listReviewComments(t, fixture, "project-a"))
		})
	}
}

// TestCompleteReviewRunOverlappingAttempts completes a superseded attempt and
// the current one at once, each keeping a different earlier result whose
// anchor moved. The superseded attempt is parked after its first comment
// update; the current one must wait for it rather than deadlock, then commit.
func TestCompleteReviewRunOverlappingAttempts(t *testing.T) {
	t.Parallel()
	fixture := newStorePostgresFixture(t, `
		INSERT INTO issue (id, creator, project, name, status, type)
		VALUES (301, 'admin@example.com', 'default', 'Issue', 'OPEN', 'DATABASE_CHANGE');
	`)
	ctx, s := fixture.ctx, fixture.store
	finding := func(comment, fingerprint string, line int32) *store.IssueCommentMessage {
		r := reviewResult("default", comment, storepb.ReviewRuleType_REQUIRE_WHERE)
		r.Payload.ReviewMetadata.Fingerprint = fingerprint
		r.Payload.StatementAnchor.StartPosition = &storepb.Position{Line: line}
		r.Payload.StatementAnchor.EndPosition = &storepb.Position{Line: line}
		return r
	}
	completeReviewRun(t, fixture, "default", storepb.ReviewRun_DONE,
		finding("x", "x", 1), finding("y", "y", 2))

	claim := func() *store.ClaimedReviewRun {
		t.Helper()
		_, err := s.CreateReviewRun(ctx, "default", reviewResultIssue)
		require.NoError(t, err)
		claimed, err := s.ClaimAvailableReviewRuns(ctx, "replica-1")
		require.NoError(t, err)
		require.Len(t, claimed, 1)
		return claimed[0]
	}
	// A re-run does not stop the execution it supersedes.
	superseded, current := claim(), claim()

	const barrierID = 9974
	barrier := newMaintenanceLockBarrier(ctx, t, fixture.db, barrierID)
	installMaintenanceLockBarrier(t, fixture.db, barrierID, "AFTER UPDATE ON issue_comment FOR EACH STATEMENT")

	type completion struct {
		updated bool
		err     error
	}
	complete := func(claimed *store.ClaimedReviewRun, kept *store.IssueCommentMessage) <-chan completion {
		done := make(chan completion, 1)
		go func() {
			updated, err := s.CompleteReviewRun(ctx, claimed, "replica-1", storepb.ReviewRun_DONE, nil, []*store.IssueCommentMessage{kept})
			done <- completion{updated: updated, err: err}
		}()
		return done
	}
	// Each keeps one result, moved down a line, and supersedes the other.
	supersededDone := complete(superseded, finding("x moved", "x", 3))
	waitForMaintenanceBarrier(ctx, t, fixture.db, barrierID)
	supersededPID := maintenanceBarrierWaitingPID(ctx, t, fixture.db, barrierID)
	currentDone := complete(current, finding("y moved", "y", 4))
	require.Eventually(t, func() bool {
		var waiting bool
		err := fixture.db.QueryRowContext(ctx, `SELECT EXISTS (
			SELECT 1 FROM pg_stat_activity
			WHERE datname = current_database() AND pid <> $1 AND cardinality(pg_blocking_pids(pid)) > 0
		)`, supersededPID).Scan(&waiting)
		return err == nil && waiting
	}, maintenanceLockWait, 10*time.Millisecond, "the current attempt should be waiting")
	barrier.release(t)

	for name, done := range map[string]<-chan completion{"superseded": supersededDone, "current": currentDone} {
		select {
		case c := <-done:
			require.NoError(t, c.err, name)
			require.Equal(t, name == "current", c.updated, name)
		case <-time.After(maintenanceLockWait):
			t.Fatalf("timed out waiting for the %s attempt", name)
		}
	}
	got, _ := listSupersession(t, fixture, "default")
	require.Equal(t, []supersession{
		{Comment: "x", ThreadState: "RESOLVED", Superseded: true},
		{Comment: "y moved", ThreadState: "OPEN"},
	}, got)
	require.Equal(t, reviewResultSheet+"/4", reviewResultAnchors(t, fixture, "default")["y moved"])
}
