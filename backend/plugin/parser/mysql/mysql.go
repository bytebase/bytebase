package mysql

import (
	"github.com/bytebase/omni/mysql/analysis"
	"github.com/bytebase/omni/mysql/ast"
	omnicompletion "github.com/bytebase/omni/mysql/completion"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	engine := base.OmniEngine(analysis.Engine, omnicompletion.Engine)
	engine.StatementTypes = GetStatementTypes
	base.Register(engine, storepb.Engine_MYSQL)
}

// GetNode extracts the omni AST node from a base.AST interface.
func GetNode(a base.AST) (ast.Node, bool) {
	return base.ExtractNode[ast.Node](a)
}

// SplitSQL splits a script through the registered MySQL engine.
func SplitSQL(statement string) ([]base.Statement, error) {
	return base.SplitMultiSQL(storepb.Engine_MYSQL, statement)
}

func GetStatementTypes(asts []base.AST) ([]storepb.StatementType, error) {
	return base.CollectOmniStatementTypes(asts, GetNode, "expected AST for MySQL", analysis.StatementTypeOf)
}

// StatementTypeWithPosition contains a statement type and its position information.
type StatementTypeWithPosition = base.StatementTypeWithPosition

// GetStatementTypesWithPositions returns statement types with position information,
// preserving statement order. Line numbers are one-based.
func GetStatementTypesWithPositions(asts []base.AST) ([]StatementTypeWithPosition, error) {
	// STATEMENT_TYPE_UNSPECIFIED entries are included so the SDL gate fails closed.
	return base.CollectStatementTypesWithPosition(asts, func(a base.AST) ([]storepb.StatementType, string, bool, error) {
		node, ok := GetNode(a)
		if !ok {
			return nil, "", false, errors.New("expected AST for MySQL")
		}
		if node == nil {
			return nil, "", false, nil
		}
		return []storepb.StatementType{base.OmniStatementType(analysis.StatementTypeOf(node))}, base.ASTSource(a), true, nil
	})
}
