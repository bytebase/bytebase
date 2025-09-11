package schema

import (
	"github.com/antlr4-go/antlr/v4"
)

type SDLChunk struct {
	Identifier string
	ASTNode    antlr.ParserRuleContext // The parsed AST node for this chunk
}

// GetText extracts text from the AST node using the provided token stream
func (c *SDLChunk) GetText(tokens *antlr.CommonTokenStream) string {
	if c.ASTNode == nil || tokens == nil {
		return ""
	}

	start := c.ASTNode.GetStart()
	stop := c.ASTNode.GetStop()

	if start == nil || stop == nil {
		return ""
	}

	return tokens.GetTextFromTokens(start, stop)
}

type SDLChunks struct {
	Tables    map[string]*SDLChunk     // key: table name/identifier
	Views     map[string]*SDLChunk     // key: view name/identifier
	Functions map[string]*SDLChunk     // key: function name/identifier
	Indexes   map[string]*SDLChunk     // key: index name/identifier
	Sequences map[string]*SDLChunk     // key: sequence name/identifier
	Tokens    *antlr.CommonTokenStream // Token stream for extracting text from AST nodes
}

type SDLDiff struct {
	CurrentChunks  *SDLChunks
	PreviousChunks *SDLChunks
	Added          []*SDLChunk
	Removed        []*SDLChunk
	Modified       []*SDLChunk
}
