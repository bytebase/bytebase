package schema

import (
	"github.com/antlr4-go/antlr/v4"
)

type SDLChunk struct {
	Identifier        string
	ASTNode           antlr.ParserRuleContext   // The parsed AST node for this chunk (e.g., CREATE SEQUENCE)
	AlterStatements   []antlr.ParserRuleContext // Additional ALTER statements for this object (e.g., ALTER SEQUENCE OWNED BY)
	CommentStatements []antlr.ParserRuleContext // COMMENT ON statements for this object
}

// GetText extracts text from the AST node using its own token stream
// For chunks with ALTER statements, it combines the main statement with all ALTER statements
// For chunks with COMMENT statements, it combines them as well
func (c *SDLChunk) GetText() string {
	if c.ASTNode == nil {
		return ""
	}

	// Extract text from the main AST node (e.g., CREATE SEQUENCE)
	mainText := extractTextFromNode(c.ASTNode)

	// Collect all parts: main, ALTER statements, and COMMENT statements
	parts := []string{mainText}

	// Add ALTER statements
	for _, alterNode := range c.AlterStatements {
		alterText := extractTextFromNode(alterNode)
		if alterText != "" {
			parts = append(parts, alterText)
		}
	}

	// Add COMMENT statements
	for _, commentNode := range c.CommentStatements {
		commentText := extractTextFromNode(commentNode)
		if commentText != "" {
			parts = append(parts, commentText)
		}
	}

	// Join with double newline to match SDL format
	result := ""
	for i, part := range parts {
		if i > 0 {
			result += "\n\n"
		}
		result += part
	}
	return result
}

// GetTextWithoutComments extracts text excluding COMMENT ON statements
// This is used for comparing object definitions without considering comments
func (c *SDLChunk) GetTextWithoutComments() string {
	if c.ASTNode == nil {
		return ""
	}

	// Extract text from the main AST node
	mainText := extractTextFromNode(c.ASTNode)

	// If there are no ALTER statements, return just the main text
	if len(c.AlterStatements) == 0 {
		return mainText
	}

	// Combine main statement with ALTER statements (excluding COMMENT statements)
	parts := []string{mainText}
	for _, alterNode := range c.AlterStatements {
		alterText := extractTextFromNode(alterNode)
		if alterText != "" {
			parts = append(parts, alterText)
		}
	}

	// Join with double newline to match SDL format
	result := ""
	for i, part := range parts {
		if i > 0 {
			result += "\n\n"
		}
		result += part
	}
	return result
}

// extractTextFromNode is a helper function to extract text from a parser rule context
func extractTextFromNode(node antlr.ParserRuleContext) string {
	if node == nil {
		return ""
	}

	// Check for interfaces that have the required methods
	type parserContext interface {
		GetParser() antlr.Parser
		GetStart() antlr.Token
		GetStop() antlr.Token
	}

	if ruleContext, ok := node.(parserContext); ok {
		if parser := ruleContext.GetParser(); parser != nil {
			if tokenStream := parser.GetTokenStream(); tokenStream != nil {
				start := ruleContext.GetStart()
				stop := ruleContext.GetStop()
				if start != nil && stop != nil {
					return tokenStream.GetTextFromTokens(start, stop)
				}
			}
		}
	}

	// Fallback to node's GetText method
	return node.GetText()
}

type SDLChunks struct {
	Tables            map[string]*SDLChunk // key: table name/identifier
	Views             map[string]*SDLChunk // key: view name/identifier
	MaterializedViews map[string]*SDLChunk // key: materialized view name/identifier
	Functions         map[string]*SDLChunk // key: function name/identifier
	Triggers          map[string]*SDLChunk // key: schema.table.trigger_name (table-scoped)
	Indexes           map[string]*SDLChunk // key: index name/identifier
	Sequences         map[string]*SDLChunk // key: sequence name/identifier
	Schemas           map[string]*SDLChunk // key: schema name/identifier
	EnumTypes         map[string]*SDLChunk // key: enum type name/identifier
	Extensions        map[string]*SDLChunk // key: extension name/identifier
	EventTriggers     map[string]*SDLChunk // key: event trigger name (database-level, no schema)

	// Column comments: map[schemaName.tableName][columnName] -> COMMENT ON COLUMN AST node
	ColumnComments map[string]map[string]antlr.ParserRuleContext

	// Index comments for table-level indexes: map[schemaName.tableName][indexName] -> COMMENT ON INDEX AST node
	IndexComments map[string]map[string]antlr.ParserRuleContext
}

type SDLDiff struct {
	CurrentChunks  *SDLChunks
	PreviousChunks *SDLChunks
	Added          []*SDLChunk
	Removed        []*SDLChunk
	Modified       []*SDLChunk
}
