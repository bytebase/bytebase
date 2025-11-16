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
	_ advisor.Advisor = (*IndexTypeNoBlobAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleIndexTypeNoBlob, &IndexTypeNoBlobAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.SchemaRuleIndexTypeNoBlob, &IndexTypeNoBlobAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.SchemaRuleIndexTypeNoBlob, &IndexTypeNoBlobAdvisor{})
}

// IndexTypeNoBlobAdvisor is the advisor checking for index type no blob.
type IndexTypeNoBlobAdvisor struct {
}

// Check checks for index type no blob.
func (*IndexTypeNoBlobAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewIndexTypeNoBlobRule(level, string(checkCtx.Rule.Type), checkCtx.OriginCatalog)

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range stmtList {
		rule.SetBaseLine(stmt.BaseLine)
		checker.SetBaseLine(stmt.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	return checker.GetAdviceList(), nil
}

// IndexTypeNoBlobRule checks for index type no blob.
type IndexTypeNoBlobRule struct {
	BaseRule
	originCatalog    *catalog.DatabaseState
	tablesNewColumns tableColumnTypes
}

// NewIndexTypeNoBlobRule creates a new IndexTypeNoBlobRule.
func NewIndexTypeNoBlobRule(level storepb.Advice_Status, title string, originCatalog *catalog.DatabaseState) *IndexTypeNoBlobRule {
	return &IndexTypeNoBlobRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		originCatalog:    originCatalog,
		tablesNewColumns: make(tableColumnTypes),
	}
}

// Name returns the rule name.
func (*IndexTypeNoBlobRule) Name() string {
	return "IndexTypeNoBlobRule"
}

// OnEnter is called when entering a parse tree node.
func (r *IndexTypeNoBlobRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeCreateTable:
		r.checkCreateTable(ctx.(*mysql.CreateTableContext))
	case NodeTypeAlterTable:
		r.checkAlterTable(ctx.(*mysql.AlterTableContext))
	case NodeTypeCreateIndex:
		r.checkCreateIndex(ctx.(*mysql.CreateIndexContext))
	default:
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*IndexTypeNoBlobRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *IndexTypeNoBlobRule) checkCreateTable(ctx *mysql.CreateTableContext) {
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

func (r *IndexTypeNoBlobRule) checkAlterTable(ctx *mysql.AlterTableContext) {
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
		case alterListItem.ADD_SYMBOL() != nil:
			switch {
			// add column.
			case alterListItem.Identifier() != nil && alterListItem.FieldDefinition() != nil:
				columnName := mysqlparser.NormalizeMySQLIdentifier(alterListItem.Identifier())
				r.checkFieldDefinition(tableName, columnName, alterListItem.FieldDefinition())
			// add multi column.
			case alterListItem.OPEN_PAR_SYMBOL() != nil && alterListItem.TableElementList() != nil:
				for _, tableElement := range alterListItem.TableElementList().AllTableElement() {
					if tableElement.ColumnDefinition() == nil || tableElement.ColumnDefinition().ColumnName() == nil || tableElement.ColumnDefinition().FieldDefinition() == nil {
						continue
					}
					_, _, columnName := mysqlparser.NormalizeMySQLColumnName(tableElement.ColumnDefinition().ColumnName())
					r.checkFieldDefinition(tableName, columnName, tableElement.ColumnDefinition().FieldDefinition())
				}
			// add constraint.
			case alterListItem.TableConstraintDef() != nil:
				r.checkConstraintDef(tableName, alterListItem.TableConstraintDef())
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
		default:
		}
	}
}

func (r *IndexTypeNoBlobRule) checkCreateIndex(ctx *mysql.CreateIndexContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.GetType_() == nil {
		return
	}
	switch ctx.GetType_().GetTokenType() {
	case mysql.MySQLParserFULLTEXT_SYMBOL, mysql.MySQLParserSPATIAL_SYMBOL, mysql.MySQLParserFOREIGN_SYMBOL:
		return
	default:
	}
	if ctx.CreateIndexTarget() == nil || ctx.CreateIndexTarget().TableRef() == nil || ctx.CreateIndexTarget().KeyListVariants() == nil {
		return
	}

	_, tableName := mysqlparser.NormalizeMySQLTableRef(ctx.CreateIndexTarget().TableRef())
	columnList := mysqlparser.NormalizeKeyListVariants(ctx.CreateIndexTarget().KeyListVariants())
	for _, columnName := range columnList {
		columnType, err := r.getColumnType(tableName, columnName)
		if err != nil {
			continue
		}
		columnType = strings.ToLower(columnType)
		r.addAdvice(tableName, columnName, columnType, ctx.GetStart().GetLine())
	}
}

func (r *IndexTypeNoBlobRule) checkFieldDefinition(tableName, columnName string, ctx mysql.IFieldDefinitionContext) {
	if ctx.DataType() == nil {
		return
	}
	columnType := mysqlparser.NormalizeMySQLDataType(ctx.DataType(), true /* compact */)
	for _, attribute := range ctx.AllColumnAttribute() {
		if attribute == nil || attribute.GetValue() == nil {
			continue
		}
		// the FieldDefinitionContext can only set primary or unique.
		switch attribute.GetValue().GetTokenType() {
		case mysql.MySQLParserPRIMARY_SYMBOL, mysql.MySQLParserUNIQUE_SYMBOL:
			// do nothing
		default:
			continue
		}
		r.addAdvice(tableName, columnName, columnType, ctx.GetStart().GetLine())
	}
	r.tablesNewColumns.set(tableName, columnName, columnType)
}

func (r *IndexTypeNoBlobRule) checkConstraintDef(tableName string, ctx mysql.ITableConstraintDefContext) {
	if ctx.GetType_() == nil {
		return
	}
	var columnList []string
	switch ctx.GetType_().GetTokenType() {
	case mysql.MySQLParserINDEX_SYMBOL, mysql.MySQLParserKEY_SYMBOL, mysql.MySQLParserPRIMARY_SYMBOL, mysql.MySQLParserUNIQUE_SYMBOL:
		if ctx.KeyListVariants() == nil {
			return
		}
		columnList = mysqlparser.NormalizeKeyListVariants(ctx.KeyListVariants())
	case mysql.MySQLParserFOREIGN_SYMBOL:
		if ctx.KeyList() == nil {
			return
		}
		columnList = mysqlparser.NormalizeKeyList(ctx.KeyList())
	default:
		return
	}

	for _, columnName := range columnList {
		columnType, err := r.getColumnType(tableName, columnName)
		if err != nil {
			continue
		}
		columnType = strings.ToLower(columnType)
		r.addAdvice(tableName, columnName, columnType, ctx.GetStart().GetLine())
	}
}

func (r *IndexTypeNoBlobRule) addAdvice(tableName, columnName, columnType string, lineNumber int) {
	if r.isBlob(columnType) {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.IndexTypeNoBlob.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("Columns in index must not be BLOB but `%s`.`%s` is %s", tableName, columnName, columnType),
			StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + lineNumber),
		})
	}
}
func (*IndexTypeNoBlobRule) isBlob(columnType string) bool {
	switch strings.ToLower(columnType) {
	case "blob", "tinyblob", "mediumblob", "longblob":
		return true
	default:
		return false
	}
}

// getColumnType gets the column type string from r.tableColumnTypes or catalog, returns empty string and non-nil error if cannot find the column in given table.
func (r *IndexTypeNoBlobRule) getColumnType(tableName string, columnName string) (string, error) {
	if columnType, ok := r.tablesNewColumns.get(tableName, columnName); ok {
		return columnType, nil
	}
	column := r.originCatalog.GetColumn("", tableName, columnName)
	if column != nil {
		return column.Type(), nil
	}
	return "", errors.Errorf("cannot find the type of `%s`.`%s`", tableName, columnName)
}
