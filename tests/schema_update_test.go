package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/vcs"
	"github.com/bytebase/bytebase/plugin/vcs/gitlab"
	"github.com/stretchr/testify/require"
)

func TestSchemaAndDataUpdate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	ctl := &controller{}
	dataDir := t.TempDir()
	err := ctl.StartServer(ctx, dataDir, getTestPort(t.Name()), "" /*pgURL*/)
	require.NoError(t, err)
	defer ctl.Close(ctx)
	err = ctl.Login()
	require.NoError(t, err)

	err = ctl.setLicense()
	require.NoError(t, err)

	// Create a project.
	project, err := ctl.createProject(api.ProjectCreate{
		Name: "Test Project",
		Key:  "TestSchemaUpdate",
	})
	require.NoError(t, err)

	// Provision an instance.
	instanceRootDir := t.TempDir()
	instanceName := "testInstance1"
	instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, instanceName)
	require.NoError(t, err)

	environments, err := ctl.getEnvironments()
	require.NoError(t, err)
	prodEnvironment, err := findEnvironment(environments, "Prod")
	require.NoError(t, err)

	// Add an instance.
	instance, err := ctl.addInstance(api.InstanceCreate{
		EnvironmentID: prodEnvironment.ID,
		Name:          instanceName,
		Engine:        db.SQLite,
		Host:          instanceDir,
	})
	require.NoError(t, err)

	// Expecting project to have no database.
	databases, err := ctl.getDatabases(api.DatabaseFind{
		ProjectID: &project.ID,
	})
	require.NoError(t, err)
	require.Zero(t, len(databases))
	// Expecting instance to have no database.
	databases, err = ctl.getDatabases(api.DatabaseFind{
		InstanceID: &instance.ID,
	})
	require.NoError(t, err)
	require.Zero(t, len(databases))

	// Create an issue that creates a database.
	databaseName := "testSchemaUpdate"
	err = ctl.createDatabase(project, instance, databaseName, nil /* labelMap */)
	require.NoError(t, err)

	// Expecting project to have 1 database.
	databases, err = ctl.getDatabases(api.DatabaseFind{
		ProjectID: &project.ID,
	})
	require.NoError(t, err)
	require.Equal(t, len(databases), 1)
	database := databases[0]
	require.Equal(t, database.Instance.ID, instance.ID)

	// Create an issue that updates database schema.
	createContext, err := json.Marshal(&api.UpdateSchemaContext{
		MigrationType: db.Migrate,
		UpdateSchemaDetailList: []*api.UpdateSchemaDetail{
			{
				DatabaseID: database.ID,
				Statement:  migrationStatement,
			},
		},
	})
	require.NoError(t, err)
	issue, err := ctl.createIssue(api.IssueCreate{
		ProjectID:   project.ID,
		Name:        fmt.Sprintf("update schema for database %q", databaseName),
		Type:        api.IssueDatabaseSchemaUpdate,
		Description: fmt.Sprintf("This updates the schema of database %q.", databaseName),
		// Assign to self.
		AssigneeID:    project.Creator.ID,
		CreateContext: string(createContext),
	})
	require.NoError(t, err)
	status, err := ctl.waitIssuePipeline(issue.ID)
	require.NoError(t, err)
	require.Equal(t, status, api.TaskDone)

	// Query schema.
	result, err := ctl.query(instance, databaseName, bookTableQuery)
	require.NoError(t, err)
	require.Equal(t, bookSchemaSQLResult, result)

	// Create an issue that updates database data.
	createContext, err = json.Marshal(&api.UpdateSchemaContext{
		MigrationType: db.Data,
		UpdateSchemaDetailList: []*api.UpdateSchemaDetail{
			{
				DatabaseID: database.ID,
				Statement:  dataUpdateStatement,
			},
		},
	})
	require.NoError(t, err)
	issue, err = ctl.createIssue(api.IssueCreate{
		ProjectID:   project.ID,
		Name:        fmt.Sprintf("update data for database %q", databaseName),
		Type:        api.IssueDatabaseDataUpdate,
		Description: fmt.Sprintf("This updates the data of database %q.", databaseName),
		// Assign to self.
		AssigneeID:    project.Creator.ID,
		CreateContext: string(createContext),
	})
	require.NoError(t, err)
	status, err = ctl.waitIssuePipeline(issue.ID)
	require.NoError(t, err)
	require.Equal(t, status, api.TaskDone)

	// Get migration history.
	histories, err := ctl.getInstanceMigrationHistory(db.MigrationHistoryFind{ID: &instance.ID})
	require.NoError(t, err)
	wantHistories := []api.MigrationHistory{
		{
			Database:   databaseName,
			Source:     db.UI,
			Type:       db.Data,
			Status:     db.Done,
			Schema:     dumpedSchema,
			SchemaPrev: dumpedSchema,
		},
		{
			Database:   databaseName,
			Source:     db.UI,
			Type:       db.Migrate,
			Status:     db.Done,
			Schema:     dumpedSchema,
			SchemaPrev: "",
		},
		{
			Database:   databaseName,
			Source:     db.UI,
			Type:       db.Baseline,
			Status:     db.Done,
			Schema:     "",
			SchemaPrev: "",
		},
	}
	require.Equal(t, len(histories), len(wantHistories))
	for i, history := range histories {
		got := api.MigrationHistory{
			Database:   history.Database,
			Source:     history.Source,
			Type:       history.Type,
			Status:     history.Status,
			Schema:     history.Schema,
			SchemaPrev: history.SchemaPrev,
		}
		want := wantHistories[i]
		require.Equal(t, got, want)
		require.NotEqual(t, history.Version, "")
	}

	// Create a manual backup.
	backup, err := ctl.createBackup(api.BackupCreate{
		DatabaseID:     database.ID,
		Name:           "name",
		Type:           api.BackupTypeManual,
		StorageBackend: api.BackupStorageBackendLocal,
	})
	require.NoError(t, err)
	err = ctl.waitBackup(backup.DatabaseID, backup.ID)
	require.NoError(t, err)

	backupPath := path.Join(dataDir, backup.Path)
	backupContent, err := os.ReadFile(backupPath)
	require.NoError(t, err)
	require.Equal(t, string(backupContent), backupDump)

	// Create an issue that creates a database.
	cloneDatabaseName := "testClone"
	err = ctl.cloneDatabaseFromBackup(project, instance, cloneDatabaseName, backup, nil /* labelMap */)
	require.NoError(t, err)

	// Query clone database book table data.
	result, err = ctl.query(instance, cloneDatabaseName, bookDataQuery)
	require.NoError(t, err)
	require.Equal(t, bookDataSQLResult, result)
	// Query clone migration history.
	histories, err = ctl.getInstanceMigrationHistory(db.MigrationHistoryFind{ID: &instance.ID, Database: &cloneDatabaseName})
	require.NoError(t, err)
	wantCloneHistories := []api.MigrationHistory{
		{
			Database:   cloneDatabaseName,
			Source:     db.UI,
			Type:       db.Branch,
			Status:     db.Done,
			Schema:     dumpedSchema,
			SchemaPrev: dumpedSchema,
		},
		{
			Database:   cloneDatabaseName,
			Source:     db.UI,
			Type:       db.Baseline,
			Status:     db.Done,
			Schema:     "",
			SchemaPrev: "",
		},
	}
	require.Equal(t, len(histories), len(wantCloneHistories))
	for i, history := range histories {
		got := api.MigrationHistory{
			Database:   history.Database,
			Source:     history.Source,
			Type:       history.Type,
			Status:     history.Status,
			Schema:     history.Schema,
			SchemaPrev: history.SchemaPrev,
		}
		want := wantCloneHistories[i]
		require.Equal(t, got, want)
	}

	// Create a sheet to mock SQL editor new tab action with UNKNOWN ProjectID.
	_, err = ctl.createSheet(api.SheetCreate{
		ProjectID:  -1,
		DatabaseID: &database.ID,
		Name:       "my-sheet",
		Statement:  "SELECT * FROM demo",
		Visibility: api.PrivateSheet,
	})
	require.NoError(t, err)

	_, err = ctl.listSheets(database.ID)
	require.NoError(t, err)

	// Test if POST /api/database/:id/data-source api is working right.
	// TODO(steven): I will add read-only data source testing to a separate test later.
	err = ctl.createDataSource(api.DataSourceCreate{
		InstanceID: instance.ID,
		DatabaseID: database.ID,
		CreatorID:  project.Creator.ID,
		Name:       "ADMIN data source",
		Type:       "ADMIN",
		Username:   "root",
		Password:   "",
	})
	require.NoError(t, err)
}

func TestVCS(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	ctl := &controller{}
	dataDir := t.TempDir()
	err := ctl.StartServer(ctx, dataDir, getTestPort(t.Name()), "" /*pgURL*/)
	require.NoError(t, err)
	defer ctl.Close(ctx)
	err = ctl.Login()
	require.NoError(t, err)
	err = ctl.setLicense()
	require.NoError(t, err)

	// Create a VCS.
	applicationID := "testApplicationID"
	applicationSecret := "testApplicationSecret"
	vcs, err := ctl.createVCS(api.VCSCreate{
		Name:          "TestVCS",
		Type:          vcs.GitLabSelfHost,
		InstanceURL:   ctl.gitURL,
		APIURL:        ctl.gitAPIURL,
		ApplicationID: applicationID,
		Secret:        applicationSecret,
	})
	require.NoError(t, err)

	// Create a project.
	project, err := ctl.createProject(api.ProjectCreate{
		Name: "Test VCS Project",
		Key:  "TestVCSSchemaUpdate",
	})
	require.NoError(t, err)

	// Create a repository.
	repositoryPath := "test/schemaUpdate"
	accessToken := "accessToken1"
	refreshToken := "refreshToken1"
	gitlabProjectID := 121
	gitlabProjectIDStr := fmt.Sprintf("%d", gitlabProjectID)
	// create a gitlab project.
	ctl.gitlab.CreateProject(gitlabProjectIDStr)
	_, err = ctl.createRepository(api.RepositoryCreate{
		VCSID:              vcs.ID,
		ProjectID:          project.ID,
		Name:               "Test Repository",
		FullPath:           repositoryPath,
		WebURL:             fmt.Sprintf("%s/%s", ctl.gitURL, repositoryPath),
		BranchFilter:       "feature/foo",
		BaseDirectory:      "bbtest",
		FilePathTemplate:   "{{ENV_NAME}}/{{DB_NAME}}__{{VERSION}}__{{TYPE}}__{{DESCRIPTION}}.sql",
		SchemaPathTemplate: "{{ENV_NAME}}/.{{DB_NAME}}__LATEST.sql",
		ExternalID:         gitlabProjectIDStr,
		AccessToken:        accessToken,
		ExpiresTs:          0,
		RefreshToken:       refreshToken,
	})
	require.NoError(t, err)

	// Provision an instance.
	instanceRootDir := t.TempDir()
	instanceName := "testInstance1"
	instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, instanceName)
	require.NoError(t, err)

	environments, err := ctl.getEnvironments()
	require.NoError(t, err)
	prodEnvironment, err := findEnvironment(environments, "Prod")
	require.NoError(t, err)

	// Add an instance.
	instance, err := ctl.addInstance(api.InstanceCreate{
		EnvironmentID: prodEnvironment.ID,
		Name:          instanceName,
		Engine:        db.SQLite,
		Host:          instanceDir,
	})
	require.NoError(t, err)

	// Create an issue that creates a database.
	databaseName := "testVCSSchemaUpdate"
	err = ctl.createDatabase(project, instance, databaseName, nil /* labelMap */)
	require.NoError(t, err)

	// Simulate Git commits for schema update.
	gitFile := "bbtest/Prod/testVCSSchemaUpdate__ver1__migrate__create_a_test_table.sql"
	pushEvent := &gitlab.WebhookPushEvent{
		ObjectKind: gitlab.WebhookPush,
		Ref:        "refs/heads/feature/foo",
		Project: gitlab.WebhookProject{
			ID: gitlabProjectID,
		},
		CommitList: []gitlab.WebhookCommit{
			{
				Timestamp: "2021-01-13T13:14:00Z",
				AddedList: []string{
					gitFile,
				},
			},
		},
	}
	err = ctl.gitlab.AddFiles(gitlabProjectIDStr, map[string]string{gitFile: migrationStatement})
	require.NoError(t, err)

	err = ctl.gitlab.SendCommits(gitlabProjectIDStr, pushEvent)
	require.NoError(t, err)

	// Get schema update issue.
	openStatus := []api.IssueStatus{api.IssueOpen}
	issues, err := ctl.getIssues(api.IssueFind{ProjectID: &project.ID, StatusList: &openStatus})
	require.NoError(t, err)
	require.Equal(t, len(issues), 1)
	issue := issues[0]
	status, err := ctl.waitIssuePipeline(issue.ID)
	require.NoError(t, err)
	require.Equal(t, status, api.TaskDone)
	_, err = ctl.patchIssueStatus(api.IssueStatusPatch{
		ID:     issue.ID,
		Status: api.IssueDone,
	})
	require.NoError(t, err)

	// Query schema.
	result, err := ctl.query(instance, databaseName, bookTableQuery)
	require.NoError(t, err)
	require.Equal(t, bookSchemaSQLResult, result)

	// Simulate Git commits for schema update.
	gitFile = "bbtest/Prod/testVCSSchemaUpdate__ver2__data__insert_data.sql"
	pushEvent = &gitlab.WebhookPushEvent{
		ObjectKind: gitlab.WebhookPush,
		Ref:        "refs/heads/feature/foo",
		Project: gitlab.WebhookProject{
			ID: gitlabProjectID,
		},
		CommitList: []gitlab.WebhookCommit{
			{
				Timestamp: "2021-01-13T13:14:00Z",
				AddedList: []string{
					gitFile,
				},
			},
		},
	}
	err = ctl.gitlab.AddFiles(gitlabProjectIDStr, map[string]string{gitFile: dataUpdateStatement})
	require.NoError(t, err)

	err = ctl.gitlab.SendCommits(gitlabProjectIDStr, pushEvent)
	require.NoError(t, err)

	// Get data update issue.
	openStatus = []api.IssueStatus{api.IssueOpen}
	issues, err = ctl.getIssues(api.IssueFind{ProjectID: &project.ID, StatusList: &openStatus})
	require.NoError(t, err)
	require.Equal(t, len(issues), 1)
	issue = issues[0]
	status, err = ctl.waitIssuePipeline(issue.ID)
	require.NoError(t, err)
	require.Equal(t, status, api.TaskDone)
	_, err = ctl.patchIssueStatus(api.IssueStatusPatch{
		ID:     issue.ID,
		Status: api.IssueDone,
	})
	require.NoError(t, err)

	// Get migration history.
	histories, err := ctl.getInstanceMigrationHistory(db.MigrationHistoryFind{ID: &instance.ID})
	require.NoError(t, err)
	wantHistories := []api.MigrationHistory{
		{
			Database:   databaseName,
			Source:     db.VCS,
			Type:       db.Data,
			Status:     db.Done,
			Schema:     dumpedSchema,
			SchemaPrev: dumpedSchema,
		},
		{
			Database:   databaseName,
			Source:     db.VCS,
			Type:       db.Migrate,
			Status:     db.Done,
			Schema:     dumpedSchema,
			SchemaPrev: "",
		},
		{
			Database:   databaseName,
			Source:     db.UI,
			Type:       db.Baseline,
			Status:     db.Done,
			Schema:     "",
			SchemaPrev: "",
		},
	}
	require.Equal(t, len(histories), len(wantHistories))

	for i, history := range histories {
		got := api.MigrationHistory{
			Database:   history.Database,
			Source:     history.Source,
			Type:       history.Type,
			Status:     history.Status,
			Schema:     history.Schema,
			SchemaPrev: history.SchemaPrev,
		}
		want := wantHistories[i]
		require.Equal(t, got, want)
		require.NotEqual(t, history.Version, "")
	}
	require.Equal(t, histories[0].Version, "ver2")
	require.Equal(t, histories[1].Version, "ver1")
}
