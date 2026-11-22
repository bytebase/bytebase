package plsql

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestPLSQLSplitSQL(t *testing.T) {
	base.RunSplitTests(t, "test-data/test_split.yaml", base.SplitTestOptions{
		SplitFunc: SplitSQL,
	})
}

func TestPLSQLSplitSQLStoredUnitWithoutSlashSeparator(t *testing.T) {
	statement := `CREATE FUNCTION calc_bonus(p_start_date DATE)
RETURN DATE
IS
  v_current_date DATE := p_start_date;
BEGIN
  RETURN v_current_date;
END calc_bonus;
CREATE PROCEDURE update_salary(p_employee_id NUMBER)
IS
BEGIN
  UPDATE employees SET salary = salary + 1 WHERE id = p_employee_id;
END update_salary;
CREATE TABLE t(id NUMBER);`

	statements, err := SplitSQL(statement)
	require.NoError(t, err)
	statements = base.FilterEmptyStatements(statements)

	require.Len(t, statements, 3)
	require.Equal(t, `CREATE FUNCTION calc_bonus(p_start_date DATE)
RETURN DATE
IS
  v_current_date DATE := p_start_date;
BEGIN
  RETURN v_current_date;
END calc_bonus;`, statements[0].Text)
	require.Equal(t, `
CREATE PROCEDURE update_salary(p_employee_id NUMBER)
IS
BEGIN
  UPDATE employees SET salary = salary + 1 WHERE id = p_employee_id;
END update_salary;`, statements[1].Text)
	require.Equal(t, `
CREATE TABLE t(id NUMBER)`, statements[2].Text)
}

func TestPLSQLSplitSQLKeepsEmbeddedSourceWhole(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source string
		suffix string
	}{
		{
			name:   "Java source",
			source: "CREATE OR REPLACE AND COMPILE JAVA SOURCE NAMED \"Dec\" AS\npublic class Dec { public static int f(int n) { return n--; } }",
			suffix: "\n/\n",
		},
		{
			name:   "MLE module",
			source: "CREATE OR REPLACE MLE MODULE dec_module LANGUAGE JAVASCRIPT AS\nexport function f(n) { return n--; }",
			suffix: "\n/\n",
		},
		{
			name:   "inline MLE call spec",
			source: "CREATE OR REPLACE FUNCTION dec_js(n NUMBER) RETURN NUMBER\nAS MLE LANGUAGE JAVASCRIPT\n{{ return n--; }};",
			suffix: "\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			statements, err := SplitSQL(tc.source + tc.suffix + "SELECT 1 FROM dual;")
			require.NoError(t, err)
			statements = base.FilterEmptyStatements(statements)
			require.Len(t, statements, 2)
			require.Equal(t, tc.source, statements[0].Text)
			require.Equal(t, "SELECT 1 FROM dual", strings.TrimSpace(statements[1].Text))
		})
	}
}

func TestPLSQLSplitSQLPreservesSQLSetStatements(t *testing.T) {
	statement := `SET DEFINE OFF
SET TRANSACTION READ ONLY;
SET ROLE app_role;
SELECT 1 FROM dual;`

	statements, err := SplitSQL(statement)
	require.NoError(t, err)
	statements = base.FilterEmptyStatements(statements)

	require.Len(t, statements, 3)
	require.Equal(t, "SET TRANSACTION READ ONLY", statements[0].Text)
	require.Equal(t, "\nSET ROLE app_role", statements[1].Text)
	require.Equal(t, "\nSELECT 1 FROM dual", statements[2].Text)
}

func TestPLSQLSplitSQLLargeInsertScriptScalesLinearly(t *testing.T) {
	const rowCount = 2000
	padding := strings.Repeat("x", 1024)
	var builder strings.Builder
	for i := 0; i < rowCount; i++ {
		fmt.Fprintf(&builder, "INSERT INTO perf_omni_oracle (id, payload) VALUES (%d, '%s');\n", i, padding)
	}

	started := time.Now()
	statements, err := SplitSQL(builder.String())
	elapsed := time.Since(started)

	require.NoError(t, err)
	require.Len(t, statements, rowCount)
	require.Less(t, elapsed, time.Second)
	require.Equal(t, int32(1), statements[0].Start.Line)
	require.Equal(t, int32(rowCount-1), statements[rowCount-1].Start.Line)
	require.Equal(t, int32(rowCount), statements[rowCount-1].End.Line)
}
