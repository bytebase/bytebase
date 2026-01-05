package mysql

import (
	"context"
	"fmt"
	"slices"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/parser/mysql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	"github.com/bytebase/bytebase/backend/store/model"
)

var (
	_ advisor.Advisor = (*IndexTotalNumberLimitAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_INDEX_TOTAL_NUMBER_LIMIT, &IndexTotalNumberLimitAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_INDEX_TOTAL_NUMBER_LIMIT, &IndexTotalNumberLimitAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_INDEX_TOTAL_NUMBER_LIMIT, &IndexTotalNumberLimitAdvisor{})
}

// IndexTotalNumberLimitAdvisor is the advisor checking for index total number limit.
type IndexTotalNumberLimitAdvisor struct {
}

// Check checks for index total number limit.
func (*IndexTotalNumberLimitAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	numberPayload := checkCtx.Rule.GetNumberPayload()
	if numberPayload == nil {
		return nil, errors.New("number_payload is required for this rule")
	}

	// Create the rule
	rule := NewIndexTotalNumberLimitRule(level, checkCtx.Rule.Type.String(), int(numberPayload.Number), checkCtx.FinalMetadata)

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

	return rule.generateAdvice(), nil
}

// IndexTotalNumberLimitRule checks for index total number limit.
type IndexTotalNumberLimitRule struct {
	BaseRule
	max           int
	lineForTable  map[string]int
	finalMetadata *model.DatabaseMetadata
}

// NewIndexTotalNumberLimitRule creates a new IndexTotalNumberLimitRule.
func NewIndexTotalNumberLimitRule(level storepb.Advice_Status, title string, maxIndexes int, finalMetadata *model.DatabaseMetadata) *IndexTotalNumberLimitRule {
	return &IndexTotalNumberLimitRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		max:           maxIndexes,
		lineForTable:  make(map[string]int),
		finalMetadata: finalMetadata,
	}
}

// Name returns the rule name.
func (*IndexTotalNumberLimitRule) Name() string {
	return "IndexTotalNumberLimitRule"
}

// OnEnter is called when entering a parse tree node.
func (r *IndexTotalNumberLimitRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeCreateTable:
		r.checkCreateTable(ctx.(*mysql.CreateTableContext))
	case NodeTypeCreateIndex:
		r.checkCreateIndex(ctx.(*mysql.CreateIndexContext))
	case NodeTypeAlterTable:
		r.checkAlterTable(ctx.(*mysql.AlterTableContext))
	default:
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*IndexTotalNumberLimitRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *IndexTotalNumberLimitRule) generateAdvice() []*storepb.Advice {
	type tableName struct {
		name string
		line int
	}
	var tableList []tableName

	for k, v := range r.lineForTable {
		tableList = append(tableList, tableName{
			name: k,
			line: v,
		})
	}
	slices.SortFunc(tableList, func(i, j tableName) int {
		if i.line < j.line {
			return -1
		}
		if i.line > j.line {
			return 1
		}
		return 0
	})

	for _, table := range tableList {
		schema := r.finalMetadata.GetSchemaMetadata("")
		if schema == nil {
			continue
		}
		tableInfo := schema.GetTable(table.name)
		if tableInfo != nil && len(tableInfo.GetProto().Indexes) > r.max {
			r.AddAdvice(&storepb.Advice{
				Status:        r.level,
				Code:          code.IndexCountExceedsLimit.Int32(),
				Title:         r.title,
				Content:       fmt.Sprintf("The count of index in table `%s` should be no more than %d, but found %d", table.name, r.max, len(tableInfo.GetProto().Indexes)),
				StartPosition: common.ConvertANTLRLineToPosition(table.line),
			})
		}
	}

	return r.adviceList
}

func (r *IndexTotalNumberLimitRule) checkCreateTable(ctx *mysql.CreateTableContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.TableName() == nil {
		return
	}

	_, tableName := mysqlparser.NormalizeMySQLTableName(ctx.TableName())
	r.lineForTable[tableName] = r.baseLine + ctx.GetStart().GetLine()
}

func (r *IndexTotalNumberLimitRule) checkCreateIndex(ctx *mysql.CreateIndexContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.CreateIndexTarget() == nil || ctx.CreateIndexTarget().TableRef() == nil {
		return
	}
	_, tableName := mysqlparser.NormalizeMySQLTableRef(ctx.CreateIndexTarget().TableRef())
	r.lineForTable[tableName] = r.baseLine + ctx.GetStart().GetLine()
}

func (r *IndexTotalNumberLimitRule) checkAlterTable(ctx *mysql.AlterTableContext) {
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
		if item == nil {
			continue
		}

		switch {
		// add column.
		case item.ADD_SYMBOL() != nil:
			switch {
			// add single columns.
			case item.Identifier() != nil && item.FieldDefinition() != nil:
				r.checkFieldDefinitionContext(tableName, item.FieldDefinition())
			// add multi columns.
			case item.OPEN_PAR_SYMBOL() != nil && item.TableElementList() != nil:
				for _, tableElement := range item.TableElementList().AllTableElement() {
					if tableElement.ColumnDefinition() == nil || tableElement.ColumnDefinition().FieldDefinition() == nil {
						continue
					}
					r.checkFieldDefinitionContext(tableName, tableElement.ColumnDefinition().FieldDefinition())
				}
				// add constraint.
			case item.TableConstraintDef() != nil:
				r.checkTableConstraintDef(tableName, item.TableConstraintDef())
			default:
			}
		// change column.
		case item.CHANGE_SYMBOL() != nil && item.ColumnInternalRef() != nil && item.Identifier() != nil:
			r.checkFieldDefinitionContext(tableName, item.FieldDefinition())
		// modify column.
		case item.MODIFY_SYMBOL() != nil && item.ColumnInternalRef() != nil && item.FieldDefinition() != nil:
			r.checkFieldDefinitionContext(tableName, item.FieldDefinition())
		default:
			continue
		}
	}
}

func (r *IndexTotalNumberLimitRule) checkFieldDefinitionContext(tableName string, ctx mysql.IFieldDefinitionContext) {
	for _, attr := range ctx.AllColumnAttribute() {
		if attr == nil || attr.GetValue() == nil {
			continue
		}
		switch attr.GetValue().GetTokenType() {
		case mysql.MySQLParserPRIMARY_SYMBOL, mysql.MySQLParserUNIQUE_SYMBOL:
			r.lineForTable[tableName] = r.baseLine + ctx.GetStart().GetLine()
		default:
		}
	}
}

func (r *IndexTotalNumberLimitRule) checkTableConstraintDef(tableName string, ctx mysql.ITableConstraintDefContext) {
	if ctx.GetType_() == nil {
		return
	}
	switch ctx.GetType_().GetTokenType() {
	case mysql.MySQLParserPRIMARY_SYMBOL, mysql.MySQLParserUNIQUE_SYMBOL, mysql.MySQLParserKEY_SYMBOL, mysql.MySQLParserINDEX_SYMBOL, mysql.MySQLParserFULLTEXT_SYMBOL:
		r.lineForTable[tableName] = r.baseLine + ctx.GetStart().GetLine()
	default:
	}
}
