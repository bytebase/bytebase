package elasticsearch

import (
	"fmt"
	"testing"
)

func TestSpliter(t *testing.T) {
	test := struct {
		input    string
		expected int
	}{
		input: `POST /_bulk
{ "index" : { "_index" : "books" } }
{"name": "Revelation Space", "author": "Alastair Reynolds", "page_count": 585}
{ "index" : { "_index" : "books" } }
{"name": "1984", "author": "George Orwell", "page_count": 328}
GET books/_search 
{
	"query": {
		"match_all": {}
	}
}
GET _cat/indsices`,
		expected: 3,
	}

	stats, err := SplitElasticsearchStatements(test.input)
	if err != nil {
		t.Fatal(err.Error())
	}

	if len(stats) != test.expected {
		t.Fail()
	}

	for _, s := range stats {
		fmt.Printf("\n%s %s\n%s\n", s.method, s.route, s.queryString)
	}
}
