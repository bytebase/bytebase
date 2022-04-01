package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"go.uber.org/zap"
)

// principalRaw is the store model for a Principal.
// Fields have exactly the same meanings as Principal.
type principalRaw struct {
	ID int

	// Standard fields
	CreatorID int
	CreatedTs int64
	UpdaterID int
	UpdatedTs int64

	// Domain specific fields
	Type  api.PrincipalType
	Name  string
	Email string
	// Do not return to the client
	PasswordHash string
}

// toPrincipal creates an instance of Principal based on the principalRaw.
// This is intended to be called when we need to compose a Principal relationship.
func (raw *principalRaw) toPrincipal() *api.Principal {
	return &api.Principal{
		ID: raw.ID,

		// Standard fields
		CreatorID: raw.CreatorID,
		CreatedTs: raw.CreatedTs,
		UpdaterID: raw.UpdaterID,
		UpdatedTs: raw.UpdatedTs,

		// Domain specific fields
		Type:  raw.Type,
		Name:  raw.Name,
		Email: raw.Email,
		// Do not return to the client
		PasswordHash: raw.PasswordHash,
	}
}

// CreatePrincipal creates an instance of Principal
func (s *Store) CreatePrincipal(ctx context.Context, create *api.PrincipalCreate) (*api.Principal, error) {
	principalRaw, err := s.createPrincipalRaw(ctx, create)
	if err != nil {
		return nil, fmt.Errorf("Failed to create Principal with PrincipalCreate[%+v], error[%w]", create, err)
	}
	// NOTE: Currently the corresponding Member object is not created yet.
	// YES, we are returning a Principal with empty Role field. OMG.
	principal := principalRaw.toPrincipal()
	return principal, nil
}

// GetPrincipalList gets a list of Principal instances
func (s *Store) GetPrincipalList(ctx context.Context) ([]*api.Principal, error) {
	principalRawList, err := s.findPrincipalRawList(ctx)
	if err != nil {
		return nil, fmt.Errorf("Failed to find Principal list, error[%w]", err)
	}
	var principalList []*api.Principal
	for _, raw := range principalRawList {
		principal, err := s.composePrincipal(ctx, raw)
		if err != nil {
			return nil, fmt.Errorf("Failed to compose Principal role with principalRaw[%+v], error[%w]", raw, err)
		}
		principalList = append(principalList, principal)
	}
	return principalList, nil
}

// FindPrincipal finds an instance of Principal
// TODO(dragonly): refactor to GetPrincipalByEmail, and redirect callers using ID to GetPrincipalByID
func (s *Store) FindPrincipal(ctx context.Context, find *api.PrincipalFind) (*api.Principal, error) {
	principalRaw, err := s.getPrincipalRaw(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("Failed to find Principal with PrincipalFind[%+v], error[%w]", find, err)
	}
	if principalRaw == nil {
		return nil, nil
	}
	principal, err := s.composePrincipal(ctx, principalRaw)
	if err != nil {
		return nil, fmt.Errorf("Failed to compose Principal role with principalRaw[%+v], error[%w]", principalRaw, err)
	}
	return principal, nil
}

// PatchPrincipal patches an instance of Principal
func (s *Store) PatchPrincipal(ctx context.Context, patch *api.PrincipalPatch) (*api.Principal, error) {
	principalRaw, err := s.patchPrincipalRaw(ctx, patch)
	if err != nil {
		return nil, fmt.Errorf("Failed to patch Principal with PrincipalPatch[%+v], error[%w]", patch, err)
	}
	principal, err := s.composePrincipal(ctx, principalRaw)
	if err != nil {
		return nil, fmt.Errorf("Failed to compose Principal role with principalRaw[%+v], error[%w]", principalRaw, err)
	}
	return principal, nil
}

// GetPrincipalByID gets an instance of Principal by ID
func (s *Store) GetPrincipalByID(ctx context.Context, id int) (*api.Principal, error) {
	principalFind := &api.PrincipalFind{
		ID: &id,
	}
	principalRaw, err := s.getPrincipalRaw(ctx, principalFind)
	if err != nil {
		return nil, err
	}
	if principalRaw == nil {
		return nil, nil
	}

	principal, err := s.composePrincipal(ctx, principalRaw)
	if err != nil {
		return nil, fmt.Errorf("Failed to compose Principal role with principalRaw[%+v], error[%w]", principalRaw, err)
	}

	return principal, nil
}

//
// private functions
//

// createPrincipalRaw creates an instance of principalRaw.
func (s *Store) createPrincipalRaw(ctx context.Context, create *api.PrincipalCreate) (*principalRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	principal, err := createPrincipalImpl(ctx, tx.PTx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	if err := s.cache.UpsertCache(api.PrincipalCache, principal.ID, principal); err != nil {
		return nil, err
	}

	return principal, nil
}

// findPrincipalRawList retrieves a list of principalRaw instances.
func (s *Store) findPrincipalRawList(ctx context.Context) ([]*principalRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := findPrincipalRawListImpl(ctx, tx.PTx, &api.PrincipalFind{})
	if err != nil {
		return nil, err
	}

	for _, principal := range list {
		if err := s.cache.UpsertCache(api.PrincipalCache, principal.ID, principal); err != nil {
			return nil, err
		}
	}

	return list, nil
}

// getPrincipalRaw retrieves an instance of principalRaw based on find.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *Store) getPrincipalRaw(ctx context.Context, find *api.PrincipalFind) (*principalRaw, error) {
	if find.ID != nil {
		principalRaw := &principalRaw{}
		has, err := s.cache.FindCache(api.PrincipalCache, *find.ID, principalRaw)
		if err != nil {
			return nil, err
		}
		if has {
			return principalRaw, nil
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := findPrincipalRawListImpl(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, nil
	} else if len(list) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: fmt.Errorf("found %d principals with PrincipalFind[%+v], expect 1", len(list), find)}
	}
	if err := s.cache.UpsertCache(api.PrincipalCache, list[0].ID, list[0]); err != nil {
		return nil, err
	}

	return list[0], nil
}

// patchPrincipalRaw updates an existing instance of principalRaw by ID.
// Returns ENOTFOUND if principal does not exist.
func (s *Store) patchPrincipalRaw(ctx context.Context, patch *api.PrincipalPatch) (*principalRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	principal, err := patchPrincipalImpl(ctx, tx.PTx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	if err := s.cache.UpsertCache(api.PrincipalCache, principal.ID, principal); err != nil {
		return nil, err
	}

	return principal, nil
}

// composePrincipal composes an instance of Principal by principalRaw
func (s *Store) composePrincipal(ctx context.Context, raw *principalRaw) (*api.Principal, error) {
	principal := raw.toPrincipal()

	if principal.ID == api.SystemBotID {
		principal.Role = api.Owner
	} else {
		memberRaw, err := s.GetMemberByPrincipalID(ctx, principal.ID)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				s.l.Error("Principal has not been assigned a role.",
					zap.Int("id", principal.ID),
					zap.String("name", principal.Name),
				)
			}
			return nil, err
		}
		if memberRaw == nil {
			return nil, fmt.Errorf("Member with PrincipalID[%d] not exist, error[%w]", principal.ID, err)
		}
		principal.Role = memberRaw.Role
	}
	return principal, nil
}

// createPrincipalImpl creates a new principal.
func createPrincipalImpl(ctx context.Context, tx *sql.Tx, create *api.PrincipalCreate) (*principalRaw, error) {
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
		VALUES ($1, $2, $3, $4, $5, $6)
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
	var principalRaw principalRaw
	if err := row.Scan(
		&principalRaw.ID,
		&principalRaw.CreatorID,
		&principalRaw.CreatedTs,
		&principalRaw.UpdaterID,
		&principalRaw.UpdatedTs,
		&principalRaw.Type,
		&principalRaw.Name,
		&principalRaw.Email,
		&principalRaw.PasswordHash,
	); err != nil {
		return nil, FormatError(err)
	}

	return &principalRaw, nil
}

func findPrincipalRawListImpl(ctx context.Context, tx *sql.Tx, find *api.PrincipalFind) ([]*principalRaw, error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.Email; v != nil {
		where, args = append(where, fmt.Sprintf("email = $%d", len(args)+1)), append(args, *v)
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

	// Iterate over result set and deserialize rows into principalRawList.
	var principalRawList []*principalRaw
	for rows.Next() {
		var principalRaw principalRaw
		if err := rows.Scan(
			&principalRaw.ID,
			&principalRaw.CreatorID,
			&principalRaw.CreatedTs,
			&principalRaw.UpdaterID,
			&principalRaw.UpdatedTs,
			&principalRaw.Type,
			&principalRaw.Name,
			&principalRaw.Email,
			&principalRaw.PasswordHash,
		); err != nil {
			return nil, FormatError(err)
		}

		principalRawList = append(principalRawList, &principalRaw)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return principalRawList, nil
}

// patchPrincipalImpl updates a principal by ID. Returns the new state of the principal after update.
func patchPrincipalImpl(ctx context.Context, tx *sql.Tx, patch *api.PrincipalPatch) (*principalRaw, error) {
	set, args := []string{"updater_id = $1"}, []interface{}{patch.UpdaterID}
	if v := patch.Name; v != nil {
		set, args = append(set, fmt.Sprintf("name = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.PasswordHash; v != nil {
		set, args = append(set, fmt.Sprintf("password_hash = $%d", len(args)+1)), append(args, *v)
	}

	args = append(args, patch.ID)

	// Execute update query with RETURNING.
	row, err := tx.QueryContext(ctx, fmt.Sprintf(`
		UPDATE principal
		SET `+strings.Join(set, ", ")+`
		WHERE id = $%d
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, type, name, email, password_hash
	`, len(args)),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	if row.Next() {
		var principalRaw principalRaw
		if err := row.Scan(
			&principalRaw.ID,
			&principalRaw.CreatorID,
			&principalRaw.CreatedTs,
			&principalRaw.UpdaterID,
			&principalRaw.UpdatedTs,
			&principalRaw.Type,
			&principalRaw.Name,
			&principalRaw.Email,
			&principalRaw.PasswordHash,
		); err != nil {
			return nil, FormatError(err)
		}

		return &principalRaw, nil
	}

	return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("principal ID not found: %d", patch.ID)}
}
