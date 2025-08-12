package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// PlanCheckRunType is the type of a plan check run.
type PlanCheckRunType string

const (
	// PlanCheckDatabaseStatementFakeAdvise is the plan check type for fake advise.
	PlanCheckDatabaseStatementFakeAdvise PlanCheckRunType = "bb.plan-check.database.statement.fake-advise"
	// PlanCheckDatabaseStatementCompatibility is the plan check type for statement compatibility.
	PlanCheckDatabaseStatementCompatibility PlanCheckRunType = "bb.plan-check.database.statement.compatibility"
	// PlanCheckDatabaseStatementAdvise is the plan check type for schema system review policy.
	PlanCheckDatabaseStatementAdvise PlanCheckRunType = "bb.plan-check.database.statement.advise"
	// PlanCheckDatabaseStatementSummaryReport is the plan check type for statement summary report.
	PlanCheckDatabaseStatementSummaryReport PlanCheckRunType = "bb.plan-check.database.statement.summary.report"
	// PlanCheckDatabaseConnect is the plan check type for database connection.
	PlanCheckDatabaseConnect PlanCheckRunType = "bb.plan-check.database.connect"
	// PlanCheckDatabaseGhostSync is the plan check type for the gh-ost sync task.
	PlanCheckDatabaseGhostSync PlanCheckRunType = "bb.plan-check.database.ghost.sync"
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
	Type   PlanCheckRunType
	Config *storepb.PlanCheckRunConfig
	Result *storepb.PlanCheckRunResult
}

// FindPlanCheckRunMessage is the message for finding plan check runs.
type FindPlanCheckRunMessage struct {
	PlanUID      *int64
	UIDs         *[]int
	Status       *[]PlanCheckRunStatus
	Type         *[]PlanCheckRunType
	ResultStatus *[]storepb.PlanCheckRunResult_Result_Status
}

// CreatePlanCheckRuns creates new plan check runs.
func (s *Store) CreatePlanCheckRuns(ctx context.Context, plan *PlanMessage, creates ...*PlanCheckRunMessage) error {
	if len(creates) == 0 {
		return nil
	}

	var query strings.Builder
	var values []any
	if _, err := query.WriteString(`INSERT INTO plan_check_run (
		plan_id,
		status,
		type,
		config,
		result
	) VALUES
	`); err != nil {
		return err
	}
	for i, create := range creates {
		config, err := protojson.Marshal(create.Config)
		if err != nil {
			return errors.Wrapf(err, "faield to marshal create config %v", create.Config)
		}
		result, err := protojson.Marshal(create.Result)
		if err != nil {
			return errors.Wrapf(err, "failed to marshal create result %v", create.Result)
		}
		values = append(values,
			create.PlanUID,
			create.Status,
			create.Type,
			config,
			result,
		)
		if i != 0 {
			if _, err := query.WriteString(","); err != nil {
				return err
			}
		}
		count := 5
		if _, err := query.WriteString(getPlaceholders(i*count+1, count)); err != nil {
			return err
		}
	}
	txn, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer txn.Rollback()
	if _, err := txn.ExecContext(ctx, "DELETE FROM plan_check_run WHERE plan_id = $1", plan.UID); err != nil {
		return errors.Wrapf(err, "failed to delete plan check run for plan %d", plan.UID)
	}
	if _, err := txn.ExecContext(ctx, query.String(), values...); err != nil {
		return errors.Wrapf(err, "failed to execute insert")
	}
	return txn.Commit()
}

// ListPlanCheckRuns returns a list of plan check runs based on find.
func (s *Store) ListPlanCheckRuns(ctx context.Context, find *FindPlanCheckRunMessage) ([]*PlanCheckRunMessage, error) {
	where, args := []string{"TRUE"}, []any{}
	if v := find.PlanUID; v != nil {
		where = append(where, fmt.Sprintf("plan_check_run.plan_id = $%d", len(args)+1))
		args = append(args, *v)
	}
	if v := find.UIDs; v != nil {
		where, args = append(where, fmt.Sprintf("plan_check_run.id = ANY($%d)", len(args)+1)), append(args, *v)
	}
	if v := find.Status; v != nil {
		where = append(where, fmt.Sprintf("plan_check_run.status = ANY($%d)", len(args)+1))
		args = append(args, *v)
	}
	if v := find.Type; v != nil {
		where = append(where, fmt.Sprintf("plan_check_run.type = ANY($%d)", len(args)+1))
		args = append(args, *v)
	}
	if v := find.ResultStatus; v != nil {
		statusStrings := make([]string, len(*v))
		for i, status := range *v {
			statusStrings[i] = status.String()
		}
		where = append(where, fmt.Sprintf("EXISTS (SELECT 1 FROM jsonb_array_elements(plan_check_run.result->'results') AS elem WHERE elem->>'status' = ANY($%d))", len(args)+1))
		args = append(args, statusStrings)
	}
	query := fmt.Sprintf(`
SELECT
	plan_check_run.id,
	plan_check_run.created_at,
	plan_check_run.updated_at,
	plan_check_run.plan_id,
	plan_check_run.status,
	plan_check_run.type,
	plan_check_run.config,
	plan_check_run.result
FROM plan_check_run
WHERE %s
ORDER BY plan_check_run.id ASC`, strings.Join(where, " AND "))
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
			&planCheckRun.Type,
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

// UpdatePlanCheckRun updates a plan check run.
func (s *Store) UpdatePlanCheckRun(ctx context.Context, status PlanCheckRunStatus, result *storepb.PlanCheckRunResult, uid int) error {
	resultBytes, err := protojson.Marshal(result)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal result %v", result)
	}
	query := `
    UPDATE plan_check_run
    SET
		updated_at = $1,
		status = $2,
		result = $3
	WHERE id = $4`
	if _, err := s.GetDB().ExecContext(ctx, query, time.Now(), status, resultBytes, uid); err != nil {
		return errors.Wrapf(err, "failed to update plan check run")
	}
	return nil
}

// BatchCancelPlanCheckRuns updates the status of planCheckRuns to CANCELED.
func (s *Store) BatchCancelPlanCheckRuns(ctx context.Context, planCheckRunUIDs []int) error {
	query := `
		UPDATE plan_check_run
		SET 
			status = $1, 
			updated_at = $2
		WHERE id = ANY($3)`
	if _, err := s.GetDB().ExecContext(ctx, query, PlanCheckRunStatusCanceled, time.Now(), planCheckRunUIDs); err != nil {
		return err
	}
	return nil
}
