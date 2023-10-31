package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/vcs"
)

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
	if vcs, ok := s.vcsIDCache.Load(id); ok {
		if v, ok := vcs.(*ExternalVersionControlMessage); ok {
			return v, nil
		}
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin transaction")
	}
	defer tx.Rollback()

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

	vcs := externalVersionControls[0]
	s.vcsIDCache.Store(vcs.ID, vcs)
	return vcs, nil
}

// ListExternalVersionControls lists all external version controls.
func (s *Store) ListExternalVersionControls(ctx context.Context) ([]*ExternalVersionControlMessage, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	externalVersionControls, err := s.findExternalVersionControlsImplV2(ctx, tx, &findExternalVersionControlMessage{})
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}

	for _, vcs := range externalVersionControls {
		s.vcsIDCache.Store(vcs.ID, vcs)
	}
	return externalVersionControls, nil
}

// CreateExternalVersionControlV2 creates an external version control.
func (s *Store) CreateExternalVersionControlV2(ctx context.Context, principalUID int, create *ExternalVersionControlMessage) (*ExternalVersionControlMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin transaction")
	}
	defer tx.Rollback()

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

	s.vcsIDCache.Store(externalVersionControl.ID, &externalVersionControl)
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
	defer tx.Rollback()

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
	s.vcsIDCache.Store(externalVersionControl.ID, &externalVersionControl)
	return &externalVersionControl, nil
}

// DeleteExternalVersionControlV2 deletes an external version control.
func (s *Store) DeleteExternalVersionControlV2(ctx context.Context, externalVersionControlUID int) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM vcs WHERE id = $1`, externalVersionControlUID); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrapf(err, "failed to commit transaction")
	}

	s.vcsIDCache.Delete(externalVersionControlUID)
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
