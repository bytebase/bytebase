package mysql

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/parser/mysql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	"github.com/bytebase/bytebase/backend/store/model"
)

var (
	_ advisor.Advisor = (*ColumnDisallowDropInIndexAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_COLUMN_DISALLOW_DROP_IN_INDEX, &ColumnDisallowDropInIndexAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_COLUMN_DISALLOW_DROP_IN_INDEX, &ColumnDisallowDropInIndexAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_COLUMN_DISALLOW_DROP_IN_INDEX, &ColumnDisallowDropInIndexAdvisor{})
}

// ColumnDisallowDropInIndexAdvisor is the advisor checking for disallow DROP COLUMN in index.
type ColumnDisallowDropInIndexAdvisor struct {
}

// Check checks for disallow Drop COLUMN in index statement.
func (*ColumnDisallowDropInIndexAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewColumnDisallowDropInIndexRule(level, checkCtx.Rule.Type.String(), checkCtx.OriginalMetadata)

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range checkCtx.ParsedStatements {
		rule.SetBaseLine(stmt.BaseLine())
		checker.SetBaseLine(stmt.BaseLine())
		if stmt.AST == nil {
			continue
		}
		antlrAST, ok := base.GetANTLRAST(stmt.AST)
		if !ok {
			continue
		}
		antlr.ParseTreeWalkerDefault.Walk(checker, antlrAST.Tree)
	}

	return checker.GetAdviceList(), nil
}

// ColumnDisallowDropInIndexRule checks for disallow DROP COLUMN in index.
type ColumnDisallowDropInIndexRule struct {
	BaseRule
	tables           tableState
	originalMetadata *model.DatabaseMetadata
}

// NewColumnDisallowDropInIndexRule creates a new ColumnDisallowDropInIndexRule.
func NewColumnDisallowDropInIndexRule(level storepb.Advice_Status, title string, originalMetadata *model.DatabaseMetadata) *ColumnDisallowDropInIndexRule {
	return &ColumnDisallowDropInIndexRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		tables:           make(tableState),
		originalMetadata: originalMetadata,
	}
}

// Name returns the rule name.
func (*ColumnDisallowDropInIndexRule) Name() string {
	return "ColumnDisallowDropInIndexRule"
}

// OnEnter is called when entering a parse tree node.
func (r *ColumnDisallowDropInIndexRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
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
func (*ColumnDisallowDropInIndexRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *ColumnDisallowDropInIndexRule) checkCreateTable(ctx *mysql.CreateTableContext) {
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
		if tableElement == nil || tableElement.TableConstraintDef() == nil {
			continue
		}
		if tableElement.TableConstraintDef().GetType_() == nil {
			continue
		}
		switch tableElement.TableConstraintDef().GetType_().GetTokenType() {
		case mysql.MySQLParserINDEX_SYMBOL, mysql.MySQLParserKEY_SYMBOL:
			// do nothing.
		default:
			continue
		}
		if tableElement.TableConstraintDef().KeyListVariants() == nil {
			continue
		}

		columnList := mysqlparser.NormalizeKeyListVariants(tableElement.TableConstraintDef().KeyListVariants())
		for _, column := range columnList {
			if r.tables[tableName] == nil {
				r.tables[tableName] = make(columnSet)
			}
			r.tables[tableName][column] = true
		}
	}
}

func (r *ColumnDisallowDropInIndexRule) checkAlterTable(ctx *mysql.AlterTableContext) {
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
	for _, item := range ctx.AlterTableActions().AlterCommandList().AlterList().AllAlterListItem() {
		if item == nil || item.DROP_SYMBOL() == nil || item.ColumnInternalRef() == nil {
			continue
		}

		table := r.originalMetadata.GetSchemaMetadata("").GetTable(tableName)
		if table != nil {
			if r.tables[tableName] == nil {
				r.tables[tableName] = make(columnSet)
			}
			for _, indexColumn := range table.ListIndexes() {
				for _, column := range indexColumn.GetProto().GetExpressions() {
					r.tables[tableName][column] = true
				}
			}
		}

		columnName := mysqlparser.NormalizeMySQLColumnInternalRef(item.ColumnInternalRef())
		if !r.canDrop(tableName, columnName) {
			r.AddAdvice(&storepb.Advice{
				Status:        r.level,
				Code:          code.DropIndexColumn.Int32(),
				Title:         r.title,
				Content:       fmt.Sprintf("`%s`.`%s` cannot drop index column", tableName, columnName),
				StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + item.GetStart().GetLine()),
			})
		}
	}
}

func (r *ColumnDisallowDropInIndexRule) canDrop(table string, column string) bool {
	if _, ok := r.tables[table][column]; ok {
		return false
	}
	return true
}
