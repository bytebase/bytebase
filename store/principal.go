package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/common/log"
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

// CreatePrincipal creates an instance of Principal.
func (s *Store) CreatePrincipal(ctx context.Context, create *api.PrincipalCreate) (*api.Principal, error) {
	principalRaw, err := s.createPrincipalRaw(ctx, create)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create Principal with PrincipalCreate[%+v]", create)
	}
	// NOTE: Currently the corresponding Member object is not created yet.
	// YES, we are returning a Principal with empty Role field. OMG.
	principal := principalRaw.toPrincipal()
	return principal, nil
}

// GetPrincipalList gets a list of Principal instances.
func (s *Store) GetPrincipalList(ctx context.Context) ([]*api.Principal, error) {
	principalRawList, err := s.findPrincipalRawList(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find Principal list")
	}
	var principalList []*api.Principal
	for _, raw := range principalRawList {
		principal, err := s.composePrincipal(ctx, raw)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to compose Principal role with principalRaw[%+v]", raw)
		}
		if principal != nil {
			principalList = append(principalList, principal)
		}
	}
	return principalList, nil
}

// GetPrincipalByEmail gets an instance of Principal.
func (s *Store) GetPrincipalByEmail(ctx context.Context, email string) (*api.Principal, error) {
	find := &api.PrincipalFind{Email: &email}
	principalRaw, err := s.getPrincipalRaw(ctx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find Principal with PrincipalFind[%+v]", find)
	}
	if principalRaw == nil {
		return nil, nil
	}
	principal, err := s.composePrincipal(ctx, principalRaw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose Principal role with principalRaw[%+v]", principalRaw)
	}
	return principal, nil
}

// PatchPrincipal patches an instance of Principal.
func (s *Store) PatchPrincipal(ctx context.Context, patch *api.PrincipalPatch) (*api.Principal, error) {
	principalRaw, err := s.patchPrincipalRaw(ctx, patch)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to patch Principal with PrincipalPatch[%+v]", patch)
	}
	principal, err := s.composePrincipal(ctx, principalRaw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose Principal role with principalRaw[%+v]", principalRaw)
	}
	return principal, nil
}

// GetPrincipalByID gets an instance of Principal by ID.
func (s *Store) GetPrincipalByID(ctx context.Context, id int) (*api.Principal, error) {
	principalFind := &api.PrincipalFind{ID: &id}
	principalRaw, err := s.getPrincipalRaw(ctx, principalFind)
	if err != nil {
		return nil, err
	}
	if principalRaw == nil {
		return nil, nil
	}

	principal, err := s.composePrincipal(ctx, principalRaw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose Principal role with principalRaw[%+v]", principalRaw)
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
	defer tx.Rollback()

	principal, err := createPrincipalImpl(ctx, tx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	if err := s.cache.UpsertCache(principalCacheNamespace, principal.ID, principal); err != nil {
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
	defer tx.Rollback()

	list, err := findPrincipalRawListImpl(ctx, tx, &api.PrincipalFind{})
	if err != nil {
		return nil, err
	}

	for _, principal := range list {
		if err := s.cache.UpsertCache(principalCacheNamespace, principal.ID, principal); err != nil {
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
		has, err := s.cache.FindCache(principalCacheNamespace, *find.ID, principalRaw)
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
	defer tx.Rollback()

	list, err := findPrincipalRawListImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, nil
	} else if len(list) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d principals with PrincipalFind[%+v], expect 1", len(list), find)}
	}
	if err := s.cache.UpsertCache(principalCacheNamespace, list[0].ID, list[0]); err != nil {
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
	defer tx.Rollback()

	principal, err := patchPrincipalImpl(ctx, tx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	if err := s.cache.UpsertCache(principalCacheNamespace, principal.ID, principal); err != nil {
		return nil, err
	}

	return principal, nil
}

// composePrincipal composes an instance of Principal by principalRaw.
func (s *Store) composePrincipal(ctx context.Context, raw *principalRaw) (*api.Principal, error) {
	principal := raw.toPrincipal()

	if principal.ID == api.SystemBotID {
		principal.Role = api.Owner
	} else {
		find := &api.MemberFind{PrincipalID: &principal.ID}
		// NOTE: watch out for recursion here, because Member also contains pointers to Principal
		memberRaw, err := s.getMemberRaw(ctx, find)
		if err != nil {
			return nil, err
		}
		if memberRaw == nil {
			log.Error("Principal has not been assigned a role.",
				zap.Int("id", principal.ID),
				zap.String("name", principal.Name),
			)
			return nil, errors.Wrapf(err, "member with PrincipalID %d not exist", principal.ID)
		}
		principal.Role = memberRaw.Role
	}
	return principal, nil
}

// createPrincipalImpl creates a new principal.
func createPrincipalImpl(ctx context.Context, tx *Tx, create *api.PrincipalCreate) (*principalRaw, error) {
	// Insert row into database.
	query := `
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
	`
	var principalRaw principalRaw
	if err := tx.QueryRowContext(ctx, query,
		create.CreatorID,
		create.CreatorID,
		create.Type,
		create.Name,
		create.Email,
		create.PasswordHash,
	).Scan(
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
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, FormatError(err)
	}
	return &principalRaw, nil
}

func findPrincipalRawListImpl(ctx context.Context, tx *Tx, find *api.PrincipalFind) ([]*principalRaw, error) {
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
func patchPrincipalImpl(ctx context.Context, tx *Tx, patch *api.PrincipalPatch) (*principalRaw, error) {
	set, args := []string{"updater_id = $1"}, []interface{}{patch.UpdaterID}
	if v := patch.Name; v != nil {
		set, args = append(set, fmt.Sprintf("name = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Email; v != nil {
		set, args = append(set, fmt.Sprintf("email = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.PasswordHash; v != nil {
		set, args = append(set, fmt.Sprintf("password_hash = $%d", len(args)+1)), append(args, *v)
	}

	args = append(args, patch.ID)

	var principalRaw principalRaw
	// Execute update query with RETURNING.
	if err := tx.QueryRowContext(ctx, fmt.Sprintf(`
		UPDATE principal
		SET `+strings.Join(set, ", ")+`
		WHERE id = $%d
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, type, name, email, password_hash
	`, len(args)),
		args...,
	).Scan(
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
		if err == sql.ErrNoRows {
			return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("principal ID not found: %d", patch.ID)}
		}
		return nil, FormatError(err)
	}
	return &principalRaw, nil
}

// FindUserMessage is the message for finding users.
type FindUserMessage struct {
	ID          *int
	Email       *string
	ShowDeleted bool
}

// UserMessage is the message for an user.
type UserMessage struct {
	ID            int
	Email         string
	Name          string
	Type          api.PrincipalType
	PasswordHash  string
	Role          api.Role
	MemberDeleted bool
}

// ListUsers list all users.
func (s *Store) ListUsers(ctx context.Context, find *FindUserMessage) ([]*UserMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	userMessages, err := s.listUserImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return userMessages, nil
}

// GetUserByID gets the user by ID.
func (s *Store) GetUserByID(ctx context.Context, id int) (*UserMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	userMessages, err := s.listUserImpl(ctx, tx, &FindUserMessage{ID: &id, ShowDeleted: true})
	if err != nil {
		return nil, err
	}
	if len(userMessages) == 0 {
		return nil, nil
	}
	if len(userMessages) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d users with id %q, expect 1", len(userMessages), id)}
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return userMessages[0], nil
}

// GetUserByEmail gets the user by email.
func (s *Store) GetUserByEmail(ctx context.Context, email string) (*UserMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	userMessages, err := s.listUserImpl(ctx, tx, &FindUserMessage{Email: &email, ShowDeleted: true})
	if err != nil {
		return nil, err
	}
	if len(userMessages) == 0 {
		return nil, nil
	}
	if len(userMessages) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d users with email %q, expect 1", len(userMessages), email)}
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return userMessages[0], nil
}

// GetUserByEmailV2 gets an instance of Principal.
func (*Store) listUserImpl(ctx context.Context, tx *Tx, find *FindUserMessage) ([]*UserMessage, error) {
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, fmt.Sprintf("principal.id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.Email; v != nil {
		where, args = append(where, fmt.Sprintf("principal.email = $%d", len(args)+1)), append(args, *v)
	}
	if !find.ShowDeleted {
		where, args = append(where, fmt.Sprintf("member.row_status = $%d", len(args)+1)), append(args, api.Normal)
	}

	var userMessages []*UserMessage
	rows, err := tx.QueryContext(ctx, `
		SELECT
			principal.id AS user_id,
			principal.name,
			principal.email,
			principal.password_hash,
			member.role,
			member.row_status AS row_status
		FROM principal
		LEFT JOIN member ON principal.id = member.principal_id
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()
	for rows.Next() {
		var userMessage UserMessage
		var rowStatus string
		if err := rows.Scan(
			&userMessage.ID,
			&userMessage.Name,
			&userMessage.Email,
			&userMessage.PasswordHash,
			&userMessage.Role,
			&rowStatus,
		); err != nil {
			return nil, FormatError(err)
		}
		userMessage.MemberDeleted = convertRowStatusToDeleted(rowStatus)
		userMessages = append(userMessages, &userMessage)
	}

	return userMessages, nil
}

// CreateUser creates an user.
func (s *Store) CreateUser(ctx context.Context, create *UserMessage, creatorID int) (*UserMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	var count int
	if err := tx.QueryRowContext(ctx,
		`SELECT COUNT(1) FROM principal WHERE type = $1`,
		api.EndUser,
	).Scan(&count); err != nil {
		return nil, err
	}
	role := api.Developer
	// Grant the member Owner role if there is no existing Owner member.
	if count == 0 {
		role = api.Owner
	}
	var userID int
	if err := tx.QueryRowContext(ctx, `
			INSERT INTO principal (
				creator_id,
				updater_id,
				type,
				name,
				email,
				password_hash
			)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id
		`,
		creatorID,
		creatorID,
		create.Type,
		create.Name,
		create.Email,
		create.PasswordHash,
	).Scan(&userID); err != nil {
		return nil, FormatError(err)
	}

	if _, err := tx.ExecContext(ctx, `
			INSERT INTO member (
				creator_id,
				updater_id,
				status,
				role,
				principal_id
			)
			VALUES ($1, $2, $3, $4, $5)
		`,
		creatorID,
		creatorID,
		api.Active,
		role,
		userID,
	); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return &UserMessage{
		ID:           userID,
		Email:        create.Email,
		Name:         create.Name,
		Type:         create.Type,
		PasswordHash: create.PasswordHash,
		Role:         role,
	}, nil
}
