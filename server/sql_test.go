package server

import (
	"testing"
)

func TestValidateSQLSelectStatement(t *testing.T) {
	tests := []struct {
		sqlStatement string
		want         bool
	}{
		{
			sqlStatement: "  seLeCT * FROM test",
			want:         true,
		},
		{
			sqlStatement: "  \n \r SELEct * from test ",
			want:         true,
		},
		{
			sqlStatement: "insert into asd",
			want:         false,
		},
		{
			sqlStatement: "SETEST * FROM test",
			want:         false,
		},
		{
			sqlStatement: "select",
			want:         true,
		},
		{
			sqlStatement: "",
			want:         false,
		},
		{
			sqlStatement: "    ",
			want:         false,
		},
	}

	for _, test := range tests {
		result := validateSQLSelectStatement(test.sqlStatement)
		if result != test.want {
			t.Errorf("Validate SQLStatement %q: got result %v, want %v.", test.sqlStatement, result, test.want)
		}
	}
}
