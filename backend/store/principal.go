package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase"
	"github.com/bytebase/bytebase/api"
)

var (
	_ api.PrincipalService = (*PrincipalService)(nil)
)

// PrincipalService represents a service for managing principal.
type PrincipalService struct {
	l  *bytebase.Logger
	db *DB
}

// NewPrincipalService returns a new instance of PrincipalService.
func NewPrincipalService(logger *bytebase.Logger, db *DB) *PrincipalService {
	return &PrincipalService{l: logger, db: db}
}

// CreatePrincipal creates a new principal.
func (s *PrincipalService) CreatePrincipal(ctx context.Context, create *api.PrincipalCreate) (*api.Principal, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	principal, err := createPrincipal(ctx, tx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return principal, nil
}

// FindPrincipalList retrieves a list of principals.
func (s *PrincipalService) FindPrincipalList(ctx context.Context, find *api.PrincipalFind) ([]*api.Principal, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := findPrincipalList(ctx, tx, find)
	if err != nil {
		return []*api.Principal{}, err
	}

	return list, nil
}

// FindPrincipal retrieves a principal based on find.
// Returns ENOTFOUND if no matching record.
// Returns the first matching one and prints a warning if finding more than 1 matching records.
func (s *PrincipalService) FindPrincipal(ctx context.Context, find *api.PrincipalFind) (*api.Principal, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := findPrincipalList(ctx, tx, find)
	if err != nil {
		return nil, err
	} else if len(list) == 0 {
		return nil, &bytebase.Error{Code: bytebase.ENOTFOUND, Message: fmt.Sprintf("principal not found: %v", find)}
	} else if len(list) > 1 {
		s.l.Warnf("found mulitple principals: %d, expect 1", len(list))
	}
	return list[0], nil
}

// PatchPrincipal updates an existing principal by ID.
// Returns ENOTFOUND if principal does not exist.
func (s *PrincipalService) PatchPrincipal(ctx context.Context, patch *api.PrincipalPatch) (*api.Principal, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	principal, err := patchPrincipal(ctx, tx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return principal, nil
}

// createPrincipal creates a new principal.
func createPrincipal(ctx context.Context, tx *Tx, create *api.PrincipalCreate) (*api.Principal, error) {
	// Insert row into database.
	row, err := tx.QueryContext(ctx, `
		INSERT INTO principal (
			creator_id,
			updater_id,
			status,
			type,
			name,
			email,
			password_hash
		)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, status, type, name, email, password_hash
	`,
		create.CreatorId,
		create.CreatorId,
		create.Status,
		create.Type,
		create.Name,
		create.Email,
		create.PasswordHash,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var principal api.Principal
	if err := row.Scan(
		&principal.ID,
		&principal.CreatorId,
		&principal.CreatedTs,
		&principal.UpdaterId,
		&principal.UpdatedTs,
		&principal.Status,
		&principal.Type,
		&principal.Name,
		&principal.Email,
		&principal.PasswordHash,
	); err != nil {
		return nil, FormatError(err)
	}

	return &principal, nil
}

func findPrincipalList(ctx context.Context, tx *Tx, find *api.PrincipalFind) (_ []*api.Principal, err error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, *v)
	}
	if v := find.Email; v != nil {
		where, args = append(where, "email = ?"), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT 
		    id,
		    creator_id,
		    created_ts,
		    updater_id,
		    updated_ts,
		    status,
		    type,
		    name,
		    email,
			password_hash
		FROM principal
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into list.
	list := make([]*api.Principal, 0)
	for rows.Next() {
		var principal api.Principal
		if err := rows.Scan(
			&principal.ID,
			&principal.CreatorId,
			&principal.CreatedTs,
			&principal.UpdaterId,
			&principal.UpdatedTs,
			&principal.Status,
			&principal.Type,
			&principal.Name,
			&principal.Email,
			&principal.PasswordHash,
		); err != nil {
			return nil, FormatError(err)
		}

		list = append(list, &principal)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return list, nil
}

// patchPrincipal updates a principal by ID. Returns the new state of the principal after update.
func patchPrincipal(ctx context.Context, tx *Tx, patch *api.PrincipalPatch) (*api.Principal, error) {
	principal := api.Principal{}
	// Update fields, if set.
	if v := patch.Name; v != nil {
		principal.Name = *v
	}

	// Execute update query with RETURNING.
	row, err := tx.QueryContext(ctx, `
		UPDATE principal
		SET name = ?, updater_id = ?
		WHERE id = ?
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, status, type, name, email, password_hash
	`,
		principal.Name,
		patch.UpdaterId,
		patch.ID,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	if row.Next() {
		var principal api.Principal
		if err := row.Scan(
			&principal.ID,
			&principal.CreatorId,
			&principal.CreatedTs,
			&principal.UpdaterId,
			&principal.UpdatedTs,
			&principal.Status,
			&principal.Type,
			&principal.Name,
			&principal.Email,
			&principal.PasswordHash,
		); err != nil {
			return nil, FormatError(err)
		}

		return &principal, nil
	}

	return nil, &bytebase.Error{Code: bytebase.ENOTFOUND, Message: fmt.Sprintf("principal ID not found: %d", patch.ID)}
}
