package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/vcs"
)

// VCSRaw is the store model for a VCS (Version Control System).
// Fields have exactly the same meanings as VCS.
type VCSRaw struct {
	ID int

	// Standard fields
	CreatorID int
	CreatedTs int64
	UpdaterID int
	UpdatedTs int64

	// Domain specific fields
	Name          string
	Type          vcs.Type
	InstanceURL   string
	APIURL        string
	ApplicationID string
	Secret        string
}

// ToVCS creates an instance of VCS based on the VCSRaw.
// This is intended to be called when we need to compose a VCS relationship.
func (raw *VCSRaw) ToVCS() *api.VCS {
	return &api.VCS{
		ID: raw.ID,

		CreatorID: raw.CreatorID,
		CreatedTs: raw.CreatedTs,
		UpdaterID: raw.UpdaterID,
		UpdatedTs: raw.UpdatedTs,

		Name:          raw.Name,
		Type:          raw.Type,
		InstanceURL:   raw.InstanceURL,
		APIURL:        raw.APIURL,
		ApplicationID: raw.ApplicationID,
		Secret:        raw.Secret,
	}
}

// CreateVCS creates an instance of VCS
func (s *Store) CreateVCS(ctx context.Context, create *api.VCSCreate) (*api.VCS, error) {
	vcsRaw, err := s.createVCSRaw(ctx, create)
	if err != nil {
		return nil, fmt.Errorf("Failed to create VCS with VCSCreate[%+v], error[%w]", create, err)
	}
	vcs, err := s.composeVCS(ctx, vcsRaw)
	if err != nil {
		return nil, fmt.Errorf("Failed to compose VCS with vcsRaw[%+v], error[%w]", vcsRaw, err)
	}
	return vcs, nil
}

// FindVCSList finds a list of VCS instances
func (s *Store) FindVCSList(ctx context.Context, find *api.VCSFind) ([]*api.VCS, error) {
	vcsRawList, err := s.findVCSListRaw(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("Failed to find VCS list, error[%w]", err)
	}
	var vcsList []*api.VCS
	for _, raw := range vcsRawList {
		vcs, err := s.composeVCS(ctx, raw)
		if err != nil {
			return nil, fmt.Errorf("Failed to compose VCS with vcsRaw[%+v], error[%w]", raw, err)
		}
		vcsList = append(vcsList, vcs)
	}
	return vcsList, nil
}

// FindVCS finds an instance of VCS
func (s *Store) FindVCS(ctx context.Context, find *api.VCSFind) (*api.VCS, error) {
	vcsRaw, err := s.findVCSRaw(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("Failed to find VCS with VCSFind[%+v], error[%w]", find, err)
	}
	vcs, err := s.composeVCS(ctx, vcsRaw)
	if err != nil {
		return nil, fmt.Errorf("Failed to compose VCS with vcsRaw[%+v], error[%w]", vcsRaw, err)
	}
	return vcs, nil
}

// PatchVCS patches an instance of VCS
func (s *Store) PatchVCS(ctx context.Context, patch *api.VCSPatch) (*api.VCS, error) {
	vcsRaw, err := s.patchVCSRaw(ctx, patch)
	if err != nil {
		return nil, fmt.Errorf("Failed to patch VCS with VCSPatch[%+v], error[%w]", patch, err)
	}
	vcs, err := s.composeVCS(ctx, vcsRaw)
	if err != nil {
		return nil, fmt.Errorf("Failed to compose VCS with vcsRaw[%+v], error[%w]", vcsRaw, err)
	}
	return vcs, nil
}

// DeleteVCS deletes an instance of VCS
func (s *Store) DeleteVCS(ctx context.Context, delete *api.VCSDelete) error {
	if err := s.deleteVCSRaw(ctx, delete); err != nil {
		return fmt.Errorf("Failed to delete VCS with VCSDelete[%+v], error[%w]", delete, err)
	}
	return nil
}

// GetVCSByID gets a composesd instance of VCS by ID
func (s *Store) GetVCSByID(ctx context.Context, id int) (*api.VCS, error) {
	vcsFind := &api.VCSFind{
		ID: &id,
	}
	vcsRaw, err := s.findVCSRaw(ctx, vcsFind)
	if err != nil {
		return nil, err
	}

	vcs, err := s.composeVCS(ctx, vcsRaw)
	if err != nil {
		return nil, err
	}
	return vcs, nil
}

//
// private functions
//

func (s *Store) composeVCS(ctx context.Context, raw *VCSRaw) (*api.VCS, error) {
	vcs := raw.ToVCS()

	creator, err := s.GetPrincipalByID(ctx, vcs.CreatorID)
	if err != nil {
		return nil, err
	}
	vcs.Creator = creator

	updater, err := s.GetPrincipalByID(ctx, vcs.UpdaterID)
	if err != nil {
		return nil, err
	}
	vcs.Updater = updater

	return vcs, nil
}

// createVCSRaw creates a new vcs.
func (s *Store) createVCSRaw(ctx context.Context, create *api.VCSCreate) (*VCSRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	vcs, err := createVCSImpl(ctx, tx.PTx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return vcs, nil
}

// findVCSListRaw retrieves a list of VCSs based on find conditions.
func (s *Store) findVCSListRaw(ctx context.Context, find *api.VCSFind) ([]*VCSRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := findVCSListImpl(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	return list, nil
}

// findVCSRaw retrieves a single vcs based on find.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *Store) findVCSRaw(ctx context.Context, find *api.VCSFind) (*VCSRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := findVCSListImpl(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("VCS not found with VCSFind[%+v]", find)}
	} else if len(list) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: fmt.Errorf("found %d VCSs with filter %+v, expect 1", len(list), find)}
	}
	return list[0], nil
}

// patchVCSRaw updates an existing vcs by ID.
// Returns ENOTFOUND if vcs does not exist.
func (s *Store) patchVCSRaw(ctx context.Context, patch *api.VCSPatch) (*VCSRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	vcs, err := patchVCSImpl(ctx, tx.PTx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return vcs, nil
}

// deleteVCSRaw deletes an existing vcs by ID.
func (s *Store) deleteVCSRaw(ctx context.Context, delete *api.VCSDelete) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FormatError(err)
	}
	defer tx.PTx.Rollback()

	if err := deleteVCSImpl(ctx, tx.PTx, delete); err != nil {
		return FormatError(err)
	}

	if err := tx.PTx.Commit(); err != nil {
		return FormatError(err)
	}

	return nil
}

// createVCSImpl creates a new vcs.
func createVCSImpl(ctx context.Context, tx *sql.Tx, create *api.VCSCreate) (*VCSRaw, error) {
	// Insert row into database.
	row, err := tx.QueryContext(ctx, `
		INSERT INTO vcs (
			creator_id,
			updater_id,
			name,
			type,
			instance_url,
			api_url,
			application_id,
			secret
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, name, type, instance_url, api_url, application_id, secret
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
	var vcs VCSRaw
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

func findVCSListImpl(ctx context.Context, tx *sql.Tx, find *api.VCSFind) ([]*VCSRaw, error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, "id = $1"), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			creator_id,
			created_ts,
			updater_id,
			updated_ts,
			name,
			type,
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
	var list []*VCSRaw
	for rows.Next() {
		var vcs VCSRaw
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

// patchVCSImpl updates a vcs by ID. Returns the new state of the vcs after update.
func patchVCSImpl(ctx context.Context, tx *sql.Tx, patch *api.VCSPatch) (*VCSRaw, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = $1"}, []interface{}{patch.UpdaterID}
	if v := patch.Name; v != nil {
		set, args = append(set, fmt.Sprintf("name = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.ApplicationID; v != nil {
		set, args = append(set, fmt.Sprintf("application_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Secret; v != nil {
		set, args = append(set, fmt.Sprintf("secret = $%d", len(args)+1)), append(args, *v)
	}
	args = append(args, patch.ID)

	// Execute update query with RETURNING.
	row, err := tx.QueryContext(ctx, fmt.Sprintf(`
		UPDATE vcs
		SET `+strings.Join(set, ", ")+`
		WHERE id = $%d
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, name, type, instance_url, api_url, application_id, secret
	`, len(args)),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	if row.Next() {
		var vcs VCSRaw
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

// deleteVCSImpl permanently deletes a vcs by ID.
func deleteVCSImpl(ctx context.Context, tx *sql.Tx, delete *api.VCSDelete) error {
	// Remove row from database.
	if _, err := tx.ExecContext(ctx, `DELETE FROM vcs WHERE id = $1`, delete.ID); err != nil {
		return FormatError(err)
	}
	return nil
}
