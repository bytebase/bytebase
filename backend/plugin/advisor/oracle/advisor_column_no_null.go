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
	_ advisor.Advisor = (*ColumnNoNullAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, advisor.OracleColumnNoNull, &ColumnNoNullAdvisor{})
	advisor.Register(storepb.Engine_DM, advisor.OracleColumnNoNull, &ColumnNoNullAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE_ORACLE, advisor.OracleColumnNoNull, &ColumnNoNullAdvisor{})
}

type columnMap map[string]int

// ColumnNoNullAdvisor is the advisor checking for column no NULL value.
type ColumnNoNullAdvisor struct {
}

// Check checks for column no NULL value.
func (*ColumnNoNullAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	tree, ok := ctx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	listener := &columnNoNullListener{
		level:           level,
		title:           string(ctx.Rule.Type),
		currentSchema:   ctx.CurrentSchema,
		nullableColumns: make(columnMap),
	}

	antlr.ParseTreeWalkerDefault.Walk(listener, tree)

	return listener.generateAdvice()
}

// columnNoNullListener is the listener for column no NULL value.
type columnNoNullListener struct {
	*parser.BasePlSqlParserListener

	level           advisor.Status
	title           string
	currentSchema   string
	nullableColumns columnMap
	tableName       string
	columnID        string
}

func (l *columnNoNullListener) generateAdvice() ([]advisor.Advice, error) {
	advice := []advisor.Advice{}

	var columnIDs []string
	for columnID := range l.nullableColumns {
		columnIDs = append(columnIDs, columnID)
	}
	sort.Strings(columnIDs)
	for _, columnID := range columnIDs {
		line := l.nullableColumns[columnID]
		advice = append(advice, advisor.Advice{
			Status:  l.level,
			Code:    advisor.ColumnCannotNull,
			Title:   l.title,
			Content: fmt.Sprintf("Column %q is nullable, which is not allowed.", lastIdentifier(columnID)),
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
func (l *columnNoNullListener) EnterCreate_table(ctx *parser.Create_tableContext) {
	schemaName := l.currentSchema
	if ctx.Schema_name() != nil {
		schemaName = normalizeIdentifier(ctx.Schema_name(), l.currentSchema)
	}
	l.tableName = fmt.Sprintf("%s.%s", schemaName, normalizeIdentifier(ctx.Table_name(), schemaName))
}

// ExitCreate_table is called when production create_table is exited.
func (l *columnNoNullListener) ExitCreate_table(_ *parser.Create_tableContext) {
	l.tableName = ""
}

// EnterColumn_definition is called when production column_definition is entered.
func (l *columnNoNullListener) EnterColumn_definition(ctx *parser.Column_definitionContext) {
	if l.tableName == "" {
		return
	}
	columnName := normalizeIdentifier(ctx.Column_name(), l.currentSchema)
	l.columnID = fmt.Sprintf(`%s.%s`, l.tableName, columnName)
	l.nullableColumns[l.columnID] = ctx.GetStart().GetLine()
}

// ExitColumn_definition is called when production column_definition is exited.
func (l *columnNoNullListener) ExitColumn_definition(_ *parser.Column_definitionContext) {
	l.columnID = ""
}

// EnterInline_constraint is called when production inline_constraint is entered.
func (l *columnNoNullListener) EnterInline_constraint(ctx *parser.Inline_constraintContext) {
	if l.columnID == "" {
		return
	}
	if ctx.NULL_() != nil {
		l.nullableColumns[l.columnID] = ctx.GetStart().GetLine()
	}
	if ctx.NOT() != nil || ctx.PRIMARY() != nil {
		delete(l.nullableColumns, l.columnID)
	}
}

// EnterOut_of_line_constraint is called when production out_of_line_constraint is entered.
func (l *columnNoNullListener) EnterOut_of_line_constraint(ctx *parser.Out_of_line_constraintContext) {
	if l.tableName == "" {
		return
	}
	if ctx.PRIMARY() != nil {
		for _, column := range ctx.AllColumn_name() {
			columnName := normalizeIdentifier(column, l.currentSchema)
			columnID := fmt.Sprintf(`%s.%s`, l.tableName, columnName)
			delete(l.nullableColumns, columnID)
		}
	}
}

// EnterAlter_table is called when production alter_table is entered.
func (l *columnNoNullListener) EnterAlter_table(ctx *parser.Alter_tableContext) {
	l.tableName = normalizeIdentifier(ctx.Tableview_name(), l.currentSchema)
}

// ExitAlter_table is called when production alter_table is exited.
func (l *columnNoNullListener) ExitAlter_table(_ *parser.Alter_tableContext) {
	l.tableName = ""
}

// EnterModify_col_properties is called when production modify_col_properties is entered.
func (l *columnNoNullListener) EnterModify_col_properties(ctx *parser.Modify_col_propertiesContext) {
	if l.tableName == "" {
		return
	}
	columnName := normalizeIdentifier(ctx.Column_name(), l.currentSchema)
	l.columnID = fmt.Sprintf(`%s.%s`, l.tableName, columnName)
}
