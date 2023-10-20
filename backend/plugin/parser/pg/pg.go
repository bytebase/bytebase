package pg

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/postgresql-parser"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

type ParseResult struct {
	Tree   antlr.Tree
	Tokens *antlr.CommonTokenStream
}

// ParsePostgreSQL parses the given SQL and returns the ParseResult.
// Use the PostgreSQL parser based on antlr4.
func ParsePostgreSQL(sql string) (*ParseResult, error) {
	lexer := parser.NewPostgreSQLLexer(antlr.NewInputStream(sql))
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := parser.NewPostgreSQLParser(stream)
	lexerErrorListener := &base.ParseErrorListener{}
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(lexerErrorListener)

	parserErrorListener := &base.ParseErrorListener{}
	p.RemoveErrorListeners()
	p.AddErrorListener(parserErrorListener)

	p.BuildParseTrees = true
	p.SetErrorHandler(antlr.NewBailErrorStrategy())

	tree := p.Root()
	if lexerErrorListener.Err != nil {
		return nil, lexerErrorListener.Err
	}

	if parserErrorListener.Err != nil {
		return nil, parserErrorListener.Err
	}

	result := &ParseResult{
		Tree:   tree,
		Tokens: stream,
	}

	return result, nil
}

// NormalizePostgreSQLAnyName normalizes the given any name.
func NormalizePostgreSQLAnyName(ctx parser.IAny_nameContext) []string {
	if ctx == nil {
		return nil
	}

	var result []string
	result = append(result, NormalizePostgreSQLColid(ctx.Colid()))
	if ctx.Attrs() != nil {
		for _, item := range ctx.Attrs().AllAttr_name() {
			result = append(result, normalizePostgreSQLAttrName(item))
		}
	}

	return result
}

func normalizePostgreSQLStringListAsTableName(list []string) (string, string, error) {
	switch len(list) {
	case 2:
		return list[0], list[1], nil
	case 1:
		return "", list[0], nil
	case 0:
		return "", "", nil
	default:
		return "", "", errors.Errorf("Invalid table name: %v", list)
	}
}

// NormalizePostgreSQLAnyNameAsTableName normalizes the given any name as table name.
func NormalizePostgreSQLAnyNameAsTableName(ctx parser.IAny_nameContext) (string, string, error) {
	if ctx == nil {
		return "", "", nil
	}

	list := NormalizePostgreSQLAnyName(ctx)
	return normalizePostgreSQLStringListAsTableName(list)
}

// NormalizePostgreSQLAnyNameAsColumnName normalizes the given any name as column name.
func NormalizePostgreSQLAnyNameAsColumnName(ctx parser.IAny_nameContext) (string, string, string, error) {
	if ctx == nil {
		return "", "", "", nil
	}

	list := NormalizePostgreSQLAnyName(ctx)

	switch len(list) {
	case 3:
		return list[0], list[1], list[2], nil
	case 2:
		return "", list[0], list[1], nil
	case 1:
		return "", "", list[0], nil
	default:
		return "", "", "", errors.Errorf("Invalid column name: %v", list)
	}
}

// NormalizePostgreSQLQualifiedNameAsTableName normalizes the given qualified name as table name.
func NormalizePostgreSQLQualifiedNameAsTableName(ctx parser.IQualified_nameContext) (string, string, error) {
	if ctx == nil {
		return "", "", nil
	}

	list := NormalizePostgreSQLQualifiedName(ctx)
	return normalizePostgreSQLStringListAsTableName(list)
}

// ParsePostgreSQLStatement parses the given SQL and returns the AST tree.
func NormalizePostgreSQLQualifiedName(ctx parser.IQualified_nameContext) []string {
	if ctx == nil {
		return []string{}
	}

	res := []string{NormalizePostgreSQLColid(ctx.Colid())}

	if ctx.Indirection() != nil {
		res = append(res, normalizePostgreSQLIndirection(ctx.Indirection())...)
	}
	return res
}

func normalizePostgreSQLIndirection(ctx parser.IIndirectionContext) []string {
	if ctx == nil {
		return []string{}
	}

	var res []string
	for _, child := range ctx.AllIndirection_el() {
		res = append(res, normalizePostgreSQLIndirectionEl(child))
	}
	return res
}

func normalizePostgreSQLIndirectionEl(ctx parser.IIndirection_elContext) string {
	if ctx == nil {
		return ""
	}

	if ctx.DOT() != nil {
		if ctx.STAR() != nil {
			return "*"
		}
		return normalizePostgreSQLAttrName(ctx.Attr_name())
	}
	return ctx.GetText()
}

func normalizePostgreSQLAttrName(ctx parser.IAttr_nameContext) string {
	return normalizePostgreSQLCollabel(ctx.Collabel())
}

func normalizePostgreSQLCollabel(ctx parser.ICollabelContext) string {
	if ctx == nil {
		return ""
	}
	if ctx.Identifier() != nil {
		return normalizePostgreSQLIdentifier(ctx.Identifier())
	}
	return strings.ToLower(ctx.GetText())
}

// NormalizePostgreSQLColid normalizes the given colid.
func NormalizePostgreSQLColid(ctx parser.IColidContext) string {
	if ctx == nil {
		return ""
	}

	if ctx.Identifier() != nil {
		return normalizePostgreSQLIdentifier(ctx.Identifier())
	}

	// For non-quote identifier, we just return the lower string for PostgreSQL.
	return strings.ToLower(ctx.GetText())
}

func normalizePostgreSQLIdentifier(ctx parser.IIdentifierContext) string {
	if ctx == nil {
		return ""
	}

	// TODO: handle USECAPE
	switch {
	case ctx.QuotedIdentifier() != nil:
		return normalizePostgreSQLQuotedIdentifier(ctx.QuotedIdentifier().GetText())
	case ctx.UnicodeQuotedIdentifier() != nil:
		return normalizePostgreSQLUnicodeQuotedIdentifier(ctx.UnicodeQuotedIdentifier().GetText())
	default:
		return strings.ToLower(ctx.GetText())
	}
}

func normalizePostgreSQLQuotedIdentifier(s string) string {
	if len(s) < 2 {
		return s
	}

	// Remove the quote and unescape the quote.
	return strings.ReplaceAll(s[1:len(s)-1], `""`, `"`)
}

func normalizePostgreSQLUnicodeQuotedIdentifier(s string) string {
	// Do nothing for now.
	return s
}
