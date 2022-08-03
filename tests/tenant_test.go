package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/vcs"
	"github.com/bytebase/bytebase/plugin/vcs/github"
	"github.com/bytebase/bytebase/plugin/vcs/gitlab"
	"github.com/bytebase/bytebase/tests/fake"
)

var (
	stagingTenantNumber = 1
	prodTenantNumber    = 3
	stagingInstanceName = "testInstanceStaging"
	prodInstanceName    = "testInstanceProd"
)

const baseDirectory = "bbtest"

func TestTenant(t *testing.T) {
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
		Name:       "Test Project",
		Key:        "TestSchemaUpdate",
		TenantMode: api.TenantModeTenant,
	})
	a.NoError(err)

	// Provision instances.
	instanceRootDir := t.TempDir()

	var stagingInstanceDirs []string
	var prodInstanceDirs []string
	for i := 0; i < stagingTenantNumber; i++ {
		instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, fmt.Sprintf("%s-%d", stagingInstanceName, i))
		a.NoError(err)
		stagingInstanceDirs = append(stagingInstanceDirs, instanceDir)
	}
	for i := 0; i < prodTenantNumber; i++ {
		instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, fmt.Sprintf("%s-%d", prodInstanceName, i))
		a.NoError(err)
		prodInstanceDirs = append(prodInstanceDirs, instanceDir)
	}
	environments, err := ctl.getEnvironments()
	a.NoError(err)
	stagingEnvironment, err := findEnvironment(environments, "Staging")
	a.NoError(err)
	prodEnvironment, err := findEnvironment(environments, "Prod")
	a.NoError(err)

	// Add the provisioned instances.
	var stagingInstances []*api.Instance
	var prodInstances []*api.Instance
	for i, stagingInstanceDir := range stagingInstanceDirs {
		instance, err := ctl.addInstance(api.InstanceCreate{
			EnvironmentID: stagingEnvironment.ID,
			Name:          fmt.Sprintf("%s-%d", stagingInstanceName, i),
			Engine:        db.SQLite,
			Host:          stagingInstanceDir,
		})
		a.NoError(err)
		stagingInstances = append(stagingInstances, instance)
	}
	for i, prodInstanceDir := range prodInstanceDirs {
		instance, err := ctl.addInstance(api.InstanceCreate{
			EnvironmentID: prodEnvironment.ID,
			Name:          fmt.Sprintf("%s-%d", prodInstanceName, i),
			Engine:        db.SQLite,
			Host:          prodInstanceDir,
		})
		a.NoError(err)
		prodInstances = append(prodInstances, instance)
	}

	// Set up label values for tenants.
	// Prod and staging are using the same tenant values. Use prodInstancesNumber because it's larger than stagingInstancesNumber.
	var tenants []string
	for i := 0; i < prodTenantNumber; i++ {
		tenants = append(tenants, fmt.Sprintf("tenant%d", i))
	}
	err = ctl.addLabelValues(api.TenantLabelKey, tenants)
	a.NoError(err)

	// Create deployment configuration.
	_, err = ctl.upsertDeploymentConfig(
		api.DeploymentConfigUpsert{
			ProjectID: project.ID,
		},
		deploymentSchedule,
	)
	a.NoError(err)

	// Create issues that create databases.
	databaseName := "testTenantSchemaUpdate"
	for i, stagingInstance := range stagingInstances {
		err := ctl.createDatabase(project, stagingInstance, databaseName, "", map[string]string{api.TenantLabelKey: fmt.Sprintf("tenant%d", i)})
		a.NoError(err)
	}
	for i, prodInstance := range prodInstances {
		err := ctl.createDatabase(project, prodInstance, databaseName, "", map[string]string{api.TenantLabelKey: fmt.Sprintf("tenant%d", i)})
		a.NoError(err)
	}

	// Getting databases for each environment.
	databases, err := ctl.getDatabases(api.DatabaseFind{
		ProjectID: &project.ID,
	})
	a.NoError(err)

	var stagingDatabases []*api.Database
	var prodDatabases []*api.Database
	for _, stagingInstance := range stagingInstances {
		for _, database := range databases {
			if database.Instance.ID == stagingInstance.ID {
				stagingDatabases = append(stagingDatabases, database)
				break
			}
		}
	}
	for _, prodInstance := range prodInstances {
		for _, database := range databases {
			if database.Instance.ID == prodInstance.ID {
				prodDatabases = append(prodDatabases, database)
				break
			}
		}
	}
	a.Equal(stagingTenantNumber, len(stagingDatabases))
	a.Equal(prodTenantNumber, len(prodDatabases))

	// Create an issue that updates database schema.
	createContext, err := json.Marshal(&api.UpdateSchemaContext{
		MigrationType: db.Migrate,
		DetailList: []*api.UpdateSchemaDetail{
			{
				DatabaseName: databaseName,
				Statement:    migrationStatement,
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
	for _, stagingInstance := range stagingInstances {
		result, err := ctl.query(stagingInstance, databaseName, bookTableQuery)
		a.NoError(err)
		a.Equal(bookSchemaSQLResult, result)
	}
	for _, prodInstance := range prodInstances {
		result, err := ctl.query(prodInstance, databaseName, bookTableQuery)
		a.NoError(err)
		a.Equal(bookSchemaSQLResult, result)
	}

	// Query migration history
	var instances []*api.Instance
	instances = append(instances, stagingInstances...)
	instances = append(instances, prodInstances...)
	hm1 := map[string]bool{}
	hm2 := map[string]bool{}
	for _, instance := range instances {
		histories, err := ctl.getInstanceMigrationHistory(db.MigrationHistoryFind{ID: &instance.ID, Database: &databaseName})
		a.NoError(err)
		a.Equal(2, len(histories))
		a.NotEqual(histories[0].Version, "")
		a.NotEqual(histories[1].Version, "")
		hm1[histories[0].Version] = true
		hm2[histories[1].Version] = true
	}
	a.Equal(1, len(hm1))
	a.Equal(1, len(hm2))
}

func TestTenantVCS(t *testing.T) {
	tests := []struct {
		name                string
		vcsProviderCreator  fake.VCSProviderCreator
		vcsType             vcs.Type
		externalID          string
		repositoryFullPath  string
		newWebhookPushEvent func(gitFile string) interface{}
	}{
		{
			name:               "GitLab",
			vcsProviderCreator: fake.NewGitLab,
			vcsType:            vcs.GitLabSelfHost,
			externalID:         "121",
			repositoryFullPath: "test/schemaUpdate",
			newWebhookPushEvent: func(gitFile string) interface{} {
				return gitlab.WebhookPushEvent{
					ObjectKind: gitlab.WebhookPush,
					Ref:        "refs/heads/feature/foo",
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
			vcsType:            vcs.GitHubCom,
			externalID:         "octocat/Hello-World",
			repositoryFullPath: "octocat/Hello-World",
			newWebhookPushEvent: func(gitFile string) interface{} {
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
							Added: []string{gitFile},
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
			defer func() { _ = ctl.Close(ctx) }()

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
					Name:       "Test VCS Project",
					Key:        "TestVCSSchemaUpdate",
					TenantMode: api.TenantModeTenant,
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
					FilePathTemplate:   "{{DB_NAME}}__{{VERSION}}__{{TYPE}}__{{DESCRIPTION}}.sql",
					SchemaPathTemplate: ".{{DB_NAME}}__LATEST.sql",
					ExternalID:         test.externalID,
					AccessToken:        "accessToken1",
					RefreshToken:       "refreshToken1",
				},
			)
			a.NoError(err)

			// Provision instances.
			instanceRootDir := t.TempDir()

			var stagingInstanceDirs []string
			var prodInstanceDirs []string
			for i := 0; i < stagingTenantNumber; i++ {
				instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, fmt.Sprintf("%s-%d", stagingInstanceName, i))
				a.NoError(err)
				stagingInstanceDirs = append(stagingInstanceDirs, instanceDir)
			}
			for i := 0; i < prodTenantNumber; i++ {
				instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, fmt.Sprintf("%s-%d", prodInstanceName, i))
				a.NoError(err)
				prodInstanceDirs = append(prodInstanceDirs, instanceDir)
			}
			environments, err := ctl.getEnvironments()
			a.NoError(err)
			stagingEnvironment, err := findEnvironment(environments, "Staging")
			a.NoError(err)
			prodEnvironment, err := findEnvironment(environments, "Prod")
			a.NoError(err)

			// Add the provisioned instances.
			var stagingInstances []*api.Instance
			var prodInstances []*api.Instance
			for i, stagingInstanceDir := range stagingInstanceDirs {
				instance, err := ctl.addInstance(
					api.InstanceCreate{
						EnvironmentID: stagingEnvironment.ID,
						Name:          fmt.Sprintf("%s-%d", stagingInstanceName, i),
						Engine:        db.SQLite,
						Host:          stagingInstanceDir,
					},
				)
				a.NoError(err)
				stagingInstances = append(stagingInstances, instance)
			}
			for i, prodInstanceDir := range prodInstanceDirs {
				instance, err := ctl.addInstance(
					api.InstanceCreate{
						EnvironmentID: prodEnvironment.ID,
						Name:          fmt.Sprintf("%s-%d", prodInstanceName, i),
						Engine:        db.SQLite,
						Host:          prodInstanceDir,
					},
				)
				a.NoError(err)
				prodInstances = append(prodInstances, instance)
			}

			// Set up label values for tenants.
			// Prod and staging are using the same tenant values. Use prodInstancesNumber because it's larger than stagingInstancesNumber.
			var tenants []string
			for i := 0; i < prodTenantNumber; i++ {
				tenants = append(tenants, fmt.Sprintf("tenant%d", i))
			}
			err = ctl.addLabelValues(api.TenantLabelKey, tenants)
			a.NoError(err)

			// Create deployment configuration.
			_, err = ctl.upsertDeploymentConfig(
				api.DeploymentConfigUpsert{
					ProjectID: project.ID,
				},
				deploymentSchedule,
			)
			a.NoError(err)

			// Create issues that create databases.
			databaseName := "testTenantVCSSchemaUpdate"
			for i, stagingInstance := range stagingInstances {
				err := ctl.createDatabase(project, stagingInstance, databaseName, "", map[string]string{api.TenantLabelKey: fmt.Sprintf("tenant%d", i)})
				a.NoError(err)
			}
			for i, prodInstance := range prodInstances {
				err := ctl.createDatabase(project, prodInstance, databaseName, "", map[string]string{api.TenantLabelKey: fmt.Sprintf("tenant%d", i)})
				a.NoError(err)
			}

			// Getting databases for each environment.
			databases, err := ctl.getDatabases(api.DatabaseFind{ProjectID: &project.ID})
			a.NoError(err)

			var stagingDatabases []*api.Database
			var prodDatabases []*api.Database
			for _, stagingInstance := range stagingInstances {
				for _, database := range databases {
					if database.Instance.ID == stagingInstance.ID {
						stagingDatabases = append(stagingDatabases, database)
						break
					}
				}
			}
			for _, prodInstance := range prodInstances {
				for _, database := range databases {
					if database.Instance.ID == prodInstance.ID {
						prodDatabases = append(prodDatabases, database)
						break
					}
				}
			}
			a.Equal(len(stagingDatabases), stagingTenantNumber)
			a.Equal(len(prodDatabases), prodTenantNumber)

			// Simulate Git commits.
			gitFile := "bbtest/testTenantVCSSchemaUpdate__ver1__migrate__create_a_test_table.sql"
			err = ctl.vcsProvider.AddFiles(test.externalID, map[string]string{gitFile: migrationStatement})
			a.NoError(err)

			payload, err := json.Marshal(test.newWebhookPushEvent(gitFile))
			a.NoError(err)
			err = ctl.vcsProvider.SendWebhookPush(test.externalID, payload)
			a.NoError(err)

			// Get schema update issue.
			openStatus := []api.IssueStatus{api.IssueOpen}
			issues, err := ctl.getIssues(
				api.IssueFind{
					ProjectID:  &project.ID,
					StatusList: &openStatus,
				},
			)
			a.NoError(err)
			a.Len(issues, 1)
			issue := issues[0]

			// Test pipeline stage patch status.
			status, err := ctl.waitIssuePipelineWithStageApproval(issue.ID)
			a.NoError(err)
			a.Equal(api.TaskDone, status)

			// Query schema.
			for _, stagingInstance := range stagingInstances {
				result, err := ctl.query(stagingInstance, databaseName, bookTableQuery)
				a.NoError(err)
				a.Equal(bookSchemaSQLResult, result)
			}
			for _, prodInstance := range prodInstances {
				result, err := ctl.query(prodInstance, databaseName, bookTableQuery)
				a.NoError(err)
				a.Equal(bookSchemaSQLResult, result)
			}

			// Query migration history
			var instances []*api.Instance
			instances = append(instances, stagingInstances...)
			instances = append(instances, prodInstances...)
			hm1 := map[string]bool{}
			hm2 := map[string]bool{}
			for _, instance := range instances {
				histories, err := ctl.getInstanceMigrationHistory(
					db.MigrationHistoryFind{
						ID:       &instance.ID,
						Database: &databaseName,
					},
				)
				a.NoError(err)
				a.Len(histories, 2)
				a.Equal(histories[0].Version, "ver1")
				a.NotEqual(histories[1].Version, "")
				hm1[histories[0].Version] = true
				hm2[histories[1].Version] = true
			}
			a.Len(hm1, 1)
			a.Len(hm2, 1)
		})
	}
}

func TestTenantDatabaseNameTemplate(t *testing.T) {
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
		Name:           "Test Project",
		Key:            "TestSchemaUpdate",
		TenantMode:     api.TenantModeTenant,
		DBNameTemplate: "{{DB_NAME}}_{{TENANT}}",
	})
	a.NoError(err)

	// Provision instances.
	instanceRootDir := t.TempDir()
	stagingInstanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, stagingInstanceName)
	a.NoError(err)
	prodInstanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, prodInstanceName)
	a.NoError(err)

	environments, err := ctl.getEnvironments()
	a.NoError(err)
	stagingEnvironment, err := findEnvironment(environments, "Staging")
	a.NoError(err)
	prodEnvironment, err := findEnvironment(environments, "Prod")
	a.NoError(err)

	// Add the provisioned instances.
	stagingInstance, err := ctl.addInstance(api.InstanceCreate{
		EnvironmentID: stagingEnvironment.ID,
		Name:          stagingInstanceName,
		Engine:        db.SQLite,
		Host:          stagingInstanceDir,
	})
	a.NoError(err)
	prodInstance, err := ctl.addInstance(api.InstanceCreate{
		EnvironmentID: prodEnvironment.ID,
		Name:          prodInstanceName,
		Engine:        db.SQLite,
		Host:          prodInstanceDir,
	})
	a.NoError(err)

	// Set up label values for tenants.
	// Prod and staging are using the same tenant values. Use prodTenantNumber because it's larger than stagingInstancesNumber.
	var tenants []string
	for i := 0; i < prodTenantNumber; i++ {
		tenants = append(tenants, fmt.Sprintf("tenant%d", i))
	}
	err = ctl.addLabelValues(api.TenantLabelKey, tenants)
	a.NoError(err)

	// Create deployment configuration.
	_, err = ctl.upsertDeploymentConfig(
		api.DeploymentConfigUpsert{
			ProjectID: project.ID,
		},
		deploymentSchedule,
	)
	a.NoError(err)

	// Create issues that create databases.
	baseDatabaseName := "testTenant"
	for i := 0; i < stagingTenantNumber; i++ {
		databaseName := fmt.Sprintf("%s_tenant%d", baseDatabaseName, i)
		err := ctl.createDatabase(project, stagingInstance, databaseName, "", map[string]string{api.TenantLabelKey: fmt.Sprintf("tenant%d", i)})
		a.NoError(err)
	}
	for i := 0; i < prodTenantNumber; i++ {
		databaseName := fmt.Sprintf("%s_tenant%d", baseDatabaseName, i)
		err := ctl.createDatabase(project, prodInstance, databaseName, "", map[string]string{api.TenantLabelKey: fmt.Sprintf("tenant%d", i)})
		a.NoError(err)
	}

	// Getting databases for each environment.
	databases, err := ctl.getDatabases(api.DatabaseFind{
		ProjectID: &project.ID,
	})
	a.NoError(err)

	var stagingDatabases []*api.Database
	var prodDatabases []*api.Database
	for i := 0; i < stagingTenantNumber; i++ {
		databaseName := fmt.Sprintf("%s_tenant%d", baseDatabaseName, i)
		for _, database := range databases {
			if database.Instance.ID == stagingInstance.ID && database.Name == databaseName {
				stagingDatabases = append(stagingDatabases, database)
				break
			}
		}
	}
	for i := 0; i < prodTenantNumber; i++ {
		databaseName := fmt.Sprintf("%s_tenant%d", baseDatabaseName, i)
		for _, database := range databases {
			if database.Instance.ID == prodInstance.ID && database.Name == databaseName {
				prodDatabases = append(prodDatabases, database)
				break
			}
		}
	}
	a.Equal(len(stagingDatabases), stagingTenantNumber)
	a.Equal(len(prodDatabases), prodTenantNumber)

	// Create an issue that updates database schema.
	createContext, err := json.Marshal(&api.UpdateSchemaContext{
		MigrationType: db.Migrate,
		DetailList: []*api.UpdateSchemaDetail{
			{
				DatabaseName: baseDatabaseName,
				Statement:    migrationStatement,
			},
		},
	})
	a.NoError(err)
	issue, err := ctl.createIssue(api.IssueCreate{
		ProjectID:   project.ID,
		Name:        "update schema for tenants",
		Type:        api.IssueDatabaseSchemaUpdate,
		Description: "This updates the schema of tenant databases.",
		// Assign to self.
		AssigneeID:    project.Creator.ID,
		CreateContext: string(createContext),
	})
	a.NoError(err)
	status, err := ctl.waitIssuePipeline(issue.ID)
	a.NoError(err)
	a.Equal(api.TaskDone, status)

	// Query schema.
	for i := 0; i < stagingTenantNumber; i++ {
		databaseName := fmt.Sprintf("%s_tenant%d", baseDatabaseName, i)
		result, err := ctl.query(stagingInstance, databaseName, bookTableQuery)
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
		newWebhookPushEvent func(gitFile string) interface{}
	}{
		{
			name:               "GitLab",
			vcsProviderCreator: fake.NewGitLab,
			vcsType:            vcs.GitLabSelfHost,
			externalID:         "121",
			repositoryFullPath: "test/schemaUpdate",
			newWebhookPushEvent: func(gitFile string) interface{} {
				return gitlab.WebhookPushEvent{
					ObjectKind: gitlab.WebhookPush,
					Ref:        "refs/heads/feature/foo",
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
			vcsType:            vcs.GitHubCom,
			externalID:         "octocat/Hello-World",
			repositoryFullPath: "octocat/Hello-World",
			newWebhookPushEvent: func(gitFile string) interface{} {
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
							Added: []string{gitFile},
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
			err := ctl.StartServer(ctx, t.TempDir(), fake.NewGitHub, getTestPort(t.Name()))
			a.NoError(err)
			defer func() { _ = ctl.Close(ctx) }()

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
					Name:           "Test VCS Project",
					Key:            "TestVCSSchemaUpdate",
					TenantMode:     api.TenantModeTenant,
					DBNameTemplate: "{{DB_NAME}}_{{TENANT}}",
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
					FilePathTemplate:   "{{DB_NAME}}__{{VERSION}}__{{TYPE}}__{{DESCRIPTION}}.sql",
					SchemaPathTemplate: ".{{DB_NAME}}__LATEST.sql",
					ExternalID:         test.externalID,
					AccessToken:        "accessToken1",
					RefreshToken:       "refreshToken1",
				},
			)
			a.NoError(err)

			// Provision instances.
			instanceRootDir := t.TempDir()

			var stagingInstanceDirs []string
			var prodInstanceDirs []string
			for i := 0; i < stagingTenantNumber; i++ {
				instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, fmt.Sprintf("%s-%d", stagingInstanceName, i))
				a.NoError(err)
				stagingInstanceDirs = append(stagingInstanceDirs, instanceDir)
			}
			for i := 0; i < prodTenantNumber; i++ {
				instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, fmt.Sprintf("%s-%d", prodInstanceName, i))
				a.NoError(err)
				prodInstanceDirs = append(prodInstanceDirs, instanceDir)
			}
			environments, err := ctl.getEnvironments()
			a.NoError(err)
			stagingEnvironment, err := findEnvironment(environments, "Staging")
			a.NoError(err)
			prodEnvironment, err := findEnvironment(environments, "Prod")
			a.NoError(err)

			// Add the provisioned instances.
			var stagingInstances []*api.Instance
			var prodInstances []*api.Instance
			for i, stagingInstanceDir := range stagingInstanceDirs {
				instance, err := ctl.addInstance(
					api.InstanceCreate{
						EnvironmentID: stagingEnvironment.ID,
						Name:          fmt.Sprintf("%s-%d", stagingInstanceName, i),
						Engine:        db.SQLite,
						Host:          stagingInstanceDir,
					},
				)
				a.NoError(err)
				stagingInstances = append(stagingInstances, instance)
			}
			for i, prodInstanceDir := range prodInstanceDirs {
				instance, err := ctl.addInstance(
					api.InstanceCreate{
						EnvironmentID: prodEnvironment.ID,
						Name:          fmt.Sprintf("%s-%d", prodInstanceName, i),
						Engine:        db.SQLite,
						Host:          prodInstanceDir,
					},
				)
				a.NoError(err)
				prodInstances = append(prodInstances, instance)
			}

			// Set up label values for tenants.
			// Prod and staging are using the same tenant values. Use prodInstancesNumber because it's larger than stagingInstancesNumber.
			var tenants []string
			for i := 0; i < prodTenantNumber; i++ {
				tenants = append(tenants, fmt.Sprintf("tenant%d", i))
			}
			err = ctl.addLabelValues(api.TenantLabelKey, tenants)
			a.NoError(err)

			// Create deployment configuration.
			_, err = ctl.upsertDeploymentConfig(
				api.DeploymentConfigUpsert{
					ProjectID: project.ID,
				},
				deploymentSchedule,
			)
			a.NoError(err)

			// Create issues that create databases.
			baseDatabaseName := "testTenantVCSSchemaUpdate"

			for i, stagingInstance := range stagingInstances {
				tenant := fmt.Sprintf("tenant%d", i)
				databaseName := baseDatabaseName + "_" + tenant
				err := ctl.createDatabase(project, stagingInstance, databaseName, "", map[string]string{api.TenantLabelKey: tenant})
				a.NoError(err)
			}
			for i, prodInstance := range prodInstances {
				tenant := fmt.Sprintf("tenant%d", i)
				databaseName := baseDatabaseName + "_" + tenant
				err := ctl.createDatabase(project, prodInstance, databaseName, "", map[string]string{api.TenantLabelKey: tenant})
				a.NoError(err)
			}

			// Getting databases for each environment.
			databases, err := ctl.getDatabases(api.DatabaseFind{ProjectID: &project.ID})
			a.NoError(err)

			var stagingDatabases []*api.Database
			var prodDatabases []*api.Database
			for _, stagingInstance := range stagingInstances {
				for _, database := range databases {
					if database.Instance.ID == stagingInstance.ID {
						stagingDatabases = append(stagingDatabases, database)
						break
					}
				}
			}
			for _, prodInstance := range prodInstances {
				for _, database := range databases {
					if database.Instance.ID == prodInstance.ID {
						prodDatabases = append(prodDatabases, database)
						break
					}
				}
			}
			a.Equal(stagingTenantNumber, len(stagingDatabases))
			a.Equal(prodTenantNumber, len(prodDatabases))

			// Simulate Git commits.
			gitFile := "bbtest/testTenantVCSSchemaUpdate__ver1__migrate__create_a_test_table.sql"
			err = ctl.vcsProvider.AddFiles(test.externalID, map[string]string{gitFile: migrationStatement})
			a.NoError(err)

			payload, err := json.Marshal(test.newWebhookPushEvent(gitFile))
			a.NoError(err)
			err = ctl.vcsProvider.SendWebhookPush(test.externalID, payload)
			a.NoError(err)

			// Get schema update issue.
			openStatus := []api.IssueStatus{api.IssueOpen}
			issues, err := ctl.getIssues(
				api.IssueFind{
					ProjectID:  &project.ID,
					StatusList: &openStatus,
				},
			)
			a.NoError(err)
			a.Len(issues, 1)
			issue := issues[0]
			status, err := ctl.waitIssuePipeline(issue.ID)
			a.NoError(err)
			a.Equal(api.TaskDone, status)

			// Query schema.
			for i, stagingInstance := range stagingInstances {
				tenant := fmt.Sprintf("tenant%d", i)
				databaseName := baseDatabaseName + "_" + tenant
				result, err := ctl.query(stagingInstance, databaseName, bookTableQuery)
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
			for i, instance := range stagingInstances {
				tenant := fmt.Sprintf("tenant%d", i)
				databaseName := baseDatabaseName + "_" + tenant
				histories, err := ctl.getInstanceMigrationHistory(
					db.MigrationHistoryFind{
						ID:       &instance.ID,
						Database: &databaseName,
					},
				)
				a.NoError(err)
				a.Len(histories, 2)
				a.Equal(histories[0].Version, "ver1")
				a.NotEqual(histories[1].Version, "")
				hm1[histories[0].Version] = true
			}
			for i, instance := range prodInstances {
				tenant := fmt.Sprintf("tenant%d", i)
				databaseName := baseDatabaseName + "_" + tenant
				histories, err := ctl.getInstanceMigrationHistory(
					db.MigrationHistoryFind{
						ID:       &instance.ID,
						Database: &databaseName,
					},
				)
				a.NoError(err)
				a.Len(histories, 2)
				a.Equal("ver1", histories[0].Version)
				a.NotEqual("", histories[1].Version)
				hm2[histories[0].Version] = true
			}

			a.Len(hm1, 1)
			a.Len(hm2, 1)

			// Check latestSchemaFile
			files, err := ctl.vcsProvider.GetFiles(test.externalID, fmt.Sprintf("%s/.%s__LATEST.sql", baseDirectory, baseDatabaseName))
			a.NoError(err)
			a.Len(files, 1)
		})
	}
}
