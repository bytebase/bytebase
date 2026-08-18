package v1

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/permission"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/iam"
	"github.com/bytebase/bytebase/backend/component/sampleprojectinstance"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/migrator"
	_ "github.com/bytebase/bytebase/backend/plugin/db/pg"
	sampleprojectinstancerunner "github.com/bytebase/bytebase/backend/runner/sampleprojectinstance"
	"github.com/bytebase/bytebase/backend/runner/schemasync"
	"github.com/bytebase/bytebase/backend/store"
)

func TestPrepareSampleProjectInstanceLifecycle(t *testing.T) {
	now := time.Date(2026, time.August, 17, 12, 0, 0, 0, time.UTC)
	ctx, fixture := newSampleProjectInstanceFixture(t, func() time.Time { return now })
	runner := sampleprojectinstancerunner.NewRunner(fixture.manager, sampleprojectinstancerunner.Options{
		Clock: func() time.Time { return now },
	})

	prepared, err := fixture.service.PrepareSampleProjectInstance(ctx, connect.NewRequest(&v1pb.PrepareSampleProjectInstanceRequest{
		Parent: common.FormatProject(fixture.projectID),
	}))
	require.NoError(t, err)
	require.Equal(t, "Sample Project Instance", prepared.Msg.Title)
	require.Equal(t, v1pb.Engine_POSTGRES, prepared.Msg.Engine)
	require.Equal(t, "environments/test", prepared.Msg.GetEnvironment())
	require.True(t, prepared.Msg.Activation)

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
	require.False(t, instance.Deleted)
	database, err = fixture.store.GetDatabase(ctx, &store.FindDatabaseMessage{
		Workspace:    fixture.workspaceID,
		ProjectID:    &fixture.projectID,
		InstanceID:   &record.instance,
		DatabaseName: &record.database,
	})
	require.NoError(t, err)
	require.NotNil(t, database)
	require.False(t, database.Deleted)

	_, err = fixture.service.PrepareSampleProjectInstance(ctx, connect.NewRequest(&v1pb.PrepareSampleProjectInstanceRequest{
		Parent: common.FormatProject(fixture.projectID),
	}))
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
}

func TestPrepareSampleProjectInstanceAdditionalLifecycleCoverage(t *testing.T) {
	now := time.Date(2026, time.August, 17, 12, 0, 0, 0, time.UTC)
	ctx, fixture := newSampleProjectInstanceFixture(t, func() time.Time { return now })
	runner := sampleprojectinstancerunner.NewRunner(fixture.manager, sampleprojectinstancerunner.Options{
		Clock: func() time.Time { return now },
	})

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

	t.Run("Cloud-only gating", func(t *testing.T) {
		fixture.service.profile.SaaS = false
		_, err := fixture.service.PrepareSampleProjectInstance(ctx, connect.NewRequest(&v1pb.PrepareSampleProjectInstanceRequest{
			Parent: common.FormatProject(fixture.projectID),
		}))
		require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
		fixture.service.profile.SaaS = true
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
		record, err := fixture.store.GetSampleProjectInstance(staleCtx, "sample-workspace-stale")
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
		require.NoError(t, fixture.target.dropResources(missingCtx, sampleprojectinstance.Allocation{
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
		allocation := sampleprojectinstance.Allocation{
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
	store        *store.Store
	service      *InstanceService
	manager      *sampleprojectinstance.Manager
	sampleTarget *sampleprojectinstance.Target
	target       *sampleTargetInspector
	workspaceID  string
	projectID    string
}

func newSampleProjectInstanceFixture(t *testing.T, clock func() time.Time) (context.Context, *sampleProjectInstanceFixture) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	t.Cleanup(cancel)

	metadata := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { metadata.Close(context.Background()) })
	target := testcontainer.GetTestTLSPgContainer(ctx, t)
	t.Cleanup(func() { target.Close(context.Background()) })

	require.NoError(t, migrator.MigrateSchema(ctx, metadata.GetDB()))
	_, err := metadata.GetDB().ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('sample-workspace');
		INSERT INTO project (resource_id, workspace, name) VALUES ('sample-project', 'sample-workspace', 'Sample Project');
	`)
	require.NoError(t, err)

	metadataURL := postgresTestURL(metadata)
	stores, err := store.New(ctx, metadataURL, false)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, stores.Close()) })

	licenseService, err := enterprise.NewLicenseService(common.ReleaseModeDev, stores, false, "")
	require.NoError(t, err)
	dbFactory := dbfactory.New(stores, licenseService)
	syncer := schemasync.NewSyncer(stores, dbFactory, licenseService)
	require.NoError(t, prepareSampleTargetBaseline(ctx, target.GetDB()))
	targetURL := tlsPostgresTestURL(target)
	sampleTarget, err := sampleprojectinstance.NewTarget(targetURL)
	require.NoError(t, err)
	inspector := newSampleTargetInspector(t, targetURL)
	manager := sampleprojectinstance.NewManagerFromURL(
		stores,
		targetURL,
		syncer,
		sampleprojectinstance.ManagerOptions{Clock: clock},
	)
	return context.WithValue(ctx, common.WorkspaceIDContextKey, "sample-workspace"), &sampleProjectInstanceFixture{
		store: stores,
		service: &InstanceService{
			store:                stores,
			profile:              &config.Profile{SaaS: true},
			licenseService:       &instanceLicenseServiceStub{instanceLimit: 10, activatedInstanceLimit: 10},
			dbFactory:            dbFactory,
			schemaSyncer:         syncer,
			sampleProjectManager: manager,
		},
		manager:      manager,
		sampleTarget: sampleTarget,
		target:       inspector,
		workspaceID:  "sample-workspace",
		projectID:    "sample-project",
	}
}

func tlsPostgresTestURL(container *testcontainer.Container) string {
	return fmt.Sprintf(
		"postgres://postgres:root-password@localhost:%s/postgres?sslmode=verify-full&sslrootcert=%s",
		container.GetPort(),
		url.QueryEscape(container.GetTLSCAPath()),
	)
}

func postgresTestURL(container *testcontainer.Container) string {
	return fmt.Sprintf(
		"postgres://postgres:root-password@%s:%s/postgres?sslmode=disable",
		container.GetHost(),
		container.GetPort(),
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
	record := sampleProjectInstanceRecord{}
	require.NoError(t, f.store.GetDB().QueryRowContext(ctx, `
		SELECT workspace, project, instance, db_name, role_name, created_at, expires_at, deleted_at
		FROM sample_project_instance
		WHERE workspace = $1
	`, workspaceID).Scan(
		&record.workspace,
		&record.project,
		&record.instance,
		&record.database,
		&record.role,
		&record.createdAt,
		&record.expiresAt,
		&record.deletedAt,
	))
	return record
}

func (f *sampleProjectInstanceFixture) sampleRecordCount(ctx context.Context, t *testing.T) int {
	return f.sampleRecordCountFor(ctx, t, "")
}

func (f *sampleProjectInstanceFixture) sampleRecordCountFor(ctx context.Context, t *testing.T, workspaceID string) int {
	t.Helper()
	var count int
	query := "SELECT COUNT(*) FROM sample_project_instance"
	args := []any{}
	if workspaceID != "" {
		query += " WHERE workspace = $1"
		args = append(args, workspaceID)
	}
	require.NoError(t, f.store.GetDB().QueryRowContext(ctx, query, args...).Scan(&count))
	return count
}

func (f *sampleProjectInstanceFixture) allocation(ctx context.Context, t *testing.T) sampleprojectinstance.Allocation {
	return f.allocationFor(ctx, t, f.workspaceID)
}

func (f *sampleProjectInstanceFixture) allocationFor(
	ctx context.Context,
	t *testing.T,
	workspaceID string,
) sampleprojectinstance.Allocation {
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
			return sampleprojectinstance.Allocation{
				Database: record.database,
				Role:     record.role,
				Password: dataSource.GetPassword(),
			}
		}
	}
	require.FailNow(t, "sample instance has no admin data source")
	return sampleprojectinstance.Allocation{}
}

func (f *sampleProjectInstanceFixture) newWorkspaceProject(ctx context.Context, t *testing.T, suffix string) (context.Context, string) {
	t.Helper()
	workspaceID := "sample-workspace-" + suffix
	projectID := "sample-project-" + suffix
	_, err := f.store.GetDB().ExecContext(ctx, `INSERT INTO workspace (resource_id) VALUES ($1)`, workspaceID)
	require.NoError(t, err)
	_, err = f.store.GetDB().ExecContext(ctx, `
		INSERT INTO project (resource_id, workspace, name) VALUES ($1, $2, $3)
	`, projectID, workspaceID, "Sample Project "+suffix)
	require.NoError(t, err)
	return context.WithValue(ctx, common.WorkspaceIDContextKey, workspaceID), projectID
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
) sampleprojectinstance.Allocation {
	t.Helper()
	record, created, err := f.store.ReserveSampleProjectInstance(ctx, &store.SampleProjectInstanceMessage{
		WorkspaceID: workspaceID,
		ProjectID:   projectID,
		InstanceID:  sampleInstanceID(workspaceID),
		DBName:      sampleDatabaseName(workspaceID),
		RoleName:    sampleRoleName(workspaceID),
	})
	require.NoError(t, err)
	require.True(t, created)
	allocation := sampleprojectinstance.Allocation{
		Database: record.DBName,
		Role:     record.RoleName,
		Password: "sample-partial-password",
	}
	require.NoError(t, f.sampleTarget.Provision(ctx, allocation))
	config, err := f.sampleTarget.InstanceConfig(allocation)
	require.NoError(t, err)
	instance, err := f.store.CreateInstance(ctx, &store.InstanceMessage{
		Workspace:     workspaceID,
		ProjectID:     &projectID,
		EnvironmentID: ptr("test"),
		ResourceID:    record.InstanceID,
		Metadata: &storepb.Instance{
			Title:      "Sample Project Instance",
			Engine:     storepb.Engine_POSTGRES,
			Activation: true,
			DataSources: []*storepb.DataSource{
				config.AdminDataSource,
			},
			SyncDatabases: &storepb.SyncDatabases{Databases: config.SyncDatabaseNames},
		},
	})
	require.NoError(t, err)
	_, _, databases, err := f.service.schemaSyncer.SyncInstance(ctx, instance)
	require.NoError(t, err)
	require.Len(t, databases, 1)
	_, err = f.store.GetDB().ExecContext(ctx, `
		UPDATE sample_project_instance
		SET created_at = $1
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
}

func newSampleTargetInspector(t *testing.T, targetURL string) *sampleTargetInspector {
	t.Helper()
	config, err := pgx.ParseConfig(targetURL)
	require.NoError(t, err)
	return &sampleTargetInspector{config: config}
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

func (t *sampleTargetInspector) sampleConnection(ctx context.Context, tst *testing.T, allocation sampleprojectinstance.Allocation) *pgx.Conn {
	tst.Helper()
	conn, err := t.connect(ctx, allocation.Database, allocation.Role, allocation.Password)
	require.NoError(tst, err)
	return conn
}

func (t *sampleTargetInspector) sampleEmployeeCount(ctx context.Context, allocation sampleprojectinstance.Allocation, assert func(int) bool) error {
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
	allocation sampleprojectinstance.Allocation,
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

func (t *sampleTargetInspector) assertIsolation(ctx context.Context, tst *testing.T, allocation sampleprojectinstance.Allocation) {
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

func (t *sampleTargetInspector) dropResources(ctx context.Context, allocation sampleprojectinstance.Allocation) error {
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

func (t *sampleTargetInspector) externalRoleConnection(ctx context.Context, tst *testing.T, allocation sampleprojectinstance.Allocation) *pgx.Conn {
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
