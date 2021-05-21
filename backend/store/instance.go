package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase"
	"github.com/bytebase/bytebase/api"
)

var (
	_ api.InstanceService = (*InstanceService)(nil)
)

// InstanceService represents a service for managing instance.
type InstanceService struct {
	l  *bytebase.Logger
	db *DB
}

// NewInstanceService returns a new instance of InstanceService.
func NewInstanceService(logger *bytebase.Logger, db *DB) *InstanceService {
	return &InstanceService{l: logger, db: db}
}

// CreateInstance creates a new instance.
func (s *InstanceService) CreateInstance(ctx context.Context, create *api.InstanceCreate) (*api.Instance, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	instance, err := createInstance(ctx, tx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return instance, nil
}

// FindInstanceList retrieves a list of instances based on find.
func (s *InstanceService) FindInstanceList(ctx context.Context, find *api.InstanceFind) ([]*api.Instance, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := findInstanceList(ctx, tx, find)
	if err != nil {
		return []*api.Instance{}, err
	}

	return list, nil
}

// FindInstance retrieves a single instance based on find.
// Returns ENOTFOUND if no matching record.
// Returns the first matching one and prints a warning if finding more than 1 matching records.
func (s *InstanceService) FindInstance(ctx context.Context, find *api.InstanceFind) (*api.Instance, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := findInstanceList(ctx, tx, find)
	if err != nil {
		return nil, err
	} else if len(list) == 0 {
		return nil, &bytebase.Error{Code: bytebase.ENOTFOUND, Message: fmt.Sprintf("instance not found: %v", find)}
	} else if len(list) > 1 {
		s.l.Warnf("found mulitple instances: %d, expect 1", len(list))
	}
	return list[0], nil
}

// PatchInstance updates an existing instance by ID.
// Returns ENOTFOUND if instance does not exist.
func (s *InstanceService) PatchInstance(ctx context.Context, patch *api.InstancePatch) (*api.Instance, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	instance, err := patchInstance(ctx, tx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return instance, nil
}

// createInstance creates a new instance.
func createInstance(ctx context.Context, tx *Tx, create *api.InstanceCreate) (*api.Instance, error) {
	// Insert row into database.
	row, err := tx.QueryContext(ctx, `
		INSERT INTO instance (
			creator_id,
			updater_id,
			workspace_id,
			environment_id,
			name,
			external_link,
			host,
			port
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id, row_status, creator_id, created_ts, updater_id, updated_ts, workspace_id, environment_id, name, external_link, host, port
	`,
		create.CreatorId,
		create.CreatorId,
		create.WorkspaceId,
		create.EnvironmentId,
		create.Name,
		create.ExternalLink,
		create.Host,
		create.Port,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var instance api.Instance
	if err := row.Scan(
		&instance.ID,
		&instance.RowStatus,
		&instance.CreatorId,
		&instance.CreatedTs,
		&instance.UpdaterId,
		&instance.UpdatedTs,
		&instance.WorkspaceId,
		&instance.EnvironmentId,
		&instance.Name,
		&instance.ExternalLink,
		&instance.Host,
		&instance.Port,
	); err != nil {
		return nil, FormatError(err)
	}

	return &instance, nil
}

func findInstanceList(ctx context.Context, tx *Tx, find *api.InstanceFind) (_ []*api.Instance, err error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, *v)
	}
	if v := find.RowStatus; v != nil {
		where, args = append(where, "row_status = ?"), append(args, *v)
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
			environment_id,
			name,
			external_link,
			host,
			port
		FROM instance
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into list.
	list := make([]*api.Instance, 0)
	for rows.Next() {
		var instance api.Instance
		if err := rows.Scan(
			&instance.ID,
			&instance.RowStatus,
			&instance.CreatorId,
			&instance.CreatedTs,
			&instance.UpdaterId,
			&instance.UpdatedTs,
			&instance.WorkspaceId,
			&instance.EnvironmentId,
			&instance.Name,
			&instance.ExternalLink,
			&instance.Host,
			&instance.Port,
		); err != nil {
			return nil, FormatError(err)
		}

		list = append(list, &instance)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return list, nil
}

// patchInstance updates a instance by ID. Returns the new state of the instance after update.
func patchInstance(ctx context.Context, tx *Tx, patch *api.InstancePatch) (*api.Instance, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = ?"}, []interface{}{patch.UpdaterId}
	if v := patch.RowStatus; v != nil {
		set, args = append(set, "row_status = ?"), append(args, api.RowStatus(*v))
	}
	if v := patch.Name; v != nil {
		set, args = append(set, "name = ?"), append(args, *v)
	}
	if v := patch.ExternalLink; v != nil {
		set, args = append(set, "external_link = ?"), append(args, *v)
	}
	if v := patch.Host; v != nil {
		set, args = append(set, "host = ?"), append(args, *v)
	}
	if v := patch.Port; v != nil {
		set, args = append(set, "port = ?"), append(args, *v)
	}

	args = append(args, patch.ID)

	// Execute update query with RETURNING.
	row, err := tx.QueryContext(ctx, `
		UPDATE instance
		SET `+strings.Join(set, ", ")+`
		WHERE id = ?
		RETURNING id, row_status, creator_id, created_ts, updater_id, updated_ts, workspace_id, environment_id, name, external_link, host, port
	`,
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	if row.Next() {
		var instance api.Instance
		if err := row.Scan(
			&instance.ID,
			&instance.RowStatus,
			&instance.CreatorId,
			&instance.CreatedTs,
			&instance.UpdaterId,
			&instance.UpdatedTs,
			&instance.WorkspaceId,
			&instance.EnvironmentId,
			&instance.Name,
			&instance.ExternalLink,
			&instance.Host,
			&instance.Port,
		); err != nil {
			return nil, FormatError(err)
		}

		return &instance, nil
	}

	return nil, &bytebase.Error{Code: bytebase.ENOTFOUND, Message: fmt.Sprintf("instance ID not found: %d", patch.ID)}
}
