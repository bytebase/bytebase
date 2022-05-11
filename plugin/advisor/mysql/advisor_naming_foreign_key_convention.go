package mysql

import (
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/pingcap/tidb/parser/ast"
)

var (
	_ advisor.Advisor = (*NamingFKConventionAdvisor)(nil)
)

func init() {
	advisor.Register(db.MySQL, advisor.MySQLNamingFKConvention, &NamingFKConventionAdvisor{})
	advisor.Register(db.TiDB, advisor.MySQLNamingFKConvention, &NamingFKConventionAdvisor{})
}

// NamingFKConventionAdvisor is the advisor checking for foreign key naming convention.
type NamingFKConventionAdvisor struct {
}

// Check checks for foreign key naming convention.
func (check *NamingFKConventionAdvisor) Check(ctx advisor.Context, statement string) ([]advisor.Advice, error) {
	root, errAdvice := parseStatement(statement, ctx.Charset, ctx.Collation)
	if errAdvice != nil {
		return errAdvice, nil
	}

	level, err := advisor.NewStatusBySchemaReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	format, templateList, err := api.UnmarshalNamingRulePayloadAsTemplate(ctx.Rule.Type, ctx.Rule.Payload)
	if err != nil {
		return nil, err
	}
	checker := &namingFKConventionChecker{
		level:        level,
		format:       format,
		templateList: templateList,
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

type namingFKConventionChecker struct {
	adviceList   []advisor.Advice
	level        advisor.Status
	format       string
	templateList []string
}

// Enter implements the ast.Visitor interface
func (checker *namingFKConventionChecker) Enter(in ast.Node) (ast.Node, bool) {
	indexDataList := checker.getMetaDataList(in)

	for _, indexData := range indexDataList {
		regex, err := getTemplateRegexp(checker.format, checker.templateList, indexData.metaData)
		if err != nil {
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    common.Internal,
				Title:   "Internal error for foreign key naming convention rule",
				Content: fmt.Sprintf("%q meet internal error %q", in.Text(), err.Error()),
			})
			continue
		}
		if !regex.MatchString(indexData.indexName) {
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    common.NamingFKConventionMismatch,
				Title:   "Mismatch foreign key naming convention",
				Content: fmt.Sprintf("Foreign key in table `%s` mismatches the naming convention, expect %q but found `%s`", indexData.tableName, regex, indexData.indexName),
			})
		}
	}

	return in, false
}

// Leave implements the ast.Visitor interface
func (checker *namingFKConventionChecker) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}

// getMetaDataList returns the list of foreign key with meta data.
func (checker *namingFKConventionChecker) getMetaDataList(in ast.Node) []*indexMetaData {
	var res []*indexMetaData

	switch node := in.(type) {
	case *ast.CreateTableStmt:
		for _, constraint := range node.Constraints {
			if constraint.Tp == ast.ConstraintForeignKey {
				var referencingColumnList []string
				for _, key := range constraint.Keys {
					referencingColumnList = append(referencingColumnList, key.Column.Name.String())
				}
				var referencedColumnList []string
				for _, spec := range constraint.Refer.IndexPartSpecifications {
					referencedColumnList = append(referencedColumnList, spec.Column.Name.String())
				}

				metaData := map[string]string{
					api.ReferencingTableNameTemplateToken:  node.Table.Name.String(),
					api.ReferencingColumnNameTemplateToken: strings.Join(referencingColumnList, "_"),
					api.ReferencedTableNameTemplateToken:   constraint.Refer.Table.Name.String(),
					api.ReferencedColumnNameTemplateToken:  strings.Join(referencedColumnList, "_"),
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
			if spec.Tp == ast.AlterTableAddConstraint && spec.Constraint.Tp == ast.ConstraintForeignKey {
				var referencingColumnList []string
				for _, key := range spec.Constraint.Keys {
					referencingColumnList = append(referencingColumnList, key.Column.Name.String())
				}
				var referencedColumnList []string
				for _, spec := range spec.Constraint.Refer.IndexPartSpecifications {
					referencedColumnList = append(referencedColumnList, spec.Column.Name.String())
				}

				metaData := map[string]string{
					api.ReferencingTableNameTemplateToken:  node.Table.Name.String(),
					api.ReferencingColumnNameTemplateToken: strings.Join(referencingColumnList, "_"),
					api.ReferencedTableNameTemplateToken:   spec.Constraint.Refer.Table.Name.String(),
					api.ReferencedColumnNameTemplateToken:  strings.Join(referencedColumnList, "_"),
				}
				res = append(res, &indexMetaData{
					indexName: spec.Constraint.Name,
					tableName: node.Table.Name.String(),
					metaData:  metaData,
				})
			}
		}
	}

	return res
}
