package store

import (
	"context"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func (s *Store) GetEnvironmentByID(ctx context.Context, id string) (*storepb.EnvironmentSetting_Environment, error) {
	environments, err := s.GetEnvironmentSetting(ctx)
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

	if err := s.GetDB().QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM db 
			WHERE db.environment = $1
		)
	`, id).Scan(&exists); err != nil {
		return false, errors.Wrapf(err, "failed to check if databases uses environment %q", id)
	}

	return exists, nil
}
