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
		{
			statement: `SELECT * FROM container c WHERE c.status != "inactive"`,
			predicatePaths: map[string]bool{
				"container.status": true,
			},
		},
		{
			statement: `SELECT * FROM container c WHERE c.country IN ("US", "UK", "CA")`,
			predicatePaths: map[string]bool{
				"container.country": true,
			},
		},
		{
			statement: `SELECT * FROM container c WHERE c.population BETWEEN 100000 AND 5000000`,
			predicatePaths: map[string]bool{
				"container.population": true,
			},
		},
		{
			statement: `SELECT * FROM container c WHERE c.country != "US" AND c._ts > 1000`,
			predicatePaths: map[string]bool{
				"container.country": true,
				"container._ts":     true,
			},
		},
		{
			statement: `SELECT * FROM container c WHERE CONTAINS(c.name, "test")`,
			predicatePaths: map[string]bool{
				"container.name": true,
			},
		},
		{
			statement: `SELECT * FROM container c WHERE c.name LIKE "Alice%"`,
			predicatePaths: map[string]bool{
				"container.name": true,
			},
		},
		{
			statement: `SELECT * FROM container c WHERE NOT c.active`,
			predicatePaths: map[string]bool{
				"container.active": true,
			},
		},
		{
			statement: `SELECT * FROM container c WHERE c.email = "x" OR c.name = "y"`,
			predicatePaths: map[string]bool{
				"container.email": true,
				"container.name":  true,
			},
		},
		{
			statement: `SELECT * FROM container c WHERE c.firstName || " " || c.lastName = "Alice Smith"`,
			predicatePaths: map[string]bool{
				"container.firstName": true,
				"container.lastName":  true,
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
