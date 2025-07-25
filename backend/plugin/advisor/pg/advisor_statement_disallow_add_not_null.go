package pg

// Framework code is generated by the generator.

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
)

var (
	_ advisor.Advisor = (*StatementDisallowAddNotNullAdvisor)(nil)
	_ ast.Visitor     = (*statementDisallowAddNotNullChecker)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.PostgreSQLDisallowAddNotNull, &StatementDisallowAddNotNullAdvisor{})
}

// StatementDisallowAddNotNullAdvisor is the advisor checking for to disallow add not null.
type StatementDisallowAddNotNullAdvisor struct {
}

// Check checks for to disallow add not null.
func (*StatementDisallowAddNotNullAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]ast.Node)
	if !ok {
		return nil, errors.Errorf("failed to convert to Node")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	checker := &statementDisallowAddNotNullChecker{
		level: level,
		title: string(checkCtx.Rule.Type),
	}

	for _, stmt := range stmtList {
		checker.line = stmt.LastLine()
		ast.Walk(checker, stmt)
	}

	return checker.adviceList, nil
}

type statementDisallowAddNotNullChecker struct {
	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
	line       int
}

// Visit implements ast.Visitor interface.
func (checker *statementDisallowAddNotNullChecker) Visit(in ast.Node) ast.Visitor {
	if node, ok := in.(*ast.SetNotNullStmt); ok {
		checker.adviceList = append(checker.adviceList, &storepb.Advice{
			Status:        checker.level,
			Code:          advisor.StatementAddNotNull.Int32(),
			Title:         checker.title,
			Content:       fmt.Sprintf("Setting NOT NULL will block reads and writes. You can use CHECK (%q IS NOT NULL) instead", node.ColumnName),
			StartPosition: common.ConvertPGParserLineToPosition(checker.line),
		})
	}

	return checker
}
