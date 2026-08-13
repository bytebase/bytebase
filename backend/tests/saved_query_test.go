package tests

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func TestListSavedQueries(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	ownerToken := ctl.authInterceptor.token

	otherProjectResp, err := ctl.projectServiceClient.CreateProject(ctx, connect.NewRequest(&v1pb.CreateProjectRequest{
		Project: &v1pb.Project{
			Title:             "Saved Query Other Project",
			AllowSelfApproval: true,
		},
		ProjectId: generateRandomString("saved-query-project"),
	}))
	a.NoError(err)
	otherProject := otherProjectResp.Msg

	otherEmail := fmt.Sprintf("saved-query-%s@example.com", generateRandomString("user"))
	otherPassword := "1024bytebase"
	otherUser, err := ctl.userServiceClient.CreateUser(ctx, connect.NewRequest(&v1pb.CreateUserRequest{
		User: &v1pb.User{
			Email:    otherEmail,
			Password: otherPassword,
			Title:    "Saved Query User",
		},
	}))
	a.NoError(err)
	_, err = ctl.addMemberToWorkspaceIAM(ctx, otherUser.Msg.Workspace, fmt.Sprintf("user:%s", otherEmail), "roles/workspaceMember")
	a.NoError(err)

	createSavedQuery := func(title, parent string) *v1pb.SavedQuery {
		resp, err := ctl.savedQueryServiceClient.CreateSavedQuery(ctx, connect.NewRequest(&v1pb.CreateSavedQueryRequest{
			Parent: parent,
			SavedQuery: &v1pb.SavedQuery{
				Title:   title,
				Content: []byte("SELECT 1;"),
			},
		}))
		a.NoError(err)
		return resp.Msg
	}

	projectOwnerSavedQuery := createSavedQuery("project-owner", ctl.project.Name)
	crossProjectOwnerSavedQuery := createSavedQuery("cross-project-owner", otherProject.Name)

	loginResp, err := ctl.authServiceClient.Login(ctx, connect.NewRequest(&v1pb.LoginRequest{
		Email:    otherEmail,
		Password: otherPassword,
	}))
	a.NoError(err)
	ctl.authInterceptor.token = loginResp.Msg.Token
	otherSavedQuery := createSavedQuery("other-creator", ctl.project.Name)

	_, err = ctl.savedQueryServiceClient.ListSavedQueries(ctx, connect.NewRequest(&v1pb.ListSavedQueriesRequest{
		Parent: "projects/-",
	}))
	a.Error(err)
	a.Equal(connect.CodePermissionDenied, connect.CodeOf(err))

	ctl.authInterceptor.token = ownerToken
	roleID := generateRandomString("saved-query-lister")
	_, err = ctl.roleServiceClient.CreateRole(ctx, connect.NewRequest(&v1pb.CreateRoleRequest{
		Role: &v1pb.Role{
			Title:       "Saved Query Lister",
			Permissions: []string{"bb.savedQueries.list"},
		},
		RoleId: roleID,
	}))
	a.NoError(err)
	_, err = ctl.addMemberToWorkspaceIAM(ctx, otherUser.Msg.Workspace, fmt.Sprintf("user:%s", otherEmail), fmt.Sprintf("roles/%s", roleID))
	a.NoError(err)

	ctl.authInterceptor.token = loginResp.Msg.Token
	listProjectResp, err := ctl.savedQueryServiceClient.ListSavedQueries(ctx, connect.NewRequest(&v1pb.ListSavedQueriesRequest{
		Parent: ctl.project.Name,
		Filter: fmt.Sprintf("creator == \"users/%s\"", otherEmail),
	}))
	a.NoError(err)
	a.ElementsMatch([]string{otherSavedQuery.Name}, savedQueryNames(listProjectResp.Msg.SavedQueries))

	listNotCreatorResp, err := ctl.savedQueryServiceClient.ListSavedQueries(ctx, connect.NewRequest(&v1pb.ListSavedQueriesRequest{
		Parent: ctl.project.Name,
		Filter: fmt.Sprintf("creator != \"users/%s\"", otherEmail),
	}))
	a.NoError(err)
	a.ElementsMatch([]string{projectOwnerSavedQuery.Name}, savedQueryNames(listNotCreatorResp.Msg.SavedQueries))

	listCrossProjectResp, err := ctl.savedQueryServiceClient.ListSavedQueries(ctx, connect.NewRequest(&v1pb.ListSavedQueriesRequest{
		Parent: "projects/-",
		Filter: `creator == "users/demo@example.com"`,
	}))
	a.NoError(err)
	a.ElementsMatch([]string{projectOwnerSavedQuery.Name, crossProjectOwnerSavedQuery.Name}, savedQueryNames(listCrossProjectResp.Msg.SavedQueries))

	firstPageResp, err := ctl.savedQueryServiceClient.ListSavedQueries(ctx, connect.NewRequest(&v1pb.ListSavedQueriesRequest{
		Parent:   "projects/-",
		Filter:   `creator == "users/demo@example.com"`,
		PageSize: 1,
	}))
	a.NoError(err)
	a.Len(firstPageResp.Msg.SavedQueries, 1)
	a.NotEmpty(firstPageResp.Msg.NextPageToken)

	secondPageResp, err := ctl.savedQueryServiceClient.ListSavedQueries(ctx, connect.NewRequest(&v1pb.ListSavedQueriesRequest{
		Parent:    "projects/-",
		Filter:    `creator == "users/demo@example.com"`,
		PageSize:  1,
		PageToken: firstPageResp.Msg.NextPageToken,
	}))
	a.NoError(err)
	a.Len(secondPageResp.Msg.SavedQueries, 1)
	a.Empty(secondPageResp.Msg.NextPageToken)
	a.ElementsMatch(
		[]string{projectOwnerSavedQuery.Name, crossProjectOwnerSavedQuery.Name},
		append(savedQueryNames(firstPageResp.Msg.SavedQueries), savedQueryNames(secondPageResp.Msg.SavedQueries)...),
	)

	_, err = ctl.savedQueryServiceClient.ListSavedQueries(ctx, connect.NewRequest(&v1pb.ListSavedQueriesRequest{
		Parent: "projects/-",
		Filter: `visibility == "PROJECT_READ"`,
	}))
	a.Error(err)
	a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestSearchSavedQueriesFilterByFolder(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	createSavedQuery := func(title string) *v1pb.SavedQuery {
		resp, err := ctl.savedQueryServiceClient.CreateSavedQuery(ctx, connect.NewRequest(&v1pb.CreateSavedQueryRequest{
			Parent: ctl.project.Name,
			SavedQuery: &v1pb.SavedQuery{
				Title:   title,
				Content: []byte("SELECT 1;"),
			},
		}))
		a.NoError(err)
		return resp.Msg
	}
	setFolder := func(savedQuery *v1pb.SavedQuery, folder string) {
		_, err := ctl.savedQueryServiceClient.UpdateSavedQuery(ctx, connect.NewRequest(&v1pb.UpdateSavedQueryRequest{
			SavedQuery: &v1pb.SavedQuery{
				Name:   savedQuery.Name,
				Folder: folder,
			},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"folder"}},
		}))
		a.NoError(err)
	}

	alphaSavedQuery := createSavedQuery("alpha")
	setFolder(alphaSavedQuery, "alpha")
	alphaChildSavedQuery := createSavedQuery("alpha-child")
	setFolder(alphaChildSavedQuery, "alpha/beta")
	alphabetSavedQuery := createSavedQuery("alphabet")
	setFolder(alphabetSavedQuery, "alphabet")
	nestedAlphaSavedQuery := createSavedQuery("nested-alpha")
	setFolder(nestedAlphaSavedQuery, "gamma/alpha")
	rootSavedQuery := createSavedQuery("root")

	resp, err := ctl.savedQueryServiceClient.SearchSavedQueries(ctx, connect.NewRequest(&v1pb.SearchSavedQueriesRequest{
		Parent: ctl.project.Name,
		Filter: `folder == "alpha"`,
	}))
	a.NoError(err)
	a.ElementsMatch(
		[]string{alphaSavedQuery.Name},
		savedQueryNames(resp.Msg.SavedQueries),
	)
	a.NotContains(savedQueryNames(resp.Msg.SavedQueries), alphaChildSavedQuery.Name)
	a.NotContains(savedQueryNames(resp.Msg.SavedQueries), alphabetSavedQuery.Name)
	a.NotContains(savedQueryNames(resp.Msg.SavedQueries), nestedAlphaSavedQuery.Name)
	a.NotContains(savedQueryNames(resp.Msg.SavedQueries), rootSavedQuery.Name)

	childResp, err := ctl.savedQueryServiceClient.SearchSavedQueries(ctx, connect.NewRequest(&v1pb.SearchSavedQueriesRequest{
		Parent: ctl.project.Name,
		Filter: `folder == "alpha/beta"`,
	}))
	a.NoError(err)
	a.ElementsMatch([]string{alphaChildSavedQuery.Name}, savedQueryNames(childResp.Msg.SavedQueries))

	rootResp, err := ctl.savedQueryServiceClient.SearchSavedQueries(ctx, connect.NewRequest(&v1pb.SearchSavedQueriesRequest{
		Parent: ctl.project.Name,
		Filter: `folder == ""`,
	}))
	a.NoError(err)
	a.ElementsMatch([]string{rootSavedQuery.Name}, savedQueryNames(rootResp.Msg.SavedQueries))

	pageFirstSavedQuery := createSavedQuery("aaa-page")
	setFolder(pageFirstSavedQuery, "paging")
	pageSecondSavedQuery := createSavedQuery("zzz-page")
	setFolder(pageSecondSavedQuery, "paging")

	firstPageResp, err := ctl.savedQueryServiceClient.SearchSavedQueries(ctx, connect.NewRequest(&v1pb.SearchSavedQueriesRequest{
		Parent:   ctl.project.Name,
		Filter:   `folder == "paging"`,
		PageSize: 1,
	}))
	a.NoError(err)
	a.Equal([]string{pageFirstSavedQuery.Name}, savedQueryNames(firstPageResp.Msg.SavedQueries))
	a.NotEmpty(firstPageResp.Msg.NextPageToken)

	secondPageResp, err := ctl.savedQueryServiceClient.SearchSavedQueries(ctx, connect.NewRequest(&v1pb.SearchSavedQueriesRequest{
		Parent:    ctl.project.Name,
		Filter:    `folder == "paging"`,
		PageSize:  1,
		PageToken: firstPageResp.Msg.NextPageToken,
	}))
	a.NoError(err)
	a.Equal([]string{pageSecondSavedQuery.Name}, savedQueryNames(secondPageResp.Msg.SavedQueries))
}

func TestSearchSavedQueriesFilterByTitle(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	createSavedQuery := func(title string) *v1pb.SavedQuery {
		resp, err := ctl.savedQueryServiceClient.CreateSavedQuery(ctx, connect.NewRequest(&v1pb.CreateSavedQueryRequest{
			Parent: ctl.project.Name,
			SavedQuery: &v1pb.SavedQuery{
				Title:   title,
				Content: []byte("SELECT 1;"),
			},
		}))
		a.NoError(err)
		return resp.Msg
	}

	matchedSavedQuery := createSavedQuery("Production Payroll Report")
	unmatchedSavedQuery := createSavedQuery("Development Scratchpad")
	percentSavedQuery := createSavedQuery("Literal 100% match")
	underscoreSavedQuery := createSavedQuery("Literal under_score match")

	resp, err := ctl.savedQueryServiceClient.SearchSavedQueries(ctx, connect.NewRequest(&v1pb.SearchSavedQueriesRequest{
		Parent: ctl.project.Name,
		Filter: `title.contains("payroll")`,
	}))
	a.NoError(err)
	a.Equal([]string{matchedSavedQuery.Name}, savedQueryNames(resp.Msg.SavedQueries))
	a.NotContains(savedQueryNames(resp.Msg.SavedQueries), unmatchedSavedQuery.Name)

	percentResp, err := ctl.savedQueryServiceClient.SearchSavedQueries(ctx, connect.NewRequest(&v1pb.SearchSavedQueriesRequest{
		Parent: ctl.project.Name,
		Filter: `title.contains("%")`,
	}))
	a.NoError(err)
	a.Equal([]string{percentSavedQuery.Name}, savedQueryNames(percentResp.Msg.SavedQueries))

	underscoreResp, err := ctl.savedQueryServiceClient.SearchSavedQueries(ctx, connect.NewRequest(&v1pb.SearchSavedQueriesRequest{
		Parent: ctl.project.Name,
		Filter: `title.contains("_")`,
	}))
	a.NoError(err)
	a.Equal([]string{underscoreSavedQuery.Name}, savedQueryNames(underscoreResp.Msg.SavedQueries))

	_, err = ctl.savedQueryServiceClient.SearchSavedQueries(ctx, connect.NewRequest(&v1pb.SearchSavedQueriesRequest{
		Parent: ctl.project.Name,
		Filter: `content.contains("SELECT")`,
	}))
	a.Error(err)

	_, err = ctl.savedQueryServiceClient.SearchSavedQueries(ctx, connect.NewRequest(&v1pb.SearchSavedQueriesRequest{
		Parent: ctl.project.Name,
		Filter: `"literal".contains("x")`,
	}))
	a.Error(err)

	_, err = ctl.savedQueryServiceClient.SearchSavedQueries(ctx, connect.NewRequest(&v1pb.SearchSavedQueriesRequest{
		Parent: ctl.project.Name,
		Filter: `title.contains(name)`,
	}))
	a.Error(err)
}

func TestBatchUpdateSavedQueriesFilterByFolder(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	createSavedQuery := func(title, folder string) *v1pb.SavedQuery {
		resp, err := ctl.savedQueryServiceClient.CreateSavedQuery(ctx, connect.NewRequest(&v1pb.CreateSavedQueryRequest{
			Parent: ctl.project.Name,
			SavedQuery: &v1pb.SavedQuery{
				Title:   title,
				Content: []byte("SELECT 1;"),
				Folder:  folder,
			},
		}))
		a.NoError(err)
		return resp.Msg
	}
	searchByFolder := func(folder string) []string {
		resp, err := ctl.savedQueryServiceClient.SearchSavedQueries(ctx, connect.NewRequest(&v1pb.SearchSavedQueriesRequest{
			Parent: ctl.project.Name,
			Filter: fmt.Sprintf("folder == %q", folder),
		}))
		a.NoError(err)
		return savedQueryNames(resp.Msg.SavedQueries)
	}

	directSavedQuery := createSavedQuery("old-direct", "old")
	childSavedQuery := createSavedQuery("old-child", "old/child")
	otherSavedQuery := createSavedQuery("other", "other")

	directResp, err := ctl.savedQueryServiceClient.BatchUpdateSavedQueries(ctx, connect.NewRequest(&v1pb.BatchUpdateSavedQueriesRequest{
		Parent:     ctl.project.Name,
		Filter:     `folder == "old"`,
		SavedQuery: &v1pb.SavedQuery{Folder: "new"},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"folder"}},
	}))
	a.NoError(err)
	a.Equal(int32(1), directResp.Msg.UpdatedCount)

	childResp, err := ctl.savedQueryServiceClient.BatchUpdateSavedQueries(ctx, connect.NewRequest(&v1pb.BatchUpdateSavedQueriesRequest{
		Parent:     ctl.project.Name,
		Filter:     `folder == "old/child"`,
		SavedQuery: &v1pb.SavedQuery{Folder: "new/child"},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"folder"}},
	}))
	a.NoError(err)
	a.Equal(int32(1), childResp.Msg.UpdatedCount)

	a.ElementsMatch([]string{directSavedQuery.Name}, searchByFolder("new"))
	a.ElementsMatch([]string{childSavedQuery.Name}, searchByFolder("new/child"))
	a.ElementsMatch([]string{otherSavedQuery.Name}, searchByFolder("other"))
	a.Empty(searchByFolder("old"))
	a.Empty(searchByFolder("old/child"))

	starResp, err := ctl.savedQueryServiceClient.UpdateSavedQueryStar(ctx, connect.NewRequest(&v1pb.UpdateSavedQueryStarRequest{
		Name:    directSavedQuery.Name,
		Starred: true,
	}))
	a.NoError(err)
	a.True(starResp.Msg.Starred)
	starredResp, err := ctl.savedQueryServiceClient.SearchSavedQueries(ctx, connect.NewRequest(&v1pb.SearchSavedQueriesRequest{
		Parent: ctl.project.Name,
		Filter: `folder == "new" && starred == true`,
	}))
	a.NoError(err)
	a.ElementsMatch([]string{directSavedQuery.Name}, savedQueryNames(starredResp.Msg.SavedQueries))

	unstarResp, err := ctl.savedQueryServiceClient.UpdateSavedQueryStar(ctx, connect.NewRequest(&v1pb.UpdateSavedQueryStarRequest{
		Name:    directSavedQuery.Name,
		Starred: false,
	}))
	a.NoError(err)
	a.False(unstarResp.Msg.Starred)

	nameResp, err := ctl.savedQueryServiceClient.BatchUpdateSavedQueries(ctx, connect.NewRequest(&v1pb.BatchUpdateSavedQueriesRequest{
		Parent:     ctl.project.Name,
		Filter:     fmt.Sprintf(`name in [%q,%q]`, directSavedQuery.Name, childSavedQuery.Name),
		SavedQuery: &v1pb.SavedQuery{Folder: "selected"},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"folder"}},
	}))
	a.NoError(err)
	a.Equal(int32(2), nameResp.Msg.UpdatedCount)
	a.ElementsMatch([]string{directSavedQuery.Name, childSavedQuery.Name}, searchByFolder("selected"))
	a.ElementsMatch([]string{otherSavedQuery.Name}, searchByFolder("other"))
}

func TestSearchSavedQueryFoldersReturnsCallerFolders(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	ownerToken := ctl.authInterceptor.token

	createSavedQuery := func(title, folder string) {
		_, err := ctl.savedQueryServiceClient.CreateSavedQuery(ctx, connect.NewRequest(&v1pb.CreateSavedQueryRequest{
			Parent: ctl.project.Name,
			SavedQuery: &v1pb.SavedQuery{
				Title:   title,
				Content: []byte("SELECT 1;"),
				Folder:  folder,
			},
		}))
		a.NoError(err)
	}

	createSavedQuery("owner", "owner/child")
	createSavedQuery("owner-second", "owner-second")
	createSavedQuery("unfiled", "")

	otherEmail := fmt.Sprintf("saved-query-folder-%s@example.com", generateRandomString("user"))
	otherPassword := "1024bytebase"
	otherUser, err := ctl.userServiceClient.CreateUser(ctx, connect.NewRequest(&v1pb.CreateUserRequest{
		User: &v1pb.User{
			Email:    otherEmail,
			Password: otherPassword,
			Title:    "Saved Query Folder User",
		},
	}))
	a.NoError(err)
	_, err = ctl.addMemberToWorkspaceIAM(ctx, otherUser.Msg.Workspace, fmt.Sprintf("user:%s", otherEmail), "roles/workspaceMember")
	a.NoError(err)
	policyResp, err := ctl.projectServiceClient.GetIamPolicy(ctx, connect.NewRequest(&v1pb.GetIamPolicyRequest{
		Resource: ctl.project.Name,
	}))
	a.NoError(err)
	policy := policyResp.Msg
	policy.Bindings = append(policy.Bindings, &v1pb.Binding{
		Role:    "roles/sqlEditorUser",
		Members: []string{fmt.Sprintf("user:%s", otherEmail)},
	})
	_, err = ctl.projectServiceClient.SetIamPolicy(ctx, connect.NewRequest(&v1pb.SetIamPolicyRequest{
		Resource: ctl.project.Name,
		Policy:   policy,
	}))
	a.NoError(err)

	loginResp, err := ctl.authServiceClient.Login(ctx, connect.NewRequest(&v1pb.LoginRequest{
		Email:    otherEmail,
		Password: otherPassword,
	}))
	a.NoError(err)
	ctl.authInterceptor.token = loginResp.Msg.Token
	createSavedQuery("other-users", "theirs/deep")

	// Each caller sees only their own folder paths, with ancestor prefixes.
	otherFolders, err := ctl.savedQueryServiceClient.SearchSavedQueryFolders(ctx, connect.NewRequest(&v1pb.SearchSavedQueryFoldersRequest{
		Parent: ctl.project.Name,
	}))
	a.NoError(err)
	a.Equal([]string{"theirs", "theirs/deep"}, otherFolders.Msg.Folders)

	ctl.authInterceptor.token = ownerToken
	ownerFolders, err := ctl.savedQueryServiceClient.SearchSavedQueryFolders(ctx, connect.NewRequest(&v1pb.SearchSavedQueryFoldersRequest{
		Parent: ctl.project.Name,
	}))
	a.NoError(err)
	a.Equal([]string{"owner", "owner-second", "owner/child"}, ownerFolders.Msg.Folders)
}

func TestSearchSavedQueriesRejectsWildcardProject(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	_, err = ctl.savedQueryServiceClient.SearchSavedQueries(ctx, connect.NewRequest(&v1pb.SearchSavedQueriesRequest{
		Parent: "projects/-",
	}))
	a.Error(err)
	a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))
}

func savedQueryNames(savedQueries []*v1pb.SavedQuery) []string {
	names := make([]string, 0, len(savedQueries))
	for _, savedQuery := range savedQueries {
		names = append(names, savedQuery.Name)
	}
	return names
}
