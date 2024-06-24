package store

import (
	"context"

	"github.com/pkg/errors"

	api "github.com/bytebase/bytebase/backend/legacyapi"
)

type DatabaseReviewConfig interface {
	GetResourceType() api.PolicyResourceType
	GetResourceUID(ctx context.Context, store *Store, database *DatabaseMessage) (int, error)
}

type databaseEnvironmentReviewConfigResource struct{}

func (*databaseEnvironmentReviewConfigResource) GetResourceType() api.PolicyResourceType {
	return api.PolicyResourceTypeEnvironment
}

func (*databaseEnvironmentReviewConfigResource) GetResourceUID(ctx context.Context, store *Store, database *DatabaseMessage) (int, error) {
	environment, err := store.GetEnvironmentV2(ctx, &FindEnvironmentMessage{ResourceID: &database.EffectiveEnvironmentID})
	if err != nil {
		return 0, err
	}
	if environment == nil {
		return 0, errors.Errorf("environment %q not found", database.EffectiveEnvironmentID)
	}

	return environment.UID, nil
}

type databaseProjectReviewConfigResource struct{}

func (*databaseProjectReviewConfigResource) GetResourceType() api.PolicyResourceType {
	return api.PolicyResourceTypeProject
}

func (*databaseProjectReviewConfigResource) GetResourceUID(ctx context.Context, store *Store, database *DatabaseMessage) (int, error) {
	project, err := store.GetProjectV2(ctx, &FindProjectMessage{ResourceID: &database.ProjectID})
	if err != nil {
		return 0, err
	}
	if project == nil {
		return 0, errors.Errorf("project %q not found", database.ProjectID)
	}

	return project.UID, nil
}

type databaseReviewConfigResource struct{}

func (*databaseReviewConfigResource) GetResourceType() api.PolicyResourceType {
	return api.PolicyResourceTypeDatabase
}

func (*databaseReviewConfigResource) GetResourceUID(_ context.Context, _ *Store, database *DatabaseMessage) (int, error) {
	return database.UID, nil
}
