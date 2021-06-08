package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase"
	"github.com/bytebase/bytebase/api"
	uuid "github.com/satori/go.uuid"
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
// Returns ENOTFOUND if no matching record.
// Returns the first matching one and prints a warning if finding more than 1 matching records.
func (s *VCSService) FindVCS(ctx context.Context, find *api.VCSFind) (*api.VCS, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := findVCSList(ctx, tx, find)
	if err != nil {
		return nil, err
	} else if len(list) == 0 {
		return nil, &bytebase.Error{Code: bytebase.ENOTFOUND, Message: fmt.Sprintf("vcs not found: %v", find)}
	} else if len(list) > 1 {
		s.l.Warn(fmt.Sprintf("found mulitple vcss: %d, expect 1", len(list)))
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
// Returns ENOTFOUND if vcs does not exist.
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
			uuid,
			`+"`type`,"+`
			instance_url,
			api_url,
			application_id,
			secret
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, name, uuid, `+"`type`, instance_url, api_url, application_id, secret, access_token"+`
	`,
		create.CreatorId,
		create.CreatorId,
		create.Name,
		uuid.NewV4(),
		create.Type,
		create.InstanceURL,
		create.ApiURL,
		create.ApplicationId,
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
		&vcs.CreatorId,
		&vcs.CreatedTs,
		&vcs.UpdaterId,
		&vcs.UpdatedTs,
		&vcs.Name,
		&vcs.Uuid,
		&vcs.Type,
		&vcs.InstanceURL,
		&vcs.ApiURL,
		&vcs.ApplicationId,
		&vcs.Secret,
		&vcs.AccessToken,
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
	if v := find.Uuid; v != nil {
		where, args = append(where, "uuid = ?"), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT 
		    id,
		    creator_id,
		    created_ts,
		    updater_id,
		    updated_ts,
			name,
			uuid,
			`+"`type`,"+`
			instance_url,
			api_url,
			application_id,
			secret,
			access_token
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
			&vcs.CreatorId,
			&vcs.CreatedTs,
			&vcs.UpdaterId,
			&vcs.UpdatedTs,
			&vcs.Name,
			&vcs.Uuid,
			&vcs.Type,
			&vcs.InstanceURL,
			&vcs.ApiURL,
			&vcs.ApplicationId,
			&vcs.Secret,
			&vcs.AccessToken,
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
	set, args := []string{"updater_id = ?"}, []interface{}{patch.UpdaterId}
	if v := patch.AccessToken; v != nil {
		set, args = append(set, "access_token = ?"), append(args, *v)
	}
	if v := patch.ExpireTs; v != nil {
		set, args = append(set, "access_token_expiration_ts = ?"), append(args, *v)
	}
	if v := patch.RefreshToken; v != nil {
		set, args = append(set, "refresh_token = ?"), append(args, *v)
	}

	args = append(args, patch.ID)

	// Execute update query with RETURNING.
	row, err := tx.QueryContext(ctx, `
		UPDATE vcs
		SET `+strings.Join(set, ", ")+`
		WHERE id = ?
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, name, uuid, `+"`type`, instance_url, api_url, application_id, secret, access_token"+`
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
			&vcs.CreatorId,
			&vcs.CreatedTs,
			&vcs.UpdaterId,
			&vcs.UpdatedTs,
			&vcs.Name,
			&vcs.Uuid,
			&vcs.Type,
			&vcs.InstanceURL,
			&vcs.ApiURL,
			&vcs.ApplicationId,
			&vcs.Secret,
			&vcs.AccessToken,
		); err != nil {
			return nil, FormatError(err)
		}

		return &vcs, nil
	}

	return nil, &bytebase.Error{Code: bytebase.ENOTFOUND, Message: fmt.Sprintf("vcs ID not found: %d", patch.ID)}
}

// deleteVCS permanently deletes a vcs by ID.
func deleteVCS(ctx context.Context, tx *Tx, delete *api.VCSDelete) error {
	// Remove row from database.
	result, err := tx.ExecContext(ctx, `DELETE FROM vcs WHERE id = ?`, delete.ID)
	if err != nil {
		return FormatError(err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return &bytebase.Error{Code: bytebase.ENOTFOUND, Message: fmt.Sprintf("vcs ID not found: %d", delete.ID)}
	}

	return nil
}
