package tests

import (
	"context"
	"database/sql"
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
			Permissions: []string{"bb.worksheets.list"},
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
	setFolders := func(savedQuery *v1pb.SavedQuery, folders []string) {
		_, err := ctl.savedQueryServiceClient.UpdateSavedQueryOrganizer(ctx, connect.NewRequest(&v1pb.UpdateSavedQueryOrganizerRequest{
			Organizer: &v1pb.SavedQueryOrganizer{
				SavedQuery: savedQuery.Name,
				Folders:    folders,
			},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"folders"}},
		}))
		a.NoError(err)
	}

	alphaSavedQuery := createSavedQuery("alpha")
	setFolders(alphaSavedQuery, []string{"alpha"})
	alphaChildSavedQuery := createSavedQuery("alpha-child")
	setFolders(alphaChildSavedQuery, []string{"alpha", "beta"})
	alphabetSavedQuery := createSavedQuery("alphabet")
	setFolders(alphabetSavedQuery, []string{"alphabet"})
	nestedAlphaSavedQuery := createSavedQuery("nested-alpha")
	setFolders(nestedAlphaSavedQuery, []string{"gamma", "alpha"})
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
	setFolders(pageFirstSavedQuery, []string{"paging"})
	pageSecondSavedQuery := createSavedQuery("zzz-page")
	setFolders(pageSecondSavedQuery, []string{"paging"})

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

func TestBatchUpdateSavedQueryOrganizerFilterByFolder(t *testing.T) {
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
	setFolders := func(savedQuery *v1pb.SavedQuery, folders []string) {
		_, err := ctl.savedQueryServiceClient.UpdateSavedQueryOrganizer(ctx, connect.NewRequest(&v1pb.UpdateSavedQueryOrganizerRequest{
			Organizer: &v1pb.SavedQueryOrganizer{
				SavedQuery: savedQuery.Name,
				Folders:    folders,
			},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"folders"}},
		}))
		a.NoError(err)
	}
	searchByFolder := func(folder string) []string {
		resp, err := ctl.savedQueryServiceClient.SearchSavedQueries(ctx, connect.NewRequest(&v1pb.SearchSavedQueriesRequest{
			Parent: ctl.project.Name,
			Filter: fmt.Sprintf("folder == %q", folder),
		}))
		a.NoError(err)
		return savedQueryNames(resp.Msg.SavedQueries)
	}

	directSavedQuery := createSavedQuery("old-direct")
	setFolders(directSavedQuery, []string{"old"})
	childSavedQuery := createSavedQuery("old-child")
	setFolders(childSavedQuery, []string{"old", "child"})
	otherSavedQuery := createSavedQuery("other")
	setFolders(otherSavedQuery, []string{"other"})

	directResp, err := ctl.savedQueryServiceClient.BatchUpdateSavedQueryOrganizer(ctx, connect.NewRequest(&v1pb.BatchUpdateSavedQueryOrganizerRequest{
		Parent: ctl.project.Name,
		Filter: `folder == "old"`,
		Organizer: &v1pb.SavedQueryOrganizer{
			Folders: []string{"new"},
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"folders"}},
	}))
	a.NoError(err)
	a.Equal(int32(1), directResp.Msg.UpdatedCount)

	childResp, err := ctl.savedQueryServiceClient.BatchUpdateSavedQueryOrganizer(ctx, connect.NewRequest(&v1pb.BatchUpdateSavedQueryOrganizerRequest{
		Parent: ctl.project.Name,
		Filter: `folder == "old/child"`,
		Organizer: &v1pb.SavedQueryOrganizer{
			Folders: []string{"new", "child"},
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"folders"}},
	}))
	a.NoError(err)
	a.Equal(int32(1), childResp.Msg.UpdatedCount)

	a.ElementsMatch([]string{directSavedQuery.Name}, searchByFolder("new"))
	a.ElementsMatch([]string{childSavedQuery.Name}, searchByFolder("new/child"))
	a.ElementsMatch([]string{otherSavedQuery.Name}, searchByFolder("other"))
	a.Empty(searchByFolder("old"))
	a.Empty(searchByFolder("old/child"))

	starResp, err := ctl.savedQueryServiceClient.BatchUpdateSavedQueryOrganizer(ctx, connect.NewRequest(&v1pb.BatchUpdateSavedQueryOrganizerRequest{
		Parent: ctl.project.Name,
		Filter: `folder == "new"`,
		Organizer: &v1pb.SavedQueryOrganizer{
			Starred: true,
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"starred"}},
	}))
	a.NoError(err)
	a.Equal(int32(1), starResp.Msg.UpdatedCount)
	starredResp, err := ctl.savedQueryServiceClient.SearchSavedQueries(ctx, connect.NewRequest(&v1pb.SearchSavedQueriesRequest{
		Parent: ctl.project.Name,
		Filter: `folder == "new" && starred == true`,
	}))
	a.NoError(err)
	a.ElementsMatch([]string{directSavedQuery.Name}, savedQueryNames(starredResp.Msg.SavedQueries))

	nameResp, err := ctl.savedQueryServiceClient.BatchUpdateSavedQueryOrganizer(ctx, connect.NewRequest(&v1pb.BatchUpdateSavedQueryOrganizerRequest{
		Parent: ctl.project.Name,
		Filter: fmt.Sprintf(`name in [%q,%q]`, directSavedQuery.Name, childSavedQuery.Name),
		Organizer: &v1pb.SavedQueryOrganizer{
			Folders: []string{"selected"},
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"folders"}},
	}))
	a.NoError(err)
	a.Equal(int32(2), nameResp.Msg.UpdatedCount)
	a.ElementsMatch([]string{directSavedQuery.Name, childSavedQuery.Name}, searchByFolder("selected"))
	a.ElementsMatch([]string{otherSavedQuery.Name}, searchByFolder("other"))
}

func TestListSavedQueryFoldersReturnsCallerFolders(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	ownerToken := ctl.authInterceptor.token

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
	setFolders := func(savedQuery *v1pb.SavedQuery, folders []string) {
		_, err := ctl.savedQueryServiceClient.UpdateSavedQueryOrganizer(ctx, connect.NewRequest(&v1pb.UpdateSavedQueryOrganizerRequest{
			Organizer: &v1pb.SavedQueryOrganizer{
				SavedQuery: savedQuery.Name,
				Folders:    folders,
			},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"folders"}},
		}))
		a.NoError(err)
	}

	ownerSavedQuery := createSavedQuery("owner")
	setFolders(ownerSavedQuery, []string{"owner", "child"})
	ownerSharedSavedQuery := createSavedQuery("owner-shared")
	setFolders(ownerSharedSavedQuery, []string{"owner-shared"})

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
	sharedSavedQuery := createSavedQuery("shared")
	setFolders(sharedSavedQuery, []string{"other", "shared"})
	privateSavedQuery := createSavedQuery("private")

	ctl.authInterceptor.token = ownerToken
	setFolders(sharedSavedQuery, []string{"owner", "child"})

	db, err := sql.Open("pgx", ctl.profile.PgURL)
	a.NoError(err)
	defer db.Close()
	_, privateSavedQueryID, err := common.GetProjectIDSavedQueryID(privateSavedQuery.Name)
	a.NoError(err)
	_, err = db.ExecContext(ctx, `
		INSERT INTO saved_query_organizer (saved_query, principal, payload)
		SELECT resource_id, $1, '{"folders":["private"]}'::jsonb
		FROM saved_query
		WHERE resource_id = $2
	`, "demo@example.com", privateSavedQueryID)
	a.NoError(err)

	resp, err := ctl.savedQueryServiceClient.ListSavedQueryFolders(ctx, connect.NewRequest(&v1pb.ListSavedQueryFoldersRequest{
		Parent: ctl.project.Name,
	}))
	a.NoError(err)
	a.ElementsMatch([]string{
		"MINE:[owner]",
		"MINE:[owner child]",
		"MINE:[owner-shared]",
		"SHARED:[owner]",
		"SHARED:[owner child]",
		"SHARED:[private]",
	}, savedQueryFolderEntries(resp.Msg.Folders))
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

func savedQueryFolderEntries(folders []*v1pb.SavedQueryFolder) []string {
	entries := make([]string, 0, len(folders))
	for _, folder := range folders {
		entries = append(entries, fmt.Sprintf("%s:%v", folder.Category, folder.Folders))
	}
	return entries
}
