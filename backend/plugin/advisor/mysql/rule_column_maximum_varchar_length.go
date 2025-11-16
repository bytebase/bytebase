package mysql

import (
	"context"
	"fmt"
	"strconv"

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
	_ advisor.Advisor = (*ColumnMaximumVarcharLengthAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleColumnMaximumVarcharLength, &ColumnMaximumVarcharLengthAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.SchemaRuleColumnMaximumVarcharLength, &ColumnMaximumVarcharLengthAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.SchemaRuleColumnMaximumVarcharLength, &ColumnMaximumVarcharLengthAdvisor{})
}

// ColumnMaximumVarcharLengthAdvisor is the advisor checking for max varchar length.
type ColumnMaximumVarcharLengthAdvisor struct {
}

// Check checks for maximum varchar length.
func (*ColumnMaximumVarcharLengthAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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
	rule := NewVarcharLengthRule(level, string(checkCtx.Rule.Type), payload.Number)

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range stmtList {
		rule.SetBaseLine(stmt.BaseLine)
		checker.SetBaseLine(stmt.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	return checker.GetAdviceList(), nil
}

// VarcharLengthRule checks for maximum varchar length.
type VarcharLengthRule struct {
	BaseRule
	maximum int
}

// NewVarcharLengthRule creates a new VarcharLengthRule.
func NewVarcharLengthRule(level storepb.Advice_Status, title string, maximum int) *VarcharLengthRule {
	return &VarcharLengthRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		maximum: maximum,
	}
}

// Name returns the rule name.
func (*VarcharLengthRule) Name() string {
	return "VarcharLengthRule"
}

// OnEnter is called when entering a parse tree node.
func (r *VarcharLengthRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
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
func (*VarcharLengthRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	// This rule doesn't need exit processing
	return nil
}

func (r *VarcharLengthRule) checkCreateTable(ctx *mysql.CreateTableContext) {
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
		length := r.getVarcharLength(tableElement.ColumnDefinition().FieldDefinition().DataType())
		if r.maximum > 0 && length > r.maximum {
			r.AddAdvice(&storepb.Advice{
				Status:        r.level,
				Code:          code.VarcharLengthExceedsLimit.Int32(),
				Title:         r.title,
				Content:       fmt.Sprintf("The length of the VARCHAR column `%s.%s` is bigger than %d", tableName, columnName, r.maximum),
				StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + tableElement.ColumnDefinition().GetStart().GetLine()),
			})
		}
	}
}

func (r *VarcharLengthRule) checkAlterTable(ctx *mysql.AlterTableContext) {
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
		varcharLengthMap := make(map[string]int)
		switch {
		// add column.
		case item.ADD_SYMBOL() != nil:
			switch {
			case item.Identifier() != nil && item.FieldDefinition() != nil:
				if item.FieldDefinition().DataType() == nil {
					continue
				}

				columnName := mysqlparser.NormalizeMySQLIdentifier(item.Identifier())
				length := r.getVarcharLength(item.FieldDefinition().DataType())
				varcharLengthMap[columnName] = length
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
					length := r.getVarcharLength(tableElement.ColumnDefinition().FieldDefinition().DataType())
					varcharLengthMap[columnName] = length
					columnList = append(columnList, columnName)
				}
			default:
			}
		// change column.
		case item.CHANGE_SYMBOL() != nil && item.ColumnInternalRef() != nil && item.Identifier() != nil && item.FieldDefinition() != nil:
			columnName := mysqlparser.NormalizeMySQLIdentifier(item.Identifier())
			length := r.getVarcharLength(item.FieldDefinition().DataType())
			varcharLengthMap[columnName] = length
			columnList = append(columnList, columnName)
		// modify column.
		case item.MODIFY_SYMBOL() != nil && item.ColumnInternalRef() != nil && item.FieldDefinition() != nil:
			length := r.getVarcharLength(item.FieldDefinition().DataType())
			columnName := mysqlparser.NormalizeMySQLColumnInternalRef(item.ColumnInternalRef())
			varcharLengthMap[columnName] = length
			columnList = append(columnList, columnName)
		default:
			continue
		}
		for _, columnName := range columnList {
			if length, ok := varcharLengthMap[columnName]; ok && r.maximum > 0 && length > r.maximum {
				r.AddAdvice(&storepb.Advice{
					Status:        r.level,
					Code:          code.VarcharLengthExceedsLimit.Int32(),
					Title:         r.title,
					Content:       fmt.Sprintf("The length of the VARCHAR column `%s.%s` is bigger than %d", tableName, columnName, r.maximum),
					StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine()),
				})
			}
		}
	}
}

func (*VarcharLengthRule) getVarcharLength(ctx mysql.IDataTypeContext) int {
	if ctx.GetType_() == nil {
		return 0
	}

	switch ctx.GetType_().GetTokenType() {
	case mysql.MySQLParserVARCHAR_SYMBOL:
		if ctx.FieldLength() == nil || ctx.FieldLength().Real_ulonglong_number() == nil {
			return 1
		}
		lengthStr := ctx.FieldLength().Real_ulonglong_number().GetText()
		length, err := strconv.Atoi(lengthStr)
		if err != nil {
			return 0
		}
		return length
	default:
		return 0
	}
}
