package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/metric"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// CountInstanceMessage is the message for counting instances.
type CountInstanceMessage struct {
	EnvironmentID *string
}

// CountUsers counts the principal.
func (s *Store) CountUsers(ctx context.Context, userType api.PrincipalType) (int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	count := 0

	if err := tx.QueryRowContext(ctx, `
	SELECT COUNT(*)
	FROM principal
	WHERE principal.type = $1`,
		userType).Scan(&count); err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return count, nil
}

// CountInstance counts the number of instances.
func (s *Store) CountInstance(ctx context.Context, find *CountInstanceMessage) (int, error) {
	where, args := []string{"instance.row_status = $1"}, []any{api.Normal}
	if v := find.EnvironmentID; v != nil {
		where, args = append(where, fmt.Sprintf("environment.resource_id = $%d", len(args)+1)), append(args, *v)
	}
	query := `
		SELECT
			count(1)
		FROM instance
		LEFT JOIN environment ON environment.resource_id = instance.environment
		WHERE ` + strings.Join(where, " AND ")

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var count int
	if err := tx.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return count, nil
}

// CountActiveUsers counts the number of endusers.
func (s *Store) CountActiveUsers(ctx context.Context) (int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	query := `
		SELECT
			count(DISTINCT principal.id)
		FROM principal
		WHERE principal.row_status = $1 AND (principal.type = $2 OR principal.type = $3)`
	var count int
	if err := tx.QueryRowContext(ctx, query, api.Normal, api.EndUser, api.ServiceAccount).Scan(&count); err != nil {
		if err == sql.ErrNoRows {
			return 0, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return count, nil
}

// CountMemberGroupByRoleAndStatus counts the number of member and group by role and status.
// Used by the metric collector.
func (s *Store) CountMemberGroupByRoleAndStatus(ctx context.Context) ([]*metric.MemberCountMetric, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, `
		SELECT role, 'ACTIVE' AS status, principal.row_status AS row_status, principal.type, COUNT(*)
		FROM member
		LEFT JOIN principal ON principal.id = member.principal_id
		GROUP BY role, status, principal.row_status, principal.type`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []*metric.MemberCountMetric
	for rows.Next() {
		var metric metric.MemberCountMetric
		if err := rows.Scan(
			&metric.Role,
			&metric.Status,
			&metric.RowStatus,
			&metric.Type,
			&metric.Count,
		); err != nil {
			return nil, err
		}
		res = append(res, &metric)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return res, nil
}

// CountProjectGroupByTenantModeAndWorkflow counts the number of projects and group by tenant mode and workflow type.
// Used by the metric collector.
func (s *Store) CountProjectGroupByTenantModeAndWorkflow(ctx context.Context) ([]*metric.ProjectCountMetric, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, `
		WITH project_workflow AS (
			SELECT
				project.tenant_mode as tenant_mode,
				project.row_status as row_status,
				(SELECT COUNT(1) FROM vcs_connector WHERE project.id = vcs_connector.project_id) > 0 AS has_connector
			FROM project
			WHERE resource_id != 'default'
		)
		SELECT
			tenant_mode,
			has_connector,
			row_status,
			COUNT(*)
		FROM project_workflow
		GROUP BY tenant_mode, has_connector, row_status`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []*metric.ProjectCountMetric
	for rows.Next() {
		var metric metric.ProjectCountMetric
		var hasConnector bool
		if err := rows.Scan(&metric.TenantMode, &hasConnector, &metric.RowStatus, &metric.Count); err != nil {
			return nil, err
		}
		workflow := v1pb.Workflow_UI
		if hasConnector {
			workflow = v1pb.Workflow_VCS
		}
		metric.WorkflowType = workflow
		res = append(res, &metric)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return res, nil
}

// CountIssueGroupByTypeAndStatus counts the number of issue and group by type and status.
// Used by the metric collector.
func (s *Store) CountIssueGroupByTypeAndStatus(ctx context.Context) ([]*metric.IssueCountMetric, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, `
		SELECT type, status, COUNT(*)
		FROM issue
		WHERE (id <= 101 AND updater_id != 1) OR id > 101
		GROUP BY type, status`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []*metric.IssueCountMetric

	for rows.Next() {
		var metric metric.IssueCountMetric
		if err := rows.Scan(&metric.Type, &metric.Status, &metric.Count); err != nil {
			return nil, err
		}
		res = append(res, &metric)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return res, nil
}

// CountInstanceGroupByEngineAndEnvironmentID counts the number of instances and group by engine and environment.
// Used by the metric collector.
func (s *Store) CountInstanceGroupByEngineAndEnvironmentID(ctx context.Context) ([]*metric.InstanceCountMetric, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, `
		SELECT engine, environment, row_status, COUNT(*)
		FROM instance
		WHERE (id <= 101 AND updater_id != 1) OR id > 101
		GROUP BY engine, environment, row_status`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []*metric.InstanceCountMetric
	for rows.Next() {
		var metric metric.InstanceCountMetric
		var engine string
		if err := rows.Scan(&engine, &metric.EnvironmentID, &metric.RowStatus, &metric.Count); err != nil {
			return nil, err
		}
		engineTypeValue, ok := storepb.Engine_value[engine]
		if !ok {
			return nil, errors.Errorf("invalid engine %s", engine)
		}
		metric.Engine = storepb.Engine(engineTypeValue)
		res = append(res, &metric)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return res, nil
}

// CountTaskGroupByTypeAndStatus counts the number of TaskGroup and group by TaskType.
// Used for the metric collector.
func (s *Store) CountTaskGroupByTypeAndStatus(ctx context.Context) ([]*metric.TaskCountMetric, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, `
		SELECT type, status, COUNT(*)
		FROM task
		WHERE (id <= 102 AND updater_id != 1) OR id > 102
		GROUP BY type, status`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []*metric.TaskCountMetric
	for rows.Next() {
		var metric metric.TaskCountMetric
		if err := rows.Scan(&metric.Type, &metric.Status, &metric.Count); err != nil {
			return nil, err
		}
		res = append(res, &metric)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return res, nil
}

// CountSheetGroupByRowstatusVisibilitySourceAndType counts the number of sheets group by row_status, visibility, source and type.
// Used by the metric collector.
func (s *Store) CountSheetGroupByRowstatusVisibilitySourceAndType(ctx context.Context) ([]*metric.SheetCountMetric, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, `
		SELECT row_status, visibility, COUNT(*) AS count
		FROM worksheet
		GROUP BY row_status, visibility`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []*metric.SheetCountMetric
	for rows.Next() {
		var sheetCount metric.SheetCountMetric
		if err := rows.Scan(
			&sheetCount.RowStatus,
			&sheetCount.Visibility,
			&sheetCount.Count,
		); err != nil {
			return nil, err
		}
		res = append(res, &sheetCount)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return res, nil
}
