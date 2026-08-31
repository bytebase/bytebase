package tests

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// TestUpdateGroupAllowMissingRequiresCreatePermission pins the create check on
// UpdateGroup's allow_missing path. Neither the ACL interceptor (CUSTOM auth) nor
// CreateGroup (called in-process) covers it, so for a while a caller with no group
// permission could create one — and name itself OWNER, which checkPermission honors.
func TestUpdateGroupAllowMissingRequiresCreatePermission(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	adminToken := ctl.authInterceptor.token
	const password = "1024bytebase"

	login := func(email string) string {
		resp, err := ctl.authServiceClient.Login(ctx, connect.NewRequest(&v1pb.LoginRequest{
			Email:    email,
			Password: password,
		}))
		a.NoError(err)
		return resp.Msg.Token
	}
	createUserWithRole := func(prefix, role string) string {
		email := fmt.Sprintf("%s-%s@example.com", prefix, generateRandomString("u"))
		created, err := ctl.userServiceClient.CreateUser(ctx, connect.NewRequest(&v1pb.CreateUserRequest{
			User: &v1pb.User{Email: email, Password: password, Title: prefix},
		}))
		a.NoError(err)
		_, err = ctl.addMemberToWorkspaceIAM(ctx, created.Msg.Workspace, "user:"+email, role)
		a.NoError(err)
		return email
	}

	// Workspace DBA holds neither bb.groups.create nor bb.groups.update.
	dbaEmail := createUserWithRole("dba", "roles/workspaceDBA")
	dbaToken := login(dbaEmail)

	// Fixture check: if this ever passes, the role gained bb.groups.create and the
	// denial below would prove nothing.
	ctl.authInterceptor.token = dbaToken
	_, err = ctl.groupServiceClient.CreateGroup(ctx, connect.NewRequest(&v1pb.CreateGroupRequest{
		Group:      &v1pb.Group{Title: "Front door"},
		GroupEmail: fmt.Sprintf("front-%s@example.com", generateRandomString("g")),
	}))
	a.Error(err)
	a.Equal(connect.CodePermissionDenied, connect.CodeOf(err), "workspace DBA must not hold bb.groups.create")

	// The regression: no back door to creation either.
	deniedGroupEmail := fmt.Sprintf("denied-%s@example.com", generateRandomString("g"))
	deniedGroupName := "groups/" + deniedGroupEmail
	_, err = ctl.groupServiceClient.UpdateGroup(ctx, connect.NewRequest(&v1pb.UpdateGroupRequest{
		Group:        &v1pb.Group{Name: deniedGroupName, Title: "Back door"},
		AllowMissing: true,
	}))
	a.Error(err)
	a.Equal(connect.CodePermissionDenied, connect.CodeOf(err), "UpdateGroup(allow_missing) must require bb.groups.create")

	// And nothing was written on the way to the denial.
	ctl.authInterceptor.token = adminToken
	_, err = ctl.groupServiceClient.GetGroup(ctx, connect.NewRequest(&v1pb.GetGroupRequest{Name: deniedGroupName}))
	a.Error(err)
	a.Equal(connect.CodeNotFound, connect.CodeOf(err), "the denied UpdateGroup must not have created the group")

	// Positive control: Workspace Member holds bb.groups.create, so allow_missing
	// still creates. Also proves the denial above is the permission, not the
	// USER_GROUPS license guard, which runs first and would deny both alike.
	memberEmail := createUserWithRole("member", "roles/workspaceMember")
	ctl.authInterceptor.token = login(memberEmail)

	allowedGroupEmail := fmt.Sprintf("allowed-%s@example.com", generateRandomString("g"))
	created, err := ctl.groupServiceClient.UpdateGroup(ctx, connect.NewRequest(&v1pb.UpdateGroupRequest{
		Group:        &v1pb.Group{Name: "groups/" + allowedGroupEmail, Title: "Allowed"},
		AllowMissing: true,
	}))
	a.NoError(err)
	a.Equal(allowedGroupEmail, created.Msg.Email)
}
