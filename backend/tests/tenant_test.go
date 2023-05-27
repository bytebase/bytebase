package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/vcs"
	"github.com/bytebase/bytebase/backend/plugin/vcs/github"
	"github.com/bytebase/bytebase/backend/plugin/vcs/gitlab"
	"github.com/bytebase/bytebase/backend/tests/fake"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

var (
	testTenantNumber = 1
	prodTenantNumber = 3
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
	err = ctl.setLicense()
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
	projectUID, err := strconv.Atoi(project.Uid)
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
	prodEnvironment, _, err := ctl.getEnvironment(ctx, "prod")
	a.NoError(err)
	testEnvironment, _, err := ctl.getEnvironment(ctx, "test")
	a.NoError(err)

	// Add the provisioned instances.
	var testInstances []*v1pb.Instance
	var prodInstances []*v1pb.Instance
	for i, testInstanceDir := range testInstanceDirs {
		instance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
			InstanceId: generateRandomString("instance", 10),
			Instance: &v1pb.Instance{
				Title:       fmt.Sprintf("%s-%d", testInstanceName, i),
				Engine:      v1pb.Engine_SQLITE,
				Environment: testEnvironment.Name,
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
				Environment: prodEnvironment.Name,
				DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: prodInstanceDir}},
			},
		})
		a.NoError(err)
		prodInstances = append(prodInstances, instance)
	}

	// Create deployment configuration.
	_, err = ctl.upsertDeploymentConfig(
		api.DeploymentConfigUpsert{
			ProjectID: projectUID,
		},
		deploymentSchedule,
	)
	a.NoError(err)

	// Create issues that create databases.
	databaseName := "testTenantSchemaUpdate"
	for i, testInstance := range testInstances {
		err := ctl.createDatabase(ctx, projectUID, testInstance, databaseName, "", map[string]string{api.TenantLabelKey: fmt.Sprintf("tenant%d", i)})
		a.NoError(err)
	}
	for i, prodInstance := range prodInstances {
		err := ctl.createDatabase(ctx, projectUID, prodInstance, databaseName, "", map[string]string{api.TenantLabelKey: fmt.Sprintf("tenant%d", i)})
		a.NoError(err)
	}

	// Getting databases for each environment.
	databases, err := ctl.getDatabases(api.DatabaseFind{
		ProjectID: &projectUID,
	})
	a.NoError(err)

	var testDatabases []*api.Database
	var prodDatabases []*api.Database
	for _, testInstance := range testInstances {
		testInstanceUID, err := strconv.Atoi(testInstance.Uid)
		a.NoError(err)
		for _, database := range databases {
			if database.Instance.ID == testInstanceUID {
				testDatabases = append(testDatabases, database)
				break
			}
		}
	}
	for _, prodInstance := range prodInstances {
		prodInstanceUID, err := strconv.Atoi(prodInstance.Uid)
		a.NoError(err)
		for _, database := range databases {
			if database.Instance.ID == prodInstanceUID {
				prodDatabases = append(prodDatabases, database)
				break
			}
		}
	}
	a.Equal(testTenantNumber, len(testDatabases))
	a.Equal(prodTenantNumber, len(prodDatabases))

	sheet, err := ctl.createSheet(api.SheetCreate{
		ProjectID:  projectUID,
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
				SheetID:       sheet.ID,
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
	for _, testInstance := range testInstances {
		result, err := ctl.query(testInstance, databaseName, bookTableQuery)
		a.NoError(err)
		a.Equal(bookSchemaSQLResult, result)
	}
	for _, prodInstance := range prodInstances {
		result, err := ctl.query(prodInstance, databaseName, bookTableQuery)
		a.NoError(err)
		a.Equal(bookSchemaSQLResult, result)
	}

	// Query migration history
	var instances []*v1pb.Instance
	instances = append(instances, testInstances...)
	instances = append(instances, prodInstances...)
	hm1 := map[string]bool{}
	for _, instance := range instances {
		instanceUID, err := strconv.Atoi(instance.Uid)
		a.NoError(err)
		histories, err := ctl.getInstanceMigrationHistory(instanceUID, db.MigrationHistoryFind{Database: &databaseName})
		a.NoError(err)
		a.Equal(1, len(histories))
		a.NotEqual(histories[0].Version, "")
		hm1[histories[0].Version] = true
	}
	a.Equal(1, len(hm1))
}

func TestTenantVCS(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                string
		vcsProviderCreator  fake.VCSProviderCreator
		vcsType             vcs.Type
		externalID          string
		repositoryFullPath  string
		newWebhookPushEvent func(gitFile, beforeSHA, afterSHA string) any
	}{
		{
			name:               "GitLab",
			vcsProviderCreator: fake.NewGitLab,
			vcsType:            vcs.GitLab,
			externalID:         "121",
			repositoryFullPath: "test/schemaUpdate",
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
			vcsType:            vcs.GitHub,
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
			projectUID, err := strconv.Atoi(project.Uid)
			a.NoError(err)

			// Create a repository.
			ctl.vcsProvider.CreateRepository(test.externalID)

			// Create the branch
			err = ctl.vcsProvider.CreateBranch(test.externalID, "feature/foo")
			a.NoError(err)

			_, err = ctl.createRepository(
				api.RepositoryCreate{
					VCSID:              apiVCS.ID,
					ProjectID:          projectUID,
					Name:               "Test Repository",
					FullPath:           test.repositoryFullPath,
					WebURL:             fmt.Sprintf("%s/%s", ctl.vcsURL, test.repositoryFullPath),
					BranchFilter:       "feature/foo",
					BaseDirectory:      baseDirectory,
					FilePathTemplate:   "{{VERSION}}##{{TYPE}}##{{DESCRIPTION}}.sql",
					SchemaPathTemplate: ".LATEST.sql",
					ExternalID:         test.externalID,
					AccessToken:        "accessToken1",
					RefreshToken:       "refreshToken1",
				},
			)
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
			prodEnvironment, _, err := ctl.getEnvironment(ctx, "prod")
			a.NoError(err)
			testEnvironment, _, err := ctl.getEnvironment(ctx, "test")
			a.NoError(err)

			// Add the provisioned instances.
			var testInstances []*v1pb.Instance
			var prodInstances []*v1pb.Instance
			for i, testInstanceDir := range testInstanceDirs {
				instance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
					InstanceId: generateRandomString("instance", 10),
					Instance: &v1pb.Instance{
						Title:       fmt.Sprintf("%s-%d", testInstanceName, i),
						Engine:      v1pb.Engine_SQLITE,
						Environment: testEnvironment.Name,
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
						Environment: prodEnvironment.Name,
						DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: prodInstanceDir}},
					},
				})
				a.NoError(err)
				prodInstances = append(prodInstances, instance)
			}

			// Create deployment configuration.
			_, err = ctl.upsertDeploymentConfig(
				api.DeploymentConfigUpsert{
					ProjectID: projectUID,
				},
				deploymentSchedule,
			)
			a.NoError(err)

			// Create issues that create databases.
			databaseName := "testTenantVCSSchemaUpdate"
			for i, testInstance := range testInstances {
				err := ctl.createDatabase(ctx, projectUID, testInstance, databaseName, "", map[string]string{api.TenantLabelKey: fmt.Sprintf("tenant%d", i)})
				a.NoError(err)
			}
			for i, prodInstance := range prodInstances {
				err := ctl.createDatabase(ctx, projectUID, prodInstance, databaseName, "", map[string]string{api.TenantLabelKey: fmt.Sprintf("tenant%d", i)})
				a.NoError(err)
			}

			// Getting databases for each environment.
			databases, err := ctl.getDatabases(api.DatabaseFind{ProjectID: &projectUID})
			a.NoError(err)

			var testDatabases []*api.Database
			var prodDatabases []*api.Database
			for _, testInstance := range testInstances {
				instanceUID, err := strconv.Atoi(testInstance.Uid)
				a.NoError(err)
				for _, database := range databases {
					if database.Instance.ID == instanceUID {
						testDatabases = append(testDatabases, database)
						break
					}
				}
			}
			for _, prodInstance := range prodInstances {
				instanceUID, err := strconv.Atoi(prodInstance.Uid)
				a.NoError(err)
				for _, database := range databases {
					if database.Instance.ID == instanceUID {
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
			payload, err := json.Marshal(test.newWebhookPushEvent(gitFile, "1", "2"))
			a.NoError(err)
			err = ctl.vcsProvider.SendWebhookPush(test.externalID, payload)
			a.NoError(err)

			// Get schema update issue.
			issues, err := ctl.getIssues(&projectUID, api.IssueOpen)
			a.NoError(err)
			a.Len(issues, 1)
			issue := issues[0]

			// Test pipeline stage patch status.
			status, err := ctl.waitIssuePipelineWithStageApproval(ctx, issue.ID)
			a.NoError(err)
			a.Equal(api.TaskDone, status)

			// Query schema.
			for _, testInstance := range testInstances {
				result, err := ctl.query(testInstance, databaseName, bookTableQuery)
				a.NoError(err)
				a.Equal(bookSchemaSQLResult, result)
			}
			for _, prodInstance := range prodInstances {
				result, err := ctl.query(prodInstance, databaseName, bookTableQuery)
				a.NoError(err)
				a.Equal(bookSchemaSQLResult, result)
			}

			// Query migration history
			var instances []*v1pb.Instance
			instances = append(instances, testInstances...)
			instances = append(instances, prodInstances...)
			hm1 := map[string]bool{}
			for _, instance := range instances {
				instanceUID, err := strconv.Atoi(instance.Uid)
				a.NoError(err)

				histories, err := ctl.getInstanceMigrationHistory(
					instanceUID,
					db.MigrationHistoryFind{
						Database: &databaseName,
					},
				)
				a.NoError(err)
				a.Len(histories, 1)
				a.Equal("ver1-ddl", histories[0].Version)
				hm1[histories[0].Version] = true
			}
			a.Len(hm1, 1)
		})
	}
}

func TestTenantDatabaseNameTemplate(t *testing.T) {
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
	err = ctl.setLicense()
	a.NoError(err)

	// Create a project.
	projectID := generateRandomString("project", 10)
	project, err := ctl.projectServiceClient.CreateProject(ctx, &v1pb.CreateProjectRequest{
		Project: &v1pb.Project{
			Name:           fmt.Sprintf("projects/%s", projectID),
			Title:          projectID,
			Key:            projectID,
			TenantMode:     v1pb.TenantMode_TENANT_MODE_ENABLED,
			DbNameTemplate: "{{DB_NAME}}_{{TENANT}}",
		},
		ProjectId: projectID,
	})
	a.NoError(err)
	projectUID, err := strconv.Atoi(project.Uid)
	a.NoError(err)

	// Provision instances.
	instanceRootDir := t.TempDir()
	testInstanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, testInstanceName)
	a.NoError(err)
	prodInstanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, prodInstanceName)
	a.NoError(err)

	prodEnvironment, _, err := ctl.getEnvironment(ctx, "prod")
	a.NoError(err)
	testEnvironment, _, err := ctl.getEnvironment(ctx, "test")
	a.NoError(err)

	// Add the provisioned instances.
	testInstance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance", 10),
		Instance: &v1pb.Instance{
			Title:       testInstanceName,
			Engine:      v1pb.Engine_SQLITE,
			Environment: testEnvironment.Name,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: testInstanceDir}},
		},
	})
	a.NoError(err)
	testInstanceUID, err := strconv.Atoi(testInstance.Uid)
	a.NoError(err)

	prodInstance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance", 10),
		Instance: &v1pb.Instance{
			Title:       testInstanceName,
			Engine:      v1pb.Engine_SQLITE,
			Environment: prodEnvironment.Name,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: prodInstanceDir}},
		},
	})
	a.NoError(err)
	prodInstanceUID, err := strconv.Atoi(prodInstance.Uid)
	a.NoError(err)

	// Create deployment configuration.
	_, err = ctl.upsertDeploymentConfig(
		api.DeploymentConfigUpsert{
			ProjectID: projectUID,
		},
		deploymentSchedule,
	)
	a.NoError(err)

	// Create issues that create databases.
	baseDatabaseName := "testTenant"
	for i := 0; i < testTenantNumber; i++ {
		databaseName := fmt.Sprintf("%s_tenant%d", baseDatabaseName, i)
		err := ctl.createDatabase(ctx, projectUID, testInstance, databaseName, "", map[string]string{api.TenantLabelKey: fmt.Sprintf("tenant%d", i)})
		a.NoError(err)
	}
	for i := 0; i < prodTenantNumber; i++ {
		databaseName := fmt.Sprintf("%s_tenant%d", baseDatabaseName, i)
		err := ctl.createDatabase(ctx, projectUID, prodInstance, databaseName, "", map[string]string{api.TenantLabelKey: fmt.Sprintf("tenant%d", i)})
		a.NoError(err)
	}

	// Getting databases for each environment.
	databases, err := ctl.getDatabases(api.DatabaseFind{
		ProjectID: &projectUID,
	})
	a.NoError(err)

	var testDatabases []*api.Database
	var prodDatabases []*api.Database
	for i := 0; i < testTenantNumber; i++ {
		databaseName := fmt.Sprintf("%s_tenant%d", baseDatabaseName, i)
		for _, database := range databases {
			if database.Instance.ID == testInstanceUID && database.Name == databaseName {
				testDatabases = append(testDatabases, database)
				break
			}
		}
	}
	for i := 0; i < prodTenantNumber; i++ {
		databaseName := fmt.Sprintf("%s_tenant%d", baseDatabaseName, i)
		for _, database := range databases {
			if database.Instance.ID == prodInstanceUID && database.Name == databaseName {
				prodDatabases = append(prodDatabases, database)
				break
			}
		}
	}
	a.Equal(len(testDatabases), testTenantNumber)
	a.Equal(len(prodDatabases), prodTenantNumber)

	sheet, err := ctl.createSheet(api.SheetCreate{
		ProjectID:  projectUID,
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
				SheetID:       sheet.ID,
			},
		},
	})
	a.NoError(err)
	issue, err := ctl.createIssue(api.IssueCreate{
		ProjectID:     projectUID,
		Name:          "update schema for tenants",
		Type:          api.IssueDatabaseSchemaUpdate,
		Description:   "This updates the schema of tenant databases.",
		AssigneeID:    api.SystemBotID,
		CreateContext: string(createContext),
	})
	a.NoError(err)
	status, err := ctl.waitIssuePipeline(ctx, issue.ID)
	a.NoError(err)
	a.Equal(api.TaskDone, status)

	// Query schema.
	for i := 0; i < testTenantNumber; i++ {
		databaseName := fmt.Sprintf("%s_tenant%d", baseDatabaseName, i)
		result, err := ctl.query(testInstance, databaseName, bookTableQuery)
		a.NoError(err)
		a.Equal(bookSchemaSQLResult, result)
	}
	for i := 0; i < prodTenantNumber; i++ {
		databaseName := fmt.Sprintf("%s_tenant%d", baseDatabaseName, i)
		result, err := ctl.query(prodInstance, databaseName, bookTableQuery)
		a.NoError(err)
		a.Equal(bookSchemaSQLResult, result)
	}
}

func TestTenantVCSDatabaseNameTemplate(t *testing.T) {
	tests := []struct {
		name                string
		vcsProviderCreator  fake.VCSProviderCreator
		vcsType             vcs.Type
		externalID          string
		repositoryFullPath  string
		newWebhookPushEvent func(gitFile, beforeSHA, afterSHA string) any
	}{
		{
			name:               "GitLab",
			vcsProviderCreator: fake.NewGitLab,
			vcsType:            vcs.GitLab,
			externalID:         "121",
			repositoryFullPath: "test/schemaUpdate",
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
			vcsType:            vcs.GitHub,
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
			projectID := generateRandomString("project", 10)
			project, err := ctl.projectServiceClient.CreateProject(ctx, &v1pb.CreateProjectRequest{
				Project: &v1pb.Project{
					Name:           fmt.Sprintf("projects/%s", projectID),
					Title:          projectID,
					Key:            projectID,
					TenantMode:     v1pb.TenantMode_TENANT_MODE_ENABLED,
					DbNameTemplate: "{{DB_NAME}}_{{TENANT}}",
				},
				ProjectId: projectID,
			})
			a.NoError(err)
			projectUID, err := strconv.Atoi(project.Uid)
			a.NoError(err)

			// Create a repository.
			ctl.vcsProvider.CreateRepository(test.externalID)

			// Create the branch
			err = ctl.vcsProvider.CreateBranch(test.externalID, "feature/foo")
			a.NoError(err)

			_, err = ctl.createRepository(
				api.RepositoryCreate{
					VCSID:              apiVCS.ID,
					ProjectID:          projectUID,
					Name:               "Test Repository",
					FullPath:           test.repositoryFullPath,
					WebURL:             fmt.Sprintf("%s/%s", ctl.vcsURL, test.repositoryFullPath),
					BranchFilter:       "feature/foo",
					BaseDirectory:      baseDirectory,
					FilePathTemplate:   "{{VERSION}}##{{TYPE}}##{{DESCRIPTION}}.sql",
					SchemaPathTemplate: ".LATEST.sql",
					ExternalID:         test.externalID,
					AccessToken:        "accessToken1",
					RefreshToken:       "refreshToken1",
				},
			)
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
			prodEnvironment, _, err := ctl.getEnvironment(ctx, "prod")
			a.NoError(err)
			testEnvironment, _, err := ctl.getEnvironment(ctx, "test")
			a.NoError(err)

			// Add the provisioned instances.
			var testInstances []*v1pb.Instance
			var prodInstances []*v1pb.Instance
			for i, testInstanceDir := range testInstanceDirs {
				instance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
					InstanceId: generateRandomString("instance", 10),
					Instance: &v1pb.Instance{
						Title:       fmt.Sprintf("%s-%d", testInstanceName, i),
						Engine:      v1pb.Engine_SQLITE,
						Environment: testEnvironment.Name,
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
						Environment: prodEnvironment.Name,
						DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: prodInstanceDir}},
					},
				})
				a.NoError(err)
				prodInstances = append(prodInstances, instance)
			}

			// Create deployment configuration.
			_, err = ctl.upsertDeploymentConfig(
				api.DeploymentConfigUpsert{
					ProjectID: projectUID,
				},
				deploymentSchedule,
			)
			a.NoError(err)

			// Create issues that create databases.
			baseDatabaseName := "testTenantVCSSchemaUpdate"

			for i, testInstance := range testInstances {
				tenant := fmt.Sprintf("tenant%d", i)
				databaseName := baseDatabaseName + "_" + tenant
				err := ctl.createDatabase(ctx, projectUID, testInstance, databaseName, "", map[string]string{api.TenantLabelKey: tenant})
				a.NoError(err)
			}
			for i, prodInstance := range prodInstances {
				tenant := fmt.Sprintf("tenant%d", i)
				databaseName := baseDatabaseName + "_" + tenant
				err := ctl.createDatabase(ctx, projectUID, prodInstance, databaseName, "", map[string]string{api.TenantLabelKey: tenant})
				a.NoError(err)
			}

			// Getting databases for each environment.
			databases, err := ctl.getDatabases(api.DatabaseFind{ProjectID: &projectUID})
			a.NoError(err)

			var testDatabases []*api.Database
			var prodDatabases []*api.Database
			for _, testInstance := range testInstances {
				instanceUID, err := strconv.Atoi(testInstance.Uid)
				a.NoError(err)
				for _, database := range databases {
					if database.Instance.ID == instanceUID {
						testDatabases = append(testDatabases, database)
						break
					}
				}
			}
			for _, prodInstance := range prodInstances {
				instanceUID, err := strconv.Atoi(prodInstance.Uid)
				a.NoError(err)
				for _, database := range databases {
					if database.Instance.ID == instanceUID {
						prodDatabases = append(prodDatabases, database)
						break
					}
				}
			}
			a.Equal(testTenantNumber, len(testDatabases))
			a.Equal(prodTenantNumber, len(prodDatabases))

			// Simulate Git commits.
			gitFile := "bbtest/ver1##migrate##create_a_test_table.sql"
			err = ctl.vcsProvider.AddFiles(test.externalID, map[string]string{gitFile: migrationStatement})
			a.NoError(err)
			err = ctl.vcsProvider.AddCommitsDiff(test.externalID, "1", "2", []vcs.FileDiff{
				{Path: gitFile, Type: vcs.FileDiffTypeAdded},
			})
			a.NoError(err)
			payload, err := json.Marshal(test.newWebhookPushEvent(gitFile, "1", "2"))
			a.NoError(err)
			err = ctl.vcsProvider.SendWebhookPush(test.externalID, payload)
			a.NoError(err)

			// Get schema update issue.
			issues, err := ctl.getIssues(&projectUID, api.IssueOpen)
			a.NoError(err)
			a.Len(issues, 1)
			issue := issues[0]
			status, err := ctl.waitIssuePipeline(ctx, issue.ID)
			a.NoError(err)
			a.Equal(api.TaskDone, status)

			// Query schema.
			for i, testInstance := range testInstances {
				tenant := fmt.Sprintf("tenant%d", i)
				databaseName := baseDatabaseName + "_" + tenant
				result, err := ctl.query(testInstance, databaseName, bookTableQuery)
				a.NoError(err)
				a.Equal(bookSchemaSQLResult, result)
			}
			for i, prodInstance := range prodInstances {
				tenant := fmt.Sprintf("tenant%d", i)
				databaseName := baseDatabaseName + "_" + tenant
				result, err := ctl.query(prodInstance, databaseName, bookTableQuery)
				a.NoError(err)
				a.Equal(bookSchemaSQLResult, result)
			}

			// Query migration history
			hm1 := map[string]bool{}
			hm2 := map[string]bool{}
			for i, instance := range testInstances {
				instanceUID, err := strconv.Atoi(instance.Uid)
				a.NoError(err)
				tenant := fmt.Sprintf("tenant%d", i)
				databaseName := baseDatabaseName + "_" + tenant
				histories, err := ctl.getInstanceMigrationHistory(
					instanceUID,
					db.MigrationHistoryFind{
						Database: &databaseName,
					},
				)
				a.NoError(err)
				a.Len(histories, 1)
				a.Equal("ver1-ddl", histories[0].Version)
				hm1[histories[0].Version] = true
			}
			for i, instance := range prodInstances {
				instanceUID, err := strconv.Atoi(instance.Uid)
				a.NoError(err)
				tenant := fmt.Sprintf("tenant%d", i)
				databaseName := baseDatabaseName + "_" + tenant
				histories, err := ctl.getInstanceMigrationHistory(
					instanceUID,
					db.MigrationHistoryFind{
						Database: &databaseName,
					},
				)
				a.NoError(err)
				a.Len(histories, 1)
				a.Equal("ver1-ddl", histories[0].Version)
				hm2[histories[0].Version] = true
			}

			a.Len(hm1, 1)
			a.Len(hm2, 1)

			// Check latestSchemaFile
			files, err := ctl.vcsProvider.GetFiles(test.externalID, fmt.Sprintf("%s/.LATEST.sql", baseDirectory))
			a.NoError(err)
			a.Len(files, 1)
		})
	}
}

// TestTenantVCSDatabaseNameTemplate_Empty tests the behavior when a tenant
// project has empty database name template where a single commit file should
// match all databases in the project, and create migration issues for all of
// them.
func TestTenantVCSDatabaseNameTemplate_Empty(t *testing.T) {
	tests := []struct {
		name                string
		vcsProviderCreator  fake.VCSProviderCreator
		vcsType             vcs.Type
		externalID          string
		repositoryFullPath  string
		newWebhookPushEvent func(gitFile, beforeSHA, afterSHA string) any
	}{
		{
			name:               "GitLab",
			vcsProviderCreator: fake.NewGitLab,
			vcsType:            vcs.GitLab,
			externalID:         "121",
			repositoryFullPath: "test/schemaUpdate",
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
			vcsType:            vcs.GitHub,
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
			projectUID, err := strconv.Atoi(project.Uid)
			a.NoError(err)

			// Create a repository.
			ctl.vcsProvider.CreateRepository(test.externalID)

			// Create the branch
			err = ctl.vcsProvider.CreateBranch(test.externalID, "feature/foo")
			a.NoError(err)

			_, err = ctl.createRepository(
				api.RepositoryCreate{
					VCSID:              apiVCS.ID,
					ProjectID:          projectUID,
					Name:               "Test Repository",
					FullPath:           test.repositoryFullPath,
					WebURL:             fmt.Sprintf("%s/%s", ctl.vcsURL, test.repositoryFullPath),
					BranchFilter:       "feature/foo",
					BaseDirectory:      baseDirectory,
					FilePathTemplate:   "{{VERSION}}##{{TYPE}}##{{DESCRIPTION}}.sql",
					SchemaPathTemplate: ".LATEST.sql",
					ExternalID:         test.externalID,
					AccessToken:        "accessToken1",
					RefreshToken:       "refreshToken1",
				},
			)
			a.NoError(err)

			// Provision instances.
			instanceRootDir := t.TempDir()

			const testTenantNumber = 2 // We need more than one tenant to test wildcard
			var testInstanceDirs []string
			for i := 0; i < testTenantNumber; i++ {
				instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, fmt.Sprintf("%s-%d", testInstanceName, i))
				a.NoError(err)
				testInstanceDirs = append(testInstanceDirs, instanceDir)
			}
			testEnvironment, _, err := ctl.getEnvironment(ctx, "test")
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
						DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: testInstanceDir}},
					},
				})
				a.NoError(err)
				testInstances = append(testInstances, instance)
			}

			// Create deployment configuration.
			_, err = ctl.upsertDeploymentConfig(
				api.DeploymentConfigUpsert{
					ProjectID: projectUID,
				},
				api.DeploymentSchedule{
					Deployments: []*api.Deployment{
						{
							Name: "Test stage",
							Spec: &api.DeploymentSpec{
								Selector: &api.LabelSelector{
									MatchExpressions: []*api.LabelSelectorRequirement{
										{
											Key:      api.EnvironmentLabelKey,
											Operator: api.InOperatorType,
											Values:   []string{"test"},
										},
									},
								},
							},
						},
					},
				},
			)
			a.NoError(err)

			// Create issues that create databases.
			const baseDatabaseName = "TestTenantVCSDatabaseNameTemplate_Empty"
			for i, testInstance := range testInstances {
				tenant := fmt.Sprintf("tenant%d", i)
				databaseName := baseDatabaseName + "_" + tenant
				err := ctl.createDatabase(ctx, projectUID, testInstance, databaseName, "", nil /* labelMap */)
				a.NoError(err)
			}

			// Getting databases for each environment.
			databases, err := ctl.getDatabases(
				api.DatabaseFind{
					ProjectID: &projectUID,
				},
			)
			a.NoError(err)

			var testDatabases []*api.Database
			for _, testInstance := range testInstances {
				instanceUID, err := strconv.Atoi(testInstance.Uid)
				a.NoError(err)
				for _, database := range databases {
					if database.Instance.ID == instanceUID {
						testDatabases = append(testDatabases, database)
						break
					}
				}
			}
			a.Equal(testTenantNumber, len(testDatabases))

			// Simulate Git commits for schema update.
			gitFile := baseDirectory + "/ver1##migrate##create_a_test_table.sql"
			err = ctl.vcsProvider.AddFiles(test.externalID, map[string]string{gitFile: migrationStatement})
			a.NoError(err)
			err = ctl.vcsProvider.AddCommitsDiff(test.externalID, "1", "2", []vcs.FileDiff{
				{Path: gitFile, Type: vcs.FileDiffTypeAdded},
			})
			a.NoError(err)
			payload, err := json.Marshal(test.newWebhookPushEvent(gitFile, "1", "2"))
			a.NoError(err)
			err = ctl.vcsProvider.SendWebhookPush(test.externalID, payload)
			a.NoError(err)

			// Get schema update issues.
			issues, err := ctl.getIssues(&projectUID, api.IssueOpen)
			a.NoError(err)
			a.Len(issues, 1)
			issue := issues[0]
			status, err := ctl.waitIssuePipeline(ctx, issue.ID)
			a.NoError(err)
			a.Equal(api.TaskDone, status)

			// Query schema.
			for i, testInstance := range testInstances {
				tenant := fmt.Sprintf("tenant%d", i)
				databaseName := baseDatabaseName + "_" + tenant
				result, err := ctl.query(testInstance, databaseName, bookTableQuery)
				a.NoError(err)
				a.Equal(bookSchemaSQLResult, result)
			}

			// Query migration history
			hm := map[string]bool{}
			for i, instance := range testInstances {
				instanceUID, err := strconv.Atoi(instance.Uid)
				a.NoError(err)
				tenant := fmt.Sprintf("tenant%d", i)
				databaseName := baseDatabaseName + "_" + tenant
				histories, err := ctl.getInstanceMigrationHistory(
					instanceUID,
					db.MigrationHistoryFind{
						Database: &databaseName,
					},
				)
				a.NoError(err)
				a.Len(histories, 1)
				a.Equal("ver1-ddl", histories[0].Version)
				hm[histories[0].Version] = true
			}

			a.Len(hm, 1)
		})
	}
}

// TestTenantVCS_YAML tests the behavior when use a YAML file to do DML in a
// tenant project.
func TestTenantVCS_YAML(t *testing.T) {
	tests := []struct {
		name                string
		vcsProviderCreator  fake.VCSProviderCreator
		vcsType             vcs.Type
		externalID          string
		repositoryFullPath  string
		newWebhookPushEvent func(gitFile, beforeSHA, afterSHA string) any
	}{
		{
			name:               "GitLab",
			vcsProviderCreator: fake.NewGitLab,
			vcsType:            vcs.GitLab,
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
			vcsType:            vcs.GitHub,
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
			require.NoError(t, err)
			defer func() {
				_ = ctl.Close(ctx)
			}()

			err = ctl.setLicense()
			require.NoError(t, err)

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
			require.NoError(t, err)

			// Create a tenant project with empty database name template.
			projectID := generateRandomString("project", 10)
			project, err := ctl.projectServiceClient.CreateProject(ctx, &v1pb.CreateProjectRequest{
				Project: &v1pb.Project{
					Name:           fmt.Sprintf("projects/%s", projectID),
					Title:          projectID,
					Key:            projectID,
					TenantMode:     v1pb.TenantMode_TENANT_MODE_ENABLED,
					DbNameTemplate: "{{DB_NAME}}_{{TENANT}}",
				},
				ProjectId: projectID,
			})
			require.NoError(t, err)
			projectUID, err := strconv.Atoi(project.Uid)
			require.NoError(t, err)

			// Create a repository.
			ctl.vcsProvider.CreateRepository(test.externalID)

			// Create the branch
			err = ctl.vcsProvider.CreateBranch(test.externalID, "feature/foo")
			require.NoError(t, err)

			_, err = ctl.createRepository(
				api.RepositoryCreate{
					VCSID:              apiVCS.ID,
					ProjectID:          projectUID,
					Name:               "Test Repository",
					FullPath:           test.repositoryFullPath,
					WebURL:             fmt.Sprintf("%s/%s", ctl.vcsURL, test.repositoryFullPath),
					BranchFilter:       "feature/foo",
					BaseDirectory:      baseDirectory,
					FilePathTemplate:   "{{VERSION}}##{{TYPE}}##{{DESCRIPTION}}.sql",
					SchemaPathTemplate: ".LATEST.sql",
					ExternalID:         test.externalID,
					AccessToken:        "accessToken1",
					RefreshToken:       "refreshToken1",
				},
			)
			require.NoError(t, err)

			// Provision instances.
			instanceRootDir := t.TempDir()

			const testTenantNumber = 2 // We need more than one tenant to test database selection
			var testInstanceDirs []string
			for i := 0; i < testTenantNumber; i++ {
				instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, fmt.Sprintf("%s-%d", testInstanceName, i))
				require.NoError(t, err)
				testInstanceDirs = append(testInstanceDirs, instanceDir)
			}
			testEnvironment, _, err := ctl.getEnvironment(ctx, "test")
			require.NoError(t, err)

			// Add the provisioned instances.
			var testInstances []*v1pb.Instance
			for i, testInstanceDir := range testInstanceDirs {
				instance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
					InstanceId: generateRandomString("instance", 10),
					Instance: &v1pb.Instance{
						Title:       fmt.Sprintf("%s-%d", testInstanceName, i),
						Engine:      v1pb.Engine_SQLITE,
						Environment: testEnvironment.Name,
						DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: testInstanceDir}},
					},
				})
				require.NoError(t, err)
				testInstances = append(testInstances, instance)
			}

			// Create deployment configuration.
			_, err = ctl.upsertDeploymentConfig(
				api.DeploymentConfigUpsert{
					ProjectID: projectUID,
				},
				api.DeploymentSchedule{
					Deployments: []*api.Deployment{
						{
							Name: "Test stage",
							Spec: &api.DeploymentSpec{
								Selector: &api.LabelSelector{
									MatchExpressions: []*api.LabelSelectorRequirement{
										{
											Key:      api.EnvironmentLabelKey,
											Operator: api.InOperatorType,
											Values:   []string{"test"},
										},
									},
								},
							},
						},
					},
				},
			)
			require.NoError(t, err)

			// Create issues that create databases.
			const baseDatabaseName = "TestTenantVCS_YAML"
			for i, testInstance := range testInstances {
				tenant := fmt.Sprintf("tenant%d", i)
				databaseName := baseDatabaseName + "_" + tenant
				err := ctl.createDatabase(ctx, projectUID, testInstance, databaseName, "", nil /* labelMap */)
				require.NoError(t, err)
			}

			// Getting databases for each environment.
			databases, err := ctl.getDatabases(
				api.DatabaseFind{
					ProjectID: &projectUID,
				},
			)
			require.NoError(t, err)

			var testDatabases []*api.Database
			for _, testInstance := range testInstances {
				instanceUID, err := strconv.Atoi(testInstance.Uid)
				require.NoError(t, err)
				for _, database := range databases {
					if database.Instance.ID == instanceUID {
						testDatabases = append(testDatabases, database)
						break
					}
				}
			}
			require.Equal(t, testTenantNumber, len(testDatabases))

			// Simulate Git commits for schema update.
			gitFile1 := baseDirectory + "/ver1##migrate##create_a_test_table.sql"
			err = ctl.vcsProvider.AddFiles(test.externalID, map[string]string{gitFile1: migrationStatement})
			require.NoError(t, err)
			err = ctl.vcsProvider.AddCommitsDiff(test.externalID, "1", "2", []vcs.FileDiff{
				{Path: gitFile1, Type: vcs.FileDiffTypeAdded},
			})
			require.NoError(t, err)
			payload, err := json.Marshal(test.newWebhookPushEvent(gitFile1, "1", "2"))
			require.NoError(t, err)
			err = ctl.vcsProvider.SendWebhookPush(test.externalID, payload)
			require.NoError(t, err)

			// Get schema update issues.
			issues, err := ctl.getIssues(&projectUID, api.IssueOpen)
			require.NoError(t, err)
			require.Len(t, issues, 1)
			status, err := ctl.waitIssuePipeline(ctx, issues[0].ID)
			require.NoError(t, err)
			require.Equal(t, api.TaskDone, status)

			// Simulate Git commits for data update.
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
						databases[0].Name,
					),
				},
			)
			require.NoError(t, err)
			err = ctl.vcsProvider.AddCommitsDiff(test.externalID, "2", "3", []vcs.FileDiff{
				{Path: gitFile2, Type: vcs.FileDiffTypeAdded},
			})
			require.NoError(t, err)
			payload, err = json.Marshal(test.newWebhookPushEvent(gitFile2, "2", "3"))
			require.NoError(t, err)
			err = ctl.vcsProvider.SendWebhookPush(test.externalID, payload)
			require.NoError(t, err)

			// Get data update issues.
			issues, err = ctl.getIssues(&projectUID, api.IssueOpen)
			require.NoError(t, err)
			status, err = ctl.waitIssuePipeline(ctx, issues[0].ID)
			require.NoError(t, err)
			require.Equal(t, api.TaskDone, status)

			// Query migration history, only the database of the first tenant should be touched
			instanceUID0, err := strconv.Atoi(testInstances[0].Uid)
			require.NoError(t, err)
			histories, err := ctl.getInstanceMigrationHistory(
				instanceUID0,
				db.MigrationHistoryFind{
					Database: &databases[0].Name,
				},
			)
			require.NoError(t, err)
			require.Len(t, histories, 2)
			require.Equal(t, "ver2-dml", histories[0].Version)

			instanceUID1, err := strconv.Atoi(testInstances[1].Uid)
			require.NoError(t, err)
			histories, err = ctl.getInstanceMigrationHistory(
				instanceUID1,
				db.MigrationHistoryFind{
					Database: &databases[1].Name,
				},
			)
			require.NoError(t, err)
			require.Len(t, histories, 1)
			require.Equal(t, "ver1-ddl", histories[0].Version)
		})
	}
}
