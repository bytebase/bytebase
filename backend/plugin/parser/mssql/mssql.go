package mssql

import (
	"github.com/bytebase/omni/mssql/analysis"
	"github.com/bytebase/omni/mssql/ast"
	mssqlcompletion "github.com/bytebase/omni/mssql/completion"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	engine := base.OmniEngine(analysis.Engine, mssqlcompletion.Engine)
	engine.StatementTypes = GetStatementTypes
	engine.ExtractChanged = extractChangedResources
	engine.TransformDMLToSelect = TransformDMLToSelect
	engine.GenerateRestoreSQL = GenerateRestoreSQL
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

func extractChangedResources(currentDatabase string, currentSchema string, dbMetadata *model.DatabaseMetadata, asts []base.AST, _ string) (*base.ChangeSummary, error) {
	return base.ExtractChangedResourcesFromASTs(dbMetadata, asts, false, func(a base.AST) (analysis.StatementChanges, bool, error) {
		node, ok := GetNode(a)
		if !ok {
			return analysis.StatementChanges{}, false, errors.New("expected AST for MSSQL")
		}
		if node == nil {
			return analysis.StatementChanges{}, false, nil
		}
		return analysis.ExtractChangedResources(node, currentDatabase, currentSchema, base.ASTSource(a)), true, nil
	})
}
