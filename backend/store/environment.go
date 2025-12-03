package store

import (
	"context"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/qb"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func (s *Store) GetEnvironmentByID(ctx context.Context, id string) (*storepb.EnvironmentSetting_Environment, error) {
	environments, err := s.GetEnvironment(ctx)
	if err != nil {
		return nil, err
	}
	for _, environment := range environments.GetEnvironments() {
		if environment.Id == id {
			return environment, nil
		}
	}
	return nil, nil
}

func (s *Store) CheckDatabaseUseEnvironment(ctx context.Context, id string) (bool, error) {
	var exists bool

	q := qb.Q().Space(`
		SELECT EXISTS (
			SELECT 1
			FROM db
			WHERE db.environment = ?
		)
	`, id)

	sql, args, err := q.ToSQL()
	if err != nil {
		return false, errors.Wrapf(err, "failed to build sql")
	}

	if err := s.GetDB().QueryRowContext(ctx, sql, args...).Scan(&exists); err != nil {
		return false, errors.Wrapf(err, "failed to check if databases uses environment %q", id)
	}

	return exists, nil
}
