package store

import (
	"context"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/qb"
)

// CountActiveInstances counts the number of instances.
func (s *Store) CountActiveInstances(ctx context.Context) (int, error) {
	q := qb.Q().Space("SELECT count(1) FROM instance WHERE instance.deleted = ?", false)

	query, args, err := q.ToSQL()
	if err != nil {
		return 0, errors.Wrapf(err, "failed to build sql")
	}

	var count int
	if err := s.GetDB().QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// CountActiveEndUsers counts the number of endusers.
func (s *Store) CountActiveEndUsers(ctx context.Context) (int, error) {
	q := qb.Q().Space("SELECT count(DISTINCT principal.id) FROM principal WHERE principal.deleted = ?", false)
	query, args, err := q.ToSQL()
	if err != nil {
		return 0, errors.Wrapf(err, "failed to build sql")
	}

	var count int
	if err := s.GetDB().QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}
