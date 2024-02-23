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
	base.RegisterCompleteFunc(store.Engine_ORACLE, Completion)
	base.RegisterCompleteFunc(store.Engine_DM, Completion)
	base.RegisterCompleteFunc(store.Engine_OCEANBASE_ORACLE, Completion)
	base.RegisterCompleteFunc(store.Engine_SNOWFLAKE, Completion)
	base.RegisterCompleteFunc(store.Engine_MSSQL, Completion)
}

// Completion is the entry point of PostgreSQL code completion.
func Completion(ctx context.Context, statement string, caretLine int, caretOffset int, defaultDatabase string, metadata base.GetDatabaseMetadataFunc, l base.ListDatabaseNamesFunc) ([]base.Candidate, error) {
	completer := NewCompleter(ctx, statement, caretLine, caretOffset, defaultDatabase, metadata, l)
	return completer.completion()
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
	parser              *pg.PostgreSQLParser
	lexer               *pg.PostgreSQLLexer
	scanner             *base.Scanner
	defaultDatabase     string
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

func NewCompleter(ctx context.Context, statement string, caretLine int, caretOffset int, defaultDatabase string, getMetadata base.GetDatabaseMetadataFunc, _ base.ListDatabaseNamesFunc) *Completer {
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
	return &Completer{
		ctx:                 ctx,
		core:                core,
		parser:              parser,
		lexer:               lexer,
		scanner:             scanner,
		defaultDatabase:     defaultDatabase,
		getMetadata:         getMetadata,
		metadataCache:       make(map[string]*model.DatabaseMetadata),
		noSeparatorRequired: newNoSeparatorRequired(),
		cteCache:            make(map[int][]*base.VirtualTableReference),
	}
}

func (c *Completer) completion() ([]base.Candidate, error) {
	// Check the caret token is quoted or not.
	// This check should be done before checking the caret token is a separator or not.
	if c.scanner.IsTokenType(pg.PostgreSQLLexerQuotedIdentifier) ||
		c.scanner.IsTokenType(pg.PostgreSQLLexerInvalidQuotedIdentifier) ||
		c.scanner.IsTokenType(pg.PostgreSQLLexerUnicodeQuotedIdentifier) {
		c.caretTokenIsQuoted = true
	}

	caretIndex := c.scanner.GetIndex()
	if caretIndex > 0 && !c.noSeparatorRequired[c.scanner.GetPreviousTokenType(false /* skipHidden */)] {
		caretIndex--
	}
	c.referencesStack = append([][]base.TableReference{{}}, c.referencesStack...)
	c.parser.Reset()
	context := c.parser.Root()

	candidates := c.core.CollectCandidates(caretIndex, context)

	for ruleName := range candidates.Rules {
		if ruleName == pg.PostgreSQLParserRULE_columnref {
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
	for _, name := range pg.GetBuiltinFunctions() {
		m.Insert(base.Candidate{
			Type: base.CandidateTypeFunction,
			Text: name + "()",
		})
	}
}

func (m CompletionMap) insertSchemas(c *Completer) {
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
		_, metadata, err := c.getMetadata(c.ctx, c.defaultDatabase)
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
				if len(column.Classification) != 0 {
					comment = column.Classification + "\n" + column.UserComment
				}
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
	defaultSchema := "public"
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
			runtimeFunctionEntries.Insert(base.Candidate{
				Type: base.CandidateTypeFunction,
				Text: strings.ToLower(entry) + "()",
			})
		default:
			keywordEntries.Insert(base.Candidate{
				Type: base.CandidateTypeKeyword,
				Text: c.quotedIdentifierIfNeeded(entry),
			})
		}
	}

	for candidate := range candidates.Rules {
		c.scanner.PopAndRestore()
		c.scanner.Push()

		c.fetchCommonTableExpression(candidates.Rules[candidate])

		switch candidate {
		case pg.PostgreSQLParserRULE_func_name:
			runtimeFunctionEntries.insertFunctions()
		case pg.PostgreSQLParserRULE_relation_expr, pg.PostgreSQLParserRULE_qualified_name:
			qualifier, flags := c.determineQualifiedName()

			if flags&ObjectFlagsShowFirst != 0 {
				schemaEntries.insertSchemas(c)
			}

			if flags&ObjectFlagsShowSecond != 0 {
				schemas := make(map[string]bool)
				if len(qualifier) == 0 {
					schemas[defaultSchema] = true
					// User didn't specify the schema, we need to append cte tables.
					schemas[""] = true
				} else {
					schemas[qualifier] = true
				}

				tableEntries.insertTables(c, schemas)
				viewEntries.insertViews(c, schemas)
			}
		case pg.PostgreSQLParserRULE_columnref:
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
				schemas[defaultSchema] = true
				// User didn't specify the schema, we need to append cte tables.
				schemas[""] = true
			}

			if flags&ObjectFlagsShowTables != 0 {
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
				if schema == table {
					schemas[defaultSchema] = true
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
	result = append(result, schemaEntries.toSlice()...)
	result = append(result, tableEntries.toSlice()...)
	result = append(result, viewEntries.toSlice()...)
	result = append(result, columnEntries.toSlice()...)

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
		if span, err := base.GetQuerySpan(
			l.context.ctx,
			store.Engine_POSTGRES,
			ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.Preparablestmt()),
			l.context.defaultDatabase,
			"",
			l.context.getMetadata,
			l.context.listDatabaseNames,
			false,
		); err == nil && len(span) == 1 {
			for _, column := range span[0].Results {
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
	position := c.scanner.GetIndex()
	if c.scanner.GetTokenChannel() != 0 {
		c.scanner.Forward(true /* skipHidden */)
	}

	if !c.scanner.IsTokenType(pg.PostgreSQLLexerONLY) && !c.lexer.IsIdentifier(c.scanner.GetTokenType()) {
		// We are at the end of an incomplete identifier spec.
		// Jump back.
		c.scanner.Backward(true /* skipHidden */)
	}

	// Go left until we hit a non-identifier token.
	if position > 0 {
		if c.lexer.IsIdentifier(c.scanner.GetTokenType()) && c.scanner.GetPreviousTokenType(false /* skipHidden */) == pg.PostgreSQLLexerDOT {
			c.scanner.Backward(true /* skipHidden */)
		}
		if c.scanner.IsTokenType(pg.PostgreSQLLexerDOT) && c.lexer.IsIdentifier(c.scanner.GetPreviousTokenType(false /* skipHidden */)) {
			c.scanner.Backward(true /* skipHidden */)
		}
	}

	// The current token is on the leading identifier.
	qualifier := ""
	temp := ""
	if c.lexer.IsIdentifier(c.scanner.GetTokenType()) {
		temp = unquote(c.scanner.GetTokenText())
		c.scanner.Forward(true /* skipHidden */)
	}

	if !c.scanner.IsTokenType(pg.PostgreSQLLexerDOT) || position <= c.scanner.GetIndex() {
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
	if tokenType != pg.PostgreSQLLexerDOT && !c.lexer.IsIdentifier(c.scanner.GetTokenType()) {
		// We are at the end of an incomplete identifier spec.
		// Jump back.
		c.scanner.Backward(true /* skipHidden */)
	}

	if position > 0 {
		if c.lexer.IsIdentifier(c.scanner.GetTokenType()) && c.scanner.GetPreviousTokenType(false /* skipHidden */) == pg.PostgreSQLLexerDOT {
			c.scanner.Backward(true /* skipHidden */)
		}
		if c.scanner.IsTokenType(pg.PostgreSQLLexerDOT) && c.lexer.IsIdentifier(c.scanner.GetPreviousTokenType(false /* skipHidden */)) {
			c.scanner.Backward(true /* skipHidden */)

			if c.scanner.GetPreviousTokenType(false /* skipHidden */) == pg.PostgreSQLLexerDOT {
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
		temp = unquote(c.scanner.GetTokenText())
		c.scanner.Forward(true /* skipHidden */)
	}

	if !c.scanner.IsTokenType(pg.PostgreSQLLexerDOT) || position <= c.scanner.GetIndex() {
		return schema, table, ObjectFlagsShowSchemas | ObjectFlagsShowTables | ObjectFlagsShowColumns
	}

	c.scanner.Forward(true /* skipHidden */) // skip dot
	table = temp
	schema = temp
	if c.lexer.IsIdentifier(c.scanner.GetTokenType()) {
		temp = unquote(c.scanner.GetTokenText())
		c.scanner.Forward(true /* skipHidden */)

		if !c.scanner.IsTokenType(pg.PostgreSQLLexerDOT) || position <= c.scanner.GetIndex() {
			return schema, table, ObjectFlagsShowTables | ObjectFlagsShowColumns
		}

		table = temp
		return schema, table, ObjectFlagsShowColumns
	}

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
	c.scanner.Push()

	level := 0
	for {
		found := c.scanner.GetTokenType() == pg.PostgreSQLLexerFROM
		for !found {
			if !c.scanner.Forward(false /* skipHidden */) {
				break
			}

			switch c.scanner.GetTokenType() {
			case pg.PostgreSQLLexerOPEN_PAREN:
				level++
			case pg.PostgreSQLLexerCLOSE_PAREN:
				if level > 0 {
					level--
				}
			case pg.PostgreSQLLexerFROM:
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
		if c.scanner.GetTokenType() == pg.PostgreSQLLexerFROM {
			c.scanner.Forward(false /* skipHidden */)
		}
	}
}

func (c *Completer) collectLeadingTableReferences(caretIndex int) {
	c.scanner.Push()

	c.scanner.SeekIndex(0)

	level := 0
	for {
		found := c.scanner.GetTokenType() == pg.PostgreSQLLexerFROM
		for !found {
			if !c.scanner.Forward(false /* skipHidden */) || c.scanner.GetIndex() >= caretIndex {
				break
			}

			switch c.scanner.GetTokenType() {
			case pg.PostgreSQLLexerOPEN_PAREN:
				level++
				c.referencesStack = append([][]base.TableReference{{}}, c.referencesStack...)
			case pg.PostgreSQLLexerCLOSE_PAREN:
				if level == 0 {
					c.scanner.PopAndRestore()
					return // We cannot go above the initial nesting level.
				}

				level--
				c.referencesStack = c.referencesStack[1:]
			case pg.PostgreSQLLexerFROM:
				found = true
			}
		}

		if !found {
			c.scanner.PopAndRestore()
			return // No more FROM clauses found.
		}

		c.parseTableReferences(c.scanner.GetFollowingText())
		if c.scanner.GetTokenType() == pg.PostgreSQLLexerFROM {
			c.scanner.Forward(false /* skipHidden */)
		}
	}
}

func (c *Completer) parseTableReferences(fromClause string) {
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
	antlr.ParseTreeWalkerDefault.Walk(listener, tree)
}

type TableRefListener struct {
	*pg.BasePostgreSQLParserListener

	context *Completer
	level   int
}

func (l *TableRefListener) EnterTable_ref(ctx *pg.Table_refContext) {
	if _, ok := ctx.GetParent().(*pg.Table_refContext); ok {
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
			list := NormalizePostgreSQLQualifiedName(ctx.Relation_expr().Qualified_name())
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
					if span, err := base.GetQuerySpan(
						l.context.ctx,
						store.Engine_POSTGRES,
						fmt.Sprintf("SELECT * FROM %s AS %s;", ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.Select_with_parens()), tableAlias),
						l.context.defaultDatabase,
						"",
						l.context.getMetadata,
						l.context.listDatabaseNames,
						false,
					); err == nil && len(span) == 1 {
						for _, column := range span[0].Results {
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
		}
	}
}

func (l *TableRefListener) ExitTable_ref(ctx *pg.Table_refContext) {
	if _, ok := ctx.GetParent().(*pg.Table_refContext); ok {
		l.level--
	}
}

func (l *TableRefListener) EnterSelect_with_parens(_ *pg.Select_with_parensContext) {
	l.level++
}

func (l *TableRefListener) ExitSelect_with_parens(_ *pg.Select_with_parensContext) {
	l.level--
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

func (c *Completer) listAllSchemas() []string {
	if _, exists := c.metadataCache[c.defaultDatabase]; !exists {
		_, metadata, err := c.getMetadata(c.ctx, c.defaultDatabase)
		if err != nil || metadata == nil {
			return nil
		}
		c.metadataCache[c.defaultDatabase] = metadata
	}

	return c.metadataCache[c.defaultDatabase].ListSchemaNames()
}

func (c *Completer) listTables(schema string) []string {
	if _, exists := c.metadataCache[c.defaultDatabase]; !exists {
		_, metadata, err := c.getMetadata(c.ctx, c.defaultDatabase)
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
		_, metadata, err := c.getMetadata(c.ctx, c.defaultDatabase)
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
		_, metadata, err := c.getMetadata(c.ctx, c.defaultDatabase)
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
		_, metadata, err := c.getMetadata(c.ctx, c.defaultDatabase)
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

	// TODO(rebelice): check reserved keywords here.
	return s
}
