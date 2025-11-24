package redshift

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/redshift"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/utils"
)

func init() {
	base.RegisterParseFunc(storepb.Engine_REDSHIFT, parseRedshiftForRegistry)
}

// parseRedshiftForRegistry is the ParseFunc for Redshift.
// Returns []*base.ParseResult (list of AST nodes, one per statement) on success.
func parseRedshiftForRegistry(statement string) (any, error) {
	return ParseRedshift(statement)
}

// ParseRedshift parses the given SQL statement and returns a list of ParseResults.
// Each ParseResult represents one statement with its AST, tokens, and base line offset.
func ParseRedshift(sql string) ([]*base.ParseResult, error) {
	stmts, err := SplitSQL(sql)
	if err != nil {
		return nil, errors.Wrap(err, "failed to split SQL")
	}

	var results []*base.ParseResult
	for _, stmt := range stmts {
		if stmt.Empty {
			continue
		}

		parseResult, err := parseSingleRedshift(stmt.Text, stmt.BaseLine)
		if err != nil {
			return nil, err
		}
		results = append(results, parseResult)
	}

	return results, nil
}

// parseSingleRedshift parses a single Redshift statement by using antlr4. Returns the AST and token stream if no error.
func parseSingleRedshift(statement string, baseLine int) (*base.ParseResult, error) {
	statement = strings.TrimRightFunc(statement, utils.IsSpaceOrSemicolon) + ";"
	inputStream := antlr.NewInputStream(statement)
	lexer := parser.NewRedshiftLexer(inputStream)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := parser.NewRedshiftParser(stream)

	// Remove default error listener and add our own error listener.
	lexer.RemoveErrorListeners()
	lexerErrorListener := &base.ParseErrorListener{
		Statement: statement,
	}
	lexer.AddErrorListener(lexerErrorListener)

	parserErrorListener := &base.ParseErrorListener{
		Statement: statement,
	}
	p.RemoveErrorListeners()
	p.AddErrorListener(parserErrorListener)

	p.BuildParseTrees = true

	tree := p.Root()

	// Return early if there are any lexer errors
	if lexerErrorListener.Err != nil {
		return nil, lexerErrorListener.Err
	}

	// Return early if there are any parser errors
	if parserErrorListener.Err != nil {
		return nil, parserErrorListener.Err
	}

	result := &base.ParseResult{
		Tree:     tree,
		Tokens:   stream,
		BaseLine: baseLine,
	}

	return result, nil
}
