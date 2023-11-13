package mysql

import (
	"context"
	"sort"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	mysql "github.com/bytebase/mysql-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
	"github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	// globalFollowSetsByState is the global follow sets by state.
	// It is shared by all MySQL completers.
	// The FollowSetsByState is the thread-safe struct.
	globalFollowSetsByState = base.NewFollowSetsByState()
)

func init() {
	base.RegisterCompleteFunc(store.Engine_MYSQL, Completion)
	base.RegisterCompleteFunc(store.Engine_MARIADB, Completion)
	base.RegisterCompleteFunc(store.Engine_TIDB, Completion)
}

// Completion is the entry point of MySQL code completion.
func Completion(ctx context.Context, statement string, caretLine int, caretOffset int, defaultDatabase string, metadata base.GetDatabaseMetadataFunc) ([]base.Candidate, error) {
	completer := NewCompleter(ctx, statement, caretLine, caretOffset, defaultDatabase, metadata)
	return completer.completion()
}

func newIgnoredTokens() map[int]bool {
	return map[int]bool{
		mysql.MySQLParserEOF:                      true,
		mysql.MySQLLexerEQUAL_OPERATOR:            true,
		mysql.MySQLLexerASSIGN_OPERATOR:           true,
		mysql.MySQLLexerNULL_SAFE_EQUAL_OPERATOR:  true,
		mysql.MySQLLexerGREATER_OR_EQUAL_OPERATOR: true,
		mysql.MySQLLexerGREATER_THAN_OPERATOR:     true,
		mysql.MySQLLexerLESS_OR_EQUAL_OPERATOR:    true,
		mysql.MySQLLexerLESS_THAN_OPERATOR:        true,
		mysql.MySQLLexerNOT_EQUAL_OPERATOR:        true,
		mysql.MySQLLexerNOT_EQUAL2_OPERATOR:       true,
		mysql.MySQLLexerPLUS_OPERATOR:             true,
		mysql.MySQLLexerMINUS_OPERATOR:            true,
		mysql.MySQLLexerMULT_OPERATOR:             true,
		mysql.MySQLLexerDIV_OPERATOR:              true,
		mysql.MySQLLexerMOD_OPERATOR:              true,
		mysql.MySQLLexerLOGICAL_NOT_OPERATOR:      true,
		mysql.MySQLLexerBITWISE_NOT_OPERATOR:      true,
		mysql.MySQLLexerSHIFT_LEFT_OPERATOR:       true,
		mysql.MySQLLexerSHIFT_RIGHT_OPERATOR:      true,
		mysql.MySQLLexerLOGICAL_AND_OPERATOR:      true,
		mysql.MySQLLexerBITWISE_AND_OPERATOR:      true,
		mysql.MySQLLexerBITWISE_XOR_OPERATOR:      true,
		mysql.MySQLLexerLOGICAL_OR_OPERATOR:       true,
		mysql.MySQLLexerBITWISE_OR_OPERATOR:       true,
		mysql.MySQLLexerDOT_SYMBOL:                true,
		mysql.MySQLLexerCOMMA_SYMBOL:              true,
		mysql.MySQLLexerSEMICOLON_SYMBOL:          true,
		mysql.MySQLLexerCOLON_SYMBOL:              true,
		mysql.MySQLLexerOPEN_PAR_SYMBOL:           true,
		mysql.MySQLLexerCLOSE_PAR_SYMBOL:          true,
		mysql.MySQLLexerOPEN_CURLY_SYMBOL:         true,
		mysql.MySQLLexerCLOSE_CURLY_SYMBOL:        true,
		mysql.MySQLLexerUNDERLINE_SYMBOL:          true,
		mysql.MySQLLexerAT_SIGN_SYMBOL:            true,
		mysql.MySQLLexerAT_AT_SIGN_SYMBOL:         true,
		mysql.MySQLLexerNULL2_SYMBOL:              true,
		mysql.MySQLLexerPARAM_MARKER:              true,
		mysql.MySQLLexerCONCAT_PIPES_SYMBOL:       true,
		mysql.MySQLLexerAT_TEXT_SUFFIX:            true,
		mysql.MySQLLexerBACK_TICK_QUOTED_ID:       true,
		mysql.MySQLLexerSINGLE_QUOTED_TEXT:        true,
		mysql.MySQLLexerDOUBLE_QUOTED_TEXT:        true,
		mysql.MySQLLexerNCHAR_TEXT:                true,
		mysql.MySQLLexerUNDERSCORE_CHARSET:        true,
		mysql.MySQLLexerIDENTIFIER:                true,
		mysql.MySQLLexerINT_NUMBER:                true,
		mysql.MySQLLexerLONG_NUMBER:               true,
		mysql.MySQLLexerULONGLONG_NUMBER:          true,
		mysql.MySQLLexerDECIMAL_NUMBER:            true,
		mysql.MySQLLexerBIN_NUMBER:                true,
		mysql.MySQLLexerHEX_NUMBER:                true,
	}
}

func newPreferredRules() map[int]bool {
	return map[int]bool{
		mysql.MySQLParserRULE_schemaRef:            true,
		mysql.MySQLParserRULE_tableRef:             true,
		mysql.MySQLParserRULE_tableRefWithWildcard: true,
		mysql.MySQLParserRULE_filterTableRef:       true,
		mysql.MySQLParserRULE_columnRef:            true,
		mysql.MySQLParserRULE_columnInternalRef:    true,
		mysql.MySQLParserRULE_tableWild:            true,
		mysql.MySQLParserRULE_functionRef:          true,
		mysql.MySQLParserRULE_functionCall:         true,
		mysql.MySQLParserRULE_runtimeFunctionCall:  true,
		mysql.MySQLParserRULE_triggerRef:           true,
		mysql.MySQLParserRULE_viewRef:              true,
		mysql.MySQLParserRULE_procedureRef:         true,
		mysql.MySQLParserRULE_logfileGroupRef:      true,
		mysql.MySQLParserRULE_tablespaceRef:        true,
		mysql.MySQLParserRULE_engineRef:            true,
		mysql.MySQLParserRULE_collationName:        true,
		mysql.MySQLParserRULE_charsetName:          true,
		mysql.MySQLParserRULE_eventRef:             true,
		mysql.MySQLParserRULE_serverRef:            true,
		mysql.MySQLParserRULE_user:                 true,
		mysql.MySQLParserRULE_userVariable:         true,
		mysql.MySQLParserRULE_systemVariable:       true,
		mysql.MySQLParserRULE_labelRef:             true,
		mysql.MySQLParserRULE_setSystemVariable:    true,
		mysql.MySQLParserRULE_parameterName:        true,
		mysql.MySQLParserRULE_procedureName:        true,
		mysql.MySQLParserRULE_identifier:           true,
		mysql.MySQLParserRULE_labelIdentifier:      true,
	}
}

func newNoSeparatorRequired() map[int]bool {
	return map[int]bool{
		mysql.MySQLLexerEQUAL_OPERATOR:            true,
		mysql.MySQLLexerASSIGN_OPERATOR:           true,
		mysql.MySQLLexerNULL_SAFE_EQUAL_OPERATOR:  true,
		mysql.MySQLLexerGREATER_OR_EQUAL_OPERATOR: true,
		mysql.MySQLLexerGREATER_THAN_OPERATOR:     true,
		mysql.MySQLLexerLESS_OR_EQUAL_OPERATOR:    true,
		mysql.MySQLLexerLESS_THAN_OPERATOR:        true,
		mysql.MySQLLexerNOT_EQUAL_OPERATOR:        true,
		mysql.MySQLLexerNOT_EQUAL2_OPERATOR:       true,
		mysql.MySQLLexerPLUS_OPERATOR:             true,
		mysql.MySQLLexerMINUS_OPERATOR:            true,
		mysql.MySQLLexerMULT_OPERATOR:             true,
		mysql.MySQLLexerDIV_OPERATOR:              true,
		mysql.MySQLLexerMOD_OPERATOR:              true,
		mysql.MySQLLexerLOGICAL_NOT_OPERATOR:      true,
		mysql.MySQLLexerBITWISE_NOT_OPERATOR:      true,
		mysql.MySQLLexerSHIFT_LEFT_OPERATOR:       true,
		mysql.MySQLLexerSHIFT_RIGHT_OPERATOR:      true,
		mysql.MySQLLexerLOGICAL_AND_OPERATOR:      true,
		mysql.MySQLLexerBITWISE_AND_OPERATOR:      true,
		mysql.MySQLLexerBITWISE_XOR_OPERATOR:      true,
		mysql.MySQLLexerLOGICAL_OR_OPERATOR:       true,
		mysql.MySQLLexerBITWISE_OR_OPERATOR:       true,
		mysql.MySQLLexerDOT_SYMBOL:                true,
		mysql.MySQLLexerCOMMA_SYMBOL:              true,
		mysql.MySQLLexerSEMICOLON_SYMBOL:          true,
		mysql.MySQLLexerCOLON_SYMBOL:              true,
		mysql.MySQLLexerOPEN_PAR_SYMBOL:           true,
		mysql.MySQLLexerCLOSE_PAR_SYMBOL:          true,
		mysql.MySQLLexerOPEN_CURLY_SYMBOL:         true,
		mysql.MySQLLexerCLOSE_CURLY_SYMBOL:        true,
		mysql.MySQLLexerPARAM_MARKER:              true,
	}
}

func newSynonyms() map[int][]string {
	return map[int][]string{
		mysql.MySQLLexerCHAR_SYMBOL:         {"CHARACTER"},
		mysql.MySQLLexerNOW_SYMBOL:          {"CURRENT_TIMESTAMP", "LOCALTIME", "LOCALTIMESTAMP"},
		mysql.MySQLLexerDAY_SYMBOL:          {"DAYOFMONTH", "SQL_TSI_DAY"},
		mysql.MySQLLexerDECIMAL_SYMBOL:      {"DEC"},
		mysql.MySQLLexerDISTINCT_SYMBOL:     {"DISTINCTROW"},
		mysql.MySQLLexerCOLUMNS_SYMBOL:      {"FIELDS"},
		mysql.MySQLLexerFLOAT_SYMBOL:        {"FLOAT4"},
		mysql.MySQLLexerDOUBLE_SYMBOL:       {"FLOAT8"},
		mysql.MySQLLexerINT_SYMBOL:          {"INTEGER", "INT4"},
		mysql.MySQLLexerRELAY_THREAD_SYMBOL: {"IO_THREAD"},
		mysql.MySQLLexerSUBSTRING_SYMBOL:    {"MID", "SUBSTR"},
		mysql.MySQLLexerMID_SYMBOL:          {"MEDIUMINT"},
		mysql.MySQLLexerMEDIUMINT_SYMBOL:    {"MIDDLEINT", "INT3"},
		mysql.MySQLLexerNDBCLUSTER_SYMBOL:   {"NDB"},
		mysql.MySQLLexerREGEXP_SYMBOL:       {"RLIKE"},
		mysql.MySQLLexerDATABASE_SYMBOL:     {"SCHEMA"},
		mysql.MySQLLexerDATABASES_SYMBOL:    {"SCHEMAS"},
		mysql.MySQLLexerUSER_SYMBOL:         {"SESSION_USER"},
		mysql.MySQLLexerSTD_SYMBOL:          {"STDDEV", "STDDEV"},
		mysql.MySQLLexerVARCHAR_SYMBOL:      {"VARCHARACTER"},
		mysql.MySQLLexerVARIANCE_SYMBOL:     {"VAR_POP"},
		mysql.MySQLLexerTINYINT_SYMBOL:      {"INT1"},
		mysql.MySQLLexerSMALLINT_SYMBOL:     {"INT2"},
		mysql.MySQLLexerBIGINT_SYMBOL:       {"INT8"},
		mysql.MySQLLexerSECOND_SYMBOL:       {"SQL_TSI_SECOND"},
		mysql.MySQLLexerMINUTE_SYMBOL:       {"SQL_TSI_MINUTE"},
		mysql.MySQLLexerHOUR_SYMBOL:         {"SQL_TSI_HOUR"},
		mysql.MySQLLexerWEEK_SYMBOL:         {"SQL_TSI_WEEK"},
		mysql.MySQLLexerMONTH_SYMBOL:        {"SQL_TSI_MONTH"},
		mysql.MySQLLexerQUARTER_SYMBOL:      {"SQL_TSI_QUARTER"},
		mysql.MySQLLexerYEAR_SYMBOL:         {"SQL_TSI_YEAR"},
	}
}

type TableReference struct {
	Database string
	Table    string
	Alias    string
}

type Completer struct {
	ctx                 context.Context
	core                *base.CodeCompletionCore
	parser              *mysql.MySQLParser
	lexer               *mysql.MySQLLexer
	scanner             *base.Scanner
	defaultDatabase     string
	getMetadata         base.GetDatabaseMetadataFunc
	metadataCache       map[string]*model.DatabaseMetadata
	noSeparatorRequired map[int]bool
	// referencesStack is a hierarchical stack of table references.
	// We'll update the stack when we encounter a new FROM clauses.
	referencesStack [][]*TableReference
	// references is the flattened table references.
	// It's helpful to look up the table reference.
	references []*TableReference
}

func NewCompleter(ctx context.Context, statement string, caretLine int, caretOffset int, defaultDatabase string, metadata base.GetDatabaseMetadataFunc) *Completer {
	parser, lexer, scanner := prepareParserAndScanner(statement, caretLine, caretOffset)
	// For all MySQL completers, we use one global follow sets by state.
	// The FollowSetsByState is the thread-safe struct.
	core := base.NewCodeCompletionCore(parser, newIgnoredTokens(), newPreferredRules(), &globalFollowSetsByState)
	return &Completer{
		ctx:                 ctx,
		core:                core,
		parser:              parser,
		lexer:               lexer,
		scanner:             scanner,
		defaultDatabase:     defaultDatabase,
		getMetadata:         metadata,
		metadataCache:       make(map[string]*model.DatabaseMetadata),
		noSeparatorRequired: newNoSeparatorRequired(),
	}
}

func (c *Completer) completion() ([]base.Candidate, error) {
	caretIndex := c.scanner.GetIndex()
	if caretIndex > 0 && !c.noSeparatorRequired[c.scanner.GetPreviousTokenType(false /* skipHidden */)] {
		caretIndex--
	}
	c.referencesStack = append([][]*TableReference{{}}, c.referencesStack...)
	c.parser.Reset()
	// TODO: we can just skip the head of the caret statement.
	context := c.parser.Script()

	candidates := c.core.CollectCandidates(caretIndex, context)

	if len(candidates.Tokens[mysql.MySQLLexerNOT2_SYMBOL]) > 0 {
		// For code completion, we don't distinguish NOT and NOT2.
		candidates.Tokens[mysql.MySQLLexerNOT_SYMBOL] = candidates.Tokens[mysql.MySQLLexerNOT2_SYMBOL]
		delete(candidates.Tokens, mysql.MySQLLexerNOT2_SYMBOL)
	}

	for ruleName := range candidates.Rules {
		if ruleName == mysql.MySQLParserRULE_columnRef {
			c.collectLeadingTableReferences(caretIndex, false /* forTableAlter */)
			c.takeReferencesSnapshot()
			c.collectRemainingTableReferences()
			c.takeReferencesSnapshot()
			break
		} else if ruleName == mysql.MySQLParserRULE_columnInternalRef {
			c.collectLeadingTableReferences(caretIndex, true /* forTableAlter */)
			c.takeReferencesSnapshot()
		}
	}

	return c.convertCandidates(candidates)
}

func (c *Completer) convertCandidates(candidates *base.CandidatesCollection) ([]base.Candidate, error) {
	synonyms := newSynonyms()
	keywordEntries := make(CompletionMap)
	runtimeFunctionEntries := make(CompletionMap)
	schemaEntries := make(CompletionMap)
	tableEntries := make(CompletionMap)
	columnEntries := make(CompletionMap)
	viewEntries := make(CompletionMap)

	for token, value := range candidates.Tokens {
		entry := c.parser.SymbolicNames[token]
		if strings.HasSuffix(entry, "_SYMBOL") {
			entry = entry[:len(entry)-7]
		} else {
			entry = unquote(entry)
		}

		list := 0
		if len(value) > 0 {
			// For function call:
			if value[0] == mysql.MySQLLexerOPEN_PAR_SYMBOL {
				list = 1
			} else {
				for _, item := range value {
					subEntry := c.parser.SymbolicNames[item]
					if strings.HasSuffix(subEntry, "_SYMBOL") {
						subEntry = subEntry[:len(subEntry)-7]
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

			// Add also synonyms, if there are any.
			if synonyms[token] != nil {
				for _, synonym := range synonyms[token] {
					keywordEntries.Insert(base.Candidate{
						Type: base.CandidateTypeKeyword,
						Text: synonym,
					})
				}
			}
		}
	}

	for candidate := range candidates.Rules {
		c.scanner.PopAndRestore()
		c.scanner.Push()

		switch candidate {
		case mysql.MySQLParserRULE_runtimeFunctionCall:
			// TODO: load runtime functions
			runtimeFunctionEntries.Insert(base.Candidate{
				Type: base.CandidateTypeFunction,
				Text: "runtimeFunction()",
			})
		case mysql.MySQLParserRULE_schemaRef:
			schemaEntries.insertDatabases(c)
		case mysql.MySQLParserRULE_tableRefWithWildcard:
			// A special form of table references (id.id.*) used only in multi-table delete.
			// Handling is similar as for column references (just that we have table/view objects instead of column refs).
			schema, _, flags := c.determineSchemaTableQualifier()
			if flags&ObjectFlagsShowSchemas != 0 {
				schemaEntries.insertDatabases(c)
			}

			schemas := make(map[string]bool)
			if len(schema) == 0 {
				schemas[c.defaultDatabase] = true
			} else {
				schemas[schema] = true
			}
			if flags&ObjectFlagsShowTables != 0 {
				tableEntries.insertTables(c, schemas)
				viewEntries.insertViews(c, schemas)
			}
		case mysql.MySQLParserRULE_tableRef, mysql.MySQLParserRULE_filterTableRef:
			qualifier, flags := c.determineQualifier()

			if flags&ObjectFlagsShowFirst != 0 {
				schemaEntries.insertDatabases(c)
			}

			if flags&ObjectFlagsShowSecond != 0 {
				schemas := make(map[string]bool)
				if len(qualifier) == 0 {
					schemas[c.defaultDatabase] = true
				} else {
					schemas[qualifier] = true
				}

				tableEntries.insertTables(c, schemas)
				viewEntries.insertViews(c, schemas)
			}
		case mysql.MySQLParserRULE_tableWild, mysql.MySQLParserRULE_columnRef:
			schema, table, flags := c.determineSchemaTableQualifier()
			if flags&ObjectFlagsShowSchemas != 0 {
				schemaEntries.insertDatabases(c)
			}

			schemas := make(map[string]bool)
			if len(schema) != 0 {
				schemas[schema] = true
			} else if len(c.references) > 0 {
				for _, reference := range c.references {
					if len(reference.Database) != 0 {
						schemas[reference.Database] = true
					}
				}
			}

			if len(schemas) == 0 {
				schemas[c.defaultDatabase] = true
			}

			if flags&ObjectFlagsShowTables != 0 {
				tableEntries.insertTables(c, schemas)
				if candidate == mysql.MySQLParserRULE_columnRef {
					viewEntries.insertViews(c, schemas)

					for _, reference := range c.references {
						if (len(schema) == 0 && len(reference.Database) == 0) || schemas[reference.Database] {
							if len(reference.Alias) == 0 {
								tableEntries.Insert(base.Candidate{
									Type: base.CandidateTypeTable,
									Text: reference.Table,
								})
							} else {
								tableEntries.Insert(base.Candidate{
									Type: base.CandidateTypeTable,
									Text: reference.Alias,
								})
							}
						}
					}
				}
			}

			if flags&ObjectFlagsShowColumns != 0 {
				if schema == table { // Schema and table are equal if it's not clear if we see a schema or table qualifier.
					schemas[c.defaultDatabase] = true
				}

				tables := make(map[string]bool)
				if len(table) != 0 {
					tables[table] = true

					// Could be an alias
					for _, reference := range c.references {
						if strings.EqualFold(reference.Alias, table) {
							tables[reference.Table] = true
							schemas[reference.Database] = true
						}
					}
				} else if len(c.references) > 0 && candidate == mysql.MySQLParserRULE_columnRef {
					for _, reference := range c.references {
						tables[reference.Table] = true
					}
				}

				if len(tables) > 0 {
					columnEntries.insertColumns(c, schemas, tables)
				}
			}

			// TODO: special handling for triggers.

		case mysql.MySQLParserRULE_viewRef:
			schema, _, flags := c.determineSchemaTableQualifier()

			if flags&ObjectFlagsShowFirst != 0 {
				schemaEntries.insertDatabases(c)
			}

			if flags&ObjectFlagsShowSecond != 0 {
				schemas := make(map[string]bool)
				if len(schema) != 0 {
					schemas[schema] = true
				} else {
					schemas[c.defaultDatabase] = true
				}
				viewEntries.insertViews(c, schemas)
			}
		}
	}

	c.scanner.PopAndRestore()
	var result []base.Candidate
	result = append(result, keywordEntries.toSLice()...)
	result = append(result, runtimeFunctionEntries.toSLice()...)
	result = append(result, schemaEntries.toSLice()...)
	result = append(result, tableEntries.toSLice()...)
	result = append(result, columnEntries.toSLice()...)
	result = append(result, viewEntries.toSLice()...)
	return result, nil
}

type ObjectFlags int

const (
	ObjectFlagsShowSchemas ObjectFlags = 1 << iota
	ObjectFlagsShowTables
	ObjectFlagsShowColumns
	ObjectFlagsShowFirst
	ObjectFlagsShowSecond
)

func (c *Completer) determineSchemaTableQualifier() (schema, table string, flags ObjectFlags) {
	position := c.scanner.GetIndex()
	if c.scanner.GetTokenChannel() != 0 {
		c.scanner.Forward(true /* skipHidden */) // First skip to the next non-hidden token.
	}

	tokenType := c.scanner.GetTokenType()
	if tokenType != mysql.MySQLLexerDOT_SYMBOL && !c.lexer.IsIdentifier(c.scanner.GetTokenType()) {
		// We are at the end of an incomplete identifier spec. Jump back, so that the other tests succeed.
		c.scanner.Backward(true /* skipHidden */)
	}

	if position > 0 {
		if c.lexer.IsIdentifier(c.scanner.GetTokenType()) && c.scanner.GetPreviousTokenType(false /* skipHidden */) == mysql.MySQLLexerDOT_SYMBOL {
			c.scanner.Backward(true /* skipHidden */)
		}
		if c.scanner.IsTokenType(mysql.MySQLLexerDOT_SYMBOL) && c.lexer.IsIdentifier(c.scanner.GetPreviousTokenType(false /* skipHidden */)) {
			c.scanner.Backward(true /* skipHidden */)

			if c.scanner.GetPreviousTokenType(false /* skipHidden */) == mysql.MySQLLexerDOT_SYMBOL {
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

	if !c.scanner.IsTokenType(mysql.MySQLLexerDOT_SYMBOL) || position <= c.scanner.GetIndex() {
		return schema, table, ObjectFlagsShowSchemas | ObjectFlagsShowTables | ObjectFlagsShowColumns
	}

	c.scanner.Forward(true /* skipHidden */) // skip dot
	table = temp
	schema = temp
	if c.lexer.IsIdentifier(c.scanner.GetTokenType()) {
		temp = unquote(c.scanner.GetTokenText())
		c.scanner.Forward(true /* skipHidden */)

		if !c.scanner.IsTokenType(mysql.MySQLLexerDOT_SYMBOL) || position <= c.scanner.GetIndex() {
			return schema, table, ObjectFlagsShowTables | ObjectFlagsShowColumns
		}

		table = temp
		return schema, table, ObjectFlagsShowColumns
	}

	return schema, table, ObjectFlagsShowTables | ObjectFlagsShowColumns
}

func (c *Completer) determineQualifier() (string, ObjectFlags) {
	// Five possible positions here:
	//   - In the first id (including the position directly after the last char).
	//   - In the space between first id and a dot.
	//   - On a dot (visually directly before the dot).
	//   - In space after the dot, that includes the position directly after the dot.
	//   - In the second id.
	// All parts are optional (though not at the same time). The on-dot position is considered the same
	// as in first id as it visually belongs to the first id

	position := c.scanner.GetIndex()
	if c.scanner.GetTokenChannel() != 0 {
		c.scanner.Forward(true /* skipHidden */) // First skip to the next non-hidden token.
	}

	if !c.scanner.IsTokenType(mysql.MySQLLexerDOT_SYMBOL) && !c.lexer.IsIdentifier(c.scanner.GetTokenType()) {
		// We are at the end of an incomplete identifier spec. Jump back, so that the other tests succeed.
		c.scanner.Backward(true /* skipHidden */)
	}

	// Go left until we find something not related to an id or find at most 1 dot.
	if position > 0 {
		if c.lexer.IsIdentifier(c.scanner.GetTokenType()) && c.scanner.GetPreviousTokenType(false /* skipHidden */) == mysql.MySQLLexerDOT_SYMBOL {
			c.scanner.Backward(true /* skipHidden */)
		}
		if c.scanner.IsTokenType(mysql.MySQLLexerDOT_SYMBOL) && c.lexer.IsIdentifier(c.scanner.GetPreviousTokenType(false /* skipHidden */)) {
			c.scanner.Backward(true /* skipHidden */)
		}
	}

	// The c.scanner is now on the leading identifier or dot (if there's no leading id).
	qualifier := ""
	temp := ""
	if c.lexer.IsIdentifier(c.scanner.GetTokenType()) {
		temp = unquote(c.scanner.GetTokenText())
		c.scanner.Forward(true /* skipHidden */)
	}

	if !c.scanner.IsTokenType(mysql.MySQLLexerDOT_SYMBOL) || position <= c.scanner.GetIndex() {
		return qualifier, ObjectFlagsShowFirst | ObjectFlagsShowSecond
	}

	qualifier = temp
	return qualifier, ObjectFlagsShowSecond
}

func (c *Completer) collectRemainingTableReferences() {
	c.scanner.Push()

	level := 0
	for {
		found := c.scanner.GetTokenType() == mysql.MySQLLexerFROM_SYMBOL
		for !found {
			if !c.scanner.Forward(false /* skipHidden */) {
				break
			}

			switch c.scanner.GetTokenType() {
			case mysql.MySQLLexerOPEN_PAR_SYMBOL:
				level++
			case mysql.MySQLLexerCLOSE_PAR_SYMBOL:
				if level > 0 {
					level--
				}
			case mysql.MySQLLexerFROM_SYMBOL:
				// Open and close parentheses don't need to match, if we come from within a subquery.
				if level == 0 {
					found = true
				}
			}
		}

		if !found {
			c.scanner.PopAndRestore()
			return // No more FROM clause found.
		}

		c.parseTableReferences(c.scanner.GetFollowingText())
		if c.scanner.GetTokenType() == mysql.MySQLLexerFROM_SYMBOL {
			c.scanner.Forward(false /* skipHidden */)
		}
	}
}

func (c *Completer) takeReferencesSnapshot() {
	for _, references := range c.referencesStack {
		c.references = append(c.references, references...)
	}
}

func (c *Completer) collectLeadingTableReferences(caretIndex int, forTableAlter bool) {
	c.scanner.Push()

	if forTableAlter {
		// nolint
		for c.scanner.Backward(false /* skipHidden */) && c.scanner.GetTokenType() != mysql.MySQLLexerALTER_SYMBOL {
			// Skip all tokens until ALTER
		}

		if c.scanner.GetTokenType() == mysql.MySQLLexerALTER_SYMBOL {
			c.scanner.SkipTokenSequence([]int{mysql.MySQLLexerALTER_SYMBOL, mysql.MySQLLexerTABLE_SYMBOL})

			var reference TableReference
			reference.Table = unquote(c.scanner.GetTokenText())
			if c.scanner.Forward(false /* skipHidden */) && c.scanner.IsTokenType(mysql.MySQLLexerDOT_SYMBOL) {
				reference.Database = reference.Table
				c.scanner.Forward(false /* skipHidden */)
				c.scanner.Forward(false /* skipHidden */)
				reference.Table = unquote(c.scanner.GetTokenText())
			}
			c.referencesStack[0] = append(c.referencesStack[0], &reference)
		}
	} else {
		c.scanner.SeekIndex(0)

		level := 0
		for {
			found := c.scanner.GetTokenType() == mysql.MySQLLexerFROM_SYMBOL
			for !found {
				if !c.scanner.Forward(false /* skipHidden */) || c.scanner.GetIndex() >= caretIndex {
					break
				}

				switch c.scanner.GetTokenType() {
				case mysql.MySQLLexerOPEN_PAR_SYMBOL:
					level++
					c.referencesStack = append([][]*TableReference{{}}, c.referencesStack...)
				case mysql.MySQLLexerCLOSE_PAR_SYMBOL:
					if level == 0 {
						c.scanner.PopAndRestore()
						return // We cannot go above the initial nesting level.
					}

					level--
					c.referencesStack = c.referencesStack[1:]
				case mysql.MySQLLexerFROM_SYMBOL:
					found = true
				}
			}

			if !found {
				c.scanner.PopAndRestore()
				return // No more FROM clause found.
			}

			c.parseTableReferences(c.scanner.GetFollowingText())
			if c.scanner.GetTokenType() == mysql.MySQLLexerFROM_SYMBOL {
				c.scanner.Forward(false /* skipHidden */)
			}
		}
	}
}

func (c *Completer) parseTableReferences(fromClause string) {
	// We use a local parser just for the FROM clause to avoid messing up tokens on the autocompletion
	// parser (which would affect the processing of the found candidates)
	input := antlr.NewInputStream(fromClause)
	lexer := mysql.NewMySQLLexer(input)
	tokens := antlr.NewCommonTokenStream(lexer, 0)
	parser := mysql.NewMySQLParser(tokens)

	parser.BuildParseTrees = true
	parser.RemoveErrorListeners()
	tree := parser.FromClause()

	listener := &TableRefListener{
		context:        c,
		fromClauseMode: true,
	}
	antlr.ParseTreeWalkerDefault.Walk(listener, tree)
}

type TableRefListener struct {
	*mysql.BaseMySQLParserListener

	context        *Completer
	fromClauseMode bool
	done           bool
	level          int
}

func (l *TableRefListener) ExitTableRef(ctx *mysql.TableRefContext) {
	if l.done {
		return
	}

	if !l.fromClauseMode || l.level == 0 {
		reference := &TableReference{}
		if ctx.QualifiedIdentifier() != nil {
			reference.Table = unquote(ctx.QualifiedIdentifier().Identifier().GetText())
			if ctx.QualifiedIdentifier().DotIdentifier() != nil {
				reference.Database = reference.Table
				reference.Table = unquote(ctx.QualifiedIdentifier().DotIdentifier().Identifier().GetText())
			}
		} else {
			reference.Table = unquote(ctx.DotIdentifier().Identifier().GetText())
		}
		l.context.referencesStack[0] = append(l.context.referencesStack[0], reference)
	}
}

func (l *TableRefListener) ExitTableAlias(ctx *mysql.TableAliasContext) {
	if l.done {
		return
	}

	if l.level == 0 && len(l.context.referencesStack) != 0 && len(l.context.referencesStack[0]) != 0 {
		// Appears after a single or derived table.
		// Since derived tables can be very complex it is not possible here to determine possible columns for
		// completion, hence we just walk over them and thus have no field where to store the found alias.
		l.context.referencesStack[0][len(l.context.referencesStack[0])-1].Alias = unquote(ctx.Identifier().GetText())
	}
}

func (l *TableRefListener) EnterSubquery(_ *mysql.SubqueryContext) {
	if l.done {
		return
	}

	if l.fromClauseMode {
		l.level++
	} else {
		l.context.referencesStack = append([][]*TableReference{{}}, l.context.referencesStack...)
	}
}

func (l *TableRefListener) ExitSubquery(_ *mysql.SubqueryContext) {
	if l.done {
		return
	}

	if l.fromClauseMode {
		l.level--
	} else {
		l.context.referencesStack = l.context.referencesStack[1:]
	}
}

func prepareParserAndScanner(statement string, caretLine int, caretOffset int) (*mysql.MySQLParser, *mysql.MySQLLexer, *base.Scanner) {
	input := antlr.NewInputStream(statement)
	lexer := mysql.NewMySQLLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := mysql.NewMySQLParser(stream)
	parser.RemoveErrorListeners()
	lexer.RemoveErrorListeners()
	scanner := base.NewScanner(stream, true /* fillInput */)
	scanner.SeekPosition(caretLine, caretOffset)
	scanner.Push()
	return parser, lexer, scanner
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

type CompletionMap map[string]base.Candidate

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

func (m CompletionMap) Insert(entry base.Candidate) {
	m[entry.String()] = entry
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
		for _, table := range c.listTables(schema) {
			m.Insert(base.Candidate{
				Type: base.CandidateTypeTable,
				Text: table,
			})
		}
	}
}

func (m CompletionMap) insertViews(c *Completer, schemas map[string]bool) {
	for schema := range schemas {
		for _, view := range c.listViews(schema) {
			m.Insert(base.Candidate{
				Type: base.CandidateTypeView,
				Text: view,
			})
		}
	}
}

func (m CompletionMap) insertColumns(c *Completer, schemas, tables map[string]bool) {
	for _, columnName := range c.listColumns(schemas, tables) {
		m.Insert(base.Candidate{
			Type: base.CandidateTypeColumn,
			Text: columnName,
		})
	}
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
		metadata, err := c.getMetadata(c.ctx, database)
		if err != nil || metadata == nil {
			return nil
		}
		c.metadataCache[database] = metadata
	}

	return c.metadataCache[database].GetSchema("").ListTableNames()
}

func (c *Completer) listViews(database string) []string {
	if _, exists := c.metadataCache[database]; !exists {
		metadata, err := c.getMetadata(c.ctx, database)
		if err != nil || metadata == nil {
			return nil
		}
		c.metadataCache[database] = metadata
	}

	return c.metadataCache[database].GetSchema("").ListViewNames()
}

func (c *Completer) listColumns(databases, tables map[string]bool) []string {
	var result []string
	for database := range databases {
		if _, exists := c.metadataCache[database]; !exists {
			metadata, err := c.getMetadata(c.ctx, database)
			if err != nil || metadata == nil {
				continue
			}
			c.metadataCache[database] = metadata
		}

		for table := range tables {
			for _, column := range c.metadataCache[database].GetSchema("").GetTable(table).GetColumns() {
				result = append(result, column.Name)
			}
		}
	}

	return result
}
