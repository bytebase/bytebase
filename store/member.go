package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/metric"
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

// toMember creates an instance of Member based on the memberRaw.
// This is intended to be called when we need to compose an Member relationship.
func (raw *memberRaw) toMember() *api.Member {
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

// FindMember finds a list of Member instances.
func (s *Store) FindMember(ctx context.Context, find *api.MemberFind) ([]*api.Member, error) {
	memberRawList, err := s.findMemberRaw(ctx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find Member list with MemberFind[%+v]", find)
	}
	var memberList []*api.Member
	for _, raw := range memberRawList {
		member, err := s.composeMember(ctx, raw)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to compose Member with memberRaw[%+v]", raw)
		}
		memberList = append(memberList, member)
	}
	return memberList, nil
}

// GetMemberByPrincipalID gets an instance of Member.
func (s *Store) GetMemberByPrincipalID(ctx context.Context, id int) (*api.Member, error) {
	find := &api.MemberFind{PrincipalID: &id}
	memberRaw, err := s.getMemberRaw(ctx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get Member with PrincipalID %d", id)
	}
	if memberRaw == nil {
		return nil, nil
	}
	member, err := s.composeMember(ctx, memberRaw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose Member with memberRaw[%+v]", memberRaw)
	}
	return member, nil
}

// GetMemberByID gets an instance of Member.
func (s *Store) GetMemberByID(ctx context.Context, id int) (*api.Member, error) {
	find := &api.MemberFind{ID: &id}
	memberRaw, err := s.getMemberRaw(ctx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get Member with ID %d", id)
	}
	if memberRaw == nil {
		return nil, nil
	}
	member, err := s.composeMember(ctx, memberRaw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose Member with memberRaw[%+v]", memberRaw)
	}
	return member, nil
}

// PatchMember patches an instance of Member.
func (s *Store) PatchMember(ctx context.Context, patch *api.MemberPatch) (*api.Member, error) {
	memberRaw, err := s.patchMemberRaw(ctx, patch)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to patch Member with MemberPatch[%+v]", patch)
	}
	member, err := s.composeMember(ctx, memberRaw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose Member with memberRaw[%+v]", memberRaw)
	}

	// Invalidate the user cache for role update.
	s.userIDCache.Delete(memberRaw.PrincipalID)
	return member, nil
}

// CountMemberGroupByRoleAndStatus counts the number of member and group by role and status.
// Used by the metric collector.
func (s *Store) CountMemberGroupByRoleAndStatus(ctx context.Context) ([]*metric.MemberCountMetric, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, `
		SELECT role, status, row_status, COUNT(*)
		FROM member
		GROUP BY role, status, row_status`,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	var res []*metric.MemberCountMetric
	for rows.Next() {
		var metric metric.MemberCountMetric
		if err := rows.Scan(&metric.Role, &metric.Status, &metric.RowStatus, &metric.Count); err != nil {
			return nil, FormatError(err)
		}
		res = append(res, &metric)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}
	return res, nil
}

// private functions
//
// findMemberRaw retrieves a list of memberRaw instances.
func (s *Store) findMemberRaw(ctx context.Context, find *api.MemberFind) ([]*memberRaw, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	memberRawList, err := findMemberImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	for _, member := range memberRawList {
		if err := s.cache.UpsertCache(memberCacheNamespace, member.PrincipalID, member); err != nil {
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
		has, err := s.cache.FindCache(memberCacheNamespace, *find.PrincipalID, memberRaw)
		if err != nil {
			return nil, err
		}
		if has {
			return memberRaw, nil
		}
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := findMemberImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, nil
	} else if len(list) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d members with filter %+v, expect 1", len(list), find)}
	}
	if err := s.cache.UpsertCache(memberCacheNamespace, list[0].PrincipalID, list[0]); err != nil {
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
	defer tx.Rollback()

	member, err := patchMemberImpl(ctx, tx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	if err := s.cache.UpsertCache(memberCacheNamespace, member.PrincipalID, member); err != nil {
		return nil, err
	}

	return member, nil
}

// composeMember composes an instance of Member by memberRaw.
func (s *Store) composeMember(ctx context.Context, raw *memberRaw) (*api.Member, error) {
	member := raw.toMember()

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

func findMemberImpl(ctx context.Context, tx *Tx, find *api.MemberFind) ([]*memberRaw, error) {
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
func patchMemberImpl(ctx context.Context, tx *Tx, patch *api.MemberPatch) (*memberRaw, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = $1"}, []interface{}{patch.UpdaterID}
	if v := patch.RowStatus; v != nil {
		set, args = append(set, fmt.Sprintf("row_status = $%d", len(args)+1)), append(args, api.RowStatus(*v))
	}
	if v := patch.Role; v != nil {
		set, args = append(set, fmt.Sprintf("role = $%d", len(args)+1)), append(args, api.Role(*v))
	}

	args = append(args, patch.ID)

	var memberRaw memberRaw
	// Execute update query with RETURNING.
	if err := tx.QueryRowContext(ctx, fmt.Sprintf(`
		UPDATE member
		SET `+strings.Join(set, ", ")+`
		WHERE id = $%d
		RETURNING id, row_status, creator_id, created_ts, updater_id, updated_ts, status, role, principal_id
	`, len(args)),
		args...,
	).Scan(
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
		if err == sql.ErrNoRows {
			return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("member ID not found: %d", patch.ID)}
		}
		return nil, FormatError(err)
	}
	return &memberRaw, nil
}
