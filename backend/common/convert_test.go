package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetQueryExportFactors(t *testing.T) {
	a := assert.New(t)
	tests := []struct {
		expression string
		want       QueryExportFactors
	}{
		{
			expression: "request.time < timestamp(\"2023-07-04T06:09:03.384Z\") && request.export_format == \"CSV\" && request.row_limit == 1000 && (resource.database == \"instances/postgres-sample/databases/employee\" && resource.schema == \"public\" && resource.table in [\"dept_manager\"])",
			want: QueryExportFactors{
				DatabaseNames: []string{"instances/postgres-sample/databases/employee"},
				ExportRows:    1000,
			},
		},
		{
			expression: "request.time < timestamp(\"2023-07-04T07:40:05.658Z\") && request.export_format == \"CSV\" && request.row_limit == 1000 && request.statement == \"c2VsZWN0ICogZnJvbSBlbXBsb3llZTs=\" && (resource.database in [\"instances/postgres-sample/databases/employee\"])",
			want: QueryExportFactors{
				DatabaseNames: []string{"instances/postgres-sample/databases/employee"},
				ExportRows:    1000,
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
