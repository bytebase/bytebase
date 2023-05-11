package server

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	parser "github.com/bytebase/bytebase/backend/plugin/parser/sql"
)

func TestValidateSQLSelectStatement(t *testing.T) {
	engines := []parser.EngineType{parser.MySQL, parser.Postgres, parser.MariaDB, parser.MSSQL, parser.Oracle, parser.TiDB, parser.Standard}
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
		for _, engine := range engines {
			result := parser.ValidateSQLForEditor(engine, test.sqlStatement)
			if result != test.want {
				t.Errorf("Validate SQLStatement %q: got result %v, want %v, for engine %s.", test.sqlStatement, result, test.want, engine)
			}
		}
	}
}

func TestEvaluateCondition(t *testing.T) {
	tests := []struct {
		testName   string
		expression string
		attributes map[string]any
		want       bool
	}{
		{
			"simple",
			``,
			map[string]any{
				"request.time":          time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				"resource.database":     "haha",
				"request.statement":     "SELECT * FROM haha.lala",
				"request.row_limit":     1000,
				"request.export_format": "CSV",
			},
			true,
		},
		{
			"request time not expired",
			`request.time < timestamp("2024-01-01T00:00:00Z")`,
			map[string]any{
				"request.time":          time.Date(2021, 1, 13, 0, 0, 0, 0, time.UTC),
				"resource.database":     "haha",
				"request.statement":     "SELECT * FROM haha.lala",
				"request.row_limit":     1000,
				"request.export_format": "CSV",
			},
			true,
		},
		{
			"request time expired",
			`request.time < timestamp("2024-01-01T00:00:00Z")`,
			map[string]any{
				"request.time":          time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC),
				"resource.database":     "haha",
				"request.statement":     "SELECT * FROM haha.lala",
				"request.row_limit":     1000,
				"request.export_format": "CSV",
			},
			false,
		},
		{
			"test database",
			`request.time < timestamp("2024-01-01T00:00:00Z") && resource.database in ["haha"]`,
			map[string]any{
				"request.time":          time.Date(2021, 1, 13, 0, 0, 0, 0, time.UTC),
				"resource.database":     "haha",
				"request.statement":     "SELECT * FROM haha.lala",
				"request.row_limit":     1000,
				"request.export_format": "CSV",
			},
			true,
		},
		{
			"test database failure",
			`request.time < timestamp("2024-01-01T00:00:00Z") && resource.database in ["hehe"]`,
			map[string]any{
				"request.time":          time.Date(2021, 1, 13, 0, 0, 0, 0, time.UTC),
				"resource.database":     "haha",
				"request.statement":     "SELECT * FROM haha.lala",
				"request.row_limit":     1000,
				"request.export_format": "CSV",
			},
			false,
		},
		{
			"test statement",
			`request.time < timestamp("2024-01-01T00:00:00Z") && request.statement == "SELECT * FROM haha.lala"`,
			map[string]any{
				"request.time":          time.Now(),
				"resource.database":     "haha",
				"request.statement":     "SELECT * FROM haha.lala",
				"request.row_limit":     1000,
				"request.export_format": "CSV",
			},
			true,
		},
		{
			"test statement failed",
			`request.time < timestamp("2024-01-01T00:00:00Z") && request.statement == "SELECT * FROM haha.lala"`,
			map[string]any{
				"request.time":          time.Now(),
				"resource.database":     "haha",
				"request.statement":     "SELECT * FROM yolo",
				"request.row_limit":     1000,
				"request.export_format": "CSV",
			},
			false,
		},
		{
			"test row limit",
			`request.time < timestamp("2024-01-01T00:00:00Z") && request.row_limit <= 1000`,
			map[string]any{
				"request.time":          time.Now(),
				"resource.database":     "haha",
				"request.statement":     "SELECT * FROM yolo",
				"request.row_limit":     1000,
				"request.export_format": "CSV",
			},
			true,
		},
		{
			"test row limit failed",
			`request.time < timestamp("2024-01-01T00:00:00Z") && request.row_limit <= 1000`,
			map[string]any{
				"request.time":          time.Now(),
				"resource.database":     "haha",
				"request.statement":     "SELECT * FROM yolo",
				"request.row_limit":     1001,
				"request.export_format": "CSV",
			},
			false,
		},
		{
			"test export format",
			`request.time < timestamp("2024-01-01T00:00:00Z")`,
			map[string]any{
				"request.time":          time.Now(),
				"resource.database":     "haha",
				"request.statement":     "SELECT * FROM yolo",
				"request.row_limit":     1000,
				"request.export_format": "QUERY",
			},
			true,
		},
		{
			"test row limit failed",
			`request.time < timestamp("2024-01-01T00:00:00Z") && request.export_format == "CSV"`,
			map[string]any{
				"request.time":          time.Now(),
				"resource.database":     "haha",
				"request.statement":     "SELECT * FROM yolo",
				"request.row_limit":     1001,
				"request.export_format": "ZIP",
			},
			false,
		},
	}
	for _, test := range tests {
		got, err := evaluateCondition(test.expression, test.attributes)
		require.NoError(t, err, test.testName)
		require.Equal(t, test.want, got, test.testName)
	}
}
