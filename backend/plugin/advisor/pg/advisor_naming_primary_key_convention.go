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
	_ advisor.Advisor = (*NamingPKConventionAdvisor)(nil)
	_ ast.Visitor     = (*namingPKConventionChecker)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.PostgreSQLNamingPKConvention, &NamingPKConventionAdvisor{})
}

// NamingPKConventionAdvisor is the advisor checking for primary key naming convention.
type NamingPKConventionAdvisor struct {
}

// Check checks for index naming convention.
func (*NamingPKConventionAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
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

	checker := &namingPKConventionChecker{
		level:        level,
		title:        string(ctx.Rule.Type),
		format:       format,
		maxLength:    maxLength,
		templateList: templateList,
		catalog:      ctx.Catalog,
	}

	for _, stmtNode := range stmts {
		ast.Walk(checker, stmtNode)
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

type namingPKConventionChecker struct {
	adviceList   []advisor.Advice
	level        advisor.Status
	title        string
	format       string
	maxLength    int
	templateList []string
	catalog      *catalog.Finder
}

// Visit implements ast.Visitor interface.
func (checker *namingPKConventionChecker) Visit(in ast.Node) ast.Visitor {
	indexDataList := checker.getMetaDataList(in)

	for _, indexData := range indexDataList {
		regex, err := getTemplateRegexp(checker.format, checker.templateList, indexData.metaData)
		if err != nil {
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    advisor.Internal,
				Title:   "Internal error for primary key naming convention rule",
				Content: fmt.Sprintf("%q meet internal error %q", in.Text(), err.Error()),
			})
			continue
		}
		if !regex.MatchString(indexData.indexName) {
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    advisor.NamingPKConventionMismatch,
				Title:   checker.title,
				Content: fmt.Sprintf(`Primary key in table "%s" mismatches the naming convention, expect %q but found "%s"`, indexData.tableName, regex, indexData.indexName),
				Line:    indexData.line,
			})
		}
		if checker.maxLength > 0 && len(indexData.indexName) > checker.maxLength {
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    advisor.NamingPKConventionMismatch,
				Title:   checker.title,
				Content: fmt.Sprintf(`Primary key "%s" in table "%s" mismatches the naming convention, its length should be within %d characters`, indexData.indexName, indexData.tableName, checker.maxLength),
				Line:    indexData.line,
			})
		}
	}

	return checker
}

// getMetaDataList returns the list of primary key with metadata.
func (checker *namingPKConventionChecker) getMetaDataList(in ast.Node) []*indexMetaData {
	var res []*indexMetaData
	switch node := in.(type) {
	case *ast.CreateTableStmt:
		for _, constraint := range node.ConstraintList {
			if metadata := checker.getPrimaryKeyMetadata(node.Name.Schema, node.Name.Name, constraint, constraint.LastLine()); metadata != nil {
				res = append(res, metadata)
			}
		}
		for _, column := range node.ColumnList {
			for _, constraint := range column.ConstraintList {
				if metadata := checker.getPrimaryKeyMetadata(node.Name.Schema, node.Name.Name, constraint, column.LastLine()); metadata != nil {
					res = append(res, metadata)
				}
			}
		}
	case *ast.AddConstraintStmt:
		constraint := node.Constraint
		if metadata := checker.getPrimaryKeyMetadata(node.Table.Schema, node.Table.Name, constraint, in.LastLine()); metadata != nil {
			res = append(res, metadata)
		}
	case *ast.AddColumnListStmt:
		for _, column := range node.ColumnList {
			for _, constraint := range column.ConstraintList {
				if metadata := checker.getPrimaryKeyMetadata(node.Table.Schema, node.Table.Name, constraint, in.LastLine()); metadata != nil {
					res = append(res, metadata)
				}
			}
		}
	case *ast.RenameConstraintStmt:
		tableName, index := checker.findIndex(node.Table.Schema, node.Table.Name, node.ConstraintName)
		if index != nil && index.Primary() {
			metaData := map[string]string{
				advisor.ColumnListTemplateToken: strings.Join(index.ExpressionList(), "_"),
				advisor.TableNameTemplateToken:  tableName,
			}
			res = append(res, &indexMetaData{
				indexName: node.NewName,
				tableName: node.Table.Name,
				line:      in.LastLine(),
				metaData:  metaData,
			})
		}
	case *ast.RenameIndexStmt:
		// "ALTER INDEX name RENAME TO new_name" doesn't take a table name
		tableName, index := checker.findIndex(node.Table.Schema, "", node.IndexName)
		if index != nil && index.Primary() {
			metaData := map[string]string{
				advisor.ColumnListTemplateToken: strings.Join(index.ExpressionList(), "_"),
				advisor.TableNameTemplateToken:  tableName,
			}
			res = append(res, &indexMetaData{
				indexName: node.NewName,
				tableName: tableName,
				line:      in.LastLine(),
				metaData:  metaData,
			})
		}
	}
	return res
}

// getPrimaryKeyMetadata returns index metadata of a primary key constraint, nil if other constraints.
func (checker *namingPKConventionChecker) getPrimaryKeyMetadata(schemaName string, tableName string, constraint *ast.ConstraintDef, line int) *indexMetaData {
	switch constraint.Type {
	case ast.ConstraintTypePrimary:
		metaData := map[string]string{
			advisor.ColumnListTemplateToken: strings.Join(constraint.KeyList, "_"),
			advisor.TableNameTemplateToken:  tableName,
		}
		return &indexMetaData{
			indexName: constraint.Name,
			tableName: tableName,
			line:      line,
			metaData:  metaData,
		}
	case ast.ConstraintTypePrimaryUsingIndex:
		tableName, index := checker.findIndex(schemaName, tableName, constraint.IndexName)
		if index != nil {
			metaData := map[string]string{
				advisor.ColumnListTemplateToken: strings.Join(index.ExpressionList(), "_"),
				advisor.TableNameTemplateToken:  tableName,
			}
			return &indexMetaData{
				indexName: constraint.Name,
				tableName: tableName,
				line:      line,
				metaData:  metaData,
			}
		}
	}
	return nil
}

// findIndex returns index found in catalogs, nil if not found.
func (checker *namingPKConventionChecker) findIndex(schemaName string, tableName string, indexName string) (string, *catalog.IndexState) {
	return checker.catalog.Origin.FindIndex(&catalog.IndexFind{
		SchemaName: normalizeSchemaName(schemaName),
		TableName:  tableName,
		IndexName:  indexName,
	})
}
