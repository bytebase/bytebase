package sqlite

import (
	"context"
	"strings"

	"github.com/bytebase/bytebase"
	"github.com/bytebase/bytebase/api"
)

var (
	_ api.PrincipalService = (*PrincipalService)(nil)
)

// PrincipalService represents a service for managing principal.
type PrincipalService struct {
	db *DB
}

// NewAuthService returns a new instance of AuthService.
func NewPrincipalService(db *DB) *PrincipalService {
	return &PrincipalService{db: db}
}

// FindPrincipalList retrieves a principal by email.
// Returns ENOTFOUND if no matching email.
func (s *PrincipalService) FindPrincipalByEmail(ctx context.Context, email string) (*api.Principal, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	list, err := findPrincipalList(ctx, tx, principalFilter{email: &email})
	if err != nil {
		return nil, err
	} else if len(list) == 0 {
		return nil, &bytebase.Error{Code: bytebase.ENOTFOUND, Message: "Principal not found."}
	}
	return list[0], nil
}

// FindPrincipalList retrieves a list of principals.
func (s *PrincipalService) FindPrincipalList(ctx context.Context) ([]*api.Principal, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	list, err := findPrincipalList(ctx, tx, principalFilter{})
	if err != nil {
		return []*api.Principal{}, err
	}

	return list, nil
}

// principalFilter represents a filter passed to findPrincipalList().
type principalFilter struct {
	// Filtering fields.
	ID    *int
	email *string `json:"email"`
}

func findPrincipalList(ctx context.Context, tx *Tx, filter principalFilter) (_ []*api.Principal, err error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := filter.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, *v)
	}
	if v := filter.email; v != nil {
		where, args = append(where, "email = ?"), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT 
		    id,
		    creator_id,
		    created_ts,
		    updater_id,
		    updated_ts,
		    status,
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
		if rows.Scan(
			&principal.ID,
			&principal.CreatorId,
			&principal.CreatorTs,
			&principal.UpdaterId,
			&principal.UpdatedTs,
			&principal.Status,
			&principal.Type,
			&principal.Name,
			&principal.Email,
			&principal.PasswordHash,
		); err != nil {
			return nil, err
		}

		list = append(list, &principal)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return list, nil
}
