package store

import (
	"context"
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

// FindMemberList retrieves a list of members.
func (s *MemberService) FindMemberList(ctx context.Context, workspaceId int) ([]*api.Member, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := findMemberList(ctx, tx, memberFilter{Workspace_ID: &workspaceId})
	if err != nil {
		return []*api.Member{}, err
	}

	return list, nil
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
func (s *MemberService) DeleteMemberByID(ctx context.Context, id int) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FormatError(err)
	}
	defer tx.Rollback()

	err = deleteMember(ctx, tx, id)
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

// findMemberByID retrieves a member by id.
// Returns ENOTFOUND if no matching id.
func findMemberByID(ctx context.Context, tx *Tx, id int) (*api.Member, error) {
	list, err := findMemberList(ctx, tx, memberFilter{ID: &id})
	if err != nil {
		return nil, err
	} else if len(list) == 0 {
		return nil, &bytebase.Error{Code: bytebase.ENOTFOUND, Message: "member not found."}
	}
	return list[0], nil
}

// memberFilter represents a filter passed to findMemberList().
type memberFilter struct {
	// Filtering fields.
	ID           *int
	Workspace_ID *int
}

func findMemberList(ctx context.Context, tx *Tx, filter memberFilter) (_ []*api.Member, err error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := filter.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, *v)
	}
	if v := filter.Workspace_ID; v != nil {
		where, args = append(where, "workspace_id = ?"), append(args, *v)
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

	return nil, &bytebase.Error{Code: bytebase.ENOTFOUND, Message: "member not found."}
}

// deleteMember permanently deletes a member by ID.
func deleteMember(ctx context.Context, tx *Tx, id int) error {
	// Verify object exists & the current user is the owner.
	if _, err := findMemberByID(ctx, tx, id); err != nil {
		return err
	}

	// Remove row from database.
	if _, err := tx.ExecContext(ctx, `DELETE FROM member WHERE id = ?`, id); err != nil {
		return FormatError(err)
	}
	return nil
}
