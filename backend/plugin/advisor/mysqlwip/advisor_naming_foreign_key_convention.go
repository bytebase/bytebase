package mysqlwip

import (
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	mysql "github.com/bytebase/mysql-parser"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*NamingFKConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ENGINE_UNSPECIFIED, advisor.MySQLNamingFKConvention, &NamingFKConventionAdvisor{})
}

// NamingFKConventionAdvisor is the advisor checking for foreign key naming convention.
type NamingFKConventionAdvisor struct {
}

// Check checks for foreign key naming convention.
func (*NamingFKConventionAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	root, ok := ctx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	format, templateList, maxLength, err := advisor.UnmarshalNamingRulePayloadAsTemplate(advisor.SQLReviewRuleType(ctx.Rule.Type), ctx.Rule.Payload)
	if err != nil {
		return nil, err
	}
	checker := &namingFKConventionChecker{
		level:        level,
		title:        string(ctx.Rule.Type),
		format:       format,
		maxLength:    maxLength,
		templateList: templateList,
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

type namingFKConventionChecker struct {
	*mysql.BaseMySQLParserListener

	baseLine     int
	text         string
	adviceList   []advisor.Advice
	level        advisor.Status
	title        string
	format       string
	maxLength    int
	templateList []string
}

func (checker *namingFKConventionChecker) EnterQuery(ctx *mysql.QueryContext) {
	checker.text = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
}

func (checker *namingFKConventionChecker) EnterCreateTable(ctx *mysql.CreateTableContext) {
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

func (checker *namingFKConventionChecker) EnterAlterTable(ctx *mysql.AlterTableContext) {
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

		// add constriant.
		if alterListItem.ADD_SYMBOL() != nil && alterListItem.TableConstraintDef() != nil {
			if metaData := checker.handleConstraintDef(tableName, alterListItem.TableConstraintDef()); metaData != nil {
				indexDataList = append(indexDataList, metaData)
			}
		}
	}
	checker.handleIndexList(indexDataList)
}

func (checker *namingFKConventionChecker) handleIndexList(indexDataList []*indexMetaData) {
	for _, indexData := range indexDataList {
		regex, err := getTemplateRegexp(checker.format, checker.templateList, indexData.metaData)
		if err != nil {
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    advisor.Internal,
				Title:   "Internal error for foreign key naming convention rule",
				Content: fmt.Sprintf("%q meet internal error %q", checker.text, err.Error()),
			})
			continue
		}
		if !regex.MatchString(indexData.indexName) {
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    advisor.NamingFKConventionMismatch,
				Title:   checker.title,
				Content: fmt.Sprintf("Foreign key in table `%s` mismatches the naming convention, expect %q but found `%s`", indexData.tableName, regex, indexData.indexName),
				Line:    indexData.line,
			})
		}
		if checker.maxLength > 0 && len(indexData.indexName) > checker.maxLength {
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    advisor.NamingFKConventionMismatch,
				Title:   checker.title,
				Content: fmt.Sprintf("Foreign key `%s` in table `%s` mismatches the naming convention, its length should be within %d characters", indexData.indexName, indexData.tableName, checker.maxLength),
				Line:    indexData.line,
			})
		}
	}
}

func (checker *namingFKConventionChecker) handleConstraintDef(tableName string, ctx mysql.ITableConstraintDefContext) *indexMetaData {
	// focus on foreign index.
	if ctx.FOREIGN_SYMBOL() == nil || ctx.KEY_SYMBOL() == nil || ctx.KeyList() == nil || ctx.References() == nil {
		return nil
	}

	indexName := ""
	// for mysql, foreign key use constraint name as unique identifier.
	if ctx.ConstraintName() != nil {
		indexName = mysqlparser.NormalizeConstraintName(ctx.ConstraintName())
	}

	referencingColumnList := mysqlparser.NormalizeKeyList(ctx.KeyList())
	referencedTable, referencedColumnList := checker.handleReferences(ctx.References())
	metaData := map[string]string{
		advisor.ReferencingTableNameTemplateToken:  tableName,
		advisor.ReferencingColumnNameTemplateToken: strings.Join(referencingColumnList, "_"),
		advisor.ReferencedTableNameTemplateToken:   referencedTable,
		advisor.ReferencedColumnNameTemplateToken:  strings.Join(referencedColumnList, "_"),
	}
	return &indexMetaData{
		indexName: indexName,
		tableName: tableName,
		metaData:  metaData,
		line:      checker.baseLine + ctx.GetStart().GetLine(),
	}
}

func (*namingFKConventionChecker) handleReferences(ctx mysql.IReferencesContext) (string, []string) {
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
