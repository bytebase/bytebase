package tsql

import (
	"context"
	"sort"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	tsql "github.com/bytebase/tsql-parser"

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
		tsql.TSqlParserEOF: true,

		// Token with EBNF symbol
		tsql.TSqlParserBACKSLASH:            true,
		tsql.TSqlParserCONVERT:              true, // 'TRY_'? 'CONVERT'
		tsql.TSqlParserDEFAULT_DOUBLE_QUOTE: true, // ["]'DEFAULT'["]
		tsql.TSqlParserDOUBLE_BACK_SLASH:    true, // '\\\\'
		tsql.TSqlParserDOUBLE_FORWARD_SLASH: true, // '//'
		tsql.TSqlParserEXECUTE:              true, // 'EXE CUTE?' // TODO(zp): Find a way to improve this because it is a common keyword.
		tsql.TSqlParserNULL_DOUBLE_QUOTE:    true, // ["]'NULL'["]
		tsql.TSqlParserPARSE:                true, // 'TRY_'? 'PARSE'

		// Abbreviation
		tsql.TSqlParserYEAR_ABBR:        true, // 'yy' | 'yyyy'
		tsql.TSqlParserQUARTER_ABBR:     true, // 'qq' | 'q'
		tsql.TSqlParserMONTH_ABBR:       true, // 'mm' | 'm'
		tsql.TSqlParserDAYOFYEAR_ABBR:   true, // 'dy' | 'y'
		tsql.TSqlParserWEEK_ABBR:        true, // 'wk' | 'ww'
		tsql.TSqlParserDAY_ABBR:         true, // 'dd' | 'd'
		tsql.TSqlParserHOUR_ABBR:        true, // 'hh'
		tsql.TSqlParserMINUTE_ABBR:      true, // 'mi' | 'n'
		tsql.TSqlParserSECOND_ABBR:      true, // 'ss' | 's'
		tsql.TSqlParserMILLISECOND_ABBR: true, // 'ms'
		tsql.TSqlParserMICROSECOND_ABBR: true, // 'mcs'
		tsql.TSqlParserNANOSECOND_ABBR:  true, // 'ns'
		tsql.TSqlParserTZOFFSET_ABBR:    true, // 'tz'
		tsql.TSqlParserISO_WEEK_ABBR:    true, // 'isowk' | 'isoww'
		tsql.TSqlParserWEEKDAY_ABBR:     true, // 'dw'

		tsql.TSqlParserDISK_DRIVE:   true, // [A-Z][:];
		tsql.TSqlParserIPV4_ADDR:    true, // DEC_DIGIT+ '.' DEC_DIGIT+ '.' DEC_DIGIT+ '.' DEC_DIGIT+;
		tsql.TSqlParserSPACE:        true,
		tsql.TSqlParserCOMMENT:      true,
		tsql.TSqlParserLINE_COMMENT: true,

		tsql.TSqlParserDOUBLE_QUOTE_ID:    true,
		tsql.TSqlParserDOUBLE_QUOTE_BLANK: true,
		tsql.TSqlParserSINGLE_QUOTE:       true,
		tsql.TSqlParserSQUARE_BRACKET_ID:  true,
		tsql.TSqlParserLOCAL_ID:           true,
		tsql.TSqlParserDECIMAL:            true,
		tsql.TSqlParserID:                 true,
		tsql.TSqlParserSTRING:             true,
		tsql.TSqlParserBINARY:             true,
		tsql.TSqlParserFLOAT:              true,
		tsql.TSqlParserREAL:               true,

		tsql.TSqlParserEQUAL:        true,
		tsql.TSqlParserGREATER:      true,
		tsql.TSqlParserLESS:         true,
		tsql.TSqlParserEXCLAMATION:  true,
		tsql.TSqlParserPLUS_ASSIGN:  true,
		tsql.TSqlParserMINUS_ASSIGN: true,
		tsql.TSqlParserMULT_ASSIGN:  true,
		tsql.TSqlParserDIV_ASSIGN:   true,
		tsql.TSqlParserMOD_ASSIGN:   true,
		tsql.TSqlParserAND_ASSIGN:   true,
		tsql.TSqlParserXOR_ASSIGN:   true,
		tsql.TSqlParserOR_ASSIGN:    true,

		tsql.TSqlParserDOUBLE_BAR:   true,
		tsql.TSqlParserDOT:          true,
		tsql.TSqlParserUNDERLINE:    true,
		tsql.TSqlParserAT:           true,
		tsql.TSqlParserSHARP:        true,
		tsql.TSqlParserDOLLAR:       true,
		tsql.TSqlParserLR_BRACKET:   true,
		tsql.TSqlParserRR_BRACKET:   true,
		tsql.TSqlParserCOMMA:        true,
		tsql.TSqlParserSEMI:         true,
		tsql.TSqlParserCOLON:        true,
		tsql.TSqlParserDOUBLE_COLON: true,
		tsql.TSqlParserSTAR:         true,
		tsql.TSqlParserDIVIDE:       true,
		tsql.TSqlParserMODULE:       true,
		tsql.TSqlParserPLUS:         true,
		tsql.TSqlParserMINUS:        true,
		tsql.TSqlParserBIT_NOT:      true,
		tsql.TSqlParserBIT_OR:       true,
		tsql.TSqlParserBIT_AND:      true,
		tsql.TSqlParserBIT_XOR:      true,
		tsql.TSqlParserPLACEHOLDER:  true,
	}
	preferredRules = map[int]bool{
		tsql.TSqlParserRULE_select_statement: true,
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

type Completer struct {
	ctx     context.Context
	core    *base.CodeCompletionCore
	parser  *tsql.TSqlParser
	lexer   *tsql.TSqlLexer
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
		ignoredTokens, /* IgnoredTokens */
		nil,           /* PreferredRules */
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

func prepareParserAndScanner(statement string, caretLine int, caretOffset int) (*tsql.TSqlParser, *tsql.TSqlLexer, *base.Scanner) {
	statement, caretLine, caretOffset = skipHeadingSQLs(statement, caretLine, caretOffset)
	input := antlr.NewInputStream(statement)
	lexer := tsql.NewTSqlLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := tsql.NewTSqlParser(stream)
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

	for token, _ := range candidates.Tokens {
		if token < 0 || token >= len(c.parser.GetSymbolicNames()) {
			continue
		}
		// ANTLR4 Golang target seems do not support vacabulary, and we presume that the symbolic name is the token text
		// in Transact-SQL grammar. So we use the symbolic name as the token text.
		// TODO(zp): filter our the token which text is not as same as the symbolic name.
		tokenSymbolicName := c.parser.GetSymbolicNames()[token]

		if !strings.HasPrefix(strings.ToUpper(tokenSymbolicName), strings.ToUpper("SEL")) {
			continue
		}

		// TODO(zp): For the token candidate(most keyword), we should filter out the prefix which is not as same as the token text. But
		// the frontend monaco-editor seems do this for us, but it may meanningful to do this in the future to decrese the data transfter.
		keywordEntries.Insert(base.Candidate{
			Type: base.CandidateTypeKeyword,
			Text: tokenSymbolicName,
		})
	}

	result := make([]base.Candidate, 0, len(keywordEntries))
	result = append(result, keywordEntries.toSLice()...)
	return result, nil
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
		// TODO(zp): here is difference from other languate, I thought we should break becaure we only
		// SKip the SQL statement before the caret position.
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
