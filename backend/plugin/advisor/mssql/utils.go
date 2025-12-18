package mssql

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// getANTLRTree extracts TSQL ANTLR parse trees from the advisor context.
func getANTLRTree(checkCtx advisor.Context) ([]*base.ParseResult, error) {
	if checkCtx.ParsedStatements == nil {
		return nil, errors.New("ParsedStatements is not provided in context")
	}

	var parseResults []*base.ParseResult
	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		antlrAST, ok := base.GetANTLRAST(stmt.AST)
		if !ok {
			return nil, errors.New("AST type mismatch: expected ANTLR-based parser result")
		}
		parseResults = append(parseResults, &base.ParseResult{
			Tree:     antlrAST.Tree,
			Tokens:   antlrAST.Tokens,
			BaseLine: stmt.BaseLine,
		})
	}
	return parseResults, nil
}

// ParsedStatementInfo contains all info needed for checking a single statement.
type ParsedStatementInfo struct {
	Tree     antlr.Tree
	Tokens   *antlr.CommonTokenStream
	BaseLine int
	Text     string
}

// getTextFromTokens extracts the original text for a rule context from the token stream.
// Uses GetTextFromRuleContext to include hidden channel tokens (whitespace, comments).
// Returns clean text without leading/trailing whitespace.
// nolint:unused
func getTextFromTokens(tokens *antlr.CommonTokenStream, ctx antlr.ParserRuleContext) string {
	if tokens == nil || ctx == nil {
		return ""
	}
	text := tokens.GetTextFromRuleContext(ctx)
	return strings.TrimSpace(text)
}

// getParsedStatements extracts statement info from the advisor context.
// This is the preferred way to access statements - use stmtInfo.Text directly
// instead of extracting text manually.
// nolint:unused
func getParsedStatements(checkCtx advisor.Context) ([]ParsedStatementInfo, error) {
	if checkCtx.ParsedStatements == nil {
		return nil, errors.New("ParsedStatements is not provided in context")
	}

	var results []ParsedStatementInfo
	for _, stmt := range checkCtx.ParsedStatements {
		// Skip empty statements (no AST)
		if stmt.AST == nil {
			continue
		}
		antlrAST, ok := base.GetANTLRAST(stmt.AST)
		if !ok {
			return nil, errors.New("AST type mismatch: expected ANTLR-based parser result")
		}
		results = append(results, ParsedStatementInfo{
			Tree:     antlrAST.Tree,
			Tokens:   antlrAST.Tokens,
			BaseLine: stmt.BaseLine,
			Text:     stmt.Text,
		})
	}
	return results, nil
}
