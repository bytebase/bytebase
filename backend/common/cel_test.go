//nolint:revive
package common

import (
	"testing"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestGetQueryExportFactors(t *testing.T) {
	a := assert.New(t)
	tests := []struct {
		expression string
		want       QueryExportFactors
	}{
		{
			expression: "request.time < timestamp(\"2023-07-04T06:09:03.384Z\") && (resource.database == \"instances/postgres-sample/databases/employee\" && resource.schema_name == \"public\" && resource.table_name in [\"dept_manager\"])",
			want: QueryExportFactors{
				Databases: []string{"instances/postgres-sample/databases/employee"},
			},
		},
		{
			expression: "request.time < timestamp(\"2023-07-04T07:40:05.658Z\") && (resource.database in [\"instances/postgres-sample/databases/employee\"])",
			want: QueryExportFactors{
				Databases: []string{"instances/postgres-sample/databases/employee"},
			},
		},
		{
			expression: "request.time < timestamp(\"2023-08-02T07:33:45.686Z\") && (resource.database == \"instances/postgres-sample/databases/employee\" && resource.schema_name == \"public\" && resource.table_name in [\"dept_emp\",\"department\"])",
			want: QueryExportFactors{
				Databases: []string{"instances/postgres-sample/databases/employee"},
			},
		},
		{
			expression: "request.time < timestamp(\"2023-07-10T08:14:34.788Z\")",
			want:       QueryExportFactors{},
		},
		{
			expression: "request.time < timestamp(\"2023-07-10T08:15:46.773Z\") && ((resource.database in [\"instances/postgres-sample/databases/blog\"]) || (resource.database == \"instances/postgres-sample/databases/employee\" && resource.schema_name in [\"public\"]))",
			want: QueryExportFactors{
				Databases: []string{"instances/postgres-sample/databases/blog", "instances/postgres-sample/databases/employee"},
			},
		},
	}
	for _, tt := range tests {
		factors, err := GetQueryExportFactors(tt.expression)
		a.NoError(err)
		a.Equal(tt.want, *factors)
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

func TestValidateFallbackApprovalExpr(t *testing.T) {
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
			err := ValidateFallbackApprovalExpr(tt.expression)
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
