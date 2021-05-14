package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase"
	"github.com/bytebase/bytebase/api"
)

var (
	_ api.EnvironmentService = (*EnvironmentService)(nil)
)

// EnvironmentService represents a service for managing environment.
type EnvironmentService struct {
	l  *bytebase.Logger
	db *DB
}

// NewEnvironmentService returns a new instance of EnvironmentService.
func NewEnvironmentService(logger *bytebase.Logger, db *DB) *EnvironmentService {
	return &EnvironmentService{l: logger, db: db}
}

// CreateEnvironment creates a new environment.
func (s *EnvironmentService) CreateEnvironment(ctx context.Context, create *api.EnvironmentCreate) (*api.Environment, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	environment, err := s.createEnvironment(ctx, tx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return environment, nil
}

// FindEnvironmentList retrieves a list of environments based on find.
func (s *EnvironmentService) FindEnvironmentList(ctx context.Context, find *api.EnvironmentFind) ([]*api.Environment, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := s.findEnvironmentList(ctx, tx, find)
	if err != nil {
		return []*api.Environment{}, err
	}

	return list, nil
}

// FindEnvironment retrieves a single environment based on find.
// Returns ENOTFOUND if no matching record.
// Returns the first matching one and prints a warning if finding more than 1 matching records.
func (s *EnvironmentService) FindEnvironment(ctx context.Context, find *api.EnvironmentFind) (*api.Environment, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := s.findEnvironmentList(ctx, tx, find)
	if err != nil {
		return nil, err
	} else if len(list) == 0 {
		return nil, &bytebase.Error{Code: bytebase.ENOTFOUND, Message: fmt.Sprintf("environment not found: %v", find)}
	} else if len(list) > 1 {
		s.l.Warnf("found mulitple environments: %d, expect 1", len(list))
	}
	return list[0], nil
}

// PatchEnvironmentByID updates an existing environment by ID.
// Returns ENOTFOUND if environment does not exist.
func (s *EnvironmentService) PatchEnvironmentByID(ctx context.Context, patch *api.EnvironmentPatch) (*api.Environment, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	environment, err := s.patchEnvironment(ctx, tx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return environment, nil
}

// createEnvironment creates a new environment.
func (s *EnvironmentService) createEnvironment(ctx context.Context, tx *Tx, create *api.EnvironmentCreate) (*api.Environment, error) {
	// The order is the MAX(order) + 1
	row1, err1 := tx.QueryContext(ctx, `
		SELECT `+"`order`"+`
		FROM environment
		WHERE workspace_id = ?
		ORDER BY `+"`order`"+` DESC
		LIMIT 1
	`,
		create.WorkspaceId)

	if err1 != nil {
		return nil, FormatError(err1)
	}
	defer row1.Close()

	row1.Next()
	var order int
	if err1 := row1.Scan(
		&order,
	); err1 != nil {
		return nil, FormatError(err1)
	}

	// Insert row into database.
	row2, err2 := tx.QueryContext(ctx, `
		INSERT INTO environment (
			creator_id,
			updater_id,
			workspace_id,
			name,
			`+"`order`"+`
		)
		VALUES (?, ?, ?, ?, ?)
		RETURNING id, row_status, creator_id, created_ts, updater_id, updated_ts, workspace_id, name, `+"`order`"+`
	`,
		create.CreatorId,
		create.CreatorId,
		create.WorkspaceId,
		create.Name,
		order+1,
	)

	if err2 != nil {
		return nil, FormatError(err2)
	}
	defer row2.Close()

	row2.Next()
	var environment api.Environment
	if err := row2.Scan(
		&environment.ID,
		&environment.RowStatus,
		&environment.CreatorId,
		&environment.CreatedTs,
		&environment.UpdaterId,
		&environment.UpdatedTs,
		&environment.WorkspaceId,
		&environment.Name,
		&environment.Order,
	); err != nil {
		return nil, FormatError(err)
	}

	return &environment, nil
}

func (s *EnvironmentService) findEnvironmentList(ctx context.Context, tx *Tx, find *api.EnvironmentFind) (_ []*api.Environment, err error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, *v)
	}
	if v := find.WorkspaceId; v != nil {
		where, args = append(where, "workspace_id = ?"), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT 
		    id,
			row_status,
		    creator_id,
		    created_ts,
		    updater_id,
		    updated_ts,
			workspace_id,
		    name,
		    `+"`order`"+`
		FROM environment
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into list.
	list := make([]*api.Environment, 0)
	for rows.Next() {
		var environment api.Environment
		if err := rows.Scan(
			&environment.ID,
			&environment.RowStatus,
			&environment.CreatorId,
			&environment.CreatedTs,
			&environment.UpdaterId,
			&environment.UpdatedTs,
			&environment.WorkspaceId,
			&environment.Name,
			&environment.Order,
		); err != nil {
			return nil, FormatError(err)
		}

		list = append(list, &environment)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return list, nil
}

// patchEnvironment updates a environment by ID. Returns the new state of the environment after update.
func (s *EnvironmentService) patchEnvironment(ctx context.Context, tx *Tx, patch *api.EnvironmentPatch) (*api.Environment, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = ?"}, []interface{}{patch.UpdaterId}
	if v := patch.RowStatus; v != nil {
		set, args = append(set, "row_status = ?"), append(args, api.RowStatus(*v))
	}
	if v := patch.Name; v != nil {
		set, args = append(set, "name = ?"), append(args, *v)
	}
	if v := patch.Order; v != nil {
		set, args = append(set, "`order` = ?"), append(args, *v)
	}

	args = append(args, patch.ID)

	// Execute update query with RETURNING.
	row, err := tx.QueryContext(ctx, `
		UPDATE environment
		SET `+strings.Join(set, ", ")+`
		WHERE id = ?
		RETURNING id, row_status, creator_id, created_ts, updater_id, updated_ts, workspace_id, name, `+"`order`"+`
	`,
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	if row.Next() {
		var environment api.Environment
		if err := row.Scan(
			&environment.ID,
			&environment.RowStatus,
			&environment.CreatorId,
			&environment.CreatedTs,
			&environment.UpdaterId,
			&environment.UpdatedTs,
			&environment.WorkspaceId,
			&environment.Name,
			&environment.Order,
		); err != nil {
			return nil, FormatError(err)
		}
		return &environment, nil
	}

	return nil, &bytebase.Error{Code: bytebase.ENOTFOUND, Message: fmt.Sprintf("environment ID not found: %d", patch.ID)}
}
