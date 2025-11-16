package mysql

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/parser/mysql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*ColumnRequirementAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleRequiredColumn, &ColumnRequirementAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.SchemaRuleRequiredColumn, &ColumnRequirementAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.SchemaRuleRequiredColumn, &ColumnRequirementAdvisor{})
}

// ColumnRequirementAdvisor is the advisor checking for column requirement.
type ColumnRequirementAdvisor struct {
}

// Check checks for the column requirement.
func (*ColumnRequirementAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	list, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parser result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	columnList, err := advisor.UnmarshalRequiredColumnList(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}
	requiredColumns := make(columnSet)
	for _, column := range columnList {
		requiredColumns[column] = true
	}

	// Create the rule
	rule := NewColumnRequiredRule(level, string(checkCtx.Rule.Type), requiredColumns)

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range list {
		rule.SetBaseLine(stmt.BaseLine)
		checker.SetBaseLine(stmt.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	return rule.generateAdviceList(), nil
}

// ColumnRequiredRule checks for column requirement.
type ColumnRequiredRule struct {
	BaseRule
	requiredColumns columnSet
	tables          tableState
	line            map[string]int
}

// NewColumnRequiredRule creates a new ColumnRequiredRule.
func NewColumnRequiredRule(level storepb.Advice_Status, title string, requiredColumns columnSet) *ColumnRequiredRule {
	return &ColumnRequiredRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		requiredColumns: requiredColumns,
		tables:          make(tableState),
		line:            make(map[string]int),
	}
}

// Name returns the rule name.
func (*ColumnRequiredRule) Name() string {
	return "ColumnRequiredRule"
}

// OnEnter is called when entering a parse tree node.
func (r *ColumnRequiredRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
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
func (*ColumnRequiredRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *ColumnRequiredRule) checkCreateTable(ctx *mysql.CreateTableContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	r.createTable(ctx)
}

func (r *ColumnRequiredRule) checkDropTable(ctx *mysql.DropTableContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.TableRefList() == nil {
		return
	}

	for _, tableRef := range ctx.TableRefList().AllTableRef() {
		_, tableName := mysqlparser.NormalizeMySQLTableRef(tableRef)
		delete(r.tables, tableName)
	}
}

func (r *ColumnRequiredRule) checkAlterTable(ctx *mysql.AlterTableContext) {
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

		lineNumber := r.baseLine + item.GetStart().GetLine()
		switch {
		// add column
		case item.ADD_SYMBOL() != nil:
			switch {
			case item.Identifier() != nil && item.FieldDefinition() != nil:
				columnName := mysqlparser.NormalizeMySQLIdentifier(item.Identifier())
				r.addColumn(tableName, columnName)
			case item.OPEN_PAR_SYMBOL() != nil && item.TableElementList() != nil:
				for _, tableElement := range item.TableElementList().AllTableElement() {
					if tableElement.ColumnDefinition() == nil || tableElement.ColumnDefinition().ColumnName() == nil || tableElement.ColumnDefinition().FieldDefinition() == nil {
						continue
					}
					_, _, columnName := mysqlparser.NormalizeMySQLColumnName(tableElement.ColumnDefinition().ColumnName())
					r.addColumn(tableName, columnName)
				}
			default:
			}
		// drop column
		case item.DROP_SYMBOL() != nil && item.ColumnInternalRef() != nil:
			columnName := mysqlparser.NormalizeMySQLColumnInternalRef(item.ColumnInternalRef())
			if r.dropColumn(tableName, columnName) {
				r.line[tableName] = lineNumber
			}
		// rename column
		case item.RENAME_SYMBOL() != nil && item.COLUMN_SYMBOL() != nil:
			oldColumnName := mysqlparser.NormalizeMySQLColumnInternalRef(item.ColumnInternalRef())
			newColumnName := mysqlparser.NormalizeMySQLIdentifier(item.Identifier())
			r.renameColumn(tableName, oldColumnName, newColumnName)
			r.line[tableName] = lineNumber
		// change column
		case item.CHANGE_SYMBOL() != nil && item.ColumnInternalRef() != nil && item.Identifier() != nil:
			oldColumnName := mysqlparser.NormalizeMySQLColumnInternalRef(item.ColumnInternalRef())
			newColumnName := mysqlparser.NormalizeMySQLIdentifier(item.Identifier())
			if r.renameColumn(tableName, oldColumnName, newColumnName) {
				r.line[tableName] = lineNumber
			}
		default:
		}
	}
}

func (r *ColumnRequiredRule) generateAdviceList() []*storepb.Advice {
	// Order it cause the random iteration order in Go, see https://go.dev/blog/maps
	tableList := r.tables.tableList()
	for _, tableName := range tableList {
		table := r.tables[tableName]
		var missingColumns []string
		for columnName := range r.requiredColumns {
			if exists, ok := table[columnName]; !ok || !exists {
				missingColumns = append(missingColumns, columnName)
			}
		}

		if len(missingColumns) > 0 {
			// Order it cause the random iteration order in Go, see https://go.dev/blog/maps
			slices.Sort(missingColumns)
			r.AddAdvice(&storepb.Advice{
				Status:        r.level,
				Code:          code.NoRequiredColumn.Int32(),
				Title:         r.title,
				Content:       fmt.Sprintf("Table `%s` requires columns: %s", tableName, strings.Join(missingColumns, ", ")),
				StartPosition: common.ConvertANTLRLineToPosition(r.line[tableName]),
			})
		}
	}

	return r.adviceList
}

func (r *ColumnRequiredRule) createTable(ctx *mysql.CreateTableContext) {
	_, tableName := mysqlparser.NormalizeMySQLTableName(ctx.TableName())
	r.line[tableName] = r.baseLine + ctx.GetStart().GetLine()
	r.initEmptyTable(tableName)

	if ctx.TableElementList() == nil {
		return
	}

	for _, tableElement := range ctx.TableElementList().AllTableElement() {
		if tableElement.ColumnDefinition() == nil {
			continue
		}
		_, _, columnName := mysqlparser.NormalizeMySQLColumnName(tableElement.ColumnDefinition().ColumnName())
		r.addColumn(tableName, columnName)
	}
}

func (r *ColumnRequiredRule) initEmptyTable(tableName string) columnSet {
	r.tables[tableName] = make(columnSet)
	return r.tables[tableName]
}

// add a column.
func (r *ColumnRequiredRule) addColumn(tableName string, columnName string) {
	if _, ok := r.requiredColumns[columnName]; !ok {
		return
	}

	if table, ok := r.tables[tableName]; !ok {
		// We do not retrospectively check.
		// So we assume it contains all required columns.
		r.initFullTable(tableName)
	} else {
		table[columnName] = true
	}
}

// drop a column
// return true if the column was successfully dropped from requirement list.
func (r *ColumnRequiredRule) dropColumn(tableName string, columnName string) bool {
	if _, ok := r.requiredColumns[columnName]; !ok {
		return false
	}
	table, ok := r.tables[tableName]
	if !ok {
		// We do not retrospectively check.
		// So we assume it contains all required columns.
		table = r.initFullTable(tableName)
	}
	table[columnName] = false
	return true
}

// rename a column
// return if the old column was dropped from requirement list.
func (r *ColumnRequiredRule) renameColumn(tableName string, oldColumn string, newColumn string) bool {
	_, oldNeed := r.requiredColumns[oldColumn]
	_, newNeed := r.requiredColumns[newColumn]
	if !oldNeed && !newNeed {
		return false
	}
	table, ok := r.tables[tableName]
	if !ok {
		// We do not retrospectively check.
		// So we assume it contains all required columns.
		table = r.initFullTable(tableName)
	}
	if oldNeed {
		table[oldColumn] = false
	}
	if newNeed {
		table[newColumn] = true
	}
	return oldNeed
}

func (r *ColumnRequiredRule) initFullTable(tableName string) columnSet {
	table := r.initEmptyTable(tableName)
	for column := range r.requiredColumns {
		table[column] = true
	}
	return table
}
