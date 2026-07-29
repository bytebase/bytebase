package tests

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

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

func worksheetNames(worksheets []*v1pb.Worksheet) []string {
	names := make([]string, 0, len(worksheets))
	for _, worksheet := range worksheets {
		names = append(names, worksheet.Name)
	}
	return names
}
