package trino

import (
	"github.com/antlr4-go/antlr/v4"
	trinoparser "github.com/bytebase/parser/trino"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/tokenizer"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_TRINO, SplitSQL)
}

// SplitSQL splits the given SQL statement into multiple SQL statements.
// Following TSQL's pattern, we try parser-based splitting first, then fall back to tokenizer.
func SplitSQL(statement string) ([]base.Statement, error) {
	result, err := splitByParser(statement)
	if err != nil {
		// Fall back to tokenizer-based split
		return splitByTokenizer(statement)
	}
	return result, nil
}

func splitByTokenizer(statement string) ([]base.Statement, error) {
	t := tokenizer.NewTokenizer(statement)
	return t.SplitStandardMultiSQL()
}

func splitByParser(statement string) ([]base.Statement, error) {
	input := antlr.NewInputStream(statement)
	lexer := trinoparser.NewTrinoLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := trinoparser.NewTrinoParser(stream)

	// Remove default error listener and add our own
	lexer.RemoveErrorListeners()
	lexerErrorListener := &base.ParseErrorListener{
		Statement: statement,
	}
	lexer.AddErrorListener(lexerErrorListener)

	parser.RemoveErrorListeners()
	parserErrorListener := &base.ParseErrorListener{
		Statement: statement,
	}
	parser.AddErrorListener(parserErrorListener)

	parser.BuildParseTrees = true
	tree := parser.Parse()

	if lexerErrorListener.Err != nil {
		return nil, lexerErrorListener.Err
	}

	if parserErrorListener.Err != nil {
		return nil, parserErrorListener.Err
	}

	var result []base.Statement
	tokens := stream.GetAllTokens()

	// Walk through all statements
	for _, stmts := range tree.AllStatements() {
		if stmts == nil {
			continue
		}

		// Get SingleStatement from StatementsContext
		singleStmt := stmts.SingleStatement()
		if singleStmt == nil {
			continue
		}

		startToken := singleStmt.GetStart()
		stopToken := singleStmt.GetStop()
		if startToken == nil || stopToken == nil {
			continue
		}

		// Find the actual start position
		startIdx := startToken.GetTokenIndex()
		endIdx := stopToken.GetTokenIndex()

		// Get the first token (any channel) for baseLine calculation
		// This is important because when ANTLR reparses fragments, it sees ALL tokens
		// including comments on hidden channels, so baseLine must be from the first token
		// of any channel for correct error position mapping: originalLine = antlrLine + baseLine
		firstToken := tokens[startIdx]

		// Find the first non-hidden token for accurate start position display
		var firstDefaultToken antlr.Token
		for i := startIdx; i <= endIdx && i < len(tokens); i++ {
			if tokens[i].GetChannel() == antlr.TokenDefaultChannel {
				firstDefaultToken = tokens[i]
				break
			}
		}

		if firstDefaultToken == nil {
			firstDefaultToken = startToken
		}

		// Check if there's a semicolon after the statement and include it
		finalEndIdx := endIdx
		if endIdx+1 < len(tokens) && tokens[endIdx+1].GetTokenType() == trinoparser.TrinoLexerSEMICOLON_ {
			finalEndIdx = endIdx + 1
		}

		// Get the text including any trailing semicolon
		text := stream.GetTextFromInterval(antlr.NewInterval(startIdx, finalEndIdx))

		// Calculate proper end position (1-based exclusive per proto spec)
		endToken := tokens[finalEndIdx]

		// Use actual token positions for Range instead of cumulative offset
		// GetStart() returns byte offset of first character, GetStop() returns byte offset of last character
		rangeStart := firstDefaultToken.GetStart()
		rangeEnd := endToken.GetStop() + 1 // exclusive end

		result = append(result, base.Statement{
			Text:     text,
			BaseLine: firstToken.GetLine() - 1,
			Range: &storepb.Range{
				Start: int32(rangeStart),
				End:   int32(rangeEnd),
			},
			Start: &storepb.Position{
				Line:   int32(firstDefaultToken.GetLine()),       // 1-based (ANTLR line is already 1-based)
				Column: int32(firstDefaultToken.GetColumn() + 1), // 1-based (ANTLR column is 0-based)
			},
			End: &storepb.Position{
				Line:   int32(endToken.GetLine()),                                 // 1-based
				Column: int32(endToken.GetColumn() + len(endToken.GetText()) + 1), // 1-based exclusive
			},
			Empty: false,
		})
	}

	return result, nil
}
