package tests

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// TestListQueryHistories covers BYT-9892: ListQueryHistories exposes a
// project's query histories across users, gated by bb.queryHistories.list on the
// project, while SearchQueryHistories and GetQueryHistory stay caller-scoped.
// It also pins the deprecated SQLService aliases delegating to
// QueryHistoryService with identical results and auth until their removal.
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

	// 2b. Create a second project with its own database and history so we can
	// assert that listing is scoped to the parent project.
	otherProjectID := generateRandomString("qh-other")
	otherProjectResp, err := ctl.projectServiceClient.CreateProject(ctx, connect.NewRequest(&v1pb.CreateProjectRequest{
		Project: &v1pb.Project{
			Name:              fmt.Sprintf("projects/%s", otherProjectID),
			Title:             otherProjectID,
			AllowSelfApproval: true,
		},
		ProjectId: otherProjectID,
	}))
	a.NoError(err)
	otherProject := otherProjectResp.Msg

	const otherDatabaseName = "history_db_other"
	err = ctl.createDatabase(ctx, otherProject, instance, nil, otherDatabaseName, "postgres")
	a.NoError(err)

	otherDatabaseResp, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Name, otherDatabaseName),
	}))
	a.NoError(err)
	otherDatabase := otherDatabaseResp.Msg

	const otherStatement = "SELECT 33;"
	otherQueryResp, err := ctl.sqlServiceClient.Query(ctx, connect.NewRequest(&v1pb.QueryRequest{
		Name:      otherDatabase.Name,
		Statement: otherStatement,
		Limit:     10,
	}))
	a.NoError(err)
	a.Len(otherQueryResp.Msg.Results, 1)

	// Grab one history name for the GetQueryHistory checks below.
	searchResp, err := ctl.queryHistoryServiceClient.SearchQueryHistories(ctx, connect.NewRequest(&v1pb.SearchQueryHistoriesRequest{
		Parent: ctl.project.Name,
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

	_, err = ctl.queryHistoryServiceClient.ListQueryHistories(ctx, connect.NewRequest(&v1pb.ListQueryHistoriesRequest{
		Parent: ctl.project.Name,
	}))
	a.Error(err)
	a.Equal(connect.CodePermissionDenied, connect.CodeOf(err))

	_, err = ctl.queryHistoryServiceClient.GetQueryHistory(ctx, connect.NewRequest(&v1pb.GetQueryHistoryRequest{
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

	listResp, err := ctl.queryHistoryServiceClient.ListQueryHistories(ctx, connect.NewRequest(&v1pb.ListQueryHistoriesRequest{
		Parent: ctl.project.Name,
	}))
	a.NoError(err)
	a.GreaterOrEqual(len(listResp.Msg.QueryHistories), 2)
	for _, history := range listResp.Msg.QueryHistories {
		a.Equal(fmt.Sprintf("users/%s", ownerEmail), history.Creator)
	}

	// Listing is scoped to the parent project: the other project's history must
	// not leak in, and its name must carry the parent project prefix.
	for _, history := range listResp.Msg.QueryHistories {
		a.Contains(history.Name, ctl.project.Name+"/queryHistories/")
		a.NotEqual(otherStatement, history.Statement)
		a.NotEqual(otherDatabase.Name, history.Database)
	}

	// The grant is on ctl.project only, so listing the other project is denied.
	_, err = ctl.queryHistoryServiceClient.ListQueryHistories(ctx, connect.NewRequest(&v1pb.ListQueryHistoriesRequest{
		Parent: otherProject.Name,
	}))
	a.Error(err)
	a.Equal(connect.CodePermissionDenied, connect.CodeOf(err))

	// The owner (workspace admin) listing the other project sees only that
	// project's single history.
	ctl.authInterceptor.token = ownerToken
	otherListResp, err := ctl.queryHistoryServiceClient.ListQueryHistories(ctx, connect.NewRequest(&v1pb.ListQueryHistoriesRequest{
		Parent: otherProject.Name,
	}))
	a.NoError(err)
	a.Len(otherListResp.Msg.QueryHistories, 1)
	a.Equal(otherStatement, otherListResp.Msg.QueryHistories[0].Statement)
	a.Contains(otherListResp.Msg.QueryHistories[0].Name, otherProject.Name+"/queryHistories/")
	ctl.authInterceptor.token = auditorToken

	// Creator filter pins to the given user.
	listResp, err = ctl.queryHistoryServiceClient.ListQueryHistories(ctx, connect.NewRequest(&v1pb.ListQueryHistoriesRequest{
		Parent: ctl.project.Name,
		Filter: fmt.Sprintf("creator == \"users/%s\"", ownerEmail),
	}))
	a.NoError(err)
	a.GreaterOrEqual(len(listResp.Msg.QueryHistories), 2)

	// Filtering by a creator with no histories returns empty.
	listResp, err = ctl.queryHistoryServiceClient.ListQueryHistories(ctx, connect.NewRequest(&v1pb.ListQueryHistoriesRequest{
		Parent: ctl.project.Name,
		Filter: fmt.Sprintf("creator == \"users/%s\"", auditorEmail),
	}))
	a.NoError(err)
	a.Empty(listResp.Msg.QueryHistories)

	// Only the creator filter is supported.
	_, err = ctl.queryHistoryServiceClient.ListQueryHistories(ctx, connect.NewRequest(&v1pb.ListQueryHistoriesRequest{
		Parent: ctl.project.Name,
		Filter: `type == "QUERY"`,
	}))
	a.Error(err)
	a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))

	// GetQueryHistory stays creator-only: even with the grant, the
	// non-creator cannot resolve an individual history.
	_, err = ctl.queryHistoryServiceClient.GetQueryHistory(ctx, connect.NewRequest(&v1pb.GetQueryHistoryRequest{
		Name: historyName,
	}))
	a.Error(err)
	a.Equal(connect.CodeNotFound, connect.CodeOf(err))

	// SearchQueryHistories stays caller-scoped: the auditor sees nothing.
	searchResp, err = ctl.queryHistoryServiceClient.SearchQueryHistories(ctx, connect.NewRequest(&v1pb.SearchQueryHistoriesRequest{
		Parent: ctl.project.Name,
	}))
	a.NoError(err)
	a.Empty(searchResp.Msg.QueryHistories)

	// 7. Cross-project listing via the AIP-159 wildcard parent "projects/-"
	// requires a workspace-level grant, so the project-scoped auditor is denied.
	_, err = ctl.queryHistoryServiceClient.ListQueryHistories(ctx, connect.NewRequest(&v1pb.ListQueryHistoriesRequest{
		Parent: "projects/-",
	}))
	a.Error(err)
	a.Equal(connect.CodePermissionDenied, connect.CodeOf(err))

	// The owner (workspace admin) lists across all projects and sees rows from
	// both projects, each named under its own project.
	ctl.authInterceptor.token = ownerToken
	wildcardResp, err := ctl.queryHistoryServiceClient.ListQueryHistories(ctx, connect.NewRequest(&v1pb.ListQueryHistoriesRequest{
		Parent: "projects/-",
	}))
	a.NoError(err)
	a.GreaterOrEqual(len(wildcardResp.Msg.QueryHistories), 3)
	mainProjectCount, otherProjectCount := 0, 0
	for _, history := range wildcardResp.Msg.QueryHistories {
		switch {
		case strings.HasPrefix(history.Name, ctl.project.Name+"/queryHistories/"):
			mainProjectCount++
		case strings.HasPrefix(history.Name, otherProject.Name+"/queryHistories/"):
			otherProjectCount++
			a.Equal(otherStatement, history.Statement)
		default:
			a.Failf("unexpected project in wildcard listing", "history %q", history.Name)
		}
	}
	a.GreaterOrEqual(mainProjectCount, 2)
	a.Equal(1, otherProjectCount)

	// The creator filter composes with the wildcard parent.
	wildcardResp, err = ctl.queryHistoryServiceClient.ListQueryHistories(ctx, connect.NewRequest(&v1pb.ListQueryHistoriesRequest{
		Parent: "projects/-",
		Filter: fmt.Sprintf("creator == \"users/%s\"", auditorEmail),
	}))
	a.NoError(err)
	a.Empty(wildcardResp.Msg.QueryHistories)

	// 8. SearchQueryHistories requires a concrete parent project and stays
	// caller-scoped; cross-project reads are reserved for the IAM-gated
	// ListQueryHistories.
	searchResp, err = ctl.queryHistoryServiceClient.SearchQueryHistories(ctx, connect.NewRequest(&v1pb.SearchQueryHistoriesRequest{
		Parent: ctl.project.Name,
	}))
	a.NoError(err)
	a.GreaterOrEqual(len(searchResp.Msg.QueryHistories), 2)
	for _, history := range searchResp.Msg.QueryHistories {
		a.Contains(history.Name, ctl.project.Name+"/queryHistories/")
		a.NotEqual(otherStatement, history.Statement)
	}

	searchResp, err = ctl.queryHistoryServiceClient.SearchQueryHistories(ctx, connect.NewRequest(&v1pb.SearchQueryHistoriesRequest{
		Parent: otherProject.Name,
	}))
	a.NoError(err)
	a.Len(searchResp.Msg.QueryHistories, 1)
	a.Equal(otherStatement, searchResp.Msg.QueryHistories[0].Statement)

	// The AIP-159 wildcard parent is rejected.
	_, err = ctl.queryHistoryServiceClient.SearchQueryHistories(ctx, connect.NewRequest(&v1pb.SearchQueryHistoriesRequest{
		Parent: "projects/-",
	}))
	a.Error(err)
	a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))

	// The parent is required.
	_, err = ctl.queryHistoryServiceClient.SearchQueryHistories(ctx, connect.NewRequest(&v1pb.SearchQueryHistoriesRequest{}))
	a.Error(err)
	a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))

	// A nonexistent parent project is rejected, consistent with
	// ListQueryHistories.
	_, err = ctl.queryHistoryServiceClient.SearchQueryHistories(ctx, connect.NewRequest(&v1pb.SearchQueryHistoriesRequest{
		Parent: "projects/no-such-project",
	}))
	a.Error(err)
	a.Equal(connect.CodeNotFound, connect.CodeOf(err))

	// 9. The deprecated SQLService method names keep working as delegating
	// aliases of QueryHistoryService until they are removed.
	aliasSearchResp, err := ctl.sqlServiceClient.SearchQueryHistories(ctx, connect.NewRequest(&v1pb.SearchQueryHistoriesRequest{ //nolint:staticcheck // Pinning the deprecated alias until removal.
		Parent: ctl.project.Name,
	}))
	a.NoError(err)
	newSearchResp, err := ctl.queryHistoryServiceClient.SearchQueryHistories(ctx, connect.NewRequest(&v1pb.SearchQueryHistoriesRequest{
		Parent: ctl.project.Name,
	}))
	a.NoError(err)
	a.Equal(len(newSearchResp.Msg.QueryHistories), len(aliasSearchResp.Msg.QueryHistories))
	for i, history := range aliasSearchResp.Msg.QueryHistories {
		a.Equal(newSearchResp.Msg.QueryHistories[i].Name, history.Name)
	}

	aliasListResp, err := ctl.sqlServiceClient.ListQueryHistories(ctx, connect.NewRequest(&v1pb.ListQueryHistoriesRequest{ //nolint:staticcheck // Pinning the deprecated alias until removal.
		Parent: otherProject.Name,
	}))
	a.NoError(err)
	a.Len(aliasListResp.Msg.QueryHistories, 1)
	a.Equal(otherStatement, aliasListResp.Msg.QueryHistories[0].Statement)

	// Both method names resolve the same individual history for its creator.
	getResp, err := ctl.queryHistoryServiceClient.GetQueryHistory(ctx, connect.NewRequest(&v1pb.GetQueryHistoryRequest{
		Name: historyName,
	}))
	a.NoError(err)
	a.Equal(historyName, getResp.Msg.Name)
	aliasGetResp, err := ctl.sqlServiceClient.GetQueryHistory(ctx, connect.NewRequest(&v1pb.GetQueryHistoryRequest{ //nolint:staticcheck // Pinning the deprecated alias until removal.
		Name: historyName,
	}))
	a.NoError(err)
	a.Equal(historyName, aliasGetResp.Msg.Name)

	// The aliases enforce the same IAM and caller scoping.
	ctl.authInterceptor.token = auditorToken
	_, err = ctl.sqlServiceClient.GetQueryHistory(ctx, connect.NewRequest(&v1pb.GetQueryHistoryRequest{ //nolint:staticcheck // Pinning the deprecated alias until removal.
		Name: historyName,
	}))
	a.Error(err)
	a.Equal(connect.CodeNotFound, connect.CodeOf(err))
	_, err = ctl.sqlServiceClient.ListQueryHistories(ctx, connect.NewRequest(&v1pb.ListQueryHistoriesRequest{ //nolint:staticcheck // Pinning the deprecated alias until removal.
		Parent: otherProject.Name,
	}))
	a.Error(err)
	a.Equal(connect.CodePermissionDenied, connect.CodeOf(err))
}
