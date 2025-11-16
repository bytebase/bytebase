// Package oracle is the advisor for oracle database.
package oracle

import (
	"context"
	"fmt"
	"slices"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/plsql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	plsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
)

var (
	_ advisor.Advisor = (*ColumnNoNullAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, advisor.SchemaRuleColumnNotNull, &ColumnNoNullAdvisor{})
}

type columnMap map[string]int

// ColumnNoNullAdvisor is the advisor checking for column no NULL value.
type ColumnNoNullAdvisor struct {
}

// Check checks for column no NULL value.
func (*ColumnNoNullAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*plsqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to ParseResult")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := NewColumnNoNullRule(level, string(checkCtx.Rule.Type), checkCtx.CurrentDatabase)
	checker := NewGenericChecker([]Rule{rule})

	for _, stmtNode := range stmtList {
		rule.SetBaseLine(stmtNode.BaseLine)
		checker.SetBaseLine(stmtNode.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmtNode.Tree)
	}

	return checker.GetAdviceList()
}

// ColumnNoNullRule is the rule implementation for column no NULL value.
type ColumnNoNullRule struct {
	BaseRule

	currentDatabase string
	nullableColumns columnMap
	tableName       string
	columnID        string
}

// NewColumnNoNullRule creates a new ColumnNoNullRule.
func NewColumnNoNullRule(level storepb.Advice_Status, title string, currentDatabase string) *ColumnNoNullRule {
	return &ColumnNoNullRule{
		BaseRule:        NewBaseRule(level, title, 0),
		currentDatabase: currentDatabase,
		nullableColumns: make(columnMap),
	}
}

// Name returns the rule name.
func (*ColumnNoNullRule) Name() string {
	return "column.no-null"
}

// OnEnter is called when the parser enters a rule context.
func (r *ColumnNoNullRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Create_table":
		r.handleCreateTable(ctx.(*parser.Create_tableContext))
	case "Column_definition":
		r.handleColumnDefinition(ctx.(*parser.Column_definitionContext))
	case "Inline_constraint":
		r.handleInlineConstraint(ctx.(*parser.Inline_constraintContext))
	case "Out_of_line_constraint":
		r.handleOutOfLineConstraint(ctx.(*parser.Out_of_line_constraintContext))
	case "Alter_table":
		r.handleAlterTable(ctx.(*parser.Alter_tableContext))
	case "Modify_col_properties":
		r.handleModifyColProperties(ctx.(*parser.Modify_col_propertiesContext))
	default:
		// Ignore other node types
	}
	return nil
}

// OnExit is called when the parser exits a rule context.
func (r *ColumnNoNullRule) OnExit(_ antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Create_table":
		r.tableName = ""
	case "Column_definition":
		r.columnID = ""
	case "Alter_table":
		r.tableName = ""
	default:
		// Ignore other node types
	}
	return nil
}

// GetAdviceList returns the advice list.
func (r *ColumnNoNullRule) GetAdviceList() ([]*storepb.Advice, error) {
	var columnIDs []string
	for columnID := range r.nullableColumns {
		columnIDs = append(columnIDs, columnID)
	}
	slices.Sort(columnIDs)
	for _, columnID := range columnIDs {
		line := r.nullableColumns[columnID]
		r.AddAdvice(
			r.level,
			code.ColumnCannotNull.Int32(),
			fmt.Sprintf("Column %q is nullable, which is not allowed.", lastIdentifier(columnID)),
			common.ConvertANTLRLineToPosition(line),
		)
	}
	return r.BaseRule.GetAdviceList()
}

func (r *ColumnNoNullRule) handleCreateTable(ctx *parser.Create_tableContext) {
	schemaName := r.currentDatabase
	if ctx.Schema_name() != nil {
		schemaName = normalizeIdentifier(ctx.Schema_name(), r.currentDatabase)
	}
	r.tableName = fmt.Sprintf("%s.%s", schemaName, normalizeIdentifier(ctx.Table_name(), schemaName))
}

func (r *ColumnNoNullRule) handleColumnDefinition(ctx *parser.Column_definitionContext) {
	if r.tableName == "" {
		return
	}
	columnName := normalizeIdentifier(ctx.Column_name(), r.currentDatabase)
	r.columnID = fmt.Sprintf(`%s.%s`, r.tableName, columnName)
	r.nullableColumns[r.columnID] = r.baseLine + ctx.GetStart().GetLine()
}

func (r *ColumnNoNullRule) handleInlineConstraint(ctx *parser.Inline_constraintContext) {
	if r.columnID == "" {
		return
	}
	if ctx.NULL_() != nil {
		r.nullableColumns[r.columnID] = r.baseLine + ctx.GetStart().GetLine()
	}
	if ctx.NOT() != nil || ctx.PRIMARY() != nil {
		delete(r.nullableColumns, r.columnID)
	}
}

func (r *ColumnNoNullRule) handleOutOfLineConstraint(ctx *parser.Out_of_line_constraintContext) {
	if r.tableName == "" {
		return
	}
	if ctx.PRIMARY() != nil {
		for _, column := range ctx.AllColumn_name() {
			columnName := normalizeIdentifier(column, r.currentDatabase)
			columnID := fmt.Sprintf(`%s.%s`, r.tableName, columnName)
			delete(r.nullableColumns, columnID)
		}
	}
}

func (r *ColumnNoNullRule) handleAlterTable(ctx *parser.Alter_tableContext) {
	r.tableName = normalizeIdentifier(ctx.Tableview_name(), r.currentDatabase)
}

func (r *ColumnNoNullRule) handleModifyColProperties(ctx *parser.Modify_col_propertiesContext) {
	if r.tableName == "" {
		return
	}
	columnName := normalizeIdentifier(ctx.Column_name(), r.currentDatabase)
	r.columnID = fmt.Sprintf(`%s.%s`, r.tableName, columnName)
}
