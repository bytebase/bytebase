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

	createWorksheet := func(title, parent string, visibility v1pb.Worksheet_Visibility) *v1pb.Worksheet {
		resp, err := ctl.worksheetServiceClient.CreateWorksheet(ctx, connect.NewRequest(&v1pb.CreateWorksheetRequest{
			Parent: parent,
			Worksheet: &v1pb.Worksheet{
				Title:      title,
				Content:    []byte("SELECT 1;"),
				Visibility: visibility,
			},
		}))
		a.NoError(err)
		return resp.Msg
	}

	projectPrivate := createWorksheet("project-private", ctl.project.Name, v1pb.Worksheet_PRIVATE)
	projectRead := createWorksheet("project-read", ctl.project.Name, v1pb.Worksheet_PROJECT_READ)
	crossProjectRead := createWorksheet("cross-project-read", otherProject.Name, v1pb.Worksheet_PROJECT_READ)

	loginResp, err := ctl.authServiceClient.Login(ctx, connect.NewRequest(&v1pb.LoginRequest{
		Email:    otherEmail,
		Password: otherPassword,
	}))
	a.NoError(err)
	ctl.authInterceptor.token = loginResp.Msg.Token
	otherWorksheet := createWorksheet("other-creator", ctl.project.Name, v1pb.Worksheet_PROJECT_WRITE)

	ctl.authInterceptor.token = ownerToken

	listProjectResp, err := ctl.worksheetServiceClient.ListWorksheets(ctx, connect.NewRequest(&v1pb.ListWorksheetsRequest{
		Parent: ctl.project.Name,
		Filter: fmt.Sprintf("creator == \"users/%s\"", otherEmail),
	}))
	a.NoError(err)
	a.ElementsMatch([]string{otherWorksheet.Name}, worksheetNames(listProjectResp.Msg.Worksheets))

	listCrossProjectResp, err := ctl.worksheetServiceClient.ListWorksheets(ctx, connect.NewRequest(&v1pb.ListWorksheetsRequest{
		Parent: "projects/-",
		Filter: `visibility == "PROJECT_READ"`,
	}))
	a.NoError(err)
	a.ElementsMatch([]string{projectRead.Name, crossProjectRead.Name}, worksheetNames(listCrossProjectResp.Msg.Worksheets))

	_, err = ctl.worksheetServiceClient.UpdateWorksheetOrganizer(ctx, connect.NewRequest(&v1pb.UpdateWorksheetOrganizerRequest{
		Organizer: &v1pb.WorksheetOrganizer{
			Worksheet: projectRead.Name,
			Starred:   true,
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"starred"}},
	}))
	a.NoError(err)
	listStarredResp, err := ctl.worksheetServiceClient.ListWorksheets(ctx, connect.NewRequest(&v1pb.ListWorksheetsRequest{
		Parent: ctl.project.Name,
		Filter: `starred == true`,
	}))
	a.NoError(err)
	a.ElementsMatch([]string{projectRead.Name}, worksheetNames(listStarredResp.Msg.Worksheets))

	a.NotEmpty(projectPrivate.Name)
}

func TestSearchWorksheets(t *testing.T) {
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
			Title:             "Search Worksheet Other Project",
			AllowSelfApproval: true,
		},
		ProjectId: generateRandomString("search-worksheet-project"),
	}))
	a.NoError(err)
	otherProject := otherProjectResp.Msg

	userEmail := fmt.Sprintf("search-worksheet-%s@example.com", generateRandomString("user"))
	userPassword := "1024bytebase"
	userResp, err := ctl.userServiceClient.CreateUser(ctx, connect.NewRequest(&v1pb.CreateUserRequest{
		User: &v1pb.User{
			Email:    userEmail,
			Password: userPassword,
			Title:    "Search Worksheet User",
		},
	}))
	a.NoError(err)
	_, err = ctl.addMemberToWorkspaceIAM(ctx, userResp.Msg.Workspace, fmt.Sprintf("user:%s", userEmail), "roles/workspaceMember")
	a.NoError(err)

	roleID := generateRandomString("worksheet-reader")
	_, err = ctl.roleServiceClient.CreateRole(ctx, connect.NewRequest(&v1pb.CreateRoleRequest{
		Role: &v1pb.Role{
			Title:       "Worksheet Reader",
			Permissions: []string{"bb.worksheets.get"},
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
		Members: []string{fmt.Sprintf("user:%s", userEmail)},
	})
	_, err = ctl.projectServiceClient.SetIamPolicy(ctx, connect.NewRequest(&v1pb.SetIamPolicyRequest{
		Resource: ctl.project.Name,
		Policy:   policy,
	}))
	a.NoError(err)

	createWorksheet := func(title, parent string, visibility v1pb.Worksheet_Visibility) *v1pb.Worksheet {
		resp, err := ctl.worksheetServiceClient.CreateWorksheet(ctx, connect.NewRequest(&v1pb.CreateWorksheetRequest{
			Parent: parent,
			Worksheet: &v1pb.Worksheet{
				Title:      title,
				Content:    []byte("SELECT 1;"),
				Visibility: visibility,
			},
		}))
		a.NoError(err)
		return resp.Msg
	}

	ownerPrivate := createWorksheet("owner-private", ctl.project.Name, v1pb.Worksheet_PRIVATE)
	ownerRead := createWorksheet("owner-read", ctl.project.Name, v1pb.Worksheet_PROJECT_READ)
	otherProjectRead := createWorksheet("other-project-read", otherProject.Name, v1pb.Worksheet_PROJECT_READ)

	loginResp, err := ctl.authServiceClient.Login(ctx, connect.NewRequest(&v1pb.LoginRequest{
		Email:    userEmail,
		Password: userPassword,
	}))
	a.NoError(err)
	ctl.authInterceptor.token = loginResp.Msg.Token
	userPrivate := createWorksheet("user-private", ctl.project.Name, v1pb.Worksheet_PRIVATE)

	searchResp, err := ctl.worksheetServiceClient.SearchWorksheets(ctx, connect.NewRequest(&v1pb.SearchWorksheetsRequest{
		Parent: ctl.project.Name,
	}))
	a.NoError(err)
	a.ElementsMatch([]string{userPrivate.Name}, worksheetNames(searchResp.Msg.Worksheets))

	_, err = ctl.worksheetServiceClient.SearchWorksheets(ctx, connect.NewRequest(&v1pb.SearchWorksheetsRequest{
		Parent: "projects/-",
	}))
	a.Error(err)
	a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))

	ctl.authInterceptor.token = ownerToken
	a.NotEmpty(ownerPrivate.Name)
	a.NotEmpty(ownerRead.Name)
	a.NotEmpty(otherProjectRead.Name)
}

func worksheetNames(worksheets []*v1pb.Worksheet) []string {
	names := make([]string, 0, len(worksheets))
	for _, worksheet := range worksheets {
		names = append(names, worksheet.Name)
	}
	return names
}
