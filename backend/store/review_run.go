package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store/qb"
)

// Reviewer type values stored in review_run.type. The column has no CHECK on
// purpose: the reviewer-id space is open.
const (
	// ReviewRunTypeRule reviews against the standard rules.
	ReviewRunTypeRule = "RULE"
	// ReviewRunTypeGuideline reviews against natural-language guidelines,
	// performed by AI.
	ReviewRunTypeGuideline = "GUIDELINE"
)

// ReviewRunMessage is one review_run slot row: the current run of one
// reviewer on one issue, reset in place on re-run.
type ReviewRunMessage struct {
	ProjectID string
	IssueUID  int64
	Type      string
	Attempt   int64
	Status    storepb.ReviewRun_Status
	Payload   *storepb.ReviewRunPayload
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CreateReviewRun creates a new AVAILABLE run in the (issue, reviewer type)
// slot. The slot row is reset in place unconditionally: created at attempt 0
// when absent, attempt bumped otherwise, with created_at restarted and
// replica_id and payload cleared. A RUNNING execution is superseded — its
// completion transaction is fenced on the old attempt and will match zero
// rows.
//
// The slot requires an active project; the INSERT reads the issue row itself,
// so a missing issue returns NotFound instead of a foreign-key error.
func (s *Store) CreateReviewRun(ctx context.Context, projectID string, issueUID int64, reviewType string) (*ReviewRunMessage, error) {
	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin tx")
	}
	defer tx.Rollback()

	if err := requireActiveProject(ctx, tx, projectID); err != nil {
		return nil, err
	}

	q := qb.Q().Space(`
		INSERT INTO review_run (project, issue_id, type, status)
		SELECT issue.project, issue.id, ?, ?
		FROM issue
		WHERE issue.project = ? AND issue.id = ?
		ON CONFLICT (project, issue_id, type) DO UPDATE
		SET status = EXCLUDED.status,
			attempt = review_run.attempt + 1,
			created_at = now(),
			updated_at = now(),
			replica_id = NULL,
			payload = '{}'
		RETURNING attempt, created_at, updated_at
	`, reviewType, storepb.ReviewRun_AVAILABLE.String(), projectID, issueUID)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	reviewRun := &ReviewRunMessage{
		ProjectID: projectID,
		IssueUID:  issueUID,
		Type:      reviewType,
		Status:    storepb.ReviewRun_AVAILABLE,
		Payload:   &storepb.ReviewRunPayload{},
	}
	if err := tx.QueryRowContext(ctx, query, args...).Scan(&reviewRun.Attempt, &reviewRun.CreatedAt, &reviewRun.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, common.Errorf(common.NotFound, "issue %d not found in project %s", issueUID, projectID)
		}
		return nil, errors.Wrapf(err, "failed to create review run")
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit tx")
	}
	return reviewRun, nil
}

// ClaimedReviewRun identifies a claimed review run slot and carries the claim
// fence (attempt).
type ClaimedReviewRun struct {
	ProjectID string
	IssueUID  int64
	Type      string
	Attempt   int64
}

// ClaimAvailableReviewRuns atomically claims every AVAILABLE review run for
// this replica by moving it to RUNNING (the ClaimAvailableTaskRuns shape).
// FOR UPDATE SKIP LOCKED lets concurrent replicas claim disjoint rows.
func (s *Store) ClaimAvailableReviewRuns(ctx context.Context, replicaID string) ([]*ClaimedReviewRun, error) {
	q := qb.Q().Space(`
		UPDATE review_run
		SET status = ?, updated_at = now(), replica_id = ?
		WHERE (project, issue_id, type) IN (
			SELECT review_run.project, review_run.issue_id, review_run.type
			FROM review_run
			JOIN project ON project.resource_id = review_run.project
			WHERE review_run.status = ?
			  AND project.deleted = FALSE
			FOR UPDATE OF review_run SKIP LOCKED
		)
		RETURNING project, issue_id, type, attempt
	`, storepb.ReviewRun_RUNNING.String(), replicaID, storepb.ReviewRun_AVAILABLE.String())

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	rows, err := s.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to claim review runs")
	}
	defer rows.Close()

	var claimed []*ClaimedReviewRun
	for rows.Next() {
		var c ClaimedReviewRun
		if err := rows.Scan(&c.ProjectID, &c.IssueUID, &c.Type, &c.Attempt); err != nil {
			return nil, errors.Wrapf(err, "failed to scan claimed review run")
		}
		claimed = append(claimed, &c)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrapf(err, "failed to iterate claimed review runs")
	}
	return claimed, nil
}

// CompleteReviewRun moves a RUNNING run this replica owns to a terminal
// status, fenced on the claim's attempt. On DONE it posts the run's results in
// the same transaction: every OPEN root the same reviewer posted on the issue
// is resolved, then the results are inserted as OPEN roots with no creator. A
// FAILED run leaves the comments alone and takes no results. False means
// superseded: the slot was reset or reaped since the claim, and nothing was
// written.
//
// Lock order: the issue's review comments, then the run slot, the order the
// project purge deletes them in.
func (s *Store) CompleteReviewRun(ctx context.Context, claimed *ClaimedReviewRun, replicaID string, status storepb.ReviewRun_Status, payload *storepb.ReviewRunPayload, results []*IssueCommentMessage) (bool, error) {
	switch status {
	case storepb.ReviewRun_DONE:
	case storepb.ReviewRun_FAILED:
		if len(results) > 0 {
			return false, errors.New("a failed review run posts no results")
		}
	default:
		return false, errors.Errorf("invalid terminal review run status %v", status)
	}
	if payload == nil {
		payload = &storepb.ReviewRunPayload{}
	}
	payloadBytes, err := protojson.Marshal(payload)
	if err != nil {
		return false, errors.Wrapf(err, "failed to marshal review run payload")
	}
	resultPayloads, err := marshalReviewResults(claimed, results)
	if err != nil {
		return false, err
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return false, errors.Wrapf(err, "failed to begin tx")
	}
	defer tx.Rollback()

	if status == storepb.ReviewRun_DONE {
		// The literal 'OPEN' lets the planner use idx_issue_comment_open_thread.
		q := qb.Q().Space(`
			UPDATE issue_comment
			SET thread_state = ?
			WHERE project = ? AND issue_id = ?
			  AND parent_id IS NULL AND thread_state = 'OPEN'
			  AND payload->'reviewMetadata'->>'runType' = ?
		`, string(ThreadStateResolved), claimed.ProjectID, claimed.IssueUID, claimed.Type)
		query, args, err := q.ToSQL()
		if err != nil {
			return false, errors.Wrapf(err, "failed to build sql")
		}
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return false, errors.Wrapf(err, "failed to resolve previous review results")
		}
		if len(resultPayloads) > 0 {
			// created_at is offset by the ordinal for the same reason as in
			// CreateIssueComments: the batch keeps its order.
			q := qb.Q().Space(`
				INSERT INTO issue_comment (creator, project, issue_id, payload, thread_state, created_at, updated_at)
				SELECT NULL, ?, ?, c.payload, 'OPEN', t.at, t.at
				FROM unnest(?::JSONB[]) WITH ORDINALITY AS c(payload, ordinality),
				     LATERAL (SELECT now() + ((c.ordinality - 1) * interval '1 microsecond')) AS t(at)
			`, claimed.ProjectID, claimed.IssueUID, resultPayloads)
			query, args, err := q.ToSQL()
			if err != nil {
				return false, errors.Wrapf(err, "failed to build sql")
			}
			if _, err := tx.ExecContext(ctx, query, args...); err != nil {
				return false, errors.Wrapf(err, "failed to post review results")
			}
		}
	}

	q := qb.Q().Space(`
		UPDATE review_run
		SET status = ?, updated_at = now(), payload = ?
		WHERE project = ? AND issue_id = ? AND type = ?
		  AND status = ? AND replica_id = ? AND attempt = ?
	`, status.String(), payloadBytes,
		claimed.ProjectID, claimed.IssueUID, claimed.Type,
		storepb.ReviewRun_RUNNING.String(), replicaID, claimed.Attempt)

	query, args, err := q.ToSQL()
	if err != nil {
		return false, errors.Wrapf(err, "failed to build sql")
	}

	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return false, errors.Wrapf(err, "failed to complete review run")
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, errors.Wrapf(err, "failed to get rows affected")
	}
	if rowsAffected == 0 {
		// Superseded: the rollback discards the results.
		return false, nil
	}
	if err := tx.Commit(); err != nil {
		return false, errors.Wrapf(err, "failed to commit tx")
	}
	return true, nil
}

// marshalReviewResults checks that every result is a root comment the
// claimed reviewer posted, with no event and no creator, and returns the
// payloads to insert.
func marshalReviewResults(claimed *ClaimedReviewRun, results []*IssueCommentMessage) ([][]byte, error) {
	payloads := make([][]byte, 0, len(results))
	for _, result := range results {
		if result.ProjectID != claimed.ProjectID || result.IssueUID != claimed.IssueUID {
			return nil, errors.Errorf("review result for issue %d in project %s does not belong to the run on issue %d in project %s", result.IssueUID, result.ProjectID, claimed.IssueUID, claimed.ProjectID)
		}
		if result.ParentID != nil || result.CreatorEmail != "" {
			return nil, errors.New("a review result is a root comment with no creator")
		}
		if result.ThreadState != nil && *result.ThreadState != ThreadStateOpen {
			return nil, errors.New("a review result starts its thread OPEN")
		}
		if result.Payload.GetEvent() != nil {
			return nil, errors.New("a review result cannot carry an event")
		}
		if result.Payload.GetReviewMetadata().GetRunType().String() != claimed.Type {
			return nil, errors.Errorf("review result names reviewer %q; the run is %q", result.Payload.GetReviewMetadata().GetRunType(), claimed.Type)
		}
		if err := validateIssueCommentPayload(result.Payload); err != nil {
			return nil, err
		}
		payload, err := protojson.Marshal(result.Payload)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to marshal review result")
		}
		payloads = append(payloads, payload)
	}
	return payloads, nil
}

// FailStaleReviewRuns fails RUNNING review runs whose owning replica has no
// recent heartbeat — the replica crashed, or its heartbeat row was cleaned
// up. NOT EXISTS covers both a missing and an expired heartbeat row (the
// FailStaleTaskRuns shape).
func (s *Store) FailStaleReviewRuns(ctx context.Context, stalenessThreshold time.Duration) (int64, error) {
	payloadBytes, err := protojson.Marshal(&storepb.ReviewRunPayload{
		Error: "Review run abandoned: owning replica stopped responding",
	})
	if err != nil {
		return 0, errors.Wrapf(err, "failed to marshal review run payload")
	}

	q := qb.Q().Space(`
		UPDATE review_run
		SET status = ?, updated_at = now(), payload = ?
		WHERE status = ?
		  AND replica_id IS NOT NULL
		  AND NOT EXISTS (
			SELECT 1 FROM replica_heartbeat rh
			WHERE rh.replica_id = review_run.replica_id
			  AND rh.last_heartbeat >= now() - ?::INTERVAL
		  )
	`, storepb.ReviewRun_FAILED.String(), payloadBytes,
		storepb.ReviewRun_RUNNING.String(), stalenessThreshold.String())

	query, args, err := q.ToSQL()
	if err != nil {
		return 0, errors.Wrapf(err, "failed to build sql")
	}

	result, err := s.GetDB().ExecContext(ctx, query, args...)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to fail stale review runs")
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrapf(err, "failed to get rows affected")
	}
	return rowsAffected, nil
}
