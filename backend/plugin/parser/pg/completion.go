package pg

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/antlr4-go/antlr/v4"
	pgparser "github.com/bytebase/omni/pg/parser"
	pg "github.com/bytebase/parser/postgresql"

	"github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	base.RegisterCompleteFunc(store.Engine_POSTGRES, Completion)
	base.RegisterCompleteFunc(store.Engine_SNOWFLAKE, Completion)
	base.RegisterCompleteFunc(store.Engine_COCKROACHDB, Completion)
}

// Completion is the entry point of PostgreSQL code completion.
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

// computeSQLAndByteOffset converts (statement, caretLine, caretOffset) to
// (trimmedSQL, byteOffset) for omni's parser.Collect API.
func computeSQLAndByteOffset(statement string, caretLine int, caretOffset int, tricky bool) (string, int) {
	var sql string
	var newLine, newOffset int
	if tricky {
		sql, newLine, newOffset = skipHeadingSQLs(statement, caretLine, caretOffset)
		sql, newLine, newOffset = skipHeadingSQLWithoutSemicolon(sql, newLine, newOffset)
	} else {
		sql, newLine, newOffset = skipHeadingSQLs(statement, caretLine, caretOffset)
	}
	return sql, lineColumnToByteOffset(sql, newLine, newOffset)
}

// lineColumnToByteOffset converts 1-based line and 0-based column (character count)
// to a byte offset. Column is measured in runes (characters), not bytes, so that
// multi-byte UTF-8 characters are handled correctly.
func lineColumnToByteOffset(sql string, line, column int) int {
	currentLine := 1
	for i := 0; i < len(sql); i++ {
		if currentLine == line {
			// Count 'column' runes from position i to get the byte offset.
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

// isNoSeparatorRequired returns true if tokenType is a punctuation/operator token
// that does not require whitespace separation from the next token.
func isNoSeparatorRequired(tokenType int) bool {
	switch tokenType {
	case '$', '(', ')', '[', ']', ',', ';', ':', '=', '.',
		'+', '-', '/', '^', '<', '>', '%', '*',
		pgparser.TYPECAST, pgparser.DOT_DOT, pgparser.COLON_EQUALS,
		pgparser.EQUALS_GREATER, pgparser.LESS_EQUALS,
		pgparser.GREATER_EQUALS, pgparser.NOT_EQUALS, pgparser.PARAM,
		pgparser.Op:
		return true
	}
	return false
}

type Completer struct {
	ctx               context.Context
	scene             base.SceneType
	sql               string           // the SQL statement (after skipHeadingSQLs)
	cursorByteOffset  int              // byte offset in sql for omni Collect
	tokens            []pgparser.Token // omni tokens for the SQL
	caretTokenIndex   int              // index in tokens at/near the caret
	instanceID        string
	defaultDatabase   string
	defaultSchema     string
	schemaNotSelected bool
	getMetadata       base.GetDatabaseMetadataFunc
	listDatabaseNames base.ListDatabaseNamesFunc
	metadataCache     map[string]*model.DatabaseMetadata
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
	sql, byteOffset := computeSQLAndByteOffset(statement, caretLine, caretOffset, true /* tricky */)
	tokens := pgparser.Tokenize(sql)
	caretTokenIndex := findCaretTokenIndex(tokens, byteOffset)
	defaultSchema := cCtx.DefaultSchema
	schemaNotSelected := defaultSchema == ""
	if defaultSchema == "" {
		defaultSchema = "public"
	}
	return &Completer{
		ctx:               ctx,
		scene:             cCtx.Scene,
		sql:               sql,
		cursorByteOffset:  byteOffset,
		tokens:            tokens,
		caretTokenIndex:   caretTokenIndex,
		instanceID:        cCtx.InstanceID,
		defaultDatabase:   cCtx.DefaultDatabase,
		defaultSchema:     defaultSchema,
		schemaNotSelected: schemaNotSelected,
		getMetadata:       cCtx.Metadata,
		listDatabaseNames: cCtx.ListDatabaseNames,
		metadataCache:     make(map[string]*model.DatabaseMetadata),
		cteCache:          make(map[int][]*base.VirtualTableReference),
	}
}

func NewStandardCompleter(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) *Completer {
	sql, byteOffset := computeSQLAndByteOffset(statement, caretLine, caretOffset, false /* tricky */)
	tokens := pgparser.Tokenize(sql)
	caretTokenIndex := findCaretTokenIndex(tokens, byteOffset)
	defaultSchema := cCtx.DefaultSchema
	schemaNotSelected := defaultSchema == ""
	if defaultSchema == "" {
		defaultSchema = "public"
	}
	return &Completer{
		ctx:               ctx,
		scene:             cCtx.Scene,
		sql:               sql,
		cursorByteOffset:  byteOffset,
		tokens:            tokens,
		caretTokenIndex:   caretTokenIndex,
		instanceID:        cCtx.InstanceID,
		defaultDatabase:   cCtx.DefaultDatabase,
		defaultSchema:     defaultSchema,
		schemaNotSelected: schemaNotSelected,
		getMetadata:       cCtx.Metadata,
		listDatabaseNames: cCtx.ListDatabaseNames,
		metadataCache:     make(map[string]*model.DatabaseMetadata),
		cteCache:          make(map[int][]*base.VirtualTableReference),
	}
}

// findCaretTokenIndex returns the index of the first token whose start (Loc)
// is at or past byteOffset. This matches ANTLR Scanner.SeekPosition semantics:
// the caret token is the one that starts at or after the cursor position.
// Uses >= so that when the cursor is exactly at a token boundary, we pick
// that token (not the one before it).
// Returns len(tokens) if all tokens are before byteOffset.
func findCaretTokenIndex(tokens []pgparser.Token, byteOffset int) int {
	for i, tok := range tokens {
		if tok.Loc >= byteOffset {
			return i
		}
	}
	return len(tokens)
}

func (c *Completer) completion() ([]base.Candidate, error) {
	// Check if the caret token is quoted.
	if c.caretTokenIndex < len(c.tokens) {
		tok := c.tokens[c.caretTokenIndex]
		if tok.Type == pgparser.IDENT && tok.Loc < len(c.sql) && c.sql[tok.Loc] == '"' {
			c.caretTokenIsQuoted = true
		}
	}

	caretIndex := c.caretTokenIndex
	if caretIndex > 0 && !isNoSeparatorRequired(c.tokens[caretIndex-1].Type) {
		caretIndex--
	}
	c.referencesStack = append([][]base.TableReference{{}}, c.referencesStack...)

	// Use omni parser to collect grammar candidates instead of C3.
	candidates := pgparser.Collect(c.sql, c.cursorByteOffset)

	// If Collect returned no rule candidates and the cursor is at the end of a
	// partial identifier token (prefix), retry Collect at the start of that
	// token.  This enables completion for "SELECT * FROM t|" style inputs
	// where the parser sees a complete token and doesn't know what grammar
	// rule to suggest.
	if len(candidates.Rules) == 0 {
		if prefixTok, ok := c.prefixToken(); ok {
			candidates = pgparser.Collect(c.sql, prefixTok.Loc)
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

// prefixToken returns the token immediately before (or containing) the cursor
// position when that token is an identifier-like token that the user is still
// typing.  Returns false if no such token exists.
func (c *Completer) prefixToken() (pgparser.Token, bool) {
	// caretTokenIndex points to the first token at or past cursorByteOffset.
	// The partial-prefix token is the one just before it (when the cursor is
	// right after it) or the token itself (when the cursor is inside it).
	idx := c.caretTokenIndex

	// Check if the cursor is inside the token at caretTokenIndex.
	if idx < len(c.tokens) {
		tok := c.tokens[idx]
		if tok.Loc < c.cursorByteOffset && c.cursorByteOffset <= tok.End && isIdentLikeToken(tok.Type) {
			return tok, true
		}
	}

	// Check the token before caretTokenIndex (cursor is right after it).
	if idx > 0 {
		tok := c.tokens[idx-1]
		if tok.End == c.cursorByteOffset && isIdentLikeToken(tok.Type) {
			return tok, true
		}
	}

	return pgparser.Token{}, false
}

// isIdentLikeToken returns true for token types that represent identifiers or
// unreserved keywords that a user might be partially typing.
func isIdentLikeToken(tokenType int) bool {
	return pgparser.IsIdentifierTokenType(tokenType)
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

func (m CompletionMap) insertSequencesWithPrefix(c *Completer, schemas map[string]bool, includeSchemaPrefix bool) {
	for schema := range schemas {
		if len(schema) == 0 {
			continue
		}
		for _, seq := range c.listSequences(schema) {
			text := c.quotedIdentifierIfNeeded(seq)
			if includeSchemaPrefix && schema != c.defaultSchema {
				text = c.quotedIdentifierIfNeeded(schema) + "." + text
			}
			m.Insert(base.Candidate{
				Type: base.CandidateTypeSequence,
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
			if tableMeta != nil {
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
				continue
			}
			// Check foreign tables.
			if extMeta := schemaMeta.GetExternalTable(table); extMeta != nil {
				for _, column := range extMeta.GetProto().GetColumns() {
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
				continue
			}
			// Check materialized views by resolving their definitions.
			if mvMeta := schemaMeta.GetMaterializedView(table); mvMeta != nil {
				if columns, err := c.resolveMaterializedViewColumns(mvMeta.GetDefinition(), schema, table); err == nil {
					for _, col := range columns {
						m.Insert(col)
					}
				}
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
		for _, ft := range schemaMeta.ListForeignTableNames() {
			extMeta := schemaMeta.GetExternalTable(ft)
			if extMeta == nil {
				continue
			}
			for _, column := range extMeta.GetProto().GetColumns() {
				definition := fmt.Sprintf("%s.%s | %s", schema, ft, column.Type)
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
		for _, mv := range schemaMeta.ListMaterializedViewNames() {
			mvMeta := schemaMeta.GetMaterializedView(mv)
			if mvMeta == nil {
				continue
			}
			if columns, err := c.resolveMaterializedViewColumns(mvMeta.GetDefinition(), schema, mv); err == nil {
				for _, col := range columns {
					m.Insert(col)
				}
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

func (c *Completer) convertCandidates(candidates *pgparser.CandidateSet) ([]base.Candidate, error) {
	keywordEntries := make(CompletionMap)
	runtimeFunctionEntries := make(CompletionMap)
	schemaEntries := make(CompletionMap)
	tableEntries := make(CompletionMap)
	columnEntries := make(CompletionMap)
	viewEntries := make(CompletionMap)
	sequenceEntries := make(CompletionMap)

	// Token candidates → keywords.
	// Skip single-char punctuation/operator tokens (they are not useful keyword completions).
	for _, tok := range candidates.Tokens {
		if tok > 0 && tok < 256 {
			continue
		}
		name := pgparser.TokenName(tok)
		if name == "" {
			continue
		}
		keywordEntries.Insert(base.Candidate{
			Type: base.CandidateTypeKeyword,
			Text: name,
		})
	}

	// Rule candidates → semantic objects
	for _, rc := range candidates.Rules {
		c.fetchCommonTableExpression(candidates.CTEPositions)

		switch rc.Rule {
		case "func_name":
			runtimeFunctionEntries.insertFunctions()
		case "relation_expr", "qualified_name", "any_name":
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
				sequenceEntries.insertSequencesWithPrefix(c, schemas, includeSchemaPrefix)
			}
		case "columnref":
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
					// Only suggest SELECT item aliases in ORDER BY / GROUP BY / HAVING.
					if c.isInAliasAllowedContext() {
						list := c.fetchSelectItemAliases(candidates.SelectAliasPositions)
						for _, alias := range list {
							columnEntries.Insert(base.Candidate{
								Type: base.CandidateTypeColumn,
								Text: c.quotedIdentifierIfNeeded(alias),
							})
						}
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

	var result []base.Candidate
	result = append(result, keywordEntries.toSlice()...)
	result = append(result, runtimeFunctionEntries.toSlice()...)
	result = append(result, schemaEntries.toSlice()...)
	result = append(result, tableEntries.toSlice()...)
	result = append(result, viewEntries.toSlice()...)
	result = append(result, sequenceEntries.toSlice()...)
	result = append(result, columnEntries.toSlice()...)

	return result, nil
}

func (c *Completer) fetchCommonTableExpression(ctePositions []int) {
	c.cteTables = nil
	for _, pos := range ctePositions {
		c.cteTables = append(c.cteTables, c.extractCTETables(pos)...)
	}
}

func (c *Completer) extractCTETables(pos int) []*base.VirtualTableReference {
	if metadata, exists := c.cteCache[pos]; exists {
		return metadata
	}
	if pos >= len(c.sql) {
		return nil
	}
	followingText := c.sql[pos:]
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
			base.Statement{Text: ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.Preparablestmt())},
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

// isInAliasAllowedContext checks if the caret is in a context where SELECT item
// aliases are valid completions: ORDER BY, GROUP BY, or HAVING clauses.
// Aliases are NOT allowed in WHERE, JOIN ON, FROM, or SELECT contexts.
func (c *Completer) isInAliasAllowedContext() bool {
	level := 0
	for i := c.caretTokenIndex - 1; i >= 0; i-- {
		switch c.tokens[i].Type {
		case ')':
			level++
		case '(':
			if level > 0 {
				level--
			} else {
				return false // inside a subquery — don't look further
			}
		default:
		}
		if level > 0 {
			continue // skip tokens inside nested parens
		}
		switch c.tokens[i].Type {
		case pgparser.ORDER, pgparser.GROUP_P, pgparser.HAVING:
			return true
		case pgparser.WHERE, pgparser.ON, pgparser.FROM, pgparser.SELECT,
			pgparser.LIMIT, pgparser.OFFSET, pgparser.WINDOW, pgparser.FOR:
			return false
		}
	}
	return false
}

func (c *Completer) fetchSelectItemAliases(aliasPositions []int) []string {
	aliasMap := make(map[string]bool)
	for _, pos := range aliasPositions {
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
}

func (c *Completer) extractAliasText(pos int) string {
	if pos >= len(c.sql) {
		return ""
	}
	followingText := c.sql[pos:]
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
	if ctx.Bare_col_label() != nil {
		l.result = normalizePostgreSQLBareColLabel(ctx.Bare_col_label())
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
	// Walk backward from the caret token to find qualifier.identifier pattern.
	// We look for: [identifier] [.] [identifier_at_caret]
	idx := c.caretTokenIndex

	// If the caret token is an identifier or ONLY keyword, we're on the trailing identifier.
	// Otherwise we might be right after the trailing identifier (e.g. after a dot).
	if idx < len(c.tokens) {
		tt := c.tokens[idx].Type
		if tt != pgparser.ONLY && !pgparser.IsIdentifierTokenType(tt) {
			idx--
		}
	} else {
		idx = len(c.tokens) - 1
	}

	if idx < 0 {
		return "", ObjectFlagsShowFirst | ObjectFlagsShowSecond
	}

	// Check if there's a dot before the current identifier, indicating a qualifier.
	if idx >= 2 && pgparser.IsIdentifierTokenType(c.tokens[idx].Type) &&
		c.tokens[idx-1].Type == '.' &&
		pgparser.IsIdentifierTokenType(c.tokens[idx-2].Type) {
		qualifier := normalizeIdentifier(c.tokens[idx-2].Str)
		return qualifier, ObjectFlagsShowSecond
	}

	// If caret is right after a dot (e.g. "schema.|"), the dot is at idx
	if idx >= 1 && c.tokens[idx].Type == '.' &&
		pgparser.IsIdentifierTokenType(c.tokens[idx-1].Type) {
		qualifier := normalizeIdentifier(c.tokens[idx-1].Str)
		return qualifier, ObjectFlagsShowSecond
	}

	return "", ObjectFlagsShowFirst | ObjectFlagsShowSecond
}

func (c *Completer) determineColumnRef() (schema, table string, flags ObjectFlags) {
	// Walk backward from the caret token to find schema.table.column pattern.
	idx := c.caretTokenIndex

	// If the caret token is not a dot and not an identifier, step back.
	if idx < len(c.tokens) {
		tt := c.tokens[idx].Type
		if tt != '.' && !pgparser.IsIdentifierTokenType(tt) {
			idx--
		}
	} else {
		idx = len(c.tokens) - 1
	}

	if idx < 0 {
		return "", "", ObjectFlagsShowSchemas | ObjectFlagsShowTables | ObjectFlagsShowColumns
	}

	// Walk backward to find the leftmost identifier in a dotted chain.
	// Possible patterns:
	//   column                    → show schemas, tables, columns
	//   table.column              → show tables, columns
	//   table.                    → show tables, columns
	//   schema.table.column       → show columns
	//   schema.table.             → show columns

	// Collect identifiers and dots backward.
	parts := []string{} // identifiers found, from right to left
	pos := idx

	// If current position is an identifier, collect it and move left.
	if pos >= 0 && pgparser.IsIdentifierTokenType(c.tokens[pos].Type) {
		parts = append(parts, normalizeIdentifier(c.tokens[pos].Str))
		pos--
	}

	// Check for dot + identifier patterns going left.
	for pos >= 1 && c.tokens[pos].Type == '.' && pgparser.IsIdentifierTokenType(c.tokens[pos-1].Type) {
		parts = append(parts, normalizeIdentifier(c.tokens[pos-1].Str))
		pos -= 2
	}

	// Handle trailing dot case: if current token is a dot, the parts collected
	// so far are qualifiers. E.g. "table." means table is a qualifier.
	if idx >= 0 && c.tokens[idx].Type == '.' {
		// The dot is the last token. Parts are everything before the dot.
		// e.g., for "schema.table." parts = ["table", "schema"]
		// We treat this as schema=schema, table=table, show columns.
		switch len(parts) {
		case 0:
			// Just a lone dot — shouldn't happen, but treat as no qualifier.
			return "", "", ObjectFlagsShowSchemas | ObjectFlagsShowTables | ObjectFlagsShowColumns
		case 1:
			// "table." → table=parts[0], show tables + columns
			return parts[0], parts[0], ObjectFlagsShowTables | ObjectFlagsShowColumns
		default:
			// "schema.table." → schema=parts[1], table=parts[0], show columns
			return parts[1], parts[0], ObjectFlagsShowColumns
		}
	}

	// Not a trailing dot case. Parts are from the identifier chain (right-to-left).
	// parts[0] = rightmost (column), parts[1] = next left (table or schema), etc.
	switch len(parts) {
	case 0:
		return "", "", ObjectFlagsShowSchemas | ObjectFlagsShowTables | ObjectFlagsShowColumns
	case 1:
		// Single identifier, e.g. "col" → show schemas, tables, columns
		return "", "", ObjectFlagsShowSchemas | ObjectFlagsShowTables | ObjectFlagsShowColumns
	case 2:
		// "table.column" → We don't know yet if parts[1] is a schema or table.
		// Return schema=table=parts[1] so the downstream code can check aliases
		// and resolve against both schema and table metadata.
		return parts[1], parts[1], ObjectFlagsShowTables | ObjectFlagsShowColumns
	default:
		// "schema.table.column" → schema=leftmost, table=second from left
		return parts[len(parts)-1], parts[len(parts)-2], ObjectFlagsShowColumns
	}
}

func normalizeIdentifier(tokenText string) string {
	if len(tokenText) >= 2 && tokenText[0] == '"' && tokenText[len(tokenText)-1] == '"' {
		return normalizePostgreSQLQuotedIdentifier(tokenText)
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
	level := 0
	for i := c.caretTokenIndex; i < len(c.tokens); i++ {
		switch c.tokens[i].Type {
		case '(':
			level++
		case ')':
			if level > 0 {
				level--
			}
		case pgparser.FROM:
			if level == 0 {
				c.parseTableReferences(c.sql[c.tokens[i].Loc:])
			}
		default:
		}
	}
}

func (c *Completer) collectLeadingTableReferences(caretIndex int) {
	level := 0
	for i := 0; i < len(c.tokens) && i < caretIndex; i++ {
		switch c.tokens[i].Type {
		case '(':
			level++
			c.referencesStack = append([][]base.TableReference{{}}, c.referencesStack...)
		case ')':
			if level == 0 {
				return // We cannot go above the initial nesting level.
			}
			level--
			c.referencesStack = c.referencesStack[1:]
		case pgparser.FROM:
			c.parseTableReferences(c.sql[c.tokens[i].Loc:])
		default:
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
					if span, err := GetQuerySpan(
						l.context.ctx,
						base.GetQuerySpanContext{
							InstanceID:              l.context.instanceID,
							GetDatabaseMetadataFunc: l.context.getMetadata,
							ListDatabaseNamesFunc:   l.context.listDatabaseNames,
						},
						base.Statement{Text: fmt.Sprintf("SELECT * FROM %s AS %s;", ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.Select_with_parens()), tableAlias)},
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
			// Other cases
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

// caretLine is 1-based and caretOffset is 0-based.
func skipHeadingSQLs(statement string, caretLine int, caretOffset int) (string, int, int) {
	newCaretLine, newCaretOffset := caretLine, caretOffset
	list, err := SplitSQL(statement)
	if err != nil || len(base.FilterEmptyStatements(list)) <= 1 {
		return statement, caretLine, caretOffset
	}

	start := 0
	for i, sql := range list {
		// Both caretLine and End.Line are 1-based
		// End.Column is 1-based exclusive (points after the last character)
		// caretOffset is 0-based
		sqlEndLine := int(sql.End.GetLine())
		sqlEndColumn := int(sql.End.GetColumn())
		// Use > for End.Column comparison because it's exclusive (points after last char)
		if sqlEndLine > caretLine || (sqlEndLine == caretLine && sqlEndColumn > caretOffset) {
			start = i
			if i == 0 {
				// The caret is in the first SQL statement, so we don't need to skip any SQL statements.
				break
			}
			previousSQLEndLine := int(list[i-1].End.GetLine())
			previousSQLEndColumn := int(list[i-1].End.GetColumn())
			newCaretLine = caretLine - previousSQLEndLine + 1
			if caretLine == previousSQLEndLine {
				// The caret is in the same line as the last line of the previous SQL statement.
				// We need to adjust the caret offset.
				// previousSQLEndColumn is 1-based exclusive, caretOffset is 0-based
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
	tokens := pgparser.Tokenize(statement)
	caretByteOff := lineColumnToByteOffset(statement, caretLine, caretOffset)

	latestSelectOffset := -1
	latestSelectLine := 0
	newCaretLine, newCaretOffset := caretLine, caretOffset

	for _, tok := range tokens {
		if tok.Loc >= caretByteOff {
			break
		}
		if tok.Type == pgparser.SELECT {
			// Check that this SELECT starts at column 0 of its line.
			// Either it's the first byte of the string, or the byte immediately
			// before it is a newline.
			atColumn0 := tok.Loc == 0 || statement[tok.Loc-1] == '\n'
			if atColumn0 {
				latestSelectOffset = tok.Loc
				// Compute the line number of this token.
				line := 1
				for j := 0; j < tok.Loc; j++ {
					if statement[j] == '\n' {
						line++
					}
				}
				latestSelectLine = line
				newCaretLine = caretLine - latestSelectLine + 1
				newCaretOffset = caretOffset
			}
		}
	}

	if latestSelectOffset < 0 {
		return statement, caretLine, caretOffset
	}

	return statement[latestSelectOffset:], newCaretLine, newCaretOffset
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

func (c *Completer) resolveMaterializedViewColumns(definition, schema, table string) ([]base.Candidate, error) {
	if len(definition) == 0 {
		return nil, nil
	}
	span, err := GetQuerySpan(
		c.ctx,
		base.GetQuerySpanContext{
			InstanceID:              c.instanceID,
			GetDatabaseMetadataFunc: c.getMetadata,
			ListDatabaseNamesFunc:   c.listDatabaseNames,
		},
		base.Statement{Text: definition + ";"},
		c.defaultDatabase,
		"",
		false,
	)
	if err != nil || span.NotFoundError != nil {
		return nil, err
	}
	var candidates []base.Candidate
	for _, col := range span.Results {
		candidates = append(candidates, base.Candidate{
			Type:       base.CandidateTypeColumn,
			Text:       c.quotedIdentifierIfNeeded(col.Name),
			Definition: fmt.Sprintf("%s.%s", schema, table),
		})
	}
	return candidates, nil
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

func (c *Completer) listSequences(schema string) []string {
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
	return schemaMeta.ListSequenceNames()
}

func (c *Completer) quotedIdentifierIfNeeded(s string) string {
	if c.caretTokenIsQuoted {
		return s
	}
	if strings.ToLower(s) != s {
		return fmt.Sprintf(`"%s"`, s)
	}
	if pgparser.IsReservedKeyword(strings.ToUpper(s)) {
		return fmt.Sprintf(`"%s"`, s)
	}
	// PostgreSQL requires double quotes for identifiers with special characters or start with digits
	if !isValidUnquotedIdentifier(s) {
		// If the identifier contains double quotes, we need to escape them by doubling them
		if strings.Contains(s, `"`) {
			s = strings.ReplaceAll(s, `"`, `""`)
		}
		return fmt.Sprintf(`"%s"`, s)
	}
	return s
}

// isValidUnquotedIdentifier checks if the identifier can be used without quotes in PostgreSQL.
// PostgreSQL unquoted identifiers must:
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
