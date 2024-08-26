package elasticsearch

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSpliter(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []*statement
	}{
		// Test 1: basic test.
		{
			name: "test 1",
			input: `POST /_bulk
{ "index" : { "_index" : "books" } }
{"name": "Revelation Space", "author": "天舟", "page_count": 585}
{ "index" : { "_index" : "books" } }
{"name": "1984", "author": "天舟", "page_count": 328}
GET books/_search 
{
	"query": {
		"match_all": {}
	}
}
GET _cat/indices`,
			expected: []*statement{
				{
					method: []byte("POST"),
					route:  []byte("/_bulk"),
					queryBody: []byte(`{ "index" : { "_index" : "books" } }
{"name": "Revelation Space", "author": "天舟", "page_count": 585}
{ "index" : { "_index" : "books" } }
{"name": "1984", "author": "天舟", "page_count": 328}
`),
				},
				{
					method: []byte("GET"),
					route:  []byte(`/books/_search`),
					queryBody: []byte(`{
	"query": {
		"match_all": {}
	}
}
`),
				},
				{
					method: []byte("GET"),
					route:  []byte(`/_cat/indices`),
				}},
		},
		// Test 2: with CR characters.
		{
			name:  "test 2",
			input: "\nGET /_cat/indices\r\n\r\nget /_cat/indices\r\nget /_cat/indices\r\n\r\n",
			expected: []*statement{
				{
					method: []byte("GET"),
					route:  []byte("/_cat/indices"),
				},
				{
					method: []byte("GET"),
					route:  []byte("/_cat/indices"),
				},
				{
					method: []byte("GET"),
					route:  []byte("/_cat/indices"),
				},
			},
		},
		// Test 3: with emojis.
		{
			name:  "test 3",
			input: "\r\n\rPUT /my_index\r\n\r\nPOST /my_index/_doc\r\n{\r\n\t\"emoji\":\"😈😈😈😈\"}\r\nGET /",
			expected: []*statement{
				{
					method: []byte("PUT"),
					route:  []byte("/my_index"),
				},
				{
					method:    []byte("POST"),
					route:     []byte("/my_index/_doc"),
					queryBody: []byte("{\r\n\t\"emoji\":\"😈😈😈😈\"}\n"),
				},
				{
					method: []byte("GET"),
					route:  []byte("/"),
				},
			},
		},
		// Test 4: bulk APIs.
		{
			name:  "test 4",
			input: "\r\nPOST /_bulk\r\n{ \"index\" : { \"_index\" : \"test\", \"_id\" : \"1\" } }\r\n{ \"field1\" : \"value1\" }\r\n\r\n",
			expected: []*statement{
				{
					method:    []byte("POST"),
					route:     []byte("/_bulk"),
					queryBody: []byte("{ \"index\" : { \"_index\" : \"test\", \"_id\" : \"1\" } }\n{ \"field1\" : \"value1\" }\n\n"),
				},
			},
		},
	}

	a := require.New(t)
	for _, test := range tests {
		stmts, err := splitElasticsearchStatements(test.input)
		a.NoError(err, test.name)
		a.Equal(test.expected, stmts, test.name)
	}
}
