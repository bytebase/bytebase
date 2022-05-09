package mysql

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/pingcap/tidb/parser/ast"
)

var (
	_ advisor.Advisor = (*NamingIndexConventionAdvisor)(nil)

	ruleMapping = map[api.SchemaReviewRuleType]ast.ConstraintType{
		api.SchemaRulePKNaming:  ast.ConstraintPrimaryKey,
		api.SchemaRuleIDXNaming: ast.ConstraintIndex,
		api.SchemaRuleUKNaming:  ast.ConstraintUniq,
		api.SchemaRuleFKNaming:  ast.ConstraintForeignKey,
	}
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

	format, templateList, err := api.UnmarshalNamingRulePayloadAsTemplate(ctx.Rule.Type, ctx.Rule.Payload)
	if err != nil {
		return nil, err
	}
	checker := &namingIndexConventionChecker{
		level:        level,
		ruleType:     ctx.Rule.Type,
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

type namingIndexConventionChecker struct {
	adviceList   []advisor.Advice
	level        advisor.Status
	ruleType     api.SchemaReviewRuleType
	format       string
	templateList []string
}

func (checker *namingIndexConventionChecker) Enter(in ast.Node) (ast.Node, bool) {
	indexDataList := checker.getIndexMetaDataList(in)

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
		if !regex.MatchString(indexData.index) {
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    common.NamingIndexConventionMismatch,
				Title:   "Mismatch index naming convention",
				Content: fmt.Sprintf("%q mismatches index naming convention, expect %q but found %q", in.Text(), checker.format, indexData.index),
			})
		}
	}

	if len(checker.adviceList) == 0 {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  advisor.Success,
			Code:    common.Ok,
			Title:   "OK",
			Content: "",
		})
	}

	return in, false
}

func (checker *namingIndexConventionChecker) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}

type indexMetaData struct {
	index    string
	metaData map[string]string
}

// getIndexMetaData returns the list of index with meta data.
func (checker *namingIndexConventionChecker) getIndexMetaDataList(in ast.Node) []*indexMetaData {
	var res []*indexMetaData

	switch node := in.(type) {
	case *ast.CreateTableStmt:
		for _, constraint := range node.Constraints {
			if c, ok := ruleMapping[checker.ruleType]; !ok || c != constraint.Tp {
				continue
			}
			switch constraint.Tp {
			case ast.ConstraintIndex, ast.ConstraintPrimaryKey, ast.ConstraintUniq:
				var columnList []string
				for _, key := range constraint.Keys {
					columnList = append(columnList, key.Column.Name.String())
				}
				metaData := map[string]string{
					api.ColumnListTemplateToken: strings.Join(columnList, "_"),
					api.TableNameTemplateToken:  node.Table.Name.String(),
				}
				res = append(res, &indexMetaData{
					index:    constraint.Name,
					metaData: metaData,
				})
			case ast.ConstraintForeignKey:
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
					index:    constraint.Name,
					metaData: metaData,
				})
			}
		}
	case *ast.AlterTableStmt:
		for _, spec := range node.Specs {
			if c, ok := ruleMapping[checker.ruleType]; !ok || c != spec.Constraint.Tp {
				continue
			}
			switch spec.Tp {
			case ast.AlterTableRenameIndex:
				// TODO: how to get the releated column list through old index name
				metaData := map[string]string{
					api.ColumnListTemplateToken: "*.",
					api.TableNameTemplateToken:  node.Table.Name.String(),
				}
				res = append(res, &indexMetaData{
					index:    spec.ToKey.String(),
					metaData: metaData,
				})
			case ast.AlterTableAddConstraint:
				switch spec.Constraint.Tp {
				case ast.ConstraintIndex, ast.ConstraintPrimaryKey, ast.ConstraintUniq:
					var columnList []string
					for _, key := range spec.Constraint.Keys {
						columnList = append(columnList, key.Column.Name.String())
					}

					metaData := map[string]string{
						api.ColumnListTemplateToken: strings.Join(columnList, "_"),
						api.TableNameTemplateToken:  node.Table.Name.String(),
					}
					res = append(res, &indexMetaData{
						index:    spec.Constraint.Name,
						metaData: metaData,
					})
				case ast.ConstraintForeignKey:
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
						index:    spec.Constraint.Name,
						metaData: metaData,
					})
				}
			}
		}
	case *ast.CreateIndexStmt:
		if (node.KeyType == ast.IndexKeyTypeUnique && checker.ruleType == api.SchemaRuleUKNaming) ||
			(node.KeyType != ast.IndexKeyTypeUnique && checker.ruleType != api.SchemaRuleUKNaming) {
			var columnList []string
			for _, spec := range node.IndexPartSpecifications {
				columnList = append(columnList, spec.Column.Name.String())
			}
			metaData := map[string]string{
				api.ColumnListTemplateToken: strings.Join(columnList, "_"),
				api.TableNameTemplateToken:  node.Table.Name.String(),
			}
			res = append(res, &indexMetaData{
				index:    node.IndexName,
				metaData: metaData,
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
