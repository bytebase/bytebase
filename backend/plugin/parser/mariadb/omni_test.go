package mariadb

import (
	"testing"

	"github.com/stretchr/testify/require"

	mysqlomniparser "github.com/bytebase/omni/mysql/parser"
)

// TestOmniMariaDBRejectsMySQLOnlySyntax pins the statements omni's MySQL parser
// accepts and omni's MariaDB parser refuses. Each line is deliberate: the
// engine has no such syntax, so Bytebase now agrees with the server it talks to
// instead of admitting the MySQL form.
func TestOmniMariaDBRejectsMySQLOnlySyntax(t *testing.T) {
	for _, tc := range []struct{ name, statement string }{
		{"TABLE statement", "TABLE t;"},
		{"INSERT ... TABLE", "INSERT INTO t TABLE s;"},
		{"VALUES ROW constructor", "SELECT * FROM (VALUES ROW(1), ROW(2)) AS v;"},
		{"CAST AS ARRAY", "SELECT CAST(j AS JSON ARRAY) FROM t;"},
		{"INSERT row alias", "INSERT INTO t VALUES (1) AS new ON DUPLICATE KEY UPDATE a = new.a;"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, mysqlErr := mysqlomniparser.Parse(tc.statement)
			require.NoError(t, mysqlErr)
			_, err := ParseMariaDB(tc.statement)
			require.Error(t, err)
		})
	}
}

// TestOmniMariaDBParserGaps pins syntax a real MariaDB server accepts and omni's
// MariaDB parser does not, so the list cannot grow unnoticed and shrinks loudly
// when omni closes a gap. These are omni bugs, not decisions.
func TestOmniMariaDBParserGaps(t *testing.T) {
	for _, tc := range []struct{ name, statement string }{
		{"WEIGHT_STRING with AS", "SELECT WEIGHT_STRING(c AS CHAR(4)) FROM t;"},
		{"WEIGHT_STRING with LEVEL", "SELECT WEIGHT_STRING(c LEVEL 1) FROM t;"},
		{"adjacent literals in a default", "CREATE TABLE t (c VARCHAR(8) DEFAULT 'a' 'b');"},
		{"adjacent literals in a predicate", "SELECT * FROM t WHERE c = 'a' 'b';"},
		{"adjacent literals in a row", "INSERT INTO t VALUES ('a' 'b');"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, mysqlErr := mysqlomniparser.Parse(tc.statement)
			require.NoError(t, mysqlErr)
			_, err := ParseMariaDB(tc.statement)
			require.Error(t, err, "omni/mariadb now accepts this; drop the case")
		})
	}
}

// TestOmniMariaDBLexerAcceptsUnterminatedLiterals pins the other direction:
// omni/mysql's lexer reports an unterminated string, comment or quoted
// identifier and omni/mariadb's does not, so a statement the server refuses
// passes Bytebase's syntax check. Pre-existing — MariaDB already reached this
// parser for anything omni's MySQL parser rejected.
func TestOmniMariaDBLexerAcceptsUnterminatedLiterals(t *testing.T) {
	for _, tc := range []struct{ name, statement string }{
		{"string", "SELECT 'oops;"},
		{"block comment", "SELECT 1 /* oops;"},
		{"quoted identifier", "SELECT `oops;"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, mysqlErr := mysqlomniparser.Parse(tc.statement)
			require.Error(t, mysqlErr)
			_, err := ParseMariaDB(tc.statement)
			require.NoError(t, err, "omni/mariadb now rejects this; drop the case")
		})
	}
}
