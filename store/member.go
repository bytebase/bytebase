package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
)

// memberRaw is the store model for an Member.
// Fields have exactly the same meanings as Member.
type memberRaw struct {
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

// ToMember creates an instance of Member based on the memberRaw.
// This is intended to be called when we need to compose an Member relationship.
func (raw *memberRaw) ToMember() *api.Member {
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

// CreateMember creates an instance of Member
func (s *Store) CreateMember(ctx context.Context, create *api.MemberCreate) (*api.Member, error) {
	memberRaw, err := s.createMemberRaw(ctx, create)
	if err != nil {
		return nil, fmt.Errorf("failed to create Member with MemberCreate[%+v], error[%w]", create, err)
	}
	member, err := s.composeMember(ctx, memberRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to compose Member with memberRaw[%+v], error[%w]", memberRaw, err)
	}
	return member, nil
}

// FindMember finds a list of Member instances
func (s *Store) FindMember(ctx context.Context, find *api.MemberFind) ([]*api.Member, error) {
	memberRawList, err := s.findMemberRaw(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("failed to find Member list, error[%w]", err)
	}
	var memberList []*api.Member
	for _, raw := range memberRawList {
		member, err := s.composeMember(ctx, raw)
		if err != nil {
			return nil, fmt.Errorf("failed to compose Member with memberRaw[%+v], error[%w]", raw, err)
		}
		memberList = append(memberList, member)
	}
	return memberList, nil
}

// GetMemberByPrincipalID gets an instance of Member
func (s *Store) GetMemberByPrincipalID(ctx context.Context, id int) (*api.Member, error) {
	find := &api.MemberFind{PrincipalID: &id}
	memberRaw, err := s.getMemberRaw(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("failed to get Member with PrincipalID[%d], error[%w]", id, err)
	}
	if memberRaw == nil {
		return nil, nil
	}
	member, err := s.composeMember(ctx, memberRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to compose Member with memberRaw[%+v], error[%w]", memberRaw, err)
	}
	return member, nil
}

// GetMemberByID gets an instance of Member
func (s *Store) GetMemberByID(ctx context.Context, id int) (*api.Member, error) {
	find := &api.MemberFind{ID: &id}
	memberRaw, err := s.getMemberRaw(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("failed to get Member with ID[%d], error[%w]", id, err)
	}
	if memberRaw == nil {
		return nil, nil
	}
	member, err := s.composeMember(ctx, memberRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to compose Member with memberRaw[%+v], error[%w]", memberRaw, err)
	}
	return member, nil
}

// PatchMember patches an instance of Member
func (s *Store) PatchMember(ctx context.Context, patch *api.MemberPatch) (*api.Member, error) {
	memberRaw, err := s.patchMemberRaw(ctx, patch)
	if err != nil {
		return nil, fmt.Errorf("failed to patch Member with MemberPatch[%+v], error[%w]", patch, err)
	}
	member, err := s.composeMember(ctx, memberRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to compose Member with memberRaw[%+v], error[%w]", memberRaw, err)
	}
	return member, nil
}

//
// private functions
//

// createMemberRaw creates a new member.
func (s *Store) createMemberRaw(ctx context.Context, create *api.MemberCreate) (*memberRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	member, err := createMemberImpl(ctx, tx.PTx, create)
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

// findMemberRaw retrieves a list of memberRaw instances.
func (s *Store) findMemberRaw(ctx context.Context, find *api.MemberFind) ([]*memberRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	memberRawList, err := findMemberImpl(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	for _, member := range memberRawList {
		if err := s.cache.UpsertCache(api.MemberCache, member.PrincipalID, member); err != nil {
			return nil, err
		}
	}

	return memberRawList, nil
}

// getMemberRaw retrieves an instance of memberRaw.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *Store) getMemberRaw(ctx context.Context, find *api.MemberFind) (*memberRaw, error) {
	if find.PrincipalID != nil {
		memberRaw := &memberRaw{}
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

	list, err := findMemberImpl(ctx, tx.PTx, find)
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

// patchMemberRaw updates an existing instance of memberRaw by ID.
// Returns ENOTFOUND if member does not exist.
func (s *Store) patchMemberRaw(ctx context.Context, patch *api.MemberPatch) (*memberRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	member, err := patchMemberImpl(ctx, tx.PTx, patch)
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

// composeMember composes an instance of Member by memberRaw
func (s *Store) composeMember(ctx context.Context, raw *memberRaw) (*api.Member, error) {
	member := raw.ToMember()

	creator, err := s.GetPrincipalByID(ctx, member.CreatorID)
	if err != nil {
		return nil, err
	}
	member.Creator = creator

	updater, err := s.GetPrincipalByID(ctx, member.UpdaterID)
	if err != nil {
		return nil, err
	}
	member.Updater = updater

	principal, err := s.GetPrincipalByID(ctx, member.PrincipalID)
	if err != nil {
		return nil, err
	}
	member.Principal = principal

	return member, nil
}

// createMemberImpl creates a new member.
func createMemberImpl(ctx context.Context, tx *sql.Tx, create *api.MemberCreate) (*memberRaw, error) {
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
	var memberRaw memberRaw
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

func findMemberImpl(ctx context.Context, tx *sql.Tx, find *api.MemberFind) ([]*memberRaw, error) {
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
	var memberRawList []*memberRaw
	for rows.Next() {
		var memberRaw memberRaw
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

// patchMemberImpl updates a member by ID. Returns the new state of the member after update.
func patchMemberImpl(ctx context.Context, tx *sql.Tx, patch *api.MemberPatch) (*memberRaw, error) {
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
		var memberRaw memberRaw
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
