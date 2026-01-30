// Package mongodb provides MongoDB shell parser functionality for LSP features.
package mongodb

import (
	"github.com/antlr4-go/antlr/v4"

	"github.com/bytebase/parser/mongodb"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterParseStatementsFunc(storepb.Engine_MONGODB, ParseMongoShell)
}

// rawParseResult contains the raw result from ANTLR parsing.
// Used internally by Diagnose and statement_ranges which need raw ANTLR data.
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

// ParseMongoShell parses a MongoDB shell script and returns parsed statements
// with their ASTs. Conforms to the standard ParseStatementsFunc interface.
func ParseMongoShell(statement string) ([]base.ParsedStatement, error) {
	raw := parseMongoShellRaw(statement)
	if raw.Tree == nil {
		return nil, nil
	}

	runes := []rune(statement)
	var result []base.ParsedStatement

	for _, stmt := range raw.Tree.AllStatement() {
		if stmt == nil {
			continue
		}
		start := stmt.GetStart()
		stop := stmt.GetStop()
		if start == nil || stop == nil {
			continue
		}

		startOffset := start.GetStart()
		endOffset := min(stop.GetStop()+1, len(runes))

		text := string(runes[startOffset:endOffset])
		byteStart := len(string(runes[:startOffset]))
		byteEnd := len(string(runes[:endOffset]))

		ps := base.ParsedStatement{
			Statement: base.Statement{
				Text: text,
				Start: &storepb.Position{
					Line:   int32(start.GetLine()),
					Column: int32(start.GetColumn() + 1),
				},
				End: common.ConvertANTLRTokenToExclusiveEndPosition(
					int32(stop.GetLine()),
					int32(stop.GetColumn()),
					stop.GetText(),
				),
				Range: &storepb.Range{
					Start: int32(byteStart),
					End:   int32(byteEnd),
				},
			},
			AST: &base.ANTLRAST{
				StartPosition: &storepb.Position{
					Line:   int32(start.GetLine()),
					Column: int32(start.GetColumn() + 1),
				},
				Tree:   stmt,
				Tokens: raw.Stream,
			},
		}
		result = append(result, ps)
	}

	return result, nil
}
