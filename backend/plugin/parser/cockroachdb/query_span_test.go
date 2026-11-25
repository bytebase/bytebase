package cockroachdb

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func TestGetQuerySpanSeam(t *testing.T) {
	gCtx := base.GetQuerySpanContext{
		GetDatabaseMetadataFunc: func(context.Context, string, string) (string, *model.DatabaseMetadata, error) {
			return "db", databaseMetadata(), nil
		},
	}
	// A view's column also names the table column behind it.
	spans, err := base.GetQuerySpan(context.Background(), gCtx, storepb.Engine_COCKROACHDB, []base.Statement{{Text: "SELECT ssn FROM v"}}, "db", "", false)
	require.NoError(t, err)
	require.Equal(t, base.SourceColumnSet{
		{Database: "db", Schema: "public", Table: "v", Column: "ssn"}: true,
		{Database: "db", Schema: "public", Table: "t", Column: "ssn"}: true,
	}, spans[0].Results[0].SourceColumns)

	tableColumn := base.ColumnResource{Database: "db", Schema: "public", Table: "t", Column: "ssn"}
	// CockroachDB-only syntax the PostgreSQL grammar rejects.
	for _, statement := range []string{"SELECT ssn FROM t AS OF SYSTEM TIME '-10s'", "SELECT ssn FROM t@primary"} {
		spans, err = base.GetQuerySpan(context.Background(), gCtx, storepb.Engine_COCKROACHDB, []base.Statement{{Text: statement}}, "db", "", false)
		require.NoError(t, err, statement)
		require.Equal(t, base.Select, spans[0].Type, statement)
		require.Equal(t, base.SourceColumnSet{tableColumn: true}, spans[0].Results[0].SourceColumns, statement)
	}

	spans, err = base.GetQuerySpan(context.Background(), gCtx, storepb.Engine_COCKROACHDB, []base.Statement{{Text: "UPSERT INTO t VALUES ('x')"}}, "db", "", false)
	require.NoError(t, err)
	require.Equal(t, base.DML, spans[0].Type)

	// A selected schema goes before public on the search path.
	spans, err = base.GetQuerySpan(context.Background(), gCtx, storepb.Engine_COCKROACHDB, []base.Statement{{Text: "SELECT ssn FROM t"}}, "db", "app", false)
	require.NoError(t, err)
	require.Equal(t, base.SourceColumnSet{tableColumn: true}, spans[0].Results[0].SourceColumns)

	_, err = base.GetQuerySpan(context.Background(), gCtx, storepb.Engine_COCKROACHDB, []base.Statement{{Text: "SELECT '你好' FROM WHERE"}}, "db", "", false)
	var syntaxErr *base.SyntaxError
	require.ErrorAs(t, err, &syntaxErr)
	require.Equal(t, &storepb.Position{Line: 1, Column: int32(len([]rune("SELECT '你好' FROM ")) + 1)}, syntaxErr.Position)
}
