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
	_ advisor.Advisor = (*NamingUKConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleUKNaming, &NamingUKConventionAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.SchemaRuleUKNaming, &NamingUKConventionAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.SchemaRuleUKNaming, &NamingUKConventionAdvisor{})
}

// NamingUKConventionAdvisor is the advisor checking for unique key naming convention.
type NamingUKConventionAdvisor struct {
}

// Check checks for index naming convention.
func (*NamingUKConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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
	rule := NewNamingUKConventionRule(level, string(checkCtx.Rule.Type), format, maxLength, templateList, checkCtx.OriginCatalog)

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmtNode := range root {
		rule.SetBaseLine(stmtNode.BaseLine)
		checker.SetBaseLine(stmtNode.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmtNode.Tree)
	}

	return checker.GetAdviceList(), nil
}

// ukIndexMetaData is the metadata for unique key.
type ukIndexMetaData struct {
	indexName string
	tableName string
	metaData  map[string]string
	line      int
}

// NamingUKConventionRule checks for unique key naming convention.
type NamingUKConventionRule struct {
	BaseRule
	text          string
	format        string
	maxLength     int
	templateList  []string
	originCatalog *catalog.DatabaseState
}

// NewNamingUKConventionRule creates a new NamingUKConventionRule.
func NewNamingUKConventionRule(level storepb.Advice_Status, title string, format string, maxLength int, templateList []string, originCatalog *catalog.DatabaseState) *NamingUKConventionRule {
	return &NamingUKConventionRule{
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
func (*NamingUKConventionRule) Name() string {
	return "NamingUKConventionRule"
}

// OnEnter is called when entering a parse tree node.
func (r *NamingUKConventionRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
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
		// Other node types
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*NamingUKConventionRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *NamingUKConventionRule) checkCreateTable(ctx *mysql.CreateTableContext) {
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

	var indexDataList []*ukIndexMetaData
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

func (r *NamingUKConventionRule) checkAlterTable(ctx *mysql.AlterTableContext) {
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
	var indexDataList []*ukIndexMetaData
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
			indexStateMap := r.originCatalog.Index("", tableName)
			if indexStateMap == nil {
				continue
			}
			indexState, ok := (*indexStateMap)[oldIndexName]
			if !ok {
				continue
			}
			if !indexState.Unique() {
				continue
			}
			columnList := indexState.ExpressionList()
			metaData := map[string]string{
				advisor.ColumnListTemplateToken: strings.Join(columnList, "_"),
				advisor.TableNameTemplateToken:  tableName,
			}
			indexData := &ukIndexMetaData{
				indexName: newIndexName,
				tableName: tableName,
				metaData:  metaData,
				line:      r.baseLine + ctx.GetStart().GetLine(),
			}
			indexDataList = append(indexDataList, indexData)
		default:
			// Other alter operations
		}
	}
	r.handleIndexList(indexDataList)
}

func (r *NamingUKConventionRule) checkCreateIndex(ctx *mysql.CreateIndexContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	// only focus on unique index.
	if ctx.UNIQUE_SYMBOL() == nil {
		return
	}
	if ctx.IndexName() == nil || ctx.CreateIndexTarget() == nil || ctx.CreateIndexTarget().TableRef() == nil {
		return
	}

	indexName := mysqlparser.NormalizeIndexName(ctx.IndexName())
	_, tableName := mysqlparser.NormalizeMySQLTableRef(ctx.CreateIndexTarget().TableRef())

	if ctx.CreateIndexTarget().KeyListVariants() == nil {
		return
	}
	columnList := mysqlparser.NormalizeKeyListVariants(ctx.CreateIndexTarget().KeyListVariants())
	metaData := map[string]string{
		advisor.ColumnListTemplateToken: strings.Join(columnList, "_"),
		advisor.TableNameTemplateToken:  tableName,
	}
	indexDataList := []*ukIndexMetaData{
		{
			indexName: indexName,
			tableName: tableName,
			metaData:  metaData,
			line:      r.baseLine + ctx.GetStart().GetLine(),
		},
	}
	r.handleIndexList(indexDataList)
}

func (r *NamingUKConventionRule) handleIndexList(indexDataList []*ukIndexMetaData) {
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
				Code:          code.NamingUKConventionMismatch.Int32(),
				Title:         r.title,
				Content:       fmt.Sprintf("Unique key in table `%s` mismatches the naming convention, expect %q but found `%s`", indexData.tableName, regex, indexData.indexName),
				StartPosition: common.ConvertANTLRLineToPosition(indexData.line),
			})
		}
		if r.maxLength > 0 && len(indexData.indexName) > r.maxLength {
			r.AddAdvice(&storepb.Advice{
				Status:        r.level,
				Code:          code.NamingUKConventionMismatch.Int32(),
				Title:         r.title,
				Content:       fmt.Sprintf("Unique key `%s` in table `%s` mismatches the naming convention, its length should be within %d characters", indexData.indexName, indexData.tableName, r.maxLength),
				StartPosition: common.ConvertANTLRLineToPosition(indexData.line),
			})
		}
	}
}

func (r *NamingUKConventionRule) handleConstraintDef(tableName string, ctx mysql.ITableConstraintDefContext) *ukIndexMetaData {
	// focus on unique index.
	if ctx.UNIQUE_SYMBOL() == nil || ctx.KeyListVariants() == nil {
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
	return &ukIndexMetaData{
		indexName: indexName,
		tableName: tableName,
		metaData:  metaData,
		line:      r.baseLine + ctx.GetStart().GetLine(),
	}
}
