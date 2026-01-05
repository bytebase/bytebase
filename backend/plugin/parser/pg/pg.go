package pg

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterParseFunc(storepb.Engine_POSTGRES, parsePostgreSQLForRegistry)
	base.RegisterParseStatementsFunc(storepb.Engine_POSTGRES, parsePgStatements)
	base.RegisterGetStatementTypes(storepb.Engine_POSTGRES, GetStatementTypesForRegistry)
}

// parsePostgreSQLForRegistry is the ParseFunc for PostgreSQL.
// Returns []base.AST with *ANTLRAST instances.
func parsePostgreSQLForRegistry(statement string) ([]base.AST, error) {
	parseResults, err := ParsePostgreSQL(statement)
	if err != nil {
		return nil, err
	}
	asts := make([]base.AST, len(parseResults))
	for i, r := range parseResults {
		asts[i] = r
	}
	return asts, nil
}

// parsePgStatements is the ParseStatementsFunc for PostgreSQL.
// Returns []ParsedStatement with both text and AST populated.
func parsePgStatements(statement string) ([]base.ParsedStatement, error) {
	// First split to get Statement with text and positions
	stmts, err := SplitSQL(statement)
	if err != nil {
		return nil, err
	}

	// Then parse to get ASTs
	parseResults, err := ParsePostgreSQL(statement)
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

// ParsePostgreSQL parses the given SQL and returns a list of ANTLRAST (one per statement).
// Use the PostgreSQL parser based on antlr4.
func ParsePostgreSQL(sql string) ([]*base.ANTLRAST, error) {
	stmts, err := SplitSQL(sql)
	if err != nil {
		return nil, err
	}

	var results []*base.ANTLRAST
	for _, stmt := range stmts {
		if stmt.Empty {
			continue
		}

		parseResult, err := parseSinglePostgreSQL(stmt.Text, stmt.BaseLine())
		if err != nil {
			return nil, err
		}
		results = append(results, parseResult)
	}

	return results, nil
}

// parseSinglePostgreSQL parses a single PostgreSQL statement and returns the ANTLRAST.
func parseSinglePostgreSQL(sql string, baseLine int) (*base.ANTLRAST, error) {
	lexer := parser.NewPostgreSQLLexer(antlr.NewInputStream(sql))
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := parser.NewPostgreSQLParser(stream)
	startPosition := &storepb.Position{Line: int32(baseLine) + 1}
	lexerErrorListener := &base.ParseErrorListener{
		Statement:     sql,
		StartPosition: startPosition,
	}
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(lexerErrorListener)

	parserErrorListener := &base.ParseErrorListener{
		Statement:     sql,
		StartPosition: startPosition,
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

	result := &base.ANTLRAST{
		StartPosition: &storepb.Position{Line: int32(baseLine) + 1},
		Tree:          tree,
		Tokens:        stream,
	}

	return result, nil
}

// ParsePostgreSQLPLBlock parses the given PL/pgSQL block (BEGIN...END) and returns the ANTLRAST.
// Use the PostgreSQL parser based on antlr4, starting from pl_block rule.
func ParsePostgreSQLPLBlock(plBlock string) (*base.ANTLRAST, error) {
	lexer := parser.NewPostgreSQLLexer(antlr.NewInputStream(plBlock))
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := parser.NewPostgreSQLParser(stream)
	lexerErrorListener := &base.ParseErrorListener{
		Statement:     plBlock,
		StartPosition: &storepb.Position{Line: 1},
	}
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(lexerErrorListener)

	parserErrorListener := &base.ParseErrorListener{
		Statement:     plBlock,
		StartPosition: &storepb.Position{Line: 1},
	}
	p.RemoveErrorListeners()
	p.AddErrorListener(parserErrorListener)

	p.BuildParseTrees = true

	// Parse starting from pl_block rule instead of root
	tree := p.Pl_block()
	if lexerErrorListener.Err != nil {
		return nil, lexerErrorListener.Err
	}

	if parserErrorListener.Err != nil {
		return nil, parserErrorListener.Err
	}

	result := &base.ANTLRAST{
		StartPosition: &storepb.Position{Line: 1},
		Tree:          tree,
		Tokens:        stream,
	}

	return result, nil
}

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

func normalizePostgreSQLSetTarget(ctx parser.ISet_targetContext) []string {
	if ctx == nil {
		return []string{}
	}

	var res []string
	res = append(res, NormalizePostgreSQLColid(ctx.Colid()))
	res = append(res, normalizePostgreSQLOptIndirection(ctx.Opt_indirection())...)
	return res
}

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

func normalizePostgreSQLBareColLabel(ctx parser.IBare_col_labelContext) string {
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
