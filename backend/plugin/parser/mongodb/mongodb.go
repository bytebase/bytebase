// Package mongodb provides MongoDB shell parser functionality for LSP features.
package mongodb

import (
	"github.com/antlr4-go/antlr/v4"

	"github.com/bytebase/parser/mongodb"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// ParseResult contains the result of parsing a MongoDB shell script.
type ParseResult struct {
	Tree       mongodb.IProgramContext
	Statements []StatementInfo
	Errors     []*base.SyntaxError
}

// StatementInfo contains information about a parsed statement.
type StatementInfo struct {
	StartOffset int
	EndOffset   int
	StartLine   int
	StartColumn int
	EndLine     int
	EndColumn   int
}

// ParseMongoShell parses a MongoDB shell script and returns the parse result.
func ParseMongoShell(statement string) *ParseResult {
	inputStream := antlr.NewInputStream(statement)
	lexer := mongodb.NewMongoShellLexer(inputStream)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := mongodb.NewMongoShellParser(stream)

	// Set up error listener
	lexer.RemoveErrorListeners()
	parser.RemoveErrorListeners()
	errorListener := mongodb.NewMongoShellErrorListener()
	lexer.AddErrorListener(errorListener)
	parser.AddErrorListener(errorListener)

	// Parse the input
	tree := parser.Program()

	result := &ParseResult{
		Tree: tree,
	}

	// Convert errors
	for _, err := range errorListener.Errors {
		result.Errors = append(result.Errors, &base.SyntaxError{
			Position: &storepb.Position{
				Line:   int32(err.Line),
				Column: int32(err.Column + 1), // Convert to 1-based column
			},
			RawMessage: err.Message,
			Message:    err.Message,
		})
	}

	// Extract statement information
	if tree != nil {
		for _, stmt := range tree.AllStatement() {
			if stmt == nil {
				continue
			}
			start := stmt.GetStart()
			stop := stmt.GetStop()
			if start == nil || stop == nil {
				continue
			}

			info := StatementInfo{
				StartOffset: start.GetStart(),
				EndOffset:   stop.GetStop() + 1, // End is exclusive
				StartLine:   start.GetLine(),
				StartColumn: start.GetColumn(),
				EndLine:     stop.GetLine(),
				EndColumn:   stop.GetColumn() + len(stop.GetText()),
			}
			result.Statements = append(result.Statements, info)
		}
	}

	return result
}
