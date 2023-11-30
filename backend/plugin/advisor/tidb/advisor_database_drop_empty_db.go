package tidb

import (
	"fmt"

	"github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/catalog"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*DatabaseAllowDropIfEmptyAdvisor)(nil)
	_ ast.Visitor     = (*allowDropEmptyDBChecker)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, advisor.MySQLDatabaseAllowDropIfEmpty, &DatabaseAllowDropIfEmptyAdvisor{})
}

// DatabaseAllowDropIfEmptyAdvisor is the advisor checking the MySQLDatabaseAllowDropIfEmpty rule.
type DatabaseAllowDropIfEmptyAdvisor struct {
}

// Check checks for drop table naming convention.
func (*DatabaseAllowDropIfEmptyAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	root, ok := ctx.AST.([]ast.StmtNode)
	if !ok {
		return nil, errors.Errorf("failed to convert to StmtNode")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &allowDropEmptyDBChecker{
		level:   level,
		title:   string(ctx.Rule.Type),
		catalog: ctx.Catalog,
	}
	for _, stmtNode := range root {
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

type allowDropEmptyDBChecker struct {
	adviceList []advisor.Advice
	level      advisor.Status
	title      string
	catalog    *catalog.Finder
}

// Enter implements the ast.Visitor interface.
func (v *allowDropEmptyDBChecker) Enter(in ast.Node) (ast.Node, bool) {
	if node, ok := in.(*ast.DropDatabaseStmt); ok {
		if v.catalog.Origin.DatabaseName() != node.Name.O {
			v.adviceList = append(v.adviceList, advisor.Advice{
				Status:  v.level,
				Code:    advisor.NotCurrentDatabase,
				Title:   v.title,
				Content: fmt.Sprintf("Database `%s` that is trying to be deleted is not the current database `%s`", node.Name, v.catalog.Origin.DatabaseName()),
				Line:    node.OriginTextPosition(),
			})
		} else if !v.catalog.Origin.HasNoTable() {
			v.adviceList = append(v.adviceList, advisor.Advice{
				Status:  v.level,
				Code:    advisor.DatabaseNotEmpty,
				Title:   v.title,
				Content: fmt.Sprintf("Database `%s` is not allowed to drop if not empty", node.Name),
				Line:    node.OriginTextPosition(),
			})
		}
	}
	return in, false
}

// Leave implements the ast.Visitor interface.
func (*allowDropEmptyDBChecker) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}
