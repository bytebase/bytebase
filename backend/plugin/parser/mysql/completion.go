package mysql

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/bytebase/omni/mysql/ast"
	mysqlparser "github.com/bytebase/omni/mysql/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	base.RegisterCompleteFunc(storepb.Engine_MYSQL, Completion)
	base.RegisterCompleteFunc(storepb.Engine_MARIADB, Completion)
	base.RegisterCompleteFunc(storepb.Engine_TIDB, Completion)
	base.RegisterCompleteFunc(storepb.Engine_OCEANBASE, Completion)
	base.RegisterCompleteFunc(storepb.Engine_CLICKHOUSE, Completion)
	base.RegisterCompleteFunc(storepb.Engine_STARROCKS, Completion)
	base.RegisterCompleteFunc(storepb.Engine_DORIS, Completion)
}

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

func isNoSeparatorRequired(tokenType int) bool {
	switch tokenType {
	case '.', '(', ')', ',', ';', '=', '<', '>', '+', '-', '*', '/', '%', '^', '~', '!', '@':
		return true
	}
	return false
}

type Completer struct {
	ctx                context.Context
	scene              base.SceneType
	sql                string
	cursorByteOffset   int
	tokens             []mysqlparser.Token
	caretTokenIndex    int
	instanceID         string
	defaultDatabase    string
	getMetadata        base.GetDatabaseMetadataFunc
	listDatabaseNames  base.ListDatabaseNamesFunc
	metadataCache      map[string]*model.DatabaseMetadata
	referencesStack    [][]base.TableReference
	references         []base.TableReference
	cteCache           map[int][]*base.VirtualTableReference
	cteTables          []*base.VirtualTableReference
	caretTokenIsQuoted bool
}

func NewStandardCompleter(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) *Completer {
	sql, byteOffset := computeSQLAndByteOffset(statement, caretLine, caretOffset, false)
	tokens := mysqlparser.Tokenize(sql)
	caretTokenIndex := findCaretTokenIndex(tokens, byteOffset)
	return &Completer{
		ctx:               ctx,
		scene:             cCtx.Scene,
		sql:               sql,
		cursorByteOffset:  byteOffset,
		tokens:            tokens,
		caretTokenIndex:   caretTokenIndex,
		instanceID:        cCtx.InstanceID,
		defaultDatabase:   cCtx.DefaultDatabase,
		getMetadata:       cCtx.Metadata,
		listDatabaseNames: cCtx.ListDatabaseNames,
		metadataCache:     make(map[string]*model.DatabaseMetadata),
		cteCache:          make(map[int][]*base.VirtualTableReference),
	}
}

func NewTrickyCompleter(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) *Completer {
	sql, byteOffset := computeSQLAndByteOffset(statement, caretLine, caretOffset, true)
	tokens := mysqlparser.Tokenize(sql)
	caretTokenIndex := findCaretTokenIndex(tokens, byteOffset)
	return &Completer{
		ctx:               ctx,
		scene:             cCtx.Scene,
		sql:               sql,
		cursorByteOffset:  byteOffset,
		tokens:            tokens,
		caretTokenIndex:   caretTokenIndex,
		instanceID:        cCtx.InstanceID,
		defaultDatabase:   cCtx.DefaultDatabase,
		getMetadata:       cCtx.Metadata,
		listDatabaseNames: cCtx.ListDatabaseNames,
		metadataCache:     make(map[string]*model.DatabaseMetadata),
		cteCache:          make(map[int][]*base.VirtualTableReference),
	}
}

func findCaretTokenIndex(tokens []mysqlparser.Token, byteOffset int) int {
	for i, tok := range tokens {
		if tok.Loc >= byteOffset {
			return i
		}
	}
	return len(tokens)
}

func (c *Completer) completion() ([]base.Candidate, error) {
	if c.caretTokenIndex < len(c.tokens) {
		tok := c.tokens[c.caretTokenIndex]
		if mysqlparser.IsIdentTokenType(tok.Type) && tok.Loc < len(c.sql) && c.sql[tok.Loc] == '`' {
			c.caretTokenIsQuoted = true
		}
	}

	caretIndex := c.caretTokenIndex
	if caretIndex > 0 && !isNoSeparatorRequired(c.tokens[caretIndex-1].Type) {
		caretIndex--
	}
	c.referencesStack = append([][]base.TableReference{{}}, c.referencesStack...)

	candidates := mysqlparser.Collect(c.sql, c.cursorByteOffset)

	if len(candidates.Rules) == 0 {
		if prefixTok, ok := c.prefixToken(); ok {
			candidates = mysqlparser.Collect(c.sql, prefixTok.Loc)
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

func (c *Completer) prefixToken() (mysqlparser.Token, bool) {
	idx := c.caretTokenIndex
	if idx < len(c.tokens) {
		tok := c.tokens[idx]
		if tok.Loc < c.cursorByteOffset && c.cursorByteOffset <= tok.End && mysqlparser.IsIdentTokenType(tok.Type) {
			return tok, true
		}
	}
	if idx > 0 {
		tok := c.tokens[idx-1]
		if tok.End == c.cursorByteOffset && mysqlparser.IsIdentTokenType(tok.Type) {
			return tok, true
		}
	}
	return mysqlparser.Token{}, false
}

type CompletionMap map[string]base.Candidate

func (m CompletionMap) Insert(entry base.Candidate) {
	m[entry.String()] = entry
}

func (m CompletionMap) insertFunctions() {
	for _, name := range getMySQLBuiltinFunctions() {
		m.Insert(base.Candidate{
			Type: base.CandidateTypeFunction,
			Text: name + "()",
		})
	}
}

func (m CompletionMap) insertDatabases(c *Completer) {
	for _, database := range c.listAllDatabases() {
		m.Insert(base.Candidate{
			Type: base.CandidateTypeDatabase,
			Text: database,
		})
	}
}

func (m CompletionMap) insertTables(c *Completer, schemas map[string]bool) {
	for schema := range schemas {
		if len(schema) == 0 {
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

func (m CompletionMap) insertColumns(c *Completer, databases, tables map[string]bool) {
	for database := range databases {
		if len(database) == 0 {
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
		if _, exists := c.metadataCache[database]; !exists {
			_, metadata, err := c.getMetadata(c.ctx, c.instanceID, database)
			if err != nil || metadata == nil {
				continue
			}
			c.metadataCache[database] = metadata
		}

		for table := range tables {
			tableMeta := c.metadataCache[database].GetSchemaMetadata("").GetTable(table)
			if tableMeta == nil {
				continue
			}
			for _, column := range tableMeta.GetProto().GetColumns() {
				definition := fmt.Sprintf("%s | %s", table, column.Type)
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
				definition := fmt.Sprintf("%s | %s", table, column.Type)
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

func (c *Completer) convertCandidates(candidates *mysqlparser.CandidateSet) ([]base.Candidate, error) {
	keywordEntries := make(CompletionMap)
	runtimeFunctionEntries := make(CompletionMap)
	databaseEntries := make(CompletionMap)
	tableEntries := make(CompletionMap)
	columnEntries := make(CompletionMap)
	viewEntries := make(CompletionMap)

	// Token candidates → keywords.
	for _, tok := range candidates.Tokens {
		if tok > 0 && tok < 256 {
			continue
		}
		name := mysqlparser.TokenName(tok)
		if name == "" {
			continue
		}
		keywordEntries.Insert(base.Candidate{
			Type: base.CandidateTypeKeyword,
			Text: name,
		})
	}

	for _, rc := range candidates.Rules {
		c.fetchCommonTableExpression(candidates.CTEPositions)

		switch rc.Rule {
		case "func_name":
			runtimeFunctionEntries.insertFunctions()
		case "database_ref":
			databaseEntries.insertDatabases(c)
		case "table_ref":
			qualifier, flags := c.determineQualifier()

			if flags&ObjectFlagsShowFirst != 0 {
				databaseEntries.insertDatabases(c)
			}

			if flags&ObjectFlagsShowSecond != 0 {
				schemas := make(map[string]bool)
				if len(qualifier) == 0 {
					schemas[c.defaultDatabase] = true
					schemas[""] = true // CTE tables
				} else {
					schemas[qualifier] = true
				}
				tableEntries.insertTables(c, schemas)
				viewEntries.insertViews(c, schemas)
			}
		case "columnref":
			schema, table, flags := c.determineColumnRef()

			if flags&ObjectFlagsShowSchemas != 0 {
				databaseEntries.insertDatabases(c)
			}

			databases := make(map[string]bool)
			if len(schema) != 0 {
				databases[schema] = true
			} else if len(c.references) > 0 {
				for _, reference := range c.references {
					if physicalTable, ok := reference.(*base.PhysicalTableReference); ok {
						if len(physicalTable.Database) > 0 {
							databases[physicalTable.Database] = true
						}
					}
				}
			}

			if len(schema) == 0 {
				databases[c.defaultDatabase] = true
				databases[""] = true // CTE tables
			}

			if flags&ObjectFlagsShowTables != 0 {
				tableEntries.insertTables(c, databases)
				viewEntries.insertViews(c, databases)

				for _, reference := range c.references {
					switch reference := reference.(type) {
					case *base.PhysicalTableReference:
						if len(schema) == 0 && len(reference.Database) == 0 || databases[reference.Database] {
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
					databases[c.defaultDatabase] = true
					databases[""] = true
				}

				tables := make(map[string]bool)
				if len(table) != 0 {
					tables[table] = true

					for _, reference := range c.references {
						switch reference := reference.(type) {
						case *base.PhysicalTableReference:
							if reference.Alias == table {
								tables[reference.Table] = true
								databases[reference.Database] = true
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
							databases[reference.Database] = true
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
					columnEntries.insertAllColumns(c)
				}

				if len(tables) > 0 {
					columnEntries.insertColumns(c, databases, tables)
				}
			}
		default:
		}
	}

	var result []base.Candidate
	result = append(result, keywordEntries.toSlice()...)
	result = append(result, runtimeFunctionEntries.toSlice()...)
	result = append(result, databaseEntries.toSlice()...)
	result = append(result, tableEntries.toSlice()...)
	result = append(result, viewEntries.toSlice()...)
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

	const suffix = " SELECT 1"
	tokens := mysqlparser.Tokenize(followingText)

	var selStmt *ast.SelectStmt
	var wrappedSQL string
	for end := len(tokens); end > 0; end-- {
		if end < len(tokens) {
			lastTok := tokens[end-1]
			wrappedSQL = followingText[:lastTok.End] + suffix
		} else {
			wrappedSQL = followingText + suffix
		}
		parsed, err := ParseMySQLOmni(wrappedSQL)
		if err != nil || parsed == nil || len(parsed.Items) == 0 {
			continue
		}
		sel, ok := parsed.Items[0].(*ast.SelectStmt)
		if ok && len(sel.CTEs) > 0 {
			selStmt = sel
			break
		}
	}
	if selStmt == nil {
		c.cteCache[pos] = nil
		return nil
	}

	var tables []*base.VirtualTableReference
	for _, cte := range selStmt.CTEs {
		table := &base.VirtualTableReference{
			Table: cte.Name,
		}
		if len(cte.Columns) > 0 {
			table.Columns = cte.Columns
		} else if cte.Select != nil {
			// Extract query text from wrappedSQL using Loc.
			loc := cte.Select.Loc
			if loc.Start >= 0 && loc.End > loc.Start && loc.End <= len(wrappedSQL) {
				queryText := wrappedSQL[loc.Start:loc.End]
				if span, err := GetQuerySpan(
					c.ctx,
					base.GetQuerySpanContext{
						InstanceID:              c.instanceID,
						GetDatabaseMetadataFunc: c.getMetadata,
						ListDatabaseNamesFunc:   c.listDatabaseNames,
					},
					base.Statement{Text: queryText},
					c.defaultDatabase,
					"",
					false,
				); err == nil && span.NotFoundError == nil {
					for _, column := range span.Results {
						table.Columns = append(table.Columns, column.Name)
					}
				}
			}
		}
		tables = append(tables, table)
	}

	c.cteCache[pos] = tables
	return tables
}

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
				return false
			}
		default:
		}
		if level > 0 {
			continue
		}
		switch c.tokens[i].Type {
		case mysqlparser.ORDER, mysqlparser.GROUP, mysqlparser.HAVING:
			return true
		case mysqlparser.WHERE, mysqlparser.ON, mysqlparser.FROM, mysqlparser.SELECT,
			mysqlparser.LIMIT, mysqlparser.FOR:
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

	tokens := mysqlparser.Tokenize(followingText)
	if len(tokens) == 0 {
		return ""
	}
	idx := 0
	if tokens[0].Type == mysqlparser.AS && len(tokens) > 1 {
		idx = 1
	}
	if idx >= len(tokens) || !mysqlparser.IsIdentTokenType(tokens[idx].Type) {
		return ""
	}
	return unquote(tokens[idx].Str)
}

type ObjectFlags int

const (
	ObjectFlagsShowSchemas ObjectFlags = 1 << iota
	ObjectFlagsShowTables
	ObjectFlagsShowColumns
	ObjectFlagsShowFirst
	ObjectFlagsShowSecond
)

func (c *Completer) determineQualifier() (string, ObjectFlags) {
	idx := c.caretTokenIndex
	if idx < len(c.tokens) && !mysqlparser.IsIdentTokenType(c.tokens[idx].Type) {
		idx--
	} else if idx >= len(c.tokens) {
		idx = len(c.tokens) - 1
	}

	if idx < 0 {
		return "", ObjectFlagsShowFirst | ObjectFlagsShowSecond
	}

	if idx >= 2 && mysqlparser.IsIdentTokenType(c.tokens[idx].Type) &&
		c.tokens[idx-1].Type == '.' &&
		mysqlparser.IsIdentTokenType(c.tokens[idx-2].Type) {
		qualifier := unquote(c.tokens[idx-2].Str)
		return qualifier, ObjectFlagsShowSecond
	}

	if idx >= 1 && c.tokens[idx].Type == '.' &&
		mysqlparser.IsIdentTokenType(c.tokens[idx-1].Type) {
		qualifier := unquote(c.tokens[idx-1].Str)
		return qualifier, ObjectFlagsShowSecond
	}

	return "", ObjectFlagsShowFirst | ObjectFlagsShowSecond
}

func (c *Completer) determineColumnRef() (schema, table string, flags ObjectFlags) {
	idx := c.caretTokenIndex
	if idx < len(c.tokens) {
		tt := c.tokens[idx].Type
		if tt != '.' && !mysqlparser.IsIdentTokenType(tt) {
			idx--
		}
	} else {
		idx = len(c.tokens) - 1
	}

	if idx < 0 {
		return "", "", ObjectFlagsShowSchemas | ObjectFlagsShowTables | ObjectFlagsShowColumns
	}

	parts := []string{}
	pos := idx

	if pos >= 0 && mysqlparser.IsIdentTokenType(c.tokens[pos].Type) {
		parts = append(parts, unquote(c.tokens[pos].Str))
		pos--
	}

	for pos >= 1 && c.tokens[pos].Type == '.' && mysqlparser.IsIdentTokenType(c.tokens[pos-1].Type) {
		parts = append(parts, unquote(c.tokens[pos-1].Str))
		pos -= 2
	}

	if idx >= 0 && c.tokens[idx].Type == '.' {
		switch len(parts) {
		case 0:
			return "", "", ObjectFlagsShowSchemas | ObjectFlagsShowTables | ObjectFlagsShowColumns
		case 1:
			return parts[0], parts[0], ObjectFlagsShowTables | ObjectFlagsShowColumns
		default:
			return parts[1], parts[0], ObjectFlagsShowColumns
		}
	}

	switch len(parts) {
	case 0:
		return "", "", ObjectFlagsShowSchemas | ObjectFlagsShowTables | ObjectFlagsShowColumns
	case 1:
		return "", "", ObjectFlagsShowSchemas | ObjectFlagsShowTables | ObjectFlagsShowColumns
	case 2:
		return parts[1], parts[1], ObjectFlagsShowTables | ObjectFlagsShowColumns
	default:
		return parts[len(parts)-1], parts[len(parts)-2], ObjectFlagsShowColumns
	}
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
		case mysqlparser.FROM:
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
				return
			}
			level--
			c.referencesStack = c.referencesStack[1:]
		case mysqlparser.FROM:
			c.parseTableReferences(c.sql[c.tokens[i].Loc:])
		case mysqlparser.INSERT, mysqlparser.INTO:
			c.parseInsertTableReferences(c.sql[c.tokens[i].Loc:])
		default:
		}
	}
}

func (c *Completer) parseTableReferences(fromClause string) {
	const prefix = "SELECT * "
	tokens := mysqlparser.Tokenize(fromClause)

	var selStmt *ast.SelectStmt
	var wrappedSQL string
	for end := len(tokens); end > 0; end-- {
		if end < len(tokens) {
			lastTok := tokens[end-1]
			wrappedSQL = prefix + fromClause[:lastTok.End]
		} else {
			wrappedSQL = prefix + fromClause
		}
		parsed, err := ParseMySQLOmni(wrappedSQL)
		if err != nil || parsed == nil || len(parsed.Items) == 0 {
			continue
		}
		sel, ok := parsed.Items[0].(*ast.SelectStmt)
		if ok && len(sel.From) > 0 {
			selStmt = sel
			break
		}
	}
	if selStmt == nil {
		return
	}

	for _, item := range selStmt.From {
		c.extractFromItem(item, wrappedSQL)
	}
}

func (c *Completer) parseInsertTableReferences(insertClause string) {
	tokens := mysqlparser.Tokenize(insertClause)
	idx := 0

	// Skip INSERT keyword.
	if idx < len(tokens) && tokens[idx].Type == mysqlparser.INSERT {
		idx++
	}
	// Skip INTO keyword.
	if idx < len(tokens) && tokens[idx].Type == mysqlparser.INTO {
		idx++
	}
	if idx >= len(tokens) || !mysqlparser.IsIdentTokenType(tokens[idx].Type) {
		return
	}

	ref := &base.PhysicalTableReference{}
	ref.Table = unquote(tokens[idx].Str)
	idx++

	if idx+1 < len(tokens) && tokens[idx].Type == '.' && mysqlparser.IsIdentTokenType(tokens[idx+1].Type) {
		ref.Database = ref.Table
		ref.Table = unquote(tokens[idx+1].Str)
	}

	if ref.Database == "" {
		ref.Database = c.defaultDatabase
	}
	c.referencesStack[0] = append(c.referencesStack[0], ref)
}

func (c *Completer) extractFromItem(item ast.TableExpr, wrappedSQL string) {
	switch v := item.(type) {
	case *ast.TableRef:
		ref := &base.PhysicalTableReference{
			Database: v.Schema,
			Table:    v.Name,
			Alias:    v.Alias,
		}
		if ref.Database == "" {
			ref.Database = c.defaultDatabase
		}
		c.referencesStack[0] = append(c.referencesStack[0], ref)

	case *ast.JoinClause:
		if v.Left != nil {
			c.extractFromItem(v.Left, wrappedSQL)
		}
		if v.Right != nil {
			c.extractFromItem(v.Right, wrappedSQL)
		}

	case *ast.SubqueryExpr:
		if v.Alias == "" {
			return
		}
		virtualRef := &base.VirtualTableReference{
			Table: v.Alias,
		}
		if len(v.Columns) > 0 {
			virtualRef.Columns = v.Columns
		} else if v.Select != nil {
			loc := v.Select.Loc
			if loc.Start >= 0 && loc.End > loc.Start && loc.End <= len(wrappedSQL) {
				subqueryText := wrappedSQL[loc.Start:loc.End]
				if span, err := GetQuerySpan(
					c.ctx,
					base.GetQuerySpanContext{
						InstanceID:              c.instanceID,
						GetDatabaseMetadataFunc: c.getMetadata,
						ListDatabaseNamesFunc:   c.listDatabaseNames,
					},
					base.Statement{Text: fmt.Sprintf("SELECT * FROM (%s) AS %s;", subqueryText, v.Alias)},
					c.defaultDatabase,
					"",
					false,
				); err == nil && span.NotFoundError == nil {
					for _, column := range span.Results {
						virtualRef.Columns = append(virtualRef.Columns, column.Name)
					}
				}
			}
		}
		c.referencesStack[0] = append(c.referencesStack[0], virtualRef)

	default:
	}
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
		sqlEndLine := int(sql.End.GetLine())
		sqlEndColumn := int(sql.End.GetColumn())
		if sqlEndLine > caretLine || (sqlEndLine == caretLine && sqlEndColumn > caretOffset) {
			start = i
			if i == 0 {
				break
			}
			previousSQLEndLine := int(list[i-1].End.GetLine())
			previousSQLEndColumn := int(list[i-1].End.GetColumn())
			newCaretLine = caretLine - previousSQLEndLine + 1
			if caretLine == previousSQLEndLine {
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
	tokens := mysqlparser.Tokenize(statement)
	caretByteOff := lineColumnToByteOffset(statement, caretLine, caretOffset)

	latestSelectOffset := -1
	newCaretLine, newCaretOffset := caretLine, caretOffset

	for _, tok := range tokens {
		if tok.Loc >= caretByteOff {
			break
		}
		if tok.Type == mysqlparser.SELECT {
			atColumn0 := tok.Loc == 0 || statement[tok.Loc-1] == '\n'
			if atColumn0 {
				latestSelectOffset = tok.Loc
				line := 1
				for j := 0; j < tok.Loc; j++ {
					if statement[j] == '\n' {
						line++
					}
				}
				newCaretLine = caretLine - line + 1
				newCaretOffset = caretOffset
			}
		}
	}

	if latestSelectOffset < 0 {
		return statement, caretLine, caretOffset
	}

	return statement[latestSelectOffset:], newCaretLine, newCaretOffset
}

func unquote(s string) string {
	if len(s) < 2 {
		return s
	}
	if (s[0] == '`' || s[0] == '\'' || s[0] == '"') && s[0] == s[len(s)-1] {
		return s[1 : len(s)-1]
	}
	return s
}

func (c *Completer) listAllDatabases() []string {
	var result []string
	if c.defaultDatabase != "" {
		result = append(result, c.defaultDatabase)
	}
	for databaseName := range c.metadataCache {
		if databaseName != c.defaultDatabase {
			result = append(result, databaseName)
		}
	}
	return result
}

func (c *Completer) listTables(database string) []string {
	if _, exists := c.metadataCache[database]; !exists {
		_, metadata, err := c.getMetadata(c.ctx, c.instanceID, database)
		if err != nil || metadata == nil {
			return nil
		}
		c.metadataCache[database] = metadata
	}
	return c.metadataCache[database].GetSchemaMetadata("").ListTableNames()
}

func (c *Completer) listViews(database string) []string {
	if _, exists := c.metadataCache[database]; !exists {
		_, metadata, err := c.getMetadata(c.ctx, c.instanceID, database)
		if err != nil || metadata == nil {
			return nil
		}
		c.metadataCache[database] = metadata
	}
	return c.metadataCache[database].GetSchemaMetadata("").ListViewNames()
}

func (c *Completer) quotedIdentifierIfNeeded(s string) string {
	if c.caretTokenIsQuoted {
		return s
	}
	if mysqlparser.IsIdentTokenType(mysqlparser.Tokenize(strings.ToUpper(s))[0].Type) {
		// It's an identifier or non-reserved keyword, check if it needs quoting.
		for _, r := range s {
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '$' {
				return fmt.Sprintf("`%s`", s)
			}
		}
		if len(s) > 0 && unicode.IsDigit(rune(s[0])) {
			return fmt.Sprintf("`%s`", s)
		}
		return s
	}
	// It's a reserved keyword, quote it.
	return fmt.Sprintf("`%s`", s)
}

// getMySQLBuiltinFunctions returns a list of MySQL builtin function names.
func getMySQLBuiltinFunctions() []string {
	return []string{
		"ABS", "ACOS", "ADDDATE", "ADDTIME", "AES_DECRYPT", "AES_ENCRYPT",
		"ASCII", "ASIN", "ATAN", "ATAN2", "AVG",
		"BENCHMARK", "BIN", "BIT_AND", "BIT_COUNT", "BIT_LENGTH", "BIT_OR", "BIT_XOR",
		"CAST", "CEIL", "CEILING", "CHAR", "CHAR_LENGTH", "CHARACTER_LENGTH",
		"COALESCE", "COERCIBILITY", "COLLATION", "COMPRESS", "CONCAT", "CONCAT_WS",
		"CONNECTION_ID", "CONV", "CONVERT", "CONVERT_TZ", "COS", "COT", "COUNT",
		"CRC32", "CURDATE", "CURRENT_DATE", "CURRENT_TIME", "CURRENT_TIMESTAMP",
		"CURRENT_USER", "CURTIME",
		"DATABASE", "DATE", "DATE_ADD", "DATE_FORMAT", "DATE_SUB", "DATEDIFF",
		"DAY", "DAYNAME", "DAYOFMONTH", "DAYOFWEEK", "DAYOFYEAR", "DECODE",
		"DEFAULT", "DEGREES", "DES_DECRYPT", "DES_ENCRYPT",
		"ELT", "ENCODE", "ENCRYPT", "EXP", "EXPORT_SET", "EXTRACT",
		"FIELD", "FIND_IN_SET", "FLOOR", "FORMAT", "FOUND_ROWS", "FROM_BASE64",
		"FROM_DAYS", "FROM_UNIXTIME",
		"GET_FORMAT", "GET_LOCK", "GREATEST", "GROUP_CONCAT",
		"HEX", "HOUR",
		"IF", "IFNULL", "IN", "INET_ATON", "INET_NTOA", "INET6_ATON", "INET6_NTOA",
		"INSERT", "INSTR", "IS_FREE_LOCK", "IS_IPV4", "IS_IPV4_COMPAT",
		"IS_IPV4_MAPPED", "IS_IPV6", "IS_USED_LOCK", "ISNULL",
		"JSON_ARRAY", "JSON_ARRAYAGG", "JSON_CONTAINS", "JSON_CONTAINS_PATH",
		"JSON_DEPTH", "JSON_EXTRACT", "JSON_INSERT", "JSON_KEYS", "JSON_LENGTH",
		"JSON_MERGE", "JSON_MERGE_PATCH", "JSON_MERGE_PRESERVE", "JSON_OBJECT",
		"JSON_OBJECTAGG", "JSON_OVERLAPS", "JSON_PRETTY", "JSON_QUOTE",
		"JSON_REMOVE", "JSON_REPLACE", "JSON_SCHEMA_VALID", "JSON_SEARCH",
		"JSON_SET", "JSON_STORAGE_FREE", "JSON_STORAGE_SIZE", "JSON_TABLE",
		"JSON_TYPE", "JSON_UNQUOTE", "JSON_VALID", "JSON_VALUE",
		"LAST_DAY", "LAST_INSERT_ID", "LCASE", "LEAST", "LEFT", "LENGTH",
		"LN", "LOAD_FILE", "LOCALTIME", "LOCALTIMESTAMP", "LOCATE", "LOG",
		"LOG10", "LOG2", "LOWER", "LPAD", "LTRIM",
		"MAKE_SET", "MAKEDATE", "MAKETIME", "MAX", "MD5", "MICROSECOND",
		"MID", "MIN", "MINUTE", "MOD", "MONTH", "MONTHNAME",
		"NOW", "NULLIF",
		"OCT", "OCTET_LENGTH", "ORD",
		"PASSWORD", "PERIOD_ADD", "PERIOD_DIFF", "PI", "POW", "POWER",
		"QUARTER",
		"RADIANS", "RAND", "RANDOM_BYTES", "REGEXP_INSTR", "REGEXP_LIKE",
		"REGEXP_REPLACE", "REGEXP_SUBSTR", "RELEASE_ALL_LOCKS", "RELEASE_LOCK",
		"REPEAT", "REPLACE", "REVERSE", "RIGHT", "ROUND", "ROW_COUNT", "RPAD", "RTRIM",
		"SCHEMA", "SEC_TO_TIME", "SECOND", "SESSION_USER", "SHA1", "SHA2",
		"SIGN", "SIN", "SLEEP", "SOUNDEX", "SPACE", "SQRT",
		"STD", "STDDEV", "STDDEV_POP", "STDDEV_SAMP",
		"STR_TO_DATE", "STRCMP", "SUBDATE", "SUBSTR", "SUBSTRING",
		"SUBSTRING_INDEX", "SUM", "SYSDATE", "SYSTEM_USER",
		"TAN", "TIME", "TIME_FORMAT", "TIME_TO_SEC", "TIMEDIFF", "TIMESTAMP",
		"TIMESTAMPADD", "TIMESTAMPDIFF", "TO_BASE64", "TO_DAYS", "TO_SECONDS",
		"TRIM", "TRUNCATE",
		"UCASE", "UNCOMPRESS", "UNCOMPRESSED_LENGTH", "UNHEX",
		"UNIX_TIMESTAMP", "UPPER", "USER", "UTC_DATE", "UTC_TIME", "UTC_TIMESTAMP",
		"UUID", "UUID_SHORT", "UUID_TO_BIN",
		"VALUES", "VAR_POP", "VAR_SAMP", "VARIANCE", "VERSION",
		"WEEK", "WEEKDAY", "WEEKOFYEAR", "WEIGHT_STRING",
		"YEAR", "YEARWEEK",
	}
}
