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
	_ advisor.Advisor = (*NamingTableConventionAdvisor)(nil)
)

func init() {
	advisor.Register(db.Postgres, advisor.PostgreSQLNamingTableConvention, &NamingTableConventionAdvisor{})
}

// NamingTableConventionAdvisor is the advisor checking for table naming convention.
type NamingTableConventionAdvisor struct {
}

// Check checks for table naming convention.
func (adv *NamingTableConventionAdvisor) Check(ctx advisor.Context, statement string) ([]advisor.Advice, error) {
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
	checker := &namingTableConventionChecker{
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

type namingTableConventionChecker struct {
	adviceList []advisor.Advice
	level      advisor.Status
	title      string
	format     *regexp.Regexp
}

func (checker *namingTableConventionChecker) check(ctx interface{}, node interface{}) (stop bool) {
	var tableNames []string

	switch n := node.(type) {
	// CREATE TABLE
	case *tree.CreateTable:
		tableNames = append(tableNames, n.Table.Table())
	// ALTER TABLE
	case *tree.AlterTable:
		for _, cmd := range n.Cmds {
			if c, ok := cmd.(*tree.AlterTableRenameTable); ok {
				tableNames = append(tableNames, c.NewName.Table())
			}
		}
	// RENAME TABLE
	case *tree.RenameTable:
		if !n.IsSequence && !n.IsView {
			tableNames = append(tableNames, string(n.NewName.ToTableName().TableName))
		}
	}

	for _, tableName := range tableNames {
		if !checker.format.MatchString(tableName) {
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    advisor.NamingTableConventionMismatch,
				Title:   checker.title,
				Content: fmt.Sprintf(`"%s" mismatches table naming convention, naming format should be %q`, tableName, checker.format),
			})
		}
	}

	return false
}
