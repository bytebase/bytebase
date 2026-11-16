package plsql

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common/yamltest"
)

func TestGetStatementWithResultLimit(t *testing.T) {
	runLimitTest(t, "test_limit.yaml", false /* record */)
}

func TestAddResultLimitKeepsNonSelectStatementsUnchanged(t *testing.T) {
	tests := []struct {
		name          string
		statement     string
		engineVersion string
	}{
		{
			name:          "12c update",
			statement:     "UPDATE employees SET salary = salary + 1",
			engineVersion: "19.0.0.0.0",
		},
		{
			name:          "11g delete",
			statement:     "DELETE FROM employees WHERE id = 1",
			engineVersion: "11.2.0.4.0",
		},
		{
			name:          "leading whitespace insert",
			statement:     "\n\tINSERT INTO employees(id) VALUES (1)",
			engineVersion: "19.0.0.0.0",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := statementWithResultLimit(tc.statement, 10, tc.engineVersion)

			require.Equal(t, tc.statement, got)
		})
	}
}

func TestAddResultLimitSkipsOnlySimpleDualSelect(t *testing.T) {
	tests := []struct {
		name      string
		statement string
		want      string
	}{
		{
			name:      "simple dual select",
			statement: "SELECT 1 FROM DUAL",
			want:      "SELECT 1 FROM DUAL",
		},
		{
			name:      "dual select with scalar subquery",
			statement: "SELECT (SELECT COUNT(*) FROM employees) FROM DUAL",
			want:      "SELECT (SELECT COUNT(*) FROM employees) FROM DUAL FETCH NEXT 10 ROWS ONLY",
		},
		{
			name:      "dual select star keeps historical no-skip behavior",
			statement: "SELECT * FROM DUAL",
			want:      "SELECT * FROM DUAL FETCH NEXT 10 ROWS ONLY",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := statementWithResultLimit(tc.statement, 10, "19.0.0.0.0")

			require.Equal(t, tc.want, got)
		})
	}
}

func TestAddLimitFor12cAndLaterFallsBackForPercentFetch(t *testing.T) {
	stmt := "SELECT * FROM employees FETCH NEXT 10 PERCENT ROWS ONLY"

	got := statementWithResultLimit(stmt, 5, "19.0.0.0.0")

	require.Equal(t, "SELECT * FROM (SELECT * FROM employees FETCH NEXT 10 PERCENT ROWS ONLY) WHERE ROWNUM <= 5", got)
}

func runLimitTest(t *testing.T, file string, record bool) {
	var testCases []limitTestData
	filepath := filepath.Join("test-data", file)
	yamlFile, err := os.Open(filepath)
	require.NoError(t, err)
	defer yamlFile.Close()

	byteValue, err := io.ReadAll(yamlFile)
	require.NoError(t, err)
	err = yaml.Unmarshal(byteValue, &testCases)
	require.NoError(t, err)

	for i, tc := range testCases {
		want := statementWithResultLimit(tc.Stmt, tc.Limit, "19.0.0.0.0")
		if record {
			testCases[i].Want = want
		} else {
			require.Equal(t, tc.Want, want, tc.Stmt)
		}
	}

	if record {
		err := yamlFile.Close()
		require.NoError(t, err)
		yamltest.Record(t, filepath, testCases)
	}
}

type limitTestData struct {
	Stmt  string `yaml:"stmt"`
	Limit int    `yaml:"limit"`
	Want  string `yaml:"want"`
}
