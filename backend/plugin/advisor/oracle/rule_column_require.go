// Package oracle is the advisor for oracle database.
package oracle

import (
	"context"
	"fmt"
	"slices"
	"strings"

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
	_ advisor.Advisor = (*ColumnRequireAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, advisor.SchemaRuleRequiredColumn, &ColumnRequireAdvisor{})
}

// ColumnRequireAdvisor is the advisor checking for column requirement.
type ColumnRequireAdvisor struct {
}

// Check checks for column requirement.
func (*ColumnRequireAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*plsqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to ParseResult")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	columnList, err := advisor.UnmarshalRequiredColumnList(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	rule := NewColumnRequireRule(level, string(checkCtx.Rule.Type), checkCtx.CurrentDatabase, columnList)
	checker := NewGenericChecker([]Rule{rule})

	for _, stmtNode := range stmtList {
		rule.SetBaseLine(stmtNode.BaseLine)
		checker.SetBaseLine(stmtNode.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmtNode.Tree)
	}

	return checker.GetAdviceList()
}

type columnSet map[string]bool

// ColumnRequireRule is the rule implementation for column requirement.
type ColumnRequireRule struct {
	BaseRule

	currentDatabase string
	requiredColumns columnSet
	missingColumns  columnSet
}

// NewColumnRequireRule creates a new ColumnRequireRule.
func NewColumnRequireRule(level storepb.Advice_Status, title string, currentDatabase string, columnList []string) *ColumnRequireRule {
	requiredColumns := make(columnSet)
	for _, column := range columnList {
		requiredColumns[column] = true
	}
	return &ColumnRequireRule{
		BaseRule:        NewBaseRule(level, title, 0),
		currentDatabase: currentDatabase,
		requiredColumns: requiredColumns,
	}
}

// Name returns the rule name.
func (*ColumnRequireRule) Name() string {
	return "column.require"
}

// OnEnter is called when the parser enters a rule context.
func (r *ColumnRequireRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Create_table":
		r.handleCreateTableEnter(ctx.(*parser.Create_tableContext))
	case "Column_definition":
		r.handleColumnDefinition(ctx.(*parser.Column_definitionContext))
	case "Alter_table":
		r.handleAlterTableEnter(ctx.(*parser.Alter_tableContext))
	case "Drop_column_clause":
		r.handleDropColumnClause(ctx.(*parser.Drop_column_clauseContext))
	case "Rename_column_clause":
		r.handleRenameColumnClause(ctx.(*parser.Rename_column_clauseContext))
	default:
		// Ignore other node types
	}
	return nil
}

// OnExit is called when the parser exits a rule context.
func (r *ColumnRequireRule) OnExit(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Create_table":
		r.handleCreateTableExit(ctx.(*parser.Create_tableContext))
	case "Alter_table":
		r.handleAlterTableExit(ctx.(*parser.Alter_tableContext))
	default:
		// Ignore other node types
	}
	return nil
}

func (r *ColumnRequireRule) handleCreateTableEnter(_ *parser.Create_tableContext) {
	r.missingColumns = make(columnSet)
	for column := range r.requiredColumns {
		r.missingColumns[column] = true
	}
}

func (r *ColumnRequireRule) handleCreateTableExit(ctx *parser.Create_tableContext) {
	missingColumns := []string{}
	for column := range r.missingColumns {
		missingColumns = append(missingColumns, fmt.Sprintf("%q", column))
	}
	r.missingColumns = nil

	if len(missingColumns) == 0 {
		return
	}

	slices.Sort(missingColumns)
	tableName := normalizeIdentifier(ctx.Table_name(), r.currentDatabase)
	r.AddAdvice(
		r.level,
		code.NoRequiredColumn.Int32(),
		fmt.Sprintf("Table %q requires columns: %s", tableName, strings.Join(missingColumns, ", ")),
		common.ConvertANTLRLineToPosition(r.baseLine+ctx.GetStop().GetLine()),
	)
}

func (r *ColumnRequireRule) handleColumnDefinition(ctx *parser.Column_definitionContext) {
	if ctx.Column_name() == nil || r.missingColumns == nil {
		return
	}
	columnName := normalizeIdentifier(ctx.Column_name(), r.currentDatabase)
	delete(r.missingColumns, columnName)
}

func (r *ColumnRequireRule) handleAlterTableEnter(_ *parser.Alter_tableContext) {
	r.missingColumns = make(columnSet)
}

func (r *ColumnRequireRule) handleAlterTableExit(ctx *parser.Alter_tableContext) {
	missingColumns := []string{}
	for column := range r.missingColumns {
		missingColumns = append(missingColumns, fmt.Sprintf("%q", column))
	}
	r.missingColumns = nil

	if len(missingColumns) == 0 {
		return
	}

	slices.Sort(missingColumns)
	tableName := lastIdentifier(normalizeIdentifier(ctx.Tableview_name(), r.currentDatabase))
	r.AddAdvice(
		r.level,
		code.NoRequiredColumn.Int32(),
		fmt.Sprintf("Table %q requires columns: %s", tableName, strings.Join(missingColumns, ", ")),
		common.ConvertANTLRLineToPosition(r.baseLine+ctx.GetStop().GetLine()),
	)
}

func (r *ColumnRequireRule) handleDropColumnClause(ctx *parser.Drop_column_clauseContext) {
	if r.missingColumns == nil {
		return
	}
	for _, columnName := range ctx.AllColumn_name() {
		name := normalizeIdentifier(columnName, r.currentDatabase)
		if _, exists := r.requiredColumns[name]; exists {
			r.missingColumns[name] = true
		}
	}
}

func (r *ColumnRequireRule) handleRenameColumnClause(ctx *parser.Rename_column_clauseContext) {
	if r.missingColumns == nil {
		return
	}
	oldName := normalizeIdentifier(ctx.Old_column_name().Column_name(), r.currentDatabase)
	newName := normalizeIdentifier(ctx.New_column_name().Column_name(), r.currentDatabase)
	if oldName == newName {
		return
	}

	if _, exists := r.requiredColumns[oldName]; exists {
		r.missingColumns[oldName] = true
	}
}
