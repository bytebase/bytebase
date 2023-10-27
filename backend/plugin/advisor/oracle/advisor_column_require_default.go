// Package oracle is the advisor for oracle database.
package oracle

import (
	"fmt"
	"sort"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/plsql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*ColumnRequireDefaultAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, advisor.OracleRequireColumnDefault, &ColumnRequireDefaultAdvisor{})
	advisor.Register(storepb.Engine_DM, advisor.OracleRequireColumnDefault, &ColumnRequireDefaultAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE_ORACLE, advisor.OracleRequireColumnDefault, &ColumnRequireDefaultAdvisor{})
}

// ColumnRequireDefaultAdvisor is the advisor checking for column default requirement.
type ColumnRequireDefaultAdvisor struct {
}

// Check checks for column default requirement.
func (*ColumnRequireDefaultAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	tree, ok := ctx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	listener := &columnRequireDefaultListener{
		level:            level,
		title:            string(ctx.Rule.Type),
		currentSchema:    ctx.CurrentSchema,
		noDefaultColumns: make(columnMap),
	}

	antlr.ParseTreeWalkerDefault.Walk(listener, tree)

	return listener.generateAdvice()
}

// columnRequireDefaultListener is the listener for column default requirement.
type columnRequireDefaultListener struct {
	*parser.BasePlSqlParserListener

	level            advisor.Status
	title            string
	currentSchema    string
	noDefaultColumns columnMap
	tableName        string
}

func (l *columnRequireDefaultListener) generateAdvice() ([]advisor.Advice, error) {
	advice := []advisor.Advice{}

	var columnIDs []string
	for columnID := range l.noDefaultColumns {
		columnIDs = append(columnIDs, columnID)
	}
	sort.Strings(columnIDs)
	for _, columnID := range columnIDs {
		line := l.noDefaultColumns[columnID]
		advice = append(advice, advisor.Advice{
			Status:  l.level,
			Code:    advisor.NoDefault,
			Title:   l.title,
			Content: fmt.Sprintf("Column %q doesn't have default value", lastIdentifier(columnID)),
			Line:    line,
		})
	}

	if len(advice) == 0 {
		advice = append(advice, advisor.Advice{
			Status:  advisor.Success,
			Code:    advisor.Ok,
			Title:   "OK",
			Content: "",
		})
	}
	return advice, nil
}

// EnterCreate_table is called when production create_table is entered.
func (l *columnRequireDefaultListener) EnterCreate_table(ctx *parser.Create_tableContext) {
	schemaName := l.currentSchema
	if ctx.Schema_name() != nil {
		schemaName = normalizeIdentifier(ctx.Schema_name(), l.currentSchema)
	}
	l.tableName = fmt.Sprintf("%s.%s", schemaName, normalizeIdentifier(ctx.Table_name(), schemaName))
}

// ExitCreate_table is called when production create_table is exited.
func (l *columnRequireDefaultListener) ExitCreate_table(_ *parser.Create_tableContext) {
	l.tableName = ""
}

// EnterColumn_definition is called when production column_definition is entered.
func (l *columnRequireDefaultListener) EnterColumn_definition(ctx *parser.Column_definitionContext) {
	if l.tableName == "" {
		return
	}
	columnName := normalizeIdentifier(ctx.Column_name(), l.currentSchema)
	columnID := fmt.Sprintf(`%s.%s`, l.tableName, columnName)
	if ctx.DEFAULT() == nil {
		l.noDefaultColumns[columnID] = ctx.GetStart().GetLine()
	} else {
		delete(l.noDefaultColumns, columnID)
	}
}

// EnterAlter_table is called when production alter_table is entered.
func (l *columnRequireDefaultListener) EnterAlter_table(ctx *parser.Alter_tableContext) {
	l.tableName = normalizeIdentifier(ctx.Tableview_name(), l.currentSchema)
}

// ExitAlter_table is called when production alter_table is exited.
func (l *columnRequireDefaultListener) ExitAlter_table(_ *parser.Alter_tableContext) {
	l.tableName = ""
}

// EnterModify_col_properties is called when production modify_col_properties is entered.
func (l *columnRequireDefaultListener) EnterModify_col_properties(ctx *parser.Modify_col_propertiesContext) {
	if l.tableName == "" {
		return
	}
	columnName := normalizeIdentifier(ctx.Column_name(), l.currentSchema)
	if ctx.DEFAULT() != nil {
		columnID := fmt.Sprintf(`%s.%s`, l.tableName, columnName)
		delete(l.noDefaultColumns, columnID)
	}
}
