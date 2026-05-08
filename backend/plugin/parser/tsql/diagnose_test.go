package tsql

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestDiagnoseUsesOmniParseError(t *testing.T) {
	tests := []struct {
		name      string
		statement string
		message   string
		line      uint32
		character uint32
	}{
		{
			name:      "single line",
			statement: "SELECT FROM",
			message:   `syntax error at or near "FROM"`,
			line:      0,
			character: 7,
		},
		{
			name:      "second line",
			statement: "SELECT 1;\nSELECT FROM",
			message:   `syntax error at or near "FROM"`,
			line:      1,
			character: 7,
		},
		{
			name:      "line after non-ascii",
			statement: "SELECT N'你好';\nSELECT FROM",
			message:   `syntax error at or near "FROM"`,
			line:      1,
			character: 7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diagnostics, err := Diagnose(context.Background(), base.DiagnoseContext{}, tt.statement)
			require.NoError(t, err)
			require.Len(t, diagnostics, 1)
			require.Equal(t, tt.message, diagnostics[0].Message)
			require.Equal(t, tt.line, diagnostics[0].Range.Start.Line)
			require.Equal(t, tt.character, diagnostics[0].Range.Start.Character)
		})
	}
}
