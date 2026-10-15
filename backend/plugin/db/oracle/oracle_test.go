package oracle

import (
	"fmt"
	"maps"
	"strings"
	"testing"
	"time"

	goora "github.com/sijms/go-ora/v2"
	"github.com/sijms/go-ora/v2/configurations"
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

func TestConnectionOptions(t *testing.T) {
	tests := []struct {
		name            string
		sid             string
		extraParameters map[string]string
		want            map[string]string
	}{
		{
			name: "no extra parameters",
			want: map[string]string{"CONNECTION TIMEOUT": "10"},
		},
		{
			name: "SID field",
			sid:  "ORCL",
			want: map[string]string{"CONNECTION TIMEOUT": "10", "SID": "ORCL"},
		},
		{
			name:            "dial timeout in the exact spelling",
			extraParameters: map[string]string{"CONNECTION TIMEOUT": "5"},
			want:            map[string]string{"CONNECTION TIMEOUT": "5"},
		},
		{
			name:            "dial timeout in lowercase",
			extraParameters: map[string]string{"connection timeout": "5"},
			want:            map[string]string{"connection timeout": "5"},
		},
		{
			name:            "dial timeout alias",
			extraParameters: map[string]string{"CONNECT TIMEOUT": "5"},
			want:            map[string]string{"CONNECT TIMEOUT": "5"},
		},
		{
			name:            "dial timeout alias in mixed case",
			extraParameters: map[string]string{"Connect Timeout": "5"},
			want:            map[string]string{"Connect Timeout": "5"},
		},
		{
			name:            "dial timeout turned off",
			extraParameters: map[string]string{"CONNECTION TIMEOUT": "0"},
			want:            map[string]string{"CONNECTION TIMEOUT": "0"},
		},
		{
			name:            "read timeout is a different option",
			extraParameters: map[string]string{"TIMEOUT": "30"},
			want:            map[string]string{"TIMEOUT": "30", "CONNECTION TIMEOUT": "10"},
		},
		{
			name:            "SID key in the exact spelling",
			sid:             "ORCL",
			extraParameters: map[string]string{"SID": "STANDBY"},
			want:            map[string]string{"CONNECTION TIMEOUT": "10", "SID": "STANDBY"},
		},
		{
			name:            "SID key in lowercase",
			sid:             "ORCL",
			extraParameters: map[string]string{"sid": "STANDBY"},
			want:            map[string]string{"CONNECTION TIMEOUT": "10", "sid": "STANDBY"},
		},
		{
			name:            "SID key without the SID field",
			extraParameters: map[string]string{"Sid": "STANDBY"},
			want:            map[string]string{"CONNECTION TIMEOUT": "10", "Sid": "STANDBY"},
		},
		{
			name: "other keys pass through in their own spelling",
			sid:  "ORCL",
			extraParameters: map[string]string{
				"SERVER":          "standby-1.example.com:1521, standby-2.example.com:1521",
				"SSL":             "true",
				"ssl verify":      "false",
				"WALLET":          "/opt/oracle/wallet",
				"WALLET PASSWORD": "wallet-secret",
				"AUTH TYPE":       "TCPS",
				"prefetch_rows":   "500",
			},
			want: map[string]string{
				"CONNECTION TIMEOUT": "10",
				"SID":                "ORCL",
				"SERVER":             "standby-1.example.com:1521, standby-2.example.com:1521",
				"SSL":                "true",
				"ssl verify":         "false",
				"WALLET":             "/opt/oracle/wallet",
				"WALLET PASSWORD":    "wallet-secret",
				"AUTH TYPE":          "TCPS",
				"prefetch_rows":      "500",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			extraParameters := maps.Clone(test.extraParameters)
			got := connectionOptions(test.sid, extraParameters)
			require.Equal(t, test.want, got)
			require.Equal(t, test.extraParameters, extraParameters)
		})
	}
}

// go-ora reads the options back from the URL, so these cases check what the driver ends up
// with rather than what Bytebase sent.
func TestConnectionOptionsAsReadByDriver(t *testing.T) {
	parse := func(t *testing.T, sid string, extraParameters map[string]string) *configurations.ConnectionConfig {
		t.Helper()
		dsn := goora.BuildUrl("primary.example.com", 1521, "ORCLPDB1", "bytebase", "secret", connectionOptions(sid, extraParameters))
		config, err := configurations.ParseConfig(dsn)
		require.NoError(t, err)
		return config
	}

	t.Run("default", func(t *testing.T) {
		config := parse(t, "", nil)
		require.Equal(t, 10*time.Second, config.ConnectTimeout)
		// A read timeout would cut off long statements.
		require.Zero(t, config.Timeout)
		require.Empty(t, config.SID)
	})

	for _, key := range []string{"CONNECTION TIMEOUT", "connection timeout", "CONNECT TIMEOUT", "connect timeout", "Connect Timeout"} {
		t.Run("dial timeout as "+key, func(t *testing.T) {
			config := parse(t, "", map[string]string{key: "5"})
			require.Equal(t, 5*time.Second, config.ConnectTimeout)
			require.Zero(t, config.Timeout)
		})
	}

	t.Run("dial timeout turned off", func(t *testing.T) {
		config := parse(t, "", map[string]string{"connect timeout": "0"})
		require.Zero(t, config.ConnectTimeout)
	})

	t.Run("SID field", func(t *testing.T) {
		config := parse(t, "ORCL", nil)
		require.Equal(t, "ORCL", config.SID)
	})

	t.Run("SID key replaces the SID field", func(t *testing.T) {
		config := parse(t, "ORCL", map[string]string{"sid": "STANDBY"})
		require.Equal(t, "STANDBY", config.SID)
	})

	t.Run("second address over TCPS", func(t *testing.T) {
		config := parse(t, "", map[string]string{
			"SERVER":     "standby-1.example.com:2484, standby-2.example.com:2484",
			"SSL":        "true",
			"SSL VERIFY": "false",
			"AUTH TYPE":  "TCPS",
		})
		require.Equal(t, []configurations.ServerAddr{
			{Addr: "primary.example.com", Port: 1521},
			{Addr: "standby-1.example.com", Port: 2484},
			{Addr: "standby-2.example.com", Port: 2484},
		}, config.Servers)
		require.True(t, config.SSL)
		require.False(t, config.SSLVerify)
		require.Equal(t, configurations.TCPS, config.AuthType)
		require.Equal(t, 10*time.Second, config.ConnectTimeout)
	})
}
