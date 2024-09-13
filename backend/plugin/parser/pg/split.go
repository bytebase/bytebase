package pg

import (
	"log/slog"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	parser "github.com/bytebase/postgresql-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_POSTGRES, SplitSQL)
	base.RegisterSplitterFunc(storepb.Engine_REDSHIFT, SplitSQL)
	base.RegisterSplitterFunc(storepb.Engine_RISINGWAVE, SplitSQL)
	base.RegisterSplitterFunc(storepb.Engine_COCKROACHDB, SplitSQL)
}

// SplitSQL splits the given SQL statement into multiple SQL statements.
func SplitSQL(statement string) ([]base.SingleSQL, error) {
	lexer := parser.NewPostgreSQLLexer(antlr.NewInputStream(statement))
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	list, err := splitSQLImpl(stream)
	if err != nil {
		slog.Info("failed to split PostgreSQL statement", "statement", statement)
		// Use parser to split statement.
		return splitByParser(lexer, stream)
	}
	return list, nil
}

func splitByParser(lexer *parser.PostgreSQLLexer, stream *antlr.CommonTokenStream) ([]base.SingleSQL, error) {
	p := parser.NewPostgreSQLParser(stream)
	lexerErrorListener := &base.ParseErrorListener{}
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(lexerErrorListener)

	parserErrorListener := &base.ParseErrorListener{}
	p.RemoveErrorListeners()
	p.AddErrorListener(parserErrorListener)

	p.BuildParseTrees = true
	p.SetErrorHandler(antlr.NewBailErrorStrategy())

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

	var result []base.SingleSQL
	tokens := stream.GetAllTokens()

	start := 0
	for _, semi := range tree.Stmtblock().Stmtmulti().AllSEMI() {
		pos := semi.GetSymbol().GetStart()
		line, col := base.FirstDefaultChannelTokenPosition(tokens[start : pos+1])
		// From antlr4, the line is ONE based, and the column is ZERO based.
		// So we should minus 1 for the line.
		result = append(result, base.SingleSQL{
			Text:                 stream.GetTextFromTokens(tokens[start], tokens[pos]),
			BaseLine:             tokens[start].GetLine() - 1,
			LastLine:             tokens[pos].GetLine() - 1,
			LastColumn:           tokens[pos].GetColumn(),
			FirstStatementLine:   line,
			FirstStatementColumn: col,
			Empty:                base.IsEmpty(tokens[start:pos+1], parser.PostgreSQLParserSEMI),
		})
		start = pos + 1
	}
	// For the last statement, it may not end with semicolon symbol, EOF symbol instead.
	eofPos := len(tokens) - 1
	if start < eofPos {
		line, col := base.FirstDefaultChannelTokenPosition(tokens[start:])
		// From antlr4, the line is ONE based, and the column is ZERO based.
		// So we should minus 1 for the line.
		result = append(result, base.SingleSQL{
			Text:                 stream.GetTextFromTokens(tokens[start], tokens[eofPos-1]),
			BaseLine:             tokens[start].GetLine() - 1,
			LastLine:             tokens[eofPos-1].GetLine() - 1,
			LastColumn:           tokens[eofPos-1].GetColumn(),
			FirstStatementLine:   line,
			FirstStatementColumn: col,
			Empty:                base.IsEmpty(tokens[start:eofPos], parser.PostgreSQLParserSEMI),
		})
	}
	return result, nil
}

type openParenthesis struct {
	tokenType int
	pos       int
}

func splitSQLImpl(stream *antlr.CommonTokenStream) ([]base.SingleSQL, error) {
	var result []base.SingleSQL
	stream.Fill()
	tokens := stream.GetAllTokens()

	var beginCaseStack, ifStack, loopStack []openParenthesis
	var semicolonStack []int

	for i, token := range tokens {
		switch token.GetTokenType() {
		case parser.PostgreSQLParserBEGIN_P:
			if isBeginTransaction(tokens, i) {
				continue
			}

			beginCaseStack = append(beginCaseStack, openParenthesis{
				tokenType: token.GetTokenType(),
				pos:       i,
			})
		case parser.PostgreSQLParserCASE:
			beginCaseStack = append(beginCaseStack, openParenthesis{
				tokenType: token.GetTokenType(),
				pos:       i,
			})
		case parser.PostgreSQLParserIF_P:
			ifStack = append(ifStack, openParenthesis{
				tokenType: token.GetTokenType(),
				pos:       i,
			})
		case parser.PostgreSQLParserLOOP:
			loopStack = append(loopStack, openParenthesis{
				tokenType: token.GetTokenType(),
				pos:       i,
			})
		case parser.PostgreSQLParserSEMI:
			semicolonStack = append(semicolonStack, i)
		case parser.PostgreSQLParserEND_P:
			if isEndTransaction(tokens, i) {
				continue
			}

			nextToken := base.GetDefaultChannelTokenType(tokens, i, 1)
			switch nextToken {
			case parser.PostgreSQLParserIF_P:
				if len(ifStack) == 0 {
					return nil, errors.New("invalid statement: failed to split multiple statements")
				}
				// There are two cases:
				// 1. The IF statement with END IF statement.
				// 2. The IF statement without END IF statement.
				// We should match the longest IF statement, so we should check the first IF statement in the stack.
				semicolonStack = popSemicolonStack(semicolonStack, ifStack[0].pos)
				ifStack = ifStack[:len(ifStack)-1]
			case parser.PostgreSQLParserLOOP:
				if len(loopStack) == 0 {
					return nil, errors.New("invalid statement: failed to split multiple statements")
				}
				semicolonStack = popSemicolonStack(semicolonStack, loopStack[len(semicolonStack)-1].pos)
				loopStack = loopStack[:len(loopStack)-1]
			case parser.PostgreSQLParserCASE:
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
		}
	}

	start := 0
	for _, pos := range semicolonStack {
		line, col := base.FirstDefaultChannelTokenPosition(tokens[start : pos+1])
		// From antlr4, the line is ONE based, and the column is ZERO based.
		// So we should minus 1 for the line.
		result = append(result, base.SingleSQL{
			Text:                 stream.GetTextFromTokens(tokens[start], tokens[pos]),
			BaseLine:             tokens[start].GetLine() - 1,
			LastLine:             tokens[pos].GetLine() - 1,
			LastColumn:           tokens[pos].GetColumn(),
			FirstStatementLine:   line,
			FirstStatementColumn: col,
			Empty:                base.IsEmpty(tokens[start:pos+1], parser.PostgreSQLParserSEMI),
		})
		start = pos + 1
	}
	// For the last statement, it may not end with semicolon symbol, EOF symbol instead.
	eofPos := len(tokens) - 1
	if start < eofPos {
		line, col := base.FirstDefaultChannelTokenPosition(tokens[start:])
		// From antlr4, the line is ONE based, and the column is ZERO based.
		// So we should minus 1 for the line.
		result = append(result, base.SingleSQL{
			Text:                 stream.GetTextFromTokens(tokens[start], tokens[eofPos-1]),
			BaseLine:             tokens[start].GetLine() - 1,
			LastLine:             tokens[eofPos-1].GetLine() - 1,
			LastColumn:           tokens[eofPos-1].GetColumn(),
			FirstStatementLine:   line,
			FirstStatementColumn: col,
			Empty:                base.IsEmpty(tokens[start:eofPos], parser.PostgreSQLParserSEMI),
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
	if tokens[index].GetTokenType() != parser.PostgreSQLLexerEND_P {
		return false
	}

	switch base.GetDefaultChannelTokenType(tokens, index, 1) {
	case parser.PostgreSQLParserTRANSACTION,
		parser.PostgreSQLParserWORK,
		parser.PostgreSQLParserSEMI:
		return true
	case parser.PostgreSQLParserAND:
		if base.GetDefaultChannelTokenType(tokens, index, 2) == parser.PostgreSQLParserNO {
			return base.GetDefaultChannelTokenType(tokens, index, 3) == parser.PostgreSQLParserCHAIN
		}
		return base.GetDefaultChannelTokenType(tokens, index, 2) == parser.PostgreSQLParserCHAIN
	default:
		return false
	}
}

func isBeginTransaction(tokens []antlr.Token, index int) bool {
	if tokens[index].GetTokenType() != parser.PostgreSQLLexerBEGIN_P {
		return false
	}

	switch base.GetDefaultChannelTokenType(tokens, index, 1) {
	case parser.PostgreSQLParserTRANSACTION,
		parser.PostgreSQLParserWORK,
		parser.PostgreSQLParserSEMI,
		parser.PostgreSQLParserISOLATION,
		parser.PostgreSQLParserREAD,
		parser.PostgreSQLLexerDEFERRABLE:
		return true
	case parser.PostgreSQLParserNOT:
		return base.GetDefaultChannelTokenType(tokens, index, 2) == parser.PostgreSQLParserDEFERRABLE
	default:
		return false
	}
}
