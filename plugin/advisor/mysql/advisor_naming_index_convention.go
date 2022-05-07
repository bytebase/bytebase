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
	_ advisor.Advisor = (*NamingIndexConventionAdvisor)(nil)

	ruleMapping = map[api.SchemaReviewRuleType]ast.ConstraintType{
		api.SchemaRulePKNaming:  ast.ConstraintPrimaryKey,
		api.SchemaRuleIDXNaming: ast.ConstraintIndex,
		api.SchemaRuleUKNaming:  ast.ConstraintUniqIndex,
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
func (adv *NamingIndexConventionAdvisor) Check(ctx advisor.Context, statement string) ([]advisor.Advice, error) {
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

	if len(checker.advisorList) == 0 {
		checker.advisorList = append(checker.advisorList, advisor.Advice{
			Status:  advisor.Success,
			Code:    common.Ok,
			Title:   "OK",
			Content: "",
		})
	}
	return checker.advisorList, nil
}

type namingIndexConventionChecker struct {
	advisorList  []advisor.Advice
	level        advisor.Status
	ruleType     api.SchemaReviewRuleType
	format       string
	templateList []string
}

func (v *namingIndexConventionChecker) Enter(in ast.Node) (ast.Node, bool) {
	indexDataList := v.getIndexMetaData(in)

	code := common.Ok

	for _, indexData := range indexDataList {
		template := formatTemplate(v.format, v.templateList, indexData.metaData)
		if template != indexData.index {
			code = common.IndexNamingConventionMismatch
			break
		}
	}

	if code != common.Ok {
		v.advisorList = append(v.advisorList, advisor.Advice{
			Status:  v.level,
			Code:    code,
			Title:   "Mismatch index naming convention",
			Content: fmt.Sprintf("%q mismatches index naming convention", in.Text()),
		})
	}

	return in, false
}

func (v *namingIndexConventionChecker) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}

type indexMetaData struct {
	index    string
	metaData map[string]string
}

// getIndexMetaData returns the list of index with meta data.
func (v *namingIndexConventionChecker) getIndexMetaData(in ast.Node) []*indexMetaData {
	var res []*indexMetaData

	switch node := in.(type) {
	case *ast.CreateTableStmt:
		for _, constraint := range node.Constraints {
			switch constraint.Tp {
			case ast.ConstraintIndex, ast.ConstraintPrimaryKey, ast.ConstraintUniqIndex:
				if c, ok := ruleMapping[v.ruleType]; !ok || c != constraint.Tp {
					continue
				}
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
			}
		}
	case *ast.AlterTableStmt:
		for _, spec := range node.Specs {
			if spec.Tp == ast.AlterTableAddConstraint {
				switch spec.Constraint.Tp {
				case ast.ConstraintIndex, ast.ConstraintPrimaryKey, ast.ConstraintUniqIndex:
					if c, ok := ruleMapping[v.ruleType]; !ok || c != spec.Constraint.Tp {
						continue
					}
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
				}
			}
		}
	case *ast.CreateIndexStmt:
		if (node.KeyType == ast.IndexKeyTypeUnique && v.ruleType == api.SchemaRuleUKNaming) ||
			(node.KeyType != ast.IndexKeyTypeUnique && v.ruleType != api.SchemaRuleUKNaming) {
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

// formatTemplate formats the template.
func formatTemplate(template string, templateList []string, tokens map[string]string) string {
	for _, key := range templateList {
		if token, ok := tokens[key]; ok {
			template = strings.ReplaceAll(template, key, token)
		}
	}
	return template
}
