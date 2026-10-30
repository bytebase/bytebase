package tests

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"testing"

	"connectrpc.com/connect"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/protobuf/proto"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// countRows reads a table's row count on the human path, which no MCP check
// applies to.
func countRows(ctx context.Context, t *testing.T, ctl *controller, database, table string) int {
	t.Helper()
	resp, err := ctl.sqlServiceClient.Query(ctx, connect.NewRequest(&v1pb.QueryRequest{
		Name:      database,
		Statement: fmt.Sprintf("SELECT count(*) FROM %s", table),
	}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.Results, 1)
	require.Empty(t, resp.Msg.Results[0].Error)
	value := resp.Msg.Results[0].Rows[0].Values[0]
	if s := value.GetStringValue(); s != "" {
		n, err := strconv.Atoi(s)
		require.NoError(t, err)
		return n
	}
	return int(value.GetInt64Value())
}

// An instance with a read-only data source, where the query data policy lets
// writes use ADMIN. query_database sends the read-only data source, and the
// read-only account can only SELECT.
func TestMCPQueryDatabaseRoutesWritesAsWrites(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctl, ctx := startWorkspace(context.Background(), t)

	pg := provisionPgInstance(t)
	_, err := pg.GetDB().Exec("CREATE ROLE mcp_write_ro LOGIN PASSWORD 'mcp_write_ro_password'")
	a.NoError(err)
	instanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("mcp-write"),
		Instance: &v1pb.Instance{
			Title:       "MCP write",
			Engine:      v1pb.Engine_POSTGRES,
			Environment: new("environments/prod"),
			Activation:  true,
			DataSources: []*v1pb.DataSource{pg.adminDataSource()},
		},
	}))
	a.NoError(err)
	const databaseName = "mcpwritedb"
	a.NoError(ctl.createDatabase(ctx, ctl.project, instanceResp.Msg, nil, databaseName, "postgres"))
	database := fmt.Sprintf("%s/databases/%s", instanceResp.Msg.Name, databaseName)
	databaseResp, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{Name: database}))
	a.NoError(err)
	setup, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
		Parent: ctl.project.Name,
		Sheet:  &v1pb.Sheet{Content: []byte(`CREATE TABLE employee(id INT PRIMARY KEY, name TEXT); INSERT INTO employee VALUES (1, 'Bytebase');`)},
	}))
	a.NoError(err)
	a.NoError(ctl.changeDatabase(ctx, ctl.project, databaseResp.Msg, setup.Msg, false))

	grants, err := sql.Open("pgx", fmt.Sprintf("postgresql://postgres:root-password@%s:%s/%s?sslmode=disable", pg.GetHost(), pg.GetPort(), databaseName))
	a.NoError(err)
	defer grants.Close()
	for _, statement := range []string{
		fmt.Sprintf("GRANT CONNECT ON DATABASE %q TO mcp_write_ro", databaseName),
		"GRANT USAGE ON SCHEMA public TO mcp_write_ro",
		"GRANT SELECT ON ALL TABLES IN SCHEMA public TO mcp_write_ro",
	} {
		_, err = grants.Exec(statement)
		a.NoError(err)
	}
	_, err = ctl.instanceServiceClient.AddDataSource(ctx, connect.NewRequest(&v1pb.AddDataSourceRequest{
		Name: instanceResp.Msg.Name,
		DataSource: &v1pb.DataSource{
			Id:       "readonly",
			Type:     v1pb.DataSourceType_READ_ONLY,
			Host:     pg.GetHost(),
			Port:     pg.GetPort(),
			Username: "mcp_write_ro",
			Password: "mcp_write_ro_password",
		},
	}))
	a.NoError(err)
	workspaceID, err := getStore(t, ctl.server).GetWorkspaceID(ctx)
	a.NoError(err)
	_, err = ctl.orgPolicyServiceClient.CreatePolicy(ctx, connect.NewRequest(&v1pb.CreatePolicyRequest{
		Parent: common.FormatWorkspace(workspaceID),
		Policy: &v1pb.Policy{
			Type:   v1pb.PolicyType_DATA_QUERY,
			Policy: &v1pb.Policy_QueryDataPolicy{QueryDataPolicy: &v1pb.QueryDataPolicy{AllowAdminDataSource: true}},
		},
	}))
	a.NoError(err)

	a.NoError(ctl.setMCPCapability(ctx, v1pb.MCPSetting_READ_WRITE))
	token, _ := mintMCPOAuthToken(t, ctl, ctl.authInterceptor.token)
	session := openMCPSession(ctx, t, ctl, token)
	defer session.Close()

	read := queryDatabaseOnSession(ctx, t, session, databaseName, "SELECT current_user")
	a.False(read.isError, read.text)
	a.Equal([][]any{{"mcp_write_ro"}}, read.output.Rows, "a read keeps the data source query_database sends")

	// A SET returns no data, so the verdict cannot show this request is a read,
	// while the SQL Editor's routing reads it as one.
	const setsTheSession = "SELECT current_user; SET search_path TO public"
	routed := queryDatabaseOnSession(ctx, t, session, databaseName, setsTheSession)
	a.False(routed.isError, routed.text)
	a.Equal([][]any{{"postgres"}}, routed.output.Rows, "the verdict routes the request to ADMIN")
	refused := callAPIOnSession(ctx, t, session, "SQLService/Query", map[string]any{
		"name":         database,
		"dataSourceId": "readonly",
		"statement":    setsTheSession,
	})
	a.Equal(http.StatusBadRequest, refused.Status, refused.Error)
	a.Equal(auth.MCPConfirmationReason, confirmationErrorInfo(t, refused.RawResponse).GetReason(), "call_api cannot confirm a possible write")

	written := queryDatabaseOnSession(ctx, t, session, databaseName, "INSERT INTO employee VALUES (2, 'query_database')")
	a.False(written.isError, "query_database writes on ADMIN, though it sends the read-only data source: %s", written.text)
	a.Equal(2, countRows(ctx, t, ctl, database, "employee"))
}

// confirmationErrorInfo decodes the ErrorInfo a refused request's body carries,
// as query_database does.
func confirmationErrorInfo(t *testing.T, raw string) *errdetails.ErrorInfo {
	t.Helper()
	var body struct {
		Details []struct {
			Type  string `json:"type"`
			Value string `json:"value"`
		} `json:"details"`
	}
	require.NoError(t, json.Unmarshal([]byte(raw), &body))
	require.Len(t, body.Details, 1)
	require.Equal(t, "google.rpc.ErrorInfo", body.Details[0].Type)
	value, err := base64.RawStdEncoding.DecodeString(body.Details[0].Value)
	require.NoError(t, err)
	var info errdetails.ErrorInfo
	require.NoError(t, proto.Unmarshal(value, &info))
	return &info
}

// formAnswerer is a client that answers every form the same way and keeps the
// messages it was shown.
type formAnswerer struct {
	answer   *mcp.ElicitResult
	mu       sync.Mutex
	messages []string
}

func (f *formAnswerer) handle(_ context.Context, req *mcp.ElicitRequest) (*mcp.ElicitResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.messages = append(f.messages, req.Params.Message)
	return f.answer, nil
}

func (f *formAnswerer) asked() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.messages...)
}

// A Read-write session whose client shows forms: the user's answer decides
// whether a statement Bytebase cannot show is a read runs.
func TestMCPQueryDatabaseAsksBeforeAWrite(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctl, ctx := startWorkspace(context.Background(), t)
	f := setupMCPClampFixture(ctx, t, ctl)
	a.NoError(ctl.setMCPCapability(ctx, v1pb.MCPSetting_READ_WRITE))
	token, _ := mintMCPOAuthToken(t, ctl, ctl.authInterceptor.token)
	answering := func(answer *mcp.ElicitResult) (*mcp.ClientSession, *formAnswerer) {
		user := &formAnswerer{answer: answer}
		session := openMCPSessionWithOptions(ctx, t, ctl, token, &mcp.ClientOptions{ElicitationHandler: user.handle})
		t.Cleanup(func() { session.Close() })
		return session, user
	}
	const insert = "INSERT INTO employee VALUES (2, 'agent')"

	changing, changeUser := answering(&mcp.ElicitResult{Action: "accept", Content: map[string]any{"choice": "change"}})
	changed := queryDatabaseOnSession(ctx, t, changing, f.name, insert)
	a.False(changed.isError, changed.text)
	a.Contains(changed.text, "Not run: the user wants to change this request")
	a.Len(changeUser.asked(), 1)
	instanceID, _, err := common.GetInstanceDatabaseID(f.database)
	a.NoError(err)
	a.Equal(fmt.Sprintf("Run on database %q (instance %q, environment \"prod\"):\n%s", f.name, instanceID, insert), changeUser.asked()[0])
	a.Equal(1, f.employeeCount(t))

	declining, _ := answering(&mcp.ElicitResult{Action: "decline"})
	declined := queryDatabaseOnSession(ctx, t, declining, f.name, insert)
	a.Contains(declined.text, "Not run: the user declined")
	a.Equal(1, f.employeeCount(t))

	running, runUser := answering(&mcp.ElicitResult{Action: "accept", Content: map[string]any{"choice": "run"}})
	read := queryDatabaseOnSession(ctx, t, running, f.name, "SELECT name FROM employee")
	a.False(read.isError, read.text)
	a.Empty(runUser.asked(), "a read runs without asking")
	ran := queryDatabaseOnSession(ctx, t, running, f.name, insert)
	a.False(ran.isError, ran.text)
	a.Len(runUser.asked(), 1)
	a.Equal(2, f.employeeCount(t))

	direct := callAPIOnSession(ctx, t, running, "SQLService/Query", map[string]any{
		"name":      f.database,
		"statement": "INSERT INTO employee VALUES (3, 'call_api')",
	})
	a.Equal(http.StatusBadRequest, direct.Status, direct.Error)
	a.Contains(direct.Error, "query_database")
	info := confirmationErrorInfo(t, direct.RawResponse)
	a.Equal(auth.MCPConfirmationReason, info.GetReason())
	a.Equal(map[string]string{"environment": "prod", "schema": ""}, info.GetMetadata())
	a.Equal(2, f.employeeCount(t))

	explained := callAPIOnSession(ctx, t, running, "SQLService/Query", map[string]any{
		"name":      f.database,
		"statement": "INSERT INTO employee VALUES (4, 'explain')",
		"explain":   true,
	})
	a.Equal(http.StatusForbidden, explained.Status, explained.Error)
	a.Contains(explained.Error, "explains only statements Bytebase can show are reads")
}

// Export skips statement validation on MySQL, and the driver runs a statement
// that returns no rows.
func TestMCPExportRunsOnlyReads(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctl, ctx := startWorkspace(context.Background(), t)
	database, _ := ctl.createTestMySQLDatabase(ctx, t)
	setup, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
		Parent: ctl.project.Name,
		Sheet:  &v1pb.Sheet{Content: []byte(`CREATE TABLE employee(id INT PRIMARY KEY); INSERT INTO employee VALUES (1);`)},
	}))
	a.NoError(err)
	a.NoError(ctl.changeDatabase(ctx, ctl.project, database, setup.Msg, false))

	a.NoError(ctl.setMCPCapability(ctx, v1pb.MCPSetting_READ_WRITE))
	token, _ := mintMCPOAuthToken(t, ctl, ctl.authInterceptor.token)
	session := openMCPSession(ctx, t, ctl, token)
	defer session.Close()

	exported := callAPIOnSession(ctx, t, session, "SQLService/Export", map[string]any{
		"name":      database.Name,
		"statement": "SELECT id FROM employee",
		"format":    "JSON",
	})
	a.Equal(http.StatusOK, exported.Status, "a read exports: %s", exported.Error)

	refused := callAPIOnSession(ctx, t, session, "SQLService/Export", map[string]any{
		"name":      database.Name,
		"statement": "INSERT INTO employee VALUES (2)",
		"format":    "JSON",
	})
	a.Equal(http.StatusForbidden, refused.Status, "an export must not write: %s", refused.Error)
	a.Contains(refused.Error, "exports only statements Bytebase can show are reads")
	a.Equal(1, countRows(ctx, t, ctl, database.Name, "employee"))

	_, err = ctl.sqlServiceClient.Export(ctx, connect.NewRequest(&v1pb.ExportRequest{
		Name:      database.Name,
		Statement: "SET @n = 1; SELECT id FROM employee",
		Format:    v1pb.ExportFormat_JSON,
	}))
	a.NoError(err, "the console still exports a request the verdict cannot show is a read")
}
