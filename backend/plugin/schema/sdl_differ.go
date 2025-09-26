package schema

import (
	"github.com/antlr4-go/antlr/v4"
)

type SDLChunk struct {
	Identifier string
	ASTNode    antlr.ParserRuleContext // The parsed AST node for this chunk
}

// GetText extracts text from the AST node using its own token stream
func (c *SDLChunk) GetText() string {
	if c.ASTNode == nil {
		return ""
	}

	// Check for interfaces that have the required methods
	type parserContext interface {
		GetParser() antlr.Parser
		GetStart() antlr.Token
		GetStop() antlr.Token
	}

	if ruleContext, ok := c.ASTNode.(parserContext); ok {
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
	return c.ASTNode.GetText()
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
