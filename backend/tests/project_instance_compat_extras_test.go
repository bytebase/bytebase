package tests

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/alexmullins/zip"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/bytebase/bytebase/backend/common"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// TestProjectInstanceExport verifies that SQL export accepts the canonical
// project-scoped database name for a database on a project instance and
// rejects cross-project and workspace-form names for the same database.
func TestProjectInstanceExport(t *testing.T) {
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	pg, err := provisionPgInstance(ctx, t)
	a.NoError(err)
	const instanceID = "bot37-export-instance"
	const databaseID = "bot37_export_database"
	createPgDatabase(t, pg, databaseID)

	instance := createProjectInstanceTestInstance(ctx, t, ctl, &ctl.project.Name, instanceID, "export project instance", pg)
	_, err = ctl.instanceServiceClient.SyncInstance(ctx, connect.NewRequest(&v1pb.SyncInstanceRequest{Name: instance.Name}))
	a.NoError(err)
	databaseName := fmt.Sprintf("%s/databases/%s", instance.Name, databaseID)
	database, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{Name: databaseName}))
	a.NoError(err)
	a.Equal(databaseName, database.Msg.Name)
	a.Equal(ctl.project.Name, database.Msg.Project)
	_, err = ctl.databaseServiceClient.SyncDatabase(ctx, connect.NewRequest(&v1pb.SyncDatabaseRequest{Name: databaseName}))
	a.NoError(err)

	export := func(name string) error {
		_, err := ctl.sqlServiceClient.Export(ctx, connect.NewRequest(&v1pb.ExportRequest{
			Name:         name,
			Format:       v1pb.ExportFormat_CSV,
			Statement:    "SELECT 1;",
			DataSourceId: "admin",
		}))
		return err
	}

	// The canonical project-scoped name succeeds and yields a parseable export.
	exportResp, err := ctl.sqlServiceClient.Export(ctx, connect.NewRequest(&v1pb.ExportRequest{
		Name:         databaseName,
		Format:       v1pb.ExportFormat_CSV,
		Statement:    "SELECT 1;",
		DataSourceId: "admin",
	}))
	a.NoError(err)
	zipReader, err := zip.NewReader(bytes.NewReader(exportResp.Msg.Content), int64(len(exportResp.Msg.Content)))
	a.NoError(err)
	a.NotEmpty(zipReader.File)
	foundResult := false
	for _, compressedFile := range zipReader.File {
		if !strings.HasSuffix(compressedFile.Name, ".result.csv") {
			continue
		}
		foundResult = true
		file, err := compressedFile.Open()
		a.NoError(err)
		content, err := io.ReadAll(file)
		a.NoError(err)
		a.Equal("?column?\n1", string(content))
	}
	a.True(foundResult, "export zip must contain a .result.csv file")

	// A cross-project name for the same database is rejected.
	otherProject := createProjectForProjectInstanceTest(ctx, t, ctl, "bot37-export-other-project")
	crossProjectName := fmt.Sprintf("%s/instances/%s/databases/%s", otherProject.Name, instanceID, databaseID)
	err = export(crossProjectName)
	a.Error(err)
	a.Equal(connect.CodeNotFound, connect.CodeOf(err))

	// The workspace-form name for a project-instance database is rejected by
	// the ACL layer (workspace-scoped instance resolution) before the handler
	// runs, so it can never leak through the workspace alias.
	workspaceFormName := fmt.Sprintf("instances/%s/databases/%s", instanceID, databaseID)
	err = export(workspaceFormName)
	a.Error(err)
	a.Equal(connect.CodeNotFound, connect.CodeOf(err))
	a.Contains(err.Error(), fmt.Sprintf("instance %q not found", instanceID))
}

// TestProjectInstanceSavedQuery verifies that saved queries accept the
// canonical project-scoped database name for databases on a project instance
// and reject cross-project references.
func TestProjectInstanceSavedQuery(t *testing.T) {
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	pg, err := provisionPgInstance(ctx, t)
	a.NoError(err)
	const instanceID = "bot37-saved-query-instance"
	const databaseID = "bot37_saved_query_database"
	createPgDatabase(t, pg, databaseID)

	instance := createProjectInstanceTestInstance(ctx, t, ctl, &ctl.project.Name, instanceID, "saved query project instance", pg)
	_, err = ctl.instanceServiceClient.SyncInstance(ctx, connect.NewRequest(&v1pb.SyncInstanceRequest{Name: instance.Name}))
	a.NoError(err)
	databaseName := fmt.Sprintf("%s/databases/%s", instance.Name, databaseID)
	_, err = ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{Name: databaseName}))
	a.NoError(err)

	createSavedQuery := func(database string) (*v1pb.SavedQuery, error) {
		resp, err := ctl.savedQueryServiceClient.CreateSavedQuery(ctx, connect.NewRequest(&v1pb.CreateSavedQueryRequest{
			Parent: ctl.project.Name,
			SavedQuery: &v1pb.SavedQuery{
				Title:    "project instance saved query",
				Content:  []byte("SELECT 1;"),
				Database: database,
			},
		}))
		if err != nil {
			return nil, err
		}
		return resp.Msg, nil
	}

	// The canonical project-scoped name is accepted and retained.
	created, err := createSavedQuery(databaseName)
	a.NoError(err)
	a.Equal(databaseName, created.Database)
	got, err := ctl.savedQueryServiceClient.GetSavedQuery(ctx, connect.NewRequest(&v1pb.GetSavedQueryRequest{Name: created.Name}))
	a.NoError(err)
	a.Equal(databaseName, got.Msg.Database)

	// A project-instance database referenced from a different project is
	// rejected with NotFound rather than leaking across projects.
	otherProject := createProjectForProjectInstanceTest(ctx, t, ctl, "bot37-saved-query-other-project")
	crossProjectName := fmt.Sprintf("%s/instances/%s/databases/%s", otherProject.Name, instanceID, databaseID)
	_, err = createSavedQuery(crossProjectName)
	a.Error(err)
	a.Equal(connect.CodeNotFound, connect.CodeOf(err))
	projectID, projectIDErr := common.GetProjectID(ctl.project.Name)
	a.NoError(projectIDErr)
	a.Contains(err.Error(), fmt.Sprintf("database %q not found in project %q", crossProjectName, projectID))

	// The workspace-form name for a project-instance database is rejected as
	// non-canonical instead of resolving through the workspace alias.
	workspaceFormName := fmt.Sprintf("instances/%s/databases/%s", instanceID, databaseID)
	_, err = createSavedQuery(workspaceFormName)
	a.Error(err)
	a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))
	a.Contains(err.Error(), fmt.Sprintf("database name %q is not canonical for its instance", workspaceFormName))
}

// TestProjectInstanceAccessGrant verifies that access grants targeting a
// project-instance database work inside the owning project and are rejected
// across projects.
func TestProjectInstanceAccessGrant(t *testing.T) {
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	pg, err := provisionPgInstance(ctx, t)
	a.NoError(err)
	const instanceID = "bot37-grant-instance"
	const databaseID = "bot37_grant_database"
	createPgDatabase(t, pg, databaseID)

	instance := createProjectInstanceTestInstance(ctx, t, ctl, &ctl.project.Name, instanceID, "grant project instance", pg)
	_, err = ctl.instanceServiceClient.SyncInstance(ctx, connect.NewRequest(&v1pb.SyncInstanceRequest{Name: instance.Name}))
	a.NoError(err)
	databaseName := fmt.Sprintf("%s/databases/%s", instance.Name, databaseID)
	_, err = ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{Name: databaseName}))
	a.NoError(err)
	_, err = ctl.databaseServiceClient.SyncDatabase(ctx, connect.NewRequest(&v1pb.SyncDatabaseRequest{Name: databaseName}))
	a.NoError(err)

	createGrant := func(parent string, targets []string) (*v1pb.AccessGrant, error) {
		resp, err := ctl.accessGrantServiceClient.CreateAccessGrant(ctx, connect.NewRequest(&v1pb.CreateAccessGrantRequest{
			Parent: parent,
			AccessGrant: &v1pb.AccessGrant{
				Creator:    ctl.principalName,
				Targets:    targets,
				Query:      "SELECT 1",
				Reason:     "bot37 access grant compatibility",
				Expiration: &v1pb.AccessGrant_Ttl{Ttl: durationpb.New(time.Hour)},
			},
		}))
		if err != nil {
			return nil, err
		}
		return resp.Msg, nil
	}

	// A grant targeting the canonical project-scoped database works in the
	// owning project and retains the target.
	grant, err := createGrant(ctl.project.Name, []string{databaseName})
	a.NoError(err)
	a.NotEmpty(grant.Name)
	a.Equal([]string{databaseName}, grant.Targets)

	// The same database targeted from a different project is rejected.
	otherProject := createProjectForProjectInstanceTest(ctx, t, ctl, "bot37-grant-other-project")
	_, err = createGrant(otherProject.Name, []string{databaseName})
	a.Error(err)
	a.Equal(connect.CodeNotFound, connect.CodeOf(err))
	a.Contains(err.Error(), fmt.Sprintf("database %q not found in project %q", databaseName, "bot37-grant-other-project"))

	// A project-scoped target naming a different project is rejected even when
	// requested inside the owning project.
	crossProjectName := fmt.Sprintf("%s/instances/%s/databases/%s", otherProject.Name, instanceID, databaseID)
	_, err = createGrant(ctl.project.Name, []string{crossProjectName})
	a.Error(err)
	a.Equal(connect.CodeNotFound, connect.CodeOf(err))
	projectID, projectIDErr := common.GetProjectID(ctl.project.Name)
	a.NoError(projectIDErr)
	a.Contains(err.Error(), fmt.Sprintf("database %q not found in project %q", crossProjectName, projectID))

	// The workspace-form target for a project-instance database is rejected as
	// non-canonical instead of resolving through the workspace alias.
	workspaceFormName := fmt.Sprintf("instances/%s/databases/%s", instanceID, databaseID)
	_, err = createGrant(ctl.project.Name, []string{workspaceFormName})
	a.Error(err)
	a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))
	a.Contains(err.Error(), fmt.Sprintf("target %q is not canonical for its instance", workspaceFormName))
}

// TestBatchSyncInstancesCompatibility verifies BatchSyncInstances collection
// enforcement with mixed workspace and project instance targets, rejection of
// invalid members with zero side effects, and retained workspace batch
// behavior.
func TestBatchSyncInstancesCompatibility(t *testing.T) {
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	pg, err := provisionPgInstance(ctx, t)
	a.NoError(err)
	const (
		workspaceInstanceID = "bot37-batch-workspace-instance"
		projectInstanceID   = "bot37-batch-project-instance"
		workspaceDB         = "bot37_batch_ws_db"
		workspaceNewDB      = "bot37_batch_ws_new_db"
		projectDB           = "bot37_batch_proj_db"
		projectNewDB        = "bot37_batch_proj_new_db"
	)
	// Create both instances while the server has no physical databases yet:
	// instance creation itself discovers databases, so the "zero side
	// effects" probes below rely on databases created after both instances.
	workspaceInstance := createProjectInstanceTestInstance(ctx, t, ctl, nil, workspaceInstanceID, "batch workspace instance", pg)
	a.Equal("instances/"+workspaceInstanceID, workspaceInstance.Name)
	projectInstance := createProjectInstanceTestInstance(ctx, t, ctl, &ctl.project.Name, projectInstanceID, "batch project instance", pg)
	a.Equal(fmt.Sprintf("%s/instances/%s", ctl.project.Name, projectInstanceID), projectInstance.Name)
	for _, database := range []string{workspaceDB, workspaceNewDB, projectDB, projectNewDB} {
		createPgDatabase(t, pg, database)
	}

	workspaceDBName := fmt.Sprintf("instances/%s/databases/%s", workspaceInstanceID, workspaceDB)
	workspaceNewDBName := fmt.Sprintf("instances/%s/databases/%s", workspaceInstanceID, workspaceNewDB)
	projectDBName := fmt.Sprintf("%s/instances/%s/databases/%s", ctl.project.Name, projectInstanceID, projectDB)
	projectNewDBName := fmt.Sprintf("%s/instances/%s/databases/%s", ctl.project.Name, projectInstanceID, projectNewDB)

	getDatabase := func(name string) error {
		_, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{Name: name}))
		return err
	}
	batchSync := func(parent *string, names ...string) error {
		requests := make([]*v1pb.SyncInstanceRequest, 0, len(names))
		for _, name := range names {
			requests = append(requests, &v1pb.SyncInstanceRequest{Name: name})
		}
		_, err := ctl.instanceServiceClient.BatchSyncInstances(ctx, connect.NewRequest(&v1pb.BatchSyncInstancesRequest{
			Parent:   parent,
			Requests: requests,
		}))
		return err
	}

	// A project instance is not in the workspace collection: the mixed batch is
	// rejected before any member is synced.
	err = batchSync(nil, workspaceInstance.Name, projectInstance.Name)
	a.Error(err)
	a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))
	a.Contains(err.Error(), fmt.Sprintf("instance %q is not in its requested collection", projectInstance.Name))
	a.Error(getDatabase(workspaceNewDBName), "a rejected batch must not sync the workspace member")
	a.Error(getDatabase(workspaceDBName))

	// A workspace instance is not in the project collection either.
	err = batchSync(&ctl.project.Name, projectInstance.Name, workspaceInstance.Name)
	a.Error(err)
	a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))
	a.Contains(err.Error(), fmt.Sprintf("instance %q is not in its requested collection", workspaceInstance.Name))
	a.Error(getDatabase(projectNewDBName), "a rejected batch must not sync the project member")
	a.Error(getDatabase(projectDBName))

	// A nonexistent member aborts the whole batch with zero side effects.
	err = batchSync(nil, workspaceInstance.Name, "instances/bot37-batch-ghost")
	a.Error(err)
	a.Equal(connect.CodeNotFound, connect.CodeOf(err))
	a.Error(getDatabase(workspaceNewDBName), "a batch with an invalid member must not sync the valid member")

	// A functional batch over the exact project collection syncs the project
	// instance and discovers both pre-existing databases.
	a.NoError(batchSync(&ctl.project.Name, projectInstance.Name))
	a.NoError(getDatabase(projectDBName))
	a.NoError(getDatabase(projectNewDBName))

	// Workspace batch behavior is retained: a batch without a parent syncs
	// workspace instances.
	a.NoError(batchSync(nil, workspaceInstance.Name))
	a.NoError(getDatabase(workspaceDBName))
	a.NoError(getDatabase(workspaceNewDBName))
}
