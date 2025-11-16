package mysql

import (
	"context"
	"fmt"
	"regexp"

	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/parser/mysql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var _ advisor.Advisor = (*NamingColumnConventionAdvisor)(nil)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleColumnNaming, &NamingColumnConventionAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.SchemaRuleColumnNaming, &NamingColumnConventionAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.SchemaRuleColumnNaming, &NamingColumnConventionAdvisor{})
}

// NamingColumnConventionAdvisor is the advisor checking for column naming convention.
type NamingColumnConventionAdvisor struct {
}

// Check checks for column naming convention.
func (*NamingColumnConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	list, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql ParseResult")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	format, maxLength, err := advisor.UnmarshalNamingRulePayloadAsRegexp(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewNamingColumnRule(level, string(checkCtx.Rule.Type), format, maxLength)

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range list {
		rule.SetBaseLine(stmt.BaseLine)
		checker.SetBaseLine(stmt.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	return checker.GetAdviceList(), nil
}

// NamingColumnRule checks for column naming convention.
type NamingColumnRule struct {
	BaseRule
	format    *regexp.Regexp
	maxLength int
}

// NewNamingColumnRule creates a new NamingColumnRule.
func NewNamingColumnRule(level storepb.Advice_Status, title string, format *regexp.Regexp, maxLength int) *NamingColumnRule {
	return &NamingColumnRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		format:    format,
		maxLength: maxLength,
	}
}

// Name returns the rule name.
func (*NamingColumnRule) Name() string {
	return "NamingColumnRule"
}

// OnEnter is called when entering a parse tree node.
func (r *NamingColumnRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
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
func (*NamingColumnRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *NamingColumnRule) checkCreateTable(ctx *mysql.CreateTableContext) {
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
		if tableElement.ColumnDefinition().ColumnName() == nil {
			continue
		}

		_, _, columnName := mysqlparser.NormalizeMySQLColumnName(tableElement.ColumnDefinition().ColumnName())
		r.handleColumn(tableName, columnName, tableElement.ColumnDefinition().GetStart().GetLine())
	}
}

func (r *NamingColumnRule) checkAlterTable(ctx *mysql.AlterTableContext) {
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

		switch {
		// add column
		case item.ADD_SYMBOL() != nil:
			switch {
			case item.Identifier() != nil && item.FieldDefinition() != nil:
				columnName := mysqlparser.NormalizeMySQLIdentifier(item.Identifier())
				r.handleColumn(tableName, columnName, item.GetStart().GetLine())
			case item.OPEN_PAR_SYMBOL() != nil && item.TableElementList() != nil:
				for _, tableElement := range item.TableElementList().AllTableElement() {
					if tableElement.ColumnDefinition() == nil || tableElement.ColumnDefinition().ColumnName() == nil || tableElement.ColumnDefinition().FieldDefinition() == nil {
						continue
					}
					_, _, columnName := mysqlparser.NormalizeMySQLColumnName(tableElement.ColumnDefinition().ColumnName())
					r.handleColumn(tableName, columnName, tableElement.ColumnDefinition().GetStart().GetLine())
				}
			default:
			}
		// rename column
		case item.RENAME_SYMBOL() != nil && item.COLUMN_SYMBOL() != nil:
			// only focus on new column-name.
			columnName := mysqlparser.NormalizeMySQLIdentifier(item.Identifier())
			r.handleColumn(tableName, columnName, item.GetStart().GetLine())
		// change column
		case item.CHANGE_SYMBOL() != nil && item.ColumnInternalRef() != nil && item.Identifier() != nil:
			// only focus on new column-name.
			columnName := mysqlparser.NormalizeMySQLIdentifier(item.Identifier())
			r.handleColumn(tableName, columnName, item.GetStart().GetLine())
		default:
			continue
		}
	}
}

func (r *NamingColumnRule) handleColumn(tableName string, columnName string, lineNumber int) {
	// we need to accumulate line number for each statement and elements of statements.
	lineNumber += r.baseLine
	if !r.format.MatchString(columnName) {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.NamingColumnConventionMismatch.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("`%s`.`%s` mismatches column naming convention, naming format should be %q", tableName, columnName, r.format),
			StartPosition: common.ConvertANTLRLineToPosition(lineNumber),
		})
	}
	if r.maxLength > 0 && len(columnName) > r.maxLength {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.NamingColumnConventionMismatch.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("`%s`.`%s` mismatches column naming convention, its length should be within %d characters", tableName, columnName, r.maxLength),
			StartPosition: common.ConvertANTLRLineToPosition(lineNumber),
		})
	}
}
