package tidb

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// tidbDispatcherOmniFallbackTotal counts dispatcher fallback events per
// classified reason. This is the operations signal that drives the eventual
// Option B → A retirement decision (plan §1.5.0 invariant #8 + §Phase 3
// "Telemetry instrumentation"): when fallback firing-frequency drops below a
// threshold (e.g., < 0.1% of statements across customer reviews for 30 days),
// the fallback can be removed and the dispatcher can hard-fail.
//
// Debug logs are NOT a substitute — production telemetry pipelines drop debug
// logs before aggregation. Both surfaces (counter + slog.Debug per fallback)
// ship together; missing either leaves Option B blind in different ways.
//
// Registered against prometheus.DefaultRegisterer. The /metrics endpoint at
// backend/server/echo_routes.go folds DefaultGatherer with the echo-local
// custom registry so this counter is scrape-visible.
var tidbDispatcherOmniFallbackTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "tidb_dispatcher_omni_fallback_total",
		Help: "Count of TiDB dispatcher omni-parse failures that fell back to pingcap, labeled by classified reason.",
	},
	[]string{"reason"},
)

// omniFallbackReasonPatterns maps known Tier-4-deferred grammar keywords to
// counter labels. Each pattern is matched (case-insensitively) against the
// concatenation of (omni error message) + " " + (input SQL).
//
// Why both sides of the haystack: empirically (probed against omni
// v0.0.0-20260513072939-39c04c4cca0f) the omni parser echoes the offending
// keyword for FLASHBACK and BATCH but reports CREATE SEQUENCE as
// "unexpected token after CREATE" — the keyword SEQUENCE never appears in
// the error string. Without input-side matching we'd silently lose the
// SEQUENCE telemetry signal entirely.
//
// PRECEDENCE: first-match-wins iteration order. The current three Tier-4
// patterns (FLASHBACK / SEQUENCE / BATCH) have no overlap, so order is
// presently moot. If a future Tier-4 grammar gap introduces an overlapping
// keyword, the patterns list will need explicit ordering or
// more-specific-first sorting; this comment documents the contract before
// it becomes load-bearing.
var omniFallbackReasonPatterns = []struct {
	pattern string // uppercase substring; matched against UPPER(err.Error() + " " + sql)
	reason  string // counter label value
}{
	{"FLASHBACK", "flashback"},
	{"SEQUENCE", "sequence"},
	{"BATCH", "batch_dml"},
}

// classifyOmniParseError maps an omni parse error + the input SQL into a
// counter label. The input SQL is required because the omni error string
// does not always echo the offending keyword (notably for CREATE SEQUENCE).
// Returns "unknown" for nil error or any error+sql whose haystack contains
// no known Tier-4 keyword.
func classifyOmniParseError(err error, sql string) string {
	if err == nil {
		return "unknown"
	}
	haystack := strings.ToUpper(err.Error() + " " + sql)
	for _, p := range omniFallbackReasonPatterns {
		if strings.Contains(haystack, p.pattern) {
			return p.reason
		}
	}
	return "unknown"
}
