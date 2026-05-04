package tidb

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/tidb/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	advisorcode "github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*CompatibilityAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_SCHEMA_BACKWARD_COMPATIBILITY, &CompatibilityAdvisor{})
}

// CompatibilityAdvisor is the advisor checking for schema backward compatibility.
type CompatibilityAdvisor struct {
}

// Check checks schema backward compatibility.
func (*CompatibilityAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBOmniNodes(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	c := &compatibilityChecker{
		level: level,
		title: checkCtx.Rule.Type.String(),
	}
	for _, ostmt := range stmts {
		c.checkStmt(ostmt)
	}

	return c.adviceList, nil
}

type compatibilityChecker struct {
	adviceList      []*storepb.Advice
	level           storepb.Advice_Status
	title           string
	lastCreateTable string
}

func (c *compatibilityChecker) checkStmt(ostmt OmniStmt) {
	// Capture line and code in the same arm so they can never drift.
	// A separate locStart switch would be a synchronization hazard:
	// adding a node type to one switch without the other would silently
	// degrade line numbers to the default of line 1.
	code := advisorcode.Ok
	line := 0
	switch n := ostmt.Node.(type) {
	case *ast.CreateTableStmt:
		if n.Table != nil {
			c.lastCreateTable = n.Table.Name
		}
		// CREATE TABLE never produces compatibility advice; line capture
		// is intentionally skipped — code stays Ok.
	case *ast.DropDatabaseStmt:
		code = advisorcode.CompatibilityDropDatabase
		line = ostmt.AbsoluteLine(n.Loc.Start)
	case *ast.RenameTableStmt:
		code = advisorcode.CompatibilityRenameTable
		line = ostmt.AbsoluteLine(n.Loc.Start)
	case *ast.DropTableStmt:
		code = advisorcode.CompatibilityDropTable
		line = ostmt.AbsoluteLine(n.Loc.Start)
	case *ast.AlterTableStmt:
		if n.Table != nil && n.Table.Name == c.lastCreateTable {
			break
		}
		code = c.classifyAlterTable(n)
		line = ostmt.AbsoluteLine(n.Loc.Start)
	case *ast.CreateIndexStmt:
		if n.Table != nil && n.Table.Name != c.lastCreateTable && n.Unique {
			code = advisorcode.CompatibilityAddUniqueKey
			line = ostmt.AbsoluteLine(n.Loc.Start)
		}
	default:
	}

	if code != advisorcode.Ok {
		c.adviceList = append(c.adviceList, &storepb.Advice{
			Status:        c.level,
			Code:          code.Int32(),
			Title:         c.title,
			Content:       fmt.Sprintf("\"%s\" may cause incompatibility with the existing data and code", ostmt.Text),
			StartPosition: common.ConvertANTLRLineToPosition(line),
		})
	}
}

// classifyAlterTable inspects ALTER TABLE commands and returns the first
// compatibility-impacting code, mirroring the pingcap-AST behavior of
// breaking on the first hit.
func (*compatibilityChecker) classifyAlterTable(n *ast.AlterTableStmt) advisorcode.Code {
	for _, cmd := range n.Commands {
		if cmd == nil {
			continue
		}
		switch cmd.Type {
		case ast.ATRenameColumn:
			return advisorcode.CompatibilityRenameColumn
		case ast.ATDropColumn:
			return advisorcode.CompatibilityDropColumn
		case ast.ATRenameTable:
			return advisorcode.CompatibilityRenameTable
		case ast.ATAddConstraint:
			if c := classifyAddConstraint(cmd.Constraint); c != advisorcode.Ok {
				return c
			}
		case ast.ATAlterCheckEnforced:
			// CHECK ENFORCED is only meaningful when the constraint is
			// enforced (the inverse of NotEnforced).
			if cmd.Constraint != nil && !cmd.Constraint.NotEnforced {
				return advisorcode.CompatibilityAlterCheck
			}
		case ast.ATModifyColumn, ast.ATChangeColumn:
			// Pingcap parity: we don't know the current type of the column,
			// so treat all MODIFY/CHANGE COLUMN as potentially incompatible.
			// False positives on type widening (INT → BIGINT) and metadata-
			// only changes (comments, NULL → NOT NULL) are inherited.
			return advisorcode.CompatibilityAlterColumn
		default:
		}
	}
	return advisorcode.Ok
}

func classifyAddConstraint(constraint *ast.Constraint) advisorcode.Code {
	if constraint == nil {
		return advisorcode.Ok
	}
	switch constraint.Type {
	case ast.ConstrPrimaryKey:
		return advisorcode.CompatibilityAddPrimaryKey
	case ast.ConstrUnique:
		// Pingcap had three distinct unique constraint types
		// (ConstraintUniq / ConstraintUniqKey / ConstraintUniqIndex);
		// omni unifies them under ConstrUnique. NOTE: this is also a
		// silent bug fix — the original pingcap-typed code at lines
		// 104-106 only checked ConstraintUniq + ConstraintUniqKey and
		// missed ConstraintUniqIndex, so `ALTER TABLE t ADD UNIQUE INDEX
		// idx (col)` previously emitted no compatibility advice. Post-
		// migration all three forms route here and are flagged
		// consistently.
		return advisorcode.CompatibilityAddUniqueKey
	case ast.ConstrForeignKey:
		return advisorcode.CompatibilityAddForeignKey
	case ast.ConstrCheck:
		// CHECK is only meaningful when enforced.
		if !constraint.NotEnforced {
			return advisorcode.CompatibilityAddCheck
		}
	default:
	}
	return advisorcode.Ok
}
