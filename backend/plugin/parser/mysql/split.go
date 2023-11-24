package mysql

import (
	"io"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/mysql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_MYSQL, SplitSQL)
	base.RegisterSplitterFunc(storepb.Engine_MARIADB, SplitSQL)
	base.RegisterSplitterFunc(storepb.Engine_OCEANBASE, SplitSQL)
}

// SplitSQL splits the given SQL statement into multiple SQL statements.
func SplitSQL(statement string) ([]base.SingleSQL, error) {
	statement = strings.TrimRight(statement, " \r\n\t\f;") + "\n;"
	var err error
	statement, err = DealWithDelimiter(statement)
	if err != nil {
		return nil, err
	}
	lexer := parser.NewMySQLLexer(antlr.NewInputStream(statement))
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)

	list, err := splitMySQLStatement(stream)
	if err != nil {
		return nil, err
	}
	var results []base.SingleSQL
	for _, sql := range list {
		if sql.Empty {
			continue
		}
		results = append(results, sql)
	}
	return results, nil
}

// SplitMultiSQLStream splits MySQL multiSQL to stream.
// Note that the reader is read completely into memory and so it must actually
// have a stopping point - you cannot pass in a reader on an open-ended source such
// as a socket for instance.
func SplitMultiSQLStream(src io.Reader, f func(string) error) ([]base.SingleSQL, error) {
	result, err := SplitMySQLStream(src)
	if err != nil {
		return nil, err
	}

	for _, sql := range result {
		if f != nil {
			if err := f(sql.Text); err != nil {
				return nil, err
			}
		}
	}

	return result, nil
}

// SplitMySQLStream splits the given SQL stream into multiple SQL statements.
// Note that the reader is read completely into memory and so it must actually
// have a stopping point - you cannot pass in a reader on an open-ended source such
// as a socket for instance.
func SplitMySQLStream(src io.Reader) ([]base.SingleSQL, error) {
	text := antlr.NewIoStream(src).String()
	return SplitSQL(text)
}

func splitMySQLStatement(stream *antlr.CommonTokenStream) ([]base.SingleSQL, error) {
	var result []base.SingleSQL
	stream.Fill()
	tokens := stream.GetAllTokens()
	start := 0
	// Splitting multiple statements by semicolon symbol should consider the special case.
	// For CASE/REPLACE/IF/LOOP/WHILE/REPEAT statement, the semicolon symbol is not the end of the statement.
	// So we should skip the semicolon symbol in these statements.
	// These statements are begin with BEGIN/REPLACE/IF/LOOP/WHILE/REPEAT symbol and end with END/END REPLACE/END IF/END LOOP/END WHILE/END REPEAT symbol.
	// So this is a parenthesis matching problem.
	type openParenthesis struct {
		tokenType int
		pos       int
	}
	var stack []openParenthesis
	for i := 0; i < len(tokens); i++ {
		switch tokens[i].GetTokenType() {
		case parser.MySQLParserBEGIN_SYMBOL:
			isBeginWork := getDefaultChannelTokenType(tokens, i, 1) == parser.MySQLParserWORK_SYMBOL
			isBeginWork = isBeginWork || (getDefaultChannelTokenType(tokens, i, 1) == parser.MySQLParserSEMICOLON_SYMBOL)
			isBeginWork = isBeginWork || (getDefaultChannelTokenType(tokens, i, 1) == parser.MySQLParserEOF)
			if isBeginWork {
				continue
			}

			isXa := getDefaultChannelTokenType(tokens, i, -1) == parser.MySQLParserXA_SYMBOL
			if isXa {
				continue
			}

			stack = append(stack, openParenthesis{tokenType: tokens[i].GetTokenType(), pos: i})
		case parser.MySQLParserCASE_SYMBOL:
			isEndCase := getDefaultChannelTokenType(tokens, i, -1) == parser.MySQLParserEND_SYMBOL
			if isEndCase {
				continue
			}

			stack = append(stack, openParenthesis{tokenType: tokens[i].GetTokenType(), pos: i})
		case parser.MySQLParserIF_SYMBOL:
			isEndIf := getDefaultChannelTokenType(tokens, i, -1) == parser.MySQLParserEND_SYMBOL
			if isEndIf {
				continue
			}

			isIfExists := getDefaultChannelTokenType(tokens, i, 1) == parser.MySQLParserEXISTS_SYMBOL
			if isIfExists {
				continue
			}

			isIfNotExists := (getDefaultChannelTokenType(tokens, i, 1) == parser.MySQLParserNOT_SYMBOL ||
				getDefaultChannelTokenType(tokens, i, 1) == parser.MySQLParserNOT2_SYMBOL) &&
				getDefaultChannelTokenType(tokens, i, 2) == parser.MySQLParserEXISTS_SYMBOL
			if isIfNotExists {
				continue
			}

			stack = append(stack, openParenthesis{tokenType: tokens[i].GetTokenType(), pos: i})
		case parser.MySQLParserLOOP_SYMBOL:
			isEndLoop := getDefaultChannelTokenType(tokens, i, -1) == parser.MySQLParserEND_SYMBOL
			if isEndLoop {
				continue
			}

			stack = append(stack, openParenthesis{tokenType: tokens[i].GetTokenType(), pos: i})
		case parser.MySQLParserWHILE_SYMBOL:
			isEndWhile := getDefaultChannelTokenType(tokens, i, -1) == parser.MySQLParserEND_SYMBOL
			if isEndWhile {
				continue
			}

			stack = append(stack, openParenthesis{tokenType: tokens[i].GetTokenType(), pos: i})
		case parser.MySQLParserREPEAT_SYMBOL:
			isEndRepeat := getDefaultChannelTokenType(tokens, i, -1) == parser.MySQLParserUNTIL_SYMBOL
			if isEndRepeat {
				continue
			}

			stack = append(stack, openParenthesis{tokenType: tokens[i].GetTokenType(), pos: i})
		case parser.MySQLParserEND_SYMBOL:
			isXa := getDefaultChannelTokenType(tokens, i, -1) == parser.MySQLParserXA_SYMBOL
			if isXa {
				continue
			}

			// There are some special case for IF and REPEAT statement.
			// MySQL has two functions: IF(expr1,expr2,expr3) and REPEAT(str,count).
			// So we may meet single IF/REPEAT symbol without END IF/REPEAT symbol.
			// For these cases, we will see the END XXX symbol is not matched with the top of the stack.
			// We should skip these single IF/REPEAT symbol and backtracking these processes after IF/REPEAT symbol.
			if len(stack) == 0 {
				return nil, errors.New("invalid statement: failed to split multiple statements")
			}

			nextDefaultChannelTokenType := getDefaultChannelTokenType(tokens, i, 1)

			isEndIf := nextDefaultChannelTokenType == parser.MySQLParserIF_SYMBOL
			if isEndIf {
				if stack[len(stack)-1].tokenType != parser.MySQLParserIF_SYMBOL {
					// Backtracking the process.
					i = stack[len(stack)-1].pos
					stack = stack[:len(stack)-1]
					continue
				}
				stack = stack[:len(stack)-1]
				continue
			}

			isEndCase := nextDefaultChannelTokenType == parser.MySQLParserCASE_SYMBOL
			if isEndCase {
				if stack[len(stack)-1].tokenType != parser.MySQLParserCASE_SYMBOL {
					// Backtracking the process.
					i = stack[len(stack)-1].pos
					stack = stack[:len(stack)-1]
					continue
				}
				stack = stack[:len(stack)-1]
				continue
			}

			isEndLoop := nextDefaultChannelTokenType == parser.MySQLParserLOOP_SYMBOL
			if isEndLoop {
				if stack[len(stack)-1].tokenType != parser.MySQLParserLOOP_SYMBOL {
					// Backtracking the process.
					i = stack[len(stack)-1].pos
					stack = stack[:len(stack)-1]
					continue
				}
				stack = stack[:len(stack)-1]
				continue
			}

			isEndWhile := nextDefaultChannelTokenType == parser.MySQLParserWHILE_SYMBOL
			if isEndWhile {
				if stack[len(stack)-1].tokenType != parser.MySQLParserWHILE_SYMBOL {
					// Backtracking the process.
					i = stack[len(stack)-1].pos
					stack = stack[:len(stack)-1]
					continue
				}
				stack = stack[:len(stack)-1]
				continue
			}

			isEndRepeat := nextDefaultChannelTokenType == parser.MySQLParserREPEAT_SYMBOL
			if isEndRepeat {
				if stack[len(stack)-1].tokenType != parser.MySQLParserREPEAT_SYMBOL {
					// Backtracking the process.
					i = stack[len(stack)-1].pos
					stack = stack[:len(stack)-1]
					continue
				}
				stack = stack[:len(stack)-1]
				continue
			}

			// is BEGIN ... END or CASE .. END case
			leftTokenType := stack[len(stack)-1].tokenType
			if leftTokenType != parser.MySQLParserBEGIN_SYMBOL && leftTokenType != parser.MySQLParserCASE_SYMBOL {
				// Backtracking the process.
				i = stack[len(stack)-1].pos
				stack = stack[:len(stack)-1]
				continue
			}
			stack = stack[:len(stack)-1]
		case parser.MySQLParserSEMICOLON_SYMBOL:
			if len(stack) != 0 {
				continue
			}

			line, col := firstDefaultChannelTokenPosition(tokens[start : i+1])
			result = append(result, base.SingleSQL{
				Text:                 stream.GetTextFromTokens(tokens[start], tokens[i]),
				BaseLine:             tokens[start].GetLine() - 1,
				LastLine:             tokens[i].GetLine(),
				LastColumn:           tokens[i].GetColumn(),
				FirstStatementLine:   line,
				FirstStatementColumn: col,
			})
			start = i + 1
		case parser.MySQLParserEOF:
			if len(stack) != 0 {
				// Backtracking the process.
				i = stack[len(stack)-1].pos
				stack = stack[:len(stack)-1]
				continue
			}

			if start <= i-1 {
				line, col := firstDefaultChannelTokenPosition(tokens[start:i])
				result = append(result, base.SingleSQL{
					Text:                 stream.GetTextFromTokens(tokens[start], tokens[i-1]),
					BaseLine:             tokens[start].GetLine() - 1,
					LastLine:             tokens[i-1].GetLine(),
					LastColumn:           tokens[i-1].GetColumn(),
					FirstStatementLine:   line,
					FirstStatementColumn: col,
				})
			}
		}
	}
	return result, nil
}

func firstDefaultChannelTokenPosition(tokens []antlr.Token) (int, int) {
	for _, token := range tokens {
		if token.GetChannel() == antlr.TokenDefaultChannel {
			return token.GetLine(), token.GetColumn()
		}
	}
	return tokens[len(tokens)-1].GetLine(), tokens[len(tokens)-1].GetColumn()
}
