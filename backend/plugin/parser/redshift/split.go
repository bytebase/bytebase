package redshift

import (
	"log/slog"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	parser "github.com/bytebase/parser/redshift"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	// Unregister the PostgreSQL splitter for Redshift
	// and register the Redshift-specific splitter
	base.RegisterSplitterFunc(storepb.Engine_REDSHIFT, SplitSQL)
}

// SplitSQL splits the given SQL statement into multiple SQL statements.
func SplitSQL(statement string) ([]base.Statement, error) {
	lexer := parser.NewRedshiftLexer(antlr.NewInputStream(statement))
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	list, err := splitSQLImpl(stream, statement)
	if err != nil {
		slog.Info("failed to split Redshift statement", "statement", statement)
		// Use parser to split statement.
		return splitByParser(statement, lexer, stream)
	}
	return list, nil
}

func splitByParser(statement string, lexer *parser.RedshiftLexer, stream *antlr.CommonTokenStream) ([]base.Statement, error) {
	p := parser.NewRedshiftParser(stream)
	lexerErrorListener := &base.ParseErrorListener{
		Statement: statement,
	}
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(lexerErrorListener)

	parserErrorListener := &base.ParseErrorListener{
		Statement: statement,
	}
	p.RemoveErrorListeners()
	p.AddErrorListener(parserErrorListener)

	p.BuildParseTrees = true

	tree := p.Root()
	if lexerErrorListener.Err != nil {
		return nil, lexerErrorListener.Err
	}

	if parserErrorListener.Err != nil {
		return nil, parserErrorListener.Err
	}

	if tree == nil || tree.Stmtblock() == nil || tree.Stmtblock().Stmtmulti() == nil {
		return nil, errors.New("failed to split multiple statements")
	}

	var result []base.Statement
	tokens := stream.GetAllTokens()

	byteOffset := 0
	start := 0
	for _, semi := range tree.Stmtblock().Stmtmulti().AllSEMI() {
		pos := semi.GetSymbol().GetStart()
		stmtText := stream.GetTextFromTokens(tokens[start], tokens[pos])
		stmtByteLength := len(stmtText)

		// Calculate start position from byte offset (first character of Text)
		startLine, startColumn := base.CalculateLineAndColumn(statement, byteOffset)
		result = append(result, base.Statement{
			Text: stmtText,
			Range: &storepb.Range{
				Start: int32(byteOffset),
				End:   int32(byteOffset + stmtByteLength),
			},
			End: common.ConvertANTLRTokenToExclusiveEndPosition(
				int32(tokens[pos].GetLine()),
				int32(tokens[pos].GetColumn()),
				tokens[pos].GetText(),
			),
			Start: &storepb.Position{
				Line:   int32(startLine + 1),
				Column: int32(startColumn + 1),
			},
			Empty: base.IsEmpty(tokens[start:pos+1], parser.RedshiftParserSEMI),
		})
		byteOffset += stmtByteLength
		start = pos + 1
	}
	// For the last statement, it may not end with semicolon symbol, EOF symbol instead.
	eofPos := len(tokens) - 1
	if start < eofPos {
		stmtText := stream.GetTextFromTokens(tokens[start], tokens[eofPos-1])
		stmtByteLength := len(stmtText)

		// Calculate start position from byte offset (first character of Text)
		startLine, startColumn := base.CalculateLineAndColumn(statement, byteOffset)
		result = append(result, base.Statement{
			Text: stmtText,
			Range: &storepb.Range{
				Start: int32(byteOffset),
				End:   int32(byteOffset + stmtByteLength),
			},
			End: common.ConvertANTLRTokenToExclusiveEndPosition(
				int32(tokens[eofPos-1].GetLine()),
				int32(tokens[eofPos-1].GetColumn()),
				tokens[eofPos-1].GetText(),
			),
			Start: &storepb.Position{
				Line:   int32(startLine + 1),
				Column: int32(startColumn + 1),
			},
			Empty: base.IsEmpty(tokens[start:eofPos], parser.RedshiftParserSEMI),
		})
	}
	return result, nil
}

type openParenthesis struct {
	tokenType int
	pos       int
}

func splitSQLImpl(stream *antlr.CommonTokenStream, statement string) ([]base.Statement, error) {
	var result []base.Statement
	stream.Fill()
	tokens := stream.GetAllTokens()

	var beginCaseStack, ifStack, loopStack []openParenthesis
	var semicolonStack []int

	for i, token := range tokens {
		switch token.GetTokenType() {
		case parser.RedshiftParserBEGIN_P:
			if isBeginTransaction(tokens, i) {
				continue
			}

			beginCaseStack = append(beginCaseStack, openParenthesis{
				tokenType: token.GetTokenType(),
				pos:       i,
			})
		case parser.RedshiftParserCASE:
			beginCaseStack = append(beginCaseStack, openParenthesis{
				tokenType: token.GetTokenType(),
				pos:       i,
			})
		case parser.RedshiftParserIF_P:
			ifStack = append(ifStack, openParenthesis{
				tokenType: token.GetTokenType(),
				pos:       i,
			})
		case parser.RedshiftParserLOOP:
			loopStack = append(loopStack, openParenthesis{
				tokenType: token.GetTokenType(),
				pos:       i,
			})
		case parser.RedshiftParserSEMI:
			semicolonStack = append(semicolonStack, i)
		case parser.RedshiftParserEND_P:
			if isEndTransaction(tokens, i) {
				continue
			}

			nextToken := base.GetDefaultChannelTokenType(tokens, i, 1)
			switch nextToken {
			case parser.RedshiftParserIF_P:
				if len(ifStack) == 0 {
					return nil, errors.New("invalid statement: failed to split multiple statements")
				}
				// There are two cases:
				// 1. The IF statement with END IF statement.
				// 2. The IF statement without END IF statement.
				// We should match the longest IF statement, so we should check the first IF statement in the stack.
				semicolonStack = popSemicolonStack(semicolonStack, ifStack[0].pos)
				ifStack = ifStack[:len(ifStack)-1]
			case parser.RedshiftParserLOOP:
				if len(loopStack) == 0 {
					return nil, errors.New("invalid statement: failed to split multiple statements")
				}
				semicolonStack = popSemicolonStack(semicolonStack, loopStack[len(semicolonStack)-1].pos)
				loopStack = loopStack[:len(loopStack)-1]
			case parser.RedshiftParserCASE:
				if len(beginCaseStack) == 0 {
					return nil, errors.New("invalid statement: failed to split multiple statements")
				}
				semicolonStack = popSemicolonStack(semicolonStack, beginCaseStack[len(beginCaseStack)-1].pos)
				beginCaseStack = beginCaseStack[:len(beginCaseStack)-1]
			default:
				if len(beginCaseStack) == 0 {
					return nil, errors.New("invalid statement: failed to split multiple statements")
				}
				semicolonStack = popSemicolonStack(semicolonStack, beginCaseStack[len(beginCaseStack)-1].pos)
				beginCaseStack = beginCaseStack[:len(beginCaseStack)-1]
			}
		default:
		}
	}

	byteOffset := 0
	start := 0
	for _, pos := range semicolonStack {
		stmtText := stream.GetTextFromTokens(tokens[start], tokens[pos])
		stmtByteLength := len(stmtText)

		// Calculate start position from byte offset (first character of Text)
		startLine, startColumn := base.CalculateLineAndColumn(statement, byteOffset)
		result = append(result, base.Statement{
			Text: stmtText,
			Range: &storepb.Range{
				Start: int32(byteOffset),
				End:   int32(byteOffset + stmtByteLength),
			},
			End: common.ConvertANTLRTokenToExclusiveEndPosition(
				int32(tokens[pos].GetLine()),
				int32(tokens[pos].GetColumn()),
				tokens[pos].GetText(),
			),
			Start: &storepb.Position{
				Line:   int32(startLine + 1),
				Column: int32(startColumn + 1),
			},
			Empty: base.IsEmpty(tokens[start:pos+1], parser.RedshiftParserSEMI),
		})
		byteOffset += stmtByteLength
		start = pos + 1
	}
	// For the last statement, it may not end with semicolon symbol, EOF symbol instead.
	eofPos := len(tokens) - 1
	if start < eofPos {
		stmtText := stream.GetTextFromTokens(tokens[start], tokens[eofPos-1])
		stmtByteLength := len(stmtText)

		// Calculate start position from byte offset (first character of Text)
		startLine, startColumn := base.CalculateLineAndColumn(statement, byteOffset)
		result = append(result, base.Statement{
			Text: stmtText,
			Range: &storepb.Range{
				Start: int32(byteOffset),
				End:   int32(byteOffset + stmtByteLength),
			},
			End: common.ConvertANTLRTokenToExclusiveEndPosition(
				int32(tokens[eofPos-1].GetLine()),
				int32(tokens[eofPos-1].GetColumn()),
				tokens[eofPos-1].GetText(),
			),
			Start: &storepb.Position{
				Line:   int32(startLine + 1),
				Column: int32(startColumn + 1),
			},
			Empty: base.IsEmpty(tokens[start:eofPos], parser.RedshiftParserSEMI),
		})
	}

	return result, nil
}

func popSemicolonStack(semicolonStack []int, pos int) []int {
	if len(semicolonStack) == 0 {
		return semicolonStack
	}

	for i := len(semicolonStack) - 1; i >= 0; i-- {
		if semicolonStack[i] < pos {
			return semicolonStack[:i+1]
		}
	}

	return []int{}
}

func isEndTransaction(tokens []antlr.Token, index int) bool {
	if tokens[index].GetTokenType() != parser.RedshiftLexerEND_P {
		return false
	}

	switch base.GetDefaultChannelTokenType(tokens, index, 1) {
	case parser.RedshiftParserTRANSACTION,
		parser.RedshiftParserWORK,
		parser.RedshiftParserSEMI:
		return true
	case parser.RedshiftParserAND:
		if base.GetDefaultChannelTokenType(tokens, index, 2) == parser.RedshiftParserNO {
			return base.GetDefaultChannelTokenType(tokens, index, 3) == parser.RedshiftParserCHAIN
		}
		return base.GetDefaultChannelTokenType(tokens, index, 2) == parser.RedshiftParserCHAIN
	default:
		return false
	}
}

func isBeginTransaction(tokens []antlr.Token, index int) bool {
	if tokens[index].GetTokenType() != parser.RedshiftLexerBEGIN_P {
		return false
	}

	switch base.GetDefaultChannelTokenType(tokens, index, 1) {
	case parser.RedshiftParserTRANSACTION,
		parser.RedshiftParserWORK,
		parser.RedshiftParserSEMI,
		parser.RedshiftParserISOLATION,
		parser.RedshiftParserREAD,
		parser.RedshiftLexerDEFERRABLE:
		return true
	case parser.RedshiftParserNOT:
		return base.GetDefaultChannelTokenType(tokens, index, 2) == parser.RedshiftParserDEFERRABLE
	default:
		return false
	}
}
