package tests

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// TestAccessGrantApprovalIsDecidedAtCreation holds a console access request to
// what it did before its approval finding moved ahead of the grant's creation:
// the seeded fallback rule makes it wait for a project owner, and a workspace
// with no rule activates it at once. An MCP session cannot activate it either
// way.
func TestAccessGrantApprovalIsDecidedAtCreation(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl, ctx := startWorkspace(ctx, t)

	pgContainer := sharedPgTarget(t)
	instance, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("inst"),
		Instance: &v1pb.Instance{
			SyncDatabases: &v1pb.SyncDatabases{},
			Title:         "Test Instance",
			Engine:        v1pb.Engine_POSTGRES,
			Environment:   new("environments/prod"),
			Activation:    true,
			DataSources:   []*v1pb.DataSource{pgContainer.adminDataSource()},
		},
	}))
	a.NoError(err)
	dbName := generateRandomString("db")
	a.NoError(ctl.createDatabase(ctx, ctl.project, instance.Msg, nil, dbName, ""))
	target := fmt.Sprintf("%s/databases/%s", instance.Msg.Name, dbName)
	request := func() *connect.Request[v1pb.CreateAccessGrantRequest] {
		return connect.NewRequest(&v1pb.CreateAccessGrantRequest{
			Parent: ctl.project.Name,
			AccessGrant: &v1pb.AccessGrant{
				Targets:    []string{target},
				Query:      "SELECT 1",
				Reason:     "testing",
				Expiration: &v1pb.AccessGrant_Ttl{Ttl: durationpb.New(time.Hour)},
			},
		})
	}

	pending, err := ctl.accessGrantServiceClient.CreateAccessGrant(ctx, request())
	a.NoError(err)
	a.Equal(v1pb.AccessGrant_PENDING, pending.Msg.Status, "the seeded fallback rule requires a project owner")
	issue, err := ctl.issueServiceClient.GetIssue(ctx, connect.NewRequest(&v1pb.GetIssueRequest{Name: pending.Msg.Issue}))
	a.NoError(err)
	a.Equal([]string{"roles/projectOwner"}, issue.Msg.GetApprovalTemplate().GetFlow().GetRoles())
	a.Equal(v1pb.ApprovalStatus_PENDING, issue.Msg.ApprovalStatus)

	_, err = ctl.settingServiceClient.UpdateSetting(ctx, connect.NewRequest(&v1pb.UpdateSettingRequest{
		AllowMissing: true,
		Setting: &v1pb.Setting{
			Name: "settings/WORKSPACE_APPROVAL",
			Value: &v1pb.SettingValue{Value: &v1pb.SettingValue_WorkspaceApproval{
				WorkspaceApproval: &v1pb.WorkspaceApprovalSetting{},
			}},
		},
	}))
	a.NoError(err)
	active, err := ctl.accessGrantServiceClient.CreateAccessGrant(ctx, request())
	a.NoError(err)
	a.Equal(v1pb.AccessGrant_ACTIVE, active.Msg.Status, "with no rule, a console request needs no approval")

	a.NoError(ctl.setMCPCapability(ctx, v1pb.MCPSetting_READ_WRITE))
	session := openMCPSession(ctx, t, ctl, ctl.authInterceptor.token)
	defer session.Close()
	activate := callAPIOnSession(ctx, t, session, "AccessGrantService/ActivateAccessGrant", map[string]any{
		"name": pending.Msg.Name,
	})
	a.Equal(http.StatusForbidden, activate.Status)
	a.Contains(activate.Error, "not available to MCP sessions")
	a.Contains(activate.Error, "approv", "refused as an approval decision, not as workspace administration")
	still, err := ctl.accessGrantServiceClient.GetAccessGrant(ctx, connect.NewRequest(&v1pb.GetAccessGrantRequest{Name: pending.Msg.Name}))
	a.NoError(err)
	a.Equal(v1pb.AccessGrant_PENDING, still.Msg.Status)
}
