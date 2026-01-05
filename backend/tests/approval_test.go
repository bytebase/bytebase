package tests

import (
	"context"
	"fmt"
	"strings"
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
	err = ctl.createDatabase(ctx, ctl.project, instanceResp.Msg, nil, dbName, "")
	a.NoError(err)

	// Create approval rule directly via WorkspaceApprovalSetting
	// This tests the new direct approval rule API (without risk layer)
	_, err = ctl.settingServiceClient.UpdateSetting(ctx, connect.NewRequest(&v1pb.UpdateSettingRequest{
		AllowMissing: true,
		Setting: &v1pb.Setting{
			Name: "settings/WORKSPACE_APPROVAL",
			Value: &v1pb.SettingValue{
				Value: &v1pb.SettingValue_WorkspaceApproval{
					WorkspaceApproval: &v1pb.WorkspaceApprovalSetting{
						Rules: []*v1pb.WorkspaceApprovalSetting_Rule{
							{
								Source: v1pb.WorkspaceApprovalSetting_Rule_CHANGE_DATABASE,
								Condition: &expr.Expr{
									Expression: `resource.db_engine == "SQLITE"`, // Use db_engine to test CEL with variables
								},
								Template: &v1pb.ApprovalTemplate{
									Title:       "Prod Change Database Approval",
									Description: "Requires workspace owner approval for database changes in prod",
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

	// Verify that the approval template was correctly assigned
	// Note: The status may be SKIPPED/APPROVED if the user is a workspace owner
	// who can self-approve, but the template should still be assigned
	a.NotNil(issue.ApprovalTemplate, "Approval template should be assigned")
	a.Equal("Prod Change Database Approval", issue.GetApprovalTemplate().GetTitle(), "Correct approval template should be applied")
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
	err = ctl.createDatabase(ctx, ctl.project, instanceResp.Msg, nil, dbName, "")
	a.NoError(err)

	// Create multiple approval rules - first one should match
	// Rule 1: Specific condition (should match first)
	// Rule 2: Catch-all condition (would also match, but shouldn't be used)
	_, err = ctl.settingServiceClient.UpdateSetting(ctx, connect.NewRequest(&v1pb.UpdateSettingRequest{
		AllowMissing: true,
		Setting: &v1pb.Setting{
			Name: "settings/WORKSPACE_APPROVAL",
			Value: &v1pb.SettingValue{
				Value: &v1pb.SettingValue_WorkspaceApproval{
					WorkspaceApproval: &v1pb.WorkspaceApprovalSetting{
						Rules: []*v1pb.WorkspaceApprovalSetting_Rule{
							{
								// Rule 1: Specific - prod environment
								Source: v1pb.WorkspaceApprovalSetting_Rule_CHANGE_DATABASE,
								Condition: &expr.Expr{
									Expression: `resource.environment_id == "prod"`,
								},
								Template: &v1pb.ApprovalTemplate{
									Title: "Prod Change Database - First Rule",
									Flow: &v1pb.ApprovalFlow{
										Roles: []string{"roles/workspaceOwner"},
									},
								},
							},
							{
								// Rule 2: Catch-all - matches everything
								Source: v1pb.WorkspaceApprovalSetting_Rule_CHANGE_DATABASE,
								Condition: &expr.Expr{
									Expression: `true`,
								},
								Template: &v1pb.ApprovalTemplate{
									Title: "Catch-all Change Database - Second Rule",
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

	// Verify the first rule was applied (not the catch-all)
	a.Equal("Prod Change Database - First Rule", issue.GetApprovalTemplate().GetTitle(), "First matching rule should be applied")
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
	err = ctl.createDatabase(ctx, ctl.project, instanceResp.Msg, nil, dbName, "")
	a.NoError(err)

	// Create approval rule only for prod environment
	_, err = ctl.settingServiceClient.UpdateSetting(ctx, connect.NewRequest(&v1pb.UpdateSettingRequest{
		AllowMissing: true,
		Setting: &v1pb.Setting{
			Name: "settings/WORKSPACE_APPROVAL",
			Value: &v1pb.SettingValue{
				Value: &v1pb.SettingValue_WorkspaceApproval{
					WorkspaceApproval: &v1pb.WorkspaceApprovalSetting{
						Rules: []*v1pb.WorkspaceApprovalSetting_Rule{
							{
								Source: v1pb.WorkspaceApprovalSetting_Rule_CHANGE_DATABASE,
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

	// Since no rule matches test environment, no approval template should be assigned
	// and the issue should be auto-approved (or skipped)
	a.Nil(issue.ApprovalTemplate, "No approval template should be assigned when no rule matches")
	// Status should be APPROVED or SKIPPED (auto-approved)
	a.True(issue.ApprovalStatus == v1pb.Issue_APPROVED || issue.ApprovalStatus == v1pb.Issue_SKIPPED,
		"Issue should be auto-approved when no rule matches, got status: %v", issue.ApprovalStatus)
}

// TestFallbackRuleValidation tests that SOURCE_UNSPECIFIED rules
// can only use resource.project_id in conditions.
func TestFallbackRuleValidation(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}

	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	// Test 1: Valid fallback rule with project_id should succeed
	_, err = ctl.settingServiceClient.UpdateSetting(ctx, connect.NewRequest(&v1pb.UpdateSettingRequest{
		AllowMissing: true,
		Setting: &v1pb.Setting{
			Name: "settings/WORKSPACE_APPROVAL",
			Value: &v1pb.SettingValue{
				Value: &v1pb.SettingValue_WorkspaceApproval{
					WorkspaceApproval: &v1pb.WorkspaceApprovalSetting{
						Rules: []*v1pb.WorkspaceApprovalSetting_Rule{
							{
								Source: v1pb.WorkspaceApprovalSetting_Rule_SOURCE_UNSPECIFIED,
								Condition: &expr.Expr{
									Expression: `resource.project_id == "proj-123"`,
								},
								Template: &v1pb.ApprovalTemplate{
									Title: "Valid Fallback Rule",
									Flow:  &v1pb.ApprovalFlow{Roles: []string{"roles/workspaceOwner"}},
								},
							},
						},
					},
				},
			},
		},
	}))
	a.NoError(err, "Fallback rule with project_id should be valid")

	// Test 2: Invalid fallback rule with environment_id should fail
	_, err = ctl.settingServiceClient.UpdateSetting(ctx, connect.NewRequest(&v1pb.UpdateSettingRequest{
		AllowMissing: true,
		Setting: &v1pb.Setting{
			Name: "settings/WORKSPACE_APPROVAL",
			Value: &v1pb.SettingValue{
				Value: &v1pb.SettingValue_WorkspaceApproval{
					WorkspaceApproval: &v1pb.WorkspaceApprovalSetting{
						Rules: []*v1pb.WorkspaceApprovalSetting_Rule{
							{
								Source: v1pb.WorkspaceApprovalSetting_Rule_SOURCE_UNSPECIFIED,
								Condition: &expr.Expr{
									Expression: `resource.environment_id == "prod"`,
								},
								Template: &v1pb.ApprovalTemplate{
									Title: "Invalid Fallback Rule",
									Flow:  &v1pb.ApprovalFlow{Roles: []string{"roles/workspaceOwner"}},
								},
							},
						},
					},
				},
			},
		},
	}))
	a.Error(err, "Fallback rule with environment_id should be rejected")
	a.Contains(err.Error(), "fallback rules can only use resource.project_id")
}

// TestFallbackRuleMatchesWhenSourceSpecificDoesNot tests that
// SOURCE_UNSPECIFIED rules are used as fallback when no source-specific rules match.
func TestFallbackRuleMatchesWhenSourceSpecificDoesNot(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}

	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	// Create instance in TEST environment (not prod)
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
	err = ctl.createDatabase(ctx, ctl.project, instanceResp.Msg, nil, dbName, "")
	a.NoError(err)

	// Create rules:
	// 1. Source-specific rule for prod only (won't match test environment)
	// 2. Fallback rule for this project (should match)
	projectID := strings.TrimPrefix(ctl.project.Name, "projects/")
	_, err = ctl.settingServiceClient.UpdateSetting(ctx, connect.NewRequest(&v1pb.UpdateSettingRequest{
		AllowMissing: true,
		Setting: &v1pb.Setting{
			Name: "settings/WORKSPACE_APPROVAL",
			Value: &v1pb.SettingValue{
				Value: &v1pb.SettingValue_WorkspaceApproval{
					WorkspaceApproval: &v1pb.WorkspaceApprovalSetting{
						Rules: []*v1pb.WorkspaceApprovalSetting_Rule{
							{
								// Source-specific: only matches prod
								Source: v1pb.WorkspaceApprovalSetting_Rule_CHANGE_DATABASE,
								Condition: &expr.Expr{
									Expression: `resource.environment_id == "prod"`,
								},
								Template: &v1pb.ApprovalTemplate{
									Title: "Prod Only Rule",
									Flow:  &v1pb.ApprovalFlow{Roles: []string{"roles/workspaceOwner"}},
								},
							},
							{
								// Fallback: matches this project
								Source: v1pb.WorkspaceApprovalSetting_Rule_SOURCE_UNSPECIFIED,
								Condition: &expr.Expr{
									Expression: fmt.Sprintf(`resource.project_id == "%s"`, projectID),
								},
								Template: &v1pb.ApprovalTemplate{
									Title: "Fallback Rule",
									Flow:  &v1pb.ApprovalFlow{Roles: []string{"roles/projectOwner"}},
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
			Content: []byte("CREATE TABLE fallback_test (id INTEGER PRIMARY KEY);"),
		},
	}))
	a.NoError(err)

	// Create plan targeting test environment
	planResp, err := ctl.planServiceClient.CreatePlan(ctx, connect.NewRequest(&v1pb.CreatePlanRequest{
		Parent: ctl.project.Name,
		Plan: &v1pb.Plan{
			Title: "Test Fallback Plan",
			Specs: []*v1pb.Plan_Spec{{
				Id: uuid.NewString(),
				Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
					ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
						Targets: []string{fmt.Sprintf("%s/databases/%s", instanceResp.Msg.Name, dbName)},
						Sheet:   sheet.Msg.Name,
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
			Title:       "Test Issue for Fallback Rule",
			Type:        v1pb.Issue_DATABASE_CHANGE,
			Description: "Testing fallback rule matching",
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

	// Verify fallback rule was applied
	a.NotNil(issue)
	a.NotEqual(v1pb.Issue_CHECKING, issue.ApprovalStatus)
	a.NotNil(issue.ApprovalTemplate, "Fallback approval template should be assigned")
	a.Equal("Fallback Rule", issue.GetApprovalTemplate().GetTitle(), "Fallback rule should be applied")
}

// TestSourceSpecificRuleTakesPriorityOverFallback verifies that
// source-specific rules are always tried before fallback rules.
func TestSourceSpecificRuleTakesPriorityOverFallback(t *testing.T) {
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
	err = ctl.createDatabase(ctx, ctl.project, instanceResp.Msg, nil, dbName, "")
	a.NoError(err)

	// Create rules with fallback FIRST in the list, but source-specific should still win
	projectID := strings.TrimPrefix(ctl.project.Name, "projects/")
	_, err = ctl.settingServiceClient.UpdateSetting(ctx, connect.NewRequest(&v1pb.UpdateSettingRequest{
		AllowMissing: true,
		Setting: &v1pb.Setting{
			Name: "settings/WORKSPACE_APPROVAL",
			Value: &v1pb.SettingValue{
				Value: &v1pb.SettingValue_WorkspaceApproval{
					WorkspaceApproval: &v1pb.WorkspaceApprovalSetting{
						Rules: []*v1pb.WorkspaceApprovalSetting_Rule{
							{
								// Fallback rule FIRST in list
								Source: v1pb.WorkspaceApprovalSetting_Rule_SOURCE_UNSPECIFIED,
								Condition: &expr.Expr{
									Expression: fmt.Sprintf(`resource.project_id == "%s"`, projectID),
								},
								Template: &v1pb.ApprovalTemplate{
									Title: "Fallback Rule (should NOT be used)",
									Flow:  &v1pb.ApprovalFlow{Roles: []string{"roles/projectOwner"}},
								},
							},
							{
								// Source-specific rule SECOND in list
								Source: v1pb.WorkspaceApprovalSetting_Rule_CHANGE_DATABASE,
								Condition: &expr.Expr{
									Expression: `resource.environment_id == "prod"`,
								},
								Template: &v1pb.ApprovalTemplate{
									Title: "Source Specific Rule (should be used)",
									Flow:  &v1pb.ApprovalFlow{Roles: []string{"roles/workspaceOwner"}},
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
			Content: []byte("CREATE TABLE priority_test (id INTEGER PRIMARY KEY);"),
		},
	}))
	a.NoError(err)

	// Create plan
	planResp, err := ctl.planServiceClient.CreatePlan(ctx, connect.NewRequest(&v1pb.CreatePlanRequest{
		Parent: ctl.project.Name,
		Plan: &v1pb.Plan{
			Title: "Test Priority Plan",
			Specs: []*v1pb.Plan_Spec{{
				Id: uuid.NewString(),
				Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
					ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
						Targets: []string{fmt.Sprintf("%s/databases/%s", instanceResp.Msg.Name, dbName)},
						Sheet:   sheet.Msg.Name,
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
			Title:       "Test Issue for Priority",
			Type:        v1pb.Issue_DATABASE_CHANGE,
			Description: "Testing source-specific priority over fallback",
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

	// Verify source-specific rule was applied, NOT the fallback
	a.NotNil(issue)
	a.NotEqual(v1pb.Issue_CHECKING, issue.ApprovalStatus)
	a.NotNil(issue.ApprovalTemplate)
	a.Equal("Source Specific Rule (should be used)", issue.GetApprovalTemplate().GetTitle(),
		"Source-specific rule should take priority over fallback even when fallback is first in list")
}
