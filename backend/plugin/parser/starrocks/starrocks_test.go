package starrocks

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestStarRocksParserAndDiagnoseSeam(t *testing.T) {
	stmts, err := base.ParseStatements(storepb.Engine_STARROCKS, "SELECT * FROM schema1.t1 JOIN schema2.t2 ON t1.c1 = t2.c1")
	require.NoError(t, err)
	require.Len(t, stmts, 1)
	require.NotNil(t, stmts[0].AST)

	_, err = base.ParseStatements(storepb.Engine_STARROCKS, "SELECT a > (select max(a) from t1) FROM")
	var syntaxErr *base.SyntaxError
	require.ErrorAs(t, err, &syntaxErr)

	// StarRocks fork routing via registry: generated column syntax is accepted, invalid SQL reports diagnostics.
	diags, err := base.Diagnose(context.Background(), base.DiagnoseContext{}, storepb.Engine_STARROCKS, "CREATE TABLE t (a INT, b INT AS (a + 1)) DUPLICATE KEY(a) DISTRIBUTED BY HASH(a)")
	require.NoError(t, err)
	require.Empty(t, diags)

	diags, err = base.Diagnose(context.Background(), base.DiagnoseContext{}, storepb.Engine_STARROCKS, "SELECT FROM WHERE")
	require.NoError(t, err)
	require.NotEmpty(t, diags)
}
