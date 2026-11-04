package store

import (
	"context"
	"database/sql"
	"slices"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store/qb"
)

// Reviewer type values stored in review_run.type. The column has no CHECK on
// purpose: the reviewer-id space is open.
const (
	// ReviewRunTypeRule reviews against the standard rules.
	ReviewRunTypeRule = "RULE"
	// ReviewRunTypeAI reviews against the natural-language AI review policy,
	// judged by a model.
	ReviewRunTypeAI = "AI"
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
// when absent, attempt bumped otherwise, with created_at restarted,
// replica_id cleared, and the payload reset to the approval input version of
// the issue's plan as of this write. A RUNNING execution is superseded — its
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
		INSERT INTO review_run (project, issue_id, type, status, payload)
		SELECT issue.project, issue.id, ?, ?,
			jsonb_strip_nulls(jsonb_build_object('approvalInputVersion', plan.config->'approvalInputVersion'))
		FROM issue
		LEFT JOIN plan ON plan.project = issue.project AND plan.id = issue.plan_id
		WHERE issue.project = ? AND issue.id = ?
		ON CONFLICT (project, issue_id, type) DO UPDATE
		SET status = EXCLUDED.status,
			attempt = review_run.attempt + 1,
			created_at = now(),
			updated_at = now(),
			replica_id = NULL,
			payload = EXCLUDED.payload
		RETURNING attempt, payload, created_at, updated_at
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
	var payload []byte
	if err := tx.QueryRowContext(ctx, query, args...).Scan(&reviewRun.Attempt, &payload, &reviewRun.CreatedAt, &reviewRun.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, common.Errorf(common.NotFound, "issue %d not found in project %s", issueUID, projectID)
		}
		return nil, errors.Wrapf(err, "failed to create review run")
	}
	if err := common.ProtojsonUnmarshaler.Unmarshal(payload, reviewRun.Payload); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal review run payload")
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit tx")
	}
	return reviewRun, nil
}

// keepApprovalInputVersion is the payload a terminal write stores: the new
// payload, with the slot's approval input version carried over. Losing it would
// make a finished run look stale to the rollout creator, which would start it
// again.
const keepApprovalInputVersion = `CAST(? AS JSONB) || jsonb_strip_nulls(jsonb_build_object('approvalInputVersion', review_run.payload->'approvalInputVersion'))`

// GetReviewRun returns the run in the (issue, reviewer type) slot as stored,
// or nil when the slot was never created.
func (s *Store) GetReviewRun(ctx context.Context, projectID string, issueUID int64, reviewType string) (*ReviewRunMessage, error) {
	q := qb.Q().Space(`
		SELECT attempt, status, payload, created_at, updated_at
		FROM review_run
		WHERE project = ? AND issue_id = ? AND type = ?
	`, projectID, issueUID, reviewType)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	reviewRun := &ReviewRunMessage{
		ProjectID: projectID,
		IssueUID:  issueUID,
		Type:      reviewType,
		Payload:   &storepb.ReviewRunPayload{},
	}
	var statusString string
	var payload []byte
	if err := s.GetDB().QueryRowContext(ctx, query, args...).Scan(
		&reviewRun.Attempt,
		&statusString,
		&payload,
		&reviewRun.CreatedAt,
		&reviewRun.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "failed to get review run")
	}
	statusValue, ok := storepb.ReviewRun_Status_value[statusString]
	if !ok {
		return nil, errors.Errorf("invalid review run status %q", statusString)
	}
	reviewRun.Status = storepb.ReviewRun_Status(statusValue)
	if err := common.ProtojsonUnmarshaler.Unmarshal(payload, reviewRun.Payload); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal review run payload")
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
// the same transaction. A current root of the same reviewer on the issue (one
// no run has superseded) whose fingerprint a result repeats keeps its thread,
// state and replies included, and takes the result's text and anchor; that
// result is not posted again. Every other current root of the reviewer is marked
// superseded and, if OPEN, resolved; the remaining results are inserted as
// OPEN roots with no creator, carrying the run's attempt; and a
// ReviewRunCompletion event records the counts. A FAILED run leaves the
// comments alone and takes no results. False means superseded: the slot was
// reset or reaped since the claim, and nothing was written.
//
// Lock order: the issue's review comments, then the run slot, then the issue
// (the insert's foreign key), the order the project purge deletes them in.
// The fence therefore runs before the inserts. A re-run does not stop the
// execution it supersedes, so two completions of one slot can overlap and
// split the reviewer's roots differently; each locks every root it may
// update, in primary-key order, before updating any.
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
	resultPayloads, fingerprints, err := marshalReviewResults(claimed, results)
	if err != nil {
		return false, err
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return false, errors.Wrapf(err, "failed to begin tx")
	}
	defer tx.Rollback()

	var supersededCount int64
	if status == storepb.ReviewRun_DONE {
		q := qb.Q().Space(`
			SELECT resource_id FROM issue_comment
			WHERE project = ? AND issue_id = ?
			  AND parent_id IS NULL
			  AND payload->'reviewMetadata'->>'runType' = ?
			  AND (payload->'reviewMetadata'->'superseded' IS NULL OR thread_state = 'OPEN')
			ORDER BY resource_id
			FOR UPDATE
		`, claimed.ProjectID, claimed.IssueUID, claimed.Type)
		query, args, err := q.ToSQL()
		if err != nil {
			return false, errors.Wrapf(err, "failed to build sql")
		}
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return false, errors.Wrapf(err, "failed to lock previous review results")
		}

		// updated_at marks content edits, so the updates leave it alone.
		q = qb.Q().Space(`
			UPDATE issue_comment
			SET thread_state = ?,
			    payload = jsonb_set(payload, '{reviewMetadata,superseded}', 'true'::jsonb)
			WHERE project = ? AND issue_id = ?
			  AND parent_id IS NULL
			  AND payload->'reviewMetadata'->>'runType' = ?
			  AND payload->'reviewMetadata'->'superseded' IS NULL
			  AND COALESCE(payload->'reviewMetadata'->>'fingerprint', '') <> ALL(?::TEXT[])
		`, string(ThreadStateResolved), claimed.ProjectID, claimed.IssueUID, claimed.Type, fingerprints)
		query, args, err = q.ToSQL()
		if err != nil {
			return false, errors.Wrapf(err, "failed to build sql")
		}
		result, err := tx.ExecContext(ctx, query, args...)
		if err != nil {
			return false, errors.Wrapf(err, "failed to supersede previous review results")
		}
		if supersededCount, err = result.RowsAffected(); err != nil {
			return false, errors.Wrapf(err, "failed to get rows affected")
		}

		// A superseded root someone reopened is resolved again.
		q = qb.Q().Space(`
			UPDATE issue_comment
			SET thread_state = ?
			WHERE project = ? AND issue_id = ?
			  AND parent_id IS NULL
			  AND payload->'reviewMetadata'->>'runType' = ?
			  AND payload->'reviewMetadata'->'superseded' IS NOT NULL
			  AND thread_state = 'OPEN'
		`, string(ThreadStateResolved), claimed.ProjectID, claimed.IssueUID, claimed.Type)
		query, args, err = q.ToSQL()
		if err != nil {
			return false, errors.Wrapf(err, "failed to build sql")
		}
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return false, errors.Wrapf(err, "failed to resolve reopened review results")
		}

		// A kept root takes the text and the anchor of the result that
		// repeats it. The fingerprint leaves the text out, so the text can
		// differ: omni rewords a finding, or names what it found, such as an
		// object's kind, differently. Equal fingerprints mean equal statement
		// text, and an anchor left on an earlier sheet can lose its place in
		// the current one. A result on the whole change has no anchor and
		// leaves the kept one alone.
		q = qb.Q().Space(`
			UPDATE issue_comment kept
			SET payload = kept.payload || jsonb_strip_nulls(jsonb_build_object(
			      'comment', r.payload->'comment',
			      'statementAnchor', r.payload->'statementAnchor'))
			FROM unnest(?::JSONB[]) AS r(payload)
			WHERE kept.project = ? AND kept.issue_id = ?
			  AND kept.parent_id IS NULL
			  AND kept.payload->'reviewMetadata'->>'runType' = ?
			  AND kept.payload->'reviewMetadata'->'superseded' IS NULL
			  AND kept.payload->'reviewMetadata'->>'fingerprint' = r.payload->'reviewMetadata'->>'fingerprint'
			  AND ((r.payload ?? 'comment' AND kept.payload->'comment' IS DISTINCT FROM r.payload->'comment')
			    OR (r.payload ?? 'statementAnchor' AND kept.payload->'statementAnchor' IS DISTINCT FROM r.payload->'statementAnchor'))
		`, resultPayloads, claimed.ProjectID, claimed.IssueUID, claimed.Type)
		query, args, err = q.ToSQL()
		if err != nil {
			return false, errors.Wrapf(err, "failed to build sql")
		}
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return false, errors.Wrapf(err, "failed to update kept review results")
		}
	}

	q := qb.Q().Space(`
		UPDATE review_run
		SET status = ?, updated_at = now(), payload = `+keepApprovalInputVersion+`
		WHERE project = ? AND issue_id = ? AND type = ?
		  AND status = ? AND replica_id = ? AND attempt = ?
	`, status.String(), string(payloadBytes),
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
		// Superseded: the rollback discards the resolution.
		return false, nil
	}
	if status != storepb.ReviewRun_DONE {
		if err := tx.Commit(); err != nil {
			return false, errors.Wrapf(err, "failed to commit tx")
		}
		return true, nil
	}

	var postedCount int64
	if len(resultPayloads) > 0 {
		// After the update above, the reviewer's current roots are the kept
		// ones. created_at is offset by the ordinal for the same reason as in
		// CreateIssueComments, starting a microsecond after the completion
		// event inserted below, so the results sort after it.
		q := qb.Q().Space(`
			INSERT INTO issue_comment (creator, project, issue_id, payload, thread_state, created_at, updated_at)
			SELECT NULL, ?, ?, c.payload, 'OPEN', t.at, t.at
			FROM unnest(?::JSONB[]) WITH ORDINALITY AS c(payload, ordinality),
			     LATERAL (SELECT now() + (c.ordinality * interval '1 microsecond')) AS t(at)
			WHERE NOT EXISTS (
				SELECT 1 FROM issue_comment kept
				WHERE kept.project = ? AND kept.issue_id = ?
				  AND kept.parent_id IS NULL
				  AND kept.payload->'reviewMetadata'->>'runType' = ?
				  AND kept.payload->'reviewMetadata'->'superseded' IS NULL
				  AND kept.payload->'reviewMetadata'->>'fingerprint' = c.payload->'reviewMetadata'->>'fingerprint'
			)
		`, claimed.ProjectID, claimed.IssueUID, resultPayloads, claimed.ProjectID, claimed.IssueUID, claimed.Type)
		query, args, err := q.ToSQL()
		if err != nil {
			return false, errors.Wrapf(err, "failed to build sql")
		}
		result, err := tx.ExecContext(ctx, query, args...)
		if err != nil {
			return false, errors.Wrapf(err, "failed to post review results")
		}
		if postedCount, err = result.RowsAffected(); err != nil {
			return false, errors.Wrapf(err, "failed to get rows affected")
		}
	}

	runType := storepb.ReviewRun_Type(storepb.ReviewRun_Type_value[claimed.Type])
	completion, err := protojson.Marshal(&storepb.IssueCommentPayload{
		Event: &storepb.IssueCommentPayload_ReviewRunCompletion_{
			ReviewRunCompletion: &storepb.IssueCommentPayload_ReviewRunCompletion{
				RunType:         runType,
				Attempt:         claimed.Attempt,
				PostedCount:     int32(postedCount),
				KeptCount:       int32(int64(len(resultPayloads)) - postedCount),
				SupersededCount: int32(supersededCount),
			},
		},
	})
	if err != nil {
		return false, errors.Wrapf(err, "failed to marshal review run completion")
	}
	q = qb.Q().Space(`
		INSERT INTO issue_comment (creator, project, issue_id, payload, created_at, updated_at)
		VALUES (NULL, ?, ?, ?, now(), now())
	`, claimed.ProjectID, claimed.IssueUID, string(completion))
	query, args, err = q.ToSQL()
	if err != nil {
		return false, errors.Wrapf(err, "failed to build sql")
	}
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return false, errors.Wrapf(err, "failed to record review run completion")
	}

	if err := tx.Commit(); err != nil {
		return false, errors.Wrapf(err, "failed to commit tx")
	}
	return true, nil
}

// marshalReviewResults checks that every result is a current root comment
// the claimed reviewer posted, with no event, no creator, and a fingerprint
// no other result repeats, and returns the payloads to insert, stamped with
// the run's attempt, and their fingerprints. The fingerprints are never nil:
// <> ALL(NULL) is NULL, which would keep every current root from being
// superseded.
func marshalReviewResults(claimed *ClaimedReviewRun, results []*IssueCommentMessage) ([][]byte, []string, error) {
	payloads := make([][]byte, 0, len(results))
	fingerprints := make([]string, 0, len(results))
	for _, result := range results {
		if result.ProjectID != claimed.ProjectID || result.IssueUID != claimed.IssueUID {
			return nil, nil, errors.Errorf("review result for issue %d in project %s does not belong to the run on issue %d in project %s", result.IssueUID, result.ProjectID, claimed.IssueUID, claimed.ProjectID)
		}
		if result.ParentID != nil || result.CreatorEmail != "" {
			return nil, nil, errors.New("a review result is a root comment with no creator")
		}
		if result.ThreadState != nil && *result.ThreadState != ThreadStateOpen {
			return nil, nil, errors.New("a review result starts its thread OPEN")
		}
		if result.Payload.GetEvent() != nil {
			return nil, nil, errors.New("a review result cannot carry an event")
		}
		metadata := result.Payload.GetReviewMetadata()
		if metadata.GetRunType().String() != claimed.Type {
			return nil, nil, errors.Errorf("review result names reviewer %q; the run is %q", metadata.GetRunType(), claimed.Type)
		}
		if metadata.GetSuperseded() {
			return nil, nil, errors.New("a review result cannot start superseded")
		}
		if fingerprint := metadata.GetFingerprint(); fingerprint != "" {
			if slices.Contains(fingerprints, fingerprint) {
				return nil, nil, errors.Errorf("two review results share the fingerprint %q", fingerprint)
			}
			fingerprints = append(fingerprints, fingerprint)
		}
		if err := validateIssueCommentPayload(result.Payload); err != nil {
			return nil, nil, err
		}
		stamped := proto.CloneOf(result.Payload)
		stamped.ReviewMetadata.Attempt = claimed.Attempt
		payload, err := protojson.Marshal(stamped)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to marshal review result")
		}
		payloads = append(payloads, payload)
	}
	return payloads, fingerprints, nil
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
		SET status = ?, updated_at = now(), payload = `+keepApprovalInputVersion+`
		WHERE status = ?
		  AND replica_id IS NOT NULL
		  AND NOT EXISTS (
			SELECT 1 FROM replica_heartbeat rh
			WHERE rh.replica_id = review_run.replica_id
			  AND rh.last_heartbeat >= now() - ?::INTERVAL
		  )
	`, storepb.ReviewRun_FAILED.String(), string(payloadBytes),
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
