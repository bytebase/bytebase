package mysql

import (
	"context"
	"fmt"
	"log"

	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/bytebase/bytebase/plugin/advisor/catalog"
	"github.com/pingcap/tidb/parser/ast"
)

var (
	_ advisor.Advisor = (*DatabaseAllowDropIfEmptyAdvisor)(nil)
)

func init() {
	advisor.Register(advisor.MySQL, advisor.MySQLDatabaseAllowDropIfEmpty, &DatabaseAllowDropIfEmptyAdvisor{})
	advisor.Register(advisor.TiDB, advisor.MySQLDatabaseAllowDropIfEmpty, &DatabaseAllowDropIfEmptyAdvisor{})
}

// DatabaseAllowDropIfEmptyAdvisor is the advisor checking the MySQLDatabaseAllowDropIfEmpty rule.
type DatabaseAllowDropIfEmptyAdvisor struct {
}

// Check checks for drop table naming convention.
func (adv *DatabaseAllowDropIfEmptyAdvisor) Check(ctx advisor.Context, statement string) ([]advisor.Advice, error) {
	root, errAdvice := parseStatement(statement, ctx.Charset, ctx.Collation)
	if errAdvice != nil {
		return errAdvice, nil
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
	catalog    catalog.Catalog
}

// Enter implements the ast.Visitor interface
func (v *allowDropEmptyDBChecker) Enter(in ast.Node) (ast.Node, bool) {
	if node, ok := in.(*ast.DropDatabaseStmt); ok {
		ctx := context.Background()
		tableList, err := v.catalog.FindTable(ctx, &catalog.TableFind{
			DatabaseName: node.Name,
		})

		if err != nil {
			log.Printf(
				"Cannot find table in database %s with error %v\n",
				node.Name,
				err,
			)
		}
		if len(tableList) > 0 {
			v.adviceList = append(v.adviceList, advisor.Advice{
				Status:  v.level,
				Code:    advisor.DatabaseNotEmpty,
				Title:   v.title,
				Content: fmt.Sprintf("Database `%s` is not allowed to drop if not empty", node.Name),
			})
		}
	}
	return in, false
}

// Leave implements the ast.Visitor interface
func (v *allowDropEmptyDBChecker) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}
