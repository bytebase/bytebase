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
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"

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
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

var (
	bookDataQuery     = `SELECT * FROM book;`
	bookDataSQLResult = &v1pb.QueryResult{
		ColumnNames:     []string{"id", "name"},
		ColumnTypeNames: []string{"INTEGER", "TEXT"},
		Masked:          []bool{false, false},
		Sensitive:       []bool{false, false},
		Rows: []*v1pb.QueryRow{
			{
				Values: []*v1pb.RowValue{
					{Kind: &v1pb.RowValue_Int64Value{Int64Value: 1}},
					{Kind: &v1pb.RowValue_StringValue{StringValue: "byte"}},
				},
			},
			{
				Values: []*v1pb.RowValue{
					{Kind: &v1pb.RowValue_Int64Value{Int64Value: 2}},
					{Kind: &v1pb.RowValue_NullValue{NullValue: structpb.NullValue_NULL_VALUE}},
				},
			},
		},
		Statement: "SELECT * FROM book",
	}
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

	// Create a project.
	project, err := ctl.createProject(ctx)
	a.NoError(err)
	projectUID, err := strconv.Atoi(project.Uid)
	a.NoError(err)

	// Provision an instance.
	instanceRootDir := t.TempDir()
	instanceName := "testInstance1"
	instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, instanceName)
	a.NoError(err)

	prodEnvironment, err := ctl.getEnvironment(ctx, "prod")
	a.NoError(err)

	instance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance", 10),
		Instance: &v1pb.Instance{
			Title:       instanceName,
			Engine:      v1pb.Engine_SQLITE,
			Environment: prodEnvironment.Name,
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: instanceDir}},
		},
	})
	a.NoError(err)

	// Create an issue that creates a database.
	databaseName := "testSchemaUpdate"
	err = ctl.createDatabase(ctx, projectUID, instance, databaseName, "", nil /* labelMap */)
	a.NoError(err)

	database, err := ctl.databaseServiceClient.GetDatabase(ctx, &v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Name, databaseName),
	})
	a.NoError(err)
	databaseUID, err := strconv.Atoi(database.Uid)
	a.NoError(err)

	sheet, err := ctl.sheetServiceClient.CreateSheet(ctx, &v1pb.CreateSheetRequest{
		Parent: project.Name,
		Sheet: &v1pb.Sheet{
			Title:      "migration statement sheet",
			Content:    []byte(migrationStatement),
			Visibility: v1pb.Sheet_VISIBILITY_PROJECT,
			Source:     v1pb.Sheet_SOURCE_BYTEBASE_ARTIFACT,
			Type:       v1pb.Sheet_TYPE_SQL,
		},
	})
	a.NoError(err)
	sheetUID, err := strconv.Atoi(strings.TrimPrefix(sheet.Name, fmt.Sprintf("%s/sheets/", project.Name)))
	a.NoError(err)

	// Create an issue that updates database schema.
	createContext, err := json.Marshal(&api.MigrationContext{
		DetailList: []*api.MigrationDetail{
			{
				MigrationType: db.Migrate,
				DatabaseID:    databaseUID,
				SheetID:       sheetUID,
			},
		},
	})
	a.NoError(err)
	issue, err := ctl.createIssue(api.IssueCreate{
		ProjectID:     projectUID,
		Name:          fmt.Sprintf("update schema for database %q", databaseName),
		Type:          api.IssueDatabaseSchemaUpdate,
		Description:   fmt.Sprintf("This updates the schema of database %q.", databaseName),
		AssigneeID:    api.SystemBotID,
		CreateContext: string(createContext),
	})
	a.NoError(err)
	status, err := ctl.waitIssuePipeline(ctx, issue.ID)
	a.NoError(err)
	a.Equal(api.TaskDone, status)

	// Query schema.
	dbMetadata, err := ctl.databaseServiceClient.GetDatabaseSchema(ctx, &v1pb.GetDatabaseSchemaRequest{Name: fmt.Sprintf("%s/schema", database.Name)})
	a.NoError(err)
	a.Equal(wantBookSchema, dbMetadata.Schema)

	sheet, err = ctl.sheetServiceClient.CreateSheet(ctx, &v1pb.CreateSheetRequest{
		Parent: project.Name,
		Sheet: &v1pb.Sheet{
			Title:      "dataUpdateStatement",
			Content:    []byte(dataUpdateStatement),
			Visibility: v1pb.Sheet_VISIBILITY_PROJECT,
			Source:     v1pb.Sheet_SOURCE_BYTEBASE_ARTIFACT,
			Type:       v1pb.Sheet_TYPE_SQL,
		},
	})
	a.NoError(err)
	sheetUID, err = strconv.Atoi(strings.TrimPrefix(sheet.Name, fmt.Sprintf("%s/sheets/", project.Name)))
	a.NoError(err)

	// Create an issue that updates database data.
	createContext, err = json.Marshal(&api.MigrationContext{
		DetailList: []*api.MigrationDetail{
			{
				MigrationType: db.Data,
				DatabaseID:    databaseUID,
				SheetID:       sheetUID,
			},
		},
	})
	a.NoError(err)
	issue, err = ctl.createIssue(api.IssueCreate{
		ProjectID:     projectUID,
		Name:          fmt.Sprintf("update data for database %q", databaseName),
		Type:          api.IssueDatabaseDataUpdate,
		Description:   fmt.Sprintf("This updates the data of database %q.", databaseName),
		AssigneeID:    api.SystemBotID,
		CreateContext: string(createContext),
	})
	a.NoError(err)
	status, err = ctl.waitIssuePipeline(ctx, issue.ID)
	a.NoError(err)
	a.Equal(api.TaskDone, status)

	// Get migration history.
	resp, err := ctl.databaseServiceClient.ListChangeHistories(ctx, &v1pb.ListChangeHistoriesRequest{
		Parent: database.Name,
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
		a.True(proto.Equal(got, want))
		a.NotEqual(history.Version, "")
	}

	// Create a manual backup.
	backup, err := ctl.databaseServiceClient.CreateBackup(ctx, &v1pb.CreateBackupRequest{
		Parent: database.Name,
		Backup: &v1pb.Backup{
			Name:       fmt.Sprintf("%s/backups/name", database.Name),
			BackupType: v1pb.Backup_MANUAL,
		},
	})
	a.NoError(err)
	err = ctl.waitBackup(ctx, database.Name, backup.Name)
	a.NoError(err)

	// Create an issue that creates a database.
	cloneDatabaseName := "testClone"
	err = ctl.cloneDatabaseFromBackup(ctx, projectUID, instance, cloneDatabaseName, backup, nil /* labelMap */)
	a.NoError(err)
	cloneDatabase, err := ctl.databaseServiceClient.GetDatabase(ctx, &v1pb.GetDatabaseRequest{Name: fmt.Sprintf("%s/databases/%s", instance.Name, cloneDatabaseName)})
	a.NoError(err)

	// Query clone database book table data.
	queryResp, err := ctl.sqlServiceClient.Query(ctx, &v1pb.QueryRequest{
		Name: instance.Name, ConnectionDatabase: cloneDatabaseName, Statement: bookDataQuery,
	})
	a.NoError(err)
	a.Equal(1, len(queryResp.Results))
	diff := cmp.Diff(bookDataSQLResult, queryResp.Results[0], protocmp.Transform(), protocmp.IgnoreMessages(&durationpb.Duration{}))
	a.Equal("", diff)

	// Query clone migration history.
	resp, err = ctl.databaseServiceClient.ListChangeHistories(ctx, &v1pb.ListChangeHistoriesRequest{
		Parent: cloneDatabase.Name,
	})
	a.NoError(err)
	histories = resp.ChangeHistories
	wantCloneHistories := []*v1pb.ChangeHistory{
		{
			Source:     v1pb.ChangeHistory_UI,
			Type:       v1pb.ChangeHistory_BRANCH,
			Status:     v1pb.ChangeHistory_DONE,
			Schema:     dumpedSchema,
			PrevSchema: dumpedSchema,
		},
	}
	a.Equal(len(wantCloneHistories), len(histories))
	for i, history := range histories {
		got := &v1pb.ChangeHistory{
			Source:     history.Source,
			Type:       history.Type,
			Status:     history.Status,
			Schema:     history.Schema,
			PrevSchema: history.PrevSchema,
		}
		want := wantCloneHistories[i]
		a.True(proto.Equal(got, want))
	}

	_, err = ctl.sheetServiceClient.SearchSheets(ctx, &v1pb.SearchSheetsRequest{
		Parent: "projects/-",
		Filter: "creator = users/demo@example.com",
	})
	a.NoError(err)
}

func TestVCS1(t *testing.T) {
	tests := []struct {
		name                string
		vcsProviderCreator  fake.VCSProviderCreator
		vcsType             v1pb.ExternalVersionControl_Type
		externalID          string
		repositoryFullPath  string
		newWebhookPushEvent func(added, modified [][]string, beforeSHA, afterSHA string) any
	}{
		{
			name:               "GitLab",
			vcsProviderCreator: fake.NewGitLab,
			vcsType:            v1pb.ExternalVersionControl_GITLAB,
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
			vcsType:            v1pb.ExternalVersionControl_GITHUB,
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
			vcsType:            v1pb.ExternalVersionControl_BITBUCKET,
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
				dataDir:                   t.TempDir(),
				vcsProviderCreator:        test.vcsProviderCreator,
				developmentUseV2Scheduler: true,
			})
			a.NoError(err)
			defer func() {
				_ = ctl.Close(ctx)
			}()
			err = ctl.setLicense()
			a.NoError(err)

			// Create a VCS.
			evcs, err := ctl.evcsClient.CreateExternalVersionControl(ctx, &v1pb.CreateExternalVersionControlRequest{
				ExternalVersionControl: &v1pb.ExternalVersionControl{
					Title:         t.Name(),
					Type:          test.vcsType,
					Url:           ctl.vcsURL,
					ApiUrl:        ctl.vcsProvider.APIURL(ctl.vcsURL),
					ApplicationId: "testApplicationID",
					Secret:        "testApplicationSecret",
				},
			})
			a.NoError(err)

			// Create a project.
			project, err := ctl.createProject(ctx)
			a.NoError(err)
			projectUID, err := strconv.Atoi(project.Uid)
			a.NoError(err)

			// Create a repository.
			ctl.vcsProvider.CreateRepository(test.externalID)

			// Create the branch
			err = ctl.vcsProvider.CreateBranch(test.externalID, "feature/foo")
			a.NoError(err)

			_, err = ctl.projectServiceClient.UpdateProjectGitOpsInfo(ctx, &v1pb.UpdateProjectGitOpsInfoRequest{
				ProjectGitopsInfo: &v1pb.ProjectGitOpsInfo{
					Name:               fmt.Sprintf("%s/gitOpsInfo", project.Name),
					VcsUid:             strings.TrimPrefix(evcs.Name, "externalVersionControls/"),
					Title:              "Test Repository",
					FullPath:           test.repositoryFullPath,
					WebUrl:             fmt.Sprintf("%s/%s", ctl.vcsURL, test.repositoryFullPath),
					BranchFilter:       "feature/foo",
					BaseDirectory:      baseDirectory,
					FilePathTemplate:   "{{ENV_ID}}/{{DB_NAME}}##{{VERSION}}##{{TYPE}}##{{DESCRIPTION}}.sql",
					SchemaPathTemplate: "{{ENV_ID}}/.{{DB_NAME}}##LATEST.sql",
					ExternalId:         test.externalID,
					AccessToken:        "accessToken1",
					RefreshToken:       "refreshToken1",
				},
				AllowMissing: true,
			})
			a.NoError(err)

			// Provision an instance.
			instanceName := "testInstance1"
			instanceDir, err := ctl.provisionSQLiteInstance(t.TempDir(), instanceName)
			a.NoError(err)

			prodEnvironment, err := ctl.getEnvironment(ctx, "prod")
			a.NoError(err)

			instance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
				InstanceId: generateRandomString("instance", 10),
				Instance: &v1pb.Instance{
					Title:       instanceName,
					Engine:      v1pb.Engine_SQLITE,
					Environment: prodEnvironment.Name,
					Activation:  true,
					DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: instanceDir}},
				},
			})
			a.NoError(err)

			// Create an issue that creates a database.
			databaseName := "testVCSSchemaUpdate"
			err = ctl.createDatabaseV2(ctx, instance, project.Name, databaseName, "", nil /* labelMap */)
			a.NoError(err)

			database, err := ctl.databaseServiceClient.GetDatabase(ctx, &v1pb.GetDatabaseRequest{
				Name: fmt.Sprintf("%s/databases/%s", instance.Name, databaseName),
			})
			a.NoError(err)
			databaseUID, err := strconv.Atoi(database.Uid)
			a.NoError(err)

			// Simulate Git commits for schema update.
			// We create multiple commits in one push event to test for the behavior of creating one issue per database.
			gitFile3 := "bbtest/prod/testVCSSchemaUpdate##ver3##migrate##create_table_book3.sql"
			err = ctl.vcsProvider.AddFiles(test.externalID, map[string]string{gitFile3: migrationStatement3})
			a.NoError(err)
			gitFile2 := "bbtest/prod/testVCSSchemaUpdate##ver2##migrate##新建create_table_book2.sql"
			err = ctl.vcsProvider.AddFiles(test.externalID, map[string]string{gitFile2: migrationStatement2})
			a.NoError(err)
			gitFile1 := "bbtest/prod/testVCSSchemaUpdate##ver1##migrate##😊create_table_book.sql"
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
			issues, err := ctl.getIssues(&projectUID, api.IssueOpen)
			a.NoError(err)
			a.Len(issues, 1)
			issue := issues[0]
			err = ctl.waitRollout(ctx, fmt.Sprintf("%s/rollouts/%d", project.Name, issue.Pipeline.ID))
			a.NoError(err)
			issue, err = ctl.getIssue(issue.ID)
			a.NoError(err)
			// TODO(p0ny): expose task DAG list and check the dependency.
			a.Equal(3, len(issue.Pipeline.StageList[0].TaskList))
			a.Equal(api.TaskDatabaseSchemaUpdate, issue.Pipeline.StageList[0].TaskList[0].Type)
			a.Equal("[testVCSSchemaUpdate] Alter schema: 😊create table book", issue.Name)
			a.Equal("By VCS files:\n\nprod/testVCSSchemaUpdate##ver1##migrate##😊create_table_book.sql\nprod/testVCSSchemaUpdate##ver2##migrate##新建create_table_book2.sql\nprod/testVCSSchemaUpdate##ver3##migrate##create_table_book3.sql\n", issue.Description)
			err = ctl.closeIssue(ctx, project.Name, fmt.Sprintf("%s/issues/%d", project.Name, issue.Pipeline.ID))
			a.NoError(err)

			// Query schema.
			dbMetadata, err := ctl.databaseServiceClient.GetDatabaseSchema(ctx, &v1pb.GetDatabaseSchemaRequest{Name: fmt.Sprintf("%s/schema", database.Name)})
			a.NoError(err)
			a.Equal(want3BookSchema, dbMetadata.Schema)

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
			issues, err = ctl.getIssues(&projectUID, api.IssueOpen)
			a.NoError(err)
			a.Len(issues, 1)
			issue = issues[0]
			err = ctl.waitRollout(ctx, fmt.Sprintf("%s/rollouts/%d", project.Name, issue.Pipeline.ID))
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

			// Get data update issue.
			issues, err = ctl.getIssues(&projectUID, api.IssueOpen)
			a.NoError(err)
			a.Len(issues, 1)
			issue = issues[0]

			// TODO(d): waiting for approval finding to complete.
			time.Sleep(2 * time.Second)
			rollout, err := ctl.rolloutServiceClient.GetRollout(ctx, &v1pb.GetRolloutRequest{Name: fmt.Sprintf("%s/rollouts/%d", project.Name, issue.Pipeline.ID)})
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

			err = ctl.waitRollout(ctx, fmt.Sprintf("%s/rollouts/%d", project.Name, issue.Pipeline.ID))
			a.NoError(err)
			issue, err = ctl.getIssue(issue.ID)
			a.NoError(err)
			a.Equal(api.TaskDatabaseDataUpdate, issue.Pipeline.StageList[0].TaskList[0].Type)
			a.Equal("[testVCSSchemaUpdate] Change data: Insert data", issue.Name)
			a.Equal("By VCS files:\n\nprod/testVCSSchemaUpdate##ver4##data##insert_data.sql\n", issue.Description)
			err = ctl.closeIssue(ctx, project.Name, fmt.Sprintf("%s/issues/%d", project.Name, issue.Pipeline.ID))
			a.NoError(err)

			sheet, err := ctl.sheetServiceClient.CreateSheet(ctx, &v1pb.CreateSheetRequest{
				Parent: project.Name,
				Sheet: &v1pb.Sheet{
					Title:      "migration statement 4 sheet",
					Content:    []byte(migrationStatement4),
					Visibility: v1pb.Sheet_VISIBILITY_PROJECT,
					Source:     v1pb.Sheet_SOURCE_BYTEBASE_ARTIFACT,
					Type:       v1pb.Sheet_TYPE_SQL,
				},
			})
			a.NoError(err)
			sheetUID, err := strconv.Atoi(strings.TrimPrefix(sheet.Name, fmt.Sprintf("%s/sheets/", project.Name)))
			a.NoError(err)

			// Schema change from UI.
			// Create an issue that updates database schema.
			createContext, err := json.Marshal(&api.MigrationContext{
				DetailList: []*api.MigrationDetail{
					{
						MigrationType: db.Migrate,
						DatabaseID:    databaseUID,
						SheetID:       sheetUID,
					},
				},
			})
			a.NoError(err)
			issue, err = ctl.createIssue(api.IssueCreate{
				ProjectID:     projectUID,
				Name:          fmt.Sprintf("update schema for database %q", databaseName),
				Type:          api.IssueDatabaseSchemaUpdate,
				Description:   fmt.Sprintf("This updates the schema of database %q.", databaseName),
				AssigneeID:    api.SystemBotID,
				CreateContext: string(createContext),
			})
			a.NoError(err)
			err = ctl.waitRollout(ctx, fmt.Sprintf("%s/rollouts/%d", project.Name, issue.Pipeline.ID))
			a.NoError(err)
			environmentResourceID := strings.TrimPrefix(prodEnvironment.Name, "environments/")
			latestFileName := fmt.Sprintf("%s/%s/.%s##LATEST.sql", baseDirectory, environmentResourceID, databaseName)
			files, err := ctl.vcsProvider.GetFiles(test.externalID, latestFileName)
			a.NoError(err)
			a.Len(files, 1)
			a.Equal(dumpedSchema4, files[latestFileName])

			// Get migration history.
			resp, err := ctl.databaseServiceClient.ListChangeHistories(ctx, &v1pb.ListChangeHistoriesRequest{
				Parent: database.Name,
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

			a.Equal("ver4-dml", histories[1].Version)
			a.Equal("ver3-ddl", histories[2].Version)
			a.Equal("ver2-ddl", histories[3].Version)
			a.Equal("ver1-ddl", histories[4].Version)
		})
	}
}

func TestVCS_SDL_POSTGRES(t *testing.T) {
	// TODO(rebelice): remove skip when support PostgreSQL SDL.
	t.Skip()
	tests := []struct {
		name                string
		vcsProviderCreator  fake.VCSProviderCreator
		vcsType             v1pb.ExternalVersionControl_Type
		externalID          string
		repositoryFullPath  string
		newWebhookPushEvent func(added, modified []string, beforeSHA, afterSHA string) any
	}{
		{
			name:               "GitLab",
			vcsProviderCreator: fake.NewGitLab,
			vcsType:            v1pb.ExternalVersionControl_GITLAB,
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
			vcsType:            v1pb.ExternalVersionControl_GITHUB,
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
			ctx, err := ctl.StartServerWithExternalPg(ctx, &config{
				dataDir:                   t.TempDir(),
				vcsProviderCreator:        test.vcsProviderCreator,
				developmentUseV2Scheduler: true,
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
			evcs, err := ctl.evcsClient.CreateExternalVersionControl(ctx, &v1pb.CreateExternalVersionControlRequest{
				ExternalVersionControl: &v1pb.ExternalVersionControl{
					Title:         t.Name(),
					Type:          test.vcsType,
					Url:           ctl.vcsURL,
					ApiUrl:        ctl.vcsProvider.APIURL(ctl.vcsURL),
					ApplicationId: "testApplicationID",
					Secret:        "testApplicationSecret",
				},
			})
			a.NoError(err)

			// Create a project
			project, err := ctl.createProject(ctx)
			a.NoError(err)
			projectUID, err := strconv.Atoi(project.Uid)
			a.NoError(err)

			// Create a repository
			ctl.vcsProvider.CreateRepository(test.externalID)

			// Create the branch
			err = ctl.vcsProvider.CreateBranch(test.externalID, "feature/foo")
			a.NoError(err)

			_, err = ctl.projectServiceClient.UpdateProjectGitOpsInfo(ctx, &v1pb.UpdateProjectGitOpsInfoRequest{
				ProjectGitopsInfo: &v1pb.ProjectGitOpsInfo{
					Name:               fmt.Sprintf("%s/gitOpsInfo", project.Name),
					VcsUid:             strings.TrimPrefix(evcs.Name, "externalVersionControls/"),
					Title:              "Test Repository",
					FullPath:           test.repositoryFullPath,
					WebUrl:             fmt.Sprintf("%s/%s", ctl.vcsURL, test.repositoryFullPath),
					BranchFilter:       "feature/foo",
					BaseDirectory:      baseDirectory,
					FilePathTemplate:   "{{ENV_ID}}/{{DB_NAME}}##{{VERSION}}##{{TYPE}}##{{DESCRIPTION}}.sql",
					SchemaPathTemplate: "{{ENV_ID}}/.{{DB_NAME}}##LATEST.sql",
					ExternalId:         test.externalID,
					AccessToken:        "accessToken1",
					RefreshToken:       "refreshToken1",
				},
				AllowMissing: true,
			})
			a.NoError(err)

			prodEnvironment, err := ctl.getEnvironment(ctx, "prod")
			a.NoError(err)

			instance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
				InstanceId: generateRandomString("instance", 10),
				Instance: &v1pb.Instance{
					Title:       "pgInstance",
					Engine:      v1pb.Engine_POSTGRES,
					Environment: prodEnvironment.Name,
					Activation:  true,
					DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: "/tmp", Port: strconv.Itoa(pgPort), Username: "bytebase", Password: "bytebase"}},
				},
			})
			a.NoError(err)

			// Create an issue that creates a database
			err = ctl.createDatabaseV2(ctx, instance, project.Name, databaseName, "bytebase", nil /* labelMap */)
			a.NoError(err)

			database, err := ctl.databaseServiceClient.GetDatabase(ctx, &v1pb.GetDatabaseRequest{Name: fmt.Sprintf("%s/databases/%s", instance.Name, databaseName)})
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
			issues, err := ctl.getIssues(&projectUID, api.IssueOpen)
			a.NoError(err)
			a.Len(issues, 1)
			issue := issues[0]
			err = ctl.waitRollout(ctx, fmt.Sprintf("%s/rollouts/%d", project.Name, issue.Pipeline.ID))
			a.NoError(err)
			issue, err = ctl.getIssue(issue.ID)
			a.NoError(err)
			a.Equal("[testVCSSchemaUpdate] Alter schema", issue.Name)
			a.Equal("Apply schema diff by file prod/.testVCSSchemaUpdate##LATEST.sql", issue.Description)
			err = ctl.closeIssue(ctx, project.Name, fmt.Sprintf("%s/issues/%d", project.Name, issue.Pipeline.ID))
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
			issues, err = ctl.getIssues(&projectUID, api.IssueOpen)
			a.NoError(err)
			a.Len(issues, 1)
			issue = issues[0]
			err = ctl.waitRollout(ctx, fmt.Sprintf("%s/rollouts/%d", project.Name, issue.Pipeline.ID))
			a.NoError(err)
			issue, err = ctl.getIssue(issue.ID)
			a.NoError(err)
			a.Equal("[testVCSSchemaUpdate] Change data", issue.Name)
			a.Equal("By VCS files:\n\nprod/testVCSSchemaUpdate##ver2##data##insert_data.sql\n", issue.Description)
			err = ctl.closeIssue(ctx, project.Name, fmt.Sprintf("%s/issues/%d", project.Name, issue.Pipeline.ID))
			a.NoError(err)

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

			// Query list of tables
			dbMetadata, err := ctl.databaseServiceClient.GetDatabaseSchema(ctx, &v1pb.GetDatabaseSchemaRequest{Name: fmt.Sprintf("%s/schema", database.Name)})
			a.NoError(err)
			a.Equal(updatedSchema, dbMetadata.Schema)

			resp, err := ctl.databaseServiceClient.ListChangeHistories(ctx, &v1pb.ListChangeHistoriesRequest{
				Parent: database.Name,
			})
			a.NoError(err)
			histories := resp.ChangeHistories
			wantHistories := []*v1pb.ChangeHistory{
				{
					Type:       v1pb.ChangeHistory_DATA,
					Status:     v1pb.ChangeHistory_DONE,
					Schema:     updatedSchema,
					PrevSchema: updatedSchema,
				},
				{
					Type:       v1pb.ChangeHistory_MIGRATE_SDL,
					Status:     v1pb.ChangeHistory_DONE,
					Schema:     updatedSchema,
					PrevSchema: initialSchema,
				},
				{
					Type:       v1pb.ChangeHistory_MIGRATE,
					Status:     v1pb.ChangeHistory_DONE,
					Schema:     initialSchema,
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
				a.True(proto.Equal(got, want))
				a.NotEqual(history.Version, "")
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
		vcsType               v1pb.ExternalVersionControl_Type
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
			vcsType:            v1pb.ExternalVersionControl_GITLAB,
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
			vcsType:            v1pb.ExternalVersionControl_GITLAB,
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
			vcsType:            v1pb.ExternalVersionControl_GITLAB,
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
			vcsType:            v1pb.ExternalVersionControl_GITLAB,
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
			vcsType:            v1pb.ExternalVersionControl_GITLAB,
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
			vcsType:            v1pb.ExternalVersionControl_GITLAB,
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
			ctx, err := ctl.StartServerWithExternalPg(ctx, &config{
				dataDir:                   t.TempDir(),
				vcsProviderCreator:        test.vcsProviderCreator,
				developmentUseV2Scheduler: true,
			})
			a.NoError(err)
			defer func() {
				_ = ctl.Close(ctx)
			}()

			err = ctl.setLicense()
			a.NoError(err)

			// Create a VCS.
			evcs, err := ctl.evcsClient.CreateExternalVersionControl(ctx, &v1pb.CreateExternalVersionControlRequest{
				ExternalVersionControl: &v1pb.ExternalVersionControl{
					Title:         t.Name(),
					Type:          test.vcsType,
					Url:           ctl.vcsURL,
					ApiUrl:        ctl.vcsProvider.APIURL(ctl.vcsURL),
					ApplicationId: "testApplicationID",
					Secret:        "testApplicationSecret",
				},
			})
			a.NoError(err)

			// Create a project.
			project, err := ctl.createProject(ctx)
			a.NoError(err)
			projectUID, err := strconv.Atoi(project.Uid)
			a.NoError(err)

			// Create a repository.
			ctl.vcsProvider.CreateRepository(externalID)

			// Create the branch.
			err = ctl.vcsProvider.CreateBranch(externalID, branchFilter)
			a.NoError(err)

			_, err = ctl.projectServiceClient.UpdateProjectGitOpsInfo(ctx, &v1pb.UpdateProjectGitOpsInfoRequest{
				ProjectGitopsInfo: &v1pb.ProjectGitOpsInfo{
					Name:               fmt.Sprintf("%s/gitOpsInfo", project.Name),
					VcsUid:             strings.TrimPrefix(evcs.Name, "externalVersionControls/"),
					Title:              "Test Repository",
					FullPath:           repoFullPath,
					WebUrl:             fmt.Sprintf("%s/%s", ctl.vcsURL, repoFullPath),
					BranchFilter:       branchFilter,
					BaseDirectory:      test.baseDirectory,
					FilePathTemplate:   test.filePathTemplate,
					SchemaPathTemplate: "{{ENV_ID}}/.{{DB_NAME}}##LATEST.sql",
					ExternalId:         externalID,
					AccessToken:        "accessToken1",
					RefreshToken:       "refreshToken1",
				},
				AllowMissing: true,
			})
			a.NoError(err)

			environment, err := ctl.environmentServiceClient.CreateEnvironment(ctx,
				&v1pb.CreateEnvironmentRequest{
					Environment:   &v1pb.Environment{Title: test.envName},
					EnvironmentId: strings.ToLower(test.envName),
				})
			a.NoError(err)
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
					Environment: environment.Name,
					Activation:  true,
					DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: instanceDir}},
				},
			})
			a.NoError(err)

			// Create an issue that creates a database.
			err = ctl.createDatabaseV2(ctx, instance, project.Name, dbName, "", nil /* labelMap */)
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
				issues, err := ctl.getIssues(&projectUID, api.IssueOpen)
				a.NoError(err)
				if test.expect[idx] {
					a.Len(issues, 1)
					issue := issues[0]
					err = ctl.waitRollout(ctx, fmt.Sprintf("%s/rollouts/%d", project.Name, issue.Pipeline.ID))
					a.NoError(err)
					err = ctl.closeIssue(ctx, project.Name, fmt.Sprintf("%s/issues/%d", project.Name, issue.Pipeline.ID))
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
		vcsType                 v1pb.ExternalVersionControl_Type
		externalID              string
		repositoryFullPath      string
		getEmptySQLReviewResult func(gitOpsInfo *v1pb.ProjectGitOpsInfo, filePath, rootURL string) *api.VCSSQLReviewResult
		getSQLReviewResult      func(gitOpsInfo *v1pb.ProjectGitOpsInfo, filePath string) *api.VCSSQLReviewResult
	}{
		{
			name:               "GitLab",
			vcsProviderCreator: fake.NewGitLab,
			vcsType:            v1pb.ExternalVersionControl_GITLAB,
			externalID:         "121",
			repositoryFullPath: "test/schemaUpdate",
			getEmptySQLReviewResult: func(gitOpsInfo *v1pb.ProjectGitOpsInfo, filePath, rootURL string) *api.VCSSQLReviewResult {
				pathes := strings.Split(filePath, "/")
				return &api.VCSSQLReviewResult{
					Status: advisor.Warn,
					Content: []string{
						fmt.Sprintf(
							"<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<testsuites name=\"SQL Review\">\n<testsuite name=\"%s\">\n<testcase name=\"[WARN] %s#L1: SQL review is disabled\" classname=\"%s\" file=\"%s#L1\">\n<failure>\nError: Cannot found SQL review policy or instance license. You can configure the SQL review policy on %s/setting/sql-review, and assign license to the instance.\nPlease check the docs at https://www.bytebase.com/docs/reference/error-code/advisor#3\n</failure>\n</testcase>\n</testsuite>\n</testsuites>",
							filePath,
							pathes[len(pathes)-1],
							filePath,
							filePath,
							rootURL,
						),
					},
				}
			},
			getSQLReviewResult: func(gitOpsInfo *v1pb.ProjectGitOpsInfo, filePath string) *api.VCSSQLReviewResult {
				pathes := strings.Split(filePath, "/")
				filename := pathes[len(pathes)-1]
				return &api.VCSSQLReviewResult{
					Status: advisor.Warn,
					Content: []string{
						fmt.Sprintf(
							"<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<testsuites name=\"SQL Review\">\n<testsuite name=\"%s\">\n<testcase name=\"[WARN] %s#L1: column.required\" classname=\"%s\" file=\"%s#L1\">\n<failure>\nError: Table \"book\" requires columns: created_ts, creator_id, updated_ts, updater_id.\nPlease check the docs at https://www.bytebase.com/docs/reference/error-code/advisor#401\n</failure>\n</testcase>\n<testcase name=\"[WARN] %s#L1: column.no-null\" classname=\"%s\" file=\"%s#L1\">\n<failure>\nError: Column \"name\" in \"public\".\"book\" cannot have NULL value.\nPlease check the docs at https://www.bytebase.com/docs/reference/error-code/advisor#402\n</failure>\n</testcase>\n</testsuite>\n</testsuites>",
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
			vcsType:            v1pb.ExternalVersionControl_GITHUB,
			externalID:         "octocat/Hello-World",
			repositoryFullPath: "octocat/Hello-World",
			getEmptySQLReviewResult: func(gitOpsInfo *v1pb.ProjectGitOpsInfo, filePath, rootURL string) *api.VCSSQLReviewResult {
				return &api.VCSSQLReviewResult{
					Status: advisor.Warn,
					Content: []string{
						fmt.Sprintf(
							"::warning file=%s,line=1,col=1,endColumn=2,title=SQL review is disabled (3)::Cannot found SQL review policy or instance license. You can configure the SQL review policy on %s/setting/sql-review, and assign license to the instance%%0ADoc: https://www.bytebase.com/docs/reference/error-code/advisor#3",
							filePath,
							rootURL,
						),
					},
				}
			},
			getSQLReviewResult: func(gitOpsInfo *v1pb.ProjectGitOpsInfo, filePath string) *api.VCSSQLReviewResult {
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
			ctx, err := ctl.StartServerWithExternalPg(ctx, &config{
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
			evcs, err := ctl.evcsClient.CreateExternalVersionControl(ctx, &v1pb.CreateExternalVersionControlRequest{
				ExternalVersionControl: &v1pb.ExternalVersionControl{
					Title:         t.Name(),
					Type:          test.vcsType,
					Url:           ctl.vcsURL,
					ApiUrl:        ctl.vcsProvider.APIURL(ctl.vcsURL),
					ApplicationId: "testApplicationID",
					Secret:        "testApplicationSecret",
				},
			})
			a.NoError(err)

			// Create a project.
			project, err := ctl.createProject(ctx)
			a.NoError(err)
			projectUID, err := strconv.Atoi(project.Uid)
			a.NoError(err)

			prodEnvironment, err := ctl.getEnvironment(ctx, "prod")
			a.NoError(err)

			instance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
				InstanceId: generateRandomString("instance", 10),
				Instance: &v1pb.Instance{
					Title:       "pgInstance",
					Engine:      v1pb.Engine_POSTGRES,
					Environment: prodEnvironment.Name,
					Activation:  true,
					DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: "/tmp", Port: strconv.Itoa(pgPort), Username: "bytebase", Password: "bytebase"}},
				},
			})
			a.NoError(err)

			// Create an issue that creates a database.
			err = ctl.createDatabase(ctx, projectUID, instance, databaseName, "bytebase", nil)
			a.NoError(err)

			// Create a repository.
			ctl.vcsProvider.CreateRepository(test.externalID)

			// Create the branch.
			err = ctl.vcsProvider.CreateBranch(test.externalID, "feature/foo")
			a.NoError(err)

			gitOpsInfo, err := ctl.projectServiceClient.UpdateProjectGitOpsInfo(ctx, &v1pb.UpdateProjectGitOpsInfoRequest{
				ProjectGitopsInfo: &v1pb.ProjectGitOpsInfo{
					Name:               fmt.Sprintf("%s/gitOpsInfo", project.Name),
					VcsUid:             strings.TrimPrefix(evcs.Name, "externalVersionControls/"),
					Title:              "Test Repository",
					FullPath:           test.repositoryFullPath,
					WebUrl:             fmt.Sprintf("%s/%s", ctl.vcsURL, test.repositoryFullPath),
					BranchFilter:       "feature/foo",
					BaseDirectory:      baseDirectory,
					FilePathTemplate:   "{{ENV_ID}}/{{DB_NAME}}##{{VERSION}}##{{TYPE}}##{{DESCRIPTION}}.sql",
					SchemaPathTemplate: "{{ENV_ID}}/.{{DB_NAME}}##LATEST.sql",
					ExternalId:         test.externalID,
					AccessToken:        "accessToken1",
					RefreshToken:       "refreshToken1",
				},
				AllowMissing: true,
			})
			a.NoError(err)
			a.NotNil(gitOpsInfo)
			a.Equal(false, gitOpsInfo.EnableSqlReviewCi)

			resp, err := ctl.projectServiceClient.SetupProjectSQLReviewCI(ctx, &v1pb.SetupSQLReviewCIRequest{
				Name: fmt.Sprintf("%s/gitOpsInfo", project.Name),
			})
			a.NoError(err)
			a.NotEmpty(resp.PullRequestUrl)

			gitOpsInfo, err = ctl.projectServiceClient.GetProjectGitOpsInfo(ctx, &v1pb.GetProjectGitOpsInfoRequest{
				Name: fmt.Sprintf("%s/gitOpsInfo", project.Name),
			})
			a.NoError(err)
			a.Equal(true, gitOpsInfo.EnableSqlReviewCi)

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
			res, err := postVCSSQLReview(ctl, gitOpsInfo, &api.VCSSQLReviewRequest{
				RepositoryID:  gitOpsInfo.ExternalId,
				PullRequestID: fmt.Sprintf("%d", prID),
				WebURL:        gitOpsInfo.WebUrl,
			})
			a.NoError(err)

			emptySQLReview := test.getEmptySQLReviewResult(gitOpsInfo, gitFile, ctl.rootURL)
			a.Equal(emptySQLReview.Status, res.Status)
			a.Equal(emptySQLReview.Content, res.Content)

			// create the SQL review policy then re-trigger the VCS SQL review.
			reviewPolicy, err := prodTemplateSQLReviewPolicyForPostgreSQL()
			a.NoError(err)

			_, err = ctl.orgPolicyServiceClient.CreatePolicy(ctx, &v1pb.CreatePolicyRequest{
				Parent: prodEnvironment.Name,
				Policy: &v1pb.Policy{
					Type: v1pb.PolicyType_SQL_REVIEW,
					Policy: &v1pb.Policy_SqlReviewPolicy{
						SqlReviewPolicy: reviewPolicy,
					},
				},
			})
			a.NoError(err)

			reviewResult, err := postVCSSQLReview(ctl, gitOpsInfo, &api.VCSSQLReviewRequest{
				RepositoryID:  gitOpsInfo.ExternalId,
				PullRequestID: fmt.Sprintf("%d", prID),
				WebURL:        gitOpsInfo.WebUrl,
			})
			a.NoError(err)

			expectResult := test.getSQLReviewResult(gitOpsInfo, gitFile)
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
		vcsType            v1pb.ExternalVersionControl_Type
		vcsProviderCreator fake.VCSProviderCreator
		externalID         string
		repoFullPath       string
		caseList           []testCase
	}

	tests := []vcsTestCase{
		{
			vcsType:            v1pb.ExternalVersionControl_GITLAB,
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
			vcsType:            v1pb.ExternalVersionControl_GITHUB,
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
			ctx, err := ctl.StartServerWithExternalPg(ctx, &config{
				dataDir:            t.TempDir(),
				vcsProviderCreator: vcsTest.vcsProviderCreator,
			})
			a.NoError(err)

			defer func() {
				_ = ctl.Close(ctx)
			}()

			// Create a VCS.
			evcs, err := ctl.evcsClient.CreateExternalVersionControl(ctx, &v1pb.CreateExternalVersionControlRequest{
				ExternalVersionControl: &v1pb.ExternalVersionControl{
					Title:         "testName",
					Type:          vcsTest.vcsType,
					Url:           ctl.vcsURL,
					ApiUrl:        ctl.vcsProvider.APIURL(ctl.vcsURL),
					ApplicationId: "testID",
					Secret:        "testSecret",
				},
			})
			a.NoError(err)

			// Create a project.
			project, err := ctl.createProject(ctx)
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
					_, err = ctl.projectServiceClient.UpdateProjectGitOpsInfo(ctx, &v1pb.UpdateProjectGitOpsInfoRequest{
						ProjectGitopsInfo: &v1pb.ProjectGitOpsInfo{
							Name:               fmt.Sprintf("%s/gitOpsInfo", project.Name),
							VcsUid:             strings.TrimPrefix(evcs.Name, "externalVersionControls/"),
							Title:              "Test Repository",
							FullPath:           vcsTest.repoFullPath,
							WebUrl:             fmt.Sprintf("%s/%s", ctl.vcsURL, vcsTest.repoFullPath),
							BranchFilter:       test.branchFilter,
							BaseDirectory:      "",
							FilePathTemplate:   "{{ENV_ID}}/{{DB_NAME}}##{{VERSION}}##{{TYPE}}##{{DESCRIPTION}}.sql",
							SchemaPathTemplate: "",
							ExternalId:         vcsTest.externalID,
							AccessToken:        "accessToken1",
							RefreshToken:       "refreshToken1",
						},
						AllowMissing: true,
					})

					if test.want {
						a.Error(err)
					} else {
						a.NoError(err)
						_, err = ctl.projectServiceClient.UnsetProjectGitOpsInfo(ctx, &v1pb.UnsetProjectGitOpsInfoRequest{
							Name: fmt.Sprintf("%s/gitOpsInfo", project.Name),
						})
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
func postVCSSQLReview(ctl *controller, gitOpsInfo *v1pb.ProjectGitOpsInfo, request *api.VCSSQLReviewRequest) (*api.VCSSQLReviewResult, error) {
	url := fmt.Sprintf("%s/hook/sql-review/%s", ctl.rootURL, gitOpsInfo.WebhookEndpointId)

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
		wantDatabaseMetadata *v1pb.DatabaseMetadata
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
			wantDatabaseMetadata: &v1pb.DatabaseMetadata{
				Name:         "latestSchema",
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
			wantDatabaseMetadata: &v1pb.DatabaseMetadata{
				Name:         "latestSchema",
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
	err = ctl.setLicense()
	a.NoError(err)
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
			case db.Postgres:
				stopInstance := postgres.SetupTestInstance(t, dbPort, resourceDir)
				defer stopInstance()
			case db.MySQL:
				stopInstance := mysql.SetupTestInstance(t, dbPort, mysqlBinDir)
				defer stopInstance()
			default:
				a.FailNow("unsupported db type")
			}
			project, err := ctl.createProject(ctx)
			a.NoError(err)
			projectUID, err := strconv.Atoi(project.Uid)
			a.NoError(err)

			// Add an instance.
			var instance *v1pb.Instance
			switch test.dbType {
			case db.Postgres:
				instance, err = ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
					InstanceId: generateRandomString("instance", 10),
					Instance: &v1pb.Instance{
						Title:       test.name,
						Engine:      v1pb.Engine_POSTGRES,
						Environment: environment.Name,
						Activation:  true,
						DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: "/tmp", Port: strconv.Itoa(dbPort), Username: "root"}},
					},
				})
			case db.MySQL:
				instance, err = ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
					InstanceId: generateRandomString("instance", 10),
					Instance: &v1pb.Instance{
						Title:       "mysqlInstance",
						Engine:      v1pb.Engine_MYSQL,
						Environment: environment.Name,
						Activation:  true,
						DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: "127.0.0.1", Port: strconv.Itoa(dbPort), Username: "root"}},
					},
				})
			default:
				a.FailNow("unsupported db type")
			}
			a.NoError(err)

			err = ctl.createDatabase(ctx, projectUID, instance, test.databaseName, "root", nil /* labelMap */)
			a.NoError(err)

			database, err := ctl.databaseServiceClient.GetDatabase(ctx, &v1pb.GetDatabaseRequest{
				Name: fmt.Sprintf("%s/databases/%s", instance.Name, test.databaseName),
			})
			a.NoError(err)
			databaseUID, err := strconv.Atoi(database.Uid)
			a.NoError(err)

			ddlSheet, err := ctl.sheetServiceClient.CreateSheet(ctx, &v1pb.CreateSheetRequest{
				Parent: project.Name,
				Sheet: &v1pb.Sheet{
					Title:      "test ddl",
					Content:    []byte(test.ddl),
					Visibility: v1pb.Sheet_VISIBILITY_PROJECT,
					Source:     v1pb.Sheet_SOURCE_BYTEBASE_ARTIFACT,
					Type:       v1pb.Sheet_TYPE_SQL,
				},
			})
			a.NoError(err)
			ddlSheetUID, err := strconv.Atoi(strings.TrimPrefix(ddlSheet.Name, fmt.Sprintf("%s/sheets/", project.Name)))
			a.NoError(err)

			// Create an issue that updates database schema.
			createContext, err := json.Marshal(&api.MigrationContext{
				DetailList: []*api.MigrationDetail{
					{
						MigrationType: db.Migrate,
						DatabaseID:    databaseUID,
						SheetID:       ddlSheetUID,
					},
				},
			})
			a.NoError(err)
			issue, err := ctl.createIssue(api.IssueCreate{
				ProjectID:     projectUID,
				Name:          fmt.Sprintf("update schema for database %q", test.databaseName),
				Type:          api.IssueDatabaseSchemaUpdate,
				Description:   fmt.Sprintf("This updates the schema of database %q.", test.databaseName),
				AssigneeID:    api.SystemBotID,
				CreateContext: string(createContext),
			})
			a.NoError(err)
			status, err := ctl.waitIssuePipeline(ctx, issue.ID)
			a.NoError(err)
			a.Equal(api.TaskDone, status)
			latestSchema, err := ctl.databaseServiceClient.GetDatabaseSchema(ctx, &v1pb.GetDatabaseSchemaRequest{
				Name: fmt.Sprintf("%s/schema", database.Name),
			})
			a.NoError(err)
			a.Equal(test.wantRawSchema, latestSchema.Schema)
			if test.dbType == db.MySQL {
				latestSchemaSDL, err := ctl.databaseServiceClient.GetDatabaseSchema(ctx, &v1pb.GetDatabaseSchemaRequest{
					Name:      fmt.Sprintf("%s/schema", database.Name),
					SdlFormat: true,
				})
				a.NoError(err)
				a.Equal(test.wantSDL, latestSchemaSDL.Schema)
			}
			latestSchemaMetadata, err := ctl.databaseServiceClient.GetDatabaseMetadata(ctx, &v1pb.GetDatabaseMetadataRequest{
				Name: fmt.Sprintf("%s/metadata", database.Name),
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

	// Create a project.
	project, err := ctl.createProject(ctx)
	a.NoError(err)
	projectUID, err := strconv.Atoi(project.Uid)
	a.NoError(err)

	// Provision an instance.
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
			Title:       instanceName,
			Engine:      v1pb.Engine_SQLITE,
			Environment: prodEnvironment.Name,
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: instanceDir}},
		},
	})
	a.NoError(err)

	// Create an issue that creates a database.
	databaseName := "testSchemaUpdate"
	err = ctl.createDatabase(ctx, projectUID, instance, databaseName, "", nil /* labelMap */)
	a.NoError(err)

	database, err := ctl.databaseServiceClient.GetDatabase(ctx, &v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Name, databaseName),
	})
	a.NoError(err)
	databaseUID, err := strconv.Atoi(database.Uid)
	a.NoError(err)

	sheet, err := ctl.sheetServiceClient.CreateSheet(ctx, &v1pb.CreateSheetRequest{
		Parent: project.Name,
		Sheet: &v1pb.Sheet{
			Title:      "migration statement sheet",
			Content:    []byte(migrationStatement),
			Visibility: v1pb.Sheet_VISIBILITY_PROJECT,
			Source:     v1pb.Sheet_SOURCE_BYTEBASE_ARTIFACT,
			Type:       v1pb.Sheet_TYPE_SQL,
		},
	})
	a.NoError(err)
	sheetUID, err := strconv.Atoi(strings.TrimPrefix(sheet.Name, fmt.Sprintf("%s/sheets/", project.Name)))
	a.NoError(err)

	// Create an issue that updates database schema.
	createContext, err := json.Marshal(&api.MigrationContext{
		DetailList: []*api.MigrationDetail{
			{
				MigrationType: db.Migrate,
				DatabaseID:    databaseUID,
				SheetID:       sheetUID,
			},
		},
	})
	a.NoError(err)
	issue, err := ctl.createIssue(api.IssueCreate{
		ProjectID:     projectUID,
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
	}, issue.Pipeline.ID, task.ID)
	a.NoError(err)
	a.Equal(api.TaskDone, task.Status)

	var payload api.TaskDatabaseSchemaUpdatePayload
	err = json.Unmarshal([]byte(task.Payload), &payload)
	a.NoError(err)
	a.Equal(true, payload.Skipped)
	a.Equal(skippedReason, payload.SkippedReason)

	status, err := ctl.waitIssuePipelineWithNoApproval(ctx, issue.ID)
	a.NoError(err)
	a.Equal(api.TaskDone, status)

	// Query schema.
	dbMetadata, err := ctl.databaseServiceClient.GetDatabaseSchema(ctx, &v1pb.GetDatabaseSchemaRequest{Name: fmt.Sprintf("%s/schema", database.Name)})
	a.NoError(err)
	a.Equal("", dbMetadata.Schema)
}

func TestVCS_SDL_MySQL(t *testing.T) {
	tests := []struct {
		name                string
		vcsProviderCreator  fake.VCSProviderCreator
		vcsType             v1pb.ExternalVersionControl_Type
		externalID          string
		repositoryFullPath  string
		newWebhookPushEvent func(added, modified []string, beforeSHA, afterSHA string) any
	}{
		{
			name:               "GitLab",
			vcsProviderCreator: fake.NewGitLab,
			vcsType:            v1pb.ExternalVersionControl_GITLAB,
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
			vcsType:            v1pb.ExternalVersionControl_GITHUB,
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
			ctx, err := ctl.StartServerWithExternalPg(ctx, &config{
				dataDir:                   t.TempDir(),
				vcsProviderCreator:        test.vcsProviderCreator,
				developmentUseV2Scheduler: true,
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
			evcs, err := ctl.evcsClient.CreateExternalVersionControl(ctx, &v1pb.CreateExternalVersionControlRequest{
				ExternalVersionControl: &v1pb.ExternalVersionControl{
					Title:         t.Name(),
					Type:          test.vcsType,
					Url:           ctl.vcsURL,
					ApiUrl:        ctl.vcsProvider.APIURL(ctl.vcsURL),
					ApplicationId: "testApplicationID",
					Secret:        "testApplicationSecret",
				},
			})
			a.NoError(err)

			// Create a project
			projectID := generateRandomString("project", 10)
			project, err := ctl.projectServiceClient.CreateProject(ctx, &v1pb.CreateProjectRequest{
				Project: &v1pb.Project{
					Name:         fmt.Sprintf("projects/%s", projectID),
					Title:        projectID,
					Key:          projectID,
					SchemaChange: v1pb.SchemaChange_SDL,
				},
				ProjectId: projectID,
			})
			a.NoError(err)
			projectUID, err := strconv.Atoi(project.Uid)
			a.NoError(err)

			// Create a repository
			ctl.vcsProvider.CreateRepository(test.externalID)

			// Create the branch
			err = ctl.vcsProvider.CreateBranch(test.externalID, "feature/foo")
			a.NoError(err)

			_, err = ctl.projectServiceClient.UpdateProjectGitOpsInfo(ctx, &v1pb.UpdateProjectGitOpsInfoRequest{
				ProjectGitopsInfo: &v1pb.ProjectGitOpsInfo{
					Name:               fmt.Sprintf("%s/gitOpsInfo", project.Name),
					VcsUid:             strings.TrimPrefix(evcs.Name, "externalVersionControls/"),
					Title:              "Test Repository",
					FullPath:           test.repositoryFullPath,
					WebUrl:             fmt.Sprintf("%s/%s", ctl.vcsURL, test.repositoryFullPath),
					BranchFilter:       "feature/foo",
					BaseDirectory:      baseDirectory,
					FilePathTemplate:   "{{ENV_ID}}/{{DB_NAME}}##{{VERSION}}##{{TYPE}}##{{DESCRIPTION}}.sql",
					SchemaPathTemplate: "{{ENV_ID}}/.{{DB_NAME}}##LATEST.sql",
					ExternalId:         test.externalID,
					AccessToken:        "accessToken1",
					RefreshToken:       "refreshToken1",
				},
				AllowMissing: true,
			})
			a.NoError(err)

			prodEnvironment, err := ctl.getEnvironment(ctx, "prod")
			a.NoError(err)

			// Add an instance
			instance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
				InstanceId: generateRandomString("instance", 10),
				Instance: &v1pb.Instance{
					Title:       "mysqlInstance",
					Engine:      v1pb.Engine_MYSQL,
					Environment: prodEnvironment.Name,
					Activation:  true,
					DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: "127.0.0.1", Port: strconv.Itoa(mysqlPort), Username: "bytebase", Password: "bytebase"}},
				},
			})
			a.NoError(err)

			// Create an issue that creates a database
			err = ctl.createDatabaseV2(ctx, instance, project.Name, databaseName, "bytebase", nil /* labelMap */)
			a.NoError(err)

			database, err := ctl.databaseServiceClient.GetDatabase(ctx, &v1pb.GetDatabaseRequest{Name: fmt.Sprintf("%s/databases/%s", instance.Name, databaseName)})
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
			issues, err := ctl.getIssues(&projectUID, api.IssueOpen)
			a.NoError(err)
			a.Len(issues, 1)
			issue := issues[0]
			err = ctl.waitRollout(ctx, fmt.Sprintf("%s/rollouts/%d", project.Name, issue.Pipeline.ID))
			a.NoError(err)
			issue, err = ctl.getIssue(issue.ID)
			a.NoError(err)
			a.Equal(fmt.Sprintf("[%s] Alter schema", databaseName), issue.Name)
			a.Equal(fmt.Sprintf("Apply schema diff by file prod/.%s##LATEST.sql", databaseName), issue.Description)
			err = ctl.closeIssue(ctx, project.Name, fmt.Sprintf("%s/issues/%d", project.Name, issue.Pipeline.ID))
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
			issues, err = ctl.getIssues(&projectUID, api.IssueOpen)
			a.NoError(err)
			a.Len(issues, 1)
			issue = issues[0]
			err = ctl.waitRollout(ctx, fmt.Sprintf("%s/rollouts/%d", project.Name, issue.Pipeline.ID))
			a.NoError(err)
			issue, err = ctl.getIssue(issue.ID)
			a.NoError(err)
			a.Equal(fmt.Sprintf("[%s] Change data: Insert data", databaseName), issue.Name)
			a.Equal(fmt.Sprintf("By VCS files:\n\nprod/%s##ver2##data##insert_data.sql\n", databaseName), issue.Description)
			err = ctl.closeIssue(ctx, project.Name, fmt.Sprintf("%s/issues/%d", project.Name, issue.Pipeline.ID))
			a.NoError(err)

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

			// Query list of tables
			dbMetadata, err := ctl.databaseServiceClient.GetDatabaseSchema(ctx, &v1pb.GetDatabaseSchemaRequest{Name: fmt.Sprintf("%s/schema", database.Name)})
			a.NoError(err)
			a.Equal(updatedSchema, dbMetadata.Schema)

			resp, err := ctl.databaseServiceClient.ListChangeHistories(ctx, &v1pb.ListChangeHistoriesRequest{
				Parent: database.Name,
			})
			a.NoError(err)
			histories := resp.ChangeHistories
			wantHistories := []*v1pb.ChangeHistory{
				{
					Type:       v1pb.ChangeHistory_DATA,
					Status:     v1pb.ChangeHistory_DONE,
					Schema:     updatedSchema,
					PrevSchema: updatedSchema,
				},
				{
					Type:       v1pb.ChangeHistory_MIGRATE_SDL,
					Status:     v1pb.ChangeHistory_DONE,
					Schema:     updatedSchema,
					PrevSchema: initialSchema,
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

			// Test SDL format.
			sdlHistory, err := ctl.databaseServiceClient.GetChangeHistory(ctx, &v1pb.GetChangeHistoryRequest{
				Name:      histories[1].Name,
				SdlFormat: true,
			})
			a.NoError(err)
			a.Equal(updatedSDL, sdlHistory.Schema)
			a.Equal(initialSDL, sdlHistory.PrevSchema)
		})
	}
}
