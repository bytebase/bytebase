package tests

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/testing/protocmp"

	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/vcs"
	"github.com/bytebase/bytebase/backend/plugin/vcs/bitbucket"
	"github.com/bytebase/bytebase/backend/plugin/vcs/github"
	"github.com/bytebase/bytebase/backend/plugin/vcs/gitlab"
	"github.com/bytebase/bytebase/backend/resources/mysql"
	"github.com/bytebase/bytebase/backend/resources/postgres"
	"github.com/bytebase/bytebase/backend/tests/fake"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func TestSchemaAndDataUpdate(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	dataDir := t.TempDir()
	err := ctl.StartServerWithExternalPg(ctx, &config{
		dataDir:            dataDir,
		vcsProviderCreator: fake.NewGitLab,
	})
	a.NoError(err)
	defer ctl.Close(ctx)

	// Create a project.
	project, err := ctl.createProject(api.ProjectCreate{
		ResourceID: generateRandomString("project", 10),
		Name:       "Test Project",
		Key:        "TestSchemaUpdate",
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
		ResourceID:    generateRandomString("instance", 10),
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

	migrationStatementSheet, err := ctl.createSheet(api.SheetCreate{
		ProjectID:  project.ID,
		Name:       "migration statement sheet",
		Statement:  migrationStatement,
		Visibility: api.ProjectSheet,
		Source:     api.SheetFromBytebaseArtifact,
		Type:       api.SheetForSQL,
	})
	a.NoError(err)

	// Create an issue that updates database schema.
	createContext, err := json.Marshal(&api.MigrationContext{
		DetailList: []*api.MigrationDetail{
			{
				MigrationType: db.Migrate,
				DatabaseID:    database.ID,
				SheetID:       migrationStatementSheet.ID,
			},
		},
	})
	a.NoError(err)
	issue, err := ctl.createIssue(api.IssueCreate{
		ProjectID:     project.ID,
		Name:          fmt.Sprintf("update schema for database %q", databaseName),
		Type:          api.IssueDatabaseSchemaUpdate,
		Description:   fmt.Sprintf("This updates the schema of database %q.", databaseName),
		AssigneeID:    api.SystemBotID,
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

	dataUpdateStatementSheet, err := ctl.createSheet(api.SheetCreate{
		ProjectID:  project.ID,
		Name:       "dataUpdateStatement",
		Statement:  dataUpdateStatement,
		Visibility: api.ProjectSheet,
		Source:     api.SheetFromBytebaseArtifact,
		Type:       api.SheetForSQL,
	})
	a.NoError(err)

	// Create an issue that updates database data.
	createContext, err = json.Marshal(&api.MigrationContext{
		DetailList: []*api.MigrationDetail{
			{
				MigrationType: db.Data,
				DatabaseID:    database.ID,
				SheetID:       dataUpdateStatementSheet.ID,
			},
		},
	})
	a.NoError(err)
	issue, err = ctl.createIssue(api.IssueCreate{
		ProjectID:     project.ID,
		Name:          fmt.Sprintf("update data for database %q", databaseName),
		Type:          api.IssueDatabaseDataUpdate,
		Description:   fmt.Sprintf("This updates the data of database %q.", databaseName),
		AssigneeID:    api.SystemBotID,
		CreateContext: string(createContext),
	})
	a.NoError(err)
	status, err = ctl.waitIssuePipeline(issue.ID)
	a.NoError(err)
	a.Equal(api.TaskDone, status)

	// Get migration history.
	histories, err := ctl.getInstanceMigrationHistory(instance.ID, db.MigrationHistoryFind{})
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
	histories, err = ctl.getInstanceMigrationHistory(instance.ID, db.MigrationHistoryFind{Database: &cloneDatabaseName})
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
	}
	a.Equal(len(wantCloneHistories), len(histories))
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
		Source:     api.SheetFromBytebase,
	})
	a.NoError(err)

	_, err = ctl.listMySheets()
	a.NoError(err)
}

func TestVCS(t *testing.T) {
	tests := []struct {
		name                string
		vcsProviderCreator  fake.VCSProviderCreator
		vcsType             vcs.Type
		externalID          string
		repositoryFullPath  string
		newWebhookPushEvent func(added, modified [][]string, beforeSHA, afterSHA string) any
	}{
		{
			name:               "GitLab",
			vcsProviderCreator: fake.NewGitLab,
			vcsType:            vcs.GitLab,
			externalID:         "121",
			repositoryFullPath: "test/schemaUpdate",
			newWebhookPushEvent: func(added, modified [][]string, beforeSHA, afterSHA string) any {
				var commitList []gitlab.WebhookCommit
				for i := range added {
					commitList = append(commitList, gitlab.WebhookCommit{
						Timestamp:    time.Now().Format(time.RFC3339),
						AddedList:    added[i],
						ModifiedList: modified[i],
					})
				}
				return gitlab.WebhookPushEvent{
					ObjectKind: gitlab.WebhookPush,
					Ref:        "refs/heads/feature/foo",
					Before:     beforeSHA,
					After:      afterSHA,
					Project: gitlab.WebhookProject{
						ID: 121,
					},
					CommitList: commitList,
				}
			},
		},
		{
			name:               "GitHub",
			vcsProviderCreator: fake.NewGitHub,
			vcsType:            vcs.GitHub,
			externalID:         "octocat/Hello-World",
			repositoryFullPath: "octocat/Hello-World",
			newWebhookPushEvent: func(added, modified [][]string, beforeSHA, afterSHA string) any {
				var commits []github.WebhookCommit
				for i := range added {
					commits = append(commits, github.WebhookCommit{
						ID:        "fake_github_commit_id",
						Distinct:  true,
						Message:   "Fake GitHub commit message",
						Timestamp: time.Now(),
						URL:       "https://api.github.com/octocat/Hello-World/commits/fake_github_commit_id",
						Author: github.WebhookCommitAuthor{
							Name:  "fake_github_author",
							Email: "fake_github_author@localhost",
						},
						Added:    added[i],
						Modified: modified[i],
					})
				}
				return github.WebhookPushEvent{
					Ref:    "refs/heads/feature/foo",
					Before: beforeSHA,
					After:  afterSHA,
					Repository: github.WebhookRepository{
						ID:       211,
						FullName: "octocat/Hello-World",
						HTMLURL:  "https://github.com/octocat/Hello-World",
					},
					Sender: github.WebhookSender{
						Login: "fake_github_author",
					},
					Commits: commits,
				}
			},
		},
		{
			name:               "Bitbucket",
			vcsProviderCreator: fake.NewBitbucket,
			vcsType:            vcs.Bitbucket,
			externalID:         "octocat/Hello-World",
			repositoryFullPath: "octocat/Hello-World",
			newWebhookPushEvent: func(added, _ [][]string, beforeSHA, afterSHA string) any {
				var commits []bitbucket.WebhookCommit
				for range added {
					commits = append(commits, bitbucket.WebhookCommit{
						Hash: afterSHA,
						Date: time.Now(),
						Author: bitbucket.Author{
							Raw: "fake_bitbucket_author <fake_bitbucket_author@localhost>",
							User: bitbucket.User{
								Nickname: "fake_bitbucket_author",
							},
						},
						Message: "Fake Bitbucket commit message",
						Links: bitbucket.Links{
							HTML: bitbucket.Link{
								Href: "https://bitbucket.org/octocat/Hello-World/commits/fake_github_commit_id",
							},
						},
						Parents: []bitbucket.Target{
							{Hash: beforeSHA},
						},
					})
				}
				return bitbucket.WebhookPushEvent{
					Push: bitbucket.WebhookPush{
						Changes: []bitbucket.WebhookPushChange{
							{
								Old: bitbucket.Branch{
									Name: "feature/foo",
									Target: bitbucket.Target{
										Hash: beforeSHA,
									},
								},
								New: bitbucket.Branch{
									Name: "feature/foo",
									Target: bitbucket.Target{
										Hash: afterSHA,
									},
								},
								Commits: commits,
							},
						},
					},
					Repository: bitbucket.Repository{
						FullName: "octocat/Hello-World",
						Links: bitbucket.Links{
							HTML: bitbucket.Link{
								Href: "https://bitbuket.org/octocat/Hello-World",
							},
						},
					},
					Actor: bitbucket.User{
						Nickname: "fake_bitbucket_author",
					},
				}
			},
		},
	}
	for _, test := range tests {
		// Fix the problem that closure in a for loop will always use the last element.
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			a := require.New(t)
			ctx := context.Background()
			ctl := &controller{}
			err := ctl.StartServerWithExternalPg(ctx, &config{
				dataDir:            t.TempDir(),
				vcsProviderCreator: test.vcsProviderCreator,
			})
			a.NoError(err)
			defer func() {
				_ = ctl.Close(ctx)
			}()
			err = ctl.setLicense()
			a.NoError(err)

			// Create a VCS.
			apiVCS, err := ctl.createVCS(
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
					ResourceID: generateRandomString("project", 10),
					Name:       "Test VCS Project",
					Key:        "TestVCSSchemaUpdate",
				},
			)
			a.NoError(err)

			// Create a repository.
			ctl.vcsProvider.CreateRepository(test.externalID)

			// Create the branch
			err = ctl.vcsProvider.CreateBranch(test.externalID, "feature/foo")
			a.NoError(err)

			_, err = ctl.createRepository(
				api.RepositoryCreate{
					VCSID:              apiVCS.ID,
					ProjectID:          project.ID,
					Name:               "Test Repository",
					FullPath:           test.repositoryFullPath,
					WebURL:             fmt.Sprintf("%s/%s", ctl.vcsURL, test.repositoryFullPath),
					BranchFilter:       "feature/foo",
					BaseDirectory:      baseDirectory,
					FilePathTemplate:   "{{ENV_ID}}/{{DB_NAME}}##{{VERSION}}##{{TYPE}}##{{DESCRIPTION}}.sql",
					SchemaPathTemplate: "{{ENV_ID}}/.{{DB_NAME}}##LATEST.sql",
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
				ResourceID:    generateRandomString("instance", 10),
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
			// Expecting project to have 1 database.
			databases, err := ctl.getDatabases(api.DatabaseFind{
				ProjectID: &project.ID,
			})
			a.NoError(err)
			a.Equal(1, len(databases))
			database := databases[0]
			a.Equal(instance.ID, database.Instance.ID)

			// Simulate Git commits for schema update.
			// We create multiple commits in one push event to test for the behavior of creating one issue per database.
			gitFile3 := "bbtest/prod/testVCSSchemaUpdate##ver3##migrate##create_table_book3.sql"
			err = ctl.vcsProvider.AddFiles(test.externalID, map[string]string{gitFile3: migrationStatement3})
			a.NoError(err)
			gitFile2 := "bbtest/prod/testVCSSchemaUpdate##ver2##migrate##create_table_book2.sql"
			err = ctl.vcsProvider.AddFiles(test.externalID, map[string]string{gitFile2: migrationStatement2})
			a.NoError(err)
			gitFile1 := "bbtest/prod/testVCSSchemaUpdate##ver1##migrate##create_table_book.sql"
			err = ctl.vcsProvider.AddFiles(test.externalID, map[string]string{gitFile1: migrationStatement})
			a.NoError(err)
			// This file is merged from other branch and included in this push event's commits.
			// But it is already merged into the main branch and the commits diff does not contain it.
			// So this file should be excluded when generating the issue.
			gitFileMergeFromOtherBranch := "bbtest/prod/testVCSSchemaUpdate##ver0##migrate##merge_from_other_branch.sql"
			err = ctl.vcsProvider.AddFiles(test.externalID, map[string]string{gitFileMergeFromOtherBranch: "SELECT 1;"})
			a.NoError(err)
			err = ctl.vcsProvider.AddCommitsDiff(test.externalID, "1", "2", []vcs.FileDiff{
				{Path: gitFile1, Type: vcs.FileDiffTypeAdded},
				{Path: gitFile2, Type: vcs.FileDiffTypeAdded},
				{Path: gitFile3, Type: vcs.FileDiffTypeAdded},
			})
			a.NoError(err)

			payload, err := json.Marshal(test.newWebhookPushEvent([][]string{{gitFile1}, {gitFile2}, {gitFile3}, {gitFileMergeFromOtherBranch}}, [][]string{nil, nil, nil, nil}, "1", "2"))
			a.NoError(err)
			err = ctl.vcsProvider.SendWebhookPush(test.externalID, payload)
			a.NoError(err)

			// Get schema update issue.
			issues, err := ctl.getIssues(&project.ID, api.IssueOpen)
			a.NoError(err)
			a.Len(issues, 1)
			issue := issues[0]
			status, err := ctl.waitIssuePipeline(issue.ID)
			a.NoError(err)
			a.Equal(api.TaskDone, status)
			issue, err = ctl.getIssue(issue.ID)
			a.NoError(err)
			// TODO(p0ny): expose task DAG list and check the dependency.
			a.Equal(3, len(issue.Pipeline.StageList[0].TaskList))
			a.Equal(api.TaskDatabaseSchemaUpdate, issue.Pipeline.StageList[0].TaskList[0].Type)
			a.Equal("[testVCSSchemaUpdate] Alter schema", issue.Name)
			a.Equal("By VCS files:\n\nprod/testVCSSchemaUpdate##ver1##migrate##create_table_book.sql\nprod/testVCSSchemaUpdate##ver2##migrate##create_table_book2.sql\nprod/testVCSSchemaUpdate##ver3##migrate##create_table_book3.sql\n", issue.Description)
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
			gitFile4 := "bbtest/prod/testVCSSchemaUpdate##ver4##data##insert_data.sql"
			err = ctl.vcsProvider.AddFiles(test.externalID, map[string]string{gitFile4: dataUpdateStatementWrong})
			a.NoError(err)
			err = ctl.vcsProvider.AddCommitsDiff(test.externalID, "2", "3", []vcs.FileDiff{
				{Path: gitFile4, Type: vcs.FileDiffTypeAdded},
			})
			a.NoError(err)
			payload, err = json.Marshal(test.newWebhookPushEvent([][]string{{gitFile4}}, [][]string{nil}, "2", "3"))
			a.NoError(err)
			err = ctl.vcsProvider.SendWebhookPush(test.externalID, payload)
			a.NoError(err)

			// Get data update issue.
			issues, err = ctl.getIssues(&project.ID, api.IssueOpen)
			a.NoError(err)
			a.Len(issues, 1)
			issue = issues[0]
			status, err = ctl.waitIssuePipeline(issue.ID)
			a.Error(err)
			a.Equal(api.TaskFailed, status)

			// Simulate Git commits for a correct modified date update.
			err = ctl.vcsProvider.AddFiles(test.externalID, map[string]string{gitFile4: dataUpdateStatement})
			a.NoError(err)
			err = ctl.vcsProvider.AddCommitsDiff(test.externalID, "3", "4", []vcs.FileDiff{
				{Path: gitFile4, Type: vcs.FileDiffTypeModified},
			})
			a.NoError(err)
			payload, err = json.Marshal(test.newWebhookPushEvent([][]string{nil}, [][]string{{gitFile4}}, "3", "4"))
			a.NoError(err)
			err = ctl.vcsProvider.SendWebhookPush(test.externalID, payload)
			a.NoError(err)

			// Get data update issue.
			issues, err = ctl.getIssues(&project.ID, api.IssueOpen)
			a.NoError(err)
			a.Len(issues, 1)
			issue = issues[0]

			a.Len(issue.Pipeline.StageList, 1)
			stage := issue.Pipeline.StageList[0]
			a.Len(stage.TaskList, 1)
			task := stage.TaskList[0]
			// simulate retrying the failed task.
			_, err = ctl.patchTaskStatus(api.TaskStatusPatch{
				ID:        task.ID,
				UpdaterID: api.SystemBotID,
				Status:    api.TaskPendingApproval,
			}, issue.PipelineID, task.ID)
			a.NoError(err)

			status, err = ctl.waitIssuePipeline(issue.ID)
			a.NoError(err)
			a.Equal(api.TaskDone, status)
			issue, err = ctl.getIssue(issue.ID)
			a.NoError(err)
			a.Equal(api.TaskDatabaseDataUpdate, issue.Pipeline.StageList[0].TaskList[0].Type)
			a.Equal("[testVCSSchemaUpdate] Change data", issue.Name)
			a.Equal("By VCS files:\n\nprod/testVCSSchemaUpdate##ver4##data##insert_data.sql\n", issue.Description)
			_, err = ctl.patchIssueStatus(
				api.IssueStatusPatch{
					ID:     issue.ID,
					Status: api.IssueDone,
				},
			)
			a.NoError(err)

			sheet, err := ctl.createSheet(api.SheetCreate{
				ProjectID:  project.ID,
				Name:       "migration statement 4 sheet",
				Statement:  migrationStatement4,
				Visibility: api.ProjectSheet,
				Source:     api.SheetFromBytebaseArtifact,
				Type:       api.SheetForSQL,
			})
			a.NoError(err)

			// Schema change from UI.
			// Create an issue that updates database schema.
			createContext, err := json.Marshal(&api.MigrationContext{
				DetailList: []*api.MigrationDetail{
					{
						MigrationType: db.Migrate,
						DatabaseID:    database.ID,
						SheetID:       sheet.ID,
					},
				},
			})
			a.NoError(err)
			issue, err = ctl.createIssue(api.IssueCreate{
				ProjectID:     project.ID,
				Name:          fmt.Sprintf("update schema for database %q", databaseName),
				Type:          api.IssueDatabaseSchemaUpdate,
				Description:   fmt.Sprintf("This updates the schema of database %q.", databaseName),
				AssigneeID:    api.SystemBotID,
				CreateContext: string(createContext),
			})
			a.NoError(err)
			status, err = ctl.waitIssuePipeline(issue.ID)
			a.NoError(err)
			a.Equal(api.TaskDone, status)
			latestFileName := fmt.Sprintf("%s/%s/.%s##LATEST.sql", baseDirectory, prodEnvironment.ResourceID, database.Name)
			files, err := ctl.vcsProvider.GetFiles(test.externalID, latestFileName)
			a.NoError(err)
			a.Len(files, 1)
			a.Equal(dumpedSchema4, files[latestFileName])

			// Get migration history.
			histories, err := ctl.getInstanceMigrationHistory(instance.ID, db.MigrationHistoryFind{})
			a.NoError(err)

			var historiesDeref []api.MigrationHistory
			for _, history := range histories {
				historiesDeref = append(historiesDeref, *history)
			}

			wantHistories := []api.MigrationHistory{
				{
					Database:   databaseName,
					Source:     db.UI,
					Type:       db.Migrate,
					Status:     db.Done,
					Schema:     dumpedSchema4,
					SchemaPrev: dumpedSchema3,
				},
				{
					Database:   databaseName,
					Source:     db.VCS,
					Type:       db.Data,
					Status:     db.Done,
					Schema:     dumpedSchema3,
					SchemaPrev: dumpedSchema3,
				},
				{
					Database:   databaseName,
					Source:     db.VCS,
					Type:       db.Migrate,
					Status:     db.Done,
					Schema:     dumpedSchema3,
					SchemaPrev: dumpedSchema2,
				},
				{
					Database:   databaseName,
					Source:     db.VCS,
					Type:       db.Migrate,
					Status:     db.Done,
					Schema:     dumpedSchema2,
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
				a.Equalf(wantHistories[i], got, "got histories %+v", historiesDeref)
				a.NotEmpty(history.Version)
			}
			a.Equal("ver4-dml", histories[1].Version)
			a.Equal("ver3-ddl", histories[2].Version)
			a.Equal("ver2-ddl", histories[3].Version)
			a.Equal("ver1-ddl", histories[4].Version)
		})
	}
}

func TestVCS_SDL(t *testing.T) {
	// TODO(rebelice): remove skip when support PostgreSQL SDL.
	t.Skip()
	tests := []struct {
		name                string
		vcsProviderCreator  fake.VCSProviderCreator
		vcsType             vcs.Type
		externalID          string
		repositoryFullPath  string
		newWebhookPushEvent func(added, modified []string, beforeSHA, afterSHA string) any
	}{
		{
			name:               "GitLab",
			vcsProviderCreator: fake.NewGitLab,
			vcsType:            vcs.GitLab,
			externalID:         "121",
			repositoryFullPath: "test/schemaUpdate",
			newWebhookPushEvent: func(added, modified []string, beforeSHA, afterSHA string) any {
				return gitlab.WebhookPushEvent{
					ObjectKind: gitlab.WebhookPush,
					Ref:        "refs/heads/feature/foo",
					Before:     beforeSHA,
					After:      afterSHA,
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
			vcsType:            vcs.GitHub,
			externalID:         "octocat/Hello-World",
			repositoryFullPath: "octocat/Hello-World",
			newWebhookPushEvent: func(added, modified []string, beforeSHA, afterSHA string) any {
				return github.WebhookPushEvent{
					Ref:    "refs/heads/feature/foo",
					Before: beforeSHA,
					After:  afterSHA,
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
		// Fix the problem that closure in a for loop will always use the last element.
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			a := require.New(t)
			ctx := context.Background()
			ctl := &controller{}
			err := ctl.StartServerWithExternalPg(ctx, &config{
				dataDir:            t.TempDir(),
				vcsProviderCreator: test.vcsProviderCreator,
			})
			a.NoError(err)
			defer func() {
				_ = ctl.Close(ctx)
			}()

			// Create a PostgreSQL instance.
			pgPort := getTestPort()
			stopInstance := postgres.SetupTestInstance(t, pgPort, resourceDir)
			defer stopInstance()

			pgDB, err := sql.Open("pgx", fmt.Sprintf("host=/tmp port=%d user=root database=postgres", pgPort))
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
			apiVCS, err := ctl.createVCS(
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
					ResourceID:       generateRandomString("project", 10),
					Name:             "Test VCS Project",
					Key:              "TestVCSSchemaUpdate",
					SchemaChangeType: api.ProjectSchemaChangeTypeSDL,
				},
			)
			a.NoError(err)

			// Create a repository
			ctl.vcsProvider.CreateRepository(test.externalID)

			// Create the branch
			err = ctl.vcsProvider.CreateBranch(test.externalID, "feature/foo")
			a.NoError(err)

			_, err = ctl.createRepository(
				api.RepositoryCreate{
					VCSID:              apiVCS.ID,
					ProjectID:          project.ID,
					Name:               "Test Repository",
					FullPath:           test.repositoryFullPath,
					WebURL:             fmt.Sprintf("%s/%s", ctl.vcsURL, test.repositoryFullPath),
					BranchFilter:       "feature/foo",
					BaseDirectory:      baseDirectory,
					FilePathTemplate:   "{{ENV_ID}}/{{DB_NAME}}##{{VERSION}}##{{TYPE}}##{{DESCRIPTION}}.sql",
					SchemaPathTemplate: "{{ENV_ID}}/.{{DB_NAME}}##LATEST.sql",
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
					ResourceID:    generateRandomString("instance", 10),
					EnvironmentID: prodEnvironment.ID,
					Name:          "pgInstance",
					Engine:        db.Postgres,
					Host:          "/tmp",
					Port:          strconv.Itoa(pgPort),
					Username:      "bytebase",
					Password:      "bytebase",
				},
			)
			a.NoError(err)

			// Create an issue that creates a database
			err = ctl.createDatabase(project, instance, databaseName, "bytebase", nil /* labelMap */)
			a.NoError(err)

			// Simulate Git commits for schema update to create a new table "users".
			schemaFile := fmt.Sprintf("bbtest/prod/.%s##LATEST.sql", databaseName)
			schemaFileContent += "\nCREATE TABLE users (id serial PRIMARY KEY);"
			err = ctl.vcsProvider.AddFiles(test.externalID, map[string]string{
				schemaFile: schemaFileContent,
			})
			a.NoError(err)
			err = ctl.vcsProvider.AddCommitsDiff(test.externalID, "1", "2", []vcs.FileDiff{
				{Path: schemaFile, Type: vcs.FileDiffTypeAdded},
			})
			a.NoError(err)
			payload, err := json.Marshal(test.newWebhookPushEvent(nil /* added */, []string{schemaFile}, "1", "2"))
			a.NoError(err)
			err = ctl.vcsProvider.SendWebhookPush(test.externalID, payload)
			a.NoError(err)

			// Get schema update issue
			issues, err := ctl.getIssues(&project.ID, api.IssueOpen)
			a.NoError(err)
			a.Len(issues, 1)
			issue := issues[0]
			status, err := ctl.waitIssuePipeline(issue.ID)
			a.NoError(err)
			a.Equal(api.TaskDone, status)
			issue, err = ctl.getIssue(issue.ID)
			a.NoError(err)
			a.Equal("[testVCSSchemaUpdate] Alter schema", issue.Name)
			a.Equal("Apply schema diff by file prod/.testVCSSchemaUpdate##LATEST.sql", issue.Description)
			_, err = ctl.patchIssueStatus(
				api.IssueStatusPatch{
					ID:     issue.ID,
					Status: api.IssueDone,
				},
			)
			a.NoError(err)

			// Simulate Git commits for data update to the table "users".
			const dataFile = "bbtest/prod/testVCSSchemaUpdate##ver2##data##insert_data.sql"
			err = ctl.vcsProvider.AddFiles(test.externalID, map[string]string{
				dataFile: `INSERT INTO users (id) VALUES (1);`,
			})
			a.NoError(err)
			err = ctl.vcsProvider.AddCommitsDiff(test.externalID, "2", "3", []vcs.FileDiff{
				{Path: dataFile, Type: vcs.FileDiffTypeAdded},
			})
			a.NoError(err)
			payload, err = json.Marshal(test.newWebhookPushEvent([]string{dataFile}, nil /* modified */, "2", "3"))
			a.NoError(err)
			err = ctl.vcsProvider.SendWebhookPush(test.externalID, payload)
			a.NoError(err)

			// Get data update issue
			issues, err = ctl.getIssues(&project.ID, api.IssueOpen)
			a.NoError(err)
			a.Len(issues, 1)
			issue = issues[0]
			status, err = ctl.waitIssuePipeline(issue.ID)
			a.NoError(err)
			a.Equal(api.TaskDone, status)
			issue, err = ctl.getIssue(issue.ID)
			a.NoError(err)
			a.Equal("[testVCSSchemaUpdate] Change data", issue.Name)
			a.Equal("By VCS files:\n\nprod/testVCSSchemaUpdate##ver2##data##insert_data.sql\n", issue.Description)
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
			a.Equal(`[["table_name"],["NAME"],[["projects"],["users"]],[false]]`, result)

			// Get migration history
			const initialSchema = `
SET statement_timeout = 0;
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
			const updatedSchema = `
SET statement_timeout = 0;
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

			histories, err := ctl.getInstanceMigrationHistory(instance.ID, db.MigrationHistoryFind{})
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
					Type:       db.MigrateSDL,
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

	defaultNewWebhookPushEvent := func(added []string, beforeSHA, afterSHA string) any {
		return gitlab.WebhookPushEvent{
			ObjectKind: gitlab.WebhookPush,
			Ref:        fmt.Sprintf("refs/heads/%s", branchFilter),
			Before:     beforeSHA,
			After:      afterSHA,
			Project: gitlab.WebhookProject{
				ID: 121,
			},
			CommitList: []gitlab.WebhookCommit{
				{
					Timestamp: "2021-01-13T13:14:00Z",
					AddedList: added,
				},
			},
		}
	}
	tests := []struct {
		name                  string
		vcsProviderCreator    fake.VCSProviderCreator
		vcsType               vcs.Type
		baseDirectory         string
		envName               string
		filePathTemplate      string
		commitNewFileNames    []string
		commitNewFileContents []string
		expect                []bool
		newWebhookPushEvent   func(added []string, beforeSHA, afterSHA string) any
	}{
		{
			name:               "singleAsterisk",
			vcsProviderCreator: fake.NewGitLab,
			vcsType:            vcs.GitLab,
			baseDirectory:      "bbtest",
			envName:            "wildcard",
			filePathTemplate:   "{{ENV_ID}}/*/{{DB_NAME}}##{{VERSION}}##{{TYPE}}##{{DESCRIPTION}}.sql",
			commitNewFileNames: []string{
				// Normal
				fmt.Sprintf("%s/%s/foo/%s##ver1##migrate##create_table_t1.sql", baseDirectory, "wildcard", dbName),
				// One singleAsterisk cannot match two directories.
				fmt.Sprintf("%s/%s/foo/bar/%s##ver2##data##insert_data.sql", baseDirectory, "wildcard", dbName),
				// One singleAsterisk cannot match zero directory.
				fmt.Sprintf("%s/%s/%s##ver3##migrate##create_table_t3.sql", baseDirectory, "wildcard", dbName),
			},
			commitNewFileContents: []string{
				"CREATE TABLE t1 (id INT);",
				"INSERT INTO t1 VALUES (1);",
				"CREATE TABLE t3 (id INT);",
			},
			expect: []bool{
				true,
				false,
				false,
			},
			newWebhookPushEvent: defaultNewWebhookPushEvent,
		},
		{
			name:               "continuousSingleAsterisk",
			vcsProviderCreator: fake.NewGitLab,
			vcsType:            vcs.GitLab,
			baseDirectory:      "bbtest",
			envName:            "wildcard",
			filePathTemplate:   "{{ENV_ID}}/*/*/{{DB_NAME}}##{{VERSION}}##{{TYPE}}##{{DESCRIPTION}}.sql",
			commitNewFileNames: []string{
				// The second single asterisk represents empty folder.
				fmt.Sprintf("%s/%s/foo/bar/%s##ver1##migrate##create_table_t1.sql", baseDirectory, "wildcard", dbName),
				// Any singleAsterisk cannot match zero directory.
				fmt.Sprintf("%s/%s/foo/%s##ver2##data##insert_data.sql", baseDirectory, "wildcard", dbName),
				fmt.Sprintf("%s/%s/%s##ver3##migrate##create_table_t3.sql", baseDirectory, "wildcard", dbName),
			},
			commitNewFileContents: []string{
				"CREATE TABLE t1 (id INT);",
				"INSERT INTO t1 VALUES (1);",
				"CREATE TABLE t3 (id INT);",
			},
			expect: []bool{
				true,
				false,
				false,
			},
			newWebhookPushEvent: defaultNewWebhookPushEvent,
		},
		{
			name:               "doubleAsterisks",
			vcsProviderCreator: fake.NewGitLab,
			vcsType:            vcs.GitLab,
			baseDirectory:      "bbtest",
			envName:            "wildcard",
			filePathTemplate:   "{{ENV_ID}}/**/{{DB_NAME}}##{{VERSION}}##{{TYPE}}##{{DESCRIPTION}}.sql",
			commitNewFileNames: []string{
				// Two singleAsterisk can match one directory.
				fmt.Sprintf("%s/%s/foo/%s##ver1##migrate##create_table_t1.sql", baseDirectory, "wildcard", dbName),
				// Two singleAsterisk can match two directories.
				fmt.Sprintf("%s/%s/foo/bar/%s##ver2##migrate##create_table_t2.sql", baseDirectory, "wildcard", dbName),
				// Two singleAsterisk can match three directories or more.
				fmt.Sprintf("%s/%s/foo/bar/foo/%s##ver3##migrate##create_table_t3.sql", baseDirectory, "wildcard", dbName),
				// Two singleAsterisk cannot match zero directory.
				fmt.Sprintf("%s/%s/%s##ver4##migrate##create_table_t4.sql", baseDirectory, "wildcard", dbName),
			},
			commitNewFileContents: []string{
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
			newWebhookPushEvent: defaultNewWebhookPushEvent,
		},
		{
			name:               "emptyBaseAndMixAsterisks",
			vcsProviderCreator: fake.NewGitLab,
			envName:            "wildcard",
			baseDirectory:      "",
			vcsType:            vcs.GitLab,
			filePathTemplate:   "{{ENV_ID}}/**/foo/*/{{DB_NAME}}##{{VERSION}}##{{TYPE}}##{{DESCRIPTION}}.sql",
			commitNewFileNames: []string{
				// ** matches foo, foo matches foo, * matches bar
				fmt.Sprintf("%s/foo/foo/bar/%s##ver1##migrate##create_table_t1.sql", "wildcard", dbName),
				// ** matches foo/bar/foo, foo matches foo, * matches bar
				fmt.Sprintf("%s/foo/bar/foo/foo/bar/%s##ver2##migrate##create_table_t2.sql", "wildcard", dbName),
				// cannot match
				fmt.Sprintf("%s/%s##ver3##migrate##create_table_t3.sql", "wildcard", dbName),
			},
			commitNewFileContents: []string{
				"CREATE TABLE t1 (id INT);",
				"CREATE TABLE t2 (id INT);",
				"CREATE TABLE t3 (id INT);",
			},
			expect: []bool{
				true,
				true,
				false,
			},
			newWebhookPushEvent: defaultNewWebhookPushEvent,
		},
		// We test the combination of ** and *, and the place holder is not fully represented by the ascii character set.
		{
			name:               "mixAsterisks",
			vcsProviderCreator: fake.NewGitLab,
			envName:            "prod1",
			baseDirectory:      "bbtest",
			vcsType:            vcs.GitLab,
			filePathTemplate:   "{{ENV_ID}}/**/foo/*/{{DB_NAME}}##{{VERSION}}##{{TYPE}}##{{DESCRIPTION}}.sql",
			commitNewFileNames: []string{
				// ** matches foo, foo matches foo, * matches bar
				fmt.Sprintf("%s/%s/foo/foo/bar/%s##ver1##migrate##create_table_t1.sql", baseDirectory, "prod1", dbName),
				// ** matches foo/bar/foo, foo matches foo, * matches bar
				fmt.Sprintf("%s/%s/foo/bar/foo/foo/bar/%s##ver2##migrate##create_table_t2.sql", baseDirectory, "prod1", dbName),
				// cannot match
				fmt.Sprintf("%s/%s/%s##ver3##migrate##create_table_t3.sql", baseDirectory, "prod1", dbName),
			},
			commitNewFileContents: []string{
				"CREATE TABLE t1 (id INT);",
				"CREATE TABLE t2 (id INT);",
				"CREATE TABLE t3 (id INT);",
			},
			expect: []bool{
				true,
				true,
				false,
			},
			newWebhookPushEvent: defaultNewWebhookPushEvent,
		},
		// No asterisk
		{
			name:               "placeholderAsFolder",
			vcsProviderCreator: fake.NewGitLab,
			envName:            "ZO",
			baseDirectory:      "bbtest",
			vcsType:            vcs.GitLab,
			filePathTemplate:   "{{ENV_ID}}/{{DB_NAME}}/sql/{{DB_NAME}}##{{VERSION}}##{{TYPE}}##{{DESCRIPTION}}.sql",
			commitNewFileNames: []string{
				fmt.Sprintf("%s/%s/%s/sql/%s##ver1##migrate##create_table_t1.sql", baseDirectory, "zo", dbName, dbName),
				fmt.Sprintf("%s/%s/%s/%s##ver2##migrate##create_table_t2.sql", baseDirectory, "zo", dbName, dbName),
				fmt.Sprintf("%s/%s/%s/sql/%s##ver3##migrate##create_table_t3.sql", baseDirectory, "zo", dbName, dbName),
			},
			commitNewFileContents: []string{
				"CREATE TABLE t1 (id INT);",
				"CREATE TABLE t2 (id INT);",
				"CREATE TABLE t3 (id INT);",
			},
			expect: []bool{
				true,
				false,
				true,
			},
			newWebhookPushEvent: defaultNewWebhookPushEvent,
		},
	}
	for _, test := range tests {
		// Fix the problem that closure in a for loop will always use the last element.
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			a := require.New(t)
			ctx := context.Background()
			ctl := &controller{}
			err := ctl.StartServerWithExternalPg(ctx, &config{
				dataDir:            t.TempDir(),
				vcsProviderCreator: test.vcsProviderCreator,
			})
			a.NoError(err)
			defer func() {
				_ = ctl.Close(ctx)
			}()

			err = ctl.setLicense()
			a.NoError(err)

			// Create a VCS.
			apiVCS, err := ctl.createVCS(
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
					ResourceID: generateRandomString("project", 10),
					Name:       "Test VCS Project",
					Key:        "TVP",
				},
			)
			a.NoError(err)

			// Create a repository.
			ctl.vcsProvider.CreateRepository(externalID)

			// Create the branch.
			err = ctl.vcsProvider.CreateBranch(externalID, branchFilter)
			a.NoError(err)

			_, err = ctl.createRepository(
				api.RepositoryCreate{
					VCSID:              apiVCS.ID,
					ProjectID:          project.ID,
					Name:               "Test Repository",
					FullPath:           repoFullPath,
					WebURL:             fmt.Sprintf("%s/%s", ctl.vcsURL, repoFullPath),
					BranchFilter:       branchFilter,
					BaseDirectory:      test.baseDirectory,
					FilePathTemplate:   test.filePathTemplate,
					SchemaPathTemplate: "{{ENV_ID}}/.{{DB_NAME}}##LATEST.sql",
					ExternalID:         externalID,
					AccessToken:        "accessToken1",
					RefreshToken:       "refreshToken1",
				},
			)
			a.NoError(err)

			environment, err := ctl.createEnvironment(api.EnvironmentCreate{
				Name: test.envName,
			})
			a.NoError(err)
			// Provision an instance.
			instanceRootDir := t.TempDir()
			instanceName := "testInstance1"
			instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, instanceName)
			a.NoError(err)
			instance, err := ctl.addInstance(api.InstanceCreate{
				ResourceID:    generateRandomString("instance", 10),
				EnvironmentID: environment.ID,
				Name:          instanceName,
				Engine:        db.SQLite,
				Host:          instanceDir,
			})
			a.NoError(err)

			// Create an issue that creates a database.
			err = ctl.createDatabase(project, instance, dbName, "", nil /* labelMap */)
			a.NoError(err)

			a.Equal(len(test.expect), len(test.commitNewFileNames))
			a.Equal(len(test.expect), len(test.commitNewFileContents))

			for idx, commitFileName := range test.commitNewFileNames {
				// Simulate Git commits for schema update.
				err = ctl.vcsProvider.AddFiles(externalID, map[string]string{commitFileName: test.commitNewFileContents[idx]})
				a.NoError(err)
				// We always commit one file at a time in this test.
				beforeCommit := strconv.Itoa(idx)
				afterCommit := strconv.Itoa(idx + 1)
				err = ctl.vcsProvider.AddCommitsDiff(externalID, beforeCommit, afterCommit, []vcs.FileDiff{
					{Path: commitFileName, Type: vcs.FileDiffTypeAdded},
				})
				a.NoError(err)
				payload, err := json.Marshal(test.newWebhookPushEvent([]string{commitFileName}, beforeCommit, afterCommit))
				a.NoError(err)
				err = ctl.vcsProvider.SendWebhookPush(externalID, payload)
				a.NoError(err)

				// Check for newly generated issues.
				issues, err := ctl.getIssues(&project.ID, api.IssueOpen)
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

func TestVCS_SQL_Review(t *testing.T) {
	tests := []struct {
		name                    string
		vcsProviderCreator      fake.VCSProviderCreator
		vcsType                 vcs.Type
		externalID              string
		repositoryFullPath      string
		getEmptySQLReviewResult func(repo *api.Repository, filePath, rootURL string) *api.VCSSQLReviewResult
		getSQLReviewResult      func(repo *api.Repository, filePath string) *api.VCSSQLReviewResult
	}{
		{
			name:               "GitLab",
			vcsProviderCreator: fake.NewGitLab,
			vcsType:            vcs.GitLab,
			externalID:         "121",
			repositoryFullPath: "test/schemaUpdate",
			getEmptySQLReviewResult: func(repo *api.Repository, filePath, rootURL string) *api.VCSSQLReviewResult {
				pathes := strings.Split(filePath, "/")
				return &api.VCSSQLReviewResult{
					Status: advisor.Warn,
					Content: []string{
						fmt.Sprintf(
							"<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<testsuites name=\"SQL Review\">\n<testsuite name=\"%s\">\n<testcase name=\"[WARN] %s#L1: SQL review policy not found\" classname=\"%s\" file=\"%s#L1\">\n<failure>\nError: You can configure the SQL review policy on %s/setting/sql-review.\nYou can check the docs at https://www.bytebase.com/docs/reference/error-code/advisor#2\n</failure>\n</testcase>\n</testsuite>\n</testsuites>",
							filePath,
							pathes[len(pathes)-1],
							filePath,
							filePath,
							rootURL,
						),
					},
				}
			},
			getSQLReviewResult: func(repo *api.Repository, filePath string) *api.VCSSQLReviewResult {
				pathes := strings.Split(filePath, "/")
				filename := pathes[len(pathes)-1]
				return &api.VCSSQLReviewResult{
					Status: advisor.Warn,
					Content: []string{
						fmt.Sprintf(
							"<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<testsuites name=\"SQL Review\">\n<testsuite name=\"%s\">\n<testcase name=\"[WARN] %s#L1: column.required\" classname=\"%s\" file=\"%s#L1\">\n<failure>\nError: Table \"book\" requires columns: created_ts, creator_id, updated_ts, updater_id.\nYou can check the docs at https://www.bytebase.com/docs/reference/error-code/advisor#401\n</failure>\n</testcase>\n<testcase name=\"[WARN] %s#L1: column.no-null\" classname=\"%s\" file=\"%s#L1\">\n<failure>\nError: Column \"name\" in \"public\".\"book\" cannot have NULL value.\nYou can check the docs at https://www.bytebase.com/docs/reference/error-code/advisor#402\n</failure>\n</testcase>\n</testsuite>\n</testsuites>",
							filePath,
							filename,
							filePath,
							filePath,
							filename,
							filePath,
							filePath,
						),
					},
				}
			},
		},
		{
			name:               "GitHub",
			vcsProviderCreator: fake.NewGitHub,
			vcsType:            vcs.GitHub,
			externalID:         "octocat/Hello-World",
			repositoryFullPath: "octocat/Hello-World",
			getEmptySQLReviewResult: func(repo *api.Repository, filePath, rootURL string) *api.VCSSQLReviewResult {
				return &api.VCSSQLReviewResult{
					Status: advisor.Warn,
					Content: []string{
						fmt.Sprintf(
							"::warning file=%s,line=1,col=1,endColumn=2,title=SQL review policy not found (2)::You can configure the SQL review policy on %s/setting/sql-review%%0ADoc: https://www.bytebase.com/docs/reference/error-code/advisor#2",
							filePath,
							rootURL,
						),
					},
				}
			},
			getSQLReviewResult: func(repo *api.Repository, filePath string) *api.VCSSQLReviewResult {
				return &api.VCSSQLReviewResult{
					Status: advisor.Warn,
					Content: []string{
						fmt.Sprintf(
							"::warning file=%s,line=1,col=1,endColumn=2,title=column.required (401)::Table \"book\" requires columns: created_ts, creator_id, updated_ts, updater_id%%0ADoc: https://www.bytebase.com/docs/reference/error-code/advisor#401",
							filePath,
						),
						fmt.Sprintf(
							"::warning file=%s,line=1,col=1,endColumn=2,title=column.no-null (402)::Column \"name\" in \"public\".\"book\" cannot have NULL value%%0ADoc: https://www.bytebase.com/docs/reference/error-code/advisor#402",
							filePath,
						),
					},
				}
			},
		},
	}

	for _, test := range tests {
		// Fix the problem that closure in a for loop will always use the last element.
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			a := require.New(t)
			ctx := context.Background()
			ctl := &controller{}
			err := ctl.StartServerWithExternalPg(ctx, &config{
				dataDir:            t.TempDir(),
				vcsProviderCreator: test.vcsProviderCreator,
				// We check against empty SQL Review policy, while our onboarding data generation
				// will create a SQL Review policy. Thus we need to skip onboarding data generation.
				skipOnboardingData: true,
			})
			a.NoError(err)
			defer func() {
				_ = ctl.Close(ctx)
			}()

			err = ctl.setLicense()
			a.NoError(err)

			// Create a PostgreSQL instance.
			pgPort := getTestPort()
			stopInstance := postgres.SetupTestInstance(t, pgPort, resourceDir)
			defer stopInstance()

			pgDB, err := sql.Open("pgx", fmt.Sprintf("host=/tmp port=%d user=root database=postgres", pgPort))
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

			// Create a VCS.
			vcsData, err := ctl.createVCS(
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
					ResourceID: generateRandomString("project", 10),
					Name:       "Test VCS Project",
					Key:        "TestVCSSchemaUpdate",
				},
			)
			a.NoError(err)

			environments, err := ctl.getEnvironments()
			a.NoError(err)
			prodEnvironment, err := findEnvironment(environments, "Prod")
			a.NoError(err)

			// Add an instance.
			instance, err := ctl.addInstance(api.InstanceCreate{
				ResourceID:    generateRandomString("instance", 10),
				EnvironmentID: prodEnvironment.ID,
				Name:          "pgInstance",
				Engine:        db.Postgres,
				Host:          "/tmp",
				Port:          strconv.Itoa(pgPort),
				Username:      "bytebase",
				Password:      "bytebase",
			})
			a.NoError(err)

			// Create an issue that creates a database.
			err = ctl.createDatabase(project, instance, databaseName, "bytebase", nil)
			a.NoError(err)

			// Create a repository.
			ctl.vcsProvider.CreateRepository(test.externalID)

			// Create the branch.
			err = ctl.vcsProvider.CreateBranch(test.externalID, "feature/foo")
			a.NoError(err)

			repository, err := ctl.createRepository(
				api.RepositoryCreate{
					VCSID:              vcsData.ID,
					ProjectID:          project.ID,
					Name:               "Test Repository",
					FullPath:           test.repositoryFullPath,
					WebURL:             fmt.Sprintf("%s/%s", ctl.vcsURL, test.repositoryFullPath),
					BranchFilter:       "feature/foo",
					BaseDirectory:      baseDirectory,
					FilePathTemplate:   "{{ENV_ID}}/{{DB_NAME}}##{{VERSION}}##{{TYPE}}##{{DESCRIPTION}}.sql",
					SchemaPathTemplate: "{{ENV_ID}}/.{{DB_NAME}}##LATEST.sql",
					ExternalID:         test.externalID,
					AccessToken:        "accessToken1",
					RefreshToken:       "refreshToken1",
				},
			)
			a.NoError(err)
			a.NotNil(repository)
			a.Equal(false, repository.EnableSQLReviewCI)

			sqlReviewCI, err := ctl.createSQLReviewCI(project.ID, repository.ID)
			a.NoError(err)
			a.NotNil(sqlReviewCI)

			repositoryList, err := ctl.listRepository(project.ID)
			a.NoError(err)
			a.NotNil(repositoryList)
			a.Equal(1, len(repositoryList))
			a.Equal(true, repositoryList[0].EnableSQLReviewCI)

			// Simulate Git commits and pull request for SQL review.
			prID := rand.Int()
			gitFile := "bbtest/prod/testVCSSchemaUpdate##ver3##migrate##create_table_book.sql"
			fileContent := "CREATE TABLE book (id serial PRIMARY KEY, name TEXT);"
			err = ctl.vcsProvider.AddFiles(test.externalID, map[string]string{gitFile: fileContent})
			a.NoError(err)

			err = ctl.vcsProvider.AddPullRequest(
				test.externalID,
				prID,
				[]*vcs.PullRequestFile{
					{
						Path:         gitFile,
						LastCommitID: "last_commit_id",
						IsDeleted:    false,
					},
				},
			)
			a.NoError(err)

			// trigger SQL review with empty policy.
			res, err := postVCSSQLReview(ctl, repository, &api.VCSSQLReviewRequest{
				RepositoryID:  repository.ExternalID,
				PullRequestID: fmt.Sprintf("%d", prID),
				WebURL:        repository.WebURL,
			})
			a.NoError(err)

			emptySQLReview := test.getEmptySQLReviewResult(repository, gitFile, ctl.rootURL)
			a.Equal(emptySQLReview.Status, res.Status)
			a.Equal(emptySQLReview.Content, res.Content)

			// create the SQL review policy then re-trigger the VCS SQL review.
			policyPayload, err := prodTemplateSQLReviewPolicyForPostgreSQL()
			a.NoError(err)

			_, err = ctl.upsertPolicy(api.PolicyResourceTypeEnvironment, prodEnvironment.ID, api.PolicyTypeSQLReview, api.PolicyUpsert{
				Payload: &policyPayload,
			})
			a.NoError(err)

			reviewResult, err := postVCSSQLReview(ctl, repository, &api.VCSSQLReviewRequest{
				RepositoryID:  repository.ExternalID,
				PullRequestID: fmt.Sprintf("%d", prID),
				WebURL:        repository.WebURL,
			})
			a.NoError(err)

			expectResult := test.getSQLReviewResult(repository, gitFile)
			a.Equal(expectResult.Status, reviewResult.Status)
			a.Equal(expectResult.Content, reviewResult.Content)
		})
	}
}

func TestBranchNameInVCSSetupAndUpdate(t *testing.T) {
	type testCase struct {
		name              string
		existedBranchList []string
		branchFilter      string
		want              bool
	}
	type vcsTestCase struct {
		vcsType            vcs.Type
		vcsProviderCreator fake.VCSProviderCreator
		externalID         string
		repoFullPath       string
		caseList           []testCase
	}

	tests := []vcsTestCase{
		{
			vcsType:            vcs.GitLab,
			vcsProviderCreator: fake.NewGitLab,
			externalID:         "1234",
			repoFullPath:       "1234",
			caseList: []testCase{
				{
					name: "mainBranchWithGitLab",
					existedBranchList: []string{
						"main",
						"test/branch",
					},
					branchFilter: "main",
					want:         false,
				}, {
					name: "customBranchWithGitLab",
					existedBranchList: []string{
						"main",
						"test/branch",
					},
					branchFilter: "test/branch",
					want:         false,
				}, {
					name: "nonExistedBranchWithGitLab",
					existedBranchList: []string{
						"main",
						"test/branch",
					},
					branchFilter: "non_existed_branch",
					want:         true,
				}, {
					name: "wildcardBranchWithGitLab",
					existedBranchList: []string{
						"main",
					},
					branchFilter: "main*",
					want:         false,
				},
			},
		},
		{
			vcsType:            vcs.GitHub,
			vcsProviderCreator: fake.NewGitHub,
			externalID:         "test/branch",
			repoFullPath:       "test/branch",
			caseList: []testCase{
				{
					name: "mainBranchWithGitHub",
					existedBranchList: []string{
						"main",
						"test/branch",
					},
					branchFilter: "main",
					want:         false,
				}, {
					name: "customBranchWithGitHub",
					existedBranchList: []string{
						"main",
						"test/branch",
					},
					branchFilter: "test/branch",
					want:         false,
				}, {
					name: "nonExistedBranchWithGitHub",
					existedBranchList: []string{
						"main",
						"test/branch",
					},
					branchFilter: "non_existed_branch",
					want:         true,
				}, {
					name: "wildcardBranchWithGitHub",
					existedBranchList: []string{
						"main",
					},
					branchFilter: "main*",
					want:         false,
				},
			},
		},
	}

	for _, vcsTest := range tests {
		// Wrap the defer statement in an anonymous func to make it work properly.
		(func() {
			a := require.New(t)
			ctx := context.Background()
			ctl := &controller{}

			// Create a server.
			err := ctl.StartServerWithExternalPg(ctx, &config{
				dataDir:            t.TempDir(),
				vcsProviderCreator: vcsTest.vcsProviderCreator,
			})
			a.NoError(err)

			defer func() {
				_ = ctl.Close(ctx)
			}()

			// Create a VCS.
			apiVCS, err := ctl.createVCS(
				api.VCSCreate{
					Name:          "testName",
					Type:          vcsTest.vcsType,
					InstanceURL:   ctl.vcsURL,
					APIURL:        ctl.vcsProvider.APIURL(ctl.vcsURL),
					ApplicationID: "testID",
					Secret:        "testSecret",
				},
			)
			a.NoError(err)

			// Create a project.
			project, err := ctl.createProject(
				api.ProjectCreate{
					ResourceID: generateRandomString("project", 10),
					Name:       "Test VSC Project",
					Key:        "TVP",
				},
			)
			a.NoError(err)

			for _, test := range vcsTest.caseList {
				test := test
				t.Run(test.name, func(t *testing.T) {
					// Create a repository in the fake vsc provider.
					ctl.vcsProvider.CreateRepository(vcsTest.externalID)

					// Create existed branches.
					for _, existedBranch := range test.existedBranchList {
						err := ctl.vcsProvider.CreateBranch(vcsTest.externalID, existedBranch)
						a.NoError(err)
					}

					// Create a repository.
					_, err = ctl.createRepository(
						api.RepositoryCreate{
							VCSID:              apiVCS.ID,
							ProjectID:          project.ID,
							Name:               "Test Repository",
							FullPath:           vcsTest.repoFullPath,
							WebURL:             fmt.Sprintf("%s/%s", ctl.vcsURL, vcsTest.repoFullPath),
							BranchFilter:       test.branchFilter,
							BaseDirectory:      "",
							FilePathTemplate:   "{{ENV_ID}}/{{DB_NAME}}##{{VERSION}}##{{TYPE}}##{{DESCRIPTION}}.sql",
							SchemaPathTemplate: "",
							ExternalID:         vcsTest.externalID,
							AccessToken:        "accessToken1",
							RefreshToken:       "refreshToken1",
						},
					)

					if test.want {
						a.Error(err)
					} else {
						a.NoError(err)
						err = ctl.unlinkRepository(project.ID)
						a.NoError(err)
					}
				})
			}
		})()
	}
}

func getWorkspaceID(ctl *controller) (string, error) {
	body, err := ctl.getOpenAPI("/actuator/info", nil)
	if err != nil {
		return "", err
	}
	bs, err := io.ReadAll(body)
	if err != nil {
		return "", err
	}
	actuatorInfo := new(v1pb.ActuatorInfo)
	if err = protojson.Unmarshal(bs, actuatorInfo); err != nil {
		return "", errors.Wrap(err, "fail to unmarshal get actuator response")
	}
	return actuatorInfo.WorkspaceId, nil
}

// postVCSSQLReview will create the VCS SQL review and get the response.
func postVCSSQLReview(ctl *controller, repo *api.Repository, request *api.VCSSQLReviewRequest) (*api.VCSSQLReviewResult, error) {
	url := fmt.Sprintf("%s/hook/sql-review/%s", ctl.rootURL, repo.WebhookEndpointID)

	payload, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", url, bytes.NewReader(payload))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create a new POST request to %q", url)
	}

	workspaceID, err := getWorkspaceID(ctl)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-SQL-Review-Token", workspaceID)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}

	res := new(api.VCSSQLReviewResult)
	if err := json.Unmarshal([]byte(body), res); err != nil {
		return nil, err
	}

	return res, nil
}

func TestGetLatestSchema(t *testing.T) {
	tests := []struct {
		name                 string
		dbType               db.Type
		databaseName         string
		ddl                  string
		wantRawSchema        string
		wantSDL              string
		wantDatabaseMetadata *storepb.DatabaseMetadata
	}{
		{
			name:         "MySQL",
			dbType:       db.MySQL,
			databaseName: "latestSchema",
			ddl:          `CREATE TABLE book(id INT, name TEXT);`,
			wantRawSchema: "SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0;\n" +
				"SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0;\n" +
				"--\n" +
				"-- Table structure for `book`\n" +
				"--\n" +
				"CREATE TABLE `book` (\n" +
				"  `id` int DEFAULT NULL,\n" +
				"  `name` text COLLATE utf8mb4_general_ci\n" +
				") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;\n\n" +
				"SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS;\n" +
				"SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS;\n",
			wantSDL: "CREATE TABLE `book` (\n" +
				"  `id` INT DEFAULT NULL,\n" +
				"  `name` TEXT COLLATE utf8mb4_general_ci\n" +
				") ENGINE=InnoDB DEFAULT CHARACTER SET=UTF8MB4 DEFAULT COLLATE=UTF8MB4_GENERAL_CI;\n\n",
			wantDatabaseMetadata: &storepb.DatabaseMetadata{
				Name:         "latestSchema",
				CharacterSet: "utf8mb4",
				Collation:    "utf8mb4_general_ci",
				Schemas: []*storepb.SchemaMetadata{
					{
						Tables: []*storepb.TableMetadata{
							{
								Name:      "book",
								Engine:    "InnoDB",
								Collation: "utf8mb4_general_ci",
								DataSize:  16384,
								Columns: []*storepb.ColumnMetadata{
									{
										Name:     "id",
										Position: 1,
										Nullable: true,
										Type:     "int",
									},
									{
										Name:         "name",
										Position:     2,
										Nullable:     true,
										Type:         "text",
										CharacterSet: "utf8mb4",
										Collation:    "utf8mb4_general_ci",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name:         "PostgreSQL",
			dbType:       db.Postgres,
			databaseName: "latestSchema",
			ddl:          `CREATE TABLE book(id INT, name TEXT);`,
			wantRawSchema: `
SET statement_timeout = 0;
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

CREATE TABLE public.book (
    id integer,
    name text
);

`,
			wantSDL: ``,
			wantDatabaseMetadata: &storepb.DatabaseMetadata{
				Name:         "latestSchema",
				CharacterSet: "UTF8",
				Collation:    "en_US.UTF-8",
				Schemas: []*storepb.SchemaMetadata{
					{
						Name: "public",
						Tables: []*storepb.TableMetadata{
							{
								Name:     "book",
								DataSize: 8192,
								Columns: []*storepb.ColumnMetadata{
									{Name: "id", Position: 1, Nullable: true, Type: "integer"},
									{Name: "name", Position: 2, Nullable: true, Type: "text"},
								},
							},
						},
					},
				},
			},
		},
	}
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	err := ctl.StartServerWithExternalPg(ctx, &config{
		dataDir:            t.TempDir(),
		vcsProviderCreator: fake.NewGitLab,
	})
	a.NoError(err)
	defer func() {
		_ = ctl.Close(ctx)
	}()
	err = ctl.setLicense()
	a.NoError(err)
	environmentName := t.Name()
	environment, err := ctl.createEnvironment(api.EnvironmentCreate{
		Name: environmentName,
	})
	a.NoError(err)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dbPort := getTestPort()
			switch test.dbType {
			case db.Postgres:
				stopInstance := postgres.SetupTestInstance(t, dbPort, resourceDir)
				defer stopInstance()
			case db.MySQL:
				stopInstance := mysql.SetupTestInstance(t, dbPort, mysqlBinDir)
				defer stopInstance()
			default:
				a.FailNow("unsupported db type")
			}
			project, err := ctl.createProject(
				api.ProjectCreate{
					ResourceID: generateRandomString("project", 10),
					Name:       test.name,
					Key:        test.name,
				},
			)
			a.NoError(err)
			// Add an instance.
			var instance *api.Instance
			switch test.dbType {
			case db.Postgres:
				instance, err = ctl.addInstance(api.InstanceCreate{
					ResourceID:    generateRandomString("instance", 10),
					EnvironmentID: environment.ID,
					Name:          test.name,
					Engine:        db.Postgres,
					Host:          "/tmp",
					Port:          strconv.Itoa(dbPort),
					Username:      "root",
				})
			case db.MySQL:
				instance, err = ctl.addInstance(api.InstanceCreate{
					ResourceID:    generateRandomString("instance", 10),
					EnvironmentID: environment.ID,
					Name:          "mysqlInstance",
					Engine:        db.MySQL,
					Host:          "127.0.0.1",
					Port:          strconv.Itoa(dbPort),
					Username:      "root",
				})
			default:
				a.FailNow("unsupported db type")
			}

			a.NoError(err)
			err = ctl.createDatabase(project, instance, test.databaseName, "root", nil /* labelMap */)
			a.NoError(err)
			databases, err := ctl.getDatabases(api.DatabaseFind{
				InstanceID: &instance.ID,
			})
			a.NoError(err)
			// Find databases
			var database *api.Database
			for _, db := range databases {
				if db.Name == test.databaseName {
					database = db
					break
				}
			}
			a.NotNil(database)

			ddlSheet, err := ctl.createSheet(api.SheetCreate{
				ProjectID:  project.ID,
				Name:       "test ddl",
				Statement:  test.ddl,
				Visibility: api.ProjectSheet,
				Source:     api.SheetFromBytebaseArtifact,
				Type:       api.SheetForSQL,
			})
			a.NoError(err)

			// Create an issue that updates database schema.
			createContext, err := json.Marshal(&api.MigrationContext{
				DetailList: []*api.MigrationDetail{
					{
						MigrationType: db.Migrate,
						DatabaseID:    database.ID,
						SheetID:       ddlSheet.ID,
					},
				},
			})
			a.NoError(err)
			issue, err := ctl.createIssue(api.IssueCreate{
				ProjectID:     project.ID,
				Name:          fmt.Sprintf("update schema for database %q", test.databaseName),
				Type:          api.IssueDatabaseSchemaUpdate,
				Description:   fmt.Sprintf("This updates the schema of database %q.", test.databaseName),
				AssigneeID:    api.SystemBotID,
				CreateContext: string(createContext),
			})
			a.NoError(err)
			status, err := ctl.waitIssuePipeline(issue.ID)
			a.NoError(err)
			a.Equal(api.TaskDone, status)
			latestSchemaDump, err := ctl.getLatestSchemaDump(database.ID)
			a.NoError(err)
			a.Equal(test.wantRawSchema, latestSchemaDump)
			if test.dbType == db.MySQL {
				latestSchemaSDL, err := ctl.getLatestSchemaSDL(database.ID)
				a.NoError(err)
				a.Equal(test.wantSDL, latestSchemaSDL)
			}
			latestSchemaMetadataString, err := ctl.getLatestSchemaMetadata(database.ID)
			a.NoError(err)
			var latestSchemaMetadata storepb.DatabaseMetadata
			err = protojson.Unmarshal([]byte(latestSchemaMetadataString), &latestSchemaMetadata)
			a.NoError(err)
			diff := cmp.Diff(test.wantDatabaseMetadata, &latestSchemaMetadata, protocmp.Transform())
			a.Equal("", diff)
		})
	}
}

func TestMarkTaskAsDone(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	dataDir := t.TempDir()
	err := ctl.StartServerWithExternalPg(ctx, &config{
		dataDir:            dataDir,
		vcsProviderCreator: fake.NewGitLab,
	})
	a.NoError(err)
	defer ctl.Close(ctx)

	// Create a project.
	project, err := ctl.createProject(api.ProjectCreate{
		ResourceID: generateRandomString("project", 10),
		Name:       "Test Project",
		Key:        "TestSchemaUpdate",
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
		ResourceID:    generateRandomString("instance", 10),
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

	sheet, err := ctl.createSheet(api.SheetCreate{
		ProjectID:  project.ID,
		Name:       "migration statement sheet",
		Statement:  migrationStatement,
		Visibility: api.ProjectSheet,
		Source:     api.SheetFromBytebaseArtifact,
		Type:       api.SheetForSQL,
	})
	a.NoError(err)

	// Create an issue that updates database schema.
	createContext, err := json.Marshal(&api.MigrationContext{
		DetailList: []*api.MigrationDetail{
			{
				MigrationType: db.Migrate,
				DatabaseID:    database.ID,
				SheetID:       sheet.ID,
			},
		},
	})
	a.NoError(err)
	issue, err := ctl.createIssue(api.IssueCreate{
		ProjectID:     project.ID,
		Name:          fmt.Sprintf("update schema for database %q", databaseName),
		Type:          api.IssueDatabaseSchemaUpdate,
		Description:   fmt.Sprintf("This updates the schema of database %q.", databaseName),
		AssigneeID:    api.SystemBotID,
		CreateContext: string(createContext),
	})
	a.NoError(err)

	// Skip the task.
	a.Equal(1, len(issue.Pipeline.StageList))
	a.Equal(1, len(issue.Pipeline.StageList[0].TaskList))
	task := issue.Pipeline.StageList[0].TaskList[0]
	skippedReason := "skip it!"
	task, err = ctl.patchTaskStatus(api.TaskStatusPatch{
		Status:  api.TaskDone,
		Comment: &skippedReason,
	}, issue.PipelineID, task.ID)
	a.NoError(err)
	a.Equal(api.TaskDone, task.Status)

	var payload api.TaskDatabaseSchemaUpdatePayload
	err = json.Unmarshal([]byte(task.Payload), &payload)
	a.NoError(err)
	a.Equal(true, payload.Skipped)
	a.Equal(skippedReason, payload.SkippedReason)

	status, err := ctl.waitIssuePipelineWithNoApproval(issue.ID)
	a.NoError(err)
	a.Equal(api.TaskDone, status)

	// Query schema.
	result, err := ctl.query(instance, databaseName, bookTableQuery)
	a.NoError(err)
	a.NotEqual(bookSchemaSQLResult, result)
}

func TestVCS_SDL_MySQL(t *testing.T) {
	tests := []struct {
		name                string
		vcsProviderCreator  fake.VCSProviderCreator
		vcsType             vcs.Type
		externalID          string
		repositoryFullPath  string
		newWebhookPushEvent func(added, modified []string, beforeSHA, afterSHA string) any
	}{
		{
			name:               "GitLab",
			vcsProviderCreator: fake.NewGitLab,
			vcsType:            vcs.GitLab,
			externalID:         "121",
			repositoryFullPath: "test/schemaUpdate",
			newWebhookPushEvent: func(added, modified []string, beforeSHA, afterSHA string) any {
				return gitlab.WebhookPushEvent{
					ObjectKind: gitlab.WebhookPush,
					Ref:        "refs/heads/feature/foo",
					Before:     beforeSHA,
					After:      afterSHA,
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
			vcsType:            vcs.GitHub,
			externalID:         "octocat/Hello-World",
			repositoryFullPath: "octocat/Hello-World",
			newWebhookPushEvent: func(added, modified []string, beforeSHA, afterSHA string) any {
				return github.WebhookPushEvent{
					Ref:    "refs/heads/feature/foo",
					Before: beforeSHA,
					After:  afterSHA,
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
		// Fix the problem that closure in a for loop will always use the last element.
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			a := require.New(t)
			ctx := context.Background()
			ctl := &controller{}
			err := ctl.StartServerWithExternalPg(ctx, &config{
				dataDir:            t.TempDir(),
				vcsProviderCreator: test.vcsProviderCreator,
			})
			a.NoError(err)
			defer func() {
				_ = ctl.Close(ctx)
			}()

			// Create a MySQL instance.
			mysqlPort := getTestPort()
			stopInstance := mysql.SetupTestInstance(t, mysqlPort, mysqlBinDir)
			defer stopInstance()

			mysqlDB, err := sql.Open("mysql", fmt.Sprintf("root@tcp(127.0.0.1:%d)/mysql", mysqlPort))
			a.NoError(err)
			defer mysqlDB.Close()

			const databaseName = "testVCSSchemaUpdateMySQL"
			_, err = mysqlDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %v", databaseName))
			a.NoError(err)

			_, err = mysqlDB.Exec("DROP USER IF EXISTS bytebase")
			a.NoError(err)
			_, err = mysqlDB.Exec("CREATE USER 'bytebase' IDENTIFIED WITH mysql_native_password BY 'bytebase'")
			a.NoError(err)

			_, err = mysqlDB.Exec("GRANT ALTER, ALTER ROUTINE, CREATE, CREATE ROUTINE, CREATE VIEW, DELETE, DROP, EVENT, EXECUTE, INDEX, INSERT, PROCESS, REFERENCES, SELECT, SHOW DATABASES, SHOW VIEW, TRIGGER, UPDATE, USAGE, REPLICATION CLIENT, REPLICATION SLAVE, LOCK TABLES, RELOAD ON *.* to bytebase")
			a.NoError(err)

			// Create a table in the database
			schemaFileContent := `CREATE TABLE projects (id int, PRIMARY KEY (id));`
			_, err = mysqlDB.Exec(schemaFileContent)
			a.NoError(err)

			// Create a VCS
			apiVCS, err := ctl.createVCS(
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
					ResourceID:       generateRandomString("project", 10),
					Name:             "Test VCS Project",
					Key:              "TestVCSSchemaUpdate",
					SchemaChangeType: api.ProjectSchemaChangeTypeSDL,
				},
			)
			a.NoError(err)

			// Create a repository
			ctl.vcsProvider.CreateRepository(test.externalID)

			// Create the branch
			err = ctl.vcsProvider.CreateBranch(test.externalID, "feature/foo")
			a.NoError(err)

			_, err = ctl.createRepository(
				api.RepositoryCreate{
					VCSID:              apiVCS.ID,
					ProjectID:          project.ID,
					Name:               "Test Repository",
					FullPath:           test.repositoryFullPath,
					WebURL:             fmt.Sprintf("%s/%s", ctl.vcsURL, test.repositoryFullPath),
					BranchFilter:       "feature/foo",
					BaseDirectory:      baseDirectory,
					FilePathTemplate:   "{{ENV_ID}}/{{DB_NAME}}##{{VERSION}}##{{TYPE}}##{{DESCRIPTION}}.sql",
					SchemaPathTemplate: "{{ENV_ID}}/.{{DB_NAME}}##LATEST.sql",
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
			instance, err := ctl.addInstance(api.InstanceCreate{
				ResourceID:    generateRandomString("instance", 10),
				EnvironmentID: prodEnvironment.ID,
				Name:          "mysqlInstance",
				Engine:        db.MySQL,
				Host:          "127.0.0.1",
				Port:          strconv.Itoa(mysqlPort),
				Username:      "bytebase",
				Password:      "bytebase",
			})
			a.NoError(err)

			// Create an issue that creates a database
			err = ctl.createDatabase(project, instance, databaseName, "bytebase", nil /* labelMap */)
			a.NoError(err)

			// Simulate Git commits for schema update to create a new table "users".
			schemaFile := fmt.Sprintf("bbtest/prod/.%s##LATEST.sql", databaseName)
			schemaFileContent += "\nCREATE TABLE users (id int, PRIMARY KEY (id));"
			err = ctl.vcsProvider.AddFiles(test.externalID, map[string]string{
				schemaFile: schemaFileContent,
			})
			a.NoError(err)
			err = ctl.vcsProvider.AddCommitsDiff(test.externalID, "1", "2", []vcs.FileDiff{
				{Path: schemaFile, Type: vcs.FileDiffTypeAdded},
			})
			a.NoError(err)
			payload, err := json.Marshal(test.newWebhookPushEvent(nil /* added */, []string{schemaFile}, "1", "2"))
			a.NoError(err)
			err = ctl.vcsProvider.SendWebhookPush(test.externalID, payload)
			a.NoError(err)

			// Get schema update issue
			issues, err := ctl.getIssues(&project.ID, api.IssueOpen)
			a.NoError(err)
			a.Len(issues, 1)
			issue := issues[0]
			status, err := ctl.waitIssuePipeline(issue.ID)
			a.NoError(err)
			a.Equal(api.TaskDone, status)
			issue, err = ctl.getIssue(issue.ID)
			a.NoError(err)
			a.Equal(fmt.Sprintf("[%s] Alter schema", databaseName), issue.Name)
			a.Equal(fmt.Sprintf("Apply schema diff by file prod/.%s##LATEST.sql", databaseName), issue.Description)
			_, err = ctl.patchIssueStatus(
				api.IssueStatusPatch{
					ID:     issue.ID,
					Status: api.IssueDone,
				},
			)
			a.NoError(err)

			// Simulate Git commits for data update to the table "users".
			dataFile := fmt.Sprintf("bbtest/prod/%s##ver2##data##insert_data.sql", databaseName)
			err = ctl.vcsProvider.AddFiles(test.externalID, map[string]string{
				dataFile: `INSERT INTO users (id) VALUES (1);`,
			})
			a.NoError(err)
			err = ctl.vcsProvider.AddCommitsDiff(test.externalID, "2", "3", []vcs.FileDiff{
				{Path: dataFile, Type: vcs.FileDiffTypeAdded},
			})
			a.NoError(err)
			payload, err = json.Marshal(test.newWebhookPushEvent([]string{dataFile}, nil /* modified */, "2", "3"))
			a.NoError(err)
			err = ctl.vcsProvider.SendWebhookPush(test.externalID, payload)
			a.NoError(err)

			// Get data update issue
			issues, err = ctl.getIssues(&project.ID, api.IssueOpen)
			a.NoError(err)
			a.Len(issues, 1)
			issue = issues[0]
			status, err = ctl.waitIssuePipeline(issue.ID)
			a.NoError(err)
			a.Equal(api.TaskDone, status)
			issue, err = ctl.getIssue(issue.ID)
			a.NoError(err)
			a.Equal(fmt.Sprintf("[%s] Change data", databaseName), issue.Name)
			a.Equal(fmt.Sprintf("By VCS files:\n\nprod/%s##ver2##data##insert_data.sql\n", databaseName), issue.Description)
			_, err = ctl.patchIssueStatus(
				api.IssueStatusPatch{
					ID:     issue.ID,
					Status: api.IssueDone,
				},
			)
			a.NoError(err)

			// Query list of tables
			result, err := ctl.query(instance, databaseName, fmt.Sprintf(`
SELECT table_name 
    FROM information_schema.tables 
WHERE table_schema = '%s'; 
`, databaseName))
			a.NoError(err)
			a.Equal(`[["TABLE_NAME"],["VARCHAR"],[["projects"],["users"]],[false]]`, result)

			// Get migration history
			const initialSchema = "SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0;\n" +
				"SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0;\n" +
				"SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS;\n" +
				"SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS;\n"

			const updatedSchema = "SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0;\n" +
				"SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0;\n" +
				"--\n" +
				"-- Table structure for `projects`\n" +
				"--\n" +
				"CREATE TABLE `projects` (\n" +
				"  `id` int NOT NULL,\n" +
				"  PRIMARY KEY (`id`)\n" +
				") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;\n\n" +
				"--\n" +
				"-- Table structure for `users`\n" +
				"--\n" +
				"CREATE TABLE `users` (\n" +
				"  `id` int NOT NULL,\n" +
				"  PRIMARY KEY (`id`)\n" +
				") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;\n\n" +
				"SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS;\n" +
				"SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS;\n"

			const initialSDL = ""
			const updatedSDL = "CREATE TABLE `projects` (\n" +
				"  `id` INT NOT NULL,\n" +
				"  PRIMARY KEY (`id`)\n" +
				") ENGINE=InnoDB DEFAULT CHARACTER SET=UTF8MB4 DEFAULT COLLATE=UTF8MB4_GENERAL_CI;\n\n" +
				"CREATE TABLE `users` (\n" +
				"  `id` INT NOT NULL,\n" +
				"  PRIMARY KEY (`id`)\n" +
				") ENGINE=InnoDB DEFAULT CHARACTER SET=UTF8MB4 DEFAULT COLLATE=UTF8MB4_GENERAL_CI;\n\n"

			histories, err := ctl.getInstanceMigrationHistory(instance.ID, db.MigrationHistoryFind{})
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
					Type:       db.MigrateSDL,
					Status:     db.Done,
					Schema:     updatedSchema,
					SchemaPrev: initialSchema,
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

			// Test SDL format.
			sdlHistory, err := ctl.getInstanceSDLMigrationHistory(instance.ID, histories[1].ID)
			a.NoError(err)
			a.Equal(updatedSDL, sdlHistory.Schema)
			a.Equal(initialSDL, sdlHistory.SchemaPrev)
		})
	}
}
