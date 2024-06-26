package tsql

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/antlr4-go/antlr/v4"
	tsqlparser "github.com/bytebase/tsql-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterCompleteFunc(storepb.Engine_MSSQL, Completion)
}

var (

	// Check tableRefListener is implementing the TSqlParserListener interface.
	_ tsqlparser.TSqlParserListener = &tableRefListener{}
	_ tsqlparser.TSqlParserListener = &cteExtractor{}

	globalFellowSetsByState = base.NewFollowSetsByState()
	ignoredTokens           = map[int]bool{
		// Common EOF
		tsqlparser.TSqlParserEOF: true,

		// Token with EBNF symbol
		tsqlparser.TSqlParserBACKSLASH:            true,
		tsqlparser.TSqlParserCONVERT:              true, // 'TRY_'? 'CONVERT'
		tsqlparser.TSqlParserDEFAULT_DOUBLE_QUOTE: true, // ["]'DEFAULT'["]
		tsqlparser.TSqlParserDOUBLE_BACK_SLASH:    true, // '\\\\'
		tsqlparser.TSqlParserDOUBLE_FORWARD_SLASH: true, // '//'
		tsqlparser.TSqlParserEXECUTE:              true, // 'EXE CUTE?' // TODO(zp): Find a way to improve this because it is a common keyword.
		tsqlparser.TSqlParserNULL_DOUBLE_QUOTE:    true, // ["]'NULL'["]
		tsqlparser.TSqlParserPARSE:                true, // 'TRY_'? 'PARSE'

		// Abbreviation
		tsqlparser.TSqlParserYEAR_ABBR:        true, // 'yy' | 'yyyy'
		tsqlparser.TSqlParserQUARTER_ABBR:     true, // 'qq' | 'q'
		tsqlparser.TSqlParserMONTH_ABBR:       true, // 'mm' | 'm'
		tsqlparser.TSqlParserDAYOFYEAR_ABBR:   true, // 'dy' | 'y'
		tsqlparser.TSqlParserWEEK_ABBR:        true, // 'wk' | 'ww'
		tsqlparser.TSqlParserDAY_ABBR:         true, // 'dd' | 'd'
		tsqlparser.TSqlParserHOUR_ABBR:        true, // 'hh'
		tsqlparser.TSqlParserMINUTE_ABBR:      true, // 'mi' | 'n'
		tsqlparser.TSqlParserSECOND_ABBR:      true, // 'ss' | 's'
		tsqlparser.TSqlParserMILLISECOND_ABBR: true, // 'ms'
		tsqlparser.TSqlParserMICROSECOND_ABBR: true, // 'mcs'
		tsqlparser.TSqlParserNANOSECOND_ABBR:  true, // 'ns'
		tsqlparser.TSqlParserTZOFFSET_ABBR:    true, // 'tz'
		tsqlparser.TSqlParserISO_WEEK_ABBR:    true, // 'isowk' | 'isoww'
		tsqlparser.TSqlParserWEEKDAY_ABBR:     true, // 'dw'

		tsqlparser.TSqlParserDISK_DRIVE:   true, // [A-Z][:];
		tsqlparser.TSqlParserIPV4_ADDR:    true, // DEC_DIGIT+ '.' DEC_DIGIT+ '.' DEC_DIGIT+ '.' DEC_DIGIT+;
		tsqlparser.TSqlParserSPACE:        true,
		tsqlparser.TSqlParserCOMMENT:      true,
		tsqlparser.TSqlParserLINE_COMMENT: true,

		tsqlparser.TSqlParserDOUBLE_QUOTE_ID:    true,
		tsqlparser.TSqlParserDOUBLE_QUOTE_BLANK: true,
		tsqlparser.TSqlParserSINGLE_QUOTE:       true,
		tsqlparser.TSqlParserSQUARE_BRACKET_ID:  true,
		tsqlparser.TSqlParserLOCAL_ID:           true,
		tsqlparser.TSqlParserDECIMAL:            true,
		tsqlparser.TSqlParserID:                 true,
		tsqlparser.TSqlParserSTRING:             true,
		tsqlparser.TSqlParserBINARY:             true,
		tsqlparser.TSqlParserFLOAT:              true,
		tsqlparser.TSqlParserREAL:               true,

		tsqlparser.TSqlParserEQUAL:        true,
		tsqlparser.TSqlParserGREATER:      true,
		tsqlparser.TSqlParserLESS:         true,
		tsqlparser.TSqlParserEXCLAMATION:  true,
		tsqlparser.TSqlParserPLUS_ASSIGN:  true,
		tsqlparser.TSqlParserMINUS_ASSIGN: true,
		tsqlparser.TSqlParserMULT_ASSIGN:  true,
		tsqlparser.TSqlParserDIV_ASSIGN:   true,
		tsqlparser.TSqlParserMOD_ASSIGN:   true,
		tsqlparser.TSqlParserAND_ASSIGN:   true,
		tsqlparser.TSqlParserXOR_ASSIGN:   true,
		tsqlparser.TSqlParserOR_ASSIGN:    true,

		tsqlparser.TSqlParserDOUBLE_BAR:   true,
		tsqlparser.TSqlParserDOT:          true,
		tsqlparser.TSqlParserUNDERLINE:    true,
		tsqlparser.TSqlParserAT:           true,
		tsqlparser.TSqlParserSHARP:        true,
		tsqlparser.TSqlParserDOLLAR:       true,
		tsqlparser.TSqlParserLR_BRACKET:   true,
		tsqlparser.TSqlParserRR_BRACKET:   true,
		tsqlparser.TSqlParserCOMMA:        true,
		tsqlparser.TSqlParserSEMI:         true,
		tsqlparser.TSqlParserCOLON:        true,
		tsqlparser.TSqlParserDOUBLE_COLON: true,
		tsqlparser.TSqlParserSTAR:         true,
		tsqlparser.TSqlParserDIVIDE:       true,
		tsqlparser.TSqlParserMODULE:       true,
		tsqlparser.TSqlParserPLUS:         true,
		tsqlparser.TSqlParserMINUS:        true,
		tsqlparser.TSqlParserBIT_NOT:      true,
		tsqlparser.TSqlParserBIT_OR:       true,
		tsqlparser.TSqlParserBIT_AND:      true,
		tsqlparser.TSqlParserBIT_XOR:      true,
		tsqlparser.TSqlParserPLACEHOLDER:  true,
	}
	preferredRules = map[int]bool{
		tsqlparser.TSqlParserRULE_built_in_functions: true,
		// full_table_name appears in the rule stack:
		// table_sources -> table_source -> table_source_item_joined -> table_source_item -> full_table_name
		tsqlparser.TSqlParserRULE_full_table_name:  true,
		tsqlparser.TSqlParserRULE_asterisk:         true,
		tsqlparser.TSqlParserRULE_full_column_name: true,
	}
)

type CompletionMap map[string]base.Candidate

func (m CompletionMap) Insert(entry base.Candidate) {
	m[entry.String()] = entry
}

func (m CompletionMap) toSlice() []base.Candidate {
	var result []base.Candidate
	for _, candidate := range m {
		result = append(result, candidate)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Type != result[j].Type {
			return result[i].Type < result[j].Type
		}
		return result[i].Text < result[j].Text
	})
	return result
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

	allDatabase, err := c.databaseNamesLister(c.ctx)
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

	allDBNames, err := c.databaseNamesLister(c.ctx)
	if err != nil {
		return
	}
	for _, dbName := range allDBNames {
		if strings.EqualFold(dbName, anchor) {
			anchor = dbName
			break
		}
	}

	_, databaseMetadata, err := c.metadataGetter(c.ctx, anchor)
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
	if linkedServer != "" {
		return
	}

	databaseName, schemaName := c.defaultDatabase, c.defaultSchema
	if database != "" {
		databaseName = database
	}
	if schema != "" {
		schemaName = schema
	}
	if databaseName == "" || schemaName == "" {
		return
	}

	_, databaseMetadata, err := c.metadataGetter(c.ctx, databaseName)
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

	schemaMetadata := databaseMetadata.GetSchema(schemaName)
	if schemaMetadata == nil {
		return
	}
	for _, table := range schemaMetadata.ListTableNames() {
		if _, ok := m[table]; !ok {
			m[table] = base.Candidate{
				Type: base.CandidateTypeTable,
				Text: c.quotedIdentifierIfNeeded(table),
			}
		}
	}
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
	databaseNames, err := c.databaseNamesLister(c.ctx)
	if err != nil {
		return
	}
	for _, dbName := range databaseNames {
		if strings.EqualFold(dbName, databaseName) {
			databaseName = dbName
			break
		}
	}
	_, databaseMetadata, err := c.metadataGetter(c.ctx, databaseName)
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
	schemaMetadata := databaseMetadata.GetSchema(schemaName)
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
		for _, column := range tableMetadata.GetColumns() {
			if _, ok := m[column.Name]; !ok {
				definition := fmt.Sprintf("%s | %s", table, column.Type)
				if !column.Nullable {
					definition += ", NOT NULL"
				}
				comment := column.UserComment
				m[column.Name] = base.Candidate{
					Type:       base.CandidateTypeColumn,
					Text:       c.quotedIdentifierIfNeeded(column.Name),
					Definition: definition,
					Comment:    comment,
				}
			}
		}
	}
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
	if linkedServer != "" {
		return
	}

	databaseName, schemaName := c.defaultDatabase, c.defaultSchema
	if database != "" {
		databaseName = database
	}
	if schema == "" {
		schemaName = schema
	}
	if databaseName == "" || schemaName == "" {
		return
	}

	_, databaseMetadata, err := c.metadataGetter(c.ctx, databaseName)
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

	schemaMetadata := databaseMetadata.GetSchema(schemaName)
	if schemaMetadata == nil {
		return
	}
	for _, view := range schemaMetadata.ListViewNames() {
		if _, ok := m[view]; !ok {
			m[view] = base.Candidate{
				Type: base.CandidateTypeView,
				Text: c.quotedIdentifierIfNeeded(view),
			}
		}
	}
	for _, materializeView := range schemaMetadata.ListMaterializedViewNames() {
		if _, ok := m[materializeView]; !ok {
			m[materializeView] = base.Candidate{
				Type: base.CandidateTypeView,
				Text: c.quotedIdentifierIfNeeded(materializeView),
			}
		}
	}
	for _, foreignTable := range schemaMetadata.ListForeignTableNames() {
		if _, ok := m[foreignTable]; !ok {
			m[foreignTable] = base.Candidate{
				Type: base.CandidateTypeView,
				Text: c.quotedIdentifierIfNeeded(foreignTable),
			}
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
	core    *base.CodeCompletionCore
	scene   base.SceneType
	parser  *tsqlparser.TSqlParser
	lexer   *tsqlparser.TSqlLexer
	scanner *base.Scanner

	defaultDatabase     string
	defaultSchema       string
	metadataGetter      base.GetDatabaseMetadataFunc
	databaseNamesLister base.ListDatabaseNamesFunc

	noSeparatorRequired map[int]bool
	// referencesStack is a hierarchical stack of table references.
	// We'll update the stack when we encounter a new FROM clauses.
	referencesStack [][]base.TableReference
	// references is the flattened table references.
	// It's helpful to look up the table reference.
	references         []base.TableReference
	cteCache           map[int][]*base.VirtualTableReference
	cteTables          []*base.VirtualTableReference
	caretTokenIsQuoted quotedType
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
	core := base.NewCodeCompletionCore(
		parser,
		ignoredTokens,  /* IgnoredTokens */
		preferredRules, /* PreferredRules */
		&globalFellowSetsByState,
		tsqlparser.TSqlParserRULE_select_statement,            /* queryRule */
		tsqlparser.TSqlParserRULE_select_statement_standalone, /* shadowQueryRule */
		tsqlparser.TSqlParserRULE_as_column_alias,             /* selectItemAliasRule */
		-1, /* cteRule */
	)

	return &Completer{
		ctx:                 ctx,
		core:                core,
		scene:               cCtx.Scene,
		parser:              parser,
		lexer:               lexer,
		scanner:             scanner,
		defaultDatabase:     cCtx.DefaultDatabase,
		defaultSchema:       "dbo",
		metadataGetter:      cCtx.Metadata,
		databaseNamesLister: cCtx.ListDatabaseNames,
		noSeparatorRequired: nil,
		cteCache:            nil,
	}
}

func NewTrickyCompleter(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) *Completer {
	parser, lexer, scanner := prepareTrickyParserAndScanner(statement, caretLine, caretOffset)
	core := base.NewCodeCompletionCore(
		parser,
		ignoredTokens,  /* IgnoredTokens */
		preferredRules, /* PreferredRules */
		&globalFellowSetsByState,
		tsqlparser.TSqlParserRULE_select_statement,            /* queryRule */
		tsqlparser.TSqlParserRULE_select_statement_standalone, /* shadowQueryRule */
		tsqlparser.TSqlParserRULE_as_column_alias,             /* selectItemAliasRule */
		-1, /* cteRule */
	)

	return &Completer{
		ctx:                 ctx,
		core:                core,
		scene:               cCtx.Scene,
		parser:              parser,
		lexer:               lexer,
		scanner:             scanner,
		defaultDatabase:     cCtx.DefaultDatabase,
		defaultSchema:       "dbo",
		metadataGetter:      cCtx.Metadata,
		databaseNamesLister: cCtx.ListDatabaseNames,
		noSeparatorRequired: nil,
		cteCache:            nil,
	}
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
	lexerErrorListener := &base.ParseErrorListener{}
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
	if c.scanner.IsTokenType(tsqlparser.TSqlLexerDOUBLE_QUOTE_ID) {
		c.caretTokenIsQuoted = quotedTypeDoubleQuote
	} else if c.scanner.IsTokenType(tsqlparser.TSqlLexerSQUARE_BRACKET_ID) {
		c.caretTokenIsQuoted = quotedTypeSquareBracket
	}
	caretIndex := c.scanner.GetIndex()
	if caretIndex > 0 && !c.noSeparatorRequired[c.scanner.GetPreviousTokenType(true)] {
		caretIndex--
	}
	c.referencesStack = append([][]base.TableReference{{}}, c.referencesStack...)
	c.parser.Reset()
	var context antlr.ParserRuleContext
	if c.scene == base.SceneTypeQuery {
		context = c.parser.Select_statement_standalone()
	} else {
		context = c.parser.Tsql_file()
	}
	candidates := c.core.CollectCandidates(caretIndex, context)

	for ruleName := range candidates.Rules {
		if ruleName == tsqlparser.TSqlParserRULE_asterisk || ruleName == tsqlparser.TSqlParserRULE_full_column_name {
			c.collectLeadingTableReferences(caretIndex)
			c.takeReferencesSnapshot()
			c.collectRemainingTableReferences()
			c.takeReferencesSnapshot()
			break
		}
	}
	return c.convertCandidates(candidates)
}

func (c *Completer) convertCandidates(candidates *base.CandidatesCollection) ([]base.Candidate, error) {
	keywordEntries := make(CompletionMap)
	functionEntries := make(CompletionMap)
	databaseEntries := make(CompletionMap)
	schemaEntries := make(CompletionMap)
	tableEntries := make(CompletionMap)
	columnEntries := make(CompletionMap)
	viewEntries := make(CompletionMap)

	for tokenCandidate, continuous := range candidates.Tokens {
		if tokenCandidate < 0 || tokenCandidate >= len(c.parser.SymbolicNames) {
			continue
		}

		candidateText := c.parser.SymbolicNames[tokenCandidate]
		for _, continuous := range continuous {
			if continuous < 0 || continuous >= len(c.parser.SymbolicNames) {
				continue
			}
			continuousText := c.parser.SymbolicNames[continuous]
			candidateText += " " + continuousText
		}
		keywordEntries.Insert(base.Candidate{
			Type: base.CandidateTypeKeyword,
			Text: candidateText,
		})
	}

	for ruleCandidate, ruleStack := range candidates.Rules {
		c.scanner.PopAndRestore()
		c.scanner.Push()

		switch ruleCandidate {
		case tsqlparser.TSqlParserRULE_built_in_functions:
			functionEntries.insertBuiltinFunctions()
		case tsqlparser.TSqlParserRULE_full_table_name:
			// full_table_name also appears in the full_column_name rule, we would handle it in the full_column_name rule in this case.
			if len(ruleStack) > 0 && ruleStack[len(ruleStack)-1].ID == tsqlparser.TSqlParserRULE_full_column_name {
				continue
			}
			completionContexts := c.determineFullTableNameContext()
			for _, context := range completionContexts {
				if context.flags&objectFlagShowDatabase != 0 {
					databaseEntries.insertMetadataDatabases(c, context.linkedServer)
				}
				if context.flags&objectFlagShowSchema != 0 {
					schemaEntries.insertMetadataSchemas(c, context.linkedServer, context.database)
				}
				if context.flags&objectFlagShowObject != 0 {
					tableEntries.insertMetadataTables(c, context.linkedServer, context.database, context.schema)
					viewEntries.insertMetadataViews(c, context.linkedServer, context.database, context.schema)
				}
				if context.linkedServer == "" && context.database == "" && context.schema == "" && context.flags&objectFlagShowObject != 0 {
					// User do not specify the server, database and schema, and want us complete the objects, we should also insert the ctes.
					tableEntries.insertCTEs(c)
				}
			}
		case tsqlparser.TSqlParserRULE_asterisk:
			completionContexts := c.determineAsteriskContext()
			for _, context := range completionContexts {
				if context.flags&objectFlagShowDatabase != 0 {
					databaseEntries.insertMetadataDatabases(c, context.linkedServer)
				}
				if context.flags&objectFlagShowSchema != 0 {
					schemaEntries.insertMetadataSchemas(c, context.linkedServer, context.database)
				}
				if context.flags&objectFlagShowObject != 0 {
					tableEntries.insertMetadataTables(c, context.linkedServer, context.database, context.schema)
					viewEntries.insertMetadataViews(c, context.linkedServer, context.database, context.schema)
				}
				if context.linkedServer == "" && context.database == "" && context.schema == "" && context.flags&objectFlagShowObject != 0 {
					// User do not specify the server, database and schema, and want us complete the objects, we should also insert the ctes.
					tableEntries.insertCTEs(c)
				}
			}
		case tsqlparser.TSqlParserRULE_full_column_name:
			completionContexts := c.determineFullColumnName()
			for _, context := range completionContexts {
				if context.flags&objectFlagShowDatabase != 0 {
					databaseEntries.insertMetadataDatabases(c, context.linkedServer)
				}
				if context.flags&objectFlagShowSchema != 0 {
					schemaEntries.insertMetadataSchemas(c, context.linkedServer, context.database)
				}
				if context.flags&objectFlagShowObject != 0 {
					tableEntries.insertMetadataTables(c, context.linkedServer, context.database, context.schema)
					viewEntries.insertMetadataViews(c, context.linkedServer, context.database, context.schema)
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
						}
					}
					if context.linkedServer == "" && context.database == "" && context.schema == "" {
						// User do not specify the server, database and schema, and want us complete the objects, we should also insert the ctes.
						tableEntries.insertCTEs(c)
					}
				}
				if context.flags&objectFlagShowColumn != 0 {
					list := c.fetchSelectItemAliases(ruleStack)
					for _, alias := range list {
						columnEntries.Insert(base.Candidate{
							Type: base.CandidateTypeColumn,
							Text: c.quotedIdentifierIfNeeded(alias),
						})
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
				}
			}
		}
	}

	c.scanner.PopAndRestore()
	var result []base.Candidate
	result = append(result, keywordEntries.toSlice()...)
	result = append(result, functionEntries.toSlice()...)
	result = append(result, databaseEntries.toSlice()...)
	result = append(result, schemaEntries.toSlice()...)
	result = append(result, tableEntries.toSlice()...)
	result = append(result, columnEntries.toSlice()...)
	result = append(result, viewEntries.toSlice()...)
	return result, nil
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
	}

	if len(results) == 0 {
		results = append(results, refCtx.clone())
	}

	return results
}

// skipHeadingSQLs skips the SQL statements which before the caret position.
// caretLine is 1-based and caretOffset is 0-based.
func skipHeadingSQLs(statement string, caretLine int, caretOffset int) (string, int, int) {
	newCaretLine, newCaretOffset := caretLine, caretOffset
	list, err := SplitSQL(statement)
	if err != nil || notEmptySQLCount(list) <= 1 {
		return statement, caretLine, caretOffset
	}

	caretLine-- // Convert to 0-based.

	start := 0
	for i, sql := range list {
		if sql.LastLine > caretLine || (sql.LastLine == caretLine && sql.LastColumn >= caretOffset) {
			start = i
			if i == 0 {
				// The caret is in the first SQL statement, so we don't need to skip any SQL statements.
				continue
			}
			newCaretLine = caretLine - list[i-1].LastLine + 1 // Convert to 1-based.
			if caretLine == list[i-1].LastLine {
				// The caret is in the same line as the last line of the previous SQL statement.
				// We need to adjust the caret offset.
				newCaretOffset = caretOffset - list[i-1].LastColumn - 1 // Convert to 0-based.
			}
		}
	}

	var buf strings.Builder
	for i := start; i < len(list); i++ {
		if _, err := buf.WriteString(list[i].Text); err != nil {
			return statement, caretLine, caretOffset
		}
	}

	return buf.String(), newCaretLine, newCaretOffset
}

func notEmptySQLCount(list []base.SingleSQL) int {
	count := 0
	for _, sql := range list {
		if !sql.Empty {
			count++
		}
	}
	return count
}

func (c *Completer) takeReferencesSnapshot() {
	for _, references := range c.referencesStack {
		c.references = append(c.references, references...)
	}
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
		_ /* Linked Server */, database, schema, table := normalizeFullTableNameFallback(ctx, "", "", "")
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
		if span, err := base.GetQuerySpan(
			l.context.ctx,
			base.GetQuerySpanContext{
				GetDatabaseMetadataFunc: l.context.metadataGetter,
				ListDatabaseNamesFunc:   l.context.databaseNamesLister,
			},
			storepb.Engine_MSSQL,
			fmt.Sprintf("SELECT * FROM (%s);", ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)),
			l.context.defaultDatabase,
			l.context.defaultSchema,
			true,
		); err == nil && len(span) == 1 {
			for _, column := range span[0].Results {
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
		if span, err := base.GetQuerySpan(
			c.completer.ctx,
			base.GetQuerySpanContext{
				GetDatabaseMetadataFunc: c.completer.metadataGetter,
				ListDatabaseNamesFunc:   c.completer.databaseNamesLister,
			},
			storepb.Engine_MSSQL,
			statement,
			c.completer.defaultDatabase,
			c.completer.defaultSchema,
			true,
		); err == nil && len(span) == 1 {
			var columns []string
			for _, column := range span[0].Results {
				columns = append(columns, column.Name)
			}
			c.virtualReferences = append(c.virtualReferences, &base.VirtualTableReference{
				Table:   unquote(cteName),
				Columns: columns,
			})
		}
	}
}

func (c *Completer) fetchSelectItemAliases(ruleStack []*base.RuleContext) []string {
	canUseAliases := false
	for i := len(ruleStack) - 1; i >= 0; i-- {
		switch ruleStack[i].ID {
		case tsqlparser.TSqlParserRULE_select_statement, tsqlparser.TSqlParserRULE_select_statement_standalone:
			if !canUseAliases {
				return nil
			}
			aliasMap := make(map[string]bool)
			for pos := range ruleStack[i].SelectItemAliases {
				if aliasText := c.extractAliasText(pos); len(aliasText) > 0 {
					aliasMap[aliasText] = true
				}
			}

			var result []string
			for alias := range aliasMap {
				result = append(result, alias)
			}
			sort.Slice(result, func(i, j int) bool {
				return result[i] < result[j]
			})
			return result
		case tsqlparser.TSqlParserRULE_group_by_clause, tsqlparser.TSqlParserRULE_order_by_clause, tsqlparser.TSqlParserRULE_having_clause:
			canUseAliases = true
		}
	}

	return nil
}

func (c *Completer) extractAliasText(pos int) string {
	followingText := c.scanner.GetFollowingTextAfter(pos)
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
