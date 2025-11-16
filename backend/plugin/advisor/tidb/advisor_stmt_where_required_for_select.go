package tidb

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	advisorcode "github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/pingcap/tidb/pkg/parser/ast"
)

var (
	_ advisor.Advisor = (*WhereRequirementForSelectAdvisor)(nil)
	_ ast.Visitor     = (*whereRequirementForSelectChecker)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, advisor.SchemaRuleStatementRequireWhereForSelect, &WhereRequirementForSelectAdvisor{})
}

// WhereRequirementForSelectAdvisor is the advisor checking for the WHERE clause requirement for SELECT statements.
type WhereRequirementForSelectAdvisor struct {
}

// Check checks for the WHERE clause requirement.
func (*WhereRequirementForSelectAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	root, ok := checkCtx.AST.([]ast.StmtNode)
	if !ok {
		return nil, errors.Errorf("failed to convert to StmtNode")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	checker := &whereRequirementForSelectChecker{
		level: level,
		title: string(checkCtx.Rule.Type),
	}
	for _, stmtNode := range root {
		checker.text = stmtNode.Text()
		checker.line = stmtNode.OriginTextPosition()
		(stmtNode).Accept(checker)
	}

	return checker.adviceList, nil
}

type whereRequirementForSelectChecker struct {
	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
	text       string
	line       int
}

// Enter implements the ast.Visitor interface.
func (v *whereRequirementForSelectChecker) Enter(in ast.Node) (ast.Node, bool) {
	code := advisorcode.Ok
	if node, ok := in.(*ast.SelectStmt); ok {
		// Allow SELECT queries without a FROM clause to proceed, e.g. SELECT 1.
		if node.Where == nil && node.From != nil {
			code = advisorcode.StatementNoWhere
		}
	}

	if code != advisorcode.Ok {
		v.adviceList = append(v.adviceList, &storepb.Advice{
			Status:        v.level,
			Code:          code.Int32(),
			Title:         v.title,
			Content:       fmt.Sprintf("\"%s\" requires WHERE clause", v.text),
			StartPosition: common.ConvertANTLRLineToPosition(v.line),
		})
	}
	return in, false
}

// Leave implements the ast.Visitor interface.
func (*whereRequirementForSelectChecker) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}
