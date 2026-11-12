package tests

import (
	"context"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// TestCreateWorkloadIdentityRefusesWhatItCannotKeep goes through the server's
// ACL, which refuses a missing project itself, but lets an archived project
// through and authorizes "projects/-" at workspace scope.
//
//nolint:tparallel // The last case archives the project the others create in.
func TestCreateWorkloadIdentityRefusesWhatItCannotKeep(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl, ctx := startWorkspace(ctx, t)

	create := func(parent, id string) error {
		_, err := ctl.workloadIdentityServiceClient.CreateWorkloadIdentity(ctx, connect.NewRequest(&v1pb.CreateWorkloadIdentityRequest{
			Parent:             parent,
			WorkloadIdentityId: id,
			WorkloadIdentity: &v1pb.WorkloadIdentity{Title: id, WorkloadIdentityConfig: &v1pb.WorkloadIdentityConfig{
				ProviderType:     v1pb.WorkloadIdentityConfig_GITHUB,
				IssuerUrl:        "https://token.actions.githubusercontent.com",
				AllowedAudiences: []string{"bytebase"},
				SubjectPattern:   "repo:acme-corp/deploy:*",
			}},
		}))
		return err
	}
	stored := func(email string) bool {
		_, err := ctl.workloadIdentityServiceClient.GetWorkloadIdentity(ctx, connect.NewRequest(&v1pb.GetWorkloadIdentityRequest{Name: "workloadIdentities/" + email}))
		if connect.CodeOf(err) == connect.CodeNotFound {
			return false
		}
		a.NoError(err)
		return true
	}
	projectID := strings.TrimPrefix(ctl.project.Name, "projects/")

	t.Run("a missing project is refused by the ACL", func(t *testing.T) {
		require.Equal(t, connect.CodeNotFound, connect.CodeOf(create("projects/no-such-project", "ci")))
	})
	t.Run("every project's placeholder is refused, not failed", func(t *testing.T) {
		err := create("projects/-", "ci")
		require.Error(t, err)
		require.NotContains(t, []connect.Code{connect.CodeInternal, connect.CodeUnknown}, connect.CodeOf(err), "%v", err)
	})
	t.Run("an ID with a slash is refused and nothing is stored", func(t *testing.T) {
		err := create(ctl.project.Name, "deploy/ci")
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "%v", err)
		listed, err := ctl.workloadIdentityServiceClient.ListWorkloadIdentities(ctx, connect.NewRequest(&v1pb.ListWorkloadIdentitiesRequest{Parent: ctl.project.Name, ShowDeleted: true}))
		require.NoError(t, err)
		require.Empty(t, listed.Msg.WorkloadIdentities)
	})
	t.Run("an archived project is refused and nothing is stored", func(t *testing.T) {
		_, err := ctl.projectServiceClient.DeleteProject(ctx, connect.NewRequest(&v1pb.DeleteProjectRequest{Name: ctl.project.Name}))
		require.NoError(t, err)
		err = create(ctl.project.Name, "ci")
		require.Equal(t, connect.CodeNotFound, connect.CodeOf(err), "%v", err)
		require.False(t, stored("ci@"+projectID+".workload.bytebase.com"))
	})
}
