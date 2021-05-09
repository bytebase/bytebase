package sqlite

import (
	"context"
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

// FindPrincipalByEmail retrieves a principal by email.
// Returns ENOTFOUND if no matching email.
func (s *PrincipalService) FindPrincipalByEmail(ctx context.Context, email string) (*api.Principal, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := findPrincipalList(ctx, tx, principalFilter{email: &email})
	if err != nil {
		return nil, err
	} else if len(list) == 0 {
		return nil, &bytebase.Error{Code: bytebase.ENOTFOUND, Message: "principal not found."}
	}
	return list[0], nil
}

// FindPrincipalByID retrieves a principal by id.
// Returns ENOTFOUND if no matching id.
func (s *PrincipalService) FindPrincipalByID(ctx context.Context, id int) (*api.Principal, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	return findPrincipalByID(ctx, tx, id)
}

// FindPrincipalList retrieves a list of principals.
func (s *PrincipalService) FindPrincipalList(ctx context.Context) ([]*api.Principal, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := findPrincipalList(ctx, tx, principalFilter{})
	if err != nil {
		return []*api.Principal{}, err
	}

	return list, nil
}

// PatchPrincipalByID updates an existing principal by ID.
// Returns ENOTFOUND if principal does not exist.
func (s *PrincipalService) PatchPrincipalByID(ctx context.Context, id int, patch *api.PrincipalPatch) (*api.Principal, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	principal, err := patchPrincipal(ctx, tx, id, patch)
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
		RETURNING *
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

// principalFilter represents a filter passed to findPrincipalList().
type principalFilter struct {
	// Filtering fields.
	ID    *int
	email *string
}

func findPrincipalList(ctx context.Context, tx *Tx, filter principalFilter) (_ []*api.Principal, err error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := filter.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, *v)
	}
	if v := filter.email; v != nil {
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

// findPrincipalByID retrieves a principal by id.
// Returns ENOTFOUND if no matching id.
func findPrincipalByID(ctx context.Context, tx *Tx, id int) (*api.Principal, error) {
	list, err := findPrincipalList(ctx, tx, principalFilter{ID: &id})
	if err != nil {
		return nil, err
	} else if len(list) == 0 {
		return nil, &bytebase.Error{Code: bytebase.ENOTFOUND, Message: "principal not found."}
	}
	return list[0], nil
}

// patchPrincipal updates a principal by ID. Returns the new state of the principal after update.
func patchPrincipal(ctx context.Context, tx *Tx, id int, patch *api.PrincipalPatch) (*api.Principal, error) {
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
		RETURNING *
	`,
		principal.Name,
		principal.UpdaterId,
		id,
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

	return nil, &bytebase.Error{Code: bytebase.ENOTFOUND, Message: "principal not found."}
}
