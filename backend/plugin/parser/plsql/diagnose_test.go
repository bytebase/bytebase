package plsql

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestDiagnoseUsesOmniParseErrorPosition(t *testing.T) {
	tests := []struct {
		name      string
		statement string
		line      uint32
	}{
		{
			name:      "single line",
			statement: "SELECT * FROM",
			line:      0,
		},
		{
			name:      "second sql statement",
			statement: "SELECT 1 FROM DUAL;\n\nSELECT * FROM",
			line:      2,
		},
		{
			name:      "line after non-ascii",
			statement: "SELECT '你好' FROM DUAL;\nSELECT * FROM",
			line:      1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			diagnostics, err := Diagnose(context.Background(), base.DiagnoseContext{}, tc.statement)
			require.NoError(t, err)
			require.Len(t, diagnostics, 1)
			require.Equal(t, tc.line, diagnostics[0].Range.Start.Line)
		})
	}
}

func TestDiagnoseFallsBackToANTLR(t *testing.T) {
	statement := `CREATE OR REPLACE TRIGGER trg
BEFORE INSERT OR UPDATE OF col1, col2 ON tbl
REFERENCING OLD AS o NEW AS n
FOR EACH ROW
WHEN (n.col1 > 0)
BEGIN
  :n.col2 := :o.col2 + 1;
END;`

	diagnostics, err := Diagnose(context.Background(), base.DiagnoseContext{}, statement)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
}

func TestDiagnoseAcceptsAlterSystem(t *testing.T) {
	tests := []string{
		"alter system set dg_broker_start=true;",
		"alter system switch logfile;",
	}

	for _, statement := range tests {
		t.Run(statement, func(t *testing.T) {
			diagnostics, err := Diagnose(context.Background(), base.DiagnoseContext{}, statement)
			require.NoError(t, err)
			require.Empty(t, diagnostics)

			stmts, err := base.ParseStatements(store.Engine_ORACLE, statement)
			require.NoError(t, err)
			require.Len(t, stmts, 1)
		})
	}
}
