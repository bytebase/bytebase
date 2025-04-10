package tests

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"

	"github.com/bytebase/bytebase/backend/base"

	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func TestDeleteUser(t *testing.T) {
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	expectErrorMsg := "rpc error: code = InvalidArgument desc = workspace must have at least one admin"

	member, err := ctl.userServiceClient.CreateUser(ctx, &v1pb.CreateUserRequest{
		User: &v1pb.User{
			Title:    "member",
			UserType: v1pb.UserType_USER,
			Email:    "member@bytebase.com",
			Password: "1024bytebase",
		},
	})
	a.NoError(err)

	_, err = ctl.userServiceClient.DeleteUser(ctx, &v1pb.DeleteUserRequest{
		Name: member.Name,
	})
	a.NoError(err)

	// Test: cannot delete the last admin.
	_, err = ctl.userServiceClient.DeleteUser(ctx, &v1pb.DeleteUserRequest{
		Name: ctl.principalName,
	})
	a.Error(err)
	a.EqualError(err, expectErrorMsg)

	serviceAccount, err := ctl.userServiceClient.CreateUser(ctx, &v1pb.CreateUserRequest{
		User: &v1pb.User{
			Title:    "bot",
			UserType: v1pb.UserType_SERVICE_ACCOUNT,
			Email:    "bot@service.bytebase.com",
		},
	})
	a.NoError(err)

	policy, err := ctl.workspaceServiceClient.GetIamPolicy(ctx, &v1pb.GetIamPolicyRequest{})
	a.NoError(err)

	// Test: only count the end user.
	for _, binding := range policy.Bindings {
		if binding.Role == "roles/workspaceAdmin" {
			binding.Members = append(binding.Members, fmt.Sprintf("user:%s", serviceAccount.Email))
			break
		}
	}
	updatedPolicy, err := ctl.workspaceServiceClient.SetIamPolicy(ctx, &v1pb.SetIamPolicyRequest{
		Etag:   policy.Etag,
		Policy: policy,
	})
	a.NoError(err)

	_, err = ctl.userServiceClient.DeleteUser(ctx, &v1pb.DeleteUserRequest{
		Name: ctl.principalName,
	})
	a.Error(err)
	a.EqualError(err, expectErrorMsg)

	_, err = ctl.userServiceClient.UndeleteUser(ctx, &v1pb.UndeleteUserRequest{
		Name: member.Name,
	})
	a.NoError(err)

	// Test: can delete the admin if member count > 1
	for _, binding := range updatedPolicy.Bindings {
		if binding.Role == "roles/workspaceAdmin" {
			binding.Members = append(binding.Members, fmt.Sprintf("user:%s", member.Email))
			break
		}
	}
	newPolicy, err := ctl.workspaceServiceClient.SetIamPolicy(ctx, &v1pb.SetIamPolicyRequest{
		Etag:   updatedPolicy.Etag,
		Policy: updatedPolicy,
	})
	a.NoError(err)

	_, err = ctl.userServiceClient.DeleteUser(ctx, &v1pb.DeleteUserRequest{
		Name: ctl.principalName,
	})
	a.NoError(err)

	// Switch context.
	resp, err := ctl.authServiceClient.Login(ctx, &v1pb.LoginRequest{
		Email:    member.Email,
		Password: "1024bytebase",
	})
	a.NoError(err)
	ctl.cookie = fmt.Sprintf("access-token=%s", resp.Token)
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs(
		"Authorization",
		fmt.Sprintf("Bearer %s", resp.Token),
	))

	// Test: check allUser in the binding
	for _, binding := range newPolicy.Bindings {
		if binding.Role == "roles/workspaceAdmin" {
			binding.Members = []string{base.AllUsers}
			break
		}
	}
	_, err = ctl.workspaceServiceClient.SetIamPolicy(ctx, &v1pb.SetIamPolicyRequest{
		Etag:   newPolicy.Etag,
		Policy: newPolicy,
	})
	a.NoError(err)

	_, err = ctl.userServiceClient.DeleteUser(ctx, &v1pb.DeleteUserRequest{
		Name: member.Name,
	})
	a.Error(err)
	a.EqualError(err, expectErrorMsg)

	_, err = ctl.userServiceClient.UndeleteUser(ctx, &v1pb.UndeleteUserRequest{
		Name: ctl.principalName,
	})
	a.NoError(err)

	_, err = ctl.userServiceClient.DeleteUser(ctx, &v1pb.DeleteUserRequest{
		Name: member.Name,
	})
	a.NoError(err)
}
