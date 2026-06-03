package oracle

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/oracle/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
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
	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule})
}

type StatementDisallowTruncateRule struct{ BaseRule }

func (*StatementDisallowTruncateRule) Name() string { return "statement.disallow-truncate" }

// OnStatement checks TRUNCATE statements and ALTER TABLE truncate actions in the omni AST.
func (r *StatementDisallowTruncateRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.TruncateStmt:
		if n.Cluster {
			return
		}
		name := omniLastObjectName(n.Table)
		if n.Table != nil {
			if raw := r.rawText(n.Table.Loc); raw != "" {
				name = raw
			}
		}
		r.AddAdvice(
			r.level,
			code.StatementDisallowTruncate.Int32(),
			fmt.Sprintf(`TRUNCATE TABLE %q is not allowed: it issues an implicit COMMIT and cannot be rolled back. Any prior uncommitted work in the same transaction is also committed. Prior-backup treats this as DDL and does not produce row-level snapshots.`, name),
			common.ConvertANTLRLineToPosition(r.locLine(n.Loc)),
		)
	case *ast.AlterTableStmt:
		table := omniLastObjectName(n.Name)
		if n.Name != nil {
			if raw := r.rawText(n.Name.Loc); raw != "" {
				table = raw
			}
		}
		for _, cmd := range omniAlterTableCmds(n) {
			if cmd.Action != ast.AT_TRUNCATE_PARTITION {
				continue
			}
			keyword := "PARTITION"
			if strings.EqualFold(cmd.Subtype, "SUBPARTITION") {
				keyword = "SUBPARTITION"
			}
			target := cmd.ColumnName
			if target == "" {
				target = cmd.NewName
			}
			if rawAction := r.rawText(cmd.Loc); rawAction != "" {
				upperAction := strings.ToUpper(rawAction)
				if idx := strings.Index(upperAction, keyword); idx >= 0 {
					if rawTarget := strings.TrimSpace(rawAction[idx+len(keyword):]); rawTarget != "" {
						if strings.HasPrefix(strings.ToUpper(rawTarget), "FOR ") {
							rawTarget = "FOR" + strings.TrimSpace(rawTarget[len("FOR "):])
						}
						target = rawTarget
					}
				}
			}
			r.AddAdvice(
				r.level,
				code.StatementDisallowTruncate.Int32(),
				fmt.Sprintf(`ALTER TABLE %q TRUNCATE %s %q is not allowed: partition truncate shares the implicit-commit gap of TRUNCATE TABLE on Oracle. Prior-backup treats this as DDL and does not produce row-level snapshots.`, table, keyword, target),
				common.ConvertANTLRLineToPosition(r.locLine(cmd.Loc)),
			)
		}
	default:
	}
}

// oracleEnclosingAlterTable walks up from a child context to the enclosing
// Alter_table and returns its Tableview_name text. Returns "" if the parent
// chain is absent (should not happen for a well-formed AST).

// oracleExtractTruncatePartitionTarget returns ("PARTITION"|"SUBPARTITION",
// name) for a Truncate_table_partition context. For the `FOR (value)` form,
// the target is rendered as `FOR(<value>)`. For comma-separated lists the
// names are joined with `, ` into a single advice target; the single-advice
// shape matches a single AST node and the grammar treats the whole list as
// one truncate command.

// oraclePartitionNames renders the target string for either partition- or
// subpartition-extended names. Prefers explicit identifier names; falls back
// to FOR(value) literal when the grammar took the FOR branch.
