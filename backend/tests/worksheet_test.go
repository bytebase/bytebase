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

func TestListWorksheets(t *testing.T) {
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
			Title:             "Worksheet Other Project",
			AllowSelfApproval: true,
		},
		ProjectId: generateRandomString("worksheet-project"),
	}))
	a.NoError(err)
	otherProject := otherProjectResp.Msg

	otherEmail := fmt.Sprintf("worksheet-%s@example.com", generateRandomString("user"))
	otherPassword := "1024bytebase"
	otherUser, err := ctl.userServiceClient.CreateUser(ctx, connect.NewRequest(&v1pb.CreateUserRequest{
		User: &v1pb.User{
			Email:    otherEmail,
			Password: otherPassword,
			Title:    "Worksheet User",
		},
	}))
	a.NoError(err)
	_, err = ctl.addMemberToWorkspaceIAM(ctx, otherUser.Msg.Workspace, fmt.Sprintf("user:%s", otherEmail), "roles/workspaceMember")
	a.NoError(err)

	createWorksheet := func(title, parent string) *v1pb.Worksheet {
		resp, err := ctl.worksheetServiceClient.CreateWorksheet(ctx, connect.NewRequest(&v1pb.CreateWorksheetRequest{
			Parent: parent,
			Worksheet: &v1pb.Worksheet{
				Title:      title,
				Content:    []byte("SELECT 1;"),
				Visibility: v1pb.Worksheet_PROJECT_READ,
			},
		}))
		a.NoError(err)
		return resp.Msg
	}

	projectOwnerWorksheet := createWorksheet("project-owner", ctl.project.Name)
	crossProjectOwnerWorksheet := createWorksheet("cross-project-owner", otherProject.Name)

	loginResp, err := ctl.authServiceClient.Login(ctx, connect.NewRequest(&v1pb.LoginRequest{
		Email:    otherEmail,
		Password: otherPassword,
	}))
	a.NoError(err)
	ctl.authInterceptor.token = loginResp.Msg.Token
	otherWorksheet := createWorksheet("other-creator", ctl.project.Name)

	_, err = ctl.worksheetServiceClient.ListWorksheets(ctx, connect.NewRequest(&v1pb.ListWorksheetsRequest{
		Parent: "projects/-",
	}))
	a.Error(err)
	a.Equal(connect.CodePermissionDenied, connect.CodeOf(err))

	ctl.authInterceptor.token = ownerToken
	roleID := generateRandomString("worksheet-lister")
	_, err = ctl.roleServiceClient.CreateRole(ctx, connect.NewRequest(&v1pb.CreateRoleRequest{
		Role: &v1pb.Role{
			Title:       "Worksheet Lister",
			Permissions: []string{"bb.worksheets.list"},
		},
		RoleId: roleID,
	}))
	a.NoError(err)
	_, err = ctl.addMemberToWorkspaceIAM(ctx, otherUser.Msg.Workspace, fmt.Sprintf("user:%s", otherEmail), fmt.Sprintf("roles/%s", roleID))
	a.NoError(err)

	ctl.authInterceptor.token = loginResp.Msg.Token
	listProjectResp, err := ctl.worksheetServiceClient.ListWorksheets(ctx, connect.NewRequest(&v1pb.ListWorksheetsRequest{
		Parent: ctl.project.Name,
		Filter: fmt.Sprintf("creator == \"users/%s\"", otherEmail),
	}))
	a.NoError(err)
	a.ElementsMatch([]string{otherWorksheet.Name}, worksheetNames(listProjectResp.Msg.Worksheets))

	listNotCreatorResp, err := ctl.worksheetServiceClient.ListWorksheets(ctx, connect.NewRequest(&v1pb.ListWorksheetsRequest{
		Parent: ctl.project.Name,
		Filter: fmt.Sprintf("creator != \"users/%s\"", otherEmail),
	}))
	a.NoError(err)
	a.ElementsMatch([]string{projectOwnerWorksheet.Name}, worksheetNames(listNotCreatorResp.Msg.Worksheets))

	listCrossProjectResp, err := ctl.worksheetServiceClient.ListWorksheets(ctx, connect.NewRequest(&v1pb.ListWorksheetsRequest{
		Parent: "projects/-",
		Filter: `creator == "users/demo@example.com"`,
	}))
	a.NoError(err)
	a.ElementsMatch([]string{projectOwnerWorksheet.Name, crossProjectOwnerWorksheet.Name}, worksheetNames(listCrossProjectResp.Msg.Worksheets))

	firstPageResp, err := ctl.worksheetServiceClient.ListWorksheets(ctx, connect.NewRequest(&v1pb.ListWorksheetsRequest{
		Parent:   "projects/-",
		Filter:   `creator == "users/demo@example.com"`,
		PageSize: 1,
	}))
	a.NoError(err)
	a.Len(firstPageResp.Msg.Worksheets, 1)
	a.NotEmpty(firstPageResp.Msg.NextPageToken)

	secondPageResp, err := ctl.worksheetServiceClient.ListWorksheets(ctx, connect.NewRequest(&v1pb.ListWorksheetsRequest{
		Parent:    "projects/-",
		Filter:    `creator == "users/demo@example.com"`,
		PageSize:  1,
		PageToken: firstPageResp.Msg.NextPageToken,
	}))
	a.NoError(err)
	a.Len(secondPageResp.Msg.Worksheets, 1)
	a.Empty(secondPageResp.Msg.NextPageToken)
	a.ElementsMatch(
		[]string{projectOwnerWorksheet.Name, crossProjectOwnerWorksheet.Name},
		append(worksheetNames(firstPageResp.Msg.Worksheets), worksheetNames(secondPageResp.Msg.Worksheets)...),
	)

	_, err = ctl.worksheetServiceClient.ListWorksheets(ctx, connect.NewRequest(&v1pb.ListWorksheetsRequest{
		Parent: "projects/-",
		Filter: `visibility == "PROJECT_READ"`,
	}))
	a.Error(err)
	a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestSearchWorksheetsFilterByFolder(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	createWorksheet := func(title string) *v1pb.Worksheet {
		resp, err := ctl.worksheetServiceClient.CreateWorksheet(ctx, connect.NewRequest(&v1pb.CreateWorksheetRequest{
			Parent: ctl.project.Name,
			Worksheet: &v1pb.Worksheet{
				Title:      title,
				Content:    []byte("SELECT 1;"),
				Visibility: v1pb.Worksheet_PRIVATE,
			},
		}))
		a.NoError(err)
		return resp.Msg
	}
	setFolders := func(worksheet *v1pb.Worksheet, folders []string) {
		_, err := ctl.worksheetServiceClient.UpdateWorksheetOrganizer(ctx, connect.NewRequest(&v1pb.UpdateWorksheetOrganizerRequest{
			Organizer: &v1pb.WorksheetOrganizer{
				Worksheet: worksheet.Name,
				Folders:   folders,
			},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"folders"}},
		}))
		a.NoError(err)
	}

	alphaWorksheet := createWorksheet("alpha")
	setFolders(alphaWorksheet, []string{"alpha"})
	alphaChildWorksheet := createWorksheet("alpha-child")
	setFolders(alphaChildWorksheet, []string{"alpha", "beta"})
	alphabetWorksheet := createWorksheet("alphabet")
	setFolders(alphabetWorksheet, []string{"alphabet"})
	nestedAlphaWorksheet := createWorksheet("nested-alpha")
	setFolders(nestedAlphaWorksheet, []string{"gamma", "alpha"})
	rootWorksheet := createWorksheet("root")

	resp, err := ctl.worksheetServiceClient.SearchWorksheets(ctx, connect.NewRequest(&v1pb.SearchWorksheetsRequest{
		Parent: ctl.project.Name,
		Filter: `folder == "alpha"`,
	}))
	a.NoError(err)
	a.ElementsMatch(
		[]string{alphaWorksheet.Name},
		worksheetNames(resp.Msg.Worksheets),
	)
	a.NotContains(worksheetNames(resp.Msg.Worksheets), alphaChildWorksheet.Name)
	a.NotContains(worksheetNames(resp.Msg.Worksheets), alphabetWorksheet.Name)
	a.NotContains(worksheetNames(resp.Msg.Worksheets), nestedAlphaWorksheet.Name)
	a.NotContains(worksheetNames(resp.Msg.Worksheets), rootWorksheet.Name)

	childResp, err := ctl.worksheetServiceClient.SearchWorksheets(ctx, connect.NewRequest(&v1pb.SearchWorksheetsRequest{
		Parent: ctl.project.Name,
		Filter: `folder == "alpha/beta"`,
	}))
	a.NoError(err)
	a.ElementsMatch([]string{alphaChildWorksheet.Name}, worksheetNames(childResp.Msg.Worksheets))

	rootResp, err := ctl.worksheetServiceClient.SearchWorksheets(ctx, connect.NewRequest(&v1pb.SearchWorksheetsRequest{
		Parent: ctl.project.Name,
		Filter: `folder == ""`,
	}))
	a.NoError(err)
	a.ElementsMatch([]string{rootWorksheet.Name}, worksheetNames(rootResp.Msg.Worksheets))

	pageFirstWorksheet := createWorksheet("aaa-page")
	setFolders(pageFirstWorksheet, []string{"paging"})
	pageSecondWorksheet := createWorksheet("zzz-page")
	setFolders(pageSecondWorksheet, []string{"paging"})

	firstPageResp, err := ctl.worksheetServiceClient.SearchWorksheets(ctx, connect.NewRequest(&v1pb.SearchWorksheetsRequest{
		Parent:   ctl.project.Name,
		Filter:   `folder == "paging"`,
		PageSize: 1,
	}))
	a.NoError(err)
	a.Equal([]string{pageFirstWorksheet.Name}, worksheetNames(firstPageResp.Msg.Worksheets))
	a.NotEmpty(firstPageResp.Msg.NextPageToken)

	secondPageResp, err := ctl.worksheetServiceClient.SearchWorksheets(ctx, connect.NewRequest(&v1pb.SearchWorksheetsRequest{
		Parent:    ctl.project.Name,
		Filter:    `folder == "paging"`,
		PageSize:  1,
		PageToken: firstPageResp.Msg.NextPageToken,
	}))
	a.NoError(err)
	a.Equal([]string{pageSecondWorksheet.Name}, worksheetNames(secondPageResp.Msg.Worksheets))
}

func TestSearchWorksheetsFilterByTitle(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	createWorksheet := func(title string) *v1pb.Worksheet {
		resp, err := ctl.worksheetServiceClient.CreateWorksheet(ctx, connect.NewRequest(&v1pb.CreateWorksheetRequest{
			Parent: ctl.project.Name,
			Worksheet: &v1pb.Worksheet{
				Title:      title,
				Content:    []byte("SELECT 1;"),
				Visibility: v1pb.Worksheet_PRIVATE,
			},
		}))
		a.NoError(err)
		return resp.Msg
	}

	matchedWorksheet := createWorksheet("Production Payroll Report")
	unmatchedWorksheet := createWorksheet("Development Scratchpad")
	percentWorksheet := createWorksheet("Literal 100% match")
	underscoreWorksheet := createWorksheet("Literal under_score match")

	resp, err := ctl.worksheetServiceClient.SearchWorksheets(ctx, connect.NewRequest(&v1pb.SearchWorksheetsRequest{
		Parent: ctl.project.Name,
		Filter: `title.contains("payroll")`,
	}))
	a.NoError(err)
	a.Equal([]string{matchedWorksheet.Name}, worksheetNames(resp.Msg.Worksheets))
	a.NotContains(worksheetNames(resp.Msg.Worksheets), unmatchedWorksheet.Name)

	percentResp, err := ctl.worksheetServiceClient.SearchWorksheets(ctx, connect.NewRequest(&v1pb.SearchWorksheetsRequest{
		Parent: ctl.project.Name,
		Filter: `title.contains("%")`,
	}))
	a.NoError(err)
	a.Equal([]string{percentWorksheet.Name}, worksheetNames(percentResp.Msg.Worksheets))

	underscoreResp, err := ctl.worksheetServiceClient.SearchWorksheets(ctx, connect.NewRequest(&v1pb.SearchWorksheetsRequest{
		Parent: ctl.project.Name,
		Filter: `title.contains("_")`,
	}))
	a.NoError(err)
	a.Equal([]string{underscoreWorksheet.Name}, worksheetNames(underscoreResp.Msg.Worksheets))

	_, err = ctl.worksheetServiceClient.SearchWorksheets(ctx, connect.NewRequest(&v1pb.SearchWorksheetsRequest{
		Parent: ctl.project.Name,
		Filter: `content.contains("SELECT")`,
	}))
	a.Error(err)

	_, err = ctl.worksheetServiceClient.SearchWorksheets(ctx, connect.NewRequest(&v1pb.SearchWorksheetsRequest{
		Parent: ctl.project.Name,
		Filter: `"literal".contains("x")`,
	}))
	a.Error(err)

	_, err = ctl.worksheetServiceClient.SearchWorksheets(ctx, connect.NewRequest(&v1pb.SearchWorksheetsRequest{
		Parent: ctl.project.Name,
		Filter: `title.contains(name)`,
	}))
	a.Error(err)
}

func TestBatchUpdateWorksheetOrganizerFilterByFolder(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	createWorksheet := func(title string) *v1pb.Worksheet {
		resp, err := ctl.worksheetServiceClient.CreateWorksheet(ctx, connect.NewRequest(&v1pb.CreateWorksheetRequest{
			Parent: ctl.project.Name,
			Worksheet: &v1pb.Worksheet{
				Title:      title,
				Content:    []byte("SELECT 1;"),
				Visibility: v1pb.Worksheet_PRIVATE,
			},
		}))
		a.NoError(err)
		return resp.Msg
	}
	setFolders := func(worksheet *v1pb.Worksheet, folders []string) {
		_, err := ctl.worksheetServiceClient.UpdateWorksheetOrganizer(ctx, connect.NewRequest(&v1pb.UpdateWorksheetOrganizerRequest{
			Organizer: &v1pb.WorksheetOrganizer{
				Worksheet: worksheet.Name,
				Folders:   folders,
			},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"folders"}},
		}))
		a.NoError(err)
	}
	searchByFolder := func(folder string) []string {
		resp, err := ctl.worksheetServiceClient.SearchWorksheets(ctx, connect.NewRequest(&v1pb.SearchWorksheetsRequest{
			Parent: ctl.project.Name,
			Filter: fmt.Sprintf("folder == %q", folder),
		}))
		a.NoError(err)
		return worksheetNames(resp.Msg.Worksheets)
	}

	directWorksheet := createWorksheet("old-direct")
	setFolders(directWorksheet, []string{"old"})
	childWorksheet := createWorksheet("old-child")
	setFolders(childWorksheet, []string{"old", "child"})
	otherWorksheet := createWorksheet("other")
	setFolders(otherWorksheet, []string{"other"})

	directResp, err := ctl.worksheetServiceClient.BatchUpdateWorksheetOrganizer(ctx, connect.NewRequest(&v1pb.BatchUpdateWorksheetOrganizerRequest{
		Parent: ctl.project.Name,
		Filter: `folder == "old"`,
		Organizer: &v1pb.WorksheetOrganizer{
			Folders: []string{"new"},
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"folders"}},
	}))
	a.NoError(err)
	a.Equal(int32(1), directResp.Msg.UpdatedCount)

	childResp, err := ctl.worksheetServiceClient.BatchUpdateWorksheetOrganizer(ctx, connect.NewRequest(&v1pb.BatchUpdateWorksheetOrganizerRequest{
		Parent: ctl.project.Name,
		Filter: `folder == "old/child"`,
		Organizer: &v1pb.WorksheetOrganizer{
			Folders: []string{"new", "child"},
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"folders"}},
	}))
	a.NoError(err)
	a.Equal(int32(1), childResp.Msg.UpdatedCount)

	a.ElementsMatch([]string{directWorksheet.Name}, searchByFolder("new"))
	a.ElementsMatch([]string{childWorksheet.Name}, searchByFolder("new/child"))
	a.ElementsMatch([]string{otherWorksheet.Name}, searchByFolder("other"))
	a.Empty(searchByFolder("old"))
	a.Empty(searchByFolder("old/child"))

	starResp, err := ctl.worksheetServiceClient.BatchUpdateWorksheetOrganizer(ctx, connect.NewRequest(&v1pb.BatchUpdateWorksheetOrganizerRequest{
		Parent: ctl.project.Name,
		Filter: `folder == "new"`,
		Organizer: &v1pb.WorksheetOrganizer{
			Starred: true,
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"starred"}},
	}))
	a.NoError(err)
	a.Equal(int32(1), starResp.Msg.UpdatedCount)
	starredResp, err := ctl.worksheetServiceClient.SearchWorksheets(ctx, connect.NewRequest(&v1pb.SearchWorksheetsRequest{
		Parent: ctl.project.Name,
		Filter: `folder == "new" && starred == true`,
	}))
	a.NoError(err)
	a.ElementsMatch([]string{directWorksheet.Name}, worksheetNames(starredResp.Msg.Worksheets))

	nameResp, err := ctl.worksheetServiceClient.BatchUpdateWorksheetOrganizer(ctx, connect.NewRequest(&v1pb.BatchUpdateWorksheetOrganizerRequest{
		Parent: ctl.project.Name,
		Filter: fmt.Sprintf(`name in [%q,%q]`, directWorksheet.Name, childWorksheet.Name),
		Organizer: &v1pb.WorksheetOrganizer{
			Folders: []string{"selected"},
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"folders"}},
	}))
	a.NoError(err)
	a.Equal(int32(2), nameResp.Msg.UpdatedCount)
	a.ElementsMatch([]string{directWorksheet.Name, childWorksheet.Name}, searchByFolder("selected"))
	a.ElementsMatch([]string{otherWorksheet.Name}, searchByFolder("other"))
}

func TestListWorksheetFoldersReturnsCallerFolders(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	ownerToken := ctl.authInterceptor.token

	createWorksheet := func(title string, visibility v1pb.Worksheet_Visibility) *v1pb.Worksheet {
		resp, err := ctl.worksheetServiceClient.CreateWorksheet(ctx, connect.NewRequest(&v1pb.CreateWorksheetRequest{
			Parent: ctl.project.Name,
			Worksheet: &v1pb.Worksheet{
				Title:      title,
				Content:    []byte("SELECT 1;"),
				Visibility: visibility,
			},
		}))
		a.NoError(err)
		return resp.Msg
	}
	setFolders := func(worksheet *v1pb.Worksheet, folders []string) {
		_, err := ctl.worksheetServiceClient.UpdateWorksheetOrganizer(ctx, connect.NewRequest(&v1pb.UpdateWorksheetOrganizerRequest{
			Organizer: &v1pb.WorksheetOrganizer{
				Worksheet: worksheet.Name,
				Folders:   folders,
			},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"folders"}},
		}))
		a.NoError(err)
	}

	ownerWorksheet := createWorksheet("owner", v1pb.Worksheet_PRIVATE)
	setFolders(ownerWorksheet, []string{"owner", "child"})
	ownerSharedWorksheet := createWorksheet("owner-shared", v1pb.Worksheet_PROJECT_READ)
	setFolders(ownerSharedWorksheet, []string{"owner-shared"})

	otherEmail := fmt.Sprintf("worksheet-folder-%s@example.com", generateRandomString("user"))
	otherPassword := "1024bytebase"
	otherUser, err := ctl.userServiceClient.CreateUser(ctx, connect.NewRequest(&v1pb.CreateUserRequest{
		User: &v1pb.User{
			Email:    otherEmail,
			Password: otherPassword,
			Title:    "Worksheet Folder User",
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
	sharedWorksheet := createWorksheet("shared", v1pb.Worksheet_PROJECT_READ)
	setFolders(sharedWorksheet, []string{"other", "shared"})
	privateWorksheet := createWorksheet("private", v1pb.Worksheet_PRIVATE)

	ctl.authInterceptor.token = ownerToken
	setFolders(sharedWorksheet, []string{"owner", "child"})

	db, err := sql.Open("pgx", ctl.profile.PgURL)
	a.NoError(err)
	defer db.Close()
	_, privateWorksheetID, err := common.GetProjectIDWorksheetID(privateWorksheet.Name)
	a.NoError(err)
	_, err = db.ExecContext(ctx, `
		INSERT INTO worksheet_organizer (worksheet, principal, payload)
		SELECT resource_id, $1, '{"folders":["private"]}'::jsonb
		FROM worksheet
		WHERE resource_id = $2
	`, "demo@example.com", privateWorksheetID)
	a.NoError(err)

	resp, err := ctl.worksheetServiceClient.ListWorksheetFolders(ctx, connect.NewRequest(&v1pb.ListWorksheetFoldersRequest{
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
	}, worksheetFolderEntries(resp.Msg.Folders))
}

func TestSearchWorksheetsRejectsWildcardProject(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	_, err = ctl.worksheetServiceClient.SearchWorksheets(ctx, connect.NewRequest(&v1pb.SearchWorksheetsRequest{
		Parent: "projects/-",
	}))
	a.Error(err)
	a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))
}

func worksheetNames(worksheets []*v1pb.Worksheet) []string {
	names := make([]string, 0, len(worksheets))
	for _, worksheet := range worksheets {
		names = append(names, worksheet.Name)
	}
	return names
}

func worksheetFolderEntries(folders []*v1pb.WorksheetFolder) []string {
	entries := make([]string, 0, len(folders))
	for _, folder := range folders {
		entries = append(entries, fmt.Sprintf("%s:%v", folder.Category, folder.Folders))
	}
	return entries
}
