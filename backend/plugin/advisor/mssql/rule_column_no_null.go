package mssql

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/tsql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	tsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
)

var (
	_ advisor.Advisor = (*ColumnNoNullAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, advisor.SchemaRuleColumnNotNull, &ColumnNoNullAdvisor{})
}

// ColumnNoNullAdvisor is the advisor checking for column no NULL value..
type ColumnNoNullAdvisor struct {
}

// Check checks for column no NULL value..
func (*ColumnNoNullAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, ok := checkCtx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewColumnNoNullRule(level, string(checkCtx.Rule.Type))

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	antlr.ParseTreeWalkerDefault.Walk(checker, tree)

	return checker.GetAdviceList(), nil
}

// ColumnNoNullRule checks for column no NULL value.
type ColumnNoNullRule struct {
	BaseRule

	// currentNormalizedTableName is the normalized table name of the current table.
	currentNormalizedTableName string
	// isCurrentTableColumnNullable is the map of column name to whether the column is nullable.
	isCurrentTableColumnNullable map[string]bool
	// currentTableColumnIsNullableLine is the map of column name to the line number of the column definition.
	currentTableColumnIsNullableLine map[string]int
}

// NewColumnNoNullRule creates a new ColumnNoNullRule.
func NewColumnNoNullRule(level storepb.Advice_Status, title string) *ColumnNoNullRule {
	return &ColumnNoNullRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		isCurrentTableColumnNullable:     make(map[string]bool),
		currentTableColumnIsNullableLine: make(map[string]int),
	}
}

// Name returns the rule name.
func (*ColumnNoNullRule) Name() string {
	return "ColumnNoNullRule"
}

// OnEnter is called when entering a parse tree node.
func (r *ColumnNoNullRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeCreateTable:
		r.enterCreateTable(ctx.(*parser.Create_tableContext))
	case NodeTypeTableConstraint:
		r.enterTableConstraint(ctx.(*parser.Table_constraintContext))
	case NodeTypeColumnDefinition:
		r.enterColumnDefinition(ctx.(*parser.Column_definitionContext))
	case NodeTypeAlterTable:
		r.enterAlterTable(ctx.(*parser.Alter_tableContext))
	default:
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (r *ColumnNoNullRule) OnExit(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeCreateTable:
		r.exitCreateTable(ctx.(*parser.Create_tableContext))
	case NodeTypeAlterTable:
		r.exitAlterTable(ctx.(*parser.Alter_tableContext))
	default:
	}
	return nil
}

func (r *ColumnNoNullRule) enterCreateTable(ctx *parser.Create_tableContext) {
	tableName := ctx.Table_name()
	if tableName == nil {
		return
	}
	normalizedTableName := tsqlparser.NormalizeTSQLTableName(tableName, "" /* fallbackDatabase */, "dbo" /* fallbackSchema */, false /* caseSensitive */)
	r.currentNormalizedTableName = normalizedTableName
}

func (r *ColumnNoNullRule) exitCreateTable(_ *parser.Create_tableContext) {
	r.currentNormalizedTableName = ""
	for columnName, isNullable := range r.isCurrentTableColumnNullable {
		if !isNullable {
			continue
		}
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.ColumnCannotNull.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("Column [%s] is nullable, which is not allowed.", columnName),
			StartPosition: common.ConvertANTLRLineToPosition(r.currentTableColumnIsNullableLine[columnName]),
		})
	}

	r.isCurrentTableColumnNullable = make(map[string]bool)
	r.currentTableColumnIsNullableLine = make(map[string]int)
}

func (r *ColumnNoNullRule) enterTableConstraint(ctx *parser.Table_constraintContext) {
	parent := ctx.GetParent()
	switch parent.(type) {
	case *parser.Column_def_table_constraintContext:
	default:
		return
	}
	if ctx.PRIMARY() != nil {
		allColumns := ctx.Column_name_list_with_order().AllColumn_name_with_order()
		for _, column := range allColumns {
			_, columnName := tsqlparser.NormalizeTSQLIdentifier(column.Id_())
			r.isCurrentTableColumnNullable[columnName] = false
		}
	}
}

func (r *ColumnNoNullRule) enterColumnDefinition(ctx *parser.Column_definitionContext) {
	parent := ctx.GetParent()
	switch parent.(type) {
	case *parser.Alter_tableContext:
	case *parser.Column_def_table_constraintContext:
	default:
		return
	}
	_, columnName := tsqlparser.NormalizeTSQLIdentifier(ctx.Id_())
	r.isCurrentTableColumnNullable[columnName] = true
	r.currentTableColumnIsNullableLine[columnName] = ctx.Id_().GetStart().GetLine()
	allColumnDefinitionElements := ctx.AllColumn_definition_element()
	for _, columnDefinitionElement := range allColumnDefinitionElements {
		if v := columnDefinitionElement.Column_constraint(); v != nil {
			if v.PRIMARY() != nil {
				r.isCurrentTableColumnNullable[columnName] = false
				break
			}
			if (v.Null_notnull() != nil && v.Null_notnull().NOT() != nil) || v.Null_notnull() == nil {
				r.isCurrentTableColumnNullable[columnName] = false
				break
			}
		}
	}
}

func (r *ColumnNoNullRule) enterAlterTable(ctx *parser.Alter_tableContext) {
	tableName := ctx.Table_name(0)
	if tableName == nil {
		return
	}
	if (len(ctx.AllALTER()) == 2 && ctx.COLUMN() != nil) /* ALTER COLUMN */ || (len(ctx.AllALTER()) == 1 && ctx.ADD() != nil && ctx.WITH() == nil) /* ALTER */ {
		normalizedTableName := tsqlparser.NormalizeTSQLTableName(tableName, "" /* fallbackDatabase */, "dbo" /* fallbackSchema */, false /* caseSensitive */)
		r.currentNormalizedTableName = normalizedTableName
	}
}

func (r *ColumnNoNullRule) exitAlterTable(_ *parser.Alter_tableContext) {
	r.currentNormalizedTableName = ""
	for columnName, isNullable := range r.isCurrentTableColumnNullable {
		if !isNullable {
			continue
		}
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.ColumnCannotNull.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("Column [%s] is nullable, which is not allowed.", columnName),
			StartPosition: common.ConvertANTLRLineToPosition(r.currentTableColumnIsNullableLine[columnName]),
		})
	}

	r.isCurrentTableColumnNullable = make(map[string]bool)
	r.currentTableColumnIsNullableLine = make(map[string]int)
}
