package pg

import (
	"strings"
	"testing"

	"github.com/bytebase/omni/pg/ast"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

func TestOmniIsRoleOrSearchPathSet(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		want []string
	}{
		{
			name: "set role keyword syntax",
			sql:  "SET ROLE admin; INSERT INTO t VALUES (1);",
			want: []string{"SET ROLE admin"},
		},
		{
			name: "set role generic syntax",
			sql:  "SET role = 'admin'; INSERT INTO t VALUES (1);",
			want: []string{"SET role = 'admin'"},
		},
		{
			name: "set search path",
			sql:  "SET search_path = myschema, public; UPDATE t SET a = 1;",
			want: []string{"SET search_path = myschema, public"},
		},
		{
			name: "reset all",
			sql:  "SET search_path = myschema; RESET ALL; UPDATE t SET a = 1;",
			want: []string{"SET search_path = myschema", "RESET ALL"},
		},
		{
			name: "ignore unrelated set variable",
			sql:  "SET statement_timeout = '1s'; DELETE FROM t;",
			want: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			statements, err := base.ParseStatements(storepb.Engine_POSTGRES, tc.sql)
			require.NoError(t, err)

			var preExecutions []string
			for _, stmt := range statements {
				if stmt.AST == nil {
					continue
				}
				node, ok := pgparser.GetOmniNode(stmt.AST)
				if !ok {
					continue
				}
				if vs, ok := node.(*ast.VariableSetStmt); ok {
					if omniIsRoleOrSearchPathSet(vs) {
						preExecutions = append(preExecutions, strings.TrimRight(strings.TrimSpace(stmt.Text), ";"))
					}
				}
			}

			require.Equal(t, tc.want, preExecutions)
		})
	}
}

func TestSessionSettings(t *testing.T) {
	var settings sessionSettings
	add := func(statement string) {
		statements, err := base.ParseStatements(storepb.Engine_POSTGRES, statement)
		require.NoError(t, err)
		for _, stmt := range statements {
			node, ok := pgparser.GetOmniNode(stmt.AST)
			require.True(t, ok)
			settings.add(node, strings.TrimRight(strings.TrimSpace(stmt.Text), ";"))
		}
	}

	add(`SET search_path = z;
DISCARD ALL;
SET search_path = a;
BEGIN;
SET LOCAL search_path = b;
SET ROLE r;
COMMIT;
BEGIN;
SET search_path = c;
ROLLBACK;
BEGIN;
SET search_path = d;
COMMIT AND CHAIN;
ROLLBACK;
SET LOCAL search_path = e;
SET statement_timeout = '1s';
RESET search_path;`)
	// The settings replay in statement order, so RESET follows the SET LOCAL it overrides.
	require.Equal(t, []string{"SET search_path = a", "SET ROLE r", "SET search_path = d", "SET LOCAL search_path = e", "RESET search_path"}, settings.statements())

	// A nested BEGIN only warns, so the ROLLBACK returns to the settings before the first BEGIN, and
	// the transaction's end also ends the SET LOCAL.
	add(`BEGIN;
SET search_path = f;
BEGIN;
SET search_path = g;
ROLLBACK;`)
	require.Equal(t, []string{"SET search_path = a", "SET ROLE r", "SET search_path = d", "RESET search_path"}, settings.statements())

	// A ROLLBACK outside a transaction only warns, so it keeps the session settings.
	add(`SET search_path = h;
SET LOCAL search_path = i;
ROLLBACK;`)
	require.Equal(t, []string{"SET search_path = a", "SET ROLE r", "SET search_path = d", "RESET search_path", "SET search_path = h"}, settings.statements())

	// SET ... FROM CURRENT keeps the SET LOCAL value of its own setting past COMMIT.
	add(`BEGIN;
SET LOCAL ROLE s;
SET LOCAL search_path = j;
SET search_path FROM CURRENT;
COMMIT;`)
	require.Equal(t, []string{"SET search_path = a", "SET ROLE r", "SET search_path = d", "RESET search_path", "SET search_path = h", "SET LOCAL search_path = j", "SET search_path FROM CURRENT"}, settings.statements())
}
