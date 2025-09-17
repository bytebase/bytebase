package pg

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/postgresql"

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
	lexerErrorListener := &base.ParseErrorListener{
		Statement: sql,
	}
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(lexerErrorListener)

	parserErrorListener := &base.ParseErrorListener{
		Statement: sql,
	}
	p.RemoveErrorListeners()
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
		Tree:   tree,
		Tokens: stream,
	}

	return result, nil
}

// nolint:unused
func normalizePostgreSQLTableAlias(ctx parser.ITable_aliasContext) string {
	if ctx == nil {
		return ""
	}

	switch {
	case ctx.Identifier() != nil:
		return normalizePostgreSQLIdentifier(ctx.Identifier())
	default:
		// For non-quote identifier, we just return the lower string for PostgreSQL.
		return strings.ToLower(ctx.GetText())
	}
}

// nolint:unused
func normalizePostgreSQLNameList(ctx parser.IName_listContext) []string {
	if ctx == nil {
		return nil
	}

	var result []string
	for _, item := range ctx.AllName() {
		result = append(result, normalizePostgreSQLName(item))
	}

	return result
}

func normalizePostgreSQLName(ctx parser.INameContext) string {
	if ctx == nil {
		return ""
	}

	if ctx.Colid() != nil {
		return NormalizePostgreSQLColid(ctx.Colid())
	}

	return ""
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

// nolint:unused
func normalizePostgreSQLSetTarget(ctx parser.ISet_targetContext) []string {
	if ctx == nil {
		return []string{}
	}

	var res []string
	res = append(res, NormalizePostgreSQLColid(ctx.Colid()))
	res = append(res, normalizePostgreSQLOptIndirection(ctx.Opt_indirection())...)
	return res
}

// nolint:unused
func normalizePostgreSQLOptIndirection(ctx parser.IOpt_indirectionContext) []string {
	var res []string
	for _, child := range ctx.AllIndirection_el() {
		res = append(res, normalizePostgreSQLIndirectionEl(child))
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

// nolint:unused
func normalizePostgreSQLAnyIdentifier(ctx parser.IAny_identifierContext) string {
	if ctx == nil {
		return ""
	}

	switch {
	case ctx.Colid() != nil:
		return NormalizePostgreSQLColid(ctx.Colid())
	case ctx.Plsql_unreserved_keyword() != nil:
		return strings.ToLower(ctx.Plsql_unreserved_keyword().GetText())
	default:
		return ""
	}
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

// NormalizePostgreSQLFuncName normalizes the given function name.
func NormalizePostgreSQLFuncName(ctx parser.IFunc_nameContext) []string {
	if ctx == nil {
		return []string{}
	}

	var result []string

	// Handle type_function_name (simple identifiers)
	if ctx.Type_function_name() != nil {
		result = append(result, normalizePostgreSQLTypeFunctionName(ctx.Type_function_name()))
	}

	// Handle qualified function name (colid + indirection)
	if ctx.Colid() != nil {
		result = append(result, NormalizePostgreSQLColid(ctx.Colid()))

		// Handle indirection for qualified names
		if ctx.Indirection() != nil {
			parts := normalizePostgreSQLIndirection(ctx.Indirection())
			result = append(result, parts...)
		}
	}

	// Handle builtin function names
	if ctx.Builtin_function_name() != nil {
		result = append(result, ctx.Builtin_function_name().GetText())
	}

	// Handle special keywords LEFT/RIGHT
	if len(result) == 0 && ctx.GetText() != "" {
		// Fallback for special cases like LEFT, RIGHT keywords
		result = append(result, strings.ToLower(ctx.GetText()))
	}

	return result
}

// normalizePostgreSQLTypeFunctionName normalizes a type_function_name context.
func normalizePostgreSQLTypeFunctionName(ctx parser.IType_function_nameContext) string {
	if ctx == nil {
		return ""
	}

	// type_function_name can be identifier, unreserved_keyword, plsql_unreserved_keyword, or type_func_name_keyword
	text := ctx.GetText()

	// Remove quotes if present and convert to lowercase for normalization
	if len(text) >= 2 && text[0] == '"' && text[len(text)-1] == '"' {
		// Quoted identifier - preserve case but remove quotes
		return text[1 : len(text)-1]
	}

	// Unquoted identifier - convert to lowercase
	return strings.ToLower(text)
}

// NormalizePostgreSQLName normalizes the given name.
// nolint:revive
func NormalizePostgreSQLName(ctx parser.INameContext) string {
	return normalizePostgreSQLName(ctx)
}
