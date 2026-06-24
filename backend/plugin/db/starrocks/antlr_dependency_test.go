package starrocks

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStarRocksQueryDoesNotUseANTLROrMySQLParser(t *testing.T) {
	content, err := os.ReadFile("query.go")
	require.NoError(t, err)
	source := string(content)

	require.NotContains(t, source, "github.com/antlr4-go/antlr/v4")
	require.NotContains(t, source, "github.com/bytebase/parser/mysql")
	require.NotContains(t, source, "ParseMySQL(")
	require.NotContains(t, source, "TokenStreamRewriter")
	require.NotContains(t, source, "BaseMySQLParserListener")
	require.NotContains(t, source, "omni/mysql/ast")
	require.NotContains(t, source, "omni/mysql/parser")
	require.NotContains(t, source, "plugin/parser/mysql")
}
