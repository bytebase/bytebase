package tidb

import (
	"testing"

	"github.com/stretchr/testify/require"

	omniast "github.com/bytebase/omni/tidb/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	tidbparser "github.com/bytebase/bytebase/backend/plugin/parser/tidb"
)

// TestGetTiDBOmniNodesCachesAcrossRules pins the Phase 1.5 single-parse-per-
// review invariant: when multiple migrated advisors call getTiDBOmniNodes
// with the same advisor.Context, omni parsing happens once and the result
// slice is reused. Without this, parse cost scales with migrated-advisor
// count and review latency degrades monotonically across the migration
// window. See plans/2026-04-23-omni-tidb-completion-plan.md §1.5.0.
func TestGetTiDBOmniNodesCachesAcrossRules(t *testing.T) {
	ctx := advisor.Context{
		ParsedStatements: []base.ParsedStatement{
			{
				Statement: base.Statement{
					Text:  "CREATE TABLE t (id INT)",
					Start: &storepb.Position{Line: 1},
				},
			},
		},
	}
	ctx.InitMemo()

	first, err := getTiDBOmniNodes(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, first, "first call should return a parsed statement")

	second, err := getTiDBOmniNodes(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, second)

	// Cache hit: same backing array returned both calls. Address equality
	// on the first element proves the slice header was reused, not a fresh
	// re-parse that would allocate a new []OmniStmt.
	require.True(t, &first[0] == &second[0],
		"expected cached []OmniStmt; got fresh re-parse — single-parse-per-review invariant broken")
}

// TestGetTiDBOmniNodesSoftFailsOnGrammarGap pins the Phase 1.5 soft-fail
// invariant: a statement that fails to parse with omni is logged and
// skipped, not propagated as an error that breaks the advisor. The review
// continues with whatever statements omni did parse.
//
// This test feeds a statement that is valid TiDB SQL but uses a deferred
// Phase 2 grammar feature (BATCH ... DRY RUN ...) that omni/tidb does not
// yet support. The expectation is that getTiDBOmniNodes returns the
// successfully-parsed statements and skips the BATCH one, with no error.
func TestGetTiDBOmniNodesSoftFailsOnGrammarGap(t *testing.T) {
	ctx := advisor.Context{
		ParsedStatements: []base.ParsedStatement{
			{
				Statement: base.Statement{
					Text:  "CREATE TABLE t (id INT)",
					Start: &storepb.Position{Line: 1},
				},
			},
			{
				Statement: base.Statement{
					Text:  "BATCH ON id LIMIT 5000 UPDATE t SET v = v + 1",
					Start: &storepb.Position{Line: 2},
				},
			},
			{
				Statement: base.Statement{
					Text:  "INSERT INTO t (id) VALUES (1)",
					Start: &storepb.Position{Line: 3},
				},
			},
		},
	}
	ctx.InitMemo()

	got, err := getTiDBOmniNodes(ctx)
	require.NoError(t, err, "soft-fail invariant: parse errors must not propagate")
	require.GreaterOrEqual(t, len(got), 2,
		"expected at least the two non-BATCH statements to parse and be returned")
}

// TestGetTiDBNodesSoftFailsOnBridgeMiss pins the post-flip soft-fail
// contract on the un-migrated-advisor side: when an OmniAST's
// AsPingCapAST() returns (nil, false) — i.e. omni accepted the statement
// but pingcap rejected the same text — getTiDBNodes must SKIP that
// statement, not abort the rule with "AST type mismatch".
//
// Codex caught this on PR #20179: without the soft-fail, post-flip every
// un-migrated advisor emits "Rule check failed: AST type mismatch" for
// statements omni accepts but pingcap doesn't, breaking the migration
// continuity the bridge is meant to preserve.
func TestGetTiDBNodesSoftFailsOnBridgeMiss(t *testing.T) {
	// Pingcap rejects this; omni doesn't matter for this test — what
	// matters is that the bridge returns (nil, false) and getTiDBNodes
	// must skip rather than error.
	bridgeMiss := &tidbparser.OmniAST{
		Text:          "SELECT FROM WHERE;", // pingcap-invalid
		StartPosition: &storepb.Position{Line: 1},
	}
	bridgeHit := &tidbparser.OmniAST{
		Text:          "SELECT 1",
		StartPosition: &storepb.Position{Line: 2},
	}

	ctx := advisor.Context{
		ParsedStatements: []base.ParsedStatement{
			{
				Statement: base.Statement{
					Text:  "SELECT FROM WHERE;",
					Start: &storepb.Position{Line: 1},
				},
				AST: bridgeMiss,
			},
			{
				Statement: base.Statement{
					Text:  "SELECT 1",
					Start: &storepb.Position{Line: 2},
				},
				AST: bridgeHit,
			},
		},
	}

	got, err := getTiDBNodes(ctx)
	require.NoError(t, err,
		"bridge miss must be soft-failed (skip), not surfaced as 'AST type mismatch' — see Phase 1.5 invariant #2")
	require.Len(t, got, 1,
		"expected 1 statement from the bridge-hit branch; the bridge-miss statement should be skipped silently")
}

// TestCollectFKCreateTableNilConstraintGuard pins the defensive nil-check
// in collectFKCreateTable. The original batch-4 form passed
// `ostmt.AbsoluteLine(constraint.Loc.Start)` as the line argument to
// buildFKMetaData, which evaluated the dereference BEFORE the helper's
// internal nil check could fire — a nil entry in n.Constraints would
// panic. Per Codex round-1 review on PR #20204; sibling collectFKAlterTable
// was safe because it precomputes stmtLine outside the loop, and
// collectIndexCreateTable / collectUKCreateTable already nil-checked.
//
// Omni's parser doesn't currently produce nil entries from valid SQL, so
// the YAML fixture suite can't reach this branch. The synthetic AST here
// pins the contract: a future change that re-orders the loop body or
// removes the guard fails here.
func TestCollectFKCreateTableNilConstraintGuard(t *testing.T) {
	stmt := &omniast.CreateTableStmt{
		Table: &omniast.TableRef{Name: "t"},
		Constraints: []*omniast.Constraint{
			nil, // would panic on AbsoluteLine(constraint.Loc.Start) without the guard
			{
				Type:       omniast.ConstrForeignKey,
				Name:       "fk_t_author_id_author_id",
				Columns:    []string{"author_id"},
				RefTable:   &omniast.TableRef{Name: "author"},
				RefColumns: []string{"id"},
			},
		},
	}
	ostmt := OmniStmt{
		Node:     stmt,
		Text:     "CREATE TABLE t (author_id INT, FOREIGN KEY (author_id) REFERENCES author(id))",
		BaseLine: 0,
	}

	require.NotPanics(t, func() {
		result := collectFKCreateTable(ostmt, stmt)
		// The non-nil FK constraint must still be collected — the nil guard
		// is `continue`, not an early exit.
		require.Len(t, result, 1, "non-nil FK constraint should still be collected after the nil guard skips the nil entry")
		require.Equal(t, "fk_t_author_id_author_id", result[0].indexName)
	})
}
