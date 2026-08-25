package selfhost

import (
	"context"
	"log/slog"
	"path/filepath"
	"sync"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/sample"
	"github.com/bytebase/bytebase/backend/resources/postgres"
	"github.com/bytebase/bytebase/backend/store"
)

const (
	legacyTestInstanceID = "test-sample-instance"
	legacyProdInstanceID = "prod-sample-instance"
)

type legacyAdapter struct {
	store   *store.Store
	profile *config.Profile
	mu      sync.Mutex
	running bool
	stops   []func()
}

func newLegacyAdapter(stores *store.Store, profile *config.Profile) *legacyAdapter {
	return &legacyAdapter{store: stores, profile: profile}
}

func isLegacyInstanceID(instanceID string) bool {
	return instanceID == legacyTestInstanceID || instanceID == legacyProdInstanceID
}

func (a *legacyAdapter) instances(ctx context.Context, workspaceID string) ([]*store.InstanceMessage, error) {
	var result []*store.InstanceMessage
	for _, id := range []string{legacyTestInstanceID, legacyProdInstanceID} {
		instance, err := a.store.GetInstance(ctx, &store.FindInstanceMessage{
			Workspace:   workspaceID,
			ResourceID:  &id,
			ShowDeleted: true,
		})
		if err != nil {
			return nil, err
		}
		if instance != nil {
			result = append(result, instance)
		}
	}
	return result, nil
}

func (a *legacyAdapter) exists(ctx context.Context, workspaceID string) (bool, error) {
	instances, err := a.instances(ctx, workspaceID)
	return len(instances) > 0, err
}

func (a *legacyAdapter) start(ctx context.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.running {
		return
	}
	seedData, err := sample.LoadSeedData()
	if err != nil {
		slog.Error("failed to load legacy sample data", log.BBError(err))
		return
	}
	for index, entry := range []struct {
		name     string
		database string
	}{
		{name: "test", database: sampleDatabaseTest},
		{name: "prod", database: sampleDatabaseProd},
	} {
		stop, err := postgres.StartEmbeddedInstance(ctx, postgres.EmbeddedInstanceConfig{
			DataDir:      filepath.Join(a.profile.DataDir, "pgdata-sample", entry.name),
			Port:         a.profile.Port + 3 + index,
			User:         sampleUser,
			DatabaseName: entry.database,
			SeedData:     seedData,
		})
		if err != nil {
			slog.Error("failed to start legacy sample instance", slog.String("instance", entry.name), log.BBError(err))
			continue
		}
		a.stops = append(a.stops, stop)
	}
	a.running = len(a.stops) > 0
}

func (a *legacyAdapter) stop() {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, stop := range a.stops {
		stop()
	}
	a.stops = nil
	a.running = false
}

func (a *legacyAdapter) handleLifecycle(ctx context.Context, workspaceID, instanceID string, deleted bool) error {
	if !isLegacyInstanceID(instanceID) {
		return nil
	}
	if !deleted {
		a.start(ctx)
		return nil
	}
	instances, err := a.instances(ctx, workspaceID)
	if err != nil {
		return err
	}
	for _, instance := range instances {
		if !instance.Deleted {
			return nil
		}
	}
	a.stop()
	return nil
}

func (a *legacyAdapter) info(ctx context.Context, workspaceID string) ([]sampleInstanceInfo, error) {
	instances, err := a.instances(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	result := make([]sampleInstanceInfo, 0, len(instances))
	for _, instance := range instances {
		result = append(result, sampleInstanceInfo{instance: common.FormatInstance(instance.ResourceID)})
	}
	return result, nil
}
