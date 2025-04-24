package trino

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	trino "github.com/bytebase/trino-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
	"github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	// globalFollowSetsByState is the global follow sets by state.
	// It is shared by all Trino completers.
	// The FollowSetsByState is the thread-safe struct.
	globalFollowSetsByState = base.NewFollowSetsByState()
)

func init() {
	base.RegisterCompleteFunc(store.Engine_TRINO, Completion)
}

// Completion is the entry point of Trino code completion.
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
		antlr.TokenEOF:                         true,
		trino.TrinoLexerDOLLAR_:                true,
		trino.TrinoLexerLPAREN_:                true,
		trino.TrinoLexerRPAREN_:                true,
		trino.TrinoLexerLSQUARE_:               true,
		trino.TrinoLexerRSQUARE_:               true,
		trino.TrinoLexerCOMMA_:                 true,
		trino.TrinoLexerSEMICOLON_:             true,
		trino.TrinoLexerCOLON_:                 true,
		trino.TrinoLexerEQ_:                    true,
		trino.TrinoLexerDOT_:                   true,
		trino.TrinoLexerPLUS_:                  true,
		trino.TrinoLexerMINUS_:                 true,
		trino.TrinoLexerSLASH_:                 true,
		trino.TrinoLexerCARET_:                 true,
		trino.TrinoLexerLT_:                    true,
		trino.TrinoLexerGT_:                    true,
		trino.TrinoLexerLTE_:                   true,
		trino.TrinoLexerGTE_:                   true,
		trino.TrinoLexerNEQ_:                   true,
		trino.TrinoLexerPERCENT_:               true,
		trino.TrinoLexerCONCAT_:                true,
		trino.TrinoLexerQUESTION_MARK_:         true,
		trino.TrinoLexerVBAR_:                  true,
		trino.TrinoLexerASTERISK_:              true,
		trino.TrinoLexerSTRING_:                true,
		trino.TrinoLexerUNICODE_STRING_:        true,
		trino.TrinoLexerBINARY_LITERAL_:        true,
		trino.TrinoLexerINTEGER_VALUE_:         true,
		trino.TrinoLexerDECIMAL_VALUE_:         true,
		trino.TrinoLexerDOUBLE_VALUE_:          true,
		trino.TrinoLexerIDENTIFIER_:            true,
		trino.TrinoLexerDIGIT_IDENTIFIER_:      true,
		trino.TrinoLexerQUOTED_IDENTIFIER_:     true,
		trino.TrinoLexerBACKQUOTED_IDENTIFIER_: true,
	}
}

func newPreferredRules() map[int]bool {
	return map[int]bool{
		trino.TrinoParserRULE_relationPrimary:   true,
		trino.TrinoParserRULE_qualifiedName:     true,
		trino.TrinoParserRULE_primaryExpression: true,
		trino.TrinoParserRULE_expression:        true,
	}
}

func newNoSeparatorRequired() map[int]bool {
	return map[int]bool{
		trino.TrinoLexerDOLLAR_:        true,
		trino.TrinoLexerLPAREN_:        true,
		trino.TrinoLexerRPAREN_:        true,
		trino.TrinoLexerLSQUARE_:       true,
		trino.TrinoLexerRSQUARE_:       true,
		trino.TrinoLexerCOMMA_:         true,
		trino.TrinoLexerSEMICOLON_:     true,
		trino.TrinoLexerCOLON_:         true,
		trino.TrinoLexerEQ_:            true,
		trino.TrinoLexerDOT_:           true,
		trino.TrinoLexerPLUS_:          true,
		trino.TrinoLexerMINUS_:         true,
		trino.TrinoLexerSLASH_:         true,
		trino.TrinoLexerCARET_:         true,
		trino.TrinoLexerLT_:            true,
		trino.TrinoLexerGT_:            true,
		trino.TrinoLexerLTE_:           true,
		trino.TrinoLexerGTE_:           true,
		trino.TrinoLexerNEQ_:           true,
		trino.TrinoLexerPERCENT_:       true,
		trino.TrinoLexerCONCAT_:        true,
		trino.TrinoLexerQUESTION_MARK_: true,
		trino.TrinoLexerVBAR_:          true,
		trino.TrinoLexerASTERISK_:      true,
	}
}

type Completer struct {
	ctx                 context.Context
	core                *base.CodeCompletionCore
	scene               base.SceneType
	parser              *trino.TrinoParser
	lexer               *trino.TrinoLexer
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
	// For all Trino completers, we use one global follow sets by state.
	// The FollowSetsByState is the thread-safe struct.
	core := base.NewCodeCompletionCore(
		parser,
		newIgnoredTokens(),
		newPreferredRules(),
		&globalFollowSetsByState,
		trino.TrinoParserRULE_query,
		trino.TrinoParserRULE_queryNoWith,
		trino.TrinoParserRULE_aliasedRelation,
		trino.TrinoParserRULE_with,
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
		listDatabaseNames:   cCtx.ListDatabaseNames,
		metadataCache:       make(map[string]*model.DatabaseMetadata),
		noSeparatorRequired: newNoSeparatorRequired(),
		cteCache:            make(map[int][]*base.VirtualTableReference),
	}
}

func NewStandardCompleter(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) *Completer {
	parser, lexer, scanner := prepareParserAndScanner(statement, caretLine, caretOffset)
	// For all Trino completers, we use one global follow sets by state.
	// The FollowSetsByState is the thread-safe struct.
	core := base.NewCodeCompletionCore(
		parser,
		newIgnoredTokens(),
		newPreferredRules(),
		&globalFollowSetsByState,
		trino.TrinoParserRULE_query,
		trino.TrinoParserRULE_queryNoWith,
		trino.TrinoParserRULE_aliasedRelation,
		trino.TrinoParserRULE_with,
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
		listDatabaseNames:   cCtx.ListDatabaseNames,
		metadataCache:       make(map[string]*model.DatabaseMetadata),
		noSeparatorRequired: newNoSeparatorRequired(),
		cteCache:            make(map[int][]*base.VirtualTableReference),
	}
}

func (c *Completer) completion() ([]base.Candidate, error) {
	// Check the caret token is quoted or not.
	// This check should be done before checking the caret token is a separator or not.
	if c.scanner.IsTokenType(trino.TrinoLexerQUOTED_IDENTIFIER_) ||
		c.scanner.IsTokenType(trino.TrinoLexerBACKQUOTED_IDENTIFIER_) {
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
		context = c.parser.QueryNoWith()
	} else {
		context = c.parser.SingleStatement()
	}

	candidates := c.core.CollectCandidates(caretIndex, context)

	for ruleName := range candidates.Rules {
		if ruleName == trino.TrinoParserRULE_primaryExpression {
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
	// Trino has many built-in functions
	// This is a representative set of common Trino functions organized by category
	
	// Aggregate functions
	aggregateFunctions := []string{
		"avg", "count", "max", "min", "sum", 
		"array_agg", "map_agg", "multimap_agg", "string_agg",
		"approx_distinct", "approx_percentile", "bitwise_and_agg", "bitwise_or_agg",
		"corr", "covar_pop", "covar_samp", "geometric_mean", "numeric_histogram", 
		"regr_intercept", "regr_slope", "skewness", "kurtosis",
	}
	
	// Array functions
	arrayFunctions := []string{
		"array_distinct", "array_intersect", "array_union", "array_except", 
		"array_join", "array_max", "array_min", "array_position", "array_remove",
		"array_sort", "arrays_overlap", "concat", "contains", "element_at", 
		"filter", "flatten", "reduce", "reverse", "sequence", "shuffle", 
		"slice", "zip", "zip_with",
	}
	
	// Date/Time functions
	dateTimeFunctions := []string{
		"current_date", "current_time", "current_timestamp", "current_timezone",
		"date", "date_add", "date_diff", "date_format", "date_parse", "date_trunc",
		"day", "day_of_month", "day_of_week", "day_of_year", "extract", "from_unixtime",
		"from_iso8601_timestamp", "last_day_of_month", "localtime", "localtimestamp",
		"month", "now", "quarter", "time", "timestamp", "timestamp_diff", "to_unixtime",
		"with_timezone", "year", "year_of_week",
	}
	
	// Mathematical functions
	mathFunctions := []string{
		"abs", "acos", "asin", "atan", "atan2", "cbrt", "ceil", "ceiling", 
		"cos", "cosh", "degrees", "e", "exp", "floor", "ln", "log", "log2", "log10",
		"mod", "pi", "pow", "power", "radians", "rand", "random", "round",
		"sign", "sin", "sinh", "sqrt", "tan", "tanh", "truncate",
	}
	
	// String functions
	stringFunctions := []string{
		"chr", "concat", "format", "hamming_distance", "length", "levenshtein_distance", 
		"lower", "lpad", "ltrim", "normalize", "position", "regexp_extract", 
		"regexp_extract_all", "regexp_like", "regexp_replace", "replace", "reverse",
		"rpad", "rtrim", "split", "split_part", "strpos", "substr", "substring",
		"trim", "upper", "word_stem",
	}
	
	// JSON functions
	jsonFunctions := []string{
		"is_json_scalar", "json_array_contains", "json_array_get", "json_array_length",
		"json_extract", "json_extract_scalar", "json_format", "json_parse",
		"json_query", "json_value", "json_size",
	}
	
	// Window functions
	windowFunctions := []string{
		"cume_dist", "dense_rank", "first_value", "lag", "last_value", "lead",
		"nth_value", "ntile", "percent_rank", "rank", "row_number",
	}
	
	// URL functions
	urlFunctions := []string{
		"url_decode", "url_encode", "url_extract_fragment", "url_extract_host",
		"url_extract_parameter", "url_extract_path", "url_extract_port",
		"url_extract_protocol", "url_extract_query",
	}

	// Combine all function categories
	trinoFunctions := make([]string, 0)
	trinoFunctions = append(trinoFunctions, aggregateFunctions...)
	trinoFunctions = append(trinoFunctions, arrayFunctions...)
	trinoFunctions = append(trinoFunctions, dateTimeFunctions...)
	trinoFunctions = append(trinoFunctions, mathFunctions...)
	trinoFunctions = append(trinoFunctions, stringFunctions...)
	trinoFunctions = append(trinoFunctions, jsonFunctions...)
	trinoFunctions = append(trinoFunctions, windowFunctions...)
	trinoFunctions = append(trinoFunctions, urlFunctions...)

	for _, name := range trinoFunctions {
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

func (m CompletionMap) insertCatalogs(c *Completer) {
	// In Trino, catalogs are first-level objects above schemas
	if c.listDatabaseNames != nil {
		catalogs, err := c.listDatabaseNames(c.ctx, c.instanceID)
		if err != nil {
			return
		}
		for _, catalog := range catalogs {
			m.Insert(base.Candidate{
				Type: base.CandidateTypeDatabase,
				Text: c.quotedIdentifierIfNeeded(catalog),
			})
		}
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
	keywordEntries := make(CompletionMap)
	runtimeFunctionEntries := make(CompletionMap)
	catalogEntries := make(CompletionMap)
	schemaEntries := make(CompletionMap)
	tableEntries := make(CompletionMap)
	columnEntries := make(CompletionMap)
	viewEntries := make(CompletionMap)

	for token, value := range candidates.Tokens {
		if token < 0 {
			continue
		}
		entry := c.parser.SymbolicNames[token]
		if strings.HasSuffix(entry, "_") {
			entry = entry[:len(entry)-1]
		} else {
			entry = unquote(entry)
		}

		list := 0
		if len(value) > 0 {
			// For function call:
			if value[0] == trino.TrinoLexerLPAREN_ {
				list = 1
			} else {
				for _, item := range value {
					subEntry := c.parser.SymbolicNames[item]
					if strings.HasSuffix(subEntry, "_") {
						subEntry = subEntry[:len(subEntry)-1]
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
		case trino.TrinoParserRULE_expression:
			runtimeFunctionEntries.insertFunctions()
		case trino.TrinoParserRULE_relationPrimary, trino.TrinoParserRULE_qualifiedName:
			qualifier, flags := c.determineQualifiedName()

			if flags&ObjectFlagsShowFirst != 0 {
				catalogEntries.insertCatalogs(c)
			}

			if flags&ObjectFlagsShowSecond != 0 {
				if len(qualifier) == 0 {
					// No qualifier - show schemas
					schemaEntries.insertSchemas(c)
				} else {
					// Could be a catalog (showing its schemas) or a schema (showing its tables)
					// Try both possibilities since we can't always determine from context
					schemas := make(map[string]bool)
					// Assume qualifier is a schema and show its tables
					schemas[qualifier] = true
					tableEntries.insertTables(c, schemas)
					viewEntries.insertViews(c, schemas)
					
					// For Trino's hierarchical naming, also try getting schemas for this catalog
					if c.listDatabaseNames != nil {
						catalogs, err := c.listDatabaseNames(c.ctx, c.instanceID)
						if err == nil {
							for _, catalog := range catalogs {
								if catalog == qualifier {
									// If qualifier matches a catalog name, show its schemas
									schemaEntries.insertSchemas(c)
									break
								}
							}
						}
					}
				}
			}

			if flags&ObjectFlagsShowThird != 0 {
				// If we get here, we're showing tables within a schema context
				schemas := make(map[string]bool)
				schemas[c.defaultSchema] = true
				// User didn't specify the schema, we need to append cte tables.
				schemas[""] = true
				tableEntries.insertTables(c, schemas)
				viewEntries.insertViews(c, schemas)
			}
		case trino.TrinoParserRULE_primaryExpression:
			catalog, schema, table, flags := c.determineColumnRef()

			if flags&ObjectFlagsShowCatalogs != 0 {
				catalogEntries.insertCatalogs(c)
			}

			if flags&ObjectFlagsShowSchemas != 0 {
				schemaEntries.insertSchemas(c)
			}

			if flags&ObjectFlagsShowTables != 0 {
				schemas := make(map[string]bool)
				if len(schema) != 0 {
					schemas[schema] = true
				} else {
					schemas[c.defaultSchema] = true
					// User didn't specify the schema, we need to append cte tables.
					schemas[""] = true
				}

				tableEntries.insertTables(c, schemas)
				viewEntries.insertViews(c, schemas)

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
					}
				}
			}

			if flags&ObjectFlagsShowColumns != 0 {
				schemas := make(map[string]bool)

				if len(catalog) != 0 && len(schema) != 0 {
					schemas[schema] = true
				} else if len(schema) != 0 {
					schemas[schema] = true
				} else {
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
		}
	}

	c.scanner.PopAndRestore()
	var result []base.Candidate
	result = append(result, keywordEntries.toSlice()...)
	result = append(result, runtimeFunctionEntries.toSlice()...)
	result = append(result, catalogEntries.toSlice()...)
	result = append(result, schemaEntries.toSlice()...)
	result = append(result, tableEntries.toSlice()...)
	result = append(result, viewEntries.toSlice()...)
	result = append(result, columnEntries.toSlice()...)

	return result, nil
}

func (c *Completer) fetchCommonTableExpression(ruleStack []*base.RuleContext) {
	c.cteTables = nil
	for _, rule := range ruleStack {
		if rule.ID == trino.TrinoParserRULE_queryNoWith {
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
	lexer := trino.NewTrinoLexer(input)
	tokens := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := trino.NewTrinoParser(tokens)

	parser.BuildParseTrees = true
	parser.RemoveErrorListeners()
	lexer.RemoveErrorListeners()
	tree := parser.With()

	listener := &CTETableListener{context: c}
	antlr.ParseTreeWalkerDefault.Walk(listener, tree)

	c.cteCache[pos] = listener.tables
	return listener.tables
}

type CTETableListener struct {
	*trino.BaseTrinoParserListener

	context *Completer
	tables  []*base.VirtualTableReference
}

func (l *CTETableListener) EnterNamedQuery(ctx *trino.NamedQueryContext) {
	table := &base.VirtualTableReference{}
	if ctx.Identifier() != nil {
		table.Table = NormalizeTrinoIdentifier(ctx.Identifier().GetText())
	}
	if ctx.ColumnAliases() != nil {
		for _, column := range ctx.ColumnAliases().AllIdentifier() {
			table.Columns = append(table.Columns, NormalizeTrinoIdentifier(column.GetText()))
		}
	} else {
		if span, err := GetQuerySpan(
			l.context.ctx,
			base.GetQuerySpanContext{
				InstanceID:              l.context.instanceID,
				GetDatabaseMetadataFunc: l.context.getMetadata,
				ListDatabaseNamesFunc:   l.context.listDatabaseNames,
			},
			ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.Query()),
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
		case trino.TrinoParserRULE_query, trino.TrinoParserRULE_queryNoWith:
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
		case trino.TrinoParserRULE_sortItem, trino.TrinoParserRULE_groupBy:
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
	lexer := trino.NewTrinoLexer(input)
	tokens := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := trino.NewTrinoParser(tokens)

	parser.BuildParseTrees = true
	parser.RemoveErrorListeners()
	lexer.RemoveErrorListeners()
	tree := parser.AliasedRelation() // Use AliasedRelation instead of TableAlias

	listener := &TableAliasListener{}
	antlr.ParseTreeWalkerDefault.Walk(listener, tree)
	return listener.result
}

type TableAliasListener struct {
	*trino.BaseTrinoParserListener

	result string
}

func (l *TableAliasListener) EnterAliasedRelation(ctx *trino.AliasedRelationContext) {
	if ctx.Identifier() != nil {
		l.result = NormalizeTrinoIdentifier(ctx.Identifier().GetText())
	}
}

type ObjectFlags int

const (
	ObjectFlagsShowCatalogs ObjectFlags = 1 << iota
	ObjectFlagsShowSchemas
	ObjectFlagsShowTables
	ObjectFlagsShowColumns
	ObjectFlagsShowFirst
	ObjectFlagsShowSecond
	ObjectFlagsShowThird
)

// determineQualifiedName analyzes the current scanner position to determine
// what is being typed regarding qualified names (catalog.schema.table). 
// It returns the current qualifier and appropriate object flags to indicate 
// what types of objects should be suggested for autocompletion.
func (c *Completer) determineQualifiedName() (string, ObjectFlags) {
	position := c.scanner.GetIndex()
	if c.scanner.GetTokenChannel() != 0 {
		c.scanner.Forward(true /* skipHidden */)
	}

	if !c.scanner.IsTokenType(trino.TrinoLexerONLY_) && !isIdentifier(c.scanner.GetTokenType()) {
		// We are at the end of an incomplete identifier spec.
		// Jump back.
		c.scanner.Backward(true /* skipHidden */)
	}

	// Go left until we hit a non-identifier token.
	if position > 0 {
		if isIdentifier(c.scanner.GetTokenType()) && c.scanner.GetPreviousTokenType(false /* skipHidden */) == trino.TrinoLexerDOT_ {
			c.scanner.Backward(true /* skipHidden */)
		}
		if c.scanner.IsTokenType(trino.TrinoLexerDOT_) && isIdentifier(c.scanner.GetPreviousTokenType(false /* skipHidden */)) {
			c.scanner.Backward(true /* skipHidden */)

			// Check for one more level in Trino's three-part naming (catalog.schema.table)
			if c.scanner.GetPreviousTokenType(false /* skipHidden */) == trino.TrinoLexerDOT_ {
				c.scanner.Backward(true /* skipHidden */)
				if isIdentifier(c.scanner.GetPreviousTokenType(false /* skipHidden */)) {
					c.scanner.Backward(true /* skipHidden */)
				}
			}
		}
	}

	// The current token is on the leading identifier.
	qualifier := ""
	temp := ""
	if isIdentifier(c.scanner.GetTokenType()) {
		temp = unquote(c.scanner.GetTokenText())
		c.scanner.Forward(true /* skipHidden */)
	}

	if !c.scanner.IsTokenType(trino.TrinoLexerDOT_) || position <= c.scanner.GetIndex() {
		return qualifier, ObjectFlagsShowFirst | ObjectFlagsShowSecond | ObjectFlagsShowThird
	}

	qualifier = temp
	return qualifier, ObjectFlagsShowSecond | ObjectFlagsShowThird
}

// determineColumnRef analyzes the current scanner position to determine
// what column reference is being typed. For Trino, this could be a qualified name with 
// up to three parts: catalog.schema.table.column. 
// It returns the appropriate parts and flags to indicate what to suggest for completion.
func (c *Completer) determineColumnRef() (catalog, schema, table string, flags ObjectFlags) {
	position := c.scanner.GetIndex()
	if c.scanner.GetTokenChannel() != 0 {
		c.scanner.Forward(true /* skipHidden */)
	}

	tokenType := c.scanner.GetTokenType()
	if tokenType != trino.TrinoLexerDOT_ && !isIdentifier(c.scanner.GetTokenType()) {
		// We are at the end of an incomplete identifier spec.
		// Jump back.
		c.scanner.Backward(true /* skipHidden */)
	}

	if position > 0 {
		if isIdentifier(c.scanner.GetTokenType()) && c.scanner.GetPreviousTokenType(false /* skipHidden */) == trino.TrinoLexerDOT_ {
			c.scanner.Backward(true /* skipHidden */)
		}
		if c.scanner.IsTokenType(trino.TrinoLexerDOT_) && isIdentifier(c.scanner.GetPreviousTokenType(false /* skipHidden */)) {
			c.scanner.Backward(true /* skipHidden */)

			if c.scanner.GetPreviousTokenType(false /* skipHidden */) == trino.TrinoLexerDOT_ {
				c.scanner.Backward(true /* skipHidden */)
				if isIdentifier(c.scanner.GetPreviousTokenType(false /* skipHidden */)) {
					c.scanner.Backward(true /* skipHidden */)

					// One more level for Trino's catalog.schema.table.column
					if c.scanner.GetPreviousTokenType(false /* skipHidden */) == trino.TrinoLexerDOT_ {
						c.scanner.Backward(true /* skipHidden */)
						if isIdentifier(c.scanner.GetPreviousTokenType(false /* skipHidden */)) {
							c.scanner.Backward(true /* skipHidden */)
						}
					}
				}
			}
		}
	}

	catalog = ""
	schema = ""
	table = ""
	temp := ""
	if isIdentifier(c.scanner.GetTokenType()) {
		temp = unquote(c.scanner.GetTokenText())
		c.scanner.Forward(true /* skipHidden */)
	}

	if !c.scanner.IsTokenType(trino.TrinoLexerDOT_) || position <= c.scanner.GetIndex() {
		return catalog, schema, table, ObjectFlagsShowCatalogs | ObjectFlagsShowSchemas | ObjectFlagsShowTables | ObjectFlagsShowColumns
	}

	c.scanner.Forward(true /* skipHidden */) // skip dot

	// First part could be catalog or schema (depending on context)
	catalog = temp

	if isIdentifier(c.scanner.GetTokenType()) {
		temp = unquote(c.scanner.GetTokenText())
		c.scanner.Forward(true /* skipHidden */)

		if !c.scanner.IsTokenType(trino.TrinoLexerDOT_) || position <= c.scanner.GetIndex() {
			// This is catalog.schema or schema.table
			schema = temp
			return catalog, schema, table, ObjectFlagsShowTables | ObjectFlagsShowColumns
		}

		c.scanner.Forward(true /* skipHidden */) // skip dot
		schema = temp

		if isIdentifier(c.scanner.GetTokenType()) {
			temp = unquote(c.scanner.GetTokenText())
			c.scanner.Forward(true /* skipHidden */)

			if !c.scanner.IsTokenType(trino.TrinoLexerDOT_) || position <= c.scanner.GetIndex() {
				// This is catalog.schema.table
				table = temp
				return catalog, schema, table, ObjectFlagsShowColumns
			}

			// This is catalog.schema.table.
			table = temp
			return catalog, schema, table, ObjectFlagsShowColumns
		}

		// This is catalog.schema.
		return catalog, schema, table, ObjectFlagsShowTables | ObjectFlagsShowColumns
	}

	// This is catalog.
	schema = catalog
	catalog = ""
	return catalog, schema, table, ObjectFlagsShowTables | ObjectFlagsShowColumns
}

func unquote(s string) string {
	if len(s) < 2 {
		return s
	}

	if (s[0] == '\'' || s[0] == '"' || s[0] == '`') && s[0] == s[len(s)-1] {
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
		found := c.scanner.GetTokenType() == trino.TrinoLexerFROM_
		for !found {
			if !c.scanner.Forward(false /* skipHidden */) {
				break
			}

			switch c.scanner.GetTokenType() {
			case trino.TrinoLexerLPAREN_:
				level++
			case trino.TrinoLexerRPAREN_:
				if level > 0 {
					level--
				}
			case trino.TrinoLexerFROM_:
				// Open and close parenthesis don't need to match, if we come from within a subquery.
				if level == 0 {
					found = true
				}
			}
		}

		if !found {
			c.scanner.PopAndRestore()
			return // No more FROM clauses found.
		}

		c.parseTableReferences(c.scanner.GetFollowingText())
		if c.scanner.GetTokenType() == trino.TrinoLexerFROM_ {
			c.scanner.Forward(false /* skipHidden */)
		}
	}
}

func (c *Completer) collectLeadingTableReferences(caretIndex int) {
	c.scanner.Push()

	c.scanner.SeekIndex(0)

	level := 0
	for {
		found := c.scanner.GetTokenType() == trino.TrinoLexerFROM_
		for !found {
			if !c.scanner.Forward(false /* skipHidden */) || c.scanner.GetIndex() >= caretIndex {
				break
			}

			switch c.scanner.GetTokenType() {
			case trino.TrinoLexerLPAREN_:
				level++
				c.referencesStack = append([][]base.TableReference{{}}, c.referencesStack...)
			case trino.TrinoLexerRPAREN_:
				if level == 0 {
					c.scanner.PopAndRestore()
					return // We cannot go above the initial nesting level.
				}

				level--
				c.referencesStack = c.referencesStack[1:]
			case trino.TrinoLexerFROM_:
				found = true
			}
		}

		if !found {
			c.scanner.PopAndRestore()
			return // No more FROM clauses found.
		}

		c.parseTableReferences(c.scanner.GetFollowingText())
		if c.scanner.GetTokenType() == trino.TrinoLexerFROM_ {
			c.scanner.Forward(false /* skipHidden */)
		}
	}
}

func (c *Completer) parseTableReferences(fromClause string) {
	input := antlr.NewInputStream(fromClause)
	lexer := trino.NewTrinoLexer(input)
	tokens := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := trino.NewTrinoParser(tokens)

	parser.BuildParseTrees = true
	parser.RemoveErrorListeners()
	lexer.RemoveErrorListeners()
	tree := parser.Relation() // Use Relation instead of FromClause

	listener := &TableRefListener{
		context: c,
	}
	antlr.ParseTreeWalkerDefault.Walk(listener, tree)
}

type TableRefListener struct {
	*trino.BaseTrinoParserListener

	context *Completer
	level   int
}

func (l *TableRefListener) EnterTableName(ctx *trino.TableNameContext) {
	// TableName is equivalent to TableRelation in our implementation
	if l.level == 0 {
		var reference base.TableReference
		physicalReference := &base.PhysicalTableReference{}
		// We should use the physical reference as the default reference.
		reference = physicalReference

		if ctx.QualifiedName() != nil {
			parts := ExtractQualifiedNameParts(ctx.QualifiedName())

			switch len(parts) {
			case 1:
				physicalReference.Table = parts[0]
			case 2:
				physicalReference.Schema = parts[0]
				physicalReference.Table = parts[1]
			case 3:
				physicalReference.Database = parts[0]
				physicalReference.Schema = parts[1]
				physicalReference.Table = parts[2]
			default:
				return
			}
		}

		// Table aliases are handled in AliasedRelation, not here
		// We'll check for them in EnterAliasedRelation instead

		l.context.referencesStack[0] = append(l.context.referencesStack[0], reference)
	}
}

func (l *TableRefListener) EnterSubqueryRelation(ctx *trino.SubqueryRelationContext) {
	// Increment nesting level to track depth of relations
	l.level++
	
	// Note: SubqueryRelationContext doesn't have TableAlias() method
	// TableAlias info should be accessed through AliasedRelation in Trino
	// We still want to capture the subquery query context
	if l.level == 1 { // Adjusted level check to match our nesting logic
		// We need to store this subquery context for later use
		// Let EnterAliasedRelation handle the aliases when a subquery is aliased
		
		// Get query context from the subquery
		if ctx.Query() != nil {
			// Just increment the level here
			// The actual aliasing will be handled in EnterAliasedRelation
		}
	}
}

func (l *TableRefListener) ExitSubqueryRelation(ctx *trino.SubqueryRelationContext) {
	// Decrement nesting level when exiting the relation
	l.level--
}

func (l *TableRefListener) EnterParenthesizedRelation(ctx *trino.ParenthesizedRelationContext) {
	l.level++
}

func (l *TableRefListener) ExitParenthesizedRelation(ctx *trino.ParenthesizedRelationContext) {
	l.level--
}

func (l *TableRefListener) EnterQuery(ctx *trino.QueryContext) {
	l.level++
}

func (l *TableRefListener) ExitQuery(ctx *trino.QueryContext) {
	l.level--
}

// Add an EnterAliasedRelation method to TableRefListener to handle aliases
func (l *TableRefListener) EnterAliasedRelation(ctx *trino.AliasedRelationContext) {
	// Handle aliasing for either tables or subqueries
	if l.level == 0 && ctx.Identifier() != nil {
		// Get the alias
		alias := NormalizeTrinoIdentifier(ctx.Identifier().GetText())
		
		// Check if we have column aliases
		var columnAliases []string
		if ctx.ColumnAliases() != nil {
			for _, column := range ctx.ColumnAliases().AllIdentifier() {
				columnAliases = append(columnAliases, NormalizeTrinoIdentifier(column.GetText()))
			}
		}
		
		// Check if we're already processed a physical/virtual table
		if len(l.context.referencesStack[0]) > 0 {
			lastRef := l.context.referencesStack[0][len(l.context.referencesStack[0])-1]
			
			// Update the last reference with alias information
			if physRef, ok := lastRef.(*base.PhysicalTableReference); ok {
				if len(columnAliases) > 0 {
					// Convert to virtual reference if we have column aliases
					virtualRef := &base.VirtualTableReference{
						Table:   alias,
						Columns: columnAliases,
					}
					// Replace the last item
					l.context.referencesStack[0][len(l.context.referencesStack[0])-1] = virtualRef
				} else {
					// Just set the alias
					physRef.Alias = alias
				}
			}
		}
	}
}

func prepareTrickyParserAndScanner(statement string, caretLine int, caretOffset int) (*trino.TrinoParser, *trino.TrinoLexer, *base.Scanner) {
	statement, caretLine, caretOffset = skipHeadingSQLs(statement, caretLine, caretOffset)
	statement, caretLine, caretOffset = skipHeadingSQLWithoutSemicolon(statement, caretLine, caretOffset)
	input := antlr.NewInputStream(statement)
	lexer := trino.NewTrinoLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := trino.NewTrinoParser(stream)
	parser.RemoveErrorListeners()
	lexer.RemoveErrorListeners()
	scanner := base.NewScanner(stream, true /* fillInput */)
	scanner.SeekPosition(caretLine, caretOffset)
	scanner.Push()
	return parser, lexer, scanner
}

func prepareParserAndScanner(statement string, caretLine int, caretOffset int) (*trino.TrinoParser, *trino.TrinoLexer, *base.Scanner) {
	statement, caretLine, caretOffset = skipHeadingSQLs(statement, caretLine, caretOffset)
	input := antlr.NewInputStream(statement)
	lexer := trino.NewTrinoLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := trino.NewTrinoParser(stream)
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
	lexer := trino.NewTrinoLexer(input)
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
		if token.GetTokenType() == trino.TrinoLexerSELECT_ && token.GetColumn() == 0 {
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

	// Check if the word is a Trino keyword
	// This is a comprehensive list of Trino reserved words
	keywords := map[string]bool{
		// SQL keywords
		"add": true, "admin": true, "all": true, "alter": true, "analyze": true,
		"and": true, "any": true, "array": true, "as": true, "asc": true,
		"at": true, "bernoulli": true, "between": true, "by": true, "call": true,
		"called": true, "cascade": true, "case": true, "cast": true, "catalogs": true,
		"column": true, "columns": true, "comment": true, "commit": true, "committed": true,
		"constraint": true, "create": true, "cross": true, "cube": true, "current": true,
		"current_date": true, "current_path": true, "current_role": true, "current_time": true, 
		"current_timestamp": true, "current_user": true, "data": true, "date": true, "day": true,
		"deallocate": true, "default": true, "define": true, "definer": true, "delete": true,
		"desc": true, "describe": true, "distinct": true, "distributed": true, "drop": true,
		"else": true, "end": true, "escape": true, "except": true, "excluding": true, "execute": true,
		"exists": true, "explain": true, "extract": true, "false": true, "fetch": true, 
		"filter": true, "first": true, "following": true, "for": true, "format": true, "from": true,
		"full": true, "function": true, "functions": true, "grant": true, "granted": true,
		"grants": true, "graphviz": true, "group": true, "grouping": true, "having": true,
		"hour": true, "if": true, "in": true, "including": true, "inner": true, "input": true,
		"insert": true, "intersect": true, "interval": true, "into": true, "invoker": true,
		"io": true, "is": true, "isolation": true, "join": true, "language": true, "last": true,
		"lateral": true, "left": true, "level": true, "like": true, "limit": true, "localtime": true,
		"localtimestamp": true, "logical": true, "map": true, "materialized": true, "merge": true,
		"minute": true, "month": true, "natural": true, "nested": true, "nfc": true, "nfd": true,
		"nfkc": true, "nfkd": true, "no": true, "none": true, "normalize": true, "not": true,
		"null": true, "nullif": true, "nulls": true, "offset": true, "on": true, "only": true,
		"option": true, "or": true, "order": true, "ordinality": true, "outer": true, "output": true,
		"over": true, "partition": true, "partitions": true, "position": true, "preceding": true,
		"prepare": true, "privileges": true, "properties": true, "range": true, "read": true,
		"recursive": true, "refresh": true, "rename": true, "repeatable": true, "replace": true,
		"reset": true, "respect": true, "restrict": true, "return": true, "returns": true,
		"revoke": true, "right": true, "rollback": true, "rollup": true, "row": true, "rows": true,
		"schema": true, "schemas": true, "second": true, "security": true, "select": true,
		"serializable": true, "session": true, "set": true, "sets": true, "show": true, "some": true,
		"start": true, "stats": true, "substring": true, "system": true, "table": true, "tables": true,
		"tablesample": true, "text": true, "then": true, "ties": true, "time": true, "timestamp": true,
		"to": true, "transaction": true, "true": true, "try_cast": true, "type": true, "uescape": true,
		"unbounded": true, "uncommitted": true, "union": true, "unnest": true, "update": true,
		"use": true, "user": true, "using": true, "validate": true, "values": true, "verbose": true,
		"view": true, "when": true, "where": true, "window": true, "with": true, "without": true,
		"work": true, "write": true, "year": true, "zone": true,
		
		// Trino-specific keywords
		"catalog": true, "vacuum": true, "optimize": true, 
		"storage": true,
	}

	if keywords[strings.ToLower(s)] {
		return fmt.Sprintf(`"%s"`, s)
	}

	return s
}

// isIdentifier is a helper function to check if a token is an identifier in Trino.
// Trino supports multiple types of identifiers:
// - Standard identifiers (letter followed by alphanumeric characters)
// - Quoted identifiers (in double quotes)
// - Backquoted identifiers (in backticks)
// - Digit identifiers (starting with a digit, which is valid in some contexts)
func isIdentifier(tokenType int) bool {
	return tokenType == trino.TrinoLexerIDENTIFIER_ ||
		tokenType == trino.TrinoLexerQUOTED_IDENTIFIER_ ||
		tokenType == trino.TrinoLexerBACKQUOTED_IDENTIFIER_ ||
		tokenType == trino.TrinoLexerDIGIT_IDENTIFIER_
}
