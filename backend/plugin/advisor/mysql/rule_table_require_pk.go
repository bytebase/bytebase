package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	"github.com/bytebase/parser/mysql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/catalog"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

const (
	primaryKeyName = "PRIMARY"
)

var (
	_ advisor.Advisor = (*TableRequirePKAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleTableRequirePK, &TableRequirePKAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.SchemaRuleTableRequirePK, &TableRequirePKAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.SchemaRuleTableRequirePK, &TableRequirePKAdvisor{})
}

// TableRequirePKAdvisor is the advisor checking table requires PK.
type TableRequirePKAdvisor struct {
}

// Check checks table requires PK.
func (*TableRequirePKAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	root, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewTableRequirePKRule(level, string(checkCtx.Rule.Type), checkCtx.OriginCatalog)

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmtNode := range root {
		rule.SetBaseLine(stmtNode.BaseLine)
		checker.SetBaseLine(stmtNode.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmtNode.Tree)
	}

	// Generate advice after walking
	rule.generateAdviceList()

	return checker.GetAdviceList(), nil
}

// TableRequirePKRule checks table requires PK.
type TableRequirePKRule struct {
	BaseRule
	tables        map[string]columnSet
	line          map[string]int
	originCatalog *catalog.DatabaseState
}

// NewTableRequirePKRule creates a new TableRequirePKRule.
func NewTableRequirePKRule(level storepb.Advice_Status, title string, originCatalog *catalog.DatabaseState) *TableRequirePKRule {
	return &TableRequirePKRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		tables:        make(map[string]columnSet),
		line:          make(map[string]int),
		originCatalog: originCatalog,
	}
}

// Name returns the rule name.
func (*TableRequirePKRule) Name() string {
	return "TableRequirePKRule"
}

// OnEnter is called when entering a parse tree node.
func (r *TableRequirePKRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeCreateTable:
		r.checkCreateTable(ctx.(*mysql.CreateTableContext))
	case NodeTypeDropTable:
		r.checkDropTable(ctx.(*mysql.DropTableContext))
	case NodeTypeAlterTable:
		r.checkAlterTable(ctx.(*mysql.AlterTableContext))
	default:
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*TableRequirePKRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	// This rule doesn't need exit processing
	return nil
}

func (r *TableRequirePKRule) checkCreateTable(ctx *mysql.CreateTableContext) {
	if ctx.TableName() == nil {
		return
	}
	_, tableName := mysqlparser.NormalizeMySQLTableName(ctx.TableName())
	r.createTable(tableName, ctx)
	r.line[tableName] = r.baseLine + ctx.GetStart().GetLine()
}

func (r *TableRequirePKRule) createTable(tableName string, ctx *mysql.CreateTableContext) {
	if ctx.TableElementList() == nil {
		return
	}
	r.initEmptyTable(tableName)

	for _, tableElement := range ctx.TableElementList().AllTableElement() {
		switch {
		// add primary key from column definition.
		case tableElement.ColumnDefinition() != nil:
			if tableElement.ColumnDefinition().ColumnName() == nil || tableElement.ColumnDefinition().FieldDefinition() == nil {
				continue
			}
			_, _, columnName := mysqlparser.NormalizeMySQLColumnName(tableElement.ColumnDefinition().ColumnName())
			r.handleFieldDefinition(tableName, columnName, tableElement.ColumnDefinition().FieldDefinition())
		// add primary key from table constraint.
		case tableElement.TableConstraintDef() != nil:
			r.handleTableConstraintDef(tableName, tableElement.TableConstraintDef())
		default:
		}
	}
}

func (r *TableRequirePKRule) handleFieldDefinition(tableName string, columnName string, ctx mysql.IFieldDefinitionContext) {
	for _, attr := range ctx.AllColumnAttribute() {
		if attr == nil || attr.PRIMARY_SYMBOL() == nil {
			continue
		}
		r.tables[tableName] = newColumnSet([]string{columnName})
	}
}

func (r *TableRequirePKRule) handleTableConstraintDef(tableName string, ctx mysql.ITableConstraintDefContext) {
	if ctx.GetType_() != nil {
		if ctx.GetType_().GetTokenType() == mysql.MySQLParserPRIMARY_SYMBOL {
			list := mysqlparser.NormalizeKeyListVariants(ctx.KeyListVariants())
			r.tables[tableName] = newColumnSet(list)
		}
	}
}

func (r *TableRequirePKRule) checkDropTable(ctx *mysql.DropTableContext) {
	if ctx.TableRefList() == nil {
		return
	}
	for _, tableRef := range ctx.TableRefList().AllTableRef() {
		_, tableName := mysqlparser.NormalizeMySQLTableRef(tableRef)
		delete(r.tables, tableName)
	}
}

func (r *TableRequirePKRule) checkAlterTable(ctx *mysql.AlterTableContext) {
	if ctx.TableRef() == nil {
		return
	}
	if ctx.AlterTableActions() == nil || ctx.AlterTableActions().AlterCommandList() == nil || ctx.AlterTableActions().AlterCommandList() == nil {
		return
	}
	_, tableName := mysqlparser.NormalizeMySQLTableRef(ctx.TableRef())

	lineNumber := r.baseLine + ctx.GetStart().GetLine()
	for _, option := range ctx.AlterTableActions().AlterCommandList().AlterList().AllAlterListItem() {
		switch {
		// ADD CONSTRANIT
		case option.ADD_SYMBOL() != nil && option.TableConstraintDef() != nil:
			r.handleTableConstraintDef(tableName, option.TableConstraintDef())
		// DROP PRIMARY KEY
		case option.DROP_SYMBOL() != nil && option.PRIMARY_SYMBOL() != nil:
			r.initEmptyTable(tableName)
			r.line[tableName] = lineNumber
			// DROP INDEX/KEY
		case option.DROP_SYMBOL() != nil && option.KeyOrIndex() != nil && option.IndexRef() != nil:
			_, _, indexName := mysqlparser.NormalizeIndexRef(option.IndexRef())
			if strings.ToUpper(indexName) == primaryKeyName {
				r.initEmptyTable(tableName)
				r.line[tableName] = lineNumber
			}
		// ADD COLUMNS
		case option.ADD_SYMBOL() != nil && option.FieldDefinition() != nil:
			columnName := mysqlparser.NormalizeMySQLIdentifier(option.Identifier())
			r.handleFieldDefinition(tableName, columnName, option.FieldDefinition())
		// CHANGE COLUMN
		case option.CHANGE_SYMBOL() != nil && option.ColumnInternalRef() != nil && option.Identifier() != nil && option.FieldDefinition() != nil:
			oldColumn := mysqlparser.NormalizeMySQLColumnInternalRef(option.ColumnInternalRef())
			newColumn := mysqlparser.NormalizeMySQLIdentifier(option.Identifier())
			if r.changeColumn(tableName, oldColumn, newColumn) {
				r.line[tableName] = lineNumber
			}
			r.handleFieldDefinition(tableName, newColumn, option.FieldDefinition())
		// MODIFY COLUMN
		case option.MODIFY_SYMBOL() != nil && option.ColumnInternalRef() != nil && option.FieldDefinition() != nil:
			columnName := mysqlparser.NormalizeMySQLColumnInternalRef(option.ColumnInternalRef())
			r.handleFieldDefinition(tableName, columnName, option.FieldDefinition())
		// DROP COLUMN
		case option.DROP_SYMBOL() != nil && option.ColumnInternalRef() != nil:
			columnName := mysqlparser.NormalizeMySQLColumnInternalRef(option.ColumnInternalRef())
			if r.dropColumn(tableName, columnName) {
				r.line[tableName] = lineNumber
			}
		default:
		}
	}
}

func (r *TableRequirePKRule) generateAdviceList() {
	tableList := r.getTableList()
	for _, tableName := range tableList {
		if len(r.tables[tableName]) == 0 {
			r.AddAdvice(&storepb.Advice{
				Status:        r.level,
				Code:          code.TableNoPK.Int32(),
				Title:         r.title,
				Content:       fmt.Sprintf("Table `%s` requires PRIMARY KEY", tableName),
				StartPosition: common.ConvertANTLRLineToPosition(r.line[tableName]),
			})
		}
	}
}

func (r *TableRequirePKRule) changeColumn(tableName string, oldColumn string, newColumn string) bool {
	if r.dropColumn(tableName, oldColumn) {
		pk := r.tables[tableName]
		pk[newColumn] = true
		return true
	}
	return false
}

func (r *TableRequirePKRule) dropColumn(tableName string, columnName string) bool {
	if _, ok := r.tables[tableName]; !ok {
		_, pk := r.originCatalog.GetIndex("", tableName, primaryKeyName)
		if pk == nil {
			return false
		}
		r.tables[tableName] = newColumnSet(pk.ExpressionList())
	}

	pk := r.tables[tableName]
	_, columnInPk := pk[columnName]
	delete(r.tables[tableName], columnName)
	return columnInPk
}

func (r *TableRequirePKRule) initEmptyTable(name string) {
	r.tables[name] = make(columnSet)
}

func (r *TableRequirePKRule) getTableList() []string {
	var tableList []string
	for tableName := range r.tables {
		tableList = append(tableList, tableName)
	}
	return tableList
}
