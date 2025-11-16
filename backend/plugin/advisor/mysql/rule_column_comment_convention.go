package mysql

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/parser/mysql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*ColumnCommentConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleColumnCommentConvention, &ColumnCommentConventionAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.SchemaRuleColumnCommentConvention, &ColumnCommentConventionAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.SchemaRuleColumnCommentConvention, &ColumnCommentConventionAdvisor{})
}

// ColumnCommentConventionAdvisor is the advisor checking for column comment convention.
type ColumnCommentConventionAdvisor struct {
}

// Check checks for column comment convention.
func (*ColumnCommentConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	payload, err := advisor.UnmarshalCommentConventionRulePayload(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewColumnCommentConventionRule(level, string(checkCtx.Rule.Type), payload, checkCtx.ClassificationConfig)

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range stmtList {
		rule.SetBaseLine(stmt.BaseLine)
		checker.SetBaseLine(stmt.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	return checker.GetAdviceList(), nil
}

// ColumnCommentConventionRule checks for column comment convention.
type ColumnCommentConventionRule struct {
	BaseRule
	payload              *advisor.CommentConventionRulePayload
	classificationConfig *storepb.DataClassificationSetting_DataClassificationConfig
}

// NewColumnCommentConventionRule creates a new ColumnCommentConventionRule.
func NewColumnCommentConventionRule(level storepb.Advice_Status, title string, payload *advisor.CommentConventionRulePayload, classificationConfig *storepb.DataClassificationSetting_DataClassificationConfig) *ColumnCommentConventionRule {
	return &ColumnCommentConventionRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		payload:              payload,
		classificationConfig: classificationConfig,
	}
}

// Name returns the rule name.
func (*ColumnCommentConventionRule) Name() string {
	return "ColumnCommentConventionRule"
}

// OnEnter is called when entering a parse tree node.
func (r *ColumnCommentConventionRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
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
func (*ColumnCommentConventionRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *ColumnCommentConventionRule) checkCreateTable(ctx *mysql.CreateTableContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.TableName() == nil {
		return
	}
	if ctx.TableElementList() == nil {
		return
	}

	_, tableName := mysqlparser.NormalizeMySQLTableName(ctx.TableName())
	for _, tableElement := range ctx.TableElementList().AllTableElement() {
		if tableElement == nil {
			continue
		}
		if tableElement.ColumnDefinition() == nil {
			continue
		}

		_, _, columnName := mysqlparser.NormalizeMySQLColumnName(tableElement.ColumnDefinition().ColumnName())
		if tableElement.ColumnDefinition().FieldDefinition() == nil {
			continue
		}
		r.checkFieldDefinition(tableName, columnName, tableElement.ColumnDefinition().FieldDefinition())
	}
}

func (r *ColumnCommentConventionRule) checkAlterTable(ctx *mysql.AlterTableContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.AlterTableActions() == nil {
		return
	}
	if ctx.AlterTableActions().AlterCommandList() == nil {
		return
	}
	if ctx.AlterTableActions().AlterCommandList().AlterList() == nil {
		return
	}

	_, tableName := mysqlparser.NormalizeMySQLTableRef(ctx.TableRef())
	// alter table add column, change column, modify column.
	for _, item := range ctx.AlterTableActions().AlterCommandList().AlterList().AllAlterListItem() {
		if item == nil {
			continue
		}

		var columnName string
		switch {
		// add column
		case item.ADD_SYMBOL() != nil:
			switch {
			case item.Identifier() != nil && item.FieldDefinition() != nil:
				columnName := mysqlparser.NormalizeMySQLIdentifier(item.Identifier())
				r.checkFieldDefinition(tableName, columnName, item.FieldDefinition())
			case item.OPEN_PAR_SYMBOL() != nil && item.TableElementList() != nil:
				for _, tableElement := range item.TableElementList().AllTableElement() {
					if tableElement.ColumnDefinition() == nil || tableElement.ColumnDefinition().ColumnName() == nil || tableElement.ColumnDefinition().FieldDefinition() == nil {
						continue
					}
					_, _, columnName := mysqlparser.NormalizeMySQLColumnName(tableElement.ColumnDefinition().ColumnName())
					r.checkFieldDefinition(tableName, columnName, tableElement.ColumnDefinition().FieldDefinition())
				}
			default:
			}
		// modify column
		case item.MODIFY_SYMBOL() != nil && item.ColumnInternalRef() != nil && item.FieldDefinition() != nil:
			columnName = mysqlparser.NormalizeMySQLColumnInternalRef(item.ColumnInternalRef())
			r.checkFieldDefinition(tableName, columnName, item.FieldDefinition())
		// change column
		case item.CHANGE_SYMBOL() != nil && item.ColumnInternalRef() != nil && item.Identifier() != nil && item.FieldDefinition() != nil:
			columnName = mysqlparser.NormalizeMySQLIdentifier(item.Identifier())
			r.checkFieldDefinition(tableName, columnName, item.FieldDefinition())
		default:
		}
	}
}

func (r *ColumnCommentConventionRule) checkFieldDefinition(tableName, columnName string, ctx mysql.IFieldDefinitionContext) {
	comment := ""
	for _, attribute := range ctx.AllColumnAttribute() {
		if attribute == nil || attribute.GetValue() == nil {
			continue
		}
		if attribute.GetValue().GetTokenType() != mysql.MySQLParserCOMMENT_SYMBOL {
			continue
		}
		if attribute.TextLiteral() == nil {
			continue
		}
		comment = mysqlparser.NormalizeMySQLTextLiteral(attribute.TextLiteral())
		if r.payload.MaxLength >= 0 && len(comment) > r.payload.MaxLength {
			r.AddAdvice(&storepb.Advice{
				Status:        r.level,
				Code:          code.CommentTooLong.Int32(),
				Title:         r.title,
				Content:       fmt.Sprintf("The length of column `%s`.`%s` comment should be within %d characters", tableName, columnName, r.payload.MaxLength),
				StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine()),
			})
		}

		if r.payload.RequiredClassification {
			if classification, _ := common.GetClassificationAndUserComment(comment, r.classificationConfig); classification == "" {
				r.AddAdvice(&storepb.Advice{
					Status:        r.level,
					Code:          code.CommentMissingClassification.Int32(),
					Title:         r.title,
					Content:       fmt.Sprintf("Column `%s`.`%s` comment requires classification", tableName, columnName),
					StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine()),
				})
			}
		}

		break
	}

	if len(comment) == 0 && r.payload.Required {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.CommentEmpty.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("Column `%s`.`%s` requires comments", tableName, columnName),
			StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine()),
		})
	}
}
