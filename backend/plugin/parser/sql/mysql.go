// Package parser is the parser for SQL statement.
package parser

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/mysql-parser"
)

// MySQLParseResult is the result of parsing a MySQL statement.
type MySQLParseResult struct {
	Tree     antlr.Tree
	Tokens   *antlr.CommonTokenStream
	BaseLine int
}

// ParseMySQL parses the given SQL statement and returns the AST.
func ParseMySQL(statement string) ([]*MySQLParseResult, error) {
	statement = strings.TrimRight(statement, " \r\n\t\f;") + "\n;"
	var err error
	statement, err = DealWithDelimiter(statement)
	if err != nil {
		return nil, err
	}

	return parseInputStream(antlr.NewInputStream(statement))
}

// DealWithDelimiter deals with delimiter in the given SQL statement.
func DealWithDelimiter(statement string) (string, error) {
	has, list, err := hasDelimiter(statement)
	if err != nil {
		return "", err
	}
	if has {
		var result []string
		delimiter := `;`
		for _, sql := range list {
			if IsDelimiter(sql.Text) {
				delimiter, err = ExtractDelimiter(sql.Text)
				if err != nil {
					return "", err
				}
				result = append(result, "-- "+sql.Text)
				continue
			}
			// TODO(rebelice): after deal with delimiter, we may cannot get the right line number, fix it.
			if delimiter != ";" {
				result = append(result, fmt.Sprintf("%s;", strings.TrimSuffix(sql.Text, delimiter)))
			} else {
				result = append(result, sql.Text)
			}
		}

		statement = strings.Join(result, "\n")
	}
	return statement, nil
}

// SplitMySQL splits the given SQL statement into multiple SQL statements.
func SplitMySQL(statement string) ([]SingleSQL, error) {
	statement = strings.TrimRight(statement, " \r\n\t\f;") + "\n;"
	var err error
	statement, err = DealWithDelimiter(statement)
	if err != nil {
		return nil, err
	}

	lexer := parser.NewMySQLLexer(antlr.NewInputStream(statement))
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	return splitMySQLStatement(stream)
}

// SplitMySQLStream splits the given SQL stream into multiple SQL statements.
// Note that the reader is read completely into memory and so it must actually
// have a stopping point - you cannot pass in a reader on an open-ended source such
// as a socket for instance.
func SplitMySQLStream(src io.Reader) ([]SingleSQL, error) {
	text := antlr.NewIoStream(src).String()
	return SplitMySQL(text)
}

// ParseMySQLStream parses the given SQL stream and returns the AST.
// Note that the reader is read completely into memory and so it must actually
// have a stopping point - you cannot pass in a reader on an open-ended source such
// as a socket for instance.
func ParseMySQLStream(src io.Reader) ([]*MySQLParseResult, error) {
	text := antlr.NewIoStream(src).String()
	return ParseMySQL(text)
}

func getDefaultChannelTokenType(tokens []antlr.Token, base int, offset int) int {
	current := base
	step := 1
	remaining := offset
	if offset < 0 {
		step = -1
		remaining = -offset
	}
	for remaining != 0 {
		current += step
		if current < 0 || current >= len(tokens) {
			return parser.MySQLParserEOF
		}

		if tokens[current].GetChannel() == antlr.TokenDefaultChannel {
			remaining--
		}
	}

	return tokens[current].GetTokenType()
}

func splitMySQLStatement(stream *antlr.CommonTokenStream) ([]SingleSQL, error) {
	var result []SingleSQL
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

			result = append(result, SingleSQL{
				Text:     stream.GetTextFromTokens(tokens[start], tokens[i]),
				BaseLine: tokens[start].GetLine() - 1,
				LastLine: tokens[i].GetLine()},
			)
			start = i + 1
		case parser.MySQLParserEOF:
			if len(stack) != 0 {
				// Backtracking the process.
				i = stack[len(stack)-1].pos
				stack = stack[:len(stack)-1]
				continue
			}

			if start <= i-1 {
				result = append(result, SingleSQL{
					Text:     stream.GetTextFromTokens(tokens[start], tokens[i-1]),
					BaseLine: tokens[start].GetLine() - 1,
					LastLine: tokens[i-1].GetLine()},
				)
			}
		}
	}
	return result, nil
}

func parseSingleStatement(statement string) (antlr.Tree, *antlr.CommonTokenStream, error) {
	input := antlr.NewInputStream(statement)
	lexer := parser.NewMySQLLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)

	p := parser.NewMySQLParser(stream)

	lexerErrorListener := &ParseErrorListener{}
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(lexerErrorListener)

	parserErrorListener := &ParseErrorListener{}
	p.RemoveErrorListeners()
	p.AddErrorListener(parserErrorListener)

	p.BuildParseTrees = true

	tree := p.Script()

	if lexerErrorListener.err != nil {
		return nil, nil, lexerErrorListener.err
	}

	if parserErrorListener.err != nil {
		return nil, nil, parserErrorListener.err
	}

	return tree, stream, nil
}

func parseInputStream(input *antlr.InputStream) ([]*MySQLParseResult, error) {
	var result []*MySQLParseResult
	lexer := parser.NewMySQLLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)

	list, err := splitMySQLStatement(stream)
	if err != nil {
		return nil, err
	}

	for _, s := range list {
		tree, tokens, err := parseSingleStatement(s.Text)
		if err != nil {
			return nil, err
		}

		result = append(result, &MySQLParseResult{
			Tree:     tree,
			Tokens:   tokens,
			BaseLine: s.BaseLine,
		})
	}

	return result, nil
}

// MySQLValidateForEditor validates the given SQL statement for editor.
func MySQLValidateForEditor(tree antlr.Tree) error {
	l := &mysqlValidateForEditorListener{
		validate: true,
	}

	antlr.ParseTreeWalkerDefault.Walk(l, tree)
	if !l.validate {
		return errors.New("Malformed sql execute request, only support SELECT sql statement")
	}
	return nil
}

type mysqlValidateForEditorListener struct {
	*parser.BaseMySQLParserListener

	validate bool
}

// EnterQuery is called when production query is entered.
func (l *mysqlValidateForEditorListener) EnterQuery(ctx *parser.QueryContext) {
	if ctx.BeginWork() != nil {
		l.validate = false
	}
}

// EnterSimpleStatement is called when production simpleStatement is entered.
func (l *mysqlValidateForEditorListener) EnterSimpleStatement(ctx *parser.SimpleStatementContext) {
	if ctx.SelectStatement() == nil && ctx.UtilityStatement() == nil {
		l.validate = false
	}
}

// EnterUtilityStatement is called when production utilityStatement is entered.
func (l *mysqlValidateForEditorListener) EnterUtilityStatement(ctx *parser.UtilityStatementContext) {
	if ctx.ExplainStatement() == nil {
		l.validate = false
	}
}

// EnterExplainableStatement is called when production explainableStatement is entered.
func (l *mysqlValidateForEditorListener) EnterExplainableStatement(ctx *parser.ExplainableStatementContext) {
	if ctx.DeleteStatement() != nil || ctx.UpdateStatement() != nil || ctx.InsertStatement() != nil || ctx.ReplaceStatement() != nil {
		l.validate = false
	}
}

func extractMySQLResourceList(currentDatabase string, statement string) ([]SchemaResource, error) {
	treeList, err := ParseMySQL(statement)
	if err != nil {
		return nil, err
	}

	l := &mysqlResourceExtractListener{
		currentDatabase: currentDatabase,
		resourceMap:     make(map[string]SchemaResource),
	}

	var result []SchemaResource
	for _, tree := range treeList {
		if tree == nil {
			continue
		}
		antlr.ParseTreeWalkerDefault.Walk(l, tree.Tree)
	}
	for _, resource := range l.resourceMap {
		result = append(result, resource)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].String() < result[j].String()
	})

	return result, nil
}

type mysqlResourceExtractListener struct {
	*parser.BaseMySQLParserListener

	currentDatabase string
	resourceMap     map[string]SchemaResource
}

// EnterTableRef is called when production tableRef is entered.
func (l *mysqlResourceExtractListener) EnterTableRef(ctx *parser.TableRefContext) {
	resource := SchemaResource{Database: l.currentDatabase}
	if ctx.DotIdentifier() != nil {
		resource.Table = NormalizeMySQLIdentifier(ctx.DotIdentifier().Identifier())
	}
	db, table := normalizeMySQLQualifiedIdentifier(ctx.QualifiedIdentifier())
	if db != "" {
		resource.Database = db
	}
	resource.Table = table
	l.resourceMap[resource.String()] = resource
}

// NormalizeMySQLTableName normalizes the given table name.
func NormalizeMySQLTableName(ctx parser.ITableNameContext) (string, string) {
	if ctx.QualifiedIdentifier() != nil {
		return normalizeMySQLQualifiedIdentifier(ctx.QualifiedIdentifier())
	}
	if ctx.DotIdentifier() != nil {
		return "", NormalizeMySQLIdentifier(ctx.DotIdentifier().Identifier())
	}
	return "", ""
}

// NormalizeMySQLColumnName normalizes the given column name.
func NormalizeMySQLColumnName(ctx parser.IColumnNameContext) (string, string, string) {
	if ctx.Identifier() != nil {
		return "", "", NormalizeMySQLIdentifier(ctx.Identifier())
	}
	return normalizeMySQLFieldIdentifier(ctx.FieldIdentifier())
}

func normalizeMySQLFieldIdentifier(ctx parser.IFieldIdentifierContext) (string, string, string) {
	list := []string{}
	if ctx.QualifiedIdentifier() != nil {
		id1, id2 := normalizeMySQLQualifiedIdentifier(ctx.QualifiedIdentifier())
		list = append(list, id1, id2)
	}

	if ctx.DotIdentifier() != nil {
		list = append(list, NormalizeMySQLIdentifier(ctx.DotIdentifier().Identifier()))
	}

	for len(list) < 3 {
		list = append([]string{""}, list...)
	}

	return list[0], list[1], list[2]
}

func normalizeMySQLQualifiedIdentifier(qualifiedIdentifier parser.IQualifiedIdentifierContext) (string, string) {
	list := []string{NormalizeMySQLIdentifier(qualifiedIdentifier.Identifier())}
	if qualifiedIdentifier.DotIdentifier() != nil {
		list = append(list, NormalizeMySQLIdentifier(qualifiedIdentifier.DotIdentifier().Identifier()))
	}

	if len(list) == 1 {
		list = append([]string{""}, list...)
	}

	return list[0], list[1]
}

func NormalizeMySQLIdentifier(identifier parser.IIdentifierContext) string {
	if identifier.PureIdentifier() != nil {
		if identifier.PureIdentifier().IDENTIFIER() != nil {
			return identifier.PureIdentifier().IDENTIFIER().GetText()
		}
		// For back tick quoted identifier, we need to remove the back tick.
		text := identifier.PureIdentifier().BACK_TICK_QUOTED_ID().GetText()
		return text[1 : len(text)-1]
	}
	return identifier.GetText()
}

// IsMySQLAffectedRowsStatement returns true if the given statement is an affected rows statement.
func IsMySQLAffectedRowsStatement(statement string) bool {
	lexer := parser.NewMySQLLexer(antlr.NewInputStream(statement))
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	stream.Fill()
	tokens := stream.GetAllTokens()

	for _, token := range tokens {
		if token.GetChannel() == antlr.TokenDefaultChannel {
			switch token.GetTokenType() {
			case parser.MySQLParserDELETE_SYMBOL, parser.MySQLParserINSERT_SYMBOL, parser.MySQLParserREPLACE_SYMBOL, parser.MySQLParserUPDATE_SYMBOL:
				return true
			default:
				return false
			}
		}
	}

	return false
}
