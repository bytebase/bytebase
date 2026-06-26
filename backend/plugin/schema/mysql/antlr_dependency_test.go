package mysql

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMySQLSchemaDoesNotKeepLegacyANTLRFallbacks(t *testing.T) {
	for _, name := range []string{
		"get_database_metadata.go",
		"walk_through.go",
	} {
		_, err := os.Stat(name)
		require.ErrorIs(t, err, os.ErrNotExist)
	}

	files, err := filepath.Glob("*.go")
	require.NoError(t, err)
	for _, file := range files {
		if filepath.Base(file) == "antlr_dependency_test.go" {
			continue
		}
		content, err := os.ReadFile(file)
		require.NoError(t, err)
		source := string(content)

		require.NotContains(t, source, "github.com/antlr4-go/antlr/v4", file)
		require.NotContains(t, source, "github.com/bytebase/parser/mysql", file)
		require.NotContains(t, source, "base.ANTLRAST", file)
		require.NotContains(t, source, "GetANTLRAST", file)
		require.NotContains(t, source, "ParseMySQL(", file)
		require.NotContains(t, source, "NormalizeMySQL", file)
		require.NotContains(t, source, "IsTopMySQLRule", file)
	}
}
