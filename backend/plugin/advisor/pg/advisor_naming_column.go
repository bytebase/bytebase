package pg

import (
	"context"
	"fmt"
	"regexp"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/pg/legacy/ast"
)

var (
	_ advisor.Advisor = (*NamingColumnConventionAdvisor)(nil)
	_ ast.Visitor     = (*namingColumnConventionChecker)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleColumnNaming, &NamingColumnConventionAdvisor{})
}

// NamingColumnConventionAdvisor is the advisor checking for column convention.
type NamingColumnConventionAdvisor struct {
}

// Check checks for column naming convention.
func (*NamingColumnConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, ok := checkCtx.AST.([]ast.Node)
	if !ok {
		return nil, errors.Errorf("failed to convert to Node")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	format, maxLength, err := advisor.UnmarshalNamingRulePayloadAsRegexp(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}
	checker := &namingColumnConventionChecker{
		level:     level,
		title:     string(checkCtx.Rule.Type),
		format:    format,
		maxLength: maxLength,
	}

	for _, stmt := range stmts {
		ast.Walk(checker, stmt)
	}

	return checker.adviceList, nil
}

type namingColumnConventionChecker struct {
	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
	format     *regexp.Regexp
	maxLength  int
}

// Visit implements the ast.Visitor interface.
func (checker *namingColumnConventionChecker) Visit(node ast.Node) ast.Visitor {
	type columnData struct {
		name string
		line int
	}
	var columnList []columnData
	var tableName string

	switch n := node.(type) {
	// CREATE TABLE
	case *ast.CreateTableStmt:
		tableName = n.Name.Name
		for _, col := range n.ColumnList {
			columnList = append(columnList, columnData{
				name: col.ColumnName,
				line: col.LastLine(),
			})
		}
	// ALTER TABLE ADD COLUMN
	case *ast.AddColumnListStmt:
		tableName = n.Table.Name
		for _, col := range n.ColumnList {
			columnList = append(columnList, columnData{
				name: col.ColumnName,
				line: n.LastLine(),
			})
		}
	// ALTER TABLE RENAME COLUMN
	case *ast.RenameColumnStmt:
		tableName = n.Table.Name
		columnList = append(columnList, columnData{
			name: n.NewName,
			line: n.LastLine(),
		})
	}

	for _, column := range columnList {
		if !checker.format.MatchString(column.name) {
			checker.adviceList = append(checker.adviceList, &storepb.Advice{
				Status:        checker.level,
				Code:          advisor.NamingColumnConventionMismatch.Int32(),
				Title:         checker.title,
				Content:       fmt.Sprintf("\"%s\".\"%s\" mismatches column naming convention, naming format should be %q", tableName, column.name, checker.format),
				StartPosition: common.ConvertPGParserLineToPosition(column.line),
			})
		}

		if checker.maxLength > 0 && len(column.name) > checker.maxLength {
			checker.adviceList = append(checker.adviceList, &storepb.Advice{
				Status:        checker.level,
				Code:          advisor.NamingColumnConventionMismatch.Int32(),
				Title:         checker.title,
				Content:       fmt.Sprintf("\"%s\".\"%s\" mismatches column naming convention, its length should be within %d characters", tableName, column.name, checker.maxLength),
				StartPosition: common.ConvertPGParserLineToPosition(column.line),
			})
		}
	}

	return checker
}
