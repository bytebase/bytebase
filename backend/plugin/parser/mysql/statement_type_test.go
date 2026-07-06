package mysql

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common/yamltest"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

type statementTypeTest struct {
	Statement string
	Want      []string
}

func TestGetStatementType(t *testing.T) {
	tests := []statementTypeTest{}

	const (
		record = false
	)

	var (
		filepath = "test-data/test_statement_type.yaml"
	)

	a := require.New(t)
	yamlFile, err := os.Open(filepath)
	a.NoError(err)

	byteValue, err := io.ReadAll(yamlFile)
	a.NoError(yamlFile.Close())
	a.NoError(err)
	a.NoError(yaml.Unmarshal(byteValue, &tests))

	for i, test := range tests {
		stmts, err := base.ParseStatements(storepb.Engine_MYSQL, test.Statement)
		a.NoError(err)
		asts := base.ExtractASTs(stmts)

		sqlType, err := GetStatementTypes(asts)
		a.NoError(err)

		// Convert enum to string for comparison
		sqlTypeStrings := make([]string, len(sqlType))
		for j, t := range sqlType {
			sqlTypeStrings[j] = t.String()
		}

		if record {
			tests[i].Want = sqlTypeStrings
		} else {
			a.Equal(test.Want, sqlTypeStrings)
		}
	}

	if record {
		yamltest.Record(t, filepath, tests)
	}
}

// TestGetStatementTypesWithPositionsSETNarrowing pins the SDL-gate narrowing (BYT-9832 P2):
// GetStatementTypesWithPositions feeds ONLY the declarative-release gate, so it downgrades any
// SET that is not the dumper's session-context framing back to STATEMENT_TYPE_UNSPECIFIED (so
// the gate rejects it fail-closed). The general GetStatementTypes path is unaffected and still
// classifies every SET as StatementType_SET — asserted at the end.
func TestGetStatementTypesWithPositionsSETNarrowing(t *testing.T) {
	cases := []struct {
		name string
		sql  string
		want storepb.StatementType
	}{
		// Allowed — exactly the forms writeSDLSessionContextPrefix emits.
		{"save sql_mode", "SET @saved_sql_mode = @@sql_mode;", storepb.StatementType_SET},
		{"set sql_mode literal", "SET sql_mode = 'ANSI_QUOTES';", storepb.StatementType_SET},
		{"restore sql_mode", "SET sql_mode = @saved_sql_mode;", storepb.StatementType_SET},
		{"save time_zone", "SET @saved_time_zone = @@time_zone;", storepb.StatementType_SET},
		{"set time_zone literal", "SET time_zone = '+05:30';", storepb.StatementType_SET},
		{"restore time_zone", "SET time_zone = @saved_time_zone;", storepb.StatementType_SET},
		{"explicit SESSION sql_mode", "SET SESSION sql_mode = 'ANSI_QUOTES';", storepb.StatementType_SET},
		{"case-insensitive var", "SET SQL_MODE = 'ANSI_QUOTES';", storepb.StatementType_SET},

		// Disallowed — user-authored / non-declarative SET must fail closed (UNSPECIFIED).
		{"global scope", "SET GLOBAL max_connections = 1;", storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED},
		{"persist scope", "SET PERSIST sql_mode = 'ANSI_QUOTES';", storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED},
		{"persist_only scope", "SET PERSIST_ONLY sql_mode = 'ANSI_QUOTES';", storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED},
		{"foreign_key_checks", "SET FOREIGN_KEY_CHECKS = 0;", storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED},
		{"session other var", "SET SESSION unique_checks = 0;", storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED},
		{"autocommit", "SET autocommit = 0;", storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED},
		{"explicit @@GLOBAL sql_mode", "SET @@GLOBAL.sql_mode = 'ANSI_QUOTES';", storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED},
		{"set names", "SET NAMES utf8mb4;", storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED},
		{"mixed allowed+disallowed", "SET sql_mode = 'ANSI_QUOTES', FOREIGN_KEY_CHECKS = 0;", storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := require.New(t)
			stmts, err := base.ParseStatements(storepb.Engine_MYSQL, tc.sql)
			a.NoError(err, tc.sql)
			got, err := GetStatementTypesWithPositions(base.ExtractASTs(stmts))
			a.NoError(err)
			a.Len(got, 1, tc.sql)
			a.Equal(tc.want, got[0].Type, "gate narrowing for %q", tc.sql)

			// The general classifier path is unchanged: every SET is StatementType_SET.
			general, err := GetStatementTypes(base.ExtractASTs(stmts))
			a.NoError(err)
			a.Equal([]storepb.StatementType{storepb.StatementType_SET}, general, "general classifier must keep %q as SET", tc.sql)
		})
	}
}
