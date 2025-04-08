package elasticsearch

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestParseElasticsearchREST(t *testing.T) {
	type testCase struct {
		Description string       `yaml:"description,omitempty"`
		Statement   string       `yaml:"statement,omitempty"`
		Result      *ParseResult `yaml:"result,omitempty"`
	}

	var (
		filepath = "test-data/parse-elasticsearch-rest.yaml"
		record   = false
	)

	a := require.New(t)
	yamlFile, err := os.Open(filepath)
	a.NoError(err)
	byteValue, err := io.ReadAll(yamlFile)
	a.NoError(err)
	a.NoError(yamlFile.Close())

	var testCases []testCase
	a.NoError(yaml.Unmarshal(byteValue, &testCases))

	for i, tc := range testCases {
		got, err := ParseElasticsearchREST(tc.Statement)
		a.NoErrorf(err, "description: %s", tc.Description)
		if record {
			testCases[i].Result = got
		} else {
			a.Equalf(tc.Result, got, "description: %s", tc.Description)
		}
	}

	if record {
		byteValue, err := yaml.Marshal(testCases)
		a.NoError(err)
		err = os.WriteFile(filepath, byteValue, 0644)
		a.NoError(err)
	}
}

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

func TestGetEditorRequest(t *testing.T) {
	testCases := []struct {
		description           string
		content               []string
		adjustedParsedRequest adjustedParsedRequest
		want                  *editorRequest
	}{
		{
			description: "cleans up any text following the url",
			content:     []string{"GET _search // inline comment"},
			adjustedParsedRequest: adjustedParsedRequest{
				startLineNumber: 0,
				endLineNumber:   0,
			},
			want: &editorRequest{
				method: "GET",
				url:    "_search",
				data:   nil,
			},
		},
		{
			description: "doesn't incorrectly removes parts of url params that include whitespaces",
			content:     []string{`GET _search?query="test test"`},
			adjustedParsedRequest: adjustedParsedRequest{
				startLineNumber: 0,
				endLineNumber:   0,
			},
			want: &editorRequest{
				method: "GET",
				url:    `_search?query="test test"`,
				data:   nil,
			},
		},
		{
			description: "correctly includes the request body",
			content: []string{
				"GET _search",
				"{",
				"  \"query\": {}",
				"}",
			},
			adjustedParsedRequest: adjustedParsedRequest{
				startLineNumber: 0,
				endLineNumber:   3,
			},
			want: &editorRequest{
				method: "GET",
				url:    `_search`,
				data: []string{
					"{\n  \"query\": {}\n}",
				},
			},
		},
		{
			description: "correctly handles nested braces",
			content: []string{
				"GET _search",
				"{",
				`  "query": "{a} {b}"`,
				"}",
				"{",
				`  "query": {}`,
				"}",
			},
			adjustedParsedRequest: adjustedParsedRequest{
				startLineNumber: 0,
				endLineNumber:   6,
			},
			want: &editorRequest{
				method: "GET",
				url:    `_search`,
				data: []string{
					"{\n  \"query\": \"{a} {b}\"\n}",
					"{\n  \"query\": {}\n}",
				},
			},
		},
		{
			description: "works for several request bodies",
			content: []string{
				"GET _search",
				"{",
				`  "query": {}`,
				"}",
				"{",
				`  "query": {}`,
				"}",
			},
			adjustedParsedRequest: adjustedParsedRequest{
				startLineNumber: 0,
				endLineNumber:   6,
			},
			want: &editorRequest{
				method: "GET",
				url:    `_search`,
				data: []string{
					"{\n  \"query\": {}\n}",
					"{\n  \"query\": {}\n}",
				},
			},
		},
		{
			description: "splits several json objects",
			content: []string{
				"GET _search",
				`{"query":"test"}`,
				`{`,
				`  "query": "test"`,
				`}`,
				`{"query":"test"}`,
			},
			adjustedParsedRequest: adjustedParsedRequest{
				startLineNumber: 0,
				endLineNumber:   5,
			},
			want: &editorRequest{
				method: "GET",
				url:    `_search`,
				data: []string{
					`{"query":"test"}`,
					"{\n  \"query\": \"test\"\n}",
					`{"query":"test"}`,
				},
			},
		},
		{
			description: "works for invalid json objects",
			content: []string{
				"GET _search",
				`{"query":"test"}`,
				`{`,
				`  "query":`,
				`{`,
			},
			adjustedParsedRequest: adjustedParsedRequest{
				startLineNumber: 0,
				endLineNumber:   4,
			},
			want: &editorRequest{
				method: "GET",
				url:    `_search`,
				data: []string{
					`{"query":"test"}`,
					"{\n  \"query\":\n{",
				},
			},
		},
	}

	a := require.New(t)
	for _, tc := range testCases {
		got := getEditorRequest(strings.Join(tc.content, "\n"), tc.adjustedParsedRequest)
		a.Equal(tc.want, got, "description: %s", tc.description)
	}
}

func TestContainsComments(t *testing.T) {
	testCases := []struct {
		description string
		input       string
		want        bool
	}{
		{
			description: "should return false for JSON with // and /* inside strings",
			input: `{
      "docs": [
        {
          "_source": {
            "trace": {
              "name": "GET /actuator/health/**"
            },
            "transaction": {
              "outcome": "success"
            }
          }
        },
        {
          "_source": {
            "vulnerability": {
              "reference": [
                "https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2020-15778"
              ]
            }
          }
        }
      ]
    }`,
			want: false,
		},
		{
			description: "should return true for text with actual line comment",
			input: `{
      // This is a comment
      "query": { "match_all": {} }
    }`,
			want: true,
		},
		{
			description: "should return true for text with actual block comment",
			input: `{
      /* Bulk insert */
      "index": { "_index": "test" },
      "field1": "value1"
    }`,
			want: true,
		},
		{
			description: "should return false for text without any comments",
			input: `{
      "field": "value"
    }`,
			want: false,
		},
		{
			description: "should return false for empty string",
			input:       ``,
			want:        false,
		},
		{
			description: "should correctly handle escaped quotes within strings",
			input: `{
      "field": \"value with \\\"escaped quotes\\\"\"
    }`,
			want: false,
		},
		{
			description: "should return true if comment is outside of strings",
			input: `{
      "field": "value" // comment here
    }`,
			want: true,
		},
	}

	a := require.New(t)
	for _, tc := range testCases {
		got := containsComments(tc.input)
		a.Equal(tc.want, got, "description: %s", tc.description)
	}
}
