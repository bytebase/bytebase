package tests

import (
	"context"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// TestMCPResolvesDatabasesThroughProjectRoles pins the database tools for a
// user whose only access is a project role. Workspace member holds no
// workspace-wide bb.databases.list, so the tools must reach the user's
// databases through the project, and must not reach any other project's.
func TestMCPResolvesDatabasesThroughProjectRoles(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctl, ctx := startWorkspace(context.Background(), t)
	f := setupMCPClampFixture(ctx, t, ctl)

	instanceName, _, found := strings.Cut(f.database, "/databases/")
	a.True(found)
	instance, err := ctl.instanceServiceClient.GetInstance(ctx, connect.NewRequest(&v1pb.GetInstanceRequest{Name: instanceName}))
	a.NoError(err)
	otherProjectID := generateRandomString("other")
	otherProject, err := ctl.projectServiceClient.CreateProject(ctx, connect.NewRequest(&v1pb.CreateProjectRequest{
		Project:   &v1pb.Project{Title: otherProjectID},
		ProjectId: otherProjectID,
	}))
	a.NoError(err)
	otherDatabase := generateRandomString("otherdb")
	a.NoError(ctl.createDatabase(ctx, otherProject.Msg, instance.Msg, nil, otherDatabase, ""))

	editorSession := openProjectRoleSession(ctx, t, ctl, "roles/sqlEditorUser")
	own := queryDatabaseOnSession(ctx, t, editorSession, f.name, "SELECT name FROM employee")
	a.False(own.isError, "a project role must reach its own project's database: %s", own.text)
	a.Equal(1, own.output.RowCount)
	other := queryDatabaseOnSession(ctx, t, editorSession, otherDatabase, "SELECT 1")
	a.True(other.isError, "a project role must not reach another project's database")
	a.Contains(other.text, "DATABASE_NOT_FOUND",
		"the answer must not reveal that a database exists in a project the user cannot read: %s", other.text)

	// A named project is listed directly, so the server's own answer for that
	// project parent comes through: 403 for an existing one, 404 for a missing one.
	for _, tc := range []struct {
		project  string
		database string
		want     string
	}{
		{project: ctl.project.Name, database: f.name, want: "Bytebase"},
		{project: otherProject.Msg.Name, database: otherDatabase, want: "PERMISSION_DENIED"},
		{project: "projects/" + generateRandomString("missing"), database: f.name, want: "DATABASE_NOT_FOUND"},
	} {
		text, _ := callToolOnSession(ctx, t, editorSession, "query_database", map[string]any{
			"database":  tc.database,
			"project":   tc.project,
			"statement": "SELECT name FROM employee",
		})
		a.Contains(text, tc.want, tc.project)
	}

	developerSession := openProjectRoleSession(ctx, t, ctl, "roles/projectDeveloper")
	schema, failed := callToolOnSession(ctx, t, developerSession, "get_schema", map[string]any{"database": f.name})
	a.False(failed, "a project developer, who holds no bb.sql.select, reads the schema: %s", schema)
	a.Contains(schema, "employee")
	otherSchema, failed := callToolOnSession(ctx, t, developerSession, "get_schema", map[string]any{"database": otherDatabase})
	a.True(failed)
	a.Contains(otherSchema, "DATABASE_NOT_FOUND")

	control := queryDatabaseOnSession(ctx, t, f.session, otherDatabase, "SELECT 1")
	a.False(control.isError, "the workspace admin resolves every database: %s", control.text)

	// Proposing a change writes a sheet, a plan and an issue, which the fixture's
	// Read-only ceiling refuses.
	a.NoError(ctl.setMCPCapability(ctx, v1pb.MCPSetting_READ_WRITE))
	proposed := proposeChangeOnSession(ctx, t, developerSession, f.name, "CREATE TABLE note (id INT)", "Add note table")
	a.False(proposed.isError, "a project developer proposes a change to its own database: %s", proposed.text)
	a.Contains(proposed.text, "Issue: "+ctl.project.Name+"/issues/")
	proposedOther := proposeChangeOnSession(ctx, t, developerSession, otherDatabase, "CREATE TABLE note (id INT)", "Add note table")
	a.True(proposedOther.isError)
	a.Contains(proposedOther.text, "DATABASE_NOT_FOUND")
}

// openProjectRoleSession opens an MCP session for a new workspace member whose
// only other role is role on the controller's project.
func openProjectRoleSession(ctx context.Context, t *testing.T, ctl *controller, role string) *mcp.ClientSession {
	t.Helper()
	token := bot35CreateProjectUser(ctx, t, ctl, ctl.project.Name, role, "resolver")
	mcpToken, _ := mintMCPOAuthToken(t, ctl, token)
	session := openMCPSession(ctx, t, ctl, mcpToken)
	t.Cleanup(func() { session.Close() })
	return session
}

// callToolOnSession runs a tool and returns the text an agent would read and
// whether the tool reported a failure.
func callToolOnSession(ctx context.Context, t *testing.T, session *mcp.ClientSession, name string, arguments map[string]any) (string, bool) {
	t.Helper()
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: arguments})
	require.NoError(t, err)
	var sb strings.Builder
	for _, content := range result.Content {
		if text, ok := content.(*mcp.TextContent); ok {
			sb.WriteString(text.Text)
		}
	}
	return sb.String(), result.IsError
}
