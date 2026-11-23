package mongodb

import (
	"context"
	"strings"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterGetQuerySpan(storepb.Engine_MONGODB, GetQuerySpan)
}

// GetQuerySpan returns a QuerySpan for a MongoDB shell statement.
// It classifies the statement type and populates PredicatePaths from
// predicate fields extracted by masking analysis.
func GetQuerySpan(_ context.Context, _ base.GetQuerySpanContext, stmt base.Statement, _ string, _ string, _ bool) (*base.QuerySpan, error) {
	queryType := getQueryType(stmt.Text)
	span := &base.QuerySpan{
		Type: queryType,
	}

	analysis, err := analyzeMaskingStatement(stmt.Text)
	if err != nil {
		return span, err
	}
	if analysis == nil {
		return span, nil
	}

	span.MongoDBAnalysis = analysis

	if len(analysis.PredicateFields) > 0 {
		span.PredicatePaths = make(map[string]*base.PathAST, len(analysis.PredicateFields))
		for _, field := range analysis.PredicateFields {
			span.PredicatePaths[field] = dotPathToPathAST(field)
		}
	}

	return span, nil
}

// dotPathToPathAST converts a dot-delimited field path (e.g. "contact.phone")
// into a PathAST with ItemSelector nodes.
func dotPathToPathAST(dotPath string) *base.PathAST {
	parts := strings.Split(dotPath, ".")
	nodes := make([]base.SelectorNode, len(parts))
	for i, part := range parts {
		nodes[i] = base.NewItemSelector(part)
	}
	return base.BuildPathAST(nodes)
}
