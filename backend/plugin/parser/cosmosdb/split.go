package cosmosdb

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/cosmosdb"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_COSMOSDB, SplitSQL)
}

// SplitSQL splits the given SQL statement into multiple SQL statements.
// Note: CosmosDB only supports single SELECT statements, so this returns the entire input as one statement.
func SplitSQL(statement string) ([]base.Statement, error) {
	// CosmosDB doesn't support multiple statements or semicolon delimiters.
	// The grammar only accepts: root: select EOF
	if strings.TrimSpace(statement) == "" {
		return nil, nil
	}

	// Use lexer to get proper position information
	lexer := parser.NewCosmosDBLexer(antlr.NewInputStream(statement))
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	stream.Fill()

	tokens := stream.GetAllTokens()
	if len(tokens) == 0 {
		return nil, nil
	}

	// Check if statement is empty (only whitespace/EOF)
	empty := true
	for _, token := range tokens {
		if token.GetTokenType() == antlr.TokenEOF {
			continue
		}
		if token.GetChannel() != antlr.TokenHiddenChannel {
			empty = false
			break
		}
	}

	var lastToken antlr.Token
	for _, token := range tokens {
		if token.GetTokenType() == antlr.TokenEOF {
			break
		}
		if token.GetChannel() == antlr.TokenDefaultChannel {
			lastToken = token
		}
	}

	if lastToken == nil && len(tokens) > 0 {
		lastToken = tokens[len(tokens)-1]
	}

	// Since Text is the entire statement (including any leading whitespace),
	// Start should point to the first character (position 1,1)
	return []base.Statement{
		{
			Text: statement,
			Range: &storepb.Range{
				Start: 0,
				End:   int32(len(statement)),
			},
			Start: &storepb.Position{
				Line:   1,
				Column: 1,
			},
			End: common.ConvertANTLRTokenToExclusiveEndPosition(
				int32(lastToken.GetLine()),
				int32(lastToken.GetColumn()),
				lastToken.GetText(),
			),
			Empty: empty,
		},
	}, nil
}
