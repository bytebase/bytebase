package mysql

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/antlr4-go/antlr/v4"

	"github.com/bytebase/parser/mysql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*IndexNoDuplicateColumnAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_INDEX_NO_DUPLICATE_COLUMN, &IndexNoDuplicateColumnAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_INDEX_NO_DUPLICATE_COLUMN, &IndexNoDuplicateColumnAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_INDEX_NO_DUPLICATE_COLUMN, &IndexNoDuplicateColumnAdvisor{})
}

// IndexNoDuplicateColumnAdvisor is the advisor checking for no duplicate columns in index.
type IndexNoDuplicateColumnAdvisor struct {
}

// Check checks for no duplicate columns in index.
func (*IndexNoDuplicateColumnAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewIndexNoDuplicateColumnRule(level, checkCtx.Rule.Type.String())

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

	return checker.GetAdviceList(), nil
}

// IndexNoDuplicateColumnRule checks for no duplicate columns in index.
type IndexNoDuplicateColumnRule struct {
	BaseRule
}

// NewIndexNoDuplicateColumnRule creates a new IndexNoDuplicateColumnRule.
func NewIndexNoDuplicateColumnRule(level storepb.Advice_Status, title string) *IndexNoDuplicateColumnRule {
	return &IndexNoDuplicateColumnRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
	}
}

// Name returns the rule name.
func (*IndexNoDuplicateColumnRule) Name() string {
	return "IndexNoDuplicateColumnRule"
}

// OnEnter is called when entering a parse tree node.
func (r *IndexNoDuplicateColumnRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
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
func (*IndexNoDuplicateColumnRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *IndexNoDuplicateColumnRule) checkCreateTable(ctx *mysql.CreateTableContext) {
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
		if tableElement.TableConstraintDef() == nil {
			continue
		}
		r.handleConstraintDef(tableName, tableElement.TableConstraintDef())
	}
}

func (r *IndexNoDuplicateColumnRule) checkAlterTable(ctx *mysql.AlterTableContext) {
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
		// add index.
		case alterListItem.ADD_SYMBOL() != nil && alterListItem.TableConstraintDef() != nil:
			r.handleConstraintDef(tableName, alterListItem.TableConstraintDef())
		default:
			continue
		}
	}
}

func (r *IndexNoDuplicateColumnRule) checkCreateIndex(ctx *mysql.CreateIndexContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	switch ctx.GetType_().GetTokenType() {
	case mysql.MySQLParserFULLTEXT_SYMBOL, mysql.MySQLParserSPATIAL_SYMBOL:
		return
	default:
	}
	if ctx.CreateIndexTarget() == nil || ctx.CreateIndexTarget().TableRef() == nil || ctx.CreateIndexTarget().KeyListVariants() == nil {
		return
	}
	indexType := ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.NewInterval(
		ctx.GetStart().GetTokenIndex(),
		ctx.CreateIndexTarget().KeyListVariants().GetStart().GetTokenIndex()-1,
	))

	indexName := ""
	if ctx.IndexName() != nil {
		indexName = mysqlparser.NormalizeIndexName(ctx.IndexName())
		indexType = ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.NewInterval(
			ctx.GetStart().GetTokenIndex(),
			ctx.IndexName().GetStart().GetTokenIndex()-1,
		))
	}
	if ctx.IndexNameAndType() != nil && ctx.IndexNameAndType().IndexName() != nil {
		indexName = mysqlparser.NormalizeIndexName(ctx.IndexNameAndType().IndexName())
		indexType = ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.NewInterval(
			ctx.GetStart().GetTokenIndex(),
			ctx.IndexNameAndType().GetStart().GetTokenIndex()-1,
		))
	}

	_, tableName := mysqlparser.NormalizeMySQLTableRef(ctx.CreateIndexTarget().TableRef())
	columnList := mysqlparser.NormalizeKeyListVariants(ctx.CreateIndexTarget().KeyListVariants())
	if column, duplicate := r.hasDuplicateColumn(columnList); duplicate {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.DuplicateColumnInIndex.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("%s`%s` has duplicate column `%s`.`%s`", indexType, indexName, tableName, column),
			StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine()),
		})
	}
}

func (r *IndexNoDuplicateColumnRule) handleConstraintDef(tableName string, ctx mysql.ITableConstraintDefContext) {
	var columnList []string
	indexType := ""
	switch ctx.GetType_().GetTokenType() {
	case mysql.MySQLParserINDEX_SYMBOL, mysql.MySQLParserKEY_SYMBOL, mysql.MySQLParserPRIMARY_SYMBOL, mysql.MySQLParserUNIQUE_SYMBOL:
		if ctx.KeyListVariants() == nil {
			return
		}
		columnList = mysqlparser.NormalizeKeyListVariants(ctx.KeyListVariants())
		indexType = ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.NewInterval(
			ctx.GetStart().GetTokenIndex(),
			ctx.KeyListVariants().GetStart().GetTokenIndex()-1,
		))
	case mysql.MySQLParserFOREIGN_SYMBOL:
		if ctx.KeyList() == nil {
			return
		}
		columnList = mysqlparser.NormalizeKeyList(ctx.KeyList())
		indexType = ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.NewInterval(
			ctx.GetStart().GetTokenIndex(),
			ctx.KeyList().GetStart().GetTokenIndex()-1,
		))
	default:
		return
	}

	indexName := ""
	if ctx.IndexName() != nil {
		indexName = mysqlparser.NormalizeIndexName(ctx.IndexName())
		indexType = ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.NewInterval(
			ctx.GetStart().GetTokenIndex(),
			ctx.IndexName().GetStart().GetTokenIndex()-1,
		))
	}
	if ctx.IndexNameAndType() != nil && ctx.IndexNameAndType().IndexName() != nil {
		indexName = mysqlparser.NormalizeIndexName(ctx.IndexNameAndType().IndexName())
		indexType = ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.NewInterval(
			ctx.GetStart().GetTokenIndex(),
			ctx.IndexNameAndType().GetStart().GetTokenIndex()-1,
		))
	}
	if column, duplicate := r.hasDuplicateColumn(columnList); duplicate {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.DuplicateColumnInIndex.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("%s`%s` has duplicate column `%s`.`%s`", indexType, indexName, tableName, column),
			StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine()),
		})
	}
}

func (*IndexNoDuplicateColumnRule) hasDuplicateColumn(keyList []string) (string, bool) {
	listMap := make(map[string]struct{})
	for _, keyName := range keyList {
		if _, exists := listMap[keyName]; exists {
			return keyName, true
		}
		listMap[keyName] = struct{}{}
	}

	return "", false
}
