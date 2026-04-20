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

var _ advisor.Advisor = (*StatementDisallowTruncateAdvisor)(nil)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_DISALLOW_TRUNCATE, &StatementDisallowTruncateAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_STATEMENT_DISALLOW_TRUNCATE, &StatementDisallowTruncateAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_STATEMENT_DISALLOW_TRUNCATE, &StatementDisallowTruncateAdvisor{})
}

type StatementDisallowTruncateAdvisor struct{}

func (*StatementDisallowTruncateAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	rule := &statementDisallowTruncateOmniRule{
		OmniBaseRule: OmniBaseRule{Level: level, Title: checkCtx.Rule.Type.String()},
	}
	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type statementDisallowTruncateOmniRule struct{ OmniBaseRule }

func (*statementDisallowTruncateOmniRule) Name() string { return "StatementDisallowTruncateRule" }

func (r *statementDisallowTruncateOmniRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.TruncateStmt:
		for _, tbl := range n.Tables {
			r.emitTable(tbl, n.Loc)
		}
	case *ast.AlterTableStmt:
		if n.Table == nil {
			return
		}
		for _, cmd := range n.Commands {
			if cmd.Type != ast.ATTruncatePartition {
				continue
			}
			if cmd.AllPartitions {
				r.emitPartition(n.Table, "ALL PARTITIONS", n.Loc)
				continue
			}
			for _, pname := range cmd.PartitionNames {
				r.emitPartition(n.Table, pname, n.Loc)
			}
		}
	default:
	}
}

func (r *statementDisallowTruncateOmniRule) emitTable(tbl *ast.TableRef, loc ast.Loc) {
	r.AddAdviceAbsolute(&storepb.Advice{
		Status:        r.Level,
		Code:          code.StatementDisallowTruncate.Int32(),
		Title:         r.Title,
		Content:       fmt.Sprintf(`TRUNCATE TABLE %q is not allowed: TRUNCATE auto-commits (breaking surrounding transactional work), bypasses triggers, and resets AUTO_INCREMENT. Prior-backup treats this as DDL and does not produce row-level snapshots.`, qualifyMySQLName(tbl)),
		StartPosition: common.ConvertANTLRLineToPosition(r.BaseLine + int(r.LocToLine(loc))),
	})
}

func (r *statementDisallowTruncateOmniRule) emitPartition(tbl *ast.TableRef, partition string, loc ast.Loc) {
	r.AddAdviceAbsolute(&storepb.Advice{
		Status:        r.Level,
		Code:          code.StatementDisallowTruncate.Int32(),
		Title:         r.Title,
		Content:       fmt.Sprintf(`ALTER TABLE %q TRUNCATE PARTITION %q is not allowed: partition truncate shares the auto-commit and prior-backup gaps of TRUNCATE TABLE on MySQL.`, qualifyMySQLName(tbl), partition),
		StartPosition: common.ConvertANTLRLineToPosition(r.BaseLine + int(r.LocToLine(loc))),
	})
}

func qualifyMySQLName(tbl *ast.TableRef) string {
	if tbl.Schema != "" {
		return tbl.Schema + "." + tbl.Name
	}
	return tbl.Name
}
