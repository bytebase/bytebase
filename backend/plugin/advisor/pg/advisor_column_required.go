package pg

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/pg/legacy/ast"
)

var (
	_ advisor.Advisor = (*ColumnRequirementAdvisor)(nil)
	_ ast.Visitor     = (*columnRequirementChecker)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleRequiredColumn, &ColumnRequirementAdvisor{})
}

type columnSet map[string]bool

// ColumnRequirementAdvisor is the advisor checking for column requirement.
type ColumnRequirementAdvisor struct {
}

// Check checks for the column requirement.
func (*ColumnRequirementAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, ok := checkCtx.AST.([]ast.Node)
	if !ok {
		return nil, errors.Errorf("failed to convert to Node")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	columnList, err := advisor.UnmarshalRequiredColumnList(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	checker := &columnRequirementChecker{
		level: level,
		title: string(checkCtx.Rule.Type),
	}

	for _, stmt := range stmts {
		checker.requiredColumns = make(columnSet)
		for _, column := range columnList {
			checker.requiredColumns[column] = true
		}
		ast.Walk(checker, stmt)
	}

	return checker.adviceList, nil
}

type columnRequirementChecker struct {
	adviceList      []*storepb.Advice
	level           storepb.Advice_Status
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
		slices.Sort(missingColumns)
		checker.adviceList = append(checker.adviceList, &storepb.Advice{
			Status:        checker.level,
			Code:          advisor.NoRequiredColumn.Int32(),
			Title:         checker.title,
			Content:       fmt.Sprintf("Table %q requires columns: %s", table.Name, strings.Join(missingColumns, ", ")),
			StartPosition: common.ConvertPGParserLineToPosition(node.LastLine()),
		})
	}

	return checker
}
