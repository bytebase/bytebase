package mysql

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	mysqlomniparser "github.com/bytebase/omni/mysql/parser"
	parser "github.com/bytebase/parser/mysql"
	pkgerrors "github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/mysqlutil"
)

func init() {
	base.RegisterParseStatementsFunc(storepb.Engine_MYSQL, parseMySQLStatements)
	base.RegisterParseStatementsFunc(storepb.Engine_MARIADB, parseMySQLStatements)
	base.RegisterParseStatementsFunc(storepb.Engine_OCEANBASE, parseMySQLStatements)
	base.RegisterGetStatementTypes(storepb.Engine_MYSQL, GetStatementTypes)
	base.RegisterGetStatementTypes(storepb.Engine_MARIADB, GetStatementTypes)
	base.RegisterGetStatementTypes(storepb.Engine_OCEANBASE, GetStatementTypes)
}

// parseMySQLStatements is the ParseStatementsFunc for MySQL, MariaDB, and OceanBase.
// Returns []ParsedStatement with both text and AST populated.
func parseMySQLStatements(statement string) ([]base.ParsedStatement, error) {
	// Split once to get Statement with text and positions
	stmts, err := SplitSQL(statement)
	if err != nil {
		return nil, err
	}

	var result []base.ParsedStatement
	for _, stmt := range stmts {
		if stmt.Empty {
			continue
		}

		list, omniErr := ParseMySQLOmni(stmt.Text)
		if omniErr != nil {
			return nil, convertOmniError(omniErr, stmt)
		}

		if list == nil || len(list.Items) == 0 {
			continue
		}

		for _, node := range list.Items {
			result = append(result, base.ParsedStatement{
				Statement: stmt,
				AST: &OmniAST{
					Node:          node,
					Text:          stmt.Text,
					StartPosition: stmt.Start,
				},
			})
		}
	}

	return result, nil
}

// convertOmniError converts an omni parser error to a base.SyntaxError with proper line:column position.
func convertOmniError(err error, stmt base.Statement) error {
	var parseErr *mysqlomniparser.ParseError
	if !errors.As(err, &parseErr) {
		return err
	}

	pos := ByteOffsetToRunePosition(stmt.Text, parseErr.Position)

	// Adjust line by the statement's base line (stmt.Start.Line is 1-based).
	if stmt.Start != nil {
		pos.Line += stmt.Start.Line - 1
	}

	msg := fmt.Sprintf("Syntax error at line %d:%d: %s", pos.Line, pos.Column, parseErr.Message)
	if parseErr.RelatedText != "" {
		msg += "\nrelated text: " + parseErr.RelatedText
	}

	return &base.SyntaxError{
		Position:   pos,
		Message:    msg,
		RawMessage: parseErr.Message,
	}
}

// ParseMySQL parses the given SQL statement and returns the AST.
func ParseMySQL(statement string) ([]*base.ANTLRAST, error) {
	stmts, err := SplitSQL(statement)
	if err != nil {
		return nil, err
	}
	return parseMySQLStatementsInternal(stmts)
}

// parseMySQLStatementsInternal parses pre-split statements without re-splitting.
// This is the internal implementation used by both ParseMySQL and parseMySQLStatements.
func parseMySQLStatementsInternal(stmts []base.Statement) ([]*base.ANTLRAST, error) {
	var result []*base.ANTLRAST

	if len(stmts) > 0 {
		// Add semicolon to the last statement if needed
		stmts[len(stmts)-1].Text = mysqlAddSemicolonIfNeeded(stmts[len(stmts)-1].Text)
	}

	for _, s := range stmts {
		if s.Empty {
			continue
		}

		tree, tokens, err := parseSingleStatement(s.BaseLine(), s.Text)
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
	}

	return result, nil
}

// DealWithDelimiter removes client-side DELIMITER directives and converts
// statements terminated by a custom delimiter back to semicolon-terminated SQL.
func DealWithDelimiter(statement string) (string, error) {
	return mysqlutil.DealWithDelimiter(statement)
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
	return "", pkgerrors.Errorf("cannot extract delimiter from %q", stmt)
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
