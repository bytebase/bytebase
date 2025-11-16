package mysql

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/parser/mysql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*ColumnRequireCollationAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleColumnRequireCollation, &ColumnRequireCollationAdvisor{})
}

// ColumnRequireCollationAdvisor is the advisor checking for require collation.
type ColumnRequireCollationAdvisor struct {
}

func (*ColumnRequireCollationAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewColumnRequireCollationRule(level, string(checkCtx.Rule.Type))

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range stmtList {
		rule.SetBaseLine(stmt.BaseLine)
		checker.SetBaseLine(stmt.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	return checker.GetAdviceList(), nil
}

// ColumnRequireCollationRule checks for require collation.
type ColumnRequireCollationRule struct {
	BaseRule
}

// NewColumnRequireCollationRule creates a new ColumnRequireCollationRule.
func NewColumnRequireCollationRule(level storepb.Advice_Status, title string) *ColumnRequireCollationRule {
	return &ColumnRequireCollationRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
	}
}

// Name returns the rule name.
func (*ColumnRequireCollationRule) Name() string {
	return "ColumnRequireCollationRule"
}

// OnEnter is called when entering a parse tree node.
func (r *ColumnRequireCollationRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeCreateTable:
		r.checkCreateTable(ctx.(*mysql.CreateTableContext))
	case NodeTypeAlterTable:
		r.checkAlterTable(ctx.(*mysql.AlterTableContext))
	default:
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*ColumnRequireCollationRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *ColumnRequireCollationRule) checkCreateTable(ctx *mysql.CreateTableContext) {
	if ctx.TableName() == nil || ctx.TableElementList() == nil {
		return
	}
	_, tableName := mysqlparser.NormalizeMySQLTableName(ctx.TableName())
	if tableName == "" {
		return
	}

	for _, tableElement := range ctx.TableElementList().AllTableElement() {
		if tableElement.ColumnDefinition() == nil {
			continue
		}
		columnDefinition := tableElement.ColumnDefinition()
		if columnDefinition.FieldDefinition() == nil || columnDefinition.FieldDefinition().DataType() == nil {
			continue
		}

		_, _, columnName := mysqlparser.NormalizeMySQLColumnName(tableElement.ColumnDefinition().ColumnName())
		dataType := columnDefinition.FieldDefinition().DataType()
		if isCharsetDataType(dataType) {
			if columnDefinition.FieldDefinition().Collate() == nil {
				r.AddAdvice(&storepb.Advice{
					Status:        r.level,
					Code:          code.NoCollation.Int32(),
					Title:         r.title,
					Content:       fmt.Sprintf("Column %s does not have a collation specified", columnName),
					StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + columnDefinition.GetStart().GetLine()),
				})
			}
		}
	}
}

func (r *ColumnRequireCollationRule) checkAlterTable(ctx *mysql.AlterTableContext) {
	if ctx.AlterTableActions() == nil || ctx.AlterTableActions().AlterCommandList() == nil || ctx.AlterTableActions().AlterCommandList().AlterList() == nil {
		return
	}
	for _, alterListItem := range ctx.AlterTableActions().AlterCommandList().AlterList().AllAlterListItem() {
		// Only check ADD COLUMN for now.
		if alterListItem.ADD_SYMBOL() == nil || alterListItem.COLUMN_SYMBOL() == nil || alterListItem.FieldDefinition() == nil {
			continue
		}

		columnName := mysqlparser.NormalizeMySQLIdentifier(alterListItem.Identifier())
		dataType := alterListItem.FieldDefinition().DataType()
		if isCharsetDataType(dataType) {
			if alterListItem.FieldDefinition().Collate() == nil {
				r.AddAdvice(&storepb.Advice{
					Status:        r.level,
					Code:          code.NoCollation.Int32(),
					Title:         r.title,
					Content:       fmt.Sprintf("Column %s does not have a collation specified", columnName),
					StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + alterListItem.GetStart().GetLine()),
				})
			}
		}
	}
}
