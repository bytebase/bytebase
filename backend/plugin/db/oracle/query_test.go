package oracle

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common/yamltest"
	plsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
)

func TestGetStatementWithResultLimit(t *testing.T) {
	runLimitTest(t, "test_limit.yaml", false /* record */)
}

func TestAddLimitFor12cAndLaterUsesOmni(t *testing.T) {
	stmt := "SELECT emp_id, salary, ROW_NUMBER() OVER (ORDER BY salary DESC) AS rn FROM employees QUALIFY rn <= 10"

	got := addLimitFor12cAndLater(stmt, 5)

	require.Equal(t, stmt+" FETCH NEXT 5 ROWS ONLY", got)
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
			got := addResultLimit(tc.statement, 10, tc.engineVersion)

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
			got := addResultLimit(tc.statement, 10, "19.0.0.0.0")

			require.Equal(t, tc.want, got)
		})
	}
}

func TestAddLimitFor12cAndLaterRegressionClauseOrder(t *testing.T) {
	tests := []struct {
		name           string
		statement      string
		limit          int
		want           string
		clausesInOrder []string
		validateParse  bool
	}{
		{
			name:           "outer order by stays before fetch",
			statement:      "SELECT id FROM employees ORDER BY id",
			limit:          10,
			want:           "SELECT id FROM employees ORDER BY id FETCH NEXT 10 ROWS ONLY",
			clausesInOrder: []string{"ORDER BY", "FETCH NEXT 10 ROWS ONLY"},
			validateParse:  true,
		},
		{
			name:           "with query outer order by",
			statement:      "WITH active_employees AS (SELECT id FROM employees WHERE active = 1) SELECT id FROM active_employees ORDER BY id",
			limit:          10,
			want:           "WITH active_employees AS (SELECT id FROM employees WHERE active = 1) SELECT id FROM active_employees ORDER BY id FETCH NEXT 10 ROWS ONLY",
			clausesInOrder: []string{"WITH", "ORDER BY", "FETCH NEXT 10 ROWS ONLY"},
			validateParse:  true,
		},
		{
			name:           "subquery order by is not treated as outer order by",
			statement:      "SELECT * FROM (SELECT id FROM employees ORDER BY id) ordered_employees",
			limit:          10,
			want:           "SELECT * FROM (SELECT id FROM employees ORDER BY id) ordered_employees FETCH NEXT 10 ROWS ONLY",
			clausesInOrder: []string{"ordered_employees", "FETCH NEXT 10 ROWS ONLY"},
			validateParse:  true,
		},
		// Oracle expects the row limiting clause before FOR UPDATE. Omni currently
		// cannot parse the rewritten statement, so these cases assert clause order
		// without validateParse to avoid regressing back to parser-friendly but invalid SQL.
		{
			name:           "for update remains the final clause",
			statement:      "SELECT * FROM employees FOR UPDATE",
			limit:          10,
			want:           "SELECT * FROM employees FETCH NEXT 10 ROWS ONLY FOR UPDATE",
			clausesInOrder: []string{"FETCH NEXT 10 ROWS ONLY", "FOR UPDATE"},
		},
		{
			name:           "order by with for update skip locked",
			statement:      "SELECT * FROM employees ORDER BY id FOR UPDATE SKIP LOCKED",
			limit:          10,
			want:           "SELECT * FROM employees ORDER BY id FETCH NEXT 10 ROWS ONLY FOR UPDATE SKIP LOCKED",
			clausesInOrder: []string{"ORDER BY", "FETCH NEXT 10 ROWS ONLY", "FOR UPDATE SKIP LOCKED"},
		},
		{
			name:           "union outer order by stays before fetch",
			statement:      "SELECT id FROM employees UNION ALL SELECT id FROM contractors ORDER BY id",
			limit:          10,
			want:           "SELECT id FROM employees UNION ALL SELECT id FROM contractors ORDER BY id FETCH NEXT 10 ROWS ONLY",
			clausesInOrder: []string{"UNION ALL", "ORDER BY", "FETCH NEXT 10 ROWS ONLY"},
			validateParse:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := addLimitFor12cAndLater(tc.statement, tc.limit)

			require.Equal(t, tc.want, got)
			assertSubstringsInOrder(t, got, tc.clausesInOrder)
			if tc.validateParse {
				_, err := plsqlparser.ParsePLSQLOmni(got)
				require.NoError(t, err, "rewritten SQL should remain parseable: %s", got)
			}
		})
	}
}

func TestAddFetchNextClauseRegressionExistingFetch(t *testing.T) {
	tests := []struct {
		name      string
		statement string
		limit     int
		want      string
		wantErr   bool
	}{
		{
			name:      "keeps lower existing fetch",
			statement: "SELECT * FROM employees FETCH NEXT 5 ROWS ONLY",
			limit:     10,
			want:      "SELECT * FROM employees FETCH NEXT 5 ROWS ONLY",
		},
		{
			name:      "lowers higher existing fetch",
			statement: "SELECT * FROM employees FETCH NEXT 15 ROWS ONLY",
			limit:     10,
			want:      "SELECT * FROM employees FETCH NEXT 10 ROWS ONLY",
		},
		{
			name:      "offset only adds fetch",
			statement: "SELECT * FROM employees ORDER BY id OFFSET 20 ROWS",
			limit:     10,
			want:      "SELECT * FROM employees ORDER BY id OFFSET 20 ROWS FETCH NEXT 10 ROWS ONLY",
		},
		{
			name:      "offset expression containing fetch still adds fetch",
			statement: "SELECT * FROM employees ORDER BY id OFFSET LENGTH('FETCH') ROWS",
			limit:     10,
			want:      "SELECT * FROM employees ORDER BY id OFFSET LENGTH('FETCH') ROWS FETCH NEXT 10 ROWS ONLY",
		},
		{
			name:      "non-constant fetch expression is not rewritten inline",
			statement: "SELECT * FROM employees FETCH NEXT :row_count ROWS ONLY",
			limit:     10,
			wantErr:   true,
		},
		{
			name:      "percent fetch expression is not a row count",
			statement: "SELECT * FROM employees FETCH NEXT 10 PERCENT ROWS ONLY",
			limit:     5,
			wantErr:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := addFetchNextClause(tc.statement, tc.limit)
			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestAddLimitFor12cAndLaterFallsBackForPercentFetch(t *testing.T) {
	stmt := "SELECT * FROM employees FETCH NEXT 10 PERCENT ROWS ONLY"

	got := addLimitFor12cAndLater(stmt, 5)

	require.Equal(t, "SELECT * FROM (SELECT * FROM employees FETCH NEXT 10 PERCENT ROWS ONLY) WHERE ROWNUM <= 5", got)
}

func assertSubstringsInOrder(t *testing.T, s string, substrings []string) {
	t.Helper()

	start := 0
	for _, substring := range substrings {
		index := strings.Index(s[start:], substring)
		require.NotEqualf(t, -1, index, "expected %q after offset %d in %q", substring, start, s)
		start += index + len(substring)
	}
}

type limitTestData struct {
	Stmt  string `yaml:"stmt"`
	Limit int    `yaml:"limit"`
	Want  string `yaml:"want"`
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
		want := addLimitFor12cAndLater(tc.Stmt, tc.Limit)
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
