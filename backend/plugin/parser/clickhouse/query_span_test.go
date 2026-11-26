package clickhouse

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestGetQuerySpan(t *testing.T) {
	// Registry dispatch, default database resolution, and explicit database qualification.
	spans, err := base.GetQuerySpan(context.Background(), base.GetQuerySpanContext{}, storepb.Engine_CLICKHOUSE, []base.Statement{
		{Text: "SELECT * FROM orders LIMIT 10"},
		{Text: "SELECT * FROM example_db.orders LIMIT 10"},
	}, "default_db", "", false)
	require.NoError(t, err)
	require.Len(t, spans, 2)

	require.Equal(t, base.Select, spans[0].Type)
	require.Equal(t, base.SourceColumnSet{{Database: "default_db", Table: "orders"}: true}, spans[0].SourceColumns)

	require.Equal(t, base.Select, spans[1].Type)
	require.Equal(t, base.SourceColumnSet{{Database: "example_db", Table: "orders"}: true}, spans[1].SourceColumns)

	for _, tc := range []struct {
		name     string
		sql      string
		wantType base.QueryType
		wantCols base.SourceColumnSet
	}{
		{
			name:     "select without table",
			sql:      "SELECT 1",
			wantType: base.Select,
			wantCols: base.SourceColumnSet{},
		},
		{
			name:     "join across databases",
			sql:      "SELECT o.id, u.name FROM example_db.orders o JOIN users u ON o.user_id = u.id",
			wantType: base.Select,
			wantCols: base.SourceColumnSet{
				{Database: "example_db", Table: "orders"}: true,
				{Database: "default_db", Table: "users"}:  true,
			},
		},
		{
			name:     "cte excludes alias and includes underlying table",
			sql:      "WITH recent AS (SELECT * FROM orders) SELECT * FROM recent",
			wantType: base.Select,
			wantCols: base.SourceColumnSet{{Database: "default_db", Table: "orders"}: true},
		},
		{
			name:     "where subquery",
			sql:      "SELECT * FROM orders WHERE user_id IN (SELECT id FROM other_db.users)",
			wantType: base.Select,
			wantCols: base.SourceColumnSet{
				{Database: "default_db", Table: "orders"}: true,
				{Database: "other_db", Table: "users"}:    true,
			},
		},
		{
			name:     "union",
			sql:      "SELECT id FROM orders UNION ALL SELECT id FROM archived_orders",
			wantType: base.Select,
			wantCols: base.SourceColumnSet{
				{Database: "default_db", Table: "orders"}:          true,
				{Database: "default_db", Table: "archived_orders"}: true,
			},
		},
		{
			name:     "system table",
			sql:      "SELECT * FROM system.tables",
			wantType: base.SelectInfoSchema,
			wantCols: base.SourceColumnSet{{Database: "system", Table: "tables"}: true},
		},
		{
			name:     "information_schema table",
			sql:      "SELECT * FROM information_schema.tables",
			wantType: base.SelectInfoSchema,
			wantCols: base.SourceColumnSet{{Database: "information_schema", Table: "tables"}: true},
		},
		{
			name:     "case-sensitive user database named System is not treated as system",
			sql:      "SELECT * FROM System.secret",
			wantType: base.Select,
			wantCols: base.SourceColumnSet{{Database: "System", Table: "secret"}: true},
		},
		{
			name:     "insert",
			sql:      "INSERT INTO example_db.orders (id) VALUES (1)",
			wantType: base.DML,
			wantCols: base.SourceColumnSet{{Database: "example_db", Table: "orders"}: true},
		},
		{
			name:     "ddl",
			sql:      "CREATE TABLE orders (id UInt64) ENGINE = MergeTree() ORDER BY id",
			wantType: base.DDL,
			wantCols: base.SourceColumnSet{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			span, err := GetQuerySpan(context.Background(), base.GetQuerySpanContext{}, base.Statement{Text: tc.sql}, "default_db", "", false)
			require.NoError(t, err)
			require.Equal(t, tc.wantType, span.Type)
			require.Equal(t, tc.wantCols, span.SourceColumns)
		})
	}

	// Mixing user and system tables must be rejected.
	_, err = GetQuerySpan(context.Background(), base.GetQuerySpanContext{}, base.Statement{
		Text: "SELECT * FROM orders JOIN system.tables ON true",
	}, "default_db", "", false)
	require.ErrorIs(t, err, base.MixUserSystemTablesError)

	// Table functions (e.g. remote(), cluster()) must fail closed instead of returning an empty span.
	_, err = GetQuerySpan(context.Background(), base.GetQuerySpanContext{}, base.Statement{
		Text: "SELECT * FROM remote('127.0.0.1', 'other_db', 'secret')",
	}, "default_db", "", false)
	require.ErrorContains(t, err, "table function")

	// Syntax error preserves statement start offset.
	_, err = GetQuerySpan(context.Background(), base.GetQuerySpanContext{}, base.Statement{
		Text:  "SELECT * FROM",
		Start: &storepb.Position{Line: 4, Column: 1},
	}, "default_db", "", false)
	var syntaxErr *base.SyntaxError
	require.ErrorAs(t, err, &syntaxErr)
	require.Equal(t, int32(4), syntaxErr.Position.Line)
}
