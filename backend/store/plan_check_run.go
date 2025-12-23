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
	Config *storepb.PlanCheckRunConfig
	Result *storepb.PlanCheckRunResult
}

// FindPlanCheckRunMessage is the message for finding plan check runs.
type FindPlanCheckRunMessage struct {
	PlanUID      *int64
	UIDs         *[]int
	Status       *[]PlanCheckRunStatus
	ResultStatus *[]storepb.Advice_Status
}

// CreatePlanCheckRuns creates new plan check runs.
func (s *Store) CreatePlanCheckRuns(ctx context.Context, plan *PlanMessage, creates ...*PlanCheckRunMessage) error {
	if len(creates) == 0 {
		return nil
	}

	txn, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer txn.Rollback()

	// Delete existing plan check runs
	q := qb.Q().Space("DELETE FROM plan_check_run WHERE plan_id = ?", plan.UID)
	query, args, err := q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build delete sql")
	}
	if _, err := txn.ExecContext(ctx, query, args...); err != nil {
		return errors.Wrapf(err, "failed to delete plan check run for plan %d", plan.UID)
	}

	// Insert new plan check runs
	q = qb.Q().Space("INSERT INTO plan_check_run (plan_id, status, config, result) VALUES")
	for i, create := range creates {
		config, err := protojson.Marshal(create.Config)
		if err != nil {
			return errors.Wrapf(err, "failed to marshal create config")
		}
		result, err := protojson.Marshal(create.Result)
		if err != nil {
			return errors.Wrapf(err, "failed to marshal create result %v", create.Result)
		}
		if i > 0 {
			q.Space(",")
		}
		q.Space("(?, ?, ?, ?)", create.PlanUID, create.Status, config, result)
	}
	query, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build insert sql")
	}
	if _, err := txn.ExecContext(ctx, query, args...); err != nil {
		return errors.Wrapf(err, "failed to execute insert")
	}
	return txn.Commit()
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
	plan_check_run.config,
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
			Config: &storepb.PlanCheckRunConfig{},
			Result: &storepb.PlanCheckRunResult{},
		}
		var config, result string
		if err := rows.Scan(
			&planCheckRun.UID,
			&planCheckRun.CreatedAt,
			&planCheckRun.UpdatedAt,
			&planCheckRun.PlanUID,
			&planCheckRun.Status,
			&config,
			&result,
		); err != nil {
			return nil, err
		}
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(config), planCheckRun.Config); err != nil {
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
	// With consolidated model, there should be only one record per plan
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
