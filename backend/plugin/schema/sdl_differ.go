package schema

import (
	"github.com/antlr4-go/antlr/v4"
)

type SDLChunk struct {
	Identifier      string
	ASTNode         antlr.ParserRuleContext   // The parsed AST node for this chunk (e.g., CREATE SEQUENCE)
	AlterStatements []antlr.ParserRuleContext // Additional ALTER statements for this object (e.g., ALTER SEQUENCE OWNED BY)
}

// GetText extracts text from the AST node using its own token stream
// For chunks with ALTER statements, it combines the main statement with all ALTER statements
func (c *SDLChunk) GetText() string {
	if c.ASTNode == nil {
		return ""
	}

	// Extract text from the main AST node (e.g., CREATE SEQUENCE)
	mainText := extractTextFromNode(c.ASTNode)

	// If there are no ALTER statements, return just the main text
	if len(c.AlterStatements) == 0 {
		return mainText
	}

	// Combine main statement with ALTER statements
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
	Tables    map[string]*SDLChunk // key: table name/identifier
	Views     map[string]*SDLChunk // key: view name/identifier
	Functions map[string]*SDLChunk // key: function name/identifier
	Indexes   map[string]*SDLChunk // key: index name/identifier
	Sequences map[string]*SDLChunk // key: sequence name/identifier
}

type SDLDiff struct {
	CurrentChunks  *SDLChunks
	PreviousChunks *SDLChunks
	Added          []*SDLChunk
	Removed        []*SDLChunk
	Modified       []*SDLChunk
}
