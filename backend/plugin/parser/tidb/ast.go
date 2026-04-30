package tidb

import (
	tidbast "github.com/pingcap/tidb/pkg/parser/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// AST is the AST implementation for TiDB parser.
// It implements the base.AST interface.
type AST struct {
	// StartPosition is the 1-based position where this statement starts.
	StartPosition *storepb.Position
	Node          tidbast.StmtNode
}

// ASTStartPosition implements base.AST interface.
func (a *AST) ASTStartPosition() *storepb.Position {
	return a.StartPosition
}

// PingCapASTProvider is an optional interface that AST implementations can
// provide to expose an underlying native pingcap-parser AST. Used during the
// Phase 1.5 advisor migration: when the registered ParseStatementsFunc is
// flipped to omni, *OmniAST values reach un-migrated callers of GetTiDBAST,
// and those callers fall through to AsPingCapAST() to keep working.
//
// Mirrors backend/plugin/parser/base/AntlrASTProvider for the mysql migration.
// See plans/2026-04-23-omni-tidb-completion-plan.md §1.5.0 invariant #4.
type PingCapASTProvider interface {
	AsPingCapAST() (*AST, bool)
}

// GetTiDBAST extracts the TiDB AST from a base.AST.
// Returns the AST and true if it is a TiDB AST, nil and false otherwise.
// Also checks for PingCapASTProvider implementations (e.g. *OmniAST during
// the advisor migration), which lazily build a native pingcap AST on demand.
func GetTiDBAST(a base.AST) (*AST, bool) {
	if a == nil {
		return nil, false
	}
	if tidbAST, ok := a.(*AST); ok {
		return tidbAST, true
	}
	if provider, ok := a.(PingCapASTProvider); ok {
		return provider.AsPingCapAST()
	}
	return nil, false
}
