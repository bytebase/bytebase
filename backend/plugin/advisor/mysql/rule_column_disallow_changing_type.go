package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	"github.com/bytebase/parser/mysql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/catalog"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*ColumnDisallowChangingTypeAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleColumnDisallowChangeType, &ColumnDisallowChangingTypeAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.SchemaRuleColumnDisallowChangeType, &ColumnDisallowChangingTypeAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.SchemaRuleColumnDisallowChangeType, &ColumnDisallowChangingTypeAdvisor{})
}

// ColumnDisallowChangingTypeAdvisor is the advisor checking for disallow changing column type.
type ColumnDisallowChangingTypeAdvisor struct {
}

// Check checks for disallow changing column type.
func (*ColumnDisallowChangingTypeAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parser result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewColumnDisallowChangingTypeRule(level, string(checkCtx.Rule.Type), checkCtx.OriginCatalog)

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range stmtList {
		rule.SetBaseLine(stmt.BaseLine)
		checker.SetBaseLine(stmt.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	return checker.GetAdviceList(), nil
}

// ColumnDisallowChangingTypeRule checks for disallow changing column type.
type ColumnDisallowChangingTypeRule struct {
	BaseRule
	text          string
	originCatalog *catalog.DatabaseState
}

// NewColumnDisallowChangingTypeRule creates a new ColumnDisallowChangingTypeRule.
func NewColumnDisallowChangingTypeRule(level storepb.Advice_Status, title string, originCatalog *catalog.DatabaseState) *ColumnDisallowChangingTypeRule {
	return &ColumnDisallowChangingTypeRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		originCatalog: originCatalog,
	}
}

// Name returns the rule name.
func (*ColumnDisallowChangingTypeRule) Name() string {
	return "ColumnDisallowChangingTypeRule"
}

// OnEnter is called when entering a parse tree node.
func (r *ColumnDisallowChangingTypeRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeQuery:
		queryCtx, ok := ctx.(*mysql.QueryContext)
		if !ok {
			return nil
		}
		r.text = queryCtx.GetParser().GetTokenStream().GetTextFromRuleContext(queryCtx)
	case NodeTypeAlterTable:
		r.checkAlterTable(ctx.(*mysql.AlterTableContext))
	default:
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*ColumnDisallowChangingTypeRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *ColumnDisallowChangingTypeRule) checkAlterTable(ctx *mysql.AlterTableContext) {
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
		// change column
		case item.CHANGE_SYMBOL() != nil && item.ColumnInternalRef() != nil && item.Identifier() != nil:
			// only focus on old colunn-name.
			columnName = mysqlparser.NormalizeMySQLColumnInternalRef(item.ColumnInternalRef())
		// MODIFY COLUMN
		case item.MODIFY_SYMBOL() != nil && item.ColumnInternalRef() != nil && item.FieldDefinition() != nil:
			columnName = mysqlparser.NormalizeMySQLColumnInternalRef(item.ColumnInternalRef())
		default:
			continue
		}
		r.changeColumnType(tableName, columnName, item.FieldDefinition().DataType())
	}
}

func normalizeColumnType(tp string) string {
	switch strings.ToLower(tp) {
	case "tinyint":
		return "tinyint(4)"
	case "tinyint unsigned":
		return "tinyint(4) unsigned"
	case "smallint":
		return "smallint(6)"
	case "smallint unsigned":
		return "smallint(6) unsigned"
	case "mediumint":
		return "mediumint(9)"
	case "mediumint unsigned":
		return "mediumint(9) unsigned"
	case "int":
		return "int(11)"
	case "int unsigned":
		return "int(11) unsigned"
	case "bigint":
		return "bigint(20)"
	case "bigint unsigned":
		return "bigint(20) unsigned"
	default:
		return strings.ToLower(tp)
	}
}

func (r *ColumnDisallowChangingTypeRule) changeColumnType(tableName, columnName string, dataType mysql.IDataTypeContext) {
	tp := dataType.GetParser().GetTokenStream().GetTextFromRuleContext(dataType)
	column := r.originCatalog.GetColumn("", tableName, columnName)

	if column == nil {
		return
	}

	if normalizeColumnType(column.Type()) != normalizeColumnType(tp) {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.ChangeColumnType.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("\"%s\" changes column type", r.text),
			StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + dataType.GetStart().GetLine()),
		})
	}
}
