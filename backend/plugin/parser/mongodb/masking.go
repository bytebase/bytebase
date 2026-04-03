package mongodb

import (
	"slices"

	"github.com/bytebase/omni/mongo/analysis"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// Type aliases so that internal code and external consumers (e.g. catalog_masking_mongodb.go)
// can continue using the short names. The canonical definitions live in the base package.
type (
	MaskableAPI      = base.MongoDBMaskableAPI
	JoinedCollection = base.MongoDBJoinedCollection
	MaskingAnalysis  = base.MongoDBAnalysis
)

const (
	MaskableAPIUnsupported     = base.MongoDBMaskableAPIUnsupported
	MaskableAPIFind            = base.MongoDBMaskableAPIFind
	MaskableAPIFindOne         = base.MongoDBMaskableAPIFindOne
	MaskableAPIUnsupportedRead = base.MongoDBMaskableAPIUnsupportedRead
	MaskableAPIAggregate       = base.MongoDBMaskableAPIAggregate
)

// AnalyzeMaskingStatement analyzes a MongoDB shell statement for masking checks.
// It returns nil,nil for statements that are not relevant to masking flow.
func AnalyzeMaskingStatement(statement string) (*MaskingAnalysis, error) {
	stmts, err := ParseMongoShell(statement)
	if err != nil {
		return nil, err
	}
	if len(stmts) != 1 {
		return nil, nil
	}

	node, ok := GetOmniNode(stmts[0].AST)
	if !ok || node == nil {
		return nil, nil
	}

	a := analysis.Analyze(node)
	if a == nil {
		return nil, nil
	}

	return omniAnalysisToMasking(a)
}

// omniAnalysisToMasking converts an omni StatementAnalysis to a MaskingAnalysis.
// Returns nil,nil for non-read operations (not relevant to masking).
func omniAnalysisToMasking(a *analysis.StatementAnalysis) (*MaskingAnalysis, error) {
	if !a.Operation.IsRead() {
		return nil, nil
	}

	result := &MaskingAnalysis{
		Operation:  a.MethodName,
		Collection: a.Collection,
	}

	switch a.Operation {
	case analysis.OpFind:
		result.API = MaskableAPIFind
		result.PredicateFields = a.PredicateFields
	case analysis.OpFindOne:
		result.API = MaskableAPIFindOne
		result.PredicateFields = a.PredicateFields
	case analysis.OpAggregate:
		if a.UnsupportedStage != "" {
			result.API = MaskableAPIUnsupportedRead
			result.UnsupportedStage = a.UnsupportedStage
			return result, nil
		}
		result.API = MaskableAPIAggregate
		result.PredicateFields = a.PredicateFields
		for _, join := range a.Joins {
			result.JoinedCollections = append(result.JoinedCollections, JoinedCollection{
				Collection: join.Collection,
				AsField:    join.AsField,
			})
		}
	default:
		result.API = MaskableAPIUnsupportedRead
	}

	if len(result.PredicateFields) > 0 {
		slices.Sort(result.PredicateFields)
	}

	return result, nil
}
