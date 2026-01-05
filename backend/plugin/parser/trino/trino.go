// Package trino provides SQL parser for Trino.
package trino

import (
	"strings"
	"unicode"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/trino"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// ParseTrino parses the given SQL statement and returns a list of ANTLRAST.
// Each ANTLRAST represents one statement with its AST, tokens, and start position.
func ParseTrino(sql string) ([]*base.ANTLRAST, error) {
	stmts, err := SplitSQL(sql)
	if err != nil {
		return nil, errors.Wrap(err, "failed to split SQL")
	}

	var results []*base.ANTLRAST
	for _, stmt := range stmts {
		if stmt.Empty {
			continue
		}

		parseResult, err := parseSingleTrino(stmt.Text, stmt.BaseLine())
		if err != nil {
			return nil, err
		}
		results = append(results, parseResult)
	}

	return results, nil
}

// parseSingleTrino parses a single Trino statement and returns the ANTLRAST.
func parseSingleTrino(sql string, baseLine int) (*base.ANTLRAST, error) {
	// Add a semicolon if it's missing to allow users to omit the semicolon
	trimmedSQL := strings.TrimRightFunc(sql, unicode.IsSpace)
	if len(trimmedSQL) > 0 && !strings.HasSuffix(trimmedSQL, ";") {
		sql += ";"
	}

	// Create lexer and parser
	lexer := parser.NewTrinoLexer(antlr.NewInputStream(sql))
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := parser.NewTrinoParser(stream)

	// Set up error listeners
	startPosition := &storepb.Position{Line: int32(baseLine) + 1}
	lexerErrorListener := &base.ParseErrorListener{
		Statement:     sql,
		StartPosition: startPosition,
	}
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(lexerErrorListener)

	parserErrorListener := &base.ParseErrorListener{
		Statement:     sql,
		StartPosition: startPosition,
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

	// Return the ANTLR AST
	return &base.ANTLRAST{
		StartPosition: startPosition,
		Tree:          tree,
		Tokens:        stream,
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
// For data access control features, we need consistent catalog.schema.table.column format
func ExtractDatabaseSchemaName(ctx parser.IQualifiedNameContext, defaultDatabase, defaultSchema string) (string, string, string) {
	parts := ExtractQualifiedNameParts(ctx)

	// Set default schema if it's empty - critical for data access control test format
	if defaultSchema == "" {
		defaultSchema = "public" // Default schema for data access control tests
	}

	switch len(parts) {
	case 1:
		// Just name (table/column)
		return defaultDatabase, defaultSchema, parts[0]
	case 2:
		// schema.name
		return defaultDatabase, parts[0], parts[1]
	case 3:
		// catalog.schema.name (Trino's model)
		// In Trino, the full 3-part name is catalog.schema.table
		return parts[0], parts[1], parts[2]
	default:
		// Handle more complex cases (rare but possible in Trino)
		if len(parts) > 3 {
			// For very long identifier chains, treat the first part as catalog,
			// second as schema, and join the rest as the object name
			name := strings.Join(parts[2:], "_")
			return parts[0], parts[1], name
		}

		if len(parts) == 0 {
			// Empty qualified name, use defaults
			return defaultDatabase, defaultSchema, ""
		}

		// Invalid case - should never reach here given Trino's grammar
		// Fallback to taking the last element as the name
		name := parts[len(parts)-1]
		return defaultDatabase, defaultSchema, name
	}
}
