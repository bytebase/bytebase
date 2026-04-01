package base

import (
	"github.com/antlr4-go/antlr/v4"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// AST is the interface that all parser AST types must implement.
// Each parser package defines its own concrete AST type with parser-specific fields.
type AST interface {
	// ASTStartPosition returns the 1-based position where this SQL statement starts
	// in the original multi-statement input. Used for error position reporting.
	// Returns nil if position is unknown.
	// Named to avoid collision with protobuf-generated GetStartPosition methods.
	ASTStartPosition() *storepb.Position
}

// ANTLRAST is the AST implementation for ANTLR-based parsers.
// Supported engines: PostgreSQL, MySQL, MariaDB, OceanBase, MSSQL, Oracle, Redshift,
// Snowflake, BigQuery, Spanner, Doris, Cassandra, Trino, PartiQL.
//
// Parser packages can use this directly or embed it to add engine-specific fields.
type ANTLRAST struct {
	// StartPosition is the 1-based position where this statement starts.
	StartPosition *storepb.Position
	Tree          antlr.Tree
	Tokens        *antlr.CommonTokenStream
}

// ASTStartPosition implements AST interface.
func (a *ANTLRAST) ASTStartPosition() *storepb.Position {
	return a.StartPosition
}

// AntlrASTProvider is an optional interface that AST implementations can provide
// to expose an underlying ANTLR AST. This is used during parser migrations when
// an AST wraps both a new AST format and a legacy ANTLR tree for backward compatibility.
type AntlrASTProvider interface {
	AsANTLRAST() (*ANTLRAST, bool)
}

// GetANTLRAST extracts the ANTLRAST from an AST interface.
// Returns the ANTLRAST and true if it is an ANTLR-based AST, nil and false otherwise.
// Also checks for AntlrASTProvider implementations (e.g. OmniAST during migration).
func GetANTLRAST(a AST) (*ANTLRAST, bool) {
	if a == nil {
		return nil, false
	}
	if antlrAST, ok := a.(*ANTLRAST); ok {
		return antlrAST, ok
	}
	if provider, ok := a.(AntlrASTProvider); ok {
		return provider.AsANTLRAST()
	}
	return nil, false
}
