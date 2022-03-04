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
	"github.com/kr/pretty"
)

var (
	stagingTenantNumber = 1
	prodTenantNumber    = 3
	stagingInstanceName = "testInstanceStaging"
	prodInstanceName    = "testInstanceProd"
)

func TestTenant(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	ctl := &controller{}
	dataDir := t.TempDir()
	if err := ctl.StartMain(ctx, dataDir, getTestPort(t.Name())); err != nil {
		t.Fatal(err)
	}
	defer ctl.Close()
	if err := ctl.Login(); err != nil {
		t.Fatal(err)
	}
	if err := ctl.setLicense(); err != nil {
		t.Fatal(err)
	}

	// Create a project.
	project, err := ctl.createProject(api.ProjectCreate{
		Name:       "Test Project",
		Key:        "TestSchemaUpdate",
		TenantMode: api.TenantModeTenant,
	})
	if err != nil {
		t.Fatalf("failed to create project, error: %v", err)
	}

	// Provision instances.
	instanceRootDir := t.TempDir()

	var stagingInstanceDirs []string
	var prodInstanceDirs []string
	for i := 0; i < stagingTenantNumber; i++ {
		instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, fmt.Sprintf("%s-%d", stagingInstanceName, i))
		if err != nil {
			t.Fatal(err)
		}
		stagingInstanceDirs = append(stagingInstanceDirs, instanceDir)
	}
	for i := 0; i < prodTenantNumber; i++ {
		instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, fmt.Sprintf("%s-%d", prodInstanceName, i))
		if err != nil {
			t.Fatal(err)
		}
		prodInstanceDirs = append(prodInstanceDirs, instanceDir)
	}
	environments, err := ctl.getEnvironments()
	if err != nil {
		t.Fatal(err)
	}
	stagingEnvironment, err := findEnvironment(environments, "Staging")
	if err != nil {
		t.Fatal(err)
	}
	prodEnvironment, err := findEnvironment(environments, "Prod")
	if err != nil {
		t.Fatal(err)
	}

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
		if err != nil {
			t.Fatalf("failed to add instance, error: %v", err)
		}
		stagingInstances = append(stagingInstances, instance)
	}
	for i, prodInstanceDir := range prodInstanceDirs {
		instance, err := ctl.addInstance(api.InstanceCreate{
			EnvironmentID: prodEnvironment.ID,
			Name:          fmt.Sprintf("%s-%d", prodInstanceName, i),
			Engine:        db.SQLite,
			Host:          prodInstanceDir,
		})
		if err != nil {
			t.Fatalf("failed to add instance, error: %v", err)
		}
		prodInstances = append(prodInstances, instance)
	}

	// Set up label values for tenants.
	// Prod and staging are using the same tenant values. Use prodInstancesNumber because it's larger than stagingInstancesNumber.
	var tenants []string
	for i := 0; i < prodTenantNumber; i++ {
		tenants = append(tenants, fmt.Sprintf("tenant%d", i))
	}
	if err := ctl.addLabelValues(api.TenantLabelKey, tenants); err != nil {
		t.Fatal(err)
	}

	// Create deployment configuration.
	if _, err := ctl.upsertDeploymentConfig(
		api.DeploymentConfigUpsert{
			ProjectID: project.ID,
		},
		deploymentSchdule,
	); err != nil {
		t.Fatal(err)
	}

	// Create issues that create databases.
	databaseName := "testTenantSchemaUpdate"
	for i, stagingInstance := range stagingInstances {
		if err := ctl.createDatabase(project, stagingInstance, databaseName, map[string]string{api.TenantLabelKey: fmt.Sprintf("tenant%d", i)}); err != nil {
			t.Fatal(err)
		}
	}
	for i, prodInstance := range prodInstances {
		if err := ctl.createDatabase(project, prodInstance, databaseName, map[string]string{api.TenantLabelKey: fmt.Sprintf("tenant%d", i)}); err != nil {
			t.Fatal(err)
		}
	}

	// Getting databases for each environment.
	databases, err := ctl.getDatabases(api.DatabaseFind{
		ProjectID: &project.ID,
	})
	if err != nil {
		t.Fatalf("failed to get databases, error: %v", err)
	}

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
	if len(stagingDatabases) != stagingTenantNumber {
		t.Fatalf("invalid number of staging databases %v in project %v, expecting %d database", project.ID, len(databases), stagingTenantNumber)
	}
	if len(prodDatabases) != prodTenantNumber {
		t.Fatalf("invalid number of prod databases %v in project %v, expecting %d database", project.ID, len(databases), prodTenantNumber)
	}

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
	if err != nil {
		t.Fatalf("failed to construct schema update issue CreateContext payload, error: %v", err)
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
		t.Fatalf("failed to create schema update issue, error: %v", err)
	}
	status, err := ctl.waitIssuePipeline(issue.ID)
	if err != nil {
		t.Fatalf("failed to wait for issue %v pipeline %v, error: %v", issue.ID, issue.Pipeline.ID, err)
	}
	if status != api.TaskDone {
		t.Fatalf("issue %v pipeline %v is expected to finish with status done got %v", issue.ID, issue.Pipeline.ID, status)
	}

	// Query schema.
	for _, stagingInstance := range stagingInstances {
		result, err := ctl.query(stagingInstance, databaseName, bookTableQuery)
		if err != nil {
			t.Fatal(err)
		}
		if bookSchemaSQLResult != result {
			t.Fatalf("SQL result want %q, got %q, diff %q", bookSchemaSQLResult, result, pretty.Diff(bookSchemaSQLResult, result))
		}
	}
	for _, prodInstance := range prodInstances {
		result, err := ctl.query(prodInstance, databaseName, bookTableQuery)
		if err != nil {
			t.Fatal(err)
		}
		if bookSchemaSQLResult != result {
			t.Fatalf("SQL result want %q, got %q, diff %q", bookSchemaSQLResult, result, pretty.Diff(bookSchemaSQLResult, result))
		}
	}

	// Query migration history
	instances := append(stagingInstances, prodInstances...)
	var hm1 map[string]bool = map[string]bool{}
	var hm2 map[string]bool = map[string]bool{}
	for _, instance := range instances {
		histories, err := ctl.getInstanceMigrationHistory(db.MigrationHistoryFind{ID: &instance.ID, Database: &databaseName})
		if err != nil {
			t.Fatal(err)
		}
		if len(histories) != 2 {
			t.Fatalf("invalid number of migration histories, want 2, got %v", len(histories))
		}
		if histories[0].Version == "" {
			t.Fatalf("empty migration history(0) version")
		}
		if histories[1].Version == "" {
			t.Fatalf("empty migration history(1) version")
		}
		hm1[histories[0].Version] = true
		hm2[histories[1].Version] = true
	}
	if len(hm1) != 1 || len(hm2) != 1 {
		t.Fatalf("migration history should have only one version in tenant mode")
	}
}

func TestTenantVCS(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	ctl := &controller{}
	dataDir := t.TempDir()
	if err := ctl.StartMain(ctx, dataDir, getTestPort(t.Name())); err != nil {
		t.Fatal(err)
	}
	defer ctl.Close()
	if err := ctl.Login(); err != nil {
		t.Fatal(err)
	}
	if err := ctl.setLicense(); err != nil {
		t.Fatal(err)
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
		t.Fatalf("failed to create VCS, error: %v", err)
	}

	// Create a project.
	project, err := ctl.createProject(api.ProjectCreate{
		Name:       "Test Project",
		Key:        "TestSchemaUpdate",
		TenantMode: api.TenantModeTenant,
	})
	if err != nil {
		t.Fatalf("failed to create project, error: %v", err)
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
		FilePathTemplate:   "{{DB_NAME}}__{{VERSION}}__{{TYPE}}__{{DESCRIPTION}}.sql",
		SchemaPathTemplate: ".{{DB_NAME}}__LATEST.sql",
		ExternalID:         gitlabProjectIDStr,
		AccessToken:        accessToken,
		ExpiresTs:          0,
		RefreshToken:       refreshToken,
	})
	if err != nil {
		t.Fatalf("failed to create repository, error: %v", err)
	}

	// Provision instances.
	instanceRootDir := t.TempDir()

	var stagingInstanceDirs []string
	var prodInstanceDirs []string
	for i := 0; i < stagingTenantNumber; i++ {
		instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, fmt.Sprintf("%s-%d", stagingInstanceName, i))
		if err != nil {
			t.Fatal(err)
		}
		stagingInstanceDirs = append(stagingInstanceDirs, instanceDir)
	}
	for i := 0; i < prodTenantNumber; i++ {
		instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, fmt.Sprintf("%s-%d", prodInstanceName, i))
		if err != nil {
			t.Fatal(err)
		}
		prodInstanceDirs = append(prodInstanceDirs, instanceDir)
	}
	environments, err := ctl.getEnvironments()
	if err != nil {
		t.Fatal(err)
	}
	stagingEnvironment, err := findEnvironment(environments, "Staging")
	if err != nil {
		t.Fatal(err)
	}
	prodEnvironment, err := findEnvironment(environments, "Prod")
	if err != nil {
		t.Fatal(err)
	}

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
		if err != nil {
			t.Fatalf("failed to add instance, error: %v", err)
		}
		stagingInstances = append(stagingInstances, instance)
	}
	for i, prodInstanceDir := range prodInstanceDirs {
		instance, err := ctl.addInstance(api.InstanceCreate{
			EnvironmentID: prodEnvironment.ID,
			Name:          fmt.Sprintf("%s-%d", prodInstanceName, i),
			Engine:        db.SQLite,
			Host:          prodInstanceDir,
		})
		if err != nil {
			t.Fatalf("failed to add instance, error: %v", err)
		}
		prodInstances = append(prodInstances, instance)
	}

	// Set up label values for tenants.
	// Prod and staging are using the same tenant values. Use prodInstancesNumber because it's larger than stagingInstancesNumber.
	var tenants []string
	for i := 0; i < prodTenantNumber; i++ {
		tenants = append(tenants, fmt.Sprintf("tenant%d", i))
	}
	if err := ctl.addLabelValues(api.TenantLabelKey, tenants); err != nil {
		t.Fatal(err)
	}

	// Create deployment configuration.
	if _, err := ctl.upsertDeploymentConfig(
		api.DeploymentConfigUpsert{
			ProjectID: project.ID,
		},
		deploymentSchdule,
	); err != nil {
		t.Fatal(err)
	}

	// Create issues that create databases.
	databaseName := "testTenantVCSSchemaUpdate"
	for i, stagingInstance := range stagingInstances {
		if err := ctl.createDatabase(project, stagingInstance, databaseName, map[string]string{api.TenantLabelKey: fmt.Sprintf("tenant%d", i)}); err != nil {
			t.Fatal(err)
		}
	}
	for i, prodInstance := range prodInstances {
		if err := ctl.createDatabase(project, prodInstance, databaseName, map[string]string{api.TenantLabelKey: fmt.Sprintf("tenant%d", i)}); err != nil {
			t.Fatal(err)
		}
	}

	// Getting databases for each environment.
	databases, err := ctl.getDatabases(api.DatabaseFind{
		ProjectID: &project.ID,
	})
	if err != nil {
		t.Fatalf("failed to get databases, error: %v", err)
	}

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
	if len(stagingDatabases) != stagingTenantNumber {
		t.Fatalf("invalid number of staging databases %v in project %v, expecting %d database", project.ID, len(databases), stagingTenantNumber)
	}
	if len(prodDatabases) != prodTenantNumber {
		t.Fatalf("invalid number of prod databases %v in project %v, expecting %d database", project.ID, len(databases), prodTenantNumber)
	}

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
	if err := ctl.gitlab.AddFiles(gitlabProjectIDStr, map[string]string{gitFile: migrationStatement}); err != nil {
		t.Fatalf("failed to add files to gitlab project %v, error %v", gitlabProjectID, err)
	}
	if err := ctl.gitlab.SendCommits(gitlabProjectIDStr, pushEvent); err != nil {
		t.Fatalf("failed to send commits to gitlab project %v, error %v", gitlabProjectID, err)
	}

	// Get schema update issue.
	openStatus := []api.IssueStatus{api.IssueOpen}
	issues, err := ctl.getIssues(api.IssueFind{ProjectID: &project.ID, StatusList: &openStatus})
	if err != nil {
		t.Fatalf("failed to get open issues for project %v, error: %v", project.ID, err)
	}
	if len(issues) != 1 {
		t.Fatalf("invalid number of open issues %v in project %v, expecting one issue", len(issues), project.ID)
	}
	issue := issues[0]
	status, err := ctl.waitIssuePipeline(issue.ID)
	if err != nil {
		t.Fatalf("failed to wait for issue %v pipeline %v, error: %v", issue.ID, issue.Pipeline.ID, err)
	}
	if status != api.TaskDone {
		t.Fatalf("issue %v pipeline %v is expected to finish with status done got %v", issue.ID, issue.Pipeline.ID, status)
	}

	// Query schema.
	for _, stagingInstance := range stagingInstances {
		result, err := ctl.query(stagingInstance, databaseName, bookTableQuery)
		if err != nil {
			t.Fatal(err)
		}
		if bookSchemaSQLResult != result {
			t.Fatalf("SQL result want %q, got %q, diff %q", bookSchemaSQLResult, result, pretty.Diff(bookSchemaSQLResult, result))
		}
	}
	for _, prodInstance := range prodInstances {
		result, err := ctl.query(prodInstance, databaseName, bookTableQuery)
		if err != nil {
			t.Fatal(err)
		}
		if bookSchemaSQLResult != result {
			t.Fatalf("SQL result want %q, got %q, diff %q", bookSchemaSQLResult, result, pretty.Diff(bookSchemaSQLResult, result))
		}
	}

	// Query migration history
	instances := append(stagingInstances, prodInstances...)
	var hm1 map[string]bool = map[string]bool{}
	var hm2 map[string]bool = map[string]bool{}
	for _, instance := range instances {
		histories, err := ctl.getInstanceMigrationHistory(db.MigrationHistoryFind{ID: &instance.ID, Database: &databaseName})
		if err != nil {
			t.Fatal(err)
		}
		if len(histories) != 2 {
			t.Fatalf("invalid number of migration histories, want 2, got %v", len(histories))
		}
		if histories[0].Version != "ver1" {
			t.Fatalf("invalid migration history(0) version, want ver1, got %q", histories[0].Version)
		}
		if histories[1].Version == "" {
			t.Fatalf("empty migration history(1) version")
		}
		hm1[histories[0].Version] = true
		hm2[histories[1].Version] = true
	}
	if len(hm1) != 1 || len(hm2) != 1 {
		t.Fatalf("migration history should have only one version in tenant mode")
	}
}

func TestTenantDatabaseNameTemplate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	ctl := &controller{}
	dataDir := t.TempDir()
	if err := ctl.StartMain(ctx, dataDir, getTestPort(t.Name())); err != nil {
		t.Fatal(err)
	}
	defer ctl.Close()
	if err := ctl.Login(); err != nil {
		t.Fatal(err)
	}
	if err := ctl.setLicense(); err != nil {
		t.Fatal(err)
	}

	// Create a project.
	project, err := ctl.createProject(api.ProjectCreate{
		Name:           "Test Project",
		Key:            "TestSchemaUpdate",
		TenantMode:     api.TenantModeTenant,
		DBNameTemplate: "{{DB_NAME}}_{{TENANT}}",
	})
	if err != nil {
		t.Fatalf("failed to create project, error: %v", err)
	}

	// Provision instances.
	instanceRootDir := t.TempDir()
	stagingInstanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, stagingInstanceName)
	if err != nil {
		t.Fatal(err)
	}
	prodInstanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, prodInstanceName)
	if err != nil {
		t.Fatal(err)
	}

	environments, err := ctl.getEnvironments()
	if err != nil {
		t.Fatal(err)
	}
	stagingEnvironment, err := findEnvironment(environments, "Staging")
	if err != nil {
		t.Fatal(err)
	}
	prodEnvironment, err := findEnvironment(environments, "Prod")
	if err != nil {
		t.Fatal(err)
	}

	// Add the provisioned instances.
	stagingInstance, err := ctl.addInstance(api.InstanceCreate{
		EnvironmentID: stagingEnvironment.ID,
		Name:          stagingInstanceName,
		Engine:        db.SQLite,
		Host:          stagingInstanceDir,
	})
	if err != nil {
		t.Fatalf("failed to add instance, error: %v", err)
	}
	prodInstance, err := ctl.addInstance(api.InstanceCreate{
		EnvironmentID: prodEnvironment.ID,
		Name:          prodInstanceName,
		Engine:        db.SQLite,
		Host:          prodInstanceDir,
	})
	if err != nil {
		t.Fatalf("failed to add instance, error: %v", err)
	}

	// Set up label values for tenants.
	// Prod and staging are using the same tenant values. Use prodTenantNumber because it's larger than stagingInstancesNumber.
	var tenants []string
	for i := 0; i < prodTenantNumber; i++ {
		tenants = append(tenants, fmt.Sprintf("tenant%d", i))
	}
	if err := ctl.addLabelValues(api.TenantLabelKey, tenants); err != nil {
		t.Fatal(err)
	}

	// Create deployment configuration.
	if _, err := ctl.upsertDeploymentConfig(
		api.DeploymentConfigUpsert{
			ProjectID: project.ID,
		},
		deploymentSchdule,
	); err != nil {
		t.Fatal(err)
	}

	// Create issues that create databases.
	baseDatabaseName := "testTenant"
	for i := 0; i < stagingTenantNumber; i++ {
		databaseName := fmt.Sprintf("%s_tenant%d", baseDatabaseName, i)
		if err := ctl.createDatabase(project, stagingInstance, databaseName, map[string]string{api.TenantLabelKey: fmt.Sprintf("tenant%d", i)}); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < prodTenantNumber; i++ {
		databaseName := fmt.Sprintf("%s_tenant%d", baseDatabaseName, i)
		if err := ctl.createDatabase(project, prodInstance, databaseName, map[string]string{api.TenantLabelKey: fmt.Sprintf("tenant%d", i)}); err != nil {
			t.Fatal(err)
		}
	}

	// Getting databases for each environment.
	databases, err := ctl.getDatabases(api.DatabaseFind{
		ProjectID: &project.ID,
	})
	if err != nil {
		t.Fatalf("failed to get databases, error: %v", err)
	}

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
	if len(stagingDatabases) != stagingTenantNumber {
		t.Fatalf("invalid number of staging databases %v in project %v, expecting %d database", project.ID, len(databases), stagingTenantNumber)
	}
	if len(prodDatabases) != prodTenantNumber {
		t.Fatalf("invalid number of prod databases %v in project %v, expecting %d database", project.ID, len(databases), prodTenantNumber)
	}

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
	if err != nil {
		t.Fatalf("failed to construct schema update issue CreateContext payload, error: %v", err)
	}
	issue, err := ctl.createIssue(api.IssueCreate{
		ProjectID:   project.ID,
		Name:        "update schema for tenants",
		Type:        api.IssueDatabaseSchemaUpdate,
		Description: "This updates the schema of tenant databases.",
		// Assign to self.
		AssigneeID:    project.Creator.ID,
		CreateContext: string(createContext),
	})
	if err != nil {
		t.Fatalf("failed to create schema update issue, error: %v", err)
	}
	status, err := ctl.waitIssuePipeline(issue.ID)
	if err != nil {
		t.Fatalf("failed to wait for issue %v pipeline %v, error: %v", issue.ID, issue.Pipeline.ID, err)
	}
	if status != api.TaskDone {
		t.Fatalf("issue %v pipeline %v is expected to finish with status done got %v", issue.ID, issue.Pipeline.ID, status)
	}

	// Query schema.
	for i := 0; i < stagingTenantNumber; i++ {
		databaseName := fmt.Sprintf("%s_tenant%d", baseDatabaseName, i)
		result, err := ctl.query(stagingInstance, databaseName, bookTableQuery)
		if err != nil {
			t.Fatal(err)
		}
		if bookSchemaSQLResult != result {
			t.Fatalf("SQL result want %q, got %q, diff %q", bookSchemaSQLResult, result, pretty.Diff(bookSchemaSQLResult, result))
		}
	}
	for i := 0; i < prodTenantNumber; i++ {
		databaseName := fmt.Sprintf("%s_tenant%d", baseDatabaseName, i)
		result, err := ctl.query(prodInstance, databaseName, bookTableQuery)
		if err != nil {
			t.Fatal(err)
		}
		if bookSchemaSQLResult != result {
			t.Fatalf("SQL result want %q, got %q, diff %q", bookSchemaSQLResult, result, pretty.Diff(bookSchemaSQLResult, result))
		}
	}
}
