package common

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
			expression: "request.time < timestamp(\"2023-07-04T06:09:03.384Z\") && request.export_format == \"CSV\" && request.row_limit == 1000 && (resource.database == \"instances/postgres-sample/databases/employee\" && resource.schema == \"public\" && resource.table in [\"dept_manager\"])",
			want: queryExportFactors{
				databaseNames: []string{"instances/postgres-sample/databases/employee"},
				exportRows:    1000,
			},
		},
	}
	for _, tt := range tests {
		factors, err := getQueryExportFactors(tt.expression)
		a.NoError(err)
		a.Equal(tt.want, *factors)
	}
	// "request.time < timestamp(\"2023-07-04T07:40:05.658Z\") && request.export_format == \"CSV\" && request.row_limit == 1000 && request.statement == \"c2VsZWN0ICogZnJvbSBlbXBsb3llZTs=\" && (resource.database in [\"instances/postgres-sample/databases/employee\"])"
	// "request.time < timestamp(\"2023-08-02T07:33:45.686Z\") && (resource.database == \"instances/postgres-sample/databases/employee\" && resource.schema == \"public\" && resource.table in [\"dept_emp\",\"department\"])"
}
