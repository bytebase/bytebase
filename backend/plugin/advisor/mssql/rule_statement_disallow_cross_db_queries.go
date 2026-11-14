package mssql

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/mssql/ast"
	mssqlv1review "github.com/bytebase/omni/mssql/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	tsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_STATEMENT_DISALLOW_CROSS_DB_QUERIES, &DisallowCrossDBQueriesAdvisor{})
}

// DisallowCrossDBQueriesAdvisor is the advisor disallowing cross-database
// queries. The judgment lives in omni
// (mssql/v1review.StatementDisallowCrossDBQueries); this adapter renders its
// findings as advice.
type DisallowCrossDBQueriesAdvisor struct{}

func (*DisallowCrossDBQueriesAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// A USE in one statement changes the current database for the ones after
	// it, so every statement is judged in one call.
	var nodes []ast.Node
	var parsed []base.ParsedStatement
	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := tsqlparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		nodes = append(nodes, node)
		parsed = append(parsed, stmt)
	}

	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, f := range mssqlv1review.StatementDisallowCrossDBQueries(nodes, checkCtx.CurrentDatabase) {
		stmt := parsed[f.Statement]
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Code:          code.StatementDisallowCrossDBQueries.Int32(),
			Title:         title,
			Content:       fmt.Sprintf("Cross database queries (target databse: '%s', current database: '%s') are prohibited", f.Name, f.Value),
			StartPosition: &storepb.Position{Line: int32(stmt.BaseLine()) + findingLine(stmt.Text, f)},
		})
	}
	return adviceList, nil
}
