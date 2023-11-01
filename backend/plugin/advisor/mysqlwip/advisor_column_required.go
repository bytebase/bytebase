package mysqlwip

import (
	"fmt"
	"sort"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	mysql "github.com/bytebase/mysql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	// only for mysqlwip test.
	advisor.Register(storepb.Engine_ENGINE_UNSPECIFIED, advisor.MySQLColumnRequirement, &ColumnRequirementAdvisor{})
}

// ColumnRequirementAdvisor is the advisor checking for column requirement.
type ColumnRequirementAdvisor struct {
}

// Check checks for the column requirement.
func (*ColumnRequirementAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	list, ok := ctx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parser result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	columnList, err := advisor.UnmarshalRequiredColumnList(ctx.Rule.Payload)
	if err != nil {
		return nil, err
	}
	requiredColumns := make(columnSet)
	for _, column := range columnList {
		requiredColumns[column] = true
	}

	checker := &columnRequirementChecker{
		level:           level,
		title:           string(ctx.Rule.Type),
		requiredColumns: requiredColumns,
		tables:          make(tableState),
		line:            make(map[string]int),
	}

	for _, stmt := range list {
		checker.baseLine = stmt.BaseLine
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	return checker.generateAdviceList(), nil
}

type columnRequirementChecker struct {
	*mysql.BaseMySQLParserListener

	baseLine        int
	adviceList      []advisor.Advice
	level           advisor.Status
	title           string
	requiredColumns columnSet
	tables          tableState
	line            map[string]int
}

// EnterCreateDatabase is called when production createDatabase is entered.
func (checker *columnRequirementChecker) EnterCreateTable(ctx *mysql.CreateTableContext) {
	checker.createTable(ctx)
}

// EnterDropTable is called when production dropTable is entered.
func (checker *columnRequirementChecker) EnterDropTable(ctx *mysql.DropTableContext) {
	if ctx.TableRefList() == nil {
		return
	}

	for _, tableRef := range ctx.TableRefList().AllTableRef() {
		_, tableName := mysqlparser.NormalizeMySQLTableRef(tableRef)
		delete(checker.tables, tableName)
	}
}

// EnterAlterTable is called when production alterTable is entered.
func (checker *columnRequirementChecker) EnterAlterTable(ctx *mysql.AlterTableContext) {
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

		lineNumber := checker.baseLine + item.GetStart().GetLine()
		switch {
		// add column
		case item.ADD_SYMBOL() != nil && item.Identifier() != nil:
			columnName := mysqlparser.NormalizeMySQLIdentifier(item.Identifier())
			checker.addColumn(tableName, columnName)
		// drop column
		case item.DROP_SYMBOL() != nil && item.ColumnInternalRef() != nil:
			columnName := mysqlparser.NormalizeMySQLColumnInternalRef(item.ColumnInternalRef())
			if checker.dropColumn(tableName, columnName) {
				checker.line[tableName] = lineNumber
			}
		// rename column
		case item.RENAME_SYMBOL() != nil && item.COLUMN_SYMBOL() != nil:
			oldColumnName := mysqlparser.NormalizeMySQLColumnInternalRef(item.ColumnInternalRef())
			newColumnName := mysqlparser.NormalizeMySQLIdentifier(item.Identifier())
			checker.renameColumn(tableName, oldColumnName, newColumnName)
			checker.line[tableName] = lineNumber
		// change column
		case item.CHANGE_SYMBOL() != nil && item.ColumnInternalRef() != nil && item.Identifier() != nil:
			oldColumnName := mysqlparser.NormalizeMySQLColumnInternalRef(item.ColumnInternalRef())
			newColumnName := mysqlparser.NormalizeMySQLIdentifier(item.Identifier())
			if checker.renameColumn(tableName, oldColumnName, newColumnName) {
				checker.line[tableName] = lineNumber
			}
		}
	}
}

func (checker *columnRequirementChecker) generateAdviceList() []advisor.Advice {
	// Order it cause the random iteration order in Go, see https://go.dev/blog/maps
	tableList := checker.tables.tableList()
	for _, tableName := range tableList {
		table := checker.tables[tableName]
		var missingColumns []string
		for columnName := range checker.requiredColumns {
			if exists, ok := table[columnName]; !ok || !exists {
				missingColumns = append(missingColumns, columnName)
			}
		}

		if len(missingColumns) > 0 {
			// Order it cause the random iteration order in Go, see https://go.dev/blog/maps
			sort.Strings(missingColumns)
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    advisor.NoRequiredColumn,
				Title:   checker.title,
				Content: fmt.Sprintf("Table `%s` requires columns: %s", tableName, strings.Join(missingColumns, ", ")),
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

func (checker *columnRequirementChecker) createTable(ctx *mysql.CreateTableContext) {
	_, tableName := mysqlparser.NormalizeMySQLTableName(ctx.TableName())
	checker.line[tableName] = checker.baseLine + ctx.GetStart().GetLine()
	checker.initEmptyTable(tableName)

	if ctx.TableElementList() == nil {
		return
	}

	for _, tableElement := range ctx.TableElementList().AllTableElement() {
		if tableElement.ColumnDefinition() == nil {
			continue
		}
		_, _, columnName := mysqlparser.NormalizeMySQLColumnName(tableElement.ColumnDefinition().ColumnName())
		checker.addColumn(tableName, columnName)
	}
}

func (checker *columnRequirementChecker) initEmptyTable(tableName string) columnSet {
	checker.tables[tableName] = make(columnSet)
	return checker.tables[tableName]
}

// add a column.
func (checker *columnRequirementChecker) addColumn(tableName string, columnName string) {
	if _, ok := checker.requiredColumns[columnName]; !ok {
		return
	}

	if table, ok := checker.tables[tableName]; !ok {
		// We do not retrospectively check.
		// So we assume it contains all required columns.
		checker.initFullTable(tableName)
	} else {
		table[columnName] = true
	}
}

// drop a column
// return true if the colum was successfully dropped from requirement list.
func (checker *columnRequirementChecker) dropColumn(tableName string, columnName string) bool {
	if _, ok := checker.requiredColumns[columnName]; !ok {
		return false
	}
	table, ok := checker.tables[tableName]
	if !ok {
		// We do not retrospectively check.
		// So we assume it contains all required columns.
		table = checker.initFullTable(tableName)
	}
	table[columnName] = false
	return true
}

// rename a column
// return if the old column was dropped from requirement list.
func (checker *columnRequirementChecker) renameColumn(tableName string, oldColumn string, newColumn string) bool {
	_, oldNeed := checker.requiredColumns[oldColumn]
	_, newNeed := checker.requiredColumns[newColumn]
	if !oldNeed && !newNeed {
		return false
	}
	table, ok := checker.tables[tableName]
	if !ok {
		// We do not retrospectively check.
		// So we assume it contains all required columns.
		table = checker.initFullTable(tableName)
	}
	if oldNeed {
		table[oldColumn] = false
	}
	if newNeed {
		table[newColumn] = true
	}
	return oldNeed
}

func (checker *columnRequirementChecker) initFullTable(tableName string) columnSet {
	table := checker.initEmptyTable(tableName)
	for column := range checker.requiredColumns {
		table[column] = true
	}
	return table
}
