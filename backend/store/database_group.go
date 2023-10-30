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
	UID       int64
	CreatedTs int64
	UpdatedTs int64
	CreatorID int
	UpdaterID int

	// Normal fields.
	//
	ProjectUID  int
	ResourceID  string
	Placeholder string
	Expression  *expr.Expr
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
	s.databaseGroupCache.Delete(getDatabaseGroupCacheKey(projectUID, resourceID))
	s.databaseGroupIDCache.Delete(projectUID)
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
		s.databaseGroupCache.Store(getDatabaseGroupCacheKey(databaseGroup.ProjectUID, databaseGroup.ResourceID), databaseGroup)
		s.databaseGroupCache.Store(databaseGroup.UID, databaseGroup)
	}

	return databaseGroups, nil
}

// GetDatabaseGroup gets a database group.
func (s *Store) GetDatabaseGroup(ctx context.Context, find *FindDatabaseGroupMessage) (*DatabaseGroupMessage, error) {
	if find.ProjectUID != nil && find.ResourceID != nil && find.UID == nil {
		if databaseGroup, ok := s.databaseGroupCache.Load(getDatabaseGroupCacheKey(*find.ProjectUID, *find.ResourceID)); ok {
			if v, ok := databaseGroup.(*DatabaseGroupMessage); ok {
				return v, nil
			}
		}
	}
	if find.UID != nil && find.ProjectUID == nil && find.ResourceID == nil {
		if databaseGroup, ok := s.databaseGroupIDCache.Load(*find.UID); ok {
			if v, ok := databaseGroup.(*DatabaseGroupMessage); ok {
				return v, nil
			}
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
	s.databaseGroupCache.Store(getDatabaseGroupCacheKey(databaseGroups[0].ProjectUID, databaseGroups[0].ResourceID), databaseGroups[0])
	s.databaseGroupIDCache.Store(databaseGroups[0].UID, databaseGroups[0])
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

	query := fmt.Sprintf(`SELECT
		id,
		created_ts,
		updated_ts,
		creator_id,
		updater_id,
		project_id,
		resource_id,
		placeholder,
		expression
	FROM db_group WHERE %s ORDER BY id DESC;`, strings.Join(where, " AND "))

	var databaseGroups []*DatabaseGroupMessage
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to scan")
	}
	defer rows.Close()
	for rows.Next() {
		var databaseGroup DatabaseGroupMessage
		var stringExpr string
		if err := rows.Scan(
			&databaseGroup.UID,
			&databaseGroup.CreatedTs,
			&databaseGroup.UpdatedTs,
			&databaseGroup.CreatorID,
			&databaseGroup.UpdaterID,
			&databaseGroup.ProjectUID,
			&databaseGroup.ResourceID,
			&databaseGroup.Placeholder,
			&stringExpr,
		); err != nil {
			return nil, errors.Wrapf(err, "failed to scan")
		}
		var expression expr.Expr
		if err := protojson.Unmarshal([]byte(stringExpr), &expression); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal expression")
		}
		databaseGroup.Expression = &expression
		databaseGroups = append(databaseGroups, &databaseGroup)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrapf(err, "failed to scan")
	}
	return databaseGroups, nil
}

// UpdateDatabaseGroup updates a database group.
func (s *Store) UpdateDatabaseGroup(ctx context.Context, updaterPrincipalID int, databaseGroupUID int64, patch *UpdateDatabaseGroupMessage) (*DatabaseGroupMessage, error) {
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
	args = append(args, databaseGroupUID)
	query := fmt.Sprintf(`
		UPDATE db_group SET 
			%s 
		WHERE id = $%d
		RETURNING id, created_ts, updated_ts, creator_id, updater_id, project_id, resource_id, placeholder, expression;
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
		&updatedDatabaseGroup.UID,
		&updatedDatabaseGroup.CreatedTs,
		&updatedDatabaseGroup.UpdatedTs,
		&updatedDatabaseGroup.CreatorID,
		&updatedDatabaseGroup.UpdaterID,
		&updatedDatabaseGroup.ProjectUID,
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
	s.databaseGroupCache.Store(getDatabaseGroupCacheKey(updatedDatabaseGroup.ProjectUID, updatedDatabaseGroup.ResourceID), &updatedDatabaseGroup)
	s.databaseGroupIDCache.Store(updatedDatabaseGroup.UID, &updatedDatabaseGroup)
	return &updatedDatabaseGroup, nil
}

// CreateDatabaseGroup creates a database group.
func (s *Store) CreateDatabaseGroup(ctx context.Context, creatorPrincipalID int, databaseGroup *DatabaseGroupMessage) (*DatabaseGroupMessage, error) {
	query := `
	INSERT INTO db_group (
		creator_id,
		updater_id,
		project_id,
		resource_id,
		placeholder,
		expression
	) VALUES ($1, $2, $3, $4, $5, $6)
	RETURNING id, created_ts, updated_ts, creator_id, updater_id, project_id, resource_id, placeholder, expression;
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
		databaseGroup.ProjectUID,
		databaseGroup.ResourceID,
		databaseGroup.Placeholder,
		jsonExpr,
	).Scan(
		&insertedDatabaseGroup.UID,
		&insertedDatabaseGroup.CreatedTs,
		&insertedDatabaseGroup.UpdatedTs,
		&insertedDatabaseGroup.CreatorID,
		&insertedDatabaseGroup.UpdaterID,
		&insertedDatabaseGroup.ProjectUID,
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
	s.databaseGroupCache.Store(getDatabaseGroupCacheKey(insertedDatabaseGroup.ProjectUID, insertedDatabaseGroup.ResourceID), &insertedDatabaseGroup)
	s.databaseGroupCache.Store(insertedDatabaseGroup.UID, &insertedDatabaseGroup)
	return &insertedDatabaseGroup, nil
}

// SchemaGroupMessage is the message for schema groups.
type SchemaGroupMessage struct {
	// Output only fields.
	//
	UID       int64
	CreatedTs int64
	UpdatedTs int64
	CreatorID int
	UpdaterID int

	// Normal fields.
	//
	DatabaseGroupUID int64
	ResourceID       string
	Placeholder      string
	Expression       *expr.Expr
}

// FindSchemaGroupMessage is the message for finding schema group.
type FindSchemaGroupMessage struct {
	DatabaseGroupUID *int64
	ResourceID       *string
}

// UpdateSchemaGroupMessage is the message for updating schema group.
type UpdateSchemaGroupMessage struct {
	Placeholder *string
	Expression  *expr.Expr
}

// DeleteSchemaGroup deletes a schema group.
func (s *Store) DeleteSchemaGroup(ctx context.Context, schemaGroupUID int64) error {
	query := `
	DELETE
		FROM schema_group
	WHERE id = $1
	RETURNING db_group_id, resource_id;
	`
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to begin transaction")
	}
	defer tx.Rollback()
	var databaseGroupUID int64
	var resourceID string
	if err := tx.QueryRowContext(ctx, query, schemaGroupUID).Scan(
		&databaseGroupUID,
		&resourceID,
	); err != nil {
		return errors.Wrapf(err, "failed to scan")
	}
	if err := tx.Commit(); err != nil {
		return errors.Wrapf(err, "failed to commit transaction")
	}
	s.schemaGroupCache.Delete(getSchemaGroupCacheKey(databaseGroupUID, resourceID))
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
	for _, schemaGroup := range schemaGroups {
		s.schemaGroupCache.Store(getSchemaGroupCacheKey(schemaGroup.DatabaseGroupUID, schemaGroup.ResourceID), schemaGroup)
	}
	return schemaGroups, nil
}

// GetSchemaGroup gets a schema group.
func (s *Store) GetSchemaGroup(ctx context.Context, find *FindSchemaGroupMessage) (*SchemaGroupMessage, error) {
	if find.DatabaseGroupUID != nil && find.ResourceID != nil {
		if schemaGroup, ok := s.schemaGroupCache.Load(getSchemaGroupCacheKey(*find.DatabaseGroupUID, *find.ResourceID)); ok {
			if v, ok := schemaGroup.(*SchemaGroupMessage); ok {
				return v, nil
			}
		}
	}
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
	s.schemaGroupCache.Store(getSchemaGroupCacheKey(schemaGroups[0].DatabaseGroupUID, schemaGroups[0].ResourceID), schemaGroups[0])
	return schemaGroups[0], nil
}

func (*Store) listSchemaGroupsImpl(ctx context.Context, tx *Tx, find *FindSchemaGroupMessage) ([]*SchemaGroupMessage, error) {
	where, args := []string{"TRUE"}, []any{}
	if v := find.DatabaseGroupUID; v != nil {
		where, args = append(where, fmt.Sprintf("db_group_id = $%d", len(args)+1)), append(args, *v)
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
		db_group_id,
		resource_id,
		placeholder,
		expression
	FROM schema_group WHERE %s ORDER BY id DESC;`, strings.Join(where, " AND "))

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
			&schemaGroup.UID,
			&schemaGroup.CreatedTs,
			&schemaGroup.UpdatedTs,
			&schemaGroup.CreatorID,
			&schemaGroup.UpdaterID,
			&schemaGroup.DatabaseGroupUID,
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
func (s *Store) UpdateSchemaGroup(ctx context.Context, updaterPrincipalID int, schemaGroupUID int64, patch *UpdateSchemaGroupMessage) (*SchemaGroupMessage, error) {
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
	args = append(args, schemaGroupUID)
	query := fmt.Sprintf(`
		UPDATE schema_group SET 
			%s 
		WHERE id = $%d
		RETURNING id, created_ts, updated_ts, creator_id, updater_id, db_group_id, resource_id, placeholder, expression;
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
		&updatedSchemaGroup.UID,
		&updatedSchemaGroup.CreatedTs,
		&updatedSchemaGroup.UpdatedTs,
		&updatedSchemaGroup.CreatorID,
		&updatedSchemaGroup.UpdaterID,
		&updatedSchemaGroup.DatabaseGroupUID,
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
	s.schemaGroupCache.Store(getSchemaGroupCacheKey(updatedSchemaGroup.DatabaseGroupUID, updatedSchemaGroup.ResourceID), &updatedSchemaGroup)
	return &updatedSchemaGroup, nil
}

// CreateSchemaGroup creates a schema group.
func (s *Store) CreateSchemaGroup(ctx context.Context, creatorPrincipalID int, schemaGroup *SchemaGroupMessage) (*SchemaGroupMessage, error) {
	query := `
	INSERT INTO schema_group (
		creator_id,
		updater_id,
		db_group_id,
		resource_id,
		placeholder,
		expression
	) VALUES ($1, $2, $3, $4, $5, $6)
	RETURNING id, created_ts, updated_ts, creator_id, updater_id, db_group_id, resource_id, placeholder, expression;
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
		schemaGroup.DatabaseGroupUID,
		schemaGroup.ResourceID,
		schemaGroup.Placeholder,
		jsonExpr,
	).Scan(
		&insertedSchemaGroup.UID,
		&insertedSchemaGroup.CreatedTs,
		&insertedSchemaGroup.UpdatedTs,
		&insertedSchemaGroup.CreatorID,
		&insertedSchemaGroup.UpdaterID,
		&insertedSchemaGroup.DatabaseGroupUID,
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
	s.schemaGroupCache.Store(getSchemaGroupCacheKey(insertedSchemaGroup.DatabaseGroupUID, insertedSchemaGroup.ResourceID), &insertedSchemaGroup)
	return &insertedSchemaGroup, nil
}
