package mysql

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/bytebase/bytebase/plugin/catalog"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/pingcap/tidb/parser/ast"
	"go.uber.org/zap"
)

var (
	_ advisor.Advisor = (*NamingIndexConventionAdvisor)(nil)
)

func init() {
	advisor.Register(db.MySQL, advisor.MySQLNamingIndexConvention, &NamingIndexConventionAdvisor{})
	advisor.Register(db.TiDB, advisor.MySQLNamingIndexConvention, &NamingIndexConventionAdvisor{})
}

// NamingIndexConventionAdvisor is the advisor checking for index naming convention.
type NamingIndexConventionAdvisor struct {
}

// Check checks for index naming convention.
func (check *NamingIndexConventionAdvisor) Check(ctx advisor.Context, statement string) ([]advisor.Advice, error) {
	root, errAdvice := parseStatement(statement, ctx.Charset, ctx.Collation)
	if errAdvice != nil {
		return errAdvice, nil
	}

	level, err := advisor.NewStatusBySchemaReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	format, templateList, err := advisor.UnmarshalNamingRulePayloadAsTemplate(ctx.Rule.Type, ctx.Rule.Payload)
	if err != nil {
		return nil, err
	}
	checker := &namingIndexConventionChecker{
		level:        level,
		title:        string(ctx.Rule.Type),
		format:       format,
		templateList: templateList,
		catalog:      ctx.Catalog,
	}
	for _, stmtNode := range root {
		(stmtNode).Accept(checker)
	}

	if len(checker.adviceList) == 0 {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  advisor.Success,
			Code:    common.Ok,
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
	templateList []string
	catalog      catalog.Catalog
}

// Enter implements the ast.Visitor interface
func (checker *namingIndexConventionChecker) Enter(in ast.Node) (ast.Node, bool) {
	indexDataList := checker.getMetaDataList(in)

	for _, indexData := range indexDataList {
		regex, err := getTemplateRegexp(checker.format, checker.templateList, indexData.metaData)
		if err != nil {
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    common.Internal,
				Title:   "Internal error for index naming convention rule",
				Content: fmt.Sprintf("%q meet internal error %q", in.Text(), err.Error()),
			})
			continue
		}
		if !regex.MatchString(indexData.indexName) {
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    common.NamingIndexConventionMismatch,
				Title:   checker.title,
				Content: fmt.Sprintf("Index in table `%s` mismatches the naming convention, expect %q but found `%s`", indexData.tableName, regex, indexData.indexName),
			})
		}
	}

	return in, false
}

// Leave implements the ast.Visitor interface
func (checker *namingIndexConventionChecker) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}

type indexMetaData struct {
	indexName string
	tableName string
	metaData  map[string]string
}

// getMetaDataList returns the list of index with meta data.
func (checker *namingIndexConventionChecker) getMetaDataList(in ast.Node) []*indexMetaData {
	var res []*indexMetaData

	switch node := in.(type) {
	case *ast.CreateTableStmt:
		for _, constraint := range node.Constraints {
			if constraint.Tp == ast.ConstraintIndex {
				var columnList []string
				for _, key := range constraint.Keys {
					columnList = append(columnList, key.Column.Name.String())
				}
				metaData := map[string]string{
					advisor.ColumnListTemplateToken: strings.Join(columnList, "_"),
					advisor.TableNameTemplateToken:  node.Table.Name.String(),
				}
				res = append(res, &indexMetaData{
					indexName: constraint.Name,
					tableName: node.Table.Name.String(),
					metaData:  metaData,
				})
			}
		}
	case *ast.AlterTableStmt:
		for _, spec := range node.Specs {
			switch spec.Tp {
			case ast.AlterTableRenameIndex:
				ctx := context.Background()
				index, err := checker.catalog.FindIndex(ctx, &catalog.IndexFind{
					TableName: node.Table.Name.String(),
					IndexName: spec.FromKey.String(),
				})
				if err != nil {
					log.Error(
						"Cannot find index in table",
						zap.String("table_name", node.Table.Name.String()),
						zap.String("index_name", spec.FromKey.String()),
						zap.Error(err),
					)
					continue
				}
				if index.Unique {
					// Unique index naming convention should in advisor_naming_unique_key_convention.go
					continue
				}
				metaData := map[string]string{
					advisor.ColumnListTemplateToken: strings.Join(index.ColumnExpressions, "_"),
					advisor.TableNameTemplateToken:  node.Table.Name.String(),
				}
				res = append(res, &indexMetaData{
					indexName: spec.ToKey.String(),
					tableName: node.Table.Name.String(),
					metaData:  metaData,
				})
			case ast.AlterTableAddConstraint:
				if spec.Constraint.Tp == ast.ConstraintIndex {
					var columnList []string
					for _, key := range spec.Constraint.Keys {
						columnList = append(columnList, key.Column.Name.String())
					}

					metaData := map[string]string{
						advisor.ColumnListTemplateToken: strings.Join(columnList, "_"),
						advisor.TableNameTemplateToken:  node.Table.Name.String(),
					}
					res = append(res, &indexMetaData{
						indexName: spec.Constraint.Name,
						tableName: node.Table.Name.String(),
						metaData:  metaData,
					})
				}
			}
		}
	case *ast.CreateIndexStmt:
		if node.KeyType != ast.IndexKeyTypeUnique {
			var columnList []string
			for _, spec := range node.IndexPartSpecifications {
				columnList = append(columnList, spec.Column.Name.String())
			}
			metaData := map[string]string{
				advisor.ColumnListTemplateToken: strings.Join(columnList, "_"),
				advisor.TableNameTemplateToken:  node.Table.Name.String(),
			}
			res = append(res, &indexMetaData{
				indexName: node.IndexName,
				tableName: node.Table.Name.String(),
				metaData:  metaData,
			})
		}
	}

	return res
}

// getTemplateRegexp formats the template as regex.
func getTemplateRegexp(template string, templateList []string, tokens map[string]string) (*regexp.Regexp, error) {
	for _, key := range templateList {
		if token, ok := tokens[key]; ok {
			template = strings.ReplaceAll(template, key, token)
		}
	}

	return regexp.Compile(template)
}
