package tidb

import (
	"github.com/bytebase/omni/tidb/analysis"
	"github.com/bytebase/omni/tidb/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterQueryValidator(storepb.Engine_TIDB, validateQuery)
}

// validateQuery validates the SQL statement for SQL editor.
func validateQuery(statement string) (bool, bool, error) {
	stmts, err := base.ParseStatements(storepb.Engine_TIDB, statement)
	if err != nil {
		return false, false, err
	}
	var nodes []ast.Node
	for _, stmt := range stmts {
		node, ok := GetNode(stmt.AST)
		if !ok {
			// An empty or comment-only statement carries no AST.
			continue
		}
		nodes = append(nodes, node)
	}
	readOnly, returnsData := analysis.ValidateQuery(nodes)
	return readOnly, returnsData, nil
}
