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
	_ api.PrincipalService = (*PrincipalService)(nil)
)

// PrincipalService represents a service for managing principal.
type PrincipalService struct {
	l  *zap.Logger
	db *DB

	cache api.CacheService
}

// NewPrincipalService returns a new instance of PrincipalService.
func NewPrincipalService(logger *zap.Logger, db *DB, cache api.CacheService) *PrincipalService {
	return &PrincipalService{l: logger, db: db, cache: cache}
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

	if err := s.cache.UpsertCache(api.PrincipalCache, principal.ID, principal); err != nil {
		return nil, err
	}

	return principal, nil
}

// FindPrincipalList retrieves a list of principals.
func (s *PrincipalService) FindPrincipalList(ctx context.Context) ([]*api.Principal, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := findPrincipalList(ctx, tx, &api.PrincipalFind{})
	if err != nil {
		return []*api.Principal{}, err
	}

	if err == nil {
		for _, principal := range list {
			if err := s.cache.UpsertCache(api.PrincipalCache, principal.ID, principal); err != nil {
				return nil, err
			}
		}
	}

	return list, nil
}

// FindPrincipal retrieves a principal based on find.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *PrincipalService) FindPrincipal(ctx context.Context, find *api.PrincipalFind) (*api.Principal, error) {
	if find.ID != nil {
		principal := &api.Principal{}
		has, err := s.cache.FindCache(api.PrincipalCache, *find.ID, principal)
		if err != nil {
			return nil, err
		}
		if has {
			return principal, nil
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := findPrincipalList(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, nil
	} else if len(list) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: fmt.Errorf("found %d principals with filter %+v, expect 1", len(list), find)}
	}
	if err := s.cache.UpsertCache(api.PrincipalCache, list[0].ID, list[0]); err != nil {
		return nil, err
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

	if err := s.cache.UpsertCache(api.PrincipalCache, principal.ID, principal); err != nil {
		return nil, err
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
			type,
			name,
			email,
			password_hash
		)
		VALUES (?, ?, ?, ?, ?, ?)
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, type, name, email, password_hash
	`,
		create.CreatorID,
		create.CreatorID,
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
		&principal.CreatorID,
		&principal.CreatedTs,
		&principal.UpdaterID,
		&principal.UpdatedTs,
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
			&principal.CreatorID,
			&principal.CreatedTs,
			&principal.UpdaterID,
			&principal.UpdatedTs,
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
	set, args := []string{"updater_id = ?"}, []interface{}{patch.UpdaterID}
	if v := patch.Name; v != nil {
		set, args = append(set, "name = ?"), append(args, *v)
	}
	if v := patch.PasswordHash; v != nil {
		set, args = append(set, "password_hash = ?"), append(args, *v)
	}

	args = append(args, patch.ID)

	// Execute update query with RETURNING.
	row, err := tx.QueryContext(ctx, `
		UPDATE principal
		SET `+strings.Join(set, ", ")+`
		WHERE id = ?
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, type, name, email, password_hash
	`,
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	if row.Next() {
		var principal api.Principal
		if err := row.Scan(
			&principal.ID,
			&principal.CreatorID,
			&principal.CreatedTs,
			&principal.UpdaterID,
			&principal.UpdatedTs,
			&principal.Type,
			&principal.Name,
			&principal.Email,
			&principal.PasswordHash,
		); err != nil {
			return nil, FormatError(err)
		}

		return &principal, nil
	}

	return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("principal ID not found: %d", patch.ID)}
}
