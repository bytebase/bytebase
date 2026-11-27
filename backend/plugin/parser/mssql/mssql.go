package mssql

import (
	"github.com/bytebase/omni/mssql/analysis"
	"github.com/bytebase/omni/mssql/ast"
	mssqlcompletion "github.com/bytebase/omni/mssql/completion"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	engine := base.OmniEngine(analysis.Engine, mssqlcompletion.Engine)
	engine.StatementTypes = GetStatementTypes
	base.Register(engine, storepb.Engine_MSSQL)
}

// GetNode extracts the omni AST node from a base.AST interface.
func GetNode(a base.AST) (ast.Node, bool) {
	return base.ExtractNode[ast.Node](a)
}

// GetStatementTypes returns the distinct types of the statements, including
// STATEMENT_TYPE_UNSPECIFIED for one omni does not classify.
func GetStatementTypes(asts []base.AST) ([]storepb.StatementType, error) {
	return base.CollectOmniStatementTypes(asts, GetNode, "expected omni AST for MSSQL", analysis.StatementTypeOf)
}
