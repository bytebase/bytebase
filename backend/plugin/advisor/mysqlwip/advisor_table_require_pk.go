package mysqlwip

import (
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	mysql "github.com/bytebase/mysql-parser"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/catalog"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

const (
	primaryKeyName = "PRIMARY"
)

var (
	_ advisor.Advisor = (*TableRequirePKAdvisor)(nil)
)

func init() {
	// only for mysqlwip test.
	advisor.Register(storepb.Engine_ENGINE_UNSPECIFIED, advisor.MySQLTableRequirePK, &TableRequirePKAdvisor{})
}

// TableRequirePKAdvisor is the advisor checking table requires PK.
type TableRequirePKAdvisor struct {
}

// Check checks table requires PK.
func (*TableRequirePKAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	root, ok := ctx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	checker := &tableRequirePKChecker{
		level:   level,
		title:   string(ctx.Rule.Type),
		tables:  make(tablePK),
		line:    make(map[string]int),
		catalog: ctx.Catalog,
	}

	for _, stmtNode := range root {
		checker.baseLine = stmtNode.BaseLine
		antlr.ParseTreeWalkerDefault.Walk(checker, stmtNode.Tree)
	}

	return checker.generateAdviceList(), nil
}

type tableRequirePKChecker struct {
	*mysql.BaseMySQLParserListener

	baseLine   int
	adviceList []advisor.Advice
	level      advisor.Status
	title      string
	tables     tablePK
	line       map[string]int
	catalog    *catalog.Finder
}

// EnterCreateTable is called when production createTable is entered.
func (checker *tableRequirePKChecker) EnterCreateTable(ctx *mysql.CreateTableContext) {
	if ctx.TableName() == nil {
		return
	}
	_, tableName := mysqlparser.NormalizeMySQLTableName(ctx.TableName())
	checker.createTable(tableName, ctx)

	checker.line[tableName] = checker.baseLine + ctx.GetStart().GetLine()
}

func (checker *tableRequirePKChecker) createTable(tableName string, ctx *mysql.CreateTableContext) {
	checker.initEmptyTable(tableName)

	for _, tableElement := range ctx.TableElementList().AllTableElement() {
		switch {
		// add primary key from column definition.
		case tableElement.ColumnDefinition() != nil:
			if tableElement.ColumnDefinition().ColumnName() == nil || tableElement.ColumnDefinition().FieldDefinition() == nil {
				continue
			}
			_, _, columnName := mysqlparser.NormalizeMySQLColumnName(tableElement.ColumnDefinition().ColumnName())
			checker.handleFieldDefinition(tableName, columnName, tableElement.ColumnDefinition().FieldDefinition())
		// add primary key from table constraint.
		case tableElement.TableConstraintDef() != nil:
			checker.handleTableConstraintDef(tableName, tableElement.TableConstraintDef())
		}
	}
}

func (checker *tableRequirePKChecker) handleFieldDefinition(tableName string, columnName string, ctx mysql.IFieldDefinitionContext) {
	for _, attr := range ctx.AllColumnAttribute() {
		if attr.GetValue().GetTokenType() == mysql.MySQLLexerKEY_SYMBOL {
			checker.tables[tableName] = newColumnSet([]string{columnName})
		}
	}
}

func (checker *tableRequirePKChecker) handleTableConstraintDef(tableName string, ctx mysql.ITableConstraintDefContext) {
	if ctx.GetType_() != nil {
		switch strings.ToUpper(ctx.GetType_().GetText()) {
		case "PRIMARY":
			list := mysqlparser.NormalizeKeyListVariants(ctx.KeyListVariants())
			checker.tables[tableName] = newColumnSet(list)
		default:
		}
	}
}

// EnterDropTable is called when production dropTable is entered.
func (checker *tableRequirePKChecker) EnterDropTable(ctx *mysql.DropTableContext) {
	for _, tableRef := range ctx.TableRefList().AllTableRef() {
		_, tableName := mysqlparser.NormalizeMySQLTableRef(tableRef)
		delete(checker.tables, tableName)
	}
}

// EnterAlterTable is called when production alterTable is entered.
func (checker *tableRequirePKChecker) EnterAlterTable(ctx *mysql.AlterTableContext) {
	_, tableName := mysqlparser.NormalizeMySQLTableRef(ctx.TableRef())

	lineNumber := checker.baseLine + ctx.GetStart().GetLine()
	for _, option := range ctx.AlterTableActions().AlterCommandList().AlterList().AllAlterListItem() {
		switch {
		// ADD CONSTRANIT
		case option.ADD_SYMBOL() != nil && option.TableConstraintDef() != nil:
			checker.handleTableConstraintDef(tableName, option.TableConstraintDef())
		// DROP PRIMARY KEY
		case option.DROP_SYMBOL() != nil && option.PRIMARY_SYMBOL() != nil:
			checker.initEmptyTable(tableName)
			checker.line[tableName] = lineNumber
		// DROP INDEX/KEY
		case option.DROP_SYMBOL() != nil && option.KeyOrIndex() != nil:
			_, _, indexName := mysqlparser.NormalizeIndexRef(option.IndexRef())
			if strings.ToUpper(indexName) == primaryKeyName {
				checker.initEmptyTable(tableName)
				checker.line[tableName] = lineNumber
			}
		// ADD COLUMNS
		case option.ADD_SYMBOL() != nil && option.FieldDefinition() != nil:
			columnName := mysqlparser.NormalizeMySQLIdentifier(option.Identifier())
			checker.handleFieldDefinition(tableName, columnName, option.FieldDefinition())
		// CHANGE COLUMN
		case option.CHANGE_SYMBOL() != nil && option.ColumnInternalRef() != nil && option.Identifier() != nil && option.FieldDefinition() != nil:
			oldColumn := mysqlparser.NormalizeMySQLColumnInternalRef(option.ColumnInternalRef())
			newColumn := mysqlparser.NormalizeMySQLIdentifier(option.Identifier())
			if checker.changeColumn(tableName, oldColumn, newColumn) {
				checker.line[tableName] = lineNumber
			}
			checker.handleFieldDefinition(tableName, newColumn, option.FieldDefinition())
		// MODIFY COLUMN
		case option.MODIFY_SYMBOL() != nil && option.ColumnInternalRef() != nil && option.FieldDefinition() != nil:
			columnName := mysqlparser.NormalizeMySQLColumnInternalRef(option.ColumnInternalRef())
			checker.handleFieldDefinition(tableName, columnName, option.FieldDefinition())
		// DROP COLUMN
		case option.DROP_SYMBOL() != nil:
			columnName := mysqlparser.NormalizeMySQLColumnInternalRef(option.ColumnInternalRef())
			if checker.dropColumn(tableName, columnName) {
				checker.line[tableName] = lineNumber
			}
		}
	}
}

func (checker *tableRequirePKChecker) generateAdviceList() []advisor.Advice {
	tableList := checker.tables.tableList()
	for _, tableName := range tableList {
		if len(checker.tables[tableName]) == 0 {
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    advisor.TableNoPK,
				Title:   checker.title,
				Content: fmt.Sprintf("Table `%s` requires PRIMARY KEY", tableName),
				Line:    checker.line[tableName],
			})
		}
	}

	if len(checker.adviceList) == 0 {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  advisor.Success,
			Code:    advisor.Ok,
			Title:   "OK",
			Content: "",
		})
	}
	return checker.adviceList
}

func (checker *tableRequirePKChecker) changeColumn(tableName string, oldColumn string, newColumn string) bool {
	if checker.dropColumn(tableName, oldColumn) {
		pk := checker.tables[tableName]
		pk[newColumn] = true
		return true
	}
	return false
}

func (checker *tableRequirePKChecker) dropColumn(tableName string, columnName string) bool {
	if _, ok := checker.tables[tableName]; !ok {
		_, pk := checker.catalog.Origin.FindIndex(&catalog.IndexFind{
			TableName: tableName,
			IndexName: primaryKeyName,
		})
		if pk == nil {
			return false
		}
		checker.tables[tableName] = newColumnSet(pk.ExpressionList())
	}

	pk := checker.tables[tableName]
	_, columnInPk := pk[columnName]
	delete(checker.tables[tableName], columnName)
	return columnInPk
}

func (checker *tableRequirePKChecker) initEmptyTable(name string) {
	checker.tables[name] = make(columnSet)
}
