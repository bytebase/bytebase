package tests

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/vcs"
	"github.com/bytebase/bytebase/plugin/vcs/github"
	"github.com/bytebase/bytebase/plugin/vcs/gitlab"
	"github.com/bytebase/bytebase/resources/postgres"
	"github.com/bytebase/bytebase/tests/fake"
)

func TestSchemaAndDataUpdate(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	dataDir := t.TempDir()
	err := ctl.StartServer(ctx, dataDir, fake.NewGitLab, getTestPort(t.Name()))
	a.NoError(err)
	defer ctl.Close(ctx)
	err = ctl.Login()
	a.NoError(err)

	err = ctl.setLicense()
	a.NoError(err)

	// Create a project.
	project, err := ctl.createProject(api.ProjectCreate{
		Name: "Test Project",
		Key:  "TestSchemaUpdate",
	})
	a.NoError(err)

	// Provision an instance.
	instanceRootDir := t.TempDir()
	instanceName := "testInstance1"
	instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, instanceName)
	a.NoError(err)

	environments, err := ctl.getEnvironments()
	a.NoError(err)
	prodEnvironment, err := findEnvironment(environments, "Prod")
	a.NoError(err)

	// Add an instance.
	instance, err := ctl.addInstance(api.InstanceCreate{
		EnvironmentID: prodEnvironment.ID,
		Name:          instanceName,
		Engine:        db.SQLite,
		Host:          instanceDir,
	})
	a.NoError(err)

	// Expecting project to have no database.
	databases, err := ctl.getDatabases(api.DatabaseFind{
		ProjectID: &project.ID,
	})
	a.NoError(err)
	a.Zero(len(databases))
	// Expecting instance to have no database.
	databases, err = ctl.getDatabases(api.DatabaseFind{
		InstanceID: &instance.ID,
	})
	a.NoError(err)
	a.Zero(len(databases))

	// Create an issue that creates a database.
	databaseName := "testSchemaUpdate"
	err = ctl.createDatabase(project, instance, databaseName, "", nil /* labelMap */)
	a.NoError(err)

	// Expecting project to have 1 database.
	databases, err = ctl.getDatabases(api.DatabaseFind{
		ProjectID: &project.ID,
	})
	a.NoError(err)
	a.Equal(1, len(databases))
	database := databases[0]
	a.Equal(instance.ID, database.Instance.ID)

	// Create an issue that updates database schema.
	createContext, err := json.Marshal(&api.MigrationContext{
		DetailList: []*api.MigrationDetail{
			{
				MigrationType: db.Migrate,
				DatabaseID:    database.ID,
				Statement:     migrationStatement,
			},
		},
	})
	a.NoError(err)
	issue, err := ctl.createIssue(api.IssueCreate{
		ProjectID:   project.ID,
		Name:        fmt.Sprintf("update schema for database %q", databaseName),
		Type:        api.IssueDatabaseSchemaUpdate,
		Description: fmt.Sprintf("This updates the schema of database %q.", databaseName),
		// Assign to self.
		AssigneeID:    project.Creator.ID,
		CreateContext: string(createContext),
	})
	a.NoError(err)
	status, err := ctl.waitIssuePipeline(issue.ID)
	a.NoError(err)
	a.Equal(api.TaskDone, status)

	// Query schema.
	result, err := ctl.query(instance, databaseName, bookTableQuery)
	a.NoError(err)
	a.Equal(bookSchemaSQLResult, result)

	// Create an issue that updates database data.
	createContext, err = json.Marshal(&api.MigrationContext{
		DetailList: []*api.MigrationDetail{
			{
				MigrationType: db.Data,
				DatabaseID:    database.ID,
				Statement:     dataUpdateStatement,
			},
		},
	})
	a.NoError(err)
	issue, err = ctl.createIssue(api.IssueCreate{
		ProjectID:   project.ID,
		Name:        fmt.Sprintf("update data for database %q", databaseName),
		Type:        api.IssueDatabaseDataUpdate,
		Description: fmt.Sprintf("This updates the data of database %q.", databaseName),
		// Assign to self.
		AssigneeID:    project.Creator.ID,
		CreateContext: string(createContext),
	})
	a.NoError(err)
	status, err = ctl.waitIssuePipeline(issue.ID)
	a.NoError(err)
	a.Equal(api.TaskDone, status)

	// Get migration history.
	histories, err := ctl.getInstanceMigrationHistory(db.MigrationHistoryFind{ID: &instance.ID})
	a.NoError(err)
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
			Type:       db.Migrate,
			Status:     db.Done,
			Schema:     "",
			SchemaPrev: "",
		},
	}
	a.Equal(len(histories), len(wantHistories))
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
		a.Equal(want, got)
		a.NotEqual(history.Version, "")
	}

	// Create a manual backup.
	backup, err := ctl.createBackup(api.BackupCreate{
		DatabaseID:     database.ID,
		Name:           "name",
		Type:           api.BackupTypeManual,
		StorageBackend: api.BackupStorageBackendLocal,
	})
	a.NoError(err)
	err = ctl.waitBackup(backup.DatabaseID, backup.ID)
	a.NoError(err)

	backupPath := path.Join(dataDir, backup.Path)
	backupContent, err := os.ReadFile(backupPath)
	a.NoError(err)
	a.Equal(string(backupContent), backupDump)

	// Create an issue that creates a database.
	cloneDatabaseName := "testClone"
	err = ctl.cloneDatabaseFromBackup(project, instance, cloneDatabaseName, backup, nil /* labelMap */)
	a.NoError(err)

	// Query clone database book table data.
	result, err = ctl.query(instance, cloneDatabaseName, bookDataQuery)
	a.NoError(err)
	a.Equal(bookDataSQLResult, result)
	// Query clone migration history.
	histories, err = ctl.getInstanceMigrationHistory(db.MigrationHistoryFind{ID: &instance.ID, Database: &cloneDatabaseName})
	a.NoError(err)
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
			Type:       db.Migrate,
			Status:     db.Done,
			Schema:     "",
			SchemaPrev: "",
		},
	}
	a.Equal(len(histories), len(wantCloneHistories))
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
		a.Equal(want, got)
	}

	// Create a sheet to mock SQL editor new tab action with UNKNOWN ProjectID.
	_, err = ctl.createSheet(api.SheetCreate{
		ProjectID:  -1,
		DatabaseID: &database.ID,
		Name:       "my-sheet",
		Statement:  "SELECT * FROM demo",
		Visibility: api.PrivateSheet,
	})
	a.NoError(err)

	_, err = ctl.listSheets(api.SheetFind{
		DatabaseID: &database.ID,
	})
	a.NoError(err)

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
	a.NoError(err)
}

func TestVCS(t *testing.T) {
	tests := []struct {
		name                string
		vcsProviderCreator  fake.VCSProviderCreator
		vcsType             vcs.Type
		externalID          string
		repositoryFullPath  string
		newWebhookPushEvent func(added []string, modified []string) interface{}
	}{
		{
			name:               "GitLab",
			vcsProviderCreator: fake.NewGitLab,
			vcsType:            vcs.GitLabSelfHost,
			externalID:         "121",
			repositoryFullPath: "test/schemaUpdate",
			newWebhookPushEvent: func(added []string, modified []string) interface{} {
				return gitlab.WebhookPushEvent{
					ObjectKind: gitlab.WebhookPush,
					Ref:        "refs/heads/feature/foo",
					Project: gitlab.WebhookProject{
						ID: 121,
					},
					CommitList: []gitlab.WebhookCommit{
						{
							Timestamp:    "2021-01-13T13:14:00Z",
							AddedList:    added,
							ModifiedList: modified,
						},
					},
				}
			},
		},
		{
			name:               "GitHub",
			vcsProviderCreator: fake.NewGitHub,
			vcsType:            vcs.GitHubCom,
			externalID:         "octocat/Hello-World",
			repositoryFullPath: "octocat/Hello-World",
			newWebhookPushEvent: func(added []string, modified []string) interface{} {
				return github.WebhookPushEvent{
					Ref: "refs/heads/feature/foo",
					Repository: github.WebhookRepository{
						ID:       211,
						FullName: "octocat/Hello-World",
						HTMLURL:  "https://github.com/octocat/Hello-World",
					},
					Sender: github.WebhookSender{
						Login: "fake_github_author",
					},
					Commits: []github.WebhookCommit{
						{
							ID:        "fake_github_commit_id",
							Distinct:  true,
							Message:   "Fake GitHub commit message",
							Timestamp: time.Now(),
							URL:       "https://api.github.com/octocat/Hello-World/commits/fake_github_commit_id",
							Author: github.WebhookCommitAuthor{
								Name:  "fake_github_author",
								Email: "fake_github_author@localhost",
							},
							Added:    added,
							Modified: modified,
						},
					},
				}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			a := require.New(t)
			ctx := context.Background()
			ctl := &controller{}
			err := ctl.StartServer(ctx, t.TempDir(), test.vcsProviderCreator, getTestPort(t.Name()))
			a.NoError(err)
			defer func() {
				_ = ctl.Close(ctx)
			}()

			err = ctl.Login()
			a.NoError(err)
			err = ctl.setLicense()
			a.NoError(err)

			// Create a VCS.
			vcs, err := ctl.createVCS(
				api.VCSCreate{
					Name:          t.Name(),
					Type:          test.vcsType,
					InstanceURL:   ctl.vcsURL,
					APIURL:        ctl.vcsProvider.APIURL(ctl.vcsURL),
					ApplicationID: "testApplicationID",
					Secret:        "testApplicationSecret",
				},
			)
			a.NoError(err)

			// Create a project.
			project, err := ctl.createProject(
				api.ProjectCreate{
					Name: "Test VCS Project",
					Key:  "TestVCSSchemaUpdate",
				},
			)
			a.NoError(err)

			// Create a repository.
			ctl.vcsProvider.CreateRepository(test.externalID)
			_, err = ctl.createRepository(
				api.RepositoryCreate{
					VCSID:              vcs.ID,
					ProjectID:          project.ID,
					Name:               "Test Repository",
					FullPath:           test.repositoryFullPath,
					WebURL:             fmt.Sprintf("%s/%s", ctl.vcsURL, test.repositoryFullPath),
					BranchFilter:       "feature/foo",
					BaseDirectory:      baseDirectory,
					FilePathTemplate:   "{{ENV_NAME}}/{{DB_NAME}}##{{VERSION}}##{{TYPE}}##{{DESCRIPTION}}.sql",
					SchemaPathTemplate: "{{ENV_NAME}}/.{{DB_NAME}}##LATEST.sql",
					ExternalID:         test.externalID,
					AccessToken:        "accessToken1",
					RefreshToken:       "refreshToken1",
				},
			)
			a.NoError(err)

			// Provision an instance.
			instanceName := "testInstance1"
			instanceDir, err := ctl.provisionSQLiteInstance(t.TempDir(), instanceName)
			a.NoError(err)

			environments, err := ctl.getEnvironments()
			a.NoError(err)
			prodEnvironment, err := findEnvironment(environments, "Prod")
			a.NoError(err)

			// Add an instance.
			instance, err := ctl.addInstance(api.InstanceCreate{
				EnvironmentID: prodEnvironment.ID,
				Name:          instanceName,
				Engine:        db.SQLite,
				Host:          instanceDir,
			})
			a.NoError(err)

			// Create an issue that creates a database.
			databaseName := "testVCSSchemaUpdate"
			err = ctl.createDatabase(project, instance, databaseName, "", nil /* labelMap */)
			a.NoError(err)

			// Simulate Git commits for schema update.
			gitFile := "bbtest/Prod/testVCSSchemaUpdate##ver1##migrate##create_a_test_table.sql"
			err = ctl.vcsProvider.AddFiles(test.externalID, map[string]string{gitFile: migrationStatement})
			a.NoError(err)

			payload, err := json.Marshal(test.newWebhookPushEvent([]string{gitFile}, nil))
			a.NoError(err)
			err = ctl.vcsProvider.SendWebhookPush(test.externalID, payload)
			a.NoError(err)

			// Get schema update issue.
			openStatus := []api.IssueStatus{api.IssueOpen}
			issues, err := ctl.getIssues(
				api.IssueFind{
					ProjectID:  &project.ID,
					StatusList: openStatus,
				},
			)
			a.NoError(err)
			a.Len(issues, 1)
			issue := issues[0]
			status, err := ctl.waitIssuePipeline(issue.ID)
			a.NoError(err)
			a.Equal(api.TaskDone, status)
			issue, err = ctl.getIssue(issue.ID)
			a.NoError(err)
			a.Equal(api.TaskDatabaseSchemaUpdate, issue.Pipeline.StageList[0].TaskList[0].Type)
			a.Equal("[testVCSSchemaUpdate] Alter schema", issue.Name)
			a.Equal("By VCS files Prod/testVCSSchemaUpdate##ver1##migrate##create_a_test_table.sql", issue.Description)
			_, err = ctl.patchIssueStatus(
				api.IssueStatusPatch{
					ID:     issue.ID,
					Status: api.IssueDone,
				},
			)
			a.NoError(err)

			// Query schema.
			result, err := ctl.query(instance, databaseName, bookTableQuery)
			a.NoError(err)
			a.Equal(bookSchemaSQLResult, result)

			// Simulate Git commits for failed data update.
			gitFile = "bbtest/Prod/testVCSSchemaUpdate##ver2##data##insert_data.sql"
			err = ctl.vcsProvider.AddFiles(test.externalID, map[string]string{gitFile: dataUpdateStatementWrong})
			a.NoError(err)

			payload, err = json.Marshal(test.newWebhookPushEvent([]string{gitFile}, nil))
			a.NoError(err)
			err = ctl.vcsProvider.SendWebhookPush(test.externalID, payload)
			a.NoError(err)

			// Get data update issue.
			openStatus = []api.IssueStatus{api.IssueOpen}
			issues, err = ctl.getIssues(
				api.IssueFind{
					ProjectID:  &project.ID,
					StatusList: openStatus,
				},
			)
			a.NoError(err)
			a.Len(issues, 1)
			issue = issues[0]
			status, err = ctl.waitIssuePipeline(issue.ID)
			a.Error(err)
			a.Equal(api.TaskFailed, status)

			// Simulate Git commits for a correct modified date update.
			err = ctl.vcsProvider.AddFiles(test.externalID, map[string]string{gitFile: dataUpdateStatement})
			a.NoError(err)
			payload, err = json.Marshal(test.newWebhookPushEvent(nil, []string{gitFile}))
			a.NoError(err)
			err = ctl.vcsProvider.SendWebhookPush(test.externalID, payload)
			a.NoError(err)

			// Get data update issue.
			openStatus = []api.IssueStatus{api.IssueOpen}
			issues, err = ctl.getIssues(
				api.IssueFind{
					ProjectID:  &project.ID,
					StatusList: openStatus,
				},
			)
			a.NoError(err)
			a.Len(issues, 1)
			issue = issues[0]
			status, err = ctl.waitIssuePipeline(issue.ID)
			a.NoError(err)
			a.Equal(api.TaskDone, status)
			issue, err = ctl.getIssue(issue.ID)
			a.NoError(err)
			a.Equal(api.TaskDatabaseDataUpdate, issue.Pipeline.StageList[0].TaskList[0].Type)
			a.Equal("[testVCSSchemaUpdate] Change data", issue.Name)
			a.Equal("By VCS files Prod/testVCSSchemaUpdate##ver2##data##insert_data.sql", issue.Description)
			_, err = ctl.patchIssueStatus(
				api.IssueStatusPatch{
					ID:     issue.ID,
					Status: api.IssueDone,
				},
			)
			a.NoError(err)

			// Get migration history.
			histories, err := ctl.getInstanceMigrationHistory(db.MigrationHistoryFind{ID: &instance.ID})
			a.NoError(err)
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
					Type:       db.Migrate,
					Status:     db.Done,
					Schema:     "",
					SchemaPrev: "",
				},
			}
			a.Equal(len(wantHistories), len(histories))

			for i, history := range histories {
				got := api.MigrationHistory{
					Database:   history.Database,
					Source:     history.Source,
					Type:       history.Type,
					Status:     history.Status,
					Schema:     history.Schema,
					SchemaPrev: history.SchemaPrev,
				}
				a.Equal(wantHistories[i], got)
				a.NotEmpty(history.Version)
			}
			a.Equal(histories[0].Version, "ver2")
			a.Equal(histories[1].Version, "ver1")
		})
	}
}

func TestVCS_SDL(t *testing.T) {
	tests := []struct {
		name                string
		vcsProviderCreator  fake.VCSProviderCreator
		vcsType             vcs.Type
		externalID          string
		repositoryFullPath  string
		newWebhookPushEvent func(added []string, modified []string) interface{}
	}{
		{
			name:               "GitLab",
			vcsProviderCreator: fake.NewGitLab,
			vcsType:            vcs.GitLabSelfHost,
			externalID:         "121",
			repositoryFullPath: "test/schemaUpdate",
			newWebhookPushEvent: func(added []string, modified []string) interface{} {
				return gitlab.WebhookPushEvent{
					ObjectKind: gitlab.WebhookPush,
					Ref:        "refs/heads/feature/foo",
					Project: gitlab.WebhookProject{
						ID: 121,
					},
					CommitList: []gitlab.WebhookCommit{
						{
							Timestamp:    "2021-01-13T13:14:00Z",
							AddedList:    added,
							ModifiedList: modified,
						},
					},
				}
			},
		},
		{
			name:               "GitHub",
			vcsProviderCreator: fake.NewGitHub,
			vcsType:            vcs.GitHubCom,
			externalID:         "octocat/Hello-World",
			repositoryFullPath: "octocat/Hello-World",
			newWebhookPushEvent: func(added []string, modified []string) interface{} {
				return github.WebhookPushEvent{
					Ref: "refs/heads/feature/foo",
					Repository: github.WebhookRepository{
						ID:       211,
						FullName: "octocat/Hello-World",
						HTMLURL:  "https://github.com/octocat/Hello-World",
					},
					Sender: github.WebhookSender{
						Login: "fake_github_author",
					},
					Commits: []github.WebhookCommit{
						{
							ID:        "fake_github_commit_id",
							Distinct:  true,
							Message:   "Fake GitHub commit message",
							Timestamp: time.Now(),
							URL:       "https://api.github.com/octocat/Hello-World/commits/fake_github_commit_id",
							Author: github.WebhookCommitAuthor{
								Name:  "fake_github_author",
								Email: "fake_github_author@localhost",
							},
							Added:    added,
							Modified: modified,
						},
					},
				}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			a := require.New(t)
			ctx := context.Background()
			ctl := &controller{}
			err := ctl.StartServer(ctx, t.TempDir(), test.vcsProviderCreator, getTestPort(t.Name()))
			a.NoError(err)
			defer func() {
				_ = ctl.Close(ctx)
			}()

			err = ctl.Login()
			a.NoError(err)
			err = ctl.setLicense()
			a.NoError(err)

			// Create a PostgreSQL instance.
			port := getTestPort(t.Name()) + 3
			_, stopInstance := postgres.SetupTestInstance(t, port)
			defer stopInstance()

			pgDB, err := sql.Open("pgx", fmt.Sprintf("host=127.0.0.1 port=%d user=root database=postgres", port))
			a.NoError(err)
			defer func() {
				_ = pgDB.Close()
			}()

			err = pgDB.Ping()
			a.NoError(err)

			const databaseName = "testVCSSchemaUpdate"
			_, err = pgDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %v", databaseName))
			a.NoError(err)
			_, err = pgDB.Exec("CREATE USER bytebase WITH ENCRYPTED PASSWORD 'bytebase'")
			a.NoError(err)
			_, err = pgDB.Exec("ALTER USER bytebase WITH SUPERUSER")
			a.NoError(err)

			// Create a table in the database
			schemaFileContent := `CREATE TABLE projects (id serial PRIMARY KEY);`
			_, err = pgDB.Exec(schemaFileContent)
			a.NoError(err)

			// Create a VCS
			vcs, err := ctl.createVCS(
				api.VCSCreate{
					Name:          t.Name(),
					Type:          test.vcsType,
					InstanceURL:   ctl.vcsURL,
					APIURL:        ctl.vcsProvider.APIURL(ctl.vcsURL),
					ApplicationID: "testApplicationID",
					Secret:        "testApplicationSecret",
				},
			)
			a.NoError(err)

			// Create a project
			project, err := ctl.createProject(
				api.ProjectCreate{
					Name:             "Test VCS Project",
					Key:              "TestVCSSchemaUpdate",
					SchemaChangeType: api.ProjectSchemaChangeTypeSDL,
				},
			)
			a.NoError(err)

			// Create a repository
			ctl.vcsProvider.CreateRepository(test.externalID)
			_, err = ctl.createRepository(
				api.RepositoryCreate{
					VCSID:              vcs.ID,
					ProjectID:          project.ID,
					Name:               "Test Repository",
					FullPath:           test.repositoryFullPath,
					WebURL:             fmt.Sprintf("%s/%s", ctl.vcsURL, test.repositoryFullPath),
					BranchFilter:       "feature/foo",
					BaseDirectory:      baseDirectory,
					FilePathTemplate:   "{{ENV_NAME}}/{{DB_NAME}}##{{VERSION}}##{{TYPE}}##{{DESCRIPTION}}.sql",
					SchemaPathTemplate: "{{ENV_NAME}}/.{{DB_NAME}}##LATEST.sql",
					ExternalID:         test.externalID,
					AccessToken:        "accessToken1",
					RefreshToken:       "refreshToken1",
				},
			)
			a.NoError(err)

			environments, err := ctl.getEnvironments()
			a.NoError(err)
			prodEnvironment, err := findEnvironment(environments, "Prod")
			a.NoError(err)

			// Add an instance
			instance, err := ctl.addInstance(
				api.InstanceCreate{
					EnvironmentID: prodEnvironment.ID,
					Name:          "pgInstance",
					Engine:        db.Postgres,
					Host:          "127.0.0.1",
					Port:          strconv.Itoa(port),
					Username:      "bytebase",
					Password:      "bytebase",
				},
			)
			a.NoError(err)

			// Create an issue that creates a database
			err = ctl.createDatabase(project, instance, databaseName, "bytebase", nil /* labelMap */)
			a.NoError(err)

			// Simulate Git commits for schema update to create a new table "users".
			const schemaFile = "bbtest/Prod/.testVCSSchemaUpdate##LATEST.sql"
			schemaFileContent += "\nCREATE TABLE users (id serial PRIMARY KEY);"
			err = ctl.vcsProvider.AddFiles(test.externalID, map[string]string{
				schemaFile: schemaFileContent,
			})
			a.NoError(err)

			payload, err := json.Marshal(test.newWebhookPushEvent(nil /* added */, []string{schemaFile}))
			a.NoError(err)
			err = ctl.vcsProvider.SendWebhookPush(test.externalID, payload)
			a.NoError(err)

			// Get schema update issue
			openStatus := []api.IssueStatus{api.IssueOpen}
			issues, err := ctl.getIssues(
				api.IssueFind{
					ProjectID:  &project.ID,
					StatusList: openStatus,
				},
			)
			a.NoError(err)
			a.Len(issues, 1)
			issue := issues[0]
			status, err := ctl.waitIssuePipeline(issue.ID)
			a.NoError(err)
			a.Equal(api.TaskDone, status)
			issue, err = ctl.getIssue(issue.ID)
			a.NoError(err)
			a.Equal("[testVCSSchemaUpdate] Alter schema", issue.Name)
			a.Equal("Apply schema diff by file Prod/.testVCSSchemaUpdate##LATEST.sql", issue.Description)
			_, err = ctl.patchIssueStatus(
				api.IssueStatusPatch{
					ID:     issue.ID,
					Status: api.IssueDone,
				},
			)
			a.NoError(err)

			// Simulate Git commits for data update to the table "users".
			const dataFile = "bbtest/Prod/testVCSSchemaUpdate##ver2##data##insert_data.sql"
			err = ctl.vcsProvider.AddFiles(test.externalID, map[string]string{
				dataFile: `INSERT INTO users (id) VALUES (1);`,
			})
			a.NoError(err)

			payload, err = json.Marshal(test.newWebhookPushEvent([]string{dataFile}, nil /* modified */))
			a.NoError(err)
			err = ctl.vcsProvider.SendWebhookPush(test.externalID, payload)
			a.NoError(err)

			// Get data update issue
			openStatus = []api.IssueStatus{api.IssueOpen}
			issues, err = ctl.getIssues(
				api.IssueFind{
					ProjectID:  &project.ID,
					StatusList: openStatus,
				},
			)
			a.NoError(err)
			a.Len(issues, 1)
			issue = issues[0]
			status, err = ctl.waitIssuePipeline(issue.ID)
			a.NoError(err)
			a.Equal(api.TaskDone, status)
			issue, err = ctl.getIssue(issue.ID)
			a.NoError(err)
			a.Equal("[testVCSSchemaUpdate] Change data", issue.Name)
			a.Equal("By VCS files Prod/testVCSSchemaUpdate##ver2##data##insert_data.sql", issue.Description)
			_, err = ctl.patchIssueStatus(
				api.IssueStatusPatch{
					ID:     issue.ID,
					Status: api.IssueDone,
				},
			)
			a.NoError(err)

			// Query list of tables
			result, err := ctl.query(instance, databaseName, `
SELECT table_name 
    FROM information_schema.tables 
WHERE table_type = 'BASE TABLE' 
    AND table_schema NOT IN 
        ('pg_catalog', 'information_schema');
`)
			a.NoError(err)
			a.Equal(`[["table_name"],["NAME"],[["projects"],["users"]]]`, result)

			// Get migration history
			const initialSchema = `SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

`
			const updatedSchema = `SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

SET default_tablespace = '';

SET default_table_access_method = heap;

CREATE TABLE public.projects (
    id integer NOT NULL
);

CREATE SEQUENCE public.projects_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.projects_id_seq OWNED BY public.projects.id;

CREATE TABLE public.users (
    id integer NOT NULL
);

CREATE SEQUENCE public.users_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.users_id_seq OWNED BY public.users.id;

ALTER TABLE ONLY public.projects ALTER COLUMN id SET DEFAULT nextval('public.projects_id_seq'::regclass);

ALTER TABLE ONLY public.users ALTER COLUMN id SET DEFAULT nextval('public.users_id_seq'::regclass);

ALTER TABLE ONLY public.projects
    ADD CONSTRAINT projects_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);

`

			histories, err := ctl.getInstanceMigrationHistory(db.MigrationHistoryFind{ID: &instance.ID})
			a.NoError(err)
			wantHistories := []api.MigrationHistory{
				{
					Database:   databaseName,
					Source:     db.VCS,
					Type:       db.Data,
					Status:     db.Done,
					Schema:     updatedSchema,
					SchemaPrev: updatedSchema,
				},
				{
					Database:   databaseName,
					Source:     db.VCS,
					Type:       db.Migrate,
					Status:     db.Done,
					Schema:     updatedSchema,
					SchemaPrev: initialSchema,
				},
				{
					Database:   databaseName,
					Source:     db.UI,
					Type:       db.Migrate,
					Status:     db.Done,
					Schema:     initialSchema,
					SchemaPrev: "",
				},
			}
			a.Equal(len(wantHistories), len(histories))

			for i, history := range histories {
				got := api.MigrationHistory{
					Database:   history.Database,
					Source:     history.Source,
					Type:       history.Type,
					Status:     history.Status,
					Schema:     history.Schema,
					SchemaPrev: history.SchemaPrev,
				}
				a.Equal(wantHistories[i], got, i)
				a.NotEmpty(history.Version)
			}
		})
	}
}

func TestWildcardInVCSFilePathTemplate(t *testing.T) {
	branchFilter := "feature/foo"
	dbName := "db1"
	externalID := "121"
	repoFullPath := "test/wildcard"

	tests := []struct {
		name                string
		vcsProviderCreator  fake.VCSProviderCreator
		vcsType             vcs.Type
		baseDirectory       string
		envName             string
		filePathTemplate    string
		commitFileNames     []string
		commitContents      []string
		expect              []bool
		newWebhookPushEvent func(gitFile string) interface{}
	}{
		{
			name:               "singleAsterisk",
			vcsProviderCreator: fake.NewGitLab,
			vcsType:            vcs.GitLabSelfHost,
			baseDirectory:      "bbtest",
			envName:            "wildcard",
			filePathTemplate:   "{{ENV_NAME}}/*/{{DB_NAME}}##{{VERSION}}##{{TYPE}}##{{DESCRIPTION}}.sql",
			commitFileNames: []string{
				// Normal
				fmt.Sprintf("%s/%s/foo/%s##ver1##migrate##create_table_t1.sql", baseDirectory, "wildcard", dbName),
				// One singleAsterisk cannot match two directories.
				fmt.Sprintf("%s/%s/foo/bar/%s##ver2##data##insert_data.sql", baseDirectory, "wildcard", dbName),
				// One singleAsterisk cannot match zero directory.
				fmt.Sprintf("%s/%s/%s##ver3##migrate##create_table_t3.sql", baseDirectory, "wildcard", dbName),
			},
			commitContents: []string{
				"CREATE TABLE t1 (id INT);",
				"INSERT INTO t1 VALUES (1);",
				"CREATE TABLE t3 (id INT);",
			},
			expect: []bool{
				true,
				false,
				false,
			},
			newWebhookPushEvent: func(gitFile string) interface{} {
				return gitlab.WebhookPushEvent{
					ObjectKind: gitlab.WebhookPush,
					Ref:        fmt.Sprintf("refs/heads/%s", branchFilter),
					Project: gitlab.WebhookProject{
						ID: 121,
					},
					CommitList: []gitlab.WebhookCommit{
						{
							ID:        "68211f18905c46e8bda58a8fee98051f2ffe40bb",
							Timestamp: "2021-01-13T13:14:00Z",
							AddedList: []string{gitFile},
						},
					},
				}
			},
		},
		{
			name:               "doubleAsterisks",
			vcsProviderCreator: fake.NewGitLab,
			vcsType:            vcs.GitLabSelfHost,
			baseDirectory:      "bbtest",
			envName:            "wildcard",
			filePathTemplate:   "{{ENV_NAME}}/**/{{DB_NAME}}##{{VERSION}}##{{TYPE}}##{{DESCRIPTION}}.sql",
			commitFileNames: []string{
				// Two singleAsterisk can match one directory.
				fmt.Sprintf("%s/%s/foo/%s##ver1##migrate##create_table_t1.sql", baseDirectory, "wildcard", dbName),
				// Two singleAsterisk can match two directories.
				fmt.Sprintf("%s/%s/foo/bar/%s##ver2##migrate##create_table_t2.sql", baseDirectory, "wildcard", dbName),
				// Two singleAsterisk can match three directories or more.
				fmt.Sprintf("%s/%s/foo/bar/foo/%s##ver3##migrate##create_table_t3.sql", baseDirectory, "wildcard", dbName),
				// Two singleAsterisk cannot match zero directory.
				fmt.Sprintf("%s/%s/%s##ver4##migrate##create_table_t4.sql", baseDirectory, "wildcard", dbName),
			},
			commitContents: []string{
				"CREATE TABLE t1 (id INT);",
				"CREATE TABLE t2 (id INT);",
				"CREATE TABLE t3 (id INT);",
				"CREATE TABLE t4 (id INT);",
			},
			expect: []bool{
				true,
				true,
				true,
				false,
			},
			newWebhookPushEvent: func(gitFile string) interface{} {
				return gitlab.WebhookPushEvent{
					ObjectKind: gitlab.WebhookPush,
					Ref:        fmt.Sprintf("refs/heads/%s", branchFilter),
					Project: gitlab.WebhookProject{
						ID: 121,
					},
					CommitList: []gitlab.WebhookCommit{
						{
							ID:        "68211f18905c46e8bda58a8fee98051f2ffe40bb",
							Timestamp: "2021-01-13T13:14:00Z",
							AddedList: []string{gitFile},
						},
					},
				}
			},
		},
		{
			name:               "emptyBaseAndMixAsterisks",
			vcsProviderCreator: fake.NewGitLab,
			envName:            "wildcard",
			baseDirectory:      "",
			vcsType:            vcs.GitLabSelfHost,
			filePathTemplate:   "{{ENV_NAME}}/**/foo/*/{{DB_NAME}}##{{VERSION}}##{{TYPE}}##{{DESCRIPTION}}.sql",
			commitFileNames: []string{
				// ** matches foo, foo matches foo, * matches bar
				fmt.Sprintf("%s/foo/foo/bar/%s##ver1##migrate##create_table_t1.sql", "wildcard", dbName),
				// ** matches foo/bar/foo, foo matches foo, * matches bar
				fmt.Sprintf("%s/foo/bar/foo/foo/bar/%s##ver2##migrate##create_table_t2.sql", "wildcard", dbName),
				// cannot match
				fmt.Sprintf("%s/%s##ver3##migrate##create_table_t3.sql", "wildcard", dbName),
			},
			commitContents: []string{
				"CREATE TABLE t1 (id INT);",
				"CREATE TABLE t2 (id INT);",
				"CREATE TABLE t3 (id INT);",
			},
			expect: []bool{
				true,
				true,
				false,
			},
			newWebhookPushEvent: func(gitFile string) interface{} {
				return gitlab.WebhookPushEvent{
					ObjectKind: gitlab.WebhookPush,
					Ref:        fmt.Sprintf("refs/heads/%s", branchFilter),
					Project: gitlab.WebhookProject{
						ID: 121,
					},
					CommitList: []gitlab.WebhookCommit{
						{
							ID:        "68211f18905c46e8bda58a8fee98051f2ffe40bb",
							Timestamp: "2021-01-13T13:14:00Z",
							AddedList: []string{gitFile},
						},
					},
				}
			},
		},
		// We test the combination of ** and *, and the place holder is not fully represented by the ascii character set.
		{
			name:               "mixAsterisks",
			vcsProviderCreator: fake.NewGitLab,
			envName:            "生产",
			baseDirectory:      "bbtest",
			vcsType:            vcs.GitLabSelfHost,
			filePathTemplate:   "{{ENV_NAME}}/**/foo/*/{{DB_NAME}}##{{VERSION}}##{{TYPE}}##{{DESCRIPTION}}.sql",
			commitFileNames: []string{
				// ** matches foo, foo matches foo, * matches bar
				fmt.Sprintf("%s/%s/foo/foo/bar/%s##ver1##migrate##create_table_t1.sql", baseDirectory, "生产", dbName),
				// ** matches foo/bar/foo, foo matches foo, * matches bar
				fmt.Sprintf("%s/%s/foo/bar/foo/foo/bar/%s##ver2##migrate##create_table_t2.sql", baseDirectory, "生产", dbName),
				// cannot match
				fmt.Sprintf("%s/%s/%s##ver3##migrate##create_table_t3.sql", baseDirectory, "生产", dbName),
			},
			commitContents: []string{
				"CREATE TABLE t1 (id INT);",
				"CREATE TABLE t2 (id INT);",
				"CREATE TABLE t3 (id INT);",
			},
			expect: []bool{
				true,
				true,
				false,
			},
			newWebhookPushEvent: func(gitFile string) interface{} {
				return gitlab.WebhookPushEvent{
					ObjectKind: gitlab.WebhookPush,
					Ref:        fmt.Sprintf("refs/heads/%s", branchFilter),
					Project: gitlab.WebhookProject{
						ID: 121,
					},
					CommitList: []gitlab.WebhookCommit{
						{
							ID:        "68211f18905c46e8bda58a8fee98051f2ffe40bb",
							Timestamp: "2021-01-13T13:14:00Z",
							AddedList: []string{gitFile},
						},
					},
				}
			},
		},
		// No asterisk
		{
			name:               "placeholderAsFolder",
			vcsProviderCreator: fake.NewGitLab,
			envName:            "ZO",
			baseDirectory:      "bbtest",
			vcsType:            vcs.GitLabSelfHost,
			filePathTemplate:   "{{ENV_NAME}}/{{DB_NAME}}/sql/{{DB_NAME}}##{{VERSION}}##{{TYPE}}##{{DESCRIPTION}}.sql",
			commitFileNames: []string{
				fmt.Sprintf("%s/%s/%s/sql/%s##ver1##migrate##create_table_t1.sql", baseDirectory, "ZO", dbName, dbName),
				fmt.Sprintf("%s/%s/%s/%s##ver2##migrate##create_table_t2.sql", baseDirectory, "ZO", dbName, dbName),
				fmt.Sprintf("%s/%s/%s/sql/%s##ver3##migrate##create_table_t3.sql", baseDirectory, "ZO", dbName, dbName),
			},
			commitContents: []string{
				"CREATE TABLE t1 (id INT);",
				"CREATE TABLE t2 (id INT);",
				"CREATE TABLE t3 (id INT);",
			},
			expect: []bool{
				true,
				false,
				true,
			},
			newWebhookPushEvent: func(gitFile string) interface{} {
				return gitlab.WebhookPushEvent{
					ObjectKind: gitlab.WebhookPush,
					Ref:        fmt.Sprintf("refs/heads/%s", branchFilter),
					Project: gitlab.WebhookProject{
						ID: 121,
					},
					CommitList: []gitlab.WebhookCommit{
						{
							ID:        "68211f18905c46e8bda58a8fee98051f2ffe40bb",
							Timestamp: "2021-01-13T13:14:00Z",
							AddedList: []string{gitFile},
						},
					},
				}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			port := getTestPort(t.Name())
			a := require.New(t)
			ctx := context.Background()
			ctl := &controller{}
			err := ctl.StartServer(ctx, t.TempDir(), test.vcsProviderCreator, port)
			a.NoError(err)
			defer func() {
				_ = ctl.Close(ctx)
			}()

			err = ctl.Login()
			a.NoError(err)
			err = ctl.setLicense()
			a.NoError(err)

			// Create a VCS.
			vcs, err := ctl.createVCS(
				api.VCSCreate{
					Name:          t.Name(),
					Type:          test.vcsType,
					InstanceURL:   ctl.vcsURL,
					APIURL:        ctl.vcsProvider.APIURL(ctl.vcsURL),
					ApplicationID: "testApplicationID",
					Secret:        "testApplicationSecret",
				},
			)
			a.NoError(err)

			// Create a project.
			project, err := ctl.createProject(
				api.ProjectCreate{
					Name: "Test VCS Project",
					Key:  "TVP",
				},
			)
			a.NoError(err)

			// Create a repository.
			ctl.vcsProvider.CreateRepository(externalID)
			_, err = ctl.createRepository(
				api.RepositoryCreate{
					VCSID:              vcs.ID,
					ProjectID:          project.ID,
					Name:               "Test Repository",
					FullPath:           repoFullPath,
					WebURL:             fmt.Sprintf("%s/%s", ctl.vcsURL, repoFullPath),
					BranchFilter:       branchFilter,
					BaseDirectory:      baseDirectory,
					FilePathTemplate:   test.filePathTemplate,
					SchemaPathTemplate: "{{ENV_NAME}}/.{{DB_NAME}}##LATEST.sql",
					ExternalID:         externalID,
					AccessToken:        "accessToken1",
					RefreshToken:       "refreshToken1",
				},
			)
			a.NoError(err)

			environment, err := ctl.createEnvrionment(api.EnvironmentCreate{
				Name: test.envName,
			})
			a.NoError(err)
			// Provision an instance.
			instanceRootDir := t.TempDir()
			instanceName := "testInstance1"
			instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, instanceName)
			a.NoError(err)
			instance, err := ctl.addInstance(api.InstanceCreate{
				EnvironmentID: environment.ID,
				Name:          instanceName,
				Engine:        db.SQLite,
				Host:          instanceDir,
			})
			a.NoError(err)

			// Create an issue that creates a database.
			err = ctl.createDatabase(project, instance, dbName, "", nil /* labelMap */)
			a.NoError(err)

			a.Equal(len(test.expect), len(test.commitFileNames))
			a.Equal(len(test.expect), len(test.commitContents))
			for idx, commitFileName := range test.commitFileNames {
				// Simulate Git commits for schema update.
				err = ctl.vcsProvider.AddFiles(externalID, map[string]string{commitFileName: test.commitContents[idx]})
				a.NoError(err)

				payload, err := json.Marshal(test.newWebhookPushEvent(commitFileName))
				a.NoError(err)
				err = ctl.vcsProvider.SendWebhookPush(externalID, payload)
				a.NoError(err)

				// Check for newly generated issues.
				openStatus := []api.IssueStatus{api.IssueOpen}
				issues, err := ctl.getIssues(
					api.IssueFind{
						ProjectID:  &project.ID,
						StatusList: openStatus,
					},
				)
				a.NoError(err)
				if test.expect[idx] {
					a.Len(issues, 1)
					issue := issues[0]
					status, err := ctl.waitIssuePipeline(issue.ID)
					a.NoError(err)
					a.Equal(api.TaskDone, status)
					_, err = ctl.patchIssueStatus(
						api.IssueStatusPatch{
							ID:     issue.ID,
							Status: api.IssueDone,
						},
					)
					a.NoError(err)
				} else {
					a.Len(issues, 0)
				}
			}
		})
	}
}
