package plsql

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/plsql"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterParseFunc(storepb.Engine_ORACLE, parsePLSQLForRegistry)
	base.RegisterParseStatementsFunc(storepb.Engine_ORACLE, parsePLSQLStatements)
	base.RegisterGetStatementTypes(storepb.Engine_ORACLE, GetStatementTypes)
}

// parsePLSQLForRegistry is the ParseFunc for PL/SQL.
// Returns []base.AST with *ANTLRAST instances.
func parsePLSQLForRegistry(statement string) ([]base.AST, error) {
	parseResults, err := ParsePLSQL(statement + ";")
	if err != nil {
		return nil, err
	}
	asts := make([]base.AST, len(parseResults))
	for i, r := range parseResults {
		asts[i] = r
	}
	return asts, nil
}

// parsePLSQLStatements is the ParseStatementsFunc for Oracle (PL/SQL).
// Returns []ParsedStatement with both text and AST populated.
func parsePLSQLStatements(statement string) ([]base.ParsedStatement, error) {
	// First split to get Statement with text and positions
	stmts, err := SplitSQL(statement)
	if err != nil {
		return nil, err
	}

	// Then parse to get ASTs (note: ParsePLSQL adds semicolon internally)
	parseResults, err := ParsePLSQL(statement + ";")
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

type Version struct {
	First  int
	Second int
}

// GTE returns true if the version is greater than or equal to the base version.
func (v *Version) GTE(base *Version) bool {
	if v.First > base.First {
		return true
	}
	if v.First == base.First {
		return v.Second >= base.Second
	}
	return false
}

func ParseVersion(banner string) (*Version, error) {
	re := regexp.MustCompile(`(\d+)\.(\d+)`)
	match := re.FindStringSubmatch(banner)
	if len(match) >= 3 {
		firstVersion, err := strconv.Atoi(match[1])
		if err != nil {
			return nil, errors.Errorf("failed to parse first version from banner: %s", banner)
		}
		secondVersion, err := strconv.Atoi(match[2])
		if err != nil {
			return nil, errors.Errorf("failed to parse second version from banner: %s", banner)
		}
		return &Version{First: firstVersion, Second: secondVersion}, nil
	}
	return nil, errors.Errorf("failed to parse version from banner: %s", banner)
}

// ParsePLSQL parses the given PLSQL and returns a list of ANTLR ASTs.
// It first parses the whole statement to get the AST, then splits by unit_statement
// and sql_plus_command nodes, and re-parses each individual statement.
func ParsePLSQL(sql string) ([]*base.ANTLRAST, error) {
	sql = addSemicolonIfNeeded(sql)

	// First pass: parse the whole statement to get the AST for splitting
	tree, tokens, err := parsePLSQLInternal(sql, 0)
	if err != nil {
		return nil, err
	}

	// Type assert to ensure we have a sql_script context
	sqlScript, ok := tree.(*parser.Sql_scriptContext)
	if !ok {
		return nil, errors.Errorf("expected sql_script context, got %T", tree)
	}

	// Iterate through children in order to preserve statement ordering and re-parse each one
	var result []*base.ANTLRAST
	prevStopTokenIndex := -1
	for _, child := range sqlScript.GetChildren() {
		var stmtText string
		var stmtBaseLine int
		var startToken, stopToken antlr.Token

		// Type assert to get the specific statement type
		if stmt, ok := child.(parser.IUnit_statementContext); ok {
			startToken = stmt.GetStart()
			stopToken = stmt.GetStop()
		} else if sqlPlusCmd, ok := child.(parser.ISql_plus_commandContext); ok {
			startToken = sqlPlusCmd.GetStart()
			stopToken = sqlPlusCmd.GetStop()
		} else {
			// Skip other node types (e.g., EOF)
			continue
		}

		// Calculate the leading whitespace/comments before this statement (like SplitSQL does)
		leadingContent := ""
		if startTokenIndex := startToken.GetTokenIndex(); startTokenIndex-1 >= 0 && prevStopTokenIndex+1 <= startTokenIndex-1 {
			leadingContent = tokens.GetTextFromTokens(tokens.Get(prevStopTokenIndex+1), tokens.Get(startTokenIndex-1))
		}

		// Include leading whitespace in the statement text
		stmtText = leadingContent + tokens.GetTextFromTokens(startToken, stopToken)

		// stmtBaseLine is where the leading content starts (for re-parsing with correct offsets).
		// This ensures token positions in the re-parsed AST are correct when combined with SplitSQL's BaseLine.
		// Formula: first token's line - 1 (convert to 0-based) - number of newlines in leading content
		stmtBaseLine = startToken.GetLine() - 1 - strings.Count(leadingContent, "\n")

		prevStopTokenIndex = stopToken.GetTokenIndex()

		// Skip empty statements
		if strings.TrimSpace(stmtText) == "" || stmtText == ";" {
			continue
		}

		// Re-parse the individual statement with correct base line
		stmtTree, stmtTokens, err := parsePLSQLInternal(stmtText, stmtBaseLine)
		if err != nil {
			return nil, err
		}

		// StartPosition points to the first character of Text (including leading whitespace),
		// not the first token. This enables replacing BaseLine with Start.GetLine() - 1 in advisors.
		result = append(result, &base.ANTLRAST{
			StartPosition: &storepb.Position{Line: int32(stmtBaseLine) + 1},
			Tree:          stmtTree,
			Tokens:        stmtTokens,
		})
	}

	return result, nil
}

// parsePLSQLInternal is the internal parsing function that parses a single SQL statement.
func parsePLSQLInternal(sql string, baseLine int) (antlr.Tree, *antlr.CommonTokenStream, error) {
	lexer := parser.NewPlSqlLexer(antlr.NewInputStream(sql))
	stream := antlr.NewCommonTokenStream(lexer, 0)
	p := parser.NewPlSqlParser(stream)
	p.SetVersion12(true)

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
	tree := p.Sql_script()

	if lexerErrorListener.Err != nil {
		return nil, nil, lexerErrorListener.Err
	}

	if parserErrorListener.Err != nil {
		return nil, nil, parserErrorListener.Err
	}

	return tree, stream, nil
}

// ParsePLSQLForStringsManipulation parses the whole SQL without splitting.
// This is used for strings manipulation which needs to see all statements together.
func ParsePLSQLForStringsManipulation(sql string) (antlr.Tree, antlr.TokenStream, error) {
	sql = addSemicolonIfNeeded(sql)
	return parsePLSQLInternal(sql, 0)
}

func addSemicolonIfNeeded(sql string) string {
	lexer := parser.NewPlSqlLexer(antlr.NewInputStream(sql))
	stream := antlr.NewCommonTokenStream(lexer, 0)
	stream.Fill()
	tokens := stream.GetAllTokens()
	for i := len(tokens) - 1; i >= 0; i-- {
		if tokens[i].GetChannel() != antlr.TokenDefaultChannel || tokens[i].GetTokenType() == parser.PlSqlParserEOF {
			continue
		}

		// The last default channel token is a semicolon.
		if tokens[i].GetTokenType() == parser.PlSqlParserSEMICOLON {
			return sql
		}

		return stream.GetTextFromInterval(antlr.NewInterval(0, tokens[i].GetTokenIndex())) + ";"
	}
	return sql
}

// IsOracleKeyword returns true if the given text is an Oracle keyword.
func IsOracleKeyword(text string) bool {
	if len(text) == 0 {
		return false
	}

	return oracleKeywords[strings.ToUpper(text)] || oracleReservedWords[strings.ToUpper(text)]
}

// NormalizeIdentifierContext returns the normalized identifier from the given context.
func NormalizeIdentifierContext(identifier parser.IIdentifierContext) string {
	if identifier == nil {
		return ""
	}

	return NormalizeIDExpression(identifier.Id_expression())
}

// NormalizeIDExpression returns the normalized identifier from the given context.
func NormalizeIDExpression(idExpression parser.IId_expressionContext) string {
	if idExpression == nil {
		return ""
	}

	regularID := idExpression.Regular_id()
	if regularID != nil {
		return strings.ToUpper(regularID.GetText())
	}

	delimitedID := idExpression.DELIMITED_ID()
	if delimitedID != nil {
		return strings.Trim(delimitedID.GetText(), "\"")
	}

	return ""
}

// NormalizeIndexName returns the normalized index name from the given context.
func NormalizeIndexName(indexName parser.IIndex_nameContext) (string, string) {
	if indexName == nil {
		return "", ""
	}

	if indexName.Id_expression() != nil {
		return NormalizeIdentifierContext(indexName.Identifier()),
			NormalizeIDExpression(indexName.Id_expression())
	}

	return "", NormalizeIdentifierContext(indexName.Identifier())
}

// NormalizeTableViewName normalizes the table name and schema name.
// Return empty string if it's xml table.
func NormalizeTableViewName(currentSchema string, ctx parser.ITableview_nameContext) ([]string, string, string) {
	if ctx.Identifier() == nil {
		return nil, "", ""
	}

	var links []string
	for _, link := range ctx.AllLink_name() {
		links = append(links, NormalizeIdentifierContext(link.Identifier()))
	}

	identifier := NormalizeIdentifierContext(ctx.Identifier())

	if ctx.Id_expression() == nil {
		return links, currentSchema, identifier
	}

	idExpression := NormalizeIDExpression(ctx.Id_expression())

	return links, identifier, idExpression
}

// NormalizeColumnName returns the normalized column name from the given context.
func NormalizeColumnName(columnName parser.IColumn_nameContext) (string, string, string) {
	if columnName == nil {
		return "", "", ""
	}

	var list []string
	list = append(list, NormalizeIdentifierContext(columnName.Identifier()))

	for _, idExpression := range columnName.AllId_expression() {
		list = append(list, NormalizeIDExpression(idExpression))
	}

	switch len(list) {
	case 1:
		return "", "", list[0]
	case 2:
		return "", list[0], list[1]
	default:
		return list[0], list[1], list[2]
	}
}

// NormalizeSchemaName returns the normalized schema name from the given context.
func NormalizeSchemaName(schemaName parser.ISchema_nameContext) string {
	if schemaName == nil {
		return ""
	}

	return NormalizeIdentifierContext(schemaName.Identifier())
}

// NormalizeTableName returns the normalized table name from the given context.
func NormalizeTableName(tableName parser.ITable_nameContext) string {
	if tableName == nil {
		return ""
	}

	return NormalizeIdentifierContext(tableName.Identifier())
}

// EquivalentType returns true if the given type is equivalent to the given text.
func EquivalentType(tp parser.IDatatypeContext, text string) (bool, error) {
	results, err := ParsePLSQL(fmt.Sprintf(`CREATE TABLE t(a %s);`, text))
	if err != nil {
		return false, err
	}
	if len(results) == 0 {
		return false, errors.New("no parse results")
	}

	listener := &typeEquivalentListener{tp: tp, equivalent: false}
	antlr.ParseTreeWalkerDefault.Walk(listener, results[0].Tree)
	return listener.equivalent, nil
}

type typeEquivalentListener struct {
	*parser.BasePlSqlParserListener

	tp         parser.IDatatypeContext
	equivalent bool
}

// EnterColumn_definition is called when production column_definition is entered.
func (l *typeEquivalentListener) EnterColumn_definition(ctx *parser.Column_definitionContext) {
	if ctx.Datatype() != nil {
		l.equivalent = equalDataType(l.tp, ctx.Datatype())
	}
}

func equalDataType(lType parser.IDatatypeContext, rType parser.IDatatypeContext) bool {
	if lType == nil || rType == nil {
		return false
	}
	lNative := lType.Native_datatype_element()
	rNative := rType.Native_datatype_element()

	if lNative != nil && rNative != nil {
		switch {
		case lNative.BINARY_INTEGER() != nil:
			return rNative.BINARY_INTEGER() != nil
		case lNative.PLS_INTEGER() != nil:
			return rNative.PLS_INTEGER() != nil
		case lNative.NATURAL() != nil:
			return rNative.NATURAL() != nil
		case lNative.BINARY_FLOAT() != nil:
			return rNative.BINARY_FLOAT() != nil
		case lNative.BINARY_DOUBLE() != nil:
			return rNative.BINARY_DOUBLE() != nil
		case lNative.NATURALN() != nil:
			return rNative.NATURALN() != nil
		case lNative.POSITIVE() != nil:
			return rNative.POSITIVE() != nil
		case lNative.POSITIVEN() != nil:
			return rNative.POSITIVEN() != nil
		case lNative.SIGNTYPE() != nil:
			return rNative.SIGNTYPE() != nil
		case lNative.SIMPLE_INTEGER() != nil:
			return rNative.SIMPLE_INTEGER() != nil
		case lNative.NVARCHAR2() != nil:
			return rNative.NVARCHAR2() != nil
		case lNative.DEC() != nil:
			return rNative.DEC() != nil
		case lNative.INTEGER() != nil:
			return rNative.INTEGER() != nil
		case lNative.INT() != nil:
			return rNative.INT() != nil
		case lNative.NUMERIC() != nil:
			return rNative.NUMERIC() != nil
		case lNative.SMALLINT() != nil:
			return rNative.SMALLINT() != nil
		case lNative.NUMBER() != nil:
			return rNative.NUMBER() != nil
		case lNative.DECIMAL() != nil:
			return rNative.DECIMAL() != nil
		case lNative.DOUBLE() != nil:
			return rNative.DOUBLE() != nil
		case lNative.FLOAT() != nil:
			return rNative.FLOAT() != nil
		case lNative.REAL() != nil:
			return rNative.REAL() != nil
		case lNative.NCHAR() != nil:
			return rNative.NCHAR() != nil
		case lNative.LONG() != nil:
			return rNative.LONG() != nil
		case lNative.CHAR() != nil:
			return rNative.CHAR() != nil
		case lNative.CHARACTER() != nil:
			return rNative.CHARACTER() != nil
		case lNative.VARCHAR2() != nil:
			return rNative.VARCHAR2() != nil
		case lNative.VARCHAR() != nil:
			return rNative.VARCHAR() != nil
		case lNative.STRING() != nil:
			return rNative.STRING() != nil
		case lNative.RAW() != nil:
			return rNative.RAW() != nil
		case lNative.BOOLEAN() != nil:
			return rNative.BOOLEAN() != nil
		case lNative.DATE() != nil:
			return rNative.DATE() != nil
		case lNative.ROWID() != nil:
			return rNative.ROWID() != nil
		case lNative.UROWID() != nil:
			return rNative.UROWID() != nil
		case lNative.YEAR() != nil:
			return rNative.YEAR() != nil
		case lNative.MONTH() != nil:
			return rNative.MONTH() != nil
		case lNative.DAY() != nil:
			return rNative.DAY() != nil
		case lNative.HOUR() != nil:
			return rNative.HOUR() != nil
		case lNative.MINUTE() != nil:
			return rNative.MINUTE() != nil
		case lNative.SECOND() != nil:
			return rNative.SECOND() != nil
		case lNative.TIMEZONE_HOUR() != nil:
			return rNative.TIMEZONE_HOUR() != nil
		case lNative.TIMEZONE_MINUTE() != nil:
			return rNative.TIMEZONE_MINUTE() != nil
		case lNative.TIMEZONE_REGION() != nil:
			return rNative.TIMEZONE_REGION() != nil
		case lNative.TIMEZONE_ABBR() != nil:
			return rNative.TIMEZONE_ABBR() != nil
		case lNative.TIMESTAMP() != nil:
			return rNative.TIMESTAMP() != nil
		case lNative.TIMESTAMP_UNCONSTRAINED() != nil:
			return rNative.TIMESTAMP_UNCONSTRAINED() != nil
		case lNative.TIMESTAMP_TZ_UNCONSTRAINED() != nil:
			return rNative.TIMESTAMP_TZ_UNCONSTRAINED() != nil
		case lNative.TIMESTAMP_LTZ_UNCONSTRAINED() != nil:
			return rNative.TIMESTAMP_LTZ_UNCONSTRAINED() != nil
		case lNative.YMINTERVAL_UNCONSTRAINED() != nil:
			return rNative.YMINTERVAL_UNCONSTRAINED() != nil
		case lNative.DSINTERVAL_UNCONSTRAINED() != nil:
			return rNative.DSINTERVAL_UNCONSTRAINED() != nil
		case lNative.BFILE() != nil:
			return rNative.BFILE() != nil
		case lNative.BLOB() != nil:
			return rNative.BLOB() != nil
		case lNative.CLOB() != nil:
			return rNative.CLOB() != nil
		case lNative.NCLOB() != nil:
			return rNative.NCLOB() != nil
		case lNative.MLSLABEL() != nil:
			return rNative.MLSLABEL() != nil
		default:
			return false
		}
	}

	if lNative != nil || rNative != nil {
		return false
	}

	return lType.GetText() == rType.GetText()
}
