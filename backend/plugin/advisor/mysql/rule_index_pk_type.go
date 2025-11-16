package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

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
	_ advisor.Advisor = (*IndexPkTypeAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleIndexPKTypeLimit, &IndexPkTypeAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.SchemaRuleIndexPKTypeLimit, &IndexPkTypeAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.SchemaRuleIndexPKTypeLimit, &IndexPkTypeAdvisor{})
}

// IndexPkTypeAdvisor is the advisor checking for correct type of PK.
type IndexPkTypeAdvisor struct {
}

// Check checks for correct type of PK.
func (*IndexPkTypeAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewIndexPkTypeRule(level, string(checkCtx.Rule.Type), checkCtx.OriginCatalog)

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range stmtList {
		rule.SetBaseLine(stmt.BaseLine)
		checker.SetBaseLine(stmt.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	return checker.GetAdviceList(), nil
}

// IndexPkTypeRule checks for correct type of PK.
type IndexPkTypeRule struct {
	BaseRule
	line             map[string]int
	originCatalog    *catalog.DatabaseState
	tablesNewColumns tableColumnTypes
}

// NewIndexPkTypeRule creates a new IndexPkTypeRule.
func NewIndexPkTypeRule(level storepb.Advice_Status, title string, originCatalog *catalog.DatabaseState) *IndexPkTypeRule {
	return &IndexPkTypeRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		line:             make(map[string]int),
		originCatalog:    originCatalog,
		tablesNewColumns: make(tableColumnTypes),
	}
}

// Name returns the rule name.
func (*IndexPkTypeRule) Name() string {
	return "IndexPkTypeRule"
}

// OnEnter is called when entering a parse tree node.
func (r *IndexPkTypeRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
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
func (*IndexPkTypeRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *IndexPkTypeRule) checkCreateTable(ctx *mysql.CreateTableContext) {
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
		if tableElement == nil {
			continue
		}
		switch {
		case tableElement.ColumnDefinition() != nil:
			if tableElement.ColumnDefinition().FieldDefinition() == nil {
				continue
			}
			_, _, columnName := mysqlparser.NormalizeMySQLColumnName(tableElement.ColumnDefinition().ColumnName())
			r.checkFieldDefinition(tableName, columnName, tableElement.ColumnDefinition().FieldDefinition())
		case tableElement.TableConstraintDef() != nil:
			r.checkConstraintDef(tableName, tableElement.TableConstraintDef())
		default:
		}
	}
}

func (r *IndexPkTypeRule) checkAlterTable(ctx *mysql.AlterTableContext) {
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
	if ctx.TableRef() == nil {
		return
	}

	_, tableName := mysqlparser.NormalizeMySQLTableRef(ctx.TableRef())
	for _, alterListItem := range ctx.AlterTableActions().AlterCommandList().AlterList().AllAlterListItem() {
		if alterListItem == nil {
			continue
		}

		switch {
		// add column
		case alterListItem.ADD_SYMBOL() != nil && alterListItem.Identifier() != nil:
			switch {
			case alterListItem.Identifier() != nil && alterListItem.FieldDefinition() != nil:
				columnName := mysqlparser.NormalizeMySQLIdentifier(alterListItem.Identifier())
				r.checkFieldDefinition(tableName, columnName, alterListItem.FieldDefinition())
			case alterListItem.OPEN_PAR_SYMBOL() != nil && alterListItem.TableElementList() != nil:
				for _, tableElement := range alterListItem.TableElementList().AllTableElement() {
					if tableElement.ColumnDefinition() == nil || tableElement.ColumnDefinition().ColumnName() == nil || tableElement.ColumnDefinition().FieldDefinition() == nil {
						continue
					}
					_, _, columnName := mysqlparser.NormalizeMySQLColumnName(tableElement.ColumnDefinition().ColumnName())
					r.checkFieldDefinition(tableName, columnName, tableElement.ColumnDefinition().FieldDefinition())
				}
			default:
			}
		// modify column
		case alterListItem.MODIFY_SYMBOL() != nil && alterListItem.ColumnInternalRef() != nil:
			columnName := mysqlparser.NormalizeMySQLColumnInternalRef(alterListItem.ColumnInternalRef())
			r.checkFieldDefinition(tableName, columnName, alterListItem.FieldDefinition())
		// change column
		case alterListItem.CHANGE_SYMBOL() != nil && alterListItem.ColumnInternalRef() != nil && alterListItem.Identifier() != nil:
			oldColumnName := mysqlparser.NormalizeMySQLColumnInternalRef(alterListItem.ColumnInternalRef())
			r.tablesNewColumns.delete(tableName, oldColumnName)
			newColumnName := mysqlparser.NormalizeMySQLIdentifier(alterListItem.Identifier())
			r.checkFieldDefinition(tableName, newColumnName, alterListItem.FieldDefinition())
		// add constriant.
		case alterListItem.ADD_SYMBOL() != nil && alterListItem.TableConstraintDef() != nil:
			r.checkConstraintDef(tableName, alterListItem.TableConstraintDef())
		default:
		}
	}
}

func (r *IndexPkTypeRule) checkFieldDefinition(tableName, columnName string, ctx mysql.IFieldDefinitionContext) {
	if ctx.DataType() == nil {
		return
	}
	columnType := r.getIntOrBigIntStr(ctx.DataType())
	for _, attribute := range ctx.AllColumnAttribute() {
		if attribute.PRIMARY_SYMBOL() != nil {
			r.addAdvice(tableName, columnName, columnType, r.baseLine+ctx.GetStart().GetLine())
		}
	}
	r.tablesNewColumns.set(tableName, columnName, columnType)
}

func (r *IndexPkTypeRule) checkConstraintDef(tableName string, ctx mysql.ITableConstraintDefContext) {
	if ctx.GetType_().GetTokenType() != mysql.MySQLParserPRIMARY_SYMBOL {
		return
	}
	if ctx.KeyListVariants() == nil {
		return
	}
	columnList := mysqlparser.NormalizeKeyListVariants(ctx.KeyListVariants())

	for _, columnName := range columnList {
		columnType, err := r.getPKColumnType(tableName, columnName)
		if err != nil {
			continue
		}
		r.addAdvice(tableName, columnName, columnType, r.baseLine+ctx.GetStart().GetLine())
	}
}

func (r *IndexPkTypeRule) addAdvice(tableName, columnName, columnType string, lineNumber int) {
	if !strings.EqualFold(columnType, "INT") && !strings.EqualFold(columnType, "BIGINT") {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.IndexPKType.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("Columns in primary key must be INT/BIGINT but `%s`.`%s` is %s", tableName, columnName, columnType),
			StartPosition: common.ConvertANTLRLineToPosition(lineNumber),
		})
	}
}

// getPKColumnType gets the column type string from r.tablesNewColumns or catalog, returns empty string and non-nil error if cannot find the column in given table.
func (r *IndexPkTypeRule) getPKColumnType(tableName string, columnName string) (string, error) {
	if columnType, ok := r.tablesNewColumns.get(tableName, columnName); ok {
		return columnType, nil
	}
	column := r.originCatalog.GetColumn("", tableName, columnName)
	if column != nil {
		return column.Type(), nil
	}
	return "", errors.Errorf("cannot find the type of `%s`.`%s`", tableName, columnName)
}

// getIntOrBigIntStr returns the type string of tp.
func (*IndexPkTypeRule) getIntOrBigIntStr(ctx mysql.IDataTypeContext) string {
	switch ctx.GetType_().GetTokenType() {
	// https://pkg.go.dev/github.com/pingcap/tidb/pkg/parser/mysql#TypeLong
	case mysql.MySQLParserINT_SYMBOL:
		// tp.String() return int(11)
		return "INT"
		// https://pkg.go.dev/github.com/pingcap/tidb/pkg/parser/mysql#TypeLonglong
	case mysql.MySQLParserBIGINT_SYMBOL:
		// tp.String() return bigint(20)
		return "BIGINT"
	default:
	}
	return strings.ToLower(ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx))
}
