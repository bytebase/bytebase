package mongodb

import (
	"context"
	"regexp"
	"slices"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/mongodb"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

// validIdentifierRegexp matches valid JavaScript identifiers that can be used with dot notation.
// Valid identifiers: start with letter/underscore/$, contain only letters/digits/underscore/$.
var validIdentifierRegexp = regexp.MustCompile(`^[a-zA-Z_$][a-zA-Z0-9_$]*$`)

var globalFollowSetsByState = base.NewFollowSetsByState()

func init() {
	base.RegisterCompleteFunc(storepb.Engine_MONGODB, Completion)
}

func newIgnoredTokens() map[int]bool {
	return map[int]bool{
		antlr.TokenEOF:                             true,
		parser.MongoShellLexerLPAREN:               true,
		parser.MongoShellLexerRPAREN:               true,
		parser.MongoShellLexerLBRACE:               true,
		parser.MongoShellLexerRBRACE:               true,
		parser.MongoShellLexerLBRACKET:             true,
		parser.MongoShellLexerRBRACKET:             true,
		parser.MongoShellLexerCOLON:                true,
		parser.MongoShellLexerCOMMA:                true,
		parser.MongoShellLexerSEMI:                 true,
		parser.MongoShellLexerDOLLAR:               true,
		parser.MongoShellLexerNUMBER:               true,
		parser.MongoShellLexerDOUBLE_QUOTED_STRING: true,
		parser.MongoShellLexerSINGLE_QUOTED_STRING: true,
		parser.MongoShellLexerIDENTIFIER:           true,
		parser.MongoShellLexerREGEX_LITERAL:        true,
	}
}

func newPreferredRules() map[int]bool {
	return map[int]bool{
		parser.MongoShellParserRULE_collectionAccess: true,
	}
}

func newNoSeparatorRequired() map[int]bool {
	return map[int]bool{
		parser.MongoShellLexerLPAREN:   true,
		parser.MongoShellLexerRPAREN:   true,
		parser.MongoShellLexerLBRACE:   true,
		parser.MongoShellLexerRBRACE:   true,
		parser.MongoShellLexerLBRACKET: true,
		parser.MongoShellLexerRBRACKET: true,
		parser.MongoShellLexerCOLON:    true,
		parser.MongoShellLexerCOMMA:    true,
		parser.MongoShellLexerDOT:      true,
		parser.MongoShellLexerSEMI:     true,
	}
}

// Completer provides MongoDB-specific code completion.
type Completer struct {
	ctx                 context.Context
	core                *base.CodeCompletionCore
	parser              *parser.MongoShellParser
	lexer               *parser.MongoShellLexer
	scanner             *base.Scanner
	instanceID          string
	defaultDatabase     string
	getMetadata         base.GetDatabaseMetadataFunc
	listDatabaseNames   base.ListDatabaseNamesFunc
	metadataCache       map[string]*model.DatabaseMetadata
	noSeparatorRequired map[int]bool
}

// NewCompleter creates a new MongoDB completer.
func NewCompleter(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) *Completer {
	input := antlr.NewInputStream(statement)
	lexer := parser.NewMongoShellLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := parser.NewMongoShellParser(stream)
	p.RemoveErrorListeners()
	lexer.RemoveErrorListeners()

	scanner := base.NewScanner(stream, true)
	scanner.SeekPosition(caretLine, caretOffset)
	scanner.Push()

	core := base.NewCodeCompletionCore(
		ctx,
		p,
		newIgnoredTokens(),
		newPreferredRules(),
		&globalFollowSetsByState,
		-1, // No query rule for MongoDB
		-1, // No shadow query rule
		-1, // No select item alias rule
		-1, // No CTE rule
	)

	return &Completer{
		ctx:                 ctx,
		core:                core,
		parser:              p,
		lexer:               lexer,
		scanner:             scanner,
		instanceID:          cCtx.InstanceID,
		defaultDatabase:     cCtx.DefaultDatabase,
		getMetadata:         cCtx.Metadata,
		listDatabaseNames:   cCtx.ListDatabaseNames,
		metadataCache:       make(map[string]*model.DatabaseMetadata),
		noSeparatorRequired: newNoSeparatorRequired(),
	}
}

// previousNonDotToken returns the token type of the previous non-DOT token,
// skipping hidden tokens. Used to determine if the caret follows a method
// call (RPAREN) or a collection access (identifier/RBRACKET).
func (c *Completer) previousNonDotToken() int {
	c.scanner.Push()
	defer c.scanner.PopAndRestore()

	for c.scanner.Backward(true) {
		t := c.scanner.GetTokenType()
		if t != parser.MongoShellLexerDOT {
			return t
		}
	}
	return antlr.TokenEOF
}

// listCollections returns collection names from the default database.
func (c *Completer) listCollections() []string {
	if c.defaultDatabase == "" {
		return nil
	}

	if _, exists := c.metadataCache[c.defaultDatabase]; !exists {
		if c.getMetadata == nil {
			return nil
		}
		_, metadata, err := c.getMetadata(c.ctx, c.instanceID, c.defaultDatabase)
		if err != nil || metadata == nil {
			return nil
		}
		c.metadataCache[c.defaultDatabase] = metadata
	}

	metadata := c.metadataCache[c.defaultDatabase]
	var collections []string
	for _, schema := range metadata.ListSchemaNames() {
		schemaMeta := metadata.GetSchemaMetadata(schema)
		if schemaMeta == nil {
			continue
		}
		collections = append(collections, schemaMeta.ListTableNames()...)
	}
	slices.Sort(collections)
	return collections
}

// complete performs the actual completion.
func (c *Completer) complete() ([]base.Candidate, error) {
	caretIndex := c.scanner.GetIndex()
	if caretIndex > 0 && !c.noSeparatorRequired[c.scanner.GetPreviousTokenType(false)] {
		caretIndex--
	}

	c.parser.Reset()
	ctx := c.parser.Program()

	candidates := c.core.CollectCandidates(caretIndex, ctx)

	return c.convertCandidates(candidates)
}

// convertCandidates converts ANTLR C3 candidates to completion candidates.
// It uses C3 preferred rules to determine context:
//   - RULE_collectionAccess: after "db." or "db[" — suggest database methods + collections
//
// C3 tokens are used in all positions. At the database level (db.|), the collectionAccess
// preferred rule prevents C3 from exploring into methodChain, so only legitimate db method
// tokens appear. At collection/cursor level, C3 tokens provide the correct method candidates.
func (c *Completer) convertCandidates(candidates *base.CandidatesCollection) ([]base.Candidate, error) {
	var result []base.Candidate

	_, hasCollectionAccess := candidates.Rules[parser.MongoShellParserRULE_collectionAccess]

	// Convert C3 token candidates to function candidates.
	// All tokens at any position are methods, so treat them as functions with "()" suffix.
	result = append(result, c.getTokenCandidates(candidates)...)

	if hasCollectionAccess {
		// At db.| or db[ position — also add collection candidates from metadata.
		prevToken := c.previousNonDotToken()
		switch prevToken {
		case parser.MongoShellLexerLBRACKET:
			for _, collection := range c.listCollections() {
				result = append(result, base.Candidate{
					Type: base.CandidateTypeTable,
					Text: formatBracketContent(collection),
				})
			}
		default:
			for _, collection := range c.listCollections() {
				if needsBracketNotation(collection) {
					result = append(result, base.Candidate{
						Type: base.CandidateTypeTable,
						Text: formatBracketNotation(collection),
					})
				} else {
					result = append(result, base.Candidate{
						Type: base.CandidateTypeTable,
						Text: collection,
					})
				}
			}
		}
	}

	// Sort and deduplicate
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

	return slices.CompactFunc(result, func(a, b base.Candidate) bool {
		return a.Type == b.Type && a.Text == b.Text
	}), nil
}

// getTokenCandidates converts C3 token candidates to function candidates.
func (c *Completer) getTokenCandidates(candidates *base.CandidatesCollection) []base.Candidate {
	var result []base.Candidate
	for token := range candidates.Tokens {
		if token < 0 || token >= len(c.lexer.LiteralNames) {
			continue
		}
		literal := c.lexer.LiteralNames[token]
		if literal == "" {
			continue
		}
		text := strings.Trim(literal, "'")
		if text == "" {
			continue
		}
		result = append(result, base.Candidate{
			Type: base.CandidateTypeFunction,
			Text: text + "()",
		})
	}
	return result
}

// needsBracketNotation returns true if the collection name contains special characters
// that require bracket notation (db["name"]) instead of dot notation (db.name).
func needsBracketNotation(name string) bool {
	return !validIdentifierRegexp.MatchString(name)
}

// formatBracketNotation formats a collection name for bracket notation access.
// It escapes double quotes and backslashes in the name.
// Used when suggesting after "db." - inserts full ["name"] syntax.
func formatBracketNotation(name string) string {
	// Escape backslashes first, then double quotes
	escaped := strings.ReplaceAll(name, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	return `["` + escaped + `"]`
}

// formatBracketContent formats a collection name for insertion inside brackets.
// Used when user typed "db[" - inserts "name"] to complete the bracket.
func formatBracketContent(name string) string {
	// Escape backslashes first, then double quotes
	escaped := strings.ReplaceAll(name, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	return `"` + escaped + `"]`
}

// Completion is the entry point for MongoDB code completion.
func Completion(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) ([]base.Candidate, error) {
	completer := NewCompleter(ctx, cCtx, statement, caretLine, caretOffset)
	return completer.complete()
}
