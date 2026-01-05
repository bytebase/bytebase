package trino

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode"

	"github.com/antlr4-go/antlr/v4"
	trinoparser "github.com/bytebase/parser/trino"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterCompleteFunc(storepb.Engine_TRINO, Completion)
}

var (
	// Check tableRefListener is implementing the TrinoParserListener interface.
	_ trinoparser.TrinoParserListener = &tableRefListener{}
	_ trinoparser.TrinoParserListener = &cteExtractor{}

	globalFellowSetsByState = base.NewFollowSetsByState()
	ignoredTokens           = map[int]bool{
		// Common EOF
		trinoparser.TrinoParserEOF: true,

		// // Whitespace and comments
		// trinoparser.TrinoLexerWS_:                true,
		// trinoparser.TrinoLexerSIMPLE_COMMENT_:    true,
		// trinoparser.TrinoLexerBRACKETED_COMMENT_: true,

		// Identifiers and literals
		trinoparser.TrinoLexerIDENTIFIER_:            true,
		trinoparser.TrinoLexerQUOTED_IDENTIFIER_:     true,
		trinoparser.TrinoLexerDIGIT_IDENTIFIER_:      true,
		trinoparser.TrinoLexerBACKQUOTED_IDENTIFIER_: true,
		trinoparser.TrinoLexerSTRING_:                true,
		trinoparser.TrinoLexerUNICODE_STRING_:        true,
		trinoparser.TrinoLexerDECIMAL_VALUE_:         true,
		trinoparser.TrinoLexerDOUBLE_VALUE_:          true,
		trinoparser.TrinoLexerINTEGER_VALUE_:         true,
		trinoparser.TrinoLexerBINARY_LITERAL_:        true,

		// // Type related tokens
		// trinoparser.TrinoLexerDOUBLE_:    true,
		// trinoparser.TrinoLexerPRECISION_: true,

		// Parameter token
		trinoparser.TrinoLexerQUESTION_MARK_: true,

		// Operators and punctuation
		trinoparser.TrinoLexerEQ_:           true,
		trinoparser.TrinoLexerNEQ_:          true,
		trinoparser.TrinoLexerLT_:           true,
		trinoparser.TrinoLexerLTE_:          true,
		trinoparser.TrinoLexerGT_:           true,
		trinoparser.TrinoLexerGTE_:          true,
		trinoparser.TrinoLexerPLUS_:         true,
		trinoparser.TrinoLexerMINUS_:        true,
		trinoparser.TrinoLexerASTERISK_:     true,
		trinoparser.TrinoLexerSLASH_:        true,
		trinoparser.TrinoLexerPERCENT_:      true,
		trinoparser.TrinoLexerCONCAT_:       true,
		trinoparser.TrinoLexerDOT_:          true,
		trinoparser.TrinoLexerCOLON_:        true,
		trinoparser.TrinoLexerSEMICOLON_:    true,
		trinoparser.TrinoLexerCOMMA_:        true,
		trinoparser.TrinoLexerLPAREN_:       true,
		trinoparser.TrinoLexerRPAREN_:       true,
		trinoparser.TrinoLexerLSQUARE_:      true,
		trinoparser.TrinoLexerRSQUARE_:      true,
		trinoparser.TrinoLexerLCURLY_:       true,
		trinoparser.TrinoLexerRCURLY_:       true,
		trinoparser.TrinoLexerLCURLYHYPHEN_: true,
		trinoparser.TrinoLexerRCURLYHYPHEN_: true,
		trinoparser.TrinoLexerLARROW_:       true,
		trinoparser.TrinoLexerRARROW_:       true,
		trinoparser.TrinoLexerRDOUBLEARROW_: true,
		trinoparser.TrinoLexerVBAR_:         true,
		trinoparser.TrinoLexerDOLLAR_:       true,
		trinoparser.TrinoLexerCARET_:        true,
		trinoparser.TrinoLexerAT_:           true,
	}
	preferredRules = map[int]bool{
		trinoparser.TrinoParserRULE_identifier:    true,
		trinoparser.TrinoParserRULE_qualifiedName: true,
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
	slices.SortFunc(result, func(i, j base.Candidate) int {
		if i.Type != j.Type {
			if i.Type < j.Type {
				return -1
			}
			return 1
		}
		if i.Text < j.Text {
			return -1
		}
		if i.Text > j.Text {
			return 1
		}
		return 0
	})
	return result
}

func (m CompletionMap) insertMetadataCatalogs(c *Completer) {
	if c.defaultCatalog != "" {
		m[c.defaultCatalog] = base.Candidate{
			Type: base.CandidateTypeDatabase,
			Text: c.quotedIdentifierIfNeeded(c.defaultCatalog),
		}
	}

	allCatalogs, err := c.catalogNamesLister(c.ctx, c.instanceID)
	if err != nil {
		return
	}

	for _, catalog := range allCatalogs {
		if _, ok := m[catalog]; !ok {
			m[catalog] = base.Candidate{
				Type: base.CandidateTypeDatabase,
				Text: c.quotedIdentifierIfNeeded(catalog),
			}
		}
	}
}

func (m CompletionMap) insertMetadataSchemas(c *Completer, catalog string) {
	anchor := c.defaultCatalog
	if catalog != "" {
		anchor = catalog
	}
	if anchor == "" {
		return
	}

	allCatalogNames, err := c.catalogNamesLister(c.ctx, c.instanceID)
	if err != nil {
		return
	}
	for _, catalogName := range allCatalogNames {
		if strings.EqualFold(catalogName, anchor) {
			anchor = catalogName
			break
		}
	}

	_, catalogMetadata, err := c.metadataGetter(c.ctx, c.instanceID, anchor)
	if err != nil {
		return
	}

	for _, schema := range catalogMetadata.ListSchemaNames() {
		if _, ok := m[schema]; !ok {
			m[schema] = base.Candidate{
				Type: base.CandidateTypeSchema,
				Text: c.quotedIdentifierIfNeeded(schema),
			}
		}
	}
}

func (m CompletionMap) insertMetadataTables(c *Completer, catalog string, schema string) {
	catalogName, schemaName := c.defaultCatalog, c.defaultSchema
	if catalog != "" {
		catalogName = catalog
	}
	if schema != "" {
		schemaName = schema
	}
	if catalogName == "" || schemaName == "" {
		return
	}

	_, catalogMetadata, err := c.metadataGetter(c.ctx, c.instanceID, catalogName)
	if err != nil {
		return
	}
	if catalogMetadata == nil {
		return
	}
	for _, schema := range catalogMetadata.ListSchemaNames() {
		if strings.EqualFold(schema, schemaName) {
			schemaName = schema
			break
		}
	}

	schemaMetadata := catalogMetadata.GetSchemaMetadata(schemaName)
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

func (m CompletionMap) insertAllColumns(c *Completer) {
	_, catalogMeta, err := c.metadataGetter(c.ctx, c.instanceID, c.defaultCatalog)
	if err != nil {
		return
	}
	for _, schema := range catalogMeta.ListSchemaNames() {
		schemaMeta := catalogMeta.GetSchemaMetadata(schema)
		if schemaMeta == nil {
			continue
		}
		for _, table := range schemaMeta.ListTableNames() {
			tableMeta := schemaMeta.GetTable(table)
			if tableMeta == nil {
				continue
			}
			for _, column := range tableMeta.GetProto().GetColumns() {
				columnID := getColumnID(c.defaultCatalog, schema, table, column.Name)
				if _, ok := m[columnID]; !ok {
					definition := fmt.Sprintf("%s.%s.%s | %s", c.defaultCatalog, schema, table, column.Type)
					if !column.Nullable {
						definition += ", NOT NULL"
					}
					m[columnID] = base.Candidate{
						Type:       base.CandidateTypeColumn,
						Text:       c.quotedIdentifierIfNeeded(column.Name),
						Definition: definition,
						Comment:    column.Comment,
						Priority:   c.getPriority(c.defaultCatalog, schema, table),
					}
				}
			}
		}
	}
}

func (c *Completer) getPriority(catalog string, schema string, table string) int {
	if catalog == "" {
		catalog = c.defaultCatalog
	}
	if schema == "" {
		schema = c.defaultSchema
	}
	if c.referenceMap == nil {
		return 1
	}
	if c.referenceMap[fmt.Sprintf("%s.%s.%s", catalog, schema, table)] {
		// The higher priority.
		return 0
	}
	return 1
}

func (m CompletionMap) insertMetadataColumns(c *Completer, catalog string, schema string, table string) {
	catalogName, schemaName, tableName := c.defaultCatalog, c.defaultSchema, ""
	if catalog != "" {
		catalogName = catalog
	}
	if schema != "" {
		schemaName = schema
	}
	if table != "" {
		tableName = table
	}
	if catalogName == "" || schemaName == "" {
		return
	}
	catalogNames, err := c.catalogNamesLister(c.ctx, c.instanceID)
	if err != nil {
		return
	}
	for _, catName := range catalogNames {
		if strings.EqualFold(catName, catalogName) {
			catalogName = catName
			break
		}
	}
	_, catalogMetadata, err := c.metadataGetter(c.ctx, c.instanceID, catalogName)
	if err != nil {
		return
	}
	if catalogMetadata == nil {
		return
	}
	for _, schema := range catalogMetadata.ListSchemaNames() {
		if strings.EqualFold(schema, schemaName) {
			schemaName = schema
			break
		}
	}
	schemaMetadata := catalogMetadata.GetSchemaMetadata(schemaName)
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
			columnID := getColumnID(catalogName, schemaName, table, column.Name)
			if _, ok := m[columnID]; !ok {
				definition := fmt.Sprintf("%s.%s.%s | %s", catalogName, schemaName, table, column.Type)
				if !column.Nullable {
					definition += ", NOT NULL"
				}
				m[columnID] = base.Candidate{
					Type:       base.CandidateTypeColumn,
					Text:       c.quotedIdentifierIfNeeded(column.Name),
					Definition: definition,
					Comment:    column.Comment,
					Priority:   c.getPriority(c.defaultCatalog, schema, table),
				}
			}
		}
	}
}

func getColumnID(catalogName, schemaName, tableName, columnName string) string {
	return fmt.Sprintf("%s.%s.%s.%s", catalogName, schemaName, tableName, columnName)
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

func (m CompletionMap) insertMetadataViews(c *Completer, catalog string, schema string) {
	catalogName, schemaName := c.defaultCatalog, c.defaultSchema
	if catalog != "" {
		catalogName = catalog
	}
	if schema != "" {
		schemaName = schema
	}

	if catalogName == "" || schemaName == "" {
		return
	}

	_, catalogMetadata, err := c.metadataGetter(c.ctx, c.instanceID, catalogName)
	if err != nil {
		return
	}
	if catalogMetadata == nil {
		return
	}

	schemaNames := catalogMetadata.ListSchemaNames()

	var foundMatch bool
	for _, schema := range schemaNames {
		if strings.EqualFold(schema, schemaName) {
			schemaName = schema
			foundMatch = true
			break
		}
	}

	if !foundMatch {
		return
	}

	schemaMetadata := catalogMetadata.GetSchemaMetadata(schemaName)
	if schemaMetadata == nil {
		return
	}

	viewNames := schemaMetadata.ListViewNames()

	for _, view := range viewNames {
		if _, ok := m[view]; !ok {
			m[view] = base.Candidate{
				Type: base.CandidateTypeView,
				Text: c.quotedIdentifierIfNeeded(view),
			}
		}
	}

	matViewNames := schemaMetadata.ListMaterializedViewNames()

	for _, materializeView := range matViewNames {
		if _, ok := m[materializeView]; !ok {
			m[materializeView] = base.Candidate{
				Type: base.CandidateTypeView,
				Text: c.quotedIdentifierIfNeeded(materializeView),
			}
		}
	}

	foreignTableNames := schemaMetadata.ListForeignTableNames()

	for _, foreignTable := range foreignTableNames {
		if _, ok := m[foreignTable]; !ok {
			m[foreignTable] = base.Candidate{
				Type: base.CandidateTypeView,
				Text: c.quotedIdentifierIfNeeded(foreignTable),
			}
		}
	}
}

// quotedType is the type of quoted token
type quotedType int

const (
	quotedTypeNone      quotedType = iota
	quotedTypeQuote                // ""
	quotedTypeBackQuote            // ``
)

type Completer struct {
	ctx     context.Context
	core    *base.CodeCompletionCore
	scene   base.SceneType
	parser  *trinoparser.TrinoParser
	lexer   *trinoparser.TrinoLexer
	scanner *base.Scanner

	instanceID         string
	defaultCatalog     string
	defaultSchema      string
	metadataGetter     base.GetDatabaseMetadataFunc
	catalogNamesLister base.ListDatabaseNamesFunc

	noSeparatorRequired map[int]bool
	// referencesStack is a hierarchical stack of table references.
	// We'll update the stack when we encounter a new FROM clauses.
	referencesStack [][]base.TableReference
	// references is the flattened table references.
	// It's helpful to look up the table reference.
	references         []base.TableReference
	referenceMap       map[string]bool
	cteCache           map[int][]*base.VirtualTableReference
	cteTables          []*base.VirtualTableReference
	caretTokenIsQuoted quotedType
}

func Completion(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) ([]base.Candidate, error) {
	// Try standard completer first
	completer := NewStandardCompleter(ctx, cCtx, statement, caretLine, caretOffset)
	completer.fetchCommonTableExpression(statement)
	result, err := completer.complete()

	if err != nil {
		return nil, err
	}
	if len(result) > 0 {
		return result, nil
	}

	// If no results, try tricky completer with skipHeadingSQLWithoutSemicolon
	trickyCompleter := NewTrickyCompleter(ctx, cCtx, statement, caretLine, caretOffset)
	trickyCompleter.fetchCommonTableExpression(statement)
	return trickyCompleter.complete()
}

func NewStandardCompleter(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) *Completer {
	parser, lexer, scanner := prepareParserAndScanner(statement, caretLine, caretOffset)
	core := base.NewCodeCompletionCore(
		ctx,
		parser,
		ignoredTokens,  /* IgnoredTokens */
		preferredRules, /* PreferredRules */
		&globalFellowSetsByState,
		trinoparser.TrinoParserRULE_queryNoWith,
		trinoparser.TrinoParserRULE_query,
		trinoparser.TrinoParserRULE_as_column_alias,
		-1,
	)

	return &Completer{
		ctx:                ctx,
		core:               core,
		scene:              cCtx.Scene,
		parser:             parser,
		lexer:              lexer,
		scanner:            scanner,
		instanceID:         cCtx.InstanceID,
		defaultCatalog:     cCtx.DefaultDatabase,
		defaultSchema:      "dbo",
		metadataGetter:     cCtx.Metadata,
		catalogNamesLister: cCtx.ListDatabaseNames,
		noSeparatorRequired: map[int]bool{
			trinoparser.TrinoLexerCOMMA_:     true,
			trinoparser.TrinoLexerSEMICOLON_: true,
			trinoparser.TrinoLexerDOT_:       true,
			trinoparser.TrinoLexerLPAREN_:    true,
			trinoparser.TrinoLexerRPAREN_:    true,
			trinoparser.TrinoLexerLSQUARE_:   true,
			trinoparser.TrinoLexerRSQUARE_:   true,
		},
		cteCache: nil,
	}
}

func NewTrickyCompleter(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) *Completer {
	parser, lexer, scanner := prepareTrickyParserAndScanner(statement, caretLine, caretOffset)
	core := base.NewCodeCompletionCore(
		ctx,
		parser,
		ignoredTokens,  /* IgnoredTokens */
		preferredRules, /* PreferredRules */
		&globalFellowSetsByState,
		trinoparser.TrinoParserRULE_queryNoWith,
		trinoparser.TrinoParserRULE_query,
		trinoparser.TrinoParserRULE_as_column_alias,
		-1,
	)

	return &Completer{
		ctx:                ctx,
		core:               core,
		scene:              cCtx.Scene,
		parser:             parser,
		lexer:              lexer,
		scanner:            scanner,
		instanceID:         cCtx.InstanceID,
		defaultCatalog:     cCtx.DefaultDatabase,
		defaultSchema:      "dbo",
		metadataGetter:     cCtx.Metadata,
		catalogNamesLister: cCtx.ListDatabaseNames,
		noSeparatorRequired: map[int]bool{
			trinoparser.TrinoLexerCOMMA_:     true,
			trinoparser.TrinoLexerSEMICOLON_: true,
			trinoparser.TrinoLexerDOT_:       true,
			trinoparser.TrinoLexerLPAREN_:    true,
			trinoparser.TrinoLexerRPAREN_:    true,
			trinoparser.TrinoLexerLSQUARE_:   true,
			trinoparser.TrinoLexerRSQUARE_:   true,
		},
		cteCache: nil,
	}
}

func prepareParserAndScanner(statement string, caretLine int, caretOffset int) (*trinoparser.TrinoParser, *trinoparser.TrinoLexer, *base.Scanner) {
	statement, caretLine, caretOffset = skipHeadingSQLs(statement, caretLine, caretOffset)
	input := antlr.NewInputStream(statement)
	lexer := trinoparser.NewTrinoLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := trinoparser.NewTrinoParser(stream)
	parser.RemoveErrorListeners()
	lexer.RemoveErrorListeners()
	scanner := base.NewScanner(stream, true /* fillInput */)
	scanner.SeekPosition(caretLine, caretOffset)
	scanner.Push()
	return parser, lexer, scanner
}

func prepareTrickyParserAndScanner(statement string, caretLine int, caretOffset int) (*trinoparser.TrinoParser, *trinoparser.TrinoLexer, *base.Scanner) {
	statement, caretLine, caretOffset = skipHeadingSQLs(statement, caretLine, caretOffset)
	statement, caretLine, caretOffset = skipHeadingSQLWithoutSemicolon(statement, caretLine, caretOffset)
	input := antlr.NewInputStream(statement)
	lexer := trinoparser.NewTrinoLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := trinoparser.NewTrinoParser(stream)
	parser.RemoveErrorListeners()
	lexer.RemoveErrorListeners()
	scanner := base.NewScanner(stream, true /* fillInput */)
	scanner.SeekPosition(caretLine, caretOffset)
	scanner.Push()
	return parser, lexer, scanner
}

func (c *Completer) complete() ([]base.Candidate, error) {
	if c.scanner.IsTokenType(trinoparser.TrinoParserQUOTED_IDENTIFIER_) {
		c.caretTokenIsQuoted = quotedTypeQuote
	}

	caretIndex := c.scanner.GetIndex()

	// Check if we need to adjust the caret position based on the previous token
	adjustedCaretIndex := caretIndex
	if caretIndex > 0 && !c.noSeparatorRequired[c.scanner.GetPreviousTokenType(true)] {
		adjustedCaretIndex--
	}

	c.referencesStack = append([][]base.TableReference{{}}, c.referencesStack...)
	c.parser.Reset()

	var context antlr.ParserRuleContext
	if c.scene == base.SceneTypeQuery {
		context = c.parser.SingleStatement()
	} else {
		context = c.parser.Parse()
	}

	// For Trino, we need special handling because whitespace tokens are included
	// We add 1 in most cases, but not when:
	// 1. We're at the very beginning (caretIndex == 0)
	// 2. We're right after an identifier that could be a partial keyword
	finalCaretIndex := adjustedCaretIndex + 1

	// Special cases where we don't add 1
	if caretIndex == 0 {
		// At the very beginning
		finalCaretIndex = adjustedCaretIndex
	} else if c.scanner.GetTokenType() == antlr.TokenEOF && c.scanner.GetPreviousTokenType(true) == trinoparser.TrinoLexerIDENTIFIER_ {
		// At EOF right after an identifier (potential partial keyword)
		finalCaretIndex = adjustedCaretIndex
	}

	candidates := c.core.CollectCandidates(finalCaretIndex, context)

	for ruleName := range candidates.Rules {
		if ruleName == trinoparser.TrinoParserRULE_identifier {
			c.collectLeadingTableReferences(adjustedCaretIndex)
			c.takeReferencesSnapshot()
			c.collectRemainingTableReferences()
			c.takeReferencesSnapshot()
			break
		}
	}

	result := c.convertCandidates(candidates)
	return result, nil
}

func (c *Completer) convertCandidates(candidates *base.CandidatesCollection) []base.Candidate {
	keywordEntries := make(CompletionMap)
	functionEntries := make(CompletionMap)
	catalogEntries := make(CompletionMap)
	schemaEntries := make(CompletionMap)
	tableEntries := make(CompletionMap)
	columnEntries := make(CompletionMap)
	viewEntries := make(CompletionMap)

	for tokenCandidate, continuous := range candidates.Tokens {
		if tokenCandidate < 0 || tokenCandidate >= len(c.parser.SymbolicNames) {
			continue
		}

		candidateText := c.parser.SymbolicNames[tokenCandidate]
		// Remove trailing underscore from keywords in Trino parser
		candidateText = strings.TrimSuffix(candidateText, "_")

		for _, continuous := range continuous {
			if continuous < 0 || continuous >= len(c.parser.SymbolicNames) {
				continue
			}
			continuousText := c.parser.SymbolicNames[continuous]
			// Remove trailing underscore from continuous tokens as well
			continuousText = strings.TrimSuffix(continuousText, "_")
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
		case trinoparser.TrinoParserRULE_qualifiedName:
			// identifier also appears in the qualifiedName rule, handle it separately
			if len(ruleStack) > 0 && ruleStack[len(ruleStack)-1].ID == trinoparser.TrinoParserRULE_identifier {
				continue
			}
			completionContexts := c.determineQualifiedNameContext()
			for _, context := range completionContexts {
				if context.flags&objectFlagShowCatalog != 0 {
					catalogEntries.insertMetadataCatalogs(c)
				}
				if context.flags&objectFlagShowSchema != 0 {
					schemaEntries.insertMetadataSchemas(c, context.catalog)
				}
				if context.flags&objectFlagShowObject != 0 {
					tableEntries.insertMetadataTables(c, context.catalog, context.schema)
					viewEntries.insertMetadataViews(c, context.catalog, context.schema)
				}
				if context.catalog == "" && context.schema == "" && context.flags&objectFlagShowObject != 0 {
					// User did not specify catalog and schema, and wants to complete objects, we should also insert the CTEs
					tableEntries.insertCTEs(c)
				}
			}
		case trinoparser.TrinoParserRULE_identifier:
			completionContexts := c.determineColumnReference()
			for _, context := range completionContexts {
				if context.flags&objectFlagShowCatalog != 0 {
					catalogEntries.insertMetadataCatalogs(c)
				}
				if context.flags&objectFlagShowSchema != 0 {
					schemaEntries.insertMetadataSchemas(c, context.catalog)
				}
				if context.flags&objectFlagShowObject != 0 {
					tableEntries.insertMetadataTables(c, context.catalog, context.schema)
					viewEntries.insertMetadataViews(c, context.catalog, context.schema)

					// Add table references from the FROM clause
					for _, reference := range c.references {
						switch reference := reference.(type) {
						case *base.PhysicalTableReference:
							tableName := reference.Table
							if reference.Alias != "" {
								tableName = reference.Alias
							}
							if strings.EqualFold(reference.Database, context.catalog) &&
								strings.EqualFold(reference.Schema, context.schema) {
								if _, ok := tableEntries[tableName]; !ok {
									tableEntries[tableName] = base.Candidate{
										Type: base.CandidateTypeTable,
										Text: c.quotedIdentifierIfNeeded(tableName),
									}
								}
							}
						case *base.VirtualTableReference:
							// We only append the virtual table reference to the completion list
							// when the catalog and schema are all empty.
							if context.catalog == "" && context.schema == "" {
								tableEntries[reference.Table] = base.Candidate{
									Type: base.CandidateTypeTable,
									Text: c.quotedIdentifierIfNeeded(reference.Table),
								}
							}
						}
					}
					if context.catalog == "" && context.schema == "" {
						// User did not specify catalog and schema, and wants to complete objects,
						// we should also insert the CTEs
						tableEntries.insertCTEs(c)
					}
				}
				if context.flags&objectFlagShowColumn != 0 {
					// Add column aliases from select items
					list := c.fetchSelectItemAliases(ruleStack)
					for _, alias := range list {
						columnEntries.Insert(base.Candidate{
							Type: base.CandidateTypeColumn,
							Text: c.quotedIdentifierIfNeeded(alias),
						})
					}

					// Add columns from metadata for the specified table
					columnEntries.insertMetadataColumns(c, context.catalog, context.schema, context.object)

					// Add columns from table references in the current query
					for _, reference := range c.references {
						switch reference := reference.(type) {
						case *base.PhysicalTableReference:
							inputCatalogName := context.catalog
							if inputCatalogName == "" {
								inputCatalogName = c.defaultCatalog
							}
							inputSchemaName := context.schema
							if inputSchemaName == "" {
								inputSchemaName = c.defaultSchema
							}
							inputTableName := context.object

							referenceCatalogName := reference.Database
							if referenceCatalogName == "" {
								referenceCatalogName = c.defaultCatalog
							}
							referenceSchemaName := reference.Schema
							if referenceSchemaName == "" {
								referenceSchemaName = c.defaultSchema
							}
							referenceTableName := reference.Table
							if reference.Alias != "" {
								referenceTableName = reference.Alias
							}

							if strings.EqualFold(referenceCatalogName, inputCatalogName) &&
								strings.EqualFold(referenceSchemaName, inputSchemaName) &&
								strings.EqualFold(referenceTableName, inputTableName) {
								columnEntries.insertMetadataColumns(c, reference.Database, reference.Schema, reference.Table)
							}
						case *base.VirtualTableReference:
							// Reference could be a physical table reference or a virtual table reference
							// If the reference is a virtual table reference and users do not specify
							// the catalog and schema, we should also insert the columns.
							if context.catalog == "" && context.schema == "" {
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

					// Add columns from CTEs if applicable
					if context.catalog == "" && context.schema == "" {
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

					// If no specific table was referenced, show all columns from the default catalog and schema
					if context.empty() {
						columnEntries.insertAllColumns(c)
					}
				}
			}
		default:
			// Ignore other rule candidates
		}
	}

	c.scanner.PopAndRestore()
	var result []base.Candidate
	result = append(result, keywordEntries.toSlice()...)
	result = append(result, functionEntries.toSlice()...)
	result = append(result, catalogEntries.toSlice()...)
	result = append(result, schemaEntries.toSlice()...)
	result = append(result, tableEntries.toSlice()...)
	result = append(result, columnEntries.toSlice()...)
	result = append(result, viewEntries.toSlice()...)
	if result == nil {
		result = []base.Candidate{}
	}
	return result
}

type objectFlag int

const (
	objectFlagShowCatalog objectFlag = 1 << iota
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

func newObjectRefContext(options ...objectRefContextOption) *objectRefContext {
	o := &objectRefContext{
		flags: objectFlagShowCatalog | objectFlagShowSchema | objectFlagShowObject,
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
	catalog string
	schema  string
	object  string

	// column is optional considering field, for example, it should be not applicable for qualified name rule.
	column string

	flags objectFlag
}

func (o *objectRefContext) empty() bool {
	return o.catalog == "" && o.schema == "" && o.object == "" && o.column == ""
}

func (o *objectRefContext) clone() *objectRefContext {
	return &objectRefContext{
		catalog: o.catalog,
		schema:  o.schema,
		object:  o.object,
		column:  o.column,
		flags:   o.flags,
	}
}

func (o *objectRefContext) setCatalog(catalog string) *objectRefContext {
	o.catalog = catalog
	o.flags &= ^objectFlagShowCatalog
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

func (c *Completer) determineColumnReference() []*objectRefContext {
	tokenIndex := c.scanner.GetIndex()
	if c.scanner.GetTokenChannel() != antlr.TokenDefaultChannel {
		// Skip to the next non-hidden token.
		c.scanner.Forward(true /* skipHidden */)
	}

	// Check current token
	if c.scanner.GetTokenText() != "." && !isIdentifier(c.scanner.GetTokenType()) {
		c.scanner.Backward(true /* skipHidden */)
	}

	if tokenIndex > 0 {
		// Go backward until we hit a non-identifier token.
		for {
			curID := isIdentifier(c.scanner.GetTokenType()) && c.scanner.GetPreviousTokenText(false /* skipHidden */) == "."
			curDOT := c.scanner.GetTokenText() == "." && isIdentifier(c.scanner.GetPreviousTokenType(false /* skipHidden */))
			if curID || curDOT {
				c.scanner.Backward(true /* skipHidden */)
				continue
			}
			break
		}
	}

	// The scanner is now on the leading identifier (or dot?) if there's no leading id.
	var candidates []string
	var temp string
	var count int
	for {
		count++
		if isIdentifier(c.scanner.GetTokenType()) {
			temp = normalizeIdentifierText(c.scanner.GetTokenText())
			c.scanner.Forward(true /* skipHidden */)
			if !c.scanner.IsTokenType(trinoparser.TrinoParserDOT_) || tokenIndex <= c.scanner.GetIndex() {
				return deriveObjectRefContextsFromCandidates(candidates, true /* includeColumn */)
			}
			candidates = append(candidates, temp)
		}
		c.scanner.Forward(true /* skipHidden */)
		if count > 3 {
			break
		}
	}

	return deriveObjectRefContextsFromCandidates(candidates, true /* includeColumn */)
}

func (c *Completer) determineQualifiedNameContext() []*objectRefContext {
	tokenIndex := c.scanner.GetIndex()
	if c.scanner.GetTokenChannel() != antlr.TokenDefaultChannel {
		// Skip to the next non-hidden token.
		c.scanner.Forward(true /* skipHidden */)
	}

	// Check if we're at a dot or identifier
	if c.scanner.GetTokenText() != "." && !isIdentifier(c.scanner.GetTokenType()) {
		// We are at the end of an incomplete identifier spec. Jump back.
		c.scanner.Backward(true /* skipHidden */)
	}

	if tokenIndex > 0 {
		// Go backward until we hit a non-identifier token to find the start of the qualified name.
		for {
			curID := isIdentifier(c.scanner.GetTokenType()) && c.scanner.GetPreviousTokenText(false /* skipHidden */) == "."
			curDOT := c.scanner.GetTokenText() == "." && isIdentifier(c.scanner.GetPreviousTokenType(false /* skipHidden */))
			if curID || curDOT {
				c.scanner.Backward(true /* skipHidden */)
				continue
			}
			break
		}
	}

	// The scanner is now on the leading identifier (or at a dot if there's no leading id).
	var candidates []string
	var temp string
	var count int
	for {
		count++
		if isIdentifier(c.scanner.GetTokenType()) {
			temp = normalizeIdentifierText(c.scanner.GetTokenText())
			candidates = append(candidates, temp)
			c.scanner.Forward(true /* skipHidden */)
		}

		// Check if we've hit the end of the identifier chain or we're past the caret
		if !c.scanner.IsTokenType(trinoparser.TrinoParserDOT_) || tokenIndex <= c.scanner.GetIndex() {
			return deriveObjectRefContextsFromCandidates(candidates, false /* includeColumn */)
		}

		// Skip the dot and move to the next token
		c.scanner.Forward(true /* skipHidden */)
		if count > 3 {
			break
		}
	}

	return deriveObjectRefContextsFromCandidates(candidates, false /* includeColumn */)
}

// deriveObjectRefContextsFromCandidates derives the object reference contexts from the candidates.
// In Trino, a qualified name has the format [catalog_name.][schema_name.][object_name]
// For example, if the candidates are ["a", "b"], the size is 2,
// and objectRefContext would be either:
// - [catalog: "a", schema: "b", object: ""] (interpreting as catalog.schema)
// - [catalog: "", schema: "a", object: "b"] (interpreting as schema.object)
func deriveObjectRefContextsFromCandidates(candidates []string, includeColumn bool) []*objectRefContext {
	var options []objectRefContextOption
	if includeColumn {
		options = append(options, withColumn())
	}
	refCtx := newObjectRefContext(options...)

	// If we have no candidates, return an empty context
	if len(candidates) == 0 {
		return []*objectRefContext{
			refCtx.clone(),
		}
	}

	var results []*objectRefContext
	switch len(candidates) {
	case 1:
		// Single identifier could be:
		// 1. A catalog name
		// 2. A schema name
		// 3. An object name
		// 4. A column name (if includeColumn is true)
		results = append(
			results,
			refCtx.clone().setCatalog(candidates[0]),
			refCtx.clone().setCatalog("").setSchema(candidates[0]),
			refCtx.clone().setCatalog("").setSchema("").setObject(candidates[0]),
		)
		if includeColumn {
			results = append(results, refCtx.clone().setCatalog("").setSchema("").setObject("").setColumn(candidates[0]))
		}
	case 2:
		// Two identifiers could be:
		// 1. catalog.schema
		// 2. schema.object
		// 3. object.column (if includeColumn is true)
		results = append(
			results,
			refCtx.clone().setCatalog(candidates[0]).setSchema(candidates[1]),
			refCtx.clone().setCatalog("").setSchema(candidates[0]).setObject(candidates[1]),
		)
		if includeColumn {
			results = append(results, refCtx.clone().setCatalog("").setSchema("").setObject(candidates[0]).setColumn(candidates[1]))
		}
	case 3:
		// Three identifiers would be catalog.schema.object
		// Or catalog.schema.column if includeColumn is true
		results = append(
			results,
			refCtx.clone().setCatalog(candidates[0]).setSchema(candidates[1]).setObject(candidates[2]),
		)
		if includeColumn {
			results = append(results, refCtx.clone().setCatalog(candidates[0]).setSchema(candidates[1]).setObject("").setColumn(candidates[2]))
		}
	case 4:
		// Four identifiers would be catalog.schema.object.column (if includeColumn is true)
		if includeColumn {
			results = append(results, refCtx.clone().setCatalog(candidates[0]).setSchema(candidates[1]).setObject(candidates[2]).setColumn(candidates[3]))
		}
	default:
		// For cases with more than 4 identifiers, ignore them
		return []*objectRefContext{refCtx.clone()}
	}

	// If no results were generated, return the default empty context
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
		// End.Line is 1-based per proto spec, convert to 0-based for comparison with caretLine
		sqlEndLine := int(sql.End.GetLine()) - 1
		sqlEndColumn := int(sql.End.GetColumn())
		if sqlEndLine > caretLine || (sqlEndLine == caretLine && sqlEndColumn >= caretOffset) {
			start = i
			if i == 0 {
				// The caret is in the first SQL statement, so we don't need to skip any SQL statements.
				break
			}
			// End.Line is 1-based per proto spec, convert to 0-based
			previousSQLEndLine := int(list[i-1].End.GetLine()) - 1
			previousSQLEndColumn := int(list[i-1].End.GetColumn())
			newCaretLine = caretLine - previousSQLEndLine + 1 // Convert to 1-based.
			if caretLine == previousSQLEndLine {
				// The caret is in the same line as the last line of the previous SQL statement.
				// End.Column is 1-based exclusive, so (End.Column - 1) gives 0-based start of next statement.
				// newCaretOffset = caretOffset - (previousSQLEndColumn - 1)
				newCaretOffset = caretOffset - previousSQLEndColumn + 1
			}
			break
		}
	}

	var buf strings.Builder
	for i := start; i < len(list); i++ {
		if _, err := buf.WriteString(list[i].Text); err != nil {
			return statement, caretLine + 1, caretOffset // Convert back to 1-based on error
		}
	}

	return buf.String(), newCaretLine, newCaretOffset
}

func notEmptySQLCount(list []base.Statement) int {
	count := 0
	for _, sql := range list {
		if !sql.Empty {
			count++
		}
	}
	return count
}

func (c *Completer) takeReferencesSnapshot() {
	if c.referenceMap == nil {
		c.referenceMap = make(map[string]bool)
	}
	for _, references := range c.referencesStack {
		c.references = append(c.references, references...)
		for _, reference := range references {
			if r, ok := reference.(*base.PhysicalTableReference); ok {
				catalog := r.Database
				if catalog == "" {
					catalog = c.defaultCatalog
				}
				schema := r.Schema
				if schema == "" {
					schema = c.defaultSchema
				}
				tableID := fmt.Sprintf("%s.%s.%s", catalog, schema, r.Table)
				c.referenceMap[tableID] = true
			}
		}
	}
}

func (c *Completer) collectLeadingTableReferences(caretIndex int) {
	c.scanner.Push()

	c.scanner.SeekIndex(0)

	level := 0
	for {
		found := c.scanner.GetTokenType() == trinoparser.TrinoParserFROM_
		for !found {
			if !c.scanner.Forward(false) || c.scanner.GetIndex() >= caretIndex {
				break
			}

			switch c.scanner.GetTokenType() {
			case trinoparser.TrinoParserLPAREN_:
				level++
				c.referencesStack = append([][]base.TableReference{{}}, c.referencesStack...)
			case trinoparser.TrinoParserRPAREN_:
				if level == 0 {
					c.scanner.PopAndRestore()
					return // We cannot go above the initial nesting level.
				}
			case trinoparser.TrinoParserFROM_:
				found = true
			default:
				// Continue scanning for other tokens
			}
		}
		if !found {
			c.scanner.PopAndRestore()
			return // No FROM clause found.
		}
		c.parseTableReferences(c.scanner.GetFollowingText())
		if c.scanner.GetTokenType() == trinoparser.TrinoParserFROM_ {
			c.scanner.Forward(false /* skipHidden */)
		}
	}
}

func (c *Completer) collectRemainingTableReferences() {
	c.scanner.Push()

	level := 0
	for {
		found := c.scanner.GetTokenType() == trinoparser.TrinoParserFROM_
		for !found {
			if !c.scanner.Forward(false /* skipHidden */) {
				break
			}

			switch c.scanner.GetTokenType() {
			case trinoparser.TrinoParserLPAREN_:
				level++
				c.referencesStack = append([][]base.TableReference{{}}, c.referencesStack...)
			case trinoparser.TrinoParserRPAREN_:
				if level > 0 {
					level--
				}

			case trinoparser.TrinoParserFROM_:
				if level == 0 {
					found = true
				}
			default:
				// Continue scanning for other tokens
			}
		}

		if !found {
			c.scanner.PopAndRestore()
			return // No more FROM clause found.
		}

		c.parseTableReferences(c.scanner.GetFollowingText())
		if c.scanner.GetTokenType() == trinoparser.TrinoParserFROM_ {
			c.scanner.Forward(false /* skipHidden */)
		}
	}
}

func (c *Completer) parseTableReferences(fromClause string) {
	input := antlr.NewInputStream(fromClause)
	lexer := trinoparser.NewTrinoLexer(input)
	tokens := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := trinoparser.NewTrinoParser(tokens)

	parser.BuildParseTrees = true
	parser.RemoveErrorListeners()
	tree := parser.Relation()
	listener := &tableRefListener{
		context:        c,
		fromClauseMode: true,
	}
	antlr.ParseTreeWalkerDefault.Walk(listener, tree)
}

type tableRefListener struct {
	*trinoparser.BaseTrinoParserListener

	context        *Completer
	fromClauseMode bool
	done           bool
	level          int
}

func (l *tableRefListener) ExitAliasedRelation(ctx *trinoparser.AliasedRelationContext) {
	if l.done {
		return
	}
	if l.level == 0 && len(l.context.referencesStack) != 0 && len(l.context.referencesStack[0]) != 0 {
		if ctx.Identifier() != nil {
			alias := unquote(ctx.Identifier().GetText())
			if physicalTable, ok := l.context.referencesStack[0][len(l.context.referencesStack[0])-1].(*base.PhysicalTableReference); ok {
				physicalTable.Alias = alias
				return
			}
			if virtualTable, ok := l.context.referencesStack[0][len(l.context.referencesStack[0])-1].(*base.VirtualTableReference); ok {
				virtualTable.Table = alias
			}
		}
	}
}

func (l *tableRefListener) ExitColumnAliases(ctx *trinoparser.ColumnAliasesContext) {
	if l.done {
		return
	}

	if l.level == 0 && len(l.context.referencesStack) != 0 && len(l.context.referencesStack[0]) != 0 {
		if virtualTable, ok := l.context.referencesStack[0][len(l.context.referencesStack[0])-1].(*base.VirtualTableReference); ok {
			var newColumns []string
			for _, column := range ctx.AllIdentifier() {
				newColumns = append(newColumns, unquote(column.GetText()))
			}
			virtualTable.Columns = newColumns
		}
	}
}

func (l *tableRefListener) ExitQualifiedName(ctx *trinoparser.QualifiedNameContext) {
	if l.done {
		return
	}

	if !l.fromClauseMode || l.level == 0 {
		reference := &base.PhysicalTableReference{}
		catalog, schema, table := normalizeQualifiedNameFallback(ctx, "", "")
		reference.Database = catalog // Using Database field for Catalog in Trino
		reference.Schema = schema
		reference.Table = table
		l.context.referencesStack[0] = append(l.context.referencesStack[0], reference)
	}
}

func (l *tableRefListener) ExitTableFunctionInvocation(_ *trinoparser.TableFunctionInvocationContext) {
	if l.done {
		return
	}

	if !l.fromClauseMode || l.level == 0 {
		reference := &base.VirtualTableReference{}
		l.context.referencesStack[0] = append(l.context.referencesStack[0], reference)
	}
}

func (l *tableRefListener) ExitSubquery(ctx *trinoparser.SubqueryContext) {
	if l.done {
		return
	}

	// In Trino, subqueries must be aliased in an aliasedRelation
	pCtx, ok := ctx.GetParent().(*trinoparser.SubqueryRelationContext)
	if !ok {
		return
	}

	// Check for grandparent context which should be an aliasedRelation
	gCtx, ok := pCtx.GetParent().(*trinoparser.AliasedRelationContext)
	if !ok {
		return
	}

	var derivedTableName string
	if gCtx.Identifier() != nil {
		derivedTableName = unquote(gCtx.Identifier().GetText())
	} else {
		// If no explicit alias, we can't reference this subquery's columns
		return
	}

	reference := &base.VirtualTableReference{
		Table: derivedTableName,
	}

	if gCtx.ColumnAliases() == nil {
		// User did not specify the column alias, we should use query span to get the column alias.
		if span, err := GetQuerySpan(
			l.context.ctx,
			base.GetQuerySpanContext{
				InstanceID:              l.context.instanceID,
				GetDatabaseMetadataFunc: l.context.metadataGetter,
				ListDatabaseNamesFunc:   l.context.catalogNamesLister,
			},
			fmt.Sprintf("SELECT * FROM (%s);", ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)),
			l.context.defaultCatalog,
			l.context.defaultSchema,
			true,
		); err == nil && span.NotFoundError == nil {
			for _, column := range span.Results {
				reference.Columns = append(reference.Columns, column.Name)
			}
		}
	} else {
		// If column aliases are specified, use them
		for _, identifier := range gCtx.ColumnAliases().AllIdentifier() {
			reference.Columns = append(reference.Columns, unquote(identifier.GetText()))
		}
	}
	l.context.referencesStack[0] = append(l.context.referencesStack[0], reference)
}

func (l *tableRefListener) EnterSubqueryRelation(*trinoparser.SubqueryRelationContext) {
	if l.done {
		return
	}

	if l.fromClauseMode {
		l.level++
	} else {
		l.context.referencesStack = append([][]base.TableReference{{}}, l.context.referencesStack...)
	}
}

func (l *tableRefListener) ExitSubqueryRelation(*trinoparser.SubqueryRelationContext) {
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

	// Extract CTEs using the cteExtractor
	extractor := &cteExtractor{
		completer: c,
	}
	input := antlr.NewInputStream(statement)
	lexer := trinoparser.NewTrinoLexer(input)
	tokens := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := trinoparser.NewTrinoParser(tokens)
	parser.BuildParseTrees = true
	parser.RemoveErrorListeners()
	tree := parser.Parse()
	antlr.ParseTreeWalkerDefault.Walk(extractor, tree)
	c.cteTables = extractor.virtualReferences
}

type cteExtractor struct {
	*trinoparser.BaseTrinoParserListener

	completer         *Completer
	handled           bool
	virtualReferences []*base.VirtualTableReference
}

func (c *cteExtractor) EnterWith(ctx *trinoparser.WithContext) {
	if c.handled {
		return
	}
	c.handled = true

	for _, namedQuery := range ctx.AllNamedQuery() {
		cteName := namedQuery.Identifier().GetText()
		if cteName == "" {
			continue
		}

		var columns []string
		if namedQuery.ColumnAliases() != nil {
			for _, columnID := range namedQuery.ColumnAliases().AllIdentifier() {
				columns = append(columns, unquote(columnID.GetText()))
			}
			c.virtualReferences = append(c.virtualReferences, &base.VirtualTableReference{
				Table:   unquote(cteName),
				Columns: columns,
			})
			continue
		}

		// If column aliases are not specified, use query span to get the column names
		cteBody := ctx.GetParser().GetTokenStream().GetTextFromRuleContext(namedQuery.Query())
		statement := fmt.Sprintf("WITH %s AS (%s) SELECT * FROM %s", cteName, cteBody, cteName)
		if span, err := GetQuerySpan(
			c.completer.ctx,
			base.GetQuerySpanContext{
				InstanceID:              c.completer.instanceID,
				GetDatabaseMetadataFunc: c.completer.metadataGetter,
				ListDatabaseNamesFunc:   c.completer.catalogNamesLister,
			},
			statement,
			c.completer.defaultCatalog,
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

func (c *Completer) fetchSelectItemAliases(ruleStack []*base.RuleContext) []string {
	canUseAliases := false
	for i := len(ruleStack) - 1; i >= 0; i-- {
		switch ruleStack[i].ID {
		case trinoparser.TrinoParserRULE_query, trinoparser.TrinoParserRULE_querySpecification:
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
			slices.Sort(result)
			return result
		case trinoparser.TrinoParserRULE_groupBy, trinoparser.TrinoParserRULE_sortItem, trinoparser.TrinoParserRULE_booleanExpression:
			// These represent ORDER BY, GROUP BY, and HAVING contexts
			canUseAliases = true
		default:
			// Continue iterating through other rule types
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
	lexer := trinoparser.NewTrinoLexer(input)
	tokens := antlr.NewCommonTokenStream(lexer, 0)
	parser := trinoparser.NewTrinoParser(tokens)

	parser.BuildParseTrees = true
	parser.RemoveErrorListeners()
	// Use the as_column_alias rule for column aliases, matching TSQL's approach
	tree := parser.As_column_alias()

	listener := &SelectAliasListener{}
	antlr.ParseTreeWalkerDefault.Walk(listener, tree)

	return listener.result
}

type SelectAliasListener struct {
	*trinoparser.BaseTrinoParserListener

	result string
}

func (l *SelectAliasListener) EnterAs_column_alias(ctx *trinoparser.As_column_aliasContext) {
	l.result = unquote(ctx.Column_alias().GetText())
}

func (c *Completer) quotedIdentifierIfNeeded(identifier string) string {
	// If we're already in a quoted context, return the identifier as-is
	if c.caretTokenIsQuoted != quotedTypeNone {
		return identifier
	}

	// In Trino, certain identifiers need to be quoted:
	// 1. If they contain special characters or spaces
	// 2. If they're case-sensitive
	// 3. If they're reserved keywords

	// Check for characters that would require quoting
	needsQuoting := false
	for i, r := range identifier {
		// First character must be a letter or underscore
		if i == 0 && !unicode.IsLetter(r) && r != '_' {
			needsQuoting = true
			break
		}

		// Other characters must be letters, numbers, or underscores
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			needsQuoting = true
			break
		}
	}

	// Quote the identifier if needed
	if needsQuoting {
		return fmt.Sprintf("\"%s\"", identifier)
	}

	return identifier
}

// Helper functions for handling Trino identifiers

// isIdentifier checks if token is one of the identifier types in Trino
func isIdentifier(tokenType int) bool {
	return tokenType == trinoparser.TrinoParserIDENTIFIER_ ||
		tokenType == trinoparser.TrinoParserQUOTED_IDENTIFIER_ ||
		tokenType == trinoparser.TrinoParserDIGIT_IDENTIFIER_ ||
		tokenType == trinoparser.TrinoParserBACKQUOTED_IDENTIFIER_
}

// normalizeIdentifierText normalizes an identifier text by unquoting it if needed
func normalizeIdentifierText(text string) string {
	return unquote(text)
}

// unquote removes quotes from identifiers if present
func unquote(text string) string {
	// Remove double quotes from quoted identifiers
	if len(text) >= 2 && text[0] == '"' && text[len(text)-1] == '"' {
		return text[1 : len(text)-1]
	}

	// Remove backticks from backticked identifiers
	if len(text) >= 2 && text[0] == '`' && text[len(text)-1] == '`' {
		return text[1 : len(text)-1]
	}

	return text
}

// normalizeQualifiedNameFallback extracts the catalog, schema, and table names from a QualifiedNameContext
// In Trino, a qualifiedName can have 1-3 parts: [catalog.]schema.table or just table
func normalizeQualifiedNameFallback(ctx *trinoparser.QualifiedNameContext, _ string, _ string) (string, string, string) {
	parts := []string{}

	// Extract all identifiers from the qualified name
	for _, identifier := range ctx.AllIdentifier() {
		parts = append(parts, unquote(identifier.GetText()))
	}

	catalog, schema, table := "", "", ""

	// Assign parts based on how many we found
	switch len(parts) {
	case 1:
		// Just a table name
		table = parts[0]
	case 2:
		// schema.table
		schema = parts[0]
		table = parts[1]
	case 3:
		// catalog.schema.table
		catalog = parts[0]
		schema = parts[1]
		table = parts[2]
	default:
		// Handle other cases (0 or more than 3 parts)
	}

	return catalog, schema, table
}

// skipHeadingSQLWithoutSemicolon skips the heading SQL statements that don't end with semicolon,
// by detecting SELECT at column 0. This is similar to TSQL's implementation.
func skipHeadingSQLWithoutSemicolon(statement string, caretLine int, caretOffset int) (string, int, int) {
	input := antlr.NewInputStream(statement)
	lexer := trinoparser.NewTrinoLexer(input)
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
		// We want to find SELECT statements that come BEFORE the caret position
		// but we should skip the one we're currently in
		if token.GetLine() > caretLine || (token.GetLine() == caretLine && token.GetColumn() >= caretOffset) {
			break
		}

		if token.GetTokenType() == trinoparser.TrinoParserSELECT_ && token.GetColumn() == 0 {
			latestSelect = token.GetTokenIndex()
			newCaretLine = caretLine - token.GetLine() + 1 // convert to 1-based
			// When we're on the same line as the SELECT, we need to adjust the offset
			// by the position where we start extracting the substring
			if token.GetLine() == caretLine {
				// The token's column is where the SELECT starts in the original string
				// We need to subtract this from the caret offset since we're extracting from this position
				newCaretOffset = caretOffset - token.GetColumn()
			} else {
				newCaretOffset = caretOffset
			}
		}
	}

	if latestSelect == 0 {
		return statement, caretLine, caretOffset
	}

	// Extract the substring starting from the SELECT token
	result := stream.GetTextFromInterval(antlr.NewInterval(latestSelect, stream.Size()))
	return result, newCaretLine, newCaretOffset
}
