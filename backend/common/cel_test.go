package common

import (
	"testing"
	"time"

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
			expr:  "request.time < timestamp(\"2024-02-01T00:00:00Z\") && request.row_limit <= 1000",
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
			expr:  "request.time < timestamp(\"2024-02-01T00:00:00Z\") && request.row_limit <= 1000",
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
			expression: "request.time < timestamp(\"2023-07-04T06:09:03.384Z\") && request.row_limit == 1000 && (resource.database == \"instances/postgres-sample/databases/employee\" && resource.schema == \"public\" && resource.table in [\"dept_manager\"])",
			want: QueryExportFactors{
				DatabaseNames: []string{"instances/postgres-sample/databases/employee"},
				ExportRows:    1000,
			},
		},
		{
			expression: "request.time < timestamp(\"2023-07-04T07:40:05.658Z\") && request.row_limit == 1000 && request.statement == \"c2VsZWN0ICogZnJvbSBlbXBsb3llZTs=\" && (resource.database in [\"instances/postgres-sample/databases/employee\"])",
			want: QueryExportFactors{
				DatabaseNames: []string{"instances/postgres-sample/databases/employee"},
				ExportRows:    1000,
				Statement:     "select * from employee;",
			},
		},
		{
			expression: "request.time < timestamp(\"2023-08-02T07:33:45.686Z\") && (resource.database == \"instances/postgres-sample/databases/employee\" && resource.schema == \"public\" && resource.table in [\"dept_emp\",\"department\"])",
			want: QueryExportFactors{
				DatabaseNames: []string{"instances/postgres-sample/databases/employee"},
			},
		},
		{
			expression: "request.time < timestamp(\"2023-07-10T08:14:34.788Z\")",
			want:       QueryExportFactors{},
		},
		{
			expression: "request.time < timestamp(\"2023-07-10T08:15:46.773Z\") && ((resource.database in [\"instances/postgres-sample/databases/blog\"]) || (resource.database == \"instances/postgres-sample/databases/employee\" && resource.schema in [\"public\"]))",
			want: QueryExportFactors{
				DatabaseNames: []string{"instances/postgres-sample/databases/blog", "instances/postgres-sample/databases/employee"},
			},
		},
	}
	for _, tt := range tests {
		factors, err := GetQueryExportFactors(tt.expression)
		a.NoError(err)
		a.Equal(tt.want, *factors)
	}
}
