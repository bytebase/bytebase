package cassandra

import (
	"context"

	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/parser/cql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterGetQuerySpan(storepb.Engine_CASSANDRA, GetQuerySpan)
}

// GetQuerySpan extracts the query span from a CQL statement.
func GetQuerySpan(_ context.Context, gCtx base.GetQuerySpanContext, statement, database, _ string, _ bool) (*base.QuerySpan, error) {
	lexer := cql.NewCqlLexer(antlr.NewInputStream(statement))
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := cql.NewCqlParser(stream)

	lexerErrorListener := &base.ParseErrorListener{
		Statement: statement,
	}
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(lexerErrorListener)

	parserErrorListener := &base.ParseErrorListener{
		Statement: statement,
	}
	p.RemoveErrorListeners()
	p.AddErrorListener(parserErrorListener)

	p.BuildParseTrees = true
	tree := p.Root()

	if lexerErrorListener.Err != nil {
		return nil, lexerErrorListener.Err
	}
	if parserErrorListener.Err != nil {
		return nil, parserErrorListener.Err
	}

	// Create extractor and walk the tree
	extractor := newQuerySpanExtractor(database, gCtx)
	antlr.ParseTreeWalkerDefault.Walk(extractor, tree)

	if extractor.err != nil {
		return nil, extractor.err
	}

	return extractor.querySpan, nil
}
