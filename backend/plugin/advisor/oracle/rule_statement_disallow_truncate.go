package oracle

import (
	"context"
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/plsql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var _ advisor.Advisor = (*StatementDisallowTruncateAdvisor)(nil)

func init() {
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_STATEMENT_DISALLOW_TRUNCATE, &StatementDisallowTruncateAdvisor{})
}

type StatementDisallowTruncateAdvisor struct{}

func (*StatementDisallowTruncateAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	rule := &StatementDisallowTruncateRule{BaseRule: NewBaseRule(level, checkCtx.Rule.Type.String(), 0)}
	checker := NewGenericChecker([]Rule{rule})
	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		antlrAST, ok := base.GetANTLRAST(stmt.AST)
		if !ok {
			continue
		}
		rule.SetBaseLine(stmt.BaseLine())
		checker.SetBaseLine(stmt.BaseLine())
		antlr.ParseTreeWalkerDefault.Walk(checker, antlrAST.Tree)
	}
	return checker.GetAdviceList()
}

type StatementDisallowTruncateRule struct{ BaseRule }

func (*StatementDisallowTruncateRule) Name() string { return "statement.disallow-truncate" }

func (r *StatementDisallowTruncateRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Truncate_table":
		tc, ok := ctx.(*parser.Truncate_tableContext)
		if !ok {
			return nil
		}
		name := ""
		if tv := tc.Tableview_name(); tv != nil {
			name = tv.GetText()
		}
		r.AddAdvice(
			r.level,
			code.StatementDisallowTruncate.Int32(),
			fmt.Sprintf(`TRUNCATE TABLE %q is not allowed: it issues an implicit COMMIT and cannot be rolled back. Any prior uncommitted work in the same transaction is also committed. Prior-backup treats this as DDL and does not produce row-level snapshots.`, name),
			common.ConvertANTLRLineToPosition(r.baseLine+ctx.GetStart().GetLine()),
		)
	case "Truncate_table_partition":
		pc, ok := ctx.(*parser.Truncate_table_partitionContext)
		if !ok {
			return nil
		}
		table, partition := oracleExtractTruncatePartition(pc)
		r.AddAdvice(
			r.level,
			code.StatementDisallowTruncate.Int32(),
			fmt.Sprintf(`ALTER TABLE %q TRUNCATE PARTITION %q is not allowed: partition truncate shares the implicit-commit gap of TRUNCATE TABLE on Oracle. Prior-backup treats this as DDL and does not produce row-level snapshots.`, table, partition),
			common.ConvertANTLRLineToPosition(r.baseLine+ctx.GetStart().GetLine()),
		)
	default:
	}
	return nil
}

func (*StatementDisallowTruncateRule) OnExit(_ antlr.ParserRuleContext, _ string) error { return nil }

// oracleExtractTruncatePartition walks up from the Truncate_table_partition
// context to the enclosing Alter_table context (via Alter_table_partitioning)
// to obtain the target table, and reads the partition name(s) off the
// current context's grammar-generated accessor.
func oracleExtractTruncatePartition(ctx *parser.Truncate_table_partitionContext) (table, partition string) {
	if names := ctx.Partition_extended_names(); names != nil {
		// Partition_extended_names covers `PARTITION ( p1, p2, ... )` as a
		// whole rule; GetText() on it concatenates tokens without spaces and
		// includes the PARTITION keyword itself. Read the partition-name
		// children directly to obtain just the identifiers.
		if pen, ok := names.(*parser.Partition_extended_namesContext); ok {
			var parts []string
			for _, pn := range pen.AllPartition_name() {
				parts = append(parts, pn.GetText())
			}
			partition = strings.Join(parts, ", ")
		}
	}
	for p := ctx.GetParent(); p != nil; p = p.GetParent() {
		if at, ok := p.(*parser.Alter_tableContext); ok {
			if tv := at.Tableview_name(); tv != nil {
				table = tv.GetText()
			}
			return table, partition
		}
	}
	return table, partition
}
