package elasticsearch

import (
	"context"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterGetQuerySpan(storepb.Engine_ELASTICSEARCH, GetQuerySpan)
}

// GetQuerySpan returns the query span for an ElasticSearch REST API request.
func GetQuerySpan(
	_ context.Context,
	_ base.GetQuerySpanContext,
	stmt base.Statement,
	_, _ string,
	_ bool,
) (*base.QuerySpan, error) {
	parseResult, err := ParseElasticsearchREST(stmt.Text)
	if err != nil {
		return nil, err
	}

	if parseResult == nil {
		return &base.QuerySpan{Type: base.QueryTypeUnknown}, nil
	}

	if len(parseResult.Errors) > 0 {
		firstErr := parseResult.Errors[0]
		return nil, errors.Errorf("syntax error at line %d, column %d: %s", firstErr.Position.Line, firstErr.Position.Column, firstErr.Message)
	}

	if len(parseResult.Requests) == 0 {
		return &base.QuerySpan{Type: base.QueryTypeUnknown}, nil
	}

	// After splitting, each statement should contain a single request.
	// Use the first request for classification.
	req := parseResult.Requests[0]
	queryType := ClassifyRequest(req.Method, req.URL)

	return &base.QuerySpan{Type: queryType}, nil
}
