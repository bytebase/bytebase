package store

import (
	"context"
	"strings"
)

// CountUsers counts the principal.
func (s *Store) CountUsers(ctx context.Context, find *FindUserMessage) (int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, FormatError(err)
	}
	defer tx.Rollback()

	where, args := getUserFindQuery(find)
	count := 0

	if err := tx.QueryRowContext(ctx, `
	SELECT COUNT(*)
	FROM principal
	LEFT JOIN member ON principal.id = member.principal_id
	LEFT JOIN idp ON principal.idp_id = idp.id
	WHERE `+strings.Join(where, " AND "),
		args...).Scan(&count); err != nil {
		return 0, FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return count, nil
}
