package tests

import (
	"context"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func TestMCPExportRefusalIsPermissionDenied(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctl, ctx := startWorkspace(context.Background(), t)
	f := setupMCPClampFixture(ctx, t, ctl)
	// Export is a WRITE-class method.
	a.NoError(ctl.setMCPCapability(ctx, v1pb.MCPSetting_READ_WRITE))

	const readerEmail = "export-reader@example.com"
	const readerPassword = "1024bytebase"
	reader, err := ctl.userServiceClient.CreateUser(ctx, connect.NewRequest(&v1pb.CreateUserRequest{
		User: &v1pb.User{Title: "export reader", Email: readerEmail, Password: readerPassword},
	}))
	a.NoError(err)
	_, err = ctl.addMemberToWorkspaceIAM(ctx, reader.Msg.Workspace, "user:"+readerEmail, "roles/workspaceMember")
	a.NoError(err)
	addProjectRole := func(role string) {
		policy, err := ctl.projectServiceClient.GetIamPolicy(ctx, connect.NewRequest(&v1pb.GetIamPolicyRequest{Resource: ctl.project.Name}))
		a.NoError(err)
		policy.Msg.Bindings = append(policy.Msg.Bindings, &v1pb.Binding{Role: role, Members: []string{"user:" + readerEmail}})
		_, err = ctl.projectServiceClient.SetIamPolicy(ctx, connect.NewRequest(&v1pb.SetIamPolicyRequest{Resource: ctl.project.Name, Policy: policy.Msg}))
		a.NoError(err)
	}
	addProjectRole("roles/sqlEditorReadUser")

	login, err := ctl.authServiceClient.Login(ctx, connect.NewRequest(&v1pb.LoginRequest{Email: readerEmail, Password: readerPassword}))
	a.NoError(err)
	session := openMCPSession(ctx, t, ctl, login.Msg.Token)
	defer session.Close()

	export := map[string]any{"name": f.database, "statement": "SELECT id FROM employee", "format": "CSV"}
	refused := callAPIOnSession(ctx, t, session, "SQLService/Export", export)
	a.Equal(http.StatusForbidden, refused.Status, refused.Error)
	a.Contains(refused.Error, "permission denied to access resources")

	addProjectRole("roles/sqlEditorExporter")
	served := callAPIOnSession(ctx, t, session, "SQLService/Export", export)
	a.Equal(http.StatusOK, served.Status, served.Error)
}
