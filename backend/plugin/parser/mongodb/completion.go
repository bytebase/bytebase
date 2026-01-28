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
		parser.MongoShellParserRULE_dbStatement:      true,
		parser.MongoShellParserRULE_collectionAccess: true,
		parser.MongoShellParserRULE_methodCall:       true,
		parser.MongoShellParserRULE_methodChain:      true,
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

// completionContext represents the detected context for completion.
type completionContext int

const (
	contextUnknown             completionContext = iota
	contextAfterDBDot                            // After "db."
	contextAfterDBDotPartial                     // After "db." with partial identifier (e.g., "db.us")
	contextAfterDBBracket                        // After "db[" - need to insert "name"]
	contextAfterDBBracketQuote                   // After "db["" or "db["x" - need to insert name"]
	contextAfterCollectionDot                    // After "db.collection."
	contextAfterMethodCall                       // After "db.collection.method()." - cursor methods only
)

// detectCompletionContext analyzes the token stream to determine completion context.
func (c *Completer) detectCompletionContext() completionContext {
	// Save current position and restore after analysis
	c.scanner.Push()
	defer c.scanner.PopAndRestore()

	// Collect tokens backwards from caret position
	var tokens []int
	for c.scanner.Backward(true) {
		tokenType := c.scanner.GetTokenType()
		if tokenType == antlr.TokenEOF || tokenType == parser.MongoShellLexerSEMI {
			break
		}
		tokens = append([]int{tokenType}, tokens...)
		if len(tokens) > 30 {
			break
		}
	}

	if len(tokens) < 2 {
		return contextUnknown
	}

	// Find DB token
	dbIndex := -1
	for i, t := range tokens {
		if t == parser.MongoShellLexerDB {
			dbIndex = i
			break
		}
	}

	if dbIndex < 0 {
		return contextUnknown
	}

	tokensAfterDB := tokens[dbIndex+1:]
	if len(tokensAfterDB) == 0 {
		return contextUnknown
	}

	lastToken := tokens[len(tokens)-1]

	// Check if we're in an incomplete bracket access: db[ or db["
	// Pattern: DB LBRACKET (without RBRACKET yet)
	if tokensAfterDB[0] == parser.MongoShellLexerLBRACKET {
		// Check if the bracket is closed
		hasClosingBracket := false
		hasOpenQuote := false
		for _, t := range tokensAfterDB {
			if t == parser.MongoShellLexerRBRACKET {
				hasClosingBracket = true
				break
			}
			// Check for string token (user started typing "...")
			if t == parser.MongoShellLexerDOUBLE_QUOTED_STRING || t == parser.MongoShellLexerSINGLE_QUOTED_STRING {
				hasOpenQuote = true
			}
		}
		if !hasClosingBracket {
			// Still typing inside bracket
			if hasOpenQuote {
				// db["x - has quote, just need name"]
				return contextAfterDBBracketQuote
			}
			// db[ - no quote yet, need "name"]
			return contextAfterDBBracket
		}
		// Bracket is closed, continue to analyze what comes after
	}

	// Count collection accesses (either db.coll or db["coll"])
	// and method calls to determine context
	collectionAccessCount := 0
	hasMethodCall := false

	// State machine to track: db.coll or db["coll"] patterns
	// Note: We don't count the last token if it's a partial identifier (still being typed)
	i := 0
	for i < len(tokensAfterDB) {
		t := tokensAfterDB[i]
		isLastToken := (i == len(tokensAfterDB)-1)

		switch t {
		case parser.MongoShellLexerDOT:
			// After a DOT, check what follows
			if i+1 < len(tokensAfterDB) {
				next := tokensAfterDB[i+1]
				nextIsLast := (i+1 == len(tokensAfterDB)-1)
				// Check if it's an identifier (collection name or method name)
				// But don't count if it's the last token (partial identifier being typed)
				if isIdentifierToken(next) && !nextIsLast {
					collectionAccessCount++
				}
			}
			// If DOT is the last token, we're at db. or db.coll. - don't count yet
			i++
		case parser.MongoShellLexerLBRACKET:
			// Skip bracket access: ["..."]
			for i < len(tokensAfterDB) && tokensAfterDB[i] != parser.MongoShellLexerRBRACKET {
				i++
			}
			if i < len(tokensAfterDB) && !isLastToken {
				// Found RBRACKET, this counts as a collection access
				collectionAccessCount++
			}
			i++
		case parser.MongoShellLexerRPAREN:
			// Check if this is followed by DOT (method chaining)
			if i+1 < len(tokensAfterDB) && tokensAfterDB[i+1] == parser.MongoShellLexerDOT {
				hasMethodCall = true
			}
			i++
		default:
			i++
		}
	}

	// Determine context based on last token and counts
	// Case 1: db. (DOT right after db, no collection access yet)
	if lastToken == parser.MongoShellLexerDOT && collectionAccessCount == 0 {
		return contextAfterDBDot
	}

	// Case 2: db.u (partial identifier after db.)
	if collectionAccessCount == 0 && isIdentifierToken(lastToken) {
		return contextAfterDBDotPartial
	}

	// Case 3: db.coll. or db["coll"]. (one collection access, ends with DOT)
	if lastToken == parser.MongoShellLexerDOT && collectionAccessCount == 1 && !hasMethodCall {
		return contextAfterCollectionDot
	}

	// Case 4: db.coll.find(). or db["coll"].find(). (has method call)
	if lastToken == parser.MongoShellLexerDOT && hasMethodCall {
		return contextAfterMethodCall
	}

	return contextUnknown
}

// isIdentifierToken returns true if the token is an identifier or a method keyword.
func isIdentifierToken(token int) bool {
	// IDENTIFIER token or any method keyword token
	if token == parser.MongoShellLexerIDENTIFIER {
		return true
	}
	// Method keywords are also valid identifiers in collection position
	return isMethodToken(token)
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
func (c *Completer) convertCandidates(candidates *base.CandidatesCollection) ([]base.Candidate, error) {
	var result []base.Candidate

	// Add keyword/method candidates from tokens
	// ANTLR C3 returns token IDs that are valid at the caret position
	for token := range candidates.Tokens {
		if token < 0 || token >= len(c.lexer.LiteralNames) {
			continue
		}

		// Get the literal name (e.g., "'find'", "'aggregate'")
		literal := c.lexer.LiteralNames[token]
		if literal == "" {
			continue
		}

		// Remove quotes from literal name
		text := strings.Trim(literal, "'")
		if text == "" {
			continue
		}

		// Determine candidate type based on token
		candidateType := c.getTokenCandidateType(token)
		if candidateType == base.CandidateTypeFunction {
			// Methods should have () suffix
			result = append(result, base.Candidate{
				Type: base.CandidateTypeFunction,
				Text: text + "()",
			})
		} else {
			result = append(result, base.Candidate{
				Type: candidateType,
				Text: text,
			})
		}
	}

	// Detect context from token stream to determine what suggestions to add
	context := c.detectCompletionContext()

	// Add candidates based on detected context
	// ANTLR C3 may not return all tokens due to grammar complexity,
	// so we inject methods based on the detected context.
	switch context {
	case contextAfterDBDot:
		// After "db." - suggest database methods and collections with bracket notation for special chars
		result = append(result, c.getDatabaseMethodCandidates()...)
		for _, collection := range c.listCollections() {
			// For collections with special characters, use bracket notation
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
	case contextAfterDBDotPartial:
		// After "db.us" - user is typing identifier, suggest plain collection names for filtering
		result = append(result, c.getDatabaseMethodCandidates()...)
		for _, collection := range c.listCollections() {
			result = append(result, base.Candidate{
				Type: base.CandidateTypeTable,
				Text: collection,
			})
		}
	case contextAfterDBBracket:
		// After "db[" - need to insert "name"]
		for _, collection := range c.listCollections() {
			result = append(result, base.Candidate{
				Type: base.CandidateTypeTable,
				Text: formatBracketContent(collection),
			})
		}
	case contextAfterDBBracketQuote:
		// After "db["" or "db["x" - just need name (quote already exists)
		for _, collection := range c.listCollections() {
			result = append(result, base.Candidate{
				Type: base.CandidateTypeTable,
				Text: collection,
			})
		}
	case contextAfterCollectionDot:
		// After "db.collection." - suggest collection and cursor methods
		result = append(result, c.getCollectionMethodCandidates()...)
	case contextAfterMethodCall:
		// After "db.collection.method()." - suggest cursor methods only
		result = append(result, c.getCursorMethodCandidates()...)
	default:
		// For other contexts, use the preferred rules from C3
		for rule := range candidates.Rules {
			switch rule {
			case parser.MongoShellParserRULE_collectionAccess:
				// Suggest collections
				for _, collection := range c.listCollections() {
					result = append(result, base.Candidate{
						Type: base.CandidateTypeTable,
						Text: collection,
					})
				}
			default:
				// Ignore other rules
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

// getTokenCandidateType returns the candidate type for a token.
func (*Completer) getTokenCandidateType(token int) base.CandidateType {
	if isMethodToken(token) {
		return base.CandidateTypeFunction
	}
	return base.CandidateTypeKeyword
}

// getDatabaseMethodCandidates returns candidates for database methods (after "db.").
func (*Completer) getDatabaseMethodCandidates() []base.Candidate {
	methods := []string{
		"getCollectionNames",
		"getCollectionInfos",
		"createCollection",
		"dropDatabase",
		"stats",
		"serverStatus",
		"serverBuildInfo",
		"version",
		"hostInfo",
		"listCommands",
		"runCommand",
		"adminCommand",
		"getName",
		"getMongo",
		"getSiblingDB",
	}
	var result []base.Candidate
	for _, m := range methods {
		result = append(result, base.Candidate{
			Type: base.CandidateTypeFunction,
			Text: m + "()",
		})
	}
	return result
}

// getCollectionMethodCandidates returns candidates for collection methods (after "db.collection.").
func (*Completer) getCollectionMethodCandidates() []base.Candidate {
	methods := []string{
		// Query methods
		"find",
		"findOne",
		"countDocuments",
		"estimatedDocumentCount",
		"distinct",
		"aggregate",
		"getIndexes",
		// Write methods
		"insertOne",
		"insertMany",
		"updateOne",
		"updateMany",
		"deleteOne",
		"deleteMany",
		"replaceOne",
		"findOneAndUpdate",
		"findOneAndReplace",
		"findOneAndDelete",
		// Index methods
		"createIndex",
		"createIndexes",
		"dropIndex",
		"dropIndexes",
		// Collection management
		"drop",
		"renameCollection",
		"stats",
		"storageSize",
		"totalIndexSize",
		"totalSize",
		"dataSize",
		"isCapped",
		"validate",
		"latencyStats",
		"watch",
		// Bulk operations
		"initializeOrderedBulkOp",
		"initializeUnorderedBulkOp",
		// Cursor methods (can be chained after find)
		"sort",
		"limit",
		"skip",
		"projection",
		"project",
		"count",
		"batchSize",
		"close",
		"collation",
		"comment",
		"explain",
		"forEach",
		"hasNext",
		"hint",
		"isClosed",
		"isExhausted",
		"itcount",
		"map",
		"max",
		"maxAwaitTimeMS",
		"maxTimeMS",
		"min",
		"next",
		"noCursorTimeout",
		"objsLeftInBatch",
		"pretty",
		"readConcern",
		"readPref",
		"returnKey",
		"showRecordId",
		"size",
		"tailable",
		"toArray",
		"tryNext",
		"allowDiskUse",
		"addOption",
	}
	var result []base.Candidate
	for _, m := range methods {
		result = append(result, base.Candidate{
			Type: base.CandidateTypeFunction,
			Text: m + "()",
		})
	}
	return result
}

// getCursorMethodCandidates returns candidates for cursor methods only (after method call like "find().").
func (*Completer) getCursorMethodCandidates() []base.Candidate {
	methods := []string{
		// Cursor methods that can be chained
		"sort",
		"limit",
		"skip",
		"projection",
		"project",
		"count",
		"batchSize",
		"close",
		"collation",
		"comment",
		"explain",
		"forEach",
		"hasNext",
		"hint",
		"isClosed",
		"isExhausted",
		"itcount",
		"map",
		"max",
		"maxAwaitTimeMS",
		"maxTimeMS",
		"min",
		"next",
		"noCursorTimeout",
		"objsLeftInBatch",
		"pretty",
		"readConcern",
		"readPref",
		"returnKey",
		"showRecordId",
		"size",
		"tailable",
		"toArray",
		"tryNext",
		"allowDiskUse",
		"addOption",
	}
	var result []base.Candidate
	for _, m := range methods {
		result = append(result, base.Candidate{
			Type: base.CandidateTypeFunction,
			Text: m + "()",
		})
	}
	return result
}

// isMethodToken returns true if the token represents a method.
func isMethodToken(token int) bool {
	switch token {
	// Collection methods
	case parser.MongoShellLexerFIND,
		parser.MongoShellLexerFIND_ONE,
		parser.MongoShellLexerCOUNT_DOCUMENTS,
		parser.MongoShellLexerESTIMATED_DOCUMENT_COUNT,
		parser.MongoShellLexerDISTINCT,
		parser.MongoShellLexerAGGREGATE,
		parser.MongoShellLexerGET_INDEXES,
		parser.MongoShellLexerINSERT_ONE,
		parser.MongoShellLexerINSERT_MANY,
		parser.MongoShellLexerUPDATE_ONE,
		parser.MongoShellLexerUPDATE_MANY,
		parser.MongoShellLexerDELETE_ONE,
		parser.MongoShellLexerDELETE_MANY,
		parser.MongoShellLexerREPLACE_ONE,
		parser.MongoShellLexerFIND_ONE_AND_UPDATE,
		parser.MongoShellLexerFIND_ONE_AND_REPLACE,
		parser.MongoShellLexerFIND_ONE_AND_DELETE,
		parser.MongoShellLexerCREATE_INDEX,
		parser.MongoShellLexerCREATE_INDEXES,
		parser.MongoShellLexerDROP_INDEX,
		parser.MongoShellLexerDROP_INDEXES,
		parser.MongoShellLexerDROP,
		parser.MongoShellLexerRENAME_COLLECTION,
		parser.MongoShellLexerSTATS,
		parser.MongoShellLexerSTORAGE_SIZE,
		parser.MongoShellLexerTOTAL_INDEX_SIZE,
		parser.MongoShellLexerTOTAL_SIZE,
		parser.MongoShellLexerDATA_SIZE,
		parser.MongoShellLexerIS_CAPPED,
		parser.MongoShellLexerVALIDATE,
		parser.MongoShellLexerLATENCY_STATS,
		parser.MongoShellLexerWATCH,
		// Database methods
		parser.MongoShellLexerGET_COLLECTION,
		parser.MongoShellLexerGET_COLLECTION_NAMES,
		parser.MongoShellLexerGET_COLLECTION_INFOS,
		parser.MongoShellLexerCREATE_COLLECTION,
		parser.MongoShellLexerDROP_DATABASE,
		parser.MongoShellLexerHOST_INFO,
		parser.MongoShellLexerLIST_COMMANDS,
		parser.MongoShellLexerSERVER_BUILD_INFO,
		parser.MongoShellLexerSERVER_STATUS,
		parser.MongoShellLexerVERSION,
		parser.MongoShellLexerRUN_COMMAND,
		parser.MongoShellLexerADMIN_COMMAND,
		parser.MongoShellLexerGET_NAME,
		parser.MongoShellLexerGET_MONGO,
		parser.MongoShellLexerGET_SIBLING_DB,
		parser.MongoShellLexerGET_DB,
		parser.MongoShellLexerGET_DB_NAMES,
		// Cursor methods
		parser.MongoShellLexerSORT,
		parser.MongoShellLexerLIMIT,
		parser.MongoShellLexerSKIP_,
		parser.MongoShellLexerPROJECTION,
		parser.MongoShellLexerPROJECT,
		parser.MongoShellLexerCOUNT,
		parser.MongoShellLexerBATCH_SIZE,
		parser.MongoShellLexerCLOSE,
		parser.MongoShellLexerCOLLATION,
		parser.MongoShellLexerCOMMENT,
		parser.MongoShellLexerEXPLAIN,
		parser.MongoShellLexerFOR_EACH,
		parser.MongoShellLexerHAS_NEXT,
		parser.MongoShellLexerHINT,
		parser.MongoShellLexerIS_CLOSED,
		parser.MongoShellLexerIS_EXHAUSTED,
		parser.MongoShellLexerIT_COUNT,
		parser.MongoShellLexerMAP,
		parser.MongoShellLexerMAX,
		parser.MongoShellLexerMAX_AWAIT_TIME_MS,
		parser.MongoShellLexerMAX_TIME_MS,
		parser.MongoShellLexerMIN,
		parser.MongoShellLexerNEXT,
		parser.MongoShellLexerNO_CURSOR_TIMEOUT,
		parser.MongoShellLexerOBJS_LEFT_IN_BATCH,
		parser.MongoShellLexerPRETTY,
		parser.MongoShellLexerREAD_CONCERN,
		parser.MongoShellLexerREAD_PREF,
		parser.MongoShellLexerRETURN_KEY,
		parser.MongoShellLexerSHOW_RECORD_ID,
		parser.MongoShellLexerSIZE,
		parser.MongoShellLexerTAILABLE,
		parser.MongoShellLexerTO_ARRAY,
		parser.MongoShellLexerTRY_NEXT,
		parser.MongoShellLexerALLOW_DISK_USE,
		parser.MongoShellLexerADD_OPTION,
		// Bulk methods
		parser.MongoShellLexerINITIALIZE_ORDERED_BULK_OP,
		parser.MongoShellLexerINITIALIZE_UNORDERED_BULK_OP,
		parser.MongoShellLexerEXECUTE,
		parser.MongoShellLexerGET_OPERATIONS,
		parser.MongoShellLexerTO_STRING,
		parser.MongoShellLexerINSERT,
		parser.MongoShellLexerREMOVE:
		return true
	default:
		return false
	}
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
