package tidb

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// TestParseTiDBStatementsOmni_OptionBFallback pins the Phase 1.5 §1.5.N+1
// dispatcher flip contract (invariant #8): when omni rejects a statement,
// the dispatcher falls back to native pingcap and the review never breaks.
// Bracketed input asserts the omni-accepted statements come back as
// *OmniAST and the omni-rejected one comes back as *AST (pingcap
// fallback) — un-migrated advisors continue to work end-to-end.
//
// Also asserts the per-fallback observability sub-contract: the
// tidb_dispatcher_omni_fallback_total{reason} counter increments by
// exactly 1 for the rejected statement, with the empirically-verified
// "sequence" label.
//
// Empirical note (invariant #9): the Option B fixture must be a statement
// omni REJECTS but pingcap ACCEPTS. BATCH non-transactional DML was the
// original choice, but omni now SUPPORTS BATCH (grammar merged in omni
// #157, consumed via the go.mod bump), so it no longer falls back.
// `FLASHBACK TABLE foo TO BEFORE DROP;` can't be used either — pingcap
// rejects that exact syntax (both-engines-reject path). `CREATE SEQUENCE`
// is the correct Option B input: omni rejects it (Tier-4 grammar gap),
// pingcap accepts it (TiDB sequences). If pingcap also rejected it, the
// require.NoError below would fail.
func TestParseTiDBStatementsOmni_OptionBFallback(t *testing.T) {
	const (
		omniAccepted1 = "CREATE TABLE t (id INT);"
		omniRejected  = "CREATE SEQUENCE seq;"
		omniAccepted2 = "INSERT INTO t (id) VALUES (1);"
	)
	input := omniAccepted1 + "\n" + omniRejected + "\n" + omniAccepted2

	// Snapshot the counter before so the assertion is delta-based — other
	// tests in the package may have incremented it already.
	before := testutil.ToFloat64(tidbDispatcherOmniFallbackTotal.WithLabelValues("sequence"))

	result, err := parseTiDBStatementsOmni(input)
	require.NoError(t, err,
		"Option B contract: omni grammar gap on one statement must NOT propagate as a review-breaking error")
	require.Len(t, result, 3,
		"all three statements must be present in the result, even when omni rejects one")

	// Assertion: accepted/rejected statements get the right AST type.
	// The Statement.Text field carries the raw input for each split, so
	// we identify by substring rather than index (split may include
	// trailing whitespace etc.).
	for _, ps := range result {
		switch {
		case strings.Contains(ps.Text, "SEQUENCE"):
			require.IsType(t, &AST{}, ps.AST,
				"omni-rejected statement must carry pingcap *AST after Option B fallback (un-migrated advisors continue to function)")
		case strings.Contains(ps.Text, "CREATE TABLE") || strings.Contains(ps.Text, "INSERT"):
			require.IsType(t, &OmniAST{}, ps.AST,
				"omni-accepted statements must carry *OmniAST")
		default:
			t.Fatalf("unexpected statement text in result: %q", ps.Text)
		}
	}

	// Counter sub-contract.
	after := testutil.ToFloat64(tidbDispatcherOmniFallbackTotal.WithLabelValues("sequence"))
	require.InDelta(t, 1.0, after-before, 0.0001,
		"counter must increment by exactly 1 for the single SEQUENCE fallback")
}

// TestParseTiDBStatementsOmni_BothEnginesReject pins the Q2 design
// choice: when omni AND pingcap both reject a statement, the dispatcher
// surfaces omni's error, not pingcap's. This sets customer-facing
// expectations matching the eventual Option A state — when the fallback
// retires, the same input will still surface the same omni error.
//
// Also pins the "no counter inflation on both-reject" contract: malformed
// SQL must NOT increment tidb_dispatcher_omni_fallback_total. Inflating
// the counter (especially the "unknown" bucket) on bad-SQL inputs would
// skew the Option B → A retirement-gate signal — after omni grammar is
// complete, customer-side garbage SQL would keep the counter non-zero
// and the gate would never fire. Per Codex round on PR #20340.
func TestParseTiDBStatementsOmni_BothEnginesReject(t *testing.T) {
	// SELECT FROM WHERE is genuine syntax garbage — both omni and pingcap
	// reject it. Verified by the metrics_test parse-test (returns
	// "unknown" classifier label).
	const input = "SELECT FROM WHERE;"

	beforeUnknown := testutil.ToFloat64(tidbDispatcherOmniFallbackTotal.WithLabelValues("unknown"))

	_, err := parseTiDBStatementsOmni(input)
	require.Error(t, err,
		"both-engines-reject must propagate as an error to the customer")

	// Surface should be omni's. Empirical omni error string for this
	// input is `syntax error at or near "FROM" (line 1, column 8)`.
	// After convertOmniParseError it is wrapped as a base.SyntaxError
	// whose RawMessage carries omni's verbatim text.
	syntaxErr, ok := err.(*base.SyntaxError)
	require.True(t, ok, "error must be base.SyntaxError after conversion; got %T", err)
	require.Contains(t, syntaxErr.RawMessage, "syntax error",
		"raw message must come from omni's parser, preserving the eventual Option A surface")

	// Counter contract: both-reject MUST NOT increment any reason bucket.
	afterUnknown := testutil.ToFloat64(tidbDispatcherOmniFallbackTotal.WithLabelValues("unknown"))
	require.Equal(t, beforeUnknown, afterUnknown,
		"both-engines-reject must NOT inflate the fallback counter (retirement-gate signal stays clean)")
}

// TestParseTiDBStatementsOmni_AllAccepted pins the happy path: when omni
// accepts every statement, no fallbacks fire and every result entry is
// an *OmniAST. The counter does not move.
func TestParseTiDBStatementsOmni_AllAccepted(t *testing.T) {
	const input = "CREATE TABLE t (id INT); INSERT INTO t VALUES (1);"

	beforeFlash := testutil.ToFloat64(tidbDispatcherOmniFallbackTotal.WithLabelValues("flashback"))
	beforeUnknown := testutil.ToFloat64(tidbDispatcherOmniFallbackTotal.WithLabelValues("unknown"))

	result, err := parseTiDBStatementsOmni(input)
	require.NoError(t, err)
	require.Len(t, result, 2)
	for _, ps := range result {
		require.IsType(t, &OmniAST{}, ps.AST,
			"happy path: every statement must be *OmniAST")
	}

	require.Equal(t, beforeFlash,
		testutil.ToFloat64(tidbDispatcherOmniFallbackTotal.WithLabelValues("flashback")),
		"happy path: flashback counter must not move")
	require.Equal(t, beforeUnknown,
		testutil.ToFloat64(tidbDispatcherOmniFallbackTotal.WithLabelValues("unknown")),
		"happy path: unknown counter must not move")
}

// TestParsePingCapSingleStatement_LineTrackingMatchesCanonical pins that
// the dispatcher's pingcap fallback helper produces *AST values
// structurally identical to the canonical pre-flip
// ParseTiDBForSyntaxCheck path — so post-flip un-migrated advisors that
// read node.OriginTextPosition() see consistent values.
//
// This is the dispatcher analog of the existing
// TestAsPingCapASTLineTrackingMatchesCanonical (which pins the bridge
// path); both fallback shapes (dispatcher fallback + bridge fallback)
// must produce identical line numbers.
func TestParsePingCapSingleStatement_LineTrackingMatchesCanonical(t *testing.T) {
	// Multi-line input so the BaseLine offset matters.
	const multi = "CREATE TABLE t1 (id INT);\n\nCREATE TABLE t2 (id INT);"

	canonical, err := ParseTiDBForSyntaxCheck(multi)
	require.NoError(t, err)
	require.Len(t, canonical, 2)

	stmts, err := base.SplitMultiSQL(storepb.Engine_TIDB, multi)
	require.NoError(t, err)
	require.Len(t, stmts, 2)

	for i, stmt := range stmts {
		got, err := parsePingCapSingleStatement(stmt)
		require.NoError(t, err)
		require.NotNil(t, got, "single-statement parse must succeed")
		canonicalAST, ok := canonical[i].(*AST)
		require.True(t, ok)
		require.Equal(t, canonicalAST.Node.OriginTextPosition(), got.Node.OriginTextPosition(),
			"dispatcher fallback line number must match canonical pre-flip path")
	}
}
