package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/tests/fake"
)

func TestArchiveProject(t *testing.T) {
	a := require.New(t)
	log.SetLevel(zapcore.DebugLevel)
	ctx := context.Background()
	ctl := &controller{}
	dataDir := t.TempDir()
	err := ctl.StartServerWithExternalPg(ctx, &config{
		dataDir:            dataDir,
		vcsProviderCreator: fake.NewGitLab,
	})
	a.NoError(err)
	defer ctl.Close(ctx)
	err = ctl.Login()
	a.NoError(err)
	err = ctl.setLicense()
	a.NoError(err)

	instanceRootDir := t.TempDir()
	instanceName := "testInstance1"
	instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, instanceName)
	a.NoError(err)

	environments, err := ctl.getEnvironments()
	a.NoError(err)
	prodEnvironment, err := findEnvironment(environments, "Prod")
	a.NoError(err)

	instance, err := ctl.addInstance(api.InstanceCreate{
		EnvironmentID: prodEnvironment.ID,
		Name:          "test",
		Engine:        db.SQLite,
		Host:          instanceDir,
	})
	a.NoError(err)

	t.Run("ArchiveProjectWithDatbase", func(t *testing.T) {
		project, err := ctl.createProject(api.ProjectCreate{
			Name:       "ProjectWithDatabase",
			Key:        "PWD",
			TenantMode: api.TenantModeDisabled,
		})
		a.NoError(err)

		databaseName := "db1"
		err = ctl.createDatabase(project, instance, databaseName, "", nil)
		a.NoError(err)

		status := string(api.Archived)
		err = ctl.patchProject(api.ProjectPatch{
			ID:        project.ID,
			RowStatus: &status,
			UpdaterID: project.Creator.ID,
		})
		a.Error(err)
	})

	t.Run("ArchiveProjectWithOpenIssue", func(t *testing.T) {
		project, err := ctl.createProject(api.ProjectCreate{
			Name:       "ProjectWithOpenIssue",
			Key:        "PWO",
			TenantMode: api.TenantModeDisabled,
		})
		a.NoError(err)

		databaseName := "fakedb"
		createDatabaseCtx := &api.CreateDatabaseContext{
			InstanceID:   instance.ID,
			DatabaseName: databaseName,
			Labels:       "",
			CharacterSet: "utf8mb4",
			Collation:    "utf8mb4_general_ci",
		}

		c, err := json.Marshal(createDatabaseCtx)
		a.NoError(err)

		_, err = ctl.createIssue(api.IssueCreate{
			ProjectID:   project.ID,
			Name:        fmt.Sprintf("create database %q", databaseName),
			Type:        api.IssueDatabaseCreate,
			Description: fmt.Sprintf("This creates a database %q.", databaseName),
			// Assign to self.
			AssigneeID:    project.Creator.ID,
			CreateContext: string(c),
		})
		a.NoError(err)

		status := string(api.Archived)
		err = ctl.patchProject(api.ProjectPatch{
			ID:        project.ID,
			RowStatus: &status,
			UpdaterID: project.Creator.ID,
		})
		a.Error(err)
	})
}
