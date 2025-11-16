package tidb

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	advisorcode "github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/pingcap/tidb/pkg/parser/ast"
)

var (
	_ advisor.Advisor = (*CompatibilityAdvisor)(nil)
	_ ast.Visitor     = (*compatibilityChecker)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, advisor.SchemaRuleSchemaBackwardCompatibility, &CompatibilityAdvisor{})
}

// CompatibilityAdvisor is the advisor checking for schema backward compatibility.
type CompatibilityAdvisor struct {
}

// Check checks schema backward compatibility.
func (*CompatibilityAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	root, ok := checkCtx.AST.([]ast.StmtNode)
	if !ok {
		return nil, errors.Errorf("failed to convert to StmtNode")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	c := &compatibilityChecker{
		level: level,
		title: string(checkCtx.Rule.Type),
	}
	for _, stmtNode := range root {
		(stmtNode).Accept(c)
	}

	return c.adviceList, nil
}

type compatibilityChecker struct {
	adviceList      []*storepb.Advice
	level           storepb.Advice_Status
	title           string
	lastCreateTable string
}

// Enter implements the ast.Visitor interface.
func (v *compatibilityChecker) Enter(in ast.Node) (ast.Node, bool) {
	code := advisorcode.Ok
	switch node := in.(type) {
	// CREATE TABLE
	case *ast.CreateTableStmt:
		v.lastCreateTable = node.Table.Name.O
	// DROP DATABASE
	case *ast.DropDatabaseStmt:
		code = advisorcode.CompatibilityDropDatabase
	// RENAME TABLE
	case *ast.RenameTableStmt:
		code = advisorcode.CompatibilityRenameTable
	// DROP TABLE/VIEW
	case *ast.DropTableStmt:
		code = advisorcode.CompatibilityDropTable
	// ALTER TABLE
	case *ast.AlterTableStmt:
		if node.Table.Name.O == v.lastCreateTable {
			break
		}
		for _, spec := range node.Specs {
			// RENAME COLUMN
			if spec.Tp == ast.AlterTableRenameColumn {
				code = advisorcode.CompatibilityRenameColumn
				break
			}
			// DROP COLUMN
			if spec.Tp == ast.AlterTableDropColumn {
				code = advisorcode.CompatibilityDropColumn
				break
			}
			// RENAME TABLE
			if spec.Tp == ast.AlterTableRenameTable {
				code = advisorcode.CompatibilityRenameTable
				break
			}

			if spec.Tp == ast.AlterTableAddConstraint {
				// ADD PRIMARY KEY
				if spec.Constraint.Tp == ast.ConstraintPrimaryKey {
					code = advisorcode.CompatibilityAddPrimaryKey
					break
				}
				// ADD UNIQUE/UNIQUE KEY/UNIQUE INDEX
				if spec.Constraint.Tp == ast.ConstraintPrimaryKey ||
					spec.Constraint.Tp == ast.ConstraintUniq ||
					spec.Constraint.Tp == ast.ConstraintUniqKey {
					code = advisorcode.CompatibilityAddUniqueKey
					break
				}
				// ADD FOREIGN KEY
				if spec.Constraint.Tp == ast.ConstraintForeignKey {
					code = advisorcode.CompatibilityAddForeignKey
					break
				}
				// Check is only supported after 8.0.16 https://dev.mysql.com/doc/refman/8.0/en/create-table-check-constraints.html
				// ADD CHECK ENFORCED
				if spec.Constraint.Tp == ast.ConstraintCheck && spec.Constraint.Enforced {
					code = advisorcode.CompatibilityAddCheck
					break
				}
			}

			// Check is only supported after 8.0.16 https://dev.mysql.com/doc/refman/8.0/en/create-table-check-constraints.html
			// ALTER CHECK ENFORCED
			if spec.Tp == ast.AlterTableAlterCheck {
				if spec.Constraint.Enforced {
					code = advisorcode.CompatibilityAlterCheck
					break
				}
			}

			// MODIFY COLUMN / CHANGE COLUMN
			// Due to the limitation that we don't know the current data type of the column before the change,
			// so we treat all as incompatible. This generates false positive when:
			// 1. Change to a compatible data type such as INT to BIGINT
			// 2. Change properties such as comment, change it to NULL
			if spec.Tp == ast.AlterTableModifyColumn || spec.Tp == ast.AlterTableChangeColumn {
				code = advisorcode.CompatibilityAlterColumn
				break
			}
		}

	// ALTER VIEW TBD: https://github.com/pingcap/parser/pull/1252
	// case *ast.AlterViewStmt:

	// CREATE UNIQUE INDEX
	case *ast.CreateIndexStmt:
		if v.lastCreateTable != node.Table.Name.O && node.KeyType == ast.IndexKeyTypeUnique {
			code = advisorcode.CompatibilityAddUniqueKey
		}
	}

	if code != advisorcode.Ok {
		v.adviceList = append(v.adviceList, &storepb.Advice{
			Status:        v.level,
			Code:          code.Int32(),
			Title:         v.title,
			Content:       fmt.Sprintf("\"%s\" may cause incompatibility with the existing data and code", in.Text()),
			StartPosition: common.ConvertANTLRLineToPosition(in.OriginTextPosition()),
		})
	}
	return in, false
}

// Leave implements the ast.Visitor interface.
func (*compatibilityChecker) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}
