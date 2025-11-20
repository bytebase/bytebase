package pg

import (
	"testing"

	"github.com/bytebase/parser/postgresql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractParameterNamesFromCreateFunction(t *testing.T) {
	tests := []struct {
		name       string
		definition string
		want       []string
	}{
		{
			name: "OUT parameters",
			definition: `CREATE FUNCTION get_user(IN user_id INT, OUT username TEXT, OUT email TEXT)
AS $$
  SELECT name, email FROM users WHERE id = user_id;
$$ LANGUAGE SQL;`,
			want: []string{"username", "email"},
		},
		{
			name: "RETURNS TABLE",
			definition: `CREATE FUNCTION get_users()
RETURNS TABLE(id INT, name TEXT, email TEXT)
AS $$
  SELECT id, name, email FROM users;
$$ LANGUAGE SQL;`,
			want: []string{"id", "name", "email"},
		},
		{
			name: "Mixed OUT and IN parameters",
			definition: `CREATE FUNCTION calculate(IN a INT, IN b INT, OUT sum INT, OUT product INT)
AS $$
BEGIN
  sum := a + b;
  product := a * b;
END;
$$ LANGUAGE plpgsql;`,
			want: []string{"sum", "product"},
		},
		{
			name: "INOUT parameter",
			definition: `CREATE FUNCTION increment(INOUT value INT)
AS $$
BEGIN
  value := value + 1;
END;
$$ LANGUAGE plpgsql;`,
			want: []string{"value"},
		},
		{
			name: "No OUT parameters",
			definition: `CREATE FUNCTION add(IN a INT, IN b INT)
RETURNS INT
AS $$
  SELECT a + b;
$$ LANGUAGE SQL;`,
			want: nil,
		},
		{
			name: "Quoted parameter names",
			definition: `CREATE FUNCTION get_data(OUT "userId" INT, OUT "userName" TEXT)
AS $$
  SELECT 1, 'test';
$$ LANGUAGE SQL;`,
			want: []string{"userId", "userName"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parseResults, err := ParsePostgreSQL(tt.definition)
			require.NoError(t, err, "failed to parse function definition")
			require.Len(t, parseResults, 1, "expected exactly 1 statement")

			root, ok := parseResults[0].Tree.(*postgresql.RootContext)
			require.True(t, ok, "expected RootContext")
			stmtblock := root.Stmtblock()
			stmtmulti := stmtblock.Stmtmulti()
			stmts := stmtmulti.AllStmt()
			require.Len(t, stmts, 1)

			createFuncStmt := stmts[0].Createfunctionstmt()
			require.NotNil(t, createFuncStmt)

			q := &querySpanExtractor{}
			got := q.extractParameterNamesFromCreateFunction(createFuncStmt)

			require.NoError(t, err)
			assert.Equal(t, tt.want, got, "parameter names mismatch")
		})
	}
}
