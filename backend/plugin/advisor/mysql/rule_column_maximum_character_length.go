package mysql

import (
	"context"
	"fmt"
	"strconv"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	"github.com/bytebase/parser/mysql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*ColumnMaximumCharacterLengthAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleColumnMaximumCharacterLength, &ColumnMaximumCharacterLengthAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.SchemaRuleColumnMaximumCharacterLength, &ColumnMaximumCharacterLengthAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.SchemaRuleColumnMaximumCharacterLength, &ColumnMaximumCharacterLengthAdvisor{})
}

// ColumnMaximumCharacterLengthAdvisor is the advisor checking for max character length.
type ColumnMaximumCharacterLengthAdvisor struct {
}

// Check checks for maximum character length.
func (*ColumnMaximumCharacterLengthAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	payload, err := advisor.UnmarshalNumberTypeRulePayload(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewColumnMaximumCharacterLengthRule(level, string(checkCtx.Rule.Type), payload.Number)

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range stmtList {
		rule.SetBaseLine(stmt.BaseLine)
		checker.SetBaseLine(stmt.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	return checker.GetAdviceList(), nil
}

// ColumnMaximumCharacterLengthRule checks for maximum character length.
type ColumnMaximumCharacterLengthRule struct {
	BaseRule
	maximum int
}

// NewColumnMaximumCharacterLengthRule creates a new ColumnMaximumCharacterLengthRule.
func NewColumnMaximumCharacterLengthRule(level storepb.Advice_Status, title string, maximum int) *ColumnMaximumCharacterLengthRule {
	return &ColumnMaximumCharacterLengthRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		maximum: maximum,
	}
}

// Name returns the rule name.
func (*ColumnMaximumCharacterLengthRule) Name() string {
	return "ColumnMaximumCharacterLengthRule"
}

// OnEnter is called when entering a parse tree node.
func (r *ColumnMaximumCharacterLengthRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
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
func (*ColumnMaximumCharacterLengthRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *ColumnMaximumCharacterLengthRule) checkCreateTable(ctx *mysql.CreateTableContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.TableElementList() == nil || ctx.TableName() == nil {
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
		if tableElement.ColumnDefinition().FieldDefinition() == nil {
			continue
		}
		if tableElement.ColumnDefinition().FieldDefinition().DataType() == nil {
			continue
		}
		_, _, columnName := mysqlparser.NormalizeMySQLColumnName(tableElement.ColumnDefinition().ColumnName())
		charLength := r.getCharLength(tableElement.ColumnDefinition().FieldDefinition().DataType())
		if r.maximum > 0 && charLength > r.maximum {
			r.AddAdvice(&storepb.Advice{
				Status:        r.level,
				Code:          code.CharLengthExceedsLimit.Int32(),
				Title:         r.title,
				Content:       fmt.Sprintf("The length of the CHAR column `%s.%s` is bigger than %d, please use VARCHAR instead", tableName, columnName, r.maximum),
				StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + tableElement.ColumnDefinition().GetStart().GetLine()),
			})
		}
	}
}

func (r *ColumnMaximumCharacterLengthRule) checkAlterTable(ctx *mysql.AlterTableContext) {
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
	if tableName == "" {
		return
	}
	// alter table add column, change column, modify column.
	for _, item := range ctx.AlterTableActions().AlterCommandList().AlterList().AllAlterListItem() {
		if item == nil {
			continue
		}

		var columnList []string
		charLengthMap := make(map[string]int)
		switch {
		// add column.
		case item.ADD_SYMBOL() != nil:
			switch {
			case item.Identifier() != nil && item.FieldDefinition() != nil:
				if item.FieldDefinition().DataType() == nil {
					continue
				}

				columnName := mysqlparser.NormalizeMySQLIdentifier(item.Identifier())
				charLength := r.getCharLength(item.FieldDefinition().DataType())
				charLengthMap[columnName] = charLength
				columnList = append(columnList, columnName)
			case item.OPEN_PAR_SYMBOL() != nil && item.TableElementList() != nil:
				for _, tableElement := range item.TableElementList().AllTableElement() {
					if tableElement.ColumnDefinition() == nil {
						continue
					}
					if tableElement.ColumnDefinition().ColumnName() == nil || tableElement.ColumnDefinition().FieldDefinition() == nil {
						continue
					}
					if tableElement.ColumnDefinition().FieldDefinition().DataType() == nil {
						continue
					}

					_, _, columnName := mysqlparser.NormalizeMySQLColumnName(tableElement.ColumnDefinition().ColumnName())
					charLength := r.getCharLength(tableElement.ColumnDefinition().FieldDefinition().DataType())
					charLengthMap[columnName] = charLength
					columnList = append(columnList, columnName)
				}
			default:
			}
		// change column.
		case item.CHANGE_SYMBOL() != nil && item.ColumnInternalRef() != nil && item.Identifier() != nil && item.FieldDefinition() != nil:
			// oldColumnName := mysqlparser.NormalizeMySQLColumnInternalRef(item.ColumnInternalRef())
			columnName := mysqlparser.NormalizeMySQLIdentifier(item.Identifier())
			charLength := r.getCharLength(item.FieldDefinition().DataType())
			charLengthMap[columnName] = charLength
			columnList = append(columnList, columnName)
		// modify column.
		case item.MODIFY_SYMBOL() != nil && item.ColumnInternalRef() != nil && item.FieldDefinition() != nil:
			charLength := r.getCharLength(item.FieldDefinition().DataType())
			columnName := mysqlparser.NormalizeMySQLColumnInternalRef(item.ColumnInternalRef())
			charLengthMap[columnName] = charLength
			columnList = append(columnList, columnName)
		default:
			continue
		}
		for _, columnName := range columnList {
			if charLength, ok := charLengthMap[columnName]; ok && r.maximum > 0 && charLength > r.maximum {
				r.AddAdvice(&storepb.Advice{
					Status:        r.level,
					Code:          code.CharLengthExceedsLimit.Int32(),
					Title:         r.title,
					Content:       fmt.Sprintf("The length of the CHAR column `%s.%s` is bigger than %d, please use VARCHAR instead", tableName, columnName, r.maximum),
					StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine()),
				})
			}
		}
	}
}

func (*ColumnMaximumCharacterLengthRule) getCharLength(ctx mysql.IDataTypeContext) int {
	if ctx.GetType_() == nil {
		return 0
	}
	switch ctx.GetType_().GetTokenType() {
	case mysql.MySQLParserCHAR_SYMBOL:
		// for mysql: create table tt(a char) == create table tt(a char(1));
		if ctx.FieldLength() == nil || ctx.FieldLength().Real_ulonglong_number() == nil {
			return 1
		}
		charLengthStr := ctx.FieldLength().Real_ulonglong_number().GetText()
		charLengthInt, err := strconv.Atoi(charLengthStr)
		if err != nil {
			return 0
		}
		return charLengthInt
	default:
		return 0
	}
}
