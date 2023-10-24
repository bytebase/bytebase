package mysql

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"

	"github.com/pingcap/tidb/parser/ast"
)

var (
	_ advisor.Advisor = (*CompatibilityAdvisor)(nil)
	_ ast.Visitor     = (*compatibilityChecker)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.MySQLMigrationCompatibility, &CompatibilityAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.MySQLMigrationCompatibility, &CompatibilityAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.MySQLMigrationCompatibility, &CompatibilityAdvisor{})
}

// CompatibilityAdvisor is the advisor checking for schema backward compatibility.
type CompatibilityAdvisor struct {
}

// Check checks schema backward compatibility.
func (*CompatibilityAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	root, ok := ctx.AST.([]ast.StmtNode)
	if !ok {
		return nil, errors.Errorf("failed to convert to StmtNode")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	c := &compatibilityChecker{
		level: level,
		title: string(ctx.Rule.Type),
	}
	for _, stmtNode := range root {
		(stmtNode).Accept(c)
	}

	if len(c.adviceList) == 0 {
		c.adviceList = append(c.adviceList, advisor.Advice{
			Status:  advisor.Success,
			Code:    advisor.Ok,
			Title:   "OK",
			Content: "",
		})
	}
	return c.adviceList, nil
}

type compatibilityChecker struct {
	adviceList      []advisor.Advice
	level           advisor.Status
	title           string
	lastCreateTable string
}

// Enter implements the ast.Visitor interface.
func (v *compatibilityChecker) Enter(in ast.Node) (ast.Node, bool) {
	code := advisor.Ok
	switch node := in.(type) {
	// CREATE TABLE
	case *ast.CreateTableStmt:
		v.lastCreateTable = node.Table.Name.O
	// DROP DATABASE
	case *ast.DropDatabaseStmt:
		code = advisor.CompatibilityDropDatabase
	// RENAME TABLE
	case *ast.RenameTableStmt:
		code = advisor.CompatibilityRenameTable
	// DROP TABLE/VIEW
	case *ast.DropTableStmt:
		code = advisor.CompatibilityDropTable
	// ALTER TABLE
	case *ast.AlterTableStmt:
		if node.Table.Name.O == v.lastCreateTable {
			break
		}
		for _, spec := range node.Specs {
			// RENAME COLUMN
			if spec.Tp == ast.AlterTableRenameColumn {
				code = advisor.CompatibilityRenameColumn
				break
			}
			// DROP COLUMN
			if spec.Tp == ast.AlterTableDropColumn {
				code = advisor.CompatibilityDropColumn
				break
			}
			// RENAME TABLE
			if spec.Tp == ast.AlterTableRenameTable {
				code = advisor.CompatibilityRenameTable
				break
			}

			if spec.Tp == ast.AlterTableAddConstraint {
				// ADD PRIMARY KEY
				if spec.Constraint.Tp == ast.ConstraintPrimaryKey {
					code = advisor.CompatibilityAddPrimaryKey
					break
				}
				// ADD UNIQUE/UNIQUE KEY/UNIQUE INDEX
				if spec.Constraint.Tp == ast.ConstraintPrimaryKey ||
					spec.Constraint.Tp == ast.ConstraintUniq ||
					spec.Constraint.Tp == ast.ConstraintUniqKey {
					code = advisor.CompatibilityAddUniqueKey
					break
				}
				// ADD FOREIGN KEY
				if spec.Constraint.Tp == ast.ConstraintForeignKey {
					code = advisor.CompatibilityAddForeignKey
					break
				}
				// Check is only supported after 8.0.16 https://dev.mysql.com/doc/refman/8.0/en/create-table-check-constraints.html
				// ADD CHECK ENFORCED
				if spec.Constraint.Tp == ast.ConstraintCheck && spec.Constraint.Enforced {
					code = advisor.CompatibilityAddCheck
					break
				}
			}

			// Check is only supported after 8.0.16 https://dev.mysql.com/doc/refman/8.0/en/create-table-check-constraints.html
			// ALTER CHECK ENFORCED
			if spec.Tp == ast.AlterTableAlterCheck {
				if spec.Constraint.Enforced {
					code = advisor.CompatibilityAlterCheck
					break
				}
			}

			// MODIFY COLUMN / CHANGE COLUMN
			// Due to the limitation that we don't know the current data type of the column before the change,
			// so we treat all as incompatible. This generates false positive when:
			// 1. Change to a compatible data type such as INT to BIGINT
			// 2. Change properties such as comment, change it to NULL
			if spec.Tp == ast.AlterTableModifyColumn || spec.Tp == ast.AlterTableChangeColumn {
				code = advisor.CompatibilityAlterColumn
				break
			}
		}

	// ALTER VIEW TBD: https://github.com/pingcap/parser/pull/1252
	// case *ast.AlterViewStmt:

	// CREATE UNIQUE INDEX
	case *ast.CreateIndexStmt:
		if v.lastCreateTable != node.Table.Name.O && node.KeyType == ast.IndexKeyTypeUnique {
			code = advisor.CompatibilityAddUniqueKey
		}
	}

	if code != advisor.Ok {
		v.adviceList = append(v.adviceList, advisor.Advice{
			Status:  v.level,
			Code:    code,
			Title:   v.title,
			Content: fmt.Sprintf("\"%s\" may cause incompatibility with the existing data and code", in.Text()),
			Line:    in.OriginTextPosition(),
		})
	}
	return in, false
}

// Leave implements the ast.Visitor interface.
func (*compatibilityChecker) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}
