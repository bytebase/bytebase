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
// Unlike mysql/OmniAST, this type does NOT implement base.AntlrASTProvider
// (AsANTLRAST) nor a hypothetical native-pingcap fallback. Rationale:
//   - TiDB has no TiDB-specific ANTLR grammar; tidb/backup.go uses the mysql
//     ANTLR grammar on its own text and never consumes *OmniAST.
//   - The 51 advisors under backend/plugin/advisor/tidb/ use getTiDBNodes() ->
//     tidbparser.GetTiDBAST(stmt.AST) which asserts to the native *tidb.AST,
//     not *OmniAST. So no legacy consumer sees *OmniAST today.
//   - Task 1.2 (tidb.go extension) keeps the registered ParseStatementsFunc
//     pointed at the native pingcap parser ("no behavioral change yet"), so
//     *OmniAST won't reach advisors until a post-Phase 1 migration.
//
// When that migration flips the default and the 51 advisors need a compat
// path, add AsPingCapAST() + a matching provider interface in base/ast.go
// alongside the migration PR — with real consumers to exercise it.
type OmniAST struct {
	// Node is the omni AST node (e.g. *ast.SelectStmt, *ast.CreateTableStmt).
	Node ast.Node
	// Text is the original SQL text of this statement.
	Text string
	// StartPosition is the 1-based position where this statement starts.
	StartPosition *storepb.Position
}

// ASTStartPosition implements base.AST.
func (a *OmniAST) ASTStartPosition() *storepb.Position {
	return a.StartPosition
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
