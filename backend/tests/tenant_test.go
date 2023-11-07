package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/vcs"
	"github.com/bytebase/bytebase/backend/plugin/vcs/github"
	"github.com/bytebase/bytebase/backend/plugin/vcs/gitlab"
	"github.com/bytebase/bytebase/backend/tests/fake"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

var (
	testTenantNumber = 1
	prodTenantNumber = 2
	testInstanceName = "testInstanceTest"
	prodInstanceName = "testInstanceProd"
)

const baseDirectory = "bbtest"

func TestTenant(t *testing.T) {
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
	projectID := generateRandomString("project", 10)
	project, err := ctl.projectServiceClient.CreateProject(ctx, &v1pb.CreateProjectRequest{
		Project: &v1pb.Project{
			Name:       fmt.Sprintf("projects/%s", projectID),
			Title:      projectID,
			Key:        projectID,
			TenantMode: v1pb.TenantMode_TENANT_MODE_ENABLED,
		},
		ProjectId: projectID,
	})
	a.NoError(err)

	// Provision instances.
	instanceRootDir := t.TempDir()

	var testInstanceDirs []string
	var prodInstanceDirs []string
	for i := 0; i < testTenantNumber; i++ {
		instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, fmt.Sprintf("%s-%d", testInstanceName, i))
		a.NoError(err)
		testInstanceDirs = append(testInstanceDirs, instanceDir)
	}
	for i := 0; i < prodTenantNumber; i++ {
		instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, fmt.Sprintf("%s-%d", prodInstanceName, i))
		a.NoError(err)
		prodInstanceDirs = append(prodInstanceDirs, instanceDir)
	}

	// Add the provisioned instances.
	var testInstances []*v1pb.Instance
	var prodInstances []*v1pb.Instance
	for i, testInstanceDir := range testInstanceDirs {
		instance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
			InstanceId: generateRandomString("instance", 10),
			Instance: &v1pb.Instance{
				Title:       fmt.Sprintf("%s-%d", testInstanceName, i),
				Engine:      v1pb.Engine_SQLITE,
				Environment: "environments/test",
				Activation:  true,
				DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: testInstanceDir}},
			},
		})
		a.NoError(err)
		testInstances = append(testInstances, instance)
	}
	for i, prodInstanceDir := range prodInstanceDirs {
		instance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
			InstanceId: generateRandomString("instance", 10),
			Instance: &v1pb.Instance{
				Title:       fmt.Sprintf("%s-%d", prodInstanceName, i),
				Engine:      v1pb.Engine_SQLITE,
				Environment: "environments/prod",
				Activation:  true,
				DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: prodInstanceDir}},
			},
		})
		a.NoError(err)
		prodInstances = append(prodInstances, instance)
	}

	// Create deployment configuration.
	_, err = ctl.projectServiceClient.UpdateDeploymentConfig(ctx, &v1pb.UpdateDeploymentConfigRequest{
		Config: &v1pb.DeploymentConfig{
			Name:     common.FormatDeploymentConfig(project.Name),
			Schedule: deploySchedule,
		},
	})
	a.NoError(err)

	// Create issues that create databases.
	databaseName := "testTenantSchemaUpdate"
	for i, testInstance := range testInstances {
		err := ctl.createDatabaseV2(ctx, project, testInstance, nil, databaseName, "", map[string]string{api.TenantLabelKey: fmt.Sprintf("tenant%d", i)})
		a.NoError(err)
	}
	for i, prodInstance := range prodInstances {
		err := ctl.createDatabaseV2(ctx, project, prodInstance, nil, databaseName, "", map[string]string{api.TenantLabelKey: fmt.Sprintf("tenant%d", i)})
		a.NoError(err)
	}

	// Getting databases for each environment.
	resp, err := ctl.databaseServiceClient.ListDatabases(ctx, &v1pb.ListDatabasesRequest{
		Parent: "instances/-",
		Filter: fmt.Sprintf(`project == "%s"`, project.Name),
	})
	a.NoError(err)
	databases := resp.Databases

	var testDatabases []*v1pb.Database
	var prodDatabases []*v1pb.Database
	for _, testInstance := range testInstances {
		for _, database := range databases {
			if strings.HasPrefix(database.Name, testInstance.Name) {
				testDatabases = append(testDatabases, database)
				break
			}
		}
	}
	for _, prodInstance := range prodInstances {
		for _, database := range databases {
			if strings.HasPrefix(database.Name, prodInstance.Name) {
				prodDatabases = append(prodDatabases, database)
				break
			}
		}
	}

	a.Equal(testTenantNumber, len(testDatabases))
	a.Equal(prodTenantNumber, len(prodDatabases))

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

	// Create an issue that updates database schema.
	step := &v1pb.Plan_Step{
		Specs: []*v1pb.Plan_Spec{
			{
				Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
					ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
						Target: fmt.Sprintf("%s/deploymentConfigs/default", project.Name),
						Sheet:  sheet.Name,
						Type:   v1pb.Plan_ChangeDatabaseConfig_MIGRATE,
					},
				},
			},
		},
	}
	_, _, _, err = ctl.changeDatabaseWithConfig(ctx, project, []*v1pb.Plan_Step{step})
	a.NoError(err)

	// Query schema.
	for _, testInstance := range testInstances {
		dbMetadata, err := ctl.databaseServiceClient.GetDatabaseSchema(ctx, &v1pb.GetDatabaseSchemaRequest{Name: fmt.Sprintf("%s/databases/%s/schema", testInstance.Name, databaseName)})
		a.NoError(err)
		a.Equal(wantBookSchema, dbMetadata.Schema)
	}
	for _, prodInstance := range prodInstances {
		dbMetadata, err := ctl.databaseServiceClient.GetDatabaseSchema(ctx, &v1pb.GetDatabaseSchemaRequest{Name: fmt.Sprintf("%s/databases/%s/schema", prodInstance.Name, databaseName)})
		a.NoError(err)
		a.Equal(wantBookSchema, dbMetadata.Schema)
	}
}

func TestTenantVCS(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                string
		vcsProviderCreator  fake.VCSProviderCreator
		vcsType             v1pb.ExternalVersionControl_Type
		externalID          string
		repositoryFullPath  string
		newWebhookPushEvent func(gitFiles []string, beforeSHA, afterSHA string) any
	}{
		{
			name:               "GitLab",
			vcsProviderCreator: fake.NewGitLab,
			vcsType:            v1pb.ExternalVersionControl_GITLAB,
			externalID:         "121",
			repositoryFullPath: "test/schemaUpdate",
			newWebhookPushEvent: func(gitFiles []string, beforeSHA, afterSHA string) any {
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
							Timestamp: "2021-01-13T13:14:00Z",
							AddedList: gitFiles,
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
			newWebhookPushEvent: func(gitFiles []string, beforeSHA, afterSHA string) any {
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
							Added: gitFiles,
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
				dataDir:            t.TempDir(),
				vcsProviderCreator: test.vcsProviderCreator,
			})
			a.NoError(err)
			defer func() {
				_ = ctl.Close(ctx)
			}()

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
			projectID := generateRandomString("project", 10)
			project, err := ctl.projectServiceClient.CreateProject(ctx, &v1pb.CreateProjectRequest{
				Project: &v1pb.Project{
					Name:       fmt.Sprintf("projects/%s", projectID),
					Title:      projectID,
					Key:        projectID,
					TenantMode: v1pb.TenantMode_TENANT_MODE_ENABLED,
				},
				ProjectId: projectID,
			})
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
					FilePathTemplate:   "{{VERSION}}##{{TYPE}}##{{DESCRIPTION}}.sql",
					SchemaPathTemplate: ".LATEST.sql",
					ExternalId:         test.externalID,
					AccessToken:        "accessToken1",
					RefreshToken:       "refreshToken1",
				},
				AllowMissing: true,
			})
			a.NoError(err)

			// Provision instances.
			instanceRootDir := t.TempDir()

			var testInstanceDirs []string
			var prodInstanceDirs []string
			for i := 0; i < testTenantNumber; i++ {
				instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, fmt.Sprintf("%s-%d", testInstanceName, i))
				a.NoError(err)
				testInstanceDirs = append(testInstanceDirs, instanceDir)
			}
			for i := 0; i < prodTenantNumber; i++ {
				instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, fmt.Sprintf("%s-%d", prodInstanceName, i))
				a.NoError(err)
				prodInstanceDirs = append(prodInstanceDirs, instanceDir)
			}

			// Add the provisioned instances.
			var testInstances []*v1pb.Instance
			var prodInstances []*v1pb.Instance
			for i, testInstanceDir := range testInstanceDirs {
				instance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
					InstanceId: generateRandomString("instance", 10),
					Instance: &v1pb.Instance{
						Title:       fmt.Sprintf("%s-%d", testInstanceName, i),
						Engine:      v1pb.Engine_SQLITE,
						Environment: "environments/test",
						Activation:  true,
						DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: testInstanceDir}},
					},
				})
				a.NoError(err)
				testInstances = append(testInstances, instance)
			}
			for i, prodInstanceDir := range prodInstanceDirs {
				instance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
					InstanceId: generateRandomString("instance", 10),
					Instance: &v1pb.Instance{
						Title:       fmt.Sprintf("%s-%d", prodInstanceName, i),
						Engine:      v1pb.Engine_SQLITE,
						Environment: "environments/prod",
						Activation:  true,
						DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: prodInstanceDir}},
					},
				})
				a.NoError(err)
				prodInstances = append(prodInstances, instance)
			}

			// Create deployment configuration.
			_, err = ctl.projectServiceClient.UpdateDeploymentConfig(ctx, &v1pb.UpdateDeploymentConfigRequest{
				Config: &v1pb.DeploymentConfig{
					Name:     common.FormatDeploymentConfig(project.Name),
					Schedule: deploySchedule,
				},
			})
			a.NoError(err)

			// Create issues that create databases.
			databaseName := "testTenantVCSSchemaUpdate"
			for i, testInstance := range testInstances {
				err := ctl.createDatabaseV2(ctx, project, testInstance, nil, databaseName, "", map[string]string{api.TenantLabelKey: fmt.Sprintf("tenant%d", i)})
				a.NoError(err)
			}
			for i, prodInstance := range prodInstances {
				err := ctl.createDatabaseV2(ctx, project, prodInstance, nil, databaseName, "", map[string]string{api.TenantLabelKey: fmt.Sprintf("tenant%d", i)})
				a.NoError(err)
			}

			// Getting databases for each environment.
			resp, err := ctl.databaseServiceClient.ListDatabases(ctx, &v1pb.ListDatabasesRequest{
				Parent: "instances/-",
				Filter: fmt.Sprintf(`project == "%s"`, project.Name),
			})
			a.NoError(err)
			databases := resp.Databases

			var testDatabases []*v1pb.Database
			var prodDatabases []*v1pb.Database
			for _, testInstance := range testInstances {
				for _, database := range databases {
					if strings.HasPrefix(database.Name, testInstance.Name) {
						testDatabases = append(testDatabases, database)
						break
					}
				}
			}
			for _, prodInstance := range prodInstances {
				for _, database := range databases {
					if strings.HasPrefix(database.Name, prodInstance.Name) {
						prodDatabases = append(prodDatabases, database)
						break
					}
				}
			}

			a.Equal(len(testDatabases), testTenantNumber)
			a.Equal(len(prodDatabases), prodTenantNumber)

			// Simulate Git commits.
			gitFile := "bbtest/ver1##migrate##create_a_test_table.sql"
			err = ctl.vcsProvider.AddFiles(test.externalID, map[string]string{gitFile: migrationStatement})
			a.NoError(err)
			err = ctl.vcsProvider.AddCommitsDiff(test.externalID, "1", "2", []vcs.FileDiff{
				{Path: gitFile, Type: vcs.FileDiffTypeAdded},
			})
			a.NoError(err)
			payload, err := json.Marshal(test.newWebhookPushEvent([]string{gitFile}, "1", "2"))
			a.NoError(err)
			err = ctl.vcsProvider.SendWebhookPush(test.externalID, payload)
			a.NoError(err)

			// Get schema update issue.
			issue, err := ctl.getLastOpenIssue(ctx, project)
			a.NoError(err)
			err = ctl.waitRollout(ctx, issue.Name, issue.Rollout)
			a.NoError(err)
			err = ctl.closeIssue(ctx, project, issue.Name)
			a.NoError(err)

			// Query schema.
			for _, testInstance := range testInstances {
				dbMetadata, err := ctl.databaseServiceClient.GetDatabaseSchema(ctx, &v1pb.GetDatabaseSchemaRequest{Name: fmt.Sprintf("%s/databases/%s/schema", testInstance.Name, databaseName)})
				a.NoError(err)
				a.Equal(wantBookSchema, dbMetadata.Schema)
			}
			for _, prodInstance := range prodInstances {
				dbMetadata, err := ctl.databaseServiceClient.GetDatabaseSchema(ctx, &v1pb.GetDatabaseSchemaRequest{Name: fmt.Sprintf("%s/databases/%s/schema", prodInstance.Name, databaseName)})
				a.NoError(err)
				a.Equal(wantBookSchema, dbMetadata.Schema)
			}

			// Test a commit with multiple files.
			gitFile2 := "bbtest/ver2##migrate##create_a_test_table.sql"
			gitFile3 := "bbtest/ver3##migrate##create_a_test_table.sql"
			err = ctl.vcsProvider.AddFiles(test.externalID, map[string]string{gitFile2: migrationStatement2, gitFile3: migrationStatement3})
			a.NoError(err)
			err = ctl.vcsProvider.AddCommitsDiff(test.externalID, "2", "3", []vcs.FileDiff{
				{Path: gitFile2, Type: vcs.FileDiffTypeAdded},
				{Path: gitFile3, Type: vcs.FileDiffTypeAdded},
			})
			a.NoError(err)
			payload, err = json.Marshal(test.newWebhookPushEvent([]string{gitFile2, gitFile3}, "2", "3"))
			a.NoError(err)
			err = ctl.vcsProvider.SendWebhookPush(test.externalID, payload)
			a.NoError(err)
			// There are two issues created.
			listIssueResp, err := ctl.issueServiceClient.ListIssues(ctx, &v1pb.ListIssuesRequest{
				Parent: project.Name,
				Filter: "status = OPEN",
			})
			a.NoError(err)
			a.Len(listIssueResp.Issues, 2)
			for i := len(listIssueResp.Issues) - 1; i >= 0; i-- {
				issue = listIssueResp.Issues[i]
				err = ctl.waitRollout(ctx, issue.Name, issue.Rollout)
				a.NoError(err)
			}

			// Query schema.
			for _, testInstance := range testInstances {
				dbMetadata, err := ctl.databaseServiceClient.GetDatabaseSchema(ctx, &v1pb.GetDatabaseSchemaRequest{Name: fmt.Sprintf("%s/databases/%s/schema", testInstance.Name, databaseName)})
				a.NoError(err)
				a.Equal(want3BookSchema, dbMetadata.Schema)
			}
			for _, prodInstance := range prodInstances {
				dbMetadata, err := ctl.databaseServiceClient.GetDatabaseSchema(ctx, &v1pb.GetDatabaseSchemaRequest{Name: fmt.Sprintf("%s/databases/%s/schema", prodInstance.Name, databaseName)})
				a.NoError(err)
				a.Equal(want3BookSchema, dbMetadata.Schema)
			}
		})
	}
}

// TestTenantVCS_YAML tests the behavior when use a YAML file to do DML in a
// tenant project.
func TestTenantVCS_YAML(t *testing.T) {
	a := require.New(t)

	tests := []struct {
		name                string
		vcsProviderCreator  fake.VCSProviderCreator
		vcsType             v1pb.ExternalVersionControl_Type
		externalID          string
		repositoryFullPath  string
		newWebhookPushEvent func(gitFile, beforeSHA, afterSHA string) any
	}{
		{
			name:               "GitLab",
			vcsProviderCreator: fake.NewGitLab,
			vcsType:            v1pb.ExternalVersionControl_GITLAB,
			externalID:         "121",
			repositoryFullPath: "test/dataUpdate",
			newWebhookPushEvent: func(gitFile, beforeSHA, afterSHA string) any {
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
							Timestamp: "2021-01-13T13:14:00Z",
							AddedList: []string{gitFile},
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
			newWebhookPushEvent: func(gitFile, beforeSHA, afterSHA string) any {
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
							Added: []string{gitFile},
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

			// Create a tenant project with empty database name template.
			projectID := generateRandomString("project", 10)
			project, err := ctl.projectServiceClient.CreateProject(ctx, &v1pb.CreateProjectRequest{
				Project: &v1pb.Project{
					Name:       fmt.Sprintf("projects/%s", projectID),
					Title:      projectID,
					Key:        projectID,
					TenantMode: v1pb.TenantMode_TENANT_MODE_ENABLED,
				},
				ProjectId: projectID,
			})
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
					FilePathTemplate:   "{{VERSION}}##{{TYPE}}##{{DESCRIPTION}}.sql",
					SchemaPathTemplate: ".LATEST.sql",
					ExternalId:         test.externalID,
					AccessToken:        "accessToken1",
					RefreshToken:       "refreshToken1",
				},
				AllowMissing: true,
			})
			a.NoError(err)

			// Provision instances.
			instanceRootDir := t.TempDir()

			const testTenantNumber = 2 // We need more than one tenant to test database selection
			var testInstanceDirs []string
			for i := 0; i < testTenantNumber; i++ {
				instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, fmt.Sprintf("%s-%d", testInstanceName, i))
				a.NoError(err)
				testInstanceDirs = append(testInstanceDirs, instanceDir)
			}
			testEnvironment, err := ctl.getEnvironment(ctx, "test")
			a.NoError(err)

			// Add the provisioned instances.
			var testInstances []*v1pb.Instance
			for i, testInstanceDir := range testInstanceDirs {
				instance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
					InstanceId: generateRandomString("instance", 10),
					Instance: &v1pb.Instance{
						Title:       fmt.Sprintf("%s-%d", testInstanceName, i),
						Engine:      v1pb.Engine_SQLITE,
						Environment: testEnvironment.Name,
						Activation:  true,
						DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: testInstanceDir}},
					},
				})
				a.NoError(err)
				testInstances = append(testInstances, instance)
			}

			// Create deployment configuration.
			_, err = ctl.projectServiceClient.UpdateDeploymentConfig(ctx, &v1pb.UpdateDeploymentConfigRequest{
				Config: &v1pb.DeploymentConfig{
					Name: common.FormatDeploymentConfig(project.Name),
					Schedule: &v1pb.Schedule{
						Deployments: []*v1pb.ScheduleDeployment{
							{
								Title: "Test stage",
								Spec: &v1pb.DeploymentSpec{
									LabelSelector: &v1pb.LabelSelector{
										MatchExpressions: []*v1pb.LabelSelectorRequirement{
											{
												Key:      api.EnvironmentLabelKey,
												Operator: v1pb.OperatorType_OPERATOR_TYPE_IN,
												Values:   []string{"test"},
											},
										},
									},
								},
							},
						},
					},
				},
			})
			a.NoError(err)

			// Create issues that create databases.
			const baseDatabaseName = "TestTenantVCS_YAML"
			for i, testInstance := range testInstances {
				tenant := fmt.Sprintf("tenant%d", i)
				databaseName := baseDatabaseName + "_" + tenant
				err := ctl.createDatabaseV2(ctx, project, testInstance, nil /* environment */, databaseName, "", nil /* labelMap */)
				a.NoError(err)
			}

			// Getting databases for each environment.
			resp, err := ctl.databaseServiceClient.ListDatabases(ctx, &v1pb.ListDatabasesRequest{
				Parent: "instances/-",
				Filter: fmt.Sprintf(`project == "%s"`, project.Name),
			})
			a.NoError(err)
			databases := resp.Databases

			var testDatabases []*v1pb.Database
			for _, testInstance := range testInstances {
				for _, database := range databases {
					if strings.HasPrefix(database.Name, testInstance.Name) {
						testDatabases = append(testDatabases, database)
						break
					}
				}
			}
			a.Equal(testTenantNumber, len(testDatabases))

			// Simulate Git commits for schema update.
			gitFile1 := baseDirectory + "/ver1##migrate##create_a_test_table.sql"
			err = ctl.vcsProvider.AddFiles(test.externalID, map[string]string{gitFile1: migrationStatement})
			a.NoError(err)
			err = ctl.vcsProvider.AddCommitsDiff(test.externalID, "1", "2", []vcs.FileDiff{
				{Path: gitFile1, Type: vcs.FileDiffTypeAdded},
			})
			a.NoError(err)
			payload, err := json.Marshal(test.newWebhookPushEvent(gitFile1, "1", "2"))
			a.NoError(err)
			err = ctl.vcsProvider.SendWebhookPush(test.externalID, payload)
			a.NoError(err)

			// Get schema update issues.
			issue, err := ctl.getLastOpenIssue(ctx, project)
			a.NoError(err)
			err = ctl.waitRollout(ctx, issue.Name, issue.Rollout)
			a.NoError(err)

			// Simulate Git commits for data update.
			database0Name := "TestTenantVCS_YAML_tenant0"
			gitFile2 := baseDirectory + "/ver2##data##insert_a_new_row.yml"
			err = ctl.vcsProvider.AddFiles(
				test.externalID,
				map[string]string{
					gitFile2: fmt.Sprintf(`
databases:
  - name: %s
statement: |
  INSERT INTO book (name) VALUES ('Star Wars')
`,
						database0Name,
					),
				},
			)
			a.NoError(err)
			err = ctl.vcsProvider.AddCommitsDiff(test.externalID, "2", "3", []vcs.FileDiff{
				{Path: gitFile2, Type: vcs.FileDiffTypeAdded},
			})
			a.NoError(err)
			payload, err = json.Marshal(test.newWebhookPushEvent(gitFile2, "2", "3"))
			a.NoError(err)
			err = ctl.vcsProvider.SendWebhookPush(test.externalID, payload)
			a.NoError(err)

			// Get data update issues.
			issue, err = ctl.getLastOpenIssue(ctx, project)
			a.NoError(err)
			err = ctl.waitRollout(ctx, issue.Name, issue.Rollout)
			a.NoError(err)
		})
	}
}
