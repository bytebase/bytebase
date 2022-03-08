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
	"github.com/kr/pretty"
)

func TestSchemaAndDataUpdate(t *testing.T) {
	t.Parallel()
	err := func() error {
		ctx := context.Background()
		ctl := &controller{}
		dataDir := t.TempDir()
		if err := ctl.StartMain(ctx, dataDir, getTestPort(t.Name())); err != nil {
			return err
		}
		defer ctl.Close()
		if err := ctl.Login(); err != nil {
			return err
		}
		if err := ctl.setLicense(); err != nil {
			return err
		}

		// Create a project.
		project, err := ctl.createProject(api.ProjectCreate{
			Name: "Test Project",
			Key:  "TestSchemaUpdate",
		})
		if err != nil {
			return fmt.Errorf("failed to create project, error: %v", err)
		}

		// Provision an instance.
		instanceRootDir := t.TempDir()
		instanceName := "testInstance1"
		instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, instanceName)
		if err != nil {
			return err
		}

		environments, err := ctl.getEnvironments()
		if err != nil {
			return err
		}
		prodEnvironment, err := findEnvironment(environments, "Prod")
		if err != nil {
			return err
		}

		// Add an instance.
		instance, err := ctl.addInstance(api.InstanceCreate{
			EnvironmentID: prodEnvironment.ID,
			Name:          instanceName,
			Engine:        db.SQLite,
			Host:          instanceDir,
		})
		if err != nil {
			return fmt.Errorf("failed to add instance, error: %v", err)
		}

		// Expecting project to have no database.
		databases, err := ctl.getDatabases(api.DatabaseFind{
			ProjectID: &project.ID,
		})
		if err != nil {
			return fmt.Errorf("failed to get databases, error: %v", err)
		}
		if len(databases) != 0 {
			return fmt.Errorf("invalid number of databases %v in project %v, expecting no database", len(databases), project.ID)
		}
		// Expecting instance to have no database.
		databases, err = ctl.getDatabases(api.DatabaseFind{
			InstanceID: &instance.ID,
		})
		if err != nil {
			return fmt.Errorf("failed to get databases, error: %v", err)
		}
		if len(databases) != 0 {
			return fmt.Errorf("invalid number of databases %v in instance %v, expecting no database", len(databases), instance.ID)
		}

		// Create an issue that creates a database.
		databaseName := "testSchemaUpdate"
		if err := ctl.createDatabase(project, instance, databaseName, nil /* labelMap */); err != nil {
			return err
		}

		// Expecting project to have 1 database.
		databases, err = ctl.getDatabases(api.DatabaseFind{
			ProjectID: &project.ID,
		})
		if err != nil {
			return fmt.Errorf("failed to get databases, error: %v", err)
		}
		if len(databases) != 1 {
			return fmt.Errorf("invalid number of databases %v in project %v, expecting one database", project.ID, len(databases))
		}
		database := databases[0]
		if database.Instance.ID != instance.ID {
			return fmt.Errorf("expect database %v name %q to be in instance %v, got %v", database.ID, database.Name, instance.ID, database.Instance.ID)
		}

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
		if err != nil {
			return fmt.Errorf("failed to construct schema update issue CreateContext payload, error: %v", err)
		}
		issue, err := ctl.createIssue(api.IssueCreate{
			ProjectID:   project.ID,
			Name:        fmt.Sprintf("update schema for database %q", databaseName),
			Type:        api.IssueDatabaseSchemaUpdate,
			Description: fmt.Sprintf("This updates the schema of database %q.", databaseName),
			// Assign to self.
			AssigneeID:    project.Creator.ID,
			CreateContext: string(createContext),
		})
		if err != nil {
			return fmt.Errorf("failed to create schema update issue, error: %v", err)
		}
		status, err := ctl.waitIssuePipeline(issue.ID)
		if err != nil {
			return fmt.Errorf("failed to wait for issue %v pipeline %v, error: %v", issue.ID, issue.Pipeline.ID, err)
		}
		if status != api.TaskDone {
			return fmt.Errorf("issue %v pipeline %v is expected to finish with status done got %v", issue.ID, issue.Pipeline.ID, status)
		}

		// Query schema.
		result, err := ctl.query(instance, databaseName, bookTableQuery)
		if err != nil {
			return err
		}
		if bookSchemaSQLResult != result {
			return fmt.Errorf("SQL result want %q, got %q, diff %q", bookSchemaSQLResult, result, pretty.Diff(bookSchemaSQLResult, result))
		}

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
		if err != nil {
			return fmt.Errorf("failed to construct data update issue CreateContext payload, error: %v", err)
		}
		issue, err = ctl.createIssue(api.IssueCreate{
			ProjectID:   project.ID,
			Name:        fmt.Sprintf("update data for database %q", databaseName),
			Type:        api.IssueDatabaseDataUpdate,
			Description: fmt.Sprintf("This updates the data of database %q.", databaseName),
			// Assign to self.
			AssigneeID:    project.Creator.ID,
			CreateContext: string(createContext),
		})
		if err != nil {
			return fmt.Errorf("failed to create data update issue, error: %v", err)
		}
		status, err = ctl.waitIssuePipeline(issue.ID)
		if err != nil {
			return fmt.Errorf("failed to wait for issue %v pipeline %v, error: %v", issue.ID, issue.Pipeline.ID, err)
		}
		if status != api.TaskDone {
			return fmt.Errorf("issue %v pipeline %v is expected to finish with status done got %v", issue.ID, issue.Pipeline.ID, status)
		}

		// Get migration history.
		histories, err := ctl.getInstanceMigrationHistory(db.MigrationHistoryFind{ID: &instance.ID})
		if err != nil {
			return err
		}
		wantHistories := []api.MigrationHistory{
			{
				ID:         3,
				Database:   databaseName,
				Source:     db.UI,
				Type:       db.Data,
				Status:     db.Done,
				Schema:     dumpedSchema,
				SchemaPrev: dumpedSchema,
			},
			{
				ID:         2,
				Database:   databaseName,
				Source:     db.UI,
				Type:       db.Migrate,
				Status:     db.Done,
				Schema:     dumpedSchema,
				SchemaPrev: "",
			},
			{
				ID:         1,
				Database:   databaseName,
				Source:     db.UI,
				Type:       db.Baseline,
				Status:     db.Done,
				Schema:     "",
				SchemaPrev: "",
			},
		}
		if len(histories) != len(wantHistories) {
			return fmt.Errorf("number of migration history got %v, want %v", len(histories), len(wantHistories))
		}
		for i, history := range histories {
			got := api.MigrationHistory{
				ID:         history.ID,
				Database:   history.Database,
				Source:     history.Source,
				Type:       history.Type,
				Status:     history.Status,
				Schema:     history.Schema,
				SchemaPrev: history.SchemaPrev,
			}
			want := wantHistories[i]
			diff := pretty.Diff(got, want)
			if len(diff) != 0 {
				return fmt.Errorf("migration history %v got %v, want %v, diff %v", i, got, want, diff)
			}
			if history.Version == "" {
				return fmt.Errorf("empty migration history version for migration %v", i)
			}
		}

		// Create a manual backup.
		backup, err := ctl.createBackup(api.BackupCreate{
			DatabaseID:     database.ID,
			Name:           "name",
			Type:           api.BackupTypeManual,
			StorageBackend: api.BackupStorageBackendLocal,
		})
		if err != nil {
			return fmt.Errorf("failed to create backup, error %v", err)
		}
		if err := ctl.waitBackup(backup.DatabaseID, backup.ID); err != nil {
			return fmt.Errorf("failed to wait for backup, error %v", err)
		}

		backupPath := path.Join(dataDir, backup.Path)
		backupContent, err := os.ReadFile(backupPath)
		if err != nil {
			return fmt.Errorf("failed to read backup file %q, error %v", backupPath, err)
		}
		if string(backupContent) != backupDump {
			return fmt.Errorf("backup content doesn't match, got %q, want %q", backupContent, backupDump)
		}

		// Create an issue that creates a database.
		cloneDatabaseName := "testClone"
		if err := ctl.cloneDatabaseFromBackup(project, instance, cloneDatabaseName, backup, nil /* labelMap */); err != nil {
			return err
		}
		// Query clone database book table data.
		result, err = ctl.query(instance, cloneDatabaseName, bookDataQuery)
		if err != nil {
			return err
		}
		if bookDataSQLResult != result {
			return fmt.Errorf("SQL result want %q, got %q, diff %q", bookSchemaSQLResult, result, pretty.Diff(bookSchemaSQLResult, result))
		}
		// Query clone migration history.
		histories, err = ctl.getInstanceMigrationHistory(db.MigrationHistoryFind{ID: &instance.ID, Database: &cloneDatabaseName})
		if err != nil {
			return err
		}
		wantCloneHistories := []api.MigrationHistory{
			{
				ID:         5,
				Database:   cloneDatabaseName,
				Source:     db.UI,
				Type:       db.Branch,
				Status:     db.Done,
				Schema:     dumpedSchema,
				SchemaPrev: dumpedSchema,
			},
			{
				ID:         4,
				Database:   cloneDatabaseName,
				Source:     db.UI,
				Type:       db.Baseline,
				Status:     db.Done,
				Schema:     "",
				SchemaPrev: "",
			},
		}
		if len(histories) != len(wantCloneHistories) {
			return fmt.Errorf("number of migration history got %v, want %v", len(histories), len(wantCloneHistories))
		}
		for i, history := range histories {
			got := api.MigrationHistory{
				ID:         history.ID,
				Database:   history.Database,
				Source:     history.Source,
				Type:       history.Type,
				Status:     history.Status,
				Schema:     history.Schema,
				SchemaPrev: history.SchemaPrev,
			}
			want := wantCloneHistories[i]
			diff := pretty.Diff(got, want)
			if len(diff) != 0 {
				return fmt.Errorf("migration history %v got %v, want %v, diff %v", i, got, want, diff)
			}
		}

		// Create a sheet to mock SQL editor new tab action with UNKNOWN ProjectID.
		_, err = ctl.createSheet(api.SheetCreate{
			ProjectID:  -1,
			DatabaseID: &database.ID,
			Name:       "my-sheet",
			Statement:  "SELECT * FROM demo",
			Visibility: api.PrivateSheet,
		})
		if err != nil {
			return fmt.Errorf("failed to create sheet, error %v", err)
		}

		_, err = ctl.listSheets(database.ID)
		if err != nil {
			return fmt.Errorf("failed to list sheet, error %v", err)
		}

		// Test if POST /api/database/:id/datasource api is working right.
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

		if err != nil {
			return fmt.Errorf("failed to create data source, error %v", err)
		}
		return nil
	}()
	if err != nil {
		t.Error(err)
	}
}

func TestVCS(t *testing.T) {
	t.Parallel()
	err := func() error {
		ctx := context.Background()
		ctl := &controller{}
		dataDir := t.TempDir()
		if err := ctl.StartMain(ctx, dataDir, getTestPort(t.Name())); err != nil {
			return err
		}
		defer ctl.Close()
		if err := ctl.Login(); err != nil {
			return err
		}
		if err := ctl.setLicense(); err != nil {
			return err
		}

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
		if err != nil {
			return fmt.Errorf("failed to create VCS, error: %v", err)
		}

		// Create a project.
		project, err := ctl.createProject(api.ProjectCreate{
			Name: "Test VCS Project",
			Key:  "TestVCSSchemaUpdate",
		})
		if err != nil {
			return fmt.Errorf("failed to create project, error: %v", err)
		}

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
		if err != nil {
			return fmt.Errorf("failed to create repository, error: %v", err)
		}

		// Provision an instance.
		instanceRootDir := t.TempDir()
		instanceName := "testInstance1"
		instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, instanceName)
		if err != nil {
			return err
		}

		environments, err := ctl.getEnvironments()
		if err != nil {
			return err
		}
		prodEnvironment, err := findEnvironment(environments, "Prod")
		if err != nil {
			return err
		}

		// Add an instance.
		instance, err := ctl.addInstance(api.InstanceCreate{
			EnvironmentID: prodEnvironment.ID,
			Name:          instanceName,
			Engine:        db.SQLite,
			Host:          instanceDir,
		})
		if err != nil {
			return fmt.Errorf("failed to add instance, error: %v", err)
		}

		// Create an issue that creates a database.
		databaseName := "testVCSSchemaUpdate"
		if err := ctl.createDatabase(project, instance, databaseName, nil /* labelMap */); err != nil {
			return err
		}

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
		if err := ctl.gitlab.AddFiles(gitlabProjectIDStr, map[string]string{gitFile: migrationStatement}); err != nil {
			return fmt.Errorf("failed to add files to gitlab project %v, error %v", gitlabProjectID, err)
		}
		if err := ctl.gitlab.SendCommits(gitlabProjectIDStr, pushEvent); err != nil {
			return fmt.Errorf("failed to send commits to gitlab project %v, error %v", gitlabProjectID, err)
		}

		// Get schema update issue.
		openStatus := []api.IssueStatus{api.IssueOpen}
		issues, err := ctl.getIssues(api.IssueFind{ProjectID: &project.ID, StatusList: &openStatus})
		if err != nil {
			return fmt.Errorf("failed to get open issues for project %v, error: %v", project.ID, err)
		}
		if len(issues) != 1 {
			return fmt.Errorf("invalid number of open issues %v in project %v, expecting one issue", len(issues), project.ID)
		}
		issue := issues[0]
		status, err := ctl.waitIssuePipeline(issue.ID)
		if err != nil {
			return fmt.Errorf("failed to wait for issue %v pipeline %v, error: %v", issue.ID, issue.Pipeline.ID, err)
		}
		if status != api.TaskDone {
			return fmt.Errorf("issue %v pipeline %v is expected to finish with status done got %v", issue.ID, issue.Pipeline.ID, status)
		}
		if _, err := ctl.patchIssueStatus(api.IssueStatusPatch{
			ID:     issue.ID,
			Status: api.IssueDone,
		}); err != nil {
			return err
		}

		// Query schema.
		result, err := ctl.query(instance, databaseName, bookTableQuery)
		if err != nil {
			return err
		}
		if bookSchemaSQLResult != result {
			return fmt.Errorf("SQL result want %q, got %q, diff %q", bookSchemaSQLResult, result, pretty.Diff(bookSchemaSQLResult, result))
		}

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
		if err := ctl.gitlab.AddFiles(gitlabProjectIDStr, map[string]string{gitFile: dataUpdateStatement}); err != nil {
			return fmt.Errorf("failed to add files to gitlab project %v, error %v", gitlabProjectID, err)
		}
		if err := ctl.gitlab.SendCommits(gitlabProjectIDStr, pushEvent); err != nil {
			return fmt.Errorf("failed to send commits to gitlab project %v, error %v", gitlabProjectID, err)
		}
		// Get data update issue.
		openStatus = []api.IssueStatus{api.IssueOpen}
		issues, err = ctl.getIssues(api.IssueFind{ProjectID: &project.ID, StatusList: &openStatus})
		if err != nil {
			return fmt.Errorf("failed to get open issues for project %v, error: %v", project.ID, err)
		}
		if len(issues) != 1 {
			return fmt.Errorf("invalid number of open issues %v in project %v, expecting one issue", len(issues), project.ID)
		}
		issue = issues[0]
		status, err = ctl.waitIssuePipeline(issue.ID)
		if err != nil {
			return fmt.Errorf("failed to wait for issue %v pipeline %v, error: %v", issue.ID, issue.Pipeline.ID, err)
		}
		if status != api.TaskDone {
			return fmt.Errorf("issue %v pipeline %v is expected to finish with status done got %v", issue.ID, issue.Pipeline.ID, status)
		}
		if _, err := ctl.patchIssueStatus(api.IssueStatusPatch{
			ID:     issue.ID,
			Status: api.IssueDone,
		}); err != nil {
			return err
		}

		// Get migration history.
		histories, err := ctl.getInstanceMigrationHistory(db.MigrationHistoryFind{ID: &instance.ID})
		if err != nil {
			return err
		}
		wantHistories := []api.MigrationHistory{
			{
				ID:         3,
				Database:   databaseName,
				Source:     db.VCS,
				Type:       db.Data,
				Status:     db.Done,
				Schema:     dumpedSchema,
				SchemaPrev: dumpedSchema,
			},
			{
				ID:         2,
				Database:   databaseName,
				Source:     db.VCS,
				Type:       db.Migrate,
				Status:     db.Done,
				Schema:     dumpedSchema,
				SchemaPrev: "",
			},
			{
				ID:         1,
				Database:   databaseName,
				Source:     db.UI,
				Type:       db.Baseline,
				Status:     db.Done,
				Schema:     "",
				SchemaPrev: "",
			},
		}
		if len(histories) != len(wantHistories) {
			return fmt.Errorf("number of migration history got %v, want %v", len(histories), len(wantHistories))
		}
		for i, history := range histories {
			got := api.MigrationHistory{
				ID:         history.ID,
				Database:   history.Database,
				Source:     history.Source,
				Type:       history.Type,
				Status:     history.Status,
				Schema:     history.Schema,
				SchemaPrev: history.SchemaPrev,
			}
			want := wantHistories[i]
			diff := pretty.Diff(got, want)
			if len(diff) != 0 {
				return fmt.Errorf("migration history %v got %v, want %v, diff %v", i, got, want, diff)
			}
			if history.Version == "" {
				return fmt.Errorf("empty migration history version for migration %v", i)
			}
		}
		if histories[0].Version != "ver2" {
			return fmt.Errorf("invalid migration(0) history version, want ver2 got %v", histories[0].Version)
		}
		if histories[1].Version != "ver1" {
			return fmt.Errorf("invalid migration(0) history version, want ver1 got %v", histories[0].Version)
		}
		return nil
	}()
	if err != nil {
		t.Error(err)
	}
}
