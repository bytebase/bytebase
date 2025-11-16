package mysql

import (
	"context"
	"fmt"
	"slices"

	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/parser/mysql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/catalog"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*ColumnNoNullAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleColumnNotNull, &ColumnNoNullAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.SchemaRuleColumnNotNull, &ColumnNoNullAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.SchemaRuleColumnNotNull, &ColumnNoNullAdvisor{})
}

// ColumnNoNullAdvisor is the advisor checking for column no NULL value.
type ColumnNoNullAdvisor struct {
}

// Check checks for column no NULL value.
func (*ColumnNoNullAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	root, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parser result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewColumnNoNullRule(level, string(checkCtx.Rule.Type), checkCtx.FinalCatalog)

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmtNode := range root {
		rule.SetBaseLine(stmtNode.BaseLine)
		checker.SetBaseLine(stmtNode.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmtNode.Tree)
	}

	// Generate advice after walking
	rule.generateAdvice()

	return checker.GetAdviceList(), nil
}

type columnName struct {
	tableName  string
	columnName string
	line       int
}

func (c columnName) name() string {
	return fmt.Sprintf("%s.%s", c.tableName, c.columnName)
}

// ColumnNoNullRule checks for column no NULL value.
type ColumnNoNullRule struct {
	BaseRule
	columnSet    map[string]columnName
	finalCatalog *catalog.DatabaseState
}

// NewColumnNoNullRule creates a new ColumnNoNullRule.
func NewColumnNoNullRule(level storepb.Advice_Status, title string, finalCatalog *catalog.DatabaseState) *ColumnNoNullRule {
	return &ColumnNoNullRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		columnSet:    make(map[string]columnName),
		finalCatalog: finalCatalog,
	}
}

// Name returns the rule name.
func (*ColumnNoNullRule) Name() string {
	return "ColumnNoNullRule"
}

// OnEnter is called when entering a parse tree node.
func (r *ColumnNoNullRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
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
func (*ColumnNoNullRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	// This rule doesn't need exit processing
	return nil
}

func (r *ColumnNoNullRule) generateAdvice() {
	var columnList []columnName
	for _, column := range r.columnSet {
		columnList = append(columnList, column)
	}
	slices.SortFunc(columnList, func(a, b columnName) int {
		if a.line != b.line {
			if a.line < b.line {
				return -1
			}
			return 1
		}
		if a.columnName < b.columnName {
			return -1
		}
		if a.columnName > b.columnName {
			return 1
		}
		return 0
	})

	for _, column := range columnList {
		col := r.finalCatalog.GetColumn("", column.tableName, column.columnName)
		if col != nil && col.Nullable() {
			r.AddAdvice(&storepb.Advice{
				Status:        r.level,
				Code:          code.ColumnCannotNull.Int32(),
				Title:         r.title,
				Content:       fmt.Sprintf("`%s`.`%s` cannot have NULL value", column.tableName, column.columnName),
				StartPosition: common.ConvertANTLRLineToPosition(column.line),
			})
		}
	}
}

func (r *ColumnNoNullRule) checkCreateTable(ctx *mysql.CreateTableContext) {
	if ctx.TableName() == nil {
		return
	}
	if ctx.TableElementList() == nil {
		return
	}

	_, tableName := mysqlparser.NormalizeMySQLTableName(ctx.TableName())
	for _, tableElement := range ctx.TableElementList().AllTableElement() {
		if tableElement == nil {
			continue
		}
		if tableElement.ColumnDefinition() == nil {
			continue
		}

		_, _, column := mysqlparser.NormalizeMySQLColumnName(tableElement.ColumnDefinition().ColumnName())
		if tableElement.ColumnDefinition().FieldDefinition() == nil {
			continue
		}
		col := columnName{
			tableName:  tableName,
			columnName: column,
			line:       r.baseLine + tableElement.ColumnDefinition().GetStart().GetLine(),
		}
		if _, exists := r.columnSet[col.name()]; !exists {
			r.columnSet[col.name()] = col
		}
	}
}

func (r *ColumnNoNullRule) checkAlterTable(ctx *mysql.AlterTableContext) {
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

		var columns []string
		switch {
		// add column
		case item.ADD_SYMBOL() != nil:
			switch {
			case item.Identifier() != nil && item.FieldDefinition() != nil:
				column := mysqlparser.NormalizeMySQLIdentifier(item.Identifier())
				columns = append(columns, column)
			case item.OPEN_PAR_SYMBOL() != nil && item.TableElementList() != nil:
				for _, tableElement := range item.TableElementList().AllTableElement() {
					if tableElement.ColumnDefinition() == nil {
						continue
					}
					_, _, column := mysqlparser.NormalizeMySQLColumnName(tableElement.ColumnDefinition().ColumnName())
					columns = append(columns, column)
				}
			default:
			}
		// change column
		case item.CHANGE_SYMBOL() != nil && item.ColumnInternalRef() != nil && item.Identifier() != nil:
			// only care new column name.
			column := mysqlparser.NormalizeMySQLIdentifier(item.Identifier())
			columns = append(columns, column)
		default:
		}

		for _, column := range columns {
			col := columnName{
				tableName:  tableName,
				columnName: column,
				line:       r.baseLine + item.GetStart().GetLine(),
			}
			if _, exists := r.columnSet[col.name()]; !exists {
				r.columnSet[col.name()] = col
			}
		}
	}
}
