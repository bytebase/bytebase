package pg

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/catalog"
	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*NamingIndexConventionAdvisor)(nil)
	_ ast.Visitor     = (*namingIndexConventionChecker)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.PostgreSQLNamingIndexConvention, &NamingIndexConventionAdvisor{})
}

// NamingIndexConventionAdvisor is the advisor checking for index naming convention.
type NamingIndexConventionAdvisor struct {
}

// Check checks for index naming convention.
func (*NamingIndexConventionAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	stmts, ok := ctx.AST.([]ast.Node)
	if !ok {
		return nil, errors.Errorf("failed to convert to Node")
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
	for _, stmt := range stmts {
		ast.Walk(checker, stmt)
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
	adviceList   []advisor.Advice
	level        advisor.Status
	title        string
	format       string
	maxLength    int
	templateList []string
	catalog      *catalog.Finder
}

// Visit implements ast.Visitor interface.
func (checker *namingIndexConventionChecker) Visit(node ast.Node) ast.Visitor {
	indexDataList := checker.getMetaDataList(node)

	for _, indexData := range indexDataList {
		regex, err := getTemplateRegexp(checker.format, checker.templateList, indexData.metaData)
		if err != nil {
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    advisor.Internal,
				Title:   "Internal error for index naming convention rule",
				Content: fmt.Sprintf("%q meet internal error %q", node.Text(), err.Error()),
			})
			continue
		}
		if !regex.MatchString(indexData.indexName) {
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    advisor.NamingIndexConventionMismatch,
				Title:   checker.title,
				Content: fmt.Sprintf("Index in table %q mismatches the naming convention, expect %q but found %q", indexData.tableName, regex, indexData.indexName),
				Line:    node.LastLine(),
			})
		}
		if checker.maxLength > 0 && len(indexData.indexName) > checker.maxLength {
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    advisor.NamingIndexConventionMismatch,
				Title:   checker.title,
				Content: fmt.Sprintf("Index %q in table %q mismatches the naming convention, its length should be within %d characters", indexData.indexName, indexData.tableName, checker.maxLength),
				Line:    node.LastLine(),
			})
		}
	}
	return checker
}

func (checker *namingIndexConventionChecker) getMetaDataList(in ast.Node) []*indexMetaData {
	var res []*indexMetaData

	switch node := in.(type) {
	case *ast.CreateIndexStmt:
		if !node.Index.Unique {
			var columnList []string
			for _, key := range node.Index.KeyList {
				columnList = append(columnList, key.Key)
			}
			metaData := map[string]string{
				advisor.ColumnListTemplateToken: strings.Join(columnList, "_"),
				advisor.TableNameTemplateToken:  node.Index.Table.Name,
			}
			res = append(res, &indexMetaData{
				indexName: node.Index.Name,
				tableName: node.Index.Table.Name,
				metaData:  metaData,
			})
		}
	case *ast.RenameIndexStmt:
		tableName, index := checker.catalog.Origin.FindIndex(&catalog.IndexFind{
			SchemaName: normalizeSchemaName(node.Table.Schema),
			TableName:  "",
			IndexName:  node.IndexName,
		})
		if index != nil && !index.Unique() {
			metaData := map[string]string{
				advisor.ColumnListTemplateToken: strings.Join(index.ExpressionList(), "_"),
				advisor.TableNameTemplateToken:  tableName,
			}
			res = append(res, &indexMetaData{
				indexName: node.NewName,
				tableName: tableName,
				metaData:  metaData,
			})
		}
	}

	return res
}
