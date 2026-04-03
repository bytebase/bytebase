package mysql

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/mysql/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*CompatibilityAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_SCHEMA_BACKWARD_COMPATIBILITY, &CompatibilityAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_SCHEMA_BACKWARD_COMPATIBILITY, &CompatibilityAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_SCHEMA_BACKWARD_COMPATIBILITY, &CompatibilityAdvisor{})
}

// CompatibilityAdvisor is the advisor checking for schema backward compatibility.
type CompatibilityAdvisor struct {
}

// Check checks schema backward compatibility.
func (*CompatibilityAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &compatibilityOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type compatibilityOmniRule struct {
	OmniBaseRule
	lastCreateTable string
}

func (*compatibilityOmniRule) Name() string {
	return "CompatibilityRule"
}

func (r *compatibilityOmniRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		r.checkCreateTable(n)
	case *ast.DropDatabaseStmt:
		r.addIncompat(code.CompatibilityDropDatabase, n.Loc)
	case *ast.RenameTableStmt:
		r.addIncompat(code.CompatibilityRenameTable, n.Loc)
	case *ast.DropTableStmt:
		r.addIncompat(code.CompatibilityDropTable, n.Loc)
	case *ast.AlterTableStmt:
		r.checkAlterTable(n)
	case *ast.CreateIndexStmt:
		r.checkCreateIndex(n)
	default:
	}
}

func (r *compatibilityOmniRule) addIncompat(c code.Code, loc ast.Loc) {
	r.AddAdvice(&storepb.Advice{
		Status:        r.Level,
		Code:          c.Int32(),
		Title:         r.Title,
		Content:       fmt.Sprintf("\"%s\" may cause incompatibility with the existing data and code", r.QueryText()),
		StartPosition: common.ConvertANTLRLineToPosition(r.BaseLine + int(r.LocToLine(loc))),
	})
}

func (r *compatibilityOmniRule) checkCreateTable(n *ast.CreateTableStmt) {
	if n.Table == nil || len(n.Columns) == 0 {
		return
	}
	r.lastCreateTable = n.Table.Name
}

func (r *compatibilityOmniRule) checkAlterTable(n *ast.AlterTableStmt) {
	if n.Table == nil {
		return
	}
	tableName := n.Table.Name
	if tableName == r.lastCreateTable {
		return
	}

	for _, cmd := range n.Commands {
		if cmd == nil {
			continue
		}
		switch cmd.Type {
		case ast.ATRenameColumn:
			r.addIncompat(code.CompatibilityRenameColumn, n.Loc)
			return
		case ast.ATDropColumn:
			r.addIncompat(code.CompatibilityDropColumn, n.Loc)
			return
		case ast.ATRenameTable:
			r.addIncompat(code.CompatibilityRenameTable, n.Loc)
			return
		case ast.ATAddConstraint:
			if cmd.Constraint == nil {
				continue
			}
			switch cmd.Constraint.Type {
			case ast.ConstrPrimaryKey:
				r.addIncompat(code.CompatibilityAddPrimaryKey, n.Loc)
				return
			case ast.ConstrUnique:
				r.addIncompat(code.CompatibilityAddUniqueKey, n.Loc)
				return
			case ast.ConstrForeignKey:
				r.addIncompat(code.CompatibilityAddForeignKey, n.Loc)
				return
			case ast.ConstrCheck:
				// add check enforced
				if !cmd.Constraint.NotEnforced {
					r.addIncompat(code.CompatibilityAddCheck, n.Loc)
					return
				}
			default:
			}
		case ast.ATAlterCheckEnforced:
			r.addIncompat(code.CompatibilityAlterCheck, n.Loc)
			return
		case ast.ATModifyColumn, ast.ATChangeColumn:
			r.addIncompat(code.CompatibilityAlterColumn, n.Loc)
			return
		default:
		}
	}
}

func (r *compatibilityOmniRule) checkCreateIndex(n *ast.CreateIndexStmt) {
	if !n.Unique {
		return
	}
	if n.Table == nil {
		return
	}
	tableName := n.Table.Name
	if r.lastCreateTable != tableName {
		r.addIncompat(code.CompatibilityAddUniqueKey, n.Loc)
	}
}
