package tsql

import (
	"context"
	"sort"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	tsqlparser "github.com/bytebase/tsql-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterCompleteFunc(storepb.Engine_MSSQL, Completion)
}

var (
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
		tsqlparser.TSqlParserRULE_full_table_name:    true,
	}
)

type CompletionMap map[string]base.Candidate

func (m CompletionMap) Insert(entry base.Candidate) {
	m[entry.String()] = entry
}

func (m CompletionMap) toSLice() []base.Candidate {
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

func (m CompletionMap) insertDatabases(c *Completer, linkedServer string) {
	if linkedServer != "" {
		return
	}

	if c.defaultDatabase != "" {
		m[c.defaultDatabase] = base.Candidate{
			Type: base.CandidateTypeDatabase,
			Text: c.defaultDatabase,
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
				Text: database,
			}
		}
	}
}

type Completer struct {
	ctx     context.Context
	core    *base.CodeCompletionCore
	parser  *tsqlparser.TSqlParser
	lexer   *tsqlparser.TSqlLexer
	scanner *base.Scanner

	defaultDatabase     string
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
	caretTokenIsQuoted bool
}

func Completion(ctx context.Context, statement string, caretLine int, caretOffset int, defaultDatabase string, metadataGetter base.GetDatabaseMetadataFunc, databaseNamesLister base.ListDatabaseNamesFunc) ([]base.Candidate, error) {
	completer := NewStandardCompleter(ctx, statement, caretLine, caretOffset, defaultDatabase, metadataGetter, databaseNamesLister)
	result, err := completer.complete()
	if err != nil {
		return nil, err
	}
	if len(result) > 0 {
		return result, nil
	}

	return []base.Candidate{
		{
			Text:       "o.O",
			Type:       base.CandidateTypeKeyword,
			Definition: "This is a test completion item.",
			Comment:    "This is item comment",
		},
	}, nil
}

func NewStandardCompleter(ctx context.Context, statement string, caretLine int, caretOffset int, defaultDatabase string, metadataGetter base.GetDatabaseMetadataFunc, databaseNamesLister base.ListDatabaseNamesFunc) *Completer {
	parser, lexer, scanner := prepareParserAndScanner(statement, caretLine, caretOffset)
	core := base.NewCodeCompletionCore(
		parser,
		ignoredTokens,  /* IgnoredTokens */
		preferredRules, /* PreferredRules */
		&globalFellowSetsByState,
		0, /* queryRule */
		0, /* shadowQueryRule */
		0, /* selectItemAliasRule */
		0, /* cteRule */
	)

	return &Completer{
		ctx:                 ctx,
		core:                core,
		parser:              parser,
		lexer:               lexer,
		scanner:             scanner,
		defaultDatabase:     defaultDatabase,
		metadataGetter:      metadataGetter,
		databaseNamesLister: databaseNamesLister,
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

func (c *Completer) complete() ([]base.Candidate, error) {
	caretIndex := c.scanner.GetIndex()
	if caretIndex > 0 && !c.noSeparatorRequired[c.scanner.GetPreviousTokenType(true)] {
		caretIndex--
	}
	c.referencesStack = append([][]base.TableReference{{}}, c.referencesStack...)
	c.parser.Reset()
	context := c.parser.Tsql_file()
	candidates := c.core.CollectCandidates(caretIndex, context)
	return c.convertCandidates(candidates)
}

func (c *Completer) convertCandidates(candidates *base.CandidatesCollection) ([]base.Candidate, error) {
	keywordEntries := make(CompletionMap)
	functionEntries := make(CompletionMap)
	databaseEntries := make(CompletionMap)
	schemaEntries := make(CompletionMap)
	tableEntries := make(CompletionMap)
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

	for ruleCandidate := range candidates.Rules {
		c.scanner.PopAndRestore()
		c.scanner.Push()

		switch ruleCandidate {
		case tsqlparser.TSqlParserRULE_built_in_functions:
			functionEntries.insertBuiltinFunctions()
		case tsqlparser.TSqlParserRULE_full_table_name:
			completionContexts := c.determineFullTableNameContext()
			for _, context := range completionContexts {
				if context.flags&objectFlagShowDatabase != 0 {
					databaseEntries.insertDatabases(c, context.linkedServer)
				}
			}
		}
	}

	c.scanner.PopAndRestore()
	var result []base.Candidate
	result = append(result, keywordEntries.toSLice()...)
	result = append(result, functionEntries.toSLice()...)
	result = append(result, databaseEntries.toSLice()...)
	result = append(result, schemaEntries.toSLice()...)
	result = append(result, tableEntries.toSLice()...)
	result = append(result, viewEntries.toSLice()...)
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

func newObjectRefContext(options ...objectRefContextOption) *objectRefContext {
	o := &objectRefContext{
		flags: objectFlagShowLinkedServer | objectFlagShowDatabase | objectFlagShowSchema | objectFlagShowObject,
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
			return deriveObjectRefContextsFromCandidates(candidates)
		}
		candidates = append(candidates, temp)
		c.scanner.Forward(true /* skipHidden */)
		if count > 3 {
			break
		}
	}

	return deriveObjectRefContextsFromCandidates(candidates)
}

// deriveObjectRefContextsFromCandidates derives the object reference contexts from the candidates.
// The T-SQL grammar's object reference likes [linked_server_name.][database_name.][schema_name.][object_name]
// The size of candidates is the window size in the object reference,
// for example, if the candidates are ["a", "b", "c"], the size is 3,
// and objectRefContext would be [linked_server_name: "a", database_name: "b", schema_name: "c", object_name: ""] or[linked_server_name: "", database_name: "a", schema_name: "b", object_name: "c"].
func deriveObjectRefContextsFromCandidates(candidates []string) []*objectRefContext {
	if len(candidates) == 0 {
		return []*objectRefContext{
			newObjectRefContext(),
		}
	}

	switch len(candidates) {
	case 1:
		return []*objectRefContext{
			newObjectRefContext().setLinkedServer(candidates[0]),
			newObjectRefContext().setLinkedServer("").setDatabase(candidates[0]),
			newObjectRefContext().setLinkedServer("").setDatabase("").setSchema(candidates[0]),
			newObjectRefContext().setLinkedServer("").setDatabase("").setSchema("").setObject(candidates[0]),
		}
	case 2:
		return []*objectRefContext{
			newObjectRefContext().setLinkedServer(candidates[0]).setDatabase(candidates[1]),
			newObjectRefContext().setLinkedServer("").setDatabase(candidates[0]).setSchema(candidates[1]),
			newObjectRefContext().setLinkedServer("").setDatabase("").setSchema(candidates[0]).setObject(candidates[1]),
		}
	case 3:
		return []*objectRefContext{
			newObjectRefContext().setLinkedServer(candidates[0]).setDatabase(candidates[1]).setSchema(candidates[2]),
			newObjectRefContext().setLinkedServer("").setDatabase(candidates[0]).setSchema(candidates[1]).setObject(candidates[2]),
		}
	case 4:
		return []*objectRefContext{
			newObjectRefContext().setLinkedServer(candidates[0]).setDatabase(candidates[1]).setSchema(candidates[2]).setObject(candidates[3]),
		}
	}

	return []*objectRefContext{
		newObjectRefContext(),
	}
}

// skipHeadingSQLs skips the SQL statements which before the caret position.
// caretLine is 1-based and caretOffset is 0-based.
func skipHeadingSQLs(statement string, caretLine int, caretOffset int) (string, int, int) {
	list, err := SplitSQL(statement)
	if err != nil || notEmptySQLCount(list) <= 1 {
		return statement, caretLine, caretOffset
	}

	// The caretLine is 1-based and caretOffset is 0-based, and our splitter returns 0-based line and 0-based column,
	// So we need to convert the caretLine to 0-based.
	caretLine-- // Convert to 0-based.

	start, newCaretLine, newCaretOffset := 0, 0, 0
	for i, sql := range list {
		if sql.LastLine < caretLine {
			continue
		}
		if sql.LastLine == caretLine && sql.LastColumn < caretOffset {
			continue
		}

		start = i
		if i == 0 {
			// The caret is in the first SQL statement, so we don't need to skip any SQL statements.
			break
		}
		newCaretLine = caretLine - list[i-1].LastLine

		if caretLine == list[i-1].LastLine {
			// The caret is in the same line as the last line of the previous SQL statement.
			// We need to adjust the caret offset.
			newCaretOffset = caretOffset - list[i-1].LastColumn
		}
		// TODO(zp): here is difference from other languate, I thought we should break because we only
		// skip the SQL statement before the caret position.
		break
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
