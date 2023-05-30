package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/vcs"
)

// vcsRaw is the store model for a VCS (Version Control System).
// Fields have exactly the same meanings as VCS.
type vcsRaw struct {
	ID int

	// Domain specific fields
	Name          string
	Type          vcs.Type
	InstanceURL   string
	APIURL        string
	ApplicationID string
	Secret        string
}

// toVCS creates an instance of VCS based on the VCSRaw.
// This is intended to be called when we need to compose a VCS relationship.
func (raw *vcsRaw) toVCS() *api.VCS {
	return &api.VCS{
		ID: raw.ID,

		Name:          raw.Name,
		Type:          raw.Type,
		InstanceURL:   raw.InstanceURL,
		APIURL:        raw.APIURL,
		ApplicationID: raw.ApplicationID,
		Secret:        raw.Secret,
	}
}

// CreateVCS creates an instance of VCS.
func (s *Store) CreateVCS(ctx context.Context, create *api.VCSCreate) (*api.VCS, error) {
	vcsRaw, err := s.createVCSRaw(ctx, create)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create VCS with VCSCreate[%+v]", create)
	}
	return vcsRaw.toVCS(), nil
}

// FindVCS finds a list of VCS instances.
func (s *Store) FindVCS(ctx context.Context, find *api.VCSFind) ([]*api.VCS, error) {
	vcsRawList, err := s.findVCSRaw(ctx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find VCS list with VCSFind[%+v]", find)
	}
	var vcsList []*api.VCS
	for _, raw := range vcsRawList {
		vcsList = append(vcsList, raw.toVCS())
	}
	return vcsList, nil
}

// GetVCSByID gets a composed instance of VCS by ID.
func (s *Store) GetVCSByID(ctx context.Context, id int) (*api.VCS, error) {
	vcsRaw, err := s.getVCSRaw(ctx, id)
	if err != nil {
		return nil, err
	}
	if vcsRaw == nil {
		return nil, nil
	}
	return vcsRaw.toVCS(), nil
}

// PatchVCS patches an instance of VCS.
func (s *Store) PatchVCS(ctx context.Context, patch *api.VCSPatch) (*api.VCS, error) {
	vcsRaw, err := s.patchVCSRaw(ctx, patch)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to patch VCS with VCSPatch[%+v]", patch)
	}
	return vcsRaw.toVCS(), nil
}

// DeleteVCS deletes an instance of VCS.
func (s *Store) DeleteVCS(ctx context.Context, delete *api.VCSDelete) error {
	if err := s.deleteVCSRaw(ctx, delete); err != nil {
		return errors.Wrapf(err, "failed to delete VCS with VCSDelete[%+v]", delete)
	}
	return nil
}

//
// private functions
//

// createVCSRaw creates a new vcs.
func (s *Store) createVCSRaw(ctx context.Context, create *api.VCSCreate) (*vcsRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	vcs, err := createVCSImpl(ctx, tx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return vcs, nil
}

// findVCSRaw retrieves a list of VCSs based on find conditions.
func (s *Store) findVCSRaw(ctx context.Context, find *api.VCSFind) ([]*vcsRaw, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	list, err := findVCSImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	return list, nil
}

// getVCSRaw retrieves a single vcs based on find.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *Store) getVCSRaw(ctx context.Context, id int) (*vcsRaw, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	find := &api.VCSFind{
		ID: &id,
	}
	list, err := findVCSImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, nil
	} else if len(list) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d VCSs with filter %+v, expect 1", len(list), find)}
	}
	return list[0], nil
}

// patchVCSRaw updates an existing vcs by ID.
// Returns ENOTFOUND if vcs does not exist.
func (s *Store) patchVCSRaw(ctx context.Context, patch *api.VCSPatch) (*vcsRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	vcs, err := patchVCSImpl(ctx, tx, patch)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return vcs, nil
}

// deleteVCSRaw deletes an existing vcs by ID.
func (s *Store) deleteVCSRaw(ctx context.Context, delete *api.VCSDelete) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := s.deleteVCSImpl(ctx, tx, delete); err != nil {
		return err
	}

	return tx.Commit()
}

//
// database access implementations
//

// createVCSImpl creates a new vcs.
func createVCSImpl(ctx context.Context, tx *Tx, create *api.VCSCreate) (*vcsRaw, error) {
	// Insert row into database.
	query := `
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
		RETURNING id, name, type, instance_url, api_url, application_id, secret
	`
	var vcs vcsRaw
	if err := tx.QueryRowContext(ctx, query,
		create.CreatorID,
		create.CreatorID,
		create.Name,
		create.Type,
		create.InstanceURL,
		create.APIURL,
		create.ApplicationID,
		create.Secret,
	).Scan(
		&vcs.ID,
		&vcs.Name,
		&vcs.Type,
		&vcs.InstanceURL,
		&vcs.APIURL,
		&vcs.ApplicationID,
		&vcs.Secret,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, err
	}
	return &vcs, nil
}

func findVCSImpl(ctx context.Context, tx *Tx, find *api.VCSFind) ([]*vcsRaw, error) {
	// Build WHERE clause.
	where, args := []string{"TRUE"}, []any{}
	if v := find.ID; v != nil {
		where, args = append(where, "id = $1"), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
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
		return nil, err
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into list.
	var list []*vcsRaw
	for rows.Next() {
		var vcs vcsRaw
		if err := rows.Scan(
			&vcs.ID,
			&vcs.Name,
			&vcs.Type,
			&vcs.InstanceURL,
			&vcs.APIURL,
			&vcs.ApplicationID,
			&vcs.Secret,
		); err != nil {
			return nil, err
		}

		list = append(list, &vcs)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return list, nil
}

// patchVCSImpl updates a vcs by ID. Returns the new state of the vcs after update.
func patchVCSImpl(ctx context.Context, tx *Tx, patch *api.VCSPatch) (*vcsRaw, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = $1"}, []any{patch.UpdaterID}
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

	var vcs vcsRaw
	// Execute update query with RETURNING.
	if err := tx.QueryRowContext(ctx, fmt.Sprintf(`
		UPDATE vcs
		SET `+strings.Join(set, ", ")+`
		WHERE id = $%d
		RETURNING id, name, type, instance_url, api_url, application_id, secret
	`, len(args)),
		args...,
	).Scan(
		&vcs.ID,
		&vcs.Name,
		&vcs.Type,
		&vcs.InstanceURL,
		&vcs.APIURL,
		&vcs.ApplicationID,
		&vcs.Secret,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("vcs ID not found: %d", patch.ID)}
		}
		return nil, err
	}
	return &vcs, nil
}

// deleteVCSImpl permanently deletes a vcs by ID.
func (*Store) deleteVCSImpl(ctx context.Context, tx *Tx, delete *api.VCSDelete) error {
	// Remove row from database.
	if _, err := tx.ExecContext(ctx, `DELETE FROM vcs WHERE id = $1`, delete.ID); err != nil {
		return err
	}
	return nil
}

// ExternalVersionControlMessage is a message for external version control.
type ExternalVersionControlMessage struct {
	// Type is the type of the external version control.
	Type vcs.Type
	// APIURL is the URL for the external version control API.
	APIURL string
	// InstanceURL is the URL for the external version control instance.
	InstanceURL string
	// Name is the name of the external version control.
	Name string
	// Secret is the secret for the external version control.
	Secret string
	// ApplicationID is the ID of the application.
	ApplicationID string

	// Output only fields.
	//
	// ID is the unique identifier of the message.
	ID int
}

// UpdateExternalVersionControlMessage is a message for updating an external version control.
type UpdateExternalVersionControlMessage struct {
	// Name is the name of the external version control.
	Name *string
	// Secret is the secret for the external version control.
	Secret *string
	// ApplicationID is the ID of the application.
	ApplicationID *string
}

type findExternalVersionControlMessage struct {
	// If specified, only external version controls with the given ID will be returned.
	id *int
}

// GetExternalVersionControlV2 gets an external version control by ID.
func (s *Store) GetExternalVersionControlV2(ctx context.Context, id int) (*ExternalVersionControlMessage, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin transaction")
	}
	externalVersionControls, err := s.findExternalVersionControlsImplV2(ctx, tx, &findExternalVersionControlMessage{id: &id})
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}
	if len(externalVersionControls) == 0 {
		return nil, nil
	} else if len(externalVersionControls) > 1 {
		return nil, errors.Errorf("expected 1 external version control with id %d, got %d", id, len(externalVersionControls))
	}

	return externalVersionControls[0], nil
}

// ListExternalVersionControls lists all external version controls.
func (s *Store) ListExternalVersionControls(ctx context.Context) ([]*ExternalVersionControlMessage, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin transaction")
	}
	externalVersionControls, err := s.findExternalVersionControlsImplV2(ctx, tx, &findExternalVersionControlMessage{})
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}
	return externalVersionControls, nil
}

// CreateExternalVersionControlV2 creates an external version control.
func (s *Store) CreateExternalVersionControlV2(ctx context.Context, principalUID int, create *ExternalVersionControlMessage) (*ExternalVersionControlMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin transaction")
	}

	query := `
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
		RETURNING id, name, type, instance_url, api_url, application_id, secret
	`
	var externalVersionControl ExternalVersionControlMessage
	if err := tx.QueryRowContext(ctx, query,
		principalUID,
		principalUID,
		create.Name,
		create.Type,
		create.InstanceURL,
		create.APIURL,
		create.ApplicationID,
		create.Secret,
	).Scan(
		&externalVersionControl.ID,
		&externalVersionControl.Name,
		&externalVersionControl.Type,
		&externalVersionControl.InstanceURL,
		&externalVersionControl.APIURL,
		&externalVersionControl.ApplicationID,
		&externalVersionControl.Secret,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}
	return &externalVersionControl, nil
}

// UpdateExternalVersionControlV2 updates an external version control.
func (s *Store) UpdateExternalVersionControlV2(ctx context.Context, principalUID int, externalVersionControlUID int, update *UpdateExternalVersionControlMessage) (*ExternalVersionControlMessage, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = $1"}, []any{principalUID}
	if v := update.Name; v != nil {
		set, args = append(set, fmt.Sprintf("name = $%d", len(args)+1)), append(args, *v)
	}
	if v := update.ApplicationID; v != nil {
		set, args = append(set, fmt.Sprintf("application_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := update.Secret; v != nil {
		set, args = append(set, fmt.Sprintf("secret = $%d", len(args)+1)), append(args, *v)
	}
	args = append(args, externalVersionControlUID)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin transaction")
	}

	var externalVersionControl ExternalVersionControlMessage
	// Execute update query with RETURNING.
	if err := tx.QueryRowContext(ctx, fmt.Sprintf(`
		UPDATE vcs
		SET `+strings.Join(set, ", ")+`
		WHERE id = $%d
		RETURNING id, name, type, instance_url, api_url, application_id, secret
	`, len(args)),
		args...,
	).Scan(
		&externalVersionControl.ID,
		&externalVersionControl.Name,
		&externalVersionControl.Type,
		&externalVersionControl.InstanceURL,
		&externalVersionControl.APIURL,
		&externalVersionControl.ApplicationID,
		&externalVersionControl.Secret,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("vcs ID not found: %d", externalVersionControlUID)}
		}
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}
	return &externalVersionControl, nil
}

// DeleteExternalVersionControlV2 deletes an external version control.
func (s *Store) DeleteExternalVersionControlV2(ctx context.Context, externalVersionControlUID int) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to begin transaction")
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM vcs WHERE id = $1`, externalVersionControlUID); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrapf(err, "failed to commit transaction")
	}
	return nil
}

func (*Store) findExternalVersionControlsImplV2(ctx context.Context, tx *Tx, find *findExternalVersionControlMessage) ([]*ExternalVersionControlMessage, error) {
	where, args := []string{"TRUE"}, []any{}
	if v := find.id; v != nil {
		// Build WHERE clause.
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
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
		return nil, err
	}
	defer rows.Close()
	var externalVersionControls []*ExternalVersionControlMessage
	for rows.Next() {
		var externalVersionControl ExternalVersionControlMessage
		if err := rows.Scan(
			&externalVersionControl.ID,
			&externalVersionControl.Name,
			&externalVersionControl.Type,
			&externalVersionControl.InstanceURL,
			&externalVersionControl.APIURL,
			&externalVersionControl.ApplicationID,
			&externalVersionControl.Secret,
		); err != nil {
			return nil, err
		}
		externalVersionControls = append(externalVersionControls, &externalVersionControl)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return externalVersionControls, nil
}
