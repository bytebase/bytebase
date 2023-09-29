package pg

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*ColumnRequirementAdvisor)(nil)
	_ ast.Visitor     = (*columnRequirementChecker)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.PostgreSQLColumnRequirement, &ColumnRequirementAdvisor{})
}

type columnSet map[string]bool

// ColumnRequirementAdvisor is the advisor checking for column requirement.
type ColumnRequirementAdvisor struct {
}

// Check checks for the column requirement.
func (*ColumnRequirementAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	stmts, ok := ctx.AST.([]ast.Node)
	if !ok {
		return nil, errors.Errorf("failed to convert to Node")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	columnList, err := advisor.UnmarshalRequiredColumnList(ctx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	checker := &columnRequirementChecker{
		level: level,
		title: string(ctx.Rule.Type),
	}

	for _, stmt := range stmts {
		checker.requiredColumns = make(columnSet)
		for _, column := range columnList {
			checker.requiredColumns[column] = true
		}
		ast.Walk(checker, stmt)
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

type columnRequirementChecker struct {
	adviceList      []advisor.Advice
	level           advisor.Status
	title           string
	requiredColumns columnSet
}

// Visit implements the ast.Visitor interface.
func (checker *columnRequirementChecker) Visit(node ast.Node) ast.Visitor {
	var table *ast.TableDef
	var missingColumns []string
	switch n := node.(type) {
	// CREATE TABLE
	case *ast.CreateTableStmt:
		for _, column := range n.ColumnList {
			delete(checker.requiredColumns, column.ColumnName)
		}
		if len(checker.requiredColumns) > 0 {
			table = n.Name
			for column := range checker.requiredColumns {
				missingColumns = append(missingColumns, column)
			}
		}
	// ALTER TABLE DROP COLUMN
	case *ast.DropColumnStmt:
		if _, yes := checker.requiredColumns[n.ColumnName]; yes {
			table = n.Table
			missingColumns = append(missingColumns, n.ColumnName)
		}
	// ALTER TABLE RENAME COLUMN
	case *ast.RenameColumnStmt:
		if _, yes := checker.requiredColumns[n.ColumnName]; yes && n.ColumnName != n.NewName {
			table = n.Table
			missingColumns = append(missingColumns, n.ColumnName)
		}
	}

	if len(missingColumns) > 0 {
		// Order it cause the random iteration order in Go, see https://go.dev/blog/maps
		sort.Strings(missingColumns)
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  checker.level,
			Code:    advisor.NoRequiredColumn,
			Title:   checker.title,
			Content: fmt.Sprintf("Table %q requires columns: %s", table.Name, strings.Join(missingColumns, ", ")),
			Line:    node.LastLine(),
		})
	}

	return checker
}
