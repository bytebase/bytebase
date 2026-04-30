package tidb

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
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
