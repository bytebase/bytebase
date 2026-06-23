package cassandra

import (
	"context"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/parser/cql"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/utils"
)

func init() {
	base.RegisterGetQuerySpan(storepb.Engine_CASSANDRA, GetQuerySpan)
}

// GetQuerySpan extracts the query span from a CQL statement.
// TODO(Phase 2): Rewrite to use omni AST instead of ANTLR tree walking.
func GetQuerySpan(ctx context.Context, gCtx base.GetQuerySpanContext, stmt base.Statement, database, _ string, _ bool) (*base.QuerySpan, error) {
	parseResults, err := parseANTLR(stmt.Text)
	if err != nil {
		return nil, err
	}
	if len(parseResults) == 0 {
		return &base.QuerySpan{
			SourceColumns: base.SourceColumnSet{},
			Results:       []base.QuerySpanResult{},
		}, nil
	}
	if len(parseResults) != 1 {
		return nil, errors.Errorf("expecting only one statement to get query span, but got %d", len(parseResults))
	}

	tree := parseResults[0].Tree

	extractor := newQuerySpanExtractor(ctx, database, gCtx)
	antlr.ParseTreeWalkerDefault.Walk(extractor, tree)

	if extractor.err != nil {
		return nil, extractor.err
	}

	return extractor.querySpan, nil
}

// parseANTLR is a temporary ANTLR-based parser used only by GetQuerySpan.
// It will be removed in Phase 2 when QuerySpan is rewritten to use omni AST.
func parseANTLR(statement string) ([]*base.ANTLRAST, error) {
	stmts, err := SplitSQL(statement)
	if err != nil {
		return nil, err
	}

	var result []*base.ANTLRAST
	for _, stmt := range stmts {
		if stmt.Empty {
			continue
		}
		text := strings.TrimRightFunc(stmt.Text, utils.IsSpaceOrSemicolon) + "\n;"
		startPosition := &storepb.Position{Line: int32(stmt.BaseLine()) + 1}

		inputStream := antlr.NewInputStream(text)
		lexer := cql.NewCqlLexer(inputStream)
		stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
		p := cql.NewCqlParser(stream)

		lexer.RemoveErrorListeners()
		lexerErrorListener := &base.ParseErrorListener{
			Statement:     text,
			StartPosition: startPosition,
		}
		lexer.AddErrorListener(lexerErrorListener)

		p.RemoveErrorListeners()
		parserErrorListener := &base.ParseErrorListener{
			Statement:     text,
			StartPosition: startPosition,
		}
		p.AddErrorListener(parserErrorListener)

		p.BuildParseTrees = true
		tree := p.Root()

		if lexerErrorListener.Err != nil {
			return nil, lexerErrorListener.Err
		}
		if parserErrorListener.Err != nil {
			return nil, parserErrorListener.Err
		}

		result = append(result, &base.ANTLRAST{
			StartPosition: startPosition,
			Tree:          tree,
			Tokens:        stream,
		})
	}
	return result, nil
}
