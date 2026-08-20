package tests

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// queryToolResult is what an agent gets back from query_database: the text a
// model reads, the structured output a client can act on, and whether the tool
// reported a failure.
type queryToolResult struct {
	text     string
	output   QueryToolOutput
	isError  bool
	rawWhole string
}

// QueryToolOutput mirrors the tool's structured output. Only the fields these
// tests assert on are named.
type QueryToolOutput struct {
	Rows                [][]any `json:"rows"`
	RowCount            int     `json:"rowCount"`
	ReadOnlyEnforcement string  `json:"readOnlyEnforcement"`
}

// queryDatabaseOnSession runs the query_database tool the way a client would.
func queryDatabaseOnSession(ctx context.Context, t *testing.T, session *mcp.ClientSession, database, statement string) queryToolResult {
	t.Helper()
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "query_database",
		Arguments: map[string]any{
			"database":  database,
			"statement": statement,
		},
	})
	require.NoError(t, err)

	out := queryToolResult{isError: result.IsError}
	var sb strings.Builder
	for _, content := range result.Content {
		if text, ok := content.(*mcp.TextContent); ok {
			sb.WriteString(text.Text)
		}
	}
	out.text = sb.String()
	if result.StructuredContent != nil {
		raw, err := json.Marshal(result.StructuredContent)
		require.NoError(t, err)
		out.rawWhole = string(raw)
		require.NoError(t, json.Unmarshal(raw, &out.output))
	}
	return out
}

// mcpClampFixture is one live server with one Postgres database holding one
// row, reachable through one open MCP session belonging to the workspace admin.
//
// The principal is deliberately the admin: RBAC would let this session write,
// so anything that refuses a write below is the ceiling and nothing else.
type mcpClampFixture struct {
	ctl *controller
	ctx context.Context
	// database is the full resource name, for the API; name is the short one
	// query_database resolves on.
	database string
	name     string
	session  *mcp.ClientSession
	token    string
}

func setupMCPClampFixture(ctx context.Context, t *testing.T) *mcpClampFixture {
	t.Helper()
	a := require.New(t)
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	t.Cleanup(func() { ctl.Close(ctx) })

	container, err := provisionPgInstance(ctx, t)
	a.NoError(err)
	instanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("mcp-clamp"),
		Instance: &v1pb.Instance{
			Title:       "MCP clamp",
			Engine:      v1pb.Engine_POSTGRES,
			Environment: new("environments/prod"),
			Activation:  true,
			DataSources: []*v1pb.DataSource{container.adminDataSource()},
		},
	}))
	a.NoError(err)

	databaseName := generateRandomString("clampdb")
	a.NoError(ctl.createDatabase(ctx, ctl.project, instanceResp.Msg, nil, databaseName, ""))
	databaseResp, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instanceResp.Msg.Name, databaseName),
	}))
	a.NoError(err)

	setup, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
		Parent: ctl.project.Name,
		Sheet: &v1pb.Sheet{Content: []byte(
			`CREATE TABLE employee(id INT PRIMARY KEY, name TEXT); INSERT INTO employee VALUES (1, 'Bytebase'); CREATE SEQUENCE employee_seq;`)},
	}))
	a.NoError(err)
	a.NoError(ctl.changeDatabase(ctx, ctl.project, databaseResp.Msg, setup.Msg, false))

	a.NoError(ctl.setMCPCapability(ctx, v1pb.WorkspaceProfileSetting_READ_ONLY))
	token, _ := mintMCPOAuthToken(t, ctl, ctl.authInterceptor.token)
	session := openMCPSession(ctx, t, ctl, token)
	t.Cleanup(func() { session.Close() })

	return &mcpClampFixture{
		ctl:      ctl,
		ctx:      ctx,
		database: databaseResp.Msg.Name,
		name:     databaseName,
		session:  session,
		token:    token,
	}
}

// employeeCount reads the row count back on the human path, which is a
// different session from the agent's and is not clamped — so a write that
// slipped through is caught by the database's state rather than by the API
// that was supposed to refuse it.
func (f *mcpClampFixture) employeeCount(t *testing.T) int {
	t.Helper()
	resp, err := f.ctl.sqlServiceClient.Query(f.ctx, connect.NewRequest(&v1pb.QueryRequest{
		Name:      f.database,
		Statement: "SELECT count(*) FROM employee",
	}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.Results, 1)
	require.Empty(t, resp.Msg.Results[0].Error)
	return int(resp.Msg.Results[0].Rows[0].Values[0].GetInt64Value())
}

// TestMCPReadOnlyCeilingRefusesAWrite is the exit criterion of the whole
// series, as an agent meets it: a live MCP session under a read-only workspace
// ceiling, held by a principal RBAC would let write, running an INSERT through
// query_database.
//
// The control at the end is what makes it about the ceiling: the same session
// and the same principal run the same INSERT under READ_WRITE and it lands.
func TestMCPReadOnlyCeilingRefusesAWrite(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	f := setupMCPClampFixture(context.Background(), t)

	// A read is served, and the response says what held it to reads. Postgres
	// gets the driver session too, so the strongest depth is what it reports.
	read := queryDatabaseOnSession(f.ctx, t, f.session, f.name, "SELECT id, name FROM employee")
	a.False(read.isError, "a read must be served under a read-only ceiling: %s", read.text)
	a.Equal(1, read.output.RowCount)
	a.Equal("STATEMENT_CLASSIFICATION_AND_READ_ONLY_SESSION", read.output.ReadOnlyEnforcement,
		"postgres opens the session read-only, so the disclosure must say so")
	a.Contains(read.text, "the database connection itself was opened read-only")

	// The write is refused, with the message the gate set the shape for.
	write := queryDatabaseOnSession(f.ctx, t, f.session, f.name,
		"INSERT INTO employee VALUES (2, 'agent')")
	a.True(write.isError, "an INSERT must be refused under a read-only ceiling: %s", write.text)
	a.Contains(write.text, "READ_ONLY", "the denial must name the ceiling that refused")
	a.Contains(write.text, "the statement is not a read", "the denial must say what was wrong with the request")
	a.Contains(write.text, "raise the MCP ceiling", "the denial must name the way out")
	a.Contains(write.text, "Bytebase console", "the denial must name where the human can do it instead")

	// Refused before execution, not after: the database is untouched.
	a.Equal(1, f.employeeCount(t), "the refused INSERT must not have reached the database")

	// The depth layer, asserted where only a real read-only session can
	// satisfy it. nextval writes, but it writes from inside a structural
	// SELECT, so the classifier calls it a read and the clamp admits it — the
	// database is what refuses, and only because the connection was opened
	// read-only. Drop the clamp term from ConnectionContext.ReadOnly and this
	// is the assertion that goes red.
	sequence := queryDatabaseOnSession(f.ctx, t, f.session, f.name, "SELECT nextval('employee_seq')")
	a.True(sequence.isError, "the read-only session must refuse a write the classifier admitted: %s", sequence.text)
	a.Contains(sequence.text, "read-only transaction",
		"the refusal must come from the database, not from the classifier")

	// The same statement on the human path proves the refusal was this
	// session's depth and not something about the statement. Query reports a
	// statement failure inside the result rather than as an RPC error, so the
	// advancing sequence is what actually distinguishes the two.
	before := f.sequenceValue(t)
	human, err := f.ctl.sqlServiceClient.Query(f.ctx, connect.NewRequest(&v1pb.QueryRequest{
		Name:      f.database,
		Statement: "SELECT nextval('employee_seq')",
	}))
	a.NoError(err)
	a.Len(human.Msg.Results, 1)
	a.Empty(human.Msg.Results[0].Error,
		"a person in the console runs the same statement without a read-only session")
	a.Greater(f.sequenceValue(t), before, "the human's nextval must actually have advanced the sequence")

	// The refusal is recorded, and attributable to the agent rather than to the
	// same human working in the console.
	// Query is a database-scoped method, so its rows are parented to the
	// project rather than to the workspace.
	rows := deniedMCPRows(f.ctx, t, f.ctl, f.ctl.project.Name, "/bytebase.v1.SQLService/Query")
	a.NotEmpty(rows, "a clamp denial must be visible to an operator with MCP provenance")
	var denied *v1pb.AuditLog
	for _, row := range rows {
		if row.Status != nil && row.Status.Code != 0 {
			denied = row
			break
		}
	}
	a.NotNil(denied, "the denied query must have produced a row of its own")
	a.Contains(denied.Status.Message, "READ_ONLY")

	// The control. Same principal, same credential, same statement; only the
	// ceiling changed, and the INSERT now lands. It runs on a session opened
	// after the widening, because a ceiling is read per request and this test
	// is about the ceiling rather than about session lifetime —
	// TestMCPReadOnlyTighteningBitesAnOpenSession covers the live-session half.
	a.NoError(f.ctl.setMCPCapability(f.ctx, v1pb.WorkspaceProfileSetting_READ_WRITE))
	widened := openMCPSession(f.ctx, t, f.ctl, f.token)
	defer widened.Close()
	allowed := queryDatabaseOnSession(f.ctx, t, widened, f.name,
		"INSERT INTO employee VALUES (2, 'agent')")
	a.False(allowed.isError, "under READ_WRITE the same principal must write: %s", allowed.text)
	a.Empty(allowed.output.ReadOnlyEnforcement, "an unclamped session must not be told it was held to reads")
	a.Equal(2, f.employeeCount(t))
}

// sequenceValue reads the sequence back on the human path, for the same reason
// employeeCount does.
func (f *mcpClampFixture) sequenceValue(t *testing.T) int64 {
	t.Helper()
	resp, err := f.ctl.sqlServiceClient.Query(f.ctx, connect.NewRequest(&v1pb.QueryRequest{
		Name: f.database,
		// last_value alone is 1 both before and after the first nextval; the
		// is_called flag is what separates them, so this counts calls: 0
		// before any, N after N.
		Statement: "SELECT CASE WHEN is_called THEN last_value ELSE 0 END FROM employee_seq",
	}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.Results, 1)
	require.Empty(t, resp.Msg.Results[0].Error)
	return resp.Msg.Results[0].Rows[0].Values[0].GetInt64Value()
}

// TestMCPReadOnlyCeilingRefusesASessionRewrite is the escape the two layers
// would otherwise leave open between them.
//
// Both statements classify as reads: SET is not a write, and nextval writes
// from inside a structural SELECT, which no classifier catches. Every statement
// of a request runs on one connection, and Postgres applies a changed
// default_transaction_read_only to the next transaction — so admitting the SET
// would hand the second statement a session with the depth switched off, and
// the sequence would advance under a ceiling that had just reported the
// connection was opened read-only.
//
// Refusing a statement that rewrites its session is what closes it, and the
// sequence value is what proves nothing ran.
func TestMCPReadOnlyCeilingRefusesASessionRewrite(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	f := setupMCPClampFixture(context.Background(), t)
	before := f.sequenceValue(t)

	disarm := queryDatabaseOnSession(f.ctx, t, f.session, f.name,
		"SET default_transaction_read_only = off; SELECT nextval('employee_seq')")
	a.True(disarm.isError, "a request that rewrites its own session must be refused: %s", disarm.text)
	a.Contains(disarm.text, "rewrites the session it runs on")
	a.Contains(disarm.text, "READ_ONLY")

	a.Equal(before, f.sequenceValue(t),
		"neither statement may run: the sequence must not have advanced")

	// The rewrite alone is refused too, so the rule is about the statement and
	// not about what follows it.
	alone := queryDatabaseOnSession(f.ctx, t, f.session, f.name,
		"SET default_transaction_read_only = off")
	a.True(alone.isError, "a bare session rewrite is refused as well: %s", alone.text)
}

// TestMCPReadOnlyCeilingJudgesTheWholeBatch pins the batch rule: a request is
// served only if every statement in it reads, and one write refuses all of it
// rather than running the reads that came before.
func TestMCPReadOnlyCeilingJudgesTheWholeBatch(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	f := setupMCPClampFixture(context.Background(), t)

	reads := queryDatabaseOnSession(f.ctx, t, f.session, f.name,
		"SELECT id FROM employee; SELECT name FROM employee;")
	a.False(reads.isError, "an all-read batch must be served whole: %s", reads.text)

	mixed := queryDatabaseOnSession(f.ctx, t, f.session, f.name,
		"SELECT id FROM employee; DELETE FROM employee; SELECT name FROM employee;")
	a.True(mixed.isError, "a batch holding a write must be refused: %s", mixed.text)
	a.Contains(mixed.text, "statement 2 of 3 is not a read",
		"the denial must name which statement refused the batch")

	a.Equal(1, f.employeeCount(t), "no part of a refused batch may run")
}

// TestMCPReadOnlyCeilingLeavesTheHumanPathAlone is the other half of the claim.
// The clamp keys on the delegated grant, so a person signed in to the console
// is untouched by it — and the read-only database session the agent's request
// opened cannot reach that person's request either, because the driver is
// opened and closed inside the handler that asked for it.
func TestMCPReadOnlyCeilingLeavesTheHumanPathAlone(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	f := setupMCPClampFixture(context.Background(), t)

	// The agent's read runs first and is SERVED, which is what actually opens
	// a read-only Postgres session; a refused statement never reaches a
	// connection at all, so it could not leak one.
	served := queryDatabaseOnSession(f.ctx, t, f.session, f.name, "SELECT id FROM employee")
	a.False(served.isError, "the agent's read must be served, or nothing opened a session: %s", served.text)
	a.Equal("STATEMENT_CLASSIFICATION_AND_READ_ONLY_SESSION", served.output.ReadOnlyEnforcement)

	refused := queryDatabaseOnSession(f.ctx, t, f.session, f.name,
		"INSERT INTO employee VALUES (3, 'agent')")
	a.True(refused.isError)

	before := f.sequenceValue(t)
	human, err := f.ctl.sqlServiceClient.Query(f.ctx, connect.NewRequest(&v1pb.QueryRequest{
		Name:      f.database,
		Statement: "INSERT INTO employee VALUES (4, 'human'); SELECT nextval('employee_seq');",
	}))
	a.NoError(err, "a workspace ceiling of READ_ONLY must not bind a person in the console")
	for _, result := range human.Msg.Results {
		a.Empty(result.Error, "no statement of the human's request may fail")
	}
	a.Equal(2, f.employeeCount(t))
	a.Greater(f.sequenceValue(t), before,
		"the nextval half must have run too: a leaked read-only session would stop it while the INSERT still counted")

	humanRead, err := f.ctl.sqlServiceClient.Query(f.ctx, connect.NewRequest(&v1pb.QueryRequest{
		Name:      f.database,
		Statement: "SELECT id FROM employee",
	}))
	a.NoError(err)
	a.Equal(v1pb.QueryResponse_READ_ONLY_ENFORCEMENT_UNSPECIFIED, humanRead.Msg.ReadOnlyEnforcement,
		"a person's query is not a capped session and must disclose nothing")
}

// TestMCPReadOnlyTighteningBitesAnOpenSession is the half of matrix row 7 the
// cutover moved. Tightening to READ_ONLY used to refuse the connection, so the
// migration matrix could assert it at the door; now the session stays open and
// the tightening bites one layer further in, per method and per statement, on
// the next request. Nothing is re-consented and no token is re-issued.
func TestMCPReadOnlyTighteningBitesAnOpenSession(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	f := setupMCPClampFixture(context.Background(), t)

	// Start read-write, on a session opened under that ceiling.
	a.NoError(f.ctl.setMCPCapability(f.ctx, v1pb.WorkspaceProfileSetting_READ_WRITE))
	session := openMCPSession(f.ctx, t, f.ctl, f.token)
	defer session.Close()
	wrote := queryDatabaseOnSession(f.ctx, t, session, f.name,
		"INSERT INTO employee VALUES (5, 'agent')")
	a.False(wrote.isError, "the session starts able to write: %s", wrote.text)
	a.Equal(2, f.employeeCount(t))

	// An admin tightens the ceiling while the session stays open.
	a.NoError(f.ctl.setMCPCapability(f.ctx, v1pb.WorkspaceProfileSetting_READ_ONLY))

	refused := queryDatabaseOnSession(f.ctx, t, session, f.name,
		"INSERT INTO employee VALUES (6, 'agent')")
	a.True(refused.isError, "the tightening must bite the very next request of the open session: %s", refused.text)
	a.Contains(refused.text, "READ_ONLY")
	a.Equal(2, f.employeeCount(t))

	// Reads keep working on that same session, so what tightened is the
	// ceiling and not the session.
	read := queryDatabaseOnSession(f.ctx, t, session, f.name, "SELECT id FROM employee")
	a.False(read.isError, "a read must still be served on the tightened session: %s", read.text)
	a.Equal("STATEMENT_CLASSIFICATION_AND_READ_ONLY_SESSION", read.output.ReadOnlyEnforcement)

	// Widening bites the same way, on the same session.
	a.NoError(f.ctl.setMCPCapability(f.ctx, v1pb.WorkspaceProfileSetting_READ_WRITE))
	again := queryDatabaseOnSession(f.ctx, t, session, f.name,
		"INSERT INTO employee VALUES (7, 'agent')")
	a.False(again.isError, "widening restores the write on the unchanged session: %s", again.text)
	a.Empty(again.output.ReadOnlyEnforcement)
	a.Equal(3, f.employeeCount(t))
}

// TestMCPReadOnlyClampCoversAnExplainRequest pins the clamp's placement outside
// the Explain guard above it. query_database sends no explain flag, but
// call_api reaches the same handler with the whole request shape, and an
// explain request carries the bare statement for the driver to prefix — so a
// clamp that skipped it would hand a read-only session an unclassified write to
// send.
func TestMCPReadOnlyClampCoversAnExplainRequest(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	f := setupMCPClampFixture(context.Background(), t)

	explained := callAPIOnSession(f.ctx, t, f.session, "SQLService/Query", map[string]any{
		"name":      f.database,
		"statement": "SELECT id FROM employee",
		"explain":   true,
	})
	a.Equal(http.StatusOK, explained.Status, "explaining a read is still a read: %s", explained.Error)

	refused := callAPIOnSession(f.ctx, t, f.session, "SQLService/Query", map[string]any{
		"name":      f.database,
		"statement": "INSERT INTO employee VALUES (8, 'agent')",
		"explain":   true,
	})
	a.Equal(http.StatusForbidden, refused.Status,
		"an explain request must be clamped like any other: %s", refused.Error)
	a.Contains(refused.Error, "READ_ONLY")
	a.Equal(1, f.employeeCount(t))
}

// TestMCPCutoverAdmitsReadOnlyAndNothingElse is the cutover itself. READ_ONLY
// opens a session and serves its class; every ceiling this build cannot serve
// still refuses the connection, including the values no Bytebase write
// produces.
func TestMCPCutoverAdmitsReadOnlyAndNothingElse(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	workspace, err := ctl.workspaceServiceClient.GetWorkspace(ctx, connect.NewRequest(&v1pb.GetWorkspaceRequest{
		Name: "workspaces/-",
	}))
	a.NoError(err)
	workspaceID := strings.TrimPrefix(workspace.Msg.Name, "workspaces/")

	project, err := ctl.projectServiceClient.CreateProject(ctx, connect.NewRequest(&v1pb.CreateProjectRequest{
		ProjectId: "mcp-cutover",
		Project:   &v1pb.Project{Title: "MCP cutover"},
	}))
	a.NoError(err)

	token, _ := mintMCPOAuthToken(t, ctl, ctl.authInterceptor.token)

	a.NoError(ctl.setMCPCapability(ctx, v1pb.WorkspaceProfileSetting_READ_ONLY))
	status, body := postMCP(t, ctl, token)
	a.Equal(http.StatusOK, status, "READ_ONLY must admit a connection now that the clamp holds it to reads; %s", body)

	session := openMCPSession(ctx, t, ctl, token)
	defer session.Close()

	a.Equal(http.StatusOK, callAPIStatus(ctx, t, session, "WorkspaceService/ListWorkspaces", nil),
		"a READ method is what a read-only ceiling serves")

	sheet := callAPIOnSession(ctx, t, session, "SheetService/CreateSheet", map[string]any{
		"parent": project.Msg.Name,
		"sheet":  map[string]any{"content": base64.StdEncoding.EncodeToString([]byte("SELECT 1;"))},
	})
	a.Equal(http.StatusForbidden, sheet.Status, "a WRITE method must stay refused under READ_ONLY")
	a.Contains(sheet.Error, "READ_ONLY")
	a.Contains(sheet.Error, "raise the MCP ceiling")

	// DISABLED is still the ceiling that closes the door.
	a.NoError(ctl.setMCPCapability(ctx, v1pb.WorkspaceProfileSetting_DISABLED))
	status, body = postMCP(t, ctl, token)
	a.Equal(http.StatusForbidden, status, "DISABLED must still refuse the connection; %s", body)

	// And so is every value this build cannot serve. The settings API refuses
	// to write these, so they are written the only way a workspace could
	// actually hold one: by hand.
	db, err := sql.Open("pgx", ctl.profile.PgURL)
	a.NoError(err)
	defer db.Close()
	for _, row := range []struct {
		value string
		why   string
	}{
		{`2`, "the reserved number was METADATA_ONLY; no mode serves it"},
		{`99`, "a number no build ever wrote"},
		{`"READ_WRTIE"`, "a mistyped ceiling is not a ceiling that permits"},
		{`"MCP_CAPABILITY_UNSPECIFIED"`, "the zero value was resolved by nobody"},
	} {
		result, err := db.ExecContext(ctx, `
			UPDATE setting SET value = jsonb_set(value, '{mcpCapability}', $2::jsonb)
			WHERE workspace = $1 AND name = 'WORKSPACE_PROFILE';
		`, workspaceID, row.value)
		a.NoError(err)
		affected, err := result.RowsAffected()
		a.NoError(err)
		a.Equal(int64(1), affected, "the workspace profile row must exist for %s to mean anything", row.value)

		status, body = postMCP(t, ctl, token)
		a.Equal(http.StatusForbidden, status, "%s: %s; %s", row.value, row.why, body)
	}
}

// TestMCPReadOnlyRoleDowngradeBitesTheNextRequest extends the live-state pin
// TestInternalChainMembershipRevocationTakesEffectNextRequest holds at the
// credential — the delegated credential carries identity and grant state only,
// so authorization is re-resolved on every request — to the clamped path a
// read-only session actually runs on.
//
// A role change is the live-state case that pin cannot reach: it revokes
// membership, which the auth interceptor answers, while a downgrade that leaves
// membership intact is the ACL's to answer, one interceptor further in and
// behind the ceiling gate. The session, the grant and the token are untouched
// throughout, and no re-consent happens.
func TestMCPReadOnlyRoleDowngradeBitesTheNextRequest(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	f := setupMCPClampFixture(context.Background(), t)

	const readerEmail = "clamp-reader@example.com"
	const readerPassword = "1024bytebase"
	reader, err := f.ctl.userServiceClient.CreateUser(f.ctx, connect.NewRequest(&v1pb.CreateUserRequest{
		User: &v1pb.User{Title: "clamp reader", Email: readerEmail, Password: readerPassword},
	}))
	a.NoError(err)
	_, err = f.ctl.addMemberToWorkspaceIAM(f.ctx, reader.Msg.Workspace, "user:"+readerEmail, "roles/workspaceMember")
	a.NoError(err)

	setProjectRole := func(roles ...string) {
		policy, err := f.ctl.projectServiceClient.GetIamPolicy(f.ctx, connect.NewRequest(&v1pb.GetIamPolicyRequest{
			Resource: f.ctl.project.Name,
		}))
		a.NoError(err)
		var kept []*v1pb.Binding
		for _, binding := range policy.Msg.Bindings {
			var members []string
			for _, member := range binding.Members {
				if !strings.Contains(member, readerEmail) {
					members = append(members, member)
				}
			}
			if len(members) > 0 {
				binding.Members = members
				kept = append(kept, binding)
			}
		}
		for _, role := range roles {
			kept = append(kept, &v1pb.Binding{Role: role, Members: []string{"user:" + readerEmail}})
		}
		policy.Msg.Bindings = kept
		_, err = f.ctl.projectServiceClient.SetIamPolicy(f.ctx, connect.NewRequest(&v1pb.SetIamPolicyRequest{
			Resource: f.ctl.project.Name,
			Policy:   policy.Msg,
		}))
		a.NoError(err)
	}

	setProjectRole("roles/sqlEditorUser")
	login, err := f.ctl.authServiceClient.Login(f.ctx, connect.NewRequest(&v1pb.LoginRequest{
		Email: readerEmail, Password: readerPassword,
	}))
	a.NoError(err)
	readerToken, _ := mintMCPOAuthToken(t, f.ctl, login.Msg.Token)
	session := openMCPSession(f.ctx, t, f.ctl, readerToken)
	defer session.Close()

	// call_api rather than query_database, so the only thing under test is the
	// permission on the query itself: query_database resolves the database by
	// listing workspace-wide first, and a listing denial would refuse this for
	// a reason that is not the one this test is about.
	query := map[string]any{"name": f.database, "statement": "SELECT id FROM employee"}
	served := callAPIOnSession(f.ctx, t, session, "SQLService/Query", query)
	a.Equal(http.StatusOK, served.Status, "the reader starts able to read: %s", served.Error)

	// The admin takes the role away. Nothing else changes: same session, same
	// token, same consented grant.
	setProjectRole()

	refused := callAPIOnSession(f.ctx, t, session, "SQLService/Query", query)
	a.Equal(http.StatusForbidden, refused.Status,
		"a role taken away must bite on the very next request: %s", refused.Error)
	a.NotContains(refused.Error, "READ_ONLY",
		"this refusal is the caller's own RBAC, not the ceiling — effective access is the intersection")

	// Live in both directions, so the refusal was the role and not damage to
	// the session.
	setProjectRole("roles/sqlEditorUser")
	restored := callAPIOnSession(f.ctx, t, session, "SQLService/Query", query)
	a.Equal(http.StatusOK, restored.Status,
		"restoring the role restores service on the unchanged session: %s", restored.Error)

	// The clamp still owns the other half for this principal: sqlEditorUser
	// carries bb.sql.dml, so RBAC would let this write and the ceiling is what
	// does not.
	written := callAPIOnSession(f.ctx, t, session, "SQLService/Query", map[string]any{
		"name": f.database, "statement": "INSERT INTO employee VALUES (9, 'reader')",
	})
	a.Equal(http.StatusForbidden, written.Status)
	a.Contains(written.Error, "READ_ONLY")
}
