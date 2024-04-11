package tsql

import (
	"context"
	"fmt"
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
)

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
		nil, /* IgnoredTokens */
		nil, /* PreferredRules */
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
	var tokenCandidates []string
	for token := range candidates.Tokens {
		if token < 0 || token >= len(c.parser.GetSymbolicNames()) {
			continue
		}
		text := c.parser.GetSymbolicNames()[token]
		if strings.HasPrefix(strings.ToUpper(text), strings.ToUpper("SEL")) {
			tokenCandidates = append(tokenCandidates, text)
		}
	}

	fmt.Println(tokenCandidates)
	return nil, nil
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
