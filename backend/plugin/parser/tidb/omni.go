package tidb

import (
	"log/slog"
	"unicode/utf8"

	"github.com/bytebase/omni/tidb/ast"
	omniparser "github.com/bytebase/omni/tidb/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// OmniAST wraps an omni/tidb AST node and implements the base.AST interface.
//
// It also implements PingCapASTProvider: AsPingCapAST() lazily produces a
// native pingcap-parsed AST for the one remaining consumer that needs it,
// advisor_builtin_prior_backup_check, which uses pingcap AST for authoritative
// DDL detection. The pingcap parse is cached per OmniAST instance.
type OmniAST struct {
	// Node is the omni AST node (e.g. *ast.SelectStmt, *ast.CreateTableStmt).
	Node ast.Node
	// Text is the original SQL text of this statement.
	Text string
	// StartPosition is the 1-based position where this statement starts.
	StartPosition *storepb.Position

	// pingcapAST is lazily populated and cached on the first AsPingCapAST() call.
	pingcapAST *AST
	// pingcapParsed tracks whether we've attempted the native parse, to
	// distinguish "not yet parsed" from "parsed and got nil".
	pingcapParsed bool
}

// ASTStartPosition implements base.AST.
func (a *OmniAST) ASTStartPosition() *storepb.Position {
	return a.StartPosition
}

// AsPingCapAST lazily parses the statement text with the native pingcap parser,
// caches the result, and returns it.
//
//   - Uses strict parsing (no error recovery); for the bridge case the syntax
//     is already omni-validated.
//   - Applies applyTiDBLineTracking so node.OriginTextPosition() and CREATE
//     TABLE per-column lines match the canonical ParseTiDBForSyntaxCheck path.
//
// Returns (nil, false) if pingcap fails to parse the text; the caller then sees
// no AST for the statement and emits no advice.
func (a *OmniAST) AsPingCapAST() (*AST, bool) {
	if a.pingcapParsed {
		return a.pingcapAST, a.pingcapAST != nil
	}
	a.pingcapParsed = true

	// Use the public ParseTiDB entry point — it configures the parser
	// consistently (window functions enabled, zero-date modes relaxed) so
	// the bridge result matches what direct ParseTiDB callers would see.
	nodes, err := ParseTiDB(a.Text, "", "")
	if err != nil {
		slog.Debug("pingcap re-parse failed in tidb omni bridge; un-migrated advisors will see no AST for this statement",
			slog.String("error", err.Error()),
		)
		return nil, false
	}
	if len(nodes) == 0 {
		slog.Debug("pingcap returned no nodes in tidb omni bridge; un-migrated advisors will see no AST for this statement")
		return nil, false
	}
	if len(nodes) > 1 {
		// Upstream contract: OmniAST.Text is always a single statement (set
		// by the dispatcher's split). nodes[1:] indicates a contract
		// violation upstream. Proceed with nodes[0] but record the surplus.
		slog.Debug("tidb omni bridge unexpectedly received multi-statement input; using nodes[0], dropping the rest",
			slog.Int("count", len(nodes)),
		)
	}
	node := nodes[0]

	// Mirror the line-tracking work that ParseTiDBForSyntaxCheck applies on
	// the canonical pre-flip path. Without this, post-flip un-migrated
	// advisors that call node.OriginTextPosition() would silently see line 1
	// instead of the statement's actual line in the multi-statement input.
	baseLine := 0
	if a.StartPosition != nil {
		baseLine = int(a.StartPosition.Line) - 1
	}
	if _, err := applyTiDBLineTracking(node, baseLine, a.Text); err != nil {
		slog.Debug("tidb omni bridge line-tracking failed; un-migrated advisors will see fallback line numbers",
			slog.String("error", err.Error()),
		)
		return nil, false
	}

	a.pingcapAST = &AST{
		StartPosition: a.StartPosition,
		Node:          node,
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
