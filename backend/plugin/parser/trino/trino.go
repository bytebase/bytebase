// Package trino provides SQL parser for Trino.
package trino

import (
	"strings"
	"unicode"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/trino-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// ParseResult is the result of parsing a Trino statement.
type ParseResult struct {
	Tree   antlr.Tree
	Tokens *antlr.CommonTokenStream
}

// ParseTrino parses the given SQL and returns the ParseResult.
func ParseTrino(sql string) (*ParseResult, error) {
	// Add a semicolon if it's missing to allow users to omit the semicolon
	trimmedSQL := strings.TrimRightFunc(sql, unicode.IsSpace)
	if len(trimmedSQL) > 0 && !strings.HasSuffix(trimmedSQL, ";") {
		sql += ";"
	}

	// Create lexer and parser with error listeners
	lexer := parser.NewTrinoLexer(antlr.NewInputStream(sql))
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := parser.NewTrinoParser(stream)

	// Set up error listeners
	lexerErrorListener := &base.ParseErrorListener{
		Statement: sql,
	}
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(lexerErrorListener)

	parserErrorListener := &base.ParseErrorListener{
		Statement: sql,
	}
	p.RemoveErrorListeners()
	p.AddErrorListener(parserErrorListener)

	p.BuildParseTrees = true

	// Parse the statement
	tree := p.SingleStatement()

	// Check for errors
	if lexerErrorListener.Err != nil {
		return nil, lexerErrorListener.Err
	}

	if parserErrorListener.Err != nil {
		return nil, parserErrorListener.Err
	}

	// Return the parse result
	return &ParseResult{
		Tree:   tree,
		Tokens: stream,
	}, nil
}

// NormalizeTrinoIdentifier normalizes the identifier for Trino.
func NormalizeTrinoIdentifier(ident string) string {
	// Trino identifiers are case-insensitive unless quoted
	if strings.HasPrefix(ident, "\"") && strings.HasSuffix(ident, "\"") {
		// Remove quotes for quoted identifiers
		return ident[1 : len(ident)-1]
	}
	return strings.ToLower(ident)
}

// NormalizeTrinoTableRef returns the normalized database and table name.
func NormalizeTrinoTableRef(ctx parser.IQualifiedNameContext) (string, string, string) {
	return ExtractDatabaseSchemaName(ctx, "", "")
}

// ExtractQualifiedNameParts extracts the parts of a qualified name.
func ExtractQualifiedNameParts(ctx parser.IQualifiedNameContext) []string {
	if ctx == nil {
		return nil
	}

	var parts []string
	for _, ident := range ctx.AllIdentifier() {
		if ident != nil {
			parts = append(parts, NormalizeTrinoIdentifier(ident.GetText()))
		}
	}

	return parts
}

// ExtractDatabaseSchemaName extracts catalog/database, schema, and table/name parts from qualified name.
func ExtractDatabaseSchemaName(ctx parser.IQualifiedNameContext, defaultDatabase, defaultSchema string) (string, string, string) {
	parts := ExtractQualifiedNameParts(ctx)
	var db, schema, name string

	switch len(parts) {
	case 1:
		// Just name (table/column)
		db = defaultDatabase
		schema = defaultSchema
		name = parts[0]
	case 2:
		// schema.name
		db = defaultDatabase
		schema = parts[0]
		name = parts[1]
	case 3:
		// catalog.schema.name (Trino's model)
		db = parts[0]
		schema = parts[1]
		name = parts[2]
	default:
		// Handle invalid cases
		if len(parts) > 0 {
			name = parts[len(parts)-1]
		}
	}

	// Apply defaults if parts were missing
	if schema == "" {
		schema = defaultSchema
	}
	if db == "" {
		db = defaultDatabase
	}

	return db, schema, name
}
