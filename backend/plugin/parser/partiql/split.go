package partiql

import (
	"errors"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/partiql-parser"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_DYNAMODB, SplitSQL)
}

func SplitSQL(statement string) ([]base.SingleSQL, error) {
	lexer := parser.NewPartiQLLexer(antlr.NewInputStream(statement))
	lexerErrorListener := &base.ParseErrorListener{
		Statement: statement,
	}
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(lexerErrorListener)

	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)

	p := parser.NewPartiQLParserParser(stream)
	parserErrorListener := &base.ParseErrorListener{
		Statement: statement,
	}
	p.RemoveErrorListeners()
	p.AddErrorListener(parserErrorListener)

	p.BuildParseTrees = true

	tree := p.Script()
	if lexerErrorListener.Err != nil {
		return nil, lexerErrorListener.Err
	}

	if parserErrorListener.Err != nil {
		return nil, parserErrorListener.Err
	}

	if tree == nil {
		return nil, errors.New("failed to split multiple statements")
	}

	var result []base.SingleSQL
	tokens := stream.GetAllTokens()

	start := 0
	for _, semi := range tree.AllCOLON_SEMI() {
		pos := semi.GetSymbol().GetTokenIndex()
		antlrPosition := base.FirstDefaultChannelTokenPosition(tokens[start : pos+1])
		// From antlr4, the line is ONE based, and the column is ZERO based.
		// So we should minus 1 for the line.
		result = append(result, base.SingleSQL{
			Text:     stream.GetTextFromTokens(tokens[start], tokens[pos]),
			BaseLine: tokens[start].GetLine() - 1,
			End:      common.ConvertANTLRPositionToPosition(&common.ANTLRPosition{Line: int32(tokens[pos].GetLine()), Column: int32(tokens[pos].GetColumn())}, statement),
			Start:    common.ConvertANTLRPositionToPosition(antlrPosition, statement),
			Empty:    base.IsEmpty(tokens[start:pos+1], parser.PartiQLLexerCOLON_SEMI),
		})
		start = pos + 1
	}
	// For the last statement, it may not end with semicolon symbol, EOF symbol instead.
	eofPos := len(tokens) - 1
	if start < eofPos {
		antlrPosition := base.FirstDefaultChannelTokenPosition(tokens[start:])
		// From antlr4, the line is ONE based, and the column is ZERO based.
		// So we should minus 1 for the line.
		result = append(result, base.SingleSQL{
			Text:     stream.GetTextFromTokens(tokens[start], tokens[eofPos-1]),
			BaseLine: tokens[start].GetLine() - 1,
			End:      common.ConvertANTLRPositionToPosition(&common.ANTLRPosition{Line: int32(tokens[eofPos].GetLine()), Column: int32(tokens[eofPos].GetColumn())}, statement),
			Start:    common.ConvertANTLRPositionToPosition(antlrPosition, statement),
			Empty:    base.IsEmpty(tokens[start:eofPos], parser.PartiQLLexerCOLON_SEMI),
		})
	}
	return result, nil
}
