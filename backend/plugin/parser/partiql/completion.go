package partiql

import (
	"context"
	"slices"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	partiqlparser "github.com/bytebase/parser/partiql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	// globalFollowSetsByState is the global follow sets by state.
	// It is shared by all PostgreSQL completers.
	// The FollowSetsByState is the thread-safe struct.
	globalFollowSetsByState = base.NewFollowSetsByState()

	ignoredTokens = map[int]bool{
		partiqlparser.PartiQLParserParserEOF: true,

		// Operator and literals.
		partiqlparser.PartiQLParserParserCARET:              true,
		partiqlparser.PartiQLParserParserCOMMA:              true,
		partiqlparser.PartiQLParserParserPLUS:               true,
		partiqlparser.PartiQLParserParserMINUS:              true,
		partiqlparser.PartiQLParserParserSLASH_FORWARD:      true,
		partiqlparser.PartiQLParserParserPERCENT:            true,
		partiqlparser.PartiQLParserParserAT_SIGN:            true,
		partiqlparser.PartiQLParserParserTILDE:              true,
		partiqlparser.PartiQLParserParserASTERISK:           true,
		partiqlparser.PartiQLParserParserLT_EQ:              true,
		partiqlparser.PartiQLParserParserGT_EQ:              true,
		partiqlparser.PartiQLParserParserEQ:                 true,
		partiqlparser.PartiQLParserParserNEQ:                true,
		partiqlparser.PartiQLParserParserCONCAT:             true,
		partiqlparser.PartiQLParserParserANGLE_LEFT:         true,
		partiqlparser.PartiQLParserParserANGLE_RIGHT:        true,
		partiqlparser.PartiQLParserParserANGLE_DOUBLE_LEFT:  true,
		partiqlparser.PartiQLParserParserANGLE_DOUBLE_RIGHT: true,
		partiqlparser.PartiQLParserParserBRACKET_LEFT:       true,
		partiqlparser.PartiQLParserParserBRACKET_RIGHT:      true,
		partiqlparser.PartiQLParserParserBRACE_LEFT:         true,
		partiqlparser.PartiQLParserParserBRACE_RIGHT:        true,
		partiqlparser.PartiQLParserParserPAREN_LEFT:         true,
		partiqlparser.PartiQLParserParserPAREN_RIGHT:        true,
		partiqlparser.PartiQLParserParserBACKTICK:           true,
		partiqlparser.PartiQLParserParserCOLON:              true,
		partiqlparser.PartiQLParserParserCOLON_SEMI:         true,
		partiqlparser.PartiQLParserParserQUESTION_MARK:      true,
		partiqlparser.PartiQLParserParserPERIOD:             true,

		// Literals & Identifiers.
		partiqlparser.PartiQLParserParserLITERAL_STRING:    true,
		partiqlparser.PartiQLParserParserLITERAL_INTEGER:   true,
		partiqlparser.PartiQLParserParserLITERAL_DECIMAL:   true,
		partiqlparser.PartiQLParserParserIDENTIFIER:        true,
		partiqlparser.PartiQLParserParserIDENTIFIER_QUOTED: true,

		// To Ignore.
		partiqlparser.PartiQLParserParserUNRECOGNIZED: true,
	}

	preferredRules = map[int]bool{
		// The parser grammar is not friendly to the completion, for the statement "SELECT * FROM Music", the fromClause parse
		// tree is:
		// fromClause -> tableReference -> TableNonJoin -> ...(12) -> varRefExpr -> IDENTIFIER
		// So we set the varRefExpr as the preferred rule to get the table name, although it's not the most accurate rule.
		partiqlparser.PartiQLParserParserRULE_varRefExpr: true,
	}
)

type CompletionMap map[string]base.Candidate

func (m CompletionMap) Insert(entry base.Candidate) {
	m[entry.String()] = entry
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

func (m CompletionMap) insertMetadataColumns(c *Completer) {
	for _, reference := range c.references {
		if physicalTableReference, ok := reference.(*base.PhysicalTableReference); ok {
			database := c.defaultDatabase
			if physicalTableReference.Database != "" {
				database = physicalTableReference.Database
			}
			if database == "" {
				continue
			}
			_, databaseMetadata, err := c.metadataGetter(c.ctx, c.instanceID, c.defaultDatabase)
			if err != nil {
				return
			}
			if databaseMetadata == nil {
				return
			}
			schema := databaseMetadata.GetSchemaMetadata("")
			if schema == nil {
				return
			}
			var tableName string
			for _, table := range schema.ListTableNames() {
				if strings.EqualFold(table, physicalTableReference.Table) {
					tableName = table
					break
				}
			}
			if tableName == "" {
				return
			}
			table := schema.GetTable(tableName)
			if table == nil {
				return
			}
			for _, column := range table.GetProto().GetColumns() {
				if _, ok := m[column.Name]; !ok {
					m.Insert(base.Candidate{
						Type: base.CandidateTypeColumn,
						Text: column.Name,
					})
				}
			}
		}
	}
}

func (m CompletionMap) insertMetadataTables(c *Completer) {
	if c.defaultDatabase == "" {
		return
	}
	_, databaseMetadata, err := c.metadataGetter(c.ctx, c.instanceID, c.defaultDatabase)
	if err != nil {
		return
	}
	if databaseMetadata == nil {
		return
	}
	schema := databaseMetadata.GetSchemaMetadata("")
	if schema == nil {
		return
	}
	for _, table := range schema.ListTableNames() {
		if _, ok := m[table]; !ok {
			m.Insert(base.Candidate{
				Type: base.CandidateTypeTable,
				Text: table,
			})
		}
	}
}

func init() {
	base.RegisterCompleteFunc(storepb.Engine_DYNAMODB, Completion)
}

type Completer struct {
	ctx     context.Context
	core    *base.CodeCompletionCore
	scene   base.SceneType
	parser  *partiqlparser.PartiQLParserParser
	lexer   *partiqlparser.PartiQLLexer
	scanner *base.Scanner

	instanceID      string
	defaultDatabase string
	defaultSchema   string
	metadataGetter  base.GetDatabaseMetadataFunc

	noSeparatorRequired map[int]bool
	// referencesStack is a hierarchical stack of table references.
	// We'll update the stack when we encounter a new FROM clauses.
	referencesStack [][]base.TableReference
	// references is the flattened table references.
	// It's helpful to look up the table reference.
	references []base.TableReference
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

func NewTrickyCompleter(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) *Completer {
	parser, lexer, scanner := prepareTrickyParserAndScanner(statement, caretLine, caretOffset)
	core := base.NewCodeCompletionCore(
		ctx,
		parser,
		ignoredTokens,  /* IgnoredTokens */
		preferredRules, /* PreferredRules */
		&globalFollowSetsByState,
		partiqlparser.PartiQLParserParserRULE_exprSelect, /* queryRule */
		-1, /* shadowQueryRule */
		-1, /* selectItemAliasRule */
		-1, /* cteRule */
	)

	return &Completer{
		ctx:                 ctx,
		core:                core,
		scene:               cCtx.Scene,
		parser:              parser,
		lexer:               lexer,
		scanner:             scanner,
		instanceID:          cCtx.InstanceID,
		defaultDatabase:     cCtx.DefaultDatabase,
		defaultSchema:       "dbo",
		metadataGetter:      cCtx.Metadata,
		noSeparatorRequired: make(map[int]bool),
	}
}

func NewStandardCompleter(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) *Completer {
	parser, lexer, scanner := prepareParserAndScanner(statement, caretLine, caretOffset)
	core := base.NewCodeCompletionCore(
		ctx,
		parser,
		ignoredTokens,  /* IgnoredTokens */
		preferredRules, /* PreferredRules */
		&globalFollowSetsByState,
		partiqlparser.PartiQLParserParserRULE_exprSelect, /* queryRule */
		-1, /* shadowQueryRule */
		-1, /* selectItemAliasRule */
		-1, /* cteRule */
	)

	return &Completer{
		ctx:                 ctx,
		core:                core,
		scene:               cCtx.Scene,
		parser:              parser,
		lexer:               lexer,
		scanner:             scanner,
		instanceID:          cCtx.InstanceID,
		defaultDatabase:     cCtx.DefaultDatabase,
		defaultSchema:       "dbo",
		metadataGetter:      cCtx.Metadata,
		noSeparatorRequired: nil,
	}
}

func (c *Completer) complete() ([]base.Candidate, error) {
	caretIndex := c.scanner.GetIndex()
	if caretIndex > 0 && !c.noSeparatorRequired[c.scanner.GetPreviousTokenType(true)] {
		caretIndex--
	}
	c.referencesStack = append([][]base.TableReference{{}}, c.referencesStack...)
	c.parser.Reset()
	var context antlr.ParserRuleContext
	if c.scene == base.SceneTypeQuery {
		context = c.parser.SelectClause()
	} else {
		context = c.parser.Script()
	}
	candidates := c.core.CollectCandidates(caretIndex, context)

	for ruleName := range candidates.Rules {
		if ruleName == partiqlparser.PartiQLParserParserRULE_varRefExpr {
			c.collectLeadingTableReferences(caretIndex)
			c.takeReferencesSnapshot()
			c.collectRemainingTableReferences()
			c.takeReferencesSnapshot()
		}
	}

	return c.convertCandidates(candidates)
}

func (c *Completer) takeReferencesSnapshot() {
	for _, references := range c.referencesStack {
		c.references = append(c.references, references...)
	}
}

func (c *Completer) convertCandidates(candidates *base.CandidatesCollection) ([]base.Candidate, error) {
	keywordEntries := make(CompletionMap)
	tableEntries := make(CompletionMap)
	columnEntries := make(CompletionMap)
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

	for ruleCandidate, ruleStack := range candidates.Rules {
		c.scanner.PopAndRestore()
		c.scanner.Push()

		switch ruleCandidate {
		case partiqlparser.PartiQLParserParserRULE_varRefExpr:
			isSelectItem := false
			isFromClause := false
			// If previous token contains from, we insert the table names, otherwise we insert the column names.
			for _, rule := range ruleStack {
				if rule.ID == partiqlparser.PartiQLParserParserRULE_selectClause {
					isSelectItem = true
					break
				} else if rule.ID == partiqlparser.PartiQLParserParserRULE_fromClause {
					isFromClause = true
					break
				}
			}

			if isSelectItem {
				columnEntries.insertMetadataColumns(c)
			}
			if isFromClause {
				tableEntries.insertMetadataTables(c)
			}
		default:
		}
	}

	c.scanner.PopAndRestore()
	var result []base.Candidate
	result = append(result, keywordEntries.toSlice()...)
	result = append(result, tableEntries.toSlice()...)
	result = append(result, columnEntries.toSlice()...)
	return result, nil
}

func (c *Completer) collectLeadingTableReferences(caretIndex int) {
	c.scanner.Push()

	c.scanner.SeekIndex(0)

	level := 0
	for {
		found := c.scanner.GetTokenType() == partiqlparser.PartiQLLexerFROM
		for !found {
			if !c.scanner.Forward(false) || c.scanner.GetIndex() >= caretIndex {
				break
			}

			switch c.scanner.GetTokenType() {
			case partiqlparser.PartiQLLexerPAREN_LEFT:
				level++
				c.referencesStack = append([][]base.TableReference{{}}, c.referencesStack...)
			case partiqlparser.PartiQLLexerPAREN_RIGHT:
				if level == 0 {
					c.scanner.PopAndRestore()
					return // We cannot go above the initial nesting level.
				}
			case partiqlparser.PartiQLLexerFROM:
				found = true
			default:
				// Continue scanning for FROM clause
			}
		}
		if !found {
			c.scanner.PopAndRestore()
			return // No FROM clause found.
		}
		c.parseTableReferences(c.scanner.GetFollowingText())
		if c.scanner.GetTokenType() == partiqlparser.PartiQLLexerFROM {
			c.scanner.Forward(false /* skipHidden */)
		}
	}
}

func (c *Completer) collectRemainingTableReferences() {
	c.scanner.Push()

	level := 0
	for {
		found := c.scanner.GetTokenType() == partiqlparser.PartiQLLexerFROM
		for !found {
			if !c.scanner.Forward(false /* skipHidden */) {
				break
			}

			switch c.scanner.GetTokenType() {
			case partiqlparser.PartiQLLexerPAREN_LEFT:
				level++
				c.referencesStack = append([][]base.TableReference{{}}, c.referencesStack...)
			case partiqlparser.PartiQLLexerPAREN_RIGHT:
				if level > 0 {
					level--
				}

			case partiqlparser.PartiQLLexerFROM:
				if level == 0 {
					found = true
				}
			default:
				// Continue scanning
			}
		}

		if !found {
			c.scanner.PopAndRestore()
			return // No more FROM clause found.
		}

		c.parseTableReferences(c.scanner.GetFollowingText())
		if c.scanner.GetTokenType() == partiqlparser.PartiQLLexerFROM {
			c.scanner.Forward(false /* skipHidden */)
		}
	}
}

func (c *Completer) parseTableReferences(fromClause string) {
	input := antlr.NewInputStream(fromClause)
	lexer := partiqlparser.NewPartiQLLexer(input)
	tokens := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := partiqlparser.NewPartiQLParserParser(tokens)

	parser.BuildParseTrees = true
	parser.RemoveErrorListeners()
	tree := parser.FromClause()
	listener := &tableRefListener{
		context:        c,
		fromClauseMode: true,
	}
	antlr.ParseTreeWalkerDefault.Walk(listener, tree)
}

type tableRefListener struct {
	*partiqlparser.BasePartiQLParserListener

	context        *Completer
	fromClauseMode bool
}

func (l *tableRefListener) EnterVarRefExpr(ctx *partiqlparser.VarRefExprContext) {
	name := unquote(ctx.GetText())
	l.context.references = append(l.context.references, &base.PhysicalTableReference{
		Table: name,
	})
}

func unquote(s string) string {
	if len(s) < 2 {
		return s
	}
	if s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}
func prepareTrickyParserAndScanner(statement string, caretLine int, caretOffset int) (*partiqlparser.PartiQLParserParser, *partiqlparser.PartiQLLexer, *base.Scanner) {
	statement, caretLine, caretOffset = skipHeadingSQLs(statement, caretLine, caretOffset)
	statement, caretLine, caretOffset = skipHeadingSQLWithoutSemicolon(statement, caretLine, caretOffset)
	input := antlr.NewInputStream(statement)
	lexer := partiqlparser.NewPartiQLLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := partiqlparser.NewPartiQLParserParser(stream)
	parser.RemoveErrorListeners()
	lexer.RemoveErrorListeners()
	scanner := base.NewScanner(stream, true /* fillInput */)
	scanner.SeekPosition(caretLine, caretOffset)
	scanner.Push()
	return parser, lexer, scanner
}

// caretLine is 1-based and caretOffset is 0-based.
func skipHeadingSQLWithoutSemicolon(statement string, caretLine int, caretOffset int) (string, int, int) {
	input := antlr.NewInputStream(statement)
	lexer := partiqlparser.NewPartiQLLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	lexer.RemoveErrorListeners()
	lexerErrorListener := &base.ParseErrorListener{
		Statement: statement,
	}
	lexer.AddErrorListener(lexerErrorListener)

	stream.Fill()
	tokens := stream.GetAllTokens()
	latestSelect := 0
	newCaretLine, newCaretOffset := caretLine, caretOffset
	for _, token := range tokens {
		if token.GetLine() > caretLine || (token.GetLine() == caretLine && token.GetColumn() >= caretOffset) {
			break
		}
		if token.GetTokenType() == partiqlparser.PartiQLLexerSELECT && token.GetColumn() == 0 {
			latestSelect = token.GetTokenIndex()
			newCaretLine = caretLine - token.GetLine() + 1 // convert to 1-based.
			newCaretOffset = caretOffset
		}
	}

	if latestSelect == 0 {
		return statement, caretLine, caretOffset
	}
	return stream.GetTextFromInterval(antlr.NewInterval(latestSelect, stream.Size())), newCaretLine, newCaretOffset
}

func prepareParserAndScanner(statement string, caretLine int, caretOffset int) (*partiqlparser.PartiQLParserParser, *partiqlparser.PartiQLLexer, *base.Scanner) {
	statement, caretLine, caretOffset = skipHeadingSQLs(statement, caretLine, caretOffset)
	input := antlr.NewInputStream(statement)
	lexer := partiqlparser.NewPartiQLLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := partiqlparser.NewPartiQLParserParser(stream)
	parser.RemoveErrorListeners()
	lexer.RemoveErrorListeners()
	scanner := base.NewScanner(stream, true /* fillInput */)
	scanner.SeekPosition(caretLine, caretOffset)
	scanner.Push()
	return parser, lexer, scanner
}

// skipHeadingSQLs skips the SQL statements which before the caret position.
// caretLine is 1-based and caretOffset is 0-based.
func skipHeadingSQLs(statement string, caretLine int, caretOffset int) (string, int, int) {
	newCaretLine, newCaretOffset := caretLine, caretOffset
	list, err := SplitSQL(statement)
	if err != nil || notEmptySQLCount(list) <= 1 {
		return statement, caretLine, caretOffset
	}

	caretLine-- // Convert to 0-based.

	start := 0
	for i, sql := range list {
		sqlEndLine := int(sql.End.GetLine())
		sqlEndColumn := int(sql.End.GetColumn())
		if sqlEndLine > caretLine || (sqlEndLine == caretLine && sqlEndColumn >= caretOffset) {
			start = i
			if i == 0 {
				// The caret is in the first SQL statement, so we don't need to skip any SQL statements.
				break
			}
			previousSQLEndLine := int(list[i-1].End.GetLine())
			previousSQLEndColumn := int(list[i-1].End.GetColumn())
			newCaretLine = caretLine - previousSQLEndLine + 1 // Convert to 1-based.
			if caretLine == previousSQLEndLine {
				// The caret is in the same line as the last line of the previous SQL statement.
				// End.Column is 1-based exclusive, so (End.Column - 1) gives 0-based start of next statement.
				// newCaretOffset = caretOffset - (previousSQLEndColumn - 1)
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
