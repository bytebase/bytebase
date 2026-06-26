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

func TestStarRocksDriverDoesNotUseANTLR(t *testing.T) {
	content, err := os.ReadFile("starrocks.go")
	require.NoError(t, err)
	source := string(content)

	require.NotContains(t, source, "github.com/antlr4-go/antlr/v4")
	require.NotContains(t, source, "github.com/bytebase/parser/mysql")
	require.NotContains(t, source, "plugin/parser/mysql")
}

func TestContainsDelimiterDirective(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		want bool
	}{
		{"basic directive", "DELIMITER ;;\nCREATE PROCEDURE p() BEGIN SELECT 1; END;;\nDELIMITER ;", true},
		{"indented directive", "  DELIMITER //\nSELECT 1//\nDELIMITER ;", true},
		{"tab-indented directive", "\tDELIMITER $$\n", true},
		{"string literal not a directive", "INSERT INTO t VALUES ('DELIMITER ');\nSELECT 1;", false},
		{"comment not a directive", "-- DELIMITER //\nSELECT 1;", false},
		{"column name not a directive", "SELECT delimiter FROM t;", false},
		{"no delimiter at all", "SELECT 1;\nSELECT 2;", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsDelimiterDirective(tt.sql)
			require.Equal(t, tt.want, got)
		})
	}
}
