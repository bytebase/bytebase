package mysql

import (
	"fmt"
	"strings"

	"github.com/pingcap/tidb/parser/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*NamingFKConventionAdvisor)(nil)
	_ ast.Visitor     = (*namingFKConventionChecker)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.MySQLNamingFKConvention, &NamingFKConventionAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.MySQLNamingFKConvention, &NamingFKConventionAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.MySQLNamingFKConvention, &NamingFKConventionAdvisor{})
	advisor.Register(storepb.Engine_TIDB, advisor.MySQLNamingFKConvention, &NamingFKConventionAdvisor{})
}

// NamingFKConventionAdvisor is the advisor checking for foreign key naming convention.
type NamingFKConventionAdvisor struct {
}

// Check checks for foreign key naming convention.
func (*NamingFKConventionAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
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
	checker := &namingFKConventionChecker{
		level:        level,
		title:        string(ctx.Rule.Type),
		format:       format,
		maxLength:    maxLength,
		templateList: templateList,
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

type namingFKConventionChecker struct {
	adviceList   []advisor.Advice
	level        advisor.Status
	title        string
	format       string
	maxLength    int
	templateList []string
}

// Enter implements the ast.Visitor interface.
func (checker *namingFKConventionChecker) Enter(in ast.Node) (ast.Node, bool) {
	indexDataList := checker.getMetaDataList(in)

	for _, indexData := range indexDataList {
		regex, err := getTemplateRegexp(checker.format, checker.templateList, indexData.metaData)
		if err != nil {
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    advisor.Internal,
				Title:   "Internal error for foreign key naming convention rule",
				Content: fmt.Sprintf("%q meet internal error %q", in.Text(), err.Error()),
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

	return in, false
}

// Leave implements the ast.Visitor interface.
func (*namingFKConventionChecker) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}

// getMetaDataList returns the list of foreign key with meta data.
func (*namingFKConventionChecker) getMetaDataList(in ast.Node) []*indexMetaData {
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
					advisor.ReferencingTableNameTemplateToken:  node.Table.Name.String(),
					advisor.ReferencingColumnNameTemplateToken: strings.Join(referencingColumnList, "_"),
					advisor.ReferencedTableNameTemplateToken:   constraint.Refer.Table.Name.String(),
					advisor.ReferencedColumnNameTemplateToken:  strings.Join(referencedColumnList, "_"),
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
					advisor.ReferencingTableNameTemplateToken:  node.Table.Name.String(),
					advisor.ReferencingColumnNameTemplateToken: strings.Join(referencingColumnList, "_"),
					advisor.ReferencedTableNameTemplateToken:   spec.Constraint.Refer.Table.Name.String(),
					advisor.ReferencedColumnNameTemplateToken:  strings.Join(referencedColumnList, "_"),
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

	return res
}
