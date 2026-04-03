package mongodb

import (
	"slices"

	"github.com/bytebase/omni/mongo/analysis"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// GetQueryType parses a MongoDB shell statement and returns its query type.
func GetQueryType(statement string) base.QueryType {
	stmts, err := ParseMongoShell(statement)
	if err != nil || len(stmts) == 0 {
		return base.DML
	}

	node, ok := GetOmniNode(stmts[0].AST)
	if !ok || node == nil {
		return base.DML
	}

	a := analysis.Analyze(node)
	if a == nil {
		return base.DML
	}

	return omniOperationToQueryType(a)
}

// omniOperationToQueryType maps an omni StatementAnalysis to a base.QueryType.
func omniOperationToQueryType(a *analysis.StatementAnalysis) base.QueryType {
	switch a.Operation {
	case analysis.OpFind, analysis.OpFindOne, analysis.OpCount, analysis.OpDistinct:
		return base.Select
	case analysis.OpAggregate:
		if slices.Contains(a.PipelineStages, "$out") || slices.Contains(a.PipelineStages, "$merge") {
			return base.DML
		}
		return base.Select
	case analysis.OpRead, analysis.OpInfo:
		return base.SelectInfoSchema
	case analysis.OpWrite:
		return base.DML
	case analysis.OpAdmin:
		return base.DDL
	case analysis.OpExplain:
		return base.Explain
	default:
		return base.DML
	}
}
