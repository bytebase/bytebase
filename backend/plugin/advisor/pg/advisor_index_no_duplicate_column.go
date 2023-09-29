package pg

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.PostgreSQLIndexNoDuplicateColumn, &IndexNoDuplicateColumnAdvisor{})
}

// IndexNoDuplicateColumnAdvisor is the advisor checking for no duplicate columns in index.
type IndexNoDuplicateColumnAdvisor struct {
}

// Check checks for no duplicate columns in index.
func (*IndexNoDuplicateColumnAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	stmtList, ok := ctx.AST.([]ast.Node)
	if !ok {
		return nil, errors.Errorf("failed to convert to Node")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	checker := &indexNoDuplicateColumnChecker{
		level: level,
		title: string(ctx.Rule.Type),
	}

	for _, stmt := range stmtList {
		checker.text = stmt.Text()
		checker.line = stmt.LastLine()
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

type indexNoDuplicateColumnChecker struct {
	adviceList []advisor.Advice
	level      advisor.Status
	title      string
	text       string
	line       int
}

type duplicateColumn struct {
	table          string
	index          string
	column         string
	constraintType string
	line           int
}

func (checker *indexNoDuplicateColumnChecker) Visit(node ast.Node) ast.Visitor {
	var columnList []duplicateColumn
	switch node := node.(type) {
	case *ast.CreateTableStmt:
		for _, constraint := range node.ConstraintList {
			switch constraint.Type {
			case ast.ConstraintTypePrimary,
				ast.ConstraintTypeForeign,
				ast.ConstraintTypePrimaryUsingIndex,
				ast.ConstraintTypeUnique,
				ast.ConstraintTypeUniqueUsingIndex:
				if column, duplicate := hasDuplicateColumn(constraint.KeyList); duplicate {
					columnList = append(columnList, duplicateColumn{
						table:          node.Name.Name,
						index:          constraint.Name,
						column:         column,
						constraintType: contraintsTypeToString(constraint.Type),
						line:           checker.line,
					})
				}
			}
		}
	case *ast.CreateIndexStmt:
		if column, duplicate := hasDuplicateColumn(node.Index.GetKeyNameList()); duplicate {
			columnList = append(columnList, duplicateColumn{
				table:          node.Index.Table.Name,
				index:          node.Index.Name,
				column:         column,
				constraintType: "INDEX",
				line:           checker.line,
			})
		}
	case *ast.AlterTableStmt:
		for _, item := range node.AlterItemList {
			switch cmd := item.(type) {
			case *ast.AddConstraintStmt:
				switch cmd.Constraint.Type {
				case ast.ConstraintTypePrimary,
					ast.ConstraintTypeForeign,
					ast.ConstraintTypePrimaryUsingIndex,
					ast.ConstraintTypeUnique,
					ast.ConstraintTypeUniqueUsingIndex:
					if column, duplicate := hasDuplicateColumn(cmd.Constraint.KeyList); duplicate {
						columnList = append(columnList, duplicateColumn{
							table:          cmd.Table.Name,
							index:          cmd.Constraint.Name,
							column:         column,
							constraintType: contraintsTypeToString(cmd.Constraint.Type),
							line:           checker.line,
						})
					}
				}
			default:
				continue
			}
		}
	}

	for _, column := range columnList {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  checker.level,
			Code:    advisor.DuplicateColumnInIndex,
			Title:   checker.title,
			Content: fmt.Sprintf("%s \"%s\" has duplicate column \"%s\".\"%s\"", column.constraintType, column.index, column.table, column.column),
			Line:    column.line,
		})
	}

	return checker
}

func hasDuplicateColumn(keyList []string) (string, bool) {
	existMap := make(map[string]bool)
	for _, key := range keyList {
		if _, isExist := existMap[key]; isExist {
			return key, true
		}
		existMap[key] = true
	}
	return "", false
}

func contraintsTypeToString(constrainType ast.ConstraintType) string {
	switch constrainType {
	case ast.ConstraintTypePrimary, ast.ConstraintTypePrimaryUsingIndex:
		return "PRIMARY KEY"
	case ast.ConstraintTypeUnique, ast.ConstraintTypeUniqueUsingIndex:
		return "UNIQUE KEY"
	case ast.ConstraintTypeForeign:
		return "FOREIGN KEY"
	}
	return "INDEX"
}
