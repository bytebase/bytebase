// Package oceanbase parses OceanBase's MySQL mode. OceanBase accepts distributed DDL
// MySQL rejects, so it carries its own omni parser and AST rather than MySQL's.
package oceanbase

import (
	"github.com/bytebase/omni/oceanbase/analysis"
	"github.com/bytebase/omni/oceanbase/ast"
	omnicompletion "github.com/bytebase/omni/oceanbase/completion"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	engine := base.OmniEngine(analysis.Engine, omnicompletion.Engine)
	engine.StatementTypes = GetStatementTypes
	engine.ExtractChanged = extractChangedResources
	base.Register(engine, storepb.Engine_OCEANBASE)
}

// GetNode extracts the omni AST node from a base.AST interface.
func GetNode(a base.AST) (ast.Node, bool) {
	return base.ExtractNode[ast.Node](a)
}

// SplitSQL splits a script through the registered OceanBase engine.
func SplitSQL(statement string) ([]base.Statement, error) {
	return base.SplitMultiSQL(storepb.Engine_OCEANBASE, statement)
}

func GetStatementTypes(asts []base.AST) ([]storepb.StatementType, error) {
	return base.CollectOmniStatementTypes(asts, GetNode, "expected AST for OceanBase", analysis.StatementTypeOf)
}

// EstimateAffectedRowsFromExplainJSON returns the EST.ROWS estimate from OceanBase
// `EXPLAIN FORMAT=JSON` output.
func EstimateAffectedRowsFromExplainJSON(plan string) (int64, error) {
	return analysis.EstimateAffectedRowsFromExplainJSON(plan)
}

func extractChangedResources(currentDatabase string, _ string, dbMetadata *model.DatabaseMetadata, asts []base.AST, _ string) (*base.ChangeSummary, error) {
	return base.ExtractChangedResourcesFromASTs(dbMetadata, asts, false, func(a base.AST) (analysis.StatementChanges, bool, error) {
		node, ok := GetNode(a)
		if !ok {
			return analysis.StatementChanges{}, false, errors.New("expected AST for MySQL")
		}
		return analysis.ExtractChangedResources(node, currentDatabase, base.ASTSource(a)), true, nil
	})
}
