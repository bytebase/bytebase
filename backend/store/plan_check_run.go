package store

import (
	"context"
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
	UID       int
	CreatedAt time.Time
	UpdatedAt time.Time

	PlanUID int64

	Status PlanCheckRunStatus
	Result *storepb.PlanCheckRunResult
}

// FindPlanCheckRunMessage is the message for finding plan check runs.
type FindPlanCheckRunMessage struct {
	PlanUID      *int64
	UIDs         *[]int
	Status       *[]PlanCheckRunStatus
	ResultStatus *[]storepb.Advice_Status
}

// CreatePlanCheckRun creates or replaces the plan check run for a plan.
// Always creates with AVAILABLE status for HA-safe scheduling.
func (s *Store) CreatePlanCheckRun(ctx context.Context, create *PlanCheckRunMessage) error {
	result, err := protojson.Marshal(create.Result)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal result")
	}

	query := `
		INSERT INTO plan_check_run (plan_id, status, result)
		VALUES ($1, $2, $3)
		ON CONFLICT (plan_id) DO UPDATE SET
			status = EXCLUDED.status,
			result = EXCLUDED.result,
			updated_at = now()
	`
	if _, err := s.GetDB().ExecContext(ctx, query, create.PlanUID, PlanCheckRunStatusAvailable, result); err != nil {
		return errors.Wrapf(err, "failed to upsert plan check run")
	}
	return nil
}

// ListPlanCheckRuns returns a list of plan check runs based on find.
func (s *Store) ListPlanCheckRuns(ctx context.Context, find *FindPlanCheckRunMessage) ([]*PlanCheckRunMessage, error) {
	q := qb.Q().Space(`
SELECT
	plan_check_run.id,
	plan_check_run.created_at,
	plan_check_run.updated_at,
	plan_check_run.plan_id,
	plan_check_run.status,
	plan_check_run.result
FROM plan_check_run
WHERE TRUE`)
	if v := find.PlanUID; v != nil {
		q.Space("AND plan_check_run.plan_id = ?", *v)
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
			&planCheckRun.PlanUID,
			&planCheckRun.Status,
			&result,
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
func (s *Store) GetPlanCheckRun(ctx context.Context, planUID int64) (*PlanCheckRunMessage, error) {
	runs, err := s.ListPlanCheckRuns(ctx, &FindPlanCheckRunMessage{PlanUID: &planUID})
	if err != nil {
		return nil, err
	}
	if len(runs) == 0 {
		return nil, nil
	}
	return runs[0], nil
}

// UpdatePlanCheckRun updates a plan check run.
func (s *Store) UpdatePlanCheckRun(ctx context.Context, status PlanCheckRunStatus, result *storepb.PlanCheckRunResult, uid int) error {
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
	WHERE id = ?`, time.Now(), status, resultBytes, uid)
	query, args, err := q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}
	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil {
		return errors.Wrapf(err, "failed to update plan check run")
	}
	return nil
}

// BatchCancelPlanCheckRuns updates the status of planCheckRuns to CANCELED.
func (s *Store) BatchCancelPlanCheckRuns(ctx context.Context, planCheckRunUIDs []int) error {
	q := qb.Q().Space(`
		UPDATE plan_check_run
		SET
			status = ?,
			updated_at = ?
		WHERE id = ANY(?)`, PlanCheckRunStatusCanceled, time.Now(), planCheckRunUIDs)
	query, args, err := q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}
	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil {
		return err
	}
	return nil
}

// ClaimedPlanCheckRun represents a plan check run that was atomically claimed.
type ClaimedPlanCheckRun struct {
	UID     int
	PlanUID int64
}

// ClaimAvailablePlanCheckRuns atomically claims all AVAILABLE plan check runs by updating them to RUNNING
// and returns the claimed UIDs. Uses FOR UPDATE SKIP LOCKED to allow concurrent schedulers to claim different runs.
func (s *Store) ClaimAvailablePlanCheckRuns(ctx context.Context) ([]*ClaimedPlanCheckRun, error) {
	q := qb.Q().Space(`
		UPDATE plan_check_run
		SET status = ?, updated_at = now()
		WHERE id IN (
			SELECT id FROM plan_check_run
			WHERE status = ?
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, plan_id
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
		if err := rows.Scan(&c.UID, &c.PlanUID); err != nil {
			return nil, err
		}
		claimed = append(claimed, &c)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return claimed, nil
}
