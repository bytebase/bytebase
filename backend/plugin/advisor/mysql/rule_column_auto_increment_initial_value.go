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
	_ advisor.Advisor = (*ColumnAutoIncrementInitialValueAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleColumnAutoIncrementInitialValue, &ColumnAutoIncrementInitialValueAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.SchemaRuleColumnAutoIncrementInitialValue, &ColumnAutoIncrementInitialValueAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.SchemaRuleColumnAutoIncrementInitialValue, &ColumnAutoIncrementInitialValueAdvisor{})
}

// ColumnAutoIncrementInitialValueAdvisor is the advisor checking for auto-increment column initial value.
type ColumnAutoIncrementInitialValueAdvisor struct {
}

// Check checks for auto-increment column initial value.
func (*ColumnAutoIncrementInitialValueAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parser result")
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
	rule := NewColumnAutoIncrementInitialValueRule(level, string(checkCtx.Rule.Type), payload.Number)

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range stmtList {
		rule.SetBaseLine(stmt.BaseLine)
		checker.SetBaseLine(stmt.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	return checker.GetAdviceList(), nil
}

// ColumnAutoIncrementInitialValueRule checks for auto-increment column initial value.
type ColumnAutoIncrementInitialValueRule struct {
	BaseRule
	value int
}

// NewColumnAutoIncrementInitialValueRule creates a new ColumnAutoIncrementInitialValueRule.
func NewColumnAutoIncrementInitialValueRule(level storepb.Advice_Status, title string, value int) *ColumnAutoIncrementInitialValueRule {
	return &ColumnAutoIncrementInitialValueRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		value: value,
	}
}

// Name returns the rule name.
func (*ColumnAutoIncrementInitialValueRule) Name() string {
	return "ColumnAutoIncrementInitialValueRule"
}

// OnEnter is called when entering a parse tree node.
func (r *ColumnAutoIncrementInitialValueRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
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
func (*ColumnAutoIncrementInitialValueRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *ColumnAutoIncrementInitialValueRule) checkCreateTable(ctx *mysql.CreateTableContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.CreateTableOptions() == nil || ctx.TableName() == nil {
		return
	}

	_, tableName := mysqlparser.NormalizeMySQLTableName(ctx.TableName())
	for _, option := range ctx.CreateTableOptions().AllCreateTableOption() {
		if option.AUTO_INCREMENT_SYMBOL() == nil || option.Ulonglong_number() == nil {
			continue
		}

		base := 10
		bitSize := 0
		value, err := strconv.ParseUint(option.Ulonglong_number().GetText(), base, bitSize)
		if err != nil {
			continue
		}
		if value != uint64(r.value) {
			r.AddAdvice(&storepb.Advice{
				Status:        r.level,
				Code:          code.AutoIncrementColumnInitialValueNotMatch.Int32(),
				Title:         r.title,
				Content:       fmt.Sprintf("The initial auto-increment value in table `%s` is %v, which doesn't equal %v", tableName, value, r.value),
				StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine()),
			})
		}
	}
}

// checkAlterTable is called when production alterTable is entered.
func (r *ColumnAutoIncrementInitialValueRule) checkAlterTable(ctx *mysql.AlterTableContext) {
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

	// alter table option.
	for _, option := range ctx.AlterTableActions().AlterCommandList().AlterList().AllCreateTableOptionsSpaceSeparated() {
		if option == nil {
			continue
		}
		for _, tableOption := range option.AllCreateTableOption() {
			if tableOption == nil || tableOption.AUTO_INCREMENT_SYMBOL() == nil || tableOption.Ulonglong_number() == nil {
				continue
			}

			base := 10
			bitSize := 0
			value, err := strconv.ParseUint(tableOption.Ulonglong_number().GetText(), base, bitSize)
			if err != nil {
				continue
			}
			if value != uint64(r.value) {
				r.AddAdvice(&storepb.Advice{
					Status:        r.level,
					Code:          code.AutoIncrementColumnInitialValueNotMatch.Int32(),
					Title:         r.title,
					Content:       fmt.Sprintf("The initial auto-increment value in table `%s` is %v, which doesn't equal %v", tableName, value, r.value),
					StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine()),
				})
			}
		}
	}
}
