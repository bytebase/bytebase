package tsql

import (
	"strings"
	"unicode"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	parser "github.com/bytebase/tsql-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

type ParseResult struct {
	Tree   antlr.Tree
	Tokens *antlr.CommonTokenStream
}

// ParseTSQL parses the given SQL statement by using antlr4. Returns the AST and token stream if no error.
func ParseTSQL(statement string) (*ParseResult, error) {
	inputStream := antlr.NewInputStream(statement)
	lexer := parser.NewTSqlLexer(inputStream)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := parser.NewTSqlParser(stream)

	// Remove default error listener and add our own error listener.
	lexer.RemoveErrorListeners()
	lexerErrorListener := &base.ParseErrorListener{
		Statement: statement,
	}
	lexer.AddErrorListener(lexerErrorListener)

	p.RemoveErrorListeners()
	parserErrorListener := &base.ParseErrorListener{
		Statement: statement,
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

	result := &ParseResult{
		Tree:   tree,
		Tokens: stream,
	}

	return result, nil
}

func normalizeSimpleNameSeparated(ctx parser.ISimple_nameContext, fallbackSchemaName string, _ bool) (string, string) {
	schema := fallbackSchemaName
	name := ""
	if s := ctx.GetSchema(); s != nil {
		if id, _ := NormalizeTSQLIdentifier(s); id != "" {
			schema = id
		}
	}
	if t := ctx.GetName(); t != nil {
		if id, _ := NormalizeTSQLIdentifier(t); id != "" {
			name = id
		}
	}
	return schema, name
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

func normalizeProcedureSeparated(ctx parser.IFunc_proc_name_schemaContext, fallbackSchemaName string, _ bool) (string, string) {
	schema := fallbackSchemaName
	name := ""
	if s := ctx.GetSchema(); s != nil {
		if id, _ := NormalizeTSQLIdentifier(s); id != "" {
			schema = id
		}
	}
	if t := ctx.GetProcedure(); t != nil {
		if id, _ := NormalizeTSQLIdentifier(t); id != "" {
			name = id
		}
	}
	return schema, name
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
