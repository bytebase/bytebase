package redshift

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
			statement: "SELECT 1;\n\nSELECT * FROM",
			line:      2,
		},
		{
			name:      "line after non-ascii",
			statement: "SELECT '你好';\nSELECT * FROM",
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

func TestDiagnoseAcceptsRedshiftSyntax(t *testing.T) {
	tests := []string{
		"SELECT * FROM sales QUALIFY row_number() OVER () = 1;",
		"COPY sales FROM 's3://bucket/path' IAM_ROLE 'arn:aws:iam::123456789012:role/redshift';",
		"UNLOAD ('SELECT * FROM sales') TO 's3://bucket/out' IAM_ROLE 'arn:aws:iam::123456789012:role/redshift';",
	}

	for _, statement := range tests {
		t.Run(statement, func(t *testing.T) {
			diagnostics, err := Diagnose(context.Background(), base.DiagnoseContext{}, statement)
			require.NoError(t, err)
			require.Empty(t, diagnostics)
		})
	}
}
