package tests

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func TestArchiveProject(t *testing.T) {
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	instanceRootDir := t.TempDir()
	instanceName := "testInstance1"
	instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, instanceName)
	a.NoError(err)

	// Add an instance.
	instanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance"),
		Instance: &v1pb.Instance{
			Title:       "test",
			Engine:      v1pb.Engine_SQLITE,
			Environment: stringPtr("environments/prod"),
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: instanceDir, Id: "admin"}},
		},
	}))
	a.NoError(err)
	instance := instanceResp.Msg

	t.Run("ArchiveProjectWithDatbase", func(_ *testing.T) {
		databaseName := "db1"
		err = ctl.createDatabase(ctx, ctl.project, instance, nil /* environment */, databaseName, "")
		a.NoError(err)

		_, err = ctl.projectServiceClient.DeleteProject(ctx, connect.NewRequest(&v1pb.DeleteProjectRequest{
			Name: ctl.project.Name,
		}))
		a.Error(err)
	})

	t.Run("ArchiveProjectWithOpenIssue", func(_ *testing.T) {
		planResp, err := ctl.planServiceClient.CreatePlan(ctx, connect.NewRequest(&v1pb.CreatePlanRequest{
			Parent: ctl.project.Name,
			Plan: &v1pb.Plan{
				Specs: []*v1pb.Plan_Spec{
					{
						Id: uuid.NewString(),
						Config: &v1pb.Plan_Spec_CreateDatabaseConfig{
							CreateDatabaseConfig: &v1pb.Plan_CreateDatabaseConfig{
								Target:   instance.Name,
								Database: "fakedb",
							},
						},
					},
				},
			},
		}))
		a.NoError(err)
		plan := planResp.Msg
		_, err = ctl.issueServiceClient.CreateIssue(ctx, connect.NewRequest(&v1pb.CreateIssueRequest{
			Parent: ctl.project.Name,
			Issue: &v1pb.Issue{
				Title:       "dummy issue",
				Description: "dummy issue",
				Type:        v1pb.Issue_DATABASE_CHANGE,
				Plan:        plan.Name,
			},
		}))
		a.NoError(err)

		_, err = ctl.projectServiceClient.DeleteProject(ctx, connect.NewRequest(&v1pb.DeleteProjectRequest{
			Name: ctl.project.Name,
		}))
		a.ErrorContains(err, "resolve all open issues before deleting the project")
	})
}
