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
