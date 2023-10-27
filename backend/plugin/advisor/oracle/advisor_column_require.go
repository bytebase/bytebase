// Package oracle is the advisor for oracle database.
package oracle

import (
	"fmt"
	"sort"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/plsql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*ColumnRequireAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, advisor.OracleColumnRequirement, &ColumnRequireAdvisor{})
	advisor.Register(storepb.Engine_DM, advisor.OracleColumnRequirement, &ColumnRequireAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE_ORACLE, advisor.OracleColumnRequirement, &ColumnRequireAdvisor{})
}

// ColumnRequireAdvisor is the advisor checking for column requirement.
type ColumnRequireAdvisor struct {
}

// Check checks for column requirement.
func (*ColumnRequireAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	tree, ok := ctx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	columnList, err := advisor.UnmarshalRequiredColumnList(ctx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	listener := &columnRequireListener{
		level:           level,
		title:           string(ctx.Rule.Type),
		currentSchema:   ctx.CurrentSchema,
		requiredColumns: make(columnSet),
	}

	for _, column := range columnList {
		listener.requiredColumns[column] = true
	}

	antlr.ParseTreeWalkerDefault.Walk(listener, tree)

	return listener.generateAdvice()
}

type columnSet map[string]bool

// columnRequireListener is the listener for column requirement.
type columnRequireListener struct {
	*parser.BasePlSqlParserListener

	level           advisor.Status
	title           string
	currentSchema   string
	requiredColumns columnSet
	missingColumns  columnSet
	adviceList      []advisor.Advice
}

func (l *columnRequireListener) generateAdvice() ([]advisor.Advice, error) {
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
func (l *columnRequireListener) EnterCreate_table(_ *parser.Create_tableContext) {
	l.missingColumns = make(columnSet)
	for column := range l.requiredColumns {
		l.missingColumns[column] = true
	}
}

// ExitCreate_table is called when production create_table is exited.
func (l *columnRequireListener) ExitCreate_table(ctx *parser.Create_tableContext) {
	missingColumns := []string{}
	for column := range l.missingColumns {
		missingColumns = append(missingColumns, fmt.Sprintf("%q", column))
	}
	l.missingColumns = nil

	if len(missingColumns) == 0 {
		return
	}

	sort.Strings(missingColumns)
	tableName := normalizeIdentifier(ctx.Table_name(), l.currentSchema)
	l.adviceList = append(l.adviceList, advisor.Advice{
		Status:  l.level,
		Code:    advisor.NoRequiredColumn,
		Title:   l.title,
		Content: fmt.Sprintf("Table %q requires columns: %s", tableName, strings.Join(missingColumns, ", ")),
		Line:    ctx.GetStop().GetLine(),
	})
}

// EnterColumn_definition is called when production column_name is entered.
func (l *columnRequireListener) EnterColumn_definition(ctx *parser.Column_definitionContext) {
	if ctx.Column_name() == nil || l.missingColumns == nil {
		return
	}
	columnName := normalizeIdentifier(ctx.Column_name(), l.currentSchema)
	delete(l.missingColumns, columnName)
}

// EnterAlter_table is called when production alter_table is entered.
func (l *columnRequireListener) EnterAlter_table(_ *parser.Alter_tableContext) {
	l.missingColumns = make(columnSet)
}

// ExitAlter_table is called when production alter_table is exited.
func (l *columnRequireListener) ExitAlter_table(ctx *parser.Alter_tableContext) {
	missingColumns := []string{}
	for column := range l.missingColumns {
		missingColumns = append(missingColumns, fmt.Sprintf("%q", column))
	}
	l.missingColumns = nil

	if len(missingColumns) == 0 {
		return
	}

	sort.Strings(missingColumns)
	tableName := lastIdentifier(normalizeIdentifier(ctx.Tableview_name(), l.currentSchema))
	l.adviceList = append(l.adviceList, advisor.Advice{
		Status:  l.level,
		Code:    advisor.NoRequiredColumn,
		Title:   l.title,
		Content: fmt.Sprintf("Table %q requires columns: %s", tableName, strings.Join(missingColumns, ", ")),
		Line:    ctx.GetStop().GetLine(),
	})
}

// EnterDrop_column_clause is called when production drop_column_clause is entered.
func (l *columnRequireListener) EnterDrop_column_clause(ctx *parser.Drop_column_clauseContext) {
	if l.missingColumns == nil {
		return
	}
	for _, columnName := range ctx.AllColumn_name() {
		name := normalizeIdentifier(columnName, l.currentSchema)
		if _, exists := l.requiredColumns[name]; exists {
			l.missingColumns[name] = true
		}
	}
}

// EnterRename_column_clause is called when production rename_column_clause is entered.
func (l *columnRequireListener) EnterRename_column_clause(ctx *parser.Rename_column_clauseContext) {
	if l.missingColumns == nil {
		return
	}
	oldName := normalizeIdentifier(ctx.Old_column_name().Column_name(), l.currentSchema)
	newName := normalizeIdentifier(ctx.New_column_name().Column_name(), l.currentSchema)
	if oldName == newName {
		return
	}

	if _, exists := l.requiredColumns[oldName]; exists {
		l.missingColumns[oldName] = true
	}
}
