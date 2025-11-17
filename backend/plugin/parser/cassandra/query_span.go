package cassandra

import (
	"context"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterGetQuerySpan(storepb.Engine_CASSANDRA, GetQuerySpan)
}

// GetQuerySpan extracts the query span from a CQL statement.
func GetQuerySpan(_ context.Context, gCtx base.GetQuerySpanContext, statement, database, _ string, _ bool) (*base.QuerySpan, error) {
	parseResults, err := ParseCassandraSQL(statement)
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

	// Create extractor and walk the tree
	extractor := newQuerySpanExtractor(database, gCtx)
	antlr.ParseTreeWalkerDefault.Walk(extractor, tree)

	if extractor.err != nil {
		return nil, extractor.err
	}

	return extractor.querySpan, nil
}
