package elasticsearch

import (
	"testing"
)

func TestSpliter(t *testing.T) {
	test := struct {
		input    string
		expected []Statement
	}{
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
		expected: []Statement{{
			method: "POST",
			route:  []byte("/_bulk"),
			queryString: []byte(`{ "index" : { "_index" : "books" } }
{"name": "Revelation Space", "author": "天舟", "page_count": 585}
{ "index" : { "_index" : "books" } }
{"name": "1984", "author": "天舟", "page_count": 328}
`),
		}, {
			method: "GET",
			route:  []byte(`books/_search`),
			queryString: []byte(`{
	"query": {
		"match_all": {}
	}
}
`),
		}, {
			method:      "GET",
			route:       []byte(`_cat/indices`),
			queryString: []byte{},
		}},
	}

	stats, err := SplitElasticsearchStatements(test.input)
	if err != nil {
		t.Fatal(err.Error())
	}

	for index, statement := range stats {
		expectedStatement := test.expected[index]
		if expectedStatement.method != statement.method ||
			string(expectedStatement.route) != string(statement.route) ||
			string(statement.queryString) != string(expectedStatement.queryString) {
			t.Fail()
		}
	}
}
