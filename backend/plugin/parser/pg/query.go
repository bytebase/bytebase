package pg

import (
	"github.com/bytebase/omni/pg/analysis"
	"github.com/bytebase/omni/pg/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterQueryValidator(storepb.Engine_POSTGRES, validateQuery)
	// Redshift has its own implementation in the redshift package
	base.RegisterQueryValidator(storepb.Engine_COCKROACHDB, validateQuery)
}

// validateQuery reports whether the statement may run in the SQL editor's
// read-only mode and whether every statement returns data.
func validateQuery(statement string) (bool, bool, error) {
	stmts, err := ParsePg(statement)
	if err != nil {
		return false, false, err
	}
	nodes := make([]ast.Node, 0, len(stmts))
	for _, stmt := range stmts {
		nodes = append(nodes, stmt.AST)
	}
	readOnly, returnsData := analysis.ValidateQuery(nodes)
	return readOnly, returnsData, nil
}
