package pg

import (
	"io"

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
}

// SplitMultiSQLStream splits multiSQL to stream.
func SplitMultiSQLStream(src io.Reader, f func(string) error) ([]base.SingleSQL, error) {
	text := antlr.NewIoStream(src).String()
	list, err := SplitSQL(text)
	if err != nil {
		return nil, err
	}
	var results []base.SingleSQL
	for _, sql := range list {
		if f != nil && !sql.Empty {
			if err := f(sql.Text); err != nil {
				return nil, err
			}
		}
		results = append(results, sql)
	}
	return results, nil
}

// SplitSQL splits the given SQL statement into multiple SQL statements.
func SplitSQL(statement string) ([]base.SingleSQL, error) {
	lexer := parser.NewPostgreSQLLexer(antlr.NewInputStream(statement))
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	return splitSQLImpl(stream)
}

func splitSQLImpl(stream *antlr.CommonTokenStream) ([]base.SingleSQL, error) {
	var result []base.SingleSQL
	stream.Fill()
	tokens := stream.GetAllTokens()
	start := 0
	// Splitting multiple statements by semicolon symbol should consider the special case.
	// For create function/procedure statements, the semicolon symbol is used as part of the statement.
	// We should skip the semicolon symbol between the "BEGIN ATOMIC" and "END" keywords.
	// So this is a parenthesis matching problem.
	type openParenthesis struct {
		tokenType int
		pos       int
	}
	var stack []openParenthesis
	for i := 0; i < len(tokens); i++ {
		switch tokens[i].GetTokenType() {
		case parser.PostgreSQLLexerBEGIN_P:
			if isBeginTransaction(tokens, i) {
				continue
			}

			stack = append(stack, openParenthesis{
				tokenType: tokens[i].GetTokenType(),
				pos:       i,
			})
		case parser.PostgreSQLLexerCASE:
			stack = append(stack, openParenthesis{
				tokenType: tokens[i].GetTokenType(),
				pos:       i,
			})
		case parser.PostgreSQLLexerEND_P:
			if isEndTransaction(tokens, i) {
				continue
			}

			if len(stack) == 0 {
				return nil, errors.New("invalid statement: failed to split multiple statements")
			}

			nextToken := base.GetDefaultChannelTokenType(tokens, i, 1)
			switch nextToken {
			case parser.PostgreSQLLexerCASE:
				if stack[len(stack)-1].tokenType != parser.PostgreSQLLexerCASE {
					// Backtracking the process.
					i = stack[len(stack)-1].pos
					stack = stack[:len(stack)-1]
					continue
				}
				stack = stack[:len(stack)-1]
				continue
			case parser.PostgreSQLLexerIF_P:
				if stack[len(stack)-1].tokenType != parser.PostgreSQLLexerIF_P {
					// Backtracking the process.
					i = stack[len(stack)-1].pos
					stack = stack[:len(stack)-1]
					continue
				}
				stack = stack[:len(stack)-1]
				continue
			case parser.PostgreSQLLexerLOOP:
				if stack[len(stack)-1].tokenType != parser.PostgreSQLLexerLOOP {
					// Backtracking the process.
					i = stack[len(stack)-1].pos
					stack = stack[:len(stack)-1]
					continue
				}
				stack = stack[:len(stack)-1]
				continue
			default:
				leftTokenType := stack[len(stack)-1].tokenType
				if leftTokenType != parser.PostgreSQLLexerBEGIN_P && leftTokenType != parser.PostgreSQLLexerCASE {
					// Backtracking the process.
					i = stack[len(stack)-1].pos
					stack = stack[:len(stack)-1]
					continue
				}
				stack = stack[:len(stack)-1]
			}
		case parser.PostgreSQLLexerSEMI:
			if len(stack) > 0 {
				continue
			}

			line, col := base.FirstDefaultChannelTokenPosition(tokens[start:i])
			// From antlr4, the line is ONE based, and the column is ZERO based.
			// So we should minus 1 for the line.
			result = append(result, base.SingleSQL{
				Text:                 stream.GetTextFromTokens(tokens[start], tokens[i]),
				BaseLine:             tokens[start].GetLine() - 1,
				LastLine:             tokens[i].GetLine() - 1,
				LastColumn:           tokens[i].GetColumn(),
				FirstStatementLine:   line,
				FirstStatementColumn: col,
				Empty:                base.IsEmpty(tokens[start:i+1], parser.PostgreSQLLexerSEMI),
			})
			start = i + 1
		case antlr.TokenEOF:
			if len(stack) > 0 {
				// Backtracking the process.
				i = stack[len(stack)-1].pos
				stack = stack[:len(stack)-1]
				continue
			}

			if start <= i-1 {
				line, col := base.FirstDefaultChannelTokenPosition(tokens[start:i])
				// From antlr4, the line is ONE based, and the column is ZERO based.
				// So we should minus 1 for the line.
				result = append(result, base.SingleSQL{
					Text:                 stream.GetTextFromTokens(tokens[start], tokens[i-1]),
					BaseLine:             tokens[start].GetLine() - 1,
					LastLine:             tokens[i-1].GetLine() - 1,
					LastColumn:           tokens[i-1].GetColumn(),
					FirstStatementLine:   line,
					FirstStatementColumn: col,
					Empty:                base.IsEmpty(tokens[start:i], parser.PostgreSQLLexerSEMI),
				})
			}
		}
	}
	return result, nil
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
