package store

import (
	"context"
	"database/sql"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/qb"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// CountInstanceMessage is the message for counting instances.
type CountInstanceMessage struct {
	EnvironmentID *string
}

// CountUsers counts the principal.
func (s *Store) CountUsers(ctx context.Context, userType storepb.PrincipalType) (int, error) {
	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	count := 0

	q := qb.Q().Space("SELECT COUNT(*) FROM principal WHERE principal.type = ?", userType.String())
	query, args, err := q.ToSQL()
	if err != nil {
		return 0, errors.Wrapf(err, "failed to build sql")
	}

	if err := tx.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return count, nil
}

// CountInstance counts the number of instances.
func (s *Store) CountInstance(ctx context.Context, find *CountInstanceMessage) (int, error) {
	q := qb.Q().Space("SELECT count(1) FROM instance WHERE instance.deleted = ?", false)
	if v := find.EnvironmentID; v != nil {
		q = q.Space("AND instance.environment = ?", *v)
	}

	query, args, err := q.ToSQL()
	if err != nil {
		return 0, errors.Wrapf(err, "failed to build sql")
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var count int
	if err := tx.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return count, nil
}

// CountActiveUsers counts the number of endusers.
func (s *Store) CountActiveUsers(ctx context.Context) (int, error) {
	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	q := qb.Q().Space("SELECT count(DISTINCT principal.id) FROM principal WHERE principal.deleted = ? AND principal.type = ?", false, storepb.PrincipalType_END_USER.String())
	query, args, err := q.ToSQL()
	if err != nil {
		return 0, errors.Wrapf(err, "failed to build sql")
	}

	var count int
	if err := tx.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		if err == sql.ErrNoRows {
			return 0, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return count, nil
}
