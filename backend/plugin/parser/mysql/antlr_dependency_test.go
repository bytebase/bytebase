package mysql

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMySQLOmniASTDoesNotFallbackToANTLR(t *testing.T) {
	content, err := os.ReadFile("omni.go")
	require.NoError(t, err)
	source := string(content)

	require.NotContains(t, source, "AsANTLRAST")
	require.NotContains(t, source, "AntlrASTProvider")
	require.NotContains(t, source, "base.ANTLRAST")
	require.NotContains(t, source, "antlrAST")
	require.NotContains(t, source, "parseSingleStatementLenient")
}
