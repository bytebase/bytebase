package tidb

import (
	"context"
	"fmt"
	"regexp"

	"github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*NamingColumnConventionAdvisor)(nil)
	_ ast.Visitor     = (*namingColumnConventionChecker)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, advisor.SchemaRuleColumnNaming, &NamingColumnConventionAdvisor{})
}

// NamingColumnConventionAdvisor is the advisor checking for column naming convention.
type NamingColumnConventionAdvisor struct {
}

// Check checks for column naming convention.
func (*NamingColumnConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	root, ok := checkCtx.AST.([]ast.StmtNode)
	if !ok {
		return nil, errors.Errorf("failed to convert to StmtNode")
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

	for _, stmtNode := range root {
		(stmtNode).Accept(checker)
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
			default:
				// Skip other alter table specification types
			}
		}
	}

	for _, column := range columnList {
		if !v.format.MatchString(column.name) {
			v.adviceList = append(v.adviceList, &storepb.Advice{
				Status:        v.level,
				Code:          code.NamingColumnConventionMismatch.Int32(),
				Title:         v.title,
				Content:       fmt.Sprintf("`%s`.`%s` mismatches column naming convention, naming format should be %q", tableName, column.name, v.format),
				StartPosition: common.ConvertANTLRLineToPosition(column.line),
			})
		}
		if v.maxLength > 0 && len(column.name) > v.maxLength {
			v.adviceList = append(v.adviceList, &storepb.Advice{
				Status:        v.level,
				Code:          code.NamingColumnConventionMismatch.Int32(),
				Title:         v.title,
				Content:       fmt.Sprintf("`%s`.`%s` mismatches column naming convention, its length should be within %d characters", tableName, column.name, v.maxLength),
				StartPosition: common.ConvertANTLRLineToPosition(column.line),
			})
		}
	}

	return in, false
}

// Leave implements the ast.Visitor interface.
func (*namingColumnConventionChecker) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}
