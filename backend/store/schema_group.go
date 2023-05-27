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

// SchemaGroupMessage is the message for schema groups.
type SchemaGroupMessage struct {
	// Output only fields.
	//
	ID        int64
	CreatedTs int64
	UpdatedTs int64
	CreatorID int
	UpdaterID int

	// Normal fields.
	//
	DatabaseGroupResourceID string
	ResourceID              string
	Placeholder             string
	Expression              *expr.Expr
}

// FindSchemaGroupMessage is the message for finding schema group.
type FindSchemaGroupMessage struct {
	DatabaseGroupResourceID *string
	ResourceID              *string
}

// UpdateSchemaGroupMessage is the message for updating schema group.
type UpdateSchemaGroupMessage struct {
	Placeholder *string
	Expression  *expr.Expr
}

// DeleteSchemaGroup deletes a schema group.
func (s *Store) DeleteSchemaGroup(ctx context.Context, resourceID string) error {
	query := `
	DELETE FROM schema_group WHERE resource_id = $1;
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

// ListSchemaGroups lists schema groups.
func (s *Store) ListSchemaGroups(ctx context.Context, find *FindSchemaGroupMessage) ([]*SchemaGroupMessage, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	schemaGroups, err := s.listSchemaGroupsImpl(ctx, tx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list schema groups")
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}

	return schemaGroups, nil
}

// GetSchemaGroup gets a schema group.
func (s *Store) GetSchemaGroup(ctx context.Context, find *FindSchemaGroupMessage) (*SchemaGroupMessage, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	schemaGroups, err := s.listSchemaGroupsImpl(ctx, tx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list schema groups")
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}

	if len(schemaGroups) == 0 {
		return nil, nil
	}
	if len(schemaGroups) > 1 {
		return nil, errors.Errorf("found multiple schema groups")
	}
	return schemaGroups[0], nil
}

func (*Store) listSchemaGroupsImpl(ctx context.Context, tx *Tx, find *FindSchemaGroupMessage) ([]*SchemaGroupMessage, error) {
	where, args := []string{}, []any{}
	if v := find.DatabaseGroupResourceID; v != nil {
		where, args = append(where, fmt.Sprintf("db_group_resource_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.ResourceID; v != nil {
		where, args = append(where, fmt.Sprintf("resource_id = $%d", len(args)+1)), append(args, *v)
	}

	query := fmt.Sprintf(`SELECT
		id,
		created_ts,
		updated_ts,
		creator_id,
		updater_id,
		db_group_resource_id,
		resource_id,
		placeholder,
		expression
	FROM schema_group %s ORDER BY id DESC;`, strings.Join(where, " AND "))

	var schemaGroups []*SchemaGroupMessage

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to scan")
	}
	defer rows.Close()

	for rows.Next() {
		var schemaGroup SchemaGroupMessage
		var stringExpr string
		if err := rows.Scan(
			&schemaGroup.ID,
			&schemaGroup.CreatedTs,
			&schemaGroup.UpdatedTs,
			&schemaGroup.CreatorID,
			&schemaGroup.UpdaterID,
			&schemaGroup.DatabaseGroupResourceID,
			&schemaGroup.ResourceID,
			&schemaGroup.Placeholder,
			&stringExpr,
		); err != nil {
			return nil, errors.Wrapf(err, "failed to scan")
		}
		var expression expr.Expr
		if err := protojson.Unmarshal([]byte(stringExpr), &expression); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal expression")
		}
		schemaGroup.Expression = &expression
		schemaGroups = append(schemaGroups, &schemaGroup)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrapf(err, "failed to scan")
	}
	return schemaGroups, nil
}

// UpdateSchemaGroup updates a schema group.
func (s *Store) UpdateSchemaGroup(ctx context.Context, updaterPrincipalID int, schemaGroupResourceID string, patch *UpdateSchemaGroupMessage) (*SchemaGroupMessage, error) {
	set, args := []string{"updater_id = $1"}, []any{fmt.Sprintf("%d", updaterPrincipalID)}
	if v := patch.Placeholder; v != nil {
		set, args = append(set, fmt.Sprintf("placeholder = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Expression; v != nil {
		jsonExpr, err := protojson.Marshal(patch.Expression)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to marshal expression")
		}
		set, args = append(set, fmt.Sprintf("expression = $%d", len(args)+1)), append(args, jsonExpr)
	}
	args = append(args, schemaGroupResourceID)
	query := fmt.Sprintf(`
		UPDATE schema_group SET 
			%s 
		WHERE resource_id = $%d
		RETURNING id, created_ts, updated_ts, creator_id, updater_id, db_group_resource_id, resource_id, placeholder, expression;
	`, strings.Join(set, ", "), len(args))

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to commit ")
	}

	var updatedSchemaGroup SchemaGroupMessage
	var stringExpr string
	if err := tx.QueryRowContext(
		ctx,
		query,
		args...,
	).Scan(
		&updatedSchemaGroup.ID,
		&updatedSchemaGroup.CreatedTs,
		&updatedSchemaGroup.UpdatedTs,
		&updatedSchemaGroup.CreatorID,
		&updatedSchemaGroup.UpdaterID,
		&updatedSchemaGroup.DatabaseGroupResourceID,
		&updatedSchemaGroup.ResourceID,
		&updatedSchemaGroup.Placeholder,
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
	updatedSchemaGroup.Expression = &expression
	return &updatedSchemaGroup, nil
}

// CreateSchemaGroup creates a schema group.
func (s *Store) CreateSchemaGroup(ctx context.Context, creatorPrincipalID int, schemaGroup *SchemaGroupMessage) (*SchemaGroupMessage, error) {
	query := `
	INSERT INTO schema_group (
		creator_id,
		updater_id,
		db_resource_id,
		resource_id,
		placeholder,
		expression
	) VALUES ($1, $2, $3, $4, $5, $6)
	RETURNING id, created_ts, updated_ts, creator_id, updater_id, db_group_resource_id, resource_id, placeholder, expression;
	`
	jsonExpr, err := protojson.Marshal(schemaGroup.Expression)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal expression")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	var insertedSchemaGroup SchemaGroupMessage
	var stringExpr string
	if err := tx.QueryRowContext(
		ctx,
		query,
		creatorPrincipalID,
		creatorPrincipalID,
		schemaGroup.DatabaseGroupResourceID,
		schemaGroup.ResourceID,
		schemaGroup.Placeholder,
		jsonExpr,
	).Scan(
		&insertedSchemaGroup.ID,
		&insertedSchemaGroup.CreatedTs,
		&insertedSchemaGroup.UpdatedTs,
		&insertedSchemaGroup.CreatorID,
		&insertedSchemaGroup.UpdaterID,
		&insertedSchemaGroup.DatabaseGroupResourceID,
		&insertedSchemaGroup.ResourceID,
		&insertedSchemaGroup.Placeholder,
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
	insertedSchemaGroup.Expression = &expression
	return &insertedSchemaGroup, nil
}
