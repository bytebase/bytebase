package pg

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/pg/legacy/ast"
)

var (
	_ advisor.Advisor = (*TableNoFKAdvisor)(nil)
	_ ast.Visitor     = (*tableNoFKChecker)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.PostgreSQLTableNoFK, &TableNoFKAdvisor{})
}

// TableNoFKAdvisor is the advisor checking table disallow foreign key.
type TableNoFKAdvisor struct {
}

// Check checks table disallow foreign key.
func (*TableNoFKAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, ok := checkCtx.AST.([]ast.Node)
	if !ok {
		return nil, errors.Errorf("failed to convert to Node")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &tableNoFKChecker{
		level: level,
		title: string(checkCtx.Rule.Type),
	}

	for _, stmt := range stmts {
		checker.text = stmt.Text()
		ast.Walk(checker, stmt)
	}

	return checker.adviceList, nil
}

type tableNoFKChecker struct {
	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
	text       string
}

// Visit implements the ast.Visitor interface.
func (checker *tableNoFKChecker) Visit(node ast.Node) ast.Visitor {
	var tableHasFK *ast.TableDef
	switch n := node.(type) {
	// CREATE TABLE
	case *ast.CreateTableStmt:
		for _, column := range n.ColumnList {
			if containFK(column.ConstraintList) {
				tableHasFK = n.Name
				break
			}
		}
		if containFK(n.ConstraintList) {
			tableHasFK = n.Name
		}
	// ADD CONSTRAINT
	case *ast.AddConstraintStmt:
		if n.Constraint.Type == ast.ConstraintTypeForeign {
			tableHasFK = n.Table
		}
	}

	if tableHasFK != nil {
		checker.adviceList = append(checker.adviceList, &storepb.Advice{
			Status: checker.level,
			Code:   advisor.TableHasFK.Int32(),
			Title:  checker.title,
			Content: fmt.Sprintf("Foreign key is not allowed in the table %q.%q, related statement: \"%s\"",
				normalizeSchemaName(tableHasFK.Schema),
				tableHasFK.Name,
				checker.text,
			),
			StartPosition: common.ConvertPGParserLineToPosition(node.LastLine()),
		})
	}

	return checker
}

func containFK(list []*ast.ConstraintDef) bool {
	for _, cons := range list {
		if cons.Type == ast.ConstraintTypeForeign {
			return true
		}
	}
	return false
}
