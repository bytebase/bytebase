package plsql

import (
	"testing"

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
