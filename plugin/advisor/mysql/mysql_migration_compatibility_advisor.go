package mysql

import (
	"fmt"

	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/bytebase/bytebase/plugin/db"

	"github.com/pingcap/parser"
	"github.com/pingcap/parser/ast"
	_ "github.com/pingcap/tidb/types/parser_driver"
)

var (
	_ advisor.Advisor = (*CompatibilityAdvisor)(nil)
)

func init() {
	advisor.Register(db.MySQL, advisor.MySQLMigrationCompatibility, &CompatibilityAdvisor{})
	advisor.Register(db.TiDB, advisor.MySQLMigrationCompatibility, &CompatibilityAdvisor{})
}

type CompatibilityAdvisor struct {
}

// A fake advisor to report 1 advice for each severity.
func (adv *CompatibilityAdvisor) Check(ctx advisor.AdvisorContext, statement string) ([]advisor.Advice, error) {
	p := parser.New()

	root, _, err := p.Parse(statement, ctx.Charset, ctx.Collation)
	if err != nil {
		return []advisor.Advice{
			{
				Status:  advisor.Error,
				Title:   "Syntax error",
				Content: err.Error(),
			},
		}, nil
	}

	c := &compatibilityChecker{}
	for _, stmtNode := range root {
		fmt.Printf("%+v\n", stmtNode)
		(stmtNode).Accept(c)
		fmt.Printf("%+v\n", c)
	}

	if len(c.advisorList) == 0 {
		c.advisorList = append(c.advisorList, advisor.Advice{
			Status:  advisor.Success,
			Title:   "OK",
			Content: "Migration is backward compatible"})
	}
	return c.advisorList, nil
}

type compatibilityChecker struct {
	advisorList []advisor.Advice
}

func (v *compatibilityChecker) Enter(in ast.Node) (ast.Node, bool) {
	compatible := true
	// DROP DATABASE
	if _, ok := in.(*ast.DropDatabaseStmt); ok {
		compatible = false
		goto END
	}
	// RENAME TABLE
	if _, ok := in.(*ast.RenameTableStmt); ok {
		compatible = false
		goto END
	}
	// DROP TABLE/VIEW
	if _, ok := in.(*ast.DropTableStmt); ok {
		compatible = false
		goto END
	}
	// ALTER TABLE
	if node, ok := in.(*ast.AlterTableStmt); ok {
		for _, spec := range node.Specs {
			fmt.Printf("spec %d: %+v\n\n", spec.Tp, spec)
			// RENAME COLUMN
			// DROP COLUMN
			if spec.Tp == ast.AlterTableRenameColumn ||
				spec.Tp == ast.AlterTableDropColumn {
				compatible = false
				goto END
			}

			if spec.Tp == ast.AlterTableAddConstraint {
				// ADD PRIMARY KEY
				// ADD UNIQUE/UNIQUE KEY/UNIQUE INDEX
				// ADD FOREIGN KEY
				if spec.Constraint.Tp == ast.ConstraintPrimaryKey ||
					spec.Constraint.Tp == ast.ConstraintUniq ||
					spec.Constraint.Tp == ast.ConstraintUniqKey ||
					spec.Constraint.Tp == ast.ConstraintUniqIndex ||
					spec.Constraint.Tp == ast.ConstraintForeignKey {
					compatible = false
					goto END
				}
				// Check is only supported after 8.0.16 https://dev.mysql.com/doc/refman/8.0/en/create-table-check-constraints.html
				// ADD CHECK ENFORCED
				if spec.Constraint.Tp == ast.ConstraintCheck && spec.Constraint.Enforced {
					compatible = false
					goto END
				}
			}

			// Check is only supported after 8.0.16 https://dev.mysql.com/doc/refman/8.0/en/create-table-check-constraints.html
			// ALTER CHECK ENFORCED
			if spec.Tp == ast.AlterTableAlterCheck {
				if spec.Constraint.Enforced {
					compatible = false
					goto END
				}
			}
		}
	}
	// ALTER VIEW TBD: https://github.com/pingcap/parser/pull/1252
	// if node, ok := in.(*ast.AlterViewStmt); ok {
	// }

	// CREATE UNIQUE INDEX
	if node, ok := in.(*ast.CreateIndexStmt); ok {
		if node.KeyType == ast.IndexKeyTypeUnique {
			compatible = false
			goto END
		}
	}

END:
	if !compatible {
		v.advisorList = append(v.advisorList, advisor.Advice{
			Status:  advisor.Warn,
			Title:   "Incompatible migration",
			Content: fmt.Sprintf("%s is backward incompatible", in.Text()),
		})
	}
	return in, false
}

func (v *compatibilityChecker) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}
