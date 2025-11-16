// Package oracle is the advisor for oracle database.
package oracle

import (
	"context"
	"fmt"
	"slices"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/plsql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	plsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
)

var (
	_ advisor.Advisor = (*ColumnRequireDefaultAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, advisor.SchemaRuleColumnRequireDefault, &ColumnRequireDefaultAdvisor{})
}

// ColumnRequireDefaultAdvisor is the advisor checking for column default requirement.
type ColumnRequireDefaultAdvisor struct {
}

// Check checks for column default requirement.
func (*ColumnRequireDefaultAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*plsqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to ParseResult")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := NewColumnRequireDefaultRule(level, string(checkCtx.Rule.Type), checkCtx.CurrentDatabase)
	checker := NewGenericChecker([]Rule{rule})

	for _, stmtNode := range stmtList {
		rule.SetBaseLine(stmtNode.BaseLine)
		checker.SetBaseLine(stmtNode.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmtNode.Tree)
	}

	return checker.GetAdviceList()
}

// ColumnRequireDefaultRule is the rule implementation for column default requirement.
type ColumnRequireDefaultRule struct {
	BaseRule

	currentDatabase  string
	noDefaultColumns columnMap
	tableName        string
}

// NewColumnRequireDefaultRule creates a new ColumnRequireDefaultRule.
func NewColumnRequireDefaultRule(level storepb.Advice_Status, title string, currentDatabase string) *ColumnRequireDefaultRule {
	return &ColumnRequireDefaultRule{
		BaseRule:         NewBaseRule(level, title, 0),
		currentDatabase:  currentDatabase,
		noDefaultColumns: make(columnMap),
	}
}

// Name returns the rule name.
func (*ColumnRequireDefaultRule) Name() string {
	return "column.require-default"
}

// OnEnter is called when the parser enters a rule context.
func (r *ColumnRequireDefaultRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Create_table":
		r.handleCreateTable(ctx.(*parser.Create_tableContext))
	case "Column_definition":
		r.handleColumnDefinition(ctx.(*parser.Column_definitionContext))
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
func (r *ColumnRequireDefaultRule) OnExit(_ antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Create_table":
		r.tableName = ""
	case "Alter_table":
		r.tableName = ""
	default:
		// Ignore other node types
	}
	return nil
}

// GetAdviceList returns the advice list.
func (r *ColumnRequireDefaultRule) GetAdviceList() ([]*storepb.Advice, error) {
	var columnIDs []string
	for columnID := range r.noDefaultColumns {
		columnIDs = append(columnIDs, columnID)
	}
	slices.Sort(columnIDs)
	for _, columnID := range columnIDs {
		line := r.noDefaultColumns[columnID]
		r.AddAdvice(
			r.level,
			code.NoDefault.Int32(),
			fmt.Sprintf("Column %q doesn't have default value", lastIdentifier(columnID)),
			common.ConvertANTLRLineToPosition(line),
		)
	}
	return r.BaseRule.GetAdviceList()
}

func (r *ColumnRequireDefaultRule) handleCreateTable(ctx *parser.Create_tableContext) {
	schemaName := r.currentDatabase
	if ctx.Schema_name() != nil {
		schemaName = normalizeIdentifier(ctx.Schema_name(), r.currentDatabase)
	}
	r.tableName = fmt.Sprintf("%s.%s", schemaName, normalizeIdentifier(ctx.Table_name(), schemaName))
}

func (r *ColumnRequireDefaultRule) handleColumnDefinition(ctx *parser.Column_definitionContext) {
	if r.tableName == "" {
		return
	}
	columnName := normalizeIdentifier(ctx.Column_name(), r.currentDatabase)
	columnID := fmt.Sprintf(`%s.%s`, r.tableName, columnName)
	if ctx.DEFAULT() == nil {
		r.noDefaultColumns[columnID] = r.baseLine + ctx.GetStart().GetLine()
	} else {
		delete(r.noDefaultColumns, columnID)
	}
}

func (r *ColumnRequireDefaultRule) handleAlterTable(ctx *parser.Alter_tableContext) {
	r.tableName = normalizeIdentifier(ctx.Tableview_name(), r.currentDatabase)
}

func (r *ColumnRequireDefaultRule) handleModifyColProperties(ctx *parser.Modify_col_propertiesContext) {
	if r.tableName == "" {
		return
	}
	columnName := normalizeIdentifier(ctx.Column_name(), r.currentDatabase)
	if ctx.DEFAULT() != nil {
		columnID := fmt.Sprintf(`%s.%s`, r.tableName, columnName)
		delete(r.noDefaultColumns, columnID)
	}
}
