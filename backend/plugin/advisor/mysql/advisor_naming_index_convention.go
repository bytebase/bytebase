package mysqlwip

import (
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	mysql "github.com/bytebase/mysql-parser"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/catalog"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*NamingIndexConventionAdvisor)(nil)
)

func init() {
	// only for mysqlwip test.
	advisor.Register(storepb.Engine_ENGINE_UNSPECIFIED, advisor.MySQLNamingIndexConvention, &NamingIndexConventionAdvisor{})
}

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
func (*NamingIndexConventionAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	root, ok := ctx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parser result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	format, templateList, maxLength, err := advisor.UnmarshalNamingRulePayloadAsTemplate(advisor.SQLReviewRuleType(ctx.Rule.Type), ctx.Rule.Payload)
	if err != nil {
		return nil, err
	}
	checker := &namingIndexConventionChecker{
		level:        level,
		title:        string(ctx.Rule.Type),
		format:       format,
		maxLength:    maxLength,
		templateList: templateList,
		catalog:      ctx.Catalog,
	}
	for _, stmtNode := range root {
		checker.baseLine = stmtNode.BaseLine
		antlr.ParseTreeWalkerDefault.Walk(checker, stmtNode.Tree)
	}

	if len(checker.adviceList) == 0 {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  advisor.Success,
			Code:    advisor.Ok,
			Title:   "OK",
			Content: "",
		})
	}

	return checker.adviceList, nil
}

type namingIndexConventionChecker struct {
	*mysql.BaseMySQLParserListener

	baseLine     int
	text         string
	adviceList   []advisor.Advice
	level        advisor.Status
	title        string
	format       string
	maxLength    int
	templateList []string
	catalog      *catalog.Finder
}

func (checker *namingIndexConventionChecker) EnterQuery(ctx *mysql.QueryContext) {
	checker.text = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
}

func (checker *namingIndexConventionChecker) EnterCreateTable(ctx *mysql.CreateTableContext) {
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
		if metaData := checker.handleConstraintDef(tableName, tableElement.TableConstraintDef()); metaData != nil {
			indexDataList = append(indexDataList, metaData)
		}
	}
	checker.handleIndexList(indexDataList)
}

func (checker *namingIndexConventionChecker) EnterAlterTable(ctx *mysql.AlterTableContext) {
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
			if metaData := checker.handleConstraintDef(tableName, alterListItem.TableConstraintDef()); metaData != nil {
				indexDataList = append(indexDataList, metaData)
			}
		// rename index.
		case alterListItem.RENAME_SYMBOL() != nil && alterListItem.KeyOrIndex() != nil && alterListItem.IndexRef() != nil && alterListItem.IndexName() != nil:
			_, _, oldIndexName := mysqlparser.NormalizeIndexRef(alterListItem.IndexRef())
			newIndexName := mysqlparser.NormalizeIndexName(alterListItem.IndexName())
			_, indexState := checker.catalog.Origin.FindIndex(&catalog.IndexFind{
				TableName: tableName,
				IndexName: oldIndexName,
			})
			if indexState == nil {
				continue
			}
			if indexState.Unique() {
				// Unique index naming convention should in advisor_naming_unique_key_convention.go
				continue
			}
			columnList := indexState.ExpressionList()
			metaData := map[string]string{
				advisor.ColumnListTemplateToken: strings.Join(columnList, "_"),
				advisor.TableNameTemplateToken:  tableName,
			}
			indexData := &indexMetaData{
				indexName: newIndexName,
				tableName: tableName,
				metaData:  metaData,
				line:      checker.baseLine + ctx.GetStart().GetLine(),
			}
			indexDataList = append(indexDataList, indexData)
		}
	}
	checker.handleIndexList(indexDataList)
}

func (checker *namingIndexConventionChecker) EnterCreateIndex(ctx *mysql.CreateIndexContext) {
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
			line:      checker.baseLine + ctx.GetStart().GetLine(),
		},
	}
	checker.handleIndexList(indexDataList)
}

func (checker *namingIndexConventionChecker) handleIndexList(indexDataList []*indexMetaData) {
	for _, indexData := range indexDataList {
		regex, err := getTemplateRegexp(checker.format, checker.templateList, indexData.metaData)
		if err != nil {
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    advisor.Internal,
				Title:   "Internal error for unique key naming convention rule",
				Content: fmt.Sprintf("%q meet internal error %q", checker.text, err.Error()),
			})
			continue
		}
		if !regex.MatchString(indexData.indexName) {
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    advisor.NamingIndexConventionMismatch,
				Title:   checker.title,
				Content: fmt.Sprintf("Index in table `%s` mismatches the naming convention, expect %q but found `%s`", indexData.tableName, regex, indexData.indexName),
				Line:    indexData.line,
			})
		}
		if checker.maxLength > 0 && len(indexData.indexName) > checker.maxLength {
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    advisor.NamingIndexConventionMismatch,
				Title:   checker.title,
				Content: fmt.Sprintf("Index `%s` in table `%s` mismatches the naming convention, its length should be within %d characters", indexData.indexName, indexData.tableName, checker.maxLength),
				Line:    indexData.line,
			})
		}
	}
}

func (checker *namingIndexConventionChecker) handleConstraintDef(tableName string, ctx mysql.ITableConstraintDefContext) *indexMetaData {
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
		line:      checker.baseLine + ctx.GetStart().GetLine(),
	}
}
