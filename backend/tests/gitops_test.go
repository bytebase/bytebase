package tests

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func TestGitOpsCheck(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	// Create a project for GitOps testing.
	projectID := generateRandomString("gitops-check")
	projectResp, err := ctl.projectServiceClient.CreateProject(ctx, connect.NewRequest(&v1pb.CreateProjectRequest{
		Project: &v1pb.Project{
			Name:  fmt.Sprintf("projects/%s", projectID),
			Title: projectID,
		},
		ProjectId: projectID,
	}))
	a.NoError(err)
	project := projectResp.Msg

	// Provision test and prod instances.
	instanceRootDir := t.TempDir()

	testInstanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, "gitops-check-test")
	a.NoError(err)

	prodInstanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, "gitops-check-prod")
	a.NoError(err)

	// Add the provisioned instances.
	testInstanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance"),
		Instance: &v1pb.Instance{
			Title:       "gitops-check-test",
			Engine:      v1pb.Engine_SQLITE,
			Environment: stringPtr("environments/test"),
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: testInstanceDir, Id: "admin"}},
		},
	}))
	a.NoError(err)
	testInstance := testInstanceResp.Msg

	prodInstanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance"),
		Instance: &v1pb.Instance{
			Title:       "gitops-check-prod",
			Engine:      v1pb.Engine_SQLITE,
			Environment: stringPtr("environments/prod"),
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: prodInstanceDir, Id: "admin"}},
		},
	}))
	a.NoError(err)
	prodInstance := prodInstanceResp.Msg

	// Create databases.
	databaseName := "gitops_check_db"
	err = ctl.createDatabaseV2(ctx, project, testInstance, nil, databaseName, "")
	a.NoError(err)
	err = ctl.createDatabaseV2(ctx, project, prodInstance, nil, databaseName, "")
	a.NoError(err)

	// Create a release with migration files simulating GitOps workflow.
	release := &v1pb.Release{
		Title: "GitOps Check Release v1.0",
		Files: []*v1pb.Release_File{
			{
				Path:       "migrations/001__create_users_table.sql",
				Type:       v1pb.Release_File_VERSIONED,
				Version:    "001",
				ChangeType: v1pb.Release_File_DDL,
				Statement: []byte(`CREATE TABLE users (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					username TEXT NOT NULL UNIQUE,
					email TEXT NOT NULL,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP
				);`),
			},
			{
				Path:       "migrations/002__add_email_index.sql",
				Type:       v1pb.Release_File_VERSIONED,
				Version:    "002",
				ChangeType: v1pb.Release_File_DDL,
				Statement:  []byte(`CREATE INDEX idx_users_email ON users(email);`),
			},
		},
	}

	// Test 1: Check release against test database.
	checkResp, err := ctl.releaseServiceClient.CheckRelease(ctx, connect.NewRequest(&v1pb.CheckReleaseRequest{
		Parent:  project.Name,
		Release: release,
		Targets: []string{
			fmt.Sprintf("%s/databases/%s", testInstance.Name, databaseName),
		},
	}))
	a.NoError(err)
	a.NotNil(checkResp)
	// The response contains results for each file-target combination
	a.Len(checkResp.Msg.Results, 2) // 2 files x 1 target = 2 results

	// Verify check results.
	targetCount := make(map[string]int)
	for _, result := range checkResp.Msg.Results {
		targetCount[result.Target]++
		// The check should complete successfully.
		a.NotNil(result)
	}
	// Should have 2 results for the single target (one for each file)
	a.Equal(2, targetCount[fmt.Sprintf("%s/databases/%s", testInstance.Name, databaseName)])

	// Test 2: Check release against multiple targets (test and prod).
	checkRespMulti, err := ctl.releaseServiceClient.CheckRelease(ctx, connect.NewRequest(&v1pb.CheckReleaseRequest{
		Parent:  project.Name,
		Release: release,
		Targets: []string{
			fmt.Sprintf("%s/databases/%s", testInstance.Name, databaseName),
			fmt.Sprintf("%s/databases/%s", prodInstance.Name, databaseName),
		},
	}))
	a.NoError(err)
	a.NotNil(checkRespMulti)
	// 2 files x 2 targets = 4 results
	a.Len(checkRespMulti.Msg.Results, 4)

	// Verify both targets were checked.
	checkedTargets := make(map[string]int)
	for _, result := range checkRespMulti.Msg.Results {
		checkedTargets[result.Target]++
	}
	// Each target should have 2 results (one for each file)
	a.Equal(2, checkedTargets[fmt.Sprintf("%s/databases/%s", testInstance.Name, databaseName)])
	a.Equal(2, checkedTargets[fmt.Sprintf("%s/databases/%s", prodInstance.Name, databaseName)])
}

func TestGitOpsRollout(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	// Create a project for GitOps testing.
	projectID := generateRandomString("gitops-rollout")
	projectResp, err := ctl.projectServiceClient.CreateProject(ctx, connect.NewRequest(&v1pb.CreateProjectRequest{
		Project: &v1pb.Project{
			Name:  fmt.Sprintf("projects/%s", projectID),
			Title: projectID,
		},
		ProjectId: projectID,
	}))
	a.NoError(err)
	project := projectResp.Msg

	// Provision test instance.
	instanceRootDir := t.TempDir()

	testInstanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, "gitops-rollout-test")
	a.NoError(err)

	// Add the provisioned instance.
	testInstanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance"),
		Instance: &v1pb.Instance{
			Title:       "gitops-rollout-test",
			Engine:      v1pb.Engine_SQLITE,
			Environment: stringPtr("environments/test"),
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: testInstanceDir, Id: "admin"}},
		},
	}))
	a.NoError(err)
	testInstance := testInstanceResp.Msg

	// Create database.
	databaseName := "gitops_rollout_db"
	err = ctl.createDatabaseV2(ctx, project, testInstance, nil, databaseName, "")
	a.NoError(err)

	// Step 1: Create a release containing migration files.
	createReleaseResp, err := ctl.releaseServiceClient.CreateRelease(ctx, connect.NewRequest(&v1pb.CreateReleaseRequest{
		Parent: project.Name,
		Release: &v1pb.Release{
			Title: "GitOps Rollout Release v1.0",
			Files: []*v1pb.Release_File{
				{
					Path:       "migrations/001__create_products_table.sql",
					Type:       v1pb.Release_File_VERSIONED,
					Version:    "001",
					ChangeType: v1pb.Release_File_DDL,
					Statement: []byte(`CREATE TABLE products (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						name TEXT NOT NULL,
						price DECIMAL(10,2) NOT NULL,
						description TEXT,
						created_at DATETIME DEFAULT CURRENT_TIMESTAMP
					);`),
				},
			},
		},
	}))
	a.NoError(err)
	a.NotNil(createReleaseResp)

	// Step 2: Create a plan with the release field set.
	planResp, err := ctl.planServiceClient.CreatePlan(ctx, connect.NewRequest(&v1pb.CreatePlanRequest{
		Parent: project.Name,
		Plan: &v1pb.Plan{
			Title:       "GitOps Deployment Plan",
			Description: "Deploy to test environment",
			Specs: []*v1pb.Plan_Spec{
				{
					Id: uuid.NewString(),
					Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
						ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
							Release: createReleaseResp.Msg.Name,
							Targets: []string{
								fmt.Sprintf("%s/databases/%s", testInstance.Name, databaseName),
							},
							Type: v1pb.Plan_ChangeDatabaseConfig_MIGRATE,
						},
					},
				},
			},
		},
	}))
	a.NoError(err)
	a.NotNil(planResp)
	plan := planResp.Msg

	// Verify the plan has the release reference.
	a.Len(plan.Specs, 1)
	changeDatabaseConfig := plan.Specs[0].GetChangeDatabaseConfig()
	a.NotNil(changeDatabaseConfig)
	a.Equal(createReleaseResp.Msg.Name, changeDatabaseConfig.Release)

	// Step 3: Create a rollout from the plan.
	rolloutResp, err := ctl.rolloutServiceClient.CreateRollout(ctx, connect.NewRequest(&v1pb.CreateRolloutRequest{
		Parent: project.Name,
		Rollout: &v1pb.Rollout{
			Plan: plan.Name,
		},
	}))
	a.NoError(err)
	a.NotNil(rolloutResp)
	rollout := rolloutResp.Msg

	// Create an issue for the rollout.
	issueResp, err := ctl.issueServiceClient.CreateIssue(ctx, connect.NewRequest(&v1pb.CreateIssueRequest{
		Parent: project.Name,
		Issue: &v1pb.Issue{
			Title:       "GitOps Rollout Issue",
			Description: "Deploy release via GitOps workflow",
			Type:        v1pb.Issue_DATABASE_CHANGE,
			Plan:        plan.Name,
			Rollout:     rollout.Name,
		},
	}))
	a.NoError(err)
	issue := issueResp.Msg

	// Wait for the rollout to complete.
	err = ctl.waitRollout(ctx, issue.Name, rollout.Name)
	a.NoError(err)

	// Verify the schema changes were applied.
	testDBSchemaResp, err := ctl.databaseServiceClient.GetDatabaseSchema(ctx, connect.NewRequest(&v1pb.GetDatabaseSchemaRequest{
		Name: fmt.Sprintf("%s/databases/%s/schema", testInstance.Name, databaseName),
	}))
	a.NoError(err)
	testDBSchema := testDBSchemaResp.Msg
	a.Contains(testDBSchema.Schema, "products")
	a.Contains(testDBSchema.Schema, "id INTEGER PRIMARY KEY AUTOINCREMENT")
	a.Contains(testDBSchema.Schema, "name TEXT NOT NULL")

	// Verify database revision after migration using RevisionService.
	revisionsResp, err := ctl.revisionServiceClient.ListRevisions(ctx, connect.NewRequest(&v1pb.ListRevisionsRequest{
		Parent: fmt.Sprintf("%s/databases/%s", testInstance.Name, databaseName),
	}))
	a.NoError(err)
	revisions := revisionsResp.Msg
	a.NotEmpty(revisions.Revisions, "Database should have revisions after migration")
	a.Len(revisions.Revisions, 1, "Should have exactly 1 revision for the single migration file")

	// Verify the revision details.
	revision := revisions.Revisions[0]
	a.NotEmpty(revision.Name, "Revision should have a name")
	a.Equal(createReleaseResp.Msg.Name, revision.Release, "Revision should reference the correct release")
	a.NotEmpty(revision.Version, "Revision should have a version")
	a.NotNil(revision.CreateTime, "Revision should have a create time")

	// Call CreateRollout on the finished plan.
	// the rollout name is the same as the rollout created above.
	rolloutResp2, err := ctl.rolloutServiceClient.CreateRollout(ctx, connect.NewRequest(&v1pb.CreateRolloutRequest{
		Parent: project.Name,
		Rollout: &v1pb.Rollout{
			Plan: plan.Name,
		},
	}))
	a.NoError(err)
	a.NotNil(rolloutResp2)
	a.Equal(rollout.Name, rolloutResp2.Msg.Name)

	// Call CreateRollout with ValidateOnly on the finished plan.
	// the rollout should contain zero stages.
	rolloutResp3, err := ctl.rolloutServiceClient.CreateRollout(ctx, connect.NewRequest(&v1pb.CreateRolloutRequest{
		Parent: project.Name,
		Rollout: &v1pb.Rollout{
			Plan: plan.Name,
		},
		ValidateOnly: true,
	}))
	a.NoError(err)
	a.NotNil(rolloutResp3)
	a.Empty(rolloutResp3.Msg.Stages)
}

// TestGitOpsRolloutMultiTarget tests a more complex GitOps scenario:
// - A release with 3 migration files targeting 2 databases (test and prod)
// - Verifies that all files are applied to all target databases
// - Ensures schema consistency across environments
func TestGitOpsRolloutMultiTarget(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	// Create a project for GitOps testing.
	projectID := generateRandomString("gitops-multi")
	projectResp, err := ctl.projectServiceClient.CreateProject(ctx, connect.NewRequest(&v1pb.CreateProjectRequest{
		Project: &v1pb.Project{
			Name:  fmt.Sprintf("projects/%s", projectID),
			Title: projectID,
		},
		ProjectId: projectID,
	}))
	a.NoError(err)
	project := projectResp.Msg

	// Provision test and prod instances.
	instanceRootDir := t.TempDir()

	testInstanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, "gitops-multi-test")
	a.NoError(err)

	prodInstanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, "gitops-multi-prod")
	a.NoError(err)

	// Add the provisioned instances.
	testInstanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance"),
		Instance: &v1pb.Instance{
			Title:       "gitops-multi-test",
			Engine:      v1pb.Engine_SQLITE,
			Environment: stringPtr("environments/test"),
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: testInstanceDir, Id: "admin"}},
		},
	}))
	a.NoError(err)
	testInstance := testInstanceResp.Msg

	prodInstanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance"),
		Instance: &v1pb.Instance{
			Title:       "gitops-multi-prod",
			Engine:      v1pb.Engine_SQLITE,
			Environment: stringPtr("environments/prod"),
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: prodInstanceDir, Id: "admin"}},
		},
	}))
	a.NoError(err)
	prodInstance := prodInstanceResp.Msg

	// Create databases.
	databaseName := "gitops_multi_db"
	err = ctl.createDatabaseV2(ctx, project, testInstance, nil, databaseName, "")
	a.NoError(err)
	err = ctl.createDatabaseV2(ctx, project, prodInstance, nil, databaseName, "")
	a.NoError(err)

	// Step 1: Create a release containing 3 simple migration files.
	createReleaseResp, err := ctl.releaseServiceClient.CreateRelease(ctx, connect.NewRequest(&v1pb.CreateReleaseRequest{
		Parent: project.Name,
		Release: &v1pb.Release{
			Title: "GitOps Multi-Target Release v1.0",
			Files: []*v1pb.Release_File{
				{
					Path:       "migrations/1.0.0__create_table_one.sql",
					Type:       v1pb.Release_File_VERSIONED,
					Version:    "1.0.0",
					ChangeType: v1pb.Release_File_DDL,
					Statement: []byte(`CREATE TABLE table_one (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						name TEXT NOT NULL
					);`),
				},
				{
					Path:       "migrations/1.0.1__create_table_two.sql",
					Type:       v1pb.Release_File_VERSIONED,
					Version:    "1.0.1",
					ChangeType: v1pb.Release_File_DDL,
					Statement: []byte(`CREATE TABLE table_two (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						value TEXT NOT NULL
					);`),
				},
				{
					Path:       "migrations/1.0.2__create_table_three.sql",
					Type:       v1pb.Release_File_VERSIONED,
					Version:    "1.0.2",
					ChangeType: v1pb.Release_File_DDL,
					Statement: []byte(`CREATE TABLE table_three (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						data TEXT NULL
					);`),
				},
			},
		},
	}))
	a.NoError(err)
	a.NotNil(createReleaseResp)

	// Step 2: Create a plan targeting both test and prod databases.
	planResp, err := ctl.planServiceClient.CreatePlan(ctx, connect.NewRequest(&v1pb.CreatePlanRequest{
		Parent: project.Name,
		Plan: &v1pb.Plan{
			Title:       "GitOps Multi-Target Deployment",
			Description: "Deploy 3 migration files to test and prod environments",
			Specs: []*v1pb.Plan_Spec{
				{
					Id: uuid.NewString(),
					Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
						ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
							Release: createReleaseResp.Msg.Name,
							Targets: []string{
								fmt.Sprintf("%s/databases/%s", testInstance.Name, databaseName),
								fmt.Sprintf("%s/databases/%s", prodInstance.Name, databaseName),
							},
							Type: v1pb.Plan_ChangeDatabaseConfig_MIGRATE,
						},
					},
				},
			},
		},
	}))
	a.NoError(err)
	a.NotNil(planResp)
	plan := planResp.Msg

	// Verify the plan configuration.
	a.Len(plan.Specs, 1)
	changeDatabaseConfig := plan.Specs[0].GetChangeDatabaseConfig()
	a.NotNil(changeDatabaseConfig)
	a.Equal(createReleaseResp.Msg.Name, changeDatabaseConfig.Release)
	a.Len(changeDatabaseConfig.Targets, 2) // Both test and prod databases

	// Step 3: Create a rollout from the plan.
	rolloutResp, err := ctl.rolloutServiceClient.CreateRollout(ctx, connect.NewRequest(&v1pb.CreateRolloutRequest{
		Parent: project.Name,
		Rollout: &v1pb.Rollout{
			Plan: plan.Name,
		},
	}))
	a.NoError(err)
	a.NotNil(rolloutResp)
	rollout := rolloutResp.Msg

	// Create an issue for the rollout.
	issueResp, err := ctl.issueServiceClient.CreateIssue(ctx, connect.NewRequest(&v1pb.CreateIssueRequest{
		Parent: project.Name,
		Issue: &v1pb.Issue{
			Title:       "GitOps Multi-Target Rollout",
			Description: "Deploy 3-file release to test and prod via GitOps workflow",
			Type:        v1pb.Issue_DATABASE_CHANGE,
			Plan:        plan.Name,
			Rollout:     rollout.Name,
		},
	}))
	a.NoError(err)
	issue := issueResp.Msg

	// Wait for the rollout to complete.
	err = ctl.waitRollout(ctx, issue.Name, rollout.Name)
	a.NoError(err)

	// Verify schema changes were applied to test database.
	testDBSchemaResp, err := ctl.databaseServiceClient.GetDatabaseSchema(ctx, connect.NewRequest(&v1pb.GetDatabaseSchemaRequest{
		Name: fmt.Sprintf("%s/databases/%s/schema", testInstance.Name, databaseName),
	}))
	a.NoError(err)
	testDBSchema := testDBSchemaResp.Msg

	// Verify all 3 migration files were applied to test database.
	a.Contains(testDBSchema.Schema, "table_one")
	a.Contains(testDBSchema.Schema, "table_two")
	a.Contains(testDBSchema.Schema, "table_three")
	a.Contains(testDBSchema.Schema, "name TEXT NOT NULL")
	a.Contains(testDBSchema.Schema, "value TEXT NOT NULL")
	a.Contains(testDBSchema.Schema, "data TEXT")

	// Verify schema changes were applied to prod database.
	prodDBSchemaResp, err := ctl.databaseServiceClient.GetDatabaseSchema(ctx, connect.NewRequest(&v1pb.GetDatabaseSchemaRequest{
		Name: fmt.Sprintf("%s/databases/%s/schema", prodInstance.Name, databaseName),
	}))
	a.NoError(err)
	prodDBSchema := prodDBSchemaResp.Msg

	// Verify all 3 migration files were applied to prod database.
	a.Contains(prodDBSchema.Schema, "table_one")
	a.Contains(prodDBSchema.Schema, "table_two")
	a.Contains(prodDBSchema.Schema, "table_three")
	a.Contains(prodDBSchema.Schema, "name TEXT NOT NULL")
	a.Contains(prodDBSchema.Schema, "value TEXT NOT NULL")
	a.Contains(prodDBSchema.Schema, "data TEXT")

	// Additional verification: Ensure both databases have identical schemas.
	a.Equal(testDBSchema.Schema, prodDBSchema.Schema, "Test and prod databases should have identical schemas after deployment")

	// Verify database revisions after migrations using RevisionService.
	testRevisionsResp, err := ctl.revisionServiceClient.ListRevisions(ctx, connect.NewRequest(&v1pb.ListRevisionsRequest{
		Parent: fmt.Sprintf("%s/databases/%s", testInstance.Name, databaseName),
	}))
	a.NoError(err)
	testRevisions := testRevisionsResp.Msg
	a.NotEmpty(testRevisions.Revisions, "Test database should have revisions after migration")
	a.Len(testRevisions.Revisions, 3, "Test database should have exactly 3 revisions for the 3 migration files")

	prodRevisionsResp, err := ctl.revisionServiceClient.ListRevisions(ctx, connect.NewRequest(&v1pb.ListRevisionsRequest{
		Parent: fmt.Sprintf("%s/databases/%s", prodInstance.Name, databaseName),
	}))
	a.NoError(err)
	prodRevisions := prodRevisionsResp.Msg
	a.NotEmpty(prodRevisions.Revisions, "Prod database should have revisions after migration")
	a.Len(prodRevisions.Revisions, 3, "Prod database should have exactly 3 revisions for the 3 migration files")

	// Verify revision details for test database.
	testVersions := make([]string, 0, len(testRevisions.Revisions))
	for _, revision := range testRevisions.Revisions {
		a.NotEmpty(revision.Name, "Test revision should have a name")
		a.Equal(createReleaseResp.Msg.Name, revision.Release, "Test revision should reference the correct release")
		a.NotEmpty(revision.Version, "Test revision should have a version")
		a.NotNil(revision.CreateTime, "Test revision should have a create time")
		testVersions = append(testVersions, revision.Version)
	}

	// Verify revision details for prod database.
	prodVersions := make([]string, 0, len(prodRevisions.Revisions))
	for _, revision := range prodRevisions.Revisions {
		a.NotEmpty(revision.Name, "Prod revision should have a name")
		a.Equal(createReleaseResp.Msg.Name, revision.Release, "Prod revision should reference the correct release")
		a.NotEmpty(revision.Version, "Prod revision should have a version")
		a.NotNil(revision.CreateTime, "Prod revision should have a create time")
		prodVersions = append(prodVersions, revision.Version)
	}

	// Both databases should have the same revision versions since they received the same migrations.
	a.ElementsMatch(testVersions, prodVersions, "Test and prod databases should have the same revision versions")

	// Verify that we have the expected versions (1.0.0, 1.0.1, 1.0.2).
	expectedVersions := []string{"1.0.0", "1.0.1", "1.0.2"}
	a.ElementsMatch(testVersions, expectedVersions, "Test database should have the expected migration versions")
	a.ElementsMatch(prodVersions, expectedVersions, "Prod database should have the expected migration versions")

	// Call CreateRollout on the finished plan.
	// the rollout name is the same as the rollout created above.
	rolloutResp2, err := ctl.rolloutServiceClient.CreateRollout(ctx, connect.NewRequest(&v1pb.CreateRolloutRequest{
		Parent: project.Name,
		Rollout: &v1pb.Rollout{
			Plan: plan.Name,
		},
	}))
	a.NoError(err)
	a.NotNil(rolloutResp2)
	a.Equal(rollout.Name, rolloutResp2.Msg.Name)

	// Call CreateRollout with ValidateOnly on the finished plan.
	// the rollout should contain zero stages.
	rolloutResp3, err := ctl.rolloutServiceClient.CreateRollout(ctx, connect.NewRequest(&v1pb.CreateRolloutRequest{
		Parent: project.Name,
		Rollout: &v1pb.Rollout{
			Plan: plan.Name,
		},
		ValidateOnly: true,
	}))
	a.NoError(err)
	a.NotNil(rolloutResp3)
	a.Empty(rolloutResp3.Msg.Stages)
}

// TestGitOpsCheckAppliedButChanged tests that CheckRelease detects files that have been applied but with different content.
func TestGitOpsCheckAppliedButChanged(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	// Create a project for GitOps testing.
	projectID := generateRandomString("gitops-changed")
	projectResp, err := ctl.projectServiceClient.CreateProject(ctx, connect.NewRequest(&v1pb.CreateProjectRequest{
		Project: &v1pb.Project{
			Name:  fmt.Sprintf("projects/%s", projectID),
			Title: projectID,
		},
		ProjectId: projectID,
	}))
	a.NoError(err)
	project := projectResp.Msg

	// Provision test instance.
	instanceRootDir := t.TempDir()
	testInstanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, "gitops-changed-test")
	a.NoError(err)

	// Add the provisioned instance.
	testInstanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance"),
		Instance: &v1pb.Instance{
			Title:       "gitops-changed-test",
			Engine:      v1pb.Engine_SQLITE,
			Environment: stringPtr("environments/test"),
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: testInstanceDir, Id: "admin"}},
		},
	}))
	a.NoError(err)
	testInstance := testInstanceResp.Msg

	// Create database.
	databaseName := "gitops_changed_db"
	err = ctl.createDatabaseV2(ctx, project, testInstance, nil, databaseName, "")
	a.NoError(err)

	// Step 1: Create a release with version 1.0.0 file.
	originalRelease := &v1pb.Release{
		Title: "Original Release v1.0.0",
		Files: []*v1pb.Release_File{
			{
				Path:       "migrations/1.0.0__create_users_table.sql",
				Type:       v1pb.Release_File_VERSIONED,
				Version:    "1.0.0",
				ChangeType: v1pb.Release_File_DDL,
				Statement: []byte(`CREATE TABLE users (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					username TEXT NOT NULL,
					email TEXT NOT NULL
				);`),
			},
		},
	}

	originalReleaseResp, err := ctl.releaseServiceClient.CreateRelease(ctx, connect.NewRequest(&v1pb.CreateReleaseRequest{
		Parent:  project.Name,
		Release: originalRelease,
	}))
	a.NoError(err)

	// Step 2: Apply the release to a database.
	planResp, err := ctl.planServiceClient.CreatePlan(ctx, connect.NewRequest(&v1pb.CreatePlanRequest{
		Parent: project.Name,
		Plan: &v1pb.Plan{
			Title:       "Apply Original Release",
			Description: "Apply original release to database",
			Specs: []*v1pb.Plan_Spec{
				{
					Id: uuid.NewString(),
					Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
						ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
							Release: originalReleaseResp.Msg.Name,
							Targets: []string{
								fmt.Sprintf("%s/databases/%s", testInstance.Name, databaseName),
							},
							Type: v1pb.Plan_ChangeDatabaseConfig_MIGRATE,
						},
					},
				},
			},
		},
	}))
	a.NoError(err)
	plan := planResp.Msg

	rolloutResp, err := ctl.rolloutServiceClient.CreateRollout(ctx, connect.NewRequest(&v1pb.CreateRolloutRequest{
		Parent: project.Name,
		Rollout: &v1pb.Rollout{
			Plan: plan.Name,
		},
	}))
	a.NoError(err)
	rollout := rolloutResp.Msg

	issueResp, err := ctl.issueServiceClient.CreateIssue(ctx, connect.NewRequest(&v1pb.CreateIssueRequest{
		Parent: project.Name,
		Issue: &v1pb.Issue{
			Title:       "Apply Original Release",
			Description: "Apply original release to database",
			Type:        v1pb.Issue_DATABASE_CHANGE,
			Plan:        plan.Name,
			Rollout:     rollout.Name,
		},
	}))
	a.NoError(err)
	issue := issueResp.Msg

	// Wait for the rollout to complete.
	err = ctl.waitRollout(ctx, issue.Name, rollout.Name)
	a.NoError(err)

	// Step 3: Create a release with version 1.0.0 file but with different content.
	modifiedRelease := &v1pb.Release{
		Title: "Modified Release v1.0.0",
		Files: []*v1pb.Release_File{
			{
				Path:       "migrations/1.0.0__create_users_table.sql",
				Type:       v1pb.Release_File_VERSIONED,
				Version:    "1.0.0",
				ChangeType: v1pb.Release_File_DDL,
				Statement: []byte(`CREATE TABLE users (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					username TEXT NOT NULL,
					email TEXT NOT NULL,
					phone TEXT,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP
				);`),
			},
		},
	}

	// Step 4: Call CheckRelease with the modified release against the same target.
	checkResp, err := ctl.releaseServiceClient.CheckRelease(ctx, connect.NewRequest(&v1pb.CheckReleaseRequest{
		Parent:  project.Name,
		Release: modifiedRelease,
		Targets: []string{
			fmt.Sprintf("%s/databases/%s", testInstance.Name, databaseName),
		},
	}))
	a.NoError(err)
	a.NotNil(checkResp)

	// Step 5: Expect that CheckRelease returns a warning about the changed file.
	a.Len(checkResp.Msg.Results, 1, "Should have 1 result for the single file-target combination")

	result := checkResp.Msg.Results[0]
	a.Equal(fmt.Sprintf("%s/databases/%s", testInstance.Name, databaseName), result.Target)
	a.Equal("migrations/1.0.0__create_users_table.sql", result.File)

	// Verify that CheckRelease detected the file has been applied but modified.
	a.Len(result.Advices, 1, "Should have exactly 1 advice about the modified file")

	advice := result.Advices[0]
	a.Equal(v1pb.Advice_WARNING, advice.Status, "Should return WARNING status for modified applied file")
	a.Equal("Applied file has been modified", advice.Title, "Should have correct warning title")
	a.Contains(advice.Content, "has already been applied to the database, but its content has been modified",
		"Should warn that the file was applied but content changed")
	a.Contains(advice.Content, "Applied SHA256:", "Should include the SHA256 of the applied file")
	a.Contains(advice.Content, "Release SHA256:", "Should include the SHA256 of the new release file")
	a.Contains(advice.Content, "migrations/1.0.0__create_users_table.sql", "Should mention the specific file")
	a.Contains(advice.Content, "version \"1.0.0\"", "Should mention the version")
}
