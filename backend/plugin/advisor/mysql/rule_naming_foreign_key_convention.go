package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/parser/mysql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*NamingFKConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleFKNaming, &NamingFKConventionAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.SchemaRuleFKNaming, &NamingFKConventionAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.SchemaRuleFKNaming, &NamingFKConventionAdvisor{})
}

// NamingFKConventionAdvisor is the advisor checking for foreign key naming convention.
type NamingFKConventionAdvisor struct {
}

// Check checks for foreign key naming convention.
func (*NamingFKConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	root, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	format, templateList, maxLength, err := advisor.UnmarshalNamingRulePayloadAsTemplate(advisor.SQLReviewRuleType(checkCtx.Rule.Type), checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewNamingFKConventionRule(level, string(checkCtx.Rule.Type), format, maxLength, templateList)

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmtNode := range root {
		rule.SetBaseLine(stmtNode.BaseLine)
		checker.SetBaseLine(stmtNode.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmtNode.Tree)
	}

	return checker.GetAdviceList(), nil
}

// indexMetaData is the metadata for foreign key.
type fkIndexMetaData struct {
	indexName string
	tableName string
	metaData  map[string]string
	line      int
}

// NamingFKConventionRule checks for foreign key naming convention.
type NamingFKConventionRule struct {
	BaseRule
	text         string
	format       string
	maxLength    int
	templateList []string
}

// NewNamingFKConventionRule creates a new NamingFKConventionRule.
func NewNamingFKConventionRule(level storepb.Advice_Status, title string, format string, maxLength int, templateList []string) *NamingFKConventionRule {
	return &NamingFKConventionRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		format:       format,
		maxLength:    maxLength,
		templateList: templateList,
	}
}

// Name returns the rule name.
func (*NamingFKConventionRule) Name() string {
	return "NamingFKConventionRule"
}

// OnEnter is called when entering a parse tree node.
func (r *NamingFKConventionRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeQuery:
		queryCtx, ok := ctx.(*mysql.QueryContext)
		if !ok {
			return nil
		}
		r.text = queryCtx.GetParser().GetTokenStream().GetTextFromRuleContext(queryCtx)
	case NodeTypeCreateTable:
		r.checkCreateTable(ctx.(*mysql.CreateTableContext))
	case NodeTypeAlterTable:
		r.checkAlterTable(ctx.(*mysql.AlterTableContext))
	default:
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*NamingFKConventionRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *NamingFKConventionRule) checkCreateTable(ctx *mysql.CreateTableContext) {
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

	var indexDataList []*fkIndexMetaData
	for _, tableElement := range ctx.TableElementList().AllTableElement() {
		if tableElement == nil {
			continue
		}
		if tableElement.TableConstraintDef() == nil {
			continue
		}
		if metaData := r.handleConstraintDef(tableName, tableElement.TableConstraintDef()); metaData != nil {
			indexDataList = append(indexDataList, metaData)
		}
	}
	r.handleIndexList(indexDataList)
}

func (r *NamingFKConventionRule) checkAlterTable(ctx *mysql.AlterTableContext) {
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
	var indexDataList []*fkIndexMetaData
	for _, alterListItem := range ctx.AlterTableActions().AlterCommandList().AlterList().AllAlterListItem() {
		if alterListItem == nil {
			continue
		}

		// add constriant.
		if alterListItem.ADD_SYMBOL() != nil && alterListItem.TableConstraintDef() != nil {
			if metaData := r.handleConstraintDef(tableName, alterListItem.TableConstraintDef()); metaData != nil {
				indexDataList = append(indexDataList, metaData)
			}
		}
	}
	r.handleIndexList(indexDataList)
}

func (r *NamingFKConventionRule) handleIndexList(indexDataList []*fkIndexMetaData) {
	for _, indexData := range indexDataList {
		regex, err := getTemplateRegexp(r.format, r.templateList, indexData.metaData)
		if err != nil {
			r.AddAdvice(&storepb.Advice{
				Status:  r.level,
				Code:    code.Internal.Int32(),
				Title:   "Internal error for foreign key naming convention rule",
				Content: fmt.Sprintf("%q meet internal error %q", r.text, err.Error()),
			})
			continue
		}
		if !regex.MatchString(indexData.indexName) {
			r.AddAdvice(&storepb.Advice{
				Status:        r.level,
				Code:          code.NamingFKConventionMismatch.Int32(),
				Title:         r.title,
				Content:       fmt.Sprintf("Foreign key in table `%s` mismatches the naming convention, expect %q but found `%s`", indexData.tableName, regex, indexData.indexName),
				StartPosition: common.ConvertANTLRLineToPosition(indexData.line),
			})
		}
		if r.maxLength > 0 && len(indexData.indexName) > r.maxLength {
			r.AddAdvice(&storepb.Advice{
				Status:        r.level,
				Code:          code.NamingFKConventionMismatch.Int32(),
				Title:         r.title,
				Content:       fmt.Sprintf("Foreign key `%s` in table `%s` mismatches the naming convention, its length should be within %d characters", indexData.indexName, indexData.tableName, r.maxLength),
				StartPosition: common.ConvertANTLRLineToPosition(indexData.line),
			})
		}
	}
}

func (r *NamingFKConventionRule) handleConstraintDef(tableName string, ctx mysql.ITableConstraintDefContext) *fkIndexMetaData {
	// focus on foreign index.
	if ctx.FOREIGN_SYMBOL() == nil || ctx.KEY_SYMBOL() == nil || ctx.KeyList() == nil || ctx.References() == nil {
		return nil
	}

	indexName := ""
	// for compatibility.
	if ctx.IndexName() != nil {
		indexName = mysqlparser.NormalizeIndexName(ctx.IndexName())
	}
	// use constraint_name if both exist at the same time.
	// for mysql, foreign key use constraint name as unique identifier.
	if ctx.ConstraintName() != nil {
		indexName = mysqlparser.NormalizeConstraintName(ctx.ConstraintName())
	}

	referencingColumnList := mysqlparser.NormalizeKeyList(ctx.KeyList())
	referencedTable, referencedColumnList := r.handleReferences(ctx.References())
	metaData := map[string]string{
		advisor.ReferencingTableNameTemplateToken:  tableName,
		advisor.ReferencingColumnNameTemplateToken: strings.Join(referencingColumnList, "_"),
		advisor.ReferencedTableNameTemplateToken:   referencedTable,
		advisor.ReferencedColumnNameTemplateToken:  strings.Join(referencedColumnList, "_"),
	}
	return &fkIndexMetaData{
		indexName: indexName,
		tableName: tableName,
		metaData:  metaData,
		line:      r.baseLine + ctx.GetStart().GetLine(),
	}
}

func (*NamingFKConventionRule) handleReferences(ctx mysql.IReferencesContext) (string, []string) {
	tableName := ""
	if ctx.TableRef() != nil {
		_, tableName = mysqlparser.NormalizeMySQLTableRef(ctx.TableRef())
	}

	var columns []string
	if ctx.IdentifierListWithParentheses() != nil {
		columns = mysqlparser.NormalizeIdentifierListWithParentheses(ctx.IdentifierListWithParentheses())
	}
	return tableName, columns
}
