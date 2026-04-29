package tsql

import (
	"context"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/antlr4-go/antlr/v4"
	mssqlparser "github.com/bytebase/omni/mssql/parser"
	tsqlparser "github.com/bytebase/parser/tsql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	base.RegisterCompleteFunc(storepb.Engine_MSSQL, Completion)
}

var (
	// Check tableRefListener is implementing the TSqlParserListener interface.
	_ tsqlparser.TSqlParserListener = &tableRefListener{}
	_ tsqlparser.TSqlParserListener = &cteExtractor{}

	tsqlDataTypes = []string{
		"INT", "BIGINT", "SMALLINT", "TINYINT",
		"VARCHAR", "NVARCHAR", "CHAR", "NCHAR",
		"TEXT", "NTEXT",
		"DATETIME", "DATETIME2", "DATE", "TIME",
		"DECIMAL", "NUMERIC", "FLOAT", "REAL",
		"BIT",
		"MONEY", "SMALLMONEY",
		"UNIQUEIDENTIFIER",
		"XML",
		"VARBINARY", "IMAGE",
		"SQL_VARIANT",
	}
	tsqlTableHints = []string{
		"NOLOCK", "READUNCOMMITTED", "READCOMMITTED", "REPEATABLEREAD",
		"SERIALIZABLE", "HOLDLOCK", "UPDLOCK", "TABLOCK", "TABLOCKX",
		"ROWLOCK", "PAGLOCK", "INDEX", "FORCESEEK",
	}
	tsqlQueryHints = []string{
		"RECOMPILE", "OPTIMIZE", "MAXDOP", "HASH JOIN", "MERGE JOIN",
		"LOOP JOIN", "FORCE ORDER",
	}
)

const asciiWhitespace = " \t\r\n"

type CompletionMap map[string]base.Candidate

func (m CompletionMap) Insert(entry base.Candidate) {
	m[entry.String()] = entry
}

func (m CompletionMap) toSlice() []base.Candidate {
	return slices.SortedFunc(maps.Values(m), compareCandidates)
}

func compareCandidates(a, b base.Candidate) int {
	if a.Type != b.Type {
		if a.Type < b.Type {
			return -1
		}
		return 1
	}
	if a.Text < b.Text {
		return -1
	}
	if a.Text > b.Text {
		return 1
	}
	return 0
}

// insertFunctions inserts the built-in functions into the completion map.
func (m CompletionMap) insertBuiltinFunctions() {
	for key := range tsqlBuiltinFunctionsMap {
		m[key] = base.Candidate{
			Type: base.CandidateTypeFunction,
			Text: key + "()",
		}
	}
}

func (m CompletionMap) insertMetadataDatabases(c *Completer, linkedServer string) {
	if linkedServer != "" {
		return
	}

	if c.defaultDatabase != "" {
		m[c.defaultDatabase] = base.Candidate{
			Type: base.CandidateTypeDatabase,
			Text: c.quotedIdentifierIfNeeded(c.defaultDatabase),
		}
	}

	allDatabase, err := c.databaseNamesLister(c.ctx, c.instanceID)
	if err != nil {
		return
	}

	for _, database := range allDatabase {
		if _, ok := m[database]; !ok {
			m[database] = base.Candidate{
				Type: base.CandidateTypeDatabase,
				Text: c.quotedIdentifierIfNeeded(database),
			}
		}
	}
}

func (m CompletionMap) insertMetadataSchemas(c *Completer, linkedServer string, database string) {
	if linkedServer != "" {
		return
	}

	anchor := c.defaultDatabase
	if database != "" {
		anchor = database
	}
	if anchor == "" {
		return
	}

	allDBNames, err := c.databaseNamesLister(c.ctx, c.instanceID)
	if err != nil {
		return
	}
	for _, dbName := range allDBNames {
		if strings.EqualFold(dbName, anchor) {
			anchor = dbName
			break
		}
	}

	_, databaseMetadata, err := c.metadataGetter(c.ctx, c.instanceID, anchor)
	if err != nil {
		return
	}

	for _, schema := range databaseMetadata.ListSchemaNames() {
		if _, ok := m[schema]; !ok {
			m[schema] = base.Candidate{
				Type: base.CandidateTypeSchema,
				Text: c.quotedIdentifierIfNeeded(schema),
			}
		}
	}
}

func (m CompletionMap) insertMetadataTables(c *Completer, linkedServer string, database string, schema string) {
	schemaMetadata := c.lookupMetadataSchema(linkedServer, database, schema)
	if schemaMetadata == nil {
		return
	}
	m.insertNamedCandidates(c, schemaMetadata.ListTableNames(), base.CandidateTypeTable)
}

func (m CompletionMap) insertAllColumns(c *Completer) {
	_, databaseMeta, err := c.metadataGetter(c.ctx, c.instanceID, c.defaultDatabase)
	if err != nil {
		return
	}
	for _, schema := range databaseMeta.ListSchemaNames() {
		schemaMeta := databaseMeta.GetSchemaMetadata(schema)
		if schemaMeta == nil {
			continue
		}
		for _, table := range schemaMeta.ListTableNames() {
			tableMeta := schemaMeta.GetTable(table)
			if tableMeta == nil {
				continue
			}
			for _, column := range tableMeta.GetProto().GetColumns() {
				columnID := getColumnID(c.defaultDatabase, schema, table, column.Name)
				if _, ok := m[columnID]; !ok {
					definition := fmt.Sprintf("%s.%s.%s | %s", c.defaultDatabase, schema, table, column.Type)
					if !column.Nullable {
						definition += ", NOT NULL"
					}
					m[columnID] = base.Candidate{
						Type:       base.CandidateTypeColumn,
						Text:       c.quotedIdentifierIfNeeded(column.Name),
						Definition: definition,
						Comment:    column.Comment,
						Priority:   c.getPriority(c.defaultDatabase, schema, table),
					}
				}
			}
		}
	}
}

func (c *Completer) getPriority(database string, schema string, table string) int {
	if database == "" {
		database = c.defaultDatabase
	}
	if schema == "" {
		schema = c.defaultSchema
	}
	if c.referenceMap == nil {
		return 1
	}
	if c.referenceMap[fmt.Sprintf("%s.%s.%s", database, schema, table)] {
		// The higher priority.
		return 0
	}
	return 1
}

func (m CompletionMap) insertMetadataColumns(c *Completer, linkedServer string, database string, schema string, table string) {
	if linkedServer != "" {
		return
	}
	databaseName, schemaName, tableName := c.defaultDatabase, c.defaultSchema, ""
	if database != "" {
		databaseName = database
	}
	if schema != "" {
		schemaName = schema
	}
	if table != "" {
		tableName = table
	}
	if databaseName == "" || schemaName == "" {
		return
	}
	databaseNames, err := c.databaseNamesLister(c.ctx, c.instanceID)
	if err != nil {
		return
	}
	for _, dbName := range databaseNames {
		if strings.EqualFold(dbName, databaseName) {
			databaseName = dbName
			break
		}
	}
	_, databaseMetadata, err := c.metadataGetter(c.ctx, c.instanceID, databaseName)
	if err != nil {
		return
	}
	if databaseMetadata == nil {
		return
	}
	for _, schema := range databaseMetadata.ListSchemaNames() {
		if strings.EqualFold(schema, schemaName) {
			schemaName = schema
			break
		}
	}
	schemaMetadata := databaseMetadata.GetSchemaMetadata(schemaName)
	if schemaMetadata == nil {
		return
	}
	var tableNames []string
	for _, table := range schemaMetadata.ListTableNames() {
		if tableName == "" {
			tableNames = append(tableNames, table)
		} else if strings.EqualFold(table, tableName) {
			tableNames = append(tableNames, table)
			break
		}
	}
	for _, table := range tableNames {
		tableMetadata := schemaMetadata.GetTable(table)
		for _, column := range tableMetadata.GetProto().GetColumns() {
			columnID := getColumnID(databaseName, schemaName, table, column.Name)
			if _, ok := m[columnID]; !ok {
				definition := fmt.Sprintf("%s.%s.%s | %s", databaseName, schemaName, table, column.Type)
				if !column.Nullable {
					definition += ", NOT NULL"
				}
				m[columnID] = base.Candidate{
					Type:       base.CandidateTypeColumn,
					Text:       c.quotedIdentifierIfNeeded(column.Name),
					Definition: definition,
					Comment:    column.Comment,
					Priority:   c.getPriority(c.defaultDatabase, schema, table),
				}
			}
		}
	}
}

func (m CompletionMap) insertMetadataColumnsExcept(c *Completer, linkedServer string, database string, schema string, table string, excluded map[string]bool) {
	columns := make(CompletionMap)
	columns.insertMetadataColumns(c, linkedServer, database, schema, table)
	if len(columns) == 0 {
		return
	}
	if len(excluded) == 0 {
		m.replaceColumns(columns)
		return
	}
	for key, candidate := range columns {
		original, _ := NormalizeTSQLIdentifierText(candidate.Text)
		if excluded[strings.ToLower(original)] {
			delete(columns, key)
		}
	}
	m.replaceColumns(columns)
}

func (m CompletionMap) replaceWithMetadataColumns(c *Completer, linkedServer string, database string, schema string, table string) {
	columns := make(CompletionMap)
	columns.insertMetadataColumns(c, linkedServer, database, schema, table)
	if len(columns) == 0 {
		return
	}
	m.replaceColumns(columns)
}

func (m CompletionMap) replaceWithLocalColumns(c *Completer, columns []string) {
	if len(columns) == 0 {
		return
	}
	entries := make(CompletionMap)
	for _, column := range columns {
		entries.Insert(base.Candidate{
			Type: base.CandidateTypeColumn,
			Text: c.quotedIdentifierIfNeeded(column),
		})
	}
	m.replaceColumns(entries)
}

func (m CompletionMap) replaceColumns(columns CompletionMap) {
	for key, candidate := range m {
		if candidate.Type == base.CandidateTypeColumn {
			delete(m, key)
		}
	}
	for _, candidate := range columns {
		m.Insert(candidate)
	}
}

func getColumnID(databaseName, schemaName, tableName, columnName string) string {
	return fmt.Sprintf("%s.%s.%s.%s", databaseName, schemaName, tableName, columnName)
}

func (m CompletionMap) insertCTEs(c *Completer) {
	for _, cte := range c.cteTables {
		if _, ok := m[cte.Table]; !ok {
			m[cte.Table] = base.Candidate{
				Type: base.CandidateTypeTable,
				Text: c.quotedIdentifierIfNeeded(cte.Table),
			}
		}
	}
}

func (m CompletionMap) insertMetadataViews(c *Completer, linkedServer string, database string, schema string) {
	schemaMetadata := c.lookupMetadataSchema(linkedServer, database, schema)
	if schemaMetadata == nil {
		return
	}
	m.insertNamedCandidates(c, schemaMetadata.ListViewNames(), base.CandidateTypeView)
	m.insertNamedCandidates(c, schemaMetadata.ListMaterializedViewNames(), base.CandidateTypeView)
	m.insertNamedCandidates(c, schemaMetadata.ListForeignTableNames(), base.CandidateTypeView)
}

func (m CompletionMap) insertMetadataSequences(c *Completer, linkedServer string, database string, schema string) {
	schemaMetadata := c.lookupMetadataSchema(linkedServer, database, schema)
	if schemaMetadata == nil {
		return
	}
	m.insertNamedCandidates(c, schemaMetadata.ListSequenceNames(), base.CandidateTypeSequence)
}

func (m CompletionMap) insertMetadataProcedures(c *Completer, linkedServer string, database string, schema string) {
	schemaMetadata := c.lookupMetadataSchema(linkedServer, database, schema)
	if schemaMetadata == nil {
		return
	}
	m.insertNamedCandidates(c, schemaMetadata.ListProcedureNames(), base.CandidateTypeRoutine)
}

func (c *Completer) lookupMetadataSchema(linkedServer, database, schema string) *model.SchemaMetadata {
	if linkedServer != "" {
		return nil
	}

	databaseName, schemaName := c.defaultDatabase, c.defaultSchema
	if database != "" {
		databaseName = database
	}
	if schema != "" {
		schemaName = schema
	}
	if databaseName == "" || schemaName == "" {
		return nil
	}

	_, databaseMetadata, err := c.metadataGetter(c.ctx, c.instanceID, databaseName)
	if err != nil || databaseMetadata == nil {
		return nil
	}
	for _, candidate := range databaseMetadata.ListSchemaNames() {
		if strings.EqualFold(candidate, schemaName) {
			schemaName = candidate
			break
		}
	}

	return databaseMetadata.GetSchemaMetadata(schemaName)
}

func (m CompletionMap) insertNamedCandidates(c *Completer, names []string, candidateType base.CandidateType) {
	for _, name := range names {
		if _, ok := m[name]; ok {
			continue
		}
		m[name] = base.Candidate{
			Type: candidateType,
			Text: c.quotedIdentifierIfNeeded(name),
		}
	}
}

// quptedType is the type of quoted token, SQL Server allows quoted identifiers by different characters.
// https://learn.microsoft.com/en-us/sql/relational-databases/databases/database-identifiers?view=sql-server-ver16
type quotedType int

const (
	quotedTypeNone          quotedType = iota
	quotedTypeDoubleQuote              // ""
	quotedTypeSquareBracket            // []
)

type Completer struct {
	ctx     context.Context
	scene   base.SceneType
	parser  *tsqlparser.TSqlParser
	lexer   *tsqlparser.TSqlLexer
	scanner *base.Scanner

	sql              string
	cursorByteOffset int
	tokens           []mssqlparser.Token
	caretTokenIndex  int

	instanceID          string
	defaultDatabase     string
	defaultSchema       string
	metadataGetter      base.GetDatabaseMetadataFunc
	databaseNamesLister base.ListDatabaseNamesFunc

	cteCache map[int][]*base.VirtualTableReference
	// referencesStack is a hierarchical stack of table references.
	// We'll update the stack when we encounter a new FROM clauses.
	referencesStack [][]base.TableReference
	// references is the flattened table references.
	// It's helpful to look up the table reference.
	references          []base.TableReference
	referenceMap        map[string]bool
	cteTables           []*base.VirtualTableReference
	caretTokenIsQuoted  quotedType
	noSeparatorRequired map[int]bool
}

func Completion(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) ([]base.Candidate, error) {
	completer := NewStandardCompleter(ctx, cCtx, statement, caretLine, caretOffset)
	completer.fetchCommonTableExpression(statement)
	result, err := completer.complete()

	if err != nil {
		return nil, err
	}
	if len(result) > 0 {
		return result, nil
	}

	trickyCompleter := NewTrickyCompleter(ctx, cCtx, statement, caretLine, caretOffset)
	trickyCompleter.fetchCommonTableExpression(statement)
	return trickyCompleter.complete()
}

func NewStandardCompleter(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) *Completer {
	parser, lexer, scanner := prepareParserAndScanner(statement, caretLine, caretOffset)
	sql, byteOffset := computeSQLAndByteOffset(statement, caretLine, caretOffset, false /* tricky */)
	return newCompleter(ctx, cCtx, parser, lexer, scanner, sql, byteOffset)
}

func NewTrickyCompleter(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) *Completer {
	parser, lexer, scanner := prepareTrickyParserAndScanner(statement, caretLine, caretOffset)
	sql, byteOffset := computeSQLAndByteOffset(statement, caretLine, caretOffset, true /* tricky */)
	return newCompleter(ctx, cCtx, parser, lexer, scanner, sql, byteOffset)
}

func newCompleter(ctx context.Context, cCtx base.CompletionContext, parser *tsqlparser.TSqlParser, lexer *tsqlparser.TSqlLexer, scanner *base.Scanner, sql string, byteOffset int) *Completer {
	tokens := mssqlparser.Tokenize(sql)

	return &Completer{
		ctx:                 ctx,
		scene:               cCtx.Scene,
		parser:              parser,
		lexer:               lexer,
		scanner:             scanner,
		sql:                 sql,
		cursorByteOffset:    byteOffset,
		tokens:              tokens,
		caretTokenIndex:     findCaretTokenIndex(tokens, byteOffset),
		instanceID:          cCtx.InstanceID,
		defaultDatabase:     cCtx.DefaultDatabase,
		defaultSchema:       "dbo",
		metadataGetter:      cCtx.Metadata,
		databaseNamesLister: cCtx.ListDatabaseNames,
		cteCache:            nil,
	}
}

func computeSQLAndByteOffset(statement string, caretLine int, caretOffset int, tricky bool) (string, int) {
	sql, newLine, newOffset := skipHeadingSQLs(statement, caretLine, caretOffset)
	if tricky {
		sql, newLine, newOffset = skipHeadingSQLWithoutSemicolon(sql, newLine, newOffset)
	}
	return sql, lineColumnToByteOffset(sql, newLine, newOffset)
}

func lineColumnToByteOffset(sql string, line, column int) int {
	currentLine := 1
	for i := 0; i < len(sql); i++ {
		if currentLine == line {
			pos := i
			for c := 0; c < column && pos < len(sql); c++ {
				_, size := utf8.DecodeRuneInString(sql[pos:])
				pos += size
			}
			if pos > len(sql) {
				return len(sql)
			}
			return pos
		}
		if sql[i] == '\n' {
			currentLine++
		}
	}
	return len(sql)
}

func findCaretTokenIndex(tokens []mssqlparser.Token, byteOffset int) int {
	for i, tok := range tokens {
		if tok.Loc >= byteOffset {
			return i
		}
	}
	return len(tokens)
}

func prepareParserAndScanner(statement string, caretLine int, caretOffset int) (*tsqlparser.TSqlParser, *tsqlparser.TSqlLexer, *base.Scanner) {
	statement, caretLine, caretOffset = skipHeadingSQLs(statement, caretLine, caretOffset)
	input := antlr.NewInputStream(statement)
	lexer := tsqlparser.NewTSqlLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := tsqlparser.NewTSqlParser(stream)
	parser.RemoveErrorListeners()
	lexer.RemoveErrorListeners()
	scanner := base.NewScanner(stream, true /* fillInput */)
	scanner.SeekPosition(caretLine, caretOffset)
	scanner.Push()
	return parser, lexer, scanner
}

func prepareTrickyParserAndScanner(statement string, caretLine int, caretOffset int) (*tsqlparser.TSqlParser, *tsqlparser.TSqlLexer, *base.Scanner) {
	statement, caretLine, caretOffset = skipHeadingSQLs(statement, caretLine, caretOffset)
	statement, caretLine, caretOffset = skipHeadingSQLWithoutSemicolon(statement, caretLine, caretOffset)
	input := antlr.NewInputStream(statement)
	lexer := tsqlparser.NewTSqlLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := tsqlparser.NewTSqlParser(stream)
	parser.RemoveErrorListeners()
	lexer.RemoveErrorListeners()
	scanner := base.NewScanner(stream, true /* fillInput */)
	scanner.SeekPosition(caretLine, caretOffset)
	scanner.Push()
	return parser, lexer, scanner
}

// caretLine is 1-based and caretOffset is 0-based.
func skipHeadingSQLWithoutSemicolon(statement string, caretLine int, caretOffset int) (string, int, int) {
	input := antlr.NewInputStream(statement)
	lexer := tsqlparser.NewTSqlLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	lexer.RemoveErrorListeners()
	lexerErrorListener := &base.ParseErrorListener{
		Statement: statement,
	}
	lexer.AddErrorListener(lexerErrorListener)

	stream.Fill()
	tokens := stream.GetAllTokens()
	latestSelect := 0
	newCaretLine, newCaretOffset := caretLine, caretOffset
	for _, token := range tokens {
		if token.GetLine() > caretLine || (token.GetLine() == caretLine && token.GetColumn() >= caretOffset) {
			break
		}
		if token.GetTokenType() == tsqlparser.TSqlLexerSELECT && token.GetColumn() == 0 {
			latestSelect = token.GetTokenIndex()
			newCaretLine = caretLine - token.GetLine() + 1 // convert to 1-based.
			newCaretOffset = caretOffset
		}
	}

	if latestSelect == 0 {
		return statement, caretLine, caretOffset
	}
	return stream.GetTextFromInterval(antlr.NewInterval(latestSelect, stream.Size())), newCaretLine, newCaretOffset
}

func (c *Completer) complete() ([]base.Candidate, error) {
	checkTokenQuoted := func(idx int) quotedType {
		if idx < 0 || idx >= len(c.tokens) {
			return quotedTypeNone
		}
		tok := c.tokens[idx]
		if !mssqlparser.IsIdentTokenType(tok.Type) || tok.Loc >= len(c.sql) {
			return quotedTypeNone
		}
		switch c.sql[tok.Loc] {
		case '"':
			return quotedTypeDoubleQuote
		case '[':
			return quotedTypeSquareBracket
		default:
			return quotedTypeNone
		}
	}
	if typ := checkTokenQuoted(c.caretTokenIndex); typ != quotedTypeNone {
		c.caretTokenIsQuoted = typ
	} else if c.caretTokenIndex > 0 {
		prev := c.tokens[c.caretTokenIndex-1]
		if prev.End >= c.cursorByteOffset {
			c.caretTokenIsQuoted = checkTokenQuoted(c.caretTokenIndex - 1)
		}
	}

	caretIndex := c.scanner.GetIndex()
	if caretIndex > 0 && !c.noSeparatorRequired[c.scanner.GetPreviousTokenType(true)] {
		caretIndex--
	}
	c.referencesStack = append([][]base.TableReference{{}}, c.referencesStack...)

	completionOffset := c.cursorByteOffset
	candidates := mssqlparser.Collect(c.sql, completionOffset)
	if len(candidates.Rules) == 0 {
		if prefixTok, ok := c.prefixToken(); ok {
			completionOffset = prefixTok.Loc
			candidates = mssqlparser.Collect(c.sql, completionOffset)
		}
	}

	for _, rc := range candidates.Rules {
		if rc.Rule == "columnref" {
			c.collectLeadingTableReferences(caretIndex)
			c.takeReferencesSnapshot()
			c.collectRemainingTableReferences()
			c.takeReferencesSnapshot()
			break
		}
	}
	return c.convertCandidates(candidates)
}

func (c *Completer) prefixToken() (mssqlparser.Token, bool) {
	idx := c.caretTokenIndex
	if idx < len(c.tokens) {
		tok := c.tokens[idx]
		if tok.Loc < c.cursorByteOffset && c.cursorByteOffset <= tok.End && mssqlparser.IsIdentTokenType(tok.Type) {
			return tok, true
		}
	}
	if idx > 0 {
		tok := c.tokens[idx-1]
		if tok.End == c.cursorByteOffset && mssqlparser.IsIdentTokenType(tok.Type) {
			return tok, true
		}
	}
	return mssqlparser.Token{}, false
}

func (c *Completer) convertCandidates(candidates *mssqlparser.CandidateSet) ([]base.Candidate, error) {
	keywordEntries := make(CompletionMap)
	functionEntries := make(CompletionMap)
	databaseEntries := make(CompletionMap)
	schemaEntries := make(CompletionMap)
	tableEntries := make(CompletionMap)
	columnEntries := make(CompletionMap)
	viewEntries := make(CompletionMap)
	sequenceEntries := make(CompletionMap)
	routineEntries := make(CompletionMap)

	for _, tokenCandidate := range candidates.Tokens {
		candidateText := mssqlparser.TokenName(tokenCandidate)
		if candidateText == "" {
			continue
		}
		keywordEntries.Insert(base.Candidate{
			Type: base.CandidateTypeKeyword,
			Text: candidateText,
		})
	}

	for _, ruleCandidate := range candidates.Rules {
		c.scanner.PopAndRestore()
		c.scanner.Push()

		switch ruleCandidate.Rule {
		case "func_name":
			functionEntries.insertBuiltinFunctions()
		case "database_ref":
			databaseEntries.insertMetadataDatabases(c, "")
		case "schema_ref":
			schemaEntries.insertMetadataSchemas(c, "", "")
		case "proc_ref":
			for _, context := range c.determineFullTableNameContext() {
				if context.flags&objectFlagShowObject != 0 {
					routineEntries.insertMetadataProcedures(c, context.linkedServer, context.database, context.schema)
				}
			}
		case "sequence_ref":
			for _, context := range c.determineFullTableNameContext() {
				if context.flags&objectFlagShowObject != 0 {
					sequenceEntries.insertMetadataSequences(c, context.linkedServer, context.database, context.schema)
				}
			}
		case "type_name":
			for _, typ := range tsqlDataTypes {
				keywordEntries.Insert(base.Candidate{
					Type: base.CandidateTypeKeyword,
					Text: typ,
				})
			}
		case "table_ref":
			completionContexts := c.determineFullTableNameContext()
			for _, context := range completionContexts {
				c.insertObjectCandidates(context, databaseEntries, schemaEntries, tableEntries, viewEntries, sequenceEntries)
			}
		case "asterisk":
			completionContexts := c.determineAsteriskContext()
			for _, context := range completionContexts {
				c.insertObjectCandidates(context, databaseEntries, schemaEntries, tableEntries, viewEntries, sequenceEntries)
			}
		case "columnref":
			completionContexts := c.determineFullColumnName()
			for _, context := range completionContexts {
				c.insertObjectCandidates(context, databaseEntries, schemaEntries, tableEntries, viewEntries, sequenceEntries)
				if context.flags&objectFlagShowObject != 0 {
					for _, reference := range c.references {
						switch reference := reference.(type) {
						case *base.PhysicalTableReference:
							tableName := reference.Table
							if reference.Alias != "" {
								tableName = reference.Alias
							}
							if context.linkedServer == "" && strings.EqualFold(reference.Database, context.database) &&
								strings.EqualFold(reference.Schema, context.schema) {
								if _, ok := tableEntries[tableName]; !ok {
									tableEntries[tableName] = base.Candidate{
										Type: base.CandidateTypeTable,
										Text: c.quotedIdentifierIfNeeded(tableName),
									}
								}
							}
						case *base.VirtualTableReference:
							// We only append the virtual table reference to the completion list when the linkedServer, database and schema are all empty.
							if context.linkedServer == "" && context.database == "" && context.schema == "" {
								tableEntries[reference.Table] = base.Candidate{
									Type: base.CandidateTypeTable,
									Text: c.quotedIdentifierIfNeeded(reference.Table),
								}
							}
						default:
						}
					}
				}
				if context.flags&objectFlagShowColumn != 0 {
					if aliasStartOffset, ok := c.selectItemAliasStartOffset(); ok {
						list := c.fetchSelectItemAliases(candidates.SelectAliasPositions, aliasStartOffset)
						for _, alias := range list {
							columnEntries.Insert(base.Candidate{
								Type: base.CandidateTypeColumn,
								Text: c.quotedIdentifierIfNeeded(alias),
							})
						}
					}
					columnEntries.insertMetadataColumns(c, context.linkedServer, context.database, context.schema, context.object)
					for _, reference := range c.references {
						switch reference := reference.(type) {
						case *base.PhysicalTableReference:
							inputLinkedServer := context.linkedServer
							inputDatabaseName := context.database
							if inputDatabaseName == "" {
								inputDatabaseName = c.defaultDatabase
							}
							inputSchemaName := context.schema
							if inputSchemaName == "" {
								inputSchemaName = c.defaultSchema
							}
							inputTableName := context.object

							referenceDatabaseName := reference.Database
							if referenceDatabaseName == "" {
								referenceDatabaseName = c.defaultDatabase
							}
							referenceSchemaName := reference.Schema
							if referenceSchemaName == "" {
								referenceSchemaName = c.defaultSchema
							}
							referenceTableName := reference.Table
							if reference.Alias != "" {
								referenceTableName = reference.Alias
							}

							if inputLinkedServer == "" && strings.EqualFold(referenceDatabaseName, inputDatabaseName) &&
								strings.EqualFold(referenceSchemaName, inputSchemaName) &&
								strings.EqualFold(referenceTableName, inputTableName) {
								columnEntries.insertMetadataColumns(c, "", reference.Database, reference.Schema, reference.Table)
							}
						case *base.VirtualTableReference:
							// Reference could be a physical table reference or a virtual table reference, if the reference is a virtual table reference,
							// and users do not specify the server, database and schema, we should also insert the columns.
							if context.linkedServer == "" && context.database == "" && context.schema == "" {
								for _, column := range reference.Columns {
									if _, ok := columnEntries[column]; !ok {
										columnEntries[column] = base.Candidate{
											Type: base.CandidateTypeColumn,
											Text: c.quotedIdentifierIfNeeded(column),
										}
									}
								}
							}
						default:
						}
					}
					if context.linkedServer == "" && context.database == "" && context.schema == "" {
						for _, cte := range c.cteTables {
							if strings.EqualFold(cte.Table, context.object) {
								for _, column := range cte.Columns {
									if _, ok := columnEntries[column]; !ok {
										columnEntries[column] = base.Candidate{
											Type: base.CandidateTypeColumn,
											Text: c.quotedIdentifierIfNeeded(column),
										}
									}
								}
							}
						}
					}
					if context.empty() {
						columnEntries.insertAllColumns(c)
					}
				}
			}
		default:
			// Handle other candidates
		}
	}
	c.insertContextualMSSQLKeywords(keywordEntries)
	c.insertContextualMSSQLCandidates(columnEntries)

	c.scanner.PopAndRestore()
	var result []base.Candidate
	result = append(result, keywordEntries.toSlice()...)
	result = append(result, functionEntries.toSlice()...)
	result = append(result, databaseEntries.toSlice()...)
	result = append(result, schemaEntries.toSlice()...)
	result = append(result, tableEntries.toSlice()...)
	result = append(result, columnEntries.toSlice()...)
	result = append(result, viewEntries.toSlice()...)
	result = append(result, sequenceEntries.toSlice()...)
	result = append(result, routineEntries.toSlice()...)
	return result, nil
}

func (c *Completer) insertObjectCandidates(context *objectRefContext, databaseEntries, schemaEntries, tableEntries, viewEntries, sequenceEntries CompletionMap) {
	if context.flags&objectFlagShowDatabase != 0 {
		databaseEntries.insertMetadataDatabases(c, context.linkedServer)
	}
	if context.flags&objectFlagShowSchema != 0 {
		schemaEntries.insertMetadataSchemas(c, context.linkedServer, context.database)
	}
	if context.flags&objectFlagShowObject == 0 {
		return
	}
	tableEntries.insertMetadataTables(c, context.linkedServer, context.database, context.schema)
	viewEntries.insertMetadataViews(c, context.linkedServer, context.database, context.schema)
	sequenceEntries.insertMetadataSequences(c, context.linkedServer, context.database, context.schema)
	if context.linkedServer == "" && context.database == "" && context.schema == "" {
		tableEntries.insertCTEs(c)
	}
}

func (c *Completer) insertContextualMSSQLKeywords(entries CompletionMap) {
	before := strings.ToUpper(strings.TrimSpace(c.sql[:min(c.cursorByteOffset, len(c.sql))]))
	insertKeywords := func(keywords ...string) {
		for _, keyword := range keywords {
			entries.Insert(base.Candidate{
				Type: base.CandidateTypeKeyword,
				Text: keyword,
			})
		}
	}

	if hasUnclosedKeywordParen(before, "WITH") {
		insertKeywords(tsqlTableHints...)
	}
	if hasUnclosedKeywordParen(before, "OPTION") {
		insertKeywords(tsqlQueryHints...)
	}
	if strings.HasSuffix(before, "FOR XML") {
		insertKeywords("PATH", "RAW", "AUTO", "EXPLICIT")
	}
	if strings.HasSuffix(before, "FOR JSON") {
		insertKeywords("PATH", "AUTO")
	}
	if strings.HasSuffix(before, " WHEN") || strings.HasSuffix(before, "WHEN") {
		insertKeywords("MATCHED", "NOT")
	}
}

func (c *Completer) insertContextualMSSQLCandidates(columnEntries CompletionMap) {
	if database, schema, table, usedColumns, ok := c.insertColumnListContext(); ok {
		columnEntries.insertMetadataColumnsExcept(c, "", database, schema, table, usedColumns)
	}
	if database, schema, table, ok := c.createIndexColumnListContext(); ok {
		columnEntries.replaceWithMetadataColumns(c, "", database, schema, table)
	}
	if columns := c.createTableForeignKeySourceColumns(); len(columns) > 0 {
		columnEntries.replaceWithLocalColumns(c, columns)
	}
	if database, schema, table, ok := c.referencesColumnListContext(); ok {
		columnEntries.replaceWithMetadataColumns(c, "", database, schema, table)
	}
	if alias, ok := c.qualifiedObjectBeforeCaret(); ok {
		columnEntries.insertDerivedTableColumns(c, alias)
	}
}

func hasUnclosedKeywordParen(before, keyword string) bool {
	start := strings.LastIndex(before, keyword+" (")
	if start < 0 {
		return false
	}
	return strings.LastIndex(before[start:], ")") < 0
}

func (c *Completer) insertColumnListContext() (string, string, string, map[string]bool, bool) {
	before := c.sql[:min(c.cursorByteOffset, len(c.sql))]
	re := regexp.MustCompile(`(?is)\bINSERT\s+INTO\s+([^\s(]+)\s*\(([^()]*)$`)
	matches := re.FindAllStringSubmatch(before, -1)
	if len(matches) == 0 {
		return "", "", "", nil, false
	}
	database, schema, table := parseMultipartIdentifier(matches[len(matches)-1][1])
	if table == "" {
		return "", "", "", nil, false
	}
	excluded := make(map[string]bool)
	for _, column := range splitTopLevelCSV(matches[len(matches)-1][2]) {
		column = strings.TrimSpace(column)
		if column == "" {
			continue
		}
		original, _ := NormalizeTSQLIdentifierText(column)
		excluded[strings.ToLower(original)] = true
	}
	return database, schema, table, excluded, true
}

func (c *Completer) createIndexColumnListContext() (string, string, string, bool) {
	before := c.sql[:min(c.cursorByteOffset, len(c.sql))]
	re := regexp.MustCompile(`(?is)\bCREATE\s+(?:UNIQUE\s+)?(?:(?:CLUSTERED|NONCLUSTERED)\s+)?INDEX\s+[^\s]+\s+ON\s+([^\s(]+)\s*\([^)]*$`)
	matches := re.FindAllStringSubmatch(before, -1)
	if len(matches) == 0 {
		return "", "", "", false
	}
	database, schema, table := parseMultipartIdentifier(matches[len(matches)-1][1])
	return database, schema, table, table != ""
}

func (c *Completer) createTableForeignKeySourceColumns() []string {
	before := c.sql[:min(c.cursorByteOffset, len(c.sql))]
	re := regexp.MustCompile(`(?is)\bCREATE\s+TABLE\s+[^\s(]+\s*\((.*)\bFOREIGN\s+KEY\s*\([^)]*$`)
	matches := re.FindAllStringSubmatch(before, -1)
	if len(matches) == 0 {
		return nil
	}
	var columns []string
	for _, item := range splitTopLevelCSV(matches[len(matches)-1][1]) {
		fields := strings.Fields(strings.TrimSpace(item))
		if len(fields) < 2 {
			continue
		}
		keyword := strings.ToUpper(fields[0])
		if keyword == "CONSTRAINT" || keyword == "PRIMARY" || keyword == "FOREIGN" || keyword == "CHECK" || keyword == "UNIQUE" {
			continue
		}
		original, _ := NormalizeTSQLIdentifierText(fields[0])
		columns = append(columns, original)
	}
	return columns
}

func (c *Completer) referencesColumnListContext() (string, string, string, bool) {
	before := c.sql[:min(c.cursorByteOffset, len(c.sql))]
	re := regexp.MustCompile(`(?is)\bREFERENCES\s+([^\s(]+)\s*\([^)]*$`)
	matches := re.FindAllStringSubmatch(before, -1)
	if len(matches) == 0 {
		return "", "", "", false
	}
	database, schema, table := parseMultipartIdentifier(matches[len(matches)-1][1])
	return database, schema, table, table != ""
}

func (c *Completer) qualifiedObjectBeforeCaret() (string, bool) {
	idx := c.previousTokenIndex()
	if idx < 1 || c.tokenText(idx) != "." {
		return "", false
	}
	original, _ := NormalizeTSQLIdentifierText(c.tokenText(idx - 1))
	if original == "" {
		return "", false
	}
	return unquoteDoubleQuoted(original), true
}

type selectAliasScope struct {
	clause      string
	selectStart int
	blocked     bool
}

func (c *Completer) selectItemAliasStartOffset() (int, bool) {
	scopes := []selectAliasScope{{selectStart: -1}}
	previous := ""
	for idx, token := range c.tokens {
		if token.Loc >= c.cursorByteOffset {
			break
		}
		text := c.tokenText(idx)
		upper := strings.ToUpper(text)
		switch text {
		case "(":
			scopes = append(scopes, selectAliasScope{
				selectStart: -1,
				blocked:     previous == "OVER" || previous == "GROUP",
			})
		case ")":
			if len(scopes) > 1 {
				scopes = scopes[:len(scopes)-1]
			}
		default:
			scope := &scopes[len(scopes)-1]
			if !scope.blocked {
				switch upper {
				case "SELECT":
					scope.clause = "SELECT"
					scope.selectStart = token.Loc
				case "FROM", "WHERE", "GROUP", "HAVING", "ORDER", "ON":
					scope.clause = upper
				case "UNION", "EXCEPT", "INTERSECT":
					scope.clause = ""
					scope.selectStart = -1
				default:
				}
			}
		}
		if upper != "" {
			previous = upper
		}
	}

	scope := scopes[len(scopes)-1]
	if scope.blocked || scope.selectStart < 0 {
		return 0, false
	}
	switch scope.clause {
	case "GROUP", "HAVING", "ORDER":
		return scope.selectStart, true
	default:
		return 0, false
	}
}

func (m CompletionMap) insertDerivedTableColumns(c *Completer, alias string) {
	for _, derived := range extractDerivedTables(c.sql) {
		if !strings.EqualFold(derived.alias, alias) {
			continue
		}
		span, err := GetQuerySpan(
			c.ctx,
			base.GetQuerySpanContext{
				InstanceID:              c.instanceID,
				GetDatabaseMetadataFunc: c.metadataGetter,
				ListDatabaseNamesFunc:   c.databaseNamesLister,
			},
			base.Statement{Text: derived.query},
			c.defaultDatabase,
			c.defaultSchema,
			true,
		)
		if err != nil || span.NotFoundError != nil {
			continue
		}
		for _, column := range span.Results {
			m.Insert(base.Candidate{
				Type: base.CandidateTypeColumn,
				Text: c.quotedIdentifierIfNeeded(column.Name),
			})
		}
	}
}

func (c *Completer) previousTokenIndex() int {
	idx := c.caretTokenIndex - 1
	for idx >= 0 && c.tokens[idx].End > c.cursorByteOffset {
		idx--
	}
	return idx
}

func (c *Completer) tokenText(idx int) string {
	if idx < 0 || idx >= len(c.tokens) {
		return ""
	}
	token := c.tokens[idx]
	if token.Loc < 0 || token.End > len(c.sql) || token.Loc > token.End {
		return ""
	}
	return c.sql[token.Loc:token.End]
}

func parseMultipartIdentifier(text string) (string, string, string) {
	parts := splitMultipartIdentifier(text)
	for i, part := range parts {
		original, _ := NormalizeTSQLIdentifierText(unquoteDoubleQuoted(strings.TrimSpace(part)))
		parts[i] = original
	}
	switch len(parts) {
	case 0:
		return "", "", ""
	case 1:
		return "", "", parts[0]
	case 2:
		return "", parts[0], parts[1]
	default:
		return parts[len(parts)-3], parts[len(parts)-2], parts[len(parts)-1]
	}
}

func splitMultipartIdentifier(text string) []string {
	var parts []string
	var buf strings.Builder
	var bracket, quote bool
	for _, r := range text {
		switch r {
		case '[':
			bracket = true
		case ']':
			bracket = false
		case '"':
			quote = !quote
		case '.':
			if !bracket && !quote {
				parts = append(parts, buf.String())
				buf.Reset()
				continue
			}
		default:
		}
		buf.WriteRune(r)
	}
	if buf.Len() > 0 {
		parts = append(parts, buf.String())
	}
	return parts
}

func splitTopLevelCSV(text string) []string {
	var parts []string
	var buf strings.Builder
	var bracket, quote bool
	level := 0
	for _, r := range text {
		switch r {
		case '[':
			bracket = true
		case ']':
			bracket = false
		case '"':
			quote = !quote
		case '(':
			if !bracket && !quote {
				level++
			}
		case ')':
			if !bracket && !quote && level > 0 {
				level--
			}
		case ',':
			if !bracket && !quote && level == 0 {
				parts = append(parts, buf.String())
				buf.Reset()
				continue
			}
		default:
		}
		buf.WriteRune(r)
	}
	parts = append(parts, buf.String())
	return parts
}

type derivedTable struct {
	query string
	alias string
}

func extractDerivedTables(sql string) []derivedTable {
	var result []derivedTable
	upperSQL := strings.ToUpper(sql)
	for idx := 0; idx < len(sql); idx++ {
		if sql[idx] != '(' {
			continue
		}
		if !isDerivedTableOpen(upperSQL, idx) {
			continue
		}
		end := matchingParen(sql, idx)
		if end < 0 {
			continue
		}
		alias, ok := readAliasAfter(sql, end+1)
		if ok {
			result = append(result, derivedTable{
				query: sql[idx+1 : end],
				alias: alias,
			})
		}
		idx = end
	}
	return result
}

func isDerivedTableOpen(upperSQL string, idx int) bool {
	after := strings.TrimLeft(upperSQL[idx+1:], asciiWhitespace)
	if !strings.HasPrefix(after, "SELECT") && !strings.HasPrefix(after, "WITH") {
		return false
	}
	before := strings.TrimRight(upperSQL[:idx], asciiWhitespace)
	for _, keyword := range []string{"FROM", "JOIN", "APPLY"} {
		if strings.HasSuffix(before, keyword) {
			return true
		}
	}
	return false
}

func matchingParen(sql string, open int) int {
	level := 0
	var bracket, doubleQuote, singleQuote bool
	for i := open; i < len(sql); i++ {
		switch sql[i] {
		case '\'':
			if !bracket && !doubleQuote {
				singleQuote = !singleQuote
			}
		case '"':
			if !bracket && !singleQuote {
				doubleQuote = !doubleQuote
			}
		case '[':
			if !singleQuote && !doubleQuote {
				bracket = true
			}
		case ']':
			if bracket {
				bracket = false
			}
		case '(':
			if !bracket && !singleQuote && !doubleQuote {
				level++
			}
		case ')':
			if !bracket && !singleQuote && !doubleQuote {
				level--
				if level == 0 {
					return i
				}
			}
		default:
		}
	}
	return -1
}

func readAliasAfter(sql string, start int) (string, bool) {
	rest := strings.TrimLeft(sql[start:], asciiWhitespace)
	upperRest := strings.ToUpper(rest)
	if strings.HasPrefix(upperRest, "AS ") {
		rest = strings.TrimLeft(rest[2:], asciiWhitespace)
	}
	if rest == "" {
		return "", false
	}
	var alias string
	switch rest[0] {
	case '[':
		end := strings.IndexByte(rest, ']')
		if end <= 0 {
			return "", false
		}
		alias = rest[1:end]
	case '"':
		end := strings.IndexByte(rest[1:], '"')
		if end < 0 {
			return "", false
		}
		alias = rest[1 : end+1]
	default:
		end := 0
		for end < len(rest) {
			r, size := utf8.DecodeRuneInString(rest[end:])
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '@' && r != '#' && r != '$' {
				break
			}
			end += size
		}
		if end == 0 {
			return "", false
		}
		alias = rest[:end]
	}
	return alias, true
}

func unquoteDoubleQuoted(identifier string) string {
	if len(identifier) >= 2 && identifier[0] == '"' && identifier[len(identifier)-1] == '"' {
		return identifier[1 : len(identifier)-1]
	}
	return identifier
}

type objectFlag int

const (
	objectFlagShowLinkedServer objectFlag = 1 << iota
	objectFlagShowDatabase
	objectFlagShowSchema
	objectFlagShowObject
	objectFlagShowColumn
)

type objectRefContextOption func(*objectRefContext)

func withColumn() objectRefContextOption {
	return func(c *objectRefContext) {
		c.column = ""
		c.flags |= objectFlagShowColumn
	}
}

func withLinkedServer() objectRefContextOption {
	return func(c *objectRefContext) {
		c.linkedServer = ""
		c.flags |= objectFlagShowLinkedServer
	}
}

func newObjectRefContext(options ...objectRefContextOption) *objectRefContext {
	o := &objectRefContext{
		flags: objectFlagShowDatabase | objectFlagShowSchema | objectFlagShowObject,
	}
	for _, option := range options {
		option(o)
	}
	return o
}

// objectRefContext provides precise completion context about the object reference,
// check the flags and the fields to determine what kind of object should be included in the completion list.
// Caller should call the newObjectRefContext to create a new objectRefContext, and modify it based on function it provides.
type objectRefContext struct {
	linkedServer string
	database     string
	schema       string
	object       string

	// column is optional considering field, for example, it should be not applicable for full table name rule.
	column string

	flags objectFlag
}

func (o *objectRefContext) empty() bool {
	return o.linkedServer == "" && o.database == "" && o.schema == "" && o.object == "" && o.column == ""
}

func (o *objectRefContext) clone() *objectRefContext {
	return &objectRefContext{
		linkedServer: o.linkedServer,
		database:     o.database,
		schema:       o.schema,
		object:       o.object,
		column:       o.column,
		flags:        o.flags,
	}
}

func (o *objectRefContext) setLinkedServer(linkedServer string) *objectRefContext {
	o.linkedServer = linkedServer
	o.flags &= ^objectFlagShowLinkedServer
	return o
}

func (o *objectRefContext) setDatabase(database string) *objectRefContext {
	o.database = database
	o.flags &= ^objectFlagShowDatabase
	return o
}

func (o *objectRefContext) setSchema(schema string) *objectRefContext {
	o.schema = schema
	o.flags &= ^objectFlagShowSchema
	return o
}

func (o *objectRefContext) setObject(object string) *objectRefContext {
	o.object = object
	o.flags &= ^objectFlagShowObject
	return o
}

func (o *objectRefContext) setColumn(column string) *objectRefContext {
	o.column = column
	o.flags &= ^objectFlagShowColumn
	return o
}

func (c *Completer) determineFullColumnName() []*objectRefContext {
	tokenIndex := c.scanner.GetIndex()
	if c.scanner.GetTokenChannel() != antlr.TokenDefaultChannel {
		// Skip to the next non-hidden token.
		c.scanner.Forward(true /* skipHidden */)
	}

	tokenType := c.scanner.GetTokenType()
	if c.scanner.GetTokenText() != "." && !c.lexer.IsID_(tokenType) && c.scanner.GetTokenText() != "DELETED" &&
		c.scanner.GetTokenText() != "INSERTED" && c.scanner.GetTokenText() != "$" &&
		c.scanner.GetTokenText() != "IDENTITY" && c.scanner.GetTokenText() != "ROWGUID" {
		c.scanner.Backward(true /* skipHidden */)
	}

	if tokenIndex > 0 {
		// Go backward until we hit a non-identifier token.
		for {
			curID := c.lexer.IsID_(c.scanner.GetTokenType()) && c.scanner.GetPreviousTokenText(false /* skipHidden */) == "."
			curDOT := c.scanner.GetTokenText() == "." && (c.lexer.IsID_(c.scanner.GetPreviousTokenType(false /* skipHidden */)) || c.scanner.GetPreviousTokenText(false /* skipHidden */) == "DELETED" || c.scanner.GetPreviousTokenText(false /* skipHidden */) == "INSERTED")
			curRowguid := c.scanner.GetTokenText() == "ROWGUID" && c.scanner.GetPreviousTokenText(false /* skipHidden */) == "$"
			curIdentity := c.scanner.GetTokenText() == "IDENTITY" && c.scanner.GetPreviousTokenText(false /* skipHidden */) == "$"
			if curID || curDOT || curRowguid || curIdentity {
				c.scanner.Backward(true /* skipHidden */)
				continue
			}
			break
		}
	}

	// The c.scanner is now on the leading identifier (or dot?) if there's no leading id.
	var candidates []string
	var temp string
	var count int
	for {
		count++
		if c.scanner.GetTokenText() == "DELETED" || c.scanner.GetTokenText() == "INSERTED" {
			candidates = append(candidates, "", "", "", "")
			count += 3
			if !c.scanner.IsTokenType(tsqlparser.TSqlParserDOT) || tokenIndex <= c.scanner.GetIndex() {
				return deriveObjectRefContextsFromCandidates(candidates, false /* ignoredLinkedServer */, true /* includeColumn */)
			}
		} else if c.lexer.IsID_(c.scanner.GetTokenType()) {
			temp, _ = NormalizeTSQLIdentifierText(c.scanner.GetTokenText())
			c.scanner.Forward(true /* skipHidden */)
			if !c.scanner.IsTokenType(tsqlparser.TSqlParserDOT) || tokenIndex <= c.scanner.GetIndex() {
				return deriveObjectRefContextsFromCandidates(candidates, false /* ignoredLinkedServer */, true /* includeColumn */)
			}
			candidates = append(candidates, temp)
		}
		c.scanner.Forward(true /* skipHidden */)
		if count > 4 {
			break
		}
	}

	return deriveObjectRefContextsFromCandidates(candidates, false /* ignoredLinkedServer */, true /* includeColumn */)
}

func (c *Completer) determineAsteriskContext() []*objectRefContext {
	tokenIndex := c.scanner.GetIndex()
	if c.scanner.GetTokenChannel() != antlr.TokenDefaultChannel {
		// Skip to the next non-hidden token.
		c.scanner.Forward(true /* skipHidden */)
	}

	tokenType := c.scanner.GetTokenType()
	if c.scanner.GetTokenText() != "." && !c.lexer.IsID_(tokenType) && c.scanner.GetTokenText() != "*" {
		// We are at the end of an incomplete identifier spec. Jump back.
		// For example, SELECT * FROM db.| WHERE a = 1, the scanner will be seek to the token ' ', and
		// forwards to WHERE because we skip to the next non-hidden token in the above code.
		// Also, for SELECT * FROM |, the scanner will be backward to the token 'FROM'.
		c.scanner.Backward(true /* skipHidden */)
	}

	if tokenIndex > 0 {
		// Go backward until we hit a non-identifier token.
		var count int
		for {
			var curAsterisk bool
			if count == 0 {
				if c.scanner.GetTokenText() == "*" && c.scanner.GetPreviousTokenText(false /* skipHidden */) == "." {
					curAsterisk = true
				}
			}
			count++
			curID := c.lexer.IsID_(c.scanner.GetTokenType()) && c.scanner.GetPreviousTokenText(false /* skipHidden */) == "."
			curDOT := c.scanner.GetTokenText() == "." && c.lexer.IsID_(c.scanner.GetPreviousTokenType(false /* skipHidden */))
			if curID || curDOT || curAsterisk {
				c.scanner.Backward(true /* skipHidden */)
				continue
			}
			break
		}
	}

	// The c.scanner is now on the leading identifier (or dot?) if there's no leading id.
	var candidates []string
	var temp string
	var count int
	for {
		count++
		if c.lexer.IsID_(c.scanner.GetTokenType()) {
			temp, _ = NormalizeTSQLIdentifierText(c.scanner.GetTokenText())
			c.scanner.Forward(true /* skipHidden */)
		}
		if !c.scanner.IsTokenType(tsqlparser.TSqlParserDOT) || tokenIndex <= c.scanner.GetIndex() {
			return deriveObjectRefContextsFromCandidates(candidates, true /* ignoredLinkedServer */, false /* includeColumn */)
		}
		candidates = append(candidates, temp)
		c.scanner.Forward(true /* skipHidden */)
		if count > 2 {
			break
		}
	}

	return deriveObjectRefContextsFromCandidates(candidates, true /* ignoredLinkedServer */, false /* includeColumn */)
}

func (c *Completer) determineFullTableNameContext() []*objectRefContext {
	tokenIndex := c.scanner.GetIndex()
	if c.scanner.GetTokenChannel() != antlr.TokenDefaultChannel {
		// Skip to the next non-hidden token.
		c.scanner.Forward(true /* skipHidden */)
	}

	tokenType := c.scanner.GetTokenType()
	if c.scanner.GetTokenText() != "." && !c.lexer.IsID_(tokenType) {
		// We are at the end of an incomplete identifier spec. Jump back.
		// For example, SELECT * FROM db.| WHERE a = 1, the scanner will be seek to the token ' ', and
		// forwards to WHERE because we skip to the next non-hidden token in the above code.
		// Also, for SELECT * FROM |, the scanner will be backward to the token 'FROM'.
		c.scanner.Backward(true /* skipHidden */)
	}

	if tokenIndex > 0 {
		// Go backward until we hit a non-identifier token.
		for {
			curID := c.lexer.IsID_(c.scanner.GetTokenType()) && c.scanner.GetPreviousTokenText(false /* skipHidden */) == "."
			curDOT := c.scanner.GetTokenText() == "." && c.lexer.IsID_(c.scanner.GetPreviousTokenType(false /* skipHidden */))
			if curID || curDOT {
				c.scanner.Backward(true /* skipHidden */)
				continue
			}
			break
		}
	}

	// The c.scanner is now on the leading identifier (or dot?) if there's no leading id.
	var candidates []string
	var temp string
	var count int
	for {
		count++
		if c.lexer.IsID_(c.scanner.GetTokenType()) {
			temp, _ = NormalizeTSQLIdentifierText(c.scanner.GetTokenText())
			c.scanner.Forward(true /* skipHidden */)
		}
		if !c.scanner.IsTokenType(tsqlparser.TSqlParserDOT) || tokenIndex <= c.scanner.GetIndex() {
			return deriveObjectRefContextsFromCandidates(candidates, false /* ignoredLinkedServer */, false /* includeColumn */)
		}
		candidates = append(candidates, temp)
		c.scanner.Forward(true /* skipHidden */)
		if count > 3 {
			break
		}
	}

	return deriveObjectRefContextsFromCandidates(candidates, false /* ignoredLinkedServer */, false /* includeColumn */)
}

// deriveObjectRefContextsFromCandidates derives the object reference contexts from the candidates.
// The T-SQL grammar's object reference likes [linked_server_name.][database_name.][schema_name.][object_name]
// The size of candidates is the window size in the object reference,
// for example, if the candidates are ["a", "b", "c"], the size is 3,
// and objectRefContext would be [linked_server_name: "a", database_name: "b", schema_name: "c", object_name: ""] or[linked_server_name: "", database_name: "a", schema_name: "b", object_name: "c"].
func deriveObjectRefContextsFromCandidates(candidates []string, ignoredLinkedServer bool, includeColumn bool) []*objectRefContext {
	var options []objectRefContextOption
	if !ignoredLinkedServer {
		options = append(options, withLinkedServer())
	}
	if includeColumn {
		options = append(options, withColumn())
	}
	refCtx := newObjectRefContext(options...)
	if len(candidates) == 0 {
		return []*objectRefContext{
			refCtx.clone(),
		}
	}

	var results []*objectRefContext
	switch len(candidates) {
	case 1:
		if !ignoredLinkedServer {
			results = append(results, refCtx.clone().setLinkedServer(candidates[0]))
		}
		results = append(
			results,
			refCtx.clone().setLinkedServer("").setDatabase(candidates[0]),
			refCtx.clone().setLinkedServer("").setDatabase("").setSchema(candidates[0]),
			refCtx.clone().setLinkedServer("").setDatabase("").setSchema("").setObject(candidates[0]),
		)
		if includeColumn {
			results = append(results, refCtx.clone().setLinkedServer("").setDatabase("").setSchema("").setObject("").setColumn(candidates[0]))
		}
	case 2:
		if !ignoredLinkedServer {
			results = append(results, refCtx.clone().setLinkedServer(candidates[0]).setDatabase(candidates[1]))
		}
		results = append(
			results,
			refCtx.clone().setLinkedServer("").setDatabase(candidates[0]).setSchema(candidates[1]),
			refCtx.clone().setLinkedServer("").setDatabase("").setSchema(candidates[0]).setObject(candidates[1]),
		)
		if includeColumn {
			results = append(results, refCtx.clone().setLinkedServer("").setDatabase("").setSchema(candidates[0]).setObject("").setColumn(candidates[1]))
		}
	case 3:
		if !ignoredLinkedServer {
			results = append(results, refCtx.clone().setLinkedServer(candidates[0]).setDatabase(candidates[1]).setSchema(candidates[2]))
		}
		results = append(
			results,
			refCtx.clone().setLinkedServer("").setDatabase(candidates[0]).setSchema(candidates[1]).setObject(candidates[2]),
		)
		if includeColumn {
			results = append(results, refCtx.clone().setLinkedServer("").setDatabase(candidates[0]).setSchema(candidates[1]).setObject("").setColumn(candidates[2]))
		}
	case 4:
		if !ignoredLinkedServer {
			results = append(results, refCtx.clone().setLinkedServer(candidates[0]).setDatabase(candidates[1]).setSchema(candidates[2]).setObject(candidates[3]))
		}
		if includeColumn {
			results = append(results, refCtx.clone().setLinkedServer("").setDatabase(candidates[0]).setSchema(candidates[1]).setObject(candidates[2]).setColumn(candidates[3]))
		}
	case 5:
		if includeColumn {
			results = append(results, refCtx.clone().setLinkedServer(candidates[0]).setDatabase(candidates[1]).setSchema(candidates[2]).setObject(candidates[3]).setColumn(candidates[4]))
		}
	default:
		// Other cases
	}

	if len(results) == 0 {
		results = append(results, refCtx.clone())
	}

	return results
}

// skipHeadingSQLs skips the SQL statements which before the caret position.
// caretLine is 1-based and caretOffset is 0-based.
func skipHeadingSQLs(statement string, caretLine int, caretOffset int) (string, int, int) {
	list, err := SplitSQL(statement)
	if err != nil {
		return statement, caretLine, caretOffset
	}
	if !hasMultipleNonEmptyStatements(list) {
		return statement, caretLine, caretOffset
	}

	caretLineZeroBased := caretLine - 1
	start := statementIndexAtCaret(list, caretLineZeroBased, caretOffset)
	if start == 0 {
		return statement, caretLine, caretOffset
	}

	newCaretLine, newCaretOffset := rebaseCaretAfterSkippedStatement(list[start-1], caretLineZeroBased, caretOffset)
	return joinStatementTexts(list[start:]), newCaretLine, newCaretOffset
}

func statementIndexAtCaret(list []base.Statement, caretLine int, caretOffset int) int {
	for i, statement := range list {
		endLine := int(statement.End.GetLine()) - 1
		endColumn := int(statement.End.GetColumn())
		if endLine > caretLine || (endLine == caretLine && endColumn >= caretOffset) {
			return i
		}
	}
	return 0
}

func rebaseCaretAfterSkippedStatement(previous base.Statement, caretLine int, caretOffset int) (int, int) {
	previousEndLine := int(previous.End.GetLine()) - 1
	previousEndColumn := int(previous.End.GetColumn())
	newCaretLine := caretLine - previousEndLine + 1
	newCaretOffset := caretOffset
	if caretLine == previousEndLine {
		newCaretOffset = caretOffset - previousEndColumn + 1
	}
	return newCaretLine, newCaretOffset
}

func joinStatementTexts(list []base.Statement) string {
	parts := make([]string, 0, len(list))
	for _, statement := range list {
		parts = append(parts, statement.Text)
	}
	return strings.Join(parts, "")
}

func hasMultipleNonEmptyStatements(statements []base.Statement) bool {
	seenNonEmpty := false
	for _, statement := range statements {
		if statement.Empty {
			continue
		}
		if seenNonEmpty {
			return true
		}
		seenNonEmpty = true
	}
	return false
}

func (c *Completer) takeReferencesSnapshot() {
	c.ensureReferenceMap()
	for _, references := range c.referencesStack {
		c.rememberReferences(references)
	}
}

func (c *Completer) ensureReferenceMap() {
	if c.referenceMap == nil {
		c.referenceMap = make(map[string]bool)
	}
}

func (c *Completer) rememberReferences(references []base.TableReference) {
	c.references = append(c.references, references...)
	for _, reference := range references {
		physical, ok := reference.(*base.PhysicalTableReference)
		if !ok {
			continue
		}
		c.referenceMap[c.physicalReferenceKey(physical)] = true
	}
}

func (c *Completer) physicalReferenceKey(reference *base.PhysicalTableReference) string {
	database := reference.Database
	if database == "" {
		database = c.defaultDatabase
	}
	schema := reference.Schema
	if schema == "" {
		schema = c.defaultSchema
	}
	return fmt.Sprintf("%s.%s.%s", database, schema, reference.Table)
}

func (c *Completer) collectLeadingTableReferences(caretIndex int) {
	c.scanner.Push()

	c.scanner.SeekIndex(0)

	level := 0
	for {
		found := c.scanner.GetTokenType() == tsqlparser.TSqlLexerFROM
		for !found {
			if !c.scanner.Forward(false) || c.scanner.GetIndex() >= caretIndex {
				break
			}

			switch c.scanner.GetTokenType() {
			case tsqlparser.TSqlLexerLR_BRACKET:
				level++
				c.referencesStack = append([][]base.TableReference{{}}, c.referencesStack...)
			case tsqlparser.TSqlLexerRR_BRACKET:
				if level == 0 {
					c.scanner.PopAndRestore()
					return // We cannot go above the initial nesting level.
				}
			case tsqlparser.TSqlLexerFROM:
				found = true
			default:
				// Other tokens, continue scanning
			}
		}
		if !found {
			c.scanner.PopAndRestore()
			return // No FROM clause found.
		}
		c.parseTableReferences(c.scanner.GetFollowingText())
		if c.scanner.GetTokenType() == tsqlparser.TSqlLexerFROM {
			c.scanner.Forward(false /* skipHidden */)
		}
	}
}

func (c *Completer) collectRemainingTableReferences() {
	c.scanner.Push()

	level := 0
	for {
		found := c.scanner.GetTokenType() == tsqlparser.TSqlLexerFROM
		for !found {
			if !c.scanner.Forward(false /* skipHidden */) {
				break
			}

			switch c.scanner.GetTokenType() {
			case tsqlparser.TSqlLexerLR_BRACKET:
				level++
				c.referencesStack = append([][]base.TableReference{{}}, c.referencesStack...)
			case tsqlparser.TSqlLexerRR_BRACKET:
				if level > 0 {
					level--
				}

			case tsqlparser.TSqlLexerFROM:
				if level == 0 {
					found = true
				}
			default:
				// Other tokens, continue scanning
			}
		}

		if !found {
			c.scanner.PopAndRestore()
			return // No more FROM clause found.
		}

		c.parseTableReferences(c.scanner.GetFollowingText())
		if c.scanner.GetTokenType() == tsqlparser.TSqlLexerFROM {
			c.scanner.Forward(false /* skipHidden */)
		}
	}
}

func (c *Completer) parseTableReferences(fromClause string) {
	input := antlr.NewInputStream(fromClause)
	lexer := tsqlparser.NewTSqlLexer(input)
	tokens := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := tsqlparser.NewTSqlParser(tokens)

	parser.BuildParseTrees = true
	parser.RemoveErrorListeners()
	tree := parser.From_table_sources()
	listener := &tableRefListener{
		context:        c,
		fromClauseMode: true,
	}
	antlr.ParseTreeWalkerDefault.Walk(listener, tree)
}

type tableRefListener struct {
	*tsqlparser.BaseTSqlParserListener

	context        *Completer
	fromClauseMode bool
	done           bool
	level          int
}

func (l *tableRefListener) ExitAs_table_alias(ctx *tsqlparser.As_table_aliasContext) {
	if l.done {
		return
	}
	if l.level == 0 && len(l.context.referencesStack) != 0 && len(l.context.referencesStack[0]) != 0 {
		if physicalTable, ok := l.context.referencesStack[0][len(l.context.referencesStack[0])-1].(*base.PhysicalTableReference); ok {
			physicalTable.Alias = unquote(ctx.Table_alias().GetText())
			return
		}
		if virtualTable, ok := l.context.referencesStack[0][len(l.context.referencesStack[0])-1].(*base.VirtualTableReference); ok {
			virtualTable.Table = unquote(ctx.Table_alias().GetText())
		}
	}
}

func (l *tableRefListener) ExitColumn_alias_list(ctx *tsqlparser.Column_alias_listContext) {
	if l.done {
		return
	}

	if l.level == 0 && len(l.context.referencesStack) != 0 && len(l.context.referencesStack[0]) != 0 {
		if virtualTable, ok := l.context.referencesStack[0][len(l.context.referencesStack[0])-1].(*base.VirtualTableReference); ok {
			var newColumns []string
			for _, column := range ctx.AllColumn_alias() {
				newColumns = append(newColumns, unquote(column.GetText()))
			}
			virtualTable.Columns = newColumns
		}
	}
}

func (l *tableRefListener) ExitFull_table_name(ctx *tsqlparser.Full_table_nameContext) {
	if l.done {
		return
	}

	if !l.fromClauseMode || l.level == 0 {
		reference := &base.PhysicalTableReference{}
		_ /* Linked Server */, database, schema, table := normalizeFullTableNameFallback(ctx, "", "")
		reference.Database = database
		reference.Schema = schema
		reference.Table = table
		l.context.referencesStack[0] = append(l.context.referencesStack[0], reference)
	}
}

func (l *tableRefListener) ExitRowset_function(*tsqlparser.Rowset_functionContext) {
	if l.done {
		return
	}

	if !l.fromClauseMode || l.level == 0 {
		reference := &base.VirtualTableReference{}

		l.context.referencesStack[0] = append(l.context.referencesStack[0], reference)
	}
}

func (l *tableRefListener) ExitDerivedTable(ctx *tsqlparser.Derived_tableContext) {
	if l.done {
		return
	}

	pCtx, ok := ctx.GetParent().(*tsqlparser.Table_source_itemContext)
	if !ok {
		return
	}

	derivedTableName := unquote(pCtx.As_table_alias().Table_alias().GetText())
	reference := &base.VirtualTableReference{
		Table: derivedTableName,
	}

	if pCtx.Column_alias_list() == nil {
		// User do not specify the column alias, we should use query span to get the column alias.
		if span, err := GetQuerySpan(
			l.context.ctx,
			base.GetQuerySpanContext{
				InstanceID:              l.context.instanceID,
				GetDatabaseMetadataFunc: l.context.metadataGetter,
				ListDatabaseNamesFunc:   l.context.databaseNamesLister,
			},
			base.Statement{Text: fmt.Sprintf("SELECT * FROM (%s);", ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx))},
			l.context.defaultDatabase,
			l.context.defaultSchema,
			true,
		); err == nil && span.NotFoundError == nil {
			for _, column := range span.Results {
				reference.Columns = append(reference.Columns, column.Name)
			}
		}
	}
	l.context.referencesStack[0] = append(l.context.referencesStack[0], reference)
}

func (l *tableRefListener) ExitChange_table(*tsqlparser.Change_tableContext) {
	if l.done {
		return
	}

	if !l.fromClauseMode || l.level == 0 {
		reference := &base.VirtualTableReference{}
		l.context.referencesStack[0] = append(l.context.referencesStack[0], reference)
	}
}

func (l *tableRefListener) ExitNodes_method(*tsqlparser.Nodes_methodContext) {
	if l.done {
		return
	}

	if !l.fromClauseMode || l.level == 0 {
		reference := &base.VirtualTableReference{}
		l.context.referencesStack[0] = append(l.context.referencesStack[0], reference)
	}
}

func (l *tableRefListener) EnterTable_source_item(ctx *tsqlparser.Table_source_itemContext) {
	if l.done {
		return
	}

	if !l.fromClauseMode || l.level == 0 {
		if ctx.GetLoc_id() != nil {
			reference := &base.VirtualTableReference{}
			l.context.referencesStack[0] = append(l.context.referencesStack[0], reference)
		} else if ctx.GetLoc_id_call() != nil {
			reference := &base.VirtualTableReference{}
			l.context.referencesStack[0] = append(l.context.referencesStack[0], reference)
		} else if ctx.GetOldstyle_fcall() != nil {
			reference := &base.VirtualTableReference{}
			l.context.referencesStack[0] = append(l.context.referencesStack[0], reference)
		}
	}
}

func (l *tableRefListener) EnterSubquery(*tsqlparser.SubqueryContext) {
	if l.done {
		return
	}

	if l.fromClauseMode {
		l.level++
	} else {
		l.context.referencesStack = append([][]base.TableReference{{}}, l.context.referencesStack...)
	}
}

func (l *tableRefListener) ExitSubquery(*tsqlparser.SubqueryContext) {
	if l.done {
		return
	}

	if l.fromClauseMode {
		l.level--
	} else {
		l.context.referencesStack = l.context.referencesStack[1:]
	}
}

func (c *Completer) fetchCommonTableExpression(statement string) {
	c.cteTables = nil

	// SQL Server only allows CTEs in the first level, the following statement is invalid:
	// SELECT * FROM (WITH t AS (SELECT * FROM [Employees]) SELECT * FROM t) t2;
	// https://stackoverflow.com/questions/1914151/how-we-can-use-cte-in-subquery-in-sql-server
	// So it's easy for SQL server to find the CTEs than other engines, we only need to construct a listener to find the CTEs.
	extractor := &cteExtractor{
		completer: c,
	}
	input := antlr.NewInputStream(statement)
	lexer := tsqlparser.NewTSqlLexer(input)
	tokens := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := tsqlparser.NewTSqlParser(tokens)
	parser.BuildParseTrees = true
	parser.RemoveErrorListeners()
	tree := parser.Tsql_file()
	antlr.ParseTreeWalkerDefault.Walk(extractor, tree)
	c.cteTables = extractor.virtualReferences
}

type cteExtractor struct {
	*tsqlparser.BaseTSqlParserListener

	completer         *Completer
	handled           bool
	virtualReferences []*base.VirtualTableReference
}

func (c *cteExtractor) EnterWith_expression(ctx *tsqlparser.With_expressionContext) {
	if c.handled {
		return
	}
	c.handled = true

	for _, cte := range ctx.AllCommon_table_expression() {
		cteName := ctx.GetParser().GetTokenStream().GetTextFromRuleContext(cte.GetExpression_name())
		if cteName == "" {
			continue
		}
		if cte.GetColumns() != nil {
			var columns []string
			for _, columnID := range cte.GetColumns().AllId_() {
				columns = append(columns, unquote(columnID.GetText()))
			}
			c.virtualReferences = append(c.virtualReferences, &base.VirtualTableReference{
				Table:   unquote(cteName),
				Columns: columns,
			})
			continue
		}

		cteBody := ctx.GetParser().GetTokenStream().GetTextFromInterval(
			antlr.Interval{
				Start: ctx.AllCommon_table_expression()[0].GetStart().GetTokenIndex(),
				Stop:  cte.GetStop().GetTokenIndex(),
			},
		)

		statement := fmt.Sprintf("WITH %s SELECT * FROM %s", cteBody, cteName)
		if span, err := GetQuerySpan(
			c.completer.ctx,
			base.GetQuerySpanContext{
				InstanceID:              c.completer.instanceID,
				GetDatabaseMetadataFunc: c.completer.metadataGetter,
				ListDatabaseNamesFunc:   c.completer.databaseNamesLister,
			},
			base.Statement{Text: statement},
			c.completer.defaultDatabase,
			c.completer.defaultSchema,
			true,
		); err == nil && span.NotFoundError == nil {
			var columns []string
			for _, column := range span.Results {
				columns = append(columns, column.Name)
			}
			c.virtualReferences = append(c.virtualReferences, &base.VirtualTableReference{
				Table:   unquote(cteName),
				Columns: columns,
			})
		}
	}
}

func (c *Completer) fetchSelectItemAliases(aliasPositions []int, startOffset int) []string {
	aliasMap := make(map[string]bool)
	for _, pos := range aliasPositions {
		if pos < startOffset || pos >= c.cursorByteOffset {
			continue
		}
		if aliasText := c.extractAliasText(pos); aliasText != "" {
			aliasMap[aliasText] = true
		}
	}

	var result []string
	for alias := range aliasMap {
		result = append(result, alias)
	}
	slices.Sort(result)
	return result
}

func (c *Completer) extractAliasText(pos int) string {
	if pos < 0 || pos >= len(c.sql) {
		return ""
	}
	followingText := c.sql[pos:]
	if len(followingText) == 0 {
		return ""
	}

	input := antlr.NewInputStream(followingText)
	lexer := tsqlparser.NewTSqlLexer(input)
	tokens := antlr.NewCommonTokenStream(lexer, 0)
	parser := tsqlparser.NewTSqlParser(tokens)

	parser.BuildParseTrees = true
	parser.RemoveErrorListeners()
	tree := parser.As_column_alias()

	listener := &SelectAliasListener{}
	antlr.ParseTreeWalkerDefault.Walk(listener, tree)

	return listener.result
}

type SelectAliasListener struct {
	*tsqlparser.BaseTSqlParserListener

	result string
}

func (l *SelectAliasListener) EnterAs_column_alias(ctx *tsqlparser.As_column_aliasContext) {
	l.result = unquote(ctx.Column_alias().GetText())
}

func (c *Completer) quotedIdentifierIfNeeded(identifier string) string {
	if c.caretTokenIsQuoted != quotedTypeNone {
		return identifier
	}
	if !isRegularIdentifier(identifier) {
		return fmt.Sprintf("[%s]", identifier)
	}
	return identifier
}

func isRegularIdentifier(identifier string) bool {
	// https://learn.microsoft.com/en-us/sql/relational-databases/databases/database-identifiers?view=sql-server-ver16#rules-for-regular-identifiers
	if len(identifier) == 0 {
		return true
	}

	firstChar := rune(identifier[0])
	isFirstCharValid := unicode.IsLetter(firstChar) || firstChar == '_' || firstChar == '@' || firstChar == '#'
	if !isFirstCharValid {
		return false
	}

	for _, r := range identifier[1:] {
		isValidChar := unicode.IsLetter(r) || unicode.IsDigit(r) || r == '@' || r == '$' || r == '#' || r == '_'
		if !isValidChar {
			return false
		}
	}

	// Rule 3: Check if the identifier is a reserved word
	// (You would need to maintain a list of reserved words for this)
	if IsTSQLReservedKeyword(identifier, false) {
		return false
	}

	// Rule 4: Check for embedded spaces or special characters
	for _, r := range identifier {
		if r == ' ' || !unicode.IsPrint(r) {
			return false
		}
	}

	// Rule 5: Check for supplementary characters
	return utf8.ValidString(identifier)
}
