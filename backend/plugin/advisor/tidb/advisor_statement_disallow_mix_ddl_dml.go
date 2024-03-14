package tidb

import (
	"github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*StatementDisallowmixDDLDMLAdvisor)(nil)
	_ ast.Visitor     = (*statementDisallowMixDDLDMLChecker)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, advisor.MySQLStatementDisallowMixDDLDML, &StatementDisallowmixDDLDMLAdvisor{})
}

// StatementDisallowmixDDLDMLAdvisor is the advisor checking for no mixed DDL and DML.
type StatementDisallowmixDDLDMLAdvisor struct {
}

// Check checks for no mixed DDL and DML.
func (*StatementDisallowmixDDLDMLAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	// TODO(p0ny): implement it.
	root, ok := ctx.AST.([]ast.StmtNode)
	if !ok {
		return nil, errors.Errorf("failed to convert to StmtNode")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	checker := &statementDisallowMixDDLDMLChecker{
		level: level,
		title: string(ctx.Rule.Type),
	}
	for _, stmtNode := range root {
		checker.text = stmtNode.Text()
		checker.line = stmtNode.OriginTextPosition()
		(stmtNode).Accept(checker)
	}

	if len(checker.adviceList) == 0 {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  advisor.Success,
			Code:    advisor.Ok,
			Title:   "OK",
			Content: "",
		})
	}
	return checker.adviceList, nil
}

type statementDisallowMixDDLDMLChecker struct {
	adviceList []advisor.Advice
	level      advisor.Status
	title      string
	text       string
	line       int
}

// Enter implements the ast.Visitor interface.
func (*statementDisallowMixDDLDMLChecker) Enter(in ast.Node) (ast.Node, bool) {
	return in, false
}

// Leave implements the ast.Visitor interface.
func (*statementDisallowMixDDLDMLChecker) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}
