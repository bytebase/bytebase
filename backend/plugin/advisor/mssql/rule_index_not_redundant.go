package mssql

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/mssql/ast"
	mssqlv1review "github.com/bytebase/omni/mssql/v1review"
	"github.com/bytebase/omni/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	tsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_INDEX_NOT_REDUNDANT, &IndexNotRedundantAdvisor{})
}

// IndexNotRedundantAdvisor is the advisor checking for redundant indexes.
// The judgment lives in omni (mssql/v1review.IndexNotRedundant); this adapter
// passes it the synced schema and renders the findings as advice.
type IndexNotRedundantAdvisor struct{}

func (IndexNotRedundantAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	stmts, nodes := scriptNodes(checkCtx.ParsedStatements)
	findings := mssqlv1review.IndexNotRedundant(nodes, v1review.Schema{Metadata: checkCtx.DBSchema}, checkCtx.CurrentDatabase)
	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, f := range findings {
		stmt := stmts[f.Statement]
		adviceList = append(adviceList, &storepb.Advice{
			Status: level,
			Title:  title,
			Code:   code.RedundantIndex.Int32(),
			Content: fmt.Sprintf("Redundant indexes with the same prefix ('%s' and '%s') in '%s.%s' is not allowed",
				f.Other, f.Name, f.Schema, f.Table),
			StartPosition: &storepb.Position{Line: int32(stmt.BaseLine()) + findingLine(stmt.Text, f)},
		})
	}
	return adviceList, nil
}

// scriptNodes returns the statements that hold an omni AST and their nodes,
// index for index.
func scriptNodes(parsed []base.ParsedStatement) ([]base.ParsedStatement, []ast.Node) {
	var stmts []base.ParsedStatement
	var nodes []ast.Node
	for _, stmt := range parsed {
		if stmt.AST == nil {
			continue
		}
		node, ok := tsqlparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		stmts = append(stmts, stmt)
		nodes = append(nodes, node)
	}
	return stmts, nodes
}
