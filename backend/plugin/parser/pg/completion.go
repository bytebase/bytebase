package pg

import (
	"context"
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
}

// Completion is the entry point of PostgreSQL code completion.
func Completion(ctx context.Context, statement string, caretLine int, caretOffset int, defaultDatabase string, metadata base.GetDatabaseMetadataFunc) ([]base.Candidate, error) {
	completer := NewCompleter(ctx, statement, caretLine, caretOffset, defaultDatabase, metadata)
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
	metadataCache       map[string]*model.DatabaseMetadata
	noSeparatorRequired map[int]bool
	// referencesStack is a hierarchical stack of table references.
	// We'll update the stack when we encounter a new FROM clauses.
	referencesStack [][]base.TableReference
	// references is the flattened table references.
	// It's helpful to look up the table reference.
	references []base.TableReference
	cteCache   map[int][]*base.VirtualTableReference
	cteTables  []*base.VirtualTableReference
}

func NewCompleter(ctx context.Context, statement string, caretLine int, caretOffset int, defaultDatabase string, getMetadata base.GetDatabaseMetadataFunc) *Completer {
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

	return nil, nil
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
	if l.level == 0 {
		switch {
		case ctx.Relation_expr() != nil:
			reference := &base.PhysicalTableReference{}
			list := NormalizePostgreSQLQualifiedName(ctx.Relation_expr().Qualified_name())
			switch len(list) {
			case 1:
				reference.Table = list[0]
			case 2:
				reference.Schema = list[0]
				reference.Table = list[1]
			case 3:
				reference.Database = list[0]
				reference.Schema = list[1]
				reference.Table = list[2]
			default:
				return
			}

			if ctx.Opt_alias_clause() != nil {

			}

			l.context.referencesStack[0] = append(l.context.referencesStack[0], reference)
		}
	}
}

func normalizeTableAlias(ctx *pg.Opt_alias_clauseContext) (string, []string) {
	if ctx == nil || ctx.Table_alias_clause() == nil {
		return "", nil
	}

	tableAlias := ""
	aliasClause := ctx.Table_alias_clause()
	if aliasClause.Table_alias() != nil {
		tableAlias = normalizePostgreSQLTableAlias(aliasClause.Table_alias())
	}

	var columnAliases []string

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
