package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
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
		dataDir:            dataDir,
		vcsProviderCreator: fake.NewGitLab,
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
	instanceUID, err := strconv.Atoi(instance.Uid)
	a.NoError(err)

	t.Run("ArchiveProjectWithDatbase", func(t *testing.T) {
		project, err := ctl.createProject(ctx)
		a.NoError(err)
		projectUID, err := strconv.Atoi(project.Uid)
		a.NoError(err)

		databaseName := "db1"
		err = ctl.createDatabase(ctx, projectUID, instance, databaseName, "", nil)
		a.NoError(err)

		_, err = ctl.projectServiceClient.DeleteProject(ctx, &v1pb.DeleteProjectRequest{
			Name: project.Name,
		})
		a.Error(err)
	})

	t.Run("ArchiveProjectWithOpenIssue", func(t *testing.T) {
		project, err := ctl.createProject(ctx)
		a.NoError(err)
		projectUID, err := strconv.Atoi(project.Uid)
		a.NoError(err)

		databaseName := "fakedb"
		createDatabaseCtx := &api.CreateDatabaseContext{
			InstanceID:   instanceUID,
			DatabaseName: databaseName,
			Labels:       "",
			CharacterSet: "utf8mb4",
			Collation:    "utf8mb4_general_ci",
		}

		c, err := json.Marshal(createDatabaseCtx)
		a.NoError(err)

		_, err = ctl.createIssue(api.IssueCreate{
			ProjectID:     projectUID,
			Name:          fmt.Sprintf("create database %q", databaseName),
			Type:          api.IssueDatabaseCreate,
			Description:   fmt.Sprintf("This creates a database %q.", databaseName),
			AssigneeID:    api.SystemBotID,
			CreateContext: string(c),
		})
		a.NoError(err)

		_, err = ctl.projectServiceClient.DeleteProject(ctx, &v1pb.DeleteProjectRequest{
			Name: project.Name,
		})
		a.ErrorContains(err, "resolve all open issues before deleting the project")
	})
}
