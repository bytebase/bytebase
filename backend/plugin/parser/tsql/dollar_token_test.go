package tsql

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// TestDollarTokenStatements validates the omni bump for $-prefixed T-SQL
// tokens (BYT-9813) through Bytebase's own entry points: base.ParseStatements
// is what the issue SQL review pipeline calls, so these pins guard the exact
// path that produced the customer-facing syntax error.
func TestDollarTokenStatements(t *testing.T) {
	accept := []string{
		// The MERGE OUTPUT $action shape that motivated BYT-9813.
		`MERGE INTO dst AS d
USING src AS s ON d.k = s.k
WHEN MATCHED THEN UPDATE SET d.v = s.v
WHEN NOT MATCHED THEN INSERT (k, v) VALUES (s.k, s.v)
OUTPUT $action, INSERTED.k, INSERTED.v;`,
		// Pseudo-columns, bare and qualified.
		"SELECT $IDENTITY, $ROWGUID FROM t;",
		"SELECT t.$IDENTITY FROM t;",
		"SELECT $node_id FROM Person;",
		// Money constants, including sign/space and non-$ symbols.
		"SELECT $12.50, -$4.78, $ 4, £10;",
		// Partition function, bare and database-qualified.
		"SELECT $PARTITION.pf1(10);",
		"SELECT db1.$PARTITION.pf1(10);",
		// Graph edge INSERT and UPDATE SET targets.
		"INSERT INTO e ($from_id, $to_id) VALUES ('a', 'b');",
		"UPDATE e SET $from_id = 'x';",
		// IDENTITYCOL / ROWGUIDCOL keyword column refs.
		"SELECT IDENTITYCOL, ROWGUIDCOL FROM t;",
	}
	for _, sql := range accept {
		t.Run(sql, func(t *testing.T) {
			_, err := base.ParseStatements(storepb.Engine_MSSQL, sql)
			require.NoError(t, err, "SQL review entry must accept %q", sql)

			_, err = ParseTSQLOmni(sql)
			require.NoError(t, err, "ParseTSQLOmni must accept %q", sql)
		})
	}

	// Unknown pseudo-columns stay rejected (engine: Msg 126).
	reject := []string{
		"SELECT $foo FROM t;",
		"SELECT $CUID FROM t;",
	}
	for _, sql := range reject {
		t.Run(sql, func(t *testing.T) {
			_, err := base.ParseStatements(storepb.Engine_MSSQL, sql)
			require.Error(t, err, "SQL review entry must reject %q", sql)
		})
	}
}
