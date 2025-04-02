package elasticsearch

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	// See https://sourcegraph.com/github.com/elastic/kibana/-/blob/src/platform/packages/shared/kbn-monaco/src/languages/console/parser.test.ts.
	testCases := []struct {
		description string
		input       string
		got         []parsedRequest
	}{
		{
			description: "returns parsedRequests if the input is correct",
			input:       "GET _search",
			got: []parsedRequest{
				{
					startOffset: 0,
					endOffset:   11,
				},
			},
		},
		{
			description: "parses several requests",
			input: `GET _search
POST _test_index`,
			got: []parsedRequest{
				{
					startOffset: 0,
					endOffset:   11,
				},
				{
					startOffset: 12,
					endOffset:   28,
				},
			},
		},
		{
			description: "parses a request with a request body",
			input: `GET _search
{
  "query": {
    "match_all": {}
  }
}`,
			got: []parsedRequest{
				{
					startOffset: 0,
					endOffset:   52,
				},
			},
		},
		{
			description: "allows upper case methods",
			input:       "GET _search\nPOST _search\nPATCH _search\nPUT _search\nHEAD _search",
			got: []parsedRequest{
				{
					startOffset: 0,
					endOffset:   11,
				},
				{
					startOffset: 12,
					endOffset:   24,
				},
				{
					startOffset: 25,
					endOffset:   38,
				},
				{
					startOffset: 39,
					endOffset:   50,
				},
				{
					startOffset: 51,
					endOffset:   63,
				},
			},
		},
		{
			description: "allows lower case methods",
			input:       "get _search\npost _search\npatch _search\nput _search\nhead _search",
			got: []parsedRequest{
				{
					startOffset: 0,
					endOffset:   11,
				},
				{
					startOffset: 12,
					endOffset:   24,
				},
				{
					startOffset: 25,
					endOffset:   38,
				},
				{
					startOffset: 39,
					endOffset:   50,
				},
				{
					startOffset: 51,
					endOffset:   63,
				},
			},
		},
		{
			description: "allows mixed case methods",
			input:       "GeT _search\npOSt _search\nPaTch _search\nPut _search\nheAD _search",
			got: []parsedRequest{
				{
					startOffset: 0,
					endOffset:   11,
				},
				{
					startOffset: 12,
					endOffset:   24,
				},
				{
					startOffset: 25,
					endOffset:   38,
				},
				{
					startOffset: 39,
					endOffset:   50,
				},
				{
					startOffset: 51,
					endOffset:   63,
				},
			},
		},
	}

	a := require.New(t)
	for _, tc := range testCases {
		p := newParser(tc.input)
		got, err := p.parse()
		require.NoError(t, err)
		a.Equal(tc.got, got, "description: %s", tc.description)
	}
}
