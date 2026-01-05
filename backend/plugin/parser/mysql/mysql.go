package mysql

import (
	"log/slog"
	"regexp"
	"strings"

	"github.com/pkg/errors"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/mysql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/tokenizer"
)

func init() {
	base.RegisterParseFunc(storepb.Engine_MYSQL, parseMySQLForRegistry)
	base.RegisterParseFunc(storepb.Engine_MARIADB, parseMySQLForRegistry)
	base.RegisterParseFunc(storepb.Engine_OCEANBASE, parseMySQLForRegistry)
	base.RegisterParseStatementsFunc(storepb.Engine_MYSQL, parseMySQLStatements)
	base.RegisterParseStatementsFunc(storepb.Engine_MARIADB, parseMySQLStatements)
	base.RegisterParseStatementsFunc(storepb.Engine_OCEANBASE, parseMySQLStatements)
	base.RegisterGetStatementTypes(storepb.Engine_MYSQL, GetStatementTypes)
	base.RegisterGetStatementTypes(storepb.Engine_MARIADB, GetStatementTypes)
	base.RegisterGetStatementTypes(storepb.Engine_OCEANBASE, GetStatementTypes)
}

// parseMySQLForRegistry is the ParseFunc for MySQL, MariaDB, and OceanBase.
// Returns []base.AST with *ANTLRAST instances.
func parseMySQLForRegistry(statement string) ([]base.AST, error) {
	parseResults, err := ParseMySQL(statement)
	if err != nil {
		return nil, err
	}
	asts := make([]base.AST, len(parseResults))
	for i, r := range parseResults {
		asts[i] = r
	}
	return asts, nil
}

// parseMySQLStatements is the ParseStatementsFunc for MySQL, MariaDB, and OceanBase.
// Returns []ParsedStatement with both text and AST populated.
func parseMySQLStatements(statement string) ([]base.ParsedStatement, error) {
	// First split to get Statement with text and positions
	stmts, err := SplitSQL(statement)
	if err != nil {
		return nil, err
	}

	// Then parse to get ASTs
	parseResults, err := ParseMySQL(statement)
	if err != nil {
		return nil, err
	}

	// Combine: Statement provides text/positions, ANTLRAST provides AST
	var result []base.ParsedStatement
	astIndex := 0
	for _, stmt := range stmts {
		ps := base.ParsedStatement{
			Statement: stmt,
		}
		if !stmt.Empty && astIndex < len(parseResults) {
			ps.AST = parseResults[astIndex]
			astIndex++
		}
		result = append(result, ps)
	}

	return result, nil
}

// ParseMySQL parses the given SQL statement and returns the AST.
func ParseMySQL(statement string) ([]*base.ANTLRAST, error) {
	statement, err := DealWithDelimiter(statement)
	if err != nil {
		return nil, err
	}
	list, err := parseInputStream(antlr.NewInputStream(statement), statement)
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
		var result strings.Builder
		delimiter := `;`
		for _, sql := range list {
			if IsDelimiter(sql.Text) {
				delimiter, err = ExtractDelimiter(sql.Text)
				if err != nil {
					return "", err
				}
				// Comment out only the DELIMITER line, preserving all other lines for correct line numbers
				lines := strings.Split(sql.Text, "\n")
				for i, line := range lines {
					if IsDelimiter(line) {
						lines[i] = "-- " + strings.TrimLeft(line, " \t")
						break
					}
				}
				result.WriteString(strings.Join(lines, "\n"))
				continue
			}
			if delimiter != ";" && !sql.Empty {
				result.WriteString(strings.TrimSuffix(sql.Text, delimiter))
				result.WriteString(";")
			} else {
				result.WriteString(sql.Text)
			}
		}

		statement = result.String()
	}
	return statement, nil
}

func parseSingleStatement(baseLine int, statement string) (antlr.Tree, *antlr.CommonTokenStream, error) {
	input := antlr.NewInputStream(statement)
	lexer := parser.NewMySQLLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)

	p := parser.NewMySQLParser(stream)

	startPosition := &storepb.Position{Line: int32(baseLine) + 1}
	lexerErrorListener := &base.ParseErrorListener{
		StartPosition: startPosition,
	}
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(lexerErrorListener)

	parserErrorListener := &base.ParseErrorListener{
		StartPosition: startPosition,
	}
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
	lexerErrorListener := &base.ParseErrorListener{
		Statement: sql,
	}
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

func parseInputStream(input *antlr.InputStream, statement string) ([]*base.ANTLRAST, error) {
	var result []*base.ANTLRAST
	lexer := parser.NewMySQLLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)

	list, err := splitMySQLStatement(stream, statement)
	if err != nil {
		return nil, err
	}

	if len(list) > 0 {
		list[len(list)-1].Text = mysqlAddSemicolonIfNeeded(list[len(list)-1].Text)
	}

	baseLine := 0
	for _, s := range list {
		tree, tokens, err := parseSingleStatement(baseLine, s.Text)
		if err != nil {
			return nil, err
		}

		if isEmptyStatement(tokens) {
			continue
		}

		result = append(result, &base.ANTLRAST{
			StartPosition: &storepb.Position{Line: int32(s.BaseLine()) + 1},
			Tree:          tree,
			Tokens:        tokens,
		})
		// s.End.Line is 1-based, but baseLine should be 0-based
		baseLine = int(s.End.Line) - 1
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

func hasDelimiter(statement string) (bool, []base.Statement, error) {
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

// IsTopMySQLRule returns true if the given context is a top-level MySQL rule.
func IsTopMySQLRule(ctx *antlr.BaseParserRuleContext) bool {
	if ctx.GetParent() == nil {
		return false
	}
	switch ctx.GetParent().(type) {
	case *parser.SimpleStatementContext:
		if ctx.GetParent().GetParent() == nil {
			return false
		}
		if _, ok := ctx.GetParent().GetParent().(*parser.QueryContext); !ok {
			return false
		}
	case *parser.CreateStatementContext, *parser.DropStatementContext, *parser.TransactionOrLockingStatementContext, *parser.AlterStatementContext:
		if ctx.GetParent().GetParent() == nil {
			return false
		}
		if ctx.GetParent().GetParent().GetParent() == nil {
			return false
		}
		if _, ok := ctx.GetParent().GetParent().GetParent().(*parser.QueryContext); !ok {
			return false
		}
	default:
		return false
	}
	return true
}
