// Package sampleinstance manages sample database instances.
package sampleinstance

import (
	"context"
	"log/slog"
	"strconv"
	"sync"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/resources/postgres"
	"github.com/bytebase/bytebase/backend/runner/schemasync"
	"github.com/bytebase/bytebase/backend/store"
)

// Manager manages sample database instances lifecycle.
type Manager struct {
	mu       sync.Mutex
	store    *store.Store
	profile  *config.Profile
	stoppers []func()
	port     int // The base port for sample instances
}

// NewManager creates a new sample instance manager.
func NewManager(store *store.Store, profile *config.Profile) *Manager {
	// Using profile.Port + 3 as our sample database port.
	// The actual startup of sample instances will be determined by checking the instance table.
	port := profile.Port + 3

	return &Manager{
		store:   store,
		profile: profile,
		port:    port,
	}
}

// StartIfExist starts sample instances if they exist in the database.
func (m *Manager) StartIfExist(ctx context.Context) error {
	// Check if sample instances exist in the database
	hasSampleInstances, err := m.store.HasSampleInstances(ctx)
	if err != nil {
		slog.Warn("failed to check for sample instances", log.BBError(err))
		return nil // Non-fatal error
	}

	if !hasSampleInstances {
		slog.Info("no sample instances found in database, skipping sample instance startup")
		return nil
	}

	// Start all sample instances
	slog.Info("starting sample instances")
	stoppers := postgres.StartAllSampleInstances(ctx, m.profile.DataDir, m.port)

	m.mu.Lock()
	defer m.mu.Unlock()

	m.stoppers = stoppers

	return nil
}

// HandleInstanceDeletion handles the deletion of a sample instance.
// We stop all sample instances only when both have been deleted.
func (m *Manager) HandleInstanceDeletion(ctx context.Context, instanceID string) error {
	if instanceID != "test-sample-instance" && instanceID != "prod-sample-instance" {
		// Not a sample instance
		return nil
	}

	// Check if any sample instances still exist after this deletion
	hasSampleInstances, err := m.store.HasSampleInstances(ctx)
	if err != nil {
		slog.Warn("failed to check for remaining sample instances", log.BBError(err))
		return nil // Non-fatal error
	}

	if hasSampleInstances {
		// At least one sample instance still exists, don't stop
		slog.Info("sample instance deleted but others remain, keeping processes running", slog.String("instance", instanceID))
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.stoppers) == 0 {
		// No sample instances running
		return nil
	}

	// Stop all sample instances when both have been deleted
	slog.Info("stopping all sample instances as none remain in database", slog.String("instance", instanceID))
	for _, stopper := range m.stoppers {
		if stopper != nil {
			stopper()
		}
	}
	m.stoppers = nil

	return nil
}

// HandleInstanceCreation handles the creation of a sample instance.
// When any sample instance is created, we start all sample instances.
func (m *Manager) HandleInstanceCreation(ctx context.Context, instanceID string) error {
	if instanceID != "test-sample-instance" && instanceID != "prod-sample-instance" {
		// Not a sample instance
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already running
	if len(m.stoppers) > 0 {
		slog.Info("sample instances already running", slog.String("instance", instanceID))
		return nil
	}

	// Start all sample instances when any one is created
	slog.Info("starting all sample instances due to creation", slog.String("instance", instanceID))
	stoppers := postgres.StartAllSampleInstances(ctx, m.profile.DataDir, m.port)
	m.stoppers = stoppers

	return nil
}

// Stop stops all sample instances.
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, stopper := range m.stoppers {
		if stopper != nil {
			stopper()
		}
	}
	m.stoppers = nil
}

// Port returns the base port for sample instances.
func (m *Manager) Port() int {
	return m.port
}

// GenerateOnboardingData generates onboarding data including project and instance.
func (m *Manager) GenerateOnboardingData(ctx context.Context, user *store.UserMessage, schemaSyncer *schemasync.Syncer) error {
	projectID := "project-sample"
	project, err := m.store.GetProject(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to find onboarding project %v", projectID)
	}
	if project == nil {
		sampleProject, err := m.store.CreateProject(ctx, &store.ProjectMessage{
			ResourceID: "project-sample",
			Title:      "Sample Project",
			Setting:    &storepb.Project{},
		}, user)
		if err != nil {
			return errors.Wrapf(err, "failed to create onboarding project")
		}
		project = sampleProject
	}

	testEnvID := common.DefaultTestEnvironmentID
	prodEnvID := common.DefaultProdEnvironmentID
	instanceMessages := []*store.InstanceMessage{
		{
			ResourceID:    "test-sample-instance",
			EnvironmentID: &testEnvID,
			Metadata: &storepb.Instance{
				Title: "Test Sample Instance",
				DataSources: []*storepb.DataSource{
					{
						Port:     strconv.Itoa(m.port),
						Database: postgres.SampleDatabaseTest,
					},
				},
			},
		},
		{
			ResourceID:    "prod-sample-instance",
			EnvironmentID: &prodEnvID,
			Metadata: &storepb.Instance{
				Title: "Prod Sample Instance",
				DataSources: []*storepb.DataSource{
					{
						Port:     strconv.Itoa(m.port + 1),
						Database: postgres.SampleDatabaseProd,
					},
				},
			},
		},
	}
	for _, instanceMessage := range instanceMessages {
		if err := m.generateInstance(ctx, project.ResourceID, instanceMessage, schemaSyncer); err != nil {
			slog.Error("failed to prepare onboarding instance", log.BBError(err), slog.String("instance", instanceMessage.ResourceID))
		}
	}

	return nil
}

func (m *Manager) generateInstance(
	ctx context.Context,
	projectID string,
	instanceMessage *store.InstanceMessage,
	schemaSyncer *schemasync.Syncer,
) error {
	// Generate Sample Instance
	instance, err := m.store.GetInstance(ctx, &store.FindInstanceMessage{
		ResourceID: &instanceMessage.ResourceID,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to find onboarding instance %v", instanceMessage.ResourceID)
	}
	if instance == nil {
		sampleInstance, err := m.store.CreateInstance(ctx, &store.InstanceMessage{
			ResourceID:    instanceMessage.ResourceID,
			EnvironmentID: instanceMessage.EnvironmentID,
			Metadata: &storepb.Instance{
				Title:        instanceMessage.Metadata.Title,
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
						Port:     instanceMessage.Metadata.DataSources[0].Port,
						Database: instanceMessage.Metadata.DataSources[0].Database,
					},
				},
			},
		})
		if err != nil {
			return errors.Wrapf(err, "failed to create onboarding instance %v", instanceMessage.ResourceID)
		}
		instance = sampleInstance

		// Start the sample instance if needed
		if err := m.HandleInstanceCreation(ctx, instance.ResourceID); err != nil {
			slog.Warn("failed to start sample instance during onboarding", log.BBError(err), slog.String("instance", instance.ResourceID))
		}
	}

	// Sync the instance schema so we can transfer the sample database later.
	if _, _, _, err := schemaSyncer.SyncInstance(ctx, instance); err != nil {
		return errors.Wrapf(err, "failed to sync onboarding instance %v", instance.ResourceID)
	}

	dbName := instanceMessage.Metadata.DataSources[0].Database
	// Transfer sample database to the just created project.
	transferDatabaseMessage := &store.UpdateDatabaseMessage{
		InstanceID:   instance.ResourceID,
		DatabaseName: dbName,
		ProjectID:    &projectID,
	}
	_, err = m.store.UpdateDatabase(ctx, transferDatabaseMessage)
	if err != nil {
		return errors.Wrapf(err, "failed to transfer sample database %v", dbName)
	}

	testDatabase, err := m.store.GetDatabase(ctx, &store.FindDatabaseMessage{
		InstanceID:   &instance.ResourceID,
		DatabaseName: &dbName,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to find onboarding database %v", dbName)
	}
	if testDatabase == nil {
		return errors.Errorf("database %q not found", dbName)
	}

	// Need to sync database schema so we can configure sensitive data policy and create the schema
	// update issue later. Also establish a baseline for schema rollback.
	syncHistory, err := schemaSyncer.SyncDatabaseSchemaToHistory(ctx, testDatabase)
	if err != nil {
		return errors.Wrapf(err, "failed to sync sample database schema %v", dbName)
	}

	if _, err := m.store.CreateChangelog(ctx, &store.ChangelogMessage{
		InstanceID:     testDatabase.InstanceID,
		DatabaseName:   dbName,
		Status:         store.ChangelogStatusDone,
		SyncHistoryUID: &syncHistory,
		Payload: &storepb.ChangelogPayload{
			Type:        storepb.ChangelogPayload_BASELINE,
			GitCommit:   m.profile.GitCommit,
			DumpVersion: schema.GetDumpFormatVersion(instance.Metadata.GetEngine()),
		},
	}); err != nil {
		return errors.Wrapf(err, "failed to create baseline changelog for %v", dbName)
	}
	return nil
}
