package pg

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/pg/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	advisorcode "github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*CompatibilityAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_SCHEMA_BACKWARD_COMPATIBILITY, &CompatibilityAdvisor{})
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

	rule := &compatibilityRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type compatibilityRule struct {
	OmniBaseRule

	lastCreateTable string
}

func (*compatibilityRule) Name() string {
	return "migration_compatibility"
}

func (r *compatibilityRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateStmt:
		r.handleCreateStmt(n)
	case *ast.DropdbStmt:
		r.handleDropdbStmt()
	case *ast.DropStmt:
		r.handleDropStmt(n)
	case *ast.RenameStmt:
		r.handleRenameStmt(n)
	case *ast.AlterTableStmt:
		r.handleAlterTableStmt(n)
	case *ast.IndexStmt:
		r.handleIndexStmt(n)
	default:
	}
}

func (r *compatibilityRule) handleCreateStmt(n *ast.CreateStmt) {
	if n.Relation != nil {
		r.lastCreateTable = omniTableName(n.Relation)
	}
}

func (r *compatibilityRule) handleDropdbStmt() {
	stmtText := r.TrimmedStmtText()
	r.AddAdvice(&storepb.Advice{
		Status:  r.Level,
		Code:    advisorcode.CompatibilityDropDatabase.Int32(),
		Title:   r.Title,
		Content: fmt.Sprintf(`"%s" may cause incompatibility with the existing data and code`, stmtText),
		StartPosition: &storepb.Position{
			Line:   r.ContentStartLine(),
			Column: 0,
		},
	})
}

func (r *compatibilityRule) handleDropStmt(n *ast.DropStmt) {
	removeType := ast.ObjectType(n.RemoveType)
	// Flag DROP TABLE and DROP MATERIALIZED VIEW (data loss risk).
	// Skip non-materialized DROP VIEW -- views hold no data.
	if removeType == ast.OBJECT_TABLE || removeType == ast.OBJECT_MATVIEW {
		stmtText := r.TrimmedStmtText()
		r.AddAdvice(&storepb.Advice{
			Status:  r.Level,
			Code:    advisorcode.CompatibilityDropTable.Int32(),
			Title:   r.Title,
			Content: fmt.Sprintf(`"%s" may cause incompatibility with the existing data and code`, stmtText),
			StartPosition: &storepb.Position{
				Line:   r.ContentStartLine(),
				Column: 0,
			},
		})
	}
}

func (r *compatibilityRule) handleRenameStmt(n *ast.RenameStmt) {
	code := advisorcode.Ok

	switch n.RenameType {
	case ast.OBJECT_COLUMN:
		// RENAME COLUMN - check if not on last created table
		if n.Relation != nil {
			tableName := omniTableName(n.Relation)
			if r.lastCreateTable != tableName {
				code = advisorcode.CompatibilityRenameColumn
			}
		}
	case ast.OBJECT_TABLE:
		code = advisorcode.CompatibilityRenameTable
	default:
	}

	if code != advisorcode.Ok {
		stmtText := r.TrimmedStmtText()
		r.AddAdvice(&storepb.Advice{
			Status:  r.Level,
			Code:    code.Int32(),
			Title:   r.Title,
			Content: fmt.Sprintf(`"%s" may cause incompatibility with the existing data and code`, stmtText),
			StartPosition: &storepb.Position{
				Line:   r.ContentStartLine(),
				Column: 0,
			},
		})
	}
}

func (r *compatibilityRule) handleAlterTableStmt(n *ast.AlterTableStmt) {
	if n.Relation == nil {
		return
	}
	tableName := omniTableName(n.Relation)

	// Skip if this is the table we just created
	if r.lastCreateTable == tableName {
		return
	}

	for _, cmd := range omniAlterTableCmds(n) {
		code := advisorcode.Ok

		switch ast.AlterTableType(cmd.Subtype) {
		case ast.AT_DropColumn:
			code = advisorcode.CompatibilityDropColumn
		case ast.AT_AlterColumnType:
			code = advisorcode.CompatibilityAlterColumn
		case ast.AT_AddConstraint:
			constraint, ok := cmd.Def.(*ast.Constraint)
			if ok {
				switch constraint.Contype {
				case ast.CONSTR_PRIMARY:
					code = advisorcode.CompatibilityAddPrimaryKey
				case ast.CONSTR_UNIQUE:
					code = advisorcode.CompatibilityAddUniqueKey
				case ast.CONSTR_FOREIGN:
					code = advisorcode.CompatibilityAddForeignKey
				case ast.CONSTR_CHECK:
					if !constraint.SkipValidation {
						code = advisorcode.CompatibilityAddCheck
					}
				default:
				}
			}
		default:
		}

		if code != advisorcode.Ok {
			stmtText := r.TrimmedStmtText()
			r.AddAdvice(&storepb.Advice{
				Status:  r.Level,
				Code:    code.Int32(),
				Title:   r.Title,
				Content: fmt.Sprintf(`"%s" may cause incompatibility with the existing data and code`, stmtText),
				StartPosition: &storepb.Position{
					Line:   r.ContentStartLine(),
					Column: 0,
				},
			})
			return
		}
	}
}

func (r *compatibilityRule) handleIndexStmt(n *ast.IndexStmt) {
	// Check if this is CREATE UNIQUE INDEX
	if !n.Unique {
		return
	}

	// Get table name
	if n.Relation == nil {
		return
	}
	tableName := omniTableName(n.Relation)

	// Skip if this is the table we just created
	if r.lastCreateTable == tableName {
		return
	}

	stmtText := r.TrimmedStmtText()
	r.AddAdvice(&storepb.Advice{
		Status:  r.Level,
		Code:    advisorcode.CompatibilityAddUniqueKey.Int32(),
		Title:   r.Title,
		Content: fmt.Sprintf(`"%s" may cause incompatibility with the existing data and code`, stmtText),
		StartPosition: &storepb.Position{
			Line:   r.ContentStartLine(),
			Column: 0,
		},
	})
}
