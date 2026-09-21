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
