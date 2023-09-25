package tests

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/tests/fake"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func TestArchiveProject(t *testing.T) {
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	dataDir := t.TempDir()
	ctx, err := ctl.StartServerWithExternalPg(ctx, &config{
		dataDir:            dataDir,
		vcsProviderCreator: fake.NewGitLab,
	})
	a.NoError(err)
	defer ctl.Close(ctx)

	instanceRootDir := t.TempDir()
	instanceName := "testInstance1"
	instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, instanceName)
	a.NoError(err)

	// Add an instance.
	instance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance", 10),
		Instance: &v1pb.Instance{
			Title:       "test",
			Engine:      v1pb.Engine_SQLITE,
			Environment: "environments/prod",
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: instanceDir}},
		},
	})
	a.NoError(err)

	t.Run("ArchiveProjectWithDatbase", func(t *testing.T) {
		databaseName := "db1"
		err = ctl.createDatabaseV2(ctx, ctl.project, instance, nil /* environment */, databaseName, "", nil)
		a.NoError(err)

		_, err = ctl.projectServiceClient.DeleteProject(ctx, &v1pb.DeleteProjectRequest{
			Name: ctl.project.Name,
		})
		a.Error(err)
	})

	t.Run("ArchiveProjectWithOpenIssue", func(t *testing.T) {
		plan, err := ctl.rolloutServiceClient.CreatePlan(ctx, &v1pb.CreatePlanRequest{
			Parent: ctl.project.Name,
			Plan: &v1pb.Plan{
				Steps: []*v1pb.Plan_Step{
					{
						Specs: []*v1pb.Plan_Spec{
							{
								Config: &v1pb.Plan_Spec_CreateDatabaseConfig{
									CreateDatabaseConfig: &v1pb.Plan_CreateDatabaseConfig{
										Target:   instance.Name,
										Database: "fakedb",
									},
								},
							},
						},
					},
				},
			},
		})
		a.NoError(err)
		_, err = ctl.issueServiceClient.CreateIssue(ctx, &v1pb.CreateIssueRequest{
			Parent: ctl.project.Name,
			Issue: &v1pb.Issue{
				Title:       "dummy issue",
				Description: "dummy issue",
				Type:        v1pb.Issue_DATABASE_CHANGE,
				Assignee:    fmt.Sprintf("users/%s", api.SystemBotEmail),
				Plan:        plan.Name,
			},
		})
		a.NoError(err)

		_, err = ctl.projectServiceClient.DeleteProject(ctx, &v1pb.DeleteProjectRequest{
			Name: ctl.project.Name,
		})
		a.ErrorContains(err, "resolve all open issues before deleting the project")
	})
}
