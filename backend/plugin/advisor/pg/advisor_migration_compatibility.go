package pg

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*CompatibilityAdvisor)(nil)
	_ ast.Visitor     = (*compatibilityChecker)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.PostgreSQLMigrationCompatibility, &CompatibilityAdvisor{})
}

// CompatibilityAdvisor is the advisor checking for schema backward compatibility.
type CompatibilityAdvisor struct {
}

// Check checks schema backward compatibility.
func (*CompatibilityAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	stmts, ok := ctx.AST.([]ast.Node)
	if !ok {
		return nil, errors.Errorf("failed to convert to Node")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &compatibilityChecker{
		level: level,
		title: string(ctx.Rule.Type),
	}
	for _, stmt := range stmts {
		checker.text = stmt.Text()
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

type compatibilityChecker struct {
	adviceList      []advisor.Advice
	level           advisor.Status
	title           string
	text            string
	lastCreateTable string
}

func (checker *compatibilityChecker) Visit(node ast.Node) ast.Visitor {
	code := advisor.Ok
	switch n := node.(type) {
	// CREATE TABLE
	case *ast.CreateTableStmt:
		checker.lastCreateTable = n.Name.Name
	// DROP DATABASE
	case *ast.DropDatabaseStmt:
		code = advisor.CompatibilityDropDatabase
	// RENAME TABLE/VIEW
	case *ast.RenameTableStmt:
		code = advisor.CompatibilityRenameTable
	// DROP TABLE/VIEW
	case *ast.DropTableStmt:
		code = advisor.CompatibilityDropTable
	// ALTER TABLE RENAME COLUMN
	case *ast.RenameColumnStmt:
		if checker.lastCreateTable != n.Table.Name {
			code = advisor.CompatibilityRenameColumn
		}
	// ALTER TABLE DROP COLUMN
	case *ast.DropColumnStmt:
		if checker.lastCreateTable != n.Table.Name {
			code = advisor.CompatibilityDropColumn
		}
	case *ast.AddConstraintStmt:
		if checker.lastCreateTable != n.Table.Name {
			switch n.Constraint.Type {
			// ADD PRIMARY KEY/ ADD PRIMARY KEY USING INDEX
			case ast.ConstraintTypePrimary, ast.ConstraintTypePrimaryUsingIndex:
				code = advisor.CompatibilityAddPrimaryKey
			// ADD UNIQUE CONSTRAINT
			case ast.ConstraintTypeUnique:
				// ConstraintTypeUniqueUsingIndex doesn't add a new constraint or unique index.
				code = advisor.CompatibilityAddUniqueKey
			// ADD FOREIGIN KEY
			case ast.ConstraintTypeForeign:
				code = advisor.CompatibilityAddForeignKey
			// ADD CHECK
			case ast.ConstraintTypeCheck:
				if !n.Constraint.SkipValidation {
					code = advisor.CompatibilityAddCheck
				}
			}
		}
	// ALTER TABLE ALTER COLUMN TYPE
	case *ast.AlterColumnTypeStmt:
		if checker.lastCreateTable != n.Table.Name {
			code = advisor.CompatibilityAlterColumn
		}
	// CREATE UNIQUE INDEX
	case *ast.CreateIndexStmt:
		if checker.lastCreateTable != n.Index.Table.Name {
			if n.Index.Unique {
				code = advisor.CompatibilityAddUniqueKey
			}
		}
	}

	if code != advisor.Ok {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  checker.level,
			Code:    code,
			Title:   checker.title,
			Content: fmt.Sprintf("\"%s\" may cause incompatibility with the existing data and code", checker.text),
			Line:    node.LastLine(),
		})
	}
	return checker
}
