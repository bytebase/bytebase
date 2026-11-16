package pg

import (
	"context"

	"github.com/bytebase/omni/pg/ast"
	pgv1review "github.com/bytebase/omni/pg/v1review"
	"github.com/bytebase/omni/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

var (
	_ advisor.Advisor = (*IndexConcurrentlyAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_INDEX_CREATE_CONCURRENTLY, &IndexConcurrentlyAdvisor{})
}

// IndexConcurrentlyAdvisor is the advisor checking for to create index
// concurrently. The judgment lives in omni
// (pg/v1review.IndexCreateConcurrently); this adapter passes the script and
// renders the findings as advice.
type IndexConcurrentlyAdvisor struct {
}

// Check checks for to create index concurrently.
func (*IndexConcurrentlyAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	stmts, nodes := scriptNodes(checkCtx.ParsedStatements)
	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, f := range pgv1review.IndexCreateConcurrently(nodes) {
		advice := &storepb.Advice{
			Status:        level,
			Title:         title,
			StartPosition: findingPosition(stmts[f.Statement], f),
		}
		switch f.Code {
		case v1review.IndexCreatedNotConcurrently:
			advice.Code = code.CreateIndexUnconcurrently.Int32()
			advice.Content = "Creating indexes will block writes on the table, unless use CONCURRENTLY"
		case v1review.IndexDroppedNotConcurrently:
			advice.Code = code.DropIndexUnconcurrently.Int32()
			advice.Content = "Droping indexes will block writes on the table, unless use CONCURRENTLY"
		default:
			continue
		}
		adviceList = append(adviceList, advice)
	}
	return adviceList, nil
}

// scriptNodes returns the statements that have an omni AST and their nodes,
// in order, so a finding's Statement indexes both.
func scriptNodes(parsed []base.ParsedStatement) ([]base.ParsedStatement, []ast.Node) {
	var stmts []base.ParsedStatement
	var nodes []ast.Node
	for _, stmt := range parsed {
		if stmt.AST == nil {
			continue
		}
		node, ok := pgparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		stmts = append(stmts, stmt)
		nodes = append(nodes, node)
	}
	return stmts, nodes
}

// findingPosition is where the V1 index rules reported a finding in stmt: the
// line of Search's first occurrence in the statement text, or the text's
// first non-blank line when the finding has no Search or the text lacks it.
func findingPosition(stmt base.ParsedStatement, f v1review.Finding) *storepb.Position {
	return &storepb.Position{
		Line: (&BaseRule{StmtText: stmt.Text}).FindLineByName(f.Search) + int32(stmt.BaseLine()),
	}
}
