package plsql

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

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
