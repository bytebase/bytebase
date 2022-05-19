package server

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
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
			sqlStatement: "SELECT\n*\nFROM\ntest",
			want:         true,
		},
		{
			sqlStatement: "SELECT * FROM test",
			want:         true,
		},
		{
			sqlStatement: "select *",
			want:         true,
		},
		{
			sqlStatement: "select ",
			want:         false,
		},
		{
			sqlStatement: "select",
			want:         false,
		},
		{
			sqlStatement: "explain select",
			want:         true,
		},
		{
			sqlStatement: "explain \n select",
			want:         true,
		},
		{
			sqlStatement: "\n explain \n \r  select",
			want:         true,
		},
		{
			sqlStatement: "explain select *",
			want:         true,
		},
		{
			sqlStatement: "  explain select ",
			want:         true,
		},
		{
			sqlStatement: "asd  explain selectasd ",
			want:         false,
		},
		{
			sqlStatement: "SELECTfoo",
			want:         false,
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
			sqlStatement: " asdexplain selectasd ",
			want:         false,
		},
		{
			sqlStatement: "",
			want:         false,
		},
		{
			sqlStatement: "SETEST 1; INSERT INTO tbl(num) VALUES(113);",
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

func TestXxx(t *testing.T) {
	type S struct {
		A string `json:"a"`
	}
	s := S{}
	err := json.Unmarshal([]byte("{}"), &s)
	require.NoError(t, err)
	t.Logf("s: %+v\n", s)
}
