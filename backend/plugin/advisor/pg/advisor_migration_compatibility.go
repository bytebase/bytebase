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
func (*CompatibilityAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, ok := checkCtx.AST.([]ast.Node)
	if !ok {
		return nil, errors.Errorf("failed to convert to Node")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &compatibilityChecker{
		level: level,
		title: string(checkCtx.Rule.Type),
	}
	for _, stmt := range stmts {
		checker.text = stmt.Text()
		ast.Walk(checker, stmt)
	}

	return checker.adviceList, nil
}

type compatibilityChecker struct {
	adviceList      []*storepb.Advice
	level           storepb.Advice_Status
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
			default:
				// Other constraint types
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
		checker.adviceList = append(checker.adviceList, &storepb.Advice{
			Status:        checker.level,
			Code:          code.Int32(),
			Title:         checker.title,
			Content:       fmt.Sprintf("\"%s\" may cause incompatibility with the existing data and code", checker.text),
			StartPosition: common.ConvertPGParserLineToPosition(node.LastLine()),
		})
	}
	return checker
}
