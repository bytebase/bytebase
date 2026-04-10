package elasticsearch

import (
	es "github.com/bytebase/omni/elasticsearch"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// ParseResult holds parsed requests and any syntax errors encountered.
type ParseResult struct {
	Requests []*Request
	Errors   []*base.SyntaxError `yaml:"errors,omitempty"`
}

// Request is a single parsed Kibana Dev Console REST request.
type Request struct {
	Method      string
	URL         string
	Data        []string `yaml:"data,omitempty"`
	StartOffset int
	EndOffset   int
}

// ParseElasticsearchREST parses the Elasticsearch REST API request text.
// It delegates to the omni elasticsearch package and converts types.
func ParseElasticsearchREST(text string) (*ParseResult, error) {
	result, err := es.ParseElasticsearchREST(text)
	if err != nil {
		return nil, err
	}
	return convertParseResult(result), nil
}

// convertParseResult converts an omni ParseResult to a bytebase ParseResult.
func convertParseResult(r *es.ParseResult) *ParseResult {
	if r == nil {
		return &ParseResult{}
	}

	var requests []*Request
	for _, req := range r.Requests {
		if req == nil {
			continue
		}
		requests = append(requests, &Request{
			Method:      req.Method,
			URL:         req.URL,
			Data:        req.Data,
			StartOffset: req.StartOffset,
			EndOffset:   req.EndOffset,
		})
	}

	var errors []*base.SyntaxError
	for _, e := range r.Errors {
		if e == nil {
			continue
		}
		errors = append(errors, &base.SyntaxError{
			Position: &storepb.Position{
				Line:   int32(e.Position.Line),
				Column: int32(e.Position.Column),
			},
			Message:    e.Message,
			RawMessage: e.RawMessage,
		})
	}

	return &ParseResult{
		Requests: requests,
		Errors:   errors,
	}
}
