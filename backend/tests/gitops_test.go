package tests

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
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
	projectID := generateRandomString("gitops-check", 10)
	project, err := ctl.projectServiceClient.CreateProject(ctx, &v1pb.CreateProjectRequest{
		Project: &v1pb.Project{
			Name:  fmt.Sprintf("projects/%s", projectID),
			Title: projectID,
		},
		ProjectId: projectID,
	})
	a.NoError(err)

	// Provision test and prod instances.
	instanceRootDir := t.TempDir()

	testInstanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, "gitops-check-test")
	a.NoError(err)

	prodInstanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, "gitops-check-prod")
	a.NoError(err)

	// Add the provisioned instances.
	testInstance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance", 10),
		Instance: &v1pb.Instance{
			Title:       "gitops-check-test",
			Engine:      v1pb.Engine_SQLITE,
			Environment: "environments/test",
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: testInstanceDir, Id: "admin"}},
		},
	})
	a.NoError(err)

	prodInstance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance", 10),
		Instance: &v1pb.Instance{
			Title:       "gitops-check-prod",
			Engine:      v1pb.Engine_SQLITE,
			Environment: "environments/prod",
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: prodInstanceDir, Id: "admin"}},
		},
	})
	a.NoError(err)

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
				Type:       v1pb.ReleaseFileType_VERSIONED,
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
				Type:       v1pb.ReleaseFileType_VERSIONED,
				Version:    "002",
				ChangeType: v1pb.Release_File_DDL,
				Statement:  []byte(`CREATE INDEX idx_users_email ON users(email);`),
			},
		},
	}

	// Test 1: Check release against test database.
	checkResp, err := ctl.releaseServiceClient.CheckRelease(ctx, &v1pb.CheckReleaseRequest{
		Parent:  project.Name,
		Release: release,
		Targets: []string{
			fmt.Sprintf("%s/databases/%s", testInstance.Name, databaseName),
		},
	})
	a.NoError(err)
	a.NotNil(checkResp)
	// The response contains results for each file-target combination
	a.Len(checkResp.Results, 2) // 2 files x 1 target = 2 results

	// Verify check results.
	targetCount := make(map[string]int)
	for _, result := range checkResp.Results {
		targetCount[result.Target]++
		// The check should complete successfully.
		a.NotNil(result)
	}
	// Should have 2 results for the single target (one for each file)
	a.Equal(2, targetCount[fmt.Sprintf("%s/databases/%s", testInstance.Name, databaseName)])

	// Test 2: Check release against multiple targets (test and prod).
	checkRespMulti, err := ctl.releaseServiceClient.CheckRelease(ctx, &v1pb.CheckReleaseRequest{
		Parent:  project.Name,
		Release: release,
		Targets: []string{
			fmt.Sprintf("%s/databases/%s", testInstance.Name, databaseName),
			fmt.Sprintf("%s/databases/%s", prodInstance.Name, databaseName),
		},
	})
	a.NoError(err)
	a.NotNil(checkRespMulti)
	// 2 files x 2 targets = 4 results
	a.Len(checkRespMulti.Results, 4)

	// Verify both targets were checked.
	checkedTargets := make(map[string]int)
	for _, result := range checkRespMulti.Results {
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
	projectID := generateRandomString("gitops-rollout", 10)
	project, err := ctl.projectServiceClient.CreateProject(ctx, &v1pb.CreateProjectRequest{
		Project: &v1pb.Project{
			Name:  fmt.Sprintf("projects/%s", projectID),
			Title: projectID,
		},
		ProjectId: projectID,
	})
	a.NoError(err)

	// Provision test instance.
	instanceRootDir := t.TempDir()

	testInstanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, "gitops-rollout-test")
	a.NoError(err)

	// Add the provisioned instance.
	testInstance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance", 10),
		Instance: &v1pb.Instance{
			Title:       "gitops-rollout-test",
			Engine:      v1pb.Engine_SQLITE,
			Environment: "environments/test",
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: testInstanceDir, Id: "admin"}},
		},
	})
	a.NoError(err)

	// Create database.
	databaseName := "gitops_rollout_db"
	err = ctl.createDatabaseV2(ctx, project, testInstance, nil, databaseName, "")
	a.NoError(err)

	// Step 1: Create a release containing migration files.
	createReleaseResp, err := ctl.releaseServiceClient.CreateRelease(ctx, &v1pb.CreateReleaseRequest{
		Parent: project.Name,
		Release: &v1pb.Release{
			Title: "GitOps Rollout Release v1.0",
			Files: []*v1pb.Release_File{
				{
					Path:       "migrations/001__create_products_table.sql",
					Type:       v1pb.ReleaseFileType_VERSIONED,
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
	})
	a.NoError(err)
	a.NotNil(createReleaseResp)

	// Step 2: Create a plan with the release field set.
	plan, err := ctl.planServiceClient.CreatePlan(ctx, &v1pb.CreatePlanRequest{
		Parent: project.Name,
		Plan: &v1pb.Plan{
			Title:       "GitOps Deployment Plan",
			Description: "Deploy to test environment",
			Specs: []*v1pb.Plan_Spec{
				{
					Id: uuid.NewString(),
					Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
						ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
							Release: createReleaseResp.Name,
							Targets: []string{
								fmt.Sprintf("%s/databases/%s", testInstance.Name, databaseName),
							},
							Type: v1pb.Plan_ChangeDatabaseConfig_MIGRATE,
						},
					},
				},
			},
		},
	})
	a.NoError(err)
	a.NotNil(plan)

	// Verify the plan has the release reference.
	a.Len(plan.Specs, 1)
	changeDatabaseConfig := plan.Specs[0].GetChangeDatabaseConfig()
	a.NotNil(changeDatabaseConfig)
	a.Equal(createReleaseResp.Name, changeDatabaseConfig.Release)

	// Step 3: Create a rollout from the plan.
	rollout, err := ctl.rolloutServiceClient.CreateRollout(ctx, &v1pb.CreateRolloutRequest{
		Parent: project.Name,
		Rollout: &v1pb.Rollout{
			Plan: plan.Name,
		},
	})
	a.NoError(err)
	a.NotNil(rollout)

	// Create an issue for the rollout.
	issue, err := ctl.issueServiceClient.CreateIssue(ctx, &v1pb.CreateIssueRequest{
		Parent: project.Name,
		Issue: &v1pb.Issue{
			Title:       "GitOps Rollout Issue",
			Description: "Deploy release via GitOps workflow",
			Type:        v1pb.Issue_DATABASE_CHANGE,
			Plan:        plan.Name,
			Rollout:     rollout.Name,
		},
	})
	a.NoError(err)

	// Wait for the rollout to complete.
	err = ctl.waitRollout(ctx, issue.Name, rollout.Name)
	a.NoError(err)

	// Verify the schema changes were applied.
	testDBSchema, err := ctl.databaseServiceClient.GetDatabaseSchema(ctx, &v1pb.GetDatabaseSchemaRequest{
		Name: fmt.Sprintf("%s/databases/%s/schema", testInstance.Name, databaseName),
	})
	a.NoError(err)
	a.Contains(testDBSchema.Schema, "products")
	a.Contains(testDBSchema.Schema, "id INTEGER PRIMARY KEY AUTOINCREMENT")
	a.Contains(testDBSchema.Schema, "name TEXT NOT NULL")
}
