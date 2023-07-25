package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
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
	// PlanCheckDatabaseStatementType is the plan check type for statement type.
	PlanCheckDatabaseStatementType PlanCheckRunType = "bb.plan-check.database.statement.type"
	// PlanCheckDatabaseStatementSummaryReport is the plan check type for statement summary report.
	PlanCheckDatabaseStatementSummaryReport PlanCheckRunType = "bb.plan-check.database.statement.summary.report"
	// PlanCheckDatabaseConnect is the plan check type for database connection.
	PlanCheckDatabaseConnect PlanCheckRunType = "bb.plan-check.database.connect"
	// PlanCheckGhostSync is the plan check type for the gh-ost sync task.
	PlanCheckGhostSync PlanCheckRunType = "bb.plan-check.database.ghost.sync"
	// PlanCheckPITRMySQL is the plan check type for MySQL PITR.
	PlanCheckPITRMySQL PlanCheckRunType = "bb.plan-check.pitr.mysql"
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
	UID        int
	CreatorUID int
	CreatedTs  int64
	UpdaterUID int
	UpdatedTs  int64

	PlanUID int

	Status PlanCheckRunStatus
	Type   PlanCheckRunType
	Config *storepb.PlanCheckRunConfig
	Result *storepb.PlanCheckRunResult
}

type FindPlanCheckRunMessage struct {
	Status *[]PlanCheckRunStatus
}

func (s *Store) CreatePlanCheckRuns(ctx context.Context, creates ...*PlanCheckRunMessage) error {
	var query strings.Builder
	var values []any
	if _, err := query.WriteString(`INSERT INTO plan_check_run (
		creator_id,
		updater_id,
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
			create.CreatorUID,
			create.UpdaterUID,
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
		count := 7
		if _, err := query.WriteString(getPlaceholders(i*count+1, count)); err != nil {
			return err
		}
	}
	if _, err := s.db.db.ExecContext(ctx, query.String(), values...); err != nil {
		return errors.Wrapf(err, "failed to execute insert")
	}
	return nil
}

func (s *Store) ListPlanCheckRuns(ctx context.Context, find *FindPlanCheckRunMessage) ([]*PlanCheckRunMessage, error) {
	where, args := []string{"TRUE"}, []any{}
	if v := find.Status; v != nil {
		var list []string
		for _, status := range *v {
			list = append(list, fmt.Sprintf("$%d", len(args)+1))
			args = append(args, status)
		}
		where = append(where, fmt.Sprintf("plan_check_run.status in (%s)", strings.Join(list, ",")))
	}
	query := fmt.Sprintf(`
		SELECT
			plan_check_run.id,
			plan_check_run.creator_id,
			plan_check_run.created_ts,
			plan_check_run.updater_id,
			plan_check_run.updated_ts,
			plan_check_run.plan_id,
			plan_check_run.status,
			plan_check_run.type,
			plan_check_run.config,
			plan_check_run.result
		FROM plan_check_run
		WHERE %s
	`, strings.Join(where, " AND "))
	rows, err := s.db.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var planCheckRuns []*PlanCheckRunMessage
	for rows.Next() {
		var planCheckRun PlanCheckRunMessage
		var config, result string
		if err := rows.Scan(
			&planCheckRun.UID,
			&planCheckRun.CreatorUID,
			&planCheckRun.CreatedTs,
			&planCheckRun.UpdaterUID,
			&planCheckRun.UpdatedTs,
			&planCheckRun.PlanUID,
			&planCheckRun.Status,
			&planCheckRun.Type,
			&config,
			&result,
		); err != nil {
			return nil, err
		}
		if err := protojson.Unmarshal([]byte(config), planCheckRun.Config); err != nil {
			return nil, err
		}
		if err := protojson.Unmarshal([]byte(result), planCheckRun.Result); err != nil {
			return nil, err
		}
		planCheckRuns = append(planCheckRuns, &planCheckRun)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return planCheckRuns, nil
}
