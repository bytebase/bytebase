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

var (
	_ PrincipalStore       = (*Store)(nil)
	_ api.PrincipalService = (*Store)(nil)
)

// PrincipalRaw is the store model for a Principal.
// Fields have exactly the same meanings as Principal.
type PrincipalRaw struct {
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

// ToPrincipal creates an instance of Principal based on the PrincipalRaw.
// This is intended to be called when we need to compose a Principal relationship.
func (raw *PrincipalRaw) ToPrincipal() *api.Principal {
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

// PrincipalStore is the store for Principal CRUD operations.
type PrincipalStore interface {
	// Raw data objects
	CreatePrincipalRaw(ctx context.Context, create *api.PrincipalCreate) (*PrincipalRaw, error)
	FindPrincipalRawList(ctx context.Context) ([]*PrincipalRaw, error)
	FindPrincipalRaw(ctx context.Context, find *api.PrincipalFind) (*PrincipalRaw, error)
	PatchPrincipalRaw(ctx context.Context, patch *api.PrincipalPatch) (*PrincipalRaw, error)

	// TODO(dragonly): ComposeByID seems to be identical to Find?
	ComposePrincipalByID(ctx context.Context, id int) (*api.Principal, error)
	ComposePrincipal(ctx context.Context, raw *PrincipalRaw) (*api.Principal, error)
}

// CreatePrincipalRaw creates an instance of PrincipalRaw.
func (s *Store) CreatePrincipalRaw(ctx context.Context, create *api.PrincipalCreate) (*PrincipalRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	principal, err := createPrincipal(ctx, tx.PTx, create)
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

// FindPrincipalRawList retrieves a list of PrincipalRaw instances.
func (s *Store) FindPrincipalRawList(ctx context.Context) ([]*PrincipalRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := findPrincipalList(ctx, tx.PTx, &api.PrincipalFind{})
	if err != nil {
		return nil, err
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

// FindPrincipalRaw retrieves an instance of PrincipalRaw based on find.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *Store) FindPrincipalRaw(ctx context.Context, find *api.PrincipalFind) (*PrincipalRaw, error) {
	if find.ID != nil {
		principalRaw := &PrincipalRaw{}
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

	list, err := findPrincipalList(ctx, tx.PTx, find)
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

// PatchPrincipalRaw updates an existing instance of PrincipalRaw by ID.
// Returns ENOTFOUND if principal does not exist.
func (s *Store) PatchPrincipalRaw(ctx context.Context, patch *api.PrincipalPatch) (*PrincipalRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	principal, err := patchPrincipal(ctx, tx.PTx, patch)
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

// CreatePrincipal creates an instance of Principal
func (s *Store) CreatePrincipal(ctx context.Context, create *api.PrincipalCreate) (*api.Principal, error) {
	principalRaw, err := s.CreatePrincipalRaw(ctx, create)
	if err != nil {
		return nil, fmt.Errorf("Failed to create Principal with PrincipalCreate[%+v], error[%w]", create, err)
	}
	// NOTE: Currently the corresponding Member object is not created yet.
	// YES, we are returning a Principal with empty Role field. OMG.
	principal := principalRaw.ToPrincipal()
	return principal, nil
}

// FindPrincipalList finds a list of Principal instances
func (s *Store) FindPrincipalList(ctx context.Context) ([]*api.Principal, error) {
	principalRawList, err := s.FindPrincipalRawList(ctx)
	if err != nil {
		return nil, fmt.Errorf("Failed to find Principal list, error[%w]", err)
	}
	var principalList []*api.Principal
	for _, raw := range principalRawList {
		principal, err := s.ComposePrincipal(ctx, raw)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				s.l.Error("Principal has not been assigned a role. Skip",
					zap.Int("id", principal.ID),
					zap.String("name", principal.Name),
				)
				continue
			}
			return nil, fmt.Errorf("Failed to compose Principal role with PrincipalRaw[%+v], error[%w]", raw, err)
		}
		principalList = append(principalList, principal)
	}
	return principalList, nil
}

// FindPrincipal finds an instance of Principal
func (s *Store) FindPrincipal(ctx context.Context, find *api.PrincipalFind) (*api.Principal, error) {
	principalRaw, err := s.FindPrincipalRaw(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("Failed to find Principal with PrincipalFind[%+v], error[%w]", find, err)
	}
	if principalRaw == nil {
		return nil, nil
	}
	principal, err := s.ComposePrincipal(ctx, principalRaw)
	if err != nil {
		return nil, fmt.Errorf("Failed to compose Principal role with PrincipalRaw[%+v], error[%w]", principalRaw, err)
	}
	return principal, nil
}

// PatchPrincipal patches an instance of Principal
func (s *Store) PatchPrincipal(ctx context.Context, patch *api.PrincipalPatch) (*api.Principal, error) {
	principalRaw, err := s.PatchPrincipalRaw(ctx, patch)
	if err != nil {
		return nil, fmt.Errorf("Failed to patch Principal with PrincipalPatch[%+v], error[%w]", patch, err)
	}
	principal, err := s.ComposePrincipal(ctx, principalRaw)
	if err != nil {
		return nil, fmt.Errorf("Failed to compose Principal role with PrincipalRaw[%+v], error[%w]", principalRaw, err)
	}
	return principal, nil
}

// ComposePrincipalByID composes an instance of Principal by ID
func (s *Store) ComposePrincipalByID(ctx context.Context, id int) (*api.Principal, error) {
	principalFind := &api.PrincipalFind{
		ID: &id,
	}
	principalRaw, err := s.FindPrincipalRaw(ctx, principalFind)
	if err != nil {
		return nil, err
	}
	if id > 0 && principalRaw == nil {
		return nil, fmt.Errorf("Principal not found with ID[%d], error[%w]", id, err)
	}

	principal, err := s.ComposePrincipal(ctx, principalRaw)
	if err != nil {
		return nil, fmt.Errorf("Failed to compose Principal role with PrincipalRaw[%+v], error[%w]", principalRaw, err)
	}

	return principal, nil
}

// ComposePrincipal composes an instance of Principal by PrincipalRaw
func (s *Store) ComposePrincipal(ctx context.Context, raw *PrincipalRaw) (*api.Principal, error) {
	principal := raw.ToPrincipal()

	if principal.ID == api.SystemBotID {
		principal.Role = api.Owner
	} else {
		memberFind := &api.MemberFind{
			PrincipalID: &principal.ID,
		}
		memberRaw, err := s.FindMemberRaw(ctx, memberFind)
		if err != nil {
			return nil, err
		}
		if principal.ID > 0 && memberRaw == nil {
			return nil, fmt.Errorf("Member not found for ID %v", principal.ID)
		}
		principal.Role = memberRaw.Role
	}
	return principal, nil
}

// createPrincipal creates a new principal.
func createPrincipal(ctx context.Context, tx *sql.Tx, create *api.PrincipalCreate) (*PrincipalRaw, error) {
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
	var principalRaw PrincipalRaw
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

func findPrincipalList(ctx context.Context, tx *sql.Tx, find *api.PrincipalFind) ([]*PrincipalRaw, error) {
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
	var principalRawList []*PrincipalRaw
	for rows.Next() {
		var principalRaw PrincipalRaw
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

// patchPrincipal updates a principal by ID. Returns the new state of the principal after update.
func patchPrincipal(ctx context.Context, tx *sql.Tx, patch *api.PrincipalPatch) (*PrincipalRaw, error) {
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
		var principalRaw PrincipalRaw
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
