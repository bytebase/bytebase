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
	"github.com/bytebase/bytebase/backend/plugin/advisor/catalog"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*NamingIndexConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleIDXNaming, &NamingIndexConventionAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.SchemaRuleIDXNaming, &NamingIndexConventionAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.SchemaRuleIDXNaming, &NamingIndexConventionAdvisor{})
}

// indexMetaData is the metadata for index.
type indexMetaData struct {
	indexName string
	tableName string
	metaData  map[string]string
	line      int
}

// NamingIndexConventionAdvisor is the advisor checking for index naming convention.
type NamingIndexConventionAdvisor struct {
}

// Check checks for index naming convention.
func (*NamingIndexConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	root, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parser result")
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
	rule := NewNamingIndexConventionRule(level, string(checkCtx.Rule.Type), format, maxLength, templateList, checkCtx.OriginCatalog)

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmtNode := range root {
		rule.SetBaseLine(stmtNode.BaseLine)
		checker.SetBaseLine(stmtNode.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmtNode.Tree)
	}

	return checker.GetAdviceList(), nil
}

// NamingIndexConventionRule checks for index naming convention.
type NamingIndexConventionRule struct {
	BaseRule
	text          string
	format        string
	maxLength     int
	templateList  []string
	originCatalog *catalog.DatabaseState
}

// NewNamingIndexConventionRule creates a new NamingIndexConventionRule.
func NewNamingIndexConventionRule(level storepb.Advice_Status, title string, format string, maxLength int, templateList []string, originCatalog *catalog.DatabaseState) *NamingIndexConventionRule {
	return &NamingIndexConventionRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		format:        format,
		maxLength:     maxLength,
		templateList:  templateList,
		originCatalog: originCatalog,
	}
}

// Name returns the rule name.
func (*NamingIndexConventionRule) Name() string {
	return "NamingIndexConventionRule"
}

// OnEnter is called when entering a parse tree node.
func (r *NamingIndexConventionRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
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
	case NodeTypeCreateIndex:
		r.checkCreateIndex(ctx.(*mysql.CreateIndexContext))
	default:
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*NamingIndexConventionRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *NamingIndexConventionRule) checkCreateTable(ctx *mysql.CreateTableContext) {
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

	var indexDataList []*indexMetaData
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

func (r *NamingIndexConventionRule) checkAlterTable(ctx *mysql.AlterTableContext) {
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
	var indexDataList []*indexMetaData
	for _, alterListItem := range ctx.AlterTableActions().AlterCommandList().AlterList().AllAlterListItem() {
		if alterListItem == nil {
			continue
		}

		switch {
		// add index.
		case alterListItem.ADD_SYMBOL() != nil && alterListItem.TableConstraintDef() != nil:
			if metaData := r.handleConstraintDef(tableName, alterListItem.TableConstraintDef()); metaData != nil {
				indexDataList = append(indexDataList, metaData)
			}
		// rename index.
		case alterListItem.RENAME_SYMBOL() != nil && alterListItem.KeyOrIndex() != nil && alterListItem.IndexRef() != nil && alterListItem.IndexName() != nil:
			_, _, oldIndexName := mysqlparser.NormalizeIndexRef(alterListItem.IndexRef())
			newIndexName := mysqlparser.NormalizeIndexName(alterListItem.IndexName())
			_, indexState := r.originCatalog.GetIndex("", tableName, oldIndexName)
			if indexState == nil {
				continue
			}
			if indexState.Unique() {
				// Unique index naming convention should in advisor_naming_unique_key_convention.go
				continue
			}
			metaData := map[string]string{
				advisor.ColumnListTemplateToken: strings.Join(indexState.ExpressionList(), "_"),
				advisor.TableNameTemplateToken:  tableName,
			}
			indexData := &indexMetaData{
				indexName: newIndexName,
				tableName: tableName,
				metaData:  metaData,
				line:      r.baseLine + ctx.GetStart().GetLine(),
			}
			indexDataList = append(indexDataList, indexData)
		default:
		}
	}
	r.handleIndexList(indexDataList)
}

func (r *NamingIndexConventionRule) checkCreateIndex(ctx *mysql.CreateIndexContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	// Unique index naming convention should in advisor_naming_unique_key_convention.go
	if ctx.UNIQUE_SYMBOL() != nil || ctx.FULLTEXT_SYMBOL() != nil || ctx.SPATIAL_SYMBOL() != nil {
		return
	}
	if ctx.CreateIndexTarget() == nil || ctx.CreateIndexTarget().TableRef() == nil {
		return
	}

	indexName := ""
	if ctx.IndexName() != nil {
		indexName = mysqlparser.NormalizeIndexName(ctx.IndexName())
	}
	if ctx.IndexNameAndType() != nil && ctx.IndexNameAndType().IndexName() != nil {
		indexName = mysqlparser.NormalizeIndexName(ctx.IndexNameAndType().IndexName())
	}

	_, tableName := mysqlparser.NormalizeMySQLTableRef(ctx.CreateIndexTarget().TableRef())

	if ctx.CreateIndexTarget().KeyListVariants() == nil {
		return
	}
	columnList := mysqlparser.NormalizeKeyListVariants(ctx.CreateIndexTarget().KeyListVariants())
	metaData := map[string]string{
		advisor.ColumnListTemplateToken: strings.Join(columnList, "_"),
		advisor.TableNameTemplateToken:  tableName,
	}
	indexDataList := []*indexMetaData{
		{
			indexName: indexName,
			tableName: tableName,
			metaData:  metaData,
			line:      r.baseLine + ctx.GetStart().GetLine(),
		},
	}
	r.handleIndexList(indexDataList)
}

func (r *NamingIndexConventionRule) handleIndexList(indexDataList []*indexMetaData) {
	for _, indexData := range indexDataList {
		regex, err := getTemplateRegexp(r.format, r.templateList, indexData.metaData)
		if err != nil {
			r.AddAdvice(&storepb.Advice{
				Status:  r.level,
				Code:    code.Internal.Int32(),
				Title:   "Internal error for unique key naming convention rule",
				Content: fmt.Sprintf("%q meet internal error %q", r.text, err.Error()),
			})
			continue
		}
		if !regex.MatchString(indexData.indexName) {
			r.AddAdvice(&storepb.Advice{
				Status:        r.level,
				Code:          code.NamingIndexConventionMismatch.Int32(),
				Title:         r.title,
				Content:       fmt.Sprintf("Index in table `%s` mismatches the naming convention, expect %q but found `%s`", indexData.tableName, regex, indexData.indexName),
				StartPosition: common.ConvertANTLRLineToPosition(indexData.line),
			})
		}
		if r.maxLength > 0 && len(indexData.indexName) > r.maxLength {
			r.AddAdvice(&storepb.Advice{
				Status:        r.level,
				Code:          code.NamingIndexConventionMismatch.Int32(),
				Title:         r.title,
				Content:       fmt.Sprintf("Index `%s` in table `%s` mismatches the naming convention, its length should be within %d characters", indexData.indexName, indexData.tableName, r.maxLength),
				StartPosition: common.ConvertANTLRLineToPosition(indexData.line),
			})
		}
	}
}

func (r *NamingIndexConventionRule) handleConstraintDef(tableName string, ctx mysql.ITableConstraintDefContext) *indexMetaData {
	// we only focus normal index.
	if ctx.UNIQUE_SYMBOL() != nil || ctx.FULLTEXT_SYMBOL() != nil || ctx.SPATIAL_SYMBOL() != nil || ctx.PRIMARY_SYMBOL() != nil {
		return nil
	}
	if ctx.KeyListVariants() == nil {
		return nil
	}

	indexName := ""
	if ctx.IndexNameAndType() != nil && ctx.IndexNameAndType().IndexName() != nil {
		indexName = mysqlparser.NormalizeIndexName(ctx.IndexNameAndType().IndexName())
	}

	columnList := mysqlparser.NormalizeKeyListVariants(ctx.KeyListVariants())
	metaData := map[string]string{
		advisor.ColumnListTemplateToken: strings.Join(columnList, "_"),
		advisor.TableNameTemplateToken:  tableName,
	}
	return &indexMetaData{
		indexName: indexName,
		tableName: tableName,
		metaData:  metaData,
		line:      r.baseLine + ctx.GetStart().GetLine(),
	}
}
