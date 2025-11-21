package spanner

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/googlesql"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/utils"
)

type ParseResult struct {
	Tree     antlr.Tree
	Tokens   *antlr.CommonTokenStream
	BaseLine int
}

// ParseSpannerGoogleSQL parses the given SQL and returns a list of ParseResult (one per statement).
// Use the GoogleSQL parser based on antlr4.
func ParseSpannerGoogleSQL(sql string) ([]*ParseResult, error) {
	stmts, err := SplitSQL(sql)
	if err != nil {
		return nil, err
	}

	var results []*ParseResult
	for _, stmt := range stmts {
		if stmt.Empty {
			continue
		}

		parseResult, err := parseSingleSpannerGoogleSQL(stmt.Text, stmt.BaseLine)
		if err != nil {
			return nil, err
		}
		results = append(results, parseResult)
	}

	return results, nil
}

// parseSingleSpannerGoogleSQL parses a single Spanner statement and returns the ParseResult.
func parseSingleSpannerGoogleSQL(statement string, baseLine int) (*ParseResult, error) {
	statement = strings.TrimRightFunc(statement, utils.IsSpaceOrSemicolon) + "\n;"
	inputStream := antlr.NewInputStream(statement)
	lexer := parser.NewGoogleSQLLexer(inputStream)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := parser.NewGoogleSQLParser(stream)

	// Remove default error listener and add our own error listener.
	lexer.RemoveErrorListeners()
	lexerErrorListener := &base.ParseErrorListener{
		Statement: statement,
		BaseLine:  baseLine,
	}
	lexer.AddErrorListener(lexerErrorListener)

	p.RemoveErrorListeners()
	parserErrorListener := &base.ParseErrorListener{
		Statement: statement,
		BaseLine:  baseLine,
	}
	p.AddErrorListener(parserErrorListener)

	p.BuildParseTrees = true

	tree := p.Root()

	if lexerErrorListener.Err != nil {
		return nil, lexerErrorListener.Err
	}

	if parserErrorListener.Err != nil {
		return nil, parserErrorListener.Err
	}

	result := &ParseResult{
		Tree:     tree,
		Tokens:   stream,
		BaseLine: baseLine,
	}

	return result, nil
}
