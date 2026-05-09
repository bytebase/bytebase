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

	omnimssql "github.com/bytebase/omni/mssql"
	mssqlparser "github.com/bytebase/omni/mssql/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	base.RegisterCompleteFunc(storepb.Engine_MSSQL, Completion)
}

var (
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
	ctx   context.Context
	scene base.SceneType

	sql              string
	cursorByteOffset int
	tokens           []mssqlparser.Token
	caretTokenIndex  int

	instanceID          string
	defaultDatabase     string
	defaultSchema       string
	metadataGetter      base.GetDatabaseMetadataFunc
	databaseNamesLister base.ListDatabaseNamesFunc

	// references is the flattened table references.
	// It's helpful to look up the table reference.
	references         []base.TableReference
	referenceMap       map[string]bool
	cteTables          []*base.VirtualTableReference
	caretTokenIsQuoted quotedType
	completionPrefix   string
	completionIntent   *omnimssql.CompletionIntent
}

func Completion(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) ([]base.Candidate, error) {
	completer := NewStandardCompleter(ctx, cCtx, statement, caretLine, caretOffset)
	result, err := completer.complete()

	if err != nil {
		return nil, err
	}
	if len(result) > 0 {
		return result, nil
	}

	trickyCompleter := NewTrickyCompleter(ctx, cCtx, statement, caretLine, caretOffset)
	return trickyCompleter.complete()
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
	tokens := mssqlparser.Tokenize(sql)

	return &Completer{
		ctx:                 ctx,
		scene:               cCtx.Scene,
		sql:                 sql,
		cursorByteOffset:    byteOffset,
		tokens:              tokens,
		caretTokenIndex:     findCaretTokenIndex(tokens, byteOffset),
		instanceID:          cCtx.InstanceID,
		defaultDatabase:     cCtx.DefaultDatabase,
		defaultSchema:       "dbo",
		metadataGetter:      cCtx.Metadata,
		databaseNamesLister: cCtx.ListDatabaseNames,
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

// caretLine is 1-based and caretOffset is 0-based.
func skipHeadingSQLWithoutSemicolon(statement string, caretLine int, caretOffset int) (string, int, int) {
	tokens := mssqlparser.Tokenize(statement)
	latestSelect := 0
	latestSelectIndex := 0
	newCaretLine, newCaretOffset := caretLine, caretOffset
	for idx, token := range tokens {
		line, column := lineColumnAtByteOffset(statement, token.Loc)
		if line > caretLine || (line == caretLine && column >= caretOffset) {
			break
		}
		if token.Type == mssqlparser.SELECT && column == 0 {
			latestSelect = token.Loc
			latestSelectIndex = idx
			newCaretLine = caretLine - line + 1 // convert to 1-based.
			newCaretOffset = caretOffset
		}
	}

	if latestSelectIndex == 0 {
		return statement, caretLine, caretOffset
	}
	return statement[latestSelect:], newCaretLine, newCaretOffset
}

func lineColumnAtByteOffset(sql string, offset int) (int, int) {
	line, column := 1, 0
	for i := 0; i < len(sql) && i < offset; {
		r, size := utf8.DecodeRuneInString(sql[i:])
		if r == '\n' {
			line++
			column = 0
		} else {
			column++
		}
		i += size
	}
	return line, column
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

	completionContext := omnimssql.CollectCompletion(c.sql, c.cursorByteOffset)
	if completionContext != nil {
		c.completionPrefix = completionContext.Prefix
		c.completionIntent = completionContext.Intent
		c.collectCompletionCTEs(completionContext)
		c.collectCompletionScopeReferences(completionContext)
	}
	candidates := (*mssqlparser.CandidateSet)(nil)
	if completionContext != nil {
		candidates = completionContext.Candidates
	}
	if candidates == nil {
		candidates = mssqlparser.Collect(c.sql, c.cursorByteOffset)
	}

	return c.convertCandidates(candidates)
}

func (c *Completer) determineObjectNameContext() []*objectRefContext {
	if c.completionIntent == nil {
		return []*objectRefContext{newObjectRefContext()}
	}
	context := newObjectRefContext()
	qualifier := c.completionIntent.Qualifier
	if qualifier.Server != "" {
		context.setLinkedServer(qualifier.Server)
	}
	if qualifier.Database != "" {
		context.setDatabase(qualifier.Database)
	}
	if qualifier.Schema != "" {
		context.setSchema(qualifier.Schema)
		if qualifier.Database == "" {
			context.flags &^= objectFlagShowDatabase
		}
	}
	if qualifier.Object != "" {
		context.setObject(qualifier.Object)
	}
	return []*objectRefContext{context}
}

func (c *Completer) determineColumnNameContext() []*objectRefContext {
	if c.completionIntent == nil || !completionIntentHasObjectKind(c.completionIntent, omnimssql.ObjectKindColumn) {
		return []*objectRefContext{newObjectRefContext(withColumn())}
	}
	context := newObjectRefContext(withColumn())
	qualifier := c.completionIntent.Qualifier
	if qualifier.Database == "" && qualifier.Schema != "" && qualifier.Object == "" &&
		(strings.EqualFold(qualifier.Schema, "INSERTED") || strings.EqualFold(qualifier.Schema, "DELETED")) {
		return []*objectRefContext{context}
	}
	if qualifier.Object != "" {
		if qualifier.Server != "" {
			context.setLinkedServer(qualifier.Server)
		}
		if qualifier.Database != "" {
			context.setDatabase(qualifier.Database)
		}
		if qualifier.Schema != "" {
			context.setSchema(qualifier.Schema)
		}
		context.setObject(qualifier.Object)
		context.flags &^= objectFlagShowDatabase | objectFlagShowSchema
		return []*objectRefContext{context}
	}
	if qualifier.Server != "" && qualifier.Database != "" && qualifier.Schema != "" {
		context.setDatabase(qualifier.Server)
		context.setSchema(qualifier.Database)
		context.setObject(qualifier.Schema)
		context.flags &^= objectFlagShowDatabase | objectFlagShowSchema
		return []*objectRefContext{context}
	}
	if qualifier.Schema != "" {
		if qualifier.Database != "" {
			context.setSchema(qualifier.Database)
		}
		context.setObject(qualifier.Schema)
		context.flags &^= objectFlagShowDatabase | objectFlagShowSchema
		return []*objectRefContext{context}
	}
	if qualifier.Database != "" {
		context.setObject(qualifier.Database)
		context.flags &^= objectFlagShowDatabase | objectFlagShowSchema
	}
	return []*objectRefContext{context}
}

func completionIntentHasObjectKind(intent *omnimssql.CompletionIntent, kind omnimssql.ObjectKind) bool {
	if intent == nil {
		return false
	}
	for _, objectKind := range intent.ObjectKinds {
		if objectKind == kind {
			return true
		}
	}
	return false
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

	handledDottedObjectContext := c.insertDottedObjectContextCandidates(databaseEntries, schemaEntries, tableEntries, viewEntries, sequenceEntries, routineEntries)
	if !handledDottedObjectContext {
		c.insertCompletionIntentCandidates(keywordEntries, functionEntries, databaseEntries, schemaEntries, tableEntries, viewEntries, sequenceEntries, routineEntries)
	}

	for _, ruleCandidate := range candidates.Rules {
		if handledDottedObjectContext && isObjectCompletionRule(ruleCandidate.Rule) {
			continue
		}
		switch ruleCandidate.Rule {
		case "func_name":
			functionEntries.insertBuiltinFunctions()
		case "database_ref":
			databaseEntries.insertMetadataDatabases(c, "")
		case "schema_ref":
			schemaEntries.insertMetadataSchemas(c, "", "")
		case "proc_ref", "proc_name":
			for _, context := range c.determineObjectNameContext() {
				if context.flags&objectFlagShowObject != 0 {
					routineEntries.insertMetadataProcedures(c, context.linkedServer, context.database, context.schema)
				}
			}
		case "sequence_ref":
			for _, context := range c.determineObjectNameContext() {
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
			completionContexts := c.determineObjectNameContext()
			for _, context := range completionContexts {
				c.insertObjectCandidates(context, databaseEntries, schemaEntries, tableEntries, viewEntries, sequenceEntries)
			}
		case "view_name", "view_ref":
			for _, context := range c.determineObjectNameContext() {
				if context.flags&objectFlagShowObject != 0 {
					viewEntries.insertMetadataViews(c, context.linkedServer, context.database, context.schema)
				}
			}
		case "asterisk":
			completionContexts := c.determineObjectNameContext()
			for _, context := range completionContexts {
				c.insertObjectCandidates(context, databaseEntries, schemaEntries, tableEntries, viewEntries, sequenceEntries)
			}
		case "columnref":
			completionContexts := c.determineColumnNameContext()
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
					if !context.empty() || len(c.references) == 0 {
						columnEntries.insertMetadataColumns(c, context.linkedServer, context.database, context.schema, context.object)
					}
					for _, reference := range c.references {
						matchedQualifiedReference := false
						switch reference := reference.(type) {
						case *base.PhysicalTableReference:
							if context.empty() {
								columnEntries.insertMetadataColumns(c, "", reference.Database, reference.Schema, reference.Table)
								continue
							}
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
							referenceTableIsAlias := false
							if reference.Alias != "" {
								referenceTableName = reference.Alias
								referenceTableIsAlias = true
							}

							if inputLinkedServer == "" && strings.EqualFold(referenceTableName, inputTableName) &&
								(referenceTableIsAlias ||
									(strings.EqualFold(referenceDatabaseName, inputDatabaseName) &&
										strings.EqualFold(referenceSchemaName, inputSchemaName))) {
								columnEntries.insertMetadataColumns(c, "", reference.Database, reference.Schema, reference.Table)
								matchedQualifiedReference = referenceTableIsAlias && context.database == "" && context.schema == ""
							}
						case *base.VirtualTableReference:
							// Reference could be a physical table reference or a virtual table reference, if the reference is a virtual table reference,
							// and users do not specify the server, database and schema, we should also insert the columns.
							if context.linkedServer == "" && context.database == "" && context.schema == "" {
								if context.object != "" && !strings.EqualFold(reference.Table, context.object) {
									continue
								}
								for _, column := range reference.Columns {
									if _, ok := columnEntries[column]; !ok {
										columnEntries[column] = base.Candidate{
											Type: base.CandidateTypeColumn,
											Text: c.quotedIdentifierIfNeeded(column),
										}
									}
								}
								matchedQualifiedReference = context.object != ""
							}
						default:
						}
						if matchedQualifiedReference {
							break
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
					if context.empty() && len(c.references) == 0 {
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
	return c.filterCandidatesByPrefix(result), nil
}

func (c *Completer) insertCompletionIntentCandidates(keywordEntries, functionEntries, databaseEntries, schemaEntries, tableEntries, viewEntries, sequenceEntries, routineEntries CompletionMap) {
	if c.completionIntent == nil {
		return
	}
	contexts := c.determineObjectNameContext()
	for _, kind := range c.completionIntent.ObjectKinds {
		for _, context := range contexts {
			switch kind {
			case omnimssql.ObjectKindDatabase:
				databaseEntries.insertMetadataDatabases(c, context.linkedServer)
			case omnimssql.ObjectKindSchema:
				schemaEntries.insertMetadataSchemas(c, context.linkedServer, context.database)
			case omnimssql.ObjectKindTable:
				if context.flags&objectFlagShowObject != 0 {
					tableEntries.insertMetadataTables(c, context.linkedServer, context.database, context.schema)
				}
			case omnimssql.ObjectKindView:
				if context.flags&objectFlagShowObject != 0 {
					viewEntries.insertMetadataViews(c, context.linkedServer, context.database, context.schema)
				}
			case omnimssql.ObjectKindSequence:
				if context.flags&objectFlagShowObject != 0 {
					sequenceEntries.insertMetadataSequences(c, context.linkedServer, context.database, context.schema)
				}
			case omnimssql.ObjectKindProcedure:
				if context.flags&objectFlagShowObject != 0 {
					routineEntries.insertMetadataProcedures(c, context.linkedServer, context.database, context.schema)
				}
			case omnimssql.ObjectKindFunction:
				functionEntries.insertBuiltinFunctions()
			case omnimssql.ObjectKindType:
				for _, typ := range tsqlDataTypes {
					keywordEntries.Insert(base.Candidate{
						Type: base.CandidateTypeKeyword,
						Text: typ,
					})
				}
			default:
			}
		}
	}
}

func (c *Completer) insertDottedObjectContextCandidates(databaseEntries, schemaEntries, tableEntries, viewEntries, sequenceEntries, routineEntries CompletionMap) bool {
	if len(databaseEntries) > 0 || len(schemaEntries) > 0 || len(tableEntries) > 0 || len(viewEntries) > 0 || len(sequenceEntries) > 0 || len(routineEntries) > 0 {
		return false
	}
	parts, beforeIdx, ok := c.multipartPartsBeforeDottedObject()
	if !ok {
		return false
	}
	kind, ok := c.dottedObjectKindBefore(beforeIdx)
	if !ok {
		return false
	}
	c.insertDottedObjectCandidatesForParts(kind, parts, schemaEntries, tableEntries, viewEntries, sequenceEntries, routineEntries)
	return true
}

func isObjectCompletionRule(rule string) bool {
	switch rule {
	case "database_ref", "schema_ref", "table_ref", "view_name", "view_ref", "sequence_ref", "proc_ref", "proc_name":
		return true
	default:
		return false
	}
}

func (c *Completer) multipartPartsBeforeDottedObject() ([]string, int, bool) {
	idx := c.previousTokenIndex()
	if idx < 1 {
		return nil, -1, false
	}
	if c.completionPrefix != "" && mssqlparser.IsIdentTokenType(c.tokens[idx].Type) {
		idx--
	}
	if idx < 1 || !isMultipartSeparator(c.tokenText(idx)) {
		return nil, -1, false
	}

	var parts []string
	for {
		separator := c.tokenText(idx)
		partIdx := idx - 1
		if partIdx < 0 {
			return nil, -1, false
		}
		if c.tokenText(partIdx) == "." {
			if separator == ".." {
				return nil, -1, false
			}
			parts = append([]string{""}, parts...)
			idx = partIdx
			continue
		}
		token := c.tokens[partIdx]
		if !mssqlparser.IsIdentTokenType(token.Type) {
			return nil, -1, false
		}
		part := normalizeCompletionIdentifier(c.tokenText(partIdx))
		if part == "" {
			return nil, -1, false
		}
		if separator == ".." {
			parts = append([]string{part, ""}, parts...)
		} else {
			parts = append([]string{part}, parts...)
		}
		if partIdx-1 < 0 || c.tokenText(partIdx-1) != "." {
			return parts, partIdx - 1, true
		}
		idx = partIdx - 1
	}
}

func isMultipartSeparator(text string) bool {
	return text == "." || text == ".."
}

type dottedObjectKind int

const (
	dottedObjectKindRelation dottedObjectKind = iota
	dottedObjectKindView
	dottedObjectKindSequence
	dottedObjectKindRoutine
)

func (c *Completer) dottedObjectKindBefore(idx int) (dottedObjectKind, bool) {
	if idx < 0 {
		return dottedObjectKindRelation, false
	}
	if strings.EqualFold(c.tokenText(idx), "EXISTS") && idx >= 1 && strings.EqualFold(c.tokenText(idx-1), "IF") {
		idx -= 2
	}
	if idx < 0 {
		return dottedObjectKindRelation, false
	}
	switch strings.ToUpper(c.tokenText(idx)) {
	case "VIEW":
		return dottedObjectKindView, true
	case "SEQUENCE":
		return dottedObjectKindSequence, true
	case "PROCEDURE", "PROC", "EXEC", "EXECUTE":
		return dottedObjectKindRoutine, true
	case "FOR":
		if idx >= 2 && strings.EqualFold(c.tokenText(idx-1), "VALUE") && strings.EqualFold(c.tokenText(idx-2), "NEXT") {
			return dottedObjectKindSequence, true
		}
		return dottedObjectKindRelation, false
	case "FROM", "JOIN", "APPLY", "INTO", "UPDATE", "USING", "REFERENCES", "TABLE", "TRUNCATE":
		return dottedObjectKindRelation, true
	default:
		return dottedObjectKindRelation, false
	}
}

func (c *Completer) insertDottedObjectCandidatesForParts(kind dottedObjectKind, parts []string, schemaEntries, tableEntries, viewEntries, sequenceEntries, routineEntries CompletionMap) {
	switch len(parts) {
	case 0:
		return
	case 1:
		schemaEntries.insertMetadataSchemas(c, "", parts[0])
		c.insertDottedObjectCandidatesForSchema(kind, "", parts[0], tableEntries, viewEntries, sequenceEntries, routineEntries)
	case 2:
		if parts[1] == "" {
			c.insertDottedObjectCandidatesForSchema(kind, parts[0], c.defaultSchema, tableEntries, viewEntries, sequenceEntries, routineEntries)
			return
		}
		c.insertDottedObjectCandidatesForSchema(kind, parts[len(parts)-2], parts[len(parts)-1], tableEntries, viewEntries, sequenceEntries, routineEntries)
	default:
		return
	}
}

func (c *Completer) insertDottedObjectCandidatesForSchema(kind dottedObjectKind, database string, schema string, tableEntries, viewEntries, sequenceEntries, routineEntries CompletionMap) {
	switch kind {
	case dottedObjectKindView:
		viewEntries.insertMetadataViews(c, "", database, schema)
	case dottedObjectKindSequence:
		sequenceEntries.insertMetadataSequences(c, "", database, schema)
	case dottedObjectKindRoutine:
		routineEntries.insertMetadataProcedures(c, "", database, schema)
	default:
		tableEntries.insertMetadataTables(c, "", database, schema)
		viewEntries.insertMetadataViews(c, "", database, schema)
		sequenceEntries.insertMetadataSequences(c, "", database, schema)
	}
}

func (c *Completer) filterCandidatesByPrefix(candidates []base.Candidate) []base.Candidate {
	prefix := c.completionPrefix
	if prefix == "" {
		return candidates
	}
	_, normalizedPrefix := NormalizeTSQLIdentifierText(prefix)
	if normalizedPrefix == "" {
		return candidates
	}
	filtered := make([]base.Candidate, 0, len(candidates))
	for _, candidate := range candidates {
		text := strings.TrimSuffix(candidate.Text, "()")
		text = unquoteDoubleQuoted(text)
		_, normalizedText := NormalizeTSQLIdentifierText(text)
		if strings.HasPrefix(normalizedText, normalizedPrefix) {
			filtered = append(filtered, candidate)
		}
	}
	return filtered
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
	for _, keyword := range []string{"FROM", "JOIN", "APPLY", "USING"} {
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

func (o *objectRefContext) setLinkedServer(linkedServer string) {
	o.linkedServer = linkedServer
	o.flags &= ^objectFlagShowLinkedServer
}

func (o *objectRefContext) setDatabase(database string) {
	o.database = database
	o.flags &= ^objectFlagShowDatabase
}

func (o *objectRefContext) setSchema(schema string) {
	o.schema = schema
	o.flags &= ^objectFlagShowSchema
}

func (o *objectRefContext) setObject(object string) {
	o.object = object
	o.flags &= ^objectFlagShowObject
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

func (c *Completer) collectCompletionScopeReferences(completionContext *omnimssql.CompletionContext) {
	if completionContext == nil || completionContext.Scope == nil {
		return
	}
	var references []base.TableReference
	appendReference := func(reference omnimssql.RangeReference) {
		if converted := c.convertCompletionScopeReference(reference); converted != nil {
			references = append(references, converted)
		}
	}
	if completionContext.Scope.DMLTarget != nil {
		appendReference(*completionContext.Scope.DMLTarget)
	}
	if completionContext.Scope.MergeTarget != nil {
		appendReference(*completionContext.Scope.MergeTarget)
	}
	if completionContext.Scope.MergeSource != nil {
		appendReference(*completionContext.Scope.MergeSource)
	}
	for _, reference := range completionContext.Scope.LocalReferences {
		appendReference(reference)
	}
	for _, level := range completionContext.Scope.OuterReferences {
		for _, reference := range level {
			appendReference(reference)
		}
	}
	c.rememberReferencesIfMissing(references)
}

func (c *Completer) collectCompletionCTEs(completionContext *omnimssql.CompletionContext) {
	if completionContext == nil {
		return
	}
	cteStart := -1
	for _, reference := range completionContext.CTEs {
		virtual := c.convertCompletionVirtualReference(reference)
		if virtual == nil {
			continue
		}
		if cteStart < 0 && reference.Loc.Start >= 0 {
			cteStart = reference.Loc.Start
		}
		if len(virtual.Columns) == 0 {
			virtual.Columns = c.inferCompletionCTEColumns(reference, cteStart)
		}
		c.appendCTEIfMissing(virtual)
	}
}

func (c *Completer) inferCompletionCTEColumns(reference omnimssql.RangeReference, cteStart int) []string {
	if cteStart < 0 || reference.Loc.End < cteStart || reference.Loc.End > len(c.sql) {
		return nil
	}
	table := normalizeCompletionIdentifier(reference.Object)
	if table == "" {
		return nil
	}
	cteBody := c.sql[cteStart:reference.Loc.End]
	statement := fmt.Sprintf("WITH %s SELECT * FROM %s", cteBody, quoteIdentifierForSQL(table))
	span, err := GetQuerySpan(
		c.ctx,
		base.GetQuerySpanContext{
			InstanceID:              c.instanceID,
			GetDatabaseMetadataFunc: c.metadataGetter,
			ListDatabaseNamesFunc:   c.databaseNamesLister,
		},
		base.Statement{Text: statement},
		c.defaultDatabase,
		c.defaultSchema,
		true,
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

func quoteIdentifierForSQL(identifier string) string {
	if isRegularIdentifier(identifier) {
		return identifier
	}
	return fmt.Sprintf("[%s]", strings.ReplaceAll(identifier, "]", "]]"))
}

func (c *Completer) appendCTEIfMissing(reference *base.VirtualTableReference) {
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

func (c *Completer) rememberReferencesIfMissing(references []base.TableReference) {
	c.ensureReferenceMap()
	for _, reference := range references {
		if c.hasReference(reference) {
			continue
		}
		c.rememberReferences([]base.TableReference{reference})
	}
}

func (c *Completer) hasReference(reference base.TableReference) bool {
	key := completionTableReferenceKey(reference)
	for _, existing := range c.references {
		if completionTableReferenceKey(existing) == key {
			return true
		}
	}
	return false
}

func completionTableReferenceKey(reference base.TableReference) string {
	switch reference := reference.(type) {
	case *base.PhysicalTableReference:
		return strings.Join([]string{"physical", reference.Database, reference.Schema, reference.Table, reference.Alias}, "\x00")
	case *base.VirtualTableReference:
		return strings.Join([]string{"virtual", reference.Table}, "\x00")
	default:
		return fmt.Sprintf("%T", reference)
	}
}

func (c *Completer) convertCompletionScopeReference(reference omnimssql.RangeReference) base.TableReference {
	switch reference.Kind {
	case omnimssql.RangeReferenceRelation, omnimssql.RangeReferenceDMLTarget, omnimssql.RangeReferenceMergeTarget, omnimssql.RangeReferenceMergeSource:
		alias := normalizeCompletionIdentifier(reference.Alias)
		if reference.Kind == omnimssql.RangeReferenceMergeSource {
			table := normalizeCompletionIdentifier(reference.Object)
			if columns := c.lookupCTEColumns(table); len(columns) > 0 {
				if alias != "" {
					table = alias
				}
				return &base.VirtualTableReference{
					Table:   table,
					Columns: columns,
				}
			}
		}
		return &base.PhysicalTableReference{
			Database: normalizeCompletionIdentifier(reference.Database),
			Schema:   normalizeCompletionIdentifier(reference.Schema),
			Table:    normalizeCompletionIdentifier(reference.Object),
			Alias:    alias,
		}
	case omnimssql.RangeReferenceCTE, omnimssql.RangeReferenceValues, omnimssql.RangeReferenceSubquery, omnimssql.RangeReferenceTableVariable, omnimssql.RangeReferenceJoinAlias, omnimssql.RangeReferenceFunction:
		return c.convertCompletionVirtualReference(reference)
	default:
		return nil
	}
}

func (c *Completer) convertCompletionVirtualReference(reference omnimssql.RangeReference) *base.VirtualTableReference {
	table := completionReferenceName(reference)
	if table == "" {
		return nil
	}
	columns := completionReferenceColumns(reference)
	if len(columns) == 0 && reference.Kind == omnimssql.RangeReferenceCTE {
		columns = c.lookupCTEColumns(table)
	}
	return &base.VirtualTableReference{
		Table:   table,
		Columns: columns,
	}
}

func completionReferenceName(reference omnimssql.RangeReference) string {
	if reference.Alias != "" {
		return normalizeCompletionIdentifier(reference.Alias)
	}
	return normalizeCompletionIdentifier(reference.Object)
}

func completionReferenceColumns(reference omnimssql.RangeReference) []string {
	columns := reference.Columns
	if len(columns) == 0 {
		columns = reference.AliasColumns
	}
	result := make([]string, 0, len(columns))
	for _, column := range columns {
		result = append(result, normalizeCompletionIdentifier(column))
	}
	return result
}

func normalizeCompletionIdentifier(identifier string) string {
	original, _ := NormalizeTSQLIdentifierText(unquoteDoubleQuoted(identifier))
	return original
}

func (c *Completer) lookupCTEColumns(table string) []string {
	for _, cte := range c.cteTables {
		if strings.EqualFold(cte.Table, table) {
			return cte.Columns
		}
	}
	return nil
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
	for _, token := range c.tokens {
		if token.Loc == pos && mssqlparser.IsIdentTokenType(token.Type) {
			return token.Str
		}
		if token.Loc > pos {
			break
		}
	}
	return ""
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
