package snowflake

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/snowflake"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	snowsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/snowflake"
)

var (
	_ advisor.Advisor = (*ColumnNoNullAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_SNOWFLAKE, advisor.SchemaRuleColumnNotNull, &ColumnNoNullAdvisor{})
}

// ColumnNoNullAdvisor is the advisor checking for column no NULL value.
type ColumnNoNullAdvisor struct {
}

// Check checks for column no NULL value.
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

	// currentOriginalTableName is the original table name of the current table.
	currentOriginalTableName string
	// columnNullable is a map from normalized column name to the line number causing the column to be nullable.
	columnNullable map[string]int
}

// NewColumnNoNullRule creates a new ColumnNoNullRule.
func NewColumnNoNullRule(level storepb.Advice_Status, title string) *ColumnNoNullRule {
	return &ColumnNoNullRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		columnNullable: make(map[string]int),
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
	case NodeTypeFullColDecl:
		r.enterFullColDecl(ctx.(*parser.Full_col_declContext))
	case NodeTypeOutOfLineConstraint:
		r.enterOutOfLineConstraint(ctx.(*parser.Out_of_line_constraintContext))
	case NodeTypeAlterTable:
		r.enterAlterTable(ctx.(*parser.Alter_tableContext))
	case NodeTypeTableColumnAction:
		r.enterTableColumnAction(ctx.(*parser.Table_column_actionContext))
	case "Alter_table_alter_column":
		r.enterAlterTableAlterColumn(ctx.(*parser.Alter_table_alter_columnContext))
	case "Alter_column_decl":
		r.enterAlterColumnDecl(ctx.(*parser.Alter_column_declContext))
	default:
		// Ignore other node types
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (r *ColumnNoNullRule) OnExit(_ antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeCreateTable:
		r.exitCreateTable()
	case NodeTypeAlterTable:
		r.exitAlterTable()
	case "Alter_table_alter_column":
		r.exitAlterTableAlterColumn()
	default:
		// Ignore other node types
	}
	return nil
}

func (r *ColumnNoNullRule) enterCreateTable(ctx *parser.Create_tableContext) {
	r.currentOriginalTableName = ctx.Object_name().GetText()
}

func (r *ColumnNoNullRule) exitCreateTable() {
	for normalizedColumnName, columnNullableLine := range r.columnNullable {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.ColumnCannotNull.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("Column %s is nullable, which is not allowed.", normalizedColumnName),
			StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + columnNullableLine),
		})
	}
	r.currentOriginalTableName = ""
	r.columnNullable = make(map[string]int)
}

func (r *ColumnNoNullRule) enterFullColDecl(ctx *parser.Full_col_declContext) {
	if r.currentOriginalTableName == "" {
		return
	}
	normalizedOriginalColumnID := snowsqlparser.NormalizeSnowSQLObjectNamePart(ctx.Col_decl().Column_name().Id_())
	r.columnNullable[normalizedOriginalColumnID] = ctx.GetStart().GetLine()
	for _, nullNotNull := range ctx.AllNull_not_null() {
		if nullNotNull.NOT() != nil {
			delete(r.columnNullable, normalizedOriginalColumnID)
			break
		}
	}
	for _, constraint := range ctx.AllInline_constraint() {
		if constraint.PRIMARY() != nil {
			delete(r.columnNullable, normalizedOriginalColumnID)
			break
		}
	}
}

func (r *ColumnNoNullRule) enterOutOfLineConstraint(ctx *parser.Out_of_line_constraintContext) {
	if r.currentOriginalTableName == "" {
		return
	}
	if ctx.PRIMARY() != nil {
		for _, columnListInParentheses := range ctx.AllColumn_list_in_parentheses() {
			for _, column := range columnListInParentheses.Column_list().AllColumn_name() {
				normalizedOriginalColumnID := snowsqlparser.NormalizeSnowSQLObjectNamePart(column.Id_())
				delete(r.columnNullable, normalizedOriginalColumnID)
			}
		}
	}
}

func (r *ColumnNoNullRule) enterAlterTable(ctx *parser.Alter_tableContext) {
	r.currentOriginalTableName = ctx.Object_name(0).GetText()
}

func (r *ColumnNoNullRule) exitAlterTable() {
	for normalizedColumnName, columnNullableLine := range r.columnNullable {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.ColumnCannotNull.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("Column %s is nullable, which is not allowed.", normalizedColumnName),
			StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + columnNullableLine),
		})
	}
	r.currentOriginalTableName = ""
	r.columnNullable = make(map[string]int)
}

func (r *ColumnNoNullRule) enterTableColumnAction(ctx *parser.Table_column_actionContext) {
	if r.currentOriginalTableName == "" {
		return
	}
	if ctx.ADD() != nil {
		normalizedNewColumnName := snowsqlparser.NormalizeSnowSQLObjectNamePart(ctx.Column_name(0).Id_())
		inlineConstraintHasPK := ctx.Inline_constraint() != nil && ctx.Inline_constraint().PRIMARY() != nil
		inlineConstraintHasNotNull := ctx.Inline_constraint() != nil && (ctx.Inline_constraint().Null_not_null() != nil && ctx.Inline_constraint().Null_not_null().NOT() != nil)
		hasNotNull := ctx.Null_not_null() != nil && ctx.Null_not_null().NOT() != nil

		if !inlineConstraintHasPK && !inlineConstraintHasNotNull && !hasNotNull {
			r.columnNullable[normalizedNewColumnName] = ctx.GetStart().GetLine()
		}
		return
	}
	if ctx.Alter_modify() != nil {
		normalizedOriginalColumnName := snowsqlparser.NormalizeSnowSQLObjectNamePart(ctx.Column_name(0).Id_())
		if len(ctx.AllDROP()) == 1 && ctx.NOT() != nil && ctx.NULL_() != nil {
			r.columnNullable[normalizedOriginalColumnName] = ctx.GetStart().GetLine()
		}
		return
	}
}

func (r *ColumnNoNullRule) enterAlterTableAlterColumn(ctx *parser.Alter_table_alter_columnContext) {
	r.currentOriginalTableName = ctx.Object_name().GetText()
}

func (r *ColumnNoNullRule) exitAlterTableAlterColumn() {
	for normalizedColumnName, columnNullableLine := range r.columnNullable {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.ColumnCannotNull.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("After dropping NOT NULL of column %s, it will be nullable, which is not allowed.", normalizedColumnName),
			StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + columnNullableLine),
		})
	}
	r.currentOriginalTableName = ""
	r.columnNullable = make(map[string]int)
}

func (r *ColumnNoNullRule) enterAlterColumnDecl(ctx *parser.Alter_column_declContext) {
	if r.currentOriginalTableName == "" {
		return
	}
	normalizedNewColumnName := snowsqlparser.NormalizeSnowSQLObjectNamePart(ctx.Column_name().Id_())
	if ctx.Alter_column_opts().DROP() != nil && ctx.Alter_column_opts().NOT() != nil && ctx.Alter_column_opts().NULL_() != nil {
		r.columnNullable[normalizedNewColumnName] = ctx.GetStart().GetLine()
	}
}
