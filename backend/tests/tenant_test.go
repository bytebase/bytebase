package tests

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/type/expr"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// TestTenantBackfill demonstrates the workflow for onboarding new tenant databases
// in a multi-tenant architecture with database groups.
//
// This test implements the workflow described in the tenant integration design document.
//
// =============================================================================
// WORKFLOW OVERVIEW
// =============================================================================
//
// When a new tenant database is created, it needs to:
// 1. Have the same schema as existing tenants (baseline database)
// 2. Be included in any pending rollouts that haven't been executed yet
//
// =============================================================================
// KEY CONCEPTS
// =============================================================================
//
//   - Baseline Database: The "source of truth" tenant database. New tenants copy
//     their initial schema from the baseline.
//
//   - Database Group: A logical grouping of tenant databases that receive the same
//     database changes. Uses CEL expressions to match databases dynamically.
//
//   - Idempotent CreateRollout: Calling CreateRollout multiple times on the same
//     plan is safe - it re-evaluates the database group and adds newly matched
//     databases without duplicating existing tasks.
//
// =============================================================================
// API WORKFLOW
// =============================================================================
//
// Step 1: Get baseline schema
//
//	DatabaseService.GetDatabaseSchema(baseline_db)
//
// Step 2: Create new tenant database
//
//	PlanService.CreatePlan + IssueService.CreateIssue + RolloutService.CreateRollout
//
// Step 3: Initialize new tenant with baseline schema
//
//	Apply baseline schema to new tenant via ChangeDatabaseConfig
//
// Step 4: Find pending rollouts (baseline has NOT_STARTED tasks)
//
//	PlanService.ListPlans(filter="has_rollout == true")
//	RolloutService.GetRollout() -> check task.Status
//
// Step 5: Add new tenant to pending rollouts
//
//	RolloutService.CreateRollout(plan) - idempotent, adds new tenant
//
// Step 6: [OPTIONAL] Execute pending tasks when ready
//
//	RolloutService.BatchRunTasks() or via Bytebase UI
func TestTenantBackfill(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	// ==========================================================================
	// SETUP: Create project and instance
	// ==========================================================================

	projectID := generateRandomString("project")
	project, err := ctl.projectServiceClient.CreateProject(ctx, connect.NewRequest(&v1pb.CreateProjectRequest{
		Project:   &v1pb.Project{Name: fmt.Sprintf("projects/%s", projectID), Title: projectID},
		ProjectId: projectID,
	}))
	a.NoError(err)

	instanceRootDir := t.TempDir()
	prodInstanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, "prod-instance")
	a.NoError(err)

	prodInstance, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance"),
		Instance: &v1pb.Instance{
			Title:       "prod-instance",
			Engine:      v1pb.Engine_SQLITE,
			Environment: stringPtr("environments/prod"),
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: prodInstanceDir, Id: "admin"}},
		},
	}))
	a.NoError(err)

	// ==========================================================================
	// STEP 1: Create baseline database
	// ==========================================================================
	// The baseline database is the "source of truth" for tenant schemas.
	// It should be in the most stable production environment.

	baselineDBName := "tenant_baseline"
	err = ctl.createDatabase(ctx, project.Msg, prodInstance.Msg, nil, baselineDBName, "")
	a.NoError(err)

	// ==========================================================================
	// STEP 2: Create database group for all tenants
	// ==========================================================================
	// Database groups use CEL expressions to dynamically match databases.
	// Expression "true" matches all databases in the project.

	databaseGroup, err := ctl.databaseGroupServiceClient.CreateDatabaseGroup(ctx, connect.NewRequest(&v1pb.CreateDatabaseGroupRequest{
		Parent:          project.Msg.Name,
		DatabaseGroupId: "tenants",
		DatabaseGroup: &v1pb.DatabaseGroup{
			Title:        "All Tenants",
			DatabaseExpr: &expr.Expr{Expression: "true"},
		},
	}))
	a.NoError(err)

	// ==========================================================================
	// STEP 3: Execute first database change on baseline (creates changelog)
	// ==========================================================================
	// This change is already executed - it will be in baseline's changelog.

	sheet1, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
		Parent: project.Msg.Name,
		Sheet:  &v1pb.Sheet{Content: []byte(migrationStatement1)}, // CREATE TABLE book
	}))
	a.NoError(err)

	err = ctl.changeDatabaseWithConfig(ctx, project.Msg, &v1pb.Plan_Spec{
		Id: uuid.NewString(),
		Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
			ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
				Targets: []string{databaseGroup.Msg.Name},
				Sheet:   sheet1.Msg.Name,
			},
		},
	})
	a.NoError(err)

	// Verify baseline has the schema.
	baselineSchema, err := ctl.databaseServiceClient.GetDatabaseSchema(ctx, connect.NewRequest(&v1pb.GetDatabaseSchemaRequest{
		Name: fmt.Sprintf("%s/databases/%s/schema", prodInstance.Msg.Name, baselineDBName),
	}))
	a.NoError(err)
	a.Equal(wantBookSchema, baselineSchema.Msg.Schema)

	// ==========================================================================
	// STEP 4: Create second database change but DON'T execute it (pending rollout)
	// ==========================================================================
	// This simulates a change that is approved but waiting for execution
	// (e.g., scheduled for maintenance window).

	sheet2, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
		Parent: project.Msg.Name,
		Sheet:  &v1pb.Sheet{Content: []byte(`ALTER TABLE book ADD COLUMN author TEXT;`)},
	}))
	a.NoError(err)

	pendingPlan, err := ctl.planServiceClient.CreatePlan(ctx, connect.NewRequest(&v1pb.CreatePlanRequest{
		Parent: project.Msg.Name,
		Plan: &v1pb.Plan{
			Title: "Add author column to book table",
			Specs: []*v1pb.Plan_Spec{{
				Id: uuid.NewString(),
				Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
					ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
						Targets: []string{databaseGroup.Msg.Name},
						Sheet:   sheet2.Msg.Name,
					},
				},
			}},
		},
	}))
	a.NoError(err)

	pendingIssue, err := ctl.issueServiceClient.CreateIssue(ctx, connect.NewRequest(&v1pb.CreateIssueRequest{
		Parent: project.Msg.Name,
		Issue: &v1pb.Issue{
			Type:        v1pb.Issue_DATABASE_CHANGE,
			Title:       "Add author column",
			Description: "Pending database change",
			Plan:        pendingPlan.Msg.Name,
		},
	}))
	a.NoError(err)

	pendingRollout, err := ctl.rolloutServiceClient.CreateRollout(ctx, connect.NewRequest(&v1pb.CreateRolloutRequest{
		Parent: pendingPlan.Msg.Name,
	}))
	a.NoError(err)

	// Verify: rollout has 1 task for baseline, status is NOT_STARTED.
	a.Len(pendingRollout.Msg.Stages, 1)
	a.Len(pendingRollout.Msg.Stages[0].Tasks, 1)
	a.Equal(v1pb.Task_NOT_STARTED, pendingRollout.Msg.Stages[0].Tasks[0].Status)

	// ==========================================================================
	// NEW TENANT ONBOARDING WORKFLOW
	// ==========================================================================
	// Now we simulate a new tenant being onboarded while there's a pending database change.

	// --------------------------------------------------------------------------
	// STEP 5: Get baseline's current schema
	// --------------------------------------------------------------------------
	// API: DatabaseService.GetDatabaseSchema()

	baselineCurrentSchema, err := ctl.databaseServiceClient.GetDatabaseSchema(ctx, connect.NewRequest(&v1pb.GetDatabaseSchemaRequest{
		Name: fmt.Sprintf("%s/databases/%s/schema", prodInstance.Msg.Name, baselineDBName),
	}))
	a.NoError(err)

	// --------------------------------------------------------------------------
	// STEP 6: Create new tenant database
	// --------------------------------------------------------------------------

	newTenantDBName := "tenant_new"
	err = ctl.createDatabase(ctx, project.Msg, prodInstance.Msg, nil, newTenantDBName, "")
	a.NoError(err)

	// --------------------------------------------------------------------------
	// STEP 7: Initialize new tenant with baseline schema
	// --------------------------------------------------------------------------
	// Apply the baseline's current schema to the new tenant database.
	// This ensures the new tenant has all previously executed database changes.

	initSheet, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
		Parent: project.Msg.Name,
		Sheet:  &v1pb.Sheet{Content: []byte(baselineCurrentSchema.Msg.Schema)},
	}))
	a.NoError(err)

	newTenantDBPath := fmt.Sprintf("%s/databases/%s", prodInstance.Msg.Name, newTenantDBName)
	initPlan, err := ctl.planServiceClient.CreatePlan(ctx, connect.NewRequest(&v1pb.CreatePlanRequest{
		Parent: project.Msg.Name,
		Plan: &v1pb.Plan{
			Title: "Initialize new tenant schema from baseline",
			Specs: []*v1pb.Plan_Spec{{
				Id: uuid.NewString(),
				Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
					ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
						Targets: []string{newTenantDBPath}, // Direct target, not database group
						Sheet:   initSheet.Msg.Name,
					},
				},
			}},
		},
	}))
	a.NoError(err)

	initIssue, err := ctl.issueServiceClient.CreateIssue(ctx, connect.NewRequest(&v1pb.CreateIssueRequest{
		Parent: project.Msg.Name,
		Issue: &v1pb.Issue{
			Type:        v1pb.Issue_DATABASE_CHANGE,
			Title:       "Initialize new tenant from baseline",
			Description: "Apply baseline schema to new tenant",
			Plan:        initPlan.Msg.Name,
		},
	}))
	a.NoError(err)

	initRollout, err := ctl.rolloutServiceClient.CreateRollout(ctx, connect.NewRequest(&v1pb.CreateRolloutRequest{
		Parent: initPlan.Msg.Name,
	}))
	a.NoError(err)

	err = ctl.waitRollout(ctx, initIssue.Msg.Name, initRollout.Msg.Name)
	a.NoError(err)

	// Verify: New tenant now has same schema as baseline.
	newTenantSchema, err := ctl.databaseServiceClient.GetDatabaseSchema(ctx, connect.NewRequest(&v1pb.GetDatabaseSchemaRequest{
		Name: fmt.Sprintf("%s/databases/%s/schema", prodInstance.Msg.Name, newTenantDBName),
	}))
	a.NoError(err)
	a.Equal(baselineCurrentSchema.Msg.Schema, newTenantSchema.Msg.Schema)

	// --------------------------------------------------------------------------
	// STEP 8: Find pending rollouts for baseline
	// --------------------------------------------------------------------------
	// Query plans with rollouts and check which ones have NOT_STARTED tasks
	// for the baseline database.
	//
	// API: PlanService.ListPlans(filter="has_rollout == true")
	//      RolloutService.GetRollout()

	plans, err := ctl.planServiceClient.ListPlans(ctx, connect.NewRequest(&v1pb.ListPlansRequest{
		Parent: project.Msg.Name,
		Filter: "has_rollout == true",
	}))
	a.NoError(err)

	// Find plans where baseline has NOT_STARTED tasks.
	var pendingPlansForBaseline []*v1pb.Plan
	for _, p := range plans.Msg.Plans {
		rollout, err := ctl.rolloutServiceClient.GetRollout(ctx, connect.NewRequest(&v1pb.GetRolloutRequest{
			Name: fmt.Sprintf("%s/rollout", p.Name),
		}))
		a.NoError(err)

		for _, stage := range rollout.Msg.Stages {
			for _, task := range stage.Tasks {
				if strings.Contains(task.Target, baselineDBName) && task.Status == v1pb.Task_NOT_STARTED {
					pendingPlansForBaseline = append(pendingPlansForBaseline, p)
					break
				}
			}
		}
	}

	// Should find the pending plan we created earlier.
	a.Len(pendingPlansForBaseline, 1)
	a.Equal(pendingPlan.Msg.Name, pendingPlansForBaseline[0].Name)

	// --------------------------------------------------------------------------
	// STEP 9: Add new tenant to pending rollouts (idempotent CreateRollout)
	// --------------------------------------------------------------------------
	// Calling CreateRollout again on the same plan will:
	// - Re-evaluate the database group expression
	// - Add tasks for newly matched databases (our new tenant)
	// - NOT duplicate tasks for existing databases
	//
	// API: RolloutService.CreateRollout()

	for _, p := range pendingPlansForBaseline {
		updatedRollout, err := ctl.rolloutServiceClient.CreateRollout(ctx, connect.NewRequest(&v1pb.CreateRolloutRequest{
			Parent: p.Name,
		}))
		a.NoError(err)

		// Verify: Rollout now has tasks for BOTH baseline and new tenant.
		a.Len(updatedRollout.Msg.Stages, 1)
		a.Len(updatedRollout.Msg.Stages[0].Tasks, 2) // baseline + new tenant

		// Both tasks should be NOT_STARTED.
		for _, task := range updatedRollout.Msg.Stages[0].Tasks {
			a.Equal(v1pb.Task_NOT_STARTED, task.Status)
		}
	}

	// ==========================================================================
	// BACKFILL COMPLETE
	// ==========================================================================
	// At this point, the automation workflow is complete:
	//
	// - New tenant has baseline's current schema (all executed database changes)
	// - New tenant is included in all pending rollouts (NOT_STARTED database changes)
	//
	// Executing the pending tasks is OPTIONAL and depends on user's choice.
	// Users can execute via:
	// - Bytebase UI (click "Run" on tasks)
	// - API: RolloutService.BatchRunTasks()
	// - Scheduled execution time

	// Verify final state before optional execution.
	finalRollout, err := ctl.rolloutServiceClient.GetRollout(ctx, connect.NewRequest(&v1pb.GetRolloutRequest{
		Name: pendingRollout.Msg.Name,
	}))
	a.NoError(err)

	notStartedCount := 0
	for _, stage := range finalRollout.Msg.Stages {
		for _, task := range stage.Tasks {
			if task.Status == v1pb.Task_NOT_STARTED {
				notStartedCount++
			}
		}
	}
	// Both baseline and new tenant tasks should be NOT_STARTED.
	a.Equal(2, notStartedCount)

	// ==========================================================================
	// OPTIONAL: Execute pending tasks
	// ==========================================================================
	// This section demonstrates task execution. In production, this would be
	// triggered by user action or scheduled execution.

	err = ctl.waitRollout(ctx, pendingIssue.Msg.Name, pendingRollout.Msg.Name)
	a.NoError(err)

	// Verify: Both databases now have the same final schema.
	finalBaselineSchema, err := ctl.databaseServiceClient.GetDatabaseSchema(ctx, connect.NewRequest(&v1pb.GetDatabaseSchemaRequest{
		Name: fmt.Sprintf("%s/databases/%s/schema", prodInstance.Msg.Name, baselineDBName),
	}))
	a.NoError(err)

	finalNewTenantSchema, err := ctl.databaseServiceClient.GetDatabaseSchema(ctx, connect.NewRequest(&v1pb.GetDatabaseSchemaRequest{
		Name: fmt.Sprintf("%s/databases/%s/schema", prodInstance.Msg.Name, newTenantDBName),
	}))
	a.NoError(err)

	a.Equal(finalBaselineSchema.Msg.Schema, finalNewTenantSchema.Msg.Schema)
	a.Contains(finalNewTenantSchema.Msg.Schema, "author") // New column from pending change

	// Verify: All tasks are DONE.
	completedRollout, err := ctl.rolloutServiceClient.GetRollout(ctx, connect.NewRequest(&v1pb.GetRolloutRequest{
		Name: pendingRollout.Msg.Name,
	}))
	a.NoError(err)
	for _, stage := range completedRollout.Msg.Stages {
		for _, task := range stage.Tasks {
			a.Equal(v1pb.Task_DONE, task.Status)
		}
	}
}
