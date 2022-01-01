package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"go.uber.org/zap"
)

var (
	_ api.VCSService = (*VCSService)(nil)
)

// VCSService represents a service for managing vcs.
type VCSService struct {
	l  *zap.Logger
	db *DB
}

// NewVCSService returns a new instance of VCSService.
func NewVCSService(logger *zap.Logger, db *DB) *VCSService {
	return &VCSService{l: logger, db: db}
}

// CreateVCS creates a new vcs.
func (s *VCSService) CreateVCS(ctx context.Context, create *api.VCSCreate) (*api.VCS, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	vcs, err := createVCS(ctx, tx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return vcs, nil
}

// FindVCSList retrieves a list of vcss based on find.
func (s *VCSService) FindVCSList(ctx context.Context, find *api.VCSFind) ([]*api.VCS, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := findVCSList(ctx, tx, find)
	if err != nil {
		return []*api.VCS{}, err
	}

	return list, nil
}

// FindVCS retrieves a single vcs based on find.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *VCSService) FindVCS(ctx context.Context, find *api.VCSFind) (*api.VCS, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := findVCSList(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, nil
	} else if len(list) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: fmt.Errorf("found %d vcss with filter %+v, expect 1", len(list), find)}
	}
	return list[0], nil
}

// PatchVCS updates an existing vcs by ID.
// Returns ENOTFOUND if vcs does not exist.
func (s *VCSService) PatchVCS(ctx context.Context, patch *api.VCSPatch) (*api.VCS, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	vcs, err := patchVCS(ctx, tx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return vcs, nil
}

// DeleteVCS deletes an existing vcs by ID.
func (s *VCSService) DeleteVCS(ctx context.Context, delete *api.VCSDelete) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FormatError(err)
	}
	defer tx.Rollback()

	err = deleteVCS(ctx, tx, delete)
	if err != nil {
		return FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return FormatError(err)
	}

	return nil
}

// createVCS creates a new vcs.
func createVCS(ctx context.Context, tx *Tx, create *api.VCSCreate) (*api.VCS, error) {
	// Insert row into database.
	row, err := tx.QueryContext(ctx, `
		INSERT INTO vcs (
			creator_id,
			updater_id,
			name,
			`+"`type`,"+`
			instance_url,
			api_url,
			application_id,
			secret
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, name, `+"`type`, instance_url, api_url, application_id, secret"+`
	`,
		create.CreatorID,
		create.CreatorID,
		create.Name,
		create.Type,
		create.InstanceURL,
		create.APIURL,
		create.ApplicationID,
		create.Secret,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var vcs api.VCS
	if err := row.Scan(
		&vcs.ID,
		&vcs.CreatorID,
		&vcs.CreatedTs,
		&vcs.UpdaterID,
		&vcs.UpdatedTs,
		&vcs.Name,
		&vcs.Type,
		&vcs.InstanceURL,
		&vcs.APIURL,
		&vcs.ApplicationID,
		&vcs.Secret,
	); err != nil {
		return nil, FormatError(err)
	}

	return &vcs, nil
}

func findVCSList(ctx context.Context, tx *Tx, find *api.VCSFind) (_ []*api.VCS, err error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT
		    id,
		    creator_id,
		    created_ts,
		    updater_id,
		    updated_ts,
			name,
			`+"`type`,"+`
			instance_url,
			api_url,
			application_id,
			secret
		FROM vcs
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into list.
	list := make([]*api.VCS, 0)
	for rows.Next() {
		var vcs api.VCS
		if err := rows.Scan(
			&vcs.ID,
			&vcs.CreatorID,
			&vcs.CreatedTs,
			&vcs.UpdaterID,
			&vcs.UpdatedTs,
			&vcs.Name,
			&vcs.Type,
			&vcs.InstanceURL,
			&vcs.APIURL,
			&vcs.ApplicationID,
			&vcs.Secret,
		); err != nil {
			return nil, FormatError(err)
		}

		list = append(list, &vcs)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return list, nil
}

// patchVCS updates a vcs by ID. Returns the new state of the vcs after update.
func patchVCS(ctx context.Context, tx *Tx, patch *api.VCSPatch) (*api.VCS, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = ?"}, []interface{}{patch.UpdaterID}
	if v := patch.Name; v != nil {
		set, args = append(set, "name = ?"), append(args, *v)
	}
	if v := patch.ApplicationID; v != nil {
		set, args = append(set, "application_id = ?"), append(args, *v)
	}
	if v := patch.Secret; v != nil {
		set, args = append(set, "secret = ?"), append(args, *v)
	}
	args = append(args, patch.ID)

	// Execute update query with RETURNING.
	row, err := tx.QueryContext(ctx, `
		UPDATE vcs
		SET `+strings.Join(set, ", ")+`
		WHERE id = ?
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, name, `+"`type`, instance_url, api_url, application_id, secret"+`
	`,
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	if row.Next() {
		var vcs api.VCS
		if err := row.Scan(
			&vcs.ID,
			&vcs.CreatorID,
			&vcs.CreatedTs,
			&vcs.UpdaterID,
			&vcs.UpdatedTs,
			&vcs.Name,
			&vcs.Type,
			&vcs.InstanceURL,
			&vcs.APIURL,
			&vcs.ApplicationID,
			&vcs.Secret,
		); err != nil {
			return nil, FormatError(err)
		}

		return &vcs, nil
	}

	return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("vcs ID not found: %d", patch.ID)}
}

// deleteVCS permanently deletes a vcs by ID.
func deleteVCS(ctx context.Context, tx *Tx, delete *api.VCSDelete) error {
	// Remove row from database.
	if _, err := tx.ExecContext(ctx, `DELETE FROM vcs WHERE id = ?`, delete.ID); err != nil {
		return FormatError(err)
	}
	return nil
}
