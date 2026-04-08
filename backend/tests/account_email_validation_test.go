package tests

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func getCurrentWorkspace(ctx context.Context, t *testing.T, ctl *controller) string {
	t.Helper()

	userResp, err := ctl.userServiceClient.GetUser(ctx, connect.NewRequest(&v1pb.GetUserRequest{
		Name: ctl.principalName,
	}))
	require.NoError(t, err)

	return userResp.Msg.Workspace
}

func TestServiceAccountEmailValidation(t *testing.T) {
	t.Run("create rejects invalid generated email", func(t *testing.T) {
		a := require.New(t)
		ctx := context.Background()
		ctl := &controller{}
		ctx, err := ctl.StartServerWithExternalPg(ctx)
		a.NoError(err)
		defer ctl.Close(ctx)

		workspace := getCurrentWorkspace(ctx, t, ctl)
		_, err = ctl.serviceAccountServiceClient.CreateServiceAccount(ctx, connect.NewRequest(&v1pb.CreateServiceAccountRequest{
			Parent:           workspace,
			ServiceAccountId: "Bot",
			ServiceAccount: &v1pb.ServiceAccount{
				Title: "bot",
			},
		}))
		a.Error(err)
		a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))
	})

	t.Run("get rejects malformed resource email", func(t *testing.T) {
		a := require.New(t)
		ctx := context.Background()
		ctl := &controller{}
		ctx, err := ctl.StartServerWithExternalPg(ctx)
		a.NoError(err)
		defer ctl.Close(ctx)

		_, err = ctl.serviceAccountServiceClient.GetServiceAccount(ctx, connect.NewRequest(&v1pb.GetServiceAccountRequest{
			Name: "serviceAccounts/Bad@service.bytebase.com",
		}))
		a.Error(err)
		a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))
	})
}

func TestWorkloadIdentityEmailValidation(t *testing.T) {
	t.Run("create rejects invalid generated email", func(t *testing.T) {
		a := require.New(t)
		ctx := context.Background()
		ctl := &controller{}
		ctx, err := ctl.StartServerWithExternalPg(ctx)
		a.NoError(err)
		defer ctl.Close(ctx)

		workspace := getCurrentWorkspace(ctx, t, ctl)
		_, err = ctl.workloadIdentityServiceClient.CreateWorkloadIdentity(ctx, connect.NewRequest(&v1pb.CreateWorkloadIdentityRequest{
			Parent:             workspace,
			WorkloadIdentityId: "Bot",
			WorkloadIdentity: &v1pb.WorkloadIdentity{
				Title: "bot",
			},
		}))
		a.Error(err)
		a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))
	})

	t.Run("get rejects malformed resource email", func(t *testing.T) {
		a := require.New(t)
		ctx := context.Background()
		ctl := &controller{}
		ctx, err := ctl.StartServerWithExternalPg(ctx)
		a.NoError(err)
		defer ctl.Close(ctx)

		_, err = ctl.workloadIdentityServiceClient.GetWorkloadIdentity(ctx, connect.NewRequest(&v1pb.GetWorkloadIdentityRequest{
			Name: "workloadIdentities/Bad@workload.bytebase.com",
		}))
		a.Error(err)
		a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))
	})
}
