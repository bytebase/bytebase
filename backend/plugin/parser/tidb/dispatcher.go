package tidb

import (
	"errors"
	"fmt"
	"log/slog"

	omniparser "github.com/bytebase/omni/tidb/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// parseTiDBStatementsOmni is the post-flip ParseStatementsFunc for TiDB
// (registered in tidb.go init). Implements Option B per plan §1.5.0
// invariant #8: omni first, pingcap fallback per statement on omni parse
// failure. The review never hard-fails at the dispatcher level on a Tier 4
// grammar gap — customer sees advice for the parseable statements; the
// omni-rejected ones get pingcap-AST so un-migrated advisors continue to
// work, OR get no AST when both engines reject (the customer-facing error
// then surfaces omni's complaint, matching the eventual Option A state).
//
// Per-fallback observability has TWO required surfaces (sub-contract):
//   - Counter tidb_dispatcher_omni_fallback_total{reason}: operations
//     signal that drives the eventual retirement decision. Debug logs are
//     dropped before reaching aggregation pipelines and cannot serve this
//     role.
//   - slog.Debug per fallback: developer signal for diagnosing individual
//     reports. Both surfaces ship together.
//
// Trade-off accepted: per-statement-skip-with-no-advice on omni-rejected
// SQL (vs full-review-failure under Option A). Strictly better than Option
// A for the migration window; less informative than the future state where
// omni grammar is complete enough to drop the fallback.
func parseTiDBStatementsOmni(statement string) ([]base.ParsedStatement, error) {
	stmts, err := base.SplitMultiSQL(storepb.Engine_TIDB, statement)
	if err != nil {
		return nil, err
	}

	var result []base.ParsedStatement
	for _, stmt := range stmts {
		if stmt.Empty {
			result = append(result, base.ParsedStatement{Statement: stmt})
			continue
		}

		// Attempt order: omni first (sub-contract). Pingcap-first would
		// defeat the architectural intent — post-flip, omni is canonical;
		// pingcap is the safety net.
		list, omniErr := ParseTiDBOmni(stmt.Text)
		if omniErr == nil {
			if list == nil || len(list.Items) == 0 {
				// Omni succeeded but produced no items (e.g. comment-only
				// statement that survived the splitter). Preserve the
				// statement position with a nil AST, matching pre-flip
				// parseTiDBStatements semantics.
				result = append(result, base.ParsedStatement{Statement: stmt})
				continue
			}
			for _, node := range list.Items {
				result = append(result, base.ParsedStatement{
					Statement: stmt,
					AST: &OmniAST{
						Node:          node,
						Text:          stmt.Text,
						StartPosition: stmt.Start,
					},
				})
			}
			continue
		}

		// Omni rejected — try Option B fallback to pingcap.
		ast, fallbackErr := parsePingCapSingleStatement(stmt)
		if fallbackErr != nil {
			// Both engines reject. Don't increment the fallback counter —
			// this is genuine bad SQL, not an omni grammar gap. Inflating
			// the counter (especially the "unknown" bucket) on bad-SQL
			// inputs would skew the Option B → A retirement-gate signal:
			// after omni grammar is complete, malformed customer SQL
			// would keep the counter non-zero and the gate would never
			// fire. Surface omni's error so customer-facing expectations
			// track the eventual Option A state (Q2 design choice — see
			// plans/2026-04-23-omni-tidb-completion-plan.md §1.5.0
			// invariant #8 + dispatcher_test.go regression pin).
			return nil, convertOmniParseError(omniErr, stmt)
		}

		// Fallback succeeded — record the omni gap that pingcap bridged.
		// Counter measures "omni rejected AND pingcap accepted" — the
		// cases that genuinely justify Option B and drive the retirement
		// decision.
		reason := classifyOmniParseError(omniErr, stmt.Text)
		tidbDispatcherOmniFallbackTotal.WithLabelValues(reason).Inc()
		// Escalate "unknown" reason to Warn so ops sees the log line in
		// production aggregation (which typically drops Debug). Without
		// escalation the counter increments but the input excerpt + error
		// string needed to add a new classifier pattern — and to drive
		// the eventual Option B → A retirement gate to zero unknowns —
		// stays invisible. Known reasons (flashback / sequence / batch_dml)
		// stay at Debug: high-frequency, expected, counter-tracked.
		logFn := slog.Debug
		if reason == "unknown" {
			logFn = slog.Warn
		}
		logFn("tidb dispatcher: omni parse failed; falling back to pingcap",
			slog.String("reason", reason),
			slog.String("excerpt", excerptForDebug(stmt.Text, 80)),
			slog.String("error", omniErr.Error()),
		)

		ps := base.ParsedStatement{Statement: stmt}
		if ast != nil {
			ps.AST = ast
		}
		result = append(result, ps)
	}

	return result, nil
}

// parsePingCapSingleStatement runs the native pingcap parser on a single
// pre-split base.Statement and applies line-tracking. Mirrors the per-loop
// body of ParseTiDBForSyntaxCheck so the dispatcher's pingcap-fallback path
// produces *AST values structurally identical to the canonical pre-flip
// path (un-migrated advisors expect identical line numbers + node shape).
//
// Returns:
//   - (*AST, nil) on a clean single-node parse.
//   - (nil, nil) when the parser returned a non-1 node count (skip — same
//     semantic as ParseTiDBForSyntaxCheck's `if len(nodes) != 1 { continue }`).
//   - (nil, err) on a hard parser error, with the syntax error's line
//     adjusted to absolute coordinates.
func parsePingCapSingleStatement(singleSQL base.Statement) (*AST, error) {
	p := newTiDBParser()
	nodes, _, err := p.Parse(singleSQL.Text, "", "")
	if err != nil {
		syntaxErr := convertParserError(err)
		if se, ok := syntaxErr.(*base.SyntaxError); ok && se.Position != nil {
			se.Position.Line = int32(singleSQL.BaseLine()) + se.Position.Line
		}
		return nil, syntaxErr
	}
	if len(nodes) != 1 {
		return nil, nil
	}
	node := nodes[0]
	actualStartLine, err := applyTiDBLineTracking(node, singleSQL.BaseLine(), singleSQL.Text)
	if err != nil {
		return nil, err
	}
	return &AST{
		StartPosition: &storepb.Position{Line: int32(actualStartLine)},
		Node:          node,
	}, nil
}

// convertOmniParseError mirrors mysql.go's convertOmniError: takes an omni
// parser error and produces a base.SyntaxError with absolute line/column
// coordinates (so the customer sees a position relative to the original
// multi-statement input, not the per-statement excerpt).
//
// Returns the original error unchanged if it is not an *omniparser.ParseError.
func convertOmniParseError(err error, stmt base.Statement) error {
	var parseErr *omniparser.ParseError
	if !errors.As(err, &parseErr) {
		return err
	}

	pos := ByteOffsetToRunePosition(stmt.Text, parseErr.Position)
	if stmt.Start != nil {
		pos.Line += stmt.Start.Line - 1
	}

	msg := fmt.Sprintf("Syntax error at line %d:%d: %s", pos.Line, pos.Column, parseErr.Message)
	if parseErr.RelatedText != "" {
		msg += "\nrelated text: " + parseErr.RelatedText
	}

	return &base.SyntaxError{
		Position:   pos,
		Message:    msg,
		RawMessage: parseErr.Message,
	}
}

// excerptForDebug truncates s for slog.Debug payloads. Keeps fallback log
// lines compact even when statement text runs long.
func excerptForDebug(s string, maxRunes int) string {
	if len(s) <= maxRunes {
		return s
	}
	return s[:maxRunes] + "..."
}
