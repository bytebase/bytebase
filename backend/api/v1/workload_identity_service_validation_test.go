package v1

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/component/config"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// TestWorkloadIdentityConfigValidation pins that Create and Update reach the
// configuration validator, and how Update treats each update-mask shape. The
// rules themselves are pinned without a database in
// TestValidateWorkloadIdentityConfig.
//
//nolint:tparallel // Subtests share one seeded identity.
func TestWorkloadIdentityConfigValidation(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(context.Background(), common.WorkspaceIDContextKey, "default")
	db, stores, _ := testcontainer.NewMetadataDB(t)
	_, err := db.ExecContext(ctx, `INSERT INTO workspace (resource_id) VALUES ('default')`)
	require.NoError(t, err)
	service := &WorkloadIdentityService{store: stores, profile: &config.Profile{}}

	valid := func() *v1pb.WorkloadIdentityConfig {
		return &v1pb.WorkloadIdentityConfig{
			ProviderType:     v1pb.WorkloadIdentityConfig_GITHUB,
			IssuerUrl:        "https://token.actions.githubusercontent.com",
			AllowedAudiences: []string{"bytebase"},
			SubjectPattern:   "repo:acme-corp/deploy:ref:refs/heads/main",
		}
	}
	create := func(id string, config *v1pb.WorkloadIdentityConfig) error {
		_, err := service.CreateWorkloadIdentity(ctx, connect.NewRequest(&v1pb.CreateWorkloadIdentityRequest{
			Parent:             "workspaces/default",
			WorkloadIdentityId: id,
			WorkloadIdentity:   &v1pb.WorkloadIdentity{Title: id, WorkloadIdentityConfig: config},
		}))
		return err
	}

	// One row, not the rule table: what this test pins is that Create and
	// Update reach the validator at all.
	t.Run("create rejects an unbindable config", func(t *testing.T) {
		a := require.New(t)
		config := valid()
		config.AllowedAudiences = nil
		err := create("wi-no-audience", config)
		a.Error(err)
		a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))
	})
	t.Run("create accepts a bindable config", func(t *testing.T) {
		require.New(t).NoError(create("wi-valid", valid()))
	})
	t.Run("create rejects an ID that holds a slash", func(t *testing.T) {
		a := require.New(t)
		err := create("deploy/ci", valid())
		a.Error(err)
		a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))
	})
	t.Run("create rejects a missing config", func(t *testing.T) {
		a := require.New(t)
		err := create("wi-none", nil)
		a.Error(err)
		a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))
	})

	update := func(config *v1pb.WorkloadIdentityConfig, paths ...string) error {
		_, err := service.UpdateWorkloadIdentity(ctx, connect.NewRequest(&v1pb.UpdateWorkloadIdentityRequest{
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

// TestCreateWorkloadIdentityChecksItsProject covers the projects the ACL lets
// through: an archived one, and "projects/-", which it authorizes at workspace
// scope.
func TestCreateWorkloadIdentityChecksItsProject(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.WithValue(context.Background(), common.WorkspaceIDContextKey, "default")
	db, stores, _ := testcontainer.NewMetadataDB(t)
	_, err := db.ExecContext(ctx, `INSERT INTO workspace (resource_id) VALUES ('default')`)
	a.NoError(err)
	_, err = db.ExecContext(ctx, `INSERT INTO project (resource_id, workspace, name, deleted) VALUES
		('hr', 'default', 'HR', FALSE), ('archived', 'default', 'Archived', TRUE)`)
	a.NoError(err)
	service := &WorkloadIdentityService{store: stores, profile: &config.Profile{}}

	create := func(parent, id string) error {
		_, err := service.CreateWorkloadIdentity(ctx, connect.NewRequest(&v1pb.CreateWorkloadIdentityRequest{
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
	stored := func(t *testing.T, email string) bool {
		var n int
		require.NoError(t, db.QueryRowContext(ctx, `SELECT count(*) FROM workload_identity WHERE email = $1`, email).Scan(&n))
		return n > 0
	}

	a.NoError(create("projects/hr", "ci"))
	a.True(stored(t, "ci@hr.workload.bytebase.com"))

	for _, tc := range []struct {
		name, parent, email string
	}{
		{"an archived project", "projects/archived", "ci@archived.workload.bytebase.com"},
		{"every project's placeholder", "projects/-", "ci@-.workload.bytebase.com"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a := require.New(t)
			err := create(tc.parent, "ci")
			a.Error(err)
			a.Equal(connect.CodeNotFound, connect.CodeOf(err))
			a.False(stored(t, tc.email))
		})
	}
}
