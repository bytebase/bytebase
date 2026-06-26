package tidb

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTiDBExecuteDoesNotUseMySQLParserPackage(t *testing.T) {
	content, err := os.ReadFile("tidb.go")
	require.NoError(t, err)
	source := string(content)

	require.NotContains(t, source, "github.com/antlr4-go/antlr/v4")
	require.NotContains(t, source, "github.com/bytebase/parser/mysql")
	require.NotContains(t, source, "plugin/parser/mysql\"")
	require.NotContains(t, source, "mysqlutil")
	require.NotContains(t, source, "DealWithDelimiter")
}
