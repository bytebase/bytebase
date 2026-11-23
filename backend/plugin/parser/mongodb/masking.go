package mongodb

import (
	"slices"

	"github.com/bytebase/omni/mongo/analysis"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// analyzeMaskingStatement analyzes a MongoDB shell statement for masking checks.
// It returns nil, nil for statements that are not relevant to masking flow.
func analyzeMaskingStatement(statement string) (*base.MongoDBAnalysis, error) {
	stmts, err := parseMongoShell(statement)
	if err != nil {
		return nil, err
	}
	if len(stmts) != 1 {
		return nil, nil
	}

	node, ok := GetNode(stmts[0].AST)
	if !ok || node == nil {
		return nil, nil
	}

	a := analysis.Analyze(node)
	if a == nil {
		return nil, nil
	}

	return analysisToMasking(a)
}

func analysisToMasking(a *analysis.StatementAnalysis) (*base.MongoDBAnalysis, error) {
	if !a.Operation.IsRead() {
		return nil, nil
	}

	result := &base.MongoDBAnalysis{
		Operation:  a.MethodName,
		Collection: a.Collection,
	}

	switch a.Operation {
	case analysis.OpFind:
		result.API = base.MongoDBMaskableAPIFind
		result.PredicateFields = a.PredicateFields
	case analysis.OpFindOne:
		result.API = base.MongoDBMaskableAPIFindOne
		result.PredicateFields = a.PredicateFields
	case analysis.OpAggregate:
		if a.UnsupportedStage != "" {
			result.API = base.MongoDBMaskableAPIUnsupportedRead
			result.UnsupportedStage = a.UnsupportedStage
			return result, nil
		}
		result.API = base.MongoDBMaskableAPIAggregate
		result.PredicateFields = a.PredicateFields
		for _, join := range a.Joins {
			result.JoinedCollections = append(result.JoinedCollections, base.MongoDBJoinedCollection{
				Collection: join.Collection,
				AsField:    join.AsField,
			})
		}
	default:
		result.API = base.MongoDBMaskableAPIUnsupportedRead
	}

	if len(result.PredicateFields) > 0 {
		slices.Sort(result.PredicateFields)
	}

	return result, nil
}
