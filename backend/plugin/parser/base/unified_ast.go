package base

import (
	"reflect"

	"github.com/antlr4-go/antlr/v4"
	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/parser/statements"
	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/sem/tree"
	"github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// UnifiedAST represents a parsed SQL statement with standardized metadata.
// It wraps different parser outputs (ANTLR, TiDB, CockroachDB) into a single unified type.
type UnifiedAST struct {
	// Engine is the database engine that parsed this SQL
	Engine storepb.Engine

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
	// TiDBNode contains TiDB parser AST node.
	// Supported engines: TiDB.
	TiDBNode *TiDBParseData
	// CockroachDBStmt contains CockroachDB parser statement.
	// Supported engines: CockroachDB.
	CockroachDBStmt *CockroachDBParseData
}

// ANTLRParseData contains ANTLR-based parser results (parse tree and token stream).
// Supported engines: PostgreSQL, MySQL, MariaDB, OceanBase, MSSQL, Oracle, Redshift, Snowflake, BigQuery, Spanner, Doris, Cassandra, Trino, PartiQL, CosmosDB.
type ANTLRParseData struct {
	Tree   antlr.Tree
	Tokens *antlr.CommonTokenStream
}

// TiDBParseData contains TiDB native parser AST node.
// Supported engines: TiDB.
type TiDBParseData struct {
	Node ast.StmtNode
}

// CockroachDBParseData contains CockroachDB native parser statement.
// Supported engines: CockroachDB.
type CockroachDBParseData struct {
	Stmt statements.Statement[tree.Statement]
}

// GetANTLRTree returns ANTLR parse data if this statement was parsed by an ANTLR-based parser.
// Returns the data and true if available, nil and false otherwise.
func (u *UnifiedAST) GetANTLRTree() (*ANTLRParseData, bool) {
	return u.ANTLRResult, u.ANTLRResult != nil
}

// GetTiDBNode returns TiDB AST node if this statement was parsed by TiDB parser.
// Returns the node and true if available, nil and false otherwise.
func (u *UnifiedAST) GetTiDBNode() (*TiDBParseData, bool) {
	return u.TiDBNode, u.TiDBNode != nil
}

// GetCockroachDBStmt returns CockroachDB statement if this statement was parsed by CockroachDB parser.
// Returns the statement and true if available, nil and false otherwise.
func (u *UnifiedAST) GetCockroachDBStmt() (*CockroachDBParseData, bool) {
	return u.CockroachDBStmt, u.CockroachDBStmt != nil
}

// GetEngine returns the database engine that parsed this SQL.
func (u *UnifiedAST) GetEngine() storepb.Engine {
	return u.Engine
}

// GetBaseLine returns the zero-based line offset where this SQL statement starts.
func (u *UnifiedAST) GetBaseLine() int {
	return u.BaseLine
}

// convertToUnifiedAST converts raw parser output to a slice of UnifiedAST.
// This function handles different parser types (ANTLR, TiDB, CockroachDB) and wraps
// them into a consistent unified representation.
func convertToUnifiedAST(engine storepb.Engine, _ string, rawAST any) ([]*UnifiedAST, error) {
	switch engine {
	case storepb.Engine_TIDB:
		// TiDB parser returns []ast.StmtNode
		nodes, ok := rawAST.([]ast.StmtNode)
		if !ok {
			return nil, errors.Errorf("expected []ast.StmtNode for TiDB, got %T", rawAST)
		}

		var results []*UnifiedAST
		for _, node := range nodes {
			results = append(results, &UnifiedAST{
				Engine:   engine,
				TiDBNode: &TiDBParseData{Node: node},
			})
		}
		return results, nil

	case storepb.Engine_COCKROACHDB:
		// CockroachDB parser returns statements.Statements
		stmts, ok := rawAST.(statements.Statements)
		if !ok {
			return nil, errors.Errorf("expected statements.Statements for CockroachDB, got %T", rawAST)
		}

		var results []*UnifiedAST
		for _, stmt := range stmts {
			results = append(results, &UnifiedAST{
				Engine:          engine,
				OriginalText:    stmt.SQL,
				CockroachDBStmt: &CockroachDBParseData{Stmt: stmt},
			})
		}
		return results, nil

	default:
		// All ANTLR-based parsers return []*ParseResult-like structure
		// This includes: PostgreSQL, MySQL, Oracle, SQL Server, Snowflake, Redshift, BigQuery, etc.
		// Each parser has its own ParseResult type, but they all have the same field structure:
		// - Tree (antlr.Tree)
		// - Tokens (*antlr.CommonTokenStream)
		// - BaseLine (int)

		return convertANTLRParseResults(engine, rawAST)
	}
}

// convertANTLRParseResults converts ANTLR parser results using reflection.
// All ANTLR parsers return a slice of structs with Tree, Tokens, and BaseLine fields.
func convertANTLRParseResults(engine storepb.Engine, rawAST any) ([]*UnifiedAST, error) {
	// Use reflection to handle slices of ParseResult from different packages
	v := reflect.ValueOf(rawAST)
	if v.Kind() != reflect.Slice {
		return nil, errors.Errorf("expected slice of ParseResult for %s, got %T", engine, rawAST)
	}

	var results []*UnifiedAST
	for i := 0; i < v.Len(); i++ {
		item := v.Index(i)

		// Handle both pointer and value types
		if item.Kind() == reflect.Ptr {
			item = item.Elem()
		}

		if item.Kind() != reflect.Struct {
			return nil, errors.Errorf("expected struct in slice for %s, got %v", engine, item.Kind())
		}

		// Extract fields using reflection
		tree, tokens, baseLine, err := extractParseResultFields(item)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract ParseResult fields for %s", engine)
		}

		results = append(results, &UnifiedAST{
			Engine:   engine,
			BaseLine: baseLine,
			ANTLRResult: &ANTLRParseData{
				Tree:   tree,
				Tokens: tokens,
			},
		})
	}

	return results, nil
}

// extractParseResultFields extracts Tree, Tokens, and BaseLine fields from a struct using reflection.
func extractParseResultFields(item reflect.Value) (antlr.Tree, *antlr.CommonTokenStream, int, error) {
	treeField := item.FieldByName("Tree")
	tokensField := item.FieldByName("Tokens")
	baseLineField := item.FieldByName("BaseLine")

	if !treeField.IsValid() || !tokensField.IsValid() || !baseLineField.IsValid() {
		return nil, nil, 0, errors.New("struct does not have required fields (Tree, Tokens, BaseLine)")
	}

	// Extract Tree (antlr.Tree is an interface)
	tree, ok := treeField.Interface().(antlr.Tree)
	if !ok {
		return nil, nil, 0, errors.Errorf("Tree field is not antlr.Tree, got %T", treeField.Interface())
	}

	// Extract Tokens (*antlr.CommonTokenStream)
	tokens, ok := tokensField.Interface().(*antlr.CommonTokenStream)
	if !ok {
		return nil, nil, 0, errors.Errorf("Tokens field is not *antlr.CommonTokenStream, got %T", tokensField.Interface())
	}

	// Extract BaseLine (int)
	baseLine, ok := baseLineField.Interface().(int)
	if !ok {
		return nil, nil, 0, errors.Errorf("BaseLine field is not int, got %T", baseLineField.Interface())
	}

	return tree, tokens, baseLine, nil
}
