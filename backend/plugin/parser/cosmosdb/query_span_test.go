package cosmosdb

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPredicateInWhere(t *testing.T) {
	testCases := []struct {
		statement      string
		predicatePaths map[string]bool
	}{
		{
			statement: "SELECT * FROM container WHERE container.name = 'test'",
			predicatePaths: map[string]bool{
				"container.name": true,
			},
		},
		{
			statement: "SELECT * FROM container WHERE container.addresses[1].country = 'Canada'",
			predicatePaths: map[string]bool{
				"container.addresses[1].country": true,
			},
		},
		{
			statement: "SELECT * FROM container c WHERE udf.foo(c.name, c.salary, c.addresses[1].country) = c.age",
			predicatePaths: map[string]bool{
				"container.name":                 true,
				"container.salary":               true,
				"container.addresses[1].country": true,
				"container.age":                  true,
			},
		},
	}

	a := require.New(t)

	for _, tc := range testCases {
		querySpan, err := getQuerySpanImpl(tc.statement)
		a.Nil(err)
		a.Equal(len(tc.predicatePaths), len(querySpan.PredicatePaths))
		for path := range tc.predicatePaths {
			_, ok := querySpan.PredicatePaths[path]
			a.True(ok)
		}
	}
}
