package tsql

import (
	"strings"
	"unicode"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	parser "github.com/bytebase/parser/tsql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterParseFunc(storepb.Engine_MSSQL, parseTSQLForRegistry)
	base.RegisterParseStatementsFunc(storepb.Engine_MSSQL, parseTSQLStatements)
	base.RegisterGetStatementTypes(storepb.Engine_MSSQL, GetStatementTypes)
}

// parseTSQLForRegistry is the ParseFunc for T-SQL.
// Returns []base.AST with *ANTLRAST instances.
func parseTSQLForRegistry(statement string) ([]base.AST, error) {
	antlrASTs, err := ParseTSQL(statement)
	if err != nil {
		return nil, err
	}
	var asts []base.AST
	for _, a := range antlrASTs {
		asts = append(asts, a)
	}
	return asts, nil
}

// parseTSQLStatements is the ParseStatementsFunc for T-SQL (MSSQL).
// Returns []ParsedStatement with both text and AST populated.
func parseTSQLStatements(statement string) ([]base.ParsedStatement, error) {
	// First split to get Statement with text and positions
	stmts, err := SplitSQL(statement)
	if err != nil {
		return nil, err
	}

	// Then parse to get ASTs
	antlrASTs, err := ParseTSQL(statement)
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
		if !stmt.Empty && astIndex < len(antlrASTs) {
			ps.AST = antlrASTs[astIndex]
			astIndex++
		}
		result = append(result, ps)
	}

	return result, nil
}

// ParseTSQL parses the given SQL and returns a list of ANTLRAST (one per statement).
// Use the T-SQL parser based on antlr4.
func ParseTSQL(sql string) ([]*base.ANTLRAST, error) {
	stmts, err := SplitSQL(sql)
	if err != nil {
		return nil, err
	}

	var results []*base.ANTLRAST
	for _, stmt := range stmts {
		if stmt.Empty {
			continue
		}

		antlrAST, err := parseSingleTSQL(stmt.Text, stmt.BaseLine())
		if err != nil {
			return nil, err
		}
		results = append(results, antlrAST)
	}

	return results, nil
}

// parseSingleTSQL parses a single T-SQL statement and returns the ANTLRAST.
func parseSingleTSQL(statement string, baseLine int) (*base.ANTLRAST, error) {
	inputStream := antlr.NewInputStream(statement)
	lexer := parser.NewTSqlLexer(inputStream)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := parser.NewTSqlParser(stream)

	// Remove default error listener and add our own error listener.
	startPosition := &storepb.Position{Line: int32(baseLine) + 1}
	lexer.RemoveErrorListeners()
	lexerErrorListener := &base.ParseErrorListener{
		Statement:     statement,
		StartPosition: startPosition,
	}
	lexer.AddErrorListener(lexerErrorListener)

	p.RemoveErrorListeners()
	parserErrorListener := &base.ParseErrorListener{
		Statement:     statement,
		StartPosition: startPosition,
	}
	p.AddErrorListener(parserErrorListener)

	p.BuildParseTrees = true

	tree := p.Tsql_file()

	if lexerErrorListener.Err != nil {
		return nil, lexerErrorListener.Err
	}

	if parserErrorListener.Err != nil {
		return nil, parserErrorListener.Err
	}

	result := &base.ANTLRAST{
		StartPosition: startPosition,
		Tree:          tree,
		Tokens:        stream,
	}

	return result, nil
}

func normalizeTableNameSeparated(ctx parser.ITable_nameContext, fallbackDatabaseName, fallbackSchemaName string, _ bool) (string, string, string) {
	database := fallbackDatabaseName
	schema := fallbackSchemaName
	table := ""
	if d := ctx.GetDatabase(); d != nil {
		if id, _ := NormalizeTSQLIdentifier(d); id != "" {
			database = id
		}
	}
	if s := ctx.GetSchema(); s != nil {
		if id, _ := NormalizeTSQLIdentifier(s); id != "" {
			schema = id
		}
	}
	if t := ctx.GetTable(); t != nil {
		if id, _ := NormalizeTSQLIdentifier(t); id != "" {
			table = id
		}
	}
	return database, schema, table
}

// NormalizeTSQLTableName returns the normalized table name.
func NormalizeTSQLTableName(ctx parser.ITable_nameContext, fallbackDatabaseName, fallbackSchemaName string, caseSensitive bool) string {
	database, schema, table := normalizeTableNameSeparated(ctx, fallbackDatabaseName, fallbackSchemaName, caseSensitive)
	tableNameParts := []string{}
	if database != "" {
		tableNameParts = append(tableNameParts, database)
	}
	if schema != "" {
		tableNameParts = append(tableNameParts, schema)
	}
	if table != "" {
		tableNameParts = append(tableNameParts, table)
	}
	return strings.Join(tableNameParts, ".")
}

// NormalizeTSQLIdentifier returns the normalized identifier.
// https://learn.microsoft.com/zh-cn/sql/relational-databases/databases/database-identifiers?view=sql-server-ver15
// TODO(zp): currently, we returns the lowercase and original of the part, we may need to get the CI/CS from the server/database.
func NormalizeTSQLIdentifier(part parser.IId_Context) (original string, lowercase string) {
	if part == nil {
		return "", ""
	}
	text := part.GetText()
	return NormalizeTSQLIdentifierText(text)
}

func NormalizeTSQLIdentifierText(text string) (original string, lowercase string) {
	if text == "" {
		return "", ""
	}
	if text[0] == '[' && text[len(text)-1] == ']' {
		text = text[1 : len(text)-1]
	}

	s := ""
	for _, r := range text {
		s += string(unicode.ToLower(r))
	}
	return text, s
}

// IsTSQLReservedKeyword returns true if the given keyword is a TSQL keywords.
func IsTSQLReservedKeyword(keyword string, caseSensitive bool) bool {
	if !caseSensitive {
		keyword = strings.ToUpper(keyword)
	}
	return tsqlReservedKeywordsMap[keyword]
}

// FlattenExecuteStatementArgExecuteStatementArgUnnamed returns the flattened unnamed execute statement arg.
func FlattenExecuteStatementArgExecuteStatementArgUnnamed(ctx parser.IExecute_statement_argContext) []parser.IExecute_statement_arg_unnamedContext {
	var queue []parser.IExecute_statement_arg_unnamedContext
	ele := ctx
	for ele.Execute_statement_arg_unnamed() != nil {
		queue = append(queue, ele.Execute_statement_arg_unnamed())
		if len(ele.AllExecute_statement_arg()) != 1 {
			break
		}
		ele = ele.AllExecute_statement_arg()[0]
	}
	return queue
}

type FullTableName struct {
	LinkedServer string
	Server       string
	Database     string
	Schema       string
	Table        string
}

// full_table_name
//
//	: (linkedServer=id_ '.' '.' schema=id_   '.'
//	|                       server=id_    '.' database=id_ '.'  schema=id_   '.'
//	|                                         database=id_ '.'  schema=id_? '.'
//	|                                                           schema=id_    '.')? table=id_
//	;
//
// Rewrite full_table_name to avoid leading optional values.
// full_table_name
//
//	: id_ (
//	        dotID+
//	        | doubleDotID dotID?
//	    )?
//	;
//
// For performance reason, we rewrite the full_table_name to the second form.
// But it's hard to get the linkedServer, server, database, schema, and table separately.
// So we use NormalizeFullTableName to get them.
func NormalizeFullTableName(ctx parser.IFull_table_nameContext) (*FullTableName, error) {
	fullTableName := FullTableName{}
	if ctx == nil {
		return &fullTableName, nil
	}

	id, _ := NormalizeTSQLIdentifier(ctx.Id_())
	if ctx.DoubleDotID() != nil {
		if len(ctx.AllDotID()) > 0 {
			// linkedServer..schema.table
			fullTableName.LinkedServer = id
			fullTableName.Schema, _ = NormalizeTSQLIdentifier(ctx.DoubleDotID().Id_())
			fullTableName.Table, _ = NormalizeTSQLIdentifier(ctx.DotID(0).Id_())
		} else {
			// database..table
			fullTableName.Database = id
			fullTableName.Table, _ = NormalizeTSQLIdentifier(ctx.DoubleDotID().Id_())
		}
	} else {
		// dotID+
		var ids []string
		ids = append(ids, id)
		for _, dotID := range ctx.AllDotID() {
			id, _ = NormalizeTSQLIdentifier(dotID.Id_())
			ids = append(ids, id)
		}
		switch len(ids) {
		case 1:
			fullTableName.Table = ids[0]
		case 2:
			fullTableName.Schema = ids[0]
			fullTableName.Table = ids[1]
		case 3:
			fullTableName.Database = ids[0]
			fullTableName.Schema = ids[1]
			fullTableName.Table = ids[2]
		case 4:
			fullTableName.Server = ids[0]
			fullTableName.Database = ids[1]
			fullTableName.Schema = ids[2]
			fullTableName.Table = ids[3]
		default:
			return nil, errors.New("invalid full table name")
		}
	}

	return &fullTableName, nil
}

func NormalizeFullColumnName(ctx parser.IFull_column_nameContext) (*FullTableName, string, error) {
	if ctx == nil {
		return nil, "", nil
	}
	fullTableName := (*FullTableName)(nil)
	var err error
	if ctx.Full_table_name() != nil {
		fullTableName, err = NormalizeFullTableName(ctx.Full_table_name())
		if err != nil {
			return nil, "", err
		}
	}
	if ctx.GetColumn_name() == nil {
		return fullTableName, "", nil
	}
	id, _ := NormalizeTSQLIdentifier(ctx.GetColumn_name())
	return fullTableName, id, nil
}
