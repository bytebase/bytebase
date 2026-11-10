//nolint:revive
package common

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/type/expr"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestPartialEval(t *testing.T) {
	a := require.New(t)

	time20240201, err := time.Parse(time.RFC3339, "2024-02-01T00:00:00Z")
	a.NoError(err)

	testCases := []struct {
		name  string
		expr  string
		input map[string]any
		want  bool
	}{
		{
			name:  "simple false 1",
			expr:  "request.time < timestamp(\"2024-02-01T00:00:00Z\")",
			input: map[string]any{"request.time": time20240201},
			want:  false,
		},
		{
			name:  "simple false 2",
			expr:  "request.time < timestamp(\"2024-02-01T00:00:00Z\")",
			input: map[string]any{"request.time": time20240201.AddDate(0, 0, 1)},
			want:  false,
		},
		{
			name:  "simple true 1",
			expr:  "request.time < timestamp(\"2024-02-01T00:00:00Z\")",
			input: map[string]any{"request.time": time20240201.AddDate(0, 0, -1)},
			want:  true,
		},
		{
			name:  "partial false 1",
			expr:  "request.time < timestamp(\"2024-02-01T00:00:00Z\")",
			input: map[string]any{"request.time": time20240201},
			want:  false,
		},
		{
			name:  "partial false 2",
			expr:  "request.time < timestamp(\"2024-02-01T00:00:00Z\") && (resource.database in [\"instances/dbdbdb/databases/db_106\",\"instances/dbdbdb/databases/db_108\",\"instances/dbdbdb/databases/db_103\"])",
			input: map[string]any{"request.time": time20240201},
			want:  false,
		},
		{
			name:  "partial true 1",
			expr:  "request.time < timestamp(\"2024-02-01T00:00:00Z\")",
			input: map[string]any{"request.time": time20240201.AddDate(0, 0, -1)},
			want:  true,
		},
		{
			name:  "partial true 2",
			expr:  "request.time < timestamp(\"2024-02-01T00:00:00Z\") && (resource.database in [\"instances/dbdbdb/databases/db_106\",\"instances/dbdbdb/databases/db_108\",\"instances/dbdbdb/databases/db_103\"])",
			input: map[string]any{"request.time": time20240201.AddDate(0, 0, -1)},
			want:  true,
		},
	}

	for _, tc := range testCases {
		res, err := doEvalBindingCondition(tc.expr, tc.input)
		a.NoError(err)
		a.Equal(tc.want, res, tc.name)
	}
}

func TestFallbackApprovalFactors(t *testing.T) {
	a := require.New(t)

	// Fallback factors should only contain project_id
	a.Len(FallbackApprovalFactors, 2) // 1 variable + 1 size limit

	// Verify the first option is the project_id variable
	// CEL EnvOptions are opaque, so we verify by creating an env
	// and checking that project_id works but environment_id doesn't
}

func TestFallbackApprovalFactorsOnlyAllowsProjectId(t *testing.T) {
	a := require.New(t)

	// Create env with fallback factors
	e, err := cel.NewEnv(FallbackApprovalFactors...)
	a.NoError(err)

	// resource.project_id should compile
	_, issues := e.Compile(`resource.project_id == "proj-123"`)
	a.Nil(issues)

	// resource.environment_id should NOT compile (not in fallback factors)
	_, issues = e.Compile(`resource.environment_id == "prod"`)
	a.NotNil(issues)
	a.Error(issues.Err())
}

func TestValidateApprovalExprForSourceFallbackOnlyAllowsProjectID(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		wantErr    bool
	}{
		{
			name:       "valid project_id condition",
			expression: `resource.project_id == "proj-123"`,
			wantErr:    false,
		},
		{
			name:       "valid true condition",
			expression: `true`,
			wantErr:    false,
		},
		{
			name:       "invalid environment_id condition",
			expression: `resource.environment_id == "prod"`,
			wantErr:    true,
		},
		{
			name:       "invalid statement condition",
			expression: `statement.affected_rows > 100`,
			wantErr:    true,
		},
		{
			name:       "empty condition",
			expression: ``,
			wantErr:    false, // empty is allowed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := require.New(t)
			err := ValidateApprovalExprForSource(tt.expression, storepb.WorkspaceApprovalSetting_Rule_SOURCE_UNSPECIFIED)
			if tt.wantErr {
				a.Error(err)
			} else {
				a.NoError(err)
			}
		})
	}
}

func TestApprovalFactorsIncludesRiskLevel(t *testing.T) {
	a := require.New(t)

	// Create env with approval factors
	e, err := cel.NewEnv(ApprovalFactors...)
	a.NoError(err)

	// risk.level should compile with equality operator
	_, issues := e.Compile(`risk.level == "HIGH"`)
	a.Nil(issues)

	// risk.level should compile with in operator
	_, issues = e.Compile(`risk.level in ["HIGH", "MODERATE"]`)
	a.Nil(issues)

	// risk.level combined with other factors should compile
	_, issues = e.Compile(`risk.level == "HIGH" && resource.environment_id == "prod"`)
	a.Nil(issues)
}

func TestApprovalFactorsIncludesIssueLabels(t *testing.T) {
	a := require.New(t)

	e, err := cel.NewEnv(ApprovalFactors...)
	a.NoError(err)

	_, issues := e.Compile(`"prod" in issue.labels`)
	a.Nil(issues)

	_, issues = e.Compile(`!("security" in issue.labels) && risk.level == "HIGH"`)
	a.Nil(issues)
}

func TestValidateApprovalExprForSourceAllowsIssueLabelsOnlyForDatabaseChange(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		source     storepb.WorkspaceApprovalSetting_Rule_Source
		wantErr    bool
	}{
		{
			name:       "allows issue labels for database change",
			expression: `"prod" in issue.labels`,
			source:     storepb.WorkspaceApprovalSetting_Rule_CHANGE_DATABASE,
			wantErr:    false,
		},
		{
			name:       "rejects issue labels for request role",
			expression: `"prod" in issue.labels`,
			source:     storepb.WorkspaceApprovalSetting_Rule_REQUEST_ROLE,
			wantErr:    true,
		},
		{
			name:       "allows existing approval factors without issue labels",
			expression: `request.role == "roles/projectViewer" && resource.project_id == "project-a"`,
			source:     storepb.WorkspaceApprovalSetting_Rule_REQUEST_ROLE,
			wantErr:    false,
		},
		{
			name:       "allows issue labels text in string literal",
			expression: `"issue.labels" == "issue.labels"`,
			source:     storepb.WorkspaceApprovalSetting_Rule_REQUEST_ROLE,
			wantErr:    false,
		},
		{
			name:       "allows true",
			expression: `true`,
			source:     storepb.WorkspaceApprovalSetting_Rule_REQUEST_ROLE,
			wantErr:    false,
		},
		{
			name:       "allows empty",
			expression: ``,
			source:     storepb.WorkspaceApprovalSetting_Rule_REQUEST_ROLE,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateApprovalExprForSource(tt.expression, tt.source)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestFallbackApprovalFactorsExcludesIssueLabels(t *testing.T) {
	a := require.New(t)

	e, err := cel.NewEnv(FallbackApprovalFactors...)
	a.NoError(err)

	_, issues := e.Compile(`"prod" in issue.labels`)
	a.NotNil(issues)
	a.Error(issues.Err())
}

func TestStringsExtensionEnabled(t *testing.T) {
	a := require.New(t)

	// ext.Strings() functions (split/join/trim) should be available in all
	// user-authored condition environments.
	envs := []struct {
		name string
		opts []cel.EnvOption
	}{
		{"approval", ApprovalFactors},
		{"iam", IAMPolicyConditionCELAttributes},
		{"maskingRule", MaskingRulePolicyCELAttributes},
		{"maskingExemption", MaskingExemptionPolicyCELAttributes},
		{"databaseGroup", DatabaseGroupCELAttributes},
	}
	for _, env := range envs {
		e, err := cel.NewEnv(env.opts...)
		a.NoError(err, env.name)

		_, issues := e.Compile(`"a-b-c".split("-")[0] == "a"`)
		a.Nil(issues, "%s: split should compile", env.name)

		_, issues = e.Compile(`["a", "b"].join("-") == "a-b"`)
		a.Nil(issues, "%s: join should compile", env.name)

		_, issues = e.Compile(`"  x  ".trim() == "x"`)
		a.Nil(issues, "%s: trim should compile", env.name)
	}

	// Evaluate a string-extension expression end to end.
	e, err := cel.NewEnv(MaskingRulePolicyCELAttributes...)
	a.NoError(err)
	ast, issues := e.Compile(`"3-1-2".split("-")[0] == "3"`)
	a.Nil(issues)
	prg, err := e.Program(ast)
	a.NoError(err)
	out, _, err := prg.Eval(map[string]any{})
	a.NoError(err)
	a.Equal(true, out.Value())
}

func TestFallbackApprovalFactorsExcludeStringsExtension(t *testing.T) {
	a := require.New(t)

	// Fallback factors are deliberately restricted; ext.Strings() functions
	// like split() must not be available there.
	e, err := cel.NewEnv(FallbackApprovalFactors...)
	a.NoError(err)

	_, issues := e.Compile(`resource.project_id.split("-")[0] == "proj"`)
	a.NotNil(issues)
	a.Error(issues.Err())
}

func TestFallbackFactorsDoNotIncludeRiskLevel(t *testing.T) {
	a := require.New(t)

	// Create env with fallback factors
	e, err := cel.NewEnv(FallbackApprovalFactors...)
	a.NoError(err)

	// risk.level should NOT compile in fallback factors
	_, issues := e.Compile(`risk.level == "HIGH"`)
	a.NotNil(issues)
	a.Error(issues.Err())
}

func TestConditionStringOperators(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		name      string
		attribute string
		validate  func(string) (cel.Program, error)
	}{
		{
			name:      "approval",
			attribute: CELAttributeResourceTableName,
			validate: func(expression string) (cel.Program, error) {
				if err := ValidateApprovalExprForSource(expression, storepb.WorkspaceApprovalSetting_Rule_CHANGE_DATABASE); err != nil {
					return nil, err
				}
				env, err := cel.NewEnv(ApprovalFactors...)
				if err != nil {
					return nil, err
				}
				ast, issues := env.Compile(expression)
				if issues != nil && issues.Err() != nil {
					return nil, issues.Err()
				}
				return env.Program(ast)
			},
		},
		{
			name:      "iam",
			attribute: CELAttributeResourceTableName,
			validate: func(expression string) (cel.Program, error) {
				return ValidateProjectMemberCELExpr(&expr.Expr{Expression: expression})
			},
		},
		{
			name:      "masking rule",
			attribute: CELAttributeResourceTableName,
			validate:  ValidateMaskingRuleCELExpr,
		},
		{
			name:      "masking exemption",
			attribute: CELAttributeResourceTableName,
			validate: func(expression string) (cel.Program, error) {
				return ValidateMaskingExemptionCELExpr(&expr.Expr{Expression: expression})
			},
		},
		{
			name:      "database group",
			attribute: CELAttributeResourceDatabaseName,
			validate:  ValidateGroupCELExpr,
		},
	}
	operators := []struct {
		name       string
		expression string
		wantMatch  bool
	}{
		{"equals", `%s == "orders_archive"`, true},
		{"not equals", `%s != "orders_archive"`, false},
		{"in", `%s in ["orders_archive", "audit_log"]`, true},
		{"not in", `!(%s in ["orders_archive", "audit_log"])`, false},
		{"contains", `%s.contains("archive")`, true},
		{"not contains", `!%s.contains("archive")`, false},
		{"starts with", `%s.startsWith("orders_")`, true},
		{"ends with", `%s.endsWith("_archive")`, true},
		{"matches", `%s.matches("^orders_[a-z]+$")`, true},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			for _, operator := range operators {
				t.Run(operator.name, func(t *testing.T) {
					program, err := scenario.validate(fmt.Sprintf(operator.expression, scenario.attribute))
					require.NoError(t, err)
					for _, input := range []struct {
						value string
						want  bool
					}{
						{"orders_archive", operator.wantMatch},
						{"customers", !operator.wantMatch},
					} {
						result, _, err := program.Eval(map[string]any{scenario.attribute: input.value})
						require.NoError(t, err, input.value)
						require.Equal(t, input.want, result.Value(), input.value)
					}
				})
			}
		})
	}
}
