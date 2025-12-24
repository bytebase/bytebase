package trino

import (
	"strings"

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
	list, err := t.SplitStandardMultiSQL()
	if err != nil {
		return nil, err
	}

	// The tokenizer doesn't provide accurate position information.
	// We need to manually calculate positions based on the text.
	lines := strings.Split(statement, "\n")
	lineStarts := make([]int, len(lines))
	pos := 0
	for i, line := range lines {
		lineStarts[i] = pos
		pos += len(line) + 1 // +1 for newline
	}

	currentPos := 0
	for i := range list {
		if list[i].Empty {
			continue
		}

		// Find the start position
		startIdx := strings.Index(statement[currentPos:], list[i].Text)
		if startIdx == -1 {
			continue
		}
		startIdx += currentPos

		// Calculate line and column for start
		startLine, startCol := 0, 0
		for j, lineStart := range lineStarts {
			if startIdx < lineStart {
				break
			}
			startLine = j
			startCol = startIdx - lineStart
		}

		// Calculate end position
		endIdx := startIdx + len(list[i].Text)
		endLine, endCol := 0, 0
		for j, lineStart := range lineStarts {
			if endIdx <= lineStart {
				break
			}
			endLine = j
			endCol = endIdx - lineStart
		}
		// If we're exactly at the start of a new line, we're at the end of the previous line
		if endCol == 0 && endLine > 0 {
			endLine--
			if endLine < len(lines) {
				endCol = len(lines[endLine])
			}
		}

		list[i].Start = &storepb.Position{
			Line:   int32(startLine + 1),
			Column: int32(startCol + 1),
		}
		list[i].End = &storepb.Position{
			Line:   int32(endLine + 1),
			Column: int32(endCol + 1),
		}

		currentPos = endIdx
	}

	return list, nil
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

	byteOffset := 0
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
		stmtByteLength := len(text)

		// Calculate proper end position
		endToken := tokens[finalEndIdx]
		endLine := endToken.GetLine()
		endColumn := endToken.GetColumn() + len(endToken.GetText())

		result = append(result, base.Statement{
			Text: text,
			Range: &storepb.Range{
				Start: int32(byteOffset),
				End:   int32(byteOffset + stmtByteLength),
			},
			Start: &storepb.Position{
				Line:   int32(firstDefaultToken.GetLine()),
				Column: int32(firstDefaultToken.GetColumn() + 1),
			},
			End: &storepb.Position{
				Line:   int32(endLine),
				Column: int32(endColumn),
			},
			Empty: false,
		})
		byteOffset += stmtByteLength
	}

	return result, nil
}
