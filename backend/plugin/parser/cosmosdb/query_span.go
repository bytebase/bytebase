package cosmosdb

import (
	"context"
	"log/slog"

	"github.com/bytebase/omni/cosmosdb/analysis"
	"github.com/bytebase/omni/cosmosdb/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterGetQuerySpan(storepb.Engine_COSMOSDB, GetQuerySpan)
}

func GetQuerySpan(_ context.Context, _ base.GetQuerySpanContext, stmt base.Statement, _, _ string, _ bool) (*base.QuerySpan, error) {
	return getQuerySpanImpl(stmt.Text)
}

func getQuerySpanImpl(statement string) (*base.QuerySpan, error) {
	parseResults, err := ParseCosmosDB(statement)
	if err != nil {
		return nil, err
	}

	if len(parseResults) == 0 {
		return &base.QuerySpan{
			Results:        []base.QuerySpanResult{},
			PredicatePaths: nil,
		}, nil
	}

	if len(parseResults) != 1 {
		return nil, errors.Errorf("expecting only one statement to get query span, but got %d", len(parseResults))
	}

	selectStmt, ok := parseResults[0].Node.(*ast.SelectStmt)
	if !ok {
		return nil, errors.Errorf("expected SelectStmt, got %T", parseResults[0].Node)
	}

	qa := analysis.Analyze(selectStmt)

	// Convert projections.
	sourceFieldPaths := make(map[string][]*base.PathAST)
	if !qa.SelectStar {
		for _, proj := range qa.Projections {
			for _, fp := range proj.SourcePaths {
				if len(fp) == 0 {
					continue
				}
				sourceFieldPaths[proj.Name] = append(sourceFieldPaths[proj.Name], fieldPathToPathAST(fp))
			}
		}
	}

	result := base.QuerySpanResult{
		SourceFieldPaths: sourceFieldPaths,
		SelectAsterisk:   qa.SelectStar,
	}

	// Convert predicates.
	var predicatePaths map[string]*base.PathAST
	if len(qa.Predicates) > 0 {
		predicatePaths = make(map[string]*base.PathAST)
		for _, fp := range qa.Predicates {
			if len(fp) == 0 {
				continue
			}
			pathAST := fieldPathToPathAST(fp)
			str, err := pathAST.String()
			if err != nil {
				slog.Warn("failed to convert path ast to string", log.BBError(err))
				continue
			}
			predicatePaths[str] = pathAST
		}
	}

	return &base.QuerySpan{
		Results:        []base.QuerySpanResult{result},
		PredicatePaths: predicatePaths,
	}, nil
}

// fieldPathToPathAST converts an omni analysis.FieldPath to a base.PathAST.
//
// The omni analysis represents array access as a separate Selector with
// Name=indexText and ArrayIndex>=0, following the preceding property selector.
// The base.PathAST representation merges these: an ArraySelector carries the
// property name AND the index. So we fold consecutive (ItemSelector, ArraySelector)
// pairs into a single base.ArraySelector.
func fieldPathToPathAST(fp analysis.FieldPath) *base.PathAST {
	if len(fp) == 0 {
		return nil
	}

	// Merge omni selectors into base nodes, folding array indices into the
	// preceding item selector.
	nodes := mergeSelectors(fp)
	if len(nodes) == 0 {
		return nil
	}

	pathAST := base.NewPathAST(nodes[0])
	current := pathAST.Root
	for i := 1; i < len(nodes); i++ {
		current.SetNext(nodes[i])
		current = nodes[i]
	}
	return pathAST
}

// mergeSelectors converts omni selectors to base.SelectorNode values,
// merging an array selector into the preceding item selector when possible.
func mergeSelectors(fp analysis.FieldPath) []base.SelectorNode {
	var nodes []base.SelectorNode
	for i := 0; i < len(fp); i++ {
		s := fp[i]
		if s.IsArray() && len(nodes) > 0 {
			// Merge with the preceding item selector: replace it with an ArraySelector
			// that keeps the property name but adds the array index.
			prev := nodes[len(nodes)-1]
			nodes[len(nodes)-1] = base.NewArraySelector(prev.GetIdentifier(), s.ArrayIndex)
		} else if s.IsArray() {
			// Array selector with no preceding item -- keep as-is (shouldn't normally happen).
			nodes = append(nodes, base.NewArraySelector(s.Name, s.ArrayIndex))
		} else {
			nodes = append(nodes, base.NewItemSelector(s.Name))
		}
	}
	return nodes
}
