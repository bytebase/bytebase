package mysql

import (
	"fmt"
	"regexp"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/pingcap/tidb/parser/ast"
)

var (
	_ advisor.Advisor = (*NamingTableConventionAdvisor)(nil)
)

func init() {
	advisor.Register(db.MySQL, advisor.MySQLNamingTableConvention, &NamingTableConventionAdvisor{})
	advisor.Register(db.TiDB, advisor.MySQLNamingTableConvention, &NamingTableConventionAdvisor{})
}

// NamingTableConventionAdvisor is the advisor checking for table naming convention.
type NamingTableConventionAdvisor struct {
}

// Check checks for table naming convention.
func (adv *NamingTableConventionAdvisor) Check(ctx advisor.Context, statement string) ([]advisor.Advice, error) {
	root, errAdvice := parseStatement(statement, ctx.Charset, ctx.Collation)
	if errAdvice != nil {
		return errAdvice, nil
	}

	level, err := advisor.NewStatusBySchemaReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	format, err := api.UnamrshalNamingRulePayloadAsRegexp(ctx.Rule.Payload)
	if err != nil {
		return nil, err
	}
	checker := &namingTableConventionChecker{
		level:  level,
		format: format,
	}
	for _, stmtNode := range root {
		(stmtNode).Accept(checker)
	}

	if len(checker.adviceList) == 0 {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  advisor.Success,
			Code:    common.Ok,
			Title:   "OK",
			Content: "",
		})
	}
	return checker.adviceList, nil
}

type namingTableConventionChecker struct {
	adviceList []advisor.Advice
	level      advisor.Status
	format     *regexp.Regexp
}

// Enter implements the ast.Visitor interface
func (v *namingTableConventionChecker) Enter(in ast.Node) (ast.Node, bool) {
	var tableNames []string
	switch node := in.(type) {
	// CREATE TABLE
	case *ast.CreateTableStmt:
		// Original string
		tableNames = append(tableNames, node.Table.Name.O)
	// ALTER TABLE
	case *ast.AlterTableStmt:
		for _, spec := range node.Specs {
			// RENAME TABLE
			if spec.Tp == ast.AlterTableRenameTable {
				tableNames = append(tableNames, spec.NewTable.Name.O)
			}
		}
	// RENAME TABLE
	case *ast.RenameTableStmt:
		for _, table2Table := range node.TableToTables {
			tableNames = append(tableNames, table2Table.NewTable.Name.O)
		}
	}

	for _, tableName := range tableNames {
		if !v.format.MatchString(tableName) {
			v.adviceList = append(v.adviceList, advisor.Advice{
				Status:  v.level,
				Code:    common.NamingTableConventionMismatch,
				Title:   "Mismatch table naming convention",
				Content: fmt.Sprintf("`%s` mismatches table naming convention, naming format should be %q", tableName, v.format),
			})
		}
	}
	return in, false
}

// Leave implements the ast.Visitor interface
func (v *namingTableConventionChecker) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}
