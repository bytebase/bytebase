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

// Oracle grammar snapshot (PlSqlParser.g4 v0.0.0-20260417075056-…):
//
//	truncate_table            : TRUNCATE TABLE tableview_name PURGE? SEMICOLON ;
//	truncate_table_partition  : TRUNCATE (partition_extended_names | subpartition_extended_names) ... ;
//	partition_extended_names  : (PARTITION | PARTITIONS) ( partition_name
//	                                                     | '(' partition_name (',' partition_name)* ')'
//	                                                     | FOR '('? partition_key_value (',' partition_key_value)* ')'? ) ;
//	subpartition_extended_names : same shape, swapping SUBPARTITION + subpartition_key_value.
//
// Advisor resolution truth table:
//   - Truncate_table           → emit table-variant advice; bare PURGE is allowed
//                                by the grammar and does not exempt.
//   - Truncate_table_partition → walk the chosen names branch (partition OR
//                                subpartition) and for each, extract either
//                                the identifier list or the FOR(value) literal.
//                                Emit one advice per named target.

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
		table := oracleEnclosingAlterTable(pc)
		keyword, target := oracleExtractTruncatePartitionTarget(pc)
		r.AddAdvice(
			r.level,
			code.StatementDisallowTruncate.Int32(),
			fmt.Sprintf(`ALTER TABLE %q TRUNCATE %s %q is not allowed: partition truncate shares the implicit-commit gap of TRUNCATE TABLE on Oracle. Prior-backup treats this as DDL and does not produce row-level snapshots.`, table, keyword, target),
			common.ConvertANTLRLineToPosition(r.baseLine+ctx.GetStart().GetLine()),
		)
	default:
	}
	return nil
}

func (*StatementDisallowTruncateRule) OnExit(_ antlr.ParserRuleContext, _ string) error { return nil }

// oracleEnclosingAlterTable walks up from a child context to the enclosing
// Alter_table and returns its Tableview_name text. Returns "" if the parent
// chain is absent (should not happen for a well-formed AST).
func oracleEnclosingAlterTable(ctx antlr.ParserRuleContext) string {
	for p := ctx.GetParent(); p != nil; p = p.GetParent() {
		if at, ok := p.(*parser.Alter_tableContext); ok {
			if tv := at.Tableview_name(); tv != nil {
				return tv.GetText()
			}
			return ""
		}
	}
	return ""
}

// oracleExtractTruncatePartitionTarget returns ("PARTITION"|"SUBPARTITION",
// name) for a Truncate_table_partition context. For the `FOR (value)` form,
// the target is rendered as `FOR(<value>)`. For comma-separated lists the
// names are joined with `, ` into a single advice target; the single-advice
// shape matches a single AST node and the grammar treats the whole list as
// one truncate command.
func oracleExtractTruncatePartitionTarget(ctx *parser.Truncate_table_partitionContext) (keyword, target string) {
	if pen := ctx.Partition_extended_names(); pen != nil {
		if c, ok := pen.(*parser.Partition_extended_namesContext); ok {
			return "PARTITION", oraclePartitionNames(c.AllPartition_name(), c.FOR() != nil, partitionKeyValueTexts(c))
		}
	}
	if sen := ctx.Subpartition_extended_names(); sen != nil {
		if c, ok := sen.(*parser.Subpartition_extended_namesContext); ok {
			return "SUBPARTITION", oraclePartitionNames(c.AllPartition_name(), c.FOR() != nil, subpartitionKeyValueTexts(c))
		}
	}
	return "PARTITION", ""
}

// oraclePartitionNames renders the target string for either partition- or
// subpartition-extended names. Prefers explicit identifier names; falls back
// to FOR(value) literal when the grammar took the FOR branch.
func oraclePartitionNames(names []parser.IPartition_nameContext, isFor bool, forValues []string) string {
	if len(names) > 0 {
		parts := make([]string, 0, len(names))
		for _, pn := range names {
			parts = append(parts, pn.GetText())
		}
		return strings.Join(parts, ", ")
	}
	if isFor && len(forValues) > 0 {
		return "FOR(" + strings.Join(forValues, ", ") + ")"
	}
	return ""
}

func partitionKeyValueTexts(c *parser.Partition_extended_namesContext) []string {
	vals := c.AllPartition_key_value()
	out := make([]string, 0, len(vals))
	for _, v := range vals {
		out = append(out, v.GetText())
	}
	return out
}

func subpartitionKeyValueTexts(c *parser.Subpartition_extended_namesContext) []string {
	vals := c.AllSubpartition_key_value()
	out := make([]string, 0, len(vals))
	for _, v := range vals {
		out = append(out, v.GetText())
	}
	return out
}
