package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/qb"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// PlanCheckRunStatus is the status of a plan check run.
type PlanCheckRunStatus string

const (
	// PlanCheckRunStatusAvailable is the plan check status for AVAILABLE.
	PlanCheckRunStatusAvailable PlanCheckRunStatus = "AVAILABLE"
	// PlanCheckRunStatusRunning is the plan check status for RUNNING.
	PlanCheckRunStatusRunning PlanCheckRunStatus = "RUNNING"
	// PlanCheckRunStatusDone is the plan check status for DONE.
	PlanCheckRunStatusDone PlanCheckRunStatus = "DONE"
	// PlanCheckRunStatusFailed is the plan check status for FAILED.
	PlanCheckRunStatusFailed PlanCheckRunStatus = "FAILED"
	// PlanCheckRunStatusCanceled is the plan check status for CANCELED.
	PlanCheckRunStatusCanceled PlanCheckRunStatus = "CANCELED"
)

// PlanCheckRunMessage is the message for a plan check run.
type PlanCheckRunMessage struct {
	UID       int64
	CreatedAt time.Time
	UpdatedAt time.Time

	ProjectID string
	PlanUID   int64

	Status PlanCheckRunStatus
	Result *storepb.PlanCheckRunResult
	// Generation is the PostgreSQL row-version token. It changes whenever the
	// durable check row changes, including a same-version rerun that preserves
	// the row UID.
	Generation int64
}

// FindPlanCheckRunMessage is the message for finding plan check runs.
type FindPlanCheckRunMessage struct {
	ProjectID    string
	ProjectIDs   *[]string
	PlanUID      *int64
	PlanUIDs     *[]int64
	UIDs         *[]int64
	Status       *[]PlanCheckRunStatus
	ResultStatus *[]storepb.Advice_Status
}

// CreatePlanCheckRun creates or refreshes the plan check run for a plan.
// It skips active same-version rows to avoid resetting checks already queued or running.
func (s *Store) CreatePlanCheckRun(ctx context.Context, create *PlanCheckRunMessage) (bool, error) {
	result, err := protojson.Marshal(create.Result)
	if err != nil {
		return false, errors.Wrapf(err, "failed to marshal result")
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return false, errors.Wrapf(err, "failed to begin tx")
	}
	defer tx.Rollback()
	if err := acquirePlanIssueRolloutAdvisoryLock(ctx, tx, create.ProjectID, create.PlanUID); err != nil {
		return false, errors.Wrap(err, "failed to acquire Plan review lock for Plan check run")
	}
	var lockedPlanUID int64
	if err := tx.QueryRowContext(ctx, `
		SELECT id
		FROM plan
		WHERE project = $1 AND id = $2
		FOR UPDATE`, create.ProjectID, create.PlanUID).Scan(&lockedPlanUID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, errors.Wrap(err, "failed to lock Plan for Plan check run")
	}

	nextID, err := nextProjectID(ctx, tx, "plan_check_run", create.ProjectID)
	if err != nil {
		return false, err
	}

	approvalInputVersion := create.Result.GetApprovalInputVersion()
	query := `
		INSERT INTO plan_check_run (id, project, plan_id, status, result)
		SELECT $1, $2, $3, $4, $5
		FROM plan
		WHERE plan.project = $2
		  AND plan.id = $3
		  AND COALESCE((plan.config->>'approvalInputVersion')::bigint, 0) = $6
		ON CONFLICT (project, plan_id) DO UPDATE SET
			status = EXCLUDED.status,
			result = EXCLUDED.result,
			updated_at = now()
		WHERE (plan_check_run.status NOT IN ($7, $8)
		   OR COALESCE((plan_check_run.result->>'approvalInputVersion')::bigint, 0) != COALESCE((EXCLUDED.result->>'approvalInputVersion')::bigint, 0)
		)
		  AND EXISTS (
			SELECT 1
			FROM plan
			WHERE plan.project = plan_check_run.project
			  AND plan.id = plan_check_run.plan_id
			  AND COALESCE((plan.config->>'approvalInputVersion')::bigint, 0) = $9
		  )
	`
	sqlResult, err := tx.ExecContext(ctx, query, nextID, create.ProjectID, create.PlanUID, PlanCheckRunStatusAvailable, result, approvalInputVersion, PlanCheckRunStatusAvailable, PlanCheckRunStatusRunning, approvalInputVersion)
	if err != nil {
		return false, errors.Wrapf(err, "failed to upsert plan check run")
	}
	rowsAffected, err := sqlResult.RowsAffected()
	if err != nil {
		return false, errors.Wrapf(err, "failed to inspect plan check run upsert")
	}

	if err := tx.Commit(); err != nil {
		return false, errors.Wrapf(err, "failed to commit tx")
	}
	return rowsAffected > 0, nil
}

// ListPlanCheckRuns returns a list of plan check runs based on find.
func (s *Store) ListPlanCheckRuns(ctx context.Context, find *FindPlanCheckRunMessage) ([]*PlanCheckRunMessage, error) {
	q := qb.Q().Space(`
		SELECT
			plan_check_run.id,
			plan_check_run.created_at,
			plan_check_run.updated_at,
			plan_check_run.project,
			plan_check_run.plan_id,
			plan_check_run.status,
			plan_check_run.result,
			plan_check_run.xmin::text::bigint
		FROM plan_check_run
		WHERE TRUE`)
	if v := find.ProjectID; v != "" {
		q.Space("AND plan_check_run.project = ?", v)
	}
	if v := find.ProjectIDs; v != nil {
		if len(*v) == 1 {
			q.Space("AND plan_check_run.project = ?", (*v)[0])
		} else if len(*v) > 1 {
			q.Space("AND plan_check_run.project = ANY(?)", *v)
		}
	}
	if v := find.PlanUID; v != nil {
		q.Space("AND plan_check_run.plan_id = ?", *v)
	}
	if v := find.PlanUIDs; v != nil {
		q.Space("AND plan_check_run.plan_id = ANY(?)", *v)
	}
	if v := find.UIDs; v != nil {
		q.Space("AND plan_check_run.id = ANY(?)", *v)
	}
	if v := find.Status; v != nil {
		q.Space("AND plan_check_run.status = ANY(?)", *v)
	}
	if v := find.ResultStatus; v != nil {
		statusStrings := make([]string, len(*v))
		for i, status := range *v {
			statusStrings[i] = status.String()
		}
		q.Space("AND EXISTS (SELECT 1 FROM jsonb_array_elements(plan_check_run.result->'results') AS elem WHERE elem->>'status' = ANY(?))", statusStrings)
	}
	q.Space("ORDER BY plan_check_run.id ASC")
	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}
	rows, err := s.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var planCheckRuns []*PlanCheckRunMessage
	for rows.Next() {
		planCheckRun := PlanCheckRunMessage{
			Result: &storepb.PlanCheckRunResult{},
		}
		var result string
		if err := rows.Scan(
			&planCheckRun.UID,
			&planCheckRun.CreatedAt,
			&planCheckRun.UpdatedAt,
			&planCheckRun.ProjectID,
			&planCheckRun.PlanUID,
			&planCheckRun.Status,
			&result,
			&planCheckRun.Generation,
		); err != nil {
			return nil, err
		}
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(result), planCheckRun.Result); err != nil {
			return nil, err
		}
		planCheckRuns = append(planCheckRuns, &planCheckRun)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return planCheckRuns, nil
}

// GetPlanCheckRun returns the plan check run for a plan.
func (s *Store) GetPlanCheckRun(ctx context.Context, projectID string, planUID int64) (*PlanCheckRunMessage, error) {
	runs, err := s.ListPlanCheckRuns(ctx, &FindPlanCheckRunMessage{ProjectID: projectID, PlanUID: &planUID})
	if err != nil {
		return nil, err
	}
	if len(runs) == 0 {
		return nil, nil
	}
	return runs[0], nil
}

// UpdatePlanCheckRun updates a plan check run.
func (s *Store) UpdatePlanCheckRun(ctx context.Context, projectID string, status PlanCheckRunStatus, result *storepb.PlanCheckRunResult, uid int64) error {
	resultBytes, err := protojson.Marshal(result)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal result %v", result)
	}
	q := qb.Q().Space(`
		UPDATE plan_check_run
		SET
			updated_at = ?,
			status = ?,
			result = ?
		WHERE id = ? AND project = ?`, time.Now(), status, resultBytes, uid, projectID)
	query, args, err := q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}
	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil {
		return errors.Wrapf(err, "failed to update plan check run")
	}
	return nil
}

// UpdatePlanCheckRunIfApprovalInputVersion updates a running plan check run only when it still matches the claimed approval input version.
//
// Plan check workers finish against the row version they claimed. We do not join back to the
// current plan here because plan edits move the row forward separately; this update only prevents
// an old worker from publishing over a row that has already been refreshed.
func (s *Store) UpdatePlanCheckRunIfApprovalInputVersion(ctx context.Context, projectID string, status PlanCheckRunStatus, result *storepb.PlanCheckRunResult, uid int64, approvalInputVersion int64) (bool, error) {
	resultBytes, err := protojson.Marshal(result)
	if err != nil {
		return false, errors.Wrapf(err, "failed to marshal result %v", result)
	}
	q := qb.Q().Space(`
		UPDATE plan_check_run
		SET
			updated_at = ?,
			status = ?,
			result = ?
		WHERE id = ?
		  AND project = ?
		  AND status = ?
		  AND COALESCE((result->>'approvalInputVersion')::bigint, 0) = ?`,
		time.Now(), status, resultBytes, uid, projectID, PlanCheckRunStatusRunning, approvalInputVersion)
	query, args, err := q.ToSQL()
	if err != nil {
		return false, errors.Wrapf(err, "failed to build sql")
	}
	sqlResult, err := s.GetDB().ExecContext(ctx, query, args...)
	if err != nil {
		return false, errors.Wrapf(err, "failed to update plan check run")
	}
	rowsAffected, err := sqlResult.RowsAffected()
	if err != nil {
		return false, errors.Wrapf(err, "failed to inspect plan check run update")
	}
	return rowsAffected > 0, nil
}

// RefreshPlanCheckRunIfStaleApprovalInputVersion refreshes a terminal stale-version plan check run to AVAILABLE.
//
// Approval recompute may observe an older terminal row while another request is already moving
// the plan forward. Only move rows to a newer materialized version; never rewind a row that has
// already advanced beyond the caller's observed plan version.
func (s *Store) RefreshPlanCheckRunIfStaleApprovalInputVersion(ctx context.Context, projectID string, planUID int64, approvalInputVersion int64) (bool, error) {
	result := &storepb.PlanCheckRunResult{ApprovalInputVersion: approvalInputVersion}
	resultBytes, err := protojson.Marshal(result)
	if err != nil {
		return false, errors.Wrapf(err, "failed to marshal result")
	}

	q := qb.Q().Space(`
		UPDATE plan_check_run
		SET
			updated_at = ?,
			status = ?,
			result = ?
		WHERE project = ?
		  AND plan_id = ?
		  AND status NOT IN (?, ?)
		  AND COALESCE((result->>'approvalInputVersion')::bigint, 0) < ?`,
		time.Now(), PlanCheckRunStatusAvailable, resultBytes, projectID, planUID, PlanCheckRunStatusAvailable, PlanCheckRunStatusRunning, approvalInputVersion)
	query, args, err := q.ToSQL()
	if err != nil {
		return false, errors.Wrapf(err, "failed to build sql")
	}
	sqlResult, err := s.GetDB().ExecContext(ctx, query, args...)
	if err != nil {
		return false, errors.Wrapf(err, "failed to refresh stale plan check run")
	}
	rowsAffected, err := sqlResult.RowsAffected()
	if err != nil {
		return false, errors.Wrapf(err, "failed to inspect stale plan check run refresh")
	}
	return rowsAffected > 0, nil
}

// BatchCancelPlanCheckRuns updates the status of planCheckRuns to CANCELED.
func (s *Store) BatchCancelPlanCheckRuns(ctx context.Context, projectID string, planCheckRunUIDs []int64) error {
	q := qb.Q().Space(`
		UPDATE plan_check_run
		SET
			status = ?,
			updated_at = ?
		WHERE id = ANY(?) AND project = ?`, PlanCheckRunStatusCanceled, time.Now(), planCheckRunUIDs, projectID)
	query, args, err := q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}
	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil {
		return err
	}
	return nil
}

// CancelPlanCheckRunIfApprovalInputVersion cancels an active plan check run only when it still matches the observed approval input version.
func (s *Store) CancelPlanCheckRunIfApprovalInputVersion(ctx context.Context, projectID string, planCheckRunUID int64, approvalInputVersion int64) (bool, error) {
	q := qb.Q().Space(`
		UPDATE plan_check_run
		SET
			status = ?,
			updated_at = ?
		WHERE id = ?
		  AND project = ?
		  AND status IN (?, ?)
		  AND COALESCE((result->>'approvalInputVersion')::bigint, 0) = ?`,
		PlanCheckRunStatusCanceled, time.Now(), planCheckRunUID, projectID, PlanCheckRunStatusAvailable, PlanCheckRunStatusRunning, approvalInputVersion)
	query, args, err := q.ToSQL()
	if err != nil {
		return false, errors.Wrapf(err, "failed to build sql")
	}
	result, err := s.GetDB().ExecContext(ctx, query, args...)
	if err != nil {
		return false, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, errors.Wrapf(err, "failed to inspect plan check run cancel")
	}
	return rowsAffected > 0, nil
}

// ClaimedPlanCheckRun represents a plan check run that was atomically claimed.
type ClaimedPlanCheckRun struct {
	UID                  int64
	ProjectID            string
	PlanUID              int64
	ApprovalInputVersion int64
}

// FailStalePlanCheckRuns marks RUNNING plan check runs as FAILED if they have been running
// longer than the timeout threshold. Unlike task runs, plan check runs don't use heartbeat
// detection - they use a simple timeout since they have a bounded execution time.
func (s *Store) FailStalePlanCheckRuns(ctx context.Context, timeout time.Duration) (int64, error) {
	q := qb.Q().Space(`
		UPDATE plan_check_run
		SET status = ?,
		    result = jsonb_build_object(
		        'approvalInputVersion', COALESCE((result->>'approvalInputVersion')::bigint, 0),
		        'results', jsonb_build_array(jsonb_build_object(
		            'status', 'ERROR',
		            'title', 'Plan check run timed out',
		            'content', 'The plan check run was abandoned because it exceeded the maximum execution time.'
		        ))
		    ),
		    updated_at = now()
		WHERE status = ?
		  AND updated_at < now() - ?::INTERVAL
	`, PlanCheckRunStatusFailed, PlanCheckRunStatusRunning, timeout.String())

	query, args, err := q.ToSQL()
	if err != nil {
		return 0, errors.Wrapf(err, "failed to build sql")
	}

	result, err := s.GetDB().ExecContext(ctx, query, args...)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to mark stale plan check runs as failed")
	}

	return result.RowsAffected()
}

// ClaimAvailablePlanCheckRuns atomically claims all AVAILABLE plan check runs by updating them to RUNNING
// and returns the claimed UIDs. Uses FOR UPDATE SKIP LOCKED to allow concurrent schedulers to claim different runs.
func (s *Store) ClaimAvailablePlanCheckRuns(ctx context.Context) ([]*ClaimedPlanCheckRun, error) {
	q := qb.Q().Space(`
		UPDATE plan_check_run
		SET status = ?, updated_at = now()
		WHERE (project, id) IN (
			SELECT project, id FROM plan_check_run
			WHERE status = ?
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, project, plan_id, COALESCE((result->>'approvalInputVersion')::bigint, 0)
	`, PlanCheckRunStatusRunning, PlanCheckRunStatusAvailable)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	rows, err := s.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to claim plan check runs")
	}
	defer rows.Close()

	var claimed []*ClaimedPlanCheckRun
	for rows.Next() {
		var c ClaimedPlanCheckRun
		if err := rows.Scan(&c.UID, &c.ProjectID, &c.PlanUID, &c.ApprovalInputVersion); err != nil {
			return nil, err
		}
		claimed = append(claimed, &c)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return claimed, nil
}
