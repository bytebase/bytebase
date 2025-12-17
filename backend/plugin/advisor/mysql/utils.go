// Package mysql implements the SQL advisor rules for MySQL.
package mysql

import (
	"regexp"
	"slices"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/parser/mysql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

type columnSet map[string]bool

func newColumnSet(columns []string) columnSet {
	res := make(columnSet)
	for _, col := range columns {
		res[col] = true
	}
	return res
}

type tableState map[string]columnSet

// tableList returns table list in lexicographical order.
func (t tableState) tableList() []string {
	var tableList []string
	for tableName := range t {
		tableList = append(tableList, tableName)
	}
	slices.Sort(tableList)
	return tableList
}

// getTemplateRegexp formats the template as regex.
func getTemplateRegexp(template string, templateList []string, tokens map[string]string) (*regexp.Regexp, error) {
	for _, key := range templateList {
		if token, ok := tokens[key]; ok {
			template = strings.ReplaceAll(template, key, token)
		}
	}

	return regexp.Compile(template)
}

// tableName --> columnName --> columnType.
type tableColumnTypes map[string]map[string]string

func (t tableColumnTypes) set(tableName string, columnName string, columnType string) {
	if _, ok := t[tableName]; !ok {
		t[tableName] = make(map[string]string)
	}
	t[tableName][columnName] = columnType
}

func (t tableColumnTypes) get(tableName string, columnName string) (columnType string, ok bool) {
	if _, ok := t[tableName]; !ok {
		return "", false
	}
	col, ok := t[tableName][columnName]
	return col, ok
}

func (t tableColumnTypes) delete(tableName string, columnName string) {
	if _, ok := t[tableName]; !ok {
		return
	}
	delete(t[tableName], columnName)
}

// isKeyword checks if the keyword is a MySQL keyword.
// TODO: We should check with map instead of linear search.
func isKeyword(suspect string) bool {
	for _, item := range mysql.Keywords80 {
		if strings.EqualFold(suspect, item.Keyword) {
			return true
		}
	}
	return false
}

// isCharsetDataType checks if the data type supports charset.
func isCharsetDataType(dataType mysql.IDataTypeContext) bool {
	return dataType != nil && (dataType.CHAR_SYMBOL() != nil ||
		dataType.VARCHAR_SYMBOL() != nil ||
		dataType.VARYING_SYMBOL() != nil ||
		dataType.TINYTEXT_SYMBOL() != nil ||
		dataType.TEXT_SYMBOL() != nil ||
		dataType.MEDIUMTEXT_SYMBOL() != nil ||
		dataType.LONGTEXT_SYMBOL() != nil)
}

// getANTLRTree extracts the ANTLR parse trees from the advisor context.
// The AST must be pre-parsed and passed via checkCtx.AST.
// Returns all parse results for multi-statement SQL review.
func getANTLRTree(checkCtx advisor.Context) ([]*base.ParseResult, error) {
	if checkCtx.AST == nil {
		return nil, errors.New("AST is not provided in context - must be parsed before calling advisor")
	}

	var parseResults []*base.ParseResult
	for _, unifiedAST := range checkCtx.AST {
		antlrAST, ok := base.GetANTLRAST(unifiedAST)
		if !ok {
			return nil, errors.New("AST type mismatch: expected ANTLR-based parser result")
		}

		// Reconstruct base.ParseResult from AST
		parseResults = append(parseResults, &base.ParseResult{
			Tree:     antlrAST.Tree,
			Tokens:   antlrAST.Tokens,
			BaseLine: base.GetLineOffset(antlrAST.StartPosition),
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
		// Fallback to old behavior for backward compatibility
		return getParsedStatementsFromAST(checkCtx)
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

// getParsedStatementsFromAST is the fallback when ParsedStatements is not available.
// Deprecated: Use getParsedStatements with ParsedStatements field instead.
// nolint:unused
func getParsedStatementsFromAST(checkCtx advisor.Context) ([]ParsedStatementInfo, error) {
	if checkCtx.AST == nil {
		return nil, errors.New("AST is not provided in context")
	}

	var results []ParsedStatementInfo
	for _, unifiedAST := range checkCtx.AST {
		antlrAST, ok := base.GetANTLRAST(unifiedAST)
		if !ok {
			return nil, errors.New("AST type mismatch: expected ANTLR-based parser result")
		}
		results = append(results, ParsedStatementInfo{
			Tree:     antlrAST.Tree,
			Tokens:   antlrAST.Tokens,
			BaseLine: base.GetLineOffset(antlrAST.StartPosition),
			Text:     "", // Not available in fallback mode
		})
	}
	return results, nil
}
