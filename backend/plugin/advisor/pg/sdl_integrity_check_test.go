package pg

import (
	"testing"

	pganalysis "github.com/bytebase/omni/pg/analysis"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

func TestCheckSDLIntegrity(t *testing.T) {
	t.Run("valid and empty statements produce no advice", func(t *testing.T) {
		for _, sql := range []string{
			"",
			"-- comment only",
			"CREATE TABLE public.users (id int, CONSTRAINT pk_users PRIMARY KEY (id));",
		} {
			advices, err := checkSingleStatement(sql)
			require.NoError(t, err)
			require.Empty(t, advices)
		}
	})

	t.Run("syntax error is rendered as StatementSyntaxError advice with nil position", func(t *testing.T) {
		for _, sql := range []string{
			"CREATE TABLE public.users (id INTEGER;",
			"CREATE TABLEX public.users (id INTEGER);",
		} {
			advices, err := checkSingleStatement(sql)
			require.NoError(t, err)
			require.Len(t, advices, 1)
			require.Equal(t, storepb.Advice_ERROR, advices[0].Status)
			require.Equal(t, code.StatementSyntaxError.Int32(), advices[0].Code)
			require.Equal(t, "SQL syntax error", advices[0].Title)
			require.Contains(t, advices[0].Content, "Failed to parse SQL in file 'statement.sql':")
			require.Nil(t, advices[0].StartPosition)
		}
	})

	t.Run("renders all single-file integrity advice branches with line numbers", func(t *testing.T) {
		sql := "CREATE TABLE public.users (\n" +
			"  id int,\n" +
			"  id text,\n" +
			"  CONSTRAINT pk_common PRIMARY KEY (id),\n" +
			"  PRIMARY KEY (id),\n" +
			"  CONSTRAINT chk_pos CHECK (id > 0),\n" +
			"  CONSTRAINT chk_pos CHECK (id < 100)\n" +
			");\n" +
			"CREATE TABLE public.users (x int);\n" +
			"CREATE TABLE public.orders (\n" +
			"  id int,\n" +
			"  user_id int,\n" +
			"  CONSTRAINT pk_common PRIMARY KEY (id),\n" +
			"  CONSTRAINT fk_missing_tbl FOREIGN KEY (user_id) REFERENCES public.missing_tbl(id),\n" +
			"  FOREIGN KEY (user_id) REFERENCES public.users(missing_col)\n" +
			");\n" +
			"CREATE INDEX idx_u ON public.users (id);\n" +
			"CREATE INDEX idx_u ON public.orders (id);\n" +
			"CREATE VIEW public.v AS SELECT * FROM public.no_such_table;"

		advices, err := checkSingleStatement(sql)
		require.NoError(t, err)

		type want struct {
			code    code.Code
			title   string
			line    int32
			contain []string
		}
		expected := []want{
			{code.SDLDuplicateColumnName, "Duplicate column name", 3, []string{"Column 'id' is defined multiple times in table 'public.users'"}},
			{code.SDLDuplicateConstraintName, "Duplicate constraint name", 7, []string{"Constraint 'chk_pos' is defined multiple times in table 'public'", "line 6", "line 7", "CHECK and FOREIGN KEY constraint names to be unique within a table"}},
			{code.SDLDuplicateTableName, "Duplicate table name", 9, []string{"Table 'public.users' is defined multiple times", "First definition at line 1", "Duplicate definition at line 9"}},
			{code.SDLDuplicateConstraintName, "Duplicate constraint name", 13, []string{"Constraint 'pk_common' is defined multiple times in schema 'public'", "on table 'users' (line 4)", "on table 'orders' (line 13)", "PRIMARY KEY and UNIQUE constraint names to be unique within a schema"}},
			{code.SDLDuplicateIndexName, "Duplicate index name", 18, []string{"Index 'public.idx_u' is defined multiple times", "First definition at line 17", "Duplicate definition at line 18"}},
			{code.SDLMultiplePrimaryKey, "Multiple primary keys defined", 5, []string{"Table 'public.users' has multiple PRIMARY KEY constraints", "1. pk_common at line 4", "2. <unnamed> at line 5"}},
			{code.SDLForeignKeyTableNotFound, "Foreign key references non-existent table", 14, []string{"Foreign key constraint 'fk_missing_tbl' on table 'public.orders'", "references table 'public.missing_tbl'", "Foreign key columns: user_id"}},
			{code.SDLForeignKeyColumnNotFound, "Foreign key references non-existent column", 15, []string{"Foreign key constraint '<unnamed>' on table 'public.orders'", "Source column: public.orders.user_id", "Referenced column: public.users.missing_col (DOES NOT EXIST)", "Available columns in 'public.users': id"}},
			{code.SDLViewDependencyNotFound, "View references non-existent table or view", 19, []string{"View 'public.v' (line 19) references table or view 'public.no_such_table'"}},
		}

		require.Len(t, advices, len(expected))
		for i, w := range expected {
			a := advices[i]
			require.Equal(t, storepb.Advice_ERROR, a.Status, "advice %d", i)
			require.Equal(t, w.code.Int32(), a.Code, "advice %d", i)
			require.Equal(t, w.title, a.Title, "advice %d", i)
			require.Equal(t, &storepb.Position{Line: w.line}, a.StartPosition, "advice %d", i)
			for _, sub := range w.contain {
				require.Contains(t, a.Content, sub, "advice %d", i)
			}
		}
	})
}

func TestSDLIntegrityHelpersAndLineCalculation(t *testing.T) {
	files := map[string]string{
		"a.sql": "line1\nline2\rline3",
	}
	require.Equal(t, 1, sdlFileLine(files, "a.sql", -1))
	require.Equal(t, 2, sdlFileLine(files, "a.sql", 6))
	require.Equal(t, 3, sdlFileLine(files, "a.sql", 12))

	require.Equal(t, "<unnamed>", sdlConstraintName(""))
	require.Equal(t, "fk_a", sdlConstraintName("fk_a"))
	require.Equal(t, "<none>", sdlJoinOr(nil, "<none>"))
	require.Equal(t, "a, b", sdlJoinOr([]string{"a", "b"}, "<none>"))

	// Empty column slices fall back to "<none>" and "<no columns>".
	fkTbl := sdlIntegrityAdvice(files, "a.sql", pganalysis.SDLFinding{
		Code:      pganalysis.SDLForeignKeyTableNotFound,
		Start:     -1,
		Schema:    "public",
		Table:     "orders",
		RefSchema: "public",
		RefTable:  "users",
	})
	require.Contains(t, fkTbl.Content, "Foreign key constraint '<unnamed>'")
	require.Contains(t, fkTbl.Content, "Foreign key columns: <none>")
	require.Equal(t, int32(1), fkTbl.StartPosition.Line)

	fkCol := sdlIntegrityAdvice(files, "a.sql", pganalysis.SDLFinding{
		Code:      pganalysis.SDLForeignKeyColumnNotFound,
		Start:     -1,
		Schema:    "public",
		Table:     "orders",
		Column:    "uid",
		RefSchema: "public",
		RefTable:  "users",
		RefColumn: "id",
	})
	require.Contains(t, fkCol.Content, "Available columns in 'public.users': <no columns>")
}
