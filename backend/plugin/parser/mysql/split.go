package mysql

import (
	"log/slog"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/mysql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_MYSQL, SplitSQL)
	base.RegisterSplitterFunc(storepb.Engine_MARIADB, SplitSQL)
	base.RegisterSplitterFunc(storepb.Engine_OCEANBASE, SplitSQL)
}

// SplitSQL splits the given SQL statement into multiple SQL statements.
func SplitSQL(statement string) ([]base.Statement, error) {
	lexer := parser.NewMySQLLexer(antlr.NewInputStream(statement))
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)

	list, err := splitMySQLStatement(stream, statement)
	if err != nil {
		slog.Info("failed to split MySQL statement, use parser instead", "statement", statement)
		// Use parser to split statement.
		return splitByParser(statement, lexer, stream)
	}
	return list, nil
}

func splitDelimiterModeSQL(stream *antlr.CommonTokenStream, statement string) ([]base.Statement, error) {
	var result []base.Statement
	delimiter := ";"
	tokens := stream.GetAllTokens()
	start := 0

	i := 0
	for i < len(tokens) {
		token := tokens[i]
		// Deal with delimiter statement.
		if token.GetChannel() == antlr.TokenDefaultChannel && token.GetTokenType() == parser.MySQLLexerDELIMITER_SYMBOL {
			newStart, delimiterStatement := extractDelimiterStatement(stream, i)
			var err error
			delimiter, err = ExtractDelimiter(delimiterStatement)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to extract delimiter from statement: %s", delimiterStatement)
			}
			start = newStart
			i = newStart
			continue
		}

		// Deal with normal statement.
		if delimiter == ";" && token.GetTokenType() == parser.MySQLLexerSEMICOLON_SYMBOL {
			stmtText := stream.GetTextFromTokens(tokens[start], tokens[i])

			// Calculate byte range by getting actual bytes from original statement
			// This works because for ASCII chars, char offset == byte offset
			// For UTF-8, we get the actual substring which handles multi-byte chars correctly
			stmtStartChar := tokens[start].GetStart()
			stmtEndChar := tokens[i].GetStop() + 1
			stmtStartByte := len(statement[:stmtStartChar])
			stmtEndByte := len(statement[:stmtEndChar])

			// Calculate start position from byte offset (first character of Text)
			startLine, startColumn := base.CalculateLineAndColumn(statement, stmtStartByte)
			// From antlr4, the line is ONE based, and the column is ZERO based.
			// So we should minus 1 for the line.
			result = append(result, base.Statement{
				Text: stmtText,
				Range: &storepb.Range{
					Start: int32(stmtStartByte),
					End:   int32(stmtEndByte),
				},
				End: common.ConvertANTLRTokenToExclusiveEndPosition(
					int32(tokens[i].GetLine()),
					int32(tokens[i].GetColumn()),
					tokens[i].GetText(),
				),
				Start: &storepb.Position{
					Line:   int32(startLine + 1),
					Column: int32(startColumn + 1),
				},
				Empty: base.IsEmpty(tokens[start:i+1], parser.MySQLLexerSEMICOLON_SYMBOL),
			})
			i++
			start = i
			continue
		}

		if token.GetChannel() != antlr.TokenDefaultChannel {
			i++
			continue
		}

		if newStart, ok := tryMatchDelimiter(stream, i, delimiter); ok {
			stmtText := stream.GetTextFromTokens(tokens[start], tokens[i-1]) + ";"

			// Calculate byte range by getting actual bytes from original statement
			stmtStartChar := tokens[start].GetStart()
			stmtEndChar := tokens[newStart-1].GetStop() + 1
			stmtStartByte := len(statement[:stmtStartChar])
			stmtEndByte := len(statement[:stmtEndChar])

			// Calculate start position from byte offset (first character of Text)
			startLine, startColumn := base.CalculateLineAndColumn(statement, stmtStartByte)
			// From antlr4, the line is ONE based, and the column is ZERO based.
			// So we should minus 1 for the line.
			result = append(result, base.Statement{
				// Use a single semicolon instead of the user defined delimiter.
				Text: stmtText,
				Range: &storepb.Range{
					Start: int32(stmtStartByte),
					End:   int32(stmtEndByte),
				},
				End: common.ConvertANTLRTokenToExclusiveEndPosition(
					int32(tokens[newStart-1].GetLine()),
					int32(tokens[newStart-1].GetColumn()),
					tokens[newStart-1].GetText(),
				),
				Start: &storepb.Position{
					Line:   int32(startLine + 1),
					Column: int32(startColumn + 1),
				},
				Empty: base.IsEmpty(tokens[start:i], parser.MySQLLexerSEMICOLON_SYMBOL),
			})
			i = newStart
			start = newStart
			continue
		}

		i++
	}

	endPos := len(tokens) - 1
	if start < endPos {
		stmtText := stream.GetTextFromTokens(tokens[start], tokens[endPos-1])

		// Calculate byte range by getting actual bytes from original statement
		stmtStartChar := tokens[start].GetStart()
		stmtEndChar := tokens[endPos-1].GetStop() + 1
		stmtStartByte := len(statement[:stmtStartChar])
		stmtEndByte := len(statement[:stmtEndChar])

		// Calculate start position from byte offset (first character of Text)
		startLine, startColumn := base.CalculateLineAndColumn(statement, stmtStartByte)
		// From antlr4, the line is ONE based, and the column is ZERO based.
		// So we should minus 1 for the line.
		result = append(result, base.Statement{
			Text: stmtText,
			Range: &storepb.Range{
				Start: int32(stmtStartByte),
				End:   int32(stmtEndByte),
			},
			End: common.ConvertANTLRTokenToExclusiveEndPosition(
				int32(tokens[endPos-1].GetLine()),
				int32(tokens[endPos-1].GetColumn()),
				tokens[endPos-1].GetText(),
			),
			Start: &storepb.Position{
				Line:   int32(startLine + 1),
				Column: int32(startColumn + 1),
			},
			Empty: base.IsEmpty(tokens[start:endPos], parser.MySQLLexerSEMICOLON_SYMBOL),
		})
	}

	return result, nil
}

func tryMatchDelimiter(stream *antlr.CommonTokenStream, pos int, delimiter string) (int, bool) {
	matchPos := 0
	length := len(stream.GetAllTokens())
	for i := pos; i < length; i++ {
		text := stream.GetTextFromInterval(antlr.Interval{Start: i, Stop: i})
		for j := 0; j < len(text); j++ {
			if j+matchPos >= len(delimiter) || text[j] != delimiter[j+matchPos] {
				return 0, false
			}
			matchPos++
			if matchPos == len(delimiter) {
				return i + 1, true
			}
		}
	}

	return 0, false
}

func extractDelimiterStatement(stream *antlr.CommonTokenStream, pos int) (int, string) {
	length := len(stream.GetAllTokens())
	for i := pos; i < length; i++ {
		if (stream.Get(i).GetTokenType() == parser.MySQLLexerWHITESPACE && stream.Get(i).GetText() == "\n") ||
			(stream.Get(i).GetTokenType() == antlr.TokenEOF) {
			return i + 1, stream.GetTextFromTokens(stream.Get(pos), stream.Get(i-1))
		}
	}

	// never reach here
	return length, stream.GetTextFromTokens(stream.Get(pos), stream.Get(length-1))
}

func hasDelimiterStatement(stream *antlr.CommonTokenStream) bool {
	tokens := stream.GetAllTokens()
	for _, token := range tokens {
		if token.GetChannel() == antlr.TokenDefaultChannel && token.GetTokenType() == parser.MySQLLexerDELIMITER_SYMBOL {
			return true
		}
	}
	return false
}

func splitByParser(statement string, lexer *parser.MySQLLexer, stream *antlr.CommonTokenStream) ([]base.Statement, error) {
	p := parser.NewMySQLParser(stream)
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

	tree := p.Script()

	if lexerErrorListener.Err != nil {
		return nil, lexerErrorListener.Err
	}

	if parserErrorListener.Err != nil {
		return nil, parserErrorListener.Err
	}

	var result []base.Statement
	tokens := stream.GetAllTokens()

	byteOffset := 0
	start := 0
	for _, semicolon := range tree.AllSEMICOLON_SYMBOL() {
		pos := semicolon.GetSymbol().GetStart()
		stmtText := stream.GetTextFromTokens(tokens[start], tokens[pos])
		stmtByteLength := len(stmtText)

		// Calculate start position from byte offset (first character of Text)
		startLine, startColumn := base.CalculateLineAndColumn(statement, byteOffset)
		// From antlr4, the line is ONE based, and the column is ZERO based.
		// So we should minus 1 for the line.
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
			Empty: base.IsEmpty(tokens[start:pos+1], parser.MySQLLexerSEMICOLON_SYMBOL),
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
		// From antlr4, the line is ONE based, and the column is ZERO based.
		// So we should minus 1 for the line.
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
			Empty: base.IsEmpty(tokens[start:eofPos], parser.MySQLLexerSEMICOLON_SYMBOL),
		})
	}
	return result, nil
}

type openParenthesis struct {
	tokenType int
	pos       int
}

func splitMySQLStatement(stream *antlr.CommonTokenStream, statement string) ([]base.Statement, error) {
	stream.Fill()
	if hasDelimiterStatement(stream) {
		return splitDelimiterModeSQL(stream, statement)
	}

	var result []base.Statement
	tokens := stream.GetAllTokens()

	var beginCaseStack, ifStack, loopStack, whileStack, repeatStack []*openParenthesis

	var semicolonStack []int

	for i, token := range tokens {
		switch token.GetTokenType() {
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

			beginCaseStack = append(beginCaseStack, &openParenthesis{tokenType: tokens[i].GetTokenType(), pos: i})
		case parser.MySQLParserCASE_SYMBOL:
			isEndCase := base.GetDefaultChannelTokenType(tokens, i, -1) == parser.MySQLParserEND_SYMBOL
			if isEndCase {
				continue
			}

			beginCaseStack = append(beginCaseStack, &openParenthesis{tokenType: tokens[i].GetTokenType(), pos: i})
		case parser.MySQLParserIF_SYMBOL:
			isEndIf := base.GetDefaultChannelTokenType(tokens, i, -1) == parser.MySQLParserEND_SYMBOL
			if isEndIf {
				continue
			}

			isIfExists := base.GetDefaultChannelTokenType(tokens, i, 1) == parser.MySQLParserEXISTS_SYMBOL
			if isIfExists {
				continue
			}

			ifStack = append(ifStack, &openParenthesis{tokenType: tokens[i].GetTokenType(), pos: i})
		case parser.MySQLParserLOOP_SYMBOL:
			isEndLoop := base.GetDefaultChannelTokenType(tokens, i, -1) == parser.MySQLParserEND_SYMBOL
			if isEndLoop {
				continue
			}

			loopStack = append(loopStack, &openParenthesis{tokenType: tokens[i].GetTokenType(), pos: i})
		case parser.MySQLParserWHILE_SYMBOL:
			isEndWhile := base.GetDefaultChannelTokenType(tokens, i, -1) == parser.MySQLParserEND_SYMBOL
			if isEndWhile {
				continue
			}

			whileStack = append(whileStack, &openParenthesis{tokenType: tokens[i].GetTokenType(), pos: i})
		case parser.MySQLParserREPEAT_SYMBOL:
			isEndRepeat := base.GetDefaultChannelTokenType(tokens, i, -1) == parser.MySQLParserUNTIL_SYMBOL
			if isEndRepeat {
				continue
			}

			repeatStack = append(repeatStack, &openParenthesis{tokenType: tokens[i].GetTokenType(), pos: i})
		case parser.MySQLParserEND_SYMBOL:
			isXa := base.GetDefaultChannelTokenType(tokens, i, -1) == parser.MySQLParserXA_SYMBOL
			if isXa {
				continue
			}

			nextDefaultChannelTokenType := base.GetDefaultChannelTokenType(tokens, i, 1)
			switch nextDefaultChannelTokenType {
			case parser.MySQLParserIF_SYMBOL:
				// There are two types of IF statement:
				// 1. IF(expr1,expr2,expr3)
				// 2. IF search_condition THEN statement_list [ELSEIF search_condition THEN statement_list] ... [ELSE statement_list] END IF
				// For the first type, we will meet single IF symbol without END IF symbol.
				if len(ifStack) == 0 {
					return nil, errors.New("invalid statement: failed to split multiple statements")
				}
				semicolonStack = popSemicolonStack(semicolonStack, ifStack[0].pos)
				ifStack = ifStack[:len(ifStack)-1]
			case parser.MySQLParserLOOP_SYMBOL:
				// For the LOOP symbol, MySQL only has LOOP with END LOOP statement.
				// Other cases are invalid.
				// So we only need to do the simple parenthesis matching.
				if len(loopStack) == 0 {
					return nil, errors.New("invalid statement: failed to split multiple statements")
				}
				semicolonStack = popSemicolonStack(semicolonStack, loopStack[len(loopStack)-1].pos)
				loopStack = loopStack[:len(loopStack)-1]
			case parser.MySQLParserWHILE_SYMBOL:
				// For the WHILE symbol, MySQL only has WHILE with END WHILE statement.
				// Other cases are invalid.
				// So we only need to do the simple parenthesis matching.
				if len(whileStack) == 0 {
					return nil, errors.New("invalid statement: failed to split multiple statements")
				}
				semicolonStack = popSemicolonStack(semicolonStack, whileStack[len(whileStack)-1].pos)
				whileStack = whileStack[:len(whileStack)-1]
			case parser.MySQLParserREPEAT_SYMBOL:
				// The are two types of REPEAT statement:
				// 1. REPEAT(expr,expr)
				// 2. REPEAT statement_list UNTIL search_condition END REPEAT
				// For the first type, we will meet single REPEAT symbol without END REPEAT symbol.
				if len(repeatStack) == 0 {
					return nil, errors.New("invalid statement: failed to split multiple statements")
				}
				semicolonStack = popSemicolonStack(semicolonStack, repeatStack[0].pos)
				repeatStack = repeatStack[:len(repeatStack)-1]
			case parser.MySQLParserCASE_SYMBOL:
				if len(beginCaseStack) == 0 {
					return nil, errors.New("invalid statement: failed to split multiple statements")
				}
				semicolonStack = popSemicolonStack(semicolonStack, beginCaseStack[len(beginCaseStack)-1].pos)
				beginCaseStack = beginCaseStack[:len(beginCaseStack)-1]
			default:
				// is BEGIN ... END or CASE .. END case
				if len(beginCaseStack) == 0 {
					return nil, errors.New("invalid statement: failed to split multiple statements")
				}
				semicolonStack = popSemicolonStack(semicolonStack, beginCaseStack[len(beginCaseStack)-1].pos)
				beginCaseStack = beginCaseStack[:len(beginCaseStack)-1]
			}
		case parser.MySQLParserSEMICOLON_SYMBOL:
			semicolonStack = append(semicolonStack, i)
		default:
			// Do nothing for other token types
		}
	}

	byteOffset := 0
	start := 0
	for _, pos := range semicolonStack {
		stmtText := stream.GetTextFromTokens(tokens[start], tokens[pos])
		stmtByteLength := len(stmtText)

		// Calculate start position from byte offset (first character of Text)
		startLine, startColumn := base.CalculateLineAndColumn(statement, byteOffset)
		// From antlr4, the line is ONE based, and the column is ZERO based.
		// So we should minus 1 for the line.
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
			Empty: base.IsEmpty(tokens[start:pos+1], parser.MySQLLexerSEMICOLON_SYMBOL),
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
		// From antlr4, the line is ONE based, and the column is ZERO based.
		// So we should minus 1 for the line.
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
			Empty: base.IsEmpty(tokens[start:eofPos], parser.MySQLLexerSEMICOLON_SYMBOL),
		})
	}

	return result, nil
}

func popSemicolonStack(stack []int, openParPos int) []int {
	if len(stack) == 0 {
		return stack
	}

	for i := len(stack) - 1; i >= 0; i-- {
		if stack[i] < openParPos {
			return stack[:i+1]
		}
	}

	return []int{}
}
