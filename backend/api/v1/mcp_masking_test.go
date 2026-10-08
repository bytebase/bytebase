package v1

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/api/mcp"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/masker"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

// mcpSessionContext is a request as the gate hands it to a handler: a
// delegated grant, and the settings the gate resolved.
func mcpSessionContext() context.Context {
	return withMCPSettings(
		contextWithAuth(&common.AuthContext{DelegatedGrant: &common.DelegatedGrant{}}),
		&storepb.MCPSetting{Capability: storepb.MCPSetting_READ_WRITE})
}

// TestMaskedWriteGuardRefusesTheSentinel covers every door an MCP session can
// put raw SQL through — the three that reach the change pipeline, the two that
// execute directly, and the two that park a statement for a person to run — and
// pins that each refuses on MCP origin alone.
func TestMaskedWriteGuardRefusesTheSentinel(t *testing.T) {
	const clean = "UPDATE employee SET name = 'Bytebase' WHERE id = 1"
	masked := "UPDATE employee SET name = '" + masker.DefaultFullMaskSubstitution + "' WHERE id = 1"

	sheetRequest := func(sql string) any {
		return &v1pb.CreateSheetRequest{Sheet: &v1pb.Sheet{Content: []byte(sql)}}
	}
	batchRequest := func(sql string) any {
		return &v1pb.BatchCreateSheetsRequest{Requests: []*v1pb.CreateSheetRequest{
			{Sheet: &v1pb.Sheet{Content: []byte(clean)}},
			{Sheet: &v1pb.Sheet{Content: []byte(sql)}},
		}}
	}
	releaseRequest := func(sql string) any {
		return &v1pb.CreateReleaseRequest{Release: &v1pb.Release{Files: []*v1pb.Release_File{
			{Statement: []byte(sql)},
		}}}
	}
	queryRequest := func(sql string) any {
		return &v1pb.QueryRequest{Statement: sql}
	}
	exportRequest := func(sql string) any {
		return &v1pb.ExportRequest{Statement: sql}
	}
	savedQueryRequest := func(sql string) any {
		return &v1pb.CreateSavedQueryRequest{SavedQuery: &v1pb.SavedQuery{Content: []byte(sql)}}
	}
	savedQueryUpdateRequest := func(sql string) any {
		// No update mask: the guard reads content wherever it finds it.
		return &v1pb.UpdateSavedQueryRequest{SavedQuery: &v1pb.SavedQuery{Content: []byte(sql)}}
	}

	for _, door := range []struct {
		name    string
		refuse  func(context.Context, any) string
		request func(string) any
	}{
		{name: "CreateSheet", refuse: refuseMaskedWriteSheet, request: sheetRequest},
		{name: "BatchCreateSheets", refuse: refuseMaskedWriteSheetBatch, request: batchRequest},
		{name: "CreateRelease", refuse: refuseMaskedWriteRelease, request: releaseRequest},
		{name: "Query", refuse: refuseMaskedWriteQuery, request: queryRequest},
		{name: "Export", refuse: refuseMaskedWriteExport, request: exportRequest},
		{name: "CreateSavedQuery", refuse: refuseMaskedWriteSavedQuery, request: savedQueryRequest},
		{name: "UpdateSavedQuery", refuse: refuseMaskedWriteSavedQueryUpdate, request: savedQueryUpdateRequest},
	} {
		t.Run(door.name, func(t *testing.T) {
			session := mcpSessionContext()

			reason := door.refuse(session, door.request(masked))
			require.NotEmpty(t, reason, "writing the mask back replaces the real value with it")
			require.Contains(t, reason, masker.DefaultFullMaskSubstitution,
				"the message has to name the literal, or the agent cannot tell what to remove")

			require.Empty(t, door.refuse(session, door.request(clean)),
				"an ordinary change is served")

			unstamped := contextWithAuth(&common.AuthContext{DelegatedGrant: &common.DelegatedGrant{}})
			require.NotEmpty(t, door.refuse(unstamped, door.request(masked)),
				"and it does not depend on the gate having stamped anything")

			human := contextWithAuth(&common.AuthContext{})
			require.Empty(t, door.refuse(human, door.request(masked)),
				"a console session may write whatever it likes")
		})
	}
}

// TestMaskedWriteGuardFailsClosedOnAWiringBug pins the arm that runs when the
// table's procedure key and its request type stop agreeing. The table is keyed
// by procedure so the type is fixed, which is exactly why a mismatch is a bug
// in this file rather than anything about the caller — and a refusal path that
// meets a bug refuses.
func TestMaskedWriteGuardFailsClosedOnAWiringBug(t *testing.T) {
	t.Parallel()
	session := mcpSessionContext()

	for _, refuse := range []func(context.Context, any) string{
		refuseMaskedWriteSheet, refuseMaskedWriteSheetBatch, refuseMaskedWriteRelease,
		refuseMaskedWriteQuery, refuseMaskedWriteExport,
		refuseMaskedWriteSavedQuery, refuseMaskedWriteSavedQueryUpdate,
	} {
		require.NotEmpty(t, refuse(session, &v1pb.GetUserRequest{}))
	}
}

// TestMaskedWriteGuardIsWiredIntoTheGate proves the guard denies before
// dispatch and is recorded, rather than only being a function that returns a
// string. The sheet never reaches the handler, so no partial change is left
// behind for the agent to build on.
func TestMaskedWriteGuardIsWiredIntoTheGate(t *testing.T) {
	masked := "INSERT INTO employee VALUES (2, '" + masker.DefaultFullMaskSubstitution + "')"

	doors := []struct {
		procedure string
		request   connect.AnyRequest
	}{
		{"/bytebase.v1.SheetService/CreateSheet",
			connect.NewRequest(&v1pb.CreateSheetRequest{Sheet: &v1pb.Sheet{Content: []byte(masked)}})},
		{"/bytebase.v1.SheetService/BatchCreateSheets",
			connect.NewRequest(&v1pb.BatchCreateSheetsRequest{Requests: []*v1pb.CreateSheetRequest{
				{Sheet: &v1pb.Sheet{Content: []byte(masked)}}}})},
		{"/bytebase.v1.ReleaseService/CreateRelease",
			connect.NewRequest(&v1pb.CreateReleaseRequest{Release: &v1pb.Release{Files: []*v1pb.Release_File{
				{Statement: []byte(masked)}}}})},
		{"/bytebase.v1.SQLService/Query",
			connect.NewRequest(&v1pb.QueryRequest{Statement: masked})},
		{"/bytebase.v1.SQLService/Export",
			connect.NewRequest(&v1pb.ExportRequest{Statement: masked})},
		{"/bytebase.v1.SavedQueryService/CreateSavedQuery",
			connect.NewRequest(&v1pb.CreateSavedQueryRequest{SavedQuery: &v1pb.SavedQuery{Content: []byte(masked)}})},
		{"/bytebase.v1.SavedQueryService/UpdateSavedQuery",
			connect.NewRequest(&v1pb.UpdateSavedQueryRequest{SavedQuery: &v1pb.SavedQuery{Content: []byte(masked)}})},
	}

	// A door added to the table but not exercised here ships wired to nothing,
	// and the unit test above would still pass. CreateIssue is the one entry in
	// mcpRequestShapeRefusals that is not a masked-write door.
	require.Len(t, doors, len(mcpRequestShapeRefusals)-1,
		"every masked-write door in the refusal table needs a row here")

	for _, door := range doors {
		t.Run(door.procedure, func(t *testing.T) {
			got := invokeMCPGate(t,
				readWriteCeiling(),
				&common.AuthContext{
					MCPMethodClass: v1pb.MCPMethodClass_WRITE,
					DelegatedGrant: &common.DelegatedGrant{},
				},
				door.procedure,
				door.request)

			require.Error(t, got.err)
			require.False(t, got.dispatched, "the refusal must land before anything is written")
			require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(got.err))
			require.Contains(t, got.err.Error(), masker.DefaultFullMaskSubstitution)
			require.True(t, got.auditMarked, "every policy denial is an audited outcome")
		})
	}
}

// TestMaskedWriteGuardCoversTheExecutionDoor is the door that needs no sheet.
// Export is the same door under another name — it takes a raw statement, it is
// WRITE class, and on MySQL the handler skips validateQueryRequest, so a DML
// statement reaches a driver that executes it.
// SQLService/Query is READ class, so every ceiling that opens a session serves
// it, and it authorizes DML per statement — under READ_WRITE an agent can
// overwrite the column it just read masked without proposing anything.
func TestMaskedWriteGuardCoversTheExecutionDoor(t *testing.T) {
	t.Parallel()
	masked := "UPDATE employee SET secret = '" + masker.DefaultFullMaskSubstitution + "' WHERE id = 1"

	got := invokeMCPGate(t,
		readWriteCeiling(),
		&common.AuthContext{
			MCPMethodClass: v1pb.MCPMethodClass_READ,
			DelegatedGrant: &common.DelegatedGrant{},
		},
		"/bytebase.v1.SQLService/Query",
		connect.NewRequest(&v1pb.QueryRequest{Statement: masked}))

	require.Error(t, got.err)
	require.False(t, got.dispatched, "the refusal must land before the statement executes")
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(got.err))
	require.Contains(t, got.err.Error(), masker.DefaultFullMaskSubstitution)
	require.True(t, got.auditMarked)

	exported := invokeMCPGate(t,
		readWriteCeiling(),
		&common.AuthContext{
			MCPMethodClass: v1pb.MCPMethodClass_WRITE,
			DelegatedGrant: &common.DelegatedGrant{},
		},
		"/bytebase.v1.SQLService/Export",
		connect.NewRequest(&v1pb.ExportRequest{Statement: masked}))

	require.Error(t, exported.err)
	require.False(t, exported.dispatched)
	require.Contains(t, exported.err.Error(), masker.DefaultFullMaskSubstitution)
}

// TestMCPRefusalsNameThemselvesToTheQueryTool is a contract between two
// processes that meet only over an error body.
//
// query_database drops its own "request the SQL Editor role" advice when it
// recognizes an MCP policy refusal, because no project role lifts a workspace
// setting. It recognizes one by wording, so every producer's wording has to stay
// inside the list the consumer keys on — including the masked-write guard, which
// is the newest producer and the one whose refusal names a literal rather than a
// policy.
func TestMCPRefusalsNameThemselvesToTheQueryTool(t *testing.T) {
	readWrite := readWriteCeiling()
	maskedWrite := "UPDATE employee SET secret = '" + masker.DefaultFullMaskSubstitution + "'"

	for _, row := range []struct {
		name    string
		stores  mcpGateStore
		class   v1pb.MCPMethodClass
		request connect.AnyRequest
	}{
		{
			name:    "the masked-write guard",
			stores:  readWrite,
			class:   v1pb.MCPMethodClass_READ,
			request: connect.NewRequest(&v1pb.QueryRequest{Statement: maskedWrite}),
		},
		{
			name:    "the ceiling gate",
			stores:  mcpGateStore{ceiling: storepb.MCPSetting_READ_ONLY},
			class:   v1pb.MCPMethodClass_WRITE,
			request: connect.NewRequest(&v1pb.QueryRequest{}),
		},
		{
			name:    "an unclassified method",
			stores:  readWrite,
			class:   v1pb.MCPMethodClass_MCP_METHOD_CLASS_UNSPECIFIED,
			request: connect.NewRequest(&v1pb.QueryRequest{}),
		},
	} {
		t.Run(row.name, func(t *testing.T) {
			got := invokeMCPGate(t, row.stores, &common.AuthContext{
				MCPMethodClass: row.class,
				DelegatedGrant: &common.DelegatedGrant{},
			}, "/bytebase.v1.SQLService/Query", row.request)

			require.Error(t, got.err)
			require.True(t, mcp.IsPolicyRefusal(got.err.Error()),
				"query_database would append project-role advice to %q", got.err.Error())
		})
	}

	// Each producer that refuses outside the gate turns on a fact the request
	// shape cannot carry, so each is checked on its own message.
	clamped := refuseNonReadOnlyStatement(storepb.Engine_POSTGRES, "UPDATE employee SET id = 1")
	require.Error(t, clamped)
	require.True(t, mcp.IsPolicyRefusal(clamped.Error()))

	mcpCtx := contextWithAuth(&common.AuthContext{DelegatedGrant: &common.DelegatedGrant{}})
	grantIssue := rejectMCPOriginatedGrantIssue(mcpCtx, v1pb.Issue_ROLE_GRANT)
	require.Error(t, grantIssue)
	require.True(t, mcp.IsPolicyRefusal(grantIssue.Error()))

	tokenMint := (&AuthService{}).rejectMCPOriginatedTokenMint(mcpCtx, nil, "switch workspaces")
	require.Error(t, tokenMint)
	require.True(t, mcp.IsPolicyRefusal(tokenMint.Error()))

	approvalProject := &store.ProjectMessage{
		ResourceID: "p",
		Setting:    &storepb.Project{RequireIssueApproval: true},
	}
	issueless := rejectMCPOriginatedIssuelessRollout(mcpCtx, approvalProject, nil, "create a rollout")
	require.Error(t, issueless)
	require.True(t, mcp.IsPolicyRefusal(issueless.Error()))
}

// TestMaskPlaceholderMatchIsDeliberatelyBroad records which side of the trade
// the guard sits on. It is a substring scan, so an asterisk banner is refused
// and a placeholder abutting other characters is caught. The first is a message
// the agent can act on; the second is data that would otherwise be gone
// silently.
func TestMaskPlaceholderMatchIsDeliberatelyBroad(t *testing.T) {
	mcpCtx := contextWithAuth(&common.AuthContext{DelegatedGrant: &common.DelegatedGrant{}})
	for _, row := range []struct {
		statement string
		refused   bool
	}{
		{statement: "UPDATE t SET c = '******'", refused: true},
		{statement: "SELECT * FROM t WHERE c = '******'", refused: true},
		{statement: "UPDATE t SET c = '******/2024'", refused: true},
		{statement: "UPDATE t SET c = '************'", refused: true},
		{statement: "/****** Script for SelectTopNRows command ******/ SELECT 1", refused: true},
		{statement: "SELECT '*****'"},
		{statement: "SELECT 1"},
	} {
		t.Run(row.statement, func(t *testing.T) {
			reason := maskedWriteRefusal(mcpCtx, []byte(row.statement))
			require.Equal(t, row.refused, reason != "", reason)
		})
	}
}
