package cosmosdb

import (
	"context"
	"log/slog"
	"strings"

	omnicosmosdb "github.com/bytebase/omni/cosmosdb"
	"github.com/bytebase/omni/cosmosdb/analysis"
	"github.com/bytebase/omni/cosmosdb/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.Register(base.Engine{
		Split:         SplitSQL,
		ValidateQuery: validateQuery,
		QuerySpan:     GetQuerySpan,
	}, storepb.Engine_COSMOSDB)
}

// AST wraps an omni CosmosDB AST node and implements the base.AST interface.
type AST = base.NodeAST[ast.Node]

func parseCosmosDB(statement string) ([]*AST, error) {
	stmts, err := omnicosmosdb.Parse(statement)
	if err != nil {
		return nil, err
	}

	var result []*AST
	positionMapper := base.NewByteOffsetPositionMapper(statement)
	for _, stmt := range stmts {
		if stmt.Empty() {
			continue
		}
		result = append(result, &AST{
			Node:          stmt.AST,
			Text:          stmt.Text,
			StartPosition: positionMapper.Position(stmt.ByteStart),
		})
	}
	return result, nil
}

// SplitSQL splits the given SQL statement into individual statements.
// CosmosDB only supports single SELECT statements.
func SplitSQL(statement string) ([]base.Statement, error) {
	if strings.TrimSpace(statement) == "" {
		return nil, nil
	}

	stmts, err := omnicosmosdb.Parse(statement)
	if err != nil {
		return nil, err
	}
	return base.BuildStatements(statement, stmts, func(s omnicosmosdb.Statement) (int, int, bool) {
		return s.ByteStart, s.ByteEnd, s.Empty()
	}, false), nil
}

func validateQuery(statement string) (bool, bool, error) {
	_, err := parseCosmosDB(statement)
	if err != nil {
		return false, false, err
	}

	return true, true, nil
}

func GetQuerySpan(_ context.Context, _ base.GetQuerySpanContext, stmt base.Statement, _, _ string, _ bool) (*base.QuerySpan, error) {
	return getQuerySpanImpl(stmt.Text)
}

func getQuerySpanImpl(statement string) (*base.QuerySpan, error) {
	parseResults, err := parseCosmosDB(statement)
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

// fieldPathToPathAST converts an omni analysis.FieldPath to a base.PathAST,
// folding consecutive (property, array-index) selectors into one base.ArraySelector.
func fieldPathToPathAST(fp analysis.FieldPath) *base.PathAST {
	if len(fp) == 0 {
		return nil
	}
	return base.BuildPathAST(mergeSelectors(fp))
}

// mergeSelectors converts omni selectors to base.SelectorNode values,
// merging an array selector into the preceding item selector when possible.
func mergeSelectors(fp analysis.FieldPath) []base.SelectorNode {
	var nodes []base.SelectorNode
	for i := 0; i < len(fp); i++ {
		s := fp[i]
		if s.IsArray() && len(nodes) > 0 {
			prev := nodes[len(nodes)-1]
			nodes[len(nodes)-1] = base.NewArraySelector(prev.GetIdentifier(), s.ArrayIndex)
		} else if s.IsArray() {
			nodes = append(nodes, base.NewArraySelector(s.Name, s.ArrayIndex))
		} else {
			nodes = append(nodes, base.NewItemSelector(s.Name))
		}
	}
	return nodes
}
