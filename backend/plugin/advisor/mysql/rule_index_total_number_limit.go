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
	"github.com/bytebase/bytebase/backend/plugin/advisor/catalog"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*IndexTotalNumberLimitAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleIndexTotalNumberLimit, &IndexTotalNumberLimitAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.SchemaRuleIndexTotalNumberLimit, &IndexTotalNumberLimitAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.SchemaRuleIndexTotalNumberLimit, &IndexTotalNumberLimitAdvisor{})
}

// IndexTotalNumberLimitAdvisor is the advisor checking for index total number limit.
type IndexTotalNumberLimitAdvisor struct {
}

// Check checks for index total number limit.
func (*IndexTotalNumberLimitAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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
	rule := NewIndexTotalNumberLimitRule(level, string(checkCtx.Rule.Type), payload.Number, checkCtx.Catalog)

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range stmtList {
		rule.SetBaseLine(stmt.BaseLine)
		checker.SetBaseLine(stmt.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	// Check all tables after processing all statements
	rule.checkAllTables()

	return checker.GetAdviceList(), nil
}

// IndexTotalNumberLimitRule checks for index total number limit.
type IndexTotalNumberLimitRule struct {
	BaseRule
	max int
	// tableIndexes tracks indexes for each table across all statements
	tableIndexes map[string]map[string]bool // tableName -> indexName -> exists
	tableLines   map[string]int             // tableName -> last line number
	catalog      *catalog.Finder
}

// NewIndexTotalNumberLimitRule creates a new IndexTotalNumberLimitRule.
func NewIndexTotalNumberLimitRule(level storepb.Advice_Status, title string, maxIndexes int, catalogFinder *catalog.Finder) *IndexTotalNumberLimitRule {
	return &IndexTotalNumberLimitRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		max:          maxIndexes,
		tableIndexes: make(map[string]map[string]bool),
		tableLines:   make(map[string]int),
		catalog:      catalogFinder,
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

func (r *IndexTotalNumberLimitRule) checkCreateTable(ctx *mysql.CreateTableContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.TableName() == nil {
		return
	}

	_, tableName := mysqlparser.NormalizeMySQLTableName(ctx.TableName())

	// Initialize table indexes map
	if r.tableIndexes[tableName] == nil {
		r.tableIndexes[tableName] = make(map[string]bool)
	}

	// Extract indexes from table elements
	if ctx.TableElementList() != nil {
		for _, tableElement := range ctx.TableElementList().AllTableElement() {
			if tableElement == nil {
				continue
			}
			// Check column definitions for inline indexes
			if tableElement.ColumnDefinition() != nil && tableElement.ColumnDefinition().FieldDefinition() != nil {
				r.checkFieldDefinitionForIndexes(tableName, tableElement.ColumnDefinition().FieldDefinition())
			}
			// Check table constraints for indexes
			if tableElement.TableConstraintDef() != nil {
				r.checkTableConstraintForIndexes(tableName, tableElement.TableConstraintDef())
			}
		}
	}

	// Track last line for this table
	r.tableLines[tableName] = ctx.GetStart().GetLine()
}

func (r *IndexTotalNumberLimitRule) checkCreateIndex(ctx *mysql.CreateIndexContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.CreateIndexTarget() == nil || ctx.CreateIndexTarget().TableRef() == nil {
		return
	}
	_, tableName := mysqlparser.NormalizeMySQLTableRef(ctx.CreateIndexTarget().TableRef())

	// Initialize table indexes map
	if r.tableIndexes[tableName] == nil {
		r.tableIndexes[tableName] = make(map[string]bool)
	}

	// Add the index (use a generic name since we just need to count)
	indexName := fmt.Sprintf("__index_%d__", len(r.tableIndexes[tableName]))
	r.tableIndexes[tableName][indexName] = true

	// Track last line for this table
	r.tableLines[tableName] = ctx.GetStart().GetLine()
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

	// Initialize table indexes map
	if r.tableIndexes[tableName] == nil {
		r.tableIndexes[tableName] = make(map[string]bool)
	}

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
				r.checkFieldDefinitionForIndexes(tableName, item.FieldDefinition())
			// add multi columns.
			case item.OPEN_PAR_SYMBOL() != nil && item.TableElementList() != nil:
				for _, tableElement := range item.TableElementList().AllTableElement() {
					if tableElement.ColumnDefinition() != nil && tableElement.ColumnDefinition().FieldDefinition() != nil {
						r.checkFieldDefinitionForIndexes(tableName, tableElement.ColumnDefinition().FieldDefinition())
					}
					if tableElement.TableConstraintDef() != nil {
						r.checkTableConstraintForIndexes(tableName, tableElement.TableConstraintDef())
					}
				}
				// add constraint.
			case item.TableConstraintDef() != nil:
				r.checkTableConstraintForIndexes(tableName, item.TableConstraintDef())
			default:
			}
		// change column.
		case item.CHANGE_SYMBOL() != nil && item.ColumnInternalRef() != nil && item.Identifier() != nil:
			r.checkFieldDefinitionForIndexes(tableName, item.FieldDefinition())
		// modify column.
		case item.MODIFY_SYMBOL() != nil && item.ColumnInternalRef() != nil && item.FieldDefinition() != nil:
			r.checkFieldDefinitionForIndexes(tableName, item.FieldDefinition())
		default:
			continue
		}
	}

	// Track last line for this table
	r.tableLines[tableName] = ctx.GetStart().GetLine()
}

// checkFieldDefinitionForIndexes checks if a field definition contains inline index definitions.
func (r *IndexTotalNumberLimitRule) checkFieldDefinitionForIndexes(tableName string, ctx mysql.IFieldDefinitionContext) {
	for _, attr := range ctx.AllColumnAttribute() {
		if attr == nil || attr.GetValue() == nil {
			continue
		}
		switch attr.GetValue().GetTokenType() {
		case mysql.MySQLParserPRIMARY_SYMBOL, mysql.MySQLParserUNIQUE_SYMBOL, mysql.MySQLParserKEY_SYMBOL:
			// Generate a unique index name for inline index
			indexName := fmt.Sprintf("__inline_index_%d__", len(r.tableIndexes[tableName]))
			r.tableIndexes[tableName][indexName] = true
		default:
		}
	}
}

// checkTableConstraintForIndexes checks table constraints for index definitions.
func (r *IndexTotalNumberLimitRule) checkTableConstraintForIndexes(tableName string, ctx mysql.ITableConstraintDefContext) {
	if ctx.GetType_() == nil {
		return
	}
	switch ctx.GetType_().GetTokenType() {
	case mysql.MySQLParserPRIMARY_SYMBOL, mysql.MySQLParserUNIQUE_SYMBOL, mysql.MySQLParserKEY_SYMBOL, mysql.MySQLParserINDEX_SYMBOL, mysql.MySQLParserFULLTEXT_SYMBOL:
		// Add the index (use a generic name since we just need to count)
		indexName := fmt.Sprintf("__index_%d__", len(r.tableIndexes[tableName]))
		r.tableIndexes[tableName][indexName] = true
	default:
	}
}

// checkAllTables checks all tables' index counts after processing all statements.
func (r *IndexTotalNumberLimitRule) checkAllTables() {
	for tableName, indexes := range r.tableIndexes {
		// Get the number of indexes created in these statements
		newIndexes := len(indexes)

		// Get the number of indexes that already exist in catalog.Origin
		existingIndexes := 0
		if table := r.catalog.Origin.FindTable(&catalog.TableFind{TableName: tableName}); table != nil {
			existingIndexes = table.CountIndex()
		}

		totalCount := existingIndexes + newIndexes
		if totalCount > r.max {
			line := r.tableLines[tableName]
			r.AddAdvice(&storepb.Advice{
				Status:        r.level,
				Code:          advisor.IndexCountExceedsLimit.Int32(),
				Title:         r.title,
				Content:       fmt.Sprintf("The count of index in table `%s` should be no more than %d, but found %d", tableName, r.max, totalCount),
				StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + line),
			})
		}
	}
}
