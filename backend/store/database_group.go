package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/genproto/googleapis/type/expr"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// DatabaseGroupMessage is the message for database groups.
type DatabaseGroupMessage struct {
	ProjectID   string
	ResourceID  string
	Placeholder string
	Expression  *expr.Expr
	Payload     *storepb.DatabaseGroupPayload
}

// FindDatabaseGroupMessage is the message for finding database group.
type FindDatabaseGroupMessage struct {
	ProjectID  *string
	ResourceID *string
}

// UpdateDatabaseGroupMessage is the message for updating database group.
type UpdateDatabaseGroupMessage struct {
	Placeholder *string
	Expression  *expr.Expr
	Payload     *storepb.DatabaseGroupPayload
}

// DeleteDatabaseGroup deletes a database group.
func (s *Store) DeleteDatabaseGroup(ctx context.Context, projectID, resourceID string) error {
	query := `
	DELETE
		FROM db_group
	WHERE project = $1 AND resource_id = $2
	`
	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to begin transaction")
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, query, projectID, resourceID); err != nil {
		return errors.Wrapf(err, "failed to execute")
	}
	if err := tx.Commit(); err != nil {
		return errors.Wrapf(err, "failed to commit transaction")
	}
	s.databaseGroupCache.Remove(getDatabaseGroupCacheKey(projectID, resourceID))
	return nil
}

// ListDatabaseGroups lists database groups.
func (s *Store) ListDatabaseGroups(ctx context.Context, find *FindDatabaseGroupMessage) ([]*DatabaseGroupMessage, error) {
	tx, err := s.GetDB().BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	databaseGroups, err := s.listDatabaseGroupImpl(ctx, tx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list database groups")
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}
	for _, databaseGroup := range databaseGroups {
		s.databaseGroupCache.Add(getDatabaseGroupCacheKey(databaseGroup.ProjectID, databaseGroup.ResourceID), databaseGroup)
	}

	return databaseGroups, nil
}

// GetDatabaseGroup gets a database group.
func (s *Store) GetDatabaseGroup(ctx context.Context, find *FindDatabaseGroupMessage) (*DatabaseGroupMessage, error) {
	if find.ProjectID != nil && find.ResourceID != nil {
		if v, ok := s.databaseGroupCache.Get(getDatabaseGroupCacheKey(*find.ProjectID, *find.ResourceID)); ok && s.enableCache {
			return v, nil
		}
	}
	tx, err := s.GetDB().BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	databaseGroups, err := s.listDatabaseGroupImpl(ctx, tx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list database groups")
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
	s.databaseGroupCache.Add(getDatabaseGroupCacheKey(databaseGroups[0].ProjectID, databaseGroups[0].ResourceID), databaseGroups[0])
	return databaseGroups[0], nil
}

func (*Store) listDatabaseGroupImpl(ctx context.Context, txn *sql.Tx, find *FindDatabaseGroupMessage) ([]*DatabaseGroupMessage, error) {
	where, args := []string{"TRUE"}, []any{}
	if v := find.ProjectID; v != nil {
		where, args = append(where, fmt.Sprintf("project = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.ResourceID; v != nil {
		where, args = append(where, fmt.Sprintf("resource_id = $%d", len(args)+1)), append(args, *v)
	}
	fields := []string{
		"project",
		"resource_id",
		"placeholder",
		"expression",
		"payload",
	}
	query := fmt.Sprintf(`SELECT %s FROM db_group WHERE %s ORDER BY project, resource_id ASC;`, strings.Join(fields, ","), strings.Join(where, " AND "))
	var databaseGroups []*DatabaseGroupMessage
	rows, err := txn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to scan")
	}
	defer rows.Close()
	for rows.Next() {
		var databaseGroup DatabaseGroupMessage
		var exprBytes, payloadBytes []byte
		dest := []any{
			&databaseGroup.ProjectID,
			&databaseGroup.ResourceID,
			&databaseGroup.Placeholder,
			&exprBytes,
			&payloadBytes,
		}
		if err := rows.Scan(dest...); err != nil {
			return nil, errors.Wrapf(err, "failed to scan")
		}
		var expression expr.Expr
		if err := common.ProtojsonUnmarshaler.Unmarshal(exprBytes, &expression); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal expression")
		}
		var payload storepb.DatabaseGroupPayload
		if err := common.ProtojsonUnmarshaler.Unmarshal(payloadBytes, &payload); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal payload")
		}
		databaseGroup.Expression = &expression
		databaseGroup.Payload = &payload
		databaseGroups = append(databaseGroups, &databaseGroup)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrapf(err, "failed to scan")
	}
	return databaseGroups, nil
}

// UpdateDatabaseGroup updates a database group.
func (s *Store) UpdateDatabaseGroup(ctx context.Context, projectID, resourceID string, patch *UpdateDatabaseGroupMessage) (*DatabaseGroupMessage, error) {
	set, args := []string{}, []any{}
	if v := patch.Placeholder; v != nil {
		set, args = append(set, fmt.Sprintf("placeholder = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Expression; v != nil {
		exprBytes, err := protojson.Marshal(patch.Expression)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to marshal expression")
		}
		set, args = append(set, fmt.Sprintf("expression = $%d", len(args)+1)), append(args, exprBytes)
	}
	if v := patch.Payload; v != nil {
		payloadBytes, err := protojson.Marshal(patch.Payload)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to marshal payload")
		}
		set, args = append(set, fmt.Sprintf("payload = $%d", len(args)+1)), append(args, payloadBytes)
	}
	args = append(args, projectID, resourceID)
	query := fmt.Sprintf(`
		UPDATE db_group SET 
			%s
		WHERE project = $%d AND resource_id = $%d
		RETURNING project, resource_id, placeholder, expression, payload;
	`, strings.Join(set, ", "), len(args)-1, len(args))

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to commit ")
	}

	var updatedDatabaseGroup DatabaseGroupMessage
	var exprBytes, payloadBytes []byte
	if err := tx.QueryRowContext(
		ctx,
		query,
		args...,
	).Scan(
		&updatedDatabaseGroup.ProjectID,
		&updatedDatabaseGroup.ResourceID,
		&updatedDatabaseGroup.Placeholder,
		&exprBytes,
		&payloadBytes,
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
	var payload storepb.DatabaseGroupPayload
	if err := common.ProtojsonUnmarshaler.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal payload")
	}
	updatedDatabaseGroup.Payload = &payload
	s.databaseGroupCache.Add(getDatabaseGroupCacheKey(updatedDatabaseGroup.ProjectID, updatedDatabaseGroup.ResourceID), &updatedDatabaseGroup)
	return &updatedDatabaseGroup, nil
}

// CreateDatabaseGroup creates a database group.
func (s *Store) CreateDatabaseGroup(ctx context.Context, create *DatabaseGroupMessage) (*DatabaseGroupMessage, error) {
	query := `
	INSERT INTO db_group (
		project,
		resource_id,
		placeholder,
		expression,
		payload
	) VALUES ($1, $2, $3, $4, $5);
	`
	exprBytes, err := protojson.Marshal(create.Expression)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal expression")
	}
	payloadBytes, err := protojson.Marshal(create.Payload)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal payload")
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(
		ctx,
		query,
		create.ProjectID,
		create.ResourceID,
		create.Placeholder,
		exprBytes,
		payloadBytes,
	); err != nil {
		return nil, errors.Wrapf(err, "failed to scan")
	}
	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}

	s.databaseGroupCache.Add(getDatabaseGroupCacheKey(create.ProjectID, create.ResourceID), create)
	return create, nil
}
