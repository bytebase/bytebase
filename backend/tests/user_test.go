package tests

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func TestDeleteUser(t *testing.T) {
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	expectErrorMsg := "workspace must have at least one admin"

	memberResp, err := ctl.userServiceClient.CreateUser(ctx, connect.NewRequest(&v1pb.CreateUserRequest{
		User: &v1pb.User{
			Title:    "member",
			UserType: v1pb.UserType_USER,
			Email:    "member@bytebase.com",
			Password: "1024bytebase",
		},
	}))
	a.NoError(err)
	member := memberResp.Msg

	_, err = ctl.userServiceClient.DeleteUser(ctx, connect.NewRequest(&v1pb.DeleteUserRequest{
		Name: member.Name,
	}))
	a.NoError(err)

	// Test: cannot delete the last admin.
	_, err = ctl.userServiceClient.DeleteUser(ctx, connect.NewRequest(&v1pb.DeleteUserRequest{
		Name: ctl.principalName,
	}))
	a.Error(err)
	a.ErrorContains(err, expectErrorMsg)

	serviceAccountResp, err := ctl.userServiceClient.CreateUser(ctx, connect.NewRequest(&v1pb.CreateUserRequest{
		User: &v1pb.User{
			Title:    "bot",
			UserType: v1pb.UserType_SERVICE_ACCOUNT,
			Email:    "bot@service.bytebase.com",
		},
	}))
	a.NoError(err)
	serviceAccount := serviceAccountResp.Msg

	policyResp, err := ctl.workspaceServiceClient.GetIamPolicy(ctx, connect.NewRequest(&v1pb.GetIamPolicyRequest{}))
	a.NoError(err)
	policy := policyResp.Msg

	// Test: only count the end user.
	for _, binding := range policy.Bindings {
		if binding.Role == "roles/workspaceAdmin" {
			binding.Members = append(binding.Members, fmt.Sprintf("user:%s", serviceAccount.Email))
			break
		}
	}
	updatedPolicyResp, err := ctl.workspaceServiceClient.SetIamPolicy(ctx, connect.NewRequest(&v1pb.SetIamPolicyRequest{
		Etag:   policy.Etag,
		Policy: policy,
	}))
	a.NoError(err)
	updatedPolicy := updatedPolicyResp.Msg

	_, err = ctl.userServiceClient.DeleteUser(ctx, connect.NewRequest(&v1pb.DeleteUserRequest{
		Name: ctl.principalName,
	}))
	a.Error(err)
	a.ErrorContains(err, expectErrorMsg)

	_, err = ctl.userServiceClient.UndeleteUser(ctx, connect.NewRequest(&v1pb.UndeleteUserRequest{
		Name: member.Name,
	}))
	a.NoError(err)

	// Test: can delete the admin if member count > 1
	for _, binding := range updatedPolicy.Bindings {
		if binding.Role == "roles/workspaceAdmin" {
			binding.Members = append(binding.Members, fmt.Sprintf("user:%s", member.Email))
			break
		}
	}
	newPolicyResp, err := ctl.workspaceServiceClient.SetIamPolicy(ctx, connect.NewRequest(&v1pb.SetIamPolicyRequest{
		Etag:   updatedPolicy.Etag,
		Policy: updatedPolicy,
	}))
	a.NoError(err)
	newPolicy := newPolicyResp.Msg

	_, err = ctl.userServiceClient.DeleteUser(ctx, connect.NewRequest(&v1pb.DeleteUserRequest{
		Name: ctl.principalName,
	}))
	a.NoError(err)

	// Switch context.
	resp, err := ctl.authServiceClient.Login(ctx, connect.NewRequest(&v1pb.LoginRequest{
		Email:    member.Email,
		Password: "1024bytebase",
	}))
	a.NoError(err)
	ctl.authInterceptor.token = resp.Msg.Token

	// Test: check allUser in the binding
	for _, binding := range newPolicy.Bindings {
		if binding.Role == "roles/workspaceAdmin" {
			binding.Members = []string{common.AllUsers}
			break
		}
	}
	_, err = ctl.workspaceServiceClient.SetIamPolicy(ctx, connect.NewRequest(&v1pb.SetIamPolicyRequest{
		Etag:   newPolicy.Etag,
		Policy: newPolicy,
	}))
	a.NoError(err)

	_, err = ctl.userServiceClient.DeleteUser(ctx, connect.NewRequest(&v1pb.DeleteUserRequest{
		Name: member.Name,
	}))
	a.Error(err)
	a.ErrorContains(err, expectErrorMsg)

	_, err = ctl.userServiceClient.UndeleteUser(ctx, connect.NewRequest(&v1pb.UndeleteUserRequest{
		Name: ctl.principalName,
	}))
	a.NoError(err)

	_, err = ctl.userServiceClient.DeleteUser(ctx, connect.NewRequest(&v1pb.DeleteUserRequest{
		Name: member.Name,
	}))
	a.NoError(err)
}
