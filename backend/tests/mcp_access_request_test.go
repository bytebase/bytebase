package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func TestMCPRequestsAccessWithTheDedicatedTool(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl, ctx := startWorkspace(ctx, t)

	project, err := ctl.projectServiceClient.CreateProject(ctx, connect.NewRequest(&v1pb.CreateProjectRequest{
		ProjectId: "mcp-request-tool",
		Project:   &v1pb.Project{Title: "MCP request tool"},
	}))
	a.NoError(err)
	a.NoError(ctl.setMCPCapability(ctx, v1pb.MCPSetting_READ_WRITE))
	session := openMCPSession(ctx, t, ctl, ctl.authInterceptor.token)
	defer session.Close()

	pgContainer := sharedPgTarget(t)
	instance, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("inst"),
		Instance: &v1pb.Instance{
			SyncDatabases: &v1pb.SyncDatabases{},
			Title:         "MCP request tool instance",
			Engine:        v1pb.Engine_POSTGRES,
			Environment:   new("environments/prod"),
			Activation:    true,
			DataSources:   []*v1pb.DataSource{pgContainer.adminDataSource()},
		},
	}))
	a.NoError(err)
	dbName := generateRandomString("db")
	a.NoError(ctl.createDatabase(ctx, project.Msg, instance.Msg, nil, dbName, ""))
	// A database on a workspace instance, whose name carries no project.
	target := fmt.Sprintf("%s/databases/%s", instance.Msg.Name, dbName)
	request := map[string]any{
		"database":      target,
		"statement":     "SELECT 1",
		"schema":        "public",
		"unmask":        true,
		"export":        true,
		"reason":        "the agent needs one read",
		"durationHours": 2,
	}

	refused, text := callToolOnSession(ctx, t, session, "request_access_grant", map[string]any{
		"database":      dbName,
		"statement":     "SELECT 1",
		"reason":        "the agent needs one read",
		"durationHours": 2,
	})
	a.True(refused, text)
	a.Contains(text, fmt.Sprintf("failed to parse %q", dbName), "the API refuses a name that is not a full database name")

	refused, text = callToolOnSession(ctx, t, session, "request_access_grant", request)
	a.True(refused, text)
	a.Contains(text, `just-in-time access is off for project "mcp-request-tool"`)

	_, err = ctl.projectServiceClient.UpdateProject(ctx, connect.NewRequest(&v1pb.UpdateProjectRequest{
		Project:    &v1pb.Project{Name: project.Msg.Name, AllowJustInTimeAccess: true},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"allow_just_in_time_access"}},
	}))
	a.NoError(err)

	failed, text := callToolOnSession(ctx, t, session, "request_access_grant", request)
	a.False(failed, text)
	grants, err := ctl.accessGrantServiceClient.ListAccessGrants(ctx, connect.NewRequest(&v1pb.ListAccessGrantsRequest{Parent: project.Msg.Name}))
	a.NoError(err)
	a.Len(grants.Msg.AccessGrants, 1)
	grant := grants.Msg.AccessGrants[0]
	a.Equal(v1pb.AccessGrant_PENDING, grant.Status)
	a.Equal([]string{target}, grant.Targets)
	a.Equal("SELECT 1", grant.Query)
	a.Equal("public", grant.Schema)
	a.True(grant.Unmask)
	a.True(grant.Export)
	a.Equal("the agent needs one read", grant.Reason)
	a.Equal(2*time.Hour, grant.GetTtl().AsDuration(), "a ttl starts at approval")
	a.Contains(text, "Requested "+grant.Name+":")
	a.Contains(text, grant.Issue)
	// The project's creator is its only owner, and new projects refuse self-approval.
	a.Contains(text, "No other person can approve it: step 1 of its approval flow needs roles/projectOwner, which no other active "+
		"workspace member holds, and this project does not allow self-approval.", text)
	a.Contains(text, "grant_masking_exemption", "an unmask request names the exemption tool as the way around")
}
