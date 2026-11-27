package mariadb

import (
	"github.com/bytebase/omni/mariadb/analysis"
	"github.com/bytebase/omni/mariadb/ast"
	omnicompletion "github.com/bytebase/omni/mariadb/completion"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	engine := base.OmniEngine(analysis.Engine, omnicompletion.Engine)
	engine.StatementTypes = GetStatementTypes
	engine.TransformDMLToSelect = TransformDMLToSelect
	engine.GenerateRestoreSQL = GenerateRestoreSQL
	base.Register(engine, storepb.Engine_MARIADB)
}

// GetNode extracts the omni AST node from a base.AST interface.
func GetNode(a base.AST) (ast.Node, bool) {
	return base.ExtractNode[ast.Node](a)
}

// SplitSQL splits a script through the registered MariaDB engine.
func SplitSQL(statement string) ([]base.Statement, error) {
	return base.SplitMultiSQL(storepb.Engine_MARIADB, statement)
}

func GetStatementTypes(asts []base.AST) ([]storepb.StatementType, error) {
	return base.CollectOmniStatementTypes(asts, GetNode, "expected AST for MariaDB", analysis.StatementTypeOf)
}
