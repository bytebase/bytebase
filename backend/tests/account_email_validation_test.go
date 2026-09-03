package tests

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

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

// TestWorkloadIdentityConfigValidation pins the write paths BYT-10151 made
// strict. The token-exchange regression in backend/plugin/idp/wif covers the
// read path; nothing else covers these.
func TestWorkloadIdentityConfigValidation(t *testing.T) {
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	workspace := getCurrentWorkspace(ctx, t, ctl)
	valid := func() *v1pb.WorkloadIdentityConfig {
		return &v1pb.WorkloadIdentityConfig{
			ProviderType:     v1pb.WorkloadIdentityConfig_GITHUB,
			IssuerUrl:        "https://token.actions.githubusercontent.com",
			AllowedAudiences: []string{"bytebase"},
			SubjectPattern:   "repo:acme-corp/deploy:ref:refs/heads/main",
		}
	}

	create := func(id string, config *v1pb.WorkloadIdentityConfig) error {
		_, err := ctl.workloadIdentityServiceClient.CreateWorkloadIdentity(ctx, connect.NewRequest(&v1pb.CreateWorkloadIdentityRequest{
			Parent:             workspace,
			WorkloadIdentityId: id,
			WorkloadIdentity:   &v1pb.WorkloadIdentity{Title: id, WorkloadIdentityConfig: config},
		}))
		return err
	}

	rejected := map[string]func(*v1pb.WorkloadIdentityConfig){
		"no-audience":      func(c *v1pb.WorkloadIdentityConfig) { c.AllowedAudiences = nil },
		"blank-audience":   func(c *v1pb.WorkloadIdentityConfig) { c.AllowedAudiences = []string{" "} },
		"no-subject":       func(c *v1pb.WorkloadIdentityConfig) { c.SubjectPattern = "" },
		"wildcard-subject": func(c *v1pb.WorkloadIdentityConfig) { c.SubjectPattern = "*" },
		"no-issuer":        func(c *v1pb.WorkloadIdentityConfig) { c.IssuerUrl = "" },
	}
	for name, mutate := range rejected {
		t.Run("create rejects "+name, func(t *testing.T) {
			a := require.New(t)
			config := valid()
			mutate(config)
			err := create("wi-"+name, config)
			a.Error(err)
			a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))
		})
	}
	t.Run("create accepts a bindable config", func(t *testing.T) {
		require.New(t).NoError(create("wi-valid", valid()))
	})
	t.Run("create rejects a missing config", func(t *testing.T) {
		a := require.New(t)
		err := create("wi-none", nil)
		a.Error(err)
		a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))
	})

	update := func(config *v1pb.WorkloadIdentityConfig, paths ...string) error {
		_, err := ctl.workloadIdentityServiceClient.UpdateWorkloadIdentity(ctx, connect.NewRequest(&v1pb.UpdateWorkloadIdentityRequest{
			WorkloadIdentity: &v1pb.WorkloadIdentity{
				Name:                   "workloadIdentities/wi-valid@workload.bytebase.com",
				Title:                  "renamed",
				WorkloadIdentityConfig: config,
			},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: paths},
		}))
		return err
	}
	t.Run("update rejects an unbindable config", func(t *testing.T) {
		a := require.New(t)
		config := valid()
		config.AllowedAudiences = nil
		err := update(config, "workload_identity_config")
		a.Error(err)
		a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))
	})
	t.Run("update rejects a config cleared through the mask", func(t *testing.T) {
		a := require.New(t)
		err := update(nil, "workload_identity_config")
		a.Error(err)
		a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))
	})
	t.Run("update accepts a title-only mask over a bindable config", func(t *testing.T) {
		require.New(t).NoError(update(nil, "title"))
	})
}
