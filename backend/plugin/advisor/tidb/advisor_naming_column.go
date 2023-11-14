package tidb

import (
	"fmt"
	"regexp"

	"github.com/pingcap/tidb/parser/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*NamingColumnConventionAdvisor)(nil)
	_ ast.Visitor     = (*namingColumnConventionChecker)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.MySQLNamingColumnConvention, &NamingColumnConventionAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.MySQLNamingColumnConvention, &NamingColumnConventionAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.MySQLNamingColumnConvention, &NamingColumnConventionAdvisor{})
	advisor.Register(storepb.Engine_TIDB, advisor.MySQLNamingColumnConvention, &NamingColumnConventionAdvisor{})
}

// NamingColumnConventionAdvisor is the advisor checking for column naming convention.
type NamingColumnConventionAdvisor struct {
}

// Check checks for column naming convention.
func (*NamingColumnConventionAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	root, ok := ctx.AST.([]ast.StmtNode)
	if !ok {
		return nil, errors.Errorf("failed to convert to StmtNode")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	format, maxLength, err := advisor.UnmarshalNamingRulePayloadAsRegexp(ctx.Rule.Payload)
	if err != nil {
		return nil, err
	}
	checker := &namingColumnConventionChecker{
		level:     level,
		title:     string(ctx.Rule.Type),
		format:    format,
		maxLength: maxLength,
		tables:    make(tableState),
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

type namingColumnConventionChecker struct {
	adviceList []advisor.Advice
	level      advisor.Status
	title      string
	format     *regexp.Regexp
	maxLength  int
	tables     tableState
}

// Enter implements the ast.Visitor interface.
func (v *namingColumnConventionChecker) Enter(in ast.Node) (ast.Node, bool) {
	type columnData struct {
		name string
		line int
	}
	var columnList []columnData
	var tableName string
	switch node := in.(type) {
	// CREATE TABLE
	case *ast.CreateTableStmt:
		tableName = node.Table.Name.O
		for _, column := range node.Cols {
			columnList = append(columnList, columnData{
				name: column.Name.Name.O,
				line: column.OriginTextPosition(),
			})
		}
	// ALTER TABLE
	case *ast.AlterTableStmt:
		tableName = node.Table.Name.O
		for _, spec := range node.Specs {
			switch spec.Tp {
			// RENAME COLUMN
			case ast.AlterTableRenameColumn:
				columnList = append(columnList, columnData{
					name: spec.NewColumnName.Name.O,
					line: in.OriginTextPosition(),
				})
			// ADD COLUMNS
			case ast.AlterTableAddColumns:
				for _, column := range spec.NewColumns {
					columnList = append(columnList, columnData{
						name: column.Name.Name.O,
						line: in.OriginTextPosition(),
					})
				}
			// CHANGE COLUMN
			case ast.AlterTableChangeColumn:
				columnList = append(columnList, columnData{
					name: spec.NewColumns[0].Name.Name.O,
					line: in.OriginTextPosition(),
				})
			}
		}
	}

	for _, column := range columnList {
		if !v.format.MatchString(column.name) {
			v.adviceList = append(v.adviceList, advisor.Advice{
				Status:  v.level,
				Code:    advisor.NamingColumnConventionMismatch,
				Title:   v.title,
				Content: fmt.Sprintf("`%s`.`%s` mismatches column naming convention, naming format should be %q", tableName, column.name, v.format),
				Line:    column.line,
			})
		}
		if v.maxLength > 0 && len(column.name) > v.maxLength {
			v.adviceList = append(v.adviceList, advisor.Advice{
				Status:  v.level,
				Code:    advisor.NamingColumnConventionMismatch,
				Title:   v.title,
				Content: fmt.Sprintf("`%s`.`%s` mismatches column naming convention, its length should be within %d characters", tableName, column.name, v.maxLength),
				Line:    column.line,
			})
		}
	}

	return in, false
}

// Leave implements the ast.Visitor interface.
func (*namingColumnConventionChecker) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}
