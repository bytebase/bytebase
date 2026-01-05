// Package snowflake is the advisor for snowflake database.
package snowflake

import (
	"context"
	"fmt"
	"slices"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/snowflake"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	snowsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/snowflake"
)

var (
	_ advisor.Advisor = (*ColumnRequireAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_SNOWFLAKE, storepb.SQLReviewRule_COLUMN_REQUIRED, &ColumnRequireAdvisor{})
}

// ColumnRequireAdvisor is the advisor checking for column requirement.
type ColumnRequireAdvisor struct {
}

// Check checks for column requirement.
func (*ColumnRequireAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	stringArrayPayload := checkCtx.Rule.GetStringArrayPayload()
	if stringArrayPayload == nil {
		return nil, errors.New("string_array_payload is required for column required rule")
	}

	requireColumns := make(map[string]any)
	for _, column := range stringArrayPayload.List {
		requireColumns[column] = true
	}

	rule := NewColumnRequireRule(level, checkCtx.Rule.Type.String(), requireColumns)
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		antlrAST, ok := base.GetANTLRAST(stmt.AST)
		if !ok {
			continue
		}
		rule.SetBaseLine(stmt.BaseLine())
		checker.SetBaseLine(stmt.BaseLine())
		antlr.ParseTreeWalkerDefault.Walk(checker, antlrAST.Tree)
	}

	return checker.GetAdviceList(), nil
}

// ColumnRequireRule checks for required columns.
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
func NewColumnRequireRule(level storepb.Advice_Status, title string, requireColumns map[string]any) *ColumnRequireRule {
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
	case NodeTypeColumnDeclItemList:
		r.enterColumnDeclItemList(ctx.(*parser.Column_decl_item_listContext))
	case NodeTypeAlterTable:
		r.enterAlterTable(ctx.(*parser.Alter_tableContext))
	case NodeTypeTableColumnAction:
		r.enterTableColumnAction(ctx.(*parser.Table_column_actionContext))
	default:
		// Ignore other node types
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (r *ColumnRequireRule) OnExit(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeCreateTable:
		r.exitCreateTable(ctx.(*parser.Create_tableContext))
	case NodeTypeAlterTable:
		r.exitAlterTable(ctx.(*parser.Alter_tableContext))
	default:
		// Ignore other node types
	}
	return nil
}

func (r *ColumnRequireRule) enterCreateTable(ctx *parser.Create_tableContext) {
	r.currentOriginalTableName = ctx.Object_name().GetText()
	r.currentMissingColumn = make(map[string]any)
	for column := range r.requireColumns {
		r.currentMissingColumn[column] = true
	}
}

func (r *ColumnRequireRule) enterColumnDeclItemList(ctx *parser.Column_decl_item_listContext) {
	if r.currentOriginalTableName == "" {
		return
	}
	allColumnDeclItems := ctx.AllColumn_decl_item()
	for _, columnDeclItem := range allColumnDeclItems {
		if fullColDecl := columnDeclItem.Full_col_decl(); fullColDecl != nil {
			normalizedColumnName := snowsqlparser.NormalizeSnowSQLObjectNamePart(fullColDecl.Col_decl().Column_name().Id_())
			delete(r.currentMissingColumn, normalizedColumnName)
		}
	}
}

func (r *ColumnRequireRule) exitCreateTable(ctx *parser.Create_tableContext) {
	columnNames := make([]string, 0, len(r.currentMissingColumn))
	for column := range r.currentMissingColumn {
		columnNames = append(columnNames, column)
	}
	if len(columnNames) == 0 {
		return
	}

	slices.Sort(columnNames)
	for _, column := range columnNames {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.NoRequiredColumn.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("Table %s missing required column %q", r.currentOriginalTableName, column),
			StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.Column_decl_item_list().GetStop().GetLine()),
		})
	}
	r.currentOriginalTableName = ""
	r.currentMissingColumn = nil
}

func (r *ColumnRequireRule) enterAlterTable(ctx *parser.Alter_tableContext) {
	r.currentOriginalTableName = ctx.Object_name(0).GetText()
	r.currentMissingColumn = make(map[string]any)
}

func (r *ColumnRequireRule) enterTableColumnAction(ctx *parser.Table_column_actionContext) {
	if r.currentOriginalTableName == "" || len(ctx.AllDROP()) != 1 || ctx.Alter_modify() != nil {
		return
	}

	for _, columnName := range ctx.Column_list().AllColumn_name() {
		originalColumName := columnName.GetText()
		normalizedColumnName := snowsqlparser.ExtractSnowSQLOrdinaryIdentifier(originalColumName)
		if _, ok := r.requireColumns[normalizedColumnName]; ok {
			r.currentMissingColumn[normalizedColumnName] = true
		}
	}
}

func (r *ColumnRequireRule) exitAlterTable(ctx *parser.Alter_tableContext) {
	columnNames := make([]string, 0, len(r.currentMissingColumn))
	for column := range r.currentMissingColumn {
		columnNames = append(columnNames, column)
	}
	if len(columnNames) == 0 {
		return
	}

	slices.Sort(columnNames)
	for _, column := range columnNames {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.NoRequiredColumn.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("Table %s missing required column %q", r.currentOriginalTableName, column),
			StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.Table_column_action().GetStart().GetLine()),
		})
	}
	r.currentOriginalTableName = ""
	r.currentMissingColumn = nil
}
