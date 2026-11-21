package cockroachdb

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestGetStatementTypesSeam(t *testing.T) {
	const script = `CREATE TABLE t (id INT PRIMARY KEY);
UPSERT INTO t VALUES (1);
EXPLAIN ANALYZE DELETE FROM t WHERE id = 1;
ALTER INDEX t@idx RENAME TO idx2;`
	got, err := base.GetStatementTypes(storepb.Engine_COCKROACHDB, parseASTs(t, script))
	require.NoError(t, err)
	require.Equal(t, []storepb.StatementType{
		storepb.StatementType_CREATE_TABLE,
		storepb.StatementType_INSERT,
		storepb.StatementType_DELETE,
		storepb.StatementType_RENAME_INDEX,
	}, got)

	_, err = base.GetStatementTypes(storepb.Engine_COCKROACHDB, []base.AST{foreignAST{}})
	require.ErrorContains(t, err, "expected CockroachDB AST")
}
