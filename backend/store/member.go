package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase"
	"github.com/bytebase/bytebase/api"
)

var (
	_ api.MemberService = (*MemberService)(nil)
)

// MemberService represents a service for managing member.
type MemberService struct {
	l  *bytebase.Logger
	db *DB
}

// NewMemberService returns a new instance of MemberService.
func NewMemberService(logger *bytebase.Logger, db *DB) *MemberService {
	return &MemberService{l: logger, db: db}
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

	return member, nil
}

// FindMemberList retrieves a list of members based on filter.
func (s *MemberService) FindMemberList(ctx context.Context, filter *api.MemberFilter) ([]*api.Member, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := findMemberList(ctx, tx, filter)
	if err != nil {
		return []*api.Member{}, err
	}

	return list, nil
}

// FindMember retrieves a single member based on filter.
// Returns ENOTFOUND if no matching record.
// Returns the first matching one and prints a warning if finding more than 1 matching records.
func (s *MemberService) FindMember(ctx context.Context, filter *api.MemberFilter) (*api.Member, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := findMemberList(ctx, tx, filter)
	if err != nil {
		return nil, err
	} else if len(list) == 0 {
		return nil, &bytebase.Error{Code: bytebase.ENOTFOUND, Message: fmt.Sprintf("member not found: %v", filter)}
	} else if len(list) > 1 {
		s.l.Logf(bytebase.WARN, "found mulitple members: %d, expect 1", len(list))
	}
	return list[0], nil
}

// PatchMemberByID updates an existing member by ID.
// Returns ENOTFOUND if member does not exist.
func (s *MemberService) PatchMemberByID(ctx context.Context, id int, patch *api.MemberPatch) (*api.Member, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	member, err := patchMember(ctx, tx, id, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return member, nil
}

// DeleteMemberByID deletes an existing member by ID.
// Returns ENOTFOUND if member does not exist.
func (s *MemberService) DeleteMemberByID(ctx context.Context, id int, delete *api.MemberDelete) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FormatError(err)
	}
	defer tx.Rollback()

	err = deleteMember(ctx, tx, id, delete)
	if err != nil {
		return FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return FormatError(err)
	}

	return nil
}

// createMember creates a new member.
func createMember(ctx context.Context, tx *Tx, create *api.MemberCreate) (*api.Member, error) {
	// Insert row into database.
	row, err := tx.QueryContext(ctx, `
		INSERT INTO member (
			workspace_id,
			creator_id,
			updater_id,
			role,
			principal_id
		)
		VALUES (?, ?, ?, ?, ?)
		RETURNING id, workspace_id, creator_id, created_ts, updater_id, updated_ts, role, principal_id
	`,
		create.WorkspaceId,
		create.CreatorId,
		create.CreatorId,
		create.Role,
		create.PrincipalId,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var member api.Member
	if err := row.Scan(
		&member.ID,
		&member.WorkspaceId,
		&member.CreatorId,
		&member.CreatedTs,
		&member.UpdaterId,
		&member.UpdatedTs,
		&member.Role,
		&member.PrincipalId,
	); err != nil {
		return nil, FormatError(err)
	}

	return &member, nil
}

func findMemberList(ctx context.Context, tx *Tx, filter *api.MemberFilter) (_ []*api.Member, err error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := filter.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, *v)
	}
	if v := filter.WorkspaceId; v != nil {
		where, args = append(where, "workspace_id = ?"), append(args, *v)
	}
	if v := filter.PrincipalId; v != nil {
		where, args = append(where, "principal_id = ?"), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT 
		    id,
			workspace_id,
		    creator_id,
		    created_ts,
		    updater_id,
		    updated_ts,
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
			&member.WorkspaceId,
			&member.CreatorId,
			&member.CreatedTs,
			&member.UpdaterId,
			&member.UpdatedTs,
			&member.Role,
			&member.PrincipalId,
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
func patchMember(ctx context.Context, tx *Tx, id int, patch *api.MemberPatch) (*api.Member, error) {
	member := api.Member{}
	// Update fields, if set.
	if v := patch.Role; v != nil {
		member.Role = api.Role(*v)
	}

	// Execute update query with RETURNING.
	row, err := tx.QueryContext(ctx, `
		UPDATE member
		SET role = ?, updater_id = ?
		WHERE id = ?
		RETURNING id, workspace_id, creator_id, created_ts, updater_id, updated_ts, role, principal_id
	`,
		member.Role,
		patch.UpdaterId,
		id,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	if row.Next() {
		var member api.Member
		if err := row.Scan(
			&member.ID,
			&member.WorkspaceId,
			&member.CreatorId,
			&member.CreatedTs,
			&member.UpdaterId,
			&member.UpdatedTs,
			&member.Role,
			&member.PrincipalId,
		); err != nil {
			return nil, FormatError(err)
		}

		return &member, nil
	}

	return nil, &bytebase.Error{Code: bytebase.ENOTFOUND, Message: fmt.Sprintf("member ID not found: %d", id)}
}

// deleteMember permanently deletes a member by ID.
func deleteMember(ctx context.Context, tx *Tx, id int, delete *api.MemberDelete) error {
	// Remove row from database.
	result, err := tx.ExecContext(ctx, `DELETE FROM member WHERE id = ?`, id)
	if err != nil {
		return FormatError(err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return &bytebase.Error{Code: bytebase.ENOTFOUND, Message: fmt.Sprintf("member ID not found: %d", id)}
	}

	return nil
}
