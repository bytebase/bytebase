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
// KEY CONCEPTS:
//   - Baseline Database: The "source of truth" for tenant schemas.
//   - Database Group: Logical grouping using CEL expressions to match databases.
//   - Idempotent CreateRollout: Re-evaluates database group, adds new databases
//     without duplicating existing tasks.
//
// WORKFLOW (Steps 1-5 in test):
//  1. Get baseline schema - DatabaseService.GetDatabaseSchema()
//  2. Create new tenant database
//  3. Initialize tenant with baseline schema - Plan + Issue + Rollout
//  4. Find pending rollouts - ListPlans(filter="has_rollout == true")
//  5. Add tenant to pending rollouts - CreateRollout() (idempotent)
//  6. [OPTIONAL] Execute pending tasks - BatchRunTasks() or UI
func TestTenantBackfill(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	// ==================== TEST SETUP ====================
	// Create baseline database, database group, and a pending rollout.

	projectID := generateRandomString("project")
	project, err := ctl.projectServiceClient.CreateProject(ctx, connect.NewRequest(&v1pb.CreateProjectRequest{
		Project: &v1pb.Project{
			Name:              fmt.Sprintf("projects/%s", projectID),
			Title:             projectID,
			AllowSelfApproval: true,
		},
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

	// Create baseline database (source of truth for tenant schemas).
	baselineDBName := "tenant_baseline"
	err = ctl.createDatabase(ctx, project.Msg, prodInstance.Msg, nil, baselineDBName, "")
	a.NoError(err)

	// Create database group matching all databases.
	databaseGroup, err := ctl.databaseGroupServiceClient.CreateDatabaseGroup(ctx, connect.NewRequest(&v1pb.CreateDatabaseGroupRequest{
		Parent:          project.Msg.Name,
		DatabaseGroupId: "tenants",
		DatabaseGroup: &v1pb.DatabaseGroup{
			Title:        "All Tenants",
			DatabaseExpr: &expr.Expr{Expression: "true"},
		},
	}))
	a.NoError(err)

	// Execute first change on baseline (CREATE TABLE).
	sheet1, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
		Parent: project.Msg.Name,
		Sheet:  &v1pb.Sheet{Content: []byte(migrationStatement1)},
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

	baselineSchema, err := ctl.databaseServiceClient.GetDatabaseSchema(ctx, connect.NewRequest(&v1pb.GetDatabaseSchemaRequest{
		Name: fmt.Sprintf("%s/databases/%s/schema", prodInstance.Msg.Name, baselineDBName),
	}))
	a.NoError(err)
	a.Equal(wantBookSchema, baselineSchema.Msg.Schema)

	// Create pending change (ALTER TABLE) - approved but not executed.
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

	a.Len(pendingRollout.Msg.Stages, 1)
	a.Len(pendingRollout.Msg.Stages[0].Tasks, 1)
	a.Equal(v1pb.Task_NOT_STARTED, pendingRollout.Msg.Stages[0].Tasks[0].Status)

	// ==================== TENANT ONBOARDING WORKFLOW ====================

	// Step 1: Get baseline's current schema.
	baselineCurrentSchema, err := ctl.databaseServiceClient.GetDatabaseSchema(ctx, connect.NewRequest(&v1pb.GetDatabaseSchemaRequest{
		Name: fmt.Sprintf("%s/databases/%s/schema", prodInstance.Msg.Name, baselineDBName),
	}))
	a.NoError(err)

	// Step 2: Create new tenant database.
	newTenantDBName := "tenant_new"
	err = ctl.createDatabase(ctx, project.Msg, prodInstance.Msg, nil, newTenantDBName, "")
	a.NoError(err)

	// Step 3: Initialize new tenant with baseline schema.
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
						Targets: []string{newTenantDBPath},
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

	newTenantSchema, err := ctl.databaseServiceClient.GetDatabaseSchema(ctx, connect.NewRequest(&v1pb.GetDatabaseSchemaRequest{
		Name: fmt.Sprintf("%s/databases/%s/schema", prodInstance.Msg.Name, newTenantDBName),
	}))
	a.NoError(err)
	a.Equal(baselineCurrentSchema.Msg.Schema, newTenantSchema.Msg.Schema)

	// Step 4: Find pending rollouts for baseline.
	plans, err := ctl.planServiceClient.ListPlans(ctx, connect.NewRequest(&v1pb.ListPlansRequest{
		Parent: project.Msg.Name,
		Filter: "has_rollout == true",
	}))
	a.NoError(err)

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

	a.Len(pendingPlansForBaseline, 1)
	a.Equal(pendingPlan.Msg.Name, pendingPlansForBaseline[0].Name)

	// Step 5: Add new tenant to pending rollouts (idempotent CreateRollout).
	// Re-evaluates database group, adds new tenant without duplicating existing tasks.
	for _, p := range pendingPlansForBaseline {
		updatedRollout, err := ctl.rolloutServiceClient.CreateRollout(ctx, connect.NewRequest(&v1pb.CreateRolloutRequest{
			Parent: p.Name,
		}))
		a.NoError(err)

		a.Len(updatedRollout.Msg.Stages, 1)
		a.Len(updatedRollout.Msg.Stages[0].Tasks, 2) // baseline + new tenant

		for _, task := range updatedRollout.Msg.Stages[0].Tasks {
			a.Equal(v1pb.Task_NOT_STARTED, task.Status)
		}
	}

	// Verify: Both databases are in pending rollout.
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
	a.Equal(2, notStartedCount)

	// Step 6 [OPTIONAL]: Execute pending tasks.
	err = ctl.waitRollout(ctx, pendingIssue.Msg.Name, pendingRollout.Msg.Name)
	a.NoError(err)

	finalBaselineSchema, err := ctl.databaseServiceClient.GetDatabaseSchema(ctx, connect.NewRequest(&v1pb.GetDatabaseSchemaRequest{
		Name: fmt.Sprintf("%s/databases/%s/schema", prodInstance.Msg.Name, baselineDBName),
	}))
	a.NoError(err)

	finalNewTenantSchema, err := ctl.databaseServiceClient.GetDatabaseSchema(ctx, connect.NewRequest(&v1pb.GetDatabaseSchemaRequest{
		Name: fmt.Sprintf("%s/databases/%s/schema", prodInstance.Msg.Name, newTenantDBName),
	}))
	a.NoError(err)

	a.Equal(finalBaselineSchema.Msg.Schema, finalNewTenantSchema.Msg.Schema)
	a.Contains(finalNewTenantSchema.Msg.Schema, "author")

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
