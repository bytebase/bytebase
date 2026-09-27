package oracle

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
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

func TestOracleSplitKeepsTrailingFragmentForDatabaseExecution(t *testing.T) {
	commands, err := plsql.SplitSQL("DROP TABLESPACE xxx; CASCADE")
	require.NoError(t, err)
	commands = base.FilterEmptyStatements(commands)
	require.Len(t, commands, 2)
	require.Equal(t, "DROP TABLESPACE xxx", commands[0].Text)
	require.Equal(t, " CASCADE", commands[1].Text)
}

func TestOracleSplitAllowsParserUnsupportedDDL(t *testing.T) {
	statement := `SET DEFINE OFF
CREATE VECTOR INDEX vec_idx ON docs (embedding);
CREATE JSON RELATIONAL DUALITY VIEW emp_dv AS SELECT employee_id FROM employees;
CREATE INDEX IDX_SALES_MONTH_YEAR ON SALES_DATA(EXTRACT(YEAR FROM SALE_DATE), EXTRACT(MONTH FROM SALE_DATE));
CREATE SEQUENCE order_seq START WITH 1 INCREMENT BY 1;
CREATE TABLE employees (salary NUMBER CHECK (salary > 0 OR salary IS NULL));
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
	require.Len(t, commands, 9)
}

func TestBuildExecuteCommandsSplitsLargeOracleScript(t *testing.T) {
	var builder strings.Builder
	builder.WriteString(`CREATE TABLE APPS.M_SKU_STOCK_KEY_TMP (
  SKU_CODE VARCHAR2(100) NOT NULL,
  STOCK_KEY VARCHAR2(100)
);
`)
	builder.WriteString("CREATE INDEX IDX_TMP_SKU_CODE ON APPS.M_SKU_STOCK_KEY_TMP(SKU_CODE);\n")
	for i := 0; builder.Len() <= common.MaxSheetCheckSize+1024; i++ {
		_, err := fmt.Fprintf(&builder, "INSERT INTO APPS.M_SKU_STOCK_KEY_TMP(SKU_CODE, STOCK_KEY) VALUES('SKU_%06d', '%s');\n", i, strings.Repeat("x", 64))
		require.NoError(t, err)
	}
	statement := builder.String()
	require.Greater(t, len(statement), common.MaxSheetCheckSize)

	commands, err := buildExecuteCommands(statement)
	require.NoError(t, err)

	require.Greater(t, len(commands), 2)
	require.NotEqual(t, statement, commands[0].Text)
	require.Equal(t, "CREATE TABLE APPS.M_SKU_STOCK_KEY_TMP (\n  SKU_CODE VARCHAR2(100) NOT NULL,\n  STOCK_KEY VARCHAR2(100)\n)", strings.TrimSpace(commands[0].Text))
	require.Equal(t, "CREATE INDEX IDX_TMP_SKU_CODE ON APPS.M_SKU_STOCK_KEY_TMP(SKU_CODE)", strings.TrimSpace(commands[1].Text))
}
