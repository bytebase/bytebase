package tests

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// TestListQueryHistories covers BYT-9892: ListQueryHistories exposes a
// project's query histories across users, gated by bb.queryHistories.list on the
// project, while SearchQueryHistories and GetQueryHistory stay caller-scoped.
func TestListQueryHistories(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	ownerToken := ctl.authInterceptor.token
	const ownerEmail = "demo@example.com"

	pgContainer, err := getPgContainer(ctx)
	defer func() {
		pgContainer.Close(ctx)
	}()
	a.NoError(err)

	// 1. Create a Postgres instance + database as the owner.
	instanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance"),
		Instance: &v1pb.Instance{
			Title:       "pgInstance",
			Engine:      v1pb.Engine_POSTGRES,
			Environment: new("environments/prod"),
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: pgContainer.host, Port: pgContainer.port, Username: "postgres", Password: "root-password", Id: "admin"}},
		},
	}))
	a.NoError(err)
	instance := instanceResp.Msg

	const databaseName = "history_db"
	err = ctl.createDatabase(ctx, ctl.project, instance, nil, databaseName, "postgres")
	a.NoError(err)

	databaseResp, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Name, databaseName),
	}))
	a.NoError(err)
	database := databaseResp.Msg

	// 2. Run queries as the owner to generate query histories.
	for _, statement := range []string{"SELECT 1;", "SELECT 2;"} {
		queryResp, err := ctl.sqlServiceClient.Query(ctx, connect.NewRequest(&v1pb.QueryRequest{
			Name:      database.Name,
			Statement: statement,
			Limit:     10,
		}))
		a.NoError(err)
		a.Len(queryResp.Msg.Results, 1)
	}

	// Grab one history name for the GetQueryHistory checks below.
	searchResp, err := ctl.sqlServiceClient.SearchQueryHistories(ctx, connect.NewRequest(&v1pb.SearchQueryHistoriesRequest{
		Filter: fmt.Sprintf("project == %q", ctl.project.Name),
	}))
	a.NoError(err)
	a.GreaterOrEqual(len(searchResp.Msg.QueryHistories), 2)
	historyName := searchResp.Msg.QueryHistories[0].Name

	// 3. Create an auditor user with no project role yet.
	auditorEmail := fmt.Sprintf("auditor-%s@example.com", generateRandomString("u"))
	auditorPassword := "1024bytebase"
	auditorUser, err := ctl.userServiceClient.CreateUser(ctx, connect.NewRequest(&v1pb.CreateUserRequest{
		User: &v1pb.User{
			Email:    auditorEmail,
			Password: auditorPassword,
			Title:    "Auditor User",
		},
	}))
	a.NoError(err)
	_, err = ctl.addMemberToWorkspaceIAM(ctx, auditorUser.Msg.Workspace, fmt.Sprintf("user:%s", auditorEmail), "roles/workspaceMember")
	a.NoError(err)

	loginResp, err := ctl.authServiceClient.Login(ctx, connect.NewRequest(&v1pb.LoginRequest{
		Email:    auditorEmail,
		Password: auditorPassword,
	}))
	a.NoError(err)
	auditorToken := loginResp.Msg.Token

	// 4. Without bb.queryHistories.list: List is denied, Get hides existence.
	ctl.authInterceptor.token = auditorToken

	_, err = ctl.sqlServiceClient.ListQueryHistories(ctx, connect.NewRequest(&v1pb.ListQueryHistoriesRequest{
		Parent: ctl.project.Name,
	}))
	a.Error(err)
	a.Equal(connect.CodePermissionDenied, connect.CodeOf(err))

	_, err = ctl.sqlServiceClient.GetQueryHistory(ctx, connect.NewRequest(&v1pb.GetQueryHistoryRequest{
		Name: historyName,
	}))
	a.Error(err)
	a.Equal(connect.CodeNotFound, connect.CodeOf(err))

	// 5. Grant a custom role carrying only bb.queryHistories.list on the project.
	ctl.authInterceptor.token = ownerToken
	_, err = ctl.roleServiceClient.CreateRole(ctx, connect.NewRequest(&v1pb.CreateRoleRequest{
		Role: &v1pb.Role{
			Title:       "query-history-auditor",
			Permissions: []string{"bb.queryHistories.list"},
		},
		RoleId: "query-history-auditor",
	}))
	a.NoError(err)

	policyResp, err := ctl.projectServiceClient.GetIamPolicy(ctx, connect.NewRequest(&v1pb.GetIamPolicyRequest{
		Resource: ctl.project.Name,
	}))
	a.NoError(err)
	policy := policyResp.Msg
	policy.Bindings = append(policy.Bindings, &v1pb.Binding{
		Role:    "roles/query-history-auditor",
		Members: []string{fmt.Sprintf("user:%s", auditorEmail)},
	})
	_, err = ctl.projectServiceClient.SetIamPolicy(ctx, connect.NewRequest(&v1pb.SetIamPolicyRequest{
		Resource: ctl.project.Name,
		Policy:   policy,
	}))
	a.NoError(err)

	// 6. With the grant: the auditor lists other users' histories.
	ctl.authInterceptor.token = auditorToken

	listResp, err := ctl.sqlServiceClient.ListQueryHistories(ctx, connect.NewRequest(&v1pb.ListQueryHistoriesRequest{
		Parent: ctl.project.Name,
	}))
	a.NoError(err)
	a.GreaterOrEqual(len(listResp.Msg.QueryHistories), 2)
	for _, history := range listResp.Msg.QueryHistories {
		a.Equal(fmt.Sprintf("users/%s", ownerEmail), history.Creator)
	}

	// Creator filter pins to the given user.
	listResp, err = ctl.sqlServiceClient.ListQueryHistories(ctx, connect.NewRequest(&v1pb.ListQueryHistoriesRequest{
		Parent: ctl.project.Name,
		Filter: fmt.Sprintf("creator == \"users/%s\"", ownerEmail),
	}))
	a.NoError(err)
	a.GreaterOrEqual(len(listResp.Msg.QueryHistories), 2)

	// Filtering by a creator with no histories returns empty.
	listResp, err = ctl.sqlServiceClient.ListQueryHistories(ctx, connect.NewRequest(&v1pb.ListQueryHistoriesRequest{
		Parent: ctl.project.Name,
		Filter: fmt.Sprintf("creator == \"users/%s\"", auditorEmail),
	}))
	a.NoError(err)
	a.Empty(listResp.Msg.QueryHistories)

	// Only the creator filter is supported.
	_, err = ctl.sqlServiceClient.ListQueryHistories(ctx, connect.NewRequest(&v1pb.ListQueryHistoriesRequest{
		Parent: ctl.project.Name,
		Filter: `type == "QUERY"`,
	}))
	a.Error(err)
	a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))

	// GetQueryHistory stays creator-only: even with the grant, the
	// non-creator cannot resolve an individual history.
	_, err = ctl.sqlServiceClient.GetQueryHistory(ctx, connect.NewRequest(&v1pb.GetQueryHistoryRequest{
		Name: historyName,
	}))
	a.Error(err)
	a.Equal(connect.CodeNotFound, connect.CodeOf(err))

	// SearchQueryHistories stays caller-scoped: the auditor sees nothing.
	searchResp, err = ctl.sqlServiceClient.SearchQueryHistories(ctx, connect.NewRequest(&v1pb.SearchQueryHistoriesRequest{}))
	a.NoError(err)
	a.Empty(searchResp.Msg.QueryHistories)

	ctl.authInterceptor.token = ownerToken
}
