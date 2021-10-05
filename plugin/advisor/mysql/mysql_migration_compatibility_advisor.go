package mysql

import (
	"fmt"

	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/bytebase/bytebase/plugin/db"

	"github.com/pingcap/parser"
	"github.com/pingcap/parser/ast"
	_ "github.com/pingcap/tidb/types/parser_driver"
)

var (
	_ advisor.Advisor = (*CompatibilityAdvisor)(nil)
)

func init() {
	advisor.Register(db.MySQL, advisor.MySQLMigrationCompatibility, &CompatibilityAdvisor{})
	advisor.Register(db.TiDB, advisor.MySQLMigrationCompatibility, &CompatibilityAdvisor{})
}

type CompatibilityAdvisor struct {
}

// A fake advisor to report 1 advice for each severity.
func (adv *CompatibilityAdvisor) Check(ctx advisor.AdvisorContext, statement string) ([]advisor.Advice, error) {
	p := parser.New()

	root, _, err := p.Parse(statement, ctx.Charset, ctx.Collation)
	if err != nil {
		return []advisor.Advice{
			{
				Status:  advisor.Error,
				Title:   "Syntax error",
				Content: err.Error(),
			},
		}, nil
	}

	c := &compatibilityChecker{}
	for _, stmtNode := range root {
		fmt.Printf("%+v\n", stmtNode)
		(stmtNode).Accept(c)
		fmt.Printf("%+v\n", c)
	}

	if len(c.advisorList) == 0 {
		c.advisorList = append(c.advisorList, advisor.Advice{
			Status:  advisor.Success,
			Title:   "OK",
			Content: "Migration is backward compatible"})
	}
	return c.advisorList, nil
}

type compatibilityChecker struct {
	advisorList []advisor.Advice
}

func (v *compatibilityChecker) Enter(in ast.Node) (ast.Node, bool) {
	if node, ok := in.(*ast.DropTableStmt); ok {
		v.advisorList = append(v.advisorList, advisor.Advice{
			Status:  advisor.Warn,
			Title:   "Incompatible migration",
			Content: fmt.Sprintf("%s is backward incompatible", node.Text()),
		})
	}
	return in, false
}

func (v *compatibilityChecker) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}
