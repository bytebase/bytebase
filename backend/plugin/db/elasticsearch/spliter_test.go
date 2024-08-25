package elasticsearch

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSpliter(t *testing.T) {
	tests := []struct {
		input    string
		expected []*statement
	}{
		// Test 1: basic test.
		{
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
					route:  []byte(`books/_search`),
					queryBody: []byte(`{
	"query": {
		"match_all": {}
	}
}
`),
				},
				{
					method: []byte("GET"),
					route:  []byte(`_cat/indices`),
				}},
		},
		// Test 2: with CR characters.
		{
			input: "GET /_cat/indices\r\n\r\nget /_cat/indices\r\nget /_cat/indices\r\n\r\n",
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
		// Test 3: with emoijs.
		{
			input: "PUT /my_index\r\n\r\nPOST /my_index/_doc\r\n{\r\n\t\"emoij\":\"😈😈😈😈\"}\r\nGET /",
			expected: []*statement{
				{
					method: []byte("PUT"),
					route:  []byte("/my_index"),
				},
				{
					method:    []byte("POST"),
					route:     []byte("/my_index/_doc"),
					queryBody: []byte("{\r\n\t\"emoij\":\"😈😈😈😈\"}\n"),
				},
				{
					method: []byte("GET"),
					route:  []byte("/"),
				},
			},
		},
		// Test 4: bulk APIs.
		{
			input: "POST /_bulk\r\n{ \"index\" : { \"_index\" : \"test\", \"_id\" : \"1\" } }\r\n{ \"field1\" : \"value1\" }\r\n\r\n",
			expected: []*statement{
				{
					method:    []byte("POST"),
					route:     []byte("/_bulk"),
					queryBody: []byte("{ \"index\" : { \"_index\" : \"test\", \"_id\" : \"1\" } }\n{ \"field1\" : \"value1\" }\n\n"),
				},
			},
		},
	}

	for _, test := range tests {
		stats, err := splitElasticsearchStatements(test.input)
		if err != nil {
			t.Fatal(err.Error())
		}
		require.Equal(t, test.expected, stats)
	}
}
