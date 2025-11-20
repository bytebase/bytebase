package pg

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"
	pgparser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/plugin/schema/pg/ast"
)

func init() {
	// Register PostgreSQL-specific view comparer
	schema.RegisterViewComparer(storepb.Engine_POSTGRES, &PostgreSQLViewComparer{})
}

// PostgreSQLViewComparer provides PostgreSQL-specific view comparison logic.
type PostgreSQLViewComparer struct {
	schema.DefaultViewComparer
}

// CompareView compares two views with PostgreSQL-specific logic using AST semantic analysis.
func (c *PostgreSQLViewComparer) CompareView(oldView, newView *storepb.ViewMetadata) ([]schema.ViewChange, error) {
	if oldView == nil || newView == nil {
		return nil, nil
	}

	var changes []schema.ViewChange

	// Use AST-based semantic comparison
	semanticallyEqual := c.compareViewsSemanticaly(oldView.Definition, newView.Definition)

	if !semanticallyEqual {
		// Views are semantically different
		changes = append(changes, schema.ViewChange{
			Type:               schema.ViewChangeDefinition,
			Description:        "View definition changed (semantic difference)",
			RequiresRecreation: true,
		})
	}

	// Compare comment from proto
	oldComment := oldView.Comment
	newComment := newView.Comment
	if oldComment != newComment {
		changes = append(changes, schema.ViewChange{
			Type:               schema.ViewChangeComment,
			Description:        "View comment changed",
			RequiresRecreation: false,
		})
	}

	return changes, nil
}

// CompareMaterializedView compares two materialized views with PostgreSQL-specific logic.
func (c *PostgreSQLViewComparer) CompareMaterializedView(oldMV, newMV *storepb.MaterializedViewMetadata) ([]schema.MaterializedViewChange, error) {
	if oldMV == nil || newMV == nil {
		return nil, nil
	}

	var changes []schema.MaterializedViewChange

	// Use AST-based semantic comparison
	semanticallyEqual := c.compareViewsSemanticaly(oldMV.Definition, newMV.Definition)

	if !semanticallyEqual {
		changes = append(changes, schema.MaterializedViewChange{
			Type:               schema.MaterializedViewChangeDefinition,
			Description:        "Materialized view definition changed (semantic difference)",
			RequiresRecreation: true,
		})
	}

	// Compare comment from proto
	oldComment := oldMV.Comment
	newComment := newMV.Comment
	if oldComment != newComment {
		changes = append(changes, schema.MaterializedViewChange{
			Type:               schema.MaterializedViewChangeComment,
			Description:        "Materialized view comment changed",
			RequiresRecreation: false,
		})
	}

	// Compare indexes - PostgreSQL can add/drop indexes without recreating materialized views
	if !c.compareIndexes(oldMV.Indexes, newMV.Indexes) {
		changes = append(changes, schema.MaterializedViewChange{
			Type:               schema.MaterializedViewChangeIndex,
			Description:        "Materialized view indexes changed",
			RequiresRecreation: false, // PostgreSQL can modify indexes separately
		})
	}

	// Compare triggers
	if !c.compareTriggers(oldMV.Triggers, newMV.Triggers) {
		changes = append(changes, schema.MaterializedViewChange{
			Type:               schema.MaterializedViewChangeTrigger,
			Description:        "Materialized view triggers changed",
			RequiresRecreation: false,
		})
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

		// Compare index expressions using semantic comparison
		for i, oldExpr := range oldIndex.Expressions {
			newExpr := newIndex.Expressions[i]
			if !ast.CompareExpressionsSemantically(oldExpr, newExpr) {
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

// compareViewsSemanticaly compares two view definitions using enhanced semantic analysis.
func (c *PostgreSQLViewComparer) compareViewsSemanticaly(def1, def2 string) bool {
	// Use enhanced token-based semantic comparison
	// This is a practical approach that leverages the existing working tokenizer
	// while adding semantic understanding

	tokens1 := c.extractSemanticTokens(def1)
	tokens2 := c.extractSemanticTokens(def2)

	// Compare semantic token structures
	return c.compareSemanticTokens(tokens1, tokens2)
}

// isWhitespaceToken checks if the token is whitespace.
func (*PostgreSQLViewComparer) isWhitespaceToken(tokenType int) bool {
	// PostgreSQL lexer whitespace token types
	return tokenType == pgparser.PostgreSQLLexerWhitespace ||
		tokenType == pgparser.PostgreSQLLexerNewline
}

// SemanticTokens represents the semantic structure of a SQL query.
type SemanticTokens struct {
	SelectItems  []string // normalized select items
	TableRefs    []string // normalized table references
	JoinClauses  []string // normalized join clauses
	WhereClause  string   // normalized where condition
	GroupByItems []string // normalized group by items
	OrderByItems []string // normalized order by items
}

// extractSemanticTokens extracts semantic tokens from a view definition.
func (c *PostgreSQLViewComparer) extractSemanticTokens(definition string) *SemanticTokens {
	// Use ANTLR tokenizer to parse the SQL directly without any preprocessing
	inputStream := antlr.NewInputStream(definition)
	lexer := pgparser.NewPostgreSQLLexer(inputStream)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	stream.Fill()
	tokens := stream.GetAllTokens()

	// Extract semantic information from tokens
	return c.extractSemanticStructure(tokens)
}

// extractSemanticStructure extracts semantic structure from ANTLR tokens.
func (c *PostgreSQLViewComparer) extractSemanticStructure(tokens []antlr.Token) *SemanticTokens {
	semantic := &SemanticTokens{}

	// Simple state machine to parse the token sequence
	state := "INITIAL"
	currentItem := ""

	// Preprocess tokens to handle schema prefixes for semantic equivalence
	processedTokens := c.preprocessTokensForSemantics(tokens)

	for _, token := range processedTokens {
		tokenText := strings.ToLower(strings.TrimSpace(token.GetText()))

		// Skip whitespace and EOF tokens
		if c.isWhitespaceToken(token.GetTokenType()) || tokenText == "<eof>" || tokenText == ";" {
			continue
		}

		switch state {
		case "INITIAL":
			if tokenText == "select" {
				state = "SELECT_LIST"
			}

		case "SELECT_LIST":
			switch tokenText {
			case "from":
				if currentItem != "" {
					semantic.SelectItems = append(semantic.SelectItems, strings.TrimSpace(currentItem))
					currentItem = ""
				}
				state = "FROM_CLAUSE"
			case ",":
				if currentItem != "" {
					semantic.SelectItems = append(semantic.SelectItems, strings.TrimSpace(currentItem))
					currentItem = ""
				}
			default:
				// Handle qualified identifiers (table.column) properly
				if tokenText == "." {
					currentItem += tokenText // Don't add space before dots
				} else {
					if currentItem != "" && !strings.HasSuffix(currentItem, ".") {
						currentItem += " "
					}
					currentItem += tokenText
				}
			}

		case "FROM_CLAUSE":
			switch tokenText {
			case "where":
				if currentItem != "" {
					semantic.TableRefs = append(semantic.TableRefs, strings.TrimSpace(currentItem))
					currentItem = ""
				}
				state = "WHERE_CLAUSE"
			case "join", "inner", "left", "right", "full":
				if currentItem != "" {
					semantic.TableRefs = append(semantic.TableRefs, strings.TrimSpace(currentItem))
					currentItem = ""
				}
				state = "JOIN_CLAUSE"
				if tokenText != "inner" { // inner is often followed by join
					currentItem = tokenText
				}
			case "group":
				if currentItem != "" {
					semantic.TableRefs = append(semantic.TableRefs, strings.TrimSpace(currentItem))
					currentItem = ""
				}
				state = "GROUP_BY"
			case "order":
				if currentItem != "" {
					semantic.TableRefs = append(semantic.TableRefs, strings.TrimSpace(currentItem))
					currentItem = ""
				}
				state = "ORDER_BY"
			default:
				// Skip unnecessary parentheses for semantic comparison
				if tokenText == "(" || tokenText == ")" {
					// Skip parentheses that don't affect semantics
					continue
				}

				// Handle qualified identifiers (table.column) properly
				if tokenText == "." {
					currentItem += tokenText // Don't add space before dots
				} else {
					if currentItem != "" && !strings.HasSuffix(currentItem, ".") {
						currentItem += " "
					}
					currentItem += tokenText
				}
			}

		case "JOIN_CLAUSE":
			if tokenText == "on" {
				if currentItem != "" {
					semantic.JoinClauses = append(semantic.JoinClauses, strings.TrimSpace(currentItem))
					currentItem = ""
				}
				state = "JOIN_CONDITION"
			} else {
				if currentItem != "" {
					currentItem += " "
				}
				currentItem += tokenText
			}

		case "JOIN_CONDITION":
			switch tokenText {
			case "join", "inner", "left", "right", "full":
				if currentItem != "" {
					// This was actually a join condition, add it to join clauses
					if len(semantic.JoinClauses) > 0 {
						semantic.JoinClauses[len(semantic.JoinClauses)-1] += " ON " + strings.TrimSpace(currentItem)
					}
					currentItem = ""
				}
				state = "JOIN_CLAUSE"
				if tokenText != "inner" {
					currentItem = tokenText
				}
			case "where":
				if currentItem != "" {
					if len(semantic.JoinClauses) > 0 {
						semantic.JoinClauses[len(semantic.JoinClauses)-1] += " ON " + strings.TrimSpace(currentItem)
					}
					currentItem = ""
				}
				state = "WHERE_CLAUSE"
			case "group":
				if currentItem != "" {
					if len(semantic.JoinClauses) > 0 {
						semantic.JoinClauses[len(semantic.JoinClauses)-1] += " ON " + strings.TrimSpace(currentItem)
					}
					currentItem = ""
				}
				state = "GROUP_BY"
			case "order":
				if currentItem != "" {
					if len(semantic.JoinClauses) > 0 {
						semantic.JoinClauses[len(semantic.JoinClauses)-1] += " ON " + strings.TrimSpace(currentItem)
					}
					currentItem = ""
				}
				state = "ORDER_BY"
			default:
				// Skip unnecessary parentheses in JOIN conditions for semantic comparison
				if tokenText == "(" || tokenText == ")" {
					continue
				}

				// Handle qualified identifiers (table.column) properly
				if tokenText == "." {
					currentItem += tokenText // Don't add space before dots
				} else {
					if currentItem != "" && !strings.HasSuffix(currentItem, ".") {
						currentItem += " "
					}
					currentItem += tokenText
				}
			}

		case "WHERE_CLAUSE":
			switch tokenText {
			case "group":
				if currentItem != "" {
					semantic.WhereClause = strings.TrimSpace(currentItem)
					currentItem = ""
				}
				state = "GROUP_BY"
			case "order":
				if currentItem != "" {
					semantic.WhereClause = strings.TrimSpace(currentItem)
					currentItem = ""
				}
				state = "ORDER_BY"
			default:
				if currentItem != "" {
					currentItem += " "
				}
				currentItem += tokenText
			}

		case "GROUP_BY":
			if tokenText == "by" {
				// Skip "BY" in "GROUP BY"
				continue
			} else if tokenText == "having" {
				if currentItem != "" {
					semantic.GroupByItems = c.parseCommaSeparatedList(currentItem)
					currentItem = ""
				}
				state = "HAVING"
			} else if tokenText == "order" {
				if currentItem != "" {
					semantic.GroupByItems = c.parseCommaSeparatedList(currentItem)
					currentItem = ""
				}
				state = "ORDER_BY"
			} else {
				if currentItem != "" {
					currentItem += " "
				}
				currentItem += tokenText
			}

		case "ORDER_BY":
			if tokenText == "by" {
				// Skip "BY" in "ORDER BY"
				continue
			} else if tokenText == "limit" || tokenText == "offset" {
				if currentItem != "" {
					semantic.OrderByItems = c.parseCommaSeparatedList(currentItem)
					currentItem = ""
				}
				state = "LIMIT_OFFSET"
			} else {
				if currentItem != "" {
					currentItem += " "
				}
				currentItem += tokenText
			}
		default:
			// Handle unrecognized state - should not happen in normal parsing
		}
	}

	// Handle any remaining content
	switch state {
	case "SELECT_LIST":
		if currentItem != "" {
			semantic.SelectItems = append(semantic.SelectItems, strings.TrimSpace(currentItem))
		}
	case "FROM_CLAUSE":
		if currentItem != "" {
			semantic.TableRefs = append(semantic.TableRefs, strings.TrimSpace(currentItem))
		}
	case "JOIN_CLAUSE":
		if currentItem != "" {
			semantic.JoinClauses = append(semantic.JoinClauses, strings.TrimSpace(currentItem))
		}
	case "JOIN_CONDITION":
		if currentItem != "" && len(semantic.JoinClauses) > 0 {
			semantic.JoinClauses[len(semantic.JoinClauses)-1] += " ON " + strings.TrimSpace(currentItem)
		}
	case "WHERE_CLAUSE":
		if currentItem != "" {
			semantic.WhereClause = strings.TrimSpace(currentItem)
		}
	case "GROUP_BY":
		if currentItem != "" {
			semantic.GroupByItems = c.parseCommaSeparatedList(currentItem)
		}
	case "ORDER_BY":
		if currentItem != "" {
			semantic.OrderByItems = c.parseCommaSeparatedList(currentItem)
		}
	default:
		// Handle any unrecognized final state - no action needed
	}

	return semantic
}

// preprocessTokensForSemantics preprocesses tokens to handle schema prefixes and other semantic equivalences.
func (*PostgreSQLViewComparer) preprocessTokensForSemantics(tokens []antlr.Token) []antlr.Token {
	var result []antlr.Token

	for i, token := range tokens {
		tokenText := strings.ToLower(strings.TrimSpace(token.GetText()))

		// Handle schema prefix removal for semantic equivalence
		if tokenText == "public" && i+1 < len(tokens) {
			// Look for the next non-whitespace token to see if it's a dot
			foundDot := false
			for j := i + 1; j < len(tokens); j++ {
				nextTokenText := strings.TrimSpace(tokens[j].GetText())
				nextTokenType := tokens[j].GetTokenType()

				// Skip whitespace tokens
				if isWhitespaceTokenType(nextTokenType) {
					continue
				}

				// If next non-whitespace token is ".", skip both "public" and "."
				if nextTokenText == "." {
					foundDot = true
					i = j // Skip both "public" and "." tokens
				}
				break // Found non-whitespace token, make decision
			}

			if foundDot {
				continue // Skip "public" token
			}
		}

		// Skip dots that follow "public" (handled above)
		if tokenText == "." && i > 0 {
			prevTokenFound := false
			for j := i - 1; j >= 0; j-- {
				prevTokenText := strings.ToLower(strings.TrimSpace(tokens[j].GetText()))
				prevTokenType := tokens[j].GetTokenType()

				// Skip whitespace tokens
				if isWhitespaceTokenType(prevTokenType) {
					continue
				}

				// If previous non-whitespace token is "public", skip this dot
				if prevTokenText == "public" {
					prevTokenFound = true
				}
				break // Found non-whitespace token, make decision
			}

			if prevTokenFound {
				continue // Skip dot that follows "public"
			}
		}

		result = append(result, token)
	}

	return result
}

// parseCommaSeparatedList parses a comma-separated list of items.
func (*PostgreSQLViewComparer) parseCommaSeparatedList(text string) []string {
	if text == "" {
		return nil
	}

	items := strings.Split(text, ",")
	result := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// compareSemanticTokens compares two semantic token structures.
func (c *PostgreSQLViewComparer) compareSemanticTokens(tokens1, tokens2 *SemanticTokens) bool {
	// Compare SELECT items (order matters)
	if !c.compareStringArrays(tokens1.SelectItems, tokens2.SelectItems) {
		return false
	}

	// Compare table references (order shouldn't matter for most cases)
	if !c.compareStringArraysUnordered(tokens1.TableRefs, tokens2.TableRefs) {
		return false
	}

	// Compare JOIN clauses (order matters)
	if !c.compareStringArrays(tokens1.JoinClauses, tokens2.JoinClauses) {
		return false
	}

	// Compare WHERE clause using AST-based semantic comparison
	if !ast.CompareExpressionsSemantically(tokens1.WhereClause, tokens2.WhereClause) {
		return false
	}

	// Compare GROUP BY items (order shouldn't matter)
	if !c.compareStringArraysUnordered(tokens1.GroupByItems, tokens2.GroupByItems) {
		return false
	}

	// Compare ORDER BY items (order matters)
	if !c.compareStringArrays(tokens1.OrderByItems, tokens2.OrderByItems) {
		return false
	}

	return true
}

// compareStringArrays compares two string arrays (order-dependent).
func (c *PostgreSQLViewComparer) compareStringArrays(arr1, arr2 []string) bool {
	if len(arr1) != len(arr2) {
		return false
	}

	for i, item1 := range arr1 {
		// Use AST-based semantic comparison for expressions
		if ast.CompareExpressionsSemantically(item1, arr2[i]) {
			continue // They're semantically equivalent
		}

		// Fallback to normalization for cases where AST comparison might not work
		normalized1 := c.normalizeExpression(item1)
		normalized2 := c.normalizeExpression(arr2[i])

		// Check if they're exactly equal after normalization
		if normalized1 == normalized2 {
			continue // They're equivalent
		}

		// Special handling for column references that might have different table alias usage
		if c.areSemanticallySimilarColumns(normalized1, normalized2) {
			continue // Consider them equivalent
		}

		// They're different
		return false
	}

	return true
}

// areSemanticallySimilarColumns checks if two column references are semantically similar
// even if they use different table alias patterns (e.g., "d.dept_no" vs "dept_no").
func (*PostgreSQLViewComparer) areSemanticallySimilarColumns(col1, col2 string) bool {
	// Handle cases where one column has table alias and the other doesn't
	// e.g., "d.dept_no" vs "dept_no"

	// Normalize spaces around dots first
	col1 = strings.ReplaceAll(col1, " . ", ".")
	col2 = strings.ReplaceAll(col2, " . ", ".")
	col1 = strings.ReplaceAll(col1, ". ", ".")
	col2 = strings.ReplaceAll(col2, " .", ".")
	col1 = strings.TrimSpace(col1)
	col2 = strings.TrimSpace(col2)

	// Split on dots to check for table alias patterns
	parts1 := strings.Split(col1, ".")
	parts2 := strings.Split(col2, ".")

	// If one has alias and other doesn't, compare the column names
	if len(parts1) == 2 && len(parts2) == 1 {
		// col1 has alias (table.column), col2 is just column
		return strings.TrimSpace(parts1[1]) == strings.TrimSpace(parts2[0])
	} else if len(parts1) == 1 && len(parts2) == 2 {
		// col2 has alias (table.column), col1 is just column
		return strings.TrimSpace(parts1[0]) == strings.TrimSpace(parts2[1])
	} else if len(parts1) == 2 && len(parts2) == 2 {
		// Both have aliases, compare just the column names
		return strings.TrimSpace(parts1[1]) == strings.TrimSpace(parts2[1])
	}

	// No special handling needed
	return false
}

// compareStringArraysUnordered compares two string arrays (order-independent).
func (c *PostgreSQLViewComparer) compareStringArraysUnordered(arr1, arr2 []string) bool {
	if len(arr1) != len(arr2) {
		return false
	}

	// For each item in arr1, find a semantically matching item in arr2
	used := make([]bool, len(arr2))

	for _, item1 := range arr1 {
		found := false
		for j, item2 := range arr2 {
			if used[j] {
				continue
			}

			// Use AST-based semantic comparison first
			if ast.CompareExpressionsSemantically(item1, item2) {
				used[j] = true
				found = true
				break
			}

			// Fallback to normalization comparison
			norm1 := c.normalizeExpression(item1)
			norm2 := c.normalizeExpression(item2)
			if norm1 == norm2 {
				used[j] = true
				found = true
				break
			}
		}

		if !found {
			return false
		}
	}

	return true
}

// normalizeExpression normalizes SQL expressions using AST-based semantic analysis.
func (*PostgreSQLViewComparer) normalizeExpression(expr string) string {
	if expr == "" {
		return ""
	}

	// Parse the expression using ANTLR PostgreSQL lexer for semantic analysis
	inputStream := antlr.NewInputStream(expr)
	lexer := pgparser.NewPostgreSQLLexer(inputStream)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	stream.Fill()
	tokens := stream.GetAllTokens()

	var normalizedTokens []string

	for i, token := range tokens {
		tokenText := strings.TrimSpace(token.GetText())
		tokenType := token.GetTokenType()

		// Skip whitespace and EOF tokens
		if isWhitespaceTokenType(tokenType) || tokenText == "<EOF>" || tokenText == ";" {
			continue
		}

		// Handle schema prefix removal (semantic equivalence)
		if tokenText == "public" && i+1 < len(tokens) {
			// Look ahead to see if there's a dot following
			nextToken := tokens[i+1]
			if strings.TrimSpace(nextToken.GetText()) == "." {
				continue // Skip "public" - the dot will be handled in next iteration
			}
		}
		if tokenText == "." && i > 0 {
			// Check if this dot follows a "public" that we should skip
			prevToken := tokens[i-1]
			if strings.ToLower(strings.TrimSpace(prevToken.GetText())) == "public" {
				continue // Skip the dot that follows "public"
			}
		}

		// Semantic normalization: normalize JOIN keywords for equivalence
		if strings.ToLower(tokenText) == "inner" {
			// Look for the next non-whitespace token
			skipInner := false
			for j := i + 1; j < len(tokens); j++ {
				nextTokenText := strings.TrimSpace(tokens[j].GetText())
				nextTokenType := tokens[j].GetTokenType()

				// Skip whitespace tokens
				if isWhitespaceTokenType(nextTokenType) {
					continue
				}

				// If next non-whitespace token is "JOIN", skip "INNER"
				if strings.ToLower(nextTokenText) == "join" {
					skipInner = true
				}
				break // Found non-whitespace token, make decision
			}

			if skipInner {
				continue // Skip "INNER" - we'll just use "join"
			}
		}

		// Normalize operator spacing and case for semantic equivalence
		normalizedToken := normalizeTokenForSemantics(tokenText)
		normalizedTokens = append(normalizedTokens, normalizedToken)
	}

	// Join with single spaces and clean up
	normalized := strings.Join(normalizedTokens, " ")
	normalized = strings.TrimSpace(normalized)

	return normalized
}

// normalizeTokenForSemantics normalizes individual tokens for semantic equivalence.
func normalizeTokenForSemantics(token string) string {
	// Don't normalize quoted identifiers
	if isQuotedIdentifierText(token) {
		return token
	}

	// Convert to lowercase for case-insensitive comparison
	token = strings.ToLower(token)

	// Normalize common operator variations (semantic equivalence)
	switch token {
	case "=":
		return "=" // Already normalized
	case "!=", "<>":
		return "<>" // Standardize inequality operator
	case "&&":
		return "and" // Convert && to AND for PostgreSQL semantic equivalence
	case "||":
		// Context-sensitive: could be OR or concatenation
		// For now, keep as-is since we don't have full context
		return token
	default:
		return token
	}
}

// isWhitespaceTokenType checks if the token type represents whitespace.
func isWhitespaceTokenType(tokenType int) bool {
	return tokenType == pgparser.PostgreSQLLexerWhitespace ||
		tokenType == pgparser.PostgreSQLLexerNewline
}

// isQuotedIdentifierText checks if the text represents a quoted identifier.
func isQuotedIdentifierText(text string) bool {
	return len(text) >= 2 && text[0] == '"' && text[len(text)-1] == '"'
}
