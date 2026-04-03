package mongodb

// Temporary ANTLR compatibility shim.
// Provides parseMongoShellRaw for files not yet migrated to omni
// (diagnose.go, masking.go, statement_ranges.go).
// This file will be removed once all consumers are rewritten.

import (
	"github.com/antlr4-go/antlr/v4"

	"github.com/bytebase/parser/mongodb"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// rawParseResult contains the raw result from ANTLR parsing.
type rawParseResult struct {
	Tree   mongodb.IProgramContext
	Errors []*base.SyntaxError
	Stream *antlr.CommonTokenStream
}

// parseMongoShellRaw performs the raw ANTLR parsing and returns the program tree,
// token stream, and any syntax errors.
func parseMongoShellRaw(statement string) *rawParseResult {
	inputStream := antlr.NewInputStream(statement)
	lexer := mongodb.NewMongoShellLexer(inputStream)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := mongodb.NewMongoShellParser(stream)

	lexer.RemoveErrorListeners()
	p.RemoveErrorListeners()
	errorListener := mongodb.NewMongoShellErrorListener()
	lexer.AddErrorListener(errorListener)
	p.AddErrorListener(errorListener)

	tree := p.Program()

	result := &rawParseResult{
		Tree:   tree,
		Stream: stream,
	}

	for _, err := range errorListener.Errors {
		result.Errors = append(result.Errors, &base.SyntaxError{
			Position: &storepb.Position{
				Line:   int32(err.Line),
				Column: int32(err.Column + 1),
			},
			RawMessage: err.Message,
			Message:    err.Message,
		})
	}

	return result
}
