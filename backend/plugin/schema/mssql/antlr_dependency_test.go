package mssql

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMSSQLSchemaMetadataDoesNotDependOnANTLR(t *testing.T) {
	forbidden := []string{
		"github.com/antlr4-go/antlr/v4",
		"github.com/bytebase/parser/tsql",
		"getDatabaseMetadataANTLR",
		"BaseTSqlParserListener",
	}

	files, err := filepath.Glob("*.go")
	require.NoError(t, err)
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}

		source, err := os.ReadFile(path)
		require.NoError(t, err)
		for _, token := range forbidden {
			require.NotContains(t, string(source), token, path)
		}
	}
}
