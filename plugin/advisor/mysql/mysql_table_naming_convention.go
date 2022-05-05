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
	_ advisor.Advisor = (*TableNamingConventionAdvisor)(nil)
)

func init() {
	advisor.Register(db.MySQL, advisor.MySQLTableNamingConvention, &TableNamingConventionAdvisor{})
	advisor.Register(db.TiDB, advisor.MySQLTableNamingConvention, &TableNamingConventionAdvisor{})
}

// TableNamingConventionAdvisor is the advisor checking for table naming convention.
type TableNamingConventionAdvisor struct {
}

// Check checks for table naming convention.
func (adv *TableNamingConventionAdvisor) Check(ctx advisor.Context, statement string) ([]advisor.Advice, error) {
	root, errAdvice := parseStatement(statement, ctx.Charset, ctx.Collation)
	if errAdvice != nil {
		return errAdvice, nil
	}

	level, err := advisor.NewStatusBySchemaReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	format, err := api.UnmarshalNamingRulePayloadFormat(ctx.Rule.Payload)
	if err != nil {
		return nil, err
	}
	checker := &tableNamingConventionChecker{
		level:  level,
		format: format,
	}
	for _, stmtNode := range root {
		(stmtNode).Accept(checker)
	}

	if len(checker.advisorList) == 0 {
		checker.advisorList = append(checker.advisorList, advisor.Advice{
			Status:  advisor.Success,
			Code:    common.Ok,
			Title:   "OK",
			Content: "",
		})
	}
	return checker.advisorList, nil
}

type tableNamingConventionChecker struct {
	advisorList []advisor.Advice
	level       advisor.Status
	format      *regexp.Regexp
}

func (v *tableNamingConventionChecker) Enter(in ast.Node) (ast.Node, bool) {
	tableNames := make([]string, 0)
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

	code := common.Ok

	for _, tableName := range tableNames {
		if !v.format.MatchString(tableName) {
			code = common.TableNamingConventionMismatch
			break
		}
	}

	if code != common.Ok {
		v.advisorList = append(v.advisorList, advisor.Advice{
			Status:  v.level,
			Code:    code,
			Title:   "Mismatch table naming convention",
			Content: fmt.Sprintf("%q mismatches table naming convention", in.Text()),
		})
	}
	return in, false
}

func (v *tableNamingConventionChecker) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}
