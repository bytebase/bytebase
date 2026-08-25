// Package selfhost manages the two workspace-level embedded PostgreSQL sample
// instances used by self-hosted Bytebase.
package selfhost

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	pkgerrors "github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/sample"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/resources/postgres"
	"github.com/bytebase/bytebase/backend/runner/schemasync"
	"github.com/bytebase/bytebase/backend/store"
)

const (
	prepareDeadline           = 3 * time.Minute
	staleReservationAge       = time.Hour
	replicaHeartbeatStaleness = time.Minute
	pollInitialDelay          = 10 * time.Millisecond
	pollMaxDelay              = time.Second
	sampleUser                = "bbsample"
	sampleDatabaseTest        = "hr_test"
	sampleDatabaseProd        = "hr_prod"
)

type sampleInstanceInfo struct {
	instance string
}

// Manager is the self-host sample implementation.
type Manager struct {
	store     *store.Store
	profile   *config.Profile
	syncer    *schemasync.Syncer
	clock     func() time.Time
	random    io.Reader
	replicaID string
	legacy    *legacyAdapter

	mu       sync.Mutex
	stoppers map[string]func()
}

// NewManager creates the self-host sample manager.
func NewManager(stores *store.Store, profile *config.Profile, syncer *schemasync.Syncer, options sample.ManagerOptions) *Manager {
	if options.Clock == nil {
		options.Clock = time.Now
	}
	if options.Random == nil {
		options.Random = rand.Reader
	}
	return &Manager{
		store:     stores,
		profile:   profile,
		syncer:    syncer,
		clock:     options.Clock,
		random:    options.Random,
		replicaID: options.ReplicaID,
		legacy:    newLegacyAdapter(stores, profile),
		stoppers:  map[string]func(){},
	}
}

func managedEntry(instanceID, title string, portOffset int32, database string) *storepb.SelfHostSampleInstanceSetupPayload_Instance {
	return &storepb.SelfHostSampleInstanceSetupPayload_Instance{
		InstanceId:   instanceID,
		Title:        title,
		PortOffset:   portOffset,
		DatabaseName: database,
		RoleName:     sampleUser,
	}
}

func newPayload(reader io.Reader, projectID string, testEnvironment, prodEnvironment *string) (*storepb.SelfHostSampleInstanceSetupPayload, error) {
	testID, err := randomInstanceID(reader)
	if err != nil {
		return nil, err
	}
	prodID, err := randomInstanceID(reader)
	if err != nil {
		return nil, err
	}
	test := managedEntry(testID, "Test Sample Instance", 0, sampleDatabaseTest)
	prod := managedEntry(prodID, "Prod Sample Instance", 1, sampleDatabaseProd)
	test.EnvironmentId = testEnvironment
	prod.EnvironmentId = prodEnvironment
	return &storepb.SelfHostSampleInstanceSetupPayload{
		DatabaseProjectId: projectID,
		Instances:         []*storepb.SelfHostSampleInstanceSetupPayload_Instance{test, prod},
	}, nil
}

func randomInstanceID(reader io.Reader) (string, error) {
	value := make([]byte, 8)
	if _, err := io.ReadFull(reader, value); err != nil {
		return "", err
	}
	return "sample-" + hex.EncodeToString(value), nil
}

func sampleConfig(profile *config.Profile, entry *storepb.SelfHostSampleInstanceSetupPayload_Instance) postgres.EmbeddedInstanceConfig {
	return postgres.EmbeddedInstanceConfig{
		DataDir:      filepath.Join(profile.DataDir, "pgdata-sample-managed", entry.InstanceId),
		Port:         profile.Port + 3 + int(entry.PortOffset),
		User:         entry.RoleName,
		DatabaseName: entry.DatabaseName,
	}
}

func decode(setup *store.SampleInstanceSetupMessage) (*storepb.SelfHostSampleInstanceSetupPayload, error) {
	if setup == nil {
		return nil, sample.NewFailure(sample.FailureFailedPrecondition, errors.New("sample setup is missing"))
	}
	payload := &storepb.SelfHostSampleInstanceSetupPayload{}
	if err := common.ProtojsonUnmarshaler.Unmarshal(setup.Payload, payload); err != nil {
		return nil, sample.NewFailure(sample.FailureFailedPrecondition, errors.New("invalid self-host sample setup payload"))
	}
	if payload.DatabaseProjectId == "" || len(payload.Instances) != 2 {
		return nil, sample.NewFailure(sample.FailureFailedPrecondition, errors.New("incomplete self-host sample setup payload"))
	}
	for _, entry := range payload.Instances {
		if entry.InstanceId == "" || entry.Title == "" || entry.DatabaseName == "" || entry.RoleName == "" {
			return nil, sample.NewFailure(sample.FailureFailedPrecondition, errors.New("incomplete self-host sample instance payload"))
		}
	}
	return payload, nil
}

// SetupSample creates two permanent workspace-level embedded sample instances.
func (m *Manager) SetupSample(ctx context.Context, request sample.SetupRequest) (*sample.SetupResult, error) {
	if m == nil || m.store == nil || m.profile == nil || m.syncer == nil || m.replicaID == "" {
		return nil, errors.New("self-host sample manager is not configured")
	}
	if request.WorkspaceID == "" || request.ProjectID == "" {
		return nil, sample.NewFailure(sample.FailureFailedPrecondition, errors.New("self-host sample requires workspace and project"))
	}
	legacy, err := m.legacy.exists(ctx, request.WorkspaceID)
	if err != nil {
		return nil, err
	}
	if legacy {
		return nil, sample.NewFailure(sample.FailureFailedPrecondition, errors.New("legacy self-host sample instances already exist"))
	}
	lifecycleCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), prepareDeadline)
	defer cancel()
	setup, created, err := m.reserve(lifecycleCtx, request)
	if err != nil {
		return nil, err
	}
	delay := pollInitialDelay
	for {
		payload, err := decode(setup)
		if err != nil {
			return nil, err
		}
		if payload.DatabaseProjectId != request.ProjectID || setup.DeletedAt != nil {
			return nil, sample.NewFailure(sample.FailureFailedPrecondition, errors.New("sample setup entitlement is already consumed"))
		}
		if created {
			return m.prepareOwned(lifecycleCtx, setup, payload, false)
		}
		if setup.ActivatedAt != nil {
			return m.activeSetup(lifecycleCtx, setup, payload)
		}
		claimed, ok, err := m.store.ClaimSampleInstanceSetup(lifecycleCtx, setup.WorkspaceID, setup.UpdatedAt, m.replicaID, prepareDeadline, replicaHeartbeatStaleness)
		if err != nil {
			return nil, pkgerrors.Wrap(err, "failed to claim abandoned self-host sample setup")
		}
		if ok {
			claimedPayload, err := decode(claimed)
			if err != nil {
				return nil, err
			}
			return m.prepareOwned(lifecycleCtx, claimed, claimedPayload, true)
		}
		if err := sample.SleepContext(lifecycleCtx, delay); err != nil {
			return nil, sample.NewFailure(sample.FailureDeadlineExceeded, err)
		}
		if delay < pollMaxDelay {
			delay *= 2
			if delay > pollMaxDelay {
				delay = pollMaxDelay
			}
		}
		setup, err = m.store.GetSampleInstanceSetup(lifecycleCtx, request.WorkspaceID)
		if err != nil {
			return nil, err
		}
		created = false
		if setup == nil {
			setup, created, err = m.reserve(lifecycleCtx, request)
			if err != nil {
				return nil, err
			}
		}
	}
}

func (m *Manager) reserve(ctx context.Context, request sample.SetupRequest) (*store.SampleInstanceSetupMessage, bool, error) {
	environments, err := m.store.GetEnvironment(ctx, request.WorkspaceID)
	if err != nil {
		return nil, false, err
	}
	var testEnvironment, prodEnvironment *string
	for _, environment := range environments.GetEnvironments() {
		switch environment.Id {
		case common.DefaultTestEnvironmentID:
			value := environment.Id
			testEnvironment = &value
		case common.DefaultProdEnvironmentID:
			value := environment.Id
			prodEnvironment = &value
		default:
		}
	}
	payload, err := newPayload(m.random, request.ProjectID, testEnvironment, prodEnvironment)
	if err != nil {
		return nil, false, pkgerrors.Wrap(err, "failed to generate self-host sample payload")
	}
	encoded, err := protojson.Marshal(payload)
	if err != nil {
		return nil, false, pkgerrors.Wrap(err, "failed to encode self-host sample payload")
	}
	return m.store.ReserveSampleInstanceSetup(ctx, &store.SampleInstanceSetupMessage{
		WorkspaceID: request.WorkspaceID,
		ReplicaID:   m.replicaID,
		Payload:     encoded,
	})
}

func (m *Manager) prepareOwned(ctx context.Context, setup *store.SampleInstanceSetupMessage, payload *storepb.SelfHostSampleInstanceSetupPayload, takeover bool) (*sample.SetupResult, error) {
	if takeover {
		if err := m.reconcile(ctx, setup.WorkspaceID, payload); err != nil {
			return nil, sample.NewFailure(sample.FailureUnavailable, err)
		}
	}
	instances := make([]*store.InstanceMessage, 0, len(payload.Instances))
	for _, entry := range payload.Instances {
		if err := m.startEntry(ctx, entry); err != nil {
			return nil, m.compensate(ctx, setup, payload, err)
		}
		registered, err := sample.CreateMetadata(ctx, m.store, sample.Registration{
			WorkspaceID:       setup.WorkspaceID,
			EnvironmentID:     entry.EnvironmentId,
			InstanceID:        entry.InstanceId,
			Title:             entry.Title,
			AdminDataSource:   dataSource(m.profile, entry),
			SyncDatabaseNames: []string{entry.DatabaseName},
		})
		if err != nil {
			return nil, m.compensate(ctx, setup, payload, err)
		}
		synced, _, databases, err := m.syncer.SyncInstance(ctx, registered)
		if err != nil || len(databases) != 1 || databases[0].DatabaseName != entry.DatabaseName || databases[0].Deleted {
			return nil, m.compensate(ctx, setup, payload, errors.Join(err, errors.New("self-host sample database discovery invariant failed")))
		}
		if err := sample.TransferDatabase(ctx, m.store, payload.DatabaseProjectId, entry.InstanceId, entry.DatabaseName); err != nil {
			return nil, m.compensate(ctx, setup, payload, err)
		}
		if err := m.createBaseline(ctx, databases[0]); err != nil {
			return nil, m.compensate(ctx, setup, payload, err)
		}
		m.syncer.SyncDatabasesAsync(databases)
		instances = append(instances, synced)
	}
	activated, err := m.store.ActivateSampleInstanceSetup(ctx, setup.WorkspaceID, m.replicaID, []string{payload.DatabaseProjectId}, m.clock(), nil)
	if err != nil || !activated {
		return nil, m.compensate(ctx, setup, payload, errors.Join(errors.New("failed to activate self-host sample setup"), err))
	}
	return &sample.SetupResult{Instances: instances}, nil
}

func dataSource(profile *config.Profile, entry *storepb.SelfHostSampleInstanceSetupPayload_Instance) *storepb.DataSource {
	return &storepb.DataSource{
		Id:       "admin",
		Type:     storepb.DataSourceType_ADMIN,
		Username: entry.RoleName,
		Host:     common.GetPostgresSocketDir(),
		Port:     strconv.Itoa(profile.Port + 3 + int(entry.PortOffset)),
		Database: entry.DatabaseName,
	}
}

func (m *Manager) createBaseline(ctx context.Context, database *store.DatabaseMessage) error {
	syncHistory, err := m.syncer.SyncDatabaseSchemaToHistory(ctx, database)
	if err != nil {
		return pkgerrors.Wrap(err, "failed to sync self-host sample baseline")
	}
	_, err = m.store.CreateChangelog(ctx, &store.ChangelogMessage{
		InstanceID:   database.InstanceID,
		DatabaseName: database.DatabaseName,
		Status:       store.ChangelogStatusDone,
		SyncHistory:  &syncHistory,
		Payload:      &storepb.ChangelogPayload{GitCommit: m.profile.GitCommit},
	})
	return err
}

func (m *Manager) activeSetup(ctx context.Context, setup *store.SampleInstanceSetupMessage, payload *storepb.SelfHostSampleInstanceSetupPayload) (*sample.SetupResult, error) {
	instances := make([]*store.InstanceMessage, 0, len(payload.Instances))
	for _, entry := range payload.Instances {
		state, err := sample.LookupMetadata(ctx, m.store, sample.MetadataLookup{
			WorkspaceID:       setup.WorkspaceID,
			DatabaseProjectID: payload.DatabaseProjectId,
			InstanceID:        entry.InstanceId,
			DatabaseName:      entry.DatabaseName,
		})
		if err != nil {
			return nil, err
		}
		if !state.Active() {
			return nil, sample.NewFailure(sample.FailureFailedPrecondition, errors.New("self-host sample metadata is not active"))
		}
		instances = append(instances, state.Instance)
	}
	return &sample.SetupResult{Instances: instances}, nil
}

func (m *Manager) compensate(ctx context.Context, setup *store.SampleInstanceSetupMessage, payload *storepb.SelfHostSampleInstanceSetupPayload, original error) error {
	if err := m.reconcile(ctx, setup.WorkspaceID, payload); err != nil {
		return sample.NewFailure(sample.FailureUnavailable, errors.Join(original, err))
	}
	deleted, err := m.store.DeletePendingSampleInstanceSetup(ctx, setup.WorkspaceID, m.replicaID)
	if err != nil || !deleted {
		return sample.NewFailure(sample.FailureUnavailable, errors.Join(original, err))
	}
	return original
}

func (m *Manager) reconcile(ctx context.Context, workspaceID string, payload *storepb.SelfHostSampleInstanceSetupPayload) error {
	var result error
	for _, entry := range payload.Instances {
		result = errors.Join(result, sample.PurgePartialMetadata(ctx, m.store, workspaceID, nil, entry.InstanceId))
		m.stopEntry(entry.InstanceId)
		result = errors.Join(result, postgres.RemoveEmbeddedInstance(sampleConfig(m.profile, entry).DataDir))
	}
	return result
}

func (m *Manager) startEntry(ctx context.Context, entry *storepb.SelfHostSampleInstanceSetupPayload_Instance) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.stoppers[entry.InstanceId]; ok {
		return nil
	}
	seedData, err := sample.LoadSeedData()
	if err != nil {
		return err
	}
	config := sampleConfig(m.profile, entry)
	config.SeedData = seedData
	stop, err := postgres.StartEmbeddedInstance(ctx, config)
	if err != nil {
		return err
	}
	m.stoppers[entry.InstanceId] = stop
	return nil
}

func (m *Manager) stopEntry(instanceID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if stop := m.stoppers[instanceID]; stop != nil {
		stop()
		delete(m.stoppers, instanceID)
	}
}

// Info reports managed payload instances or historical static instances.
func (m *Manager) Info(ctx context.Context, workspaceID string) (*sample.Info, error) {
	setup, err := m.store.GetSampleInstanceSetup(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	if setup != nil {
		payload, err := decode(setup)
		if err != nil {
			return nil, err
		}
		info := &sample.Info{Available: setup.DeletedAt == nil}
		for _, entry := range payload.Instances {
			instance := sample.InstanceInfo{Instance: common.FormatInstance(entry.InstanceId)}
			if setup.DeletedAt == nil {
				instance.ExpireTime = setup.ExpiresAt
			}
			info.Instances = append(info.Instances, instance)
		}
		return info, nil
	}
	legacy, err := m.legacy.info(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	info := &sample.Info{Available: true}
	for _, entry := range legacy {
		info.Instances = append(info.Instances, sample.InstanceInfo{Instance: entry.instance})
	}
	return info, nil
}

// Start restores active embedded sample processes after server restart.
func (m *Manager) Start(ctx context.Context, workspaceID string) error {
	setup, err := m.store.GetSampleInstanceSetup(ctx, workspaceID)
	if err != nil {
		return err
	}
	if setup != nil && setup.ActivatedAt != nil && setup.DeletedAt == nil && (setup.ExpiresAt == nil || setup.ExpiresAt.After(m.clock())) {
		payload, err := decode(setup)
		if err != nil {
			return err
		}
		active := false
		for _, entry := range payload.Instances {
			instance, err := m.store.GetInstance(ctx, &store.FindInstanceMessage{Workspace: workspaceID, ResourceID: &entry.InstanceId, ShowDeleted: true})
			if err != nil {
				return err
			}
			if instance != nil && !instance.Deleted {
				active = true
			}
		}
		if active {
			for _, entry := range payload.Instances {
				if err := m.startEntry(ctx, entry); err != nil {
					return err
				}
			}
		}
		return nil
	}
	legacy, err := m.legacy.instances(ctx, workspaceID)
	if err != nil {
		return err
	}
	for _, instance := range legacy {
		if !instance.Deleted {
			m.legacy.start(ctx)
			break
		}
	}
	return nil
}

// Cleanup reconciles stale pending self-host setups. Permanent active setups
// are not selected by the store cleanup query.
func (m *Manager) Cleanup(ctx context.Context) error {
	now := m.clock()
	staleBefore := now.Add(-staleReservationAge)
	for afterWorkspace := ""; ; {
		result, err := m.store.WithLockedSampleInstanceSetupForCleanup(ctx, now, staleBefore, afterWorkspace, func(callbackCtx context.Context, tx *store.SampleInstanceSetupTx, setup *store.SampleInstanceSetupMessage) error {
			payload, err := decode(setup)
			if err != nil {
				return err
			}
			if setup.ActivatedAt != nil {
				for _, entry := range payload.Instances {
					m.stopEntry(entry.InstanceId)
					if err := postgres.RemoveEmbeddedInstance(sampleConfig(m.profile, entry).DataDir); err != nil {
						return err
					}
					if _, err := sample.ArchiveMetadata(callbackCtx, m.store, setup.WorkspaceID, nil, entry.InstanceId); err != nil {
						return err
					}
				}
				return tx.MarkDeleted(callbackCtx, now)
			}
			if err := m.reconcile(callbackCtx, setup.WorkspaceID, payload); err != nil {
				return err
			}
			return tx.DeleteReservation(callbackCtx)
		})
		if err != nil {
			return err
		}
		if !result.Found {
			return nil
		}
		afterWorkspace = result.WorkspaceID
		if result.CallbackErr != nil {
			return result.CallbackErr
		}
	}
}

// ValidateInstanceRestore rejects restoring a physically removed self-host
// sample while allowing permanent and historical samples.
func (m *Manager) ValidateInstanceRestore(ctx context.Context, workspaceID, instanceID string) error {
	setup, err := m.store.GetSampleInstanceSetup(ctx, workspaceID)
	if err != nil || setup == nil || setup.DeletedAt == nil {
		return err
	}
	payload, err := decode(setup)
	if err != nil {
		return err
	}
	for _, entry := range payload.Instances {
		if entry.InstanceId == instanceID {
			return sample.NewFailure(sample.FailureFailedPrecondition, errors.New("expired sample instance cannot be restored"))
		}
	}
	return nil
}

// HandleInstanceLifecycle starts or stops the exact managed sample process.
func (m *Manager) HandleInstanceLifecycle(ctx context.Context, workspaceID, instanceID string, deleted bool) error {
	if isLegacyInstanceID(instanceID) {
		return m.legacy.handleLifecycle(ctx, workspaceID, instanceID, deleted)
	}
	setup, err := m.store.GetSampleInstanceSetup(ctx, workspaceID)
	if err != nil || setup == nil || setup.DeletedAt != nil {
		return err
	}
	payload, err := decode(setup)
	if err != nil {
		return err
	}
	managed := false
	for _, entry := range payload.Instances {
		managed = managed || entry.InstanceId == instanceID
	}
	if !managed {
		return nil
	}
	if !deleted {
		for _, entry := range payload.Instances {
			if err := m.startEntry(ctx, entry); err != nil {
				return err
			}
		}
		return nil
	}
	for _, entry := range payload.Instances {
		instance, err := m.store.GetInstance(ctx, &store.FindInstanceMessage{Workspace: workspaceID, ResourceID: &entry.InstanceId, ShowDeleted: true})
		if err != nil {
			return err
		}
		if instance != nil && !instance.Deleted {
			return nil
		}
	}
	for _, entry := range payload.Instances {
		m.stopEntry(entry.InstanceId)
	}
	return nil
}

// Stop stops managed and legacy embedded sample processes.
func (m *Manager) Stop() {
	m.mu.Lock()
	for id, stop := range m.stoppers {
		stop()
		delete(m.stoppers, id)
	}
	m.mu.Unlock()
	m.legacy.stop()
}
