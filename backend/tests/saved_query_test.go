package tests

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/bytebase/bytebase/backend/common"
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

	// Creating a saved query now takes bb.savedQueries.create on the parent
	// project, which workspaceMember does not carry -- a workspace role would
	// grant it in every project. The scenario needs this user to own a saved
	// query in ctl.project, so give them a role there.
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

	// Batch moves normalize like every other write path, so a folder sent
	// with boundary slashes still lands where `folder == "boxed"` finds it.
	slashResp, err := ctl.savedQueryServiceClient.BatchUpdateSavedQueries(ctx, connect.NewRequest(&v1pb.BatchUpdateSavedQueriesRequest{
		Parent:     ctl.project.Name,
		Filter:     `folder == "selected"`,
		SavedQuery: &v1pb.SavedQuery{Folder: "/boxed/"},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"folder"}},
	}))
	a.NoError(err)
	a.Equal(int32(2), slashResp.Msg.UpdatedCount)
	a.ElementsMatch([]string{directSavedQuery.Name, childSavedQuery.Name}, searchByFolder("boxed"))

	// A path the filter could never match is rejected, not written.
	_, err = ctl.savedQueryServiceClient.BatchUpdateSavedQueries(ctx, connect.NewRequest(&v1pb.BatchUpdateSavedQueriesRequest{
		Parent:     ctl.project.Name,
		Filter:     `folder == "boxed"`,
		SavedQuery: &v1pb.SavedQuery{Folder: "bad//path"},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"folder"}},
	}))
	a.Error(err)
	a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))
	a.ElementsMatch([]string{directSavedQuery.Name, childSavedQuery.Name}, searchByFolder("boxed"))
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

	// A plain project member gets nothing for everyone-else's folders: the
	// filter cannot widen what the caller may read.
	const ownerEmail = "demo@example.com"
	notMineForOther, err := ctl.savedQueryServiceClient.SearchSavedQueryFolders(ctx, connect.NewRequest(&v1pb.SearchSavedQueryFoldersRequest{
		Parent: ctl.project.Name,
		Filter: fmt.Sprintf(`creator != "users/%s"`, otherEmail),
	}))
	a.NoError(err)
	a.Empty(notMineForOther.Msg.Folders)

	// Unfiltered means "every folder I can read", so the admin backstop sees
	// the other creator's folders too. The My/Shared split the SQL Editor
	// renders comes from the creator filter, asserted below.
	ctl.authInterceptor.token = ownerToken
	ownerFolders, err := ctl.savedQueryServiceClient.SearchSavedQueryFolders(ctx, connect.NewRequest(&v1pb.SearchSavedQueryFoldersRequest{
		Parent: ctl.project.Name,
	}))
	a.NoError(err)
	a.Equal([]string{"owner", "owner-second", "owner/child", "theirs", "theirs/deep"}, ownerFolders.Msg.Folders)

	ownFolders, err := ctl.savedQueryServiceClient.SearchSavedQueryFolders(ctx, connect.NewRequest(&v1pb.SearchSavedQueryFoldersRequest{
		Parent: ctl.project.Name,
		Filter: fmt.Sprintf(`creator == "users/%s"`, ownerEmail),
	}))
	a.NoError(err)
	a.Equal([]string{"owner", "owner-second", "owner/child"}, ownFolders.Msg.Folders)

	// The admin backstop is what makes the SQL Editor's shared tree work:
	// everyone-else's folders resolve server-side, so a cold cache still
	// renders the folders holding saved queries the caller never opened.
	sharedFolders, err := ctl.savedQueryServiceClient.SearchSavedQueryFolders(ctx, connect.NewRequest(&v1pb.SearchSavedQueryFoldersRequest{
		Parent: ctl.project.Name,
		Filter: fmt.Sprintf(`creator != "users/%s"`, ownerEmail),
	}))
	a.NoError(err)
	a.Equal([]string{"theirs", "theirs/deep"}, sharedFolders.Msg.Folders)
}

func TestGetSavedQueryHidesUnreadableRows(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	ownerToken := ctl.authInterceptor.token
	created, err := ctl.savedQueryServiceClient.CreateSavedQuery(ctx, connect.NewRequest(&v1pb.CreateSavedQueryRequest{
		Parent: ctl.project.Name,
		SavedQuery: &v1pb.SavedQuery{
			Title:   "Private title that must not leak",
			Content: []byte("SELECT 1;"),
		},
	}))
	a.NoError(err)

	otherEmail := fmt.Sprintf("saved-query-probe-%s@example.com", generateRandomString("user"))
	const otherPassword = "1024bytebase"
	otherUser, err := ctl.userServiceClient.CreateUser(ctx, connect.NewRequest(&v1pb.CreateUserRequest{
		User: &v1pb.User{
			Email:    otherEmail,
			Password: otherPassword,
			Title:    "Saved Query Probe User",
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
	defer func() { ctl.authInterceptor.token = ownerToken }()

	// A saved query someone else owns answers exactly like a name that was
	// never issued -- same code, and no title in the message.
	_, err = ctl.savedQueryServiceClient.GetSavedQuery(ctx, connect.NewRequest(&v1pb.GetSavedQueryRequest{
		Name: created.Msg.Name,
	}))
	a.Error(err)
	a.Equal(connect.CodeNotFound, connect.CodeOf(err))
	a.NotContains(err.Error(), "Private title that must not leak")

	missing, err := ctl.savedQueryServiceClient.GetSavedQuery(ctx, connect.NewRequest(&v1pb.GetSavedQueryRequest{
		Name: fmt.Sprintf("%s/savedQueries/909909", ctl.project.Name),
	}))
	a.Error(err)
	a.Nil(missing)
	a.Equal(connect.CodeNotFound, connect.CodeOf(err))

	// Writes answer the same way, so a name cannot be probed by trying to
	// change or remove it either.
	_, err = ctl.savedQueryServiceClient.UpdateSavedQuery(ctx, connect.NewRequest(&v1pb.UpdateSavedQueryRequest{
		SavedQuery: &v1pb.SavedQuery{Name: created.Msg.Name, Title: "probe"},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"title"}},
	}))
	a.Error(err)
	a.Equal(connect.CodeNotFound, connect.CodeOf(err))

	_, err = ctl.savedQueryServiceClient.DeleteSavedQuery(ctx, connect.NewRequest(&v1pb.DeleteSavedQueryRequest{
		Name: created.Msg.Name,
	}))
	a.Error(err)
	a.Equal(connect.CodeNotFound, connect.CodeOf(err))
}

func TestListSavedQueriesReturnsWholeStatement(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	const ownerEmail = "demo@example.com"

	// Longer than the display cap Search truncates at, which the governance
	// surface must not apply -- a lister cannot Get another creator's row to
	// fetch the remainder.
	statement := append([]byte("SELECT 1; -- "), bytes.Repeat([]byte("x"), common.MaxSheetSize)...)
	created, err := ctl.savedQueryServiceClient.CreateSavedQuery(ctx, connect.NewRequest(&v1pb.CreateSavedQueryRequest{
		Parent: ctl.project.Name,
		SavedQuery: &v1pb.SavedQuery{
			Title:   "long statement",
			Content: statement,
		},
	}))
	a.NoError(err)

	// The governance list filters on metadata -- creator is the enumeration
	// the offboarding review is built around, and the only variable this
	// filter accepts.
	listResp, err := ctl.savedQueryServiceClient.ListSavedQueries(ctx, connect.NewRequest(&v1pb.ListSavedQueriesRequest{
		Parent: ctl.project.Name,
		Filter: fmt.Sprintf(`creator == "users/%s"`, ownerEmail),
	}))
	a.NoError(err)
	listed := findSavedQueryByName(listResp.Msg.SavedQueries, created.Msg.Name)
	a.NotNil(listed)
	a.Equal(int64(len(statement)), listed.ContentSize)
	a.Equal(len(statement), len(listed.Content))

	searchResp, err := ctl.savedQueryServiceClient.SearchSavedQueries(ctx, connect.NewRequest(&v1pb.SearchSavedQueriesRequest{
		Parent: ctl.project.Name,
		Filter: fmt.Sprintf(`name == %q`, created.Msg.Name),
	}))
	a.NoError(err)
	a.Len(searchResp.Msg.SavedQueries, 1)
	a.Less(len(searchResp.Msg.SavedQueries[0].Content), len(statement))
}

func TestCreateSavedQueryRejectsArchivedProject(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	projectID := generateRandomString("sq-archived")
	projectResp, err := ctl.projectServiceClient.CreateProject(ctx, connect.NewRequest(&v1pb.CreateProjectRequest{
		Project: &v1pb.Project{
			Name:  fmt.Sprintf("projects/%s", projectID),
			Title: projectID,
		},
		ProjectId: projectID,
	}))
	a.NoError(err)

	_, err = ctl.projectServiceClient.DeleteProject(ctx, connect.NewRequest(&v1pb.DeleteProjectRequest{
		Name: projectResp.Msg.Name,
	}))
	a.NoError(err)

	// Resource resolution returns archived projects, so without an explicit
	// check the create would reach the store's purge fence and read as a 500.
	_, err = ctl.savedQueryServiceClient.CreateSavedQuery(ctx, connect.NewRequest(&v1pb.CreateSavedQueryRequest{
		Parent: projectResp.Msg.Name,
		SavedQuery: &v1pb.SavedQuery{
			Title:   "into an archived project",
			Content: []byte("SELECT 1;"),
		},
	}))
	a.Error(err)
	a.Equal(connect.CodeNotFound, connect.CodeOf(err))
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

func findSavedQueryByName(savedQueries []*v1pb.SavedQuery, name string) *v1pb.SavedQuery {
	for _, savedQuery := range savedQueries {
		if savedQuery.Name == name {
			return savedQuery
		}
	}
	return nil
}
