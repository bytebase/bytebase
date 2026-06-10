package snowflake

import (
	omniast "github.com/bytebase/omni/snowflake/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ base.AST              = (*OmniAST)(nil)
	_ base.AntlrASTProvider = (*OmniAST)(nil)
)

// OmniAST wraps an omni AST node and implements the base.AST interface.
//
// It mirrors redshift's OmniAST with one addition: during the Snowflake
// advisor migration it ALSO carries the legacy ANTLR parse tree for the SAME
// statement (dual-AST transition layer). Migrated advisors read the omni node
// via GetOmniNode; the not-yet-migrated ANTLR advisors keep working unchanged
// because base.GetANTLRAST resolves through AsANTLRAST.
type OmniAST struct {
	// Node is the omni AST node for this statement (e.g. *ast.SelectStmt).
	// It is nil when the omni parser could not parse a statement the legacy
	// parser accepted (transition leniency); GetOmniNode then reports false.
	Node omniast.Node
	// Text is the original SQL text of this statement.
	Text string
	// StartPosition is the 1-based position where this statement starts.
	StartPosition *storepb.Position
	// LegacyAST is the legacy ANTLR parse tree for the SAME statement. It is
	// kept during the advisor migration so base.GetANTLRAST(stmt.AST) keeps
	// working for the un-migrated ANTLR advisors; drop it together with the
	// AsANTLRAST implementation once the last advisor moves onto the omni node.
	LegacyAST *base.ANTLRAST
}

// ASTStartPosition implements base.AST.
func (a *OmniAST) ASTStartPosition() *storepb.Position {
	return a.StartPosition
}

// AsANTLRAST implements base.AntlrASTProvider by exposing the wrapped legacy
// ANTLR tree, so base.GetANTLRAST works transparently on an OmniAST.
func (a *OmniAST) AsANTLRAST() (*base.ANTLRAST, bool) {
	if a == nil || a.LegacyAST == nil {
		return nil, false
	}
	return a.LegacyAST, true
}

// GetOmniNode extracts the omni AST node from a base.AST interface.
// It returns false when a is not a snowflake OmniAST or when the omni parser
// could not parse the statement (nil node).
func GetOmniNode(a base.AST) (omniast.Node, bool) {
	if a == nil {
		return nil, false
	}
	omniAST, ok := a.(*OmniAST)
	if !ok || omniAST.Node == nil {
		return nil, false
	}
	return omniAST.Node, true
}

// parseOmniStatementNode parses ONE statement's text with the omni parser and
// returns its single top-level statement node. It returns nil — never an
// error — when omni rejects the statement or yields no node: during the
// dual-AST transition a statement the legacy parser accepted must not fail
// the batch just because omni cannot parse it yet. An omni node is attached
// only when omni FULLY accepted the statement (no partial trees), so migrated
// advisors never see a half-parsed statement.
func parseOmniStatementNode(text string) omniast.Node {
	file, err := parseSnowflakeAST(text)
	if err != nil || file == nil || len(file.Stmts) == 0 {
		return nil
	}
	return file.Stmts[0]
}
