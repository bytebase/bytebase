package pg

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	pg "github.com/bytebase/postgresql-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
	"github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	// globalFollowSetsByState is the global follow sets by state.
	// It is shared by all PostgreSQL completers.
	// The FollowSetsByState is the thread-safe struct.
	globalFollowSetsByState = base.NewFollowSetsByState()
)

func init() {
	base.RegisterCompleteFunc(store.Engine_POSTGRES, Completion)
	base.RegisterCompleteFunc(store.Engine_REDSHIFT, Completion)
	base.RegisterCompleteFunc(store.Engine_RISINGWAVE, Completion)
	base.RegisterCompleteFunc(store.Engine_DM, Completion)
	base.RegisterCompleteFunc(store.Engine_SNOWFLAKE, Completion)
	base.RegisterCompleteFunc(store.Engine_COCKROACHDB, Completion)
}

// Completion is the entry point of PostgreSQL code completion.
func Completion(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) ([]base.Candidate, error) {
	fmt.Printf("DEBUG: Completion entry point - statement: %s, caretLine: %d, caretOffset: %d\n", statement, caretLine, caretOffset)
	
	completer := NewStandardCompleter(ctx, cCtx, statement, caretLine, caretOffset)
	fmt.Printf("DEBUG: Created StandardCompleter with defaultDatabase: %s, defaultSchema: %s\n", 
		completer.defaultDatabase, completer.defaultSchema)
	
	result, err := completer.completion()
	if err != nil {
		fmt.Printf("DEBUG: StandardCompleter failed with error: %v\n", err)
		return nil, err
	}
	if len(result) > 0 {
		fmt.Printf("DEBUG: StandardCompleter returned %d candidates\n", len(result))
		return result, nil
	}
	fmt.Printf("DEBUG: StandardCompleter found no candidates, trying TrickyCompleter\n")

	trickyCompleter := NewTrickyCompleter(ctx, cCtx, statement, caretLine, caretOffset)
	result, err = trickyCompleter.completion()
	if err != nil {
		fmt.Printf("DEBUG: TrickyCompleter failed with error: %v\n", err)
	} else {
		fmt.Printf("DEBUG: TrickyCompleter returned %d candidates\n", len(result))
	}
	return result, err
}

func newIgnoredTokens() map[int]bool {
	return map[int]bool{
		antlr.TokenEOF:                                                 true,
		pg.PostgreSQLLexerDollar:                                       true,
		pg.PostgreSQLLexerOPEN_PAREN:                                   true,
		pg.PostgreSQLLexerCLOSE_PAREN:                                  true,
		pg.PostgreSQLLexerOPEN_BRACKET:                                 true,
		pg.PostgreSQLLexerCLOSE_BRACKET:                                true,
		pg.PostgreSQLLexerCOMMA:                                        true,
		pg.PostgreSQLLexerSEMI:                                         true,
		pg.PostgreSQLLexerCOLON:                                        true,
		pg.PostgreSQLLexerEQUAL:                                        true,
		pg.PostgreSQLLexerDOT:                                          true,
		pg.PostgreSQLLexerPLUS:                                         true,
		pg.PostgreSQLLexerMINUS:                                        true,
		pg.PostgreSQLLexerSLASH:                                        true,
		pg.PostgreSQLLexerCARET:                                        true,
		pg.PostgreSQLLexerLT:                                           true,
		pg.PostgreSQLLexerGT:                                           true,
		pg.PostgreSQLLexerLESS_LESS:                                    true,
		pg.PostgreSQLLexerGREATER_GREATER:                              true,
		pg.PostgreSQLLexerCOLON_EQUALS:                                 true,
		pg.PostgreSQLLexerLESS_EQUALS:                                  true,
		pg.PostgreSQLLexerEQUALS_GREATER:                               true,
		pg.PostgreSQLLexerGREATER_EQUALS:                               true,
		pg.PostgreSQLLexerDOT_DOT:                                      true,
		pg.PostgreSQLLexerNOT_EQUALS:                                   true,
		pg.PostgreSQLLexerTYPECAST:                                     true,
		pg.PostgreSQLLexerPERCENT:                                      true,
		pg.PostgreSQLLexerPARAM:                                        true,
		pg.PostgreSQLLexerOperator:                                     true,
		pg.PostgreSQLLexerIdentifier:                                   true,
		pg.PostgreSQLLexerQuotedIdentifier:                             true,
		pg.PostgreSQLLexerUnterminatedQuotedIdentifier:                 true,
		pg.PostgreSQLLexerInvalidQuotedIdentifier:                      true,
		pg.PostgreSQLLexerInvalidUnterminatedQuotedIdentifier:          true,
		pg.PostgreSQLLexerUnicodeQuotedIdentifier:                      true,
		pg.PostgreSQLLexerUnterminatedUnicodeQuotedIdentifier:          true,
		pg.PostgreSQLLexerInvalidUnicodeQuotedIdentifier:               true,
		pg.PostgreSQLLexerInvalidUnterminatedUnicodeQuotedIdentifier:   true,
		pg.PostgreSQLLexerStringConstant:                               true,
		pg.PostgreSQLLexerUnterminatedStringConstant:                   true,
		pg.PostgreSQLLexerUnicodeEscapeStringConstant:                  true,
		pg.PostgreSQLLexerUnterminatedUnicodeEscapeStringConstant:      true,
		pg.PostgreSQLLexerBeginDollarStringConstant:                    true,
		pg.PostgreSQLLexerBinaryStringConstant:                         true,
		pg.PostgreSQLLexerUnterminatedBinaryStringConstant:             true,
		pg.PostgreSQLLexerInvalidBinaryStringConstant:                  true,
		pg.PostgreSQLLexerInvalidUnterminatedBinaryStringConstant:      true,
		pg.PostgreSQLLexerHexadecimalStringConstant:                    true,
		pg.PostgreSQLLexerUnterminatedHexadecimalStringConstant:        true,
		pg.PostgreSQLLexerInvalidHexadecimalStringConstant:             true,
		pg.PostgreSQLLexerInvalidUnterminatedHexadecimalStringConstant: true,
		pg.PostgreSQLLexerIntegral:                                     true,
		pg.PostgreSQLLexerNumericFail:                                  true,
		pg.PostgreSQLLexerNumeric:                                      true,
	}
}

func newPreferredRules() map[int]bool {
	return map[int]bool{
		pg.PostgreSQLParserRULE_relation_expr:  true,
		pg.PostgreSQLParserRULE_qualified_name: true,
		pg.PostgreSQLParserRULE_columnref:      true,
		pg.PostgreSQLParserRULE_func_name:      true,
	}
}

func newNoSeparatorRequired() map[int]bool {
	return map[int]bool{
		pg.PostgreSQLLexerDollar:          true,
		pg.PostgreSQLLexerOPEN_PAREN:      true,
		pg.PostgreSQLLexerCLOSE_PAREN:     true,
		pg.PostgreSQLLexerOPEN_BRACKET:    true,
		pg.PostgreSQLLexerCLOSE_BRACKET:   true,
		pg.PostgreSQLLexerCOMMA:           true,
		pg.PostgreSQLLexerSEMI:            true,
		pg.PostgreSQLLexerCOLON:           true,
		pg.PostgreSQLLexerEQUAL:           true,
		pg.PostgreSQLLexerDOT:             true,
		pg.PostgreSQLLexerPLUS:            true,
		pg.PostgreSQLLexerMINUS:           true,
		pg.PostgreSQLLexerSLASH:           true,
		pg.PostgreSQLLexerCARET:           true,
		pg.PostgreSQLLexerLT:              true,
		pg.PostgreSQLLexerGT:              true,
		pg.PostgreSQLLexerLESS_LESS:       true,
		pg.PostgreSQLLexerGREATER_GREATER: true,
		pg.PostgreSQLLexerCOLON_EQUALS:    true,
		pg.PostgreSQLLexerLESS_EQUALS:     true,
		pg.PostgreSQLLexerEQUALS_GREATER:  true,
		pg.PostgreSQLLexerGREATER_EQUALS:  true,
		pg.PostgreSQLLexerDOT_DOT:         true,
		pg.PostgreSQLLexerNOT_EQUALS:      true,
		pg.PostgreSQLLexerTYPECAST:        true,
		pg.PostgreSQLLexerPERCENT:         true,
		pg.PostgreSQLLexerPARAM:           true,
	}
}

type Completer struct {
	ctx                 context.Context
	core                *base.CodeCompletionCore
	scene               base.SceneType
	parser              *pg.PostgreSQLParser
	lexer               *pg.PostgreSQLLexer
	scanner             *base.Scanner
	instanceID          string
	defaultDatabase     string
	defaultSchema       string
	getMetadata         base.GetDatabaseMetadataFunc
	listDatabaseNames   base.ListDatabaseNamesFunc
	metadataCache       map[string]*model.DatabaseMetadata
	noSeparatorRequired map[int]bool
	// referencesStack is a hierarchical stack of table references.
	// We'll update the stack when we encounter a new FROM clauses.
	referencesStack [][]base.TableReference
	// references is the flattened table references.
	// It's helpful to look up the table reference.
	references         []base.TableReference
	cteCache           map[int][]*base.VirtualTableReference
	cteTables          []*base.VirtualTableReference
	caretTokenIsQuoted bool
}

func NewTrickyCompleter(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) *Completer {
	parser, lexer, scanner := prepareTrickyParserAndScanner(statement, caretLine, caretOffset)
	// For all PostgreSQL completers, we use one global follow sets by state.
	// The FollowSetsByState is the thread-safe struct.
	core := base.NewCodeCompletionCore(
		parser,
		newIgnoredTokens(),
		newPreferredRules(),
		&globalFollowSetsByState,
		pg.PostgreSQLParserRULE_simple_select_pramary,
		pg.PostgreSQLParserRULE_select_no_parens,
		pg.PostgreSQLParserRULE_target_alias,
		pg.PostgreSQLParserRULE_with_clause,
	)
	defaultSchema := cCtx.DefaultSchema
	if defaultSchema == "" {
		defaultSchema = "public"
	}
	return &Completer{
		ctx:                 ctx,
		core:                core,
		scene:               cCtx.Scene,
		parser:              parser,
		lexer:               lexer,
		scanner:             scanner,
		instanceID:          cCtx.InstanceID,
		defaultDatabase:     cCtx.DefaultDatabase,
		defaultSchema:       defaultSchema,
		getMetadata:         cCtx.Metadata,
		metadataCache:       make(map[string]*model.DatabaseMetadata),
		noSeparatorRequired: newNoSeparatorRequired(),
		cteCache:            make(map[int][]*base.VirtualTableReference),
	}
}

func NewStandardCompleter(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) *Completer {
	parser, lexer, scanner := prepareParserAndScanner(statement, caretLine, caretOffset)
	// For all PostgreSQL completers, we use one global follow sets by state.
	// The FollowSetsByState is the thread-safe struct.
	core := base.NewCodeCompletionCore(
		parser,
		newIgnoredTokens(),
		newPreferredRules(),
		&globalFollowSetsByState,
		pg.PostgreSQLParserRULE_simple_select_pramary,
		pg.PostgreSQLParserRULE_select_no_parens,
		pg.PostgreSQLParserRULE_target_alias,
		pg.PostgreSQLParserRULE_with_clause,
	)
	defaultSchema := cCtx.DefaultSchema
	if defaultSchema == "" {
		defaultSchema = "public"
	}
	return &Completer{
		ctx:                 ctx,
		core:                core,
		scene:               cCtx.Scene,
		parser:              parser,
		lexer:               lexer,
		scanner:             scanner,
		instanceID:          cCtx.InstanceID,
		defaultDatabase:     cCtx.DefaultDatabase,
		defaultSchema:       defaultSchema,
		getMetadata:         cCtx.Metadata,
		metadataCache:       make(map[string]*model.DatabaseMetadata),
		noSeparatorRequired: newNoSeparatorRequired(),
		cteCache:            make(map[int][]*base.VirtualTableReference),
	}
}

func (c *Completer) completion() ([]base.Candidate, error) {
	fmt.Printf("DEBUG: Starting completion() method\n")
	
	// Check the caret token is quoted or not.
	// This check should be done before checking the caret token is a separator or not.
	if c.scanner.IsTokenType(pg.PostgreSQLLexerQuotedIdentifier) ||
		c.scanner.IsTokenType(pg.PostgreSQLLexerInvalidQuotedIdentifier) ||
		c.scanner.IsTokenType(pg.PostgreSQLLexerUnicodeQuotedIdentifier) {
		c.caretTokenIsQuoted = true
		fmt.Printf("DEBUG: Caret token is quoted\n")
	}

	caretIndex := c.scanner.GetIndex()
	fmt.Printf("DEBUG: Initial caretIndex: %d, tokenType: %d\n", caretIndex, c.scanner.GetTokenType())
	
	if caretIndex > 0 && !c.noSeparatorRequired[c.scanner.GetPreviousTokenType(false /* skipHidden */)] {
		prevTokenType := c.scanner.GetPreviousTokenType(false)
		caretIndex--
		fmt.Printf("DEBUG: Adjusted caretIndex: %d, prevTokenType: %d, token doesn't need separator\n", 
			caretIndex, prevTokenType)
	}
	
	c.referencesStack = append([][]base.TableReference{{}}, c.referencesStack...)
	c.parser.Reset()
	
	var context antlr.ParserRuleContext
	if c.scene == base.SceneTypeQuery {
		fmt.Printf("DEBUG: Scene is SceneTypeQuery, using Selectstmt\n")
		context = c.parser.Selectstmt()
	} else {
		fmt.Printf("DEBUG: Scene is %v, using Root\n", c.scene)
		context = c.parser.Root()
	}

	fmt.Printf("DEBUG: Collecting candidates at caretIndex: %d\n", caretIndex)
	candidates := c.core.CollectCandidates(caretIndex, context)
	fmt.Printf("DEBUG: Collected %d token types and %d rule types\n", 
		len(candidates.Tokens), len(candidates.Rules))
	
	// Print token candidates
	for token, followTokens := range candidates.Tokens {
		if token >= 0 {
			tokenName := c.parser.SymbolicNames[token]
			fmt.Printf("DEBUG: Token candidate: %s (%d) with %d follow tokens\n", 
				tokenName, token, len(followTokens))
		}
	}
	
	// Print rule candidates
	for ruleName := range candidates.Rules {
		ruleTxt := "unknown"
		if ruleName == pg.PostgreSQLParserRULE_columnref {
			ruleTxt = "columnref"
		} else if ruleName == pg.PostgreSQLParserRULE_relation_expr {
			ruleTxt = "relation_expr"
		} else if ruleName == pg.PostgreSQLParserRULE_qualified_name {
			ruleTxt = "qualified_name"
		} else if ruleName == pg.PostgreSQLParserRULE_func_name {
			ruleTxt = "func_name"
		}
		fmt.Printf("DEBUG: Rule candidate: %s (%d)\n", ruleTxt, ruleName)
		
		if ruleName == pg.PostgreSQLParserRULE_columnref {
			fmt.Printf("DEBUG: Processing columnref rule\n")
			c.collectLeadingTableReferences(caretIndex)
			c.takeReferencesSnapshot()
			c.collectRemainingTableReferences()
			c.takeReferencesSnapshot()
			fmt.Printf("DEBUG: References collected: %d\n", len(c.references))
			for i, ref := range c.references {
				if physRef, ok := ref.(*base.PhysicalTableReference); ok {
					fmt.Printf("DEBUG:   Ref[%d]: PhysicalTable(Schema=%s, Table=%s, Alias=%s)\n", 
						i, physRef.Schema, physRef.Table, physRef.Alias)
				} else if virtRef, ok := ref.(*base.VirtualTableReference); ok {
					fmt.Printf("DEBUG:   Ref[%d]: VirtualTable(Table=%s, Columns=%v)\n", 
						i, virtRef.Table, virtRef.Columns)
				}
			}
		}
	}

	fmt.Printf("DEBUG: Converting candidates\n")
	result, err := c.convertCandidates(candidates)
	if err != nil {
		fmt.Printf("DEBUG: Error converting candidates: %v\n", err)
	} else {
		fmt.Printf("DEBUG: Converted to %d final candidates\n", len(result))
	}
	return result, err
}

type CompletionMap map[string]base.Candidate

func (m CompletionMap) Insert(entry base.Candidate) {
	m[entry.String()] = entry
}

func (m CompletionMap) insertFunctions() {
	for _, name := range pg.GetBuiltinFunctions() {
		m.Insert(base.Candidate{
			Type: base.CandidateTypeFunction,
			Text: name + "()",
		})
	}
}

func (m CompletionMap) insertSchemas(c *Completer) {
	// Skip if user has specified the schema.
	if c.defaultSchema != "" && c.defaultSchema != "public" {
		return
	}
	for _, schema := range c.listAllSchemas() {
		m.Insert(base.Candidate{
			Type: base.CandidateTypeSchema,
			Text: c.quotedIdentifierIfNeeded(schema),
		})
	}
}

func (m CompletionMap) insertTables(c *Completer, schemas map[string]bool) {
	for schema := range schemas {
		if len(schema) == 0 {
			// User didn't specify the schema, we need to append cte tables.
			for _, table := range c.cteTables {
				m.Insert(base.Candidate{
					Type: base.CandidateTypeTable,
					Text: c.quotedIdentifierIfNeeded(table.Table),
				})
			}
			continue
		}
		for _, table := range c.listTables(schema) {
			m.Insert(base.Candidate{
				Type: base.CandidateTypeTable,
				Text: c.quotedIdentifierIfNeeded(table),
			})
		}
		for _, fTable := range c.listForeignTables(schema) {
			m.Insert(base.Candidate{
				Type: base.CandidateTypeForeignTable,
				Text: c.quotedIdentifierIfNeeded(fTable),
			})
		}
	}
}

func (m CompletionMap) insertViews(c *Completer, schemas map[string]bool) {
	for schema := range schemas {
		for _, view := range c.listViews(schema) {
			m.Insert(base.Candidate{
				Type: base.CandidateTypeView,
				Text: c.quotedIdentifierIfNeeded(view),
			})
		}
		for _, matView := range c.listMaterializedViews(schema) {
			m.Insert(base.Candidate{
				Type: base.CandidateTypeMaterializedView,
				Text: c.quotedIdentifierIfNeeded(matView),
			})
		}
	}
}

func (m CompletionMap) insertColumns(c *Completer, schemas, tables map[string]bool) {
	if _, exists := c.metadataCache[c.defaultDatabase]; !exists {
		_, metadata, err := c.getMetadata(c.ctx, c.instanceID, c.defaultDatabase)
		if err != nil || metadata == nil {
			return
		}
		c.metadataCache[c.defaultDatabase] = metadata
	}

	for schema := range schemas {
		if len(schema) == 0 {
			// User didn't specify the schema, we need to append cte tables.
			for _, table := range c.cteTables {
				if tables[table.Table] {
					for _, column := range table.Columns {
						m.Insert(base.Candidate{
							Type: base.CandidateTypeColumn,
							Text: c.quotedIdentifierIfNeeded(column),
						})
					}
				}
			}
			continue
		}
		schemaMeta := c.metadataCache[c.defaultDatabase].GetSchema(schema)
		if schemaMeta == nil {
			continue
		}
		for table := range tables {
			tableMeta := schemaMeta.GetTable(table)
			if tableMeta == nil {
				continue
			}
			for _, column := range tableMeta.GetColumns() {
				definition := fmt.Sprintf("%s.%s | %s", schema, table, column.Type)
				if !column.Nullable {
					definition += ", NOT NULL"
				}
				comment := column.UserComment
				m.Insert(base.Candidate{
					Type:       base.CandidateTypeColumn,
					Text:       c.quotedIdentifierIfNeeded(column.Name),
					Definition: definition,
					Comment:    comment,
				})
			}
		}
	}
}

func (m CompletionMap) insertAllColumns(c *Completer) {
	if _, exists := c.metadataCache[c.defaultDatabase]; !exists {
		_, metadata, err := c.getMetadata(c.ctx, c.instanceID, c.defaultDatabase)
		if err != nil || metadata == nil {
			return
		}
		c.metadataCache[c.defaultDatabase] = metadata
	}

	metadata := c.metadataCache[c.defaultDatabase]
	for _, schema := range metadata.ListSchemaNames() {
		schemaMeta := metadata.GetSchema(schema)
		if schemaMeta == nil {
			continue
		}
		for _, table := range schemaMeta.ListTableNames() {
			tableMeta := schemaMeta.GetTable(table)
			if tableMeta == nil {
				continue
			}
			for _, column := range tableMeta.GetColumns() {
				definition := fmt.Sprintf("%s.%s | %s", schema, table, column.Type)
				if !column.Nullable {
					definition += ", NOT NULL"
				}
				comment := column.UserComment
				m.Insert(base.Candidate{
					Type:       base.CandidateTypeColumn,
					Text:       c.quotedIdentifierIfNeeded(column.Name),
					Definition: definition,
					Comment:    comment,
				})
			}
		}
	}
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

func (c *Completer) convertCandidates(candidates *base.CandidatesCollection) ([]base.Candidate, error) {
	fmt.Printf("DEBUG: Converting candidates to completion entries\n")
	keywordEntries := make(CompletionMap)
	runtimeFunctionEntries := make(CompletionMap)
	schemaEntries := make(CompletionMap)
	tableEntries := make(CompletionMap)
	columnEntries := make(CompletionMap)
	viewEntries := make(CompletionMap)

	// Process token candidates
	fmt.Printf("DEBUG: Processing token candidates\n")
	for token, value := range candidates.Tokens {
		if token < 0 {
			continue
		}
		entry := c.parser.SymbolicNames[token]
		if strings.HasSuffix(entry, "_P") {
			entry = entry[:len(entry)-2]
		} else {
			entry = unquote(entry)
		}

		list := 0
		if len(value) > 0 {
			// For function call:
			if value[0] == pg.PostgreSQLLexerOPEN_PAREN {
				list = 1
			} else {
				for _, item := range value {
					subEntry := c.parser.SymbolicNames[item]
					if strings.HasSuffix(subEntry, "_P") {
						subEntry = subEntry[:len(subEntry)-2]
					} else {
						subEntry = unquote(subEntry)
					}
					entry += " " + subEntry
				}
			}
		}

		switch list {
		case 1:
			fmt.Printf("DEBUG: Adding function token: %s\n", strings.ToLower(entry))
			runtimeFunctionEntries.Insert(base.Candidate{
				Type: base.CandidateTypeFunction,
				Text: strings.ToLower(entry) + "()",
			})
		default:
			fmt.Printf("DEBUG: Adding keyword token: %s\n", entry)
			keywordEntries.Insert(base.Candidate{
				Type: base.CandidateTypeKeyword,
				Text: entry,
			})
		}
	}

	// Process rule candidates
	fmt.Printf("DEBUG: Processing rule candidates\n")
	for candidate := range candidates.Rules {
		c.scanner.PopAndRestore()
		c.scanner.Push()

		c.fetchCommonTableExpression(candidates.Rules[candidate])
		ruleName := "unknown"
		switch candidate {
		case pg.PostgreSQLParserRULE_func_name:
			ruleName = "func_name"
			fmt.Printf("DEBUG: Processing func_name rule\n")
			runtimeFunctionEntries.insertFunctions()
			
		case pg.PostgreSQLParserRULE_relation_expr, pg.PostgreSQLParserRULE_qualified_name:
			if candidate == pg.PostgreSQLParserRULE_relation_expr {
				ruleName = "relation_expr"
			} else {
				ruleName = "qualified_name"
			}
			fmt.Printf("DEBUG: Processing %s rule\n", ruleName)
			
			qualifier, flags := c.determineQualifiedName()
			fmt.Printf("DEBUG: determineQualifiedName returned qualifier: '%s', flags: %d\n", qualifier, flags)

			if flags&ObjectFlagsShowFirst != 0 {
				fmt.Printf("DEBUG: Adding schemas (ObjectFlagsShowFirst)\n")
				schemaEntries.insertSchemas(c)
			}

			if flags&ObjectFlagsShowSecond != 0 {
				schemas := make(map[string]bool)
				if len(qualifier) == 0 {
					schemas[c.defaultSchema] = true
					// User didn't specify the schema, we need to append cte tables.
					schemas[""] = true
					fmt.Printf("DEBUG: Using default schema: %s and empty schema\n", c.defaultSchema)
				} else {
					schemas[qualifier] = true
					fmt.Printf("DEBUG: Using specified schema: %s\n", qualifier)
				}

				fmt.Printf("DEBUG: Adding tables and views (ObjectFlagsShowSecond)\n")
				tableEntries.insertTables(c, schemas)
				viewEntries.insertViews(c, schemas)
			}
			
		case pg.PostgreSQLParserRULE_columnref:
			ruleName = "columnref"
			fmt.Printf("DEBUG: Processing columnref rule\n")
			
			schema, table, flags := c.determineColumnRef()
			fmt.Printf("DEBUG: determineColumnRef returned schema: '%s', table: '%s', flags: %d\n", 
				schema, table, flags)
				
			if flags&ObjectFlagsShowSchemas != 0 {
				fmt.Printf("DEBUG: Adding schemas (ObjectFlagsShowSchemas)\n")
				schemaEntries.insertSchemas(c)
			}

			schemas := make(map[string]bool)
			if len(schema) != 0 {
				schemas[schema] = true
				fmt.Printf("DEBUG: Using specified schema: %s\n", schema)
			} else if len(c.references) > 0 {
				fmt.Printf("DEBUG: Extracting schemas from references\n")
				for _, reference := range c.references {
					if physicalTable, ok := reference.(*base.PhysicalTableReference); ok {
						if len(physicalTable.Schema) > 0 {
							schemas[physicalTable.Schema] = true
							fmt.Printf("DEBUG: Added schema from reference: %s\n", physicalTable.Schema)
						}
					}
				}
			}

			if len(schema) == 0 {
				schemas[c.defaultSchema] = true
				// User didn't specify the schema, we need to append cte tables.
				schemas[""] = true
				fmt.Printf("DEBUG: Adding default schema: %s and empty schema\n", c.defaultSchema)
			}

			if flags&ObjectFlagsShowTables != 0 {
				fmt.Printf("DEBUG: Adding tables and views (ObjectFlagsShowTables)\n")
				tableEntries.insertTables(c, schemas)
				viewEntries.insertViews(c, schemas)

				fmt.Printf("DEBUG: Adding tables from references\n")
				for _, reference := range c.references {
					switch reference := reference.(type) {
					case *base.PhysicalTableReference:
						if len(schema) == 0 && len(reference.Schema) == 0 || schemas[reference.Schema] {
							if len(reference.Alias) == 0 {
								tableName := c.quotedIdentifierIfNeeded(reference.Table)
								fmt.Printf("DEBUG: Adding table: %s\n", tableName)
								tableEntries.Insert(base.Candidate{
									Type: base.CandidateTypeTable,
									Text: tableName,
								})
							} else {
								aliasName := c.quotedIdentifierIfNeeded(reference.Alias)
								fmt.Printf("DEBUG: Adding table alias: %s\n", aliasName)
								tableEntries.Insert(base.Candidate{
									Type: base.CandidateTypeTable,
									Text: aliasName,
								})
							}
						}
					case *base.VirtualTableReference:
						if len(schema) > 0 {
							// If the schema is specified, we should not show the virtual table.
							continue
						}
						tableName := c.quotedIdentifierIfNeeded(reference.Table)
						fmt.Printf("DEBUG: Adding virtual table: %s\n", tableName)
						tableEntries.Insert(base.Candidate{
							Type: base.CandidateTypeTable,
							Text: tableName,
						})
					}
				}
			}

			if flags&ObjectFlagsShowColumns != 0 {
				fmt.Printf("DEBUG: Adding columns (ObjectFlagsShowColumns)\n")
				if schema == table {
					schemas[c.defaultSchema] = true
					// User didn't specify the schema, we need to append cte tables.
					schemas[""] = true
					fmt.Printf("DEBUG: Schema equals table, adding default schema: %s and empty schema\n", c.defaultSchema)
				}

				tables := make(map[string]bool)
				if len(table) != 0 {
					tables[table] = true
					fmt.Printf("DEBUG: Adding table: %s\n", table)

					fmt.Printf("DEBUG: Looking for table references matching: %s\n", table)
					for _, reference := range c.references {
						switch reference := reference.(type) {
						case *base.PhysicalTableReference:
							if reference.Alias == table {
								tables[reference.Table] = true
								schemas[reference.Schema] = true
								fmt.Printf("DEBUG: Found physical table reference with alias %s -> table: %s, schema: %s\n", 
									table, reference.Table, reference.Schema)
							}
						case *base.VirtualTableReference:
							if reference.Table == table {
								fmt.Printf("DEBUG: Found virtual table reference matching table: %s\n", table)
								for _, column := range reference.Columns {
									colName := c.quotedIdentifierIfNeeded(column)
									fmt.Printf("DEBUG: Adding column from virtual table: %s\n", colName)
									columnEntries.Insert(base.Candidate{
										Type: base.CandidateTypeColumn,
										Text: colName,
									})
								}
							}
						}
					}
				} else if len(c.references) > 0 {
					fmt.Printf("DEBUG: No table specified, but have references\n")
					
					list := c.fetchSelectItemAliases(candidates.Rules[candidate])
					fmt.Printf("DEBUG: Found %d select item aliases\n", len(list))
					for _, alias := range list {
						colName := c.quotedIdentifierIfNeeded(alias)
						fmt.Printf("DEBUG: Adding column alias: %s\n", colName)
						columnEntries.Insert(base.Candidate{
							Type: base.CandidateTypeColumn,
							Text: colName,
						})
					}
					
					for _, reference := range c.references {
						switch reference := reference.(type) {
						case *base.PhysicalTableReference:
							schemas[reference.Schema] = true
							tables[reference.Table] = true
							fmt.Printf("DEBUG: Using schema: %s, table: %s from reference\n", 
								reference.Schema, reference.Table)
						case *base.VirtualTableReference:
							fmt.Printf("DEBUG: Adding columns from virtual table: %s\n", reference.Table)
							for _, column := range reference.Columns {
								colName := c.quotedIdentifierIfNeeded(column)
								fmt.Printf("DEBUG: Adding column: %s\n", colName)
								columnEntries.Insert(base.Candidate{
									Type: base.CandidateTypeColumn,
									Text: colName,
								})
							}
						}
					}
				} else {
					// No specified table, return all columns
					fmt.Printf("DEBUG: No table specified, adding all columns\n")
					columnEntries.insertAllColumns(c)
				}

				if len(tables) > 0 {
					fmt.Printf("DEBUG: Inserting columns for schemas: %v, tables: %v\n", schemas, tables)
					columnEntries.insertColumns(c, schemas, tables)
				}
			}
		default:
			fmt.Printf("DEBUG: Unknown rule type: %d\n", candidate)
		}
	}

	c.scanner.PopAndRestore()
	var result []base.Candidate
	
	// Collect all candidates
	fmt.Printf("DEBUG: Building final candidate list\n")
	fmt.Printf("DEBUG: Adding %d keyword entries\n", len(keywordEntries))
	result = append(result, keywordEntries.toSlice()...)
	
	fmt.Printf("DEBUG: Adding %d function entries\n", len(runtimeFunctionEntries))
	result = append(result, runtimeFunctionEntries.toSlice()...)
	
	fmt.Printf("DEBUG: Adding %d schema entries\n", len(schemaEntries))
	result = append(result, schemaEntries.toSlice()...)
	
	fmt.Printf("DEBUG: Adding %d table entries\n", len(tableEntries))
	result = append(result, tableEntries.toSlice()...)
	
	fmt.Printf("DEBUG: Adding %d view entries\n", len(viewEntries))
	result = append(result, viewEntries.toSlice()...)
	
	fmt.Printf("DEBUG: Adding %d column entries\n", len(columnEntries))
	result = append(result, columnEntries.toSlice()...)

	fmt.Printf("DEBUG: Final candidate count: %d\n", len(result))
	return result, nil
}

func (c *Completer) fetchCommonTableExpression(ruleStack []*base.RuleContext) {
	c.cteTables = nil
	for _, rule := range ruleStack {
		if rule.ID == pg.PostgreSQLParserRULE_select_no_parens {
			for _, pos := range rule.CTEList {
				c.cteTables = append(c.cteTables, c.extractCTETables(pos)...)
			}
		}
	}
}

func (c *Completer) extractCTETables(pos int) []*base.VirtualTableReference {
	if metadata, exists := c.cteCache[pos]; exists {
		return metadata
	}
	followingText := c.scanner.GetFollowingTextAfter(pos)
	if len(followingText) == 0 {
		return nil
	}

	input := antlr.NewInputStream(followingText)
	lexer := pg.NewPostgreSQLLexer(input)
	tokens := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := pg.NewPostgreSQLParser(tokens)

	parser.BuildParseTrees = true
	parser.RemoveErrorListeners()
	lexer.RemoveErrorListeners()
	tree := parser.With_clause()

	listener := &CTETableListener{context: c}
	antlr.ParseTreeWalkerDefault.Walk(listener, tree)

	c.cteCache[pos] = listener.tables
	return listener.tables
}

type CTETableListener struct {
	*pg.BasePostgreSQLParserListener

	context *Completer
	tables  []*base.VirtualTableReference
}

func (l *CTETableListener) EnterCommon_table_expr(ctx *pg.Common_table_exprContext) {
	table := &base.VirtualTableReference{}
	if ctx.Name() != nil {
		table.Table = normalizePostgreSQLName(ctx.Name())
	}
	if ctx.Opt_name_list() != nil {
		for _, column := range ctx.Opt_name_list().Name_list().AllName() {
			table.Columns = append(table.Columns, normalizePostgreSQLName(column))
		}
	} else {
		if span, err := GetQuerySpan(
			l.context.ctx,
			base.GetQuerySpanContext{
				InstanceID:              l.context.instanceID,
				GetDatabaseMetadataFunc: l.context.getMetadata,
				ListDatabaseNamesFunc:   l.context.listDatabaseNames,
			},
			ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.Preparablestmt()),
			l.context.defaultDatabase,
			"",
			false,
		); err == nil && span.NotFoundError == nil {
			for _, column := range span.Results {
				table.Columns = append(table.Columns, column.Name)
			}
		}
	}

	l.tables = append(l.tables, table)
}

func (c *Completer) fetchSelectItemAliases(ruleStack []*base.RuleContext) []string {
	canUseAliases := false
	for i := len(ruleStack) - 1; i >= 0; i-- {
		switch ruleStack[i].ID {
		case pg.PostgreSQLParserRULE_simple_select_pramary, pg.PostgreSQLParserRULE_select_no_parens:
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
		case pg.PostgreSQLParserRULE_opt_sort_clause, pg.PostgreSQLParserRULE_group_clause, pg.PostgreSQLParserRULE_having_clause:
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
	lexer := pg.NewPostgreSQLLexer(input)
	tokens := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := pg.NewPostgreSQLParser(tokens)

	parser.BuildParseTrees = true
	parser.RemoveErrorListeners()
	lexer.RemoveErrorListeners()
	tree := parser.Target_alias()

	listener := &TargetAliasListener{}
	antlr.ParseTreeWalkerDefault.Walk(listener, tree)
	return listener.result
}

type TargetAliasListener struct {
	*pg.BasePostgreSQLParserListener

	result string
}

func (l *TargetAliasListener) EnterTarget_alias(ctx *pg.Target_aliasContext) {
	if ctx.Identifier() != nil {
		l.result = normalizePostgreSQLIdentifier(ctx.Identifier())
	} else if ctx.Collabel() != nil {
		l.result = normalizePostgreSQLCollabel(ctx.Collabel())
	}
}

type ObjectFlags int

const (
	ObjectFlagsShowSchemas ObjectFlags = 1 << iota
	ObjectFlagsShowTables
	ObjectFlagsShowColumns
	ObjectFlagsShowFirst
	ObjectFlagsShowSecond
)

func (c *Completer) determineQualifiedName() (string, ObjectFlags) {
	fmt.Printf("DEBUG: determineQualifiedName starting\n")
	position := c.scanner.GetIndex()
	fmt.Printf("DEBUG: Initial position: %d, token: %s, type: %d\n", position, 
		c.scanner.GetTokenText(), c.scanner.GetTokenType())
	
	if c.scanner.GetTokenChannel() != 0 {
		c.scanner.Forward(true /* skipHidden */)
		fmt.Printf("DEBUG: Skipped hidden token, new position: %d, token: %s, type: %d\n", 
			c.scanner.GetIndex(), c.scanner.GetTokenText(), c.scanner.GetTokenType())
	}

	if !c.scanner.IsTokenType(pg.PostgreSQLLexerONLY) && !c.lexer.IsIdentifier(c.scanner.GetTokenType()) {
		// We are at the end of an incomplete identifier spec.
		// Jump back.
		c.scanner.Backward(true /* skipHidden */)
		fmt.Printf("DEBUG: Not identifier, moving backward to: %d, token: %s, type: %d\n", 
			c.scanner.GetIndex(), c.scanner.GetTokenText(), c.scanner.GetTokenType())
	}

	// Go left until we hit a non-identifier token.
	if position > 0 {
		if c.lexer.IsIdentifier(c.scanner.GetTokenType()) && c.scanner.GetPreviousTokenType(false /* skipHidden */) == pg.PostgreSQLLexerDOT {
			c.scanner.Backward(true /* skipHidden */)
			fmt.Printf("DEBUG: Found DOT before identifier, moving backward to: %d, token: %s, type: %d\n", 
				c.scanner.GetIndex(), c.scanner.GetTokenText(), c.scanner.GetTokenType())
		}
		if c.scanner.IsTokenType(pg.PostgreSQLLexerDOT) && c.lexer.IsIdentifier(c.scanner.GetPreviousTokenType(false /* skipHidden */)) {
			c.scanner.Backward(true /* skipHidden */)
			fmt.Printf("DEBUG: Found identifier before DOT, moving backward to: %d, token: %s, type: %d\n", 
				c.scanner.GetIndex(), c.scanner.GetTokenText(), c.scanner.GetTokenType())
		}
	}

	// The current token is on the leading identifier.
	qualifier := ""
	temp := ""
	if c.lexer.IsIdentifier(c.scanner.GetTokenType()) {
		temp = unquote(c.scanner.GetTokenText())
		fmt.Printf("DEBUG: Found identifier: '%s'\n", temp)
		c.scanner.Forward(true /* skipHidden */)
		fmt.Printf("DEBUG: Moving forward to: %d, token: %s, type: %d\n", 
			c.scanner.GetIndex(), c.scanner.GetTokenText(), c.scanner.GetTokenType())
	}

	if !c.scanner.IsTokenType(pg.PostgreSQLLexerDOT) || position <= c.scanner.GetIndex() {
		fmt.Printf("DEBUG: No DOT found or position reached, returning flags: %d\n", 
			ObjectFlagsShowFirst|ObjectFlagsShowSecond)
		return qualifier, ObjectFlagsShowFirst | ObjectFlagsShowSecond
	}

	qualifier = temp
	fmt.Printf("DEBUG: Found qualifier: '%s', returning flags: %d\n", qualifier, ObjectFlagsShowSecond)
	return qualifier, ObjectFlagsShowSecond
}

func (c *Completer) determineColumnRef() (schema, table string, flags ObjectFlags) {
	fmt.Printf("DEBUG: determineColumnRef starting\n")
	position := c.scanner.GetIndex()
	fmt.Printf("DEBUG: Initial position: %d, token: %s, type: %d\n", 
		position, c.scanner.GetTokenText(), c.scanner.GetTokenType())
		
	if c.scanner.GetTokenChannel() != 0 {
		c.scanner.Forward(true /* skipHidden */)
		fmt.Printf("DEBUG: Skipped hidden token, new position: %d, token: %s, type: %d\n", 
			c.scanner.GetIndex(), c.scanner.GetTokenText(), c.scanner.GetTokenType())
	}

	tokenType := c.scanner.GetTokenType()
	if tokenType != pg.PostgreSQLLexerDOT && !c.lexer.IsIdentifier(c.scanner.GetTokenType()) {
		// We are at the end of an incomplete identifier spec.
		// Jump back.
		c.scanner.Backward(true /* skipHidden */)
		fmt.Printf("DEBUG: Not DOT or identifier, moving backward to: %d, token: %s, type: %d\n", 
			c.scanner.GetIndex(), c.scanner.GetTokenText(), c.scanner.GetTokenType())
	}

	if position > 0 {
		if c.lexer.IsIdentifier(c.scanner.GetTokenType()) && c.scanner.GetPreviousTokenType(false /* skipHidden */) == pg.PostgreSQLLexerDOT {
			c.scanner.Backward(true /* skipHidden */)
			fmt.Printf("DEBUG: Found DOT before identifier, moving backward to: %d, token: %s, type: %d\n", 
				c.scanner.GetIndex(), c.scanner.GetTokenText(), c.scanner.GetTokenType())
		}
		if c.scanner.IsTokenType(pg.PostgreSQLLexerDOT) && c.lexer.IsIdentifier(c.scanner.GetPreviousTokenType(false /* skipHidden */)) {
			c.scanner.Backward(true /* skipHidden */)
			fmt.Printf("DEBUG: Found identifier before DOT, moving backward to: %d, token: %s, type: %d\n", 
				c.scanner.GetIndex(), c.scanner.GetTokenText(), c.scanner.GetTokenType())

			if c.scanner.GetPreviousTokenType(false /* skipHidden */) == pg.PostgreSQLLexerDOT {
				c.scanner.Backward(true /* skipHidden */)
				fmt.Printf("DEBUG: Found another DOT, moving backward to: %d, token: %s, type: %d\n", 
					c.scanner.GetIndex(), c.scanner.GetTokenText(), c.scanner.GetTokenType())
					
				if c.lexer.IsIdentifier(c.scanner.GetPreviousTokenType(false /* skipHidden */)) {
					c.scanner.Backward(true /* skipHidden */)
					fmt.Printf("DEBUG: Found another identifier, moving backward to: %d, token: %s, type: %d\n", 
						c.scanner.GetIndex(), c.scanner.GetTokenText(), c.scanner.GetTokenType())
				}
			}
		}
	}

	schema = ""
	table = ""
	temp := ""
	if c.lexer.IsIdentifier(c.scanner.GetTokenType()) {
		temp = unquote(c.scanner.GetTokenText())
		fmt.Printf("DEBUG: Found identifier: '%s'\n", temp)
		c.scanner.Forward(true /* skipHidden */)
		fmt.Printf("DEBUG: Moving forward to: %d, token: %s, type: %d\n", 
			c.scanner.GetIndex(), c.scanner.GetTokenText(), c.scanner.GetTokenType())
	}

	if !c.scanner.IsTokenType(pg.PostgreSQLLexerDOT) || position <= c.scanner.GetIndex() {
		fmt.Printf("DEBUG: No DOT found or position reached, returning all object types\n")
		return schema, table, ObjectFlagsShowSchemas | ObjectFlagsShowTables | ObjectFlagsShowColumns
	}

	c.scanner.Forward(true /* skipHidden */) // skip dot
	fmt.Printf("DEBUG: Skipped DOT, new position: %d, token: %s, type: %d\n", 
		c.scanner.GetIndex(), c.scanner.GetTokenText(), c.scanner.GetTokenType())
		
	table = temp
	schema = temp
	fmt.Printf("DEBUG: Set schema and table to: '%s'\n", temp)
	
	if c.lexer.IsIdentifier(c.scanner.GetTokenType()) {
		temp = unquote(c.scanner.GetTokenText())
		fmt.Printf("DEBUG: Found next identifier after DOT: '%s'\n", temp)
		c.scanner.Forward(true /* skipHidden */)
		fmt.Printf("DEBUG: Moving forward to: %d, token: %s, type: %d\n", 
			c.scanner.GetIndex(), c.scanner.GetTokenText(), c.scanner.GetTokenType())

		if !c.scanner.IsTokenType(pg.PostgreSQLLexerDOT) || position <= c.scanner.GetIndex() {
			fmt.Printf("DEBUG: No second DOT or position reached, returning tables and columns\n")
			return schema, table, ObjectFlagsShowTables | ObjectFlagsShowColumns
		}

		table = temp
		fmt.Printf("DEBUG: Updated table to: '%s', returning columns only\n", table)
		return schema, table, ObjectFlagsShowColumns
	}

	fmt.Printf("DEBUG: Found no identifier after DOT, returning tables and columns\n")
	return schema, table, ObjectFlagsShowTables | ObjectFlagsShowColumns
}

func unquote(s string) string {
	if len(s) < 2 {
		return s
	}

	if (s[0] == '\'' || s[0] == '"') && s[0] == s[len(s)-1] {
		return s[1 : len(s)-1]
	}
	return s
}

func (c *Completer) takeReferencesSnapshot() {
	for _, references := range c.referencesStack {
		c.references = append(c.references, references...)
	}
}

func (c *Completer) collectRemainingTableReferences() {
	fmt.Printf("DEBUG: collectRemainingTableReferences starting\n")
	c.scanner.Push()
	fmt.Printf("DEBUG: Starting at position: %d, token: %s, type: %d\n", 
		c.scanner.GetIndex(), c.scanner.GetTokenText(), c.scanner.GetTokenType())

	level := 0
	for {
		found := c.scanner.GetTokenType() == pg.PostgreSQLLexerFROM
		if found {
			fmt.Printf("DEBUG: Found FROM token at beginning\n")
		}
		
		for !found {
			if !c.scanner.Forward(false /* skipHidden */) {
				fmt.Printf("DEBUG: Reached end of tokens\n")
				break
			}

			switch c.scanner.GetTokenType() {
			case pg.PostgreSQLLexerOPEN_PAREN:
				level++
				fmt.Printf("DEBUG: Found OPEN_PAREN, increasing level to %d\n", level)
			case pg.PostgreSQLLexerCLOSE_PAREN:
				if level > 0 {
					level--
					fmt.Printf("DEBUG: Found CLOSE_PAREN, decreasing level to %d\n", level)
				}
			case pg.PostgreSQLLexerFROM:
				// Open and close parenthesis don't need to match, if we come from within a subquery.
				if level == 0 {
					found = true
					fmt.Printf("DEBUG: Found FROM token at level 0, position: %d\n", c.scanner.GetIndex())
				} else {
					fmt.Printf("DEBUG: Found FROM token at level %d (ignoring)\n", level)
				}
			}
		}

		if !found {
			fmt.Printf("DEBUG: No FROM clause found, exiting\n")
			c.scanner.PopAndRestore()
			return // No more FROM clauses found.
		}

		fmt.Printf("DEBUG: Parsing table references from following text\n")
		followingText := c.scanner.GetFollowingText()
		fmt.Printf("DEBUG: Following text: '%s'\n", truncateString(followingText, 50))
		c.parseTableReferences(followingText)
		
		if c.scanner.GetTokenType() == pg.PostgreSQLLexerFROM {
			c.scanner.Forward(false /* skipHidden */)
			fmt.Printf("DEBUG: Moving past FROM token to: %d, token: %s\n", 
				c.scanner.GetIndex(), c.scanner.GetTokenText())
		}
	}
}

func (c *Completer) collectLeadingTableReferences(caretIndex int) {
	fmt.Printf("DEBUG: collectLeadingTableReferences starting, caretIndex: %d\n", caretIndex)
	c.scanner.Push()

	c.scanner.SeekIndex(0)
	fmt.Printf("DEBUG: Seeking to index 0, token: %s, type: %d\n", 
		c.scanner.GetTokenText(), c.scanner.GetTokenType())

	level := 0
	for {
		found := c.scanner.GetTokenType() == pg.PostgreSQLLexerFROM
		if found {
			fmt.Printf("DEBUG: Found FROM token at beginning\n")
		}
		
		for !found {
			if !c.scanner.Forward(false /* skipHidden */) || c.scanner.GetIndex() >= caretIndex {
				if !c.scanner.Forward(false) {
					fmt.Printf("DEBUG: Reached end of tokens\n")
				} else if c.scanner.GetIndex() >= caretIndex {
					fmt.Printf("DEBUG: Reached caret position: %d\n", c.scanner.GetIndex())
				}
				break
			}

			switch c.scanner.GetTokenType() {
			case pg.PostgreSQLLexerOPEN_PAREN:
				level++
				c.referencesStack = append([][]base.TableReference{{}}, c.referencesStack...)
				fmt.Printf("DEBUG: Found OPEN_PAREN, increasing level to %d, stack size: %d\n", 
					level, len(c.referencesStack))
			case pg.PostgreSQLLexerCLOSE_PAREN:
				if level == 0 {
					fmt.Printf("DEBUG: Found CLOSE_PAREN at level 0, exiting\n")
					c.scanner.PopAndRestore()
					return // We cannot go above the initial nesting level.
				}

				level--
				c.referencesStack = c.referencesStack[1:]
				fmt.Printf("DEBUG: Found CLOSE_PAREN, decreasing level to %d, stack size: %d\n", 
					level, len(c.referencesStack))
			case pg.PostgreSQLLexerFROM:
				found = true
				fmt.Printf("DEBUG: Found FROM token at position: %d\n", c.scanner.GetIndex())
			}
		}

		if !found {
			fmt.Printf("DEBUG: No FROM clause found, exiting\n")
			c.scanner.PopAndRestore()
			return // No more FROM clauses found.
		}

		fmt.Printf("DEBUG: Parsing table references from following text\n")
		followingText := c.scanner.GetFollowingText()
		fmt.Printf("DEBUG: Following text: '%s'\n", truncateString(followingText, 50))
		c.parseTableReferences(followingText)
		
		if c.scanner.GetTokenType() == pg.PostgreSQLLexerFROM {
			c.scanner.Forward(false /* skipHidden */)
			fmt.Printf("DEBUG: Moving past FROM token to: %d, token: %s\n", 
				c.scanner.GetIndex(), c.scanner.GetTokenText())
		}
	}
}

// Helper to truncate long strings for debugging
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func (c *Completer) parseTableReferences(fromClause string) {
	fmt.Printf("DEBUG: parseTableReferences starting\n")
	input := antlr.NewInputStream(fromClause)
	lexer := pg.NewPostgreSQLLexer(input)
	tokens := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := pg.NewPostgreSQLParser(tokens)

	parser.BuildParseTrees = true
	parser.RemoveErrorListeners()
	lexer.RemoveErrorListeners()
	tree := parser.From_clause()

	listener := &TableRefListener{
		context: c,
	}
	fmt.Printf("DEBUG: Walking parse tree for FROM clause\n")
	antlr.ParseTreeWalkerDefault.Walk(listener, tree)
	fmt.Printf("DEBUG: Finished walking parse tree\n")
}

type TableRefListener struct {
	*pg.BasePostgreSQLParserListener

	context *Completer
	level   int
}

func (l *TableRefListener) EnterTable_ref(ctx *pg.Table_refContext) {
	fmt.Printf("DEBUG: TableRefListener.EnterTable_ref, level: %d\n", l.level)
	
	if _, ok := ctx.GetParent().(*pg.Table_refContext); ok {
		// if the table reference is nested, we should not process it.
		l.level++
		fmt.Printf("DEBUG: Nested table reference, increasing level to: %d\n", l.level)
	}

	if l.level == 0 {
		switch {
		case ctx.Relation_expr() != nil:
			fmt.Printf("DEBUG: Processing Relation_expr\n")
			var reference base.TableReference
			physicalReference := &base.PhysicalTableReference{}
			// We should use the physical reference as the default reference.
			reference = physicalReference
			
			list := NormalizePostgreSQLQualifiedName(ctx.Relation_expr().Qualified_name())
			fmt.Printf("DEBUG: Qualified name parts: %v\n", list)
			
			switch len(list) {
			case 1:
				physicalReference.Table = list[0]
				fmt.Printf("DEBUG: Found single-part name: table=%s\n", list[0])
			case 2:
				physicalReference.Schema = list[0]
				physicalReference.Table = list[1]
				fmt.Printf("DEBUG: Found two-part name: schema=%s, table=%s\n", list[0], list[1])
			case 3:
				physicalReference.Database = list[0]
				physicalReference.Schema = list[1]
				physicalReference.Table = list[2]
				fmt.Printf("DEBUG: Found three-part name: db=%s, schema=%s, table=%s\n", 
					list[0], list[1], list[2])
			default:
				fmt.Printf("DEBUG: Unexpected qualified name parts: %v\n", list)
				return
			}

			if ctx.Opt_alias_clause() != nil {
				tableAlias, columnAlias := normalizeTableAlias(ctx.Opt_alias_clause())
				fmt.Printf("DEBUG: Found alias: table=%s, columns=%v\n", tableAlias, columnAlias)
				
				if len(columnAlias) > 0 {
					virtualReference := &base.VirtualTableReference{
						Table:   tableAlias,
						Columns: columnAlias,
					}
					fmt.Printf("DEBUG: Using virtual reference for columns\n")
					// If the table alias has the column alias, we should use the virtual reference.
					reference = virtualReference
				} else {
					physicalReference.Alias = tableAlias
					fmt.Printf("DEBUG: Using physical reference with alias\n")
				}
			}

			l.context.referencesStack[0] = append(l.context.referencesStack[0], reference)
			fmt.Printf("DEBUG: Added reference to stack, stack length: %d\n", 
				len(l.context.referencesStack[0]))
				
		case ctx.Select_with_parens() != nil:
			fmt.Printf("DEBUG: Processing Select_with_parens\n")
			if ctx.Opt_alias_clause() != nil {
				virtualReference := &base.VirtualTableReference{}
				tableAlias, columnAlias := normalizeTableAlias(ctx.Opt_alias_clause())
				virtualReference.Table = tableAlias
				fmt.Printf("DEBUG: Found alias for subquery: table=%s, explicit columns=%v\n", 
					tableAlias, columnAlias)
					
				if len(columnAlias) > 0 {
					virtualReference.Columns = columnAlias
					fmt.Printf("DEBUG: Using explicit column aliases\n")
				} else {
					fmt.Printf("DEBUG: Trying to get columns from GetQuerySpan\n")
					subquery := ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.Select_with_parens())
					fullQuery := fmt.Sprintf("SELECT * FROM %s AS %s;", subquery, tableAlias)
					fmt.Printf("DEBUG: Query for GetQuerySpan: %s\n", truncateString(fullQuery, 50))
					
					if span, err := GetQuerySpan(
						l.context.ctx,
						base.GetQuerySpanContext{
							InstanceID:              l.context.instanceID,
							GetDatabaseMetadataFunc: l.context.getMetadata,
							ListDatabaseNamesFunc:   l.context.listDatabaseNames,
						},
						fullQuery,
						l.context.defaultDatabase,
						"",
						false,
					); err == nil && span.NotFoundError == nil {
						fmt.Printf("DEBUG: GetQuerySpan succeeded, found %d columns\n", len(span.Results))
						for _, column := range span.Results {
							virtualReference.Columns = append(virtualReference.Columns, column.Name)
							fmt.Printf("DEBUG: Found column: %s\n", column.Name)
						}
					} else {
						fmt.Printf("DEBUG: GetQuerySpan failed: err=%v, notFound=%v\n", 
							err, span != nil && span.NotFoundError != nil)
					}
				}

				l.context.referencesStack[0] = append(l.context.referencesStack[0], virtualReference)
				fmt.Printf("DEBUG: Added virtual reference to stack, stack length: %d\n", 
					len(l.context.referencesStack[0]))
			}
			
		case ctx.OPEN_PAREN() != nil:
			fmt.Printf("DEBUG: Processing OPEN_PAREN\n")
			if ctx.Opt_alias_clause() != nil {
				virtualReference := &base.VirtualTableReference{}
				tableAlias, columnAlias := normalizeTableAlias(ctx.Opt_alias_clause())
				virtualReference.Table = tableAlias
				fmt.Printf("DEBUG: Found alias for parenthesized expression: table=%s, columns=%v\n", 
					tableAlias, columnAlias)
					
				if len(columnAlias) > 0 {
					virtualReference.Columns = columnAlias
					fmt.Printf("DEBUG: Using explicit column aliases\n")
				}

				l.context.referencesStack[0] = append(l.context.referencesStack[0], virtualReference)
				fmt.Printf("DEBUG: Added virtual reference to stack, stack length: %d\n", 
					len(l.context.referencesStack[0]))
			}
		}
	}
}

func (l *TableRefListener) ExitTable_ref(ctx *pg.Table_refContext) {
	if _, ok := ctx.GetParent().(*pg.Table_refContext); ok {
		l.level--
		fmt.Printf("DEBUG: Exiting nested table reference, decreasing level to: %d\n", l.level)
	}
}

func (l *TableRefListener) EnterSelect_with_parens(_ *pg.Select_with_parensContext) {
	l.level++
	fmt.Printf("DEBUG: Entering select_with_parens, increasing level to: %d\n", l.level)
}

func (l *TableRefListener) ExitSelect_with_parens(_ *pg.Select_with_parensContext) {
	l.level--
	fmt.Printf("DEBUG: Exiting select_with_parens, decreasing level to: %d\n", l.level)
}

func normalizeTableAlias(ctx pg.IOpt_alias_clauseContext) (string, []string) {
	if ctx == nil || ctx.Table_alias_clause() == nil {
		return "", nil
	}

	tableAlias := ""
	aliasClause := ctx.Table_alias_clause()
	if aliasClause.Table_alias() != nil {
		tableAlias = normalizePostgreSQLTableAlias(aliasClause.Table_alias())
	}

	var columnAliases []string
	if aliasClause.Name_list() != nil {
		columnAliases = append(columnAliases, normalizePostgreSQLNameList(aliasClause.Name_list())...)
	}

	return tableAlias, columnAliases
}
func prepareTrickyParserAndScanner(statement string, caretLine int, caretOffset int) (*pg.PostgreSQLParser, *pg.PostgreSQLLexer, *base.Scanner) {
	statement, caretLine, caretOffset = skipHeadingSQLs(statement, caretLine, caretOffset)
	statement, caretLine, caretOffset = skipHeadingSQLWithoutSemicolon(statement, caretLine, caretOffset)
	input := antlr.NewInputStream(statement)
	lexer := pg.NewPostgreSQLLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := pg.NewPostgreSQLParser(stream)
	parser.RemoveErrorListeners()
	lexer.RemoveErrorListeners()
	scanner := base.NewScanner(stream, true /* fillInput */)
	scanner.SeekPosition(caretLine, caretOffset)
	scanner.Push()
	return parser, lexer, scanner
}

func prepareParserAndScanner(statement string, caretLine int, caretOffset int) (*pg.PostgreSQLParser, *pg.PostgreSQLLexer, *base.Scanner) {
	statement, caretLine, caretOffset = skipHeadingSQLs(statement, caretLine, caretOffset)
	input := antlr.NewInputStream(statement)
	lexer := pg.NewPostgreSQLLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := pg.NewPostgreSQLParser(stream)
	parser.RemoveErrorListeners()
	lexer.RemoveErrorListeners()
	scanner := base.NewScanner(stream, true /* fillInput */)
	scanner.SeekPosition(caretLine, caretOffset)
	scanner.Push()
	return parser, lexer, scanner
}

// caretLine is 1-based and caretOffset is 0-based.
func skipHeadingSQLs(statement string, caretLine int, caretOffset int) (string, int, int) {
	newCaretLine, newCaretOffset := caretLine, caretOffset
	list, err := SplitSQL(statement)
	if err != nil || len(base.FilterEmptySQL(list)) <= 1 {
		return statement, caretLine, caretOffset
	}

	caretLine-- // Convert caretLine to 0-based.

	start := 0
	for i, sql := range list {
		if sql.LastLine > caretLine || (sql.LastLine == caretLine && sql.LastColumn >= caretOffset) {
			start = i
			if i == 0 {
				// If the caret is in the first SQL statement, we should not skip any SQL statements.
				break
			}
			newCaretLine = caretLine - list[i-1].LastLine + 1 // Convert to 1-based.
			if caretLine == list[i-1].LastLine {
				// The caret is in the same line as the last line of the previous SQL statement.
				// We need to adjust the caret offset.
				newCaretOffset = caretOffset - list[i-1].LastColumn - 1 // Convert to 0-based.
			}
			break
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

// caretLine is 1-based and caretOffset is 0-based.
func skipHeadingSQLWithoutSemicolon(statement string, caretLine int, caretOffset int) (string, int, int) {
	input := antlr.NewInputStream(statement)
	lexer := pg.NewPostgreSQLLexer(input)
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
		if token.GetTokenType() == pg.PostgreSQLLexerSELECT && token.GetColumn() == 0 {
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

func (c *Completer) listAllSchemas() []string {
	if _, exists := c.metadataCache[c.defaultDatabase]; !exists {
		_, metadata, err := c.getMetadata(c.ctx, c.instanceID, c.defaultDatabase)
		if err != nil || metadata == nil {
			return nil
		}
		c.metadataCache[c.defaultDatabase] = metadata
	}

	return c.metadataCache[c.defaultDatabase].ListSchemaNames()
}

func (c *Completer) listTables(schema string) []string {
	if _, exists := c.metadataCache[c.defaultDatabase]; !exists {
		_, metadata, err := c.getMetadata(c.ctx, c.instanceID, c.defaultDatabase)
		if err != nil || metadata == nil {
			return nil
		}
		c.metadataCache[c.defaultDatabase] = metadata
	}

	schemaMeta := c.metadataCache[c.defaultDatabase].GetSchema(schema)
	if schemaMeta == nil {
		return nil
	}
	return schemaMeta.ListTableNames()
}

func (c *Completer) listForeignTables(schema string) []string {
	if _, exists := c.metadataCache[c.defaultDatabase]; !exists {
		_, metadata, err := c.getMetadata(c.ctx, c.instanceID, c.defaultDatabase)
		if err != nil || metadata == nil {
			return nil
		}
		c.metadataCache[c.defaultDatabase] = metadata
	}

	schemaMeta := c.metadataCache[c.defaultDatabase].GetSchema(schema)
	if schemaMeta == nil {
		return nil
	}
	return schemaMeta.ListForeignTableNames()
}

func (c *Completer) listMaterializedViews(schema string) []string {
	if _, exists := c.metadataCache[c.defaultDatabase]; !exists {
		_, metadata, err := c.getMetadata(c.ctx, c.instanceID, c.defaultDatabase)
		if err != nil || metadata == nil {
			return nil
		}
		c.metadataCache[c.defaultDatabase] = metadata
	}

	schemaMeta := c.metadataCache[c.defaultDatabase].GetSchema(schema)
	if schemaMeta == nil {
		return nil
	}
	return schemaMeta.ListMaterializedViewNames()
}

func (c *Completer) listViews(schema string) []string {
	if _, exists := c.metadataCache[c.defaultDatabase]; !exists {
		_, metadata, err := c.getMetadata(c.ctx, c.instanceID, c.defaultDatabase)
		if err != nil || metadata == nil {
			return nil
		}
		c.metadataCache[c.defaultDatabase] = metadata
	}

	schemaMeta := c.metadataCache[c.defaultDatabase].GetSchema(schema)
	if schemaMeta == nil {
		return nil
	}
	return schemaMeta.ListViewNames()
}

func (c *Completer) quotedIdentifierIfNeeded(s string) string {
	if c.caretTokenIsQuoted {
		return s
	}
	if strings.ToLower(s) != s {
		return fmt.Sprintf(`"%s"`, s)
	}
	if c.lexer.IsReservedKeyword(strings.ToUpper(s)) {
		return fmt.Sprintf(`"%s"`, s)
	}
	return s
}
