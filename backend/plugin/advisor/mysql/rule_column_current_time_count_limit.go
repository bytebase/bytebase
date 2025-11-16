package mysql

import (
	"context"
	"fmt"
	"slices"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	"github.com/bytebase/parser/mysql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

const (
	maxDefaultCurrentTimeColumCount   = 2
	maxOnUpdateCurrentTimeColumnCount = 1
)

var (
	_ advisor.Advisor = (*ColumnCurrentTimeCountLimitAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleCurrentTimeColumnCountLimit, &ColumnCurrentTimeCountLimitAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.SchemaRuleCurrentTimeColumnCountLimit, &ColumnCurrentTimeCountLimitAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.SchemaRuleCurrentTimeColumnCountLimit, &ColumnCurrentTimeCountLimitAdvisor{})
}

// ColumnCurrentTimeCountLimitAdvisor is the advisor checking for current time column count limit.
type ColumnCurrentTimeCountLimitAdvisor struct {
}

// Check checks for current time column count limit.
func (*ColumnCurrentTimeCountLimitAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewColumnCurrentTimeCountLimitRule(level, string(checkCtx.Rule.Type))

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range stmtList {
		rule.SetBaseLine(stmt.BaseLine)
		checker.SetBaseLine(stmt.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	// Generate advice after walking
	rule.generateAdvice()

	return checker.GetAdviceList(), nil
}

type currentTimeTableData struct {
	tableName                string
	defaultCurrentTimeCount  int
	onUpdateCurrentTimeCount int
	line                     int
}

// ColumnCurrentTimeCountLimitRule checks for current time column count limit.
type ColumnCurrentTimeCountLimitRule struct {
	BaseRule
	tableSet map[string]currentTimeTableData
}

// NewColumnCurrentTimeCountLimitRule creates a new ColumnCurrentTimeCountLimitRule.
func NewColumnCurrentTimeCountLimitRule(level storepb.Advice_Status, title string) *ColumnCurrentTimeCountLimitRule {
	return &ColumnCurrentTimeCountLimitRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		tableSet: make(map[string]currentTimeTableData),
	}
}

// Name returns the rule name.
func (*ColumnCurrentTimeCountLimitRule) Name() string {
	return "ColumnCurrentTimeCountLimitRule"
}

// OnEnter is called when entering a parse tree node.
func (r *ColumnCurrentTimeCountLimitRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeCreateTable:
		r.checkCreateTable(ctx.(*mysql.CreateTableContext))
	case NodeTypeAlterTable:
		r.checkAlterTable(ctx.(*mysql.AlterTableContext))
	default:
		// Other node types
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*ColumnCurrentTimeCountLimitRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *ColumnCurrentTimeCountLimitRule) checkCreateTable(ctx *mysql.CreateTableContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.TableElementList() == nil || ctx.TableName() == nil {
		return
	}

	_, tableName := mysqlparser.NormalizeMySQLTableName(ctx.TableName())
	for _, tableElement := range ctx.TableElementList().AllTableElement() {
		if tableElement.ColumnDefinition() == nil || tableElement.ColumnDefinition().FieldDefinition() == nil || tableElement.ColumnDefinition().FieldDefinition().DataType() == nil {
			continue
		}
		_, _, columnName := mysqlparser.NormalizeMySQLColumnName(tableElement.ColumnDefinition().ColumnName())
		r.checkTime(tableName, columnName, tableElement.ColumnDefinition().FieldDefinition())
	}
}

func (r *ColumnCurrentTimeCountLimitRule) checkAlterTable(ctx *mysql.AlterTableContext) {
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

		var columnName string
		switch {
		// add column
		case item.ADD_SYMBOL() != nil:
			switch {
			case item.Identifier() != nil && item.FieldDefinition() != nil:
				columnName := mysqlparser.NormalizeMySQLIdentifier(item.Identifier())
				r.checkTime(tableName, columnName, item.FieldDefinition())
			case item.OPEN_PAR_SYMBOL() != nil && item.TableElementList() != nil:
				for _, tableElement := range item.TableElementList().AllTableElement() {
					if tableElement.ColumnDefinition() == nil || tableElement.ColumnDefinition().ColumnName() == nil || tableElement.ColumnDefinition().FieldDefinition() == nil {
						continue
					}
					_, _, columnName := mysqlparser.NormalizeMySQLColumnName(tableElement.ColumnDefinition().ColumnName())
					r.checkTime(tableName, columnName, tableElement.ColumnDefinition().FieldDefinition())
				}
			default:
			}
		// change column.
		case item.CHANGE_SYMBOL() != nil && item.ColumnInternalRef() != nil && item.Identifier() != nil && item.FieldDefinition() != nil:
			if item.FieldDefinition().DataType() == nil {
				continue
			}
			// only focus on new column name.
			columnName = mysqlparser.NormalizeMySQLIdentifier(item.Identifier())
			r.checkTime(tableName, columnName, item.FieldDefinition())
		// modify column.
		case item.MODIFY_SYMBOL() != nil && item.ColumnInternalRef() != nil && item.FieldDefinition() != nil:
			if item.FieldDefinition().DataType() == nil {
				continue
			}
			columnName = mysqlparser.NormalizeMySQLColumnInternalRef(item.ColumnInternalRef())
			r.checkTime(tableName, columnName, item.FieldDefinition())
		default:
			continue
		}
	}
}

func (r *ColumnCurrentTimeCountLimitRule) checkTime(tableName string, _ string, ctx mysql.IFieldDefinitionContext) {
	if ctx.DataType() == nil {
		return
	}

	switch ctx.DataType().GetType_().GetTokenType() {
	case mysql.MySQLParserDATETIME_SYMBOL, mysql.MySQLParserTIMESTAMP_SYMBOL:
		if r.isDefaultCurrentTime(ctx) {
			table, exists := r.tableSet[tableName]
			if !exists {
				table = currentTimeTableData{
					tableName: tableName,
				}
			}
			table.defaultCurrentTimeCount++
			table.line = r.baseLine + ctx.GetStart().GetLine()
			r.tableSet[tableName] = table
		}
		if r.isOnUpdateCurrentTime(ctx) {
			table, exists := r.tableSet[tableName]
			if !exists {
				table = currentTimeTableData{
					tableName: tableName,
				}
			}
			table.onUpdateCurrentTimeCount++
			table.line = r.baseLine + ctx.GetStart().GetLine()
			r.tableSet[tableName] = table
		}
	default:
	}
}

func (*ColumnCurrentTimeCountLimitRule) isDefaultCurrentTime(ctx mysql.IFieldDefinitionContext) bool {
	for _, attr := range ctx.AllColumnAttribute() {
		if attr == nil || attr.GetValue() == nil {
			continue
		}
		if attr.GetValue().GetTokenType() == mysql.MySQLParserDEFAULT_SYMBOL && attr.NOW_SYMBOL() != nil {
			return true
		}
	}
	return false
}

func (*ColumnCurrentTimeCountLimitRule) isOnUpdateCurrentTime(ctx mysql.IFieldDefinitionContext) bool {
	for _, attr := range ctx.AllColumnAttribute() {
		if attr == nil || attr.GetValue() == nil {
			continue
		}
		if attr.GetValue().GetTokenType() == mysql.MySQLParserON_SYMBOL && attr.NOW_SYMBOL() != nil {
			return true
		}
	}
	return false
}

func (r *ColumnCurrentTimeCountLimitRule) generateAdvice() {
	var tableList []currentTimeTableData
	for _, table := range r.tableSet {
		tableList = append(tableList, table)
	}
	slices.SortFunc(tableList, func(a, b currentTimeTableData) int {
		if a.line < b.line {
			return -1
		}
		if a.line > b.line {
			return 1
		}
		return 0
	})
	for _, table := range tableList {
		if table.defaultCurrentTimeCount > maxDefaultCurrentTimeColumCount {
			r.AddAdvice(&storepb.Advice{
				Status:        r.level,
				Code:          code.DefaultCurrentTimeColumnCountExceedsLimit.Int32(),
				Title:         r.title,
				Content:       fmt.Sprintf("Table `%s` has %d DEFAULT CURRENT_TIMESTAMP() columns. The count greater than %d.", table.tableName, table.defaultCurrentTimeCount, maxDefaultCurrentTimeColumCount),
				StartPosition: common.ConvertANTLRLineToPosition(table.line),
			})
		}
		if table.onUpdateCurrentTimeCount > maxOnUpdateCurrentTimeColumnCount {
			r.AddAdvice(&storepb.Advice{
				Status:        r.level,
				Code:          code.OnUpdateCurrentTimeColumnCountExceedsLimit.Int32(),
				Title:         r.title,
				Content:       fmt.Sprintf("Table `%s` has %d ON UPDATE CURRENT_TIMESTAMP() columns. The count greater than %d.", table.tableName, table.onUpdateCurrentTimeCount, maxOnUpdateCurrentTimeColumnCount),
				StartPosition: common.ConvertANTLRLineToPosition(table.line),
			})
		}
	}
}
