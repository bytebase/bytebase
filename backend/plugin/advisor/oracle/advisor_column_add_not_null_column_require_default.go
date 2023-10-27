// Package oracle is the advisor for oracle database.
package oracle

import (
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/plsql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*ColumnAddNotNullColumnRequireDefaultAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, advisor.OracleAddNotNullColumnRequireDefault, &ColumnAddNotNullColumnRequireDefaultAdvisor{})
	advisor.Register(storepb.Engine_DM, advisor.OracleAddNotNullColumnRequireDefault, &ColumnAddNotNullColumnRequireDefaultAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE_ORACLE, advisor.OracleAddNotNullColumnRequireDefault, &ColumnAddNotNullColumnRequireDefaultAdvisor{})
}

// ColumnAddNotNullColumnRequireDefaultAdvisor is the advisor checking for adding not null column requires default.
type ColumnAddNotNullColumnRequireDefaultAdvisor struct {
}

// Check checks for adding not null column requires default.
func (*ColumnAddNotNullColumnRequireDefaultAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	tree, ok := ctx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	listener := &columnAddNotNullColumnRequireDefaultListener{
		level:         level,
		title:         string(ctx.Rule.Type),
		currentSchema: ctx.CurrentSchema,
	}

	antlr.ParseTreeWalkerDefault.Walk(listener, tree)

	return listener.generateAdvice()
}

// columnAddNotNullColumnRequireDefaultListener is the listener for adding not null column requires default.
type columnAddNotNullColumnRequireDefaultListener struct {
	*parser.BasePlSqlParserListener

	level         advisor.Status
	title         string
	currentSchema string
	tableName     string
	isNotNull     bool
	adviceList    []advisor.Advice
}

func (l *columnAddNotNullColumnRequireDefaultListener) generateAdvice() ([]advisor.Advice, error) {
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

// EnterAlter_table is called when production alter_table is entered.
func (l *columnAddNotNullColumnRequireDefaultListener) EnterAlter_table(ctx *parser.Alter_tableContext) {
	l.tableName = normalizeIdentifier(ctx.Tableview_name(), l.currentSchema)
}

// ExitAlter_table is called when production alter_table is exited.
func (l *columnAddNotNullColumnRequireDefaultListener) ExitAlter_table(_ *parser.Alter_tableContext) {
	l.tableName = ""
}

// EnterInline_constraint is called when production inline_constraint is entered.
func (l *columnAddNotNullColumnRequireDefaultListener) EnterInline_constraint(ctx *parser.Inline_constraintContext) {
	if ctx.NOT() != nil {
		l.isNotNull = true
	}
}

// EnterColumn_definition is called when production column_definition is entered.
func (l *columnAddNotNullColumnRequireDefaultListener) EnterColumn_definition(_ *parser.Column_definitionContext) {
	l.isNotNull = false
}

// ExitColumn_definition is called when production column_definition is entered.
func (l *columnAddNotNullColumnRequireDefaultListener) ExitColumn_definition(ctx *parser.Column_definitionContext) {
	if l.tableName == "" || !l.isNotNull {
		return
	}

	if ctx.DEFAULT() == nil {
		l.adviceList = append(l.adviceList, advisor.Advice{
			Status:  l.level,
			Code:    advisor.NotNullColumnWithNoDefault,
			Title:   l.title,
			Content: fmt.Sprintf("Adding not null column %q requires default.", normalizeIdentifier(ctx.Column_name(), l.currentSchema)),
			Line:    ctx.GetStart().GetLine(),
		})
	}
}
