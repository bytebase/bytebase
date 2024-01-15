package mysql

import (
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/pkg/errors"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/mysql-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/tokenizer"
)

// ParseResult is the result of parsing a MySQL statement.
type ParseResult struct {
	Tree     antlr.Tree
	Tokens   *antlr.CommonTokenStream
	BaseLine int
}

// ParseMySQL parses the given SQL statement and returns the AST.
func ParseMySQL(statement string) ([]*ParseResult, error) {
	statement, err := DealWithDelimiter(statement)
	if err != nil {
		return nil, err
	}
	list, err := parseInputStream(antlr.NewInputStream(statement))
	// HACK(p0ny): the callee may end up in an infinite loop, we print the statement here to help debug.
	if err != nil && strings.Contains(err.Error(), "split SQL statement timed out") {
		slog.Info("split SQL statement timed out", "statement", statement)
	}
	return list, err
}

// DealWithDelimiter converts the delimiter statement to comment, also converts the following statement's delimiter to semicolon(`;`).
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

// RestoreDelimiter trys to restore the delimiter statement which commentted by DealWithDelimiter, assumes the delimiter comment takes up a single line
// and follows the syntax `-- DELIMITER ?`, otherwise returns an error.
func RestoreDelimiter(statement string) (string, error) {
	delimiterRegexp := regexp.MustCompile(`(?i)^-- DELIMITER\s+(?P<DELIMITER>[^\s\\]+)\s*`)
	lines := strings.Split(statement, "\n")
	var result strings.Builder
	var buf strings.Builder
	previousDelimiter := ";"

	flushBuf := func(delimiter string) error {
		bufStmt := buf.String()
		t := tokenizer.NewTokenizer(bufStmt)
		list, err := t.SplitTiDBMultiSQL()
		if err != nil {
			return errors.Errorf("failed to split multi sql %q: %v", bufStmt, err)
		}
		for _, sql := range list {
			if IsDelimiter(sql.Text) {
				return errors.Errorf("delimiter statement %q miexed with delimiter comments lap, previous delimiter %q", sql.Text, delimiter)
			}
			if _, err := result.WriteString(fmt.Sprintf("%s%s\n", strings.TrimSuffix(sql.Text, ";"), previousDelimiter)); err != nil {
				return errors.Wrapf(err, "failed to write string %q into builder", sql.Text)
			}
		}
		buf.Reset()
		return nil
	}

	for i, line := range lines {
		matchList := delimiterRegexp.FindStringSubmatch(line)
		if matchList == nil {
			target := &result
			if previousDelimiter != ";" {
				target = &buf
			}
			// Like strings.Join.
			s := line
			if i != len(lines)-1 {
				s = fmt.Sprintf("%s\n", line)
			}
			if _, err := target.WriteString(s); err != nil {
				return "", errors.Wrapf(err, "failed to write string %q into builder", line)
			}
			continue
		}
		if len(matchList) != 2 {
			return "", errors.Errorf("invalid delimiter comment %q", line)
		}

		// If meet delimiter again, we should convert the delimiter in buf to previous delimiter.
		if previousDelimiter != ";" {
			if err := flushBuf(previousDelimiter); err != nil {
				return "", err
			}
		}

		if _, err := result.WriteString(fmt.Sprintf("DELIMITER %s\n", matchList[1])); err != nil {
			return "", errors.Wrapf(err, "failed to write string %q into builder", line)
		}
		previousDelimiter = matchList[1]
	}

	if previousDelimiter != ";" {
		if err := flushBuf(previousDelimiter); err != nil {
			return "", err
		}
	}

	return result.String(), nil
}

func parseSingleStatement(statement string) (antlr.Tree, *antlr.CommonTokenStream, error) {
	input := antlr.NewInputStream(statement)
	lexer := parser.NewMySQLLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)

	p := parser.NewMySQLParser(stream)

	lexerErrorListener := &base.ParseErrorListener{}
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(lexerErrorListener)

	parserErrorListener := &base.ParseErrorListener{}
	p.RemoveErrorListeners()
	p.AddErrorListener(parserErrorListener)

	p.BuildParseTrees = true

	tree := p.Script()

	if lexerErrorListener.Err != nil {
		return nil, nil, lexerErrorListener.Err
	}

	if parserErrorListener.Err != nil {
		return nil, nil, parserErrorListener.Err
	}

	return tree, stream, nil
}

func mysqlAddSemicolonIfNeeded(sql string) string {
	lexer := parser.NewMySQLLexer(antlr.NewInputStream(sql))
	lexerErrorListener := &base.ParseErrorListener{}
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(lexerErrorListener)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	stream.Fill()
	if lexerErrorListener.Err != nil {
		// If the lexer fails, we cannot add semicolon.
		return sql
	}
	tokens := stream.GetAllTokens()
	for i := len(tokens) - 1; i >= 0; i-- {
		if tokens[i].GetChannel() != antlr.TokenDefaultChannel || tokens[i].GetTokenType() == parser.MySQLParserEOF {
			continue
		}

		// The last default channel token is a semicolon.
		if tokens[i].GetTokenType() == parser.MySQLParserSEMICOLON_SYMBOL {
			return sql
		}

		var result []string
		result = append(result, stream.GetTextFromInterval(antlr.NewInterval(0, tokens[i].GetTokenIndex())))
		result = append(result, ";")
		result = append(result, stream.GetTextFromInterval(antlr.NewInterval(tokens[i].GetTokenIndex()+1, tokens[len(tokens)-1].GetTokenIndex())))
		return strings.Join(result, "")
	}
	return sql
}

func parseInputStream(input *antlr.InputStream) ([]*ParseResult, error) {
	var result []*ParseResult
	lexer := parser.NewMySQLLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)

	list, err := splitMySQLStatement(stream)
	if err != nil {
		return nil, err
	}

	if len(list) > 0 {
		list[len(list)-1].Text = mysqlAddSemicolonIfNeeded(list[len(list)-1].Text)
	}

	for _, s := range list {
		tree, tokens, err := parseSingleStatement(s.Text)
		if err != nil {
			return nil, err
		}

		if isEmptyStatement(tokens) {
			continue
		}

		result = append(result, &ParseResult{
			Tree:     tree,
			Tokens:   tokens,
			BaseLine: s.BaseLine,
		})
	}

	return result, nil
}

func isEmptyStatement(tokens *antlr.CommonTokenStream) bool {
	for _, token := range tokens.GetAllTokens() {
		if token.GetChannel() == antlr.TokenDefaultChannel && token.GetTokenType() != parser.MySQLParserSEMICOLON_SYMBOL && token.GetTokenType() != parser.MySQLParserEOF {
			return false
		}
	}
	return true
}

// IsDelimiter returns true if the statement is a delimiter statement.
func IsDelimiter(stmt string) bool {
	delimiterRegex := `(?i)^\s*DELIMITER\s+`
	re := regexp.MustCompile(delimiterRegex)
	return re.MatchString(stmt)
}

// ExtractDelimiter extracts the delimiter from the delimiter statement.
func ExtractDelimiter(stmt string) (string, error) {
	delimiterRegex := `(?i)^\s*DELIMITER\s+(?P<DELIMITER>[^\s\\]+)\s*`
	re := regexp.MustCompile(delimiterRegex)
	matchList := re.FindStringSubmatch(stmt)
	index := re.SubexpIndex("DELIMITER")
	if index >= 0 && index < len(matchList) {
		return matchList[index], nil
	}
	return "", errors.Errorf("cannot extract delimiter from %q", stmt)
}

func hasDelimiter(statement string) (bool, []base.SingleSQL, error) {
	// use splitTiDBMultiSQL to check if the statement has delimiter
	t := tokenizer.NewTokenizer(statement)
	list, err := t.SplitTiDBMultiSQL()
	if err != nil {
		return false, nil, errors.Errorf("failed to split multi sql: %v", err)
	}

	for _, sql := range list {
		if IsDelimiter(sql.Text) {
			return true, list, nil
		}
	}

	return false, list, nil
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
