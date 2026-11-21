package doris

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestGetQuerySpanSeam(t *testing.T) {
	// Registry dispatch, default database resolution, and QueryType mapping.
	spans, err := base.GetQuerySpan(context.Background(), base.GetQuerySpanContext{}, storepb.Engine_DORIS, []base.Statement{
		{Text: "SELECT id FROM users"},
	}, "testdb", "", false)
	require.NoError(t, err)
	require.Len(t, spans, 1)
	require.Equal(t, base.Select, spans[0].Type)
	require.Equal(t, base.SourceColumnSet{{Database: "testdb", Table: "users"}: true}, spans[0].SourceColumns)

	for _, tc := range []struct {
		sql    string
		ignore bool
		want   base.QueryType
	}{
		{"SELECT * FROM MYSQL.user", true, base.SelectInfoSchema},
		{"SELECT * FROM information_schema.tables", false, base.SelectInfoSchema},
		{"INSERT INTO users (id) VALUES (1)", false, base.DML},
		{"CREATE TABLE t (id INT)", false, base.DDL},
		{"USE db1", false, base.QueryTypeUnknown},
	} {
		span, err := GetQuerySpan(context.Background(), base.GetQuerySpanContext{}, base.Statement{Text: tc.sql}, "testdb", "", tc.ignore)
		require.NoErrorf(t, err, "%q", tc.sql)
		require.Equalf(t, tc.want, span.Type, "%q", tc.sql)
	}

	// MixUserSystemTablesError conversion.
	_, err = GetQuerySpan(context.Background(), base.GetQuerySpanContext{}, base.Statement{Text: "SELECT * FROM users JOIN information_schema.tables"}, "testdb", "", false)
	require.ErrorIs(t, err, base.MixUserSystemTablesError)

	// SyntaxError position offset using Statement.Start.
	_, err = GetQuerySpan(context.Background(), base.GetQuerySpanContext{}, base.Statement{
		Text:  "SELECT a FROM secret xx yy zz",
		Start: &storepb.Position{Line: 4, Column: 1},
	}, "testdb", "", false)
	var syntaxErr *base.SyntaxError
	require.ErrorAs(t, err, &syntaxErr)
	require.Equal(t, int32(4), syntaxErr.Position.Line)
}
