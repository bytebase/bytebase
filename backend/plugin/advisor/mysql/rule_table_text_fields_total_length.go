package mysql

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	"github.com/bytebase/parser/mysql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*TableMaximumVarcharLengthAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleTableTextFieldsTotalLength, &TableMaximumVarcharLengthAdvisor{})
}

type TableMaximumVarcharLengthAdvisor struct {
}

func (*TableMaximumVarcharLengthAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
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
	rule := NewTableTextFieldsTotalLengthRule(level, string(checkCtx.Rule.Type), payload.Number)

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range stmtList {
		rule.SetBaseLine(stmt.BaseLine)
		checker.SetBaseLine(stmt.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	return checker.GetAdviceList(), nil
}

// tableColumn tracks column information for text length calculation.
type tableColumn struct {
	name       string
	columnType string
}

// TableTextFieldsTotalLengthRule checks for table text fields total length.
type TableTextFieldsTotalLengthRule struct {
	BaseRule
	maximum int
	// tableColumns tracks columns for each table across all statements
	tableColumns map[string]map[string]*tableColumn // tableName -> columnName -> column
}

// NewTableTextFieldsTotalLengthRule creates a new TableTextFieldsTotalLengthRule.
func NewTableTextFieldsTotalLengthRule(level storepb.Advice_Status, title string, maximum int) *TableTextFieldsTotalLengthRule {
	return &TableTextFieldsTotalLengthRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		maximum:      maximum,
		tableColumns: make(map[string]map[string]*tableColumn),
	}
}

// Name returns the rule name.
func (*TableTextFieldsTotalLengthRule) Name() string {
	return "TableTextFieldsTotalLengthRule"
}

// OnEnter is called when entering a parse tree node.
func (r *TableTextFieldsTotalLengthRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeCreateTable:
		r.checkCreateTable(ctx.(*mysql.CreateTableContext))
	case NodeTypeAlterTable:
		r.checkAlterTable(ctx.(*mysql.AlterTableContext))
	default:
		// Ignore other node types
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*TableTextFieldsTotalLengthRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *TableTextFieldsTotalLengthRule) checkCreateTable(ctx *mysql.CreateTableContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.TableElementList() == nil || ctx.TableName() == nil {
		return
	}

	_, tableName := mysqlparser.NormalizeMySQLTableName(ctx.TableName())
	if tableName == "" {
		return
	}

	// Initialize table columns map
	if r.tableColumns[tableName] == nil {
		r.tableColumns[tableName] = make(map[string]*tableColumn)
	}

	// Extract columns from CREATE TABLE statement
	for _, tableElement := range ctx.TableElementList().AllTableElement() {
		if tableElement.ColumnDefinition() == nil {
			continue
		}
		columnDef := tableElement.ColumnDefinition()
		if columnDef.ColumnName() == nil || columnDef.FieldDefinition() == nil || columnDef.FieldDefinition().DataType() == nil {
			continue
		}

		_, _, columnName := mysqlparser.NormalizeMySQLColumnName(columnDef.ColumnName())
		columnType := mysqlparser.NormalizeMySQLDataType(columnDef.FieldDefinition().DataType(), true /* compact */)

		r.tableColumns[tableName][strings.ToLower(columnName)] = &tableColumn{
			name:       columnName,
			columnType: columnType,
		}
	}

	// Check total text length
	r.checkTableTextLength(tableName, ctx.GetStart().GetLine())
}

func (r *TableTextFieldsTotalLengthRule) checkAlterTable(ctx *mysql.AlterTableContext) {
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

	// Initialize table columns map if not exists
	if r.tableColumns[tableName] == nil {
		r.tableColumns[tableName] = make(map[string]*tableColumn)
	}

	// Process ALTER TABLE actions
	for _, item := range ctx.AlterTableActions().AlterCommandList().AlterList().AllAlterListItem() {
		if item == nil {
			continue
		}

		switch {
		// ADD COLUMN
		case item.ADD_SYMBOL() != nil:
			switch {
			case item.Identifier() != nil && item.FieldDefinition() != nil && item.FieldDefinition().DataType() != nil:
				columnName := mysqlparser.NormalizeMySQLIdentifier(item.Identifier())
				columnType := mysqlparser.NormalizeMySQLDataType(item.FieldDefinition().DataType(), true /* compact */)
				r.tableColumns[tableName][strings.ToLower(columnName)] = &tableColumn{
					name:       columnName,
					columnType: columnType,
				}
			case item.OPEN_PAR_SYMBOL() != nil && item.TableElementList() != nil:
				for _, tableElement := range item.TableElementList().AllTableElement() {
					if tableElement.ColumnDefinition() == nil {
						continue
					}
					columnDef := tableElement.ColumnDefinition()
					if columnDef.ColumnName() == nil || columnDef.FieldDefinition() == nil || columnDef.FieldDefinition().DataType() == nil {
						continue
					}
					_, _, columnName := mysqlparser.NormalizeMySQLColumnName(columnDef.ColumnName())
					columnType := mysqlparser.NormalizeMySQLDataType(columnDef.FieldDefinition().DataType(), true /* compact */)
					r.tableColumns[tableName][strings.ToLower(columnName)] = &tableColumn{
						name:       columnName,
						columnType: columnType,
					}
				}
			}
		// CHANGE COLUMN or MODIFY COLUMN
		case item.CHANGE_SYMBOL() != nil && item.Identifier() != nil && item.FieldDefinition() != nil && item.FieldDefinition().DataType() != nil:
			columnName := mysqlparser.NormalizeMySQLIdentifier(item.Identifier())
			columnType := mysqlparser.NormalizeMySQLDataType(item.FieldDefinition().DataType(), true /* compact */)
			r.tableColumns[tableName][strings.ToLower(columnName)] = &tableColumn{
				name:       columnName,
				columnType: columnType,
			}
		case item.MODIFY_SYMBOL() != nil && item.ColumnInternalRef() != nil && item.FieldDefinition() != nil && item.FieldDefinition().DataType() != nil:
			columnName := mysqlparser.NormalizeMySQLColumnInternalRef(item.ColumnInternalRef())
			columnType := mysqlparser.NormalizeMySQLDataType(item.FieldDefinition().DataType(), true /* compact */)
			r.tableColumns[tableName][strings.ToLower(columnName)] = &tableColumn{
				name:       columnName,
				columnType: columnType,
			}
		// DROP COLUMN
		case item.DROP_SYMBOL() != nil && item.ColumnInternalRef() != nil:
			columnName := mysqlparser.NormalizeMySQLColumnInternalRef(item.ColumnInternalRef())
			delete(r.tableColumns[tableName], strings.ToLower(columnName))
		}
	}

	// Check total text length after alterations
	r.checkTableTextLength(tableName, ctx.GetStart().GetLine())
}

// checkTableTextLength calculates and checks the total text field length for a table.
func (r *TableTextFieldsTotalLengthRule) checkTableTextLength(tableName string, line int) {
	columns, exists := r.tableColumns[tableName]
	if !exists {
		return
	}

	var total int64
	for _, column := range columns {
		total += getTextLength(column.columnType)
	}

	if total > int64(r.maximum) {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          advisor.TotalTextLengthExceedsLimit.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("Table %q total text column length (%d) exceeds the limit (%d).", tableName, total, r.maximum),
			StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + line),
		})
	}
}

func getTextLength(s string) int64 {
	s = strings.ToLower(s)
	switch s {
	case "char", "binary":
		return 1
	case "tinyblob", "tinytext":
		return 255
	case "blob", "text":
		return 65_535
	case "mediumblob", "mediumtext":
		return 16_777_215
	case "longblob", "longtext":
		return 4_294_967_295
	default:
		re := regexp.MustCompile(`[a-z]+\((\d+)\)`)
		match := re.FindStringSubmatch(s)
		if len(match) >= 2 {
			n, err := strconv.ParseInt(match[1], 10, 64)
			if err == nil {
				return int64(n)
			}
		}
	}
	return 0
}
