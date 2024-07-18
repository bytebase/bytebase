package tidb

import (
	"fmt"

	"github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*NoSelectAllAdvisor)(nil)
	_ ast.Visitor     = (*noSelectAllChecker)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, advisor.MySQLNoSelectAll, &NoSelectAllAdvisor{})
}

// NoSelectAllAdvisor is the advisor checking for no "select *".
type NoSelectAllAdvisor struct {
}

// Check checks for no "select *".
func (*NoSelectAllAdvisor) Check(ctx advisor.Context, _ string) ([]*storepb.Advice, error) {
	root, ok := ctx.AST.([]ast.StmtNode)
	if !ok {
		return nil, errors.Errorf("failed to convert to StmtNode")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	checker := &noSelectAllChecker{
		level: level,
		title: string(ctx.Rule.Type),
	}
	for _, stmtNode := range root {
		checker.text = stmtNode.Text()
		checker.line = stmtNode.OriginTextPosition()
		(stmtNode).Accept(checker)
	}

	return checker.adviceList, nil
}

type noSelectAllChecker struct {
	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
	text       string
	line       int
}

// Enter implements the ast.Visitor interface.
func (v *noSelectAllChecker) Enter(in ast.Node) (ast.Node, bool) {
	if node, ok := in.(*ast.SelectStmt); ok {
		for _, field := range node.Fields.Fields {
			if field.WildCard != nil {
				v.adviceList = append(v.adviceList, &storepb.Advice{
					Status:  v.level,
					Code:    advisor.StatementSelectAll.Int32(),
					Title:   v.title,
					Content: fmt.Sprintf("\"%s\" uses SELECT all", v.text),
					StartPosition: &storepb.Position{
						Line: int32(v.line),
					},
				})
				break
			}
		}
	}
	return in, false
}

// Leave implements the ast.Visitor interface.
func (*noSelectAllChecker) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}
