package tidb

import (
	"unicode/utf8"

	"github.com/bytebase/omni/tidb/ast"
	omniparser "github.com/bytebase/omni/tidb/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// OmniAST wraps an omni/tidb AST node and implements the base.AST interface.
//
// During the Phase 1.5 advisor migration, OmniAST also implements
// PingCapASTProvider so that callers still using GetTiDBAST() (~50 of 51
// un-migrated advisors at any given point in the migration) can fall back to
// a native pingcap-parsed AST. After the dispatcher flip (§1.5.N+1) returns
// *OmniAST from ParseStatements, those un-migrated advisors keep working
// because GetTiDBAST routes through AsPingCapAST() automatically.
//
// Bridge cleanup is deferred until Phase 2 §Tier 4g (NonTransactionalDMLStmt
// + production-grade restore equivalent) ships and `dml_dry_run` (the one
// Class III advisor) can migrate. See plans/2026-04-23-omni-tidb-completion-
// plan.md §1.5.0 invariant #4 + Bridge persistence rule.
//
// Mirrors backend/plugin/parser/mysql/omni.go's AsANTLRAST pattern:
//   - Lazy parse + cache via pingcapParsed flag (matches mysql's antlrParsed).
//   - Single bridge call per OmniAST instance regardless of how many advisors
//     consume it (50+ per review under flip-last + cache).
type OmniAST struct {
	// Node is the omni AST node (e.g. *ast.SelectStmt, *ast.CreateTableStmt).
	Node ast.Node
	// Text is the original SQL text of this statement.
	Text string
	// StartPosition is the 1-based position where this statement starts.
	StartPosition *storepb.Position

	// pingcapAST is lazily populated when AsPingCapAST() is called for the
	// first time. Cached for the lifetime of this OmniAST instance.
	// Will be removed once dml_dry_run migrates and the bridge is no longer
	// needed (post-Phase-2 §Tier 4g).
	pingcapAST *AST
	// pingcapParsed tracks whether we've attempted the native parse, to
	// distinguish "not yet parsed" from "parsed and got nil".
	pingcapParsed bool
}

// ASTStartPosition implements base.AST.
func (a *OmniAST) ASTStartPosition() *storepb.Position {
	return a.StartPosition
}

// AsPingCapAST implements PingCapASTProvider for backward compatibility with
// un-migrated advisors during the Phase 1.5 migration window. Lazily parses
// the SQL text with the native pingcap parser and caches the result.
//
// Returns (nil, false) if the native parser fails to parse the text. In that
// case un-migrated advisors get no AST for this statement and emit no advice
// — same shape as the soft-fail behavior on the omni side, ensuring review
// continuity from both directions.
func (a *OmniAST) AsPingCapAST() (*AST, bool) {
	if a.pingcapParsed {
		return a.pingcapAST, a.pingcapAST != nil
	}
	a.pingcapParsed = true

	// Use the public ParseTiDB entry point — it configures the parser
	// consistently (window functions enabled, zero-date modes relaxed) so
	// the bridge result matches what direct ParseTiDB callers would see.
	nodes, err := ParseTiDB(a.Text, "", "")
	if err != nil || len(nodes) == 0 {
		return nil, false
	}
	a.pingcapAST = &AST{
		StartPosition: a.StartPosition,
		Node:          nodes[0],
	}
	return a.pingcapAST, true
}

// ParseTiDBOmni parses SQL using omni/tidb's parser and returns an ast.List.
// This is the recommended entry point for new code that needs omni/tidb AST
// nodes. Callers that need a pre-split []base.ParsedStatement should use the
// registered base.ParseStatements with storepb.Engine_TIDB instead.
func ParseTiDBOmni(sql string) (*ast.List, error) {
	return omniparser.Parse(sql)
}

// GetOmniNode extracts the omni AST node from a base.AST interface.
// Returns the node and true if it is an *OmniAST, nil and false otherwise.
func GetOmniNode(a base.AST) (ast.Node, bool) {
	if a == nil {
		return nil, false
	}
	omniAST, ok := a.(*OmniAST)
	if !ok {
		return nil, false
	}
	return omniAST.Node, true
}

// ByteOffsetToRunePosition converts a byte offset in sql to a 1-based
// line:column position, where column is measured in Unicode code points
// (runes), matching storepb.Position semantics.
//
// byteOffset values past len(sql) are clamped to len(sql). Offsets that land
// inside a multi-byte UTF-8 sequence are treated as if they pointed at the
// start of the enclosing rune (they advance 0 runes past the last complete
// rune). Offsets before 0 are treated as 0.
func ByteOffsetToRunePosition(sql string, byteOffset int) *storepb.Position {
	if byteOffset < 0 {
		byteOffset = 0
	}
	if byteOffset > len(sql) {
		byteOffset = len(sql)
	}

	line := int32(1)
	runeCol := int32(0) // 0-based rune count on current line
	i := 0
	for i < byteOffset {
		r, size := utf8.DecodeRuneInString(sql[i:])
		if i+size > byteOffset {
			// Offset lands inside a multi-byte rune. Treat it as pointing at
			// the start of the enclosing rune — don't count it.
			break
		}
		if r == '\n' {
			line++
			runeCol = 0
		} else {
			runeCol++
		}
		i += size
	}

	return &storepb.Position{
		Line:   line,
		Column: runeCol + 1, // convert to 1-based
	}
}
