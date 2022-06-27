package pg

import (
	"fmt"
	"regexp"

	"github.com/auxten/postgresql-parser/pkg/sql/parser"
	"github.com/auxten/postgresql-parser/pkg/sql/sem/tree"
	"github.com/auxten/postgresql-parser/pkg/walk"
	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/bytebase/bytebase/plugin/db"
)

var (
	_ advisor.Advisor = (*NamingColumnConventionAdvisor)(nil)
)

func init() {
	advisor.Register(db.Postgres, advisor.PostgreSQLNamingColumnConvention, &NamingColumnConventionAdvisor{})
}

// NamingColumnConventionAdvisor is the advisor checking for column convention.
type NamingColumnConventionAdvisor struct {
}

// Check checks for column naming convention.
func (adv *NamingColumnConventionAdvisor) Check(ctx advisor.Context, statement string) ([]advisor.Advice, error) {
	stmts, errAdvice := parseStatement(statement)
	if errAdvice != nil {
		return errAdvice, nil
	}

	level, err := advisor.NewStatusBySchemaReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	format, err := advisor.UnamrshalNamingRulePayloadAsRegexp(ctx.Rule.Payload)
	if err != nil {
		return nil, err
	}
	checker := &namingColumnConventionChecker{
		level:  level,
		title:  string(ctx.Rule.Type),
		format: format,
	}

	walker := &walk.AstWalker{
		Fn: checker.check,
	}

	for _, stmt := range stmts {
		if _, err := walker.Walk(parser.Statements{stmt}, nil); err != nil {
			return nil, err
		}
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

type namingColumnConventionChecker struct {
	adviceList []advisor.Advice
	level      advisor.Status
	title      string
	format     *regexp.Regexp
}

func (checker *namingColumnConventionChecker) check(ctx interface{}, node interface{}) (stop bool) {
	var columnList []string
	var tableName string

	switch n := node.(type) {
	// CREATE TABLE
	case *tree.CreateTable:
		tableName = n.Table.Table()
		for _, def := range n.Defs {
			if column, ok := def.(*tree.ColumnTableDef); ok {
				columnList = append(columnList, string(column.Name))
			}
		}
	// ALTER TABLE
	case *tree.AlterTable:
		tableName = string(n.Table.ToTableName().TableName)
		for _, cmd := range n.Cmds {
			switch c := cmd.(type) {
			// ADD COLUMN
			case *tree.AlterTableAddColumn:
				columnList = append(columnList, string(c.ColumnDef.Name))
			// RENAME COLUMN
			case *tree.AlterTableRenameColumn:
				columnList = append(columnList, string(c.NewName))
			}
		}
	}

	for _, column := range columnList {
		if !checker.format.MatchString(column) {
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    advisor.NamingColumnConventionMismatch,
				Title:   checker.title,
				Content: fmt.Sprintf("\"%s\".\"%s\" mismatches column naming convention, naming format should be %q", tableName, column, checker.format),
			})
		}
	}

	return false
}
