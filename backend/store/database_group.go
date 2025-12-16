package store

import (
	"context"
	"database/sql"

	"github.com/pkg/errors"
	"google.golang.org/genproto/googleapis/type/expr"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/qb"
)

// DatabaseGroupMessage is the message for database groups.
type DatabaseGroupMessage struct {
	ProjectID  string
	ResourceID string
	Title      string
	Expression *expr.Expr
}

// FindDatabaseGroupMessage is the message for finding database group.
type FindDatabaseGroupMessage struct {
	ProjectID  *string
	ResourceID *string
}

// UpdateDatabaseGroupMessage is the message for updating database group.
type UpdateDatabaseGroupMessage struct {
	Title      *string
	Expression *expr.Expr
}

// DeleteDatabaseGroup deletes a database group.
func (s *Store) DeleteDatabaseGroup(ctx context.Context, projectID, resourceID string) error {
	q := qb.Q().Space("DELETE FROM db_group WHERE project = ? AND resource_id = ?", projectID, resourceID)
	query, args, err := q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to begin transaction")
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return errors.Wrapf(err, "failed to execute")
	}
	if err := tx.Commit(); err != nil {
		return errors.Wrapf(err, "failed to commit transaction")
	}
	return nil
}

// ListDatabaseGroups lists database groups.
func (s *Store) ListDatabaseGroups(ctx context.Context, find *FindDatabaseGroupMessage) ([]*DatabaseGroupMessage, error) {
	tx, err := s.GetDB().BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	q := qb.Q().Space(`
		SELECT
			project,
			resource_id,
			name,
			expression
		FROM db_group
		WHERE TRUE
	`)

	if v := find.ProjectID; v != nil {
		q.And("project = ?", *v)
	}
	if v := find.ResourceID; v != nil {
		q.And("resource_id = ?", *v)
	}

	q.Space("ORDER BY project, resource_id ASC")

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	var databaseGroups []*DatabaseGroupMessage
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to scan")
	}
	defer rows.Close()
	for rows.Next() {
		var databaseGroup DatabaseGroupMessage
		var exprBytes []byte
		dest := []any{
			&databaseGroup.ProjectID,
			&databaseGroup.ResourceID,
			&databaseGroup.Title,
			&exprBytes,
		}
		if err := rows.Scan(dest...); err != nil {
			return nil, errors.Wrapf(err, "failed to scan")
		}
		var expression expr.Expr
		if err := common.ProtojsonUnmarshaler.Unmarshal(exprBytes, &expression); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal expression")
		}
		databaseGroup.Expression = &expression
		databaseGroups = append(databaseGroups, &databaseGroup)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrapf(err, "failed to scan")
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}

	return databaseGroups, nil
}

// GetDatabaseGroup gets a database group.
func (s *Store) GetDatabaseGroup(ctx context.Context, find *FindDatabaseGroupMessage) (*DatabaseGroupMessage, error) {
	tx, err := s.GetDB().BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	databaseGroups, err := s.ListDatabaseGroups(ctx, find)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}

	if len(databaseGroups) == 0 {
		return nil, nil
	}
	if len(databaseGroups) > 1 {
		return nil, errors.Errorf("found multiple database groups")
	}
	return databaseGroups[0], nil
}

// UpdateDatabaseGroup updates a database group.
func (s *Store) UpdateDatabaseGroup(ctx context.Context, projectID, resourceID string, patch *UpdateDatabaseGroupMessage) (*DatabaseGroupMessage, error) {
	set := qb.Q()
	if v := patch.Title; v != nil {
		set.Comma("name = ?", *v)
	}
	if v := patch.Expression; v != nil {
		exprBytes, err := protojson.Marshal(patch.Expression)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to marshal expression")
		}
		set.Comma("expression = ?", exprBytes)
	}
	if set.Len() == 0 {
		return nil, errors.New("no fields to update")
	}

	q := qb.Q().Space("UPDATE db_group SET ? WHERE project = ? AND resource_id = ? RETURNING project, resource_id, name, expression", set, projectID, resourceID)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to commit ")
	}

	var updatedDatabaseGroup DatabaseGroupMessage
	var exprBytes []byte
	if err := tx.QueryRowContext(
		ctx,
		query,
		args...,
	).Scan(
		&updatedDatabaseGroup.ProjectID,
		&updatedDatabaseGroup.ResourceID,
		&updatedDatabaseGroup.Title,
		&exprBytes,
	); err != nil {
		return nil, errors.Wrapf(err, "failed to scan")
	}
	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}
	var expression expr.Expr
	if err := common.ProtojsonUnmarshaler.Unmarshal(exprBytes, &expression); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal expression")
	}
	updatedDatabaseGroup.Expression = &expression
	return &updatedDatabaseGroup, nil
}

// CreateDatabaseGroup creates a database group.
func (s *Store) CreateDatabaseGroup(ctx context.Context, create *DatabaseGroupMessage) (*DatabaseGroupMessage, error) {
	exprBytes, err := protojson.Marshal(create.Expression)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal expression")
	}

	q := qb.Q().Space(`
		INSERT INTO db_group (
			project,
			resource_id,
			name,
			expression
		) VALUES (?, ?, ?, ?)
	`, create.ProjectID, create.ResourceID, create.Title, exprBytes)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return nil, errors.Wrapf(err, "failed to scan")
	}
	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}

	return create, nil
}
