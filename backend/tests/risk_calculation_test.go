package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/type/expr"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// TestApprovalRuleMatching tests that approval rules with CEL expressions
// are correctly matched and applied to issues.
// Note: As of 3.13, the risk intermediate layer is removed. Approval rules
// now directly evaluate CEL expressions instead of calculating risk levels.
func TestApprovalRuleMatching(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}

	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	// Create instance
	instanceDir := t.TempDir()
	instanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("inst"),
		Instance: &v1pb.Instance{
			Title:       "Prod Instance",
			Engine:      v1pb.Engine_SQLITE,
			Environment: stringPtr("environments/prod"),
			Activation:  true,
			DataSources: []*v1pb.DataSource{{
				Type: v1pb.DataSourceType_ADMIN,
				Host: instanceDir,
				Id:   "admin",
			}},
		},
	}))
	a.NoError(err)

	// Create database
	dbName := generateRandomString("db")
	err = ctl.createDatabaseV2(ctx, ctl.project, instanceResp.Msg, nil, dbName, "")
	a.NoError(err)

	// Note: In 3.13+, risk rules are converted to approval rules via migration.
	// This test creates a risk rule to verify that the risk API still works
	// and that the approval flow completes without errors.
	// The actual approval template matching now evaluates CEL expressions directly
	// without calculating an intermediate risk level.
	_, err = ctl.riskServiceClient.CreateRisk(ctx, connect.NewRequest(&v1pb.CreateRiskRequest{
		Risk: &v1pb.Risk{
			Title:  "Production DDL Risk",
			Source: v1pb.Risk_DDL,
			Level:  v1pb.RiskLevel_HIGH,
			Active: true,
			Condition: &expr.Expr{
				Expression: `resource.environment_id == "prod"`,
			},
		},
	}))
	a.NoError(err)

	// Create sheet with DDL statement
	sheet, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
		Parent: ctl.project.Name,
		Sheet: &v1pb.Sheet{
			Title:   "Test DDL Sheet",
			Content: []byte("CREATE TABLE test_table (id INTEGER PRIMARY KEY, name TEXT);"),
		},
	}))
	a.NoError(err)

	planResp, err := ctl.planServiceClient.CreatePlan(ctx, connect.NewRequest(&v1pb.CreatePlanRequest{
		Parent: ctl.project.Name,
		Plan: &v1pb.Plan{
			Title: "Test DDL Plan",
			Specs: []*v1pb.Plan_Spec{{
				Id: uuid.NewString(),
				Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
					ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
						Targets: []string{fmt.Sprintf("%s/databases/%s", instanceResp.Msg.Name, dbName)},
						Sheet:   sheet.Msg.Name,
						Type:    v1pb.DatabaseChangeType_MIGRATE,
					},
				},
			}},
		},
	}))
	a.NoError(err, "Plan creation should succeed")

	// Verify plan was created successfully
	a.NotNil(planResp.Msg)
	a.Equal("Test DDL Plan", planResp.Msg.Title)

	// Create issue with the plan - this triggers approval rule matching
	issueResp, err := ctl.issueServiceClient.CreateIssue(ctx, connect.NewRequest(&v1pb.CreateIssueRequest{
		Parent: ctl.project.Name,
		Issue: &v1pb.Issue{
			Title:       "Test Issue for Approval Rule Matching",
			Type:        v1pb.Issue_DATABASE_CHANGE,
			Description: "Testing approval rule matching",
			Plan:        planResp.Msg.Name,
		},
	}))
	a.NoError(err, "Issue creation should succeed")

	// Check issue status 5 times with 3 second intervals
	// Wait for approval finding to complete
	var issue *v1pb.Issue
	for i := 0; i < 5; i++ {
		if i > 0 {
			time.Sleep(3 * time.Second) // Wait 3 seconds between checks
		}

		issueGetResp, err := ctl.issueServiceClient.GetIssue(ctx, connect.NewRequest(&v1pb.GetIssueRequest{
			Name: issueResp.Msg.Name,
		}))
		a.NoError(err)
		issue = issueGetResp.Msg

		// Log the current state for debugging
		t.Logf("Check %d: ApprovalStatus=%v, ApprovalStatusError='%s'",
			i+1, issue.ApprovalStatus, issue.ApprovalStatusError)

		// Check if approval finding is complete
		if issue.ApprovalStatus != v1pb.Issue_CHECKING {
			break
		}
	}

	// Verify that the approval finding process completed successfully
	a.NotNil(issue, "Issue should be retrievable")
	a.NotEqual(v1pb.Issue_CHECKING, issue.ApprovalStatus, "Approval finding should complete")
	a.NotEqual(v1pb.Issue_ERROR, issue.ApprovalStatus, "Approval finding should not have errors")
}

// TestApprovalRuleMatchingWithInvalidSQL tests that approval rule matching
// works even when SQL is invalid and summary reports cannot be generated.
// This is a regression test for PR #16793.
func TestApprovalRuleMatchingWithInvalidSQL(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}

	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	// Create instance
	instanceDir := t.TempDir()
	instanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("inst"),
		Instance: &v1pb.Instance{
			Title:       "Prod Instance",
			Engine:      v1pb.Engine_SQLITE,
			Environment: stringPtr("environments/prod"),
			Activation:  true,
			DataSources: []*v1pb.DataSource{{
				Type: v1pb.DataSourceType_ADMIN,
				Host: instanceDir,
				Id:   "admin",
			}},
		},
	}))
	a.NoError(err)

	// Create database
	dbName := generateRandomString("db")
	err = ctl.createDatabaseV2(ctx, ctl.project, instanceResp.Msg, nil, dbName, "")
	a.NoError(err)

	// Create a risk rule that evaluates based on environment context
	// This should work without summary reports even for invalid SQL
	_, err = ctl.riskServiceClient.CreateRisk(ctx, connect.NewRequest(&v1pb.CreateRiskRequest{
		Risk: &v1pb.Risk{
			Title:  "Production DDL Risk",
			Source: v1pb.Risk_DDL,
			Level:  v1pb.RiskLevel_HIGH,
			Active: true,
			Condition: &expr.Expr{
				Expression: `resource.environment_id == "prod"`,
			},
		},
	}))
	a.NoError(err)

	// Create sheet with invalid SQL statement "hh"
	// This will definitely fail summary report generation
	sheet, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
		Parent: ctl.project.Name,
		Sheet: &v1pb.Sheet{
			Title:   "Invalid SQL Sheet",
			Content: []byte("hh"),
		},
	}))
	a.NoError(err)

	// Create plan with invalid SQL - this should succeed despite summary report failure
	planResp, err := ctl.planServiceClient.CreatePlan(ctx, connect.NewRequest(&v1pb.CreatePlanRequest{
		Parent: ctl.project.Name,
		Plan: &v1pb.Plan{
			Title: "Invalid SQL Plan",
			Specs: []*v1pb.Plan_Spec{{
				Id: uuid.NewString(),
				Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
					ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
						Targets: []string{fmt.Sprintf("%s/databases/%s", instanceResp.Msg.Name, dbName)},
						Sheet:   sheet.Msg.Name,
						Type:    v1pb.DatabaseChangeType_MIGRATE,
					},
				},
			}},
		},
	}))
	a.NoError(err, "Plan creation should succeed even with invalid SQL")

	// Create issue with the plan - this triggers approval rule matching
	issueResp, err := ctl.issueServiceClient.CreateIssue(ctx, connect.NewRequest(&v1pb.CreateIssueRequest{
		Parent: ctl.project.Name,
		Issue: &v1pb.Issue{
			Title:       "Test Issue with Invalid SQL",
			Type:        v1pb.Issue_DATABASE_CHANGE,
			Description: "Testing approval rule matching with invalid SQL",
			Plan:        planResp.Msg.Name,
		},
	}))
	a.NoError(err, "Issue creation should succeed even with invalid SQL in plan")

	// Check issue status 5 times with 3 second intervals
	// Wait for approval finding to complete
	var issue *v1pb.Issue
	for i := 0; i < 5; i++ {
		if i > 0 {
			time.Sleep(3 * time.Second) // Wait 3 seconds between checks
		}

		issueGetResp, err := ctl.issueServiceClient.GetIssue(ctx, connect.NewRequest(&v1pb.GetIssueRequest{
			Name: issueResp.Msg.Name,
		}))
		a.NoError(err)
		issue = issueGetResp.Msg

		// Log the current state for debugging
		t.Logf("Check %d: ApprovalStatus=%v, ApprovalStatusError='%s'",
			i+1, issue.ApprovalStatus, issue.ApprovalStatusError)

		// Check if approval finding is complete
		if issue.ApprovalStatus != v1pb.Issue_CHECKING {
			break
		}
	}

	// Verify that the approval finding process completed successfully despite invalid SQL
	a.NotNil(issue, "Issue should be retrievable even with invalid SQL")
	a.NotEqual(v1pb.Issue_CHECKING, issue.ApprovalStatus, "Approval finding should complete even with invalid SQL")
	a.NotEqual(v1pb.Issue_ERROR, issue.ApprovalStatus, "Approval finding should not have errors despite invalid SQL")
}
