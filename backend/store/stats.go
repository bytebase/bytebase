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
	where, args := []string{"instance.deleted = $1"}, []any{false}
	if v := find.EnvironmentID; v != nil {
		where, args = append(where, fmt.Sprintf("instance.environment = $%d", len(args)+1)), append(args, *v)
	}
	query := `
		SELECT
			count(1)
		FROM instance
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
		WHERE principal.deleted = $1 AND (principal.type = $2 OR principal.type = $3)`
	var count int
	if err := tx.QueryRowContext(ctx, query, false, api.EndUser, api.ServiceAccount).Scan(&count); err != nil {
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

// CountProjects counts the number of projects and group by workflow type.
// Used by the metric collector.
func (s *Store) CountProjects(ctx context.Context) (int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var count int
	if err := tx.QueryRowContext(ctx, `
		SELECT
			COUNT(1)
		FROM project
		WHERE deleted = FALSE`,
	).Scan(&count); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return count, nil
}

// CountIssues counts the number of issues.
// Used by the metric collector.
func (s *Store) CountIssues(ctx context.Context) (int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var count int
	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(1)
		FROM issue
		WHERE id > 101
	`).Scan(&count); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return count, nil
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
		SELECT metadata->>'engine' AS engine, environment, COUNT(1)
		FROM instance
		WHERE id > 101 AND deleted = FALSE
		GROUP BY engine, environment`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []*metric.InstanceCountMetric
	for rows.Next() {
		var metric metric.InstanceCountMetric
		var engine string
		if err := rows.Scan(&engine, &metric.EnvironmentID, &metric.Count); err != nil {
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
		WHERE id > 102
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

// CountSheetGroupByRowstatusVisibilitySourceAndType counts the number of sheets group by visibility, source and type.
// Used by the metric collector.
func (s *Store) CountSheetGroupByRowstatusVisibilitySourceAndType(ctx context.Context) ([]*metric.SheetCountMetric, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, `
		SELECT visibility, COUNT(*) AS count
		FROM worksheet
		GROUP BY visibility`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []*metric.SheetCountMetric
	for rows.Next() {
		var sheetCount metric.SheetCountMetric
		if err := rows.Scan(
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
