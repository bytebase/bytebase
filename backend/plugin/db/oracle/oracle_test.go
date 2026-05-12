package oracle

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/plsql"
)

func TestParseVersion(t *testing.T) {
	type testData struct {
		Banner string
		First  int
		Second int
	}
	tests := []testData{
		{
			Banner: "12.1.0.2.0",
			First:  12,
			Second: 1,
		},
		{
			Banner: "12.1.0.",
			First:  12,
			Second: 1,
		},
	}

	for _, test := range tests {
		v, err := plsql.ParseVersion(test.Banner)
		require.NoError(t, err)
		require.Equal(t, test.First, v.First)
		require.Equal(t, test.Second, v.Second)
	}
}

func TestValidateOracleCommandsForExecutionRejectsInvalidFragment(t *testing.T) {
	commands, err := plsql.SplitSQL("DROP TABLESPACE xxx; CASCADE")
	require.NoError(t, err)
	commands = base.FilterEmptyStatements(commands)
	require.Len(t, commands, 2)

	err = validateOracleCommandsForExecution(commands)
	require.Error(t, err)
	require.Contains(t, err.Error(), "CASCADE")
}

func TestQueryConnRejectsInvalidFragmentBeforeExecution(t *testing.T) {
	driver := &Driver{}

	var err error
	require.NotPanics(t, func() {
		_, err = driver.QueryConn(context.Background(), nil, "DROP TABLESPACE xxx; CASCADE", db.QueryContext{})
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "CASCADE")
}

func TestValidateOracleCommandsForExecutionAllowsSplitOracleCommands(t *testing.T) {
	statement := `SET DEFINE OFF
CREATE VECTOR INDEX vec_idx ON docs (embedding);
CREATE JSON RELATIONAL DUALITY VIEW emp_dv AS SELECT employee_id FROM employees;
CREATE PACKAGE pkg IS
  PROCEDURE p;
END pkg;
CREATE PACKAGE BODY pkg IS
  PROCEDURE p IS
  BEGIN
    NULL;
  END p;
END pkg;
CREATE FUNCTION calc_bonus(p_start_date DATE)
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
END update_salary;`
	commands, err := plsql.SplitSQL(statement)
	require.NoError(t, err)
	commands = base.FilterEmptyStatements(commands)
	require.Len(t, commands, 6)

	require.NoError(t, validateOracleCommandsForExecution(commands))
}
