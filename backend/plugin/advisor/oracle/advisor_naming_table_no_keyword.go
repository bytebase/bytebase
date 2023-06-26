// Package oracle is the advisor for oracle database.
package oracle

import (
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/plsql-parser"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/db"
	bbparser "github.com/bytebase/bytebase/backend/plugin/parser/sql"
)

var (
	_ advisor.Advisor = (*NamingTableNoKeywordAdvisor)(nil)
)

func init() {
	advisor.Register(db.Oracle, advisor.OracleTableNamingNoKeyword, &NamingTableNoKeywordAdvisor{})
}

// NamingTableNoKeywordAdvisor is the advisor checking for table naming convention without keyword.
type NamingTableNoKeywordAdvisor struct {
}

// Check checks for table naming convention without keyword.
func (*NamingTableNoKeywordAdvisor) Check(ctx advisor.Context, statement string) ([]advisor.Advice, error) {
	tree, ok := ctx.AST.(antlr.Tree)
	if !ok {
		return nil, fmt.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	listener := &namingTableNoKeywordListener{
		level:         level,
		title:         string(ctx.Rule.Type),
		currentSchema: ctx.CurrentSchema,
	}

	antlr.ParseTreeWalkerDefault.Walk(listener, tree)

	return listener.generateAdvice()
}

// namingTableNoKeywordListener is the listener for table naming convention without keyword.
type namingTableNoKeywordListener struct {
	*parser.BasePlSqlParserListener

	level         advisor.Status
	title         string
	currentSchema string
	adviceList    []advisor.Advice
}

func (l *namingTableNoKeywordListener) generateAdvice() ([]advisor.Advice, error) {
	if len(l.adviceList) == 0 {
		l.adviceList = append(l.adviceList, advisor.Advice{
			Status:  advisor.Success,
			Code:    advisor.Ok,
			Title:   "OK",
			Content: "",
		})
	}
	return l.adviceList, nil
}

// EnterCreate_table is called when production create_table is entered.
func (l *namingTableNoKeywordListener) EnterCreate_table(ctx *parser.Create_tableContext) {
	tableName := normalizeIdentifier(ctx.Table_name(), l.currentSchema)
	if bbparser.IsOracleKeyword(tableName) {
		l.adviceList = append(l.adviceList, advisor.Advice{
			Status:  l.level,
			Code:    advisor.NameIsKeywordIdentifier,
			Title:   l.title,
			Content: fmt.Sprintf("Table name %q is a keyword identifier and should be avoided.", tableName),
			Line:    ctx.GetStart().GetLine(),
		})
	}
}

// EnterAlter_table_properties is called when production alter_table_properties is entered.
func (l *namingTableNoKeywordListener) EnterAlter_table_properties(ctx *parser.Alter_table_propertiesContext) {
	if ctx.Tableview_name() == nil {
		return
	}
	tableName := lastIdentifier(normalizeIdentifier(ctx.Tableview_name(), l.currentSchema))
	if bbparser.IsOracleKeyword(tableName) {
		l.adviceList = append(l.adviceList, advisor.Advice{
			Status:  l.level,
			Code:    advisor.NameIsKeywordIdentifier,
			Title:   l.title,
			Content: fmt.Sprintf("Table name %q is a keyword identifier and should be avoided.", tableName),
			Line:    ctx.GetStart().GetLine(),
		})
	}
}
