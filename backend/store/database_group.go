package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/genproto/googleapis/type/expr"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// DatabaseGroupMessage is the message for database groups.
type DatabaseGroupMessage struct {
	// Output only fields.
	UID       int64
	CreatedTs int64
	UpdatedTs int64
	CreatorID int
	UpdaterID int

	// Normal fields.
	ProjectUID  int
	ResourceID  string
	Placeholder string
	Expression  *expr.Expr
	Payload     *storepb.DatabaseGroupPayload
}

// FindDatabaseGroupMessage is the message for finding database group.
type FindDatabaseGroupMessage struct {
	ProjectUID *int
	ResourceID *string
	UID        *int64
}

// UpdateDatabaseGroupMessage is the message for updating database group.
type UpdateDatabaseGroupMessage struct {
	Placeholder *string
	Expression  *expr.Expr
	Payload     *storepb.DatabaseGroupPayload
}

// DeleteDatabaseGroup deletes a database group.
func (s *Store) DeleteDatabaseGroup(ctx context.Context, databaseGroupUID int64) error {
	query := `
	DELETE
		FROM db_group
	WHERE id = $1
	RETURNING id, project_id, resource_id;
	`
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to begin transaction")
	}
	defer tx.Rollback()
	var id int64
	var projectUID int
	var resourceID string
	if err := tx.QueryRowContext(ctx, query, databaseGroupUID).Scan(
		&id,
		&projectUID,
		&resourceID,
	); err != nil {
		return errors.Wrapf(err, "failed to scan")
	}
	if err := tx.Commit(); err != nil {
		return errors.Wrapf(err, "failed to commit transaction")
	}
	s.databaseGroupCache.Remove(getDatabaseGroupCacheKey(projectUID, resourceID))
	s.databaseGroupIDCache.Remove(databaseGroupUID)
	return nil
}

// ListDatabaseGroups lists database groups.
func (s *Store) ListDatabaseGroups(ctx context.Context, find *FindDatabaseGroupMessage) ([]*DatabaseGroupMessage, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
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
		s.databaseGroupCache.Add(getDatabaseGroupCacheKey(databaseGroup.ProjectUID, databaseGroup.ResourceID), databaseGroup)
		s.databaseGroupIDCache.Add(databaseGroup.UID, databaseGroup)
	}

	return databaseGroups, nil
}

// GetDatabaseGroup gets a database group.
func (s *Store) GetDatabaseGroup(ctx context.Context, find *FindDatabaseGroupMessage) (*DatabaseGroupMessage, error) {
	if find.ProjectUID != nil && find.ResourceID != nil && find.UID == nil {
		if v, ok := s.databaseGroupCache.Get(getDatabaseGroupCacheKey(*find.ProjectUID, *find.ResourceID)); ok {
			return v, nil
		}
	}
	if find.UID != nil && find.ProjectUID == nil && find.ResourceID == nil {
		if v, ok := s.databaseGroupIDCache.Get(*find.UID); ok {
			return v, nil
		}
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
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
	s.databaseGroupCache.Add(getDatabaseGroupCacheKey(databaseGroups[0].ProjectUID, databaseGroups[0].ResourceID), databaseGroups[0])
	s.databaseGroupIDCache.Add(databaseGroups[0].UID, databaseGroups[0])
	return databaseGroups[0], nil
}

func (*Store) listDatabaseGroupImpl(ctx context.Context, tx *Tx, find *FindDatabaseGroupMessage) ([]*DatabaseGroupMessage, error) {
	where, args := []string{"TRUE"}, []any{}
	if v := find.ProjectUID; v != nil {
		where, args = append(where, fmt.Sprintf("project_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.ResourceID; v != nil {
		where, args = append(where, fmt.Sprintf("resource_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.UID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}
	fields := []string{
		"id",
		"created_ts",
		"updated_ts",
		"creator_id",
		"updater_id",
		"project_id",
		"resource_id",
		"placeholder",
		"expression",
		"payload",
	}
	query := fmt.Sprintf(`SELECT %s FROM db_group WHERE %s ORDER BY id DESC;`, strings.Join(fields, ","), strings.Join(where, " AND "))
	var databaseGroups []*DatabaseGroupMessage
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to scan")
	}
	defer rows.Close()
	for rows.Next() {
		var databaseGroup DatabaseGroupMessage
		var exprBytes, payloadBytes []byte
		dest := []any{
			&databaseGroup.UID,
			&databaseGroup.CreatedTs,
			&databaseGroup.UpdatedTs,
			&databaseGroup.CreatorID,
			&databaseGroup.UpdaterID,
			&databaseGroup.ProjectUID,
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
func (s *Store) UpdateDatabaseGroup(ctx context.Context, updaterPrincipalID int, databaseGroupUID int64, patch *UpdateDatabaseGroupMessage) (*DatabaseGroupMessage, error) {
	set, args := []string{"updater_id = $1", "updated_ts = $2"}, []any{updaterPrincipalID, time.Now().Unix()}
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
	args = append(args, databaseGroupUID)
	query := fmt.Sprintf(`
		UPDATE db_group SET 
			%s 
		WHERE id = $%d
		RETURNING id, created_ts, updated_ts, creator_id, updater_id, project_id, resource_id, placeholder, expression, payload;
	`, strings.Join(set, ", "), len(args))

	tx, err := s.db.BeginTx(ctx, nil)
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
		&updatedDatabaseGroup.UID,
		&updatedDatabaseGroup.CreatedTs,
		&updatedDatabaseGroup.UpdatedTs,
		&updatedDatabaseGroup.CreatorID,
		&updatedDatabaseGroup.UpdaterID,
		&updatedDatabaseGroup.ProjectUID,
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
	s.databaseGroupCache.Add(getDatabaseGroupCacheKey(updatedDatabaseGroup.ProjectUID, updatedDatabaseGroup.ResourceID), &updatedDatabaseGroup)
	s.databaseGroupIDCache.Add(updatedDatabaseGroup.UID, &updatedDatabaseGroup)
	return &updatedDatabaseGroup, nil
}

// CreateDatabaseGroup creates a database group.
func (s *Store) CreateDatabaseGroup(ctx context.Context, creatorPrincipalID int, create *DatabaseGroupMessage) (*DatabaseGroupMessage, error) {
	query := `
	INSERT INTO db_group (
		creator_id,
		updater_id,
		project_id,
		resource_id,
		placeholder,
		expression,
		payload
	) VALUES ($1, $2, $3, $4, $5, $6, $7)
	RETURNING id, created_ts, updated_ts;
	`
	exprBytes, err := protojson.Marshal(create.Expression)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal expression")
	}
	payloadBytes, err := protojson.Marshal(create.Payload)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal payload")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	if err := tx.QueryRowContext(
		ctx,
		query,
		creatorPrincipalID,
		creatorPrincipalID,
		create.ProjectUID,
		create.ResourceID,
		create.Placeholder,
		exprBytes,
		payloadBytes,
	).Scan(
		&create.UID,
		&create.CreatedTs,
		&create.UpdatedTs,
	); err != nil {
		return nil, errors.Wrapf(err, "failed to scan")
	}
	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}

	create.CreatorID = creatorPrincipalID
	create.UpdaterID = creatorPrincipalID
	s.databaseGroupCache.Add(getDatabaseGroupCacheKey(create.ProjectUID, create.ResourceID), create)
	s.databaseGroupIDCache.Add(create.UID, create)
	return create, nil
}
