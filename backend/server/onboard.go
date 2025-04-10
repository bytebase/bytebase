package server

import (
	"context"
	"strconv"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/base"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/resources/postgres"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// generateOnboardingData generates onboarding data after the first signup.
func (s *Server) generateOnboardingData(ctx context.Context, user *store.UserMessage) error {
	userID := user.ID
	setting := &storepb.Project{
		AllowModifyStatement: true,
		AutoResolveIssue:     true,
	}
	project, err := s.store.CreateProjectV2(ctx, &store.ProjectMessage{
		ResourceID: "project-sample",
		Title:      "Sample Project",
		Setting:    setting,
	}, userID)
	if err != nil {
		return errors.Wrapf(err, "failed to create onboarding project")
	}

	// Test Sample Instance
	testInstance, err := s.store.CreateInstanceV2(ctx, &store.InstanceMessage{
		ResourceID:    "test-sample-instance",
		EnvironmentID: base.DefaultTestEnvironmentID,
		Metadata: &storepb.Instance{
			Title:        "Test Sample Instance",
			Engine:       storepb.Engine_POSTGRES,
			ExternalLink: "",
			Activation:   false,
			DataSources: []*storepb.DataSource{
				{
					Id:       "admin",
					Type:     storepb.DataSourceType_ADMIN,
					Username: postgres.SampleUser,
					Password: "",
					Host:     common.GetPostgresSocketDir(),
					Port:     strconv.Itoa(s.profile.SampleDatabasePort),
					Database: postgres.SampleDatabaseTest,
				},
			},
		},
	})
	if err != nil {
		return errors.Wrapf(err, "failed to create test onboarding instance")
	}

	// Sync the instance schema so we can transfer the sample database later.
	if _, _, _, err := s.schemaSyncer.SyncInstance(ctx, testInstance); err != nil {
		return errors.Wrapf(err, "failed to sync test onboarding instance")
	}

	// Transfer sample database to the just created project.
	transferDatabaseMessage := &store.UpdateDatabaseMessage{
		InstanceID:   testInstance.ResourceID,
		DatabaseName: postgres.SampleDatabaseTest,
		ProjectID:    &project.ResourceID,
	}
	_, err = s.store.UpdateDatabase(ctx, transferDatabaseMessage)
	if err != nil {
		return errors.Wrapf(err, "failed to transfer test sample database")
	}

	dbName := postgres.SampleDatabaseTest
	testDatabase, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		InstanceID:      &testInstance.ResourceID,
		DatabaseName:    &dbName,
		IsCaseSensitive: store.IsObjectCaseSensitive(testInstance),
	})
	if err != nil {
		return errors.Wrapf(err, "failed to find test onboarding instance")
	}
	if testDatabase == nil {
		return errors.Errorf("database %q not found", dbName)
	}

	// Need to sync database schema so we can configure sensitive data policy and create the schema
	// update issue later.
	if err := s.schemaSyncer.SyncDatabaseSchema(ctx, testDatabase); err != nil {
		return errors.Wrapf(err, "failed to sync test sample database schema")
	}

	// Prod Sample Instance
	prodInstance, err := s.store.CreateInstanceV2(ctx, &store.InstanceMessage{
		ResourceID:    "prod-sample-instance",
		EnvironmentID: base.DefaultProdEnvironmentID,
		Metadata: &storepb.Instance{
			Title:        "Prod Sample Instance",
			Engine:       storepb.Engine_POSTGRES,
			ExternalLink: "",
			Activation:   false,
			DataSources: []*storepb.DataSource{
				{
					Id:       "admin",
					Type:     storepb.DataSourceType_ADMIN,
					Username: postgres.SampleUser,
					Password: "",
					Host:     common.GetPostgresSocketDir(),
					Port:     strconv.Itoa(s.profile.SampleDatabasePort + 1),
					Database: postgres.SampleDatabaseProd,
				},
			},
		},
	})
	if err != nil {
		return errors.Wrapf(err, "failed to create prod onboarding instance")
	}

	// Sync the instance schema so we can transfer the sample database later.
	if _, _, _, err := s.schemaSyncer.SyncInstance(ctx, prodInstance); err != nil {
		return errors.Wrapf(err, "failed to sync prod onboarding instance")
	}

	// Transfer sample database to the just created project.
	transferDatabaseMessage = &store.UpdateDatabaseMessage{
		InstanceID:   prodInstance.ResourceID,
		DatabaseName: postgres.SampleDatabaseProd,
		ProjectID:    &project.ResourceID,
	}
	_, err = s.store.UpdateDatabase(ctx, transferDatabaseMessage)
	if err != nil {
		return errors.Wrapf(err, "failed to transfer prod sample database")
	}

	dbName = postgres.SampleDatabaseProd
	prodDatabase, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		InstanceID:      &prodInstance.ResourceID,
		DatabaseName:    &dbName,
		IsCaseSensitive: store.IsObjectCaseSensitive(prodInstance),
	})
	if err != nil {
		return errors.Wrapf(err, "failed to find prod onboarding instance")
	}
	if prodDatabase == nil {
		return errors.Errorf("database %q not found", dbName)
	}

	// Need to sync database schema so we can configure sensitive data policy and create the schema
	// update issue later.
	if err := s.schemaSyncer.SyncDatabaseSchema(ctx, prodDatabase); err != nil {
		return errors.Wrapf(err, "failed to sync prod sample database schema")
	}
	return nil
}
