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

func TestGetStatementWithResultLimit(t *testing.T) {
	tests := []struct {
		sqlStatement string
		limit        int
		want         string
	}{
		{
			sqlStatement: "  seLeCT * FROM test;",
			limit:        123,
			want:         "WITH result AS (  seLeCT * FROM test) SELECT * FROM result LIMIT 123;",
		},
		{
			sqlStatement: "  \n \r SELEct * from test ",
			limit:        100,
			want:         "WITH result AS (  \n \r SELEct * from test) SELECT * FROM result LIMIT 100;",
		},
		{
			sqlStatement: "SELECT\n*\nFROM\ntest  ;\n",
			limit:        100,
			want:         "WITH result AS (SELECT\n*\nFROM\ntest) SELECT * FROM result LIMIT 100;",
		},
		{
			sqlStatement: "SELECT\n*\nFROM\ntest  ;;;\n",
			limit:        100,
			want:         "WITH result AS (SELECT\n*\nFROM\ntest) SELECT * FROM result LIMIT 100;",
		},
		{
			sqlStatement: "SELECT\n*\nFROM\n`test;`  ;;;\n",
			limit:        100,
			want:         "WITH result AS (SELECT\n*\nFROM\n`test;`) SELECT * FROM result LIMIT 100;",
		},
		{
			sqlStatement: "EXPLAIN  \n \r SELEct * from test ",
			limit:        0,
			want:         "EXPLAIN  \n \r SELEct * from test",
		},
	}

	for _, test := range tests {
		got := getStatementWithResultLimit(test.sqlStatement, test.limit)
		if got != test.want {
			t.Errorf("trimSQLStatement %q: got result %v, want %v.", test.sqlStatement, got, test.want)
		}
	}
}
