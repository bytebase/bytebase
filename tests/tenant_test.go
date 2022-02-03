package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/kr/pretty"
)

var (
	stagingInstancesNumber    = 2
	prodInstancesNumber       = 6
	stagingInstanceNamePrefix = "testInstanceStaging"
	prodInstanceNamePrefix    = "testInstanceProd"
)

func TestTenantSchemaUpdate(t *testing.T) {
	ctx := context.Background()
	ctl := &controller{}
	dataDir := t.TempDir()
	if err := ctl.StartMain(ctx, dataDir); err != nil {
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
	for i := 0; i < stagingInstancesNumber; i++ {
		instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, fmt.Sprintf("%s-%d", stagingInstanceNamePrefix, i))
		if err != nil {
			t.Fatal(err)
		}
		stagingInstanceDirs = append(stagingInstanceDirs, instanceDir)
	}
	for i := 0; i < prodInstancesNumber; i++ {
		instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, fmt.Sprintf("%s-%d", prodInstanceNamePrefix, i))
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
			Name:          fmt.Sprintf("%s-%d", stagingInstanceNamePrefix, i),
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
			Name:          fmt.Sprintf("%s-%d", prodInstanceNamePrefix, i),
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
	for i := 0; i < prodInstancesNumber; i++ {
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
		api.DeploymentSchedule{
			Deployments: []*api.Deployment{
				{
					Name: "Staging stage",
					Spec: &api.DeploymentSpec{
						Selector: &api.LabelSelector{
							MatchExpressions: []*api.LabelSelectorRequirement{
								{
									Key:      api.EnvironmentKeyName,
									Operator: api.InOperatorType,
									Values:   []string{"Staging"},
								},
								{
									Key:      api.TenantLabelKey,
									Operator: api.ExistsOperatorType,
								},
							},
						},
					},
				},
				{
					Name: "Prod stage",
					Spec: &api.DeploymentSpec{
						Selector: &api.LabelSelector{
							MatchExpressions: []*api.LabelSelectorRequirement{
								{
									Key:      api.EnvironmentKeyName,
									Operator: api.InOperatorType,
									Values:   []string{"Prod"},
								},
								{
									Key:      api.TenantLabelKey,
									Operator: api.ExistsOperatorType,
								},
							},
						},
					},
				},
			},
		},
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
	if len(stagingDatabases) != stagingInstancesNumber {
		t.Fatalf("invalid number of staging databases %v in project %v, expecting %d database", project.ID, len(databases), stagingInstancesNumber)
	}
	if len(prodDatabases) != prodInstancesNumber {
		t.Fatalf("invalid number of prod databases %v in project %v, expecting %d database", project.ID, len(databases), prodInstancesNumber)
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
		result, err := ctl.query(stagingInstance, databaseName)
		if err != nil {
			t.Fatal(err)
		}
		if schemaSQLResult != result {
			t.Fatalf("SQL result want %q, got %q, diff %q", schemaSQLResult, result, pretty.Diff(schemaSQLResult, result))
		}
	}
	for _, prodInstance := range prodInstances {
		result, err := ctl.query(prodInstance, databaseName)
		if err != nil {
			t.Fatal(err)
		}
		if schemaSQLResult != result {
			t.Fatalf("SQL result want %q, got %q, diff %q", schemaSQLResult, result, pretty.Diff(schemaSQLResult, result))
		}
	}
}
