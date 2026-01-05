package redshift

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/redshift"

	"github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/pg"
	"github.com/bytebase/bytebase/backend/store/model"
)

var (
	// globalFollowSetsByState is the global follow sets by state.
	// It is shared by all Redshift completers.
	// The FollowSetsByState is the thread-safe struct.
	globalFollowSetsByState = base.NewFollowSetsByState()
)

func init() {
	base.RegisterCompleteFunc(store.Engine_REDSHIFT, Completion)
}

// Completion is the entry point of Redshift code completion.
func Completion(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) ([]base.Candidate, error) {
	completer := NewStandardCompleter(ctx, cCtx, statement, caretLine, caretOffset)
	result, err := completer.completion()
	if err != nil {
		return nil, err
	}
	if len(result) > 0 {
		return result, nil
	}

	trickyCompleter := NewTrickyCompleter(ctx, cCtx, statement, caretLine, caretOffset)
	return trickyCompleter.completion()
}

func newIgnoredTokens() map[int]bool {
	return map[int]bool{
		antlr.TokenEOF:                                                   true,
		parser.RedshiftLexerDollar:                                       true,
		parser.RedshiftLexerOPEN_PAREN:                                   true,
		parser.RedshiftLexerCLOSE_PAREN:                                  true,
		parser.RedshiftLexerOPEN_BRACKET:                                 true,
		parser.RedshiftLexerCLOSE_BRACKET:                                true,
		parser.RedshiftLexerCOMMA:                                        true,
		parser.RedshiftLexerSEMI:                                         true,
		parser.RedshiftLexerCOLON:                                        true,
		parser.RedshiftLexerEQUAL:                                        true,
		parser.RedshiftLexerDOT:                                          true,
		parser.RedshiftLexerPLUS:                                         true,
		parser.RedshiftLexerMINUS:                                        true,
		parser.RedshiftLexerSLASH:                                        true,
		parser.RedshiftLexerCARET:                                        true,
		parser.RedshiftLexerLT:                                           true,
		parser.RedshiftLexerGT:                                           true,
		parser.RedshiftLexerLESS_LESS:                                    true,
		parser.RedshiftLexerGREATER_GREATER:                              true,
		parser.RedshiftLexerCOLON_EQUALS:                                 true,
		parser.RedshiftLexerLESS_EQUALS:                                  true,
		parser.RedshiftLexerEQUALS_GREATER:                               true,
		parser.RedshiftLexerGREATER_EQUALS:                               true,
		parser.RedshiftLexerDOT_DOT:                                      true,
		parser.RedshiftLexerNOT_EQUALS:                                   true,
		parser.RedshiftLexerTYPECAST:                                     true,
		parser.RedshiftLexerPERCENT:                                      true,
		parser.RedshiftLexerPARAM:                                        true,
		parser.RedshiftLexerOperator:                                     true,
		parser.RedshiftLexerIdentifier:                                   true,
		parser.RedshiftLexerQuotedIdentifier:                             true,
		parser.RedshiftLexerUnterminatedQuotedIdentifier:                 true,
		parser.RedshiftLexerInvalidQuotedIdentifier:                      true,
		parser.RedshiftLexerInvalidUnterminatedQuotedIdentifier:          true,
		parser.RedshiftLexerUnicodeQuotedIdentifier:                      true,
		parser.RedshiftLexerUnterminatedUnicodeQuotedIdentifier:          true,
		parser.RedshiftLexerInvalidUnicodeQuotedIdentifier:               true,
		parser.RedshiftLexerInvalidUnterminatedUnicodeQuotedIdentifier:   true,
		parser.RedshiftLexerStringConstant:                               true,
		parser.RedshiftLexerUnterminatedStringConstant:                   true,
		parser.RedshiftLexerUnicodeEscapeStringConstant:                  true,
		parser.RedshiftLexerUnterminatedUnicodeEscapeStringConstant:      true,
		parser.RedshiftLexerBeginDollarStringConstant:                    true,
		parser.RedshiftLexerBinaryStringConstant:                         true,
		parser.RedshiftLexerUnterminatedBinaryStringConstant:             true,
		parser.RedshiftLexerInvalidBinaryStringConstant:                  true,
		parser.RedshiftLexerInvalidUnterminatedBinaryStringConstant:      true,
		parser.RedshiftLexerHexadecimalStringConstant:                    true,
		parser.RedshiftLexerUnterminatedHexadecimalStringConstant:        true,
		parser.RedshiftLexerInvalidHexadecimalStringConstant:             true,
		parser.RedshiftLexerInvalidUnterminatedHexadecimalStringConstant: true,
		parser.RedshiftLexerIntegral:                                     true,
		parser.RedshiftLexerNumericFail:                                  true,
		parser.RedshiftLexerNumeric:                                      true,
	}
}

func newPreferredRules() map[int]bool {
	return map[int]bool{
		parser.RedshiftParserRULE_relation_expr:  true,
		parser.RedshiftParserRULE_qualified_name: true,
		parser.RedshiftParserRULE_columnref:      true,
		parser.RedshiftParserRULE_func_name:      true,
	}
}

func newNoSeparatorRequired() map[int]bool {
	return map[int]bool{
		parser.RedshiftLexerDollar:          true,
		parser.RedshiftLexerOPEN_PAREN:      true,
		parser.RedshiftLexerCLOSE_PAREN:     true,
		parser.RedshiftLexerOPEN_BRACKET:    true,
		parser.RedshiftLexerCLOSE_BRACKET:   true,
		parser.RedshiftLexerCOMMA:           true,
		parser.RedshiftLexerSEMI:            true,
		parser.RedshiftLexerCOLON:           true,
		parser.RedshiftLexerEQUAL:           true,
		parser.RedshiftLexerDOT:             true,
		parser.RedshiftLexerPLUS:            true,
		parser.RedshiftLexerMINUS:           true,
		parser.RedshiftLexerSLASH:           true,
		parser.RedshiftLexerCARET:           true,
		parser.RedshiftLexerLT:              true,
		parser.RedshiftLexerGT:              true,
		parser.RedshiftLexerLESS_LESS:       true,
		parser.RedshiftLexerGREATER_GREATER: true,
		parser.RedshiftLexerCOLON_EQUALS:    true,
		parser.RedshiftLexerLESS_EQUALS:     true,
		parser.RedshiftLexerEQUALS_GREATER:  true,
		parser.RedshiftLexerGREATER_EQUALS:  true,
		parser.RedshiftLexerDOT_DOT:         true,
		parser.RedshiftLexerNOT_EQUALS:      true,
		parser.RedshiftLexerTYPECAST:        true,
		parser.RedshiftLexerPERCENT:         true,
		parser.RedshiftLexerPARAM:           true,
	}
}

type Completer struct {
	ctx                 context.Context
	core                *base.CodeCompletionCore
	scene               base.SceneType
	parser              *parser.RedshiftParser
	lexer               *parser.RedshiftLexer
	scanner             *base.Scanner
	instanceID          string
	defaultDatabase     string
	defaultSchema       string
	schemaNotSelected   bool // true if user didn't explicitly select a schema
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
	p, lexer, scanner := prepareTrickyParserAndScanner(statement, caretLine, caretOffset)
	// For all Redshift completers, we use one global follow sets by state.
	// The FollowSetsByState is the thread-safe struct.
	core := base.NewCodeCompletionCore(
		ctx,
		p,
		newIgnoredTokens(),
		newPreferredRules(),
		&globalFollowSetsByState,
		parser.RedshiftParserRULE_simple_select_pramary,
		parser.RedshiftParserRULE_select_no_parens,
		parser.RedshiftParserRULE_target_alias,
		parser.RedshiftParserRULE_with_clause,
	)
	defaultSchema := cCtx.DefaultSchema
	schemaNotSelected := defaultSchema == ""
	if defaultSchema == "" {
		defaultSchema = "public"
	}
	return &Completer{
		ctx:                 ctx,
		core:                core,
		scene:               cCtx.Scene,
		parser:              p,
		lexer:               lexer,
		scanner:             scanner,
		instanceID:          cCtx.InstanceID,
		defaultDatabase:     cCtx.DefaultDatabase,
		defaultSchema:       defaultSchema,
		schemaNotSelected:   schemaNotSelected,
		getMetadata:         cCtx.Metadata,
		metadataCache:       make(map[string]*model.DatabaseMetadata),
		noSeparatorRequired: newNoSeparatorRequired(),
		cteCache:            make(map[int][]*base.VirtualTableReference),
	}
}

func NewStandardCompleter(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) *Completer {
	p, lexer, scanner := prepareParserAndScanner(statement, caretLine, caretOffset)
	// For all Redshift completers, we use one global follow sets by state.
	// The FollowSetsByState is the thread-safe struct.
	core := base.NewCodeCompletionCore(
		ctx,
		p,
		newIgnoredTokens(),
		newPreferredRules(),
		&globalFollowSetsByState,
		parser.RedshiftParserRULE_simple_select_pramary,
		parser.RedshiftParserRULE_select_no_parens,
		parser.RedshiftParserRULE_target_alias,
		parser.RedshiftParserRULE_with_clause,
	)
	defaultSchema := cCtx.DefaultSchema
	schemaNotSelected := defaultSchema == ""
	if defaultSchema == "" {
		defaultSchema = "public"
	}
	return &Completer{
		ctx:                 ctx,
		core:                core,
		scene:               cCtx.Scene,
		parser:              p,
		lexer:               lexer,
		scanner:             scanner,
		instanceID:          cCtx.InstanceID,
		defaultDatabase:     cCtx.DefaultDatabase,
		defaultSchema:       defaultSchema,
		schemaNotSelected:   schemaNotSelected,
		getMetadata:         cCtx.Metadata,
		metadataCache:       make(map[string]*model.DatabaseMetadata),
		noSeparatorRequired: newNoSeparatorRequired(),
		cteCache:            make(map[int][]*base.VirtualTableReference),
	}
}

func (c *Completer) completion() ([]base.Candidate, error) {
	// Check the caret token is quoted or not.
	// This check should be done before checking the caret token is a separator or not.
	if c.scanner.IsTokenType(parser.RedshiftLexerQuotedIdentifier) ||
		c.scanner.IsTokenType(parser.RedshiftLexerInvalidQuotedIdentifier) ||
		c.scanner.IsTokenType(parser.RedshiftLexerUnicodeQuotedIdentifier) {
		c.caretTokenIsQuoted = true
	}

	caretIndex := c.scanner.GetIndex()
	if caretIndex > 0 && !c.noSeparatorRequired[c.scanner.GetPreviousTokenType(false /* skipHidden */)] {
		caretIndex--
	}
	c.referencesStack = append([][]base.TableReference{{}}, c.referencesStack...)
	c.parser.Reset()
	var context antlr.ParserRuleContext
	if c.scene == base.SceneTypeQuery {
		context = c.parser.Selectstmt()
	} else {
		context = c.parser.Root()
	}

	candidates := c.core.CollectCandidates(caretIndex, context)

	for ruleName := range candidates.Rules {
		if ruleName == parser.RedshiftParserRULE_columnref {
			c.collectLeadingTableReferences(caretIndex)
			c.takeReferencesSnapshot()
			c.collectRemainingTableReferences()
			c.takeReferencesSnapshot()
		}
	}

	return c.convertCandidates(candidates)
}

type CompletionMap map[string]base.Candidate

func (m CompletionMap) Insert(entry base.Candidate) {
	m[entry.String()] = entry
}

func (m CompletionMap) insertFunctions() {
	// TODO: Add Redshift-specific functions
	// For now, use a basic set of common functions
	commonFunctions := []string{
		"count", "sum", "avg", "min", "max", "abs", "ceil", "floor", "round",
		"upper", "lower", "trim", "ltrim", "rtrim", "length", "substring",
		"current_date", "current_timestamp", "date_part", "extract",
		"coalesce", "nullif", "cast", "convert",
		"row_number", "rank", "dense_rank", "lead", "lag",
		"listagg", "median", "percentile_cont", "approximate_count_distinct",
	}
	for _, name := range commonFunctions {
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

func (m CompletionMap) insertTablesWithPrefix(c *Completer, schemas map[string]bool, includeSchemaPrefix bool) {
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
			text := c.quotedIdentifierIfNeeded(table)
			// Only add schema prefix if user didn't select a schema and the table is not in the default schema
			if includeSchemaPrefix && schema != c.defaultSchema {
				text = c.quotedIdentifierIfNeeded(schema) + "." + text
			}
			m.Insert(base.Candidate{
				Type: base.CandidateTypeTable,
				Text: text,
			})
		}
		for _, fTable := range c.listForeignTables(schema) {
			text := c.quotedIdentifierIfNeeded(fTable)
			// Only add schema prefix if user didn't select a schema and the table is not in the default schema
			if includeSchemaPrefix && schema != c.defaultSchema {
				text = c.quotedIdentifierIfNeeded(schema) + "." + text
			}
			m.Insert(base.Candidate{
				Type: base.CandidateTypeForeignTable,
				Text: text,
			})
		}
	}
}

func (m CompletionMap) insertViewsWithPrefix(c *Completer, schemas map[string]bool, includeSchemaPrefix bool) {
	for schema := range schemas {
		if len(schema) == 0 {
			continue
		}
		for _, view := range c.listViews(schema) {
			text := c.quotedIdentifierIfNeeded(view)
			// Only add schema prefix if user didn't select a schema and the view is not in the default schema
			if includeSchemaPrefix && schema != c.defaultSchema {
				text = c.quotedIdentifierIfNeeded(schema) + "." + text
			}
			m.Insert(base.Candidate{
				Type: base.CandidateTypeView,
				Text: text,
			})
		}
		for _, matView := range c.listMaterializedViews(schema) {
			text := c.quotedIdentifierIfNeeded(matView)
			// Only add schema prefix if user didn't select a schema and the view is not in the default schema
			if includeSchemaPrefix && schema != c.defaultSchema {
				text = c.quotedIdentifierIfNeeded(schema) + "." + text
			}
			m.Insert(base.Candidate{
				Type: base.CandidateTypeMaterializedView,
				Text: text,
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
		schemaMeta := c.metadataCache[c.defaultDatabase].GetSchemaMetadata(schema)
		if schemaMeta == nil {
			continue
		}
		for table := range tables {
			tableMeta := schemaMeta.GetTable(table)
			if tableMeta == nil {
				continue
			}
			for _, column := range tableMeta.GetProto().GetColumns() {
				definition := fmt.Sprintf("%s.%s | %s", schema, table, column.Type)
				if !column.Nullable {
					definition += ", NOT NULL"
				}
				m.Insert(base.Candidate{
					Type:       base.CandidateTypeColumn,
					Text:       c.quotedIdentifierIfNeeded(column.Name),
					Definition: definition,
					Comment:    column.Comment,
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
		schemaMeta := metadata.GetSchemaMetadata(schema)
		if schemaMeta == nil {
			continue
		}
		for _, table := range schemaMeta.ListTableNames() {
			tableMeta := schemaMeta.GetTable(table)
			if tableMeta == nil {
				continue
			}
			for _, column := range tableMeta.GetProto().GetColumns() {
				definition := fmt.Sprintf("%s.%s | %s", schema, table, column.Type)
				if !column.Nullable {
					definition += ", NOT NULL"
				}
				m.Insert(base.Candidate{
					Type:       base.CandidateTypeColumn,
					Text:       c.quotedIdentifierIfNeeded(column.Name),
					Definition: definition,
					Comment:    column.Comment,
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
	slices.SortFunc(result, func(a, b base.Candidate) int {
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
	})
	return result
}

func (c *Completer) convertCandidates(candidates *base.CandidatesCollection) ([]base.Candidate, error) {
	keywordEntries := make(CompletionMap)
	runtimeFunctionEntries := make(CompletionMap)
	schemaEntries := make(CompletionMap)
	tableEntries := make(CompletionMap)
	columnEntries := make(CompletionMap)
	viewEntries := make(CompletionMap)

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
			if value[0] == parser.RedshiftLexerOPEN_PAREN {
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
			runtimeFunctionEntries.Insert(base.Candidate{
				Type: base.CandidateTypeFunction,
				Text: strings.ToLower(entry) + "()",
			})
		default:
			keywordEntries.Insert(base.Candidate{
				Type: base.CandidateTypeKeyword,
				Text: entry,
			})
		}
	}

	for candidate := range candidates.Rules {
		c.scanner.PopAndRestore()
		c.scanner.Push()

		c.fetchCommonTableExpression(candidates.Rules[candidate])

		switch candidate {
		case parser.RedshiftParserRULE_func_name:
			runtimeFunctionEntries.insertFunctions()
		case parser.RedshiftParserRULE_relation_expr, parser.RedshiftParserRULE_qualified_name:
			qualifier, flags := c.determineQualifiedName()

			if flags&ObjectFlagsShowFirst != 0 {
				schemaEntries.insertSchemas(c)
			}

			if flags&ObjectFlagsShowSecond != 0 {
				schemas := make(map[string]bool)
				if len(qualifier) == 0 {
					// User didn't specify the schema
					// If no default schema is selected by user, search all schemas
					if c.schemaNotSelected {
						for _, schema := range c.listAllSchemas() {
							schemas[schema] = true
						}
					} else {
						schemas[c.defaultSchema] = true
					}
					// Also include CTE tables
					schemas[""] = true
				} else {
					schemas[qualifier] = true
				}

				// Pass true for includeSchemaPrefix when no schema is selected by user
				includeSchemaPrefix := len(qualifier) == 0 && c.schemaNotSelected
				tableEntries.insertTablesWithPrefix(c, schemas, includeSchemaPrefix)
				viewEntries.insertViewsWithPrefix(c, schemas, includeSchemaPrefix)
			}
		case parser.RedshiftParserRULE_columnref:
			schema, table, flags := c.determineColumnRef()
			if flags&ObjectFlagsShowSchemas != 0 {
				schemaEntries.insertSchemas(c)
			}

			schemas := make(map[string]bool)
			if len(schema) != 0 {
				schemas[schema] = true
			} else if len(c.references) > 0 {
				for _, reference := range c.references {
					if physicalTable, ok := reference.(*base.PhysicalTableReference); ok {
						if len(physicalTable.Schema) > 0 {
							schemas[physicalTable.Schema] = true
						}
					}
				}
			}

			if len(schema) == 0 {
				// If no default schema is selected by user, search all schemas
				if c.schemaNotSelected && len(c.references) == 0 {
					for _, s := range c.listAllSchemas() {
						schemas[s] = true
					}
				} else {
					schemas[c.defaultSchema] = true
				}
				// User didn't specify the schema, we need to append cte tables.
				schemas[""] = true
			}

			if flags&ObjectFlagsShowTables != 0 {
				// Pass true for includeSchemaPrefix when no schema is selected by user and no references
				includeSchemaPrefix := len(schema) == 0 && c.schemaNotSelected && len(c.references) == 0
				tableEntries.insertTablesWithPrefix(c, schemas, includeSchemaPrefix)
				viewEntries.insertViewsWithPrefix(c, schemas, includeSchemaPrefix)

				for _, reference := range c.references {
					switch reference := reference.(type) {
					case *base.PhysicalTableReference:
						if len(schema) == 0 && len(reference.Schema) == 0 || schemas[reference.Schema] {
							if len(reference.Alias) == 0 {
								tableEntries.Insert(base.Candidate{
									Type: base.CandidateTypeTable,
									Text: c.quotedIdentifierIfNeeded(reference.Table),
								})
							} else {
								tableEntries.Insert(base.Candidate{
									Type: base.CandidateTypeTable,
									Text: c.quotedIdentifierIfNeeded(reference.Alias),
								})
							}
						}
					case *base.VirtualTableReference:
						if len(schema) > 0 {
							// If the schema is specified, we should not show the virtual table.
							continue
						}
						tableEntries.Insert(base.Candidate{
							Type: base.CandidateTypeTable,
							Text: c.quotedIdentifierIfNeeded(reference.Table),
						})
					default:
					}
				}
			}

			if flags&ObjectFlagsShowColumns != 0 {
				if schema == table {
					schemas[c.defaultSchema] = true
					// User didn't specify the schema, we need to append cte tables.
					schemas[""] = true
				}

				tables := make(map[string]bool)
				if len(table) != 0 {
					tables[table] = true

					for _, reference := range c.references {
						switch reference := reference.(type) {
						case *base.PhysicalTableReference:
							if reference.Alias == table {
								tables[reference.Table] = true
								schemas[reference.Schema] = true
							}
						case *base.VirtualTableReference:
							if reference.Table == table {
								for _, column := range reference.Columns {
									columnEntries.Insert(base.Candidate{
										Type: base.CandidateTypeColumn,
										Text: c.quotedIdentifierIfNeeded(column),
									})
								}
							}
						default:
						}
					}
				} else if len(c.references) > 0 {
					list := c.fetchSelectItemAliases(candidates.Rules[candidate])
					for _, alias := range list {
						columnEntries.Insert(base.Candidate{
							Type: base.CandidateTypeColumn,
							Text: c.quotedIdentifierIfNeeded(alias),
						})
					}
					for _, reference := range c.references {
						switch reference := reference.(type) {
						case *base.PhysicalTableReference:
							schemas[reference.Schema] = true
							tables[reference.Table] = true
						case *base.VirtualTableReference:
							for _, column := range reference.Columns {
								columnEntries.Insert(base.Candidate{
									Type: base.CandidateTypeColumn,
									Text: c.quotedIdentifierIfNeeded(column),
								})
							}
						default:
						}
					}
				} else {
					// No specified table, return all columns
					columnEntries.insertAllColumns(c)
				}

				if len(tables) > 0 {
					columnEntries.insertColumns(c, schemas, tables)
				}
			}
		default:
			// No specific completion for this rule
		}
	}

	c.scanner.PopAndRestore()
	var result []base.Candidate
	result = append(result, keywordEntries.toSlice()...)
	result = append(result, runtimeFunctionEntries.toSlice()...)
	result = append(result, schemaEntries.toSlice()...)
	result = append(result, tableEntries.toSlice()...)
	result = append(result, viewEntries.toSlice()...)
	result = append(result, columnEntries.toSlice()...)

	return result, nil
}

func (c *Completer) fetchCommonTableExpression(ruleStack []*base.RuleContext) {
	c.cteTables = nil
	for _, rule := range ruleStack {
		if rule.ID == parser.RedshiftParserRULE_select_no_parens {
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
	lexer := parser.NewRedshiftLexer(input)
	tokens := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := parser.NewRedshiftParser(tokens)

	p.BuildParseTrees = true
	p.RemoveErrorListeners()
	lexer.RemoveErrorListeners()
	tree := p.With_clause()

	listener := &CTETableListener{context: c}
	antlr.ParseTreeWalkerDefault.Walk(listener, tree)

	c.cteCache[pos] = listener.tables
	return listener.tables
}

type CTETableListener struct {
	*parser.BaseRedshiftParserListener

	context *Completer
	tables  []*base.VirtualTableReference
}

func (l *CTETableListener) EnterCommon_table_expr(ctx *parser.Common_table_exprContext) {
	table := &base.VirtualTableReference{}
	if ctx.Name() != nil {
		table.Table = normalizeRedshiftName(ctx.Name())
	}
	if ctx.Opt_name_list() != nil {
		for _, column := range ctx.Opt_name_list().Name_list().AllName() {
			table.Columns = append(table.Columns, normalizeRedshiftName(column))
		}
	} else {
		// For query span extraction, we still use PostgreSQL GetQuerySpan
		// because it uses pg_query_go internally
		if span, err := pg.GetQuerySpan(
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
		case parser.RedshiftParserRULE_simple_select_pramary, parser.RedshiftParserRULE_select_no_parens:
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
		case parser.RedshiftParserRULE_opt_sort_clause, parser.RedshiftParserRULE_group_clause, parser.RedshiftParserRULE_having_clause:
			canUseAliases = true
		default:
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
	lexer := parser.NewRedshiftLexer(input)
	tokens := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := parser.NewRedshiftParser(tokens)

	p.BuildParseTrees = true
	p.RemoveErrorListeners()
	lexer.RemoveErrorListeners()
	tree := p.Target_alias()

	listener := &TargetAliasListener{}
	antlr.ParseTreeWalkerDefault.Walk(listener, tree)
	return listener.result
}

type TargetAliasListener struct {
	*parser.BaseRedshiftParserListener

	result string
}

func (l *TargetAliasListener) EnterTarget_alias(ctx *parser.Target_aliasContext) {
	if ctx.Identifier() != nil {
		l.result = normalizeRedshiftIdentifier(ctx.Identifier())
	} else if ctx.Collabel() != nil {
		l.result = normalizeRedshiftCollabel(ctx.Collabel())
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
	position := c.scanner.GetIndex()
	if c.scanner.GetTokenChannel() != 0 {
		c.scanner.Forward(true /* skipHidden */)
	}

	if !c.scanner.IsTokenType(parser.RedshiftLexerONLY) && !c.lexer.IsIdentifier(c.scanner.GetTokenType()) {
		// We are at the end of an incomplete identifier spec.
		// Jump back.
		c.scanner.Backward(true /* skipHidden */)
	}

	// Go left until we hit a non-identifier token.
	if position > 0 {
		if c.lexer.IsIdentifier(c.scanner.GetTokenType()) && c.scanner.GetPreviousTokenType(false /* skipHidden */) == parser.RedshiftLexerDOT {
			c.scanner.Backward(true /* skipHidden */)
		}
		if c.scanner.IsTokenType(parser.RedshiftLexerDOT) && c.lexer.IsIdentifier(c.scanner.GetPreviousTokenType(false /* skipHidden */)) {
			c.scanner.Backward(true /* skipHidden */)
		}
	}

	// The current token is on the leading identifier.
	qualifier := ""
	temp := ""
	if c.lexer.IsIdentifier(c.scanner.GetTokenType()) {
		temp = normalizeIdentifier(c.scanner.GetTokenText())
		c.scanner.Forward(true /* skipHidden */)
	}

	if !c.scanner.IsTokenType(parser.RedshiftLexerDOT) || position <= c.scanner.GetIndex() {
		return qualifier, ObjectFlagsShowFirst | ObjectFlagsShowSecond
	}

	qualifier = temp
	return qualifier, ObjectFlagsShowSecond
}

func (c *Completer) determineColumnRef() (schema, table string, flags ObjectFlags) {
	position := c.scanner.GetIndex()
	if c.scanner.GetTokenChannel() != 0 {
		c.scanner.Forward(true /* skipHidden */)
	}

	tokenType := c.scanner.GetTokenType()
	if tokenType != parser.RedshiftLexerDOT && !c.lexer.IsIdentifier(c.scanner.GetTokenType()) {
		// We are at the end of an incomplete identifier spec.
		// Jump back.
		c.scanner.Backward(true /* skipHidden */)
	}

	if position > 0 {
		if c.lexer.IsIdentifier(c.scanner.GetTokenType()) && c.scanner.GetPreviousTokenType(false /* skipHidden */) == parser.RedshiftLexerDOT {
			c.scanner.Backward(true /* skipHidden */)
		}
		if c.scanner.IsTokenType(parser.RedshiftLexerDOT) && c.lexer.IsIdentifier(c.scanner.GetPreviousTokenType(false /* skipHidden */)) {
			c.scanner.Backward(true /* skipHidden */)

			if c.scanner.GetPreviousTokenType(false /* skipHidden */) == parser.RedshiftLexerDOT {
				c.scanner.Backward(true /* skipHidden */)
				if c.lexer.IsIdentifier(c.scanner.GetPreviousTokenType(false /* skipHidden */)) {
					c.scanner.Backward(true /* skipHidden */)
				}
			}
		}
	}

	schema = ""
	table = ""
	temp := ""
	if c.lexer.IsIdentifier(c.scanner.GetTokenType()) {
		temp = normalizeIdentifier(c.scanner.GetTokenText())
		c.scanner.Forward(true /* skipHidden */)
	}

	if !c.scanner.IsTokenType(parser.RedshiftLexerDOT) || position <= c.scanner.GetIndex() {
		return schema, table, ObjectFlagsShowSchemas | ObjectFlagsShowTables | ObjectFlagsShowColumns
	}

	c.scanner.Forward(true /* skipHidden */) // skip dot
	table = temp
	schema = temp
	if c.lexer.IsIdentifier(c.scanner.GetTokenType()) {
		temp = normalizeIdentifier(c.scanner.GetTokenText())
		c.scanner.Forward(true /* skipHidden */)

		if !c.scanner.IsTokenType(parser.RedshiftLexerDOT) || position <= c.scanner.GetIndex() {
			return schema, table, ObjectFlagsShowTables | ObjectFlagsShowColumns
		}

		table = temp
		return schema, table, ObjectFlagsShowColumns
	}

	return schema, table, ObjectFlagsShowTables | ObjectFlagsShowColumns
}

func normalizeIdentifier(tokenText string) string {
	if len(tokenText) >= 2 && tokenText[0] == '"' && tokenText[len(tokenText)-1] == '"' {
		return normalizeRedshiftQuotedIdentifier(tokenText)
	}
	return unquote(tokenText)
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
	c.scanner.Push()

	level := 0
	for {
		found := c.scanner.GetTokenType() == parser.RedshiftLexerFROM
		for !found {
			if !c.scanner.Forward(false /* skipHidden */) {
				break
			}

			switch c.scanner.GetTokenType() {
			case parser.RedshiftLexerOPEN_PAREN:
				level++
			case parser.RedshiftLexerCLOSE_PAREN:
				if level > 0 {
					level--
				}
			case parser.RedshiftLexerFROM:
				// Open and close parenthesis don't need to match, if we come from within a subquery.
				if level == 0 {
					found = true
				}
			default:
			}
		}

		if !found {
			c.scanner.PopAndRestore()
			return // No more FROM clauses found.
		}

		c.parseTableReferences(c.scanner.GetFollowingText())
		if c.scanner.GetTokenType() == parser.RedshiftLexerFROM {
			c.scanner.Forward(false /* skipHidden */)
		}
	}
}

func (c *Completer) collectLeadingTableReferences(caretIndex int) {
	c.scanner.Push()

	c.scanner.SeekIndex(0)

	level := 0
	for {
		found := c.scanner.GetTokenType() == parser.RedshiftLexerFROM
		for !found {
			if !c.scanner.Forward(false /* skipHidden */) || c.scanner.GetIndex() >= caretIndex {
				break
			}

			switch c.scanner.GetTokenType() {
			case parser.RedshiftLexerOPEN_PAREN:
				level++
				c.referencesStack = append([][]base.TableReference{{}}, c.referencesStack...)
			case parser.RedshiftLexerCLOSE_PAREN:
				if level == 0 {
					c.scanner.PopAndRestore()
					return // We cannot go above the initial nesting level.
				}

				level--
				c.referencesStack = c.referencesStack[1:]
			case parser.RedshiftLexerFROM:
				found = true
			default:
			}
		}

		if !found {
			c.scanner.PopAndRestore()
			return // No more FROM clauses found.
		}

		c.parseTableReferences(c.scanner.GetFollowingText())
		if c.scanner.GetTokenType() == parser.RedshiftLexerFROM {
			c.scanner.Forward(false /* skipHidden */)
		}
	}
}

func (c *Completer) parseTableReferences(fromClause string) {
	input := antlr.NewInputStream(fromClause)
	lexer := parser.NewRedshiftLexer(input)
	tokens := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := parser.NewRedshiftParser(tokens)

	p.BuildParseTrees = true
	p.RemoveErrorListeners()
	lexer.RemoveErrorListeners()
	tree := p.From_clause()

	listener := &TableRefListener{
		context: c,
	}
	antlr.ParseTreeWalkerDefault.Walk(listener, tree)
}

type TableRefListener struct {
	*parser.BaseRedshiftParserListener

	context *Completer
	level   int
}

func (l *TableRefListener) EnterTable_ref(ctx *parser.Table_refContext) {
	if _, ok := ctx.GetParent().(*parser.Table_refContext); ok {
		// if the table reference is nested, we should not process it.
		l.level++
	}

	if l.level == 0 {
		switch {
		case ctx.Relation_expr() != nil:
			var reference base.TableReference
			physicalReference := &base.PhysicalTableReference{}
			// We should use the physical reference as the default reference.
			reference = physicalReference
			list := NormalizeRedshiftQualifiedName(ctx.Relation_expr().Qualified_name())
			switch len(list) {
			case 1:
				physicalReference.Table = list[0]
			case 2:
				physicalReference.Schema = list[0]
				physicalReference.Table = list[1]
			case 3:
				physicalReference.Database = list[0]
				physicalReference.Schema = list[1]
				physicalReference.Table = list[2]
			default:
				return
			}

			if ctx.Opt_alias_clause() != nil {
				tableAlias, columnAlias := normalizeTableAlias(ctx.Opt_alias_clause())
				if len(columnAlias) > 0 {
					virtualReference := &base.VirtualTableReference{
						Table:   tableAlias,
						Columns: columnAlias,
					}
					// If the table alias has the column alias, we should use the virtual reference.
					reference = virtualReference
				} else {
					physicalReference.Alias = tableAlias
				}
			}

			l.context.referencesStack[0] = append(l.context.referencesStack[0], reference)
		case ctx.Select_with_parens() != nil:
			if ctx.Opt_alias_clause() != nil {
				virtualReference := &base.VirtualTableReference{}
				tableAlias, columnAlias := normalizeTableAlias(ctx.Opt_alias_clause())
				virtualReference.Table = tableAlias
				if len(columnAlias) > 0 {
					virtualReference.Columns = columnAlias
				} else {
					// For query span extraction, we still use PostgreSQL GetQuerySpan
					// because it uses pg_query_go internally
					if span, err := pg.GetQuerySpan(
						l.context.ctx,
						base.GetQuerySpanContext{
							InstanceID:              l.context.instanceID,
							GetDatabaseMetadataFunc: l.context.getMetadata,
							ListDatabaseNamesFunc:   l.context.listDatabaseNames,
						},
						fmt.Sprintf("SELECT * FROM %s AS %s;", ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.Select_with_parens()), tableAlias),
						l.context.defaultDatabase,
						"",
						false,
					); err == nil && span.NotFoundError == nil {
						for _, column := range span.Results {
							virtualReference.Columns = append(virtualReference.Columns, column.Name)
						}
					}
				}

				l.context.referencesStack[0] = append(l.context.referencesStack[0], virtualReference)
			}
		case ctx.OPEN_PAREN() != nil:
			if ctx.Opt_alias_clause() != nil {
				virtualReference := &base.VirtualTableReference{}
				tableAlias, columnAlias := normalizeTableAlias(ctx.Opt_alias_clause())
				virtualReference.Table = tableAlias
				if len(columnAlias) > 0 {
					virtualReference.Columns = columnAlias
				}

				l.context.referencesStack[0] = append(l.context.referencesStack[0], virtualReference)
			}
		default:
		}
	}
}

func (l *TableRefListener) ExitTable_ref(ctx *parser.Table_refContext) {
	if _, ok := ctx.GetParent().(*parser.Table_refContext); ok {
		l.level--
	}
}

func (l *TableRefListener) EnterSelect_with_parens(_ *parser.Select_with_parensContext) {
	l.level++
}

func (l *TableRefListener) ExitSelect_with_parens(_ *parser.Select_with_parensContext) {
	l.level--
}

func normalizeTableAlias(ctx parser.IOpt_alias_clauseContext) (string, []string) {
	if ctx == nil || ctx.Table_alias_clause() == nil {
		return "", nil
	}

	tableAlias := ""
	aliasClause := ctx.Table_alias_clause()
	if aliasClause.Table_alias() != nil {
		tableAlias = normalizeRedshiftTableAlias(aliasClause.Table_alias())
	}

	var columnAliases []string
	if aliasClause.Name_list() != nil {
		columnAliases = append(columnAliases, normalizeRedshiftNameList(aliasClause.Name_list())...)
	}

	return tableAlias, columnAliases
}

func prepareTrickyParserAndScanner(statement string, caretLine int, caretOffset int) (*parser.RedshiftParser, *parser.RedshiftLexer, *base.Scanner) {
	statement, caretLine, caretOffset = skipHeadingSQLs(statement, caretLine, caretOffset)
	statement, caretLine, caretOffset = skipHeadingSQLWithoutSemicolon(statement, caretLine, caretOffset)
	input := antlr.NewInputStream(statement)
	lexer := parser.NewRedshiftLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := parser.NewRedshiftParser(stream)
	p.RemoveErrorListeners()
	lexer.RemoveErrorListeners()
	scanner := base.NewScanner(stream, true /* fillInput */)
	scanner.SeekPosition(caretLine, caretOffset)
	scanner.Push()
	return p, lexer, scanner
}

func prepareParserAndScanner(statement string, caretLine int, caretOffset int) (*parser.RedshiftParser, *parser.RedshiftLexer, *base.Scanner) {
	statement, caretLine, caretOffset = skipHeadingSQLs(statement, caretLine, caretOffset)
	input := antlr.NewInputStream(statement)
	lexer := parser.NewRedshiftLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := parser.NewRedshiftParser(stream)
	p.RemoveErrorListeners()
	lexer.RemoveErrorListeners()
	scanner := base.NewScanner(stream, true /* fillInput */)
	scanner.SeekPosition(caretLine, caretOffset)
	scanner.Push()
	return p, lexer, scanner
}

// caretLine is 1-based and caretOffset is 0-based.
func skipHeadingSQLs(statement string, caretLine int, caretOffset int) (string, int, int) {
	newCaretLine, newCaretOffset := caretLine, caretOffset
	list, err := SplitSQL(statement)
	if err != nil || len(base.FilterEmptyStatements(list)) <= 1 {
		return statement, caretLine, caretOffset
	}

	caretLine-- // Convert caretLine to 0-based.

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
			return statement, caretLine, caretOffset
		}
	}

	return buf.String(), newCaretLine, newCaretOffset
}

// caretLine is 1-based and caretOffset is 0-based.
func skipHeadingSQLWithoutSemicolon(statement string, caretLine int, caretOffset int) (string, int, int) {
	input := antlr.NewInputStream(statement)
	lexer := parser.NewRedshiftLexer(input)
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
		if token.GetTokenType() == parser.RedshiftLexerSELECT && token.GetColumn() == 0 {
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

	schemaMeta := c.metadataCache[c.defaultDatabase].GetSchemaMetadata(schema)
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

	schemaMeta := c.metadataCache[c.defaultDatabase].GetSchemaMetadata(schema)
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

	schemaMeta := c.metadataCache[c.defaultDatabase].GetSchemaMetadata(schema)
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

	schemaMeta := c.metadataCache[c.defaultDatabase].GetSchemaMetadata(schema)
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
	// Redshift requires double quotes for identifiers with special characters or start with digits
	if !isValidUnquotedIdentifier(s) {
		// If the identifier contains double quotes, we need to escape them by doubling them
		if strings.Contains(s, `"`) {
			s = strings.ReplaceAll(s, `"`, `""`)
		}
		return fmt.Sprintf(`"%s"`, s)
	}
	return s
}

// isValidUnquotedIdentifier checks if the identifier can be used without quotes in Redshift.
// Redshift unquoted identifiers must:
// - Begin with a letter (a-z, A-Z) or underscore
// - Contain only letters, digits, and underscores
func isValidUnquotedIdentifier(s string) bool {
	if len(s) == 0 {
		return false
	}

	// First character must be a letter or underscore
	first := rune(s[0])
	if !unicode.IsLetter(first) && first != '_' {
		return false
	}

	// Remaining characters must be letters, digits, or underscores
	for _, ch := range s[1:] {
		if !unicode.IsLetter(ch) && !unicode.IsDigit(ch) && ch != '_' {
			return false
		}
	}

	return true
}

// Normalization functions for Redshift
func normalizeRedshiftName(ctx parser.INameContext) string {
	if ctx == nil {
		return ""
	}
	text := ctx.GetText()
	return normalizeRedshiftIdentifierText(text)
}

func normalizeRedshiftColid(ctx parser.IColidContext) string {
	if ctx == nil {
		return ""
	}
	text := ctx.GetText()
	return normalizeRedshiftIdentifierText(text)
}

func normalizeRedshiftIdentifier(ctx parser.IIdentifierContext) string {
	if ctx == nil {
		return ""
	}
	text := ctx.GetText()
	return normalizeRedshiftIdentifierText(text)
}

func normalizeRedshiftCollabel(ctx parser.ICollabelContext) string {
	if ctx == nil {
		return ""
	}
	text := ctx.GetText()
	return normalizeRedshiftIdentifierText(text)
}

func normalizeRedshiftAttrName(ctx parser.IAttr_nameContext) string {
	if ctx == nil {
		return ""
	}
	text := ctx.GetText()
	return normalizeRedshiftIdentifierText(text)
}

func normalizeRedshiftTableAlias(ctx parser.ITable_aliasContext) string {
	if ctx == nil {
		return ""
	}
	text := ctx.GetText()
	return normalizeRedshiftIdentifierText(text)
}

func normalizeRedshiftNameList(ctx parser.IName_listContext) []string {
	if ctx == nil {
		return nil
	}
	var result []string
	for _, name := range ctx.AllName() {
		result = append(result, normalizeRedshiftName(name))
	}
	return result
}

func normalizeRedshiftQuotedIdentifier(text string) string {
	if len(text) < 2 {
		return text
	}
	// Remove quotes and handle escaped quotes
	return strings.ReplaceAll(text[1:len(text)-1], `""`, `"`)
}

func normalizeRedshiftIdentifierText(text string) string {
	if len(text) >= 2 && text[0] == '"' && text[len(text)-1] == '"' {
		return normalizeRedshiftQuotedIdentifier(text)
	}
	// Redshift stores unquoted identifiers in lowercase
	return strings.ToLower(text)
}

func NormalizeRedshiftQualifiedName(ctx parser.IQualified_nameContext) []string {
	if ctx == nil {
		return nil
	}
	var result []string

	// First part is the colid
	if ctx.Colid() != nil {
		result = append(result, normalizeRedshiftColid(ctx.Colid()))
	}

	// Additional parts come from indirection
	if ctx.Indirection() != nil {
		for _, el := range ctx.Indirection().AllIndirection_el() {
			if el.Attr_name() != nil {
				result = append(result, normalizeRedshiftAttrName(el.Attr_name()))
			}
		}
	}

	return result
}
