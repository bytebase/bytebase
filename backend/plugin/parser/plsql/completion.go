package plsql

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	oracleparser "github.com/bytebase/omni/oracle/parser"

	"github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	base.RegisterCompleteFunc(store.Engine_ORACLE, Completion)
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

type Completer struct {
	ctx              context.Context
	scene            base.SceneType
	sql              string
	cursorByteOffset int
	tokens           []oracleparser.Token
	caretTokenIndex  int

	instanceID        string
	getMetadata       base.GetDatabaseMetadataFunc
	listDatabaseNames base.ListDatabaseNamesFunc
	defaultDatabase   string
	metadataCache     map[string]*model.DatabaseMetadata

	references         []base.TableReference
	cteTables          []*base.VirtualTableReference
	scopeReferences    []oracleparser.RangeReference
	caretTokenIsQuoted bool
	completionPrefix   string
	completionIntent   *oracleparser.CompletionIntent
}

func NewStandardCompleter(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) *Completer {
	sql, byteOffset := computeSQLAndByteOffset(statement, caretLine, caretOffset, false /* tricky */)
	return newCompleter(ctx, cCtx, sql, byteOffset)
}

func NewTrickyCompleter(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) *Completer {
	sql, byteOffset := computeSQLAndByteOffset(statement, caretLine, caretOffset, true /* tricky */)
	return newCompleter(ctx, cCtx, sql, byteOffset)
}

func newCompleter(ctx context.Context, cCtx base.CompletionContext, sql string, byteOffset int) *Completer {
	tokens := oracleparser.Tokenize(sql)
	return &Completer{
		ctx:               ctx,
		scene:             cCtx.Scene,
		sql:               sql,
		cursorByteOffset:  byteOffset,
		tokens:            tokens,
		caretTokenIndex:   findCaretTokenIndex(tokens, byteOffset),
		instanceID:        cCtx.InstanceID,
		getMetadata:       cCtx.Metadata,
		listDatabaseNames: cCtx.ListDatabaseNames,
		defaultDatabase:   cCtx.DefaultDatabase,
		metadataCache:     make(map[string]*model.DatabaseMetadata),
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
			for c := 0; c < column && pos < len(sql); {
				r, size := utf8.DecodeRuneInString(sql[pos:])
				if r == '\n' {
					return pos
				}
				unitCount := 1
				if r > 0xFFFF {
					unitCount = 2
				}
				if c+unitCount > column {
					return pos
				}
				pos += size
				c += unitCount
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

func findCaretTokenIndex(tokens []oracleparser.Token, byteOffset int) int {
	for i, tok := range tokens {
		if tok.Loc >= byteOffset {
			return i
		}
	}
	return len(tokens)
}

func (c *Completer) completion() ([]base.Candidate, error) {
	checkTokenQuoted := func(idx int) bool {
		if idx < 0 || idx >= len(c.tokens) {
			return false
		}
		tok := c.tokens[idx]
		return tok.Loc < len(c.sql) && c.sql[tok.Loc] == '"'
	}
	if checkTokenQuoted(c.caretTokenIndex) {
		c.caretTokenIsQuoted = true
	} else if c.caretTokenIndex > 0 {
		prev := c.tokens[c.caretTokenIndex-1]
		if prev.End >= c.cursorByteOffset && checkTokenQuoted(c.caretTokenIndex-1) {
			c.caretTokenIsQuoted = true
		}
	}

	if c.querySceneDisallowsCompletion() {
		return nil, nil
	}

	completionContext := oracleparser.CollectCompletion(c.sql, c.cursorByteOffset)
	if completionContext != nil {
		c.completionPrefix = completionContext.Prefix
		c.completionIntent = completionContext.Intent
		c.collectCompletionCTEs(completionContext)
		c.collectCompletionScopeReferences(completionContext)
	}
	candidates := (*oracleparser.CandidateSet)(nil)
	if completionContext != nil {
		candidates = completionContext.Candidates
	}
	if candidates == nil {
		candidates = oracleparser.Collect(c.sql, c.cursorByteOffset)
	}
	return c.filterCandidatesByScene(c.convertCandidates(candidates)), nil
}

type CompletionMap map[string]base.Candidate

func (m CompletionMap) Insert(entry base.Candidate) {
	m[entry.String()] = entry
}

func (m CompletionMap) toSlice() []base.Candidate {
	var result []base.Candidate
	for _, candidate := range m {
		result = append(result, candidate)
	}
	slices.SortFunc(result, compareCandidates)
	return result
}

var oracleQuerySceneStartKeywords = map[string]bool{
	"EXPLAIN": true,
	"SELECT":  true,
	"WITH":    true,
}

func compareCandidates(i, j base.Candidate) int {
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
	if i.Definition < j.Definition {
		return -1
	}
	if i.Definition > j.Definition {
		return 1
	}
	return 0
}

func (m CompletionMap) insertDatabases(c *Completer) {
	for _, name := range c.listAllDatabases() {
		m.Insert(base.Candidate{
			Type: base.CandidateTypeSchema,
			Text: c.quotedIdentifierIfNeeded(name),
		})
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

func (m CompletionMap) insertSequences(c *Completer, schemas map[string]bool) {
	for schema := range schemas {
		for _, seq := range c.listSequences(schema) {
			m.Insert(base.Candidate{
				Type: base.CandidateTypeSequence,
				Text: c.quotedIdentifierIfNeeded(seq),
			})
		}
	}
}

func (m CompletionMap) insertTables(c *Completer, schemas map[string]bool) {
	for schema := range schemas {
		if schema == "" {
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

func (m CompletionMap) insertAllColumns(c *Completer) {
	for _, schema := range c.listAllDatabases() {
		if !c.ensureMetadata(schema) {
			continue
		}
		schemaMeta := c.metadataCache[schema].GetSchemaMetadata("")
		if schemaMeta == nil {
			continue
		}
		for _, table := range schemaMeta.ListTableNames() {
			tableMeta := schemaMeta.GetTable(table)
			if tableMeta == nil {
				continue
			}
			for _, column := range tableMeta.GetProto().GetColumns() {
				m.Insert(c.columnCandidate(schema, table, column.Name, column.Type, !column.Nullable, column.Comment))
			}
		}
	}
}

func (m CompletionMap) insertColumns(c *Completer, schemas, tables map[string]bool) {
	for schema := range schemas {
		if schema == "" {
			for _, table := range c.cteTables {
				if !tables[table.Table] {
					continue
				}
				for _, column := range table.Columns {
					m.Insert(base.Candidate{
						Type: base.CandidateTypeColumn,
						Text: c.quotedIdentifierIfNeeded(column),
					})
				}
			}
			continue
		}
		if !c.ensureMetadata(schema) {
			continue
		}
		schemaMeta := c.metadataCache[schema].GetSchemaMetadata("")
		if schemaMeta == nil {
			continue
		}
		for table := range tables {
			tableMeta := schemaMeta.GetTable(table)
			if tableMeta == nil {
				continue
			}
			for _, column := range tableMeta.GetProto().GetColumns() {
				m.Insert(c.columnCandidate(schema, table, column.Name, column.Type, !column.Nullable, column.Comment))
			}
		}
	}
}

func (c *Completer) columnCandidate(schema, table, name, typ string, notNull bool, comment string) base.Candidate {
	definition := fmt.Sprintf("%s.%s | %s", schema, table, typ)
	if notNull {
		definition += ", NOT NULL"
	}
	return base.Candidate{
		Type:       base.CandidateTypeColumn,
		Text:       c.quotedIdentifierIfNeeded(name),
		Definition: definition,
		Comment:    comment,
	}
}

func (c *Completer) convertCandidates(candidates *oracleparser.CandidateSet) []base.Candidate {
	keywordEntries := make(CompletionMap)
	functionEntries := make(CompletionMap)
	schemaEntries := make(CompletionMap)
	tableEntries := make(CompletionMap)
	columnEntries := make(CompletionMap)
	viewEntries := make(CompletionMap)
	sequenceEntries := make(CompletionMap)

	if candidates != nil {
		for _, tok := range candidates.Tokens {
			if tok > 0 && tok < 256 {
				continue
			}
			name := oracleparser.TokenName(tok)
			if name == "" {
				continue
			}
			keywordEntries.Insert(base.Candidate{
				Type: base.CandidateTypeKeyword,
				Text: name,
			})
		}

		for _, rule := range candidates.Rules {
			switch rule.Rule {
			case "func_name":
				for _, tok := range candidates.Tokens {
					name := oracleparser.TokenName(tok)
					if name == "" {
						continue
					}
					functionEntries.Insert(base.Candidate{
						Type: base.CandidateTypeFunction,
						Text: name + "()",
					})
				}
			case "schema_ref":
				schemaEntries.insertDatabases(c)
			case "table_ref":
				c.insertTableReferenceCandidates(schemaEntries, tableEntries, viewEntries, sequenceEntries)
			case "sequence_ref":
				sequenceEntries.insertSequences(c, c.sequenceCompletionSchemas())
			case "columnref":
				c.insertColumnContextCandidates(schemaEntries, tableEntries, columnEntries, viewEntries, sequenceEntries)
			default:
			}
		}
	}

	c.insertCompletionIntentCandidates(keywordEntries, functionEntries, schemaEntries, tableEntries, columnEntries, viewEntries, sequenceEntries)
	if len(c.qualifierParts()) > 0 && len(columnEntries) == 0 {
		c.insertColumnContextCandidates(schemaEntries, tableEntries, columnEntries, viewEntries, sequenceEntries)
	}

	var result []base.Candidate
	result = append(result, keywordEntries.toSlice()...)
	result = append(result, functionEntries.toSlice()...)
	result = append(result, schemaEntries.toSlice()...)
	result = append(result, tableEntries.toSlice()...)
	result = append(result, columnEntries.toSlice()...)
	result = append(result, viewEntries.toSlice()...)
	result = append(result, sequenceEntries.toSlice()...)
	return c.filterCandidatesByPrefix(result)
}

func (c *Completer) insertCompletionIntentCandidates(keywordEntries, functionEntries, schemaEntries, tableEntries, columnEntries, viewEntries, sequenceEntries CompletionMap) {
	if c.completionIntent == nil {
		return
	}
	for _, kind := range c.completionIntent.ObjectKinds {
		switch kind {
		case oracleparser.ObjectKindSchema:
			schemaEntries.insertDatabases(c)
		case oracleparser.ObjectKindTable:
			c.insertTableIntentCandidates(schemaEntries, tableEntries)
		case oracleparser.ObjectKindView:
			c.insertViewIntentCandidates(schemaEntries, viewEntries)
		case oracleparser.ObjectKindSequence:
			sequenceEntries.insertSequences(c, c.sequenceCompletionSchemas())
		case oracleparser.ObjectKindColumn:
			c.insertColumnContextCandidates(schemaEntries, tableEntries, columnEntries, viewEntries, sequenceEntries)
		case oracleparser.ObjectKindFunction:
			for _, tok := range oracleparser.Collect(c.sql, c.cursorByteOffset).Tokens {
				name := oracleparser.TokenName(tok)
				if name == "" {
					continue
				}
				functionEntries.Insert(base.Candidate{
					Type: base.CandidateTypeFunction,
					Text: name + "()",
				})
			}
		case oracleparser.ObjectKindType:
			for _, typ := range oracleDataTypes {
				keywordEntries.Insert(base.Candidate{Type: base.CandidateTypeKeyword, Text: typ})
			}
		default:
		}
	}
}

func (c *Completer) insertTableContextCandidates(schemaEntries, tableEntries, viewEntries, sequenceEntries CompletionMap) {
	parts := c.qualifierParts()
	schema := ""
	if len(parts) == 1 {
		schema = parts[0]
	}
	if schema == "" {
		schemaEntries.insertDatabases(c)
	}
	schemas := c.completionSchemas(schema)
	tableEntries.insertTables(c, schemas)
	viewEntries.insertViews(c, schemas)
	sequenceEntries.insertSequences(c, schemas)
}

func (c *Completer) insertTableReferenceCandidates(schemaEntries, tableEntries, viewEntries, sequenceEntries CompletionMap) {
	if c.completionIntentOnly(oracleparser.ObjectKindTable) {
		c.insertTableIntentCandidates(schemaEntries, tableEntries)
		return
	}
	if c.completionIntentOnly(oracleparser.ObjectKindView) {
		c.insertViewIntentCandidates(schemaEntries, viewEntries)
		return
	}
	c.insertTableContextCandidates(schemaEntries, tableEntries, viewEntries, sequenceEntries)
}

func (c *Completer) completionIntentOnly(kind oracleparser.ObjectKind) bool {
	return c.completionIntent != nil && len(c.completionIntent.ObjectKinds) == 1 && c.completionIntent.ObjectKinds[0] == kind
}

func (c *Completer) insertTableIntentCandidates(schemaEntries, tableEntries CompletionMap) {
	schema := c.completionSchemaQualifier()
	if schema == "" {
		schemaEntries.insertDatabases(c)
	}
	tableEntries.insertTables(c, c.completionSchemas(schema))
}

func (c *Completer) insertViewIntentCandidates(schemaEntries, viewEntries CompletionMap) {
	schema := c.completionSchemaQualifier()
	if schema == "" {
		schemaEntries.insertDatabases(c)
	}
	viewEntries.insertViews(c, c.completionSchemas(schema))
}

func (c *Completer) sequenceCompletionSchemas() map[string]bool {
	return c.completionSchemas(c.completionSchemaQualifier())
}

func (c *Completer) completionSchemaQualifier() string {
	if c.completionIntent != nil {
		if schema := c.completionIntent.Qualifier.Schema; schema != "" {
			return schema
		}
	}
	parts := c.qualifierParts()
	if len(parts) == 1 {
		return parts[0]
	}
	return ""
}

func (c *Completer) insertColumnContextCandidates(schemaEntries, tableEntries, columnEntries, viewEntries, sequenceEntries CompletionMap) {
	schema, table := c.columnQualifier()
	if table == "" && c.completionIntent != nil {
		table = c.completionIntent.Qualifier.Object
	}

	schemas := make(map[string]bool)
	if schema != "" {
		schemas[schema] = true
	} else if table == "" {
		for _, reference := range c.references {
			if physical, ok := reference.(*base.PhysicalTableReference); ok && physical.Schema != "" {
				schemas[physical.Schema] = true
			}
		}
	}

	tables := make(map[string]bool)
	if table != "" {
		tables[table] = true
		for _, reference := range c.references {
			switch reference := reference.(type) {
			case *base.PhysicalTableReference:
				if strings.EqualFold(reference.Alias, table) {
					tables[reference.Table] = true
					if reference.Schema != "" {
						schemas[reference.Schema] = true
					}
				} else if strings.EqualFold(reference.Table, table) && reference.Schema != "" {
					schemas[reference.Schema] = true
				}
			case *base.VirtualTableReference:
				if strings.EqualFold(reference.Table, table) {
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
		for _, reference := range c.references {
			switch reference := reference.(type) {
			case *base.PhysicalTableReference:
				if reference.Schema != "" {
					schemas[reference.Schema] = true
				}
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

	if len(schemas) == 0 {
		schemas[c.defaultDatabase] = true
	}
	schemas[""] = true

	if table == "" {
		schemaEntries.insertDatabases(c)
		tableEntries.insertTables(c, schemas)
		viewEntries.insertViews(c, schemas)
		sequenceEntries.insertSequences(c, schemas)
		for _, alias := range c.selectItemAliases() {
			columnEntries.Insert(base.Candidate{
				Type: base.CandidateTypeColumn,
				Text: c.quotedIdentifierIfNeeded(alias),
			})
		}
	}
	if table == "" {
		c.insertReferenceAliasCandidates(tableEntries)
	}

	if len(tables) > 0 {
		columnEntries.insertColumns(c, schemas, tables)
	}
	if table != "" && len(columnEntries) == 0 {
		c.insertAliasColumns(columnEntries, table)
	}
}

func (c *Completer) completionSchemas(schema string) map[string]bool {
	schemas := make(map[string]bool)
	if schema != "" {
		schemas[schema] = true
		return schemas
	}
	if c.defaultDatabase != "" {
		schemas[c.defaultDatabase] = true
	}
	schemas[""] = true
	return schemas
}

func (c *Completer) collectCompletionCTEs(completionContext *oracleparser.CompletionContext) {
	if completionContext == nil {
		return
	}
	for _, reference := range completionContext.CTEs {
		virtual := convertCompletionVirtualReference(reference)
		if virtual == nil {
			continue
		}
		if len(virtual.Columns) == 0 {
			virtual.Columns = c.inferCompletionVirtualColumns(reference)
		}
		c.appendVirtualReferenceIfMissing(virtual)
	}
}

func (c *Completer) collectCompletionScopeReferences(completionContext *oracleparser.CompletionContext) {
	if completionContext == nil || completionContext.Scope == nil {
		return
	}
	appendReference := func(reference oracleparser.RangeReference) {
		if converted := c.convertCompletionScopeReference(reference); converted != nil {
			c.appendReferenceIfMissing(converted)
			c.appendScopeReferenceIfMissing(reference)
		}
	}
	for _, reference := range completionContext.Scope.LocalReferences {
		appendReference(reference)
	}
	for _, level := range completionContext.Scope.OuterReferences {
		for _, reference := range level {
			appendReference(reference)
		}
	}
}

func (c *Completer) convertCompletionScopeReference(reference oracleparser.RangeReference) base.TableReference {
	switch reference.Kind {
	case oracleparser.RangeReferenceRelation, oracleparser.RangeReferenceDMLTarget, oracleparser.RangeReferenceMergeSource:
		return &base.PhysicalTableReference{
			Schema: reference.Schema,
			Table:  reference.Name,
			Alias:  reference.Alias,
		}
	case oracleparser.RangeReferenceCTE, oracleparser.RangeReferenceSubquery, oracleparser.RangeReferenceJoinAlias:
		virtual := convertCompletionVirtualReference(reference)
		if virtual != nil && reference.Kind == oracleparser.RangeReferenceSubquery {
			if columns := c.inferCompletionVirtualColumns(reference); len(columns) > 0 {
				virtual.Columns = columns
			}
		}
		if virtual != nil && len(virtual.Columns) == 0 {
			if columns := c.inferCompletionVirtualColumns(reference); len(columns) > 0 {
				virtual.Columns = columns
			}
		}
		return virtual
	default:
		return nil
	}
}

func (c *Completer) insertAliasColumns(entries CompletionMap, alias string) {
	for _, reference := range c.scopeReferences {
		if !strings.EqualFold(reference.Alias, alias) && !strings.EqualFold(reference.Name, alias) {
			continue
		}
		columns := append([]string{}, reference.Columns...)
		if len(columns) == 0 {
			columns = c.inferCompletionVirtualColumns(reference)
		}
		for _, column := range columns {
			entries.Insert(base.Candidate{
				Type: base.CandidateTypeColumn,
				Text: c.quotedIdentifierIfNeeded(column),
			})
		}
		return
	}
}

func (c *Completer) insertReferenceAliasCandidates(tableEntries CompletionMap) {
	for _, reference := range c.references {
		switch reference := reference.(type) {
		case *base.PhysicalTableReference:
			if reference.Alias == "" || strings.EqualFold(reference.Alias, reference.Table) {
				continue
			}
			tableEntries.Insert(base.Candidate{
				Type: base.CandidateTypeTable,
				Text: c.quotedIdentifierIfNeeded(reference.Alias),
			})
		case *base.VirtualTableReference:
			tableEntries.Insert(base.Candidate{
				Type: base.CandidateTypeTable,
				Text: c.quotedIdentifierIfNeeded(reference.Table),
			})
		default:
		}
	}
}

func convertCompletionVirtualReference(reference oracleparser.RangeReference) *base.VirtualTableReference {
	table := reference.Alias
	if table == "" {
		table = reference.Name
	}
	if table == "" {
		return nil
	}
	columns := append([]string{}, reference.Columns...)
	return &base.VirtualTableReference{
		Table:   table,
		Columns: columns,
	}
}

func (c *Completer) inferCompletionVirtualColumns(reference oracleparser.RangeReference) []string {
	if reference.BodyLoc.Start < 0 || reference.BodyLoc.End <= reference.BodyLoc.Start || reference.BodyLoc.End > len(c.sql) {
		return nil
	}
	body := c.sql[reference.BodyLoc.Start:reference.BodyLoc.End]
	statement := body
	if reference.Kind == oracleparser.RangeReferenceSubquery {
		alias := reference.Alias
		if alias == "" {
			alias = "x"
		}
		statement = fmt.Sprintf("SELECT * FROM (%s) %s", body, quoteIdentifierForSQL(alias))
		if withPrefix := c.withClausePrefix(); withPrefix != "" {
			statement = fmt.Sprintf("%s SELECT * FROM (%s) %s", withPrefix, body, quoteIdentifierForSQL(alias))
		}
	}
	span, err := GetQuerySpan(
		c.ctx,
		base.GetQuerySpanContext{
			InstanceID:              c.instanceID,
			GetDatabaseMetadataFunc: c.getMetadata,
			ListDatabaseNamesFunc:   c.listDatabaseNames,
		},
		base.Statement{Text: statement},
		c.defaultDatabase,
		"",
		false,
	)
	if err != nil || span.NotFoundError != nil {
		return nil
	}
	columns := make([]string, 0, len(span.Results))
	for _, column := range span.Results {
		columns = append(columns, column.Name)
	}
	return columns
}

func (c *Completer) withClausePrefix() string {
	first := -1
	for i, token := range c.tokens {
		if token.Type != ';' {
			first = i
			break
		}
	}
	if first < 0 || c.tokens[first].Type != oracleparser.WITH {
		return ""
	}
	depth := 0
	for i := first + 1; i < len(c.tokens); i++ {
		switch c.tokens[i].Type {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case oracleparser.SELECT:
			if depth == 0 {
				return strings.TrimSpace(c.sql[:c.tokens[i].Loc])
			}
		default:
		}
	}
	return ""
}

func quoteIdentifierForSQL(identifier string) string {
	if identifierNeedsQuoting(identifier) {
		return fmt.Sprintf(`"%s"`, strings.ReplaceAll(identifier, `"`, `""`))
	}
	return identifier
}

func (c *Completer) appendVirtualReferenceIfMissing(reference *base.VirtualTableReference) {
	for _, existing := range c.cteTables {
		if strings.EqualFold(existing.Table, reference.Table) {
			if len(existing.Columns) == 0 && len(reference.Columns) > 0 {
				existing.Columns = reference.Columns
			}
			return
		}
	}
	c.cteTables = append(c.cteTables, reference)
}

func (c *Completer) appendReferenceIfMissing(reference base.TableReference) {
	key := tableReferenceKey(reference)
	for _, existing := range c.references {
		if tableReferenceKey(existing) == key {
			return
		}
	}
	c.references = append(c.references, reference)
}

func (c *Completer) appendScopeReferenceIfMissing(reference oracleparser.RangeReference) {
	key := completionScopeReferenceKey(reference)
	for _, existing := range c.scopeReferences {
		if completionScopeReferenceKey(existing) == key {
			return
		}
	}
	c.scopeReferences = append(c.scopeReferences, reference)
}

func tableReferenceKey(reference base.TableReference) string {
	switch reference := reference.(type) {
	case *base.PhysicalTableReference:
		return strings.Join([]string{"physical", reference.Schema, reference.Table, reference.Alias}, "\x00")
	case *base.VirtualTableReference:
		return strings.Join([]string{"virtual", reference.Table}, "\x00")
	default:
		return fmt.Sprintf("%T", reference)
	}
}

func completionScopeReferenceKey(reference oracleparser.RangeReference) string {
	return strings.Join([]string{
		fmt.Sprint(reference.Kind),
		reference.Schema,
		reference.Name,
		reference.Alias,
		fmt.Sprint(reference.Loc.Start),
		fmt.Sprint(reference.Loc.End),
	}, "\x00")
}

func (c *Completer) qualifierParts() []string {
	collectOffset := c.cursorByteOffset - len(c.completionPrefix)
	if collectOffset > len(c.sql) {
		collectOffset = len(c.sql)
	}
	before := strings.TrimRight(c.sql[:collectOffset], " \t\r\n")
	if !strings.HasSuffix(before, ".") {
		return nil
	}
	before = strings.TrimSuffix(before, ".")
	var parts []string
	for {
		before = strings.TrimRight(before, " \t\r\n")
		part, start := lastIdentifierSpan(before)
		if part == "" {
			break
		}
		parts = append([]string{unquote(part)}, parts...)
		before = strings.TrimRight(before[:start], " \t\r\n")
		if !strings.HasSuffix(before, ".") {
			break
		}
		before = strings.TrimSuffix(before, ".")
	}
	return parts
}

func lastIdentifierSpan(s string) (string, int) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", 0
	}
	if s[len(s)-1] == '"' {
		for i := len(s) - 2; i >= 0; i-- {
			if s[i] == '"' {
				return s[i:], i
			}
		}
		return "", 0
	}
	end := len(s)
	start := end
	for start > 0 {
		r, size := utf8.DecodeLastRuneInString(s[:start])
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '$' && r != '#' {
			break
		}
		start -= size
	}
	return s[start:end], start
}

func (c *Completer) columnQualifier() (schema, table string) {
	parts := c.qualifierParts()
	switch len(parts) {
	case 0:
		return "", ""
	case 1:
		return "", parts[0]
	default:
		return parts[len(parts)-2], parts[len(parts)-1]
	}
}

func isIdentifierLike(tok oracleparser.Token) bool {
	return tok.Str != "" && oracleparser.TokenName(tok.Type) == ""
}

func (c *Completer) selectItemAliases() []string {
	if !c.isInAliasAllowedContext() {
		return nil
	}
	selectIdx, fromIdx := c.currentQueryBlockSelectFrom()
	if selectIdx < 0 || fromIdx < 0 {
		return nil
	}
	aliasMap := make(map[string]bool)
	start := selectIdx + 1
	depth := 0
	for i := start; i <= fromIdx; i++ {
		if i == fromIdx || (depth == 0 && c.tokens[i].Type == ',') {
			if alias := selectAliasFromTokens(c.tokens[start:i]); alias != "" {
				aliasMap[alias] = true
			}
			start = i + 1
			continue
		}
		switch c.tokens[i].Type {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		default:
		}
	}
	aliases := make([]string, 0, len(aliasMap))
	for alias := range aliasMap {
		aliases = append(aliases, alias)
	}
	slices.Sort(aliases)
	return aliases
}

func (c *Completer) currentQueryBlockSelectFrom() (int, int) {
	stmtStart, stmtEnd := c.statementTokenBounds()
	targetDepth := c.depthAtToken(c.caretTokenIndex)
	selectIdx := -1
	fromIdx := -1
	depth := 0
	for i := stmtStart; i < stmtEnd && i < c.caretTokenIndex; i++ {
		if c.tokens[i].Type == ')' && depth > 0 {
			depth--
		}
		if depth == targetDepth {
			switch c.tokens[i].Type {
			case oracleparser.SELECT:
				selectIdx = i
				fromIdx = -1
			case oracleparser.FROM:
				if selectIdx >= 0 && fromIdx < 0 {
					fromIdx = i
				}
			default:
			}
		}
		if c.tokens[i].Type == '(' {
			depth++
		}
	}
	return selectIdx, fromIdx
}

func (c *Completer) depthAtToken(index int) int {
	stmtStart, _ := c.statementTokenBounds()
	depth := 0
	for i := stmtStart; i < index && i < len(c.tokens); i++ {
		switch c.tokens[i].Type {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		default:
		}
	}
	return depth
}

func selectAliasFromTokens(tokens []oracleparser.Token) string {
	for len(tokens) > 0 && tokens[0].Type == ',' {
		tokens = tokens[1:]
	}
	for len(tokens) > 0 && tokens[len(tokens)-1].Type == ',' {
		tokens = tokens[:len(tokens)-1]
	}
	if len(tokens) < 2 {
		return ""
	}
	last := tokens[len(tokens)-1]
	if !isIdentifierLike(last) {
		return ""
	}
	prev := tokens[len(tokens)-2]
	if prev.Type == oracleparser.AS {
		return last.Str
	}
	if prev.Type != '.' {
		return last.Str
	}
	return ""
}

func (c *Completer) isInAliasAllowedContext() bool {
	stmtStart, _ := c.statementTokenBounds()
	for i := c.caretTokenIndex - 1; i >= stmtStart; i-- {
		switch c.tokens[i].Type {
		case oracleparser.ORDER, oracleparser.GROUP:
			return true
		case oracleparser.WHERE, oracleparser.FROM, oracleparser.SELECT:
			return false
		default:
		}
	}
	return false
}

func (c *Completer) statementTokenBounds() (int, int) {
	start := 0
	for i, tok := range c.tokens {
		if tok.Loc >= c.cursorByteOffset {
			break
		}
		if tok.Type == ';' {
			start = i + 1
		}
	}
	end := len(c.tokens)
	for i := start; i < len(c.tokens); i++ {
		if c.tokens[i].Loc < c.cursorByteOffset {
			continue
		}
		if c.tokens[i].Type == ';' {
			end = i
			break
		}
	}
	return start, end
}

func (c *Completer) listAllDatabases() []string {
	var result []string
	if c.defaultDatabase != "" {
		result = append(result, c.defaultDatabase)
	}
	if c.listDatabaseNames == nil {
		return result
	}
	list, err := c.listDatabaseNames(c.ctx, c.instanceID)
	if err != nil {
		return result
	}
	for _, name := range list {
		if name != c.defaultDatabase {
			result = append(result, name)
		}
	}
	return result
}

func (c *Completer) filterCandidatesByScene(candidates []base.Candidate) []base.Candidate {
	if c.scene != base.SceneTypeQuery || !c.isAtStatementStartCompletion() {
		return candidates
	}
	var result []base.Candidate
	for _, candidate := range candidates {
		if candidate.Type == base.CandidateTypeKeyword && !oracleQuerySceneStartKeywords[strings.ToUpper(candidate.Text)] {
			continue
		}
		result = append(result, candidate)
	}
	return result
}

func (c *Completer) querySceneDisallowsCompletion() bool {
	if c.scene != base.SceneTypeQuery {
		return false
	}
	idx := c.currentStatementMainTokenIndex()
	if idx < 0 {
		return false
	}
	if c.tokens[idx].End >= c.cursorByteOffset {
		return false
	}
	if strings.EqualFold(c.tokens[idx].Str, "EXPLAIN") {
		return false
	}
	switch c.tokens[idx].Type {
	case oracleparser.SELECT, oracleparser.WITH:
		return false
	default:
		return true
	}
}

func (c *Completer) isAtStatementStartCompletion() bool {
	collectOffset := c.cursorByteOffset - len(c.completionPrefix)
	if collectOffset < 0 {
		collectOffset = 0
	}
	start := c.currentStatementStartTokenIndex(collectOffset)
	if start >= len(c.tokens) {
		return true
	}
	token := c.tokens[start]
	return token.Type == ';' || token.Loc >= collectOffset
}

func (c *Completer) currentStatementMainTokenIndex() int {
	start := c.currentStatementStartTokenIndex(c.cursorByteOffset)
	if start >= len(c.tokens) {
		return -1
	}
	token := c.tokens[start]
	if token.Loc >= c.cursorByteOffset || token.Type == ';' {
		return -1
	}
	if token.Type == oracleparser.WITH {
		return c.withMainStatementTokenIndex(start)
	}
	return start
}

func (c *Completer) withMainStatementTokenIndex(withIdx int) int {
	depth := 0
	for i := withIdx + 1; i < len(c.tokens); i++ {
		token := c.tokens[i]
		if token.Loc >= c.cursorByteOffset || token.Type == ';' {
			break
		}
		switch token.Type {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		default:
			if depth == 0 && isOracleStatementStartToken(token.Type) {
				return i
			}
		}
	}
	return withIdx
}

func isOracleStatementStartToken(tokenType int) bool {
	switch tokenType {
	case oracleparser.SELECT, oracleparser.INSERT, oracleparser.UPDATE, oracleparser.DELETE, oracleparser.MERGE:
		return true
	default:
		return false
	}
}

func (c *Completer) currentStatementStartTokenIndex(offset int) int {
	start := 0
	for i, token := range c.tokens {
		if token.Loc >= offset {
			break
		}
		if token.Type == ';' {
			start = i + 1
		}
	}
	return start
}

func (c *Completer) listTables(schema string) []string {
	if !c.ensureMetadata(schema) {
		return nil
	}
	schemaMeta := c.metadataCache[schema].GetSchemaMetadata("")
	if schemaMeta == nil {
		return nil
	}
	return schemaMeta.ListTableNames()
}

func (c *Completer) listViews(schema string) []string {
	if !c.ensureMetadata(schema) {
		return nil
	}
	schemaMeta := c.metadataCache[schema].GetSchemaMetadata("")
	if schemaMeta == nil {
		return nil
	}
	return schemaMeta.ListViewNames()
}

func (c *Completer) listSequences(schema string) []string {
	if !c.ensureMetadata(schema) {
		return nil
	}
	schemaMeta := c.metadataCache[schema].GetSchemaMetadata("")
	if schemaMeta == nil {
		return nil
	}
	return schemaMeta.ListSequenceNames()
}

func (c *Completer) ensureMetadata(schema string) bool {
	if schema == "" || c.getMetadata == nil {
		return false
	}
	if _, exists := c.metadataCache[schema]; exists {
		return true
	}
	_, metadata, err := c.getMetadata(c.ctx, c.instanceID, schema)
	if err != nil || metadata == nil {
		return false
	}
	c.metadataCache[schema] = metadata
	return true
}

func (c *Completer) filterCandidatesByPrefix(candidates []base.Candidate) []base.Candidate {
	if c.completionPrefix == "" {
		return candidates
	}
	prefix := strings.ToUpper(c.completionPrefix)
	filtered := make([]base.Candidate, 0, len(candidates))
	for _, candidate := range candidates {
		text := strings.TrimSuffix(candidate.Text, "()")
		text = unquote(text)
		if strings.HasPrefix(strings.ToUpper(text), prefix) {
			filtered = append(filtered, candidate)
		}
	}
	return filtered
}

func unquote(s string) string {
	if len(s) < 2 {
		return strings.ToUpper(s)
	}
	if (s[0] == '`' || s[0] == '\'' || s[0] == '"') && s[0] == s[len(s)-1] {
		return s[1 : len(s)-1]
	}
	return strings.ToUpper(s)
}

func skipHeadingSQLs(statement string, caretLine int, caretOffset int) (string, int, int) {
	newCaretLine, newCaretOffset := caretLine, caretOffset
	list, err := SplitSQLForCompletion(statement)
	if err != nil || notEmptySQLCount(list) <= 1 {
		return statement, caretLine, caretOffset
	}

	caretLine--
	start := 0
	for i, sql := range list {
		sqlEndLine := int(sql.End.GetLine()) - 1
		sqlEndColumn := int(sql.End.GetColumn())
		if sqlEndLine > caretLine || (sqlEndLine == caretLine && sqlEndColumn >= caretOffset) {
			start = i
			if i == 0 {
				break
			}
			previousSQLEndLine := int(list[i-1].End.GetLine()) - 1
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

func notEmptySQLCount(list []base.Statement) int {
	count := 0
	for _, sql := range list {
		if !sql.Empty {
			count++
		}
	}
	return count
}

func skipHeadingSQLWithoutSemicolon(statement string, caretLine int, caretOffset int) (string, int, int) {
	tokens := oracleparser.Tokenize(statement)
	caretByteOffset := lineColumnToByteOffset(statement, caretLine, caretOffset)

	latestSelectOffset := -1
	newCaretLine, newCaretOffset := caretLine, caretOffset
	for _, token := range tokens {
		line, column := lineColumnAtByteOffset(statement, token.Loc)
		if line > caretLine || (line == caretLine && column >= caretOffset) {
			break
		}
		if token.Type == oracleparser.SELECT && column == 0 {
			latestSelectOffset = token.Loc
			newCaretLine = caretLine - line + 1
			newCaretOffset = caretOffset
		}
	}

	if latestSelectOffset < 0 || latestSelectOffset >= caretByteOffset {
		return statement, caretLine, caretOffset
	}
	return statement[latestSelectOffset:], newCaretLine, newCaretOffset
}

func lineColumnAtByteOffset(sql string, offset int) (int, int) {
	line, column := 1, 0
	for i := 0; i < len(sql) && i < offset; {
		r, size := utf8.DecodeRuneInString(sql[i:])
		if r == '\n' {
			line++
			column = 0
		} else if r > 0xFFFF {
			column += 2
		} else {
			column++
		}
		i += size
	}
	return line, column
}

func (c *Completer) quotedIdentifierIfNeeded(s string) string {
	if c.caretTokenIsQuoted || !identifierNeedsQuoting(s) {
		return s
	}
	return fmt.Sprintf(`"%s"`, strings.ReplaceAll(s, `"`, `""`))
}

func identifierNeedsQuoting(s string) bool {
	if s == "" {
		return true
	}
	if oracleparser.IsReservedKeyword(s) {
		return true
	}
	if s != strings.ToUpper(s) {
		return true
	}
	for i, r := range s {
		if i == 0 && !unicode.IsLetter(r) {
			return true
		}
		if i > 0 && !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '$' && r != '#' {
			return true
		}
	}
	return false
}

var oracleDataTypes = []string{
	"NUMBER",
	"VARCHAR2",
	"NVARCHAR2",
	"CHAR",
	"NCHAR",
	"DATE",
	"TIMESTAMP",
	"CLOB",
	"BLOB",
	"RAW",
}
