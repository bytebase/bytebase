package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/genproto/googleapis/type/expr"
	"google.golang.org/protobuf/encoding/protojson"
)

// DatabaseGroupMessage is the message for database groups.
type DatabaseGroupMessage struct {
	// Output only fields.
	//
	ID        int64
	CreatedTs int64
	UpdatedTs int64
	CreatorID int
	UpdaterID int

	// Normal fields.
	//
	ProjectResourceID string
	ResourceID        string
	Placeholder       string
	Expression        *expr.Expr
}

// UpdateDatabaseGroupMessage is the message for updating database group.
type UpdateDatabaseGroupMessage struct {
	Placeholder *string
	Expression  *expr.Expr
}

// DeleteDatabaseGroup deletes a database group.
func (s *Store) DeleteDatabaseGroup(ctx context.Context, resourceID string) error {
	query := `
	DELETE FROM db_group WHERE resource_id = $1;
	`
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to begin transaction")
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, query, resourceID); err != nil {
		return errors.Wrapf(err, "failed to exec")
	}
	if err := tx.Commit(); err != nil {
		return errors.Wrapf(err, "failed to commit transaction")
	}
	return nil
}

// GetDatabaseGroup gets a database group.
func (s *Store) GetDatabaseGroup(ctx context.Context, resourceID string) (*DatabaseGroupMessage, error) {
	query := `
	SELECT
		id,
		created_ts,
		updated_ts,
		creator_id,
		updater_id,
		project_resource_id,
		resource_id,
		placeholder,
		expression
	FROM db_group;
	`
	var databaseGroup DatabaseGroupMessage
	var stringExpr string
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	if err := tx.QueryRowContext(ctx, query).Scan(
		&databaseGroup.ID,
		&databaseGroup.CreatedTs,
		&databaseGroup.UpdatedTs,
		&databaseGroup.CreatorID,
		&databaseGroup.UpdaterID,
		&databaseGroup.ProjectResourceID,
		&databaseGroup.ResourceID,
		&databaseGroup.Placeholder,
		&stringExpr,
	); err != nil {
		return nil, errors.Wrapf(err, "failed to scan")
	}
	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}
	var expression expr.Expr
	if err := protojson.Unmarshal([]byte(stringExpr), &expression); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal expression")
	}
	databaseGroup.Expression = &expression
	return &databaseGroup, nil
}

// UpdateDatabaseGroup updates a database group.
func (s *Store) UpdateDatabaseGroup(ctx context.Context, updaterPrincipalID int, databaseGroupResourceID string, patch *UpdateDatabaseGroupMessage) (*DatabaseGroupMessage, error) {
	set, args := []string{"updater_id = $1"}, []any{fmt.Sprintf("%d", updaterPrincipalID)}
	if v := patch.Placeholder; v != nil {
		set, args = append(set, fmt.Sprintf("placeholder = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Expression; v != nil {
		jsonExpr, err := protojson.Marshal(patch.Expression)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to marshal expression")
		}
		set, args = append(set, fmt.Sprintf("placeholder = $%d", len(args)+1)), append(args, jsonExpr)
	}
	args = append(args, databaseGroupResourceID)
	query := fmt.Sprintf(`
		UPDATE db_group SET 
			%s 
		WHERE resource_id = $%d
		RETURNING id, created_ts, updated_ts, creator_id, updater_id, project_resource_id, resource_id, placeholder, expression;
	`, strings.Join(set, ", "), len(args))

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to commit ")
	}

	var updatedDatabaseGroup DatabaseGroupMessage
	var stringExpr string
	if err := tx.QueryRowContext(
		ctx,
		query,
		args...,
	).Scan(
		&updatedDatabaseGroup.ID,
		&updatedDatabaseGroup.CreatedTs,
		&updatedDatabaseGroup.UpdatedTs,
		&updatedDatabaseGroup.CreatorID,
		&updatedDatabaseGroup.UpdaterID,
		&updatedDatabaseGroup.ProjectResourceID,
		&updatedDatabaseGroup.ResourceID,
		&updatedDatabaseGroup.Placeholder,
		&stringExpr,
	); err != nil {
		return nil, errors.Wrapf(err, "failed to scan")
	}
	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}
	var expression expr.Expr
	if err := protojson.Unmarshal([]byte(stringExpr), &expression); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal expression")
	}
	updatedDatabaseGroup.Expression = &expression
	return &updatedDatabaseGroup, nil
}

// CreateDatabaseGroup creates a database group.
func (s *Store) CreateDatabaseGroup(ctx context.Context, creatorPrincipalID int, databaseGroup *DatabaseGroupMessage) (*DatabaseGroupMessage, error) {
	query := `
	INSERT INTO db_group (
		creator_id,
		updater_id,
		project_resource_id,
		resource_id,
		placeholder,
		expression
	) VALUES ($1, $2, $3, $4, $5, $6)
	RETURNING id, created_ts, updated_ts, creator_id, updater_id, project_resource_id, resource_id, placeholder, expression;
	`
	jsonExpr, err := protojson.Marshal(databaseGroup.Expression)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal expression")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	var insertedDatabaseGroup DatabaseGroupMessage
	var stringExpr string
	if err := tx.QueryRowContext(
		ctx,
		query,
		creatorPrincipalID,
		creatorPrincipalID,
		databaseGroup.ProjectResourceID,
		databaseGroup.ResourceID,
		databaseGroup.Placeholder,
		jsonExpr,
	).Scan(
		&insertedDatabaseGroup.ID,
		&insertedDatabaseGroup.CreatedTs,
		&insertedDatabaseGroup.UpdatedTs,
		&insertedDatabaseGroup.CreatorID,
		&insertedDatabaseGroup.UpdaterID,
		&insertedDatabaseGroup.ProjectResourceID,
		&insertedDatabaseGroup.ResourceID,
		&insertedDatabaseGroup.Placeholder,
		&stringExpr,
	); err != nil {
		return nil, errors.Wrapf(err, "failed to scan")
	}
	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}
	var expression expr.Expr
	if err := protojson.Unmarshal([]byte(stringExpr), &expression); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal expression")
	}
	insertedDatabaseGroup.Expression = &expression
	return &insertedDatabaseGroup, nil
}
