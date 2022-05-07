package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/vcs"
	"github.com/bytebase/bytebase/plugin/vcs/gitlab"
	"github.com/stretchr/testify/require"
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
	ctx := context.Background()
	ctl := &controller{}
	dataDir := t.TempDir()
	err := ctl.StartServerWithEmbedPg(ctx, dataDir, getTestPort(t.Name()))
	require.NoError(t, err)
	defer ctl.Close(ctx)
	err = ctl.Login()
	require.NoError(t, err)
	err = ctl.setLicense()
	require.NoError(t, err)

	// Create a project.
	project, err := ctl.createProject(api.ProjectCreate{
		Name:       "Test Project",
		Key:        "TestSchemaUpdate",
		TenantMode: api.TenantModeTenant,
	})
	require.NoError(t, err)

	// Provision instances.
	instanceRootDir := t.TempDir()

	var stagingInstanceDirs []string
	var prodInstanceDirs []string
	for i := 0; i < stagingTenantNumber; i++ {
		instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, fmt.Sprintf("%s-%d", stagingInstanceName, i))
		require.NoError(t, err)
		stagingInstanceDirs = append(stagingInstanceDirs, instanceDir)
	}
	for i := 0; i < prodTenantNumber; i++ {
		instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, fmt.Sprintf("%s-%d", prodInstanceName, i))
		require.NoError(t, err)
		prodInstanceDirs = append(prodInstanceDirs, instanceDir)
	}
	environments, err := ctl.getEnvironments()
	require.NoError(t, err)
	stagingEnvironment, err := findEnvironment(environments, "Staging")
	require.NoError(t, err)
	prodEnvironment, err := findEnvironment(environments, "Prod")
	require.NoError(t, err)

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
		require.NoError(t, err)
		stagingInstances = append(stagingInstances, instance)
	}
	for i, prodInstanceDir := range prodInstanceDirs {
		instance, err := ctl.addInstance(api.InstanceCreate{
			EnvironmentID: prodEnvironment.ID,
			Name:          fmt.Sprintf("%s-%d", prodInstanceName, i),
			Engine:        db.SQLite,
			Host:          prodInstanceDir,
		})
		require.NoError(t, err)
		prodInstances = append(prodInstances, instance)
	}

	// Set up label values for tenants.
	// Prod and staging are using the same tenant values. Use prodInstancesNumber because it's larger than stagingInstancesNumber.
	var tenants []string
	for i := 0; i < prodTenantNumber; i++ {
		tenants = append(tenants, fmt.Sprintf("tenant%d", i))
	}
	err = ctl.addLabelValues(api.TenantLabelKey, tenants)
	require.NoError(t, err)

	// Create deployment configuration.
	_, err = ctl.upsertDeploymentConfig(
		api.DeploymentConfigUpsert{
			ProjectID: project.ID,
		},
		deploymentSchedule,
	)
	require.NoError(t, err)

	// Create issues that create databases.
	databaseName := "testTenantSchemaUpdate"
	for i, stagingInstance := range stagingInstances {
		err := ctl.createDatabase(project, stagingInstance, databaseName, map[string]string{api.TenantLabelKey: fmt.Sprintf("tenant%d", i)})
		require.NoError(t, err)
	}
	for i, prodInstance := range prodInstances {
		err := ctl.createDatabase(project, prodInstance, databaseName, map[string]string{api.TenantLabelKey: fmt.Sprintf("tenant%d", i)})
		require.NoError(t, err)
	}

	// Getting databases for each environment.
	databases, err := ctl.getDatabases(api.DatabaseFind{
		ProjectID: &project.ID,
	})
	require.NoError(t, err)

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
	require.Equal(t, len(stagingDatabases), stagingTenantNumber)
	require.Equal(t, len(prodDatabases), prodTenantNumber)

	// Create an issue that updates database schema.
	createContext, err := json.Marshal(&api.UpdateSchemaContext{
		MigrationType: db.Migrate,
		UpdateSchemaDetailList: []*api.UpdateSchemaDetail{
			{
				DatabaseName: databaseName,
				Statement:    migrationStatement,
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
	for _, stagingInstance := range stagingInstances {
		result, err := ctl.query(stagingInstance, databaseName, bookTableQuery)
		require.NoError(t, err)
		require.Equal(t, bookSchemaSQLResult, result)
	}
	for _, prodInstance := range prodInstances {
		result, err := ctl.query(prodInstance, databaseName, bookTableQuery)
		require.NoError(t, err)
		require.Equal(t, bookSchemaSQLResult, result)
	}

	// Query migration history
	instances := append(stagingInstances, prodInstances...)
	hm1 := map[string]bool{}
	hm2 := map[string]bool{}
	for _, instance := range instances {
		histories, err := ctl.getInstanceMigrationHistory(db.MigrationHistoryFind{ID: &instance.ID, Database: &databaseName})
		require.NoError(t, err)
		require.Equal(t, len(histories), 2)
		require.NotEqual(t, histories[0].Version, "")
		require.NotEqual(t, histories[1].Version, "")
		hm1[histories[0].Version] = true
		hm2[histories[1].Version] = true
	}
	require.Equal(t, len(hm1), 1)
	require.Equal(t, len(hm2), 1)
}

func TestTenantVCS(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	ctl := &controller{}
	dataDir := t.TempDir()
	err := ctl.StartServerWithEmbedPg(ctx, dataDir, getTestPort(t.Name()))
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
		Name:       "Test Project",
		Key:        "TestSchemaUpdate",
		TenantMode: api.TenantModeTenant,
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
		BaseDirectory:      baseDirectory,
		FilePathTemplate:   "{{DB_NAME}}__{{VERSION}}__{{TYPE}}__{{DESCRIPTION}}.sql",
		SchemaPathTemplate: ".{{DB_NAME}}__LATEST.sql",
		ExternalID:         gitlabProjectIDStr,
		AccessToken:        accessToken,
		ExpiresTs:          0,
		RefreshToken:       refreshToken,
	})
	require.NoError(t, err)

	// Provision instances.
	instanceRootDir := t.TempDir()

	var stagingInstanceDirs []string
	var prodInstanceDirs []string
	for i := 0; i < stagingTenantNumber; i++ {
		instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, fmt.Sprintf("%s-%d", stagingInstanceName, i))
		require.NoError(t, err)
		stagingInstanceDirs = append(stagingInstanceDirs, instanceDir)
	}
	for i := 0; i < prodTenantNumber; i++ {
		instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, fmt.Sprintf("%s-%d", prodInstanceName, i))
		require.NoError(t, err)
		prodInstanceDirs = append(prodInstanceDirs, instanceDir)
	}
	environments, err := ctl.getEnvironments()
	require.NoError(t, err)
	stagingEnvironment, err := findEnvironment(environments, "Staging")
	require.NoError(t, err)
	prodEnvironment, err := findEnvironment(environments, "Prod")
	require.NoError(t, err)

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
		require.NoError(t, err)
		stagingInstances = append(stagingInstances, instance)
	}
	for i, prodInstanceDir := range prodInstanceDirs {
		instance, err := ctl.addInstance(api.InstanceCreate{
			EnvironmentID: prodEnvironment.ID,
			Name:          fmt.Sprintf("%s-%d", prodInstanceName, i),
			Engine:        db.SQLite,
			Host:          prodInstanceDir,
		})
		require.NoError(t, err)
		prodInstances = append(prodInstances, instance)
	}

	// Set up label values for tenants.
	// Prod and staging are using the same tenant values. Use prodInstancesNumber because it's larger than stagingInstancesNumber.
	var tenants []string
	for i := 0; i < prodTenantNumber; i++ {
		tenants = append(tenants, fmt.Sprintf("tenant%d", i))
	}
	err = ctl.addLabelValues(api.TenantLabelKey, tenants)
	require.NoError(t, err)

	// Create deployment configuration.
	_, err = ctl.upsertDeploymentConfig(
		api.DeploymentConfigUpsert{
			ProjectID: project.ID,
		},
		deploymentSchedule,
	)
	require.NoError(t, err)

	// Create issues that create databases.
	databaseName := "testTenantVCSSchemaUpdate"
	for i, stagingInstance := range stagingInstances {
		err := ctl.createDatabase(project, stagingInstance, databaseName, map[string]string{api.TenantLabelKey: fmt.Sprintf("tenant%d", i)})
		require.NoError(t, err)
	}
	for i, prodInstance := range prodInstances {
		err := ctl.createDatabase(project, prodInstance, databaseName, map[string]string{api.TenantLabelKey: fmt.Sprintf("tenant%d", i)})
		require.NoError(t, err)
	}

	// Getting databases for each environment.
	databases, err := ctl.getDatabases(api.DatabaseFind{
		ProjectID: &project.ID,
	})
	require.NoError(t, err)

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
	require.Equal(t, len(stagingDatabases), stagingTenantNumber)
	require.Equal(t, len(prodDatabases), prodTenantNumber)

	// Simulate Git commits.
	gitFile := "bbtest/testTenantVCSSchemaUpdate__ver1__migrate__create_a_test_table.sql"
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

	// Query schema.
	for _, stagingInstance := range stagingInstances {
		result, err := ctl.query(stagingInstance, databaseName, bookTableQuery)
		require.NoError(t, err)
		require.Equal(t, bookSchemaSQLResult, result)
	}
	for _, prodInstance := range prodInstances {
		result, err := ctl.query(prodInstance, databaseName, bookTableQuery)
		require.NoError(t, err)
		require.Equal(t, bookSchemaSQLResult, result)
	}

	// Query migration history
	instances := append(stagingInstances, prodInstances...)
	hm1 := map[string]bool{}
	hm2 := map[string]bool{}
	for _, instance := range instances {
		histories, err := ctl.getInstanceMigrationHistory(db.MigrationHistoryFind{ID: &instance.ID, Database: &databaseName})
		require.NoError(t, err)
		require.Equal(t, len(histories), 2)
		require.Equal(t, histories[0].Version, "ver1")
		require.NotEqual(t, histories[1].Version, "")
		hm1[histories[0].Version] = true
		hm2[histories[1].Version] = true
	}
	require.Equal(t, len(hm1), 1)
	require.Equal(t, len(hm2), 1)
}

func TestTenantDatabaseNameTemplate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	ctl := &controller{}
	dataDir := t.TempDir()
	err := ctl.StartServerWithEmbedPg(ctx, dataDir, getTestPort(t.Name()))
	require.NoError(t, err)
	defer ctl.Close(ctx)
	err = ctl.Login()
	require.NoError(t, err)
	err = ctl.setLicense()
	require.NoError(t, err)

	// Create a project.
	project, err := ctl.createProject(api.ProjectCreate{
		Name:           "Test Project",
		Key:            "TestSchemaUpdate",
		TenantMode:     api.TenantModeTenant,
		DBNameTemplate: "{{DB_NAME}}_{{TENANT}}",
	})
	require.NoError(t, err)

	// Provision instances.
	instanceRootDir := t.TempDir()
	stagingInstanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, stagingInstanceName)
	require.NoError(t, err)
	prodInstanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, prodInstanceName)
	require.NoError(t, err)

	environments, err := ctl.getEnvironments()
	require.NoError(t, err)
	stagingEnvironment, err := findEnvironment(environments, "Staging")
	require.NoError(t, err)
	prodEnvironment, err := findEnvironment(environments, "Prod")
	require.NoError(t, err)

	// Add the provisioned instances.
	stagingInstance, err := ctl.addInstance(api.InstanceCreate{
		EnvironmentID: stagingEnvironment.ID,
		Name:          stagingInstanceName,
		Engine:        db.SQLite,
		Host:          stagingInstanceDir,
	})
	require.NoError(t, err)
	prodInstance, err := ctl.addInstance(api.InstanceCreate{
		EnvironmentID: prodEnvironment.ID,
		Name:          prodInstanceName,
		Engine:        db.SQLite,
		Host:          prodInstanceDir,
	})
	require.NoError(t, err)

	// Set up label values for tenants.
	// Prod and staging are using the same tenant values. Use prodTenantNumber because it's larger than stagingInstancesNumber.
	var tenants []string
	for i := 0; i < prodTenantNumber; i++ {
		tenants = append(tenants, fmt.Sprintf("tenant%d", i))
	}
	err = ctl.addLabelValues(api.TenantLabelKey, tenants)
	require.NoError(t, err)

	// Create deployment configuration.
	_, err = ctl.upsertDeploymentConfig(
		api.DeploymentConfigUpsert{
			ProjectID: project.ID,
		},
		deploymentSchedule,
	)
	require.NoError(t, err)

	// Create issues that create databases.
	baseDatabaseName := "testTenant"
	for i := 0; i < stagingTenantNumber; i++ {
		databaseName := fmt.Sprintf("%s_tenant%d", baseDatabaseName, i)
		err := ctl.createDatabase(project, stagingInstance, databaseName, map[string]string{api.TenantLabelKey: fmt.Sprintf("tenant%d", i)})
		require.NoError(t, err)
	}
	for i := 0; i < prodTenantNumber; i++ {
		databaseName := fmt.Sprintf("%s_tenant%d", baseDatabaseName, i)
		err := ctl.createDatabase(project, prodInstance, databaseName, map[string]string{api.TenantLabelKey: fmt.Sprintf("tenant%d", i)})
		require.NoError(t, err)
	}

	// Getting databases for each environment.
	databases, err := ctl.getDatabases(api.DatabaseFind{
		ProjectID: &project.ID,
	})
	require.NoError(t, err)

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
	require.Equal(t, len(stagingDatabases), stagingTenantNumber)
	require.Equal(t, len(prodDatabases), prodTenantNumber)

	// Create an issue that updates database schema.
	createContext, err := json.Marshal(&api.UpdateSchemaContext{
		MigrationType: db.Migrate,
		UpdateSchemaDetailList: []*api.UpdateSchemaDetail{
			{
				DatabaseName: baseDatabaseName,
				Statement:    migrationStatement,
			},
		},
	})
	require.NoError(t, err)
	issue, err := ctl.createIssue(api.IssueCreate{
		ProjectID:   project.ID,
		Name:        "update schema for tenants",
		Type:        api.IssueDatabaseSchemaUpdate,
		Description: "This updates the schema of tenant databases.",
		// Assign to self.
		AssigneeID:    project.Creator.ID,
		CreateContext: string(createContext),
	})
	require.NoError(t, err)
	status, err := ctl.waitIssuePipeline(issue.ID)
	require.NoError(t, err)
	require.Equal(t, status, api.TaskDone)

	// Query schema.
	for i := 0; i < stagingTenantNumber; i++ {
		databaseName := fmt.Sprintf("%s_tenant%d", baseDatabaseName, i)
		result, err := ctl.query(stagingInstance, databaseName, bookTableQuery)
		require.NoError(t, err)
		require.Equal(t, bookSchemaSQLResult, result)
	}
	for i := 0; i < prodTenantNumber; i++ {
		databaseName := fmt.Sprintf("%s_tenant%d", baseDatabaseName, i)
		result, err := ctl.query(prodInstance, databaseName, bookTableQuery)
		require.NoError(t, err)
		require.Equal(t, bookSchemaSQLResult, result)
	}
}

func TestTenantVCSDatabaseNameTemplate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	ctl := &controller{}
	dataDir := t.TempDir()
	err := ctl.StartServerWithEmbedPg(ctx, dataDir, getTestPort(t.Name()))
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
		Name:           "Test Project",
		Key:            "TestSchemaUpdate",
		TenantMode:     api.TenantModeTenant,
		DBNameTemplate: "{{DB_NAME}}_{{TENANT}}",
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
		FilePathTemplate:   "{{DB_NAME}}__{{VERSION}}__{{TYPE}}__{{DESCRIPTION}}.sql",
		SchemaPathTemplate: ".{{DB_NAME}}__LATEST.sql",
		ExternalID:         gitlabProjectIDStr,
		AccessToken:        accessToken,
		ExpiresTs:          0,
		RefreshToken:       refreshToken,
	})
	require.NoError(t, err)

	// Provision instances.
	instanceRootDir := t.TempDir()

	var stagingInstanceDirs []string
	var prodInstanceDirs []string
	for i := 0; i < stagingTenantNumber; i++ {
		instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, fmt.Sprintf("%s-%d", stagingInstanceName, i))
		require.NoError(t, err)
		stagingInstanceDirs = append(stagingInstanceDirs, instanceDir)
	}
	for i := 0; i < prodTenantNumber; i++ {
		instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, fmt.Sprintf("%s-%d", prodInstanceName, i))
		require.NoError(t, err)
		prodInstanceDirs = append(prodInstanceDirs, instanceDir)
	}
	environments, err := ctl.getEnvironments()
	require.NoError(t, err)
	stagingEnvironment, err := findEnvironment(environments, "Staging")
	require.NoError(t, err)
	prodEnvironment, err := findEnvironment(environments, "Prod")
	require.NoError(t, err)

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
		require.NoError(t, err)
		stagingInstances = append(stagingInstances, instance)
	}
	for i, prodInstanceDir := range prodInstanceDirs {
		instance, err := ctl.addInstance(api.InstanceCreate{
			EnvironmentID: prodEnvironment.ID,
			Name:          fmt.Sprintf("%s-%d", prodInstanceName, i),
			Engine:        db.SQLite,
			Host:          prodInstanceDir,
		})
		require.NoError(t, err)
		prodInstances = append(prodInstances, instance)
	}

	// Set up label values for tenants.
	// Prod and staging are using the same tenant values. Use prodInstancesNumber because it's larger than stagingInstancesNumber.
	var tenants []string
	for i := 0; i < prodTenantNumber; i++ {
		tenants = append(tenants, fmt.Sprintf("tenant%d", i))
	}
	err = ctl.addLabelValues(api.TenantLabelKey, tenants)
	require.NoError(t, err)

	// Create deployment configuration.
	_, err = ctl.upsertDeploymentConfig(
		api.DeploymentConfigUpsert{
			ProjectID: project.ID,
		},
		deploymentSchedule,
	)
	require.NoError(t, err)

	// Create issues that create databases.
	baseDatabaseName := "testTenantVCSSchemaUpdate"

	for i, stagingInstance := range stagingInstances {
		tenant := fmt.Sprintf("tenant%d", i)
		databaseName := baseDatabaseName + "_" + tenant
		err := ctl.createDatabase(project, stagingInstance, databaseName, map[string]string{api.TenantLabelKey: tenant})
		require.NoError(t, err)
	}
	for i, prodInstance := range prodInstances {
		tenant := fmt.Sprintf("tenant%d", i)
		databaseName := baseDatabaseName + "_" + tenant
		err := ctl.createDatabase(project, prodInstance, databaseName, map[string]string{api.TenantLabelKey: tenant})
		require.NoError(t, err)
	}

	// Getting databases for each environment.
	databases, err := ctl.getDatabases(api.DatabaseFind{
		ProjectID: &project.ID,
	})
	require.NoError(t, err)

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
	require.Equal(t, len(stagingDatabases), stagingTenantNumber)
	require.Equal(t, len(prodDatabases), prodTenantNumber)

	// Simulate Git commits.
	gitFile := "bbtest/testTenantVCSSchemaUpdate__ver1__migrate__create_a_test_table.sql"
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

	// Query schema.
	for i, stagingInstance := range stagingInstances {
		tenant := fmt.Sprintf("tenant%d", i)
		databaseName := baseDatabaseName + "_" + tenant
		result, err := ctl.query(stagingInstance, databaseName, bookTableQuery)
		require.NoError(t, err)
		require.Equal(t, bookSchemaSQLResult, result)
	}
	for i, prodInstance := range prodInstances {
		tenant := fmt.Sprintf("tenant%d", i)
		databaseName := baseDatabaseName + "_" + tenant
		result, err := ctl.query(prodInstance, databaseName, bookTableQuery)
		require.NoError(t, err)
		require.Equal(t, bookSchemaSQLResult, result)
	}

	// Query migration history
	hm1 := map[string]bool{}
	hm2 := map[string]bool{}
	for i, instance := range stagingInstances {
		tenant := fmt.Sprintf("tenant%d", i)
		databaseName := baseDatabaseName + "_" + tenant
		histories, err := ctl.getInstanceMigrationHistory(db.MigrationHistoryFind{ID: &instance.ID, Database: &databaseName})
		require.NoError(t, err)
		require.Equal(t, len(histories), 2)
		require.Equal(t, histories[0].Version, "ver1")
		require.NotEqual(t, histories[1].Version, "")
		hm1[histories[0].Version] = true
	}
	for i, instance := range prodInstances {
		tenant := fmt.Sprintf("tenant%d", i)
		databaseName := baseDatabaseName + "_" + tenant
		histories, err := ctl.getInstanceMigrationHistory(db.MigrationHistoryFind{ID: &instance.ID, Database: &databaseName})
		require.NoError(t, err)
		require.Equal(t, len(histories), 2)
		require.Equal(t, histories[0].Version, "ver1")
		require.NotEqual(t, histories[1].Version, "")
		hm2[histories[0].Version] = true
	}

	require.Equal(t, len(hm1), 1)
	require.Equal(t, len(hm2), 1)

	// Check latestSchemaFile
	files, err := ctl.gitlab.GetFiles(gitlabProjectIDStr, fmt.Sprintf("%s/.%s__LATEST.sql", baseDirectory, baseDatabaseName))
	require.NoError(t, err)
	require.Equal(t, len(files), 1)

}
