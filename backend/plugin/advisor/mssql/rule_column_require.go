package mssql

import (
	"context"
	"fmt"
	"slices"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/tsql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	tsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
)

var (
	_ advisor.Advisor = (*ColumnRequireAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, advisor.SchemaRuleRequiredColumn, &ColumnRequireAdvisor{})
}

// ColumnRequireAdvisor is the advisor checking for column requirement..
type ColumnRequireAdvisor struct {
}

// Check checks for column requirement..
func (*ColumnRequireAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, ok := checkCtx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	columnList, err := advisor.UnmarshalRequiredColumnList(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewColumnRequireRule(level, string(checkCtx.Rule.Type), columnList)

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	antlr.ParseTreeWalkerDefault.Walk(checker, tree)

	return checker.GetAdviceList(), nil
}

// ColumnRequireRule checks for column requirement.
type ColumnRequireRule struct {
	BaseRule

	// requireColumns is the required columns, the key is the normalized column name.
	requireColumns map[string]any

	// The following variables should be clean up when ENTER some statement.
	//
	// currentMissingColumn is the missing column, the key is the normalized column name.
	currentMissingColumn map[string]any
	// currentOriginalTableName is the original table name, should be reset when QUIT some statement.
	currentOriginalTableName string
}

// NewColumnRequireRule creates a new ColumnRequireRule.
func NewColumnRequireRule(level storepb.Advice_Status, title string, columnList []string) *ColumnRequireRule {
	requireColumns := make(map[string]any)
	for _, column := range columnList {
		requireColumns[column] = true
	}

	return &ColumnRequireRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		requireColumns: requireColumns,
	}
}

// Name returns the rule name.
func (*ColumnRequireRule) Name() string {
	return "ColumnRequireRule"
}

// OnEnter is called when entering a parse tree node.
func (r *ColumnRequireRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeCreateTable:
		r.enterCreateTable(ctx.(*parser.Create_tableContext))
	case NodeTypeColumnDefinition:
		r.enterColumnDefinition(ctx.(*parser.Column_definitionContext))
	case NodeTypeAlterTable:
		r.enterAlterTable(ctx.(*parser.Alter_tableContext))
	default:
		// Ignore other node types
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (r *ColumnRequireRule) OnExit(ctx antlr.ParserRuleContext, nodeType string) error {
	if nodeType == NodeTypeCreateTable {
		r.exitCreateTable(ctx.(*parser.Create_tableContext))
	}
	return nil
}

func (r *ColumnRequireRule) enterCreateTable(ctx *parser.Create_tableContext) {
	r.currentOriginalTableName = ctx.Table_name().GetText()
	r.currentMissingColumn = make(map[string]any)
	for column := range r.requireColumns {
		r.currentMissingColumn[column] = true
	}
}

func (r *ColumnRequireRule) enterColumnDefinition(ctx *parser.Column_definitionContext) {
	if r.currentOriginalTableName == "" {
		return
	}

	normalizedColumnName := normalizeMSSQLIdentifier(ctx.Id_())
	delete(r.currentMissingColumn, normalizedColumnName)
}

func (r *ColumnRequireRule) exitCreateTable(ctx *parser.Create_tableContext) {
	defer func() {
		r.currentOriginalTableName = ""
		r.currentMissingColumn = nil
	}()

	if r.currentOriginalTableName == "" {
		return
	}

	if len(r.currentMissingColumn) > 0 {
		missingColumns := []string{}
		for column := range r.currentMissingColumn {
			missingColumns = append(missingColumns, column)
		}
		slices.Sort(missingColumns)

		// Generate individual advisor messages for each missing column
		for _, column := range missingColumns {
			r.AddAdvice(&storepb.Advice{
				Status:        r.level,
				Code:          code.NoRequiredColumn.Int32(),
				Title:         r.title,
				Content:       fmt.Sprintf("Table %s missing required column \"%s\"", r.currentOriginalTableName, column),
				StartPosition: common.ConvertANTLRLineToPosition(ctx.GetStart().GetLine()),
			})
		}
	}
}

func (r *ColumnRequireRule) enterAlterTable(ctx *parser.Alter_tableContext) {
	if ctx.DROP() == nil || ctx.COLUMN() == nil {
		return
	}

	tableName := ctx.Table_name(0).GetText()
	allColumnNames := ctx.AllId_()
	for _, columnName := range allColumnNames {
		_, normalizedColumnName := tsqlparser.NormalizeTSQLIdentifier(columnName)
		if _, ok := r.requireColumns[normalizedColumnName]; ok {
			r.AddAdvice(&storepb.Advice{
				Status:        r.level,
				Code:          code.NoRequiredColumn.Int32(),
				Title:         r.title,
				Content:       fmt.Sprintf("Table %s missing required column \"%s\"", tableName, normalizedColumnName),
				StartPosition: common.ConvertANTLRLineToPosition(ctx.GetStart().GetLine()),
			})
		}
	}
}

func normalizeMSSQLIdentifier(idCtx parser.IId_Context) string {
	if idCtx == nil {
		return ""
	}
	_, name := tsqlparser.NormalizeTSQLIdentifier(idCtx)
	return name
}
