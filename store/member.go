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
	_ api.MemberService = (*MemberService)(nil)
)

// MemberService represents a service for managing member.
type MemberService struct {
	l  *zap.Logger
	db *DB

	cache api.CacheService
}

// NewMemberService returns a new instance of MemberService.
func NewMemberService(logger *zap.Logger, db *DB, cache api.CacheService) *MemberService {
	return &MemberService{l: logger, db: db, cache: cache}
}

// CreateMember creates a new member.
func (s *MemberService) CreateMember(ctx context.Context, create *api.MemberCreate) (*api.Member, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	member, err := createMember(ctx, tx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	if err := s.cache.UpsertCache(api.MemberCache, member.PrincipalID, member); err != nil {
		return nil, err
	}

	return member, nil
}

// FindMemberList retrieves a list of members based on find.
func (s *MemberService) FindMemberList(ctx context.Context, find *api.MemberFind) ([]*api.Member, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := findMemberList(ctx, tx, find)
	if err != nil {
		return []*api.Member{}, err
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

// FindMember retrieves a single member based on find.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *MemberService) FindMember(ctx context.Context, find *api.MemberFind) (*api.Member, error) {
	if find.PrincipalID != nil {
		member := &api.Member{}
		has, err := s.cache.FindCache(api.MemberCache, *find.PrincipalID, member)
		if err != nil {
			return nil, err
		}
		if has {
			return member, nil
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := findMemberList(ctx, tx, find)
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

// PatchMember updates an existing member by ID.
// Returns ENOTFOUND if member does not exist.
func (s *MemberService) PatchMember(ctx context.Context, patch *api.MemberPatch) (*api.Member, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	member, err := patchMember(ctx, tx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	if err := s.cache.UpsertCache(api.MemberCache, member.PrincipalID, member); err != nil {
		return nil, err
	}

	return member, nil
}

// createMember creates a new member.
func createMember(ctx context.Context, tx *Tx, create *api.MemberCreate) (*api.Member, error) {
	// Insert row into database.
	row, err := tx.QueryContext(ctx, `
		INSERT INTO member (
			creator_id,
			updater_id,
			`+"`status`,"+`
			role,
			principal_id
		)
		VALUES (?, ?, ?, ?, ?)
		RETURNING id, row_status, creator_id, created_ts, updater_id, updated_ts, `+"`status`, role, principal_id"+`
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
	var member api.Member
	if err := row.Scan(
		&member.ID,
		&member.RowStatus,
		&member.CreatorID,
		&member.CreatedTs,
		&member.UpdaterID,
		&member.UpdatedTs,
		&member.Status,
		&member.Role,
		&member.PrincipalID,
	); err != nil {
		return nil, FormatError(err)
	}

	return &member, nil
}

func findMemberList(ctx context.Context, tx *Tx, find *api.MemberFind) (_ []*api.Member, err error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, *v)
	}
	if v := find.PrincipalID; v != nil {
		where, args = append(where, "principal_id = ?"), append(args, *v)
	}
	if v := find.Role; v != nil {
		where, args = append(where, "role = ?"), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT
		    id,
			row_status,
		    creator_id,
		    created_ts,
		    updater_id,
		    updated_ts,
			`+"`status`,"+`
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

	// Iterate over result set and deserialize rows into list.
	list := make([]*api.Member, 0)
	for rows.Next() {
		var member api.Member
		if err := rows.Scan(
			&member.ID,
			&member.RowStatus,
			&member.CreatorID,
			&member.CreatedTs,
			&member.UpdaterID,
			&member.UpdatedTs,
			&member.Status,
			&member.Role,
			&member.PrincipalID,
		); err != nil {
			return nil, FormatError(err)
		}

		list = append(list, &member)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return list, nil
}

// patchMember updates a member by ID. Returns the new state of the member after update.
func patchMember(ctx context.Context, tx *Tx, patch *api.MemberPatch) (*api.Member, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = ?"}, []interface{}{patch.UpdaterID}
	if v := patch.RowStatus; v != nil {
		set, args = append(set, "row_status = ?"), append(args, api.RowStatus(*v))
	}
	if v := patch.Role; v != nil {
		set, args = append(set, "role = ?"), append(args, api.Role(*v))
	}

	args = append(args, patch.ID)

	// Execute update query with RETURNING.
	row, err := tx.QueryContext(ctx, `
		UPDATE member
		SET `+strings.Join(set, ", ")+`
		WHERE id = ?
		RETURNING id, row_status, creator_id, created_ts, updater_id, updated_ts, `+"`status`, role, principal_id"+`
	`,
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	if row.Next() {
		var member api.Member
		if err := row.Scan(
			&member.ID,
			&member.RowStatus,
			&member.CreatorID,
			&member.CreatedTs,
			&member.UpdaterID,
			&member.UpdatedTs,
			&member.Status,
			&member.Role,
			&member.PrincipalID,
		); err != nil {
			return nil, FormatError(err)
		}

		return &member, nil
	}

	return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("member ID not found: %d", patch.ID)}
}
