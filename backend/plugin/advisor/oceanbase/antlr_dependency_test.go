package oceanbase

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOceanBaseAdvisorsDoNotUseMySQLANTLRBridge(t *testing.T) {
	files := []string{
		"advisor_disallow_offline_ddl.go",
		"advisor_insert_row_limit.go",
		"advisor_statement_affected_row_limit.go",
	}
	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			content, err := os.ReadFile(file)
			require.NoError(t, err)
			source := string(content)

			require.NotContains(t, source, "github.com/antlr4-go/antlr/v4")
			require.NotContains(t, source, "github.com/bytebase/parser/mysql")
			require.NotContains(t, source, "base.GetANTLRAST")
			require.NotContains(t, source, "BaseMySQLParserListener")
		})
	}
}
