package tidb

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/pingcap/tidb/pkg/parser/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*NamingFKConventionAdvisor)(nil)
	_ ast.Visitor     = (*namingFKConventionChecker)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_NAMING_INDEX_FK, &NamingFKConventionAdvisor{})
}

// NamingFKConventionAdvisor is the advisor checking for foreign key naming convention.
type NamingFKConventionAdvisor struct {
}

// Check checks for foreign key naming convention.
func (*NamingFKConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	root, err := getTiDBNodes(checkCtx)

	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	namingPayload := checkCtx.Rule.GetNamingPayload()
	if namingPayload == nil {
		return nil, errors.New("naming_payload is required for this rule")
	}

	format := namingPayload.Format
	templateList, _ := advisor.ParseTemplateTokens(format)

	for _, key := range templateList {
		if _, ok := advisor.TemplateNamingTokens[checkCtx.Rule.Type][key]; !ok {
			return nil, errors.Errorf("invalid template %s for rule %s", key, checkCtx.Rule.Type)
		}
	}

	maxLength := int(namingPayload.MaxLength)
	if maxLength == 0 {
		maxLength = advisor.DefaultNameLengthLimit
	}
	checker := &namingFKConventionChecker{
		level:        level,
		title:        checkCtx.Rule.Type.String(),
		format:       format,
		maxLength:    maxLength,
		templateList: templateList,
	}
	for _, stmtNode := range root {
		(stmtNode).Accept(checker)
	}

	return checker.adviceList, nil
}

type namingFKConventionChecker struct {
	adviceList   []*storepb.Advice
	level        storepb.Advice_Status
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
			checker.adviceList = append(checker.adviceList, &storepb.Advice{
				Status:  checker.level,
				Code:    code.Internal.Int32(),
				Title:   "Internal error for foreign key naming convention rule",
				Content: fmt.Sprintf("%q meet internal error %q", in.Text(), err.Error()),
			})
			continue
		}
		if !regex.MatchString(indexData.indexName) {
			checker.adviceList = append(checker.adviceList, &storepb.Advice{
				Status:        checker.level,
				Code:          code.NamingFKConventionMismatch.Int32(),
				Title:         checker.title,
				Content:       fmt.Sprintf("Foreign key in table `%s` mismatches the naming convention, expect %q but found `%s`", indexData.tableName, regex, indexData.indexName),
				StartPosition: common.ConvertANTLRLineToPosition(indexData.line),
			})
		}
		if checker.maxLength > 0 && len(indexData.indexName) > checker.maxLength {
			checker.adviceList = append(checker.adviceList, &storepb.Advice{
				Status:        checker.level,
				Code:          code.NamingFKConventionMismatch.Int32(),
				Title:         checker.title,
				Content:       fmt.Sprintf("Foreign key `%s` in table `%s` mismatches the naming convention, its length should be within %d characters", indexData.indexName, indexData.tableName, checker.maxLength),
				StartPosition: common.ConvertANTLRLineToPosition(indexData.line),
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
