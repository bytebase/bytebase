package pg

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/pg/legacy/ast"
)

var (
	_ advisor.Advisor = (*NamingFKConventionAdvisor)(nil)
	_ ast.Visitor     = (*namingFKConventionChecker)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.PostgreSQLNamingFKConvention, &NamingFKConventionAdvisor{})
}

// NamingFKConventionAdvisor is the advisor checking for foreign key naming convention.
type NamingFKConventionAdvisor struct {
}

// Check checks for foreign key naming convention.
func (*NamingFKConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	root, ok := checkCtx.AST.([]ast.Node)
	if !ok {
		return nil, errors.Errorf("failed to convert to Node")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	format, templateList, maxLength, err := advisor.UnmarshalNamingRulePayloadAsTemplate(advisor.SQLReviewRuleType(checkCtx.Rule.Type), checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	checker := &namingFKConventionChecker{
		level:        level,
		title:        string(checkCtx.Rule.Type),
		format:       format,
		maxLength:    maxLength,
		templateList: templateList,
	}

	for _, stmtNode := range root {
		ast.Walk(checker, stmtNode)
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

type indexMetaData struct {
	indexName string
	tableName string
	line      int
	metaData  map[string]string
}

// Visit implements ast.Visitor interface.
func (checker *namingFKConventionChecker) Visit(in ast.Node) ast.Visitor {
	indexDataList := checker.getMetaDataList(in)

	for _, indexData := range indexDataList {
		regex, err := getTemplateRegexp(checker.format, checker.templateList, indexData.metaData)
		if err != nil {
			checker.adviceList = append(checker.adviceList, &storepb.Advice{
				Status:  checker.level,
				Code:    advisor.Internal.Int32(),
				Title:   "Internal error for foreign key naming convention rule",
				Content: fmt.Sprintf("%q meet internal error %q", in.Text(), err.Error()),
			})
			continue
		}
		if !regex.MatchString(indexData.indexName) {
			checker.adviceList = append(checker.adviceList, &storepb.Advice{
				Status:        checker.level,
				Code:          advisor.NamingFKConventionMismatch.Int32(),
				Title:         checker.title,
				Content:       fmt.Sprintf(`Foreign key in table "%s" mismatches the naming convention, expect %q but found "%s"`, indexData.tableName, regex, indexData.indexName),
				StartPosition: common.ConvertPGParserLineToPosition(indexData.line),
			})
		}
		if checker.maxLength > 0 && len(indexData.indexName) > checker.maxLength {
			checker.adviceList = append(checker.adviceList, &storepb.Advice{
				Status:        checker.level,
				Code:          advisor.NamingFKConventionMismatch.Int32(),
				Title:         checker.title,
				Content:       fmt.Sprintf(`Foreign key "%s" in table "%s" mismatches the naming convention, its length should be within %d characters`, indexData.indexName, indexData.tableName, checker.maxLength),
				StartPosition: common.ConvertPGParserLineToPosition(indexData.line),
			})
		}
	}

	return checker
}

// getMetaDataList returns the list of foreign key with metadata.
func (*namingFKConventionChecker) getMetaDataList(in ast.Node) []*indexMetaData {
	var res []*indexMetaData
	switch node := in.(type) {
	case *ast.CreateTableStmt:
		for _, constraint := range node.ConstraintList {
			if metadata := getForeignKeyMetadata(constraint, node.Name.Name, constraint.LastLine()); metadata != nil {
				res = append(res, metadata)
			}
		}
		for _, column := range node.ColumnList {
			for _, constraint := range column.ConstraintList {
				if metadata := getForeignKeyMetadata(constraint, node.Name.Name, column.LastLine()); metadata != nil {
					res = append(res, metadata)
				}
			}
		}
	case *ast.AddConstraintStmt:
		constraint := node.Constraint
		if metadata := getForeignKeyMetadata(constraint, node.Table.Name, node.LastLine()); metadata != nil {
			res = append(res, metadata)
		}
	case *ast.AddColumnListStmt:
		for _, column := range node.ColumnList {
			for _, constraint := range column.ConstraintList {
				if metadata := getForeignKeyMetadata(constraint, node.Table.Name, node.LastLine()); metadata != nil {
					res = append(res, metadata)
				}
			}
		}
	}
	return res
}

// getForeignKeyMetadata returns index metadata of a foreign key constraint, nil if other constraints.
func getForeignKeyMetadata(constraint *ast.ConstraintDef, tableName string, line int) *indexMetaData {
	if constraint.Type == ast.ConstraintTypeForeign {
		referencingColumnList := constraint.KeyList
		referencedColumnList := constraint.Foreign.ColumnList

		metaData := map[string]string{
			advisor.ReferencingTableNameTemplateToken:  tableName,
			advisor.ReferencingColumnNameTemplateToken: strings.Join(referencingColumnList, "_"),
			advisor.ReferencedTableNameTemplateToken:   constraint.Foreign.Table.Name,
			advisor.ReferencedColumnNameTemplateToken:  strings.Join(referencedColumnList, "_"),
		}

		return &indexMetaData{
			indexName: constraint.Name,
			tableName: tableName,
			line:      line,
			metaData:  metaData,
		}
	}
	return nil
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
