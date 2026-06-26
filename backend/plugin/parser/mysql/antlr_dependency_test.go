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

func TestMySQLDelimiterHandlingDoesNotUseLegacyTokenizer(t *testing.T) {
	content, err := os.ReadFile("mysql.go")
	require.NoError(t, err)
	source := string(content)

	require.NotContains(t, source, "github.com/antlr4-go/antlr/v4")
	require.NotContains(t, source, "github.com/bytebase/parser/mysql")
	require.NotContains(t, source, "base.ANTLRAST")
	require.NotContains(t, source, "ParseMySQL(")
	require.NotContains(t, source, "parseSingleStatement")
	require.NotContains(t, source, "plugin/parser/tokenizer")
	require.NotContains(t, source, "SplitTiDBMultiSQL")
	require.NotContains(t, source, "mysqlutil")
	require.NotContains(t, source, "DealWithDelimiter")
}
