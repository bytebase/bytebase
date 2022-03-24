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
	_ MemberStore = (*MemberStoreImpl)(nil)
)

// MemberRaw is the store model for an Member.
// Fields have exactly the same meanings as Member.
type MemberRaw struct {
	ID int

	// Standard fields
	RowStatus api.RowStatus
	CreatorID int
	CreatedTs int64
	UpdaterID int
	UpdatedTs int64

	// Domain specific fields
	Status      api.MemberStatus
	Role        api.Role
	PrincipalID int
}

// ToMember creates an instance of Member based on the MemberRaw.
// This is intended to be called when we need to compose an Member relationship.
func (raw *MemberRaw) ToMember() *api.Member {
	return &api.Member{
		ID: raw.ID,

		// Standard fields
		RowStatus: raw.RowStatus,
		CreatorID: raw.CreatorID,
		CreatedTs: raw.CreatedTs,
		UpdaterID: raw.UpdaterID,
		UpdatedTs: raw.UpdatedTs,

		// Domain specific fields
		Status:      raw.Status,
		Role:        raw.Role,
		PrincipalID: raw.PrincipalID,
	}
}

// MemberStore is the service for members.
type MemberStore interface {
	Create(ctx context.Context, create *api.MemberCreate) (*MemberRaw, error)
	FindList(ctx context.Context, find *api.MemberFind) ([]*MemberRaw, error)
	Find(ctx context.Context, find *api.MemberFind) (*MemberRaw, error)
	Patch(ctx context.Context, patch *api.MemberPatch) (*MemberRaw, error)

	ComposeRelationship(ctx context.Context, raw *MemberRaw) (*api.Member, error)
}

// MemberStoreImpl represents a service for managing member.
type MemberStoreImpl struct {
	l     *zap.Logger
	db    *DB
	cache api.CacheService
	store *Store
}

// NewMemberStore returns a new instance of MemberService.
func NewMemberStore(logger *zap.Logger, db *DB, cache api.CacheService, store *Store) *MemberStoreImpl {
	return &MemberStoreImpl{
		l:     logger,
		db:    db,
		cache: cache,
		store: store,
	}
}

// Create creates a new member.
func (s *MemberStoreImpl) Create(ctx context.Context, create *api.MemberCreate) (*MemberRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	member, err := createMember(ctx, tx.PTx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	if err := s.cache.UpsertCache(api.MemberCache, member.PrincipalID, member); err != nil {
		return nil, err
	}

	return member, nil
}

// FindList retrieves a list of members based on find.
func (s *MemberStoreImpl) FindList(ctx context.Context, find *api.MemberFind) ([]*MemberRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := findMemberList(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	if err == nil {
		for _, member := range list {
			if err := s.cache.UpsertCache(api.MemberCache, member.PrincipalID, member); err != nil {
				return nil, err
			}
		}
	}

	return list, nil
}

// Find retrieves a single member based on find.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *MemberStoreImpl) Find(ctx context.Context, find *api.MemberFind) (*MemberRaw, error) {
	if find.PrincipalID != nil {
		memberRaw := &MemberRaw{}
		has, err := s.cache.FindCache(api.MemberCache, *find.PrincipalID, memberRaw)
		if err != nil {
			return nil, err
		}
		if has {
			return memberRaw, nil
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := findMemberList(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, nil
	} else if len(list) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: fmt.Errorf("found %d members with filter %+v, expect 1", len(list), find)}
	}
	if err := s.cache.UpsertCache(api.MemberCache, list[0].PrincipalID, list[0]); err != nil {
		return nil, err
	}
	return list[0], nil
}

// Patch updates an existing member by ID.
// Returns ENOTFOUND if member does not exist.
func (s *MemberStoreImpl) Patch(ctx context.Context, patch *api.MemberPatch) (*MemberRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	member, err := patchMember(ctx, tx.PTx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	if err := s.cache.UpsertCache(api.MemberCache, member.PrincipalID, member); err != nil {
		return nil, err
	}

	return member, nil
}

// ComposeRelationship composes an instance of Member by MemberRaw
func (s *MemberStoreImpl) ComposeRelationship(ctx context.Context, raw *MemberRaw) (*api.Member, error) {
	member := raw.ToMember()

	creator, err := s.store.Principal.ComposeByID(ctx, member.CreatorID)
	if err != nil {
		return nil, err
	}
	member.Creator = creator

	updater, err := s.store.Principal.ComposeByID(ctx, member.UpdaterID)
	if err != nil {
		return nil, err
	}
	member.Updater = updater

	principal, err := s.store.Principal.ComposeByID(ctx, member.PrincipalID)
	if err != nil {
		return nil, err
	}
	member.Principal = principal

	return member, nil
}

// private functions

// createMember creates a new member.
func createMember(ctx context.Context, tx *sql.Tx, create *api.MemberCreate) (*MemberRaw, error) {
	// Insert row into database.
	row, err := tx.QueryContext(ctx, `
		INSERT INTO member (
			creator_id,
			updater_id,
			status,
			role,
			principal_id
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, row_status, creator_id, created_ts, updater_id, updated_ts, status, role, principal_id
	`,
		create.CreatorID,
		create.CreatorID,
		create.Status,
		create.Role,
		create.PrincipalID,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var memberRaw MemberRaw
	if err := row.Scan(
		&memberRaw.ID,
		&memberRaw.RowStatus,
		&memberRaw.CreatorID,
		&memberRaw.CreatedTs,
		&memberRaw.UpdaterID,
		&memberRaw.UpdatedTs,
		&memberRaw.Status,
		&memberRaw.Role,
		&memberRaw.PrincipalID,
	); err != nil {
		return nil, FormatError(err)
	}

	return &memberRaw, nil
}

func findMemberList(ctx context.Context, tx *sql.Tx, find *api.MemberFind) ([]*MemberRaw, error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.PrincipalID; v != nil {
		where, args = append(where, fmt.Sprintf("principal_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.Role; v != nil {
		where, args = append(where, fmt.Sprintf("role = $%d", len(args)+1)), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			row_status,
			creator_id,
			created_ts,
			updater_id,
			updated_ts,
			status,
			role,
			principal_id
		FROM member
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into memberRawList.
	var memberRawList []*MemberRaw
	for rows.Next() {
		var memberRaw MemberRaw
		if err := rows.Scan(
			&memberRaw.ID,
			&memberRaw.RowStatus,
			&memberRaw.CreatorID,
			&memberRaw.CreatedTs,
			&memberRaw.UpdaterID,
			&memberRaw.UpdatedTs,
			&memberRaw.Status,
			&memberRaw.Role,
			&memberRaw.PrincipalID,
		); err != nil {
			return nil, FormatError(err)
		}

		memberRawList = append(memberRawList, &memberRaw)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return memberRawList, nil
}

// patchMember updates a member by ID. Returns the new state of the member after update.
func patchMember(ctx context.Context, tx *sql.Tx, patch *api.MemberPatch) (*MemberRaw, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = $1"}, []interface{}{patch.UpdaterID}
	if v := patch.RowStatus; v != nil {
		set, args = append(set, fmt.Sprintf("row_status = $%d", len(args)+1)), append(args, api.RowStatus(*v))
	}
	if v := patch.Role; v != nil {
		set, args = append(set, fmt.Sprintf("role = $%d", len(args)+1)), append(args, api.Role(*v))
	}

	args = append(args, patch.ID)

	// Execute update query with RETURNING.
	row, err := tx.QueryContext(ctx, fmt.Sprintf(`
		UPDATE member
		SET `+strings.Join(set, ", ")+`
		WHERE id = $%d
		RETURNING id, row_status, creator_id, created_ts, updater_id, updated_ts, status, role, principal_id
	`, len(args)),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	if row.Next() {
		var memberRaw MemberRaw
		if err := row.Scan(
			&memberRaw.ID,
			&memberRaw.RowStatus,
			&memberRaw.CreatorID,
			&memberRaw.CreatedTs,
			&memberRaw.UpdaterID,
			&memberRaw.UpdatedTs,
			&memberRaw.Status,
			&memberRaw.Role,
			&memberRaw.PrincipalID,
		); err != nil {
			return nil, FormatError(err)
		}

		return &memberRaw, nil
	}

	return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("member ID not found: %d", patch.ID)}
}
