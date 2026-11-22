package review

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetQueryExportFactors(t *testing.T) {
	a := assert.New(t)
	tests := []struct {
		expression string
		want       queryExportFactors
	}{
		{
			expression: "request.time < timestamp(\"2023-07-04T06:09:03.384Z\") && (resource.database == \"instances/postgres-sample/databases/employee\" && resource.schema_name == \"public\" && resource.table_name in [\"dept_manager\"])",
			want: queryExportFactors{
				Databases: []string{"instances/postgres-sample/databases/employee"},
			},
		},
		{
			expression: "request.time < timestamp(\"2023-07-04T07:40:05.658Z\") && (resource.database in [\"instances/postgres-sample/databases/employee\"])",
			want: queryExportFactors{
				Databases: []string{"instances/postgres-sample/databases/employee"},
			},
		},
		{
			expression: "request.time < timestamp(\"2023-08-02T07:33:45.686Z\") && (resource.database == \"instances/postgres-sample/databases/employee\" && resource.schema_name == \"public\" && resource.table_name in [\"dept_emp\",\"department\"])",
			want: queryExportFactors{
				Databases: []string{"instances/postgres-sample/databases/employee"},
			},
		},
		{
			expression: "request.time < timestamp(\"2023-07-10T08:14:34.788Z\")",
			want:       queryExportFactors{},
		},
		{
			expression: "request.time < timestamp(\"2023-07-10T08:15:46.773Z\") && ((resource.database in [\"instances/postgres-sample/databases/blog\"]) || (resource.database == \"instances/postgres-sample/databases/employee\" && resource.schema_name in [\"public\"]))",
			want: queryExportFactors{
				Databases: []string{"instances/postgres-sample/databases/blog", "instances/postgres-sample/databases/employee"},
			},
		},
	}
	for _, tt := range tests {
		factors, err := getQueryExportFactors(tt.expression)
		a.NoError(err)
		a.Equal(tt.want, *factors)
	}
}

func TestGetQueryExportFactorsDatabaseScope(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name       string
		expression string
		want       []string
	}{
		{"empty condition", "", nil},
		{"equality", `resource.database == "dev"`, []string{"dev"}},
		{"reversed equality", `"dev" == resource.database`, []string{"dev"}},
		{"membership", `resource.database in ["dev", "prod"]`, []string{"dev", "prod"}},
		{"duplicate membership", `resource.database in ["dev", "dev"]`, []string{"dev"}},
		{"negative membership", `!(resource.database in ["dev"])`, nil},
		{"negative equality", `!(resource.database == "dev")`, nil},
		{"not equal", `resource.database != "dev"`, nil},
		{"normalized double negation", `!!(resource.database == "dev")`, []string{"dev"}},
		{"contains", `resource.database.contains("prod")`, nil},
		{"matches", `resource.database.matches("prod.*")`, nil},
		{"positive and negative", `resource.database == "prod" && !(resource.database in ["dev"])`, []string{"prod"}},
		{"negative and positive", `!(resource.database in ["dev"]) && resource.database == "prod"`, []string{"prod"}},
		{"intersection", `resource.database in ["dev", "prod"] && resource.database in ["prod", "test"]`, []string{"prod"}},
		{"bounded disjunction", `resource.database == "dev" || resource.database == "prod"`, []string{"dev", "prod"}},
		{"unbounded right disjunction", `resource.database == "dev" || resource.table_name == "orders"`, nil},
		{"unbounded left disjunction", `resource.table_name == "orders" || resource.database == "dev"`, nil},
		{"negative disjunction", `resource.database == "dev" || !(resource.database in ["prod"])`, nil},
		{"negated group", `!(resource.database == "dev" || resource.database == "prod")`, nil},
		{"bounded nested disjunction", `(resource.database == "dev" && resource.table_name == "orders") || resource.database == "prod"`, []string{"dev", "prod"}},
		{"bounded outer conjunction", `(resource.database == "dev" || resource.table_name == "orders") && resource.database == "prod"`, []string{"prod"}},
		{"true disjunction", `resource.database == "dev" || true`, nil},
		{"false disjunction", `false || resource.database == "prod"`, []string{"prod"}},
		{"empty list disjunction", `resource.database in [] || resource.database == "prod"`, []string{"prod"}},
		{"contradiction disjunction", `(resource.database == "dev" && resource.database == "prod") || resource.database == "test"`, []string{"test"}},
		{"computed equality", `resource.database == resource.table_name`, nil},
		{"computed membership", `resource.database in ["dev", resource.table_name]`, nil},
		{"computed membership list", `resource.database in resource.table_name.split(",")`, nil},
		{"conditional expression", `resource.table_name == "orders" ? resource.database == "dev" : true`, nil},
		{"nested function", `string(resource.database == "dev") == "false"`, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			factors, err := getQueryExportFactors(tc.expression)
			if assert.NoError(t, err) {
				assert.Equal(t, tc.want, factors.Databases)
			}
		})
	}
}
