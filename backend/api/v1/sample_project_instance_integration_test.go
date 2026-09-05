package v1

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bytebase/bytebase/backend/common/testpg"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/permission"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/iam"
	"github.com/bytebase/bytebase/backend/component/sample"
	"github.com/bytebase/bytebase/backend/component/sample/saas"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	_ "github.com/bytebase/bytebase/backend/plugin/db/pg"
	samplerunner "github.com/bytebase/bytebase/backend/runner/sample"
	"github.com/bytebase/bytebase/backend/runner/schemasync"
	"github.com/bytebase/bytebase/backend/store"
)

func TestPrepareSampleProjectInstanceLifecycle(t *testing.T) {
	now := time.Date(2026, time.August, 17, 12, 0, 0, 0, time.UTC)
	ctx, fixture := newSampleProjectInstanceFixture(t, func() time.Time { return now })
	runner := samplerunner.NewRunner(fixture.manager)

	prepared, err := fixture.service.PrepareSampleProjectInstance(ctx, connect.NewRequest(&v1pb.PrepareSampleProjectInstanceRequest{
		Parent: common.FormatProject(fixture.projectID),
	}))
	require.NoError(t, err)
	require.Equal(t, "Sample Project Instance", prepared.Msg.Title)
	require.Equal(t, v1pb.Engine_POSTGRES, prepared.Msg.Engine)
	require.Equal(t, "environments/test", prepared.Msg.GetEnvironment())
	require.False(t, prepared.Msg.Activation)
	activatedInstanceCount, err := fixture.store.GetActivatedInstanceCount(ctx, fixture.workspaceID)
	require.NoError(t, err)
	require.Zero(t, activatedInstanceCount)

	record := fixture.sampleRecord(ctx, t)
	allocation := fixture.allocation(ctx, t)
	require.Equal(t, fixture.workspaceID, record.workspace)
	require.Equal(t, fixture.projectID, record.project)
	require.Equal(t, prepared.Msg.Name, common.FormatProjectInstance(record.project, record.instance))
	require.Equal(t, allocation.Database, record.database)
	require.Equal(t, allocation.Role, record.role)
	require.False(t, record.createdAt.IsZero())
	require.Equal(t, now.Add(7*24*time.Hour), record.expiresAt)
	require.Nil(t, record.deletedAt)

	database, err := fixture.store.GetDatabase(ctx, &store.FindDatabaseMessage{
		Workspace:    fixture.workspaceID,
		ProjectID:    &fixture.projectID,
		InstanceID:   &record.instance,
		DatabaseName: &record.database,
	})
	require.NoError(t, err)
	require.NotNil(t, database)
	require.NoError(t, fixture.target.sampleEmployeeCount(ctx, allocation, func(count int) bool { return count > 0 }))
	fixture.target.assertSeededOwnershipAndDDL(ctx, t, allocation)

	repeated, err := fixture.service.PrepareSampleProjectInstance(ctx, connect.NewRequest(&v1pb.PrepareSampleProjectInstanceRequest{
		Parent: common.FormatProject(fixture.projectID),
	}))
	require.NoError(t, err)
	require.Equal(t, prepared.Msg.Name, repeated.Msg.Name)
	require.Equal(t, 1, fixture.sampleRecordCount(ctx, t))
	require.True(t, fixture.target.databaseExists(ctx, allocation.Database))
	require.True(t, fixture.target.roleExists(ctx, allocation.Role))
	fixture.target.assertIsolation(ctx, t, allocation)

	now = now.Add(6 * 24 * time.Hour)
	require.NoError(t, runner.RunOnce(ctx))
	require.True(t, fixture.target.databaseExists(ctx, allocation.Database))
	require.Nil(t, fixture.sampleRecord(ctx, t).deletedAt)

	held := fixture.target.sampleConnection(ctx, t, allocation)
	t.Cleanup(func() { _ = held.Close(context.Background()) })
	now = now.Add(24*time.Hour + time.Second)
	require.NoError(t, runner.RunOnce(ctx))
	require.Error(t, held.Ping(ctx))
	require.False(t, fixture.target.databaseExists(ctx, allocation.Database))
	require.False(t, fixture.target.roleExists(ctx, allocation.Role))
	require.NotNil(t, fixture.sampleRecord(ctx, t).deletedAt)

	instance, err := fixture.store.GetInstance(ctx, &store.FindInstanceMessage{
		Workspace:  fixture.workspaceID,
		ResourceID: &record.instance,
	})
	require.NoError(t, err)
	require.NotNil(t, instance)
	require.True(t, instance.Deleted)
	instances, err := fixture.store.ListInstances(ctx, &store.FindInstanceMessage{
		Workspace:  fixture.workspaceID,
		ResourceID: &record.instance,
	})
	require.NoError(t, err)
	require.Empty(t, instances)
	databases, err := fixture.store.ListDatabases(ctx, &store.FindDatabaseMessage{
		Workspace:    fixture.workspaceID,
		ProjectID:    &fixture.projectID,
		InstanceID:   &record.instance,
		DatabaseName: &record.database,
	})
	require.NoError(t, err)
	require.Empty(t, databases)
	database, err = fixture.store.GetDatabase(ctx, &store.FindDatabaseMessage{
		Workspace:    fixture.workspaceID,
		ProjectID:    &fixture.projectID,
		InstanceID:   &record.instance,
		DatabaseName: &record.database,
		ShowDeleted:  true,
	})
	require.NoError(t, err)
	require.NotNil(t, database)
	require.False(t, database.Deleted)
	_, err = fixture.service.UndeleteInstance(ctx, connect.NewRequest(&v1pb.UndeleteInstanceRequest{Name: prepared.Msg.Name}))
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	archived, err := fixture.store.GetInstance(ctx, &store.FindInstanceMessage{
		Workspace:   fixture.workspaceID,
		ResourceID:  &record.instance,
		ShowDeleted: true,
	})
	require.NoError(t, err)
	require.True(t, archived.Deleted)

	_, err = fixture.service.PrepareSampleProjectInstance(ctx, connect.NewRequest(&v1pb.PrepareSampleProjectInstanceRequest{
		Parent: common.FormatProject(fixture.projectID),
	}))
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
}

func TestSampleProjectPurgeRejectsPendingReservation(t *testing.T) {
	ctx, fixture := newSampleProjectInstanceFixture(t, time.Now)
	payload := &storepb.SaaSSampleInstanceSetupPayload{
		ProjectId:    fixture.projectID,
		InstanceId:   "pending-sample",
		Title:        "Sample Project Instance",
		DatabaseName: "pending-database",
		RoleName:     "pending-role",
	}
	encoded, err := protojson.Marshal(payload)
	require.NoError(t, err)
	_, created, err := fixture.store.ReserveSampleInstanceSetup(ctx, &store.SampleInstanceSetupMessage{
		WorkspaceID: fixture.workspaceID,
		ReplicaID:   "replica-a",
		Payload:     encoded,
	})
	require.NoError(t, err)
	require.True(t, created)

	err = fixture.manager.HandleProjectPurge(ctx, fixture.workspaceID, fixture.projectID)
	require.ErrorContains(t, err, "sample provisioning is still in progress")

	setup, err := fixture.store.GetSampleInstanceSetup(ctx, fixture.workspaceID)
	require.NoError(t, err)
	require.NotNil(t, setup)
	require.Nil(t, setup.ActivatedAt)
	require.Nil(t, setup.DeletedAt)
}

func TestPrepareSampleProjectInstanceAdditionalLifecycleCoverage(t *testing.T) {
	now := time.Date(2026, time.August, 17, 12, 0, 0, 0, time.UTC)
	ctx, fixture := newSampleProjectInstanceFixture(t, func() time.Time { return now })
	runner := samplerunner.NewRunner(fixture.manager)

	t.Run("ACL denies callers without project permission", func(t *testing.T) {
		iamManager, err := iam.NewManager(fixture.store, nil, true)
		require.NoError(t, err)
		aclInterceptor := NewACLInterceptor(fixture.store, "", iamManager, fixture.service.profile)
		injectIdentity := connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
			return func(requestCtx context.Context, request connect.AnyRequest) (connect.AnyResponse, error) {
				requestCtx = context.WithValue(requestCtx, common.WorkspaceIDContextKey, fixture.workspaceID)
				requestCtx = context.WithValue(requestCtx, common.AuthContextKey, &common.AuthContext{
					AuthMethod: common.AuthMethodIAM,
					Permission: string(permission.InstancesCreate),
				})
				requestCtx = context.WithValue(requestCtx, common.UserContextKey, &store.UserMessage{Email: "denied@example.com"})
				return next(requestCtx, request)
			}
		})
		_, handler := v1connect.NewInstanceServiceHandler(fixture.service, connect.WithInterceptors(injectIdentity, aclInterceptor))
		server := httptest.NewServer(handler)
		t.Cleanup(server.Close)
		client := v1connect.NewInstanceServiceClient(server.Client(), server.URL)

		_, err = client.PrepareSampleProjectInstance(context.Background(), connect.NewRequest(&v1pb.PrepareSampleProjectInstanceRequest{
			Parent: common.FormatProject(fixture.projectID),
		}))
		require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
		require.Zero(t, fixture.sampleRecordCount(ctx, t))
	})

	t.Run("workspace without environments still prepares a sample", func(t *testing.T) {
		workspaceID := "sample-workspace-no-environment"
		projectID := "sample-project-no-environment"
		_, err := fixture.store.GetDB().ExecContext(ctx, `INSERT INTO workspace (resource_id) VALUES ($1)`, workspaceID)
		require.NoError(t, err)
		_, err = fixture.store.GetDB().ExecContext(ctx, `
			INSERT INTO project (resource_id, workspace, name) VALUES ($1, $2, 'No environment sample')
		`, projectID, workspaceID)
		require.NoError(t, err)
		requestCtx := context.WithValue(ctx, common.WorkspaceIDContextKey, workspaceID)
		prepared, err := fixture.service.PrepareSampleProjectInstance(requestCtx, connect.NewRequest(&v1pb.PrepareSampleProjectInstanceRequest{
			Parent: common.FormatProject(projectID),
		}))
		require.NoError(t, err)
		require.Empty(t, prepared.Msg.GetEnvironment())
	})

	concurrentCtx, concurrentProjectID := fixture.newWorkspaceProject(ctx, t, "concurrent")
	t.Run("two concurrent cold-start requests create exactly one instance", func(t *testing.T) {
		const requests = 2
		results := make(chan error, requests)
		var wg sync.WaitGroup
		for range requests {
			wg.Go(func() {
				requestCtx, cancel := context.WithTimeout(concurrentCtx, 15*time.Second)
				defer cancel()
				_, err := fixture.service.PrepareSampleProjectInstance(requestCtx, connect.NewRequest(&v1pb.PrepareSampleProjectInstanceRequest{
					Parent: common.FormatProject(concurrentProjectID),
				}))
				results <- err
			})
		}
		wg.Wait()
		close(results)
		for err := range results {
			require.NoError(t, err)
		}

		record := fixture.sampleRecordFor(concurrentCtx, t, "sample-workspace-concurrent")
		require.Equal(t, 1, fixture.sampleRecordCountFor(concurrentCtx, t, "sample-workspace-concurrent"))
		require.True(t, fixture.target.databaseExists(concurrentCtx, record.database))
		require.True(t, fixture.target.roleExists(concurrentCtx, record.role))
		fixture.assertActiveMetadata(concurrentCtx, t, record)
	})

	t.Run("cross-project repeat is rejected", func(t *testing.T) {
		otherProjectID := fixture.addProject(concurrentCtx, t, "sample-workspace-concurrent", "other")
		_, err := fixture.service.PrepareSampleProjectInstance(concurrentCtx, connect.NewRequest(&v1pb.PrepareSampleProjectInstanceRequest{
			Parent: common.FormatProject(otherProjectID),
		}))
		require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	})

	t.Run("stale partial reservation is removed after one hour", func(t *testing.T) {
		staleCtx, staleProjectID := fixture.newWorkspaceProject(ctx, t, "stale")
		allocation := fixture.createPartialReservation(staleCtx, t, "sample-workspace-stale", staleProjectID, now.Add(-time.Hour-time.Second))
		require.NoError(t, runner.RunOnce(staleCtx))
		record, err := fixture.store.GetSampleInstanceSetup(staleCtx, "sample-workspace-stale")
		require.NoError(t, err)
		require.Nil(t, record)
		require.False(t, fixture.target.databaseExists(staleCtx, allocation.Database))
		require.False(t, fixture.target.roleExists(staleCtx, allocation.Role))
		instance, err := fixture.store.GetInstance(staleCtx, &store.FindInstanceMessage{
			Workspace:   "sample-workspace-stale",
			ResourceID:  ptr(sampleInstanceID("sample-workspace-stale")),
			ShowDeleted: true,
		})
		require.NoError(t, err)
		require.Nil(t, instance)
	})

	t.Run("missing target resources are cleaned idempotently", func(t *testing.T) {
		missingCtx, missingProjectID := fixture.newWorkspaceProject(ctx, t, "missing")
		_, err := fixture.service.PrepareSampleProjectInstance(missingCtx, connect.NewRequest(&v1pb.PrepareSampleProjectInstanceRequest{
			Parent: common.FormatProject(missingProjectID),
		}))
		require.NoError(t, err)
		record := fixture.sampleRecordFor(missingCtx, t, "sample-workspace-missing")
		require.NoError(t, fixture.target.dropResources(missingCtx, testAllocation{
			Database: record.database,
			Role:     record.role,
		}))

		now = now.Add(8 * 24 * time.Hour)
		require.NoError(t, runner.RunOnce(missingCtx))
		require.NotNil(t, fixture.sampleRecordFor(missingCtx, t, "sample-workspace-missing").deletedAt)
	})

	t.Run("cleanup terminates role sessions outside the sample database", func(t *testing.T) {
		sessionCtx, sessionProjectID := fixture.newWorkspaceProject(ctx, t, "external-session")
		_, err := fixture.service.PrepareSampleProjectInstance(sessionCtx, connect.NewRequest(&v1pb.PrepareSampleProjectInstanceRequest{
			Parent: common.FormatProject(sessionProjectID),
		}))
		require.NoError(t, err)
		record := fixture.sampleRecordFor(sessionCtx, t, "sample-workspace-external-session")
		allocation := testAllocation{
			Database: record.database,
			Role:     record.role,
			Password: fixture.allocationFor(sessionCtx, t, "sample-workspace-external-session").Password,
		}
		outside := fixture.target.externalRoleConnection(sessionCtx, t, allocation)
		t.Cleanup(func() { _ = outside.Close(context.Background()) })

		now = now.Add(8 * 24 * time.Hour)
		require.NoError(t, runner.RunOnce(sessionCtx))
		require.Error(t, outside.Ping(sessionCtx))
		require.False(t, fixture.target.databaseExists(sessionCtx, allocation.Database))
		require.False(t, fixture.target.roleExists(sessionCtx, allocation.Role))
	})
}

type sampleProjectInstanceFixture struct {
	store       *store.Store
	service     *InstanceService
	manager     *saas.Manager
	target      *sampleTargetInspector
	workspaceID string
	projectID   string
}

func newSampleProjectInstanceFixture(t *testing.T, clock func() time.Time) (context.Context, *sampleProjectInstanceFixture) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	t.Cleanup(cancel)

	// Only the target is a container: it is a TLS server Bytebase connects to,
	// not metadata.
	target := testcontainer.GetTestTLSPgContainer(ctx, t)
	t.Cleanup(func() { target.Close(context.Background()) })

	metadataDB, stores, _ := testpg.New(t)
	_, err := metadataDB.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('sample-workspace');
		INSERT INTO project (resource_id, workspace, name) VALUES ('sample-project', 'sample-workspace', 'Sample Project');
	`)
	require.NoError(t, err)

	setSampleProjectInstanceTestEnvironment(ctx, t, stores, "sample-workspace")

	licenseService, err := enterprise.NewLicenseService(common.ReleaseModeDev, stores, false, "")
	require.NoError(t, err)
	dbFactory := dbfactory.New(stores, licenseService)
	syncer := schemasync.NewSyncer(stores, dbFactory, licenseService, nil)
	require.NoError(t, prepareSampleTargetBaseline(ctx, target.GetDB()))
	targetURL := tlsPostgresTestURL(target)
	inspector := newSampleTargetInspector(t, targetURL)
	manager, err := saas.NewManager(
		stores,
		targetURL,
		syncer,
		sample.ManagerOptions{Clock: clock, ReplicaID: "replica-a"},
	)
	require.NoError(t, err)
	return context.WithValue(ctx, common.WorkspaceIDContextKey, "sample-workspace"), &sampleProjectInstanceFixture{
		store: stores,
		service: &InstanceService{
			store:          stores,
			profile:        &config.Profile{SaaS: true},
			licenseService: licenseService,
			dbFactory:      dbFactory,
			schemaSyncer:   syncer,
			sampleManager:  manager,
		},
		manager:     manager,
		target:      inspector,
		workspaceID: "sample-workspace",
		projectID:   "sample-project",
	}
}

func tlsPostgresTestURL(container *testcontainer.Container) string {
	return fmt.Sprintf(
		"postgres://postgres:root-password@localhost:%s/postgres?sslmode=verify-full&sslrootcert=%s",
		container.GetPort(),
		url.QueryEscape(container.GetTLSCAPath()),
	)
}

type sampleProjectInstanceRecord struct {
	workspace string
	project   string
	instance  string
	database  string
	role      string
	createdAt time.Time
	expiresAt time.Time
	deletedAt *time.Time
}

func (f *sampleProjectInstanceFixture) sampleRecord(ctx context.Context, t *testing.T) sampleProjectInstanceRecord {
	return f.sampleRecordFor(ctx, t, f.workspaceID)
}

func (f *sampleProjectInstanceFixture) sampleRecordFor(ctx context.Context, t *testing.T, workspaceID string) sampleProjectInstanceRecord {
	t.Helper()
	setup, err := f.store.GetSampleInstanceSetup(ctx, workspaceID)
	require.NoError(t, err)
	require.NotNil(t, setup)
	payload := &storepb.SaaSSampleInstanceSetupPayload{}
	require.NoError(t, common.ProtojsonUnmarshaler.Unmarshal(setup.Payload, payload))
	require.NotNil(t, setup.ExpiresAt)
	record := sampleProjectInstanceRecord{
		workspace: workspaceID,
		project:   payload.ProjectId,
		instance:  payload.InstanceId,
		database:  payload.DatabaseName,
		role:      payload.RoleName,
		createdAt: setup.CreatedAt,
		expiresAt: *setup.ExpiresAt,
		deletedAt: setup.DeletedAt,
	}
	return record
}

func (f *sampleProjectInstanceFixture) sampleRecordCount(ctx context.Context, t *testing.T) int {
	return f.sampleRecordCountFor(ctx, t, "")
}

func (f *sampleProjectInstanceFixture) sampleRecordCountFor(ctx context.Context, t *testing.T, workspaceID string) int {
	t.Helper()
	var count int
	query := "SELECT COUNT(*) FROM sample_instance_setup"
	args := []any{}
	if workspaceID != "" {
		query += " WHERE workspace = $1"
		args = append(args, workspaceID)
	}
	require.NoError(t, f.store.GetDB().QueryRowContext(ctx, query, args...).Scan(&count))
	return count
}

type testAllocation struct {
	InstanceID string
	Database   string
	Role       string
	Password   string
}

func (f *sampleProjectInstanceFixture) allocation(ctx context.Context, t *testing.T) testAllocation {
	return f.allocationFor(ctx, t, f.workspaceID)
}

func (f *sampleProjectInstanceFixture) allocationFor(
	ctx context.Context,
	t *testing.T,
	workspaceID string,
) testAllocation {
	record := f.sampleRecordFor(ctx, t, workspaceID)
	instance, err := f.store.GetInstance(ctx, &store.FindInstanceMessage{
		Workspace:   workspaceID,
		ResourceID:  &record.instance,
		ShowDeleted: true,
	})
	require.NoError(t, err)
	require.NotNil(t, instance)
	for _, dataSource := range instance.Metadata.GetDataSources() {
		if dataSource.GetId() == "admin" {
			return testAllocation{
				InstanceID: record.instance,
				Database:   record.database,
				Role:       record.role,
				Password:   dataSource.GetPassword(),
			}
		}
	}
	require.FailNow(t, "sample instance has no admin data source")
	return testAllocation{}
}

func (f *sampleProjectInstanceFixture) newWorkspaceProject(ctx context.Context, t *testing.T, suffix string) (context.Context, string) {
	t.Helper()
	workspaceID := "sample-workspace-" + suffix
	projectID := "sample-project-" + suffix
	_, err := f.store.GetDB().ExecContext(ctx, `INSERT INTO workspace (resource_id) VALUES ($1)`, workspaceID)
	require.NoError(t, err)
	setSampleProjectInstanceTestEnvironment(ctx, t, f.store, workspaceID)
	_, err = f.store.GetDB().ExecContext(ctx, `
		INSERT INTO project (resource_id, workspace, name) VALUES ($1, $2, $3)
	`, projectID, workspaceID, "Sample Project "+suffix)
	require.NoError(t, err)
	return context.WithValue(ctx, common.WorkspaceIDContextKey, workspaceID), projectID
}

func setSampleProjectInstanceTestEnvironment(ctx context.Context, t *testing.T, stores *store.Store, workspaceID string) {
	t.Helper()
	_, err := stores.UpsertSetting(ctx, &store.SettingMessage{
		Name:      storepb.SettingName_ENVIRONMENT,
		Workspace: workspaceID,
		Value: &storepb.EnvironmentSetting{Environments: []*storepb.EnvironmentSetting_Environment{
			{Title: "Test", Id: "test"},
		}},
	})
	require.NoError(t, err)
}

func (f *sampleProjectInstanceFixture) addProject(ctx context.Context, t *testing.T, workspaceID, suffix string) string {
	t.Helper()
	projectID := "sample-project-" + suffix
	_, err := f.store.GetDB().ExecContext(ctx, `
		INSERT INTO project (resource_id, workspace, name) VALUES ($1, $2, $3)
	`, projectID, workspaceID, "Sample Project "+suffix)
	require.NoError(t, err)
	return projectID
}

func (f *sampleProjectInstanceFixture) assertActiveMetadata(ctx context.Context, t *testing.T, record sampleProjectInstanceRecord) {
	t.Helper()
	var instanceCount, databaseCount int
	require.NoError(t, f.store.GetDB().QueryRowContext(ctx, `
		SELECT COUNT(*) FROM instance
		WHERE workspace = $1 AND resource_id = $2 AND deleted = FALSE
	`, record.workspace, record.instance).Scan(&instanceCount))
	require.NoError(t, f.store.GetDB().QueryRowContext(ctx, `
		SELECT COUNT(*) FROM db
		WHERE project = $1 AND instance = $2 AND name = $3 AND deleted = FALSE
	`, record.project, record.instance, record.database).Scan(&databaseCount))
	require.Equal(t, 1, instanceCount)
	require.Equal(t, 1, databaseCount)
}

func (f *sampleProjectInstanceFixture) createPartialReservation(
	ctx context.Context,
	t *testing.T,
	workspaceID, projectID string,
	createdAt time.Time,
) testAllocation {
	t.Helper()
	payload := &storepb.SaaSSampleInstanceSetupPayload{
		ProjectId:     projectID,
		InstanceId:    sampleInstanceID(workspaceID),
		Title:         "Sample Project Instance",
		EnvironmentId: ptr("test"),
		DatabaseName:  sampleDatabaseName(workspaceID),
		RoleName:      sampleRoleName(workspaceID),
	}
	encoded, err := protojson.Marshal(payload)
	require.NoError(t, err)
	_, created, err := f.store.ReserveSampleInstanceSetup(ctx, &store.SampleInstanceSetupMessage{
		WorkspaceID: workspaceID,
		ReplicaID:   "replica-a",
		Payload:     encoded,
	})
	require.NoError(t, err)
	require.True(t, created)
	allocation := testAllocation{
		InstanceID: payload.InstanceId,
		Database:   payload.DatabaseName,
		Role:       payload.RoleName,
		Password:   "sample-partial-password",
	}
	require.NoError(t, f.target.provision(ctx, allocation))
	instance, err := f.store.CreateInstance(ctx, &store.InstanceMessage{
		Workspace:     workspaceID,
		ProjectID:     &projectID,
		EnvironmentID: ptr("test"),
		ResourceID:    payload.InstanceId,
		Metadata: &storepb.Instance{
			Title:      "Sample Project Instance",
			Engine:     storepb.Engine_POSTGRES,
			Activation: false,
			DataSources: []*storepb.DataSource{{
				Id: "admin", Type: storepb.DataSourceType_ADMIN, Host: f.target.config.Host,
				Port: fmt.Sprint(f.target.config.Port), Database: allocation.Database,
				Username: allocation.Role, Password: allocation.Password, UseSsl: true, VerifyTlsCertificate: true, SslCa: f.target.sslCA,
			}},
			SyncDatabases: &storepb.SyncDatabases{Databases: []string{allocation.Database}},
		},
	})
	require.NoError(t, err)
	_, _, databases, err := f.service.schemaSyncer.SyncInstance(ctx, instance)
	require.NoError(t, err)
	require.Len(t, databases, 1)
	_, err = f.store.GetDB().ExecContext(ctx, `
		UPDATE sample_instance_setup
		SET updated_at = $1
		WHERE workspace = $2
	`, createdAt, workspaceID)
	require.NoError(t, err)
	return allocation
}

func sampleToken(workspaceID string) string {
	sum := sha256.Sum256([]byte(workspaceID))
	return fmt.Sprintf("%x", sum[:16])
}

func sampleInstanceID(workspaceID string) string {
	return "sample-" + sampleToken(workspaceID)
}

func sampleDatabaseName(workspaceID string) string {
	return "bb_sample_" + sampleToken(workspaceID)
}

func sampleRoleName(workspaceID string) string {
	return "bb_sample_role_" + sampleToken(workspaceID)
}

type sampleTargetInspector struct {
	config *pgx.ConnConfig
	sslCA  string
}

func newSampleTargetInspector(t *testing.T, targetURL string) *sampleTargetInspector {
	t.Helper()
	config, err := pgx.ParseConfig(targetURL)
	require.NoError(t, err)
	parsed, err := url.Parse(targetURL)
	require.NoError(t, err)
	sslCA, err := os.ReadFile(parsed.Query().Get("sslrootcert"))
	require.NoError(t, err)
	return &sampleTargetInspector{config: config, sslCA: string(sslCA)}
}

func prepareSampleTargetBaseline(ctx context.Context, db *sql.DB) error {
	for _, statement := range []string{
		"REVOKE ALL PRIVILEGES ON DATABASE postgres FROM PUBLIC",
		"REVOKE ALL PRIVILEGES ON DATABASE template1 FROM PUBLIC",
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

func (t *sampleTargetInspector) connect(ctx context.Context, database, user, password string) (*pgx.Conn, error) {
	config := t.config.Copy()
	if database != "" {
		config.Database = database
	}
	if user != "" {
		config.User = user
		config.Password = password
	}
	return pgx.ConnectConfig(ctx, config)
}

func (t *sampleTargetInspector) roleExists(ctx context.Context, role string) bool {
	conn, err := t.connect(ctx, "", "", "")
	if err != nil {
		return false
	}
	defer conn.Close(ctx)
	var exists bool
	return conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $1)", role).Scan(&exists) == nil && exists
}

func (t *sampleTargetInspector) databaseExists(ctx context.Context, database string) bool {
	conn, err := t.connect(ctx, "", "", "")
	if err != nil {
		return false
	}
	defer conn.Close(ctx)
	var exists bool
	return conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)", database).Scan(&exists) == nil && exists
}

func (t *sampleTargetInspector) provision(ctx context.Context, allocation testAllocation) error {
	admin, err := t.connect(ctx, "", "", "")
	if err != nil {
		return err
	}
	defer admin.Close(ctx)
	if _, err := admin.Exec(ctx, fmt.Sprintf(
		"CREATE ROLE %s LOGIN PASSWORD %s",
		pgx.Identifier{allocation.Role}.Sanitize(),
		quoteTestLiteral(allocation.Password),
	)); err != nil {
		return err
	}
	if _, err := admin.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s", pgx.Identifier{allocation.Database}.Sanitize())); err != nil {
		return err
	}
	if _, err := admin.Exec(ctx, fmt.Sprintf(
		"GRANT CONNECT, CREATE, TEMPORARY ON DATABASE %s TO %s",
		pgx.Identifier{allocation.Database}.Sanitize(),
		pgx.Identifier{allocation.Role}.Sanitize(),
	)); err != nil {
		return err
	}
	database, err := t.connect(ctx, allocation.Database, "", "")
	if err != nil {
		return err
	}
	defer database.Close(ctx)
	_, err = database.Exec(ctx, fmt.Sprintf(
		"GRANT USAGE, CREATE ON SCHEMA public TO %s",
		pgx.Identifier{allocation.Role}.Sanitize(),
	))
	return err
}

func quoteTestLiteral(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func (t *sampleTargetInspector) sampleConnection(ctx context.Context, tst *testing.T, allocation testAllocation) *pgx.Conn {
	tst.Helper()
	conn, err := t.connect(ctx, allocation.Database, allocation.Role, allocation.Password)
	require.NoError(tst, err)
	return conn
}

func (t *sampleTargetInspector) sampleEmployeeCount(ctx context.Context, allocation testAllocation, assert func(int) bool) error {
	conn, err := t.connect(ctx, allocation.Database, allocation.Role, allocation.Password)
	if err != nil {
		return errors.New("sample target cannot be inspected")
	}
	defer conn.Close(ctx)
	var count int
	if err := conn.QueryRow(ctx, "SELECT COUNT(*) FROM employee").Scan(&count); err != nil || !assert(count) {
		return errors.New("sample target seed is not available")
	}
	return nil
}

func (t *sampleTargetInspector) assertSeededOwnershipAndDDL(
	ctx context.Context,
	tst *testing.T,
	allocation testAllocation,
) {
	tst.Helper()
	sample := t.sampleConnection(ctx, tst, allocation)
	defer sample.Close(ctx)

	for _, query := range []string{
		`SELECT nspowner = (SELECT oid FROM pg_roles WHERE rolname = current_user) FROM pg_namespace WHERE nspname = 'bbdataarchive'`,
		`SELECT relowner = (SELECT oid FROM pg_roles WHERE rolname = current_user) FROM pg_class WHERE oid = 'public.employee'::regclass`,
		`SELECT relowner = (SELECT oid FROM pg_roles WHERE rolname = current_user) FROM pg_class WHERE oid = 'public.employee_emp_no_seq'::regclass`,
		`SELECT proowner = (SELECT oid FROM pg_roles WHERE rolname = current_user) FROM pg_proc WHERE oid = 'public.log_dml_operations()'::regprocedure`,
		`SELECT typowner = (SELECT oid FROM pg_roles WHERE rolname = current_user) FROM pg_type WHERE oid = 'public.employee'::regtype`,
	} {
		var owned bool
		require.NoError(tst, sample.QueryRow(ctx, query).Scan(&owned))
		require.True(tst, owned)
	}
	_, err := sample.Exec(ctx, "ALTER TABLE employee ADD COLUMN sample_integration_marker BOOLEAN NOT NULL DEFAULT false")
	require.NoError(tst, err)
	for _, statement := range []string{
		"CREATE SCHEMA sample_integration_ddl",
		"CREATE TABLE sample_integration_ddl.example (id INT PRIMARY KEY)",
		"DROP SCHEMA sample_integration_ddl CASCADE",
	} {
		_, err := sample.Exec(ctx, statement)
		require.NoError(tst, err)
	}
}

func (t *sampleTargetInspector) assertIsolation(ctx context.Context, tst *testing.T, allocation testAllocation) {
	tst.Helper()
	admin, err := t.connect(ctx, "", "", "")
	require.NoError(tst, err)
	defer admin.Close(ctx)

	var canLogin, superuser, createDatabase, createRole, replicate, bypassRLS, publicConnect, publicCreate bool
	require.NoError(tst, admin.QueryRow(ctx, `
		SELECT rolcanlogin, rolsuper, rolcreatedb, rolcreaterole, rolreplication, rolbypassrls
		FROM pg_roles
		WHERE rolname = $1
	`, allocation.Role).Scan(&canLogin, &superuser, &createDatabase, &createRole, &replicate, &bypassRLS))
	require.NoError(tst, admin.QueryRow(ctx, `
		SELECT NOT EXISTS (
			SELECT 1
			FROM pg_database, aclexplode(COALESCE(datacl, acldefault('d', datdba))) AS privilege
			WHERE datname = $1
				AND privilege.grantee = 0
				AND privilege.privilege_type = 'CONNECT'
		)
	`, allocation.Database).Scan(&publicConnect))
	database, err := t.connect(ctx, allocation.Database, "", "")
	require.NoError(tst, err)
	defer database.Close(ctx)
	require.NoError(tst, database.QueryRow(ctx, `
		SELECT NOT EXISTS (
			SELECT 1
			FROM pg_namespace, aclexplode(COALESCE(nspacl, acldefault('n', nspowner))) AS privilege
			WHERE nspname = 'public'
				AND privilege.grantee = 0
				AND privilege.privilege_type = 'CREATE'
		)
	`).Scan(&publicCreate))
	require.True(tst, canLogin)
	require.False(tst, superuser)
	require.False(tst, createDatabase)
	require.False(tst, createRole)
	require.False(tst, replicate)
	require.False(tst, bypassRLS)
	require.True(tst, publicConnect)
	require.True(tst, publicCreate)
}

func (t *sampleTargetInspector) dropResources(ctx context.Context, allocation testAllocation) error {
	admin, err := t.connect(ctx, "", "", "")
	if err != nil {
		return err
	}
	defer admin.Close(ctx)
	if t.databaseExists(ctx, allocation.Database) {
		if _, err := admin.Exec(ctx, fmt.Sprintf("DROP DATABASE %s", pgx.Identifier{allocation.Database}.Sanitize())); err != nil {
			return err
		}
	}
	if t.roleExists(ctx, allocation.Role) {
		if _, err := admin.Exec(ctx, fmt.Sprintf("DROP ROLE %s", pgx.Identifier{allocation.Role}.Sanitize())); err != nil {
			return err
		}
	}
	return nil
}

func (t *sampleTargetInspector) externalRoleConnection(ctx context.Context, tst *testing.T, allocation testAllocation) *pgx.Conn {
	tst.Helper()
	admin, err := t.connect(ctx, "", "", "")
	require.NoError(tst, err)
	defer admin.Close(ctx)
	database := "outside_" + sampleToken(allocation.Role)
	_, err = admin.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s", pgx.Identifier{database}.Sanitize()))
	require.NoError(tst, err)
	tst.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		cleanup, cleanupErr := t.connect(cleanupCtx, "", "", "")
		if cleanupErr == nil {
			defer cleanup.Close(cleanupCtx)
			_, _ = cleanup.Exec(cleanupCtx, fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", pgx.Identifier{database}.Sanitize()))
		}
	})
	conn, err := t.connect(ctx, database, allocation.Role, allocation.Password)
	require.NoError(tst, err)
	return conn
}
