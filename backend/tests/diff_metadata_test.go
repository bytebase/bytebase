package tests

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// TestDiffMetadata covers DatabaseService.DiffMetadata: the server reads the
// source schema from the store by database name, gated by
// bb.databases.diffMetadata — granted to schema-change-authoring roles
// (admin, DBA, project owner, project developer) but deliberately absent
// from viewer/releaser/SQL-editor roles.
func TestDiffMetadata(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	ownerToken := ctl.authInterceptor.token

	pgContainer, err := getPgContainer(ctx)
	defer func() {
		pgContainer.Close(ctx)
	}()
	a.NoError(err)

	instanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance"),
		Instance: &v1pb.Instance{
			Title:       "pgInstance",
			Engine:      v1pb.Engine_POSTGRES,
			Environment: new("environments/prod"),
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: pgContainer.host, Port: pgContainer.port, Username: "postgres", Password: "root-password", Id: "admin"}},
		},
	}))
	a.NoError(err)
	instance := instanceResp.Msg

	const databaseName = "diff_metadata_db"
	err = ctl.createDatabase(ctx, ctl.project, instance, nil, databaseName, "postgres")
	a.NoError(err)

	databaseResp, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Name, databaseName),
	}))
	a.NoError(err)
	database := databaseResp.Msg

	// Fetch the synced current metadata as the baseline for the target.
	metadataResp, err := ctl.databaseServiceClient.GetDatabaseMetadata(ctx, connect.NewRequest(&v1pb.GetDatabaseMetadataRequest{
		Name: database.Name + "/metadata",
	}))
	a.NoError(err)
	currentMetadata := metadataResp.Msg

	// 0. A no-op target (the fetched metadata unchanged) must produce an empty
	// diff. This pins the store→v1→store round-trip fidelity the reshape
	// depends on: the source no longer passes through the v1 conversion, so
	// any field the converter drops would surface here as spurious DDL.
	noopResp, err := ctl.databaseServiceClient.DiffMetadata(ctx, connect.NewRequest(&v1pb.DiffMetadataRequest{
		Name:           database.Name,
		TargetMetadata: currentMetadata,
	}))
	a.NoError(err)
	a.Empty(noopResp.Msg.Diff)

	// Target = current schema plus one new table in the public schema.
	targetMetadata, ok := proto.Clone(currentMetadata).(*v1pb.DatabaseMetadata)
	a.True(ok)
	var publicSchema *v1pb.SchemaMetadata
	for _, schema := range targetMetadata.Schemas {
		if schema.Name == "public" {
			publicSchema = schema
			break
		}
	}
	a.NotNil(publicSchema)
	publicSchema.Tables = append(publicSchema.Tables, &v1pb.TableMetadata{
		Name: "t_diff",
		Columns: []*v1pb.ColumnMetadata{
			{Name: "id", Type: "integer", Nullable: false},
		},
	})

	// 1. The owner diffs the database's current schema against the target.
	diffResp, err := ctl.databaseServiceClient.DiffMetadata(ctx, connect.NewRequest(&v1pb.DiffMetadataRequest{
		Name:           database.Name,
		TargetMetadata: targetMetadata,
	}))
	a.NoError(err)
	a.Contains(diffResp.Msg.Diff, "CREATE TABLE")
	a.Contains(diffResp.Msg.Diff, "t_diff")
	// The one-table addition must not drag along spurious drops or alters.
	a.NotContains(diffResp.Msg.Diff, "DROP")
	a.NotContains(diffResp.Msg.Diff, "ALTER")

	// 2. Target metadata is required.
	_, err = ctl.databaseServiceClient.DiffMetadata(ctx, connect.NewRequest(&v1pb.DiffMetadataRequest{
		Name: database.Name,
	}))
	a.Error(err)
	a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))

	// 3. bb.databases.diffMetadata gates the RPC: a plain workspace member
	// without a project role is denied.
	memberEmail := fmt.Sprintf("member-%s@example.com", generateRandomString("u"))
	memberPassword := "1024bytebase"
	memberUser, err := ctl.userServiceClient.CreateUser(ctx, connect.NewRequest(&v1pb.CreateUserRequest{
		User: &v1pb.User{
			Email:    memberEmail,
			Password: memberPassword,
			Title:    "Member User",
		},
	}))
	a.NoError(err)
	_, err = ctl.addMemberToWorkspaceIAM(ctx, memberUser.Msg.Workspace, fmt.Sprintf("user:%s", memberEmail), "roles/workspaceMember")
	a.NoError(err)

	loginResp, err := ctl.authServiceClient.Login(ctx, connect.NewRequest(&v1pb.LoginRequest{
		Email:    memberEmail,
		Password: memberPassword,
	}))
	a.NoError(err)
	ctl.authInterceptor.token = loginResp.Msg.Token

	_, err = ctl.databaseServiceClient.DiffMetadata(ctx, connect.NewRequest(&v1pb.DiffMetadataRequest{
		Name:           database.Name,
		TargetMetadata: targetMetadata,
	}))
	a.Error(err)
	a.Equal(connect.CodePermissionDenied, connect.CodeOf(err))

	// 4. Project viewer reads schemas but does not author changes: still
	// denied with the grant boundary chosen for this permission.
	ctl.authInterceptor.token = ownerToken
	policyResp, err := ctl.projectServiceClient.GetIamPolicy(ctx, connect.NewRequest(&v1pb.GetIamPolicyRequest{
		Resource: ctl.project.Name,
	}))
	a.NoError(err)
	policy := policyResp.Msg
	policy.Bindings = append(policy.Bindings, &v1pb.Binding{
		Role:    "roles/projectViewer",
		Members: []string{fmt.Sprintf("user:%s", memberEmail)},
	})
	_, err = ctl.projectServiceClient.SetIamPolicy(ctx, connect.NewRequest(&v1pb.SetIamPolicyRequest{
		Resource: ctl.project.Name,
		Policy:   policy,
	}))
	a.NoError(err)

	ctl.authInterceptor.token = loginResp.Msg.Token
	_, err = ctl.databaseServiceClient.DiffMetadata(ctx, connect.NewRequest(&v1pb.DiffMetadataRequest{
		Name:           database.Name,
		TargetMetadata: targetMetadata,
	}))
	a.Error(err)
	a.Equal(connect.CodePermissionDenied, connect.CodeOf(err))

	// 5. Project developer authors schema changes: the diff succeeds.
	ctl.authInterceptor.token = ownerToken
	policyResp, err = ctl.projectServiceClient.GetIamPolicy(ctx, connect.NewRequest(&v1pb.GetIamPolicyRequest{
		Resource: ctl.project.Name,
	}))
	a.NoError(err)
	policy = policyResp.Msg
	policy.Bindings = append(policy.Bindings, &v1pb.Binding{
		Role:    "roles/projectDeveloper",
		Members: []string{fmt.Sprintf("user:%s", memberEmail)},
	})
	_, err = ctl.projectServiceClient.SetIamPolicy(ctx, connect.NewRequest(&v1pb.SetIamPolicyRequest{
		Resource: ctl.project.Name,
		Policy:   policy,
	}))
	a.NoError(err)

	ctl.authInterceptor.token = loginResp.Msg.Token
	developerDiffResp, err := ctl.databaseServiceClient.DiffMetadata(ctx, connect.NewRequest(&v1pb.DiffMetadataRequest{
		Name:           database.Name,
		TargetMetadata: targetMetadata,
	}))
	a.NoError(err)
	a.Equal(diffResp.Msg.Diff, developerDiffResp.Msg.Diff)
}
