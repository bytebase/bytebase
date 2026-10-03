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

	developerSession := openProjectRoleSession(ctx, t, ctl, "roles/projectDeveloper")
	schema, failed := getSchemaOnSession(ctx, t, developerSession, f.name)
	a.False(failed, "a project developer, who holds no bb.sql.select, reads the schema: %s", schema)
	a.Contains(schema, "employee")
	otherSchema, failed := getSchemaOnSession(ctx, t, developerSession, otherDatabase)
	a.True(failed)
	a.Contains(otherSchema, "DATABASE_NOT_FOUND")

	control := queryDatabaseOnSession(ctx, t, f.session, otherDatabase, "SELECT 1")
	a.False(control.isError, "the workspace admin resolves every database: %s", control.text)
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

// getSchemaOnSession runs the get_schema tool and returns the text an agent
// would read and whether the tool reported a failure.
func getSchemaOnSession(ctx context.Context, t *testing.T, session *mcp.ClientSession, database string) (string, bool) {
	t.Helper()
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "get_schema",
		Arguments: map[string]any{"database": database},
	})
	require.NoError(t, err)
	var sb strings.Builder
	for _, content := range result.Content {
		if text, ok := content.(*mcp.TextContent); ok {
			sb.WriteString(text.Text)
		}
	}
	return sb.String(), result.IsError
}
