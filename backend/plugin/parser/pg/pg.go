package pg

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/postgresql-parser"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// ParsePostgreSQL parses the given SQL and returns the AST tree.
// Use the PostgreSQL parser based on antlr4.
func ParsePostgreSQL(sql string) (antlr.Tree, error) {
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

	return tree, nil
}

// NormalizePostgreSQLQualifiedNameAsTableName normalizes the given qualified name as table name.
func NormalizePostgreSQLQualifiedNameAsTableName(ctx parser.IQualified_nameContext) (string, string, error) {
	if ctx == nil {
		return "", "", nil
	}

	list := NormalizePostgreSQLQualifiedName(ctx)
	switch len(list) {
	case 2:
		return list[0], list[1], nil
	case 1:
		return "", list[0], nil
	case 0:
		return "", "", nil
	default:
		return "", "", errors.Errorf("Invalid table name: %s", ctx.GetText())
	}
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
