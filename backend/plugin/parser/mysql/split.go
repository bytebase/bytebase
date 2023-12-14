package mysql

import (
	"io"

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
	statement = mysqlAddSemicolonIfNeeded(statement)
	var err error
	statement, err = DealWithDelimiter(statement)
	if err != nil {
		return nil, err
	}
	lexer := parser.NewMySQLLexer(antlr.NewInputStream(statement))
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)

	return splitMySQLStatement(stream)
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
			isBeginWork := base.GetDefaultChannelTokenType(tokens, i, 1) == parser.MySQLParserWORK_SYMBOL
			isBeginWork = isBeginWork || (base.GetDefaultChannelTokenType(tokens, i, 1) == parser.MySQLParserSEMICOLON_SYMBOL)
			isBeginWork = isBeginWork || (base.GetDefaultChannelTokenType(tokens, i, 1) == parser.MySQLParserEOF)
			if isBeginWork {
				continue
			}

			isXa := base.GetDefaultChannelTokenType(tokens, i, -1) == parser.MySQLParserXA_SYMBOL
			if isXa {
				continue
			}

			stack = append(stack, openParenthesis{tokenType: tokens[i].GetTokenType(), pos: i})
		case parser.MySQLParserCASE_SYMBOL:
			isEndCase := base.GetDefaultChannelTokenType(tokens, i, -1) == parser.MySQLParserEND_SYMBOL
			if isEndCase {
				continue
			}

			stack = append(stack, openParenthesis{tokenType: tokens[i].GetTokenType(), pos: i})
		case parser.MySQLParserIF_SYMBOL:
			isEndIf := base.GetDefaultChannelTokenType(tokens, i, -1) == parser.MySQLParserEND_SYMBOL
			if isEndIf {
				continue
			}

			isIfExists := base.GetDefaultChannelTokenType(tokens, i, 1) == parser.MySQLParserEXISTS_SYMBOL
			if isIfExists {
				continue
			}

			isIfNotExists := (base.GetDefaultChannelTokenType(tokens, i, 1) == parser.MySQLParserNOT_SYMBOL ||
				base.GetDefaultChannelTokenType(tokens, i, 1) == parser.MySQLParserNOT2_SYMBOL) &&
				base.GetDefaultChannelTokenType(tokens, i, 2) == parser.MySQLParserEXISTS_SYMBOL
			if isIfNotExists {
				continue
			}

			stack = append(stack, openParenthesis{tokenType: tokens[i].GetTokenType(), pos: i})
		case parser.MySQLParserLOOP_SYMBOL:
			isEndLoop := base.GetDefaultChannelTokenType(tokens, i, -1) == parser.MySQLParserEND_SYMBOL
			if isEndLoop {
				continue
			}

			stack = append(stack, openParenthesis{tokenType: tokens[i].GetTokenType(), pos: i})
		case parser.MySQLParserWHILE_SYMBOL:
			isEndWhile := base.GetDefaultChannelTokenType(tokens, i, -1) == parser.MySQLParserEND_SYMBOL
			if isEndWhile {
				continue
			}

			stack = append(stack, openParenthesis{tokenType: tokens[i].GetTokenType(), pos: i})
		case parser.MySQLParserREPEAT_SYMBOL:
			isEndRepeat := base.GetDefaultChannelTokenType(tokens, i, -1) == parser.MySQLParserUNTIL_SYMBOL
			if isEndRepeat {
				continue
			}

			stack = append(stack, openParenthesis{tokenType: tokens[i].GetTokenType(), pos: i})
		case parser.MySQLParserEND_SYMBOL:
			isXa := base.GetDefaultChannelTokenType(tokens, i, -1) == parser.MySQLParserXA_SYMBOL
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

			nextDefaultChannelTokenType := base.GetDefaultChannelTokenType(tokens, i, 1)

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

			line, col := base.FirstDefaultChannelTokenPosition(tokens[start : i+1])
			// From antlr4, the line is ONE based, and the column is ZERO based.
			// So we should minus 1 for the line.
			result = append(result, base.SingleSQL{
				Text:                 stream.GetTextFromTokens(tokens[start], tokens[i]),
				BaseLine:             tokens[start].GetLine() - 1,
				LastLine:             tokens[i].GetLine() - 1,
				LastColumn:           tokens[i].GetColumn(),
				FirstStatementLine:   line,
				FirstStatementColumn: col,
				Empty:                base.IsEmpty(tokens[start:i+1], parser.MySQLLexerSEMICOLON_SYMBOL),
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
					Empty:                base.IsEmpty(tokens[start:i], parser.MySQLLexerSEMICOLON_SYMBOL),
				})
			}
		}
	}
	return result, nil
}
