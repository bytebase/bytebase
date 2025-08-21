package cassandra

import (
	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/parser/cql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_CASSANDRA, SplitSQL)
}

// SplitSQL splits CQL statements into multiple single statements.
func SplitSQL(statement string) ([]base.SingleSQL, error) {
	lexer := cql.NewCqlLexer(antlr.NewInputStream(statement))
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := cql.NewCqlParser(stream)

	p.BuildParseTrees = true
	p.RemoveErrorListeners()

	tree := p.Root()
	if tree == nil {
		return nil, nil
	}

	// For now, return the entire statement as a single SQL
	// CQL typically doesn't support multiple statements in one query
	// unlike SQL databases
	return []base.SingleSQL{
		{
			Text:            statement,
			ByteOffsetStart: 0,
			ByteOffsetEnd:   len(statement),
		},
	}, nil
}
