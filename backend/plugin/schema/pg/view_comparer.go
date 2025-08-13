package pg

import (
	"regexp"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	pgparser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	// Register PostgreSQL-specific view comparer
	schema.RegisterViewComparer(storepb.Engine_POSTGRES, &PostgreSQLViewComparer{})
}

// PostgreSQLViewComparer provides PostgreSQL-specific view comparison logic.
type PostgreSQLViewComparer struct {
	schema.DefaultViewComparer
}

// CompareView compares two views with PostgreSQL-specific logic using ANTLR parser.
func (c *PostgreSQLViewComparer) CompareView(oldView, newView *model.ViewMetadata) ([]schema.ViewChange, error) {
	if oldView == nil || newView == nil {
		return nil, nil
	}

	var changes []schema.ViewChange

	// Compare definitions using PostgreSQL ANTLR parser
	definitionsEqual := c.compareViewDefinitions(oldView.Definition, newView.Definition)
	if !definitionsEqual {
		changes = append(changes, schema.ViewChange{
			Type:               schema.ViewChangeDefinition,
			Description:        "View definition changed",
			RequiresRecreation: true,
		})
	}

	// Compare comment from proto
	oldProto := oldView.GetProto()
	newProto := newView.GetProto()
	if oldProto != nil && newProto != nil {
		if oldProto.Comment != newProto.Comment {
			changes = append(changes, schema.ViewChange{
				Type:               schema.ViewChangeComment,
				Description:        "View comment changed",
				RequiresRecreation: false,
			})
		}
	}

	return changes, nil
}

// CompareMaterializedView compares two materialized views with PostgreSQL-specific logic.
func (c *PostgreSQLViewComparer) CompareMaterializedView(oldMV, newMV *model.MaterializedViewMetadata) ([]schema.MaterializedViewChange, error) {
	if oldMV == nil || newMV == nil {
		return nil, nil
	}

	var changes []schema.MaterializedViewChange

	// Compare definitions using PostgreSQL ANTLR parser
	definitionsEqual := c.compareViewDefinitions(oldMV.Definition, newMV.Definition)
	if !definitionsEqual {
		changes = append(changes, schema.MaterializedViewChange{
			Type:               schema.MaterializedViewChangeDefinition,
			Description:        "Materialized view definition changed",
			RequiresRecreation: true,
		})
	}

	// Compare comment from proto
	oldProto := oldMV.GetProto()
	newProto := newMV.GetProto()
	if oldProto != nil && newProto != nil {
		if oldProto.Comment != newProto.Comment {
			changes = append(changes, schema.MaterializedViewChange{
				Type:               schema.MaterializedViewChangeComment,
				Description:        "Materialized view comment changed",
				RequiresRecreation: false,
			})
		}

		// Compare indexes - PostgreSQL can add/drop indexes without recreating materialized views
		if !c.compareIndexes(oldProto.Indexes, newProto.Indexes) {
			changes = append(changes, schema.MaterializedViewChange{
				Type:               schema.MaterializedViewChangeIndex,
				Description:        "Materialized view indexes changed",
				RequiresRecreation: false, // PostgreSQL can modify indexes separately
			})
		}

		// Compare triggers
		if !c.compareTriggers(oldProto.Triggers, newProto.Triggers) {
			changes = append(changes, schema.MaterializedViewChange{
				Type:               schema.MaterializedViewChangeTrigger,
				Description:        "Materialized view triggers changed",
				RequiresRecreation: false,
			})
		}
	}

	return changes, nil
}

// compareIndexes compares two index slices for equality.
func (*PostgreSQLViewComparer) compareIndexes(oldIndexes, newIndexes []*storepb.IndexMetadata) bool {
	if len(oldIndexes) != len(newIndexes) {
		return false
	}

	// Create maps for easier comparison
	oldMap := make(map[string]*storepb.IndexMetadata)
	for _, index := range oldIndexes {
		oldMap[index.Name] = index
	}

	for _, newIndex := range newIndexes {
		oldIndex, exists := oldMap[newIndex.Name]
		if !exists {
			return false
		}

		// Compare index properties
		if oldIndex.Unique != newIndex.Unique ||
			oldIndex.Primary != newIndex.Primary ||
			oldIndex.Type != newIndex.Type ||
			oldIndex.Comment != newIndex.Comment ||
			len(oldIndex.Expressions) != len(newIndex.Expressions) {
			return false
		}

		// Compare index expressions
		for i, oldExpr := range oldIndex.Expressions {
			newExpr := newIndex.Expressions[i]
			if oldExpr != newExpr {
				return false
			}
		}
	}

	return true
}

// compareTriggers compares two trigger slices for equality.
func (*PostgreSQLViewComparer) compareTriggers(oldTriggers, newTriggers []*storepb.TriggerMetadata) bool {
	if len(oldTriggers) != len(newTriggers) {
		return false
	}

	// Create maps for easier comparison
	oldMap := make(map[string]*storepb.TriggerMetadata)
	for _, trigger := range oldTriggers {
		oldMap[trigger.Name] = trigger
	}

	for _, newTrigger := range newTriggers {
		oldTrigger, exists := oldMap[newTrigger.Name]
		if !exists {
			return false
		}
		if oldTrigger.Body != newTrigger.Body {
			return false
		}
	}

	return true
}

// compareViewDefinitions compares two view definitions using PostgreSQL ANTLR parser.
func (c *PostgreSQLViewComparer) compareViewDefinitions(oldDef, newDef string) bool {
	// Normalize both definitions
	normalizedOld := c.normalizeViewDefinition(oldDef)
	normalizedNew := c.normalizeViewDefinition(newDef)

	// Compare normalized definitions
	return normalizedOld == normalizedNew
}

// normalizeViewDefinition normalizes a view definition using PostgreSQL parser.
func (c *PostgreSQLViewComparer) normalizeViewDefinition(definition string) string {
	// Step 1: Parse the SQL using ANTLR
	inputStream := antlr.NewInputStream(definition)
	lexer := pgparser.NewPostgreSQLLexer(inputStream)

	// Step 2: Get all tokens and process them
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	stream.Fill()
	tokens := stream.GetAllTokens()

	var normalizedTokens []string

	// Step 3: Process tokens to normalize
	for i, token := range tokens {
		tokenText := token.GetText()
		tokenType := token.GetTokenType()

		// Skip whitespace tokens (spaces, tabs, newlines)
		if c.isWhitespaceToken(tokenType) {
			continue
		}

		// Skip semicolons and EOF tokens
		if tokenText == ";" || tokenText == "<EOF>" {
			continue
		}

		// Remove "public" schema references if it's an identifier
		if c.isIdentifierToken(tokenType) && strings.EqualFold(tokenText, "public") {
			// Check if next non-whitespace token is a dot (.)
			if c.isFollowedByDot(tokens, i) {
				// Skip this "public" identifier and the following dot
				continue
			}
		}

		// Skip dots that follow a skipped "public" identifier
		if tokenText == "." && i > 0 {
			prevToken := c.getPreviousNonWhitespaceToken(tokens, i)
			if prevToken != nil && strings.EqualFold(prevToken.GetText(), "public") && c.isIdentifierToken(prevToken.GetTokenType()) {
				continue
			}
		}

		// Normalize text to lowercase (unless quoted)
		if !c.isQuotedIdentifier(tokenText) {
			tokenText = strings.ToLower(tokenText)
		}

		normalizedTokens = append(normalizedTokens, tokenText)
	}

	// Step 4: Join tokens with single space and trim
	normalized := strings.Join(normalizedTokens, " ")
	normalized = strings.TrimSpace(normalized)

	// Step 5: Clean up extra spaces
	normalized = c.normalizeSpaces(normalized)

	return normalized
}

// isWhitespaceToken checks if the token is whitespace.
func (*PostgreSQLViewComparer) isWhitespaceToken(tokenType int) bool {
	// PostgreSQL lexer whitespace token types
	return tokenType == pgparser.PostgreSQLLexerWhitespace ||
		tokenType == pgparser.PostgreSQLLexerNewline
}

// isIdentifierToken checks if the token is an identifier.
func (*PostgreSQLViewComparer) isIdentifierToken(tokenType int) bool {
	return tokenType == pgparser.PostgreSQLLexerIdentifier ||
		tokenType == pgparser.PostgreSQLLexerQuotedIdentifier
}

// isQuotedIdentifier checks if the identifier is quoted.
func (*PostgreSQLViewComparer) isQuotedIdentifier(text string) bool {
	return len(text) >= 2 && text[0] == '"' && text[len(text)-1] == '"'
}

// isFollowedByDot checks if the token at index i is followed by a dot.
func (c *PostgreSQLViewComparer) isFollowedByDot(tokens []antlr.Token, index int) bool {
	for i := index + 1; i < len(tokens); i++ {
		token := tokens[i]
		if c.isWhitespaceToken(token.GetTokenType()) {
			continue
		}
		return token.GetText() == "."
	}
	return false
}

// getPreviousNonWhitespaceToken gets the previous non-whitespace token.
func (c *PostgreSQLViewComparer) getPreviousNonWhitespaceToken(tokens []antlr.Token, index int) antlr.Token {
	for i := index - 1; i >= 0; i-- {
		token := tokens[i]
		if !c.isWhitespaceToken(token.GetTokenType()) {
			return token
		}
	}
	return nil
}

// normalizeSpaces normalizes multiple spaces to single space.
func (*PostgreSQLViewComparer) normalizeSpaces(text string) string {
	re := regexp.MustCompile(`\s+`)
	return re.ReplaceAllString(text, " ")
}
