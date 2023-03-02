package store

import (
	"context"

	api "github.com/bytebase/bytebase/backend/legacyapi"
)

// CountUsers counts the principal.
func (s *Store) CountUsers(ctx context.Context, userType api.PrincipalType) (int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, FormatError(err)
	}
	defer tx.Rollback()

	count := 0

	if err := tx.QueryRowContext(ctx, `
	SELECT COUNT(*)
	FROM principal
	WHERE principal.type = $1`,
		userType).Scan(&count); err != nil {
		return 0, FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return count, nil
}
