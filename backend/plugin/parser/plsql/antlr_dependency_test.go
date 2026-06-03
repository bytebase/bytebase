package plsql

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOracleRuntimeEntrypointsDoNotFallbackToANTLR(t *testing.T) {
	files := []string{
		"diagnose.go",
		"omni.go",
		"plsql.go",
		"split.go",
		"statement_type.go",
	}
	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			content, err := os.ReadFile(file)
			require.NoError(t, err)
			source := string(content)

			require.NotContains(t, source, "github.com/antlr4-go/antlr/v4")
			require.NotContains(t, source, "github.com/bytebase/parser/plsql")
			require.NotContains(t, source, "ParsePLSQL(")
			require.NotContains(t, source, "ParsePLSQLForStringsManipulation")
			require.NotContains(t, source, "AsANTLRAST")
			require.NotContains(t, source, "GetANTLRAST")
		})
	}
}
