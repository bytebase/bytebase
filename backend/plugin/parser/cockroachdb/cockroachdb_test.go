package cockroachdb

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"

	// Registers the statement splitter that CockroachDB parsing uses.
	_ "github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

func TestParseSeam(t *testing.T) {
	// Registry dispatch combines PG splitter positions with CockroachDB ASTs.
	stmts, err := base.ParseStatements(storepb.Engine_COCKROACHDB, "CREATE TABLE t (id INT); ; UPSERT INTO t VALUES (1);")
	require.NoError(t, err)
	asts := base.ExtractASTs(stmts)
	require.Len(t, asts, 2)
	for _, a := range asts {
		_, ok := a.(*AST)
		require.True(t, ok)
		require.Equal(t, int32(1), a.ASTStartPosition().Line)
	}

	_, err = base.ParseStatements(storepb.Engine_COCKROACHDB, "SELECT FROM")
	require.Error(t, err)
}
