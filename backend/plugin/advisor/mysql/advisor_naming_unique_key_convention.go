package mysql

import (
	"fmt"
	"strings"

	"github.com/pingcap/tidb/parser/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/catalog"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*NamingUKConventionAdvisor)(nil)
	_ ast.Visitor     = (*namingUKConventionChecker)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.MySQLNamingUKConvention, &NamingUKConventionAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.MySQLNamingUKConvention, &NamingUKConventionAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.MySQLNamingUKConvention, &NamingUKConventionAdvisor{})
}

// NamingUKConventionAdvisor is the advisor checking for unique key naming convention.
type NamingUKConventionAdvisor struct {
}

// Check checks for index naming convention.
func (*NamingUKConventionAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	root, ok := ctx.AST.([]ast.StmtNode)
	if !ok {
		return nil, errors.Errorf("failed to convert to StmtNode")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	format, templateList, maxLength, err := advisor.UnmarshalNamingRulePayloadAsTemplate(advisor.SQLReviewRuleType(ctx.Rule.Type), ctx.Rule.Payload)
	if err != nil {
		return nil, err
	}
	checker := &namingUKConventionChecker{
		level:        level,
		title:        string(ctx.Rule.Type),
		format:       format,
		maxLength:    maxLength,
		templateList: templateList,
		catalog:      ctx.Catalog,
	}
	for _, stmtNode := range root {
		(stmtNode).Accept(checker)
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

type namingUKConventionChecker struct {
	adviceList   []advisor.Advice
	level        advisor.Status
	title        string
	format       string
	maxLength    int
	templateList []string
	catalog      *catalog.Finder
}

// Enter implements the ast.Visitor interface.
func (checker *namingUKConventionChecker) Enter(in ast.Node) (ast.Node, bool) {
	indexDataList := checker.getMetaDataList(in)

	for _, indexData := range indexDataList {
		regex, err := getTemplateRegexp(checker.format, checker.templateList, indexData.metaData)
		if err != nil {
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    advisor.Internal,
				Title:   "Internal error for unique key naming convention rule",
				Content: fmt.Sprintf("%q meet internal error %q", in.Text(), err.Error()),
			})
			continue
		}
		if !regex.MatchString(indexData.indexName) {
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    advisor.NamingUKConventionMismatch,
				Title:   checker.title,
				Content: fmt.Sprintf("Unique key in table `%s` mismatches the naming convention, expect %q but found `%s`", indexData.tableName, regex, indexData.indexName),
				Line:    indexData.line,
			})
		}
		if checker.maxLength > 0 && len(indexData.indexName) > checker.maxLength {
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    advisor.NamingUKConventionMismatch,
				Title:   checker.title,
				Content: fmt.Sprintf("Unique key `%s` in table `%s` mismatches the naming convention, its length should be within %d characters", indexData.indexName, indexData.tableName, checker.maxLength),
				Line:    indexData.line,
			})
		}
	}

	return in, false
}

// Leave implements the ast.Visitor interface.
func (*namingUKConventionChecker) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}

// getMetaDataList returns the list of unique key with meta data.
func (checker *namingUKConventionChecker) getMetaDataList(in ast.Node) []*indexMetaData {
	var res []*indexMetaData

	switch node := in.(type) {
	case *ast.CreateTableStmt:
		for _, constraint := range node.Constraints {
			switch constraint.Tp {
			case ast.ConstraintUniq, ast.ConstraintUniqKey, ast.ConstraintUniqIndex:
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
					line:      constraint.OriginTextPosition(),
				})
			}
		}
	case *ast.AlterTableStmt:
		for _, spec := range node.Specs {
			switch spec.Tp {
			case ast.AlterTableRenameIndex:
				_, index := checker.catalog.Origin.FindIndex(&catalog.IndexFind{
					TableName: node.Table.Name.String(),
					IndexName: spec.FromKey.String(),
				})
				if index == nil {
					continue
				}
				if !index.Unique() {
					// Index naming convention should in advisor_naming_index_convention.go
					continue
				}
				metaData := map[string]string{
					advisor.ColumnListTemplateToken: strings.Join(index.ExpressionList(), "_"),
					advisor.TableNameTemplateToken:  node.Table.Name.String(),
				}
				res = append(res, &indexMetaData{
					indexName: spec.ToKey.String(),
					tableName: node.Table.Name.String(),
					metaData:  metaData,
					line:      in.OriginTextPosition(),
				})
			case ast.AlterTableAddConstraint:
				switch spec.Constraint.Tp {
				case ast.ConstraintUniq, ast.ConstraintUniqKey, ast.ConstraintUniqIndex:
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
						line:      in.OriginTextPosition(),
					})
				}
			}
		}
	case *ast.CreateIndexStmt:
		if node.KeyType == ast.IndexKeyTypeUnique {
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
				line:      in.OriginTextPosition(),
			})
		}
	}

	return res
}
