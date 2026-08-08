package tests

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/bytebase/bytebase/backend/common"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// TestSheetProjectScope is the regression test for the global-by-SHA256 sheet
// read: a sheet created in project A must not resolve under project B's name,
// and project B must not be able to mint a reference to A's hash.
func TestSheetProjectScope(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	projectA := ctl.project
	projectBResp, err := ctl.projectServiceClient.CreateProject(ctx, connect.NewRequest(&v1pb.CreateProjectRequest{
		ProjectId: generateRandomString("sheet-scope-b"),
		Project:   &v1pb.Project{Title: "Sheet Scope B"},
	}))
	a.NoError(err)
	projectB := projectBResp.Msg

	content := []byte("SELECT 'sheet scope';")
	sheetResp, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
		Parent: projectA.Name,
		Sheet:  &v1pb.Sheet{Content: content},
	}))
	a.NoError(err)
	_, sheetSha256, err := common.GetProjectResourceIDSheetSha256(sheetResp.Msg.Name)
	a.NoError(err)

	// Readable under the owning project — raw and truncated. The raw read
	// warms the store's hash-keyed content cache.
	for _, raw := range []bool{true, false} {
		resp, err := ctl.sheetServiceClient.GetSheet(ctx, connect.NewRequest(&v1pb.GetSheetRequest{
			Name: sheetResp.Msg.Name,
			Raw:  raw,
		}))
		a.NoError(err, "GetSheet raw=%v under the owning project", raw)
		a.Equal(content, resp.Msg.Content)
	}

	// Uppercase hex in the name resolves to the same sheet: hashes are
	// canonicalized to lowercase at the parse and store boundaries, so case
	// must affect neither the ref check nor the content-cache key.
	upperName := fmt.Sprintf("%s/sheets/%s", projectA.Name, strings.ToUpper(sheetSha256))
	upperResp, err := ctl.sheetServiceClient.GetSheet(ctx, connect.NewRequest(&v1pb.GetSheetRequest{
		Name: upperName,
		Raw:  true,
	}))
	a.NoError(err, "GetSheet with uppercase hex under the owning project")
	a.Equal(content, upperResp.Msg.Content)

	// The same hash under project B is NotFound. The raw variant is the
	// cache-ordering case: the content cache was warmed above, so this fails
	// if enforcement ever runs behind the cache read instead of before it.
	foreignName := fmt.Sprintf("%s/sheets/%s", projectB.Name, sheetSha256)
	for _, raw := range []bool{true, false} {
		_, err := ctl.sheetServiceClient.GetSheet(ctx, connect.NewRequest(&v1pb.GetSheetRequest{
			Name: foreignName,
			Raw:  raw,
		}))
		a.Equal(connect.CodeNotFound, connect.CodeOf(err), "GetSheet raw=%v under a foreign project", raw)
	}

	// Project B cannot mint a reference to A's hash: a release file naming
	// the foreign sheet is rejected as not found.
	crossRelease := &v1pb.Release{
		Type: v1pb.Release_VERSIONED,
		Files: []*v1pb.Release_File{{
			Path:    "V0001__cross.sql",
			Version: "1",
			Sheet:   foreignName,
		}},
	}
	_, err = ctl.releaseServiceClient.CreateRelease(ctx, connect.NewRequest(&v1pb.CreateReleaseRequest{
		Parent:  projectB.Name,
		Release: crossRelease,
	}))
	a.Error(err, "a release in project B must not reference project A's sheet")
	a.Equal(connect.CodeNotFound, connect.CodeOf(err))

	// The owning project can make the same reference.
	_, err = ctl.releaseServiceClient.CreateRelease(ctx, connect.NewRequest(&v1pb.CreateReleaseRequest{
		Parent: projectA.Name,
		Release: &v1pb.Release{
			Type: v1pb.Release_VERSIONED,
			Files: []*v1pb.Release_File{{
				Path:    "V0001__own.sql",
				Version: "1",
				Sheet:   sheetResp.Msg.Name,
			}},
		},
	}))
	a.NoError(err, "the owning project must still reference its own sheet")

	// CheckRelease loads statements from sheet references (CreateRelease does
	// not — it stores only the hash). A sheet holding broken SQL proves the
	// content actually flowed: only a hydrated statement can produce a syntax
	// error advice.
	instance := createPgInstance(ctx, t, ctl, "sheet-scope")
	dbName := "sheet_scope_db"
	err = ctl.createDatabase(ctx, projectA, instance, nil, dbName, "")
	a.NoError(err)
	brokenResp, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
		Parent: projectA.Name,
		Sheet:  &v1pb.Sheet{Content: []byte("SELEC 'broken on purpose';")},
	}))
	a.NoError(err)
	checkResp, err := ctl.releaseServiceClient.CheckRelease(ctx, connect.NewRequest(&v1pb.CheckReleaseRequest{
		Parent: projectA.Name,
		Release: &v1pb.Release{
			Type: v1pb.Release_VERSIONED,
			Files: []*v1pb.Release_File{{
				Path:    "V0002__broken.sql",
				Version: "2",
				Sheet:   brokenResp.Msg.Name,
			}},
		},
		Targets: []string{fmt.Sprintf("%s/databases/%s", instance.Name, dbName)},
	}))
	a.NoError(err)
	foundSyntaxError := false
	for _, result := range checkResp.Msg.Results {
		for _, advice := range result.Advices {
			if advice.Status == v1pb.Advice_ERROR {
				foundSyntaxError = true
			}
		}
	}
	a.True(foundSyntaxError, "the check must run over the sheet's content, not an empty statement")
}

// TestSheetHistoryOnDatabaseTransfer pins the transfer decision in
// docs/design/sheet-history-on-database-transfer.md: the revision list follows
// the database, the statement text does not. After the move the revision is
// listed under the destination project with its sheet named under the stamped
// authoring project, and the destination project gains no read access of its
// own.
func TestSheetHistoryOnDatabaseTransfer(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	projectA := ctl.project
	projectBResp, err := ctl.projectServiceClient.CreateProject(ctx, connect.NewRequest(&v1pb.CreateProjectRequest{
		ProjectId: generateRandomString("sheet-mv-b"),
		Project:   &v1pb.Project{Title: "Sheet Transfer B", AllowSelfApproval: true},
	}))
	a.NoError(err)
	projectB := projectBResp.Msg

	instance := createPgInstance(ctx, t, ctl, "sheet-transfer")
	dbName := "sheet_transfer_db"
	err = ctl.createDatabase(ctx, projectA, instance, nil, dbName, "")
	a.NoError(err)
	databaseName := fmt.Sprintf("%s/databases/%s", instance.Name, dbName)

	// Author a release in A; the file's sheet is created under A with a ref.
	statement := []byte("CREATE TABLE moved (id int);")
	created, sheetName, sheetSha256 := createReleaseRevision(ctx, t, ctl, projectA.Name, databaseName, statement)
	a.Equal(sheetName, created.Sheet, "the sheet is named under the stamped authoring project")

	// Before the transfer the hash does not resolve under project B.
	foreignName := fmt.Sprintf("%s/sheets/%s", projectB.Name, sheetSha256)
	_, err = ctl.sheetServiceClient.GetSheet(ctx, connect.NewRequest(&v1pb.GetSheetRequest{Name: foreignName}))
	a.Equal(connect.CodeNotFound, connect.CodeOf(err))

	// Move the database to project B.
	_, err = ctl.databaseServiceClient.UpdateDatabase(ctx, connect.NewRequest(&v1pb.UpdateDatabaseRequest{
		Database:   &v1pb.Database{Name: databaseName, Project: projectB.Name},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"project"}},
	}))
	a.NoError(err)

	// The revision list is complete under project B, with the sheet still
	// named under project A — the resolved owner, not the database's current
	// project.
	listResp, err := ctl.revisionServiceClient.ListRevisions(ctx, connect.NewRequest(&v1pb.ListRevisionsRequest{
		Parent: databaseName,
	}))
	a.NoError(err)
	a.Len(listResp.Msg.Revisions, 1, "the history list follows the database")
	moved := listResp.Msg.Revisions[0]
	a.Equal(sheetName, moved.Sheet, "the sheet is named under the authoring project, not the database's current project")
	a.Equal(sheetSha256, moved.SheetSha256)

	// A caller with rights on A reads the statement through the owner's name;
	// the transfer granted project B no read access of its own.
	resp, err := ctl.sheetServiceClient.GetSheet(ctx, connect.NewRequest(&v1pb.GetSheetRequest{Name: sheetName, Raw: true}))
	a.NoError(err)
	a.Equal(statement, resp.Msg.Content)
	_, err = ctl.sheetServiceClient.GetSheet(ctx, connect.NewRequest(&v1pb.GetSheetRequest{Name: foreignName}))
	a.Equal(connect.CodeNotFound, connect.CodeOf(err), "a transfer must not grant the destination read access")
}

// createReleaseRevision authors a one-file release in the project and records
// it on the database's history as a revision with release provenance. It
// returns the created revision, the sheet's resource name under the authoring
// project, and the hash.
func createReleaseRevision(ctx context.Context, t *testing.T, ctl *controller, projectName, databaseName string, statement []byte) (*v1pb.Revision, string, string) {
	t.Helper()
	a := require.New(t)

	releaseResp, err := ctl.releaseServiceClient.CreateRelease(ctx, connect.NewRequest(&v1pb.CreateReleaseRequest{
		Parent: projectName,
		Release: &v1pb.Release{
			Type: v1pb.Release_VERSIONED,
			Files: []*v1pb.Release_File{{
				Path:      "V0001__history.sql",
				Version:   "1",
				Statement: statement,
			}},
		},
	}))
	a.NoError(err)
	file := releaseResp.Msg.Files[0]

	revResp, err := ctl.revisionServiceClient.BatchCreateRevisions(ctx, connect.NewRequest(&v1pb.BatchCreateRevisionsRequest{
		Parent: databaseName,
		Requests: []*v1pb.CreateRevisionRequest{{
			Parent: databaseName,
			Revision: &v1pb.Revision{
				Sheet:   file.Sheet,
				Version: file.Version,
				Type:    v1pb.Revision_VERSIONED,
				Release: releaseResp.Msg.Name,
				File:    common.FormatReleaseFile(releaseResp.Msg.Name, file.Path),
			},
		}},
	}))
	a.NoError(err)
	a.Len(revResp.Msg.Revisions, 1)
	return revResp.Msg.Revisions[0], file.Sheet, file.SheetSha256
}

// TestSheetHistoryAfterOwnerPurge covers the path most likely to regress:
// project purge reassigns workspace-instance databases to the default project
// and hard-deletes the purged project's rows and sheet refs. The revision
// list stays complete and the sheet stays named under the stamped authoring
// project, but the name no longer resolves — the project and its refs are
// gone, so the content is unreadable by anyone.
func TestSheetHistoryAfterOwnerPurge(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	purgedResp, err := ctl.projectServiceClient.CreateProject(ctx, connect.NewRequest(&v1pb.CreateProjectRequest{
		ProjectId: generateRandomString("sheet-purge"),
		Project:   &v1pb.Project{Title: "Sheet Purge", AllowSelfApproval: true},
	}))
	a.NoError(err)
	purged := purgedResp.Msg

	instance := createPgInstance(ctx, t, ctl, "sheet-purge")
	dbName := "sheet_purge_db"
	err = ctl.createDatabase(ctx, purged, instance, nil, dbName, "")
	a.NoError(err)
	databaseName := fmt.Sprintf("%s/databases/%s", instance.Name, dbName)

	created, createdSheetName, sheetSha256 := createReleaseRevision(ctx, t, ctl, purged.Name, databaseName, []byte("CREATE TABLE gone (id int);"))
	a.Equal(createdSheetName, created.Sheet)

	// Project purge is an explicit archive-then-purge lifecycle.
	_, err = ctl.projectServiceClient.DeleteProject(ctx,
		connect.NewRequest(&v1pb.DeleteProjectRequest{Name: purged.Name}))
	a.NoError(err)
	_, err = ctl.projectServiceClient.DeleteProject(ctx,
		connect.NewRequest(&v1pb.DeleteProjectRequest{Name: purged.Name, Purge: true}))
	a.NoError(err)

	// The workspace-instance database now belongs to the default project and
	// its history is intact. The sheet keeps the stamped authoring project's
	// name, which now dangles: reading through it is NotFound.
	listResp, err := ctl.revisionServiceClient.ListRevisions(ctx, connect.NewRequest(&v1pb.ListRevisionsRequest{
		Parent: databaseName,
	}))
	a.NoError(err)
	a.Len(listResp.Msg.Revisions, 1, "the history list survives the purge")
	rev := listResp.Msg.Revisions[0]
	a.Equal(createdSheetName, rev.Sheet, "the sheet stays named under the stamped authoring project")
	a.Equal(sheetSha256, rev.SheetSha256)
	_, err = ctl.sheetServiceClient.GetSheet(ctx, connect.NewRequest(&v1pb.GetSheetRequest{Name: rev.Sheet}))
	a.Equal(connect.CodeNotFound, connect.CodeOf(err), "the purged owner's name no longer resolves")

	// The purged project's refs are gone; the content is unreadable by
	// anyone, including the default project now holding the database.
	database, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{Name: databaseName}))
	a.NoError(err)
	_, err = ctl.sheetServiceClient.GetSheet(ctx, connect.NewRequest(&v1pb.GetSheetRequest{
		Name: fmt.Sprintf("%s/sheets/%s", database.Msg.Project, sheetSha256),
	}))
	a.Equal(connect.CodeNotFound, connect.CodeOf(err))
}

// TestCollision_SheetWrite verifies that sheet writes in project A —
// including one whose content, and therefore sha256, is identical to a sheet
// in project B — never disturb project B's sheet refs. The shared blob is the
// collision: (project, sha256) rows differ only in the project column.
func TestCollision_SheetWrite(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	fixture := setupCollidingProjects(ctx, t, ctl)

	beforeB := snapshotProject(ctx, t, ctl, fixture.ProjectB)
	a.NotEmpty(beforeB.SheetBlobRefs, "the fixture seeds a sheet per project")

	// The fixture's sheets in A and B share the exact content "SELECT 1;",
	// so re-creating it in A collides with B's ref on the shared blob.
	// The second sheet is unique to A.
	sharedContent := "SELECT 1;"
	for _, content := range []string{sharedContent, "SELECT 'only A';"} {
		_, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
			Parent: fixture.ProjectA.Name,
			Sheet:  &v1pb.Sheet{Content: []byte(content)},
		}))
		a.NoError(err, "CreateSheet(%s)", content)
	}

	// Read A's shared-content sheet through the public API.
	sharedHash := sha256.Sum256([]byte(sharedContent))
	_, err = ctl.sheetServiceClient.GetSheet(ctx, connect.NewRequest(&v1pb.GetSheetRequest{
		Name: fmt.Sprintf("%s/sheets/%s", fixture.ProjectA.Name, hex.EncodeToString(sharedHash[:])),
	}))
	a.NoError(err)

	afterB := snapshotProject(ctx, t, ctl, fixture.ProjectB)
	assertProjectUnchanged(t, beforeB, afterB, "project B after sheet writes in project A")
}
