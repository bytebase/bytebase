package base

import (
	"github.com/antlr4-go/antlr/v4"
)

// AST represents a parsed SQL statement with standardized metadata.
// It wraps different parser outputs (ANTLR, TiDB, CockroachDB) into a single type.
// Each parser package is responsible for creating AST instances.
type AST struct {
	// OriginalText is the original SQL text (may be empty for some parsers)
	OriginalText string

	// ByteOffsetStart is the start byte offset in the original multi-statement SQL
	ByteOffsetStart int

	// ByteOffsetEnd is the end byte offset in the original multi-statement SQL
	ByteOffsetEnd int

	// BaseLine stores the zero-based line offset where this SQL statement starts
	// in the original multi-statement input. Used for error position reporting.
	BaseLine int

	// Underlying representation (only one will be set based on parser type)

	// ANTLRResult contains ANTLR-based parser data.
	// Supported engines: PostgreSQL, MySQL, MariaDB, OceanBase, MSSQL, Oracle, Redshift, Snowflake, BigQuery, Spanner, Doris, Cassandra, Trino, PartiQL, CosmosDB.
	ANTLRResult *ANTLRParseData
	// TiDBNode contains TiDB parser AST node (ast.StmtNode).
	// Supported engines: TiDB.
	// Use type assertion in the tidb package to access the typed node.
	TiDBNode any
	// CockroachDBStmt contains CockroachDB parser statement (statements.Statement[tree.Statement]).
	// Supported engines: CockroachDB.
	// Use type assertion in the cockroachdb package to access the typed statement.
	CockroachDBStmt any
}

// ANTLRParseData contains ANTLR-based parser results (parse tree and token stream).
// Supported engines: PostgreSQL, MySQL, MariaDB, OceanBase, MSSQL, Oracle, Redshift, Snowflake, BigQuery, Spanner, Doris, Cassandra, Trino, PartiQL, CosmosDB.
type ANTLRParseData struct {
	Tree   antlr.Tree
	Tokens *antlr.CommonTokenStream
}

// GetANTLRTree returns ANTLR parse data if this statement was parsed by an ANTLR-based parser.
// Returns the data and true if available, nil and false otherwise.
func (u *AST) GetANTLRTree() (*ANTLRParseData, bool) {
	return u.ANTLRResult, u.ANTLRResult != nil
}
