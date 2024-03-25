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

// VCSProviderMessage is a message for VCS provider.
type VCSProviderMessage struct {
	// ResourceID is the unique resource ID.
	ResourceID string
	// Type is the type of the VCS provider.
	Type vcs.Type
	// InstanceURL is the URL for the VCS provider instance.
	InstanceURL string
	// Name is the name of the VCS provider.
	Name string
	// AccessToken is the access token for the VCS provider.
	AccessToken string

	// Output only fields.
	//
	// ID is the unique identifier of the message.
	ID int
}

// UpdateVCSProviderMessage is a message for updating an VCS provider.
type UpdateVCSProviderMessage struct {
	// Name is the name of the VCS provider.
	Name *string
	// AccessToken is the secret for the VCS provider.
	AccessToken *string
}

type FindVCSProviderMessage struct {
	// If specified, only VCS providers with the given ID will be returned.
	ID         *int
	ResourceID *string
}

// GetVCSProviderV2 gets an VCS provider by ID.
func (s *Store) GetVCSProviderV2(ctx context.Context, id int) (*VCSProviderMessage, error) {
	if v, ok := s.vcsIDCache.Get(id); ok {
		return v, nil
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	vcsProviders, err := s.findVCSProvidersImplV2(ctx, tx, &FindVCSProviderMessage{ID: &id})
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}
	if len(vcsProviders) == 0 {
		return nil, nil
	} else if len(vcsProviders) > 1 {
		return nil, errors.Errorf("expected 1 VCS provider with id %d, got %d", id, len(vcsProviders))
	}

	vcs := vcsProviders[0]
	s.vcsIDCache.Add(vcs.ID, vcs)
	return vcs, nil
}

// ListVCSProviders lists all VCS providers.
func (s *Store) ListVCSProviders(ctx context.Context) ([]*VCSProviderMessage, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	vcsProviders, err := s.findVCSProvidersImplV2(ctx, tx, &FindVCSProviderMessage{})
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}

	for _, vcs := range vcsProviders {
		s.vcsIDCache.Add(vcs.ID, vcs)
	}
	return vcsProviders, nil
}

// CreateVCSProviderV2 creates an VCS provider.
func (s *Store) CreateVCSProviderV2(ctx context.Context, principalUID int, create *VCSProviderMessage) (*VCSProviderMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	query := `
		INSERT INTO vcs (
			creator_id,
			updater_id,
			resource_id,
			name,
			type,
			instance_url,
			access_token
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, resource_id, name, type, instance_url, access_token
	`
	var vcsProvider VCSProviderMessage
	if err := tx.QueryRowContext(ctx, query,
		principalUID,
		principalUID,
		create.ResourceID,
		create.Name,
		create.Type,
		create.InstanceURL,
		create.AccessToken,
	).Scan(
		&vcsProvider.ID,
		&vcsProvider.ResourceID,
		&vcsProvider.Name,
		&vcsProvider.Type,
		&vcsProvider.InstanceURL,
		&vcsProvider.AccessToken,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}

	s.vcsIDCache.Add(vcsProvider.ID, &vcsProvider)
	return &vcsProvider, nil
}

// UpdateVCSProviderV2 updates an VCS provider.
func (s *Store) UpdateVCSProviderV2(ctx context.Context, principalUID int, vcsProviderUID int, update *UpdateVCSProviderMessage) (*VCSProviderMessage, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = $1"}, []any{principalUID}
	if v := update.Name; v != nil {
		set, args = append(set, fmt.Sprintf("name = $%d", len(args)+1)), append(args, *v)
	}
	if v := update.AccessToken; v != nil {
		set, args = append(set, fmt.Sprintf("access_token = $%d", len(args)+1)), append(args, *v)
	}
	args = append(args, vcsProviderUID)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	var vcsProvider VCSProviderMessage
	// Execute update query with RETURNING.
	if err := tx.QueryRowContext(ctx, fmt.Sprintf(`
		UPDATE vcs
		SET `+strings.Join(set, ", ")+`
		WHERE id = $%d
		RETURNING id, resource_id name, type, instance_url, access_token
	`, len(args)),
		args...,
	).Scan(
		&vcsProvider.ID,
		&vcsProvider.ResourceID,
		&vcsProvider.Name,
		&vcsProvider.Type,
		&vcsProvider.InstanceURL,
		&vcsProvider.AccessToken,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("vcs ID not found: %d", vcsProviderUID)}
		}
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}
	s.vcsIDCache.Add(vcsProvider.ID, &vcsProvider)
	return &vcsProvider, nil
}

// DeleteVCSProviderV2 deletes an VCS provider.
func (s *Store) DeleteVCSProviderV2(ctx context.Context, vcsProviderUID int) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM vcs WHERE id = $1`, vcsProviderUID); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrapf(err, "failed to commit transaction")
	}

	s.vcsIDCache.Remove(vcsProviderUID)
	return nil
}

func (*Store) findVCSProvidersImplV2(ctx context.Context, tx *Tx, find *FindVCSProviderMessage) ([]*VCSProviderMessage, error) {
	where, args := []string{"TRUE"}, []any{}
	if v := find.ID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.ResourceID; v != nil {
		where, args = append(where, fmt.Sprintf("resource_id = $%d", len(args)+1)), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			resource_id,
			name,
			type,
			instance_url,
			access_token
		FROM vcs
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var vcsProviders []*VCSProviderMessage
	for rows.Next() {
		var vcsProvider VCSProviderMessage
		if err := rows.Scan(
			&vcsProvider.ID,
			&vcsProvider.ResourceID,
			&vcsProvider.Name,
			&vcsProvider.Type,
			&vcsProvider.InstanceURL,
			&vcsProvider.AccessToken,
		); err != nil {
			return nil, err
		}
		vcsProviders = append(vcsProviders, &vcsProvider)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return vcsProviders, nil
}
