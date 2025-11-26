package base

import (
	"github.com/antlr4-go/antlr/v4"
)

// AST is the interface that all parser AST types must implement.
// Each parser package defines its own concrete AST type with parser-specific fields.
type AST interface {
	// GetBaseLine returns the zero-based line offset where this SQL statement starts
	// in the original multi-statement input. Used for error position reporting.
	GetBaseLine() int
}

// ANTLRAST is the AST implementation for ANTLR-based parsers.
// Supported engines: PostgreSQL, MySQL, MariaDB, OceanBase, MSSQL, Oracle, Redshift,
// Snowflake, BigQuery, Spanner, Doris, Cassandra, Trino, PartiQL, CosmosDB.
//
// Parser packages can use this directly or embed it to add engine-specific fields.
type ANTLRAST struct {
	BaseLine int
	Tree     antlr.Tree
	Tokens   *antlr.CommonTokenStream
}

// GetBaseLine implements AST interface.
func (a *ANTLRAST) GetBaseLine() int {
	return a.BaseLine
}

// GetANTLRAST extracts the ANTLRAST from an AST interface.
// Returns the ANTLRAST and true if it is an ANTLR-based AST, nil and false otherwise.
func GetANTLRAST(a AST) (*ANTLRAST, bool) {
	if a == nil {
		return nil, false
	}
	antlrAST, ok := a.(*ANTLRAST)
	return antlrAST, ok
}
