package pg

import (
	"testing"

	pganalysis "github.com/bytebase/omni/pg/analysis"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

func TestCheckSDLStyle(t *testing.T) {
	t.Run("valid and empty inputs produce no advice", func(t *testing.T) {
		for _, sql := range []string{
			"",
			"-- comment only",
			"CREATE TABLE public.users (id int NOT NULL, CONSTRAINT pk_users PRIMARY KEY (id));",
		} {
			advices, err := CheckSDLStyle(sql)
			require.NoError(t, err)
			require.Empty(t, advices)
		}
	})

	t.Run("syntax error is returned as error instead of advice", func(t *testing.T) {
		for _, sql := range []string{
			"CREATE TABLE public.users (id INTEGER;",
			"CREATE TABLEX public.users (id INTEGER);",
		} {
			advices, err := CheckSDLStyle(sql)
			require.Error(t, err)
			require.Nil(t, advices)
		}
	})

	t.Run("renders every style finding code, title, content, and line", func(t *testing.T) {
		sql := "CREATE TABLE users (\n" +
			"  id int PRIMARY KEY UNIQUE,\n" +
			"  ref_id int REFERENCES other(id),\n" +
			"  PRIMARY KEY (id),\n" +
			"  UNIQUE (ref_id),\n" +
			"  CHECK (id > 0),\n" +
			"  FOREIGN KEY (ref_id) REFERENCES parent(id),\n" +
			"  EXCLUDE USING gist (id WITH =)\n" +
			");\n" +
			"CREATE INDEX ON users (id);\n" +
			"CREATE INDEX idx_u ON users (id);\n" +
			"CREATE VIEW v AS SELECT 1;\n" +
			"CREATE SEQUENCE seq;\n" +
			"ALTER SEQUENCE seq RESTART;\n" +
			"CREATE FUNCTION fn() RETURNS int AS $$ SELECT 1 $$ LANGUAGE sql;"

		advices, err := CheckSDLStyle(sql)
		require.NoError(t, err)

		type want struct {
			code    code.Code
			title   string
			line    int32
			contain []string
		}
		expected := []want{
			{code.SDLRequireSchemaName, "sdl.require-schema-name", 1, []string{"Table 'users' is created without explicit schema name"}},
			{code.SDLDisallowColumnConstraint, "sdl.disallow-column-constraint", 2, []string{"Column 'id' in table 'users'", "[PRIMARY KEY UNIQUE]"}},
			{code.SDLRequireSchemaName, "sdl.require-schema-name", 3, []string{"Column 'ref_id' in table 'users'", "references", "table 'other'"}},
			{code.SDLDisallowColumnConstraint, "sdl.disallow-column-constraint", 3, []string{"Column 'ref_id' in table 'users'", "[FOREIGN KEY]"}},
			{code.SDLRequireConstraintName, "sdl.require-constraint-name", 4, []string{"PRIMARY KEY constraint without an explicit name", "PRIMARY KEY (id)"}},
			{code.SDLRequireConstraintName, "sdl.require-constraint-name", 5, []string{"UNIQUE constraint without an explicit name", "UNIQUE (column)"}},
			{code.SDLRequireConstraintName, "sdl.require-constraint-name", 6, []string{"CHECK constraint without an explicit name", "CHECK (condition)"}},
			{code.SDLRequireConstraintName, "sdl.require-constraint-name", 7, []string{"FOREIGN KEY constraint without an explicit name", "FOREIGN KEY (column) REFERENCES other_table(id)"}},
			{code.SDLRequireSchemaName, "sdl.require-schema-name", 7, []string{"Foreign key constraint in table 'users' references table 'parent'"}},
			{code.SDLRequireConstraintName, "sdl.require-constraint-name", 8, []string{"CONSTRAINT constraint without an explicit name", "CONSTRAINT constraint_name ..."}},
			{code.SDLRequireIndexName, "sdl.require-index-name", 10, []string{"Index is created without an explicit name"}},
			{code.SDLRequireSchemaName, "sdl.require-schema-name", 10, []string{"Index is created on table 'users' without explicit schema name"}},
			{code.SDLRequireSchemaName, "sdl.require-schema-name", 11, []string{"Index 'idx_u' is created on table 'users' without explicit schema name"}},
			{code.SDLRequireSchemaName, "sdl.require-schema-name", 12, []string{"View 'v' is created without explicit schema name"}},
			{code.SDLRequireSchemaName, "sdl.require-schema-name", 13, []string{"Sequence 'seq' is created without explicit schema name"}},
			{code.SDLRequireSchemaName, "sdl.require-schema-name", 14, []string{"Sequence 'seq' is altered without explicit schema name"}},
			{code.SDLRequireSchemaName, "sdl.require-schema-name", 15, []string{"Function 'fn' is created without explicit schema name"}},
		}

		require.Len(t, advices, len(expected))
		for i, w := range expected {
			a := advices[i]
			require.Equal(t, storepb.Advice_ERROR, a.Status, "advice %d", i)
			require.Equal(t, w.code.Int32(), a.Code, "advice %d", i)
			require.Equal(t, w.title, a.Title, "advice %d", i)
			require.Equal(t, &storepb.Position{Line: w.line, Column: 0}, a.StartPosition, "advice %d", i)
			for _, sub := range w.contain {
				require.Contains(t, a.Content, sub, "advice %d", i)
			}
		}
	})
}

func TestSDLStyleLineCalculation(t *testing.T) {
	// Statement-anchored findings (Start < 0) count only LF line breaks.
	stmtWithCR := "CREATE TABLE public.a (id int);\rCREATE TABLE b (id int);"
	advices, err := CheckSDLStyle(stmtWithCR)
	require.NoError(t, err)
	require.Len(t, advices, 1)
	require.Equal(t, int32(1), advices[0].StartPosition.Line)

	stmtWithLF := "CREATE TABLE public.a (id int);\nCREATE TABLE b (id int);"
	advices, err = CheckSDLStyle(stmtWithLF)
	require.NoError(t, err)
	require.Len(t, advices, 1)
	require.Equal(t, int32(2), advices[0].StartPosition.Line)

	// Node-anchored findings (Start >= 0) use ByteOffsetToRunePosition, which counts CR as a line break.
	nodeWithCR := "CREATE TABLE public.a (\rid int PRIMARY KEY\r);"
	advices, err = CheckSDLStyle(nodeWithCR)
	require.NoError(t, err)
	require.Len(t, advices, 1)
	require.Equal(t, int32(2), advices[0].StartPosition.Line)

	// Verify sdlStyleLine directly for both branches.
	require.Equal(t, int32(1), sdlStyleLine("a\rb", pganalysis.SDLFinding{Start: -1, StatementStart: 2}))
	require.Equal(t, int32(2), sdlStyleLine("a\rb", pganalysis.SDLFinding{Start: 2, StatementStart: 0}))
}
