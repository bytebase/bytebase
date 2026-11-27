package pg

import (
	"github.com/bytebase/omni/pg/analysis"
	"github.com/bytebase/omni/pg/ast"
	pgcompletion "github.com/bytebase/omni/pg/completion"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	engine := base.OmniEngine(analysis.Engine, pgcompletion.Engine)
	engine.QuerySpan = GetQuerySpan
	engine.ExtractChanged = extractChangedResources
	engine.TransformDMLToSelect = TransformDMLToSelect
	engine.GenerateRestoreSQL = GenerateRestoreSQL
	base.Register(engine, storepb.Engine_POSTGRES)
}

// GetNode extracts the omni AST node from a base.AST interface.
func GetNode(a base.AST) (ast.Node, bool) {
	return base.ExtractNode[ast.Node](a)
}

// StatementTypeWithPosition contains statement type and its position information.
type StatementTypeWithPosition = base.StatementTypeWithPosition

// GetStatementTypes returns statement types with position information.
// The line numbers are one-based.
func GetStatementTypes(asts []base.AST) ([]StatementTypeWithPosition, error) {
	return base.OmniStatementTypesWithPosition(analysis.Engine, asts)
}
