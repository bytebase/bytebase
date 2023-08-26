package tests

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"

	"github.com/bytebase/bytebase/backend/common/log"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/tests/fake"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func TestArchiveProject(t *testing.T) {
	a := require.New(t)
	log.SetLevel(zapcore.DebugLevel)
	ctx := context.Background()
	ctl := &controller{}
	dataDir := t.TempDir()
	ctx, err := ctl.StartServerWithExternalPg(ctx, &config{
		dataDir:                   dataDir,
		vcsProviderCreator:        fake.NewGitLab,
		developmentUseV2Scheduler: true,
	})
	a.NoError(err)
	defer ctl.Close(ctx)

	instanceRootDir := t.TempDir()
	instanceName := "testInstance1"
	instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, instanceName)
	a.NoError(err)

	prodEnvironment, err := ctl.getEnvironment(ctx, "prod")
	a.NoError(err)

	// Add an instance.
	instance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance", 10),
		Instance: &v1pb.Instance{
			Title:       "test",
			Engine:      v1pb.Engine_SQLITE,
			Environment: prodEnvironment.Name,
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: instanceDir}},
		},
	})
	a.NoError(err)

	t.Run("ArchiveProjectWithDatbase", func(t *testing.T) {
		project, err := ctl.createProject(ctx)
		a.NoError(err)

		databaseName := "db1"
		err = ctl.createDatabaseV2(ctx, project, instance, databaseName, "", nil)
		a.NoError(err)

		_, err = ctl.projectServiceClient.DeleteProject(ctx, &v1pb.DeleteProjectRequest{
			Name: project.Name,
		})
		a.Error(err)
	})

	t.Run("ArchiveProjectWithOpenIssue", func(t *testing.T) {
		project, err := ctl.createProject(ctx)
		a.NoError(err)

		plan, err := ctl.rolloutServiceClient.CreatePlan(ctx, &v1pb.CreatePlanRequest{
			Parent: project.Name,
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
			Parent: project.Name,
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
			Name: project.Name,
		})
		a.ErrorContains(err, "resolve all open issues before deleting the project")
	})
}
