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

func TestMoveSavedQueries(t *testing.T) {
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
	move := func(request *v1pb.MoveMySavedQueriesRequest) (int32, error) {
		request.Parent = ctl.project.Name
		resp, err := ctl.savedQueryServiceClient.MoveMySavedQueries(ctx, connect.NewRequest(request))
		if err != nil {
			return 0, err
		}
		return resp.Msg.MovedCount, nil
	}

	directSavedQuery := createSavedQuery("old-direct", "old")
	childSavedQuery := createSavedQuery("old-child", "old/child")
	otherSavedQuery := createSavedQuery("other", "other")

	// Moving a folder carries its descendants: one call, not one per path.
	moved, err := move(&v1pb.MoveMySavedQueriesRequest{SourceFolder: "old", TargetFolder: "new"})
	a.NoError(err)
	a.Equal(int32(2), moved)
	a.ElementsMatch([]string{directSavedQuery.Name}, searchByFolder("new"))
	a.ElementsMatch([]string{childSavedQuery.Name}, searchByFolder("new/child"))
	a.ElementsMatch([]string{otherSavedQuery.Name}, searchByFolder("other"))
	a.Empty(searchByFolder("old"))
	a.Empty(searchByFolder("old/child"))

	// Naming saved queries moves exactly those, descendants irrelevant.
	moved, err = move(&v1pb.MoveMySavedQueriesRequest{
		Names:        []string{directSavedQuery.Name, childSavedQuery.Name},
		TargetFolder: "selected",
	})
	a.NoError(err)
	a.Equal(int32(2), moved)
	a.ElementsMatch([]string{directSavedQuery.Name, childSavedQuery.Name}, searchByFolder("selected"))
	a.ElementsMatch([]string{otherSavedQuery.Name}, searchByFolder("other"))

	// Folders normalize like every other write path, so boundary slashes still
	// land where `folder == "boxed"` finds them.
	moved, err = move(&v1pb.MoveMySavedQueriesRequest{SourceFolder: "selected", TargetFolder: "/boxed/"})
	a.NoError(err)
	a.Equal(int32(2), moved)
	a.ElementsMatch([]string{directSavedQuery.Name, childSavedQuery.Name}, searchByFolder("boxed"))

	// An unfilable path is rejected, not written.
	_, err = move(&v1pb.MoveMySavedQueriesRequest{SourceFolder: "boxed", TargetFolder: "bad//path"})
	a.Error(err)
	a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))
	a.ElementsMatch([]string{directSavedQuery.Name, childSavedQuery.Name}, searchByFolder("boxed"))

	// Exactly one of names / source_folder.
	_, err = move(&v1pb.MoveMySavedQueriesRequest{TargetFolder: "x"})
	a.Error(err)
	a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))
	_, err = move(&v1pb.MoveMySavedQueriesRequest{
		Names:        []string{directSavedQuery.Name},
		SourceFolder: "boxed",
		TargetFolder: "x",
	})
	a.Error(err)
	a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))

	// A folder cannot swallow itself.
	_, err = move(&v1pb.MoveMySavedQueriesRequest{SourceFolder: "boxed", TargetFolder: "boxed/inner"})
	a.Error(err)
	a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))

	// Unfiling: an empty target drops the prefix rather than leaving "/tail".
	moved, err = move(&v1pb.MoveMySavedQueriesRequest{SourceFolder: "boxed", TargetFolder: ""})
	a.NoError(err)
	a.Equal(int32(2), moved)
	a.Empty(searchByFolder("boxed"))
	a.ElementsMatch([]string{directSavedQuery.Name, childSavedQuery.Name}, searchByFolder(""))
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
	theirSavedQuery := createSavedQuery("other-users", "theirs/deep")

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

	// Unfiltered means "every folder holding a saved query I can read". The
	// workspace admin holds project-level bb.savedQueries.get, which reads
	// every row, so the other creator's folders show up too.
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

	// A grant is what makes the SQL Editor's shared tree work. The folder of a
	// saved query somebody else filed resolves server-side once it is shared,
	// so a cold cache still renders folders holding rows the caller never
	// opened -- without one, a foldered shared row would be unreachable,
	// because the client can only expand into folders this call reports.
	ctl.authInterceptor.token = loginResp.Msg.Token
	theirPolicy, err := ctl.savedQueryServiceClient.GetSavedQueryPolicy(ctx, connect.NewRequest(&v1pb.GetSavedQueryPolicyRequest{
		Resource: theirSavedQuery.Name,
	}))
	a.NoError(err)
	_, err = ctl.savedQueryServiceClient.SetSavedQueryPolicy(ctx, connect.NewRequest(&v1pb.SetSavedQueryPolicyRequest{
		Resource: theirSavedQuery.Name,
		Policy: &v1pb.SavedQueryPolicy{
			Etag: theirPolicy.Msg.Etag,
			Bindings: []*v1pb.SavedQueryBinding{{
				Level:   v1pb.SavedQueryBinding_VIEWER,
				Members: []string{fmt.Sprintf("user:%s", ownerEmail)},
			}},
		},
	}))
	a.NoError(err)

	ctl.authInterceptor.token = ownerToken
	sharedFolders, err := ctl.savedQueryServiceClient.SearchSavedQueryFolders(ctx, connect.NewRequest(&v1pb.SearchSavedQueryFoldersRequest{
		Parent: ctl.project.Name,
		Filter: "shared == true",
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

// TestSavedQuerySharing covers the per-object grant model end to end: a
// private saved query is invisible to a co-member, a VIEWER binding grants
// get, an EDITOR binding grants get + update but never deletion, and the
// policy write is compare-and-swap.
func TestSavedQuerySharing(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	ownerToken := ctl.authInterceptor.token

	granteeEmail := fmt.Sprintf("saved-query-grantee-%s@example.com", generateRandomString("user"))
	granteePassword := "1024bytebase"
	grantee, err := ctl.userServiceClient.CreateUser(ctx, connect.NewRequest(&v1pb.CreateUserRequest{
		User: &v1pb.User{
			Email:    granteeEmail,
			Password: granteePassword,
			Title:    "Saved Query Grantee",
		},
	}))
	a.NoError(err)
	_, err = ctl.addMemberToWorkspaceIAM(ctx, grantee.Msg.Workspace, fmt.Sprintf("user:%s", granteeEmail), "roles/workspaceMember")
	a.NoError(err)

	// The grantee needs the project's saved-query surface to discover anything;
	// the binding is what carries content access on top of it.
	policyResp, err := ctl.projectServiceClient.GetIamPolicy(ctx, connect.NewRequest(&v1pb.GetIamPolicyRequest{
		Resource: ctl.project.Name,
	}))
	a.NoError(err)
	projectPolicy := policyResp.Msg
	projectPolicy.Bindings = append(projectPolicy.Bindings, &v1pb.Binding{
		Role:    "roles/sqlEditorUser",
		Members: []string{fmt.Sprintf("user:%s", granteeEmail)},
	})
	_, err = ctl.projectServiceClient.SetIamPolicy(ctx, connect.NewRequest(&v1pb.SetIamPolicyRequest{
		Resource: ctl.project.Name,
		Policy:   projectPolicy,
	}))
	a.NoError(err)

	createResp, err := ctl.savedQueryServiceClient.CreateSavedQuery(ctx, connect.NewRequest(&v1pb.CreateSavedQueryRequest{
		Parent: ctl.project.Name,
		SavedQuery: &v1pb.SavedQuery{
			Title:   "shared-query",
			Content: []byte("SELECT 1;"),
		},
	}))
	a.NoError(err)
	savedQuery := createResp.Msg

	// A new saved query starts private: no bindings.
	getPolicyResp, err := ctl.savedQueryServiceClient.GetSavedQueryPolicy(ctx, connect.NewRequest(&v1pb.GetSavedQueryPolicyRequest{
		Resource: savedQuery.Name,
	}))
	a.NoError(err)
	a.Empty(getPolicyResp.Msg.Bindings)
	a.NotEmpty(getPolicyResp.Msg.Etag, "etag is what the next write has to present")
	initialEtag := getPolicyResp.Msg.Etag

	loginResp, err := ctl.authServiceClient.Login(ctx, connect.NewRequest(&v1pb.LoginRequest{
		Email:    granteeEmail,
		Password: granteePassword,
	}))
	a.NoError(err)
	granteeToken := loginResp.Msg.Token

	asGrantee := func(f func()) {
		ctl.authInterceptor.token = granteeToken
		defer func() { ctl.authInterceptor.token = ownerToken }()
		f()
	}

	// Private: invisible, and NotFound rather than PermissionDenied so the name
	// cannot be probed.
	asGrantee(func() {
		_, err := ctl.savedQueryServiceClient.GetSavedQuery(ctx, connect.NewRequest(&v1pb.GetSavedQueryRequest{
			Name: savedQuery.Name,
		}))
		a.Error(err)
		a.Equal(connect.CodeNotFound, connect.CodeOf(err))

		searchResp, err := ctl.savedQueryServiceClient.SearchSavedQueries(ctx, connect.NewRequest(&v1pb.SearchSavedQueriesRequest{
			Parent: ctl.project.Name,
		}))
		a.NoError(err)
		a.NotContains(savedQueryNames(searchResp.Msg.SavedQueries), savedQuery.Name)
	})

	setPolicy := func(level v1pb.SavedQueryBinding_Level, etag string) *v1pb.SavedQueryPolicy {
		resp, err := ctl.savedQueryServiceClient.SetSavedQueryPolicy(ctx, connect.NewRequest(&v1pb.SetSavedQueryPolicyRequest{
			Resource: savedQuery.Name,
			Policy: &v1pb.SavedQueryPolicy{
				Etag: etag,
				Bindings: []*v1pb.SavedQueryBinding{
					{Level: level, Members: []string{fmt.Sprintf("user:%s", granteeEmail)}},
				},
			},
		}))
		a.NoError(err)
		return resp.Msg
	}

	viewerPolicy := setPolicy(v1pb.SavedQueryBinding_VIEWER, initialEtag)
	a.NotEqual(initialEtag, viewerPolicy.Etag, "the etag moves with the grants")

	// A stale etag must not silently reinstate what somebody just revoked.
	_, err = ctl.savedQueryServiceClient.SetSavedQueryPolicy(ctx, connect.NewRequest(&v1pb.SetSavedQueryPolicyRequest{
		Resource: savedQuery.Name,
		Policy: &v1pb.SavedQueryPolicy{
			Etag:     initialEtag,
			Bindings: nil,
		},
	}))
	a.Error(err)
	a.Equal(connect.CodeAborted, connect.CodeOf(err))

	// VIEWER: reads, and shows up under shared == true, but cannot write.
	asGrantee(func() {
		getResp, err := ctl.savedQueryServiceClient.GetSavedQuery(ctx, connect.NewRequest(&v1pb.GetSavedQueryRequest{
			Name: savedQuery.Name,
		}))
		a.NoError(err)
		a.Equal("shared-query", getResp.Msg.Title)

		sharedResp, err := ctl.savedQueryServiceClient.SearchSavedQueries(ctx, connect.NewRequest(&v1pb.SearchSavedQueriesRequest{
			Parent: ctl.project.Name,
			Filter: "shared == true",
		}))
		a.NoError(err)
		a.Contains(savedQueryNames(sharedResp.Msg.SavedQueries), savedQuery.Name)

		_, err = ctl.savedQueryServiceClient.UpdateSavedQuery(ctx, connect.NewRequest(&v1pb.UpdateSavedQueryRequest{
			SavedQuery: &v1pb.SavedQuery{Name: savedQuery.Name, Title: "renamed by viewer"},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"title"}},
		}))
		a.Error(err)
		a.Equal(connect.CodeNotFound, connect.CodeOf(err), "a missing permission answers NotFound, the one masking rule")

		// Grantees read their own level through the policy; they cannot rewrite it.
		policy, err := ctl.savedQueryServiceClient.GetSavedQueryPolicy(ctx, connect.NewRequest(&v1pb.GetSavedQueryPolicyRequest{
			Resource: savedQuery.Name,
		}))
		a.NoError(err)
		a.Len(policy.Msg.Bindings, 1)
		a.Equal(v1pb.SavedQueryBinding_VIEWER, policy.Msg.Bindings[0].Level)

		_, err = ctl.savedQueryServiceClient.SetSavedQueryPolicy(ctx, connect.NewRequest(&v1pb.SetSavedQueryPolicyRequest{
			Resource: savedQuery.Name,
			Policy:   &v1pb.SavedQueryPolicy{Etag: policy.Msg.Etag},
		}))
		a.Error(err)
		a.Equal(connect.CodeNotFound, connect.CodeOf(err))
	})

	editorPolicy := setPolicy(v1pb.SavedQueryBinding_EDITOR, viewerPolicy.Etag)

	// EDITOR: holds get and update — content and folder both write — but
	// never deletes.
	asGrantee(func() {
		updated, err := ctl.savedQueryServiceClient.UpdateSavedQuery(ctx, connect.NewRequest(&v1pb.UpdateSavedQueryRequest{
			SavedQuery: &v1pb.SavedQuery{Name: savedQuery.Name, Content: []byte("SELECT 2;")},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"content"}},
		}))
		a.NoError(err)
		a.True(bytes.Equal([]byte("SELECT 2;"), updated.Msg.Content))

		refiled, err := ctl.savedQueryServiceClient.UpdateSavedQuery(ctx, connect.NewRequest(&v1pb.UpdateSavedQueryRequest{
			SavedQuery: &v1pb.SavedQuery{Name: savedQuery.Name, Folder: "grantee"},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"folder"}},
		}))
		a.NoError(err)
		a.Equal("grantee", refiled.Msg.Folder)

		_, err = ctl.savedQueryServiceClient.DeleteSavedQuery(ctx, connect.NewRequest(&v1pb.DeleteSavedQueryRequest{
			Name: savedQuery.Name,
		}))
		a.Error(err)
		a.Equal(connect.CodeNotFound, connect.CodeOf(err), "sharing for editing must not cost the owner the saved query")
	})

	// Revoking drops the grantee back to invisible.
	_, err = ctl.savedQueryServiceClient.SetSavedQueryPolicy(ctx, connect.NewRequest(&v1pb.SetSavedQueryPolicyRequest{
		Resource: savedQuery.Name,
		Policy:   &v1pb.SavedQueryPolicy{Etag: editorPolicy.Etag},
	}))
	a.NoError(err)

	asGrantee(func() {
		_, err := ctl.savedQueryServiceClient.GetSavedQuery(ctx, connect.NewRequest(&v1pb.GetSavedQueryRequest{
			Name: savedQuery.Name,
		}))
		a.Error(err)
		a.Equal(connect.CodeNotFound, connect.CodeOf(err))
	})
}

// TestSavedQueryPerVerbRoleGrants covers the project-level permissions one by
// one: get reads a private saved query (and widens Search) without granting
// update or delete; update writes without granting delete; sharing is the
// creator's alone whatever the role carries.
func TestSavedQueryPerVerbRoleGrants(t *testing.T) {
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
			Title:   "per-verb target",
			Content: []byte("SELECT 1;"),
		},
	}))
	a.NoError(err)
	savedQuery := created.Msg

	auditorEmail := fmt.Sprintf("saved-query-verb-%s@example.com", generateRandomString("user"))
	const auditorPassword = "1024bytebase"
	auditor, err := ctl.userServiceClient.CreateUser(ctx, connect.NewRequest(&v1pb.CreateUserRequest{
		User: &v1pb.User{
			Email:    auditorEmail,
			Password: auditorPassword,
			Title:    "Saved Query Verb User",
		},
	}))
	a.NoError(err)
	_, err = ctl.addMemberToWorkspaceIAM(ctx, auditor.Msg.Workspace, fmt.Sprintf("user:%s", auditorEmail), "roles/workspaceMember")
	a.NoError(err)

	grantRole := func(permissions ...string) {
		roleID := generateRandomString("saved-query-verb-role")
		_, err := ctl.roleServiceClient.CreateRole(ctx, connect.NewRequest(&v1pb.CreateRoleRequest{
			Role: &v1pb.Role{
				Title:       roleID,
				Permissions: permissions,
			},
			RoleId: roleID,
		}))
		a.NoError(err)
		policyResp, err := ctl.projectServiceClient.GetIamPolicy(ctx, connect.NewRequest(&v1pb.GetIamPolicyRequest{
			Resource: ctl.project.Name,
		}))
		a.NoError(err)
		policy := policyResp.Msg
		policy.Bindings = append(policy.Bindings, &v1pb.Binding{
			Role:    fmt.Sprintf("roles/%s", roleID),
			Members: []string{fmt.Sprintf("user:%s", auditorEmail)},
		})
		_, err = ctl.projectServiceClient.SetIamPolicy(ctx, connect.NewRequest(&v1pb.SetIamPolicyRequest{
			Resource: ctl.project.Name,
			Policy:   policy,
		}))
		a.NoError(err)
	}

	loginResp, err := ctl.authServiceClient.Login(ctx, connect.NewRequest(&v1pb.LoginRequest{
		Email:    auditorEmail,
		Password: auditorPassword,
	}))
	a.NoError(err)
	auditorToken := loginResp.Msg.Token

	asAuditor := func(f func()) {
		ctl.authInterceptor.token = auditorToken
		defer func() { ctl.authInterceptor.token = ownerToken }()
		f()
	}

	// get alone: reads the private saved query, widens Search, stars it —
	// and stops there: no update, no delete, no re-share.
	grantRole("bb.savedQueries.search", "bb.savedQueries.get")
	asAuditor(func() {
		getResp, err := ctl.savedQueryServiceClient.GetSavedQuery(ctx, connect.NewRequest(&v1pb.GetSavedQueryRequest{
			Name: savedQuery.Name,
		}))
		a.NoError(err)
		a.Equal("per-verb target", getResp.Msg.Title)

		searchResp, err := ctl.savedQueryServiceClient.SearchSavedQueries(ctx, connect.NewRequest(&v1pb.SearchSavedQueriesRequest{
			Parent: ctl.project.Name,
		}))
		a.NoError(err)
		a.Contains(savedQueryNames(searchResp.Msg.SavedQueries), savedQuery.Name)

		// Visible through the role grant, not through a binding, so the
		// Shared view stays empty.
		sharedResp, err := ctl.savedQueryServiceClient.SearchSavedQueries(ctx, connect.NewRequest(&v1pb.SearchSavedQueriesRequest{
			Parent: ctl.project.Name,
			Filter: "shared == true",
		}))
		a.NoError(err)
		a.NotContains(savedQueryNames(sharedResp.Msg.SavedQueries), savedQuery.Name)

		starResp, err := ctl.savedQueryServiceClient.UpdateSavedQueryStar(ctx, connect.NewRequest(&v1pb.UpdateSavedQueryStarRequest{
			Name:    savedQuery.Name,
			Starred: true,
		}))
		a.NoError(err)
		a.True(starResp.Msg.Starred)

		_, err = ctl.savedQueryServiceClient.UpdateSavedQuery(ctx, connect.NewRequest(&v1pb.UpdateSavedQueryRequest{
			SavedQuery: &v1pb.SavedQuery{Name: savedQuery.Name, Title: "renamed by get"},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"title"}},
		}))
		a.Error(err)
		a.Equal(connect.CodeNotFound, connect.CodeOf(err))

		_, err = ctl.savedQueryServiceClient.DeleteSavedQuery(ctx, connect.NewRequest(&v1pb.DeleteSavedQueryRequest{
			Name: savedQuery.Name,
		}))
		a.Error(err)
		a.Equal(connect.CodeNotFound, connect.CodeOf(err))

		// The policy pair has its own permissions: get alone reads neither.
		_, err = ctl.savedQueryServiceClient.GetSavedQueryPolicy(ctx, connect.NewRequest(&v1pb.GetSavedQueryPolicyRequest{
			Resource: savedQuery.Name,
		}))
		a.Error(err)
		a.Equal(connect.CodeNotFound, connect.CodeOf(err))
		_, err = ctl.savedQueryServiceClient.SetSavedQueryPolicy(ctx, connect.NewRequest(&v1pb.SetSavedQueryPolicyRequest{
			Resource: savedQuery.Name,
			Policy:   &v1pb.SavedQueryPolicy{},
		}))
		a.Error(err)
		a.Equal(connect.CodeNotFound, connect.CodeOf(err), "no predefined role carries setIamPolicy")
	})

	// update on top: writes content and re-files, still no delete.
	grantRole("bb.savedQueries.update")
	asAuditor(func() {
		updated, err := ctl.savedQueryServiceClient.UpdateSavedQuery(ctx, connect.NewRequest(&v1pb.UpdateSavedQueryRequest{
			SavedQuery: &v1pb.SavedQuery{Name: savedQuery.Name, Content: []byte("SELECT 2;")},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"content"}},
		}))
		a.NoError(err)
		a.True(bytes.Equal([]byte("SELECT 2;"), updated.Msg.Content))

		refiled, err := ctl.savedQueryServiceClient.UpdateSavedQuery(ctx, connect.NewRequest(&v1pb.UpdateSavedQueryRequest{
			SavedQuery: &v1pb.SavedQuery{Name: savedQuery.Name, Folder: "verbed"},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"folder"}},
		}))
		a.NoError(err)
		a.Equal("verbed", refiled.Msg.Folder)

		_, err = ctl.savedQueryServiceClient.DeleteSavedQuery(ctx, connect.NewRequest(&v1pb.DeleteSavedQueryRequest{
			Name: savedQuery.Name,
		}))
		a.Error(err)
		a.Equal(connect.CodeNotFound, connect.CodeOf(err))
	})

	// The policy pair, granted: reads the policy and rewrites it — the
	// re-share capability that no predefined role carries and only an
	// explicit custom-role grant confers.
	grantRole("bb.savedQueries.getIamPolicy", "bb.savedQueries.setIamPolicy")
	asAuditor(func() {
		policyResp, err := ctl.savedQueryServiceClient.GetSavedQueryPolicy(ctx, connect.NewRequest(&v1pb.GetSavedQueryPolicyRequest{
			Resource: savedQuery.Name,
		}))
		a.NoError(err)
		updated, err := ctl.savedQueryServiceClient.SetSavedQueryPolicy(ctx, connect.NewRequest(&v1pb.SetSavedQueryPolicyRequest{
			Resource: savedQuery.Name,
			Policy: &v1pb.SavedQueryPolicy{
				Etag: policyResp.Msg.Etag,
				Bindings: []*v1pb.SavedQueryBinding{{
					Level:   v1pb.SavedQueryBinding_VIEWER,
					Members: []string{fmt.Sprintf("user:%s", auditorEmail)},
				}},
			},
		}))
		a.NoError(err)
		a.Len(updated.Msg.Bindings, 1)
	})

	// delete on top: the saved query goes, stars and all.
	grantRole("bb.savedQueries.delete")
	asAuditor(func() {
		_, err := ctl.savedQueryServiceClient.DeleteSavedQuery(ctx, connect.NewRequest(&v1pb.DeleteSavedQueryRequest{
			Name: savedQuery.Name,
		}))
		a.NoError(err)
	})
}
