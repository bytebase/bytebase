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

// TestDirectApprovalRuleMatching tests the new direct approval rule API
// where rules are created via WorkspaceApprovalSetting without the risk layer.
func TestDirectApprovalRuleMatching(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}

	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	// Create instance in prod environment
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

	// Create approval rule directly via WorkspaceApprovalSetting
	// This tests the new direct approval rule API (without risk layer)
	_, err = ctl.settingServiceClient.UpdateSetting(ctx, connect.NewRequest(&v1pb.UpdateSettingRequest{
		AllowMissing: true,
		Setting: &v1pb.Setting{
			Name: "settings/WORKSPACE_APPROVAL",
			Value: &v1pb.Value{
				Value: &v1pb.Value_WorkspaceApprovalSettingValue{
					WorkspaceApprovalSettingValue: &v1pb.WorkspaceApprovalSetting{
						Rules: []*v1pb.WorkspaceApprovalSetting_Rule{
							{
								Source: v1pb.WorkspaceApprovalSetting_Rule_DDL,
								Condition: &expr.Expr{
									Expression: `resource.db_engine == "SQLITE"`, // Use db_engine to test CEL with variables
								},
								Template: &v1pb.ApprovalTemplate{
									Title:       "Prod DDL Approval",
									Description: "Requires workspace owner approval for DDL in prod",
									Flow: &v1pb.ApprovalFlow{
										Roles: []string{"roles/workspaceOwner"},
									},
								},
							},
						},
					},
				},
			},
		},
	}))
	a.NoError(err, "Setting update should succeed")

	// Create sheet with DDL statement
	sheet, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
		Parent: ctl.project.Name,
		Sheet: &v1pb.Sheet{
			Title:   "Test DDL Sheet",
			Content: []byte("CREATE TABLE approval_test (id INTEGER PRIMARY KEY);"),
		},
	}))
	a.NoError(err)

	// Create plan
	planResp, err := ctl.planServiceClient.CreatePlan(ctx, connect.NewRequest(&v1pb.CreatePlanRequest{
		Parent: ctl.project.Name,
		Plan: &v1pb.Plan{
			Title: "Test DDL Plan for Direct Approval",
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
	a.NoError(err)

	// Create issue - this triggers approval rule matching
	issueResp, err := ctl.issueServiceClient.CreateIssue(ctx, connect.NewRequest(&v1pb.CreateIssueRequest{
		Parent: ctl.project.Name,
		Issue: &v1pb.Issue{
			Title:       "Test Issue for Direct Approval Rule",
			Type:        v1pb.Issue_DATABASE_CHANGE,
			Description: "Testing direct approval rule matching",
			Plan:        planResp.Msg.Name,
		},
	}))
	a.NoError(err)

	// Wait for approval finding to complete
	var issue *v1pb.Issue
	for i := 0; i < 5; i++ {
		if i > 0 {
			time.Sleep(3 * time.Second)
		}

		issueGetResp, err := ctl.issueServiceClient.GetIssue(ctx, connect.NewRequest(&v1pb.GetIssueRequest{
			Name: issueResp.Msg.Name,
		}))
		a.NoError(err)
		issue = issueGetResp.Msg

		if issue.ApprovalStatus != v1pb.Issue_CHECKING {
			break
		}
	}

	// Verify approval finding completed successfully
	a.NotNil(issue)
	a.NotEqual(v1pb.Issue_CHECKING, issue.ApprovalStatus, "Approval finding should complete")
	a.NotEqual(v1pb.Issue_ERROR, issue.ApprovalStatus, "Approval finding should not have errors")

	// Verify that the approval template was correctly assigned
	// Note: The status may be SKIPPED/APPROVED if the user is a workspace owner
	// who can self-approve, but the template should still be assigned
	a.NotNil(issue.ApprovalTemplate, "Approval template should be assigned")
	a.Equal("Prod DDL Approval", issue.GetApprovalTemplate().GetTitle(), "Correct approval template should be applied")
}

// TestApprovalRuleFirstMatchWins tests that the first matching approval rule is applied
// when multiple rules could potentially match.
func TestApprovalRuleFirstMatchWins(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}

	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	// Create instance in prod environment
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

	// Create multiple approval rules - first one should match
	// Rule 1: Specific condition (should match first)
	// Rule 2: Catch-all condition (would also match, but shouldn't be used)
	_, err = ctl.settingServiceClient.UpdateSetting(ctx, connect.NewRequest(&v1pb.UpdateSettingRequest{
		AllowMissing: true,
		Setting: &v1pb.Setting{
			Name: "settings/WORKSPACE_APPROVAL",
			Value: &v1pb.Value{
				Value: &v1pb.Value_WorkspaceApprovalSettingValue{
					WorkspaceApprovalSettingValue: &v1pb.WorkspaceApprovalSetting{
						Rules: []*v1pb.WorkspaceApprovalSetting_Rule{
							{
								// Rule 1: Specific - prod environment DDL
								Source: v1pb.WorkspaceApprovalSetting_Rule_DDL,
								Condition: &expr.Expr{
									Expression: `resource.environment_id == "prod"`,
								},
								Template: &v1pb.ApprovalTemplate{
									Title: "Prod DDL - First Rule",
									Flow: &v1pb.ApprovalFlow{
										Roles: []string{"roles/workspaceOwner"},
									},
								},
							},
							{
								// Rule 2: Catch-all - matches everything
								Source: v1pb.WorkspaceApprovalSetting_Rule_DDL,
								Condition: &expr.Expr{
									Expression: `true`,
								},
								Template: &v1pb.ApprovalTemplate{
									Title: "Catch-all DDL - Second Rule",
									Flow: &v1pb.ApprovalFlow{
										Roles: []string{"roles/projectOwner", "roles/workspaceOwner"},
									},
								},
							},
						},
					},
				},
			},
		},
	}))
	a.NoError(err)

	// Create sheet with DDL statement
	sheet, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
		Parent: ctl.project.Name,
		Sheet: &v1pb.Sheet{
			Title:   "Test DDL Sheet",
			Content: []byte("CREATE TABLE first_match_test (id INTEGER PRIMARY KEY);"),
		},
	}))
	a.NoError(err)

	// Create plan
	planResp, err := ctl.planServiceClient.CreatePlan(ctx, connect.NewRequest(&v1pb.CreatePlanRequest{
		Parent: ctl.project.Name,
		Plan: &v1pb.Plan{
			Title: "Test First Match Plan",
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
	a.NoError(err)

	// Create issue
	issueResp, err := ctl.issueServiceClient.CreateIssue(ctx, connect.NewRequest(&v1pb.CreateIssueRequest{
		Parent: ctl.project.Name,
		Issue: &v1pb.Issue{
			Title:       "Test Issue for First Match Wins",
			Type:        v1pb.Issue_DATABASE_CHANGE,
			Description: "Testing first match wins behavior",
			Plan:        planResp.Msg.Name,
		},
	}))
	a.NoError(err)

	// Wait for approval finding to complete
	var issue *v1pb.Issue
	for i := 0; i < 5; i++ {
		if i > 0 {
			time.Sleep(3 * time.Second)
		}

		issueGetResp, err := ctl.issueServiceClient.GetIssue(ctx, connect.NewRequest(&v1pb.GetIssueRequest{
			Name: issueResp.Msg.Name,
		}))
		a.NoError(err)
		issue = issueGetResp.Msg

		if issue.ApprovalStatus != v1pb.Issue_CHECKING {
			break
		}
	}

	// Verify approval finding completed
	a.NotNil(issue)
	a.NotEqual(v1pb.Issue_CHECKING, issue.ApprovalStatus)
	a.NotEqual(v1pb.Issue_ERROR, issue.ApprovalStatus)

	// Verify the first rule was applied (not the catch-all)
	a.Equal("Prod DDL - First Rule", issue.GetApprovalTemplate().GetTitle(), "First matching rule should be applied")
}

// TestApprovalRuleNoMatch tests that issues without matching approval rules
// are automatically approved (no approval required).
func TestApprovalRuleNoMatch(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}

	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	// Create instance in test environment (not prod)
	instanceDir := t.TempDir()
	instanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("inst"),
		Instance: &v1pb.Instance{
			Title:       "Test Instance",
			Engine:      v1pb.Engine_SQLITE,
			Environment: stringPtr("environments/test"),
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

	// Create approval rule only for prod environment
	_, err = ctl.settingServiceClient.UpdateSetting(ctx, connect.NewRequest(&v1pb.UpdateSettingRequest{
		AllowMissing: true,
		Setting: &v1pb.Setting{
			Name: "settings/WORKSPACE_APPROVAL",
			Value: &v1pb.Value{
				Value: &v1pb.Value_WorkspaceApprovalSettingValue{
					WorkspaceApprovalSettingValue: &v1pb.WorkspaceApprovalSetting{
						Rules: []*v1pb.WorkspaceApprovalSetting_Rule{
							{
								Source: v1pb.WorkspaceApprovalSetting_Rule_DDL,
								Condition: &expr.Expr{
									Expression: `resource.environment_id == "prod"`,
								},
								Template: &v1pb.ApprovalTemplate{
									Title: "Prod Only Rule",
									Flow: &v1pb.ApprovalFlow{
										Roles: []string{"roles/workspaceOwner"},
									},
								},
							},
						},
					},
				},
			},
		},
	}))
	a.NoError(err)

	// Create sheet with DDL statement
	sheet, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
		Parent: ctl.project.Name,
		Sheet: &v1pb.Sheet{
			Title:   "Test DDL Sheet",
			Content: []byte("CREATE TABLE no_match_test (id INTEGER PRIMARY KEY);"),
		},
	}))
	a.NoError(err)

	// Create plan targeting test environment (not prod)
	planResp, err := ctl.planServiceClient.CreatePlan(ctx, connect.NewRequest(&v1pb.CreatePlanRequest{
		Parent: ctl.project.Name,
		Plan: &v1pb.Plan{
			Title: "Test No Match Plan",
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
	a.NoError(err)

	// Create issue
	issueResp, err := ctl.issueServiceClient.CreateIssue(ctx, connect.NewRequest(&v1pb.CreateIssueRequest{
		Parent: ctl.project.Name,
		Issue: &v1pb.Issue{
			Title:       "Test Issue No Matching Rule",
			Type:        v1pb.Issue_DATABASE_CHANGE,
			Description: "Testing no matching approval rule",
			Plan:        planResp.Msg.Name,
		},
	}))
	a.NoError(err)

	// Wait for approval finding to complete
	var issue *v1pb.Issue
	for i := 0; i < 5; i++ {
		if i > 0 {
			time.Sleep(3 * time.Second)
		}

		issueGetResp, err := ctl.issueServiceClient.GetIssue(ctx, connect.NewRequest(&v1pb.GetIssueRequest{
			Name: issueResp.Msg.Name,
		}))
		a.NoError(err)
		issue = issueGetResp.Msg

		if issue.ApprovalStatus != v1pb.Issue_CHECKING {
			break
		}
	}

	// Verify approval finding completed
	a.NotNil(issue)
	a.NotEqual(v1pb.Issue_CHECKING, issue.ApprovalStatus)
	a.NotEqual(v1pb.Issue_ERROR, issue.ApprovalStatus)

	// Since no rule matches test environment, no approval template should be assigned
	// and the issue should be auto-approved (or skipped)
	a.Nil(issue.ApprovalTemplate, "No approval template should be assigned when no rule matches")
	// Status should be APPROVED or SKIPPED (auto-approved)
	a.True(issue.ApprovalStatus == v1pb.Issue_APPROVED || issue.ApprovalStatus == v1pb.Issue_SKIPPED,
		"Issue should be auto-approved when no rule matches, got status: %v", issue.ApprovalStatus)
}
