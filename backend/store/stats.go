package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/metric"
)

// CountInstanceMessage is the message for counting instances.
type CountInstanceMessage struct {
	EnvironmentID *string
}

// CountInstance counts the number of instances.
func (s *Store) CountInstance(ctx context.Context, find *CountInstanceMessage) (int, error) {
	where, args := []string{"instance.row_status = $1"}, []any{api.Normal}
	if v := find.EnvironmentID; v != nil {
		where, args = append(where, fmt.Sprintf("environment.resource_id = $%d", len(args)+1)), append(args, *v)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	query := `
		SELECT
			count(1)
		FROM instance
		LEFT JOIN environment ON environment.id = instance.environment_id
		WHERE ` + strings.Join(where, " AND ")
	var count int
	if err := tx.QueryRowContext(ctx, query,
		args...).Scan(&count); err != nil {
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

// CountPrincipal counts the number of endusers.
func (s *Store) CountPrincipal(ctx context.Context) (int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	query := `
		SELECT
			count(1)
		FROM principal
		WHERE principal.type = $1`
	var count int
	if err := tx.QueryRowContext(ctx, `
		SELECT
			count(1)
		FROM principal
		WHERE principal.type = $1
	`, api.EndUser).Scan(&count); err != nil {
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

// CountDatabaseGroupByBackupScheduleAndEnabled counts database, group by backup schedule and enabled.
func (s *Store) CountDatabaseGroupByBackupScheduleAndEnabled(ctx context.Context) ([]*metric.DatabaseCountMetric, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, `
		WITH database_backup_policy AS (
			SELECT db.id AS database_id, backup_policy.payload AS payload
			FROM db, instance LEFT JOIN (
				SELECT resource_id, payload
				FROM policy
				WHERE type = 'bb.policy.backup-plan'
			) AS backup_policy ON instance.environment_id = backup_policy.resource_id
			WHERE db.instance_id = instance.id
		), database_backup_setting AS(
			SELECT db.id AS database_id, backup_setting.enabled AS enabled
			FROM db LEFT JOIN backup_setting ON db.id = backup_setting.database_id
		)
		SELECT database_backup_policy.payload, database_backup_setting.enabled, COUNT(*)
		FROM database_backup_policy FULL JOIN database_backup_setting
			ON database_backup_policy.database_id = database_backup_setting.database_id
		GROUP BY database_backup_policy.payload, database_backup_setting.enabled
		`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var databaseCountMetricList []*metric.DatabaseCountMetric
	for rows.Next() {
		var optionalPayload sql.NullString
		var optionalEnabled sql.NullBool
		var count int
		if err := rows.Scan(&optionalPayload, &optionalEnabled, &count); err != nil {
			return nil, err
		}
		var backupPlanPolicySchedule *api.BackupPlanPolicySchedule
		if optionalPayload.Valid {
			backupPlanPolicy, err := api.UnmarshalBackupPlanPolicy(optionalPayload.String)
			if err != nil {
				return nil, err
			}
			backupPlanPolicySchedule = &backupPlanPolicy.Schedule
		}
		var enabled *bool
		if optionalEnabled.Valid {
			enabled = &optionalEnabled.Bool
		}
		databaseCountMetricList = append(databaseCountMetricList, &metric.DatabaseCountMetric{
			BackupPlanPolicySchedule: backupPlanPolicySchedule,
			BackupSettingEnabled:     enabled,
			Count:                    count,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return databaseCountMetricList, nil
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
		SELECT role, status, member.row_status AS row_status, principal.type, COUNT(*)
		FROM member
		LEFT JOIN principal ON principal.id = member.principal_id
		GROUP BY role, status, member.row_status, principal.type`,
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
		SELECT tenant_mode, workflow_type, row_status, COUNT(*)
		FROM project
		GROUP BY tenant_mode, workflow_type, row_status`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []*metric.ProjectCountMetric
	for rows.Next() {
		var metric metric.ProjectCountMetric
		if err := rows.Scan(&metric.TenantMode, &metric.WorkflowType, &metric.RowStatus, &metric.Count); err != nil {
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

// CountInstanceGroupByEngineAndEnvironmentID counts the number of instances and group by engine and environment_id.
// Used by the metric collector.
func (s *Store) CountInstanceGroupByEngineAndEnvironmentID(ctx context.Context) ([]*metric.InstanceCountMetric, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, `
		SELECT engine, environment_id, row_status, COUNT(*)
		FROM instance
		WHERE (id <= 101 AND updater_id != 1) OR id > 101
		GROUP BY engine, environment_id, row_status`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []*metric.InstanceCountMetric
	for rows.Next() {
		var metric metric.InstanceCountMetric
		if err := rows.Scan(&metric.Engine, &metric.EnvironmentID, &metric.RowStatus, &metric.Count); err != nil {
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
		SELECT row_status, visibility, source, type, COUNT(*) AS count
		FROM sheet
		GROUP BY row_status, visibility, source, type`)
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
			&sheetCount.Source,
			&sheetCount.Type,
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
