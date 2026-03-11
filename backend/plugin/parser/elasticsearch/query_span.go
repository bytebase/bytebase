package elasticsearch

import (
	"context"
	"strings"

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

	span := &base.QuerySpan{Type: queryType}

	analysis := AnalyzeRequest(req.Method, req.URL, strings.Join(req.Data, "\n"))
	if len(analysis.PredicateFields) > 0 {
		span.PredicatePaths = make(map[string]*base.PathAST, len(analysis.PredicateFields))
		for _, field := range analysis.PredicateFields {
			span.PredicatePaths[field] = dotPathToPathAST(field)
		}
	}

	return span, nil
}

// dotPathToPathAST converts a dot-delimited field path (e.g. "contact.phone")
// into a PathAST with ItemSelector nodes.
func dotPathToPathAST(dotPath string) *base.PathAST {
	parts := strings.Split(dotPath, ".")
	if len(parts) == 0 {
		return nil
	}
	root := base.NewItemSelector(parts[0])
	current := base.SelectorNode(root)
	for _, part := range parts[1:] {
		next := base.NewItemSelector(part)
		current.SetNext(next)
		current = next
	}
	return base.NewPathAST(root)
}
