package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	api "github.com/bytebase/bytebase/backend/legacyapi"
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
	ctx, err := ctl.StartServerWithExternalPg(ctx, &config{
		dataDir:            dataDir,
		vcsProviderCreator: fake.NewGitLab,
	})
	a.NoError(err)
	defer ctl.Close(ctx)

	// Provision an instance.
	instanceRootDir := t.TempDir()
	instanceName := "testInstance1"
	instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, instanceName)
	a.NoError(err)

	instance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance", 10),
		Instance: &v1pb.Instance{
			Title:       instanceName,
			Engine:      v1pb.Engine_SQLITE,
			Environment: "environments/prod",
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: instanceDir, Id: "admin"}},
		},
	})
	a.NoError(err)

	// Create an issue that creates a database.
	databaseName := "testSchemaUpdate"
	err = ctl.createDatabaseV2(ctx, ctl.project, instance, nil /* environment */, databaseName, "", nil /* labelMap */)
	a.NoError(err)

	database, err := ctl.databaseServiceClient.GetDatabase(ctx, &v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Name, databaseName),
	})
	a.NoError(err)

	sheet, err := ctl.sheetServiceClient.CreateSheet(ctx, &v1pb.CreateSheetRequest{
		Parent: ctl.project.Name,
		Sheet: &v1pb.Sheet{
			Title:   "migration statement sheet",
			Content: []byte(migrationStatement),
		},
	})
	a.NoError(err)

	// Create an issue that updates database schema.
	err = ctl.changeDatabase(ctx, ctl.project, database, sheet, v1pb.Plan_ChangeDatabaseConfig_MIGRATE)
	a.NoError(err)

	// Query schema.
	dbMetadata, err := ctl.databaseServiceClient.GetDatabaseSchema(ctx, &v1pb.GetDatabaseSchemaRequest{Name: fmt.Sprintf("%s/schema", database.Name)})
	a.NoError(err)
	a.Equal(wantBookSchema, dbMetadata.Schema)

	sheet, err = ctl.sheetServiceClient.CreateSheet(ctx, &v1pb.CreateSheetRequest{
		Parent: ctl.project.Name,
		Sheet: &v1pb.Sheet{
			Title:   "dataUpdateStatement",
			Content: []byte(dataUpdateStatement),
		},
	})
	a.NoError(err)

	// Create an issue that updates database data.
	err = ctl.changeDatabase(ctx, ctl.project, database, sheet, v1pb.Plan_ChangeDatabaseConfig_DATA)
	a.NoError(err)

	// Get migration history.
	resp, err := ctl.databaseServiceClient.ListChangeHistories(ctx, &v1pb.ListChangeHistoriesRequest{
		Parent: database.Name,
		View:   v1pb.ChangeHistoryView_CHANGE_HISTORY_VIEW_FULL,
	})
	a.NoError(err)
	histories := resp.ChangeHistories
	wantHistories := []*v1pb.ChangeHistory{
		{
			Source:     v1pb.ChangeHistory_UI,
			Type:       v1pb.ChangeHistory_DATA,
			Status:     v1pb.ChangeHistory_DONE,
			Schema:     dumpedSchema,
			PrevSchema: dumpedSchema,
		},
		{
			Source:     v1pb.ChangeHistory_UI,
			Type:       v1pb.ChangeHistory_MIGRATE,
			Status:     v1pb.ChangeHistory_DONE,
			Schema:     dumpedSchema,
			PrevSchema: "",
		},
	}
	a.Equal(len(wantHistories), len(histories))
	for i, history := range histories {
		got := &v1pb.ChangeHistory{
			Source:     history.Source,
			Type:       history.Type,
			Status:     history.Status,
			Schema:     history.Schema,
			PrevSchema: history.PrevSchema,
		}
		want := wantHistories[i]
		a.Equal(want, got)
		a.NotEqual(history.Version, "")
	}
}

func TestSimpleVCS(t *testing.T) {
	tests := []struct {
		name                string
		vcsProviderCreator  fake.VCSProviderCreator
		vcsType             v1pb.VCSProvider_Type
		externalID          string
		repositoryFullPath  string
		newWebhookPushEvent func(added, modified [][]string, beforeSHA, afterSHA string) any
	}{
		{
			name:               "GitLab",
			vcsProviderCreator: fake.NewGitLab,
			vcsType:            v1pb.VCSProvider_GITLAB,
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
			vcsType:            v1pb.VCSProvider_GITHUB,
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
			vcsType:            v1pb.VCSProvider_BITBUCKET,
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
			ctx, err := ctl.StartServerWithExternalPg(ctx, &config{
				dataDir:            t.TempDir(),
				vcsProviderCreator: test.vcsProviderCreator,
			})
			a.NoError(err)
			defer func() {
				_ = ctl.Close(ctx)
			}()

			// Create a VCS.
			evcs, err := ctl.evcsClient.CreateVCSProvider(ctx, &v1pb.CreateVCSProviderRequest{
				VcsProvider: &v1pb.VCSProvider{
					Title:       t.Name(),
					Type:        test.vcsType,
					Url:         ctl.vcsURL,
					AccessToken: "testApplicationSecret",
				},
				VcsProviderId: strings.ToLower(test.vcsType.String()),
			})
			a.NoError(err)
			// Create a repository.
			ctl.vcsProvider.CreateRepository(test.externalID)
			// Create the branch
			err = ctl.vcsProvider.CreateBranch(test.externalID, "feature/foo")
			a.NoError(err)

			oldVcsConnector, err := ctl.vcsConnectorServiceClient.CreateVCSConnector(ctx, &v1pb.CreateVCSConnectorRequest{
				Parent: ctl.project.Name,
				VcsConnector: &v1pb.VCSConnector{
					VcsProvider:   evcs.Name,
					Title:         "Test VCS Connector",
					FullPath:      test.repositoryFullPath,
					WebUrl:        fmt.Sprintf("%s/%s", ctl.vcsURL, test.repositoryFullPath),
					Branch:        "feature/foo",
					BaseDirectory: baseDirectory + "+invalid",
					ExternalId:    test.externalID,
				},
				VcsConnectorId: "default",
			})
			a.NoError(err)
			vcsConnector, err := ctl.vcsConnectorServiceClient.UpdateVCSConnector(ctx, &v1pb.UpdateVCSConnectorRequest{
				VcsConnector: &v1pb.VCSConnector{
					Name:          oldVcsConnector.Name,
					BaseDirectory: baseDirectory,
				},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"base_directory"}},
			})
			a.NoError(err)
			a.Equal(baseDirectory, vcsConnector.BaseDirectory)

			// Provision an instance.
			instanceName := "testInstance1"
			instanceDir, err := ctl.provisionSQLiteInstance(t.TempDir(), instanceName)
			a.NoError(err)

			instance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
				InstanceId: generateRandomString("instance", 10),
				Instance: &v1pb.Instance{
					Title:       instanceName,
					Engine:      v1pb.Engine_SQLITE,
					Environment: "environments/prod",
					Activation:  true,
					DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: instanceDir, Id: "admin"}},
				},
			})
			a.NoError(err)

			// Create an issue that creates a database.
			databaseName := "testVCSSchemaUpdate"
			err = ctl.createDatabaseV2(ctx, ctl.project, instance, nil /* environment */, databaseName, "", nil /* labelMap */)
			a.NoError(err)

			database, err := ctl.databaseServiceClient.GetDatabase(ctx, &v1pb.GetDatabaseRequest{
				Name: fmt.Sprintf("%s/databases/%s", instance.Name, databaseName),
			})
			a.NoError(err)

			// Simulate Git commits for schema update.
			// We create multiple commits in one push event to test for the behavior of creating one issue per database.
			gitFile3 := "bbtest/0003##migrate##create_table_book3.sql"
			err = ctl.vcsProvider.AddFiles(test.externalID, map[string]string{gitFile3: migrationStatement3})
			a.NoError(err)
			gitFile2 := "bbtest/0002##migrate##新建create_table_book2.sql"
			err = ctl.vcsProvider.AddFiles(test.externalID, map[string]string{gitFile2: migrationStatement2})
			a.NoError(err)
			gitFile1 := "bbtest/0001##migrate##😊create_table_book.sql"
			err = ctl.vcsProvider.AddFiles(test.externalID, map[string]string{gitFile1: migrationStatement})
			a.NoError(err)
			// This file is merged from other branch and included in this push event's commits.
			// But it is already merged into the main branch and the commits diff does not contain it.
			// So this file should be excluded when generating the issue.
			gitFileMergeFromOtherBranch := "bbtest/0000##migrate##merge_from_other_branch.sql"
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
			issue, err := ctl.getLastOpenIssue(ctx, ctl.project)
			a.NoError(err)
			err = ctl.waitRollout(ctx, issue.Name, issue.Rollout)
			a.NoError(err)
			a.Equal("Alter schema: 😊create table book", issue.Title)
			a.Equal("By VCS files:\n\n0001##migrate##😊create_table_book.sql\n0002##migrate##新建create_table_book2.sql\n0003##migrate##create_table_book3.sql\n", issue.Description)
			err = ctl.closeIssue(ctx, ctl.project, issue.Name)
			a.NoError(err)

			// Query schema.
			dbMetadata, err := ctl.databaseServiceClient.GetDatabaseSchema(ctx, &v1pb.GetDatabaseSchemaRequest{Name: fmt.Sprintf("%s/schema", database.Name)})
			a.NoError(err)
			a.Equal(want3BookSchema, dbMetadata.Schema)

			// Simulate Git commits for failed data update.
			gitFile4 := "bbtest/0004##data##insert_data.sql"
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
			issue, err = ctl.getLastOpenIssue(ctx, ctl.project)
			a.NoError(err)
			err = ctl.waitRollout(ctx, issue.Name, issue.Rollout)
			a.Error(err)

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

			// TODO(d): waiting for approval finding to complete.
			time.Sleep(2 * time.Second)
			rollout, err := ctl.rolloutServiceClient.GetRollout(ctx, &v1pb.GetRolloutRequest{Name: issue.Rollout})
			a.NoError(err)
			a.Len(rollout.Stages, 1)
			stage := rollout.Stages[0]
			a.Len(stage.Tasks, 1)
			task := stage.Tasks[0]
			// simulate retrying the failed task.
			_, err = ctl.rolloutServiceClient.BatchRunTasks(ctx, &v1pb.BatchRunTasksRequest{
				Parent: fmt.Sprintf("%s/stages/-", rollout.Name),
				Tasks:  []string{task.Name},
			})
			a.NoError(err)

			err = ctl.waitRollout(ctx, issue.Name, issue.Rollout)
			a.NoError(err)
			a.Equal("Change data: insert data", issue.Title)
			a.Equal("By VCS files:\n\n0004##data##insert_data.sql\n", issue.Description)
			err = ctl.closeIssue(ctx, ctl.project, issue.Name)
			a.NoError(err)

			sheet, err := ctl.sheetServiceClient.CreateSheet(ctx, &v1pb.CreateSheetRequest{
				Parent: ctl.project.Name,
				Sheet: &v1pb.Sheet{
					Title:   "migration statement 4 sheet",
					Content: []byte(migrationStatement4),
				},
			})
			a.NoError(err)

			// Schema change from UI.
			// Create an issue that updates database schema.
			err = ctl.changeDatabase(ctx, ctl.project, database, sheet, v1pb.Plan_ChangeDatabaseConfig_MIGRATE)
			a.NoError(err)

			// Get migration history.
			resp, err := ctl.databaseServiceClient.ListChangeHistories(ctx, &v1pb.ListChangeHistoriesRequest{
				Parent: database.Name,
				View:   v1pb.ChangeHistoryView_CHANGE_HISTORY_VIEW_FULL,
			})
			a.NoError(err)
			histories := resp.ChangeHistories
			wantHistories := []*v1pb.ChangeHistory{
				{
					Type:       v1pb.ChangeHistory_MIGRATE,
					Status:     v1pb.ChangeHistory_DONE,
					Schema:     dumpedSchema4,
					PrevSchema: dumpedSchema3,
				},
				{
					Type:       v1pb.ChangeHistory_DATA,
					Status:     v1pb.ChangeHistory_DONE,
					Schema:     dumpedSchema3,
					PrevSchema: dumpedSchema3,
				},
				{
					Type:       v1pb.ChangeHistory_MIGRATE,
					Status:     v1pb.ChangeHistory_DONE,
					Schema:     dumpedSchema3,
					PrevSchema: dumpedSchema2,
				},
				{
					Type:       v1pb.ChangeHistory_MIGRATE,
					Status:     v1pb.ChangeHistory_DONE,
					Schema:     dumpedSchema2,
					PrevSchema: dumpedSchema,
				},
				{
					Type:       v1pb.ChangeHistory_MIGRATE,
					Status:     v1pb.ChangeHistory_DONE,
					Schema:     dumpedSchema,
					PrevSchema: "",
				},
			}
			a.Equal(len(wantHistories), len(histories))
			for i, history := range histories {
				got := &v1pb.ChangeHistory{
					Type:       history.Type,
					Status:     history.Status,
					Schema:     history.Schema,
					PrevSchema: history.PrevSchema,
				}
				want := wantHistories[i]
				a.Equal(got, want)
				a.NotEqual(history.Version, "")
			}

			a.Equal("0004-dml", histories[1].Version)
			a.Equal("0003-ddl", histories[2].Version)
			a.Equal("0002-ddl", histories[3].Version)
			a.Equal("0001-ddl", histories[4].Version)

			_, err = ctl.vcsConnectorServiceClient.DeleteVCSConnector(ctx, &v1pb.DeleteVCSConnectorRequest{Name: vcsConnector.Name})
			a.NoError(err)
		})
	}
}

func TestGetLatestSchema(t *testing.T) {
	tests := []struct {
		name                 string
		dbType               storepb.Engine
		instanceID           string
		databaseName         string
		ddl                  string
		wantRawSchema        string
		wantSDL              string
		wantDatabaseMetadata *v1pb.DatabaseMetadata
	}{
		{
			name:         "MySQL",
			dbType:       storepb.Engine_MYSQL,
			instanceID:   "latest-schema-mysql",
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
			wantDatabaseMetadata: &v1pb.DatabaseMetadata{
				Name:         "instances/latest-schema-mysql/databases/latestSchema/metadata",
				CharacterSet: "utf8mb4",
				Collation:    "utf8mb4_general_ci",
				Schemas: []*v1pb.SchemaMetadata{
					{
						Tables: []*v1pb.TableMetadata{
							{
								Name:      "book",
								Engine:    "InnoDB",
								Collation: "utf8mb4_general_ci",
								DataSize:  16384,
								Columns: []*v1pb.ColumnMetadata{
									{
										Name:       "id",
										Position:   1,
										Nullable:   true,
										HasDefault: true,
										Default: &v1pb.ColumnMetadata_DefaultNull{
											DefaultNull: true,
										},
										Type: "int",
									},
									{
										Name:       "name",
										Position:   2,
										Nullable:   true,
										Type:       "text",
										HasDefault: true,
										Default: &v1pb.ColumnMetadata_DefaultNull{
											DefaultNull: true,
										},
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
			dbType:       storepb.Engine_POSTGRES,
			instanceID:   "latest-schema-postgres",
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
			wantDatabaseMetadata: &v1pb.DatabaseMetadata{
				Name:         "instances/latest-schema-postgres/databases/latestSchema/metadata",
				CharacterSet: "UTF8",
				Collation:    "en_US.UTF-8",
				Schemas: []*v1pb.SchemaMetadata{
					{
						Name: "public",
						Tables: []*v1pb.TableMetadata{
							{
								Name:     "book",
								DataSize: 8192,
								Columns: []*v1pb.ColumnMetadata{
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
	ctx, err := ctl.StartServerWithExternalPg(ctx, &config{
		dataDir:            t.TempDir(),
		vcsProviderCreator: fake.NewGitLab,
	})
	a.NoError(err)
	defer func() {
		_ = ctl.Close(ctx)
	}()
	environmentName := t.Name()
	environment, err := ctl.environmentServiceClient.CreateEnvironment(ctx,
		&v1pb.CreateEnvironmentRequest{
			Environment:   &v1pb.Environment{Title: environmentName},
			EnvironmentId: strings.ToLower(environmentName),
		})
	a.NoError(err)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dbPort := getTestPort()
			switch test.dbType {
			case storepb.Engine_POSTGRES:
				stopInstance := postgres.SetupTestInstance(pgBinDir, t.TempDir(), dbPort)
				defer stopInstance()
			case storepb.Engine_MYSQL:
				stopInstance := mysql.SetupTestInstance(t, dbPort, mysqlBinDir)
				defer stopInstance()
			default:
				a.FailNow("unsupported db type")
			}

			// Add an instance.
			var instance *v1pb.Instance
			switch test.dbType {
			case storepb.Engine_POSTGRES:
				instance, err = ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
					InstanceId: test.instanceID,
					Instance: &v1pb.Instance{
						Title:       test.name,
						Engine:      v1pb.Engine_POSTGRES,
						Environment: environment.Name,
						Activation:  true,
						DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: "/tmp", Port: strconv.Itoa(dbPort), Username: "root", Id: "admin"}},
					},
				})
			case storepb.Engine_MYSQL:
				instance, err = ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
					InstanceId: test.instanceID,
					Instance: &v1pb.Instance{
						Title:       "mysqlInstance",
						Engine:      v1pb.Engine_MYSQL,
						Environment: environment.Name,
						Activation:  true,
						DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: "127.0.0.1", Port: strconv.Itoa(dbPort), Username: "root", Id: "admin"}},
					},
				})
			default:
				a.FailNow("unsupported db type")
			}
			a.NoError(err)

			err = ctl.createDatabaseV2(ctx, ctl.project, instance, nil, test.databaseName, "root", nil /* labelMap */)
			a.NoError(err)

			database, err := ctl.databaseServiceClient.GetDatabase(ctx, &v1pb.GetDatabaseRequest{
				Name: fmt.Sprintf("%s/databases/%s", instance.Name, test.databaseName),
			})
			a.NoError(err)

			ddlSheet, err := ctl.sheetServiceClient.CreateSheet(ctx, &v1pb.CreateSheetRequest{
				Parent: ctl.project.Name,
				Sheet: &v1pb.Sheet{
					Title:   "test ddl",
					Content: []byte(test.ddl),
				},
			})
			a.NoError(err)

			// Create an issue that updates database schema.
			err = ctl.changeDatabase(ctx, ctl.project, database, ddlSheet, v1pb.Plan_ChangeDatabaseConfig_MIGRATE)
			a.NoError(err)

			latestSchema, err := ctl.databaseServiceClient.GetDatabaseSchema(ctx, &v1pb.GetDatabaseSchemaRequest{
				Name: fmt.Sprintf("%s/schema", database.Name),
			})
			a.NoError(err)
			a.Equal(test.wantRawSchema, latestSchema.Schema)
			if test.dbType == storepb.Engine_MYSQL {
				latestSchemaSDL, err := ctl.databaseServiceClient.GetDatabaseSchema(ctx, &v1pb.GetDatabaseSchemaRequest{
					Name:      fmt.Sprintf("%s/schema", database.Name),
					SdlFormat: true,
				})
				a.NoError(err)
				a.Equal(test.wantSDL, latestSchemaSDL.Schema)
			}
			latestSchemaMetadata, err := ctl.databaseServiceClient.GetDatabaseMetadata(ctx, &v1pb.GetDatabaseMetadataRequest{
				Name: fmt.Sprintf("%s/metadata", database.Name),
				View: v1pb.DatabaseMetadataView_DATABASE_METADATA_VIEW_FULL,
			})
			a.NoError(err)
			diff := cmp.Diff(test.wantDatabaseMetadata, latestSchemaMetadata, protocmp.Transform())
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
	ctx, err := ctl.StartServerWithExternalPg(ctx, &config{
		dataDir:            dataDir,
		vcsProviderCreator: fake.NewGitLab,
	})
	a.NoError(err)
	defer ctl.Close(ctx)

	// Provision an instance.
	instanceRootDir := t.TempDir()
	instanceName := "testInstance1"
	instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, instanceName)
	a.NoError(err)

	// Add an instance.
	instance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance", 10),
		Instance: &v1pb.Instance{
			Title:       instanceName,
			Engine:      v1pb.Engine_SQLITE,
			Environment: "environments/prod",
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: instanceDir, Id: "admin"}},
		},
	})
	a.NoError(err)

	// Create an issue that creates a database.
	databaseName := "testSchemaUpdate"
	err = ctl.createDatabaseV2(ctx, ctl.project, instance, nil, databaseName, "", nil /* labelMap */)
	a.NoError(err)

	database, err := ctl.databaseServiceClient.GetDatabase(ctx, &v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Name, databaseName),
	})
	a.NoError(err)

	sheet, err := ctl.sheetServiceClient.CreateSheet(ctx, &v1pb.CreateSheetRequest{
		Parent: ctl.project.Name,
		Sheet: &v1pb.Sheet{
			Title:   "migration statement sheet",
			Content: []byte(migrationStatement),
		},
	})
	a.NoError(err)

	// Create an issue that updates database schema.
	plan, err := ctl.rolloutServiceClient.CreatePlan(ctx, &v1pb.CreatePlanRequest{
		Parent: ctl.project.Name,
		Plan: &v1pb.Plan{
			Steps: []*v1pb.Plan_Step{
				{
					Specs: []*v1pb.Plan_Spec{
						{
							Id: uuid.NewString(),
							Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
								ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
									Target: database.Name,
									Sheet:  sheet.Name,
									Type:   v1pb.Plan_ChangeDatabaseConfig_MIGRATE,
								},
							},
						},
					},
				},
			},
		},
	})
	a.NoError(err)
	issue, err := ctl.issueServiceClient.CreateIssue(ctx, &v1pb.CreateIssueRequest{
		Parent: ctl.project.Name,
		Issue: &v1pb.Issue{
			Type:        v1pb.Issue_DATABASE_CHANGE,
			Title:       fmt.Sprintf("change database %s", database.Name),
			Description: fmt.Sprintf("change database %s", database.Name),
			Plan:        plan.Name,
			Assignee:    fmt.Sprintf("users/%s", api.SystemBotEmail),
		},
	})
	a.NoError(err)
	rollout, err := ctl.rolloutServiceClient.CreateRollout(ctx, &v1pb.CreateRolloutRequest{Parent: ctl.project.Name, Rollout: &v1pb.Rollout{Plan: plan.Name}})
	a.NoError(err)

	// Skip the task.
	for _, stage := range rollout.Stages {
		for _, task := range stage.Tasks {
			_, err := ctl.rolloutServiceClient.BatchSkipTasks(ctx, &v1pb.BatchSkipTasksRequest{
				Parent: stage.Name,
				Tasks:  []string{task.Name},
				Reason: "skip it!",
			})
			a.NoError(err)
		}
	}

	err = ctl.waitRollout(ctx, issue.Name, rollout.Name)
	a.NoError(err)

	// Query schema.
	dbMetadata, err := ctl.databaseServiceClient.GetDatabaseSchema(ctx, &v1pb.GetDatabaseSchemaRequest{Name: fmt.Sprintf("%s/schema", database.Name)})
	a.NoError(err)
	a.Equal("", dbMetadata.Schema)
}
