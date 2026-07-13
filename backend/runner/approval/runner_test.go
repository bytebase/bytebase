package approval

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/type/expr"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/component/bus"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/runner/plancheck"
	"github.com/bytebase/bytebase/backend/store"
)

func TestRiskLevelToString(t *testing.T) {
	tests := []struct {
		name  string
		level storepb.RiskLevel
		want  string
	}{
		{
			name:  "LOW",
			level: storepb.RiskLevel_LOW,
			want:  "LOW",
		},
		{
			name:  "MODERATE",
			level: storepb.RiskLevel_MODERATE,
			want:  "MODERATE",
		},
		{
			name:  "HIGH",
			level: storepb.RiskLevel_HIGH,
			want:  "HIGH",
		},
		{
			name:  "UNSPECIFIED defaults to LOW",
			level: storepb.RiskLevel_RISK_LEVEL_UNSPECIFIED,
			want:  "LOW",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := require.New(t)
			got := riskLevelToString(tt.level)
			a.Equal(tt.want, got)
		})
	}
}

func TestInjectRiskLevelIntoCELVars(t *testing.T) {
	tests := []struct {
		name       string
		celVars    []map[string]any
		riskLevel  storepb.RiskLevel
		wantValue  string
		wantLength int
	}{
		{
			name: "inject HIGH into single map",
			celVars: []map[string]any{
				{"resource.environment_id": "prod"},
			},
			riskLevel:  storepb.RiskLevel_HIGH,
			wantValue:  "HIGH",
			wantLength: 1,
		},
		{
			name: "inject MODERATE into multiple maps",
			celVars: []map[string]any{
				{"resource.environment_id": "prod"},
				{"resource.environment_id": "test"},
			},
			riskLevel:  storepb.RiskLevel_MODERATE,
			wantValue:  "MODERATE",
			wantLength: 2,
		},
		{
			name:       "inject into empty list",
			celVars:    []map[string]any{},
			riskLevel:  storepb.RiskLevel_LOW,
			wantValue:  "LOW",
			wantLength: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := require.New(t)

			injectRiskLevelIntoCELVars(tt.celVars, tt.riskLevel)

			a.Len(tt.celVars, tt.wantLength)
			for _, vars := range tt.celVars {
				riskLevel, ok := vars[common.CELAttributeRiskLevel]
				a.True(ok, "risk.level should be present")
				a.Equal(tt.wantValue, riskLevel)
			}
		})
	}
}

func TestCalculateRiskLevelFromCELVars(t *testing.T) {
	tests := []struct {
		name    string
		celVars []map[string]any
		want    storepb.RiskLevel
	}{
		{
			name:    "nil returns LOW",
			celVars: nil,
			want:    storepb.RiskLevel_LOW,
		},
		{
			name:    "empty returns LOW",
			celVars: []map[string]any{},
			want:    storepb.RiskLevel_LOW,
		},
		{
			name: "SELECT returns LOW",
			celVars: []map[string]any{
				{common.CELAttributeStatementSQLType: "SELECT"},
			},
			want: storepb.RiskLevel_LOW,
		},
		{
			name: "UPDATE returns MODERATE",
			celVars: []map[string]any{
				{common.CELAttributeStatementSQLType: "UPDATE"},
			},
			want: storepb.RiskLevel_MODERATE,
		},
		{
			name: "DELETE returns MODERATE",
			celVars: []map[string]any{
				{common.CELAttributeStatementSQLType: "DELETE"},
			},
			want: storepb.RiskLevel_MODERATE,
		},
		{
			name: "DROP_TABLE returns HIGH",
			celVars: []map[string]any{
				{common.CELAttributeStatementSQLType: "DROP_TABLE"},
			},
			want: storepb.RiskLevel_HIGH,
		},
		{
			name: "TRUNCATE returns HIGH",
			celVars: []map[string]any{
				{common.CELAttributeStatementSQLType: "TRUNCATE"},
			},
			want: storepb.RiskLevel_HIGH,
		},
		{
			name: "mixed SELECT and DROP_TABLE returns HIGH (highest)",
			celVars: []map[string]any{
				{common.CELAttributeStatementSQLType: "SELECT"},
				{common.CELAttributeStatementSQLType: "DROP_TABLE"},
			},
			want: storepb.RiskLevel_HIGH,
		},
		{
			name: "mixed UPDATE and DELETE returns MODERATE",
			celVars: []map[string]any{
				{common.CELAttributeStatementSQLType: "UPDATE"},
				{common.CELAttributeStatementSQLType: "DELETE"},
			},
			want: storepb.RiskLevel_MODERATE,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := require.New(t)
			got := calculateRiskLevelFromCELVars(tt.celVars)
			a.Equal(tt.want, got)
		})
	}
}

func TestApprovalTemplateMatchesUnspecifiedStatementSQLType(t *testing.T) {
	a := require.New(t)

	approvalTemplate, err := getApprovalTemplate(&storepb.WorkspaceApprovalSetting{
		Rules: []*storepb.WorkspaceApprovalSetting_Rule{
			{
				Source:    storepb.WorkspaceApprovalSetting_Rule_CHANGE_DATABASE,
				Condition: &expr.Expr{Expression: `statement.sql_type == "STATEMENT_TYPE_UNSPECIFIED"`},
				Template:  &storepb.ApprovalTemplate{Title: "Unspecified SQL type rule"},
			},
		},
	}, storepb.WorkspaceApprovalSetting_Rule_CHANGE_DATABASE, expandCELVars(map[string]any{
		common.CELAttributeResourceProjectID: "project",
	}, []storepb.StatementType{storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED}, nil))
	a.NoError(err)
	a.NotNil(approvalTemplate)
	a.Equal("Unspecified SQL type rule", approvalTemplate.Title)
}

func TestApprovalTemplateMatchesIssueLabels(t *testing.T) {
	a := require.New(t)

	celVars := map[string]any{
		common.CELAttributeResourceProjectID: "project",
		common.CELAttributeIssueLabels:       []string{"prod", "security"},
	}
	injectRiskLevelIntoCELVars([]map[string]any{celVars}, storepb.RiskLevel_HIGH)

	approvalTemplate, err := getApprovalTemplate(&storepb.WorkspaceApprovalSetting{
		Rules: []*storepb.WorkspaceApprovalSetting_Rule{
			{
				Source:    storepb.WorkspaceApprovalSetting_Rule_CHANGE_DATABASE,
				Condition: &expr.Expr{Expression: `"prod" in issue.labels && risk.level == "HIGH"`},
				Template:  &storepb.ApprovalTemplate{Title: "Production label rule"},
			},
		},
	}, storepb.WorkspaceApprovalSetting_Rule_CHANGE_DATABASE, []map[string]any{celVars})
	a.NoError(err)
	a.NotNil(approvalTemplate)
	a.Equal("Production label rule", approvalTemplate.Title)
}

func TestInjectIssueLabelsIntoCELVars(t *testing.T) {
	celVarsList := []map[string]any{
		{common.CELAttributeResourceProjectID: "project-a"},
		{common.CELAttributeResourceProjectID: "project-a"},
	}

	injectIssueLabelsIntoCELVars(celVarsList, []string{" security ", "prod", "prod"})

	for _, celVars := range celVarsList {
		require.Equal(t, []string{"prod", "security"}, celVars[common.CELAttributeIssueLabels])
	}
}

func TestBuildStatementSummaryResultMapUsesSheetSHA256(t *testing.T) {
	results := []*storepb.PlanCheckRunResult_Result{
		{
			Target:      "instances/prod/databases/app",
			Type:        storepb.PlanCheckType_PLAN_CHECK_TYPE_STATEMENT_SUMMARY_REPORT,
			SheetSha256: "sheet-a",
			Report: &storepb.PlanCheckRunResult_Result_SqlSummaryReport_{
				SqlSummaryReport: &storepb.PlanCheckRunResult_Result_SqlSummaryReport{
					AffectedRows: 10,
				},
			},
		},
		{
			Target:      "instances/prod/databases/app",
			Type:        storepb.PlanCheckType_PLAN_CHECK_TYPE_STATEMENT_SUMMARY_REPORT,
			SheetSha256: "sheet-b",
			Report: &storepb.PlanCheckRunResult_Result_SqlSummaryReport_{
				SqlSummaryReport: &storepb.PlanCheckRunResult_Result_SqlSummaryReport{
					AffectedRows: 20,
				},
			},
		},
	}

	got := buildStatementSummaryResultMap(results)

	require.Equal(t, int64(10), got[statementSummaryKey{
		InstanceID:   "prod",
		DatabaseName: "app",
		SheetSHA256:  "sheet-a",
	}].GetSqlSummaryReport().GetAffectedRows())
	require.Equal(t, int64(20), got[statementSummaryKey{
		InstanceID:   "prod",
		DatabaseName: "app",
		SheetSHA256:  "sheet-b",
	}].GetSqlSummaryReport().GetAffectedRows())
}

func TestDeriveCheckTargetsSkipsCreateDatabaseAndReleaseSpecs(t *testing.T) {
	project := &store.ProjectMessage{ResourceID: "project"}
	plan := &store.PlanMessage{
		ProjectID: "project",
		Config: &storepb.PlanConfig{
			Specs: []*storepb.PlanConfig_Spec{
				{
					Config: &storepb.PlanConfig_Spec_CreateDatabaseConfig{
						CreateDatabaseConfig: &storepb.PlanConfig_CreateDatabaseConfig{},
					},
				},
				{
					Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
						ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{
							Release: "projects/project/releases/release",
						},
					},
				},
				{
					Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
						ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{
							Targets: []string{"instances/prod/databases/app"},
						},
					},
				},
			},
		},
	}

	got, err := plancheck.DeriveCheckTargets(context.Background(), nil, project, plan, nil)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "instances/prod/databases/app", got[0].Target)
}

func TestIsPlanCheckRunCurrentForApprovalInputVersion(t *testing.T) {
	tests := []struct {
		name         string
		planVersion  int64
		planCheckRun *store.PlanCheckRunMessage
		wantCurrent  bool
		wantPending  bool
	}{
		{
			name:         "nil run waits for plan check",
			planVersion:  0,
			planCheckRun: nil,
			wantPending:  true,
		},
		{
			name:        "available run is pending",
			planVersion: 2,
			planCheckRun: &store.PlanCheckRunMessage{
				Status: store.PlanCheckRunStatusAvailable,
				Result: &storepb.PlanCheckRunResult{ApprovalInputVersion: 2},
			},
			wantPending: true,
		},
		{
			name:        "running run is pending",
			planVersion: 2,
			planCheckRun: &store.PlanCheckRunMessage{
				Status: store.PlanCheckRunStatusRunning,
				Result: &storepb.PlanCheckRunResult{ApprovalInputVersion: 2},
			},
			wantPending: true,
		},
		{
			name:        "done matching version is current",
			planVersion: 2,
			planCheckRun: &store.PlanCheckRunMessage{
				Status: store.PlanCheckRunStatusDone,
				Result: &storepb.PlanCheckRunResult{ApprovalInputVersion: 2},
			},
			wantCurrent: true,
		},
		{
			name:        "done stale version is not current",
			planVersion: 2,
			planCheckRun: &store.PlanCheckRunMessage{
				Status: store.PlanCheckRunStatusDone,
				Result: &storepb.PlanCheckRunResult{ApprovalInputVersion: 1},
			},
		},
		{
			name:        "missing result defaults to version zero",
			planVersion: 0,
			planCheckRun: &store.PlanCheckRunMessage{
				Status: store.PlanCheckRunStatusDone,
				Result: &storepb.PlanCheckRunResult{},
			},
			wantCurrent: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			current, pending := isPlanCheckRunCurrentForApprovalInputVersion(tt.planCheckRun, tt.planVersion)
			require.Equal(t, tt.wantCurrent, current)
			require.Equal(t, tt.wantPending, pending)
		})
	}
}

func TestUnfoldDatabaseTargetsUsesResolvedDatabaseGroup(t *testing.T) {
	database := &store.DatabaseMessage{
		InstanceID:   "prod",
		DatabaseName: "app",
	}
	databaseGroup := &v1pb.DatabaseGroup{
		Name: "projects/project/databaseGroups/group",
		MatchedDatabases: []*v1pb.DatabaseGroup_Database{{
			Name: common.FormatDatabase(database.InstanceID, database.DatabaseName),
		}},
	}

	got, err := unfoldDatabaseTargets(
		context.Background(),
		// nil store is intentional: the resolved-group path must not touch the store.
		nil,
		[]string{databaseGroup.Name},
		"project",
		[]*store.DatabaseMessage{database},
		databaseGroup,
	)
	require.NoError(t, err)
	require.Equal(t, []string{common.FormatDatabase(database.InstanceID, database.DatabaseName)}, got)
}

func TestUnfoldDatabaseTargetsFallsBackWhenResolvedGroupNameDiffers(t *testing.T) {
	ctx := context.Background()
	s := setupApprovalRunnerStore(ctx, t)
	allDatabases := setupApprovalDatabaseGroupFixture(ctx, t, s)

	got, err := unfoldDatabaseTargets(
		ctx,
		s,
		[]string{"projects/project-a/databaseGroups/group"},
		"project-a",
		allDatabases,
		&v1pb.DatabaseGroup{Name: "projects/project-a/databaseGroups/other"},
	)
	require.NoError(t, err)
	require.Equal(t, []string{"instances/prod/databases/app"}, got)
}

func TestUnfoldSpecTargetsDirectTargetDoesNotListProjectDatabases(t *testing.T) {
	ctx := context.Background()
	s := setupApprovalRunnerStore(ctx, t)
	setupApprovalDatabaseGroupFixture(ctx, t, s)

	_, err := s.CreateInstance(ctx, &store.InstanceMessage{
		ResourceID: "other",
		Workspace:  "default",
		Metadata: &storepb.Instance{
			Engine:      storepb.Engine_POSTGRES,
			DataSources: []*storepb.DataSource{{Id: "admin", Type: storepb.DataSourceType_ADMIN}},
		},
	})
	require.NoError(t, err)

	_, err = s.GetDB().ExecContext(ctx, `
		INSERT INTO db (instance, name, project, metadata)
		VALUES ($1, $2, $3, $4::jsonb)
	`, "other", "broken", "project-a", `{"labels":"not-a-map"}`)
	require.NoError(t, err)

	targets, err := unfoldSpecTargets(ctx, s, []*storepb.PlanConfig_Spec{{
		Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
			ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{
				Targets: []string{"instances/prod/databases/app"},
			},
		},
	}}, "project-a", nil, nil)
	require.NoError(t, err)
	require.Len(t, targets, 1)
	require.Equal(t, "prod", targets[0].database.InstanceID)
	require.Equal(t, "app", targets[0].database.DatabaseName)
}

func TestBuildCELVariablesForDatabaseChangeCreatesMissingPlanCheckRun(t *testing.T) {
	ctx := context.Background()
	s := setupApprovalRunnerStore(ctx, t)

	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID:   "project-a",
		Name:        "plan-a",
		Description: "",
		Config: &storepb.PlanConfig{
			ApprovalInputVersion: 2,
			Specs: []*storepb.PlanConfig_Spec{{
				Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
					ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{
						Targets: []string{"instances/prod/databases/app"},
					},
				},
			}},
		},
	}, "creator@example.com")
	require.NoError(t, err)

	issue, err := s.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "issue-a",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Description:  "",
		Payload:      &storepb.Issue{},
		PlanUID:      &plan.UID,
	})
	require.NoError(t, err)

	celVarsList, approvalInputVersion, done, err := buildCELVariablesForDatabaseChange(ctx, s, issue)
	require.NoError(t, err)
	require.False(t, done)
	require.Nil(t, celVarsList)
	require.EqualValues(t, 2, approvalInputVersion)

	planCheckRun, err := s.GetPlanCheckRun(ctx, "project-a", plan.UID)
	require.NoError(t, err)
	require.NotNil(t, planCheckRun)
	require.Equal(t, store.PlanCheckRunStatusAvailable, planCheckRun.Status)
	require.EqualValues(t, 2, planCheckRun.Result.GetApprovalInputVersion())
}

func TestFindApprovalTemplateForIssueSkipsDatabaseChangeAfterRolloutWhenApprovalRequired(t *testing.T) {
	ctx := context.Background()
	s := setupApprovalRunnerStore(ctx, t)
	require.NoError(t, s.UpdateProjects(ctx, &store.UpdateProjectMessage{
		ResourceID: "project-a",
		Workspace:  "default",
		Setting:    &storepb.Project{RequireIssueApproval: true},
	}))

	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID:   "project-a",
		Name:        "plan-a",
		Description: "",
		Config: &storepb.PlanConfig{
			ApprovalInputVersion: 2,
			Specs: []*storepb.PlanConfig_Spec{{
				Config: &storepb.PlanConfig_Spec_CreateDatabaseConfig{
					CreateDatabaseConfig: &storepb.PlanConfig_CreateDatabaseConfig{
						Target:      "instances/prod",
						Database:    "app",
						Environment: "prod",
					},
				},
			}},
		},
	}, "creator@example.com")
	require.NoError(t, err)

	issue, err := s.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "issue-a",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Description:  "",
		Payload:      &storepb.Issue{Labels: []string{"prod"}},
		PlanUID:      &plan.UID,
	})
	require.NoError(t, err)

	approvalInputVersion := int64(2)
	marked, _, err := s.CreateRolloutTasks(ctx, "project-a", plan.UID, &store.IssueApprovalGuard{ApprovalInputVersion: approvalInputVersion}, nil)
	require.NoError(t, err)
	require.True(t, marked)

	licenseService, err := enterprise.NewLicenseService(common.ReleaseModeDev, s, false, "")
	require.NoError(t, err)
	err = findApprovalTemplateForIssue(ctx, s, nil, licenseService, issue, &storepb.WorkspaceApprovalSetting{})
	require.NoError(t, err)

	got, err := s.GetIssue(ctx, &store.FindIssueMessage{ProjectIDs: []string{"project-a"}, UID: &issue.UID})
	require.NoError(t, err)
	require.Nil(t, got.Payload.GetApproval())
	require.Equal(t, storepb.RiskLevel_RISK_LEVEL_UNSPECIFIED, got.Payload.GetRiskLevel())
}

func TestFindApprovalTemplateForIssueCompletesAfterRolloutWhenApprovalNotRequired(t *testing.T) {
	ctx := context.Background()
	s := setupApprovalRunnerStore(ctx, t)

	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID:   "project-a",
		Name:        "plan-a",
		Description: "",
		Config: &storepb.PlanConfig{
			ApprovalInputVersion: 2,
			Specs: []*storepb.PlanConfig_Spec{{
				Config: &storepb.PlanConfig_Spec_CreateDatabaseConfig{
					CreateDatabaseConfig: &storepb.PlanConfig_CreateDatabaseConfig{
						Target:      "instances/prod",
						Database:    "app",
						Environment: "prod",
					},
				},
			}},
		},
	}, "creator@example.com")
	require.NoError(t, err)

	issue, err := s.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "issue-a",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Description:  "",
		Payload:      &storepb.Issue{Labels: []string{"prod"}},
		PlanUID:      &plan.UID,
	})
	require.NoError(t, err)

	approvalInputVersion := int64(2)
	marked, _, err := s.CreateRolloutTasks(ctx, "project-a", plan.UID, &store.IssueApprovalGuard{ApprovalInputVersion: approvalInputVersion}, nil)
	require.NoError(t, err)
	require.True(t, marked)

	licenseService, err := enterprise.NewLicenseService(common.ReleaseModeDev, s, false, "")
	require.NoError(t, err)
	err = findApprovalTemplateForIssue(ctx, s, nil, licenseService, issue, &storepb.WorkspaceApprovalSetting{})
	require.NoError(t, err)

	got, err := s.GetIssue(ctx, &store.FindIssueMessage{ProjectIDs: []string{"project-a"}, UID: &issue.UID})
	require.NoError(t, err)
	require.True(t, got.Payload.GetApproval().GetApprovalFindingDone())
	require.EqualValues(t, 2, got.Payload.GetApproval().GetApprovalInputVersion())
	require.Nil(t, got.Payload.GetApproval().GetApprovalTemplate())
}

func TestProcessIssueSkipsDraft(t *testing.T) {
	ctx := context.Background()
	s := setupApprovalRunnerStore(ctx, t)

	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a",
		Name:      "draft plan",
		Config: &storepb.PlanConfig{
			Specs: []*storepb.PlanConfig_Spec{{
				Config: &storepb.PlanConfig_Spec_CreateDatabaseConfig{
					CreateDatabaseConfig: &storepb.PlanConfig_CreateDatabaseConfig{
						Target:      "instances/prod",
						Database:    "app",
						Environment: "prod",
					},
				},
			}},
		},
	}, "creator@example.com")
	require.NoError(t, err)

	issue, err := s.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "draft issue",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Payload:      &storepb.Issue{Draft: true},
		PlanUID:      &plan.UID,
	})
	require.NoError(t, err)

	b, err := bus.New()
	require.NoError(t, err)
	licenseService, err := enterprise.NewLicenseService(common.ReleaseModeDev, s, false, "")
	require.NoError(t, err)

	NewRunner(s, b, nil, licenseService).processIssue(ctx, bus.IssueRef{
		ProjectID: issue.ProjectID,
		UID:       issue.UID,
	})

	gotIssue, err := s.GetIssue(ctx, &store.FindIssueMessage{
		ProjectIDs: []string{issue.ProjectID},
		UID:        &issue.UID,
	})
	require.NoError(t, err)
	require.NotNil(t, gotIssue)
	require.True(t, gotIssue.Payload.GetDraft())
	require.Nil(t, gotIssue.Payload.GetApproval())

	gotPlan, err := s.GetPlan(ctx, &store.FindPlanMessage{
		ProjectID: issue.ProjectID,
		UID:       &plan.UID,
	})
	require.NoError(t, err)
	require.NotNil(t, gotPlan)
	require.False(t, gotPlan.Config.GetHasRollout())
	require.Empty(t, b.RolloutCreationChan)
}

func setupApprovalRunnerStore(ctx context.Context, t *testing.T) *store.Store {
	t.Helper()

	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })

	db := container.GetDB()
	require.NoError(t, migrator.MigrateSchema(ctx, db))

	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('default');
		INSERT INTO principal (name, email, password_hash) VALUES ('creator', 'creator@example.com', 'unused');
		INSERT INTO project (resource_id, workspace, name) VALUES ('project-a', 'default', 'Project A');
	`)
	require.NoError(t, err)

	pgURL := fmt.Sprintf(
		"host=%s port=%s user=postgres password=root-password database=postgres",
		container.GetHost(), container.GetPort(),
	)
	s, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, s.Close()) })
	return s
}

func setupApprovalDatabaseGroupFixture(ctx context.Context, t *testing.T, s *store.Store) []*store.DatabaseMessage {
	t.Helper()

	_, err := s.CreateInstance(ctx, &store.InstanceMessage{
		ResourceID: "prod",
		Workspace:  "default",
		Metadata: &storepb.Instance{
			Engine:      storepb.Engine_POSTGRES,
			DataSources: []*storepb.DataSource{{Id: "admin", Type: storepb.DataSourceType_ADMIN}},
		},
	})
	require.NoError(t, err)
	_, err = s.UpsertDatabase(ctx, &store.DatabaseMessage{
		ProjectID:    "project-a",
		InstanceID:   "prod",
		DatabaseName: "app",
		Metadata:     &storepb.DatabaseMetadata{Labels: map[string]string{}},
	})
	require.NoError(t, err)
	_, err = s.CreateDatabaseGroup(ctx, &store.DatabaseGroupMessage{
		ProjectID:  "project-a",
		ResourceID: "group",
		Title:      "group",
		Expression: &expr.Expr{Expression: `resource.database_name == "app"`},
	})
	require.NoError(t, err)

	projectID := "project-a"
	allDatabases, err := s.ListDatabases(ctx, &store.FindDatabaseMessage{ProjectID: &projectID})
	require.NoError(t, err)
	return allDatabases
}
